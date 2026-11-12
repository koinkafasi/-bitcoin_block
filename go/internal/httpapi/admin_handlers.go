package httpapi

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"mime/multipart"
	"net/http"
	"net/url"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"theblock/site/internal/ai"
	"theblock/site/internal/aieditor"
	"theblock/site/internal/ceviri"
	"theblock/site/internal/platform"
	"theblock/site/internal/repo"
	"theblock/site/internal/searchsvc"
	"theblock/site/internal/webui/templkit"
)

// ---- Dashboard ----

// sparkline değer serisini SVG polyline noktalarına çevirir (84x26).
func sparkline(vals []int) string {
	if len(vals) < 2 {
		return ""
	}
	max := 1
	for _, v := range vals {
		if v > max {
			max = v
		}
	}
	n := len(vals)
	var b strings.Builder
	for i, v := range vals {
		x := float64(i) * 84.0 / float64(n-1)
		y := 24.0 - float64(v)*20.0/float64(max)
		if i > 0 {
			b.WriteString(" ")
		}
		fmt.Fprintf(&b, "%.1f,%.1f", x, y)
	}
	return b.String()
}

// pickTone bekleyen iş sayısına göre kart tonu döndürür.
func pickTone(n int) string {
	if n > 0 {
		return "danger"
	}
	return "ok"
}

func lenAll(m map[string]int) int {
	n := 0
	for _, v := range m {
		n += v
	}
	return n
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	pos := len(b)
	for n > 0 {
		pos--
		b[pos] = byte('0' + n%10)
		n /= 10
	}
	return string(b[pos:])
}

func ptr[T any](v T) *T { return &v }

// ---- İçerik listesi ----

func (s *Server) handleContentsList(w http.ResponseWriter, r *http.Request) {
	d := s.buildContentsListData(r)
	// Aynı handler: normal tarayıcı tam sayfa, HTMX yalnızca liste paneli.
	// Panel v2: içerik listesi templ bileşenleriyle render edilir.
	if IsHTMX(r) {
		s.renderIceriklerPanel(w, r, d)
		return
	}
	s.renderIceriklerPage(w, r, d)
}

func (s *Server) buildContentsListData(r *http.Request) *tmplData {
	ctx := r.Context()
	d := s.baseView("tr", "/admin", "İçerikler")
	d.User = ptr(userFromContext(ctx))
	d.Flash = s.consumeFlash(r)

	status := r.URL.Query().Get("durum")
	if status == "hepsi" {
		status = ""
	}
	q := r.URL.Query().Get("q")
	tip := r.URL.Query().Get("tip")
	kategori := parseID(r.URL.Query().Get("kategori"))
	gun, _ := strconv.Atoi(r.URL.Query().Get("gun"))
	if gun < 0 {
		gun = 0
	}
	sayfa, _ := strconv.Atoi(r.URL.Query().Get("sayfa"))
	if sayfa < 1 {
		sayfa = 1
	}
	// Faz 126 — ses/röportaj filtresi: transkripti olan / işi süren /
	// başarısız içerikleri ayırır.
	audioFilter := r.URL.Query().Get("ses")
	// Triyaj (U1): sıralama ve AI skor eşiği. Sıralama repo'da
	// beyaz listeden geçer; buradan ham değer taşınabilir.
	sirala := r.URL.Query().Get("sirala")
	// Faz 192 — ?dil=en: "bu dilde sürümü OLAN makaleler". Satır yine kök
	// dilde gösterilir (temsilci), yalnız küme daralır.
	dilFiltre := r.URL.Query().Get("dil")
	minSkor, _ := strconv.Atoi(r.URL.Query().Get("minskor"))
	minSkorStr := r.URL.Query().Get("minskor")
	switch audioFilter {
	case repo.AudioHas, repo.AudioPending, repo.AudioFailed:
	default:
		audioFilter = ""
	}
	// Faz 47 — özel tarih aralığı (gün bazlı; bitiş dahil).
	baslangicStr := r.URL.Query().Get("baslangic")
	bitisStr := r.URL.Query().Get("bitis")
	var baslangic, bitis *time.Time
	if t, err := time.ParseInLocation("2006-01-02", baslangicStr, time.Local); err == nil {
		baslangic = &t
	}
	if t, err := time.ParseInLocation("2006-01-02", bitisStr, time.Local); err == nil {
		t = t.AddDate(0, 0, 1) // günün tamamı: < ertesi gün
		bitis = &t
	}

	counts, _ := repo.CountByStatus(ctx, s.pool, s.curSiteID(r))
	baseQS := func(durum string, sayfa int, withQ bool) template.URL {
		base := "/admin/icerikler?durum=" + durum
		v := url.Values{}
		if tip != "" {
			v.Set("tip", tip)
		}
		if kategori != 0 {
			v.Set("kategori", itoa(int(kategori)))
		}
		if gun != 0 {
			v.Set("gun", itoa(gun))
		}
		if baslangicStr != "" {
			v.Set("baslangic", baslangicStr)
		}
		if bitisStr != "" {
			v.Set("bitis", bitisStr)
		}
		if withQ && q != "" {
			v.Set("q", q)
		}
		if audioFilter != "" {
			v.Set("ses", audioFilter)
		}
		// Triyaj sıralaması ve eşiği sayfa geçişlerinde korunur; aksi
		// hâlde 2. sayfada liste tarihe göre sıralanmış gibi görünürdü.
		if sirala != "" {
			v.Set("sirala", sirala)
		}
		if minSkorStr != "" {
			v.Set("minskor", minSkorStr)
		}
		if dilFiltre != "" {
			v.Set("dil", dilFiltre)
		}
		if sayfa > 1 {
			v.Set("sayfa", itoa(sayfa))
		}
		if len(v) > 0 {
			base += "&" + v.Encode()
		}
		return template.URL(base)
	}
	qsf := func(durum string) template.URL { return baseQS(durum, 0, false) }
	var statuses []statusTab
	for _, st := range []struct {
		code, label string
	}{{"yayinda", "Yayında"}, {"taslak", "Taslak"}, {"planlanmis", "Planlanmış"}, {"arsiv", "Arşiv"}, {"cop", "Çöp"}} {
		statuses = append(statuses, statusTab{Code: st.code, Label: st.label, Count: counts[st.code], Active: status == st.code, Href: qsf(st.code)})
	}
	filtre := repo.ListFilter{
		Status: status, Q: q, TypeKey: tip, CategoryID: kategori, LastDays: gun,
		Baslangic: baslangic, Bitis: bitis, Audio: audioFilter, Sort: sirala,
		MinScore: minSkor, SiteID: s.curSiteID(r), Page: sayfa, PerPage: 40,
		KokBazli: true, KokDil: "tr", Locale: dilFiltre,
	}
	items, total, err := repo.ListContents(ctx, s.pool, filtre)
	if err != nil {
		s.logger.Error("içerik listesi", "error", err)
	}
	toplamSayfa := (total + 39) / 40
	if toplamSayfa < 1 {
		toplamSayfa = 1
	}
	if sayfa > toplamSayfa {
		gercek := total
		if gercek == 0 {
			sf := func(f repo.ListFilter) repo.ListFilter { f.Page, f.PerPage = 1, 1; return f }
			sfFiltre := filtre
			sfFiltre.Status, sfFiltre.Q, sfFiltre.TypeKey = status, q, tip
			sfFiltre.CategoryID, sfFiltre.LastDays = kategori, gun
			sfFiltre.Baslangic, sfFiltre.Bitis = baslangic, bitis
			sfFiltre.Audio, sfFiltre.Sort, sfFiltre.MinScore = audioFilter, sirala, minSkor
			sfFiltre.SiteID, sfFiltre.Locale = s.curSiteID(r), dilFiltre
			if _, t2, err2 := repo.ListContents(ctx, s.pool, sf(sfFiltre)); err2 == nil && t2 > 0 {
				gercek = t2
			}
		}
		toplamSayfa = (gercek + 39) / 40
		if toplamSayfa < 1 {
			toplamSayfa = 1
		}
		sayfa = toplamSayfa
		filtre.Page, filtre.PerPage = sayfa, 40
		items, total, err = repo.ListContents(ctx, s.pool, filtre)
		if err != nil {
			s.logger.Error("içerik listesi (sayfa kıvp)", "error", err)
		}
		toplamSayfa = (total + 39) / 40
		if toplamSayfa < 1 {
			toplamSayfa = 1
		}
	}
	// Sayfalama çubuğu: çok sayfa varsa 1 … [sayfa-2..sayfa+2] … son penceresi.
	var pages []pagerItem
	addPag := func(n int, label string, dots bool) {
		pages = append(pages, pagerItem{Label: label, Href: baseQS(status, n, true), Active: n == sayfa, Dots: dots})
	}
	if toplamSayfa <= 7 {
		for n := 1; n <= toplamSayfa; n++ {
			addPag(n, itoa(n), false)
		}
	} else {
		lo, hi := sayfa-2, sayfa+2
		if lo < 1 {
			lo = 1
		}
		if hi > toplamSayfa {
			hi = toplamSayfa
		}
		if hi-lo+1 < 5 { // kenara yaslıysa pencere 5'e tamamlansın
			if lo == 1 {
				if hi+5-(hi-lo+1) < toplamSayfa {
					hi = hi + 5 - (hi - lo + 1)
				} else {
					hi = toplamSayfa
				}
			} else {
				if lo-(5-(hi-lo+1)) > 1 {
					lo = lo - (5 - (hi - lo + 1))
				} else {
					lo = 1
				}
			}
		}
		if lo > 1 {
			addPag(1, "1", false)
			if lo > 2 {
				addPag(0, "…", true)
			}
		}
		for n := lo; n <= hi; n++ {
			addPag(n, itoa(n), false)
		}
		if hi < toplamSayfa {
			if hi < toplamSayfa-1 {
				addPag(0, "…", true)
			}
			addPag(toplamSayfa, itoa(toplamSayfa), false)
		}
	}
	// Toplu "kategori taşı" seçimi + kategori filtresi için kategoriler (Faz 26/30).
	catOpts, _ := repo.ListCategories(ctx, s.pool, s.curSiteID(r))
	types, _ := repo.ListContentTypes(ctx, s.pool)

	katAd := ""
	for _, c := range catOpts {
		if c.ID == kategori {
			katAd = c.Name
			break
		}
	}

	// Faz 42 — okunma (batch) + yayınlanma tarihi görüntüsü (published_at varsa
	// onu, yoksa updated_at; yerel saat, listeye yakışır kısa biçim).
	ids := make([]int64, 0, len(items))
	for i := range items {
		ids = append(ids, items[i].ID)
	}
	hits := repo.HitsOf(ctx, s.pool, ids)
	// Faz 192 — sayfadaki köklerin dil sürümleri (tek sorgu, N+1 yok).
	kokler := make([]int64, 0, len(items))
	for i := range items {
		kokler = append(kokler, items[i].TranslationRootID())
	}
	diller := repo.DillerOf(ctx, s.pool, kokler)
	yayinlar := make(map[int64]string, len(items))
	rows := make([]contentRowView, 0, len(items))
	for i := range items {
		t := items[i].UpdatedAt
		if items[i].PublishedAt != nil {
			t = *items[i].PublishedAt
		}
		yayinlar[items[i].ID] = t.In(time.Local).Format("02.01 15:04")
		rows = append(rows, contentRowView{
			Content: items[i],
			Yayin:   yayinlar[items[i].ID],
			Hit:     hits[items[i].ID],
			Diller:  diller[items[i].TranslationRootID()],
		})
	}

	// Faz 48 — kaydedilmiş görünümler: mevcut filtre sorgusu + çip listesi.
	curV := url.Values{}
	if status != "" {
		curV.Set("durum", status)
	}
	if tip != "" {
		curV.Set("tip", tip)
	}
	if kategori != 0 {
		curV.Set("kategori", itoa(int(kategori)))
	}
	if gun != 0 {
		curV.Set("gun", itoa(gun))
	}
	if baslangicStr != "" {
		curV.Set("baslangic", baslangicStr)
	}
	if bitisStr != "" {
		curV.Set("bitis", bitisStr)
	}
	if audioFilter != "" {
		curV.Set("ses", audioFilter)
	}
	if q != "" {
		curV.Set("q", q)
	}
	if sirala != "" {
		curV.Set("sirala", sirala)
	}
	if minSkorStr != "" {
		curV.Set("minskor", minSkorStr)
	}
	currentQS := curV.Encode()
	views := s.loadListViews(ctx)

	// Yayın önceliği eksenleri: yalnız bu sayfadaki satırlar için tek sorgu.
	eksenler := map[int64]repo.IcerikEksen{}
	if len(items) > 0 {
		ids := make([]int64, 0, len(items))
		for _, it := range items {
			ids = append(ids, it.ID)
		}
		if m, err := repo.IcerikEksenleri(ctx, s.pool, ids); err == nil {
			eksenler = m
		}
	}
	d.Contents = &contentsData{Items: items, Rows: rows, Statuses: statuses, Status: status, Q: q, Total: total, Eksenler: eksenler,
		Categories: catOpts, Tip: tip, Kategori: kategori, Gun: gun, Types: types, KategoriAd: katAd,
		Baslangic: baslangicStr, Bitis: bitisStr, SesFiltre: audioFilter,
		Sirala: sirala, MinSkor: minSkorStr,
		Views: views, CurrentQS: currentQS, CurrentDurum: status,
		Yayinlar: yayinlar, Hits: hits,
		TumuHref: qsf("hepsi"), Sayfa: sayfa, ToplamSayfa: toplamSayfa, Pages: pages,
		PrevHref: baseQS(status, sayfa-1, true), NextHref: baseQS(status, sayfa+1, true),
		DilFiltre: dilFiltre, KokDil: "tr"}
	return d
}

// renderContentsPanel, HTMX aksiyonlarından sonra listeyi kullanıcının
// mevcut filtreleriyle yeniden basar. kind boşsa flash gösterilmez.
func (s *Server) renderContentsPanel(w http.ResponseWriter, r *http.Request, kind, text string) {
	d := s.buildContentsListData(withHTMXCurrentQuery(r))
	if kind != "" {
		d.Flash = &flashMsg{Kind: kind, Text: text}
	}
	s.renderIceriklerPanel(w, r, d)
}

// ---- Kaydedilmiş liste görünümleri (filtre ön ayarları — Faz 48) ----

func (s *Server) loadListViews(ctx context.Context) []ListView {
	b, err := repo.GetSetting(ctx, s.pool, "listview", "views")
	if err != nil {
		return nil
	}
	var views []ListView
	if json.Unmarshal(b, &views) != nil {
		return nil
	}
	return views
}

func (s *Server) saveListViews(ctx context.Context, views []ListView) {
	b, _ := json.Marshal(views)
	_ = repo.SetSetting(ctx, s.pool, "listview", "views", b)
}

// handleListViewSave mevcut filtre sorgusunu adlandırılmış görünüm olarak saklar.
func (s *Server) handleListViewSave(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ad := strings.TrimSpace(r.FormValue("ad"))
	qs := strings.TrimSpace(r.FormValue("qs"))
	if ad == "" || qs == "" {
		s.flashRedirect(w, r, "/admin/icerikler", "error", "Görünüm adı ve filtre gerekli.")
		return
	}
	views := s.loadListViews(ctx)
	out := make([]ListView, 0, len(views)+1)
	seen := false
	for _, v := range views {
		if v.Ad == ad {
			v.QS = qs
			seen = true
		}
		out = append(out, v)
	}
	if !seen {
		out = append(out, ListView{Ad: ad, QS: qs})
	}
	if len(out) > 12 {
		out = out[:12]
	}
	s.saveListViews(ctx, out)
	s.audit(r, "liste_gorunum_kaydet", ad)
	s.flashRedirect(w, r, "/admin/icerikler?"+qs, "info", "Görünüm kaydedildi: "+ad)
}

// handleListViewDel bir görünümü siler.
func (s *Server) handleListViewDel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ad := strings.TrimSpace(r.FormValue("ad"))
	views := s.loadListViews(ctx)
	out := make([]ListView, 0, len(views))
	for _, v := range views {
		if v.Ad != ad {
			out = append(out, v)
		}
	}
	s.saveListViews(ctx, out)
	s.audit(r, "liste_gorunum_sil", ad)
	s.flashRedirect(w, r, "/admin/icerikler", "info", "Görünüm silindi: "+ad)
}

// ---- İçerik formu (yeni/düzenle) ----

func (s *Server) handleContentFormNew(w http.ResponseWriter, r *http.Request) {
	typ := r.URL.Query().Get("tip")
	if typ == "" {
		typ = "haber"
	}
	c := &repo.Content{TypeKey: typ, Status: "taslak", CommentStatus: "acik", Locale: "tr"}
	d := s.buildContentForm(w, r, c, true)
	if IsHTMX(r) {
		s.renderIcerikFormPanelV2(w, r, d)
		return
	}
	s.renderIcerikFormPageV2(w, r, d)
}

func (s *Server) handleContentFormEdit(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	c, err := repo.GetContent(r.Context(), s.pool, id)
	if err != nil {
		s.flashRedirect(w, r, "/admin/icerikler", "error", "İçerik bulunamadı.")
		return
	}
	if !canEditContent(userFromContext(r.Context()), c) {
		s.flashRedirect(w, r, "/admin/icerikler", "error", "Bu içeriği düzenleme yetkiniz yok.")
		return
	}
	d := s.buildContentForm(w, r, &c, false)
	if IsHTMX(r) {
		s.renderIcerikFormPanelV2(w, r, d)
		return
	}
	s.renderIcerikFormPageV2(w, r, d)
}

func (s *Server) buildContentForm(w http.ResponseWriter, r *http.Request, c *repo.Content, isNew bool) *tmplData {
	ctx := r.Context()
	d := s.baseView("tr", "/admin", "İçerik Düzenle")
	d.User = ptr(userFromContext(ctx))
	d.Flash = s.consumeFlash(r)

	f := &formData{Content: *c, IsNew: isNew, TypeKey: c.TypeKey}
	if cf := repo.GetContentFields(ctx, s.pool, c.ID); cf.AudioURL != "" {
		f.AudioURL, f.AudioStatus, f.AudioSeconds = cf.AudioURL, cf.AudioStatus, cf.AudioSeconds
	}
	if types, err := repo.ListContentTypes(ctx, s.pool); err == nil {
		for _, t := range types {
			f.Types = append(f.Types, typeOpt{Key: t.Key, Label: t.Label})
		}
	}
	f.Categories, _ = repo.ListCategories(ctx, s.pool, s.curSiteID(r))
	f.Mahrets, _ = repo.ListMahrets(ctx, s.pool)
	f.Media, _ = repo.ListMedia(ctx, s.pool, s.curSiteID(r), userFromContext(ctx).ID, 200)
	// Yazar/editör ataması (Faz 51).
	f.People, _ = repo.ListPeopleAdmin(ctx, s.pool)
	f.AuthorSelected = c.AuthorID
	if !isNew {
		if e, err := repo.ContentEditor(ctx, s.pool, c.ID); err == nil {
			f.EditorSelected = e.ID
		}
	}
	f.Clusters, _ = repo.ListClusters(ctx, s.pool, s.curSiteID(r))
	// Önerilen Video seçici için video haberleri + mevcut elle bağlı id (Faz 12b).
	f.Videos, _, _ = repo.ListContents(ctx, s.pool, repo.ListFilter{
		TypeKey: "video", Status: "yayinda", SiteID: s.curSiteID(r), PerPage: 200, Page: 1,
	})
	if !isNew {
		f.RelatedVideoID, _ = repo.RelatedVideoID(ctx, s.pool, c.ID)
	}
	// Video süresi (Faz 17): düzenlenen içeriğin kendi süresi + seçici listesi için
	// tüm videoların süresi (tek sorgu).
	f.VideoSure = platform.FormatDurationTR(repo.GetContentFields(ctx, s.pool, c.ID).VideoSeconds)
	if len(f.Videos) > 0 {
		ids := make([]int64, 0, len(f.Videos))
		for _, v := range f.Videos {
			ids = append(ids, v.ID)
		}
		secs := repo.VideoDurations(ctx, s.pool, ids)
		f.VideoSureMap = make(map[int64]string, len(secs))
		for id, sec := range secs {
			f.VideoSureMap[id] = platform.FormatDurationTR(sec)
		}
		// Kategori filtresi: listedeki kategoriler, yinelenmeden (Faz 25).
		seen := map[int64]bool{}
		for _, v := range f.Videos {
			if v.CategoryID == nil || *v.CategoryID <= 0 || seen[*v.CategoryID] {
				continue
			}
			seen[*v.CategoryID] = true
			ad := ""
			if v.CategoryName != nil {
				ad = *v.CategoryName
			}
			if ad != "" {
				f.VideoCats = append(f.VideoCats, catOpt{ID: *v.CategoryID, Name: ad})
			}
		}
		sort.Slice(f.VideoCats, func(i, j int) bool {
			return strings.ToLower(f.VideoCats[i].Name) < strings.ToLower(f.VideoCats[j].Name)
		})
	}

	if c.CategoryID != nil {
		f.CategorySelected = *c.CategoryID
	}
	if c.MahrepID != nil {
		f.MahretSelected = *c.MahrepID
	}
	f.DetailText = blocksToText(c.Detail)
	f.DetailJSON = s.detailToEditorJSON(c.Detail)

	if !isNew {
		if tags, err := repo.ContentTagNames(ctx, s.pool, c.ID); err == nil {
			f.TagsText = strings.Join(tags, ", ")
		}
		if c.PublishAt != nil {
			f.PublishAtValue = c.PublishAt.Local().Format("2006-01-02T15:04")
		}
		f.ClusterSelected, _ = repo.ContentClusterID(ctx, s.pool, c.ID)
	}

	// medya slotları
	slots := []struct {
		name, help string
		sel        *int64
	}{
		{"icerik_resim", "Kapak (16:9)", c.CoverMediaID},
		{"manset_resim", "Manşet", c.MansetMediaID},
		{"sosyal_resim", "Sosyal (OG)", c.SocialMediaID},
	}
	for _, s1 := range slots {
		ms := mediaSlot{Name: s1.name, Help: s1.help}
		for _, m := range f.Media {
			ms.Options = append(ms.Options, mediaOption{
				ID: m.ID, Filename: m.Filename, Selected: s1.sel != nil && *s1.sel == m.ID,
				ThumbURL: s.mediaThumbURL(&m.ID) + "?v=" + itoa64(m.UpdatedAt.Unix()), Alt: anyStrPtr(m.Alt), Caption: anyStrPtr(m.Caption),
				Title: m.Title,
			})
		}
		f.MediaSlots = append(f.MediaSlots, ms)
	}

	// rozetler
	for _, rz := range []struct {
		code string
		on   bool
	}{
		{"AM", c.IsAM}, {"UM", c.IsUM}, {"DM", c.IsDM}, {"SD", c.IsSD}, {"FH", c.IsFH}, {"HK", c.IsHK},
	} {
		f.Rozets = append(f.Rozets, formRozet{Code: rz.code, On: rz.on})
	}

	// SEO
	words := wordCount(plainText(f.DetailText))
	h2s := countTag(f.DetailText, "h2")
	hasCover := c.CoverMediaID != nil
	innerImgs := strings.Count(f.DetailText, "[gorsel:")
	boldCount := countTag(f.DetailText, "strong") + countTag(f.DetailText, "b>")
	italicCount := countTag(f.DetailText, "em") + countTag(f.DetailText, "i>")
	tagCount := len(tagList(f.TagsText))
	f.SEO = repo.ComputeSEO(c.Title, c.Spot, words, h2s, hasCover, innerImgs, tagCount, boldCount, italicCount)
	f.SEOText = seoText(f.SEO.Score)
	f.TitleLen = len([]rune(c.Title))
	if c.ID > 0 {
		f.Fields = repo.GetContentFields(ctx, s.pool, c.ID)
	}

	// dil linkleri (çeviriler)
	f.Langs = s.langLinks(ctx, c, isNew)
	f.LangRows = s.langRows(ctx, *c, isNew)
	for _, satir := range f.LangRows {
		if !satir.Active && satir.ContentID > 0 {
			f.CeviriSayisi++
		}
	}
	// Çeviri düzenleniyorsa kaynak metni hazırla (karşılaştırma bandı).
	if c.TranslationOf != nil && c.ID > 0 {
		_, kbaslik, kspot, kgovde := s.ceviriBaglam(ctx, *c)
		f.KaynakBaslik = kbaslik
		f.KaynakSpot = kspot
		f.KaynakGovdeHTML = kgovde
	}

	// hesaplayıcı şeması
	if c.TypeKey == "hesaplayici" {
		extra, _ := repo.GetContentExtra(ctx, s.pool, c.ID)
		if len(extra) > 2 {
			f.HesaplayiciSema = prettyJSON(extra)
		} else {
			f.HesaplayiciSema = defaultCalcSema
		}
	}

	// Güncel trend önerileri (editörde etikete ekle)
	catID := int64(0)
	if c.CategoryID != nil {
		catID = *c.CategoryID
	}
	f.TrendSuggestions, _ = repo.TrendSuggestionsForContent(ctx, s.pool, s.curSiteID(r), catID, 8)

	// YMYL + kaynaklar
	f.IsYMYL = c.IsYMYL
	f.YMYLDurumu = c.YMYLDurumu
	if c.YMYLUzman != nil {
		f.YMYLUzman = *c.YMYLUzman
	}
	if !isNew {
		f.Sources, _ = repo.ListContentSources(ctx, s.pool, c.ID)
		origText := strings.Join([]string{c.Title, c.Spot, plainText(f.DetailText)}, " ")
		f.OrigIndex, _, _ = repo.OriginalityIndex(ctx, s.pool, c.ID, origText)
		f.Audio, _ = repo.ListAudioForContent(ctx, s.pool, c.SiteID, c.ID)
		// Kalite ölçümü: AI dedektörü değil; ritim, çeşitlilik, klişe ve
		// tekrar metrikleri. Sonuç yalnız bilgilendirir.
		f.Kalite = s.contentQuality(r, *c)
		f.KaliteChecked = true
	}
	for _, a := range aieditor.Actions {
		if a.Hidden {
			continue
		}
		f.AIActions = append(f.AIActions, aiActionView{
			ID: a.ID, Label: a.Label, Group: a.Group, Description: a.Description,
			Icon: a.Icon, Research: a.Research,
		})
	}
	d.Form = f
	return d
}

var defaultCalcSema = `{
  "fields": [
    {"id":"ana","label":"Ana Para (₺)","type":"number","default":100000},
    {"id":"orani","label":"Yıllık Faiz (%)","type":"number","default":42.5},
    {"id":"vade","label":"Vade (ay)","type":"number","default":12}
  ],
  "outputs": [
    {"label":"Aylık Getiri (₺)","expr":"[ana] * [orani] / 100 / [vade]"},
    {"label":"Toplam Getiri (₺)","expr":"[ana] * [orani] / 100"}
  ]
}`

func tagList(s string) []string {
	parts := strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == ';' || r == ' ' })
	var out []string
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func plainText(s string) string {
	return s
}

func seoText(score int) string {
	switch {
	case score >= 80:
		return "Çok iyi"
	case score >= 60:
		return "İyi"
	case score >= 40:
		return "Orta"
	default:
		return "Geliştirilmeli"
	}
}

// langLinks içerik için çeviri linklerini kurar.
func (s *Server) langLinks(ctx context.Context, c *repo.Content, isNew bool) []langLink {
	var links []langLink
	enabled := s.i18n.Enabled()
	existing := map[string]int64{}
	if !isNew {
		trs, err := repo.TranslationsOf(ctx, s.pool, c.ID)
		if err == nil {
			for _, t := range trs {
				existing[t.Locale] = t.ID
				if t.Locale == c.Locale {
					continue
				}
			}
		}
	}
	for _, l := range enabled {
		if l.Code == c.Locale {
			links = append(links, langLink{Label: l.NativeName, Active: true})
			continue
		}
		if id, ok := existing[l.Code]; ok {
			links = append(links, langLink{Label: l.NativeName, URL: "/admin/icerikler/" + itoa(int(id))})
		} else if !isNew {
			links = append(links, langLink{Label: l.NativeName + " +", URL: "/admin/icerikler/" + itoa(int(c.ID)) + "/cevir/" + l.Code})
		} else {
			links = append(links, langLink{Label: l.NativeName})
		}
	}
	return links
}

// ---- İçerik kaydet (yeni + güncelle) ----

func (s *Server) handleContentSave(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// ParseMultipartForm'un maxMemory'si yalnız bellekte tutulan kısmı
	// sınırlar; fazlası diske yazılır. Gövde tavanı ayrıca konur.
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
	_ = r.ParseMultipartForm(8 << 20)
	id := parseID(r.PathValue("id"))
	isNew := id == 0

	c := &repo.Content{ID: id}
	c.Version = int(parseID(r.FormValue("version")))
	c.TypeKey = r.FormValue("tip")
	if c.TypeKey == "" {
		c.TypeKey = "haber"
	}
	c.Title = strings.TrimSpace(r.FormValue("baslik"))
	c.Spot = strings.TrimSpace(r.FormValue("spot"))
	c.Detail = parseDetailInput(r.FormValue("detay"))
	c.CategoryID = optID(r.FormValue("kategori"))
	c.MahrepID = optID(r.FormValue("mahrec"))
	c.CoverMediaID = optID(r.FormValue("icerik_resim"))
	c.MansetMediaID = optID(r.FormValue("manset_resim"))
	c.SocialMediaID = optID(r.FormValue("sosyal_resim"))
	c.CommentStatus = r.FormValue("yorum_durumu")
	if c.CommentStatus == "" {
		c.CommentStatus = "acik"
	}
	c.SeoTitle = seoAlan(r.FormValue("seo_baslik"), platform.SeoBaslikSertMax)
	c.SeoDesc = seoAlan(r.FormValue("seo_aciklama"), platform.SeoAciklamaSertMax)
	c.SourceURL = optStr(r.FormValue("kaynak_url"))
	c.SourceName = optStr(r.FormValue("kaynak_ad"))
	c.PublishAt = parseLocalTime(r.FormValue("yayin_tarihi"))
	c.IsYMYL = r.FormValue("is_ymyl") != ""
	c.YMYLDurumu = r.FormValue("ymyl_durumu")
	if c.IsYMYL && c.YMYLDurumu == "" {
		c.YMYLDurumu = "dogrulanmadi"
	}
	c.YMYLUzman = optStr(r.FormValue("ymyl_uzman"))
	c.IsExclusive = r.FormValue("is_exclusive") != ""
	c.ImageGrayscale = r.FormValue("image_grayscale") != ""
	// Icerik bazli katman kilidi (kategori varsayilani dahil).
	var categoryDefault int
	if isNew && c.CategoryID != nil {
		n, err := repo.CategoryMinTierWeight(r.Context(), s.pool, *c.CategoryID)
		if err != nil {
			s.flashRedirect(w, r, "/admin/icerikler", "error", "Kategori katman kuralı okunamadı: "+err.Error())
			return
		}
		categoryDefault = n
	}
	if !isNew {
		n, err := repo.ContentMinWeight(r.Context(), s.pool, id)
		if err != nil {
			s.flashRedirect(w, r, "/admin/icerikler/"+strconv.FormatInt(id, 10), "error", "Mevcut katman okunamadı: "+err.Error())
			return
		}
		c.MinTierWeight = n
	}
	applyTierLock(r, c, categoryDefault)
	if v := r.FormValue("ai_skor"); v != "" {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			c.AiScore = n
		}
	}

	// rozetler
	for _, v := range r.Form["rozet"] {
		switch v {
		case "AM":
			c.IsAM = true
		case "UM":
			c.IsUM = true
		case "DM":
			c.IsDM = true
		case "SD":
			c.IsSD = true
		case "FH":
			c.IsFH = true
		case "HK":
			c.IsHK = true
		}
	}
	tags := tagList(r.FormValue("etiketler"))

	action := r.FormValue("kaydet")
	if action == "" {
		action = "taslak"
	}

	u := userFromContext(ctx)
	c.AuthorID = u.ID
	c.SiteID = s.curSiteID(r)

	if isNew {
		c.Locale = "tr"
		c.Status = "taslak"
		c.Slug = optStr(platform.Slugify(c.Title))
		if c.Slug == nil {
			c.Slug = optStr("icerik-" + rndSuffix())
		}
		nid, err := repo.CreateContent(ctx, s.pool, c)
		if err != nil {
			s.flashRedirect(w, r, "/admin/icerikler", "error", "Kaydedilemedi: "+err.Error())
			return
		}
		c.ID = nid
	} else {
		cur, err := repo.GetContent(ctx, s.pool, id)
		if err != nil {
			s.flashRedirect(w, r, "/admin/icerikler", "error", "İçerik bulunamadı.")
			return
		}
		if !canEditContent(u, cur) {
			s.flashRedirect(w, r, "/admin/icerikler", "error", "Bu içeriği düzenleme yetkiniz yok.")
			return
		}
		// Formda sürüm yoksa (eski istemci) mevcut sürümü kullan; yine de
		// güncelleme sırasında başka bir kayıt olursa çakışma yakalanır.
		if c.Version <= 0 {
			c.Version = cur.Version
		}
		c.Status = cur.Status
		c.Locale = cur.Locale
		c.TranslationOf = cur.TranslationOf
		c.Slug = cur.Slug
		c.AuthorID = cur.AuthorID
		// Kaydetmeden önceki hali sürüm geçmişine yazılır (geri alma için).
		_ = repo.SnapshotContentRevision(ctx, s.pool, id, &u.ID, "manuel", nil)
		if err := repo.UpdateContent(ctx, s.pool, c); err != nil {
			if errors.Is(err, repo.ErrConflict) {
				s.flashRedirect(w, r, "/admin/icerikler/"+strconv.FormatInt(id, 10), "error",
					"İçerik siz düzenlerken başka biri tarafından değiştirildi. Sayfa yenilendi; değişikliklerinizi tekrar uygulayın.")
				return
			}
			s.flashRedirect(w, r, "/admin/icerikler", "error", "Güncellenemedi: "+err.Error())
			return
		}
	}
	// Yazar / editör ataması (Faz 51): formda seçim varsa uygula.
	if v := optID(r.FormValue("author_id")); v != nil && *v > 0 && *v != c.AuthorID {
		_ = repo.SetContentAuthor(ctx, s.pool, c.ID, *v)
		c.AuthorID = *v
	}
	_ = repo.SetContentEditor(ctx, s.pool, c.ID, optID(r.FormValue("editor_id")))

	// AI alaka puanı (yeni ve mevcut içerik için)
	_ = repo.SetContentAIScore(ctx, s.pool, c.ID, c.AiScore)

	// otomatik kapak görseli (kategori image_service — spec 06 K-D)
	if c.CoverMediaID == nil {
		s.maybeAutoCover(ctx, c)
	}

	_ = repo.ReplaceContentTags(ctx, s.pool, c.ID, tags, platform.Slugify)

	// önyüz/alan spec (07): görsel kredisi + sosyal özet + AI özet
	_ = repo.SetContentEditorial(ctx, s.pool, c.ID,
		strings.TrimSpace(r.FormValue("gorsel_kredi")),
		strings.TrimSpace(r.FormValue("sosyal_ozet")),
		strings.TrimSpace(r.FormValue("ai_ozet")))

	// Önerilen Video — elle bağlanan video haber (Faz 12b); boşsa kaldır.
	var relVideoID int64
	if p := optID(r.FormValue("related_video")); p != nil {
		relVideoID = *p
	}
	_ = repo.SetRelatedVideoID(ctx, s.pool, c.ID, relVideoID)

	// Video süresi (Faz 17): "3:42" / "1:02:03" / "222" → saniye. Alan yalnız
	// video türünde basıldığı için diğer türlerde boş gelir ve süre sıfırlanır.
	_ = repo.SetContentVideoDuration(ctx, s.pool, c.ID, platform.ParseDurationTR(r.FormValue("video_sure")))

	// konu kümesi bağlama
	if kv := optID(r.FormValue("kume")); kv != nil {
		_ = repo.SetContentCluster(ctx, s.pool, c.ID, *kv)
	} else if !isNew {
		_ = repo.SetContentCluster(ctx, s.pool, c.ID, 0)
	}

	// hesaplayıcı şeması (tür hesaplayıcı ise)
	if c.TypeKey == "hesaplayici" {
		sema := strings.TrimSpace(r.FormValue("hesap_sema"))
		if sema == "" {
			sema = defaultCalcSema
		}
		if !json.Valid([]byte(sema)) {
			s.flashRedirect(w, r, "/admin/icerikler/"+itoa(int(c.ID)), "error", "Hesaplayıcı şeması geçersiz JSON — kaydedilmedi.")
			return
		}
		_ = repo.SetContentExtra(ctx, s.pool, c.ID, []byte(sema))
	}

	// durum geçişi (YMYL onayı yoksa yayın kapalı)
	msg := "Kaydedildi."
	if action == "yayinla" && c.IsYMYL && c.YMYLDurumu != "uzman" {
		action = "taslak"
		msg = "YMYL içeriği yayınlanmadı — uzman denetimi gerekli (onaya düştü)."
	}
	// kategori AI alaka kapısı (spec 06 K-A)
	if action == "yayinla" {
		if g := repo.CheckScoreGate(ctx, s.pool, *c); !g.Allowed {
			action = "taslak"
			msg = "Yayın kapısı: " + g.Reason + " — içerik taslağa alındı."
		}
	}
	// Çeviri kapısı (Faz 120): kaynaksız makine çevirisi yayına alınamaz.
	if action == "yayinla" {
		if g := s.translationGate(ctx, *c); !g.Allowed {
			action = "taslak"
			msg = g.Reason + " — içerik taslağa alındı."
		}
	}
	switch action {
	case "yayinla":
		if c.PublishAt != nil && c.PublishAt.After(time.Now()) {
			// planlanmış yayın
			if err := repo.SetContentStatus(ctx, s.pool, c.ID, "planlanmis"); err != nil {
				s.logger.Error("planla", "error", err)
			}
		} else {
			_ = repo.SetContentStatus(ctx, s.pool, c.ID, "yayinda")
			// kalıntı ileri planlama varsa temizle
			_ = repo.ClearFuturePublish(ctx, s.pool, c.ID)
		}
	case "arsiv":
		_ = repo.SetContentStatus(ctx, s.pool, c.ID, "arsiv")
	default: // taslak
		_ = repo.SetContentStatus(ctx, s.pool, c.ID, "taslak")
	}

	s.invalidateContentCache()
	s.cdShellCache.Clear()
	if s.sayfaCache != nil {
		s.sayfaCache.temizle()
	}

	s.audit(r, "icerik_kaydet", action+": "+c.Title+" (id="+itoa(int(c.ID))+")")
	if action == "yayinla" && !(c.PublishAt != nil && c.PublishAt.After(time.Now())) {
		c.Status = "yayinda"
	}
	s.indexContent(ctx, c)
	// Faz 38: form üzerinden yayınlanan içerik de IndexNow'a bildirilir
	// (auto_index ayarı açıkken; otomatik paylaşım değil — sadece URL ping).
	if c.Status == "yayinda" {
		s.autoIndex(ctx, s.curSiteID(r), c.ID)
	}
	// WebSub: yeni yayını hub'lara bildir (beslemeyi anında tazeletir).
	if c.Status == "yayinda" {
		go func() {
			// İstek bağlamı KASITLI kullanılmıyor: HTTP yanıtı dönünce
			// iptal olur ve bildirim yarıda kesilirdi. Çıplak Background()
			// ise bağlam değerlerini de kaybediyor ve süresiz çalışıyordu
			// (gosec G118). Sınırlı bağlam ikisini de çözüyor.
			bg, iptal := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
			defer iptal()
			s.websubYayinBildir(bg, []int64{c.ID})
		}()
	}
	s.publishContentLive(ctx, c.ID, c.SiteID, false)
	s.publishEditorSaved(ctx, c.ID, c.SiteID, 0)
	s.contentFormResult(w, r, c.ID, "info", msg)
}

// contentFormResult kaydetme sonrası: HTMX'te düzenleyiciye döner (reload
// yok, flash üstte), normal istekte yönlendirir.
func (s *Server) contentFormResult(w http.ResponseWriter, r *http.Request, id int64, kind, text string) {
	if IsHTMX(r) {
		c, err := repo.GetContent(r.Context(), s.pool, id)
		if err != nil {
			s.flashRedirect(w, r, "/admin/icerikler", "error", "Kaydedildi; içerik yeniden yüklenemedi.")
			return
		}
		d := s.buildContentForm(w, r, &c, false)
		d.Flash = &flashMsg{Kind: kind, Text: text}
		s.renderIcerikFormPanelV2(w, r, d)
		return
	}
	s.flashRedirect(w, r, "/admin/icerikler/"+itoa(int(id)), kind, text)
}

// indexContent içeriği arama servisine yollar (servis yoksa sessizce atlanır).
func (s *Server) indexContent(ctx context.Context, c *repo.Content) {
	if s.search == nil || c == nil || c.Status != "yayinda" {
		return
	}
	tags, _ := repo.TagsOf(ctx, s.pool, c.ID)
	body := s.blocksToHTML(c.Detail, nil)
	doc := searchsvc.Doc{
		ID:       itoa(int(c.ID)),
		Title:    c.Title,
		Spot:     c.Spot,
		Body:     stripTags(string(body)),
		Tags:     tags,
		URL:      publicArticleURL(c.Locale, c.DisplaySlug(), c.ID),
		Category: catSlugPtr(c.CategorySlug),
	}
	if _, err := s.search.Index(ctx, []searchsvc.Doc{doc}); err != nil {
		s.logger.Warn("search-svc indeksleme hatasi", "content", c.ID, "error", err)
	}
}

// stripTags basit HTML etiket temizliği (arama indeksi gövdesi).
func stripTags(s string) string {
	var b strings.Builder
	depth := 0
	for _, r := range s {
		switch r {
		case '<':
			depth++
		case '>':
			if depth > 0 {
				depth--
			}
		default:
			if depth == 0 {
				b.WriteRune(r)
			}
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

// ---- Hızlı işlemler (yayınla/arşivle/sil/rozet) ----

func (s *Server) handleContentsBulk(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_ = r.ParseForm()
	islem := strings.TrimSpace(r.FormValue("islem"))

	var ids []int64
	for _, v := range r.Form["id"] {
		if p := optID(v); p != nil && *p > 0 {
			ids = append(ids, *p)
		}
	}
	geri := func(kind, msg string) {
		if IsHTMX(r) {
			s.renderContentsPanel(w, r, kind, msg)
			return
		}
		s.flashRedirect(w, r, "/admin/icerikler", kind, msg)
	}
	if len(ids) == 0 {
		geri("error", "Hiç içerik seçilmedi.")
		return
	}
	if len(ids) > 200 {
		ids = ids[:200]
	}

	// Yazar/Muhabir yalnızca kendi içeriklerinde toplu işlem yapabilir.
	atlandi := 0
	izinli := ids[:0]
	for _, id := range ids {
		c, err := repo.GetContent(ctx, s.pool, id)
		if err != nil || !canEditContent(userFromContext(ctx), c) {
			atlandi++
			continue
		}
		izinli = append(izinli, id)
	}
	ids = izinli
	if len(ids) == 0 {
		geri("error", "Seçili içeriklerde işlem yetkiniz yok.")
		return
	}
	basarili := 0
	switch islem {
	case "yayinla", "arsiv":
		status := "yayinda"
		if islem == "arsiv" {
			status = "arsiv"
		}
		for _, id := range ids {
			if status == "yayinda" {
				ok := true
				if c, err := repo.GetContent(ctx, s.pool, id); err != nil {
					ok = false
				} else if c.IsYMYL && c.YMYLDurumu != "uzman" {
					ok = false
				} else if g := repo.CheckScoreGate(ctx, s.pool, c); !g.Allowed {
					ok = false
				}
				if !ok {
					atlandi++
					continue
				}
			}
			if err := repo.SetContentStatus(ctx, s.pool, id, status); err != nil {
				atlandi++
				continue
			}
			basarili++
			s.publishContentLive(ctx, id, s.curSiteID(r), false)
			if status == "yayinda" {
				s.autoShare(ctx, s.curSiteID(r), id)
				s.autoIndex(ctx, s.curSiteID(r), id)
				// Faz 34: toplu yayınlanan içerik de Tantivy'ye girmeli.
				if c, err := repo.GetContent(ctx, s.pool, id); err == nil {
					c.Status = "yayinda"
					s.indexContent(ctx, &c)
				}
			}
		}
	case "kategori":
		ham := strings.TrimSpace(r.FormValue("kategori"))
		if ham == "" {
			geri("error", "Kategori seçilmedi.")
			return
		}
		catID := parseID(ham) // "0" → kategorisiz
		for _, id := range ids {
			if err := repo.SetContentCategory(ctx, s.pool, id, catID); err != nil {
				atlandi++
				continue
			}
			basarili++
		}
	case "planla":
		pt := parseLocalTime(strings.TrimSpace(r.FormValue("yayin_tarihi")))
		if pt == nil {
			geri("error", "Geçerli bir yayın tarihi girin (ör. 2026-10-20 09:00).")
			return
		}
		if !pt.After(time.Now()) {
			geri("error", "Yayın tarihi gelecekte olmalı.")
			return
		}
		for _, id := range ids {
			ok := true
			if c, err := repo.GetContent(ctx, s.pool, id); err != nil {
				ok = false
			} else if c.IsYMYL && c.YMYLDurumu != "uzman" {
				ok = false
			} else if g := repo.CheckScoreGate(ctx, s.pool, c); !g.Allowed {
				ok = false
			}
			if !ok {
				atlandi++
				continue
			}
			if err := repo.SetContentStatus(ctx, s.pool, id, "planlanmis"); err != nil {
				atlandi++
				continue
			}
			if err := repo.SetContentPublishAt(ctx, s.pool, id, pt); err != nil {
				atlandi++
				continue
			}
			s.publishContentLive(ctx, id, s.curSiteID(r), false)
			basarili++
		}
	case "rozet":
		// Toplu rozet ata/kaldır (Faz 39): AM/UM/DM/SD/FH/HK + açık/kapalı.
		rozet := strings.ToUpper(strings.TrimSpace(r.FormValue("rozet")))
		rozetAc := r.FormValue("rozet_durum") != "kapat"
		if _, ok := map[string]bool{"AM": true, "UM": true, "DM": true, "SD": true, "FH": true, "HK": true}[rozet]; !ok {
			geri("error", "Geçersiz rozet.")
			return
		}
		for _, id := range ids {
			if err := repo.SetContentRozet(ctx, s.pool, id, rozet, rozetAc); err != nil {
				atlandi++
				continue
			}
			basarili++
		}
	case "cevir":
		// Toplu çeviri: seçili içerikler için çeviri kuyruğuna istek
		// yazılır. Çeviriyi burada ÜRETMİYORUZ — kuyruk işleyicisi
		// sırayla alıyor. Yoksa 50 içerik seçen kullanıcı isteği
		// zaman aşımına uğrardı ve yarıda kalan çeviriler olurdu.
		hedefDil := strings.TrimSpace(r.FormValue("hedef_dil"))
		if hedefDil == "" {
			hedefDil = "tr"
		}
		if s.ai == nil || !s.ai.Enabled() {
			geri("error", "AI sağlayıcısı yapılandırılmadı (Ayarlar → Yapay Zeka).")
			return
		}
		atlandiDil := 0
		for _, id := range ids {
			c, err := repo.GetContent(ctx, s.pool, id)
			if err != nil {
				atlandi++
				continue
			}
			// Zaten hedef dildeyse çeviri anlamsız.
			if strings.EqualFold(c.Locale, hedefDil) {
				atlandiDil++
				continue
			}
			if err := ceviri.IstekEkle(ctx, s.pool, id, hedefDil); err != nil {
				atlandi++
				continue
			}
			basarili++
		}
		s.audit(r, "toplu_ceviri", hedefDil+": "+itoa(basarili)+" içerik")
		if atlandiDil > 0 {
			defer func() {
				s.logger.Info("toplu çeviri: zaten hedef dilde olanlar atlandı",
					"adet", atlandiDil, "dil", hedefDil)
			}()
		}
	case "erisim":
		// Toplu erişim kademesi (12 Kasım 2026): "herkese açık" ya da
		// belirli bir abonelik kademesi. Kilitler elle tek tek
		// işaretlendiği için aralarında rastgele kalanlar oluyordu —
		// SEO için açık tutulması gereken evergreen rehber bile
		// kilitlenmişti (ölçüldü).
		agirlik := 0
		if v := strings.TrimSpace(r.FormValue("kademe")); v != "" {
			agirlik = parseIntOr(v)
		}
		if agirlik < 0 || agirlik > 5 {
			geri("error", "Geçersiz abonelik kademesi.")
			return
		}
		for _, id := range ids {
			if err := repo.SetContentErisim(ctx, s.pool, id, agirlik); err != nil {
				atlandi++
				continue
			}
			basarili++
		}
		if agirlik == 0 {
			s.audit(r, "icerik_erisim", "toplu herkese acik: "+itoa(len(ids)))
		} else {
			s.audit(r, "icerik_erisim", "toplu kademe "+itoa(agirlik)+": "+itoa(len(ids)))
		}
	case "sil":
		for _, id := range ids {
			if err := repo.DeleteContent(ctx, s.pool, id); err != nil {
				atlandi++
				continue
			}
			s.publishContentLive(ctx, id, s.curSiteID(r), true)
			s.audit(r, "icerik_sil", "toplu id="+itoa(int(id)))
			basarili++
		}
	default:
		geri("error", "Bilinmeyen işlem.")
		return
	}

	if atlandi > 0 {
		s.audit(r, "icerik_toplu_"+islem, "basarili="+itoa(basarili)+" atlandi="+itoa(atlandi))
		geri("info", itoa(basarili)+" içerik işlendi, "+itoa(atlandi)+" içerik atlandı (yayın kapısı veya hata).")
		return
	}
	s.audit(r, "icerik_toplu_"+islem, "basarili="+itoa(basarili))
	eylem := "işlendi"
	if islem == "planla" {
		eylem = "planlandı"
	}
	if islem == "rozet" {
		eylem = "rozetlendi"
	}
	geri("ok", itoa(basarili)+" içerik "+eylem+".")
}

func (s *Server) handleContentAutosave(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := parseID(r.PathValue("id"))
	if id <= 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "hata": "geçersiz içerik"})
		return
	}
	if r.ContentLength > 4<<20 {
		s.writeJSON(w, http.StatusRequestEntityTooLarge, map[string]any{"ok": false, "hata": "içerik çok büyük"})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 6<<20)
	if err := r.ParseMultipartForm(4 << 20); err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "hata": "form okunamadı"})
		return
	}
	c, derr := repo.GetContent(ctx, s.pool, id)
	if derr != nil {
		s.writeJSON(w, http.StatusNotFound, map[string]any{"ok": false, "hata": "içerik bulunamadı"})
		return
	}
	if !canEditContent(userFromContext(ctx), c) {
		s.writeJSON(w, http.StatusForbidden, map[string]any{"ok": false, "hata": "bu içeriği düzenleme yetkiniz yok"})
		return
	}
	baslik := strings.TrimSpace(r.FormValue("baslik"))
	if baslik == "" {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "hata": "başlık boş"})
		return
	}
	version := parseID(r.FormValue("version"))
	detay := parseDetailInput(r.FormValue("detay"))
	newVersion, err := repo.SaveContentDraft(ctx, s.pool, id, version, baslik,
		strings.TrimSpace(r.FormValue("spot")), detay)
	if err != nil {
		s.logger.Error("otomatik taslak kaydetme", "error", err)
		if errors.Is(err, repo.ErrConflict) {
			s.writeJSON(w, http.StatusConflict, map[string]any{
				"ok": false, "cakisma": true, "hata": "İçerik başka biri tarafından değiştirildi; sayfayı yenileyin.",
				"surum": c.Version,
			})
			return
		}
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "hata": "kaydedilemedi"})
		return
	}
	s.publishEditorSaved(ctx, id, c.SiteID, newVersion)
	s.writeJSON(w, http.StatusOK, map[string]any{
		"ok": true, "zaman": time.Now().Format("15:04:05"), "surum": newVersion,
	})
}

// handleQuickStatus tekil yayınla / taslağa al / arşivle.
func (s *Server) handleQuickStatus(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	status := "yayinda"
	switch {
	case strings.HasSuffix(r.URL.Path, "/arsiv"):
		status = "arsiv"
	case strings.HasSuffix(r.URL.Path, "/taslak"):
		status = "taslak"
	}
	// HTMX'te hata da liste paneli içinde gösterilir; JSON API davranışı korunur.
	statusErr := func(msg string) {
		if IsHTMX(r) {
			s.renderContentsPanel(w, r, "error", msg)
			return
		}
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "hata": msg})
	}
	c, err := repo.GetContent(r.Context(), s.pool, id)
	if err != nil {
		statusErr("İçerik bulunamadı.")
		return
	}
	if !canEditContent(userFromContext(r.Context()), c) {
		statusErr("Bu içeriği düzenleme yetkiniz yok.")
		return
	}
	// kategori AI alaka kapısı (spec 06 K-A): alt eşiğin altındaki içerik yayınlanamaz
	if status == "yayinda" {
		if c.IsYMYL && c.YMYLDurumu != "uzman" {
			statusErr("YMYL — uzman denetimi gerekli.")
			return
		}
		if g := repo.CheckScoreGate(r.Context(), s.pool, c); !g.Allowed {
			statusErr("Yayın kapısı: " + g.Reason)
			return
		}
		// Çeviri kapısı (Faz 120): kaynaksız makine çevirisi yayına alınamaz.
		if g := s.translationGate(r.Context(), c); !g.Allowed {
			statusErr(g.Reason)
			return
		}

		if repo.IskeletGovde(c.Detail) {
			statusErr("Bu içerik yayına hazır değil: gövdesi AI kaynak iskeleti " +
				"içeriyor. AI sonucunu uygulayın ya da gövdeyi gerçek " +
				"haber metniyle değiştirin.")
			return
		}
	}
	if err := repo.SetContentStatus(r.Context(), s.pool, id, status); err != nil {
		if IsHTMX(r) {
			s.renderContentsPanel(w, r, "error", "Durum değiştirilemedi.")
			return
		}
		s.writeJSON(w, http.StatusNotFound, map[string]any{"ok": false})
		return
	}
	if status == "yayinda" {
		// otomatik sosyal paylaşım + hızlı indeksleme (P4-7 / P4-6)
		s.autoShare(r.Context(), s.curSiteID(r), id)
		s.autoIndex(r.Context(), s.curSiteID(r), id)
		// Faz 34: liste üzerinden yayınlanan içerik de Tantivy arama indeksine
		// girmeli (eskiden yalnız form kaydetme indeksliyor, o da eksikti).
		if c, err := repo.GetContent(r.Context(), s.pool, id); err == nil {
			c.Status = "yayinda"
			s.indexContent(r.Context(), &c)
		}
	}
	s.publishContentLive(r.Context(), id, s.curSiteID(r), false)
	s.publishEditorSaved(r.Context(), id, s.curSiteID(r), 0)
	if IsHTMX(r) {
		etiket := map[string]string{"yayinda": "İçerik yayınlandı.", "taslak": "İçerik taslağa alındı.", "arsiv": "İçerik arşivlendi."}[status]
		s.renderContentsPanel(w, r, "info", etiket)
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "href": "/admin/icerikler" + "?durum=" + status})
}

func (s *Server) handleContentDelete(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	c, err := repo.GetContent(r.Context(), s.pool, id)
	if err != nil {
		if IsHTMX(r) {
			s.renderContentsPanel(w, r, "error", "İçerik bulunamadı.")
			return
		}
		s.flashRedirect(w, r, "/admin/icerikler", "error", "İçerik bulunamadı.")
		return
	}
	if !canEditContent(userFromContext(r.Context()), c) {
		if IsHTMX(r) {
			s.renderContentsPanel(w, r, "error", "Bu içeriği silme yetkiniz yok.")
			return
		}
		s.flashRedirect(w, r, "/admin/icerikler", "error", "Bu içeriği silme yetkiniz yok.")
		return
	}
	s.audit(r, "icerik_sil", "id="+itoa(int(id)))
	if err := repo.DeleteContent(r.Context(), s.pool, id); err != nil {
		if IsHTMX(r) {
			s.renderContentsPanel(w, r, "error", "Silinemedi: "+err.Error())
			return
		}
		s.flashRedirect(w, r, "/admin/icerikler", "error", "Silinemedi: "+err.Error())
		return
	}
	s.publishContentLive(r.Context(), id, c.SiteID, true)
	if IsHTMX(r) {
		s.renderContentsPanel(w, r, "info", "İçerik silindi.")
		return
	}
	s.flashRedirect(w, r, "/admin/icerikler", "info", "İçerik silindi.")
}

func (s *Server) handleToggleRozet(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	code := strings.ToUpper(r.PathValue("code"))
	ctx := r.Context()
	c, err := repo.GetContent(ctx, s.pool, id)
	if err != nil {
		s.writeJSON(w, http.StatusNotFound, map[string]any{"ok": false})
		return
	}
	if !canEditContent(userFromContext(ctx), c) {
		s.writeJSON(w, http.StatusForbidden, map[string]any{"ok": false})
		return
	}
	// flip
	field := map[string]*bool{
		"AM": &c.IsAM, "UM": &c.IsUM, "DM": &c.IsDM, "SD": &c.IsSD, "FH": &c.IsFH, "HK": &c.IsHK,
	}[code]
	if field == nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false})
		return
	}
	*field = !*field
	if err := repo.UpdateContent(ctx, s.pool, &c); err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// ---- Çeviri oluştur ----

func (s *Server) handleCreateTranslation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := parseID(r.PathValue("id"))
	lang := r.PathValue("lang")
	src, err := repo.GetContent(ctx, s.pool, id)
	if err != nil {
		s.ceviriFlash(w, r, "/admin/icerikler", "error", "Kaynak içerik bulunamadı.")
		return
	}
	if !canEditContent(userFromContext(ctx), src) {
		s.ceviriFlash(w, r, "/admin/icerikler", "error", "Bu içerikten çeviri oluşturma yetkiniz yok.")
		return
	}
	// köke işaret et (zincir değil)
	root := src.ID
	if src.TranslationOf != nil {
		root = *src.TranslationOf
	}
	// aynı dilde çeviri varsa ona git
	if trs, err := repo.TranslationsOf(ctx, s.pool, root); err == nil {
		for _, t := range trs {
			if t.Locale == lang {
				s.ceviriYonlendir(w, r, "/admin/icerikler/"+itoa(int(t.ID)))
				return
			}
		}
	}

	if r.FormValue("ai") == "1" {
		s.ceviriSirayaAl(w, r, src, lang)
		return
	}

	c := src
	c.ID = 0
	c.Locale = lang
	c.TranslationOf = ptr(root)
	c.Status = "taslak"
	c.PublishedAt = nil
	c.PublishAt = nil
	c.Slug = optStr(platform.Slugify(c.Title) + "-" + lang)
	nid, err := repo.CreateContent(ctx, s.pool, &c)
	if err != nil {
		// Ham SQL metni kullanıcıya gösterilmez: aynı sınıf hata (FK ihlali)
		// kuyruk yolunda da ham metinle sızıyordu. Ayrıntı loga gider.
		s.logger.Warn("ceviri: boş taslak oluşturulamadı", "kaynak", id, "dil", lang, "error", err)
		s.ceviriFlash(w, r, "/admin/icerikler", "error", "Çeviri taslağı oluşturulamadı. Ayrıntı sunucu günlüğüne yazıldı.")
		return
	}
	// etiketleri kopyala
	if tags, err := repo.ContentTagNames(ctx, s.pool, id); err == nil {
		_ = repo.ReplaceContentTags(ctx, s.pool, nid, tags, platform.Slugify)
	}
	s.ceviriFlash(w, r, "/admin/icerikler/"+itoa(int(nid)), "info", lang+" taslağı oluşturuldu. Tercümeyi düzenleyip yayınlayın.")
}

func (s *Server) ceviriYonlendir(w http.ResponseWriter, r *http.Request, hedef string) {
	if IsHTMX(r) {
		w.Header().Set("HX-Redirect", hedef)
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, hedef, http.StatusFound)
}

func (s *Server) ceviriFlash(w http.ResponseWriter, r *http.Request, url, kind, text string) {
	s.flash(w, r, kind, text)
	s.ceviriYonlendir(w, r, url)
}

func (s *Server) ceviriSirayaAl(w http.ResponseWriter, r *http.Request, src repo.Content, lang string) {
	ctx := r.Context()
	kaynakURL := "/admin/icerikler/" + itoa(int(src.ID))

	if s.ai == nil || !s.ai.Enabled() {
		s.renderTemplPanel(w, r, templkit.CeviriHata(
			"AI sağlayıcısı yapılandırılmadı. Yönetim panelinden AI ayarlarını kontrol edin.", kaynakURL))
		return
	}
	if err := ceviri.IstekEkle(ctx, s.pool, src.ID, lang); err != nil {
		s.logger.Warn("ceviri: istek yazılamadı", "kaynak", src.ID, "dil", lang, "error", err)
		s.renderTemplPanel(w, r, templkit.CeviriHata(
			"Çeviri sıraya alınamadı. Ayrıntı sunucu günlüğüne yazıldı.", kaynakURL))
		return
	}
	s.logger.Info("ceviri: editör isteği sıraya alındı", "kaynak", src.ID, "dil", lang)
	s.renderTemplPanel(w, r, templkit.CeviriBekliyor(
		s.dilAdiTR(lang), kaynakURL+"/cevir/"+lang+"/durum", kaynakURL))
}

func (s *Server) handleTranslationStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := parseID(r.PathValue("id"))
	lang := r.PathValue("lang")
	if id <= 0 || lang == "" {
		http.NotFound(w, r)
		return
	}
	src, err := repo.GetContent(ctx, s.pool, id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	kaynakURL := "/admin/icerikler/" + itoa(int(src.ID))
	durumURL := kaynakURL + "/cevir/" + lang + "/durum"

	is, varMi, err := ceviri.IstekDurum(ctx, s.pool, id, lang)
	if err != nil {
		s.logger.Warn("ceviri: istek durumu okunamadı", "kaynak", id, "dil", lang, "error", err)
		s.renderTemplPanel(w, r, templkit.CeviriHata(
			"Çeviri durumu okunamadı. Sayfayı yenileyip yeniden deneyin.", kaynakURL))
		return
	}
	if !varMi {
		// İstek satırı yok: çeviri bu arada oluşmuş olabilir (başka sekme,
		// arka plan kuyruğu). Varsa aç, yoksa dürüst hata ver.
		if hedef := s.ceviriVarMi(ctx, src, lang); hedef != "" {
			s.ceviriYonlendir(w, r, hedef)
			return
		}
		s.renderTemplPanel(w, r, templkit.CeviriHata(
			"Çeviri isteği bulunamadı. Düğmeye yeniden basın.", kaynakURL))
		return
	}

	switch is.Durum {
	case ceviri.IstekBitti:
		if is.SonucID > 0 {
			s.ceviriYonlendir(w, r, "/admin/icerikler/"+itoa(int(is.SonucID)))
			return
		}
		if hedef := s.ceviriVarMi(ctx, src, lang); hedef != "" {
			s.ceviriYonlendir(w, r, hedef)
			return
		}
		s.renderTemplPanel(w, r, templkit.CeviriHata(
			"Çeviri tamamlandı ancak kaydı bulunamadı. İçerik listesinden kontrol edin.", kaynakURL))
	case ceviri.IstekHata:
		mesaj := is.Hata
		if mesaj == "" {
			mesaj = "Çeviri üretilemedi. Ayrıntı sunucu günlüğüne yazıldı."
		}
		s.renderTemplPanel(w, r, templkit.CeviriHata(mesaj, kaynakURL))
	default:
		// Kuyruk 15 dakikadır hiç ilerlemediyse sonsuz yoklama gösterme.
		if is.Durum == ceviri.IstekBekliyor && time.Since(is.UpdatedAt) > 15*time.Minute {
			s.renderTemplPanel(w, r, templkit.CeviriHata(
				"Çeviri sırası ilerlemedi (arka plan işçisi çalışmıyor olabilir). "+
					"Sunucu günlüğünü kontrol edip yeniden deneyin.", kaynakURL))
			return
		}
		s.renderTemplPanel(w, r, templkit.CeviriBekliyor(
			s.dilAdiTR(lang), durumURL, kaynakURL))
	}
}

// ceviriVarMi içeriğin verilen dilde çevirisi varsa editör adresini döndürür.
func (s *Server) ceviriVarMi(ctx context.Context, src repo.Content, lang string) string {
	trs, err := repo.TranslationsOf(ctx, s.pool, src.TranslationRootID())
	if err != nil {
		return ""
	}
	for _, t := range trs {
		if t.Locale == lang && t.ID != src.ID {
			return "/admin/icerikler/" + itoa(int(t.ID))
		}
	}
	return ""
}

// ---- AI eylemi ----

func (s *Server) handleAIAction(w http.ResponseWriter, r *http.Request) {
	u := userFromContext(r.Context())

	// HTMX: form alanları + HX-Trigger ile sonucu hedef alana yaz.
	if IsHTMX(r) {
		_ = r.ParseForm()
		action := strings.TrimSpace(r.FormValue("action"))
		contentID := parseID(r.FormValue("content_id"))
		input := strings.TrimSpace(r.FormValue("input"))
		if input == "" {
			for _, key := range []string{"baslik", "spot", "detay"} {
				if v := strings.TrimSpace(r.FormValue(key)); v != "" {
					input = v
					break
				}
			}
		}
		target := strings.TrimSpace(r.FormValue("target"))
		kind := strings.TrimPrefix(action, "ai_")
		result, err := s.runAIAction(r, kind, input, contentID, u.ID)
		payload := map[string]any{"ok": true, "action": kind, "target": target, "result": result}
		if err != nil {
			payload["result"] = mockResult(kind, input)
			payload["note"] = err.Error()
		}
		s.writeHTMXTrigger(w, "aiResult", payload)
		return
	}

	var req struct {
		Action    string `json:"action"`
		ContentID int64  `json:"content_id"`
		Input     string `json:"input"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "geçersiz istek"})
		return
	}
	kind := strings.TrimPrefix(req.Action, "ai_")
	result, err := s.runAIAction(r, kind, req.Input, req.ContentID, u.ID)
	if err != nil {
		s.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "result": mockResult(kind, req.Input), "note": err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "result": result})
}

func (s *Server) runAIAction(r *http.Request, kind, input string, contentID, userID int64) (string, error) {
	prompt := userPrompt(kind, input)
	name := "mock"
	if s.ai.Enabled() {
		name = s.ai.Providers()[0]
	}
	out, err := s.ai.Chat(r.Context(), []ai.Msg{
		{Role: ai.RoleSystem, Content: systemPrompt()},
		{Role: ai.RoleUser, Content: prompt},
	})
	if err != nil {
		return "", err
	}
	out = normalizeAIString(out)
	// kullanım logu
	pt := len(input) / 4
	ct := len(out) / 4
	if _, lerr := repo.LogAIUsage(r.Context(), s.pool, &repo.AIUsage{
		UserID: ptr(userID), ContentID: optID64(contentID), Provider: name, Action: kind,
		PromptTokens: pt, CompletionTokens: ct,
	}); lerr != nil {
		s.logger.Warn("ai kullanım logu", "error", lerr)
	}
	return out, nil
}

// mockResult gerçek sağlayıcı yokken istemciye zarif değer döner.
func mockResult(kind, input string) string {
	switch kind {
	case "title":
		return input + " — Güncel Başlık"
	case "spot":
		return input + " hakkındaki haberde tüm detaylar masaya yatırıldı."
	case "seo_title":
		return input + " | theblock.com.tr"
	case "seo_description":
		return input + " ile ilgili en güncel gelişmeler theblock.com.tr'de."
	case "rewrite":
		return input
	case "summarize":
		return input
	case "trend_brief":
		return input + " — gündemde yükselen konu. Editör notu: site içeriğinde henüz yeterli kapsama yok; haber başlığını doğrulayıp kısa ve kaynaklı bir taslak hazırlayın."
	default:
		return input
	}
}

func systemPrompt() string {
	return `Sen Türkçe yayın editör asistanısın. Talimat:
- Yanıtların tamamı Türkçe olmalı; İngilizce sözcük, başlık veya açıklama kullanma.
- Türkçe karakterleri (ç, ğ, ı, İ, ö, ş, ü) her zaman doğru yaz; harfleri ASCII'ye (c, g, i, o, s, u) çevirme.
- title: tek cümlelik, dikkat çekici haber başlıkları; YALNIZCA JSON dizi (ör. ["..."]).
- spot: 120-160 karakterlik haber özeti; düz metin (JSON veya tırnak yok).
- seo_title: 40-70 karakter SEO başlığı; düz metin.
- seo_description: 120-160 karakter meta açıklaması; düz metin.
- rewrite: metni Türkçe imla ve karakterleri koruyarak özgünleştir, kısaltma/Yİ.
- summarize: metni 3 maddeyle Türkçe özetle.
- trend_brief: Google Trends terimi ve haber başlıklarından editöre 3-5 maddelik kısa brifing üret; başlık önerisi, açı ve kaynak uyarısı ver.
Asla uydurma bilgi ekleme; kaynak metne sadık kal. Yanıtın dışına açıklama, kod bloğu veya etiket ekleme.`
}

func userPrompt(kind, input string) string {
	switch kind {
	case "title":
		return "JSON:title " + input
	case "spot":
		return "JSON:spot " + input
	case "seo_title":
		return "JSON:seo_title " + input
	case "seo_description":
		return "JSON:seo_description " + input
	case "rewrite":
		return "JSON:rewrite " + input
	case "summarize":
		return "JSON:summarize " + input
	case "trend_brief":
		return "JSON:trend_brief " + input
	default:
		return "JSON:" + input
	}
}

func normalizeAIString(s string) string {
	s = stripCodeFence(strings.TrimSpace(s))
	if s == "" {
		return ""
	}
	if strings.HasPrefix(s, "[") {
		var list []string
		if json.Unmarshal([]byte(s), &list) == nil {
			for _, item := range list {
				if strings.TrimSpace(item) != "" {
					return item
				}
			}
		}
		var anyList []any
		if json.Unmarshal([]byte(s), &anyList) == nil {
			for _, item := range anyList {
				if str, ok := item.(string); ok && strings.TrimSpace(str) != "" {
					return str
				}
			}
		}
	}
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		var str string
		if json.Unmarshal([]byte(s), &str) == nil {
			return str
		}
		return s[1 : len(s)-1]
	}
	// Tırnaksız düz metinde de \uXXXX / \n kaçışları çözülsün.
	if strings.Contains(s, `\`) && !strings.Contains(s, `"`) {
		var str string
		if json.Unmarshal([]byte(`"`+s+`"`), &str) == nil {
			return str
		}
	}
	return s
}

// stripCodeFence, modelin döndürdüğü \`\`\`json ... \`\`\` sarmalını kaldırır.
func stripCodeFence(s string) string {
	if !strings.HasPrefix(s, "```") {
		return s
	}
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[i+1:]
	}
	if i := strings.LastIndex(s, "```"); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}

// ---- İçerik türleri yardımcıdan: kategori CRUD ----

func (s *Server) handleCatsGet(w http.ResponseWriter, r *http.Request) {
	d := s.buildCatsData(r)
	// Panel v2: kategoriler templ bileşenleriyle render edilir.
	if IsHTMX(r) {
		s.renderKategorilerPanel(w, r, d)
		return
	}
	s.renderKategorilerPage(w, r, d)
}

func (s *Server) buildCatsData(r *http.Request) *tmplData {
	d := s.baseView("tr", "/admin", "Kategoriler")
	d.User = ptr(userFromContext(r.Context()))
	d.Flash = s.consumeFlash(r)
	items, _ := repo.ListCategories(r.Context(), s.pool, s.curSiteID(r))
	custom := map[string]bool{}
	if keys, err := repo.ListLayoutKeys(r.Context(), s.pool, s.curSiteID(r), "kategori:"); err == nil {
		for _, k := range keys {
			custom[strings.TrimPrefix(k, "kategori:")] = true
		}
	}
	for i := range items {
		items[i].CustomDesign = custom[items[i].Slug]
		if items[i].LayoutPreset != "" {
			for _, p := range categoryPresets {
				if p.Key == items[i].LayoutPreset {
					items[i].DesignName = p.Name
					break
				}
			}
		}
	}
	var parents []repo.Category
	for _, c := range items {
		if c.ParentID == nil {
			parents = append(parents, c)
		}
	}
	d.Cats = &catsData{Items: items, Parents: parents, Presets: categoryPresets}
	return d
}

func (s *Server) renderCatsPanel(w http.ResponseWriter, r *http.Request, kind, text string) {
	d := s.buildCatsData(r)
	if kind != "" {
		d.Flash = &flashMsg{Kind: kind, Text: text}
	}
	s.renderKategorilerPanel(w, r, d)
}

func (s *Server) handleCatsCreate(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.FormValue("ad"))
	if name == "" {
		if IsHTMX(r) {
			s.renderCatsPanel(w, r, "error", "Ad gerekli.")
			return
		}
		s.flashRedirect(w, r, "/admin/kategoriler", "error", "Ad gerekli.")
		return
	}
	c := &repo.Category{Name: name, Slug: platform.Slugify(name), Status: "yayinda", SiteID: s.curSiteID(r)}
	c.InHeader = r.FormValue("header") != ""
	c.InMenu = r.FormValue("menu") != ""
	c.InFooter = r.FormValue("footer") != ""
	c.ListView = strings.TrimSpace(r.FormValue("gorunum"))
	if !validCategoryView(c.ListView) {
		c.ListView = ""
	}
	applyCategoryGateFields(r, c)
	s.applyCategoryParent(r, c)
	if _, err := repo.CreateCategory(r.Context(), s.pool, c); err != nil {
		if IsHTMX(r) {
			s.renderCatsPanel(w, r, "error", "Ekleme hatası: "+err.Error())
			return
		}
		s.flashRedirect(w, r, "/admin/kategoriler", "error", "Ekleme hatası: "+err.Error())
		return
	}
	s.invalidateContentCache()
	if IsHTMX(r) {
		s.renderCatsPanel(w, r, "info", "Kategori eklendi.")
		return
	}
	s.flashRedirect(w, r, "/admin/kategoriler", "info", "Kategori eklendi.")
}

func (s *Server) categoryByID(ctx context.Context, siteID, id int64) (repo.Category, error) {
	cats, err := repo.ListCategories(ctx, s.pool, siteID)
	if err != nil {
		return repo.Category{}, err
	}
	for _, c := range cats {
		if c.ID == id {
			return c, nil
		}
	}
	return repo.Category{}, errors.New("kategori bulunamadı")
}

func (s *Server) applyCategoryPreset(ctx context.Context, siteID int64, cat *repo.Category, p categoryPreset) error {
	key := "kategori:" + cat.Slug
	if p.Blocks == "" {
		if err := repo.DeleteLayout(ctx, s.pool, siteID, key); err != nil {
			return err
		}
	} else if err := repo.SaveLayout(ctx, s.pool, siteID, key, []byte(p.Blocks)); err != nil {
		return err
	}
	return repo.SetCategoryDesign(ctx, s.pool, cat.ID, p.Key, p.View)
}

func (s *Server) handleCatsUpdate(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	ctx := r.Context()
	// Mevcut slug'ı koru: formda slug alanı yok; boş slug ile güncelleme
	// kategori bağlantılarını bozardı.
	existing, err := s.categoryByID(ctx, s.curSiteID(r), id)
	if err != nil {
		if IsHTMX(r) {
			s.renderCatsPanel(w, r, "error", "Kategori bulunamadı.")
			return
		}
		s.flashRedirect(w, r, "/admin/kategoriler", "error", "Kategori bulunamadı.")
		return
	}
	c := &repo.Category{ID: id, Name: strings.TrimSpace(r.FormValue("ad")), Slug: existing.Slug, SortOrder: int(parseID(r.FormValue("sira")))}
	c.InHeader = r.FormValue("header") != ""
	c.InMenu = r.FormValue("menu") != ""
	c.InFooter = r.FormValue("footer") != ""
	// Formda olmayan alanlar korunur: aksi halde her kayıt özel URL'yi,
	// hedefi ve yayın durumunu sessizce sıfırlardı.
	c.CustomURL = existing.CustomURL
	c.OpenTarget = existing.OpenTarget
	c.Status = existing.Status
	c.MinTierWeight = existing.MinTierWeight
	c.ListView = strings.TrimSpace(r.FormValue("gorunum"))
	if !validCategoryView(c.ListView) {
		c.ListView = ""
	}
	applyCategoryGateFields(r, c)
	s.applyCategoryParent(r, c)
	if c.Name == "" {
		if IsHTMX(r) {
			s.renderCatsPanel(w, r, "error", "Ad gerekli.")
			return
		}
		s.flashRedirect(w, r, "/admin/kategoriler", "error", "Ad gerekli.")
		return
	}
	if err := repo.UpdateCategory(ctx, s.pool, c); err != nil {
		if IsHTMX(r) {
			s.renderCatsPanel(w, r, "error", "Güncelleme hatası")
			return
		}
		s.flashRedirect(w, r, "/admin/kategoriler", "error", "Güncelleme hatası")
		return
	}
	msg := "Güncellendi."
	if r.FormValue("sablon_uygula") != "" {
		key := strings.TrimSpace(r.FormValue("sablon"))
		p, ok := categoryPresetByKey(key)
		if !ok {
			if IsHTMX(r) {
				s.renderCatsPanel(w, r, "error", "Tasarım şablonu geçersiz.")
				return
			}
			s.flashRedirect(w, r, "/admin/kategoriler", "error", "Tasarım şablonu geçersiz.")
			return
		}
		if err := s.applyCategoryPreset(ctx, s.curSiteID(r), c, p); err != nil {
			if IsHTMX(r) {
				s.renderCatsPanel(w, r, "error", "Şablon uygulanamadı: "+err.Error())
				return
			}
			s.flashRedirect(w, r, "/admin/kategoriler", "error", "Şablon uygulanamadı.")
			return
		}
		msg = "Güncellendi · şablon uygulandı: " + p.Name
	}
	s.invalidateContentCache()
	if IsHTMX(r) {
		s.renderCatsPanel(w, r, "info", msg)
		return
	}
	s.flashRedirect(w, r, "/admin/kategoriler", "info", msg)
}

func (s *Server) applyCategoryParent(r *http.Request, c *repo.Category) {
	c.ParentID = nil
	raw := strings.TrimSpace(r.FormValue("ust"))
	if raw == "" || raw == "0" {
		return
	}
	id := parseID(raw)
	if id == 0 || (c.ID != 0 && id == c.ID) {
		return
	}
	var isTop bool
	err := s.pool.QueryRow(r.Context(), `
		SELECT parent_id IS NULL FROM categories WHERE id = $1 AND site_id = $2`,
		id, s.curSiteID(r)).Scan(&isTop)
	if err != nil || !isTop {
		return
	}
	// Alt kategorisi olan bir kategori başka kategoriye bağlanamaz (tek seviye).
	if c.ID != 0 {
		var ownChildren int
		if err := s.pool.QueryRow(r.Context(), `
			SELECT count(*) FROM categories WHERE parent_id = $1`, c.ID).Scan(&ownChildren); err != nil || ownChildren > 0 {
			return
		}
	}
	c.ParentID = &id
}

// maybeAutoCover kategori görsel hizmeti tanımlıysa ve içerikte kapak görseli yoksa
// otomatik kapak üretir (Pexels ya da marka üretimi) ve içeriğe bağlar. spec 06 K-D
func (s *Server) maybeAutoCover(ctx context.Context, c *repo.Content) {
	if c.CategoryID == nil {
		return
	}
	var service *string
	var catName string
	err := s.pool.QueryRow(ctx, `SELECT image_service, name FROM categories WHERE id=$1`, *c.CategoryID).Scan(&service, &catName)
	if err != nil || service == nil || *service == "" {
		return
	}
	svc := *service
	if svc != "pexels" && svc != "generate" && svc != "otomatik" {
		return
	}
	query := catName + " " + c.Title
	if svc == "pexels" {
		query = catName // pexels'te kategori bazlı arama daha isabetli
	}
	data, source, err := platform.AutoCover(platform.AutoCoverSpec{
		Title: c.Title, Category: catName, Slug: c.DisplaySlug(), Query: query,
	})
	if err != nil {
		s.logger.Warn("otomatik kapak üretilemedi", "content", c.ID, "error", err)
		return
	}
	path, err := platform.SaveMedia(data, "kapak.jpg")
	if err != nil {
		s.logger.Warn("otomatik kapak kaydedilemedi", "error", err)
		return
	}
	mid, err := repo.CreateMedia(ctx, s.pool, &repo.Media{
		UserID: userFromContext(ctx).ID, Kind: "image", Filename: c.DisplaySlug() + ".jpg",
		Path: path, Mime: "image/jpeg", SizeBytes: ptr(int64(len(data))),
		Width: ptr(1200), Height: ptr(675), Alt: &c.Title, Source: source,
	})
	if err != nil {
		s.logger.Warn("otomatik kapak kaydı oluşturulamadı", "error", err)
		return
	}
	_ = repo.SetContentCoverMedia(ctx, s.pool, c.ID, mid)
	c.CoverMediaID = &mid
	s.logger.Info("otomatik kapak üretildi", "content", c.ID, "source", source, "media", mid)
}

// applyCategoryGateFields kategori formundaki skor kapısı/ikon/görsel alanlarını okur (spec 06 K-A).
func applyCategoryGateFields(r *http.Request, c *repo.Category) {
	c.MinScore = int(parseID(r.FormValue("min_skor")))
	c.HomepageMinScore = int(parseID(r.FormValue("ana_skor")))
	if c.MinScore < 0 || c.MinScore > 100 {
		c.MinScore = 0
	}
	if c.HomepageMinScore < 0 || c.HomepageMinScore > 100 {
		c.HomepageMinScore = 0
	}
	if vals, ok := r.Form["kategori_katman"]; ok && len(vals) > 0 {
		if n := int(parseID(vals[0])); n >= 0 && n <= 5 {
			c.MinTierWeight = n
		}
	}
	if ic := strings.TrimSpace(r.FormValue("ikon")); ic != "" {
		c.Icon = ic
	}
	if c.Icon == "" {
		c.Icon = "news"
	}
	c.ImageService = optStr(strings.TrimSpace(r.FormValue("gorsel_servis")))
	c.NameEN = optStr(strings.TrimSpace(r.FormValue("ad_en")))
	c.NameES = optStr(strings.TrimSpace(r.FormValue("ad_es")))
	c.Label = optStr(strings.TrimSpace(r.FormValue("etiket")))
}

func (s *Server) handleCatsDelete(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	if err := repo.DeleteCategory(r.Context(), s.pool, id); err != nil {
		if IsHTMX(r) {
			s.renderCatsPanel(w, r, "error", "Kategori silinemez (içerik bağlı olabilir).")
			return
		}
		s.flashRedirect(w, r, "/admin/kategoriler", "error", "Kategori silinemez (içerik bağlı olabilir).")
		return
	}
	s.invalidateContentCache()
	if IsHTMX(r) {
		s.renderCatsPanel(w, r, "info", "Silindi.")
		return
	}
	s.flashRedirect(w, r, "/admin/kategoriler", "info", "Silindi.")
}

// ---- Mahreç CRUD ----

func (s *Server) handleMahretsGet(w http.ResponseWriter, r *http.Request) {
	d := s.buildMahretsData(r)
	comp := templkit.MahreclerPanel(mahreclerViewFrom(d.Mahrets, d.Flash))
	if IsHTMX(r) {
		s.renderTemplPanel(w, r, comp)
		return
	}
	s.renderTemplPage(w, r, d, comp)
}

func (s *Server) buildMahretsData(r *http.Request) *tmplData {
	d := s.baseView("tr", "/admin", "Mahreçler")
	d.User = ptr(userFromContext(r.Context()))
	d.Flash = s.consumeFlash(r)
	items, _ := repo.ListMahrets(r.Context(), s.pool)
	d.Mahrets = &mahretsData{Items: items}
	return d
}

func (s *Server) renderMahretsPanel(w http.ResponseWriter, r *http.Request, kind, text string) {
	d := s.buildMahretsData(r)
	if kind != "" {
		d.Flash = &flashMsg{Kind: kind, Text: text}
	}
	s.renderTemplPanel(w, r, templkit.MahreclerPanel(mahreclerViewFrom(d.Mahrets, d.Flash)))
}

func (s *Server) handleMahretsCreate(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.FormValue("ad"))
	if name == "" {
		if IsHTMX(r) {
			s.renderMahretsPanel(w, r, "error", "Ad gerekli.")
			return
		}
		s.flashRedirect(w, r, "/admin/mahrecler", "error", "Ad gerekli.")
		return
	}
	if _, err := repo.CreateMahret(r.Context(), s.pool, name, platform.Slugify(name)); err != nil {
		if IsHTMX(r) {
			s.renderMahretsPanel(w, r, "error", "Ekleme hatası (çakışma olabilir).")
			return
		}
		s.flashRedirect(w, r, "/admin/mahrecler", "error", "Ekleme hatası (çakışma olabilir).")
		return
	}
	if IsHTMX(r) {
		s.renderMahretsPanel(w, r, "info", "Mahreç eklendi.")
		return
	}
	s.flashRedirect(w, r, "/admin/mahrecler", "info", "Mahreç eklendi.")
}

func (s *Server) handleMahretsDelete(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	if err := repo.DeleteMahret(r.Context(), s.pool, id); err != nil {
		if IsHTMX(r) {
			s.renderMahretsPanel(w, r, "error", "Mahreç silinemedi.")
			return
		}
		s.flashRedirect(w, r, "/admin/mahrecler", "error", "Mahreç silinemedi.")
		return
	}
	if IsHTMX(r) {
		s.renderMahretsPanel(w, r, "info", "Silindi.")
		return
	}
	s.flashRedirect(w, r, "/admin/mahrecler", "info", "Silindi.")
}

// ---- Yorum yönetimi (manuel onay) ----

func (s *Server) handleCommentsGet(w http.ResponseWriter, r *http.Request) {
	d := s.buildCommentsData(r)
	comp := templkit.YorumlarPanel(yorumlarViewFrom(d.Yorumlar, d.Flash))
	if IsHTMX(r) {
		s.renderTemplPanel(w, r, comp)
		return
	}
	s.renderTemplPage(w, r, d, comp)
}

func (s *Server) buildCommentsData(r *http.Request) *tmplData {
	ctx := r.Context()
	tab := r.URL.Query().Get("durum")
	if tab == "" {
		tab = "onay_bekliyor"
	}
	sid := s.curSiteID(r)
	ara := strings.TrimSpace(r.URL.Query().Get("q"))
	items, err := repo.ListCommentsAra(ctx, s.pool, tab, sid, ara)
	if err != nil {
		items = nil
	}
	pending, _ := repo.CountPendingComments(ctx, s.pool, sid)

	d := s.baseView("tr", "/admin", "Yorumlar")
	d.User = ptr(userFromContext(ctx))
	d.Flash = s.consumeFlash(r)
	d.Yorumlar = &commentsData{Items: items, Tab: tab, Pending: pending, Ara: ara}
	return d
}

func (s *Server) renderCommentsPanel(w http.ResponseWriter, r *http.Request, kind, text string) {
	d := s.buildCommentsData(withHTMXCurrentQuery(r))
	if kind != "" {
		d.Flash = &flashMsg{Kind: kind, Text: text}
	}
	s.renderTemplPanel(w, r, templkit.YorumlarPanel(yorumlarViewFrom(d.Yorumlar, d.Flash)))
}

func (s *Server) handleCommentApprove(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	s.audit(r, "yorum_onay", "id="+itoa(int(id)))
	if err := repo.ApprComment(r.Context(), s.pool, id); err != nil {
		if IsHTMX(r) {
			s.renderCommentsPanel(w, r, "error", "Onaylanamadı.")
			return
		}
		s.flashRedirect(w, r, "/admin/yorumlar?durum=onay_bekliyor", "error", "Onaylanamadı.")
		return
	}
	if IsHTMX(r) {
		s.renderCommentsPanel(w, r, "info", "Yorum onaylandı.")
		return
	}
	s.flashRedirect(w, r, "/admin/yorumlar?durum=onay_bekliyor", "info", "Yorum onaylandı.")
}

func (s *Server) handleCommentDelete(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	if err := repo.DeleteComment(r.Context(), s.pool, id); err != nil {
		if IsHTMX(r) {
			s.renderCommentsPanel(w, r, "error", "Silinemedi.")
			return
		}
		s.flashRedirect(w, r, "/admin/yorumlar", "error", "Silinemedi.")
		return
	}
	if IsHTMX(r) {
		s.renderCommentsPanel(w, r, "info", "Yorum silindi.")
		return
	}
	s.flashRedirect(w, r, "/admin/yorumlar?durum=onay_bekliyor", "info", "Yorum silindi.")
}

// handleCommentsBulk yorum moderasyonunda toplu işlem (Faz 33):
// `islem` = `onayla` | `sil`. Geçerli sekmeye geri döner (`durum` korunur).
func (s *Server) handleCommentsBulk(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_ = r.ParseForm()
	islem := strings.TrimSpace(r.FormValue("islem"))
	tab := r.FormValue("durum")
	if tab == "" {
		tab = "onay_bekliyor"
	}
	geri := func(kind, msg string) {
		if IsHTMX(r) {
			s.renderCommentsPanel(w, r, kind, msg)
			return
		}
		s.flashRedirect(w, r, "/admin/yorumlar?durum="+tab, kind, msg)
	}

	var ids []int64
	for _, v := range r.Form["id"] {
		if p := optID(v); p != nil && *p > 0 {
			ids = append(ids, *p)
		}
	}
	if len(ids) == 0 {
		geri("error", "Hiç yorum seçilmedi.")
		return
	}
	if len(ids) > 300 {
		ids = ids[:300]
	}

	var n int64
	var err error
	switch islem {
	case "onayla":
		n, err = repo.ApproveComments(ctx, s.pool, ids)
	case "sil":
		n, err = repo.DeleteComments(ctx, s.pool, ids)
	default:
		geri("error", "Bilinmeyen işlem.")
		return
	}
	if err != nil {
		s.logger.Error("yorum toplu", "error", err)
		geri("error", "İşlem başarısız.")
		return
	}
	s.audit(r, "yorum_toplu_"+islem, "adet="+itoa(int(n)))
	eylem := "işlendi"
	if islem == "onayla" {
		eylem = "onaylandı"
	} else if islem == "sil" {
		eylem = "silindi"
	}
	geri("ok", itoa(int(n))+" yorum "+eylem+".")
}

// ---- Medya yöneticisi ----

func (s *Server) handleMediaGet(w http.ResponseWriter, r *http.Request) {
	d := s.buildMediaData(r)
	// Sonsuz kaydırma dilimi: yalnız gruplar + yeni gözcü.
	if r.URL.Query().Get("parca") == "1" {
		s.renderMedyaParca(w, r, d)
		return
	}
	// Panel v2: medya paneli templ bileşenleriyle render edilir.
	if IsHTMX(r) {
		s.renderMedyaPanel(w, r, d)
		return
	}
	s.renderMedyaPage(w, r, d)
}

// buildMediaData medya panforumun ortak verisini kurar (tam sayfa + partial).
func (s *Server) buildMediaData(r *http.Request) *tmplData {
	d := s.baseView("tr", "/admin", "Medya")
	d.User = ptr(userFromContext(r.Context()))
	d.Flash = s.consumeFlash(r)
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	category := strings.TrimSpace(r.URL.Query().Get("kategori"))
	year, _ := strconv.Atoi(r.URL.Query().Get("yil"))
	month, _ := strconv.Atoi(r.URL.Query().Get("ay"))
	siteID := s.curSiteID(r)
	page, _ := strconv.Atoi(r.URL.Query().Get("sayfa"))
	if page < 1 {
		page = 1
	}
	// Sonsuz kaydırma: ilk sayfa ekranı dolduracak kadar büyük, mobilde
	// gereğinden ağır olmayacak kadar küçük.
	const perPage = 48
	filter := repo.MediaFilter{
		Query: q, Category: category, Year: year, Month: month,
	}
	total, _ := repo.CountMediaLibrary(r.Context(), s.pool, siteID, filter)
	pages := (total + perPage - 1) / perPage
	if pages < 1 {
		pages = 1
	}
	if page > pages {
		page = pages
	}
	filter.Limit = perPage
	filter.Offset = (page - 1) * perPage
	items, _ := repo.ListMediaLibrary(r.Context(), s.pool, siteID, filter)
	// Chip sayaçları aktif arama/tarih filtresiyle hesaplanır (kategori
	// hariç); chip'te yazan sayı ile tıklanınca gelen sonuç aynıdır.
	categories, _ := repo.MediaCategoryCounts(r.Context(), s.pool, siteID, filter)
	periods, _ := repo.ListMediaPeriods(r.Context(), s.pool, siteID)
	storageBytes, _ := repo.MediaStorageBytes(r.Context(), s.pool, siteID)
	md := &mediaData{
		Items: items, Categories: categories, Periods: periods,
		Query: q, Category: category, Year: r.URL.Query().Get("yil"), Month: r.URL.Query().Get("ay"),
		Page: page, Pages: pages, PerPage: perPage, Total: total, Storage: humanBytes(storageBytes),
		HasPrev: page > 1, HasNext: page < pages,
	}
	seenYears := map[int]bool{}
	for _, p := range periods {
		if !seenYears[p.Year] {
			seenYears[p.Year] = true
			md.Years = append(md.Years, p.Year)
		}
	}
	groupIndex := map[int]int{}
	for _, item := range items {
		key := item.CreatedAt.Year()*100 + int(item.CreatedAt.Month())
		idx, ok := groupIndex[key]
		if !ok {
			idx = len(md.Groups)
			groupIndex[key] = idx
			md.Groups = append(md.Groups, mediaGroup{
				Label: mediaMonthLabel(item.CreatedAt.Year(), int(item.CreatedAt.Month())),
				Year:  item.CreatedAt.Year(), Month: int(item.CreatedAt.Month()),
			})
		}
		md.Groups[idx].Items = append(md.Groups[idx].Items, item)
	}
	d.Media = md
	return d
}

// renderMediaPanel HTMX aksiyonlarından sonra medya panelini yeniden basar.
func (s *Server) renderMediaPanel(w http.ResponseWriter, r *http.Request, kind, text string) {
	d := s.buildMediaData(r)
	if kind != "" {
		d.Flash = &flashMsg{Kind: kind, Text: text}
	}
	s.renderMedyaPanel(w, r, d)
}

type mediaUploadResult struct {
	ID       int64
	URL      string
	Thumb    string
	Filename string
	Title    string
}

func (s *Server) handleMediaUpload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	u := userFromContext(ctx)
	jsonOut := r.URL.Query().Get("json") == "1"
	mediaErr := func(msg string) {
		if jsonOut {
			s.writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": msg})
			return
		}
		if IsHTMX(r) {
			s.renderMediaPanel(w, r, "error", msg)
			return
		}
		s.flashRedirect(w, r, "/admin/dosyalar", "error", msg)
	}
	mediaOk := func(msg string) {
		if IsHTMX(r) {
			s.renderMediaPanel(w, r, "info", msg)
			return
		}
		s.flashRedirect(w, r, "/admin/dosyalar", "info", msg)
	}
	r.Body = http.MaxBytesReader(w, r.Body, 200<<20)
	if err := r.ParseMultipartForm(210 << 20); err != nil {
		mediaErr("Dosya çok büyük (>200MB toplam).")
		return
	}
	files := r.MultipartForm.File["dosya"]
	if len(files) == 0 {
		mediaErr("Dosya seçilmedi.")
		return
	}
	category := strings.TrimSpace(r.FormValue("kategori"))
	if category == "" {
		category = "genel"
	}
	alt := strings.TrimSpace(r.FormValue("alt"))
	caption := strings.TrimSpace(r.FormValue("caption"))
	description := strings.TrimSpace(r.FormValue("aciklama"))
	formTitle := strings.TrimSpace(r.FormValue("baslik"))
	results := make([]mediaUploadResult, 0, len(files))
	var lastErr error
	for _, hdr := range files {
		title := formTitle
		if len(files) > 1 || title == "" {
			title = hdr.Filename
		}
		res, err := s.storeUploadedMedia(ctx, r, u.ID, hdr, title, category, alt, caption, description)
		if err != nil {
			lastErr = err
			continue
		}
		results = append(results, res)
	}
	if len(results) == 0 {
		msg := "Medya yüklenemedi."
		if lastErr != nil {
			msg = lastErr.Error()
		}
		mediaErr(msg)
		return
	}
	if jsonOut {
		if len(results) == 1 {
			res := results[0]
			s.writeJSON(w, http.StatusOK, map[string]any{
				"ok": true, "id": res.ID, "url": res.URL, "thumb": res.Thumb,
				"filename": res.Filename, "title": res.Title,
			})
			return
		}
		items := make([]map[string]any, 0, len(results))
		for _, res := range results {
			items = append(items, map[string]any{
				"id": res.ID, "url": res.URL, "thumb": res.Thumb,
				"filename": res.Filename, "title": res.Title,
			})
		}
		s.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "items": items, "count": len(items)})
		return
	}
	msg := strconv.Itoa(len(results)) + " medya yüklendi (önizleme üretildi)."
	if len(results) < len(files) {
		msg += " " + strconv.Itoa(len(files)-len(results)) + " dosya atlandı."
	}
	mediaOk(msg)
}

func (s *Server) storeUploadedMedia(ctx context.Context, r *http.Request, userID int64, hdr *multipart.FileHeader, title, category, alt, caption, description string) (mediaUploadResult, error) {
	f, err := hdr.Open()
	if err != nil {
		return mediaUploadResult{}, errors.New("Dosya okunamadı.")
	}
	defer f.Close()
	data := make([]byte, hdr.Size)
	if _, err := readFull(f, data); err != nil {
		return mediaUploadResult{}, errors.New("Dosya okunamadı.")
	}
	// Native yükleme: görsel/ses/video/dosya uzantıları kabul edilir.
	// Yalnız görsellerde decode + boyut + küçük önizleme üretilir.
	kind, ok := mediaKindByExt(hdr.Filename)
	if !ok {
		return mediaUploadResult{}, errors.New("Desteklenmeyen dosya türü: " + hdr.Filename + " (jpg/png/webp/svg, ses, video, pdf/office).")
	}
	var width, height int
	var thumbPath *string
	if kind == "image" && strings.HasSuffix(strings.ToLower(hdr.Filename), ".svg") {
		temiz, w, h, err := platform.SanitizeSVGBoyut(data)
		if err != nil {
			return mediaUploadResult{}, errors.New("SVG reddedildi: " + err.Error())
		}
		data = temiz
		width, height = w, h // 0 olabilir: belgede ölçü yok
		// Küçük önizleme üretilmez: SVG zaten her ölçekte net; raster
		// kopya üretmek vektörün tek avantajını silerdi.
	} else if kind == "image" {
		decoded, err := platform.DecodeImage(data)
		if err != nil {
			return mediaUploadResult{}, errors.New("Desteklenmeyen görsel: " + hdr.Filename + " (jpg/png/webp).")
		}
		b := decoded.Bounds()
		width, height = b.Dx(), b.Dy()
		if thumb, err := platform.CreateThumb(data, 480); err == nil {
			if p, err := platform.SaveMedia(thumb, "thumb.jpg"); err == nil {
				thumbPath = &p
			}
		}
	}
	storedPath, err := platform.SaveMedia(data, hdr.Filename)
	if err != nil {
		return mediaUploadResult{}, errors.New("Depolama hatası.")
	}
	size := int64(len(data))
	mimeType, _ := platformFileExt(hdr.Filename)
	if title == "" {
		title = hdr.Filename
	}
	var widthPtr, heightPtr *int
	if kind == "image" {
		widthPtr, heightPtr = &width, &height
	}
	id, err := repo.CreateMedia(ctx, s.pool, &repo.Media{
		SiteID: s.curSiteID(r), UserID: userID, Kind: kind, Filename: hdr.Filename, Path: storedPath,
		Mime: mimeType, SizeBytes: &size, Width: widthPtr, Height: heightPtr, Title: title, Category: category,
		Alt: &alt, Caption: &caption, Description: description, ThumbPath: thumbPath, Source: "upload",
	})
	if err != nil {
		_ = platform.DeleteMedia(storedPath)
		if thumbPath != nil {
			_ = platform.DeleteMedia(*thumbPath)
		}
		s.logger.Error("media kaydı", "error", err)
		return mediaUploadResult{}, errors.New("Kayıt hatası.")
	}
	s.audit(r, "medya_yukle", hdr.Filename)
	return mediaUploadResult{
		ID: id, URL: s.mediaURL(&id), Thumb: s.mediaThumbURL(&id),
		Filename: hdr.Filename, Title: title,
	}, nil
}

func (s *Server) handleMediaMeta(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	ctx := r.Context()
	media, err := repo.GetMedia(ctx, s.pool, id)
	if err != nil || media.SiteID != s.curSiteID(r) {
		s.flashRedirect(w, r, "/admin/dosyalar", "error", "Medya bulunamadı.")
		return
	}
	_ = r.ParseForm()
	filename := strings.TrimSpace(r.FormValue("filename"))
	if filename == "" {
		filename = media.Filename
	}
	title := strings.TrimSpace(r.FormValue("title"))
	if title == "" {
		title = filename
	}
	category := strings.TrimSpace(r.FormValue("category"))
	if category == "" {
		category = "genel"
	}
	media.Filename = filename
	media.Title = title
	media.Category = category
	media.Description = strings.TrimSpace(r.FormValue("description"))
	media.SEOTitle = strings.TrimSpace(r.FormValue("seo_title"))
	media.SEODescription = strings.TrimSpace(r.FormValue("seo_description"))
	alt := strings.TrimSpace(r.FormValue("alt"))
	caption := strings.TrimSpace(r.FormValue("caption"))
	media.Alt = &alt
	media.Caption = &caption
	if err := repo.UpdateMediaMetadata(ctx, s.pool, s.curSiteID(r), id, &media); err != nil {
		s.logger.Error("medya meta güncelleme", "error", err)
		s.flashRedirect(w, r, "/admin/dosyalar", "error", "Medya bilgileri güncellenemedi.")
		return
	}
	s.audit(r, "medya_meta", filename)
	if IsHTMX(r) {
		s.renderMediaPanel(w, r, "info", "Medya bilgileri güncellendi.")
		return
	}
	// Medya adi/alt metni/gorunurlugu degisti: sayfa onbellegi bayat.
	s.medyaOnbellekDusur(id)

	s.flashRedirect(w, r, "/admin/dosyalar", "info", "Medya bilgileri güncellendi.")
}

func (s *Server) handleMediaDelete(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	ctx := r.Context()
	mediaErr := func(msg string) {
		if IsHTMX(r) {
			s.renderMediaPanel(w, r, "error", msg)
			return
		}
		s.flashRedirect(w, r, "/admin/dosyalar", "error", msg)
	}
	media, err := repo.GetMedia(ctx, s.pool, id)
	if err != nil {
		mediaErr("Medya bulunamadı.")
		return
	}
	if err := repo.DeleteMedia(ctx, s.pool, id); err != nil {
		if errors.Is(err, repo.ErrMediaInUse) {
			mediaErr("Medya bir içerikte kullanılıyor — silinemedi.")
		} else {
			s.logger.Error("medya silme", "error", err)
			mediaErr("Medya silinemedi.")
		}
		return
	}
	_ = platform.DeleteMedia(media.Path)
	if media.ThumbPath != nil {
		_ = platform.DeleteMedia(*media.ThumbPath)
	}
	platform.DeleteMediaVariants(id)
	// Silinen medya hala basilan sayfalarda gorunmesin.
	s.medyaOnbellekDusur(id)
	s.audit(r, "medya_sil", media.Filename)
	if IsHTMX(r) {
		s.renderMediaPanel(w, r, "info", media.Filename+" silindi.")
		return
	}
	s.flashRedirect(w, r, "/admin/dosyalar", "info", "Silindi.")
}

func (s *Server) handleMediaBulk(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_ = r.ParseForm()
	islem := strings.TrimSpace(r.FormValue("islem"))

	var ids []int64
	for _, v := range r.Form["id"] {
		if p := optID(v); p != nil && *p > 0 {
			ids = append(ids, *p)
		}
	}
	geri := func(kind, msg string) { s.flashRedirect(w, r, "/admin/dosyalar", kind, msg) }
	if len(ids) == 0 {
		geri("error", "Hiç medya seçilmedi.")
		return
	}
	if len(ids) > 300 {
		ids = ids[:300]
	}

	basarili, atlandi := 0, 0
	switch islem {
	case "sil":
		for _, id := range ids {
			media, err := repo.GetMedia(ctx, s.pool, id)
			if err != nil {
				atlandi++
				continue
			}
			if err := repo.DeleteMedia(ctx, s.pool, id); err != nil {
				atlandi++
				continue
			}
			_ = platform.DeleteMedia(media.Path)
			if media.ThumbPath != nil {
				_ = platform.DeleteMedia(*media.ThumbPath)
			}
			platform.DeleteMediaVariants(id)
			s.audit(r, "medya_toplu_sil", media.Filename)
			basarili++
		}
	case "yeniden-adlandir":
		base := strings.TrimSpace(r.FormValue("yeni_ad"))
		if base == "" {
			geri("error", "Yeni ad boş olamaz.")
			return
		}
		for i, id := range ids {
			media, err := repo.GetMedia(ctx, s.pool, id)
			if err != nil {
				atlandi++
				continue
			}
			ext := filepath.Ext(media.Filename)
			yeni := base + ext
			if i > 0 {
				yeni = base + "-" + itoa(i) + ext
			}
			if err := repo.RenameMedia(ctx, s.pool, id, yeni); err != nil {
				atlandi++
				continue
			}
			s.audit(r, "medya_toplu_adlandir", media.Filename+" → "+yeni)
			basarili++
		}
	case "kategori":
		category := strings.TrimSpace(r.FormValue("yeni_kategori"))
		if category == "" {
			geri("error", "Yeni kategori boş olamaz.")
			return
		}
		n, err := repo.SetMediaCategory(ctx, s.pool, s.curSiteID(r), ids, category)
		if err != nil {
			s.logger.Error("medya toplu kategori", "error", err)
			geri("error", "Kategoriler güncellenemedi.")
			return
		}
		basarili = n
		atlandi = len(ids) - n
		s.audit(r, "medya_toplu_kategori", category+" adet="+itoa(n))
	default:
		geri("error", "Bilinmeyen işlem.")
		return
	}

	eylem := "işlendi"
	if islem == "sil" {
		eylem = "silindi"
	} else if islem == "yeniden-adlandir" {
		eylem = "adlandırıldı"
	} else if islem == "kategori" {
		eylem = "kategoriye taşındı"
	}
	if atlandi > 0 {
		s.audit(r, "medya_toplu_"+islem, "basarili="+itoa(basarili)+" atlandi="+itoa(atlandi))
		geri("info", itoa(basarili)+" medya "+eylem+", "+itoa(atlandi)+" atlandı (kullanımda ya da bulunamadı).")
		return
	}
	s.audit(r, "medya_toplu_"+islem, "basarili="+itoa(basarili))
	geri("ok", itoa(basarili)+" medya "+eylem+".")
}

// ---- Raporlar ----

func (s *Server) handleReports(w http.ResponseWriter, r *http.Request) {
	d := s.buildReportsData(r)
	if IsHTMX(r) {
		s.renderRaporlarPanelV2(w, r, d)
		return
	}
	s.renderRaporlarPageV2(w, r, d)
}

func (s *Server) buildReportsData(r *http.Request) *tmplData {
	d := s.baseView("tr", "/admin", "Raporlar")
	d.User = ptr(userFromContext(r.Context()))
	d.Flash = s.consumeFlash(r)
	ctx := r.Context()
	most, _ := repo.MostRead(ctx, s.pool, "tr", 10)
	usage, _ := repo.AIUsageSummary(r.Context(), s.pool)
	rpm := repo.GetDefaultRPM(r.Context(), s.pool)
	revenue, _ := repo.ComputeContentRevenue(r.Context(), s.pool, s.curSiteID(r), rpm, 15)
	mostComment, _ := repo.MostCommented(r.Context(), s.pool, s.curSiteID(r), 8)
	yorum, _ := repo.CommentSummary(r.Context(), s.pool, s.curSiteID(r))
	botTrap, _ := repo.CountBotTrapHits(r.Context(), s.pool)
	botTrap24, _ := repo.BotTrap24h(r.Context(), s.pool)
	blocked, _ := repo.CountBlockedIPs(r.Context(), s.pool)
	rd := &reportsData{MostRead: most, AIUsage: usage, Revenue: revenue, RPMDefault: rpm,
		MostCommented: mostComment, Yorum: yorum, BotTrapTotal: botTrap, BotTrap24: botTrap24, BlockedIPs: blocked}

	// grafik verisi
	type pr struct {
		Label string `json:"label"`
		Value int64  `json:"value"`
	}
	arr := make([]pr, 0, len(most))
	for _, m := range most {
		arr = append(arr, pr{Label: m.Title, Value: m.Hits})
	}
	if b, err := json.Marshal(arr); err == nil {
		rd.MostJSON = template.JS(b)
	} else {
		rd.MostJSON = template.JS("[]")
	}
	d.Reports = rd
	return d
}

func (s *Server) handleReportsAIExport(w http.ResponseWriter, r *http.Request) {
	rows, err := repo.AIUsageCSV(r.Context(), s.pool)
	if err != nil {
		s.flashRedirect(w, r, "/admin/raporlar", "error", "Dışa aktarılamadı: "+err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="ai-usage.csv"`)
	// Excel'in Türkçe karakterleri (içerik adları, e-postalar) doğru
	// okuması için UTF-8 BOM; taşıma örnek CSV'siyle aynı davranış.
	_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF})
	cw := csv.NewWriter(w)
	for _, row := range rows {
		_ = cw.Write(row)
	}
	cw.Flush()
}

func applyTierLock(r *http.Request, c *repo.Content, categoryDefault int) {
	weight := 0
	formdaAgirlik := false
	if v := strings.TrimSpace(r.FormValue("min_tier_weight")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 && n <= 5 {
			weight = n
			formdaAgirlik = true
		}
	}
	switch {
	case formdaAgirlik:
		// Form degeri her zaman kazanir.
	case c.ID == 0:
		// Yeni icerik: kategori varsayilanini devral.
		weight = categoryDefault
	default:
		// Guncelleme: alan gonderilmediyse mevcut degeri KORU (istemeden
		// kademe dusurmeyi onler).
		weight = c.MinTierWeight
	}
	if weight > 0 {
		c.MinTierWeight = weight
		c.IsExclusive = true
		return
	}
	if c.IsExclusive {
		c.MinTierWeight = 1
		return
	}
	c.MinTierWeight = 0
}

// seoAlan SEO alanını kırpıp pointer'a çevirir; boşsa nil (alan temizlenir).
func seoAlan(deger string, max int) *string {
	s := platform.SeoKirp(strings.TrimSpace(deger), max)
	if s == "" {
		return nil
	}
	return &s
}
