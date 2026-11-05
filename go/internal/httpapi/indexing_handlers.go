package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"theblock/site/internal/platform"
	"theblock/site/internal/repo"
)

// ---- P4-6: IndexNow + Google Indexing API ----

type indexingData struct {
	IndexNowKey  string
	KeyURL       string
	Endpoint     string
	SAConfigured bool
	Auto         bool
	Logs         []repo.IndexLog
	Contents     []repo.Content
	TrapCount    int64
	TrapHits     []repo.BotTrapHit
	BlockedCount int64
	Blocked      []repo.BlockedIP
	Allowlist    string
	// Haber sitemap'i (Faz 23): mutlak adres + son doğrulama raporu.
	NewsSitemapURL string
	NewsReport     *NewsSitemapReport
}

func (s *Server) seoSetting(ctx context.Context, key string) string {
	b, err := repo.GetSetting(ctx, s.pool, "seo", key)
	if err != nil {
		return ""
	}
	var v string
	if json.Unmarshal(b, &v) == nil {
		return v
	}
	return strings.Trim(string(b), `"`)
}

func (s *Server) setSeoSetting(ctx context.Context, key, val string) {
	b, _ := json.Marshal(val)
	_ = repo.SetSetting(ctx, s.pool, "seo", key, b)
}

func (s *Server) indexNowKey(ctx context.Context) string {
	k := s.seoSetting(ctx, "indexnow_key")
	if k == "" {
		k = rndSuffix() + rndSuffix()
		s.setSeoSetting(ctx, "indexnow_key", k)
	}
	return k
}

func (s *Server) keyLocation(ctx context.Context) string {
	return strings.TrimRight(s.cfg.BaseURL, "/") + "/" + s.indexNowKey(ctx) + ".txt"
}

func (s *Server) handleIndexingAdmin(w http.ResponseWriter, r *http.Request) {
	d := s.buildIndexingData(r)
	if IsHTMX(r) {
		s.renderIndexlemePanelV2(w, r, d)
		return
	}
	s.renderIndexlemePageV2(w, r, d)
}

func (s *Server) buildIndexingData(r *http.Request) *tmplData {
	ctx := r.Context()
	sid := s.curSiteID(r)
	d := s.baseView("tr", "/admin", "İndeksleme (SEO)")
	d.User = ptr(userFromContext(ctx))
	d.Flash = s.consumeFlash(r)
	items, _, _ := repo.ListContents(ctx, s.pool, repo.ListFilter{Status: "yayinda", SiteID: sid, PerPage: 100, Page: 1})
	logs, _ := repo.ListIndexing(ctx, s.pool, sid, 40)
	trapCount, _ := repo.CountBotTrapHits(ctx, s.pool)
	trapHits, _ := repo.ListBotTrapHits(ctx, s.pool, 20)
	blockCount, _ := repo.CountBlockedIPs(ctx, s.pool)
	blocked, _ := repo.ListBlockedIPs(ctx, s.pool, 100)
	d.Indexleme = &indexingData{
		IndexNowKey:    s.indexNowKey(ctx),
		KeyURL:         s.keyLocation(ctx),
		Endpoint:       s.seoSetting(ctx, "indexnow_endpoint"),
		SAConfigured:   s.seoSetting(ctx, "gsc_service_account") != "",
		Auto:           s.seoSetting(ctx, "auto_index") == "1",
		Logs:           logs,
		Contents:       items,
		TrapCount:      trapCount,
		TrapHits:       trapHits,
		BlockedCount:   blockCount,
		Blocked:        blocked,
		Allowlist:      s.seoSetting(ctx, "blok_izinli"),
		NewsSitemapURL: strings.TrimRight(s.absBase(r), "/") + "/sitemap-news.xml",
		NewsReport:     s.lastNewsSitemapReport(ctx),
	}
	return d
}

func (s *Server) renderIndexingPanel(w http.ResponseWriter, r *http.Request, kind, text string) {
	d := s.buildIndexingData(r)
	if kind != "" {
		d.Flash = &flashMsg{Kind: kind, Text: text}
	}
	s.renderIndexlemePanelV2(w, r, d)
}

func (s *Server) indexingResult(w http.ResponseWriter, r *http.Request, kind, text string) {
	if IsHTMX(r) {
		s.renderIndexingPanel(w, r, kind, text)
		return
	}
	s.flashRedirect(w, r, "/admin/indexleme", kind, text)
}

// lastNewsSitemapReport kayıtlı son doğrulama raporunu okur (yoksa nil).
func (s *Server) lastNewsSitemapReport(ctx context.Context) *NewsSitemapReport {
	b, err := repo.GetSetting(ctx, s.pool, "seo", "news_sitemap_report")
	if err != nil || len(b) == 0 {
		return nil
	}
	var rep NewsSitemapReport
	if json.Unmarshal(b, &rep) != nil {
		return nil
	}
	return &rep
}

// handleNewsSitemapCheck haber sitemap'ini yapısal olarak doğrular, sonucu
// kaydeder (bildirimler tablosu) ve panele döner. Ağ isteği yapılmaz.
func (s *Server) handleNewsSitemapCheck(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sid := s.curSiteID(r)
	abs := strings.TrimRight(s.absBase(r), "/")

	items, _ := repo.RecentPublishedAllLang(ctx, s.pool, sid, 200)
	rep := validateNewsSitemapXML([]byte(s.newsSitemap(abs, items)))
	rep.URL = abs + "/sitemap-news.xml"

	if b, err := json.Marshal(rep); err == nil {
		_ = repo.SetSetting(ctx, s.pool, "seo", "news_sitemap_report", b)
	}
	_ = repo.LogIndexing(ctx, s.pool, sid, "news-sitemap", rep.URL, rep.Durum, rep.Mesaj)
	s.audit(r, "news_sitemap_kontrol", "seo")

	kind := map[string]string{"ok": "ok", "uyari": "info", "hata": "error"}[rep.Durum]
	if kind == "" {
		kind = "info"
	}
	s.indexingResult(w, r, kind, "Haber sitemap'i: "+rep.Mesaj)
}

func (s *Server) handleIndexingSave(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	s.setSeoSetting(ctx, "indexnow_endpoint", strings.TrimSpace(r.FormValue("endpoint")))
	s.setSeoSetting(ctx, "blok_izinli", strings.TrimSpace(r.FormValue("blok_izinli")))
	if sa := r.FormValue("service_account"); sa != "" {
		s.setSeoSetting(ctx, "gsc_service_account", sa)
	}
	auto := "0"
	if r.FormValue("auto") != "" {
		auto = "1"
	}
	s.setSeoSetting(ctx, "auto_index", auto)
	s.audit(r, "indeksleme_ayar", "seo")
	s.indexingResult(w, r, "info", "İndeksleme ayarları kaydedildi.")
}

// handleIndexNowSubmit IndexNow'a bir URL (veya seçili içerik) bildirir.
func (s *Server) handleIndexNowSubmit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sid := s.curSiteID(r)
	var urls []string
	if id := parseID(r.FormValue("icerik")); id > 0 {
		if c, err := repo.GetContent(ctx, s.pool, id); err == nil {
			urls = append(urls, strings.TrimRight(s.cfg.BaseURL, "/")+publicArticleURL(c.Locale, c.DisplaySlug(), c.ID))
		}
	} else if u := strings.TrimSpace(r.FormValue("url")); u != "" {
		urls = append(urls, u)
	}
	if len(urls) == 0 {
		s.indexingResult(w, r, "error", "İçerik seçin veya URL girin.")
		return
	}
	ok := s.submitIndexNow(ctx, sid, urls)
	if ok {
		s.indexingResult(w, r, "info", "IndexNow'a bildirildi: "+itoa(len(urls))+" URL")
		return
	}
	s.indexingResult(w, r, "error", "IndexNow bildirimi başarısız (ayrıntı logda).")
}

// submitIndexNow IndexNow uçlarına URL listesi gönderir ve loglar.
func (s *Server) submitIndexNow(ctx context.Context, siteID int64, urls []string) bool {
	host := strings.TrimPrefix(strings.TrimPrefix(s.cfg.BaseURL, "https://"), "http://")
	host = strings.TrimRight(host, "/")
	code, err := platform.SubmitIndexNow(ctx, s.seoSetting(ctx, "indexnow_endpoint"), host, s.indexNowKey(ctx), s.keyLocation(ctx), urls)
	for _, u := range urls {
		if err != nil {
			_ = repo.LogIndexing(ctx, s.pool, siteID, "indexnow", u, "hata", err.Error())
		} else {
			_ = repo.LogIndexing(ctx, s.pool, siteID, "indexnow", u, "ok", "HTTP "+itoa(code))
		}
	}
	return err == nil
}

// handleGoogleIndexingSubmit yapılandırılmış servis hesabıyla Indexing API'ye bildirir.
func (s *Server) handleGoogleIndexingSubmit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sid := s.curSiteID(r)
	id := parseID(r.FormValue("icerik"))
	if id == 0 {
		s.indexingResult(w, r, "error", "İçerik seçin.")
		return
	}
	c, err := repo.GetContent(ctx, s.pool, id)
	if err != nil {
		s.indexingResult(w, r, "error", "İçerik bulunamadı.")
		return
	}
	sa := s.seoSetting(ctx, "gsc_service_account")
	if sa == "" {
		s.indexingResult(w, r, "error", "Servis hesabı JSON'u girilmemiş.")
		return
	}
	url := strings.TrimRight(s.cfg.BaseURL, "/") + publicArticleURL(c.Locale, c.DisplaySlug(), c.ID)
	tok, err := platform.ServiceAccountToken(ctx, []byte(sa), "https://www.googleapis.com/auth/indexing")
	if err != nil {
		_ = repo.LogIndexing(ctx, s.pool, sid, "google-indexing", url, "hata", err.Error())
		s.indexingResult(w, r, "error", "Token alınamadı: "+err.Error())
		return
	}
	code, err := platform.GoogleIndexingPublish(ctx, tok, url)
	if err != nil {
		_ = repo.LogIndexing(ctx, s.pool, sid, "google-indexing", url, "hata", err.Error())
		s.indexingResult(w, r, "error", "Indexing API hatası: "+err.Error())
		return
	}
	_ = repo.LogIndexing(ctx, s.pool, sid, "google-indexing", url, "ok", "HTTP "+itoa(code))
	s.indexingResult(w, r, "info", "Google Indexing API'ye bildirildi.")
}

// handleBlockUnblock bir IP'nin bloğunu kaldırır.
func (s *Server) handleBlockUnblock(w http.ResponseWriter, r *http.Request) {
	ip := strings.TrimSpace(r.FormValue("ip"))
	if ip == "" {
		s.indexingResult(w, r, "error", "IP gerekli.")
		return
	}
	if err := repo.UnblockIP(r.Context(), s.pool, ip); err != nil {
		s.indexingResult(w, r, "error", "Kaldırılamadı.")
		return
	}
	s.invalidateIPBlock()
	s.audit(r, "ip_blok_kaldir", ip)
	s.indexingResult(w, r, "info", "Blok kaldırıldı: "+ip)
}

// handleIndexNowKeyFile IndexNow anahtar dosyasını sunar (/indexnow/{key}.txt).
func (s *Server) handleIndexNowKeyFile(w http.ResponseWriter, r *http.Request) {
	file := r.PathValue("file")
	want := s.indexNowKey(r.Context()) + ".txt"
	if file != want {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Write([]byte(s.indexNowKey(r.Context())))
}

// autoIndex yayınlanan içeriği otomatik olarak IndexNow'a bildirir (ayar açıksa).
func (s *Server) autoIndex(ctx context.Context, siteID, contentID int64) {
	if s.seoSetting(ctx, "auto_index") != "1" {
		return
	}
	c, err := repo.GetContent(ctx, s.pool, contentID)
	if err != nil || c.Status != "yayinda" {
		return
	}
	url := strings.TrimRight(s.cfg.BaseURL, "/") + publicArticleURL(c.Locale, c.DisplaySlug(), c.ID)
	// #nosec G118 -- Bilinçli: IndexNow ping'i yanıtı bekletmemeli; istek
	// context'i yanıt bitince iptal edilir, gönderim kendi zaman aşımını
	// kullanır.
	go func() { s.submitIndexNow(context.Background(), siteID, []string{url}) }()
}
