package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"theblock/site/internal/aieditor"
	"theblock/site/internal/ceviri"
	"theblock/site/internal/repo"
	"theblock/site/internal/scheduler"
	"theblock/site/internal/trends"
)

// botData Bot/Haber Akışı sayfa verisidir.
type botData struct {
	Feeds  []repo.Feed
	Cats   []repo.Category
	Rules  []repo.WordRule
	Auto   bool
	Dakika int
	// SeoOtomatik / SeoDakika: çekilen haberlerin SEO alanları AI ile
	// doldurulsun mu ve dakikada kaçı işlenecek (Faz 188). Boş alanlar
	// doldurulur, dolu alanlar korunur.
	SeoOtomatik bool
	// SeoDakika: dakikada kaç haberin SEO'su doldurulacak (hız).
	SeoDakika int
	// Faz 194 — otomatik AI ile yazım.
	//
	// YazimOtomatik açıksa kaynaklı yeni taslaklar AI'dan geçip gerçek haber
	// metnine dönüşür (başlık, spot, gövde, SEO). Kapalıyken bot yalnız RSS
	// özetini yazar (eski davranış).
	YazimOtomatik bool
	YazimDakika   int
	YazimBekleyen int
	YazimKuyrukta int
	YazimDurum    string
	// Faz 189 — otomatik çeviri ayarları ve kuyruk derinliği.
	CeviriOtomatik bool
	CeviriDiller   string
	CeviriDurum    string
	CeviriDakika   int
	CeviriBekleyen int
	Recent         []repo.Content
	RulesTxt       string
	Trends         []repo.TrendItem
	TrendSrcs      []repo.TrendSource
	TrendTxt       string
	TrendDakika    int

	// Trend Merkezi (Google Trends + kapsama + GSC/GA4)
	TCStats       repo.TrendStats
	TCItems       []trendTermView
	TCSources     []repo.TrendSourceRow
	TCAlerts      []repo.TrendAlertRow
	TCQ           string
	TCFilter      string
	TCMinScore    int
	TCWeights     trends.Weights
	TCThreshold   int
	TCPaid        bool
	TCNotify      bool
	TCAuto        bool
	TCBrief       *trendBriefView
	SearchTop     []repo.SearchQueryRow
	GSCTop        []repo.GSCQueryRow
	GSCProperty   string
	GSCConfigured bool
	GA4Property   string
	GA4Olcum      string
	GA4Configured bool
	GA4           *scheduler.GA4Summary
}

// handleBotGet bot sayfası.
func (s *Server) handleBotGet(w http.ResponseWriter, r *http.Request) {
	d := s.buildBotData(r)
	if IsHTMX(r) {
		s.renderTrendPanelV2(w, r, d)
		return
	}
	s.renderTrendPageV2(w, r, d)
}

func (s *Server) buildBotData(r *http.Request) *tmplData {
	ctx := r.Context()
	d := s.baseView("tr", "/admin", "Bot / Haber Akışı")
	d.User = ptr(userFromContext(ctx))
	d.Flash = s.consumeFlash(r)

	bd := &botData{}
	feeds, err := repo.ListFeeds(ctx, s.pool)
	if err == nil {
		bd.Feeds = feeds
	}
	bd.Cats, _ = repo.ListCategories(ctx, s.pool, s.curSiteID(r))
	bd.Rules, _ = repo.LoadWordRules(ctx, s.pool)
	bd.Auto = true
	if b, err := repo.GetSetting(ctx, s.pool, "bot", "otomatik"); err == nil {
		bd.Auto = string(b) == "true"
	}
	bd.Dakika = 5
	if b, err := repo.GetSetting(ctx, s.pool, "bot", "dakika"); err == nil {
		if n, e := strconv.Atoi(strings.Trim(string(b), `"`)); e == nil {
			bd.Dakika = n
		}
	}
	seoAyar := aieditor.SeoAyarlariOku(ctx, s.pool)
	bd.SeoOtomatik = seoAyar.Aktif
	bd.SeoDakika = seoAyar.Dakika
	yazim := aieditor.YazimAyarlariOku(ctx, s.pool)
	bd.YazimOtomatik = yazim.Aktif
	bd.YazimDakika = yazim.Dakika
	ozet := repo.YazimDurumOzetiOku(ctx, s.pool, s.curSiteID(r))
	bd.YazimBekleyen = ozet.Bekleyen
	bd.YazimKuyrukta = ozet.Kuyrukta
	bd.YazimDurum = yazimDurumMetni(ozet, yazim)
	cevAyar := ceviri.AyarlariOku(ctx, s.pool)
	bd.CeviriOtomatik = cevAyar.Aktif
	bd.CeviriDiller = strings.Join(cevAyar.Diller, ",")
	bd.CeviriDurum = cevAyar.Durum
	bd.CeviriDakika = cevAyar.Dakika
	if len(cevAyar.Diller) > 0 {
		bd.CeviriBekleyen = ceviri.BekleyenSayisi(ctx, s.pool, s.curSiteID(r), cevAyar.Diller[0])
	}
	// kelime kurallarını metne çevir (eski|yeni satır başına)
	var sb strings.Builder
	for _, rl := range bd.Rules {
		sb.WriteString(rl.From + "|" + rl.To + "\n")
	}
	bd.RulesTxt = sb.String()
	// son aktarılanlar
	rec, _, _ := repo.ListContents(ctx, s.pool, repo.ListFilter{Status: "taslak", PerPage: 12})
	bd.Recent = rec

	// Gündem
	bd.Trends, _ = repo.ListTrends(ctx, s.pool, 24)
	bd.TrendSrcs, _ = repo.GetTrendSources(ctx, s.pool)
	var t2 strings.Builder
	for _, ts := range bd.TrendSrcs {
		t2.WriteString(ts.Name + "|" + ts.URL + "\n")
	}
	bd.TrendTxt = t2.String()
	bd.TrendDakika = 15
	if b, err := repo.GetSetting(ctx, s.pool, "gundem", "dakika"); err == nil {
		if n, e := strconv.Atoi(strings.Trim(string(b), `"`)); e == nil && n > 0 {
			bd.TrendDakika = n
		}
	}

	s.fillTrendCenterData(r, bd)

	d.Bot = bd
	return d
}

func (s *Server) renderBotPanel(w http.ResponseWriter, r *http.Request, kind, text string) {
	d := s.buildBotData(r)
	if kind != "" {
		d.Flash = &flashMsg{Kind: kind, Text: text}
	}
	s.renderTrendPanelV2(w, r, d)
}

func (s *Server) botResult(w http.ResponseWriter, r *http.Request, kind, text string) {
	if IsHTMX(r) {
		s.renderBotPanel(w, r, kind, text)
		return
	}
	s.flashRedirect(w, r, "/admin/bot", kind, text)
}

// handleBotEkle feed ekler.
func (s *Server) handleBotEkle(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.FormValue("name"))
	url := strings.TrimSpace(r.FormValue("url"))
	if name == "" || url == "" {
		s.botResult(w, r, "error", "Ad ve URL gerekli.")
		return
	}
	f := &repo.Feed{
		SiteID:     1,
		Name:       name,
		URL:        url,
		CategoryID: optID(r.FormValue("kategori")),
		Aktif:      r.FormValue("pasif") == "",
	}
	if _, err := repo.CreateFeed(r.Context(), s.pool, f); err != nil {
		s.botResult(w, r, "error", "Eklenemedi: "+err.Error())
		return
	}
	s.audit(r, "bot_feed_ekle", name)
	s.botResult(w, r, "info", name+" eklendi.")
}

// handleBotSil feed siler.
func (s *Server) handleBotSil(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	if err := repo.DeleteFeed(r.Context(), s.pool, id); err != nil {
		s.botResult(w, r, "error", "Silinemedi.")
		return
	}
	s.botResult(w, r, "info", "Silindi.")
}

// handleBotCek feed(leri) şimdi çeker.
func (s *Server) handleBotCek(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	author := s.botAuthorDefault(ctx)

	feeds := []repo.Feed{}
	if id := parseID(r.PathValue("id")); id != 0 {
		all, _ := repo.ListFeeds(ctx, s.pool)
		for _, f := range all {
			if f.ID == id {
				feeds = append(feeds, f)
			}
		}
	} else {
		feeds, _ = repo.ListFeeds(ctx, s.pool)
	}
	rules, _ := repo.LoadWordRules(ctx, s.pool)

	total := 0
	lastErr := ""
	for _, f := range feeds {
		if !f.Aktif {
			continue
		}
		n, err := repo.CrawlFeed(ctx, s.pool, &f, rules, author)
		total += n
		if err != nil {
			lastErr = err.Error()
		}
	}
	msg := "Çekildi: " + itoa(total) + " yeni içerik taslağa eklendi."
	kind := "info"
	if lastErr != "" {
		msg += " (" + lastErr + ")"
		kind = "error"
	}
	s.botResult(w, r, kind, msg)
}

// handleBotKelime kelime değiştirici kurallarını kaydeder.
func (s *Server) handleBotKelime(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	raw := r.FormValue("kelimeler")
	var rules []repo.WordRule
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 2)
		from := strings.TrimSpace(parts[0])
		to := ""
		if len(parts) == 2 {
			to = strings.TrimSpace(parts[1])
		}
		if from != "" {
			rules = append(rules, repo.WordRule{From: from, To: to})
		}
	}
	if err := repo.SaveWordRules(ctx, s.pool, rules); err != nil {
		s.botResult(w, r, "error", "Kaydedilemedi.")
		return
	}
	// otomatik + dakika
	v := "false"
	if r.FormValue("otomatik") != "" {
		v = "true"
	}
	_ = repo.SetSetting(ctx, s.pool, "bot", "otomatik", []byte(v))
	// Faz 188: SEO otomatik doldurma anahtarı + tur dilimi.
	seoAktif := r.FormValue("seo_otomatik") != ""
	seoDakika, _ := strconv.Atoi(r.FormValue("seo_dakika"))
	if err := aieditor.SeoDoldurAyarlari(ctx, s.pool, seoAktif, seoDakika); err != nil {
		s.logger.Warn("seo ayarları yazılamadı", "error", err)
	}
	// Faz 189: otomatik çeviri ayarları.
	var diller []string
	for _, d := range strings.Split(r.FormValue("ceviri_diller"), ",") {
		if d = strings.TrimSpace(d); d != "" {
			diller = append(diller, d)
		}
	}
	cevDakika, _ := strconv.Atoi(r.FormValue("ceviri_dakika"))
	if err := ceviri.AyarlariYaz(ctx, s.pool,
		r.FormValue("ceviri_otomatik") != "", diller,
		r.FormValue("ceviri_durum"), cevDakika); err != nil {
		s.logger.Warn("ceviri ayarları yazılamadı", "error", err)
	}
	// Faz 194: otomatik AI ile yazım anahtarı + hız.
	yazimDakika, _ := strconv.Atoi(r.FormValue("yazim_dakika"))
	if err := aieditor.YazimAyarlariYaz(ctx, s.pool,
		r.FormValue("yazim_otomatik") != "", yazimDakika); err != nil {
		s.logger.Warn("yazım ayarları yazılamadı", "error", err)
	}
	if n, err := strconv.Atoi(r.FormValue("dakika")); err == nil && n > 0 {
		_ = repo.SetSetting(ctx, s.pool, "bot", "dakika", []byte(strconv.Itoa(n)))
	}
	s.botResult(w, r, "info", "Ayarlar kaydedildi.")
}

// botAuthorDefault varsayılan editör id'yi döndürür (bot/editorde; yoksa 1).
func (s *Server) botAuthorDefault(ctx context.Context) int64 {
	if b, err := repo.GetSetting(ctx, s.pool, "bot", "editorde"); err == nil {
		if n, e := strconv.Atoi(strings.Trim(string(b), `"`)); e == nil && n > 0 {
			return int64(n)
		}
	}
	return 1
}

// handleTrendYenile gündem kaynaklarını çeker.
func (s *Server) handleTrendYenile(w http.ResponseWriter, r *http.Request) {
	added, err := repo.FetchTrends(r.Context(), s.pool)
	if err != nil {
		s.botResult(w, r, "error", "Gündem çekilemedi: "+err.Error())
		return
	}
	s.audit(r, "gundem_yenile", itoa(added))
	s.botResult(w, r, "info", "Gündem çekildi: "+itoa(added)+" yeni öğe.")
}

// handleTrendKaynak gündem kaynaklarını kaydeder.
func (s *Server) handleTrendKaynak(w http.ResponseWriter, r *http.Request) {
	var srcs []repo.TrendSource
	for _, line := range strings.Split(r.FormValue("gundem_kaynak"), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 2)
		if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
			srcs = append(srcs, repo.TrendSource{Name: strings.TrimSpace(parts[0]), URL: strings.TrimSpace(parts[1])})
		}
	}
	if len(srcs) == 0 {
		s.botResult(w, r, "error", "En az bir kaynak gerekli (Ad|URL).")
		return
	}
	if err := repo.SaveTrendSources(r.Context(), s.pool, srcs); err != nil {
		s.botResult(w, r, "error", "Kaydedilemedi.")
		return
	}
	if n, e := strconv.Atoi(strings.TrimSpace(r.FormValue("gundem_dakika"))); e == nil && n >= 5 && n <= 1440 {
		if b, err := json.Marshal(n); err == nil {
			_ = repo.SetSetting(r.Context(), s.pool, "gundem", "dakika", b)
		}
	}
	s.botResult(w, r, "info", "Gündem kaynakları ve çekim aralığı kaydedildi.")
}

// handleTrendTaslak gündem öğesinden taslak içerik üretir.
func (s *Server) handleTrendTaslak(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	author := s.botAuthorDefault(r.Context())
	nid, err := repo.TrendToContent(r.Context(), s.pool, id, author)
	if err != nil {
		s.botResult(w, r, "error", "Taslak oluşturulamadı: "+err.Error())
		return
	}
	s.audit(r, "gundem_taslak", itoa(int(id)))
	s.flashRedirect(w, r, "/admin/icerikler/"+itoa(int(nid)), "info", "Gündem öğesi taslağa alındı.")
}

// yazimDurumMetni panelde tek satır durum metni üretir.
//
// Neden: anahtar açık ama işçi çalışmıyorsa kullanıcı bunu göremez.
// "240 taslak bekliyor" yazıp hiçbir şey olmaması sessiz hatadır.
func yazimDurumMetni(o repo.YazimDurumOzeti, a aieditor.YazimAyarlar) string {
	if !a.Aktif {
		return "kapalı — yeni taslaklar RSS özetiyle kalır"
	}
	var parca []string
	if o.Bekleyen > 0 {
		parca = append(parca, itoa(o.Bekleyen)+" taslak bekliyor")
	}
	if o.Kuyrukta > 0 {
		parca = append(parca, itoa(o.Kuyrukta)+" iş kuyrukta")
	}
	if o.Uygulanacak > 0 {
		parca = append(parca, itoa(o.Uygulanacak)+" sonuç yazılacak")
	}
	if len(parca) == 0 {
		parca = append(parca, "işlenecek taslak yok")
	}
	metin := strings.Join(parca, " · ")
	if o.SonUygulama != nil {
		metin += " · son yazım " + o.SonUygulama.In(time.Local).Format("02.01 15:04")
	}
	return metin
}
