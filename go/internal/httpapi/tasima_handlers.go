package httpapi

import (
	"bytes"
	"context"
	"encoding/csv"
	"io"
	"net/http"
	"strings"
	"time"

	"theblock/site/internal/platform"
	"theblock/site/internal/repo"
)

// ---- Taşıma & 301 yönlendirme (spec 01 §D7/J2, 03 P4-9) ----

// redirCacheEntry site başına yönlendirme haritasının 60 sn'lik önbelleği.
type redirCacheEntry struct {
	ats time.Time
	m   map[string]repo.Redirect
}

// tasimaData panel görünümü (yönlendirmeler + CSV içe aktarım).
type tasimaData struct {
	Redirects        []repo.Redirect
	Count            int64
	Query            string
	RedirectImported int
	// içerik CSV önizleme / içe aktarım
	Preview    []tasimaRow
	RawCSV     string
	Imported   int
	Apply      bool
	ImportErrs []string
	Categories []string
}

type tasimaRow struct {
	Line     int
	Title    string
	Category string
	Status   string
	Tags     string
	OK       bool
	Err      string
}

// redirectMap önbellekli yönlendirme haritası (60 sn TTL).
func (s *Server) redirectMap(ctx context.Context, siteID int64) map[string]repo.Redirect {
	s.redirMu.Lock()
	defer s.redirMu.Unlock()
	if e, ok := s.redirCache[siteID]; ok && time.Since(e.ats) < 60*time.Second {
		return e.m
	}
	m, err := repo.LoadRedirectMap(ctx, s.pool, siteID)
	if err != nil {
		return nil
	}
	s.redirCache[siteID] = redirCacheEntry{ats: time.Now(), m: m}
	return m
}

// invalidateRedirects önbelleği düşürür (yönlendirme değiştiğinde).
func (s *Server) invalidateRedirects() {
	s.redirMu.Lock()
	s.redirCache = map[int64]redirCacheEntry{}
	s.redirMu.Unlock()
}

// maybeRedirect istek yoluna karşılık gelen 301/302 yönlendirmesini uygular.
// Uygulandıysa true döner (çağıran başka işlem yapmamalı).
func (s *Server) maybeRedirect(w http.ResponseWriter, r *http.Request, siteID int64) bool {
	m := s.redirectMap(r.Context(), siteID)
	if len(m) == 0 {
		return false
	}
	path := r.URL.Path
	rd, ok := m[path]
	if !ok && len(path) > 1 {
		rd, ok = m[strings.TrimRight(path, "/")]
	}
	if !ok {
		return false
	}
	code := rd.StatusCode
	switch code {
	case 301, 302, 307, 308:
	default:
		code = 301
	}
	_ = repo.IncrementRedirectHit(r.Context(), s.pool, siteID, rd.SourcePath)
	http.Redirect(w, r, rd.TargetURL, code)
	return true
}

// ---- Panel: Taşıma & Yönlendirme sayfası ----

func (s *Server) handleTasimaGet(w http.ResponseWriter, r *http.Request) {
	d := s.buildTasimaData(r)
	if IsHTMX(r) {
		s.renderTasimaPanelV2(w, r, d)
		return
	}
	s.renderTasimaPageV2(w, r, d)
}

func (s *Server) buildTasimaData(r *http.Request) *tmplData {
	ctx := r.Context()
	sid := s.curSiteID(r)
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	redirs, _ := repo.ListRedirects(ctx, s.pool, sid, q, 500)
	total, _ := repo.CountRedirects(ctx, s.pool, sid)
	cats, _ := repo.ListCategories(ctx, s.pool, sid)
	names := make([]string, 0, len(cats))
	for _, c := range cats {
		names = append(names, c.Name)
	}
	d := s.baseView("tr", "/admin", "Taşıma & Yönlendirme")
	d.User = ptr(userFromContext(ctx))
	d.Flash = s.consumeFlash(r)
	d.Tasima = &tasimaData{Redirects: redirs, Count: total, Query: q, Categories: names}
	return d
}

func (s *Server) renderTasimaPanel(w http.ResponseWriter, r *http.Request, kind, text string) {
	d := s.buildTasimaData(withHTMXCurrentQuery(r))
	if kind != "" {
		d.Flash = &flashMsg{Kind: kind, Text: text}
	}
	s.renderTasimaPanelV2(w, r, d)
}

func (s *Server) tasimaResult(w http.ResponseWriter, r *http.Request, kind, text string) {
	if IsHTMX(r) {
		s.renderTasimaPanel(w, r, kind, text)
		return
	}
	s.flashRedirect(w, r, "/admin/tasima", kind, text)
}

// ---- 301 yönlendirme CRUD ----

func (s *Server) handleRedirectSave(w http.ResponseWriter, r *http.Request) {
	src := normalizeSourcePath(r.FormValue("kaynak"))
	dest := strings.TrimSpace(r.FormValue("hedef"))
	if src == "" || dest == "" {
		s.tasimaResult(w, r, "error", "Kaynak yol ve hedef URL gerekli.")
		return
	}
	code := parseIntOr(r.FormValue("kod"))
	rd := &repo.Redirect{SiteID: s.curSiteID(r), SourcePath: src, TargetURL: dest, StatusCode: code, Note: strings.TrimSpace(r.FormValue("not"))}
	if err := repo.UpsertRedirect(r.Context(), s.pool, rd); err != nil {
		s.logger.Error("yönlendirme kaydı", "error", err)
		s.tasimaResult(w, r, "error", "Kaydedilemedi.")
		return
	}
	s.invalidateRedirects()
	s.audit(r, "yonlendirme_kaydet", src+" -> "+dest)
	s.tasimaResult(w, r, "info", "Yönlendirme kaydedildi: "+src)
}

func (s *Server) handleRedirectDelete(w http.ResponseWriter, r *http.Request) {
	src := normalizeSourcePath(r.FormValue("kaynak"))
	if err := repo.DeleteRedirect(r.Context(), s.pool, s.curSiteID(r), src); err != nil {
		s.tasimaResult(w, r, "error", "Silinemedi.")
		return
	}
	s.invalidateRedirects()
	s.audit(r, "yonlendirme_sil", src)
	s.tasimaResult(w, r, "info", "Yönlendirme silindi: "+src)
}

func (s *Server) handleRedirectImport(w http.ResponseWriter, r *http.Request) {
	data, err := readCSVInput(r, 8<<20)
	if err != nil || len(bytes.TrimSpace(data)) == 0 {
		s.flashRedirect(w, r, "/admin/tasima", "error", "CSV içeriği boş veya okunamadı.")
		return
	}
	recs, err := parseAnyCSV(data)
	if err != nil || len(recs) < 2 {
		s.flashRedirect(w, r, "/admin/tasima", "error", "CSV çözümlenemedi (en az başlık + 1 satır gerekli).")
		return
	}
	idx := headerIndex(recs[0], [][]string{
		{"kaynak", "kaynak_yol", "eski", "eski_url", "source", "source_path", "from", "url"},
		{"hedef", "hedef_url", "yeni", "yeni_url", "target", "target_url", "to", "destination"},
		{"kod", "durum", "status", "status_code", "code"},
		{"not", "note", "aciklama", "açıklama"},
	})
	if idx[0] < 0 || idx[1] < 0 {
		s.flashRedirect(w, r, "/admin/tasima", "error", "CSV başlıklarında 'kaynak' ve 'hedef' sütunları bulunamadı.")
		return
	}
	sid := s.curSiteID(r)
	var list []repo.Redirect
	skipped := 0
	for _, row := range recs[1:] {
		src := normalizeSourcePath(at(row, idx[0]))
		dest := strings.TrimSpace(at(row, idx[1]))
		if src == "" || dest == "" {
			skipped++
			continue
		}
		code := 301
		if idx[2] >= 0 {
			if c := parseIntOr(at(row, idx[2])); c == 301 || c == 302 {
				code = c
			}
		}
		note := ""
		if idx[3] >= 0 {
			note = strings.TrimSpace(at(row, idx[3]))
		}
		list = append(list, repo.Redirect{SiteID: sid, SourcePath: src, TargetURL: dest, StatusCode: code, Note: note})
	}
	if len(list) == 0 {
		s.flashRedirect(w, r, "/admin/tasima", "error", "Geçerli satır yok.")
		return
	}
	n, err := repo.BulkUpsertRedirects(r.Context(), s.pool, list)
	if err != nil {
		s.logger.Error("yönlendirme toplu içe aktarım", "error", err)
		s.flashRedirect(w, r, "/admin/tasima", "error", "İçe aktarım hatası: "+err.Error())
		return
	}
	s.invalidateRedirects()
	s.audit(r, "yonlendirme_ice_aktar", itoa(n)+" satır")
	msg := itoa(n) + " yönlendirme içe aktarıldı."
	if skipped > 0 {
		msg += " " + itoa(skipped) + " satır atlandı (boş kaynak/hedef)."
	}
	s.flashRedirect(w, r, "/admin/tasima", "info", msg)
}

// ---- CSV içerik toplu içe aktarım ----

func (s *Server) handleContentImport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	r.Body = http.MaxBytesReader(w, r.Body, 20<<20)
	_ = r.ParseMultipartForm(16 << 20)
	apply := r.FormValue("uygula") == "1"
	data, err := readCSVInput(r, 16<<20)
	if err != nil || len(bytes.TrimSpace(data)) == 0 {
		s.flashRedirect(w, r, "/admin/tasima", "error", "CSV içeriği boş veya okunamadı.")
		return
	}
	sid := s.curSiteID(r)
	u := userFromContext(ctx)
	rows, imported, errs := s.importContentsCSV(ctx, sid, u.ID, data, apply)

	cats, _ := repo.ListCategories(ctx, s.pool, sid)
	names := make([]string, 0, len(cats))
	for _, c := range cats {
		names = append(names, c.Name)
	}
	redirs, _ := repo.ListRedirects(ctx, s.pool, sid, "", 500)
	total, _ := repo.CountRedirects(ctx, s.pool, sid)

	d := s.baseView("tr", "/admin", "Taşıma & Yönlendirme")
	d.User = ptr(u)
	if !apply {
		d.Flash = &flashMsg{Kind: "info", Text: "CSV önizleme — henüz kaydedilmedi. Onaylarsanız 'İçe Aktar'a basın."}
	} else {
		d.Flash = &flashMsg{Kind: "info", Text: itoa(imported) + " içerik içe aktarıldı."}
	}
	d.Tasima = &tasimaData{
		Redirects: redirs, Count: total, Categories: names,
		Preview: rows, RawCSV: string(data), Imported: imported, Apply: apply, ImportErrs: errs,
	}
	s.renderTasimaPageV2(w, r, d)
}

func (s *Server) importContentsCSV(ctx context.Context, siteID, authorID int64, data []byte, apply bool) (rows []tasimaRow, imported int, errs []string) {
	recs, err := parseAnyCSV(data)
	if err != nil || len(recs) < 2 {
		return nil, 0, []string{"CSV çözümlenemedi (en az başlık + 1 satır gerekli)."}
	}
	idx := headerIndex(recs[0], [][]string{
		{"baslik", "başlık", "title", "ad", "headline"},
		{"spot", "özet", "ozet", "summary", "aciklama", "açıklama"},
		{"detay", "içerik", "icerik", "metin", "body", "content", "text"},
		{"kategori", "category", "bolum", "bölüm"},
		{"etiket", "etiketler", "tags", "tag"},
		{"durum", "status"},
		{"slug"},
		{"seo_baslik", "seo_başlık", "seo_title"},
		{"seo_aciklama", "seo_açıklama", "seo_description"},
		{"kaynak_url", "source_url"},
	})
	if idx[0] < 0 {
		return nil, 0, []string{"CSV başlıklarında 'baslik' sütunu bulunamadı."}
	}

	cats, _ := repo.ListCategories(ctx, s.pool, siteID)
	catByKey := map[string]int64{}
	for _, c := range cats {
		catByKey[foldTR(c.Name)] = c.ID
		catByKey[foldTR(c.Slug)] = c.ID
	}

	for ln, row := range recs[1:] {
		title := strings.TrimSpace(at(row, idx[0]))
		if title == "" {
			continue
		}
		r := tasimaRow{Line: ln + 2, Title: title}
		catName := ""
		if idx[3] >= 0 {
			catName = strings.TrimSpace(at(row, idx[3]))
		}
		r.Category = catName
		if idx[4] >= 0 {
			r.Tags = strings.TrimSpace(at(row, idx[4]))
		}
		status := "taslak"
		if idx[5] >= 0 {
			status = normalizeStatus(at(row, idx[5]))
		}
		r.Status = status
		var catID *int64
		if catName != "" {
			if id, ok := catByKey[foldTR(catName)]; ok {
				catID = &id
			} else {
				r.Err = "kategori bulunamadı: " + catName
			}
		}
		r.OK = r.Err == ""
		rows = append(rows, r)

		if !apply {
			continue
		}
		c := &repo.Content{
			TypeKey:       "haber",
			Status:        status,
			Title:         title,
			Spot:          atField(row, idx, 1),
			Detail:        textToBlocks(atField(row, idx, 2)),
			CategoryID:    catID,
			AuthorID:      authorID,
			SiteID:        siteID,
			CommentStatus: "acik",
			SourceURL:     optStr(atField(row, idx, 9)),
			SeoTitle:      optStr(atField(row, idx, 7)),
			SeoDesc:       optStr(atField(row, idx, 8)),
		}
		if sl := strings.TrimSpace(atField(row, idx, 6)); sl != "" {
			c.Slug = optStr(platform.Slugify(sl))
		} else {
			c.Slug = optStr(platform.Slugify(title))
		}
		c.Locale = "tr"
		if status == "yayinda" {
			now := time.Now()
			c.PublishedAt = &now
		}
		id, err := repo.CreateContent(ctx, s.pool, c)
		if err != nil {
			r.OK = false
			r.Err = "kaydedilemedi: " + err.Error()
			rows[len(rows)-1] = r
			errs = append(errs, "satır "+itoa(ln+2)+": "+err.Error())
			continue
		}
		if tags := splitTags(atField(row, idx, 4)); len(tags) > 0 {
			_ = repo.ReplaceContentTags(ctx, s.pool, id, tags, platform.Slugify)
		}
		imported++
	}
	return rows, imported, errs
}

// ---- Örnek CSV indirme ----

func (s *Server) handleTasimaSampleCSV(w http.ResponseWriter, r *http.Request) {
	tur := r.URL.Query().Get("tur")
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+tur+"-ornek.csv\"")
	w.Write([]byte{0xEF, 0xBB, 0xBF}) // BOM (Excel Türkçe uyumu)
	switch tur {
	case "yonlendirme":
		io.WriteString(w, "kaynak,hedef,kod\n/eski-haber-123,https://theblock.com.tr/yeni-haber-456,301\n/2019/05/kampanya,https://theblock.com.tr/kampanya,301\n")
	default:
		io.WriteString(w, "baslik,spot,detay,kategori,etiketler,durum,slug,seo_baslik,seo_aciklama,kaynak_url\n")
		io.WriteString(w, "Örnek Haber Başlığı,Kısa özet metni.,Paragraf bir.\nParagraf iki.,Ekonomi,\"ekonomi,piyasa\",taslak,,,,\n")
	}
}

// ---- CSV yardımcıları ----

// readCSVInput dosya yükleme veya textarea içeriğini okur.
func readCSVInput(r *http.Request, limit int64) ([]byte, error) {
	if f, _, err := r.FormFile("dosya"); err == nil {
		defer f.Close()
		return io.ReadAll(io.LimitReader(f, limit))
	}
	return []byte(r.FormValue("csv")), nil
}

// parseAnyCSV BOM'u ve otomatik ayraç tespitini yaparak CSV'yi çözer.
func parseAnyCSV(data []byte) ([][]string, error) {
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF, 0xFE})
	rd := csv.NewReader(bytes.NewReader(data))
	rd.Comma = detectDelim(data)
	rd.FieldsPerRecord = -1
	rd.LazyQuotes = true
	rd.TrimLeadingSpace = true
	return rd.ReadAll()
}

func detectDelim(data []byte) rune {
	line := string(data)
	if i := strings.IndexAny(line, "\r\n"); i >= 0 {
		line = line[:i]
	}
	counts := map[rune]int{',': 0, ';': 0, '\t': 0, '|': 0}
	for _, r := range line {
		if _, ok := counts[r]; ok {
			counts[r]++
		}
	}
	best, bestN := ',', 0
	for _, d := range []rune{',', ';', '\t', '|'} {
		if counts[d] > bestN {
			best, bestN = d, counts[d]
		}
	}
	return best
}

// headerIndex başlık satırında alias'lara göre sütun indekslerini bulur (-1 yoksa).
func headerIndex(header []string, aliases [][]string) []int {
	norm := make([]string, len(header))
	for i, h := range header {
		norm[i] = foldTR(strings.TrimSpace(h))
	}
	out := make([]int, len(aliases))
	for i, alts := range aliases {
		out[i] = -1
		for _, a := range alts {
			fa := foldTR(a)
			for j, h := range norm {
				if h == fa {
					out[i] = j
					break
				}
			}
			if out[i] >= 0 {
				break
			}
		}
	}
	return out
}

func at(row []string, i int) string {
	if i >= 0 && i < len(row) {
		return row[i]
	}
	return ""
}

func atField(row []string, idx []int, i int) string {
	if i >= 0 && i < len(idx) {
		return at(row, idx[i])
	}
	return ""
}

// foldTR Türkçe karakterleri ASCII'ye katlar ve küçük harfe çevirir (eşleştirme için).
func foldTR(s string) string {
	repl := strings.NewReplacer(
		"I", "i", "İ", "i", "ı", "i", "Ş", "s", "ş", "s",
		"Ğ", "g", "ğ", "g", "Ü", "u", "ü", "u", "Ö", "o", "ö", "o",
		"Ç", "c", "ç", "c", "Â", "a", "â", "a", "Î", "i", "î", "i", "Û", "u", "û", "u",
	)
	return strings.ToLower(repl.Replace(s))
}

// normalizeSourcePath yönlendirme kaynağını normalize eder (baştaki / zorunlu).
func normalizeSourcePath(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if !strings.HasPrefix(s, "/") {
		s = "/" + s
	}
	return s
}

func normalizeStatus(s string) string {
	switch foldTR(strings.TrimSpace(s)) {
	case "yayinda", "yayinla", "publish", "published", "canli":
		return "yayinda"
	case "arsiv", "archive":
		return "arsiv"
	case "planlanmis", "planli", "scheduled":
		return "planlanmis"
	default:
		return "taslak"
	}
}

func splitTags(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == '|' || r == ';' })
	var out []string
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
