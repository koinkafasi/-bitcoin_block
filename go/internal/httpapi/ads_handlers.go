package httpapi

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"time"

	"theblock/site/internal/platform"
	"theblock/site/internal/repo"
)

// ---- Reklam yönetimi (spec 01 §E / 02 P2-4, 05 §3) ----

// adCacheEntry aktif reklamların 30 sn'lik bellek önbelleğidir.
type adCacheEntry struct {
	ats   time.Time
	slots map[string][]repo.Ad
}

type adsData struct {
	Ads     []repo.Ad
	Slots   []repo.AdSlot
	Stats   *repo.AdStatsSummary
	Status  string
	SlotMap map[string]template.HTML
	Has     bool
}

func (s *Server) fillAdSlots(ctx context.Context, siteID int64, d *tmplData) {
	if d.Ads == nil {
		d.Ads = &adsData{}
	}
	if len(d.Ads.SlotMap) > 0 {
		return
	}
	active := s.activeAds(ctx, siteID)
	if len(active) == 0 {
		d.Ads.Has = false
		return
	}
	d.Ads.Has = true
	d.Ads.SlotMap = map[string]template.HTML{}
	for key, list := range active {
		var b strings.Builder
		for i, a := range list {
			b.WriteString(buildAdHTML(a, i))
		}
		d.Ads.SlotMap[key] = template.HTML(b.String())
	}
}

// buildAdHTML bir reklamı render/hazır HTML'e çevirir.
// - reklamda özel HTML verildiyse süzülmüş hâliyle,
// - yoksa görsel + bağlantı (tıklama sayacı): <a href="/api/ad-tikla/{id}?y=hedef">
func buildAdHTML(a repo.Ad, idx int) string {
	beacon := fmt.Sprintf(`<img src="/api/ad-gorselleme?id=%d" width="1" height="1" alt="" aria-hidden="true" style="width:1px;height:1px;border:0;display:block;margin:0 auto">`, a.ID)
	// Depodaki kayıt süzgeçten geçmiş olsa da (adFromForm) eski satırlar ham
	// olabilir: her basımda yeniden süzülür (savunma derinliği).
	if custom := platform.SanitizeAdHTML(a.HTML); custom != "" {
		return fmt.Sprintf(`<div class="ad reb" data-ad-id="%d" style="margin:10px 0;text-align:center;overflow:hidden">%s%s</div>`,
			a.ID, custom, beacon)
	}
	target := a.TargetURL
	if target == "" {
		target = "/"
	}
	cls := "adz adz-" + safeClass(a.Kind) + " adz-" + safeClass(a.SlotKey)
	label := `<div class="adz-label">Reklam</div>`
	if a.ImageURL != "" {
		img := fmt.Sprintf(`<a href="/api/ad-tikla/%d?y=%s" rel="nofollow sponsored" aria-label="%s"><img src="%s" alt="Reklam %d" class="adz-img" loading="lazy"></a>`,
			a.ID, url.QueryEscape(target), htmlEscapeAttr(a.Name), a.ImageURL, idx)
		return fmt.Sprintf(`<div class="%s" data-ad-id="%d" style="text-align:center;overflow:hidden">%s%s%s</div>`, cls, a.ID, label, img, beacon)
	}
	link := fmt.Sprintf(`<a href="/api/ad-tikla/%d?y=%s" rel="nofollow sponsored" class="adz-text">%s</a>`,
		a.ID, url.QueryEscape(target), htmlEscape(a.Name))
	return fmt.Sprintf(`<div class="%s" data-ad-id="%d" style="text-align:center">%s%s%s</div>`, cls, a.ID, label, link, beacon)
}

// activeAds önbellekli aktif reklam sorgusu.
func (s *Server) activeAds(ctx context.Context, siteID int64) map[string][]repo.Ad {
	s.adMu.Lock()
	defer s.adMu.Unlock()
	if e, ok := s.adCache[siteID]; ok && time.Since(e.ats) < 30*time.Second {
		return e.slots
	}
	slots, err := repo.ActiveAds(ctx, s.pool, siteID)
	if err != nil {
		return nil
	}
	// boş sonuç da kısa süre önbellekte tutulsun
	s.adCache[siteID] = adCacheEntry{ats: time.Now(), slots: slots}
	return slots
}

// htmlEscapeAttr…
// jsonld.go içindeki htmlEscape kullanılır.
func htmlEscapeAttr(s string) string { return htmlEscape(s) }

func safeClass(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	return b.String()
}

// ---- Panel: reklam listesi + yerleşimler + istatistik ----

func (s *Server) handleAdsGet(w http.ResponseWriter, r *http.Request) {
	d := s.buildAdsData(r)
	if IsHTMX(r) {
		s.renderReklamlarPanelV2(w, r, d)
		return
	}
	s.renderReklamlarPageV2(w, r, d)
}

func (s *Server) buildAdsData(r *http.Request) *tmplData {
	ctx := r.Context()
	sid := s.curSiteID(r)
	status := r.URL.Query().Get("durum")
	ads, _ := repo.ListAds(ctx, s.pool, sid, status)
	slots, _ := repo.ListAdSlots(ctx, s.pool, sid)
	stats, _ := repo.AdStats(ctx, s.pool, sid, 30)
	d := s.baseView("tr", "/admin", "Reklam Yönetimi")
	d.User = ptr(userFromContext(ctx))
	d.Flash = s.consumeFlash(r)
	d.Ads = &adsData{Ads: ads, Slots: slots, Stats: stats, Status: status}
	return d
}

func (s *Server) renderAdsPanel(w http.ResponseWriter, r *http.Request, kind, text string) {
	d := s.buildAdsData(withHTMXCurrentQuery(r))
	if kind != "" {
		d.Flash = &flashMsg{Kind: kind, Text: text}
	}
	s.renderReklamlarPanelV2(w, r, d)
}

// adsResult normal istekte flash redirect, HTMX'te panel tazeleme.
func (s *Server) adsResult(w http.ResponseWriter, r *http.Request, kind, text string) {
	if IsHTMX(r) {
		s.renderAdsPanel(w, r, kind, text)
		return
	}
	s.flashRedirect(w, r, "/admin/reklamlar", kind, text)
}

// yerleşim (slot) CRUD
func (s *Server) handleAdSlotCreate(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimSpace(r.FormValue("key"))
	label := strings.TrimSpace(r.FormValue("label"))
	if key == "" || label == "" {
		s.adsResult(w, r, "error", "Anahtar ve etiket gerekli.")
		return
	}
	if _, err := repo.CreateAdSlot(r.Context(), s.pool, s.curSiteID(r), key, label,
		parseIntOr(r.FormValue("genislik")), parseIntOr(r.FormValue("yukseklik")), r.FormValue("yapistir") == "1"); err != nil {
		s.adsResult(w, r, "error", "Yerleşim eklenemedi (anahtar benzersiz olmalı).")
		return
	}
	s.audit(r, "reklam_yeri_ekle", key)
	s.adsResult(w, r, "info", "Yerleşim eklendi: "+label)
}

func (s *Server) handleAdSlotUpdate(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	key := strings.TrimSpace(r.FormValue("key"))
	label := strings.TrimSpace(r.FormValue("label"))
	if err := repo.UpdateAdSlot(r.Context(), s.pool, id, key, label,
		parseIntOr(r.FormValue("genislik")), parseIntOr(r.FormValue("yukseklik")),
		r.FormValue("yapistir") == "1", r.FormValue("durum")); err != nil {
		s.adsResult(w, r, "error", "Güncellenemedi.")
		return
	}
	s.audit(r, "reklam_yeri_guncelle", key)
	s.adsResult(w, r, "info", "Yerleşim güncellendi.")
}

func (s *Server) handleAdSlotDelete(w http.ResponseWriter, r *http.Request) {
	if err := repo.DeleteAdSlot(r.Context(), s.pool, parseID(r.PathValue("id"))); err != nil {
		s.adsResult(w, r, "error", "Silinemedi (bağlı reklam var).")
		return
	}
	s.audit(r, "reklam_yeri_sil", r.PathValue("id"))
	s.adsResult(w, r, "info", "Yerleşim silindi.")
}

// reklam CRUD
func (s *Server) handleAdCreate(w http.ResponseWriter, r *http.Request) {
	a := adFromForm(r, s)
	if a.Name == "" {
		s.adsResult(w, r, "error", "Reklam adı gerekli.")
		return
	}
	_, err := repo.CreateAd(r.Context(), s.pool, a)
	if err != nil {
		s.logger.Error("reklam ekleme", "error", err)
		s.adsResult(w, r, "error", "Eklenemedi.")
		return
	}
	s.audit(r, "reklam_ekle", a.Name)
	s.adsResult(w, r, "info", "Reklam eklendi: "+a.Name)
}

func (s *Server) handleAdUpdate(w http.ResponseWriter, r *http.Request) {
	a := adFromForm(r, s)
	a.ID = parseID(r.PathValue("id"))
	if a.Name == "" {
		s.adsResult(w, r, "error", "Reklam adı gerekli.")
		return
	}
	if err := repo.UpdateAd(r.Context(), s.pool, a); err != nil {
		s.logger.Error("reklam güncelleme", "error", err)
		s.adsResult(w, r, "error", "Güncellenemedi.")
		return
	}
	s.audit(r, "reklam_guncelle", a.Name)
	s.adsResult(w, r, "info", "Reklam güncellendi: "+a.Name)
}

func (s *Server) handleAdDelete(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	if err := repo.DeleteAd(r.Context(), s.pool, id); err != nil {
		s.adsResult(w, r, "error", "Silinemedi.")
		return
	}
	s.audit(r, "reklam_sil", "id="+itoa(int(id)))
	s.adsResult(w, r, "info", "Reklam silindi.")
}

// adFromForm form alanlarını Ad yapısına çevirir.
func adFromForm(r *http.Request, s *Server) *repo.Ad {
	_ = s
	starts, _ := time.Parse("2006-01-02", r.FormValue("baslangic"))
	ends, _ := time.Parse("2006-01-02", r.FormValue("bitis"))
	var startsPtr, endsPtr *time.Time
	if !starts.IsZero() {
		startCopy := starts
		startsPtr = &startCopy
	}
	if !ends.IsZero() {
		ends = ends.Add(24*time.Hour - time.Second)
		endCopy := ends
		endsPtr = &endCopy
	}
	var img *int64
	if v := parseID(r.FormValue("gorsel_id")); v > 0 {
		img = &v
	}
	status := r.FormValue("durum")
	switch status {
	case "aktif", "bitmis", "pasif":
	default:
		status = "taslak"
	}
	return &repo.Ad{
		SiteID:       s.curSiteID(r),
		Name:         strings.TrimSpace(r.FormValue("ad")),
		Kind:         firstNonEmpty(r.FormValue("tur"), "display"),
		SlotKey:      strings.TrimSpace(r.FormValue("yer")),
		HTML:         platform.SanitizeAdHTML(r.FormValue("html")),
		ImageMediaID: img,
		TargetURL:    strings.TrimSpace(r.FormValue("hedef")),
		StartsAt:     startsPtr,
		EndsAt:       endsPtr,
		Status:       status,
		Weight:       parseIntOr(r.FormValue("agirlik")),
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func parseIntOr(v string) int {
	if v == "" {
		return 0
	}
	n := 0
	for _, r := range v {
		if r < '0' || r > '9' {
			return 0
		}
		n = n*10 + int(r-'0')
	}
	return n
}

// ---- Public ölçüm ----

// handleAdClick tıklamayı sayar ve hedefe yönlendirir.
//
// Güvenlik: yönlendirme hedefi öncelikle reklamın DB'de kayıtlı target_url'i
// olur (yönetici kontrollü). Kullanıcı kontrollü ?y= parametresi yalnızca
// site-içi göreli yol için kabul edilir; böylece açık yönlendirme engellenir.
func (s *Server) handleAdClick(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	site := s.siteByDomain(r.Context(), r.Host)
	if id > 0 {
		_ = repo.RecordAdClick(r.Context(), s.pool, site.ID, id)
	}
	dest := ""
	if id > 0 {
		if u, err := repo.AdTarget(r.Context(), s.pool, site.ID, id); err == nil {
			dest = u
		}
	}
	if !isSafeAdTarget(dest) {
		dest = sameSitePath(r.URL.Query().Get("y"))
	}
	if dest == "" {
		dest = "/"
	}
	http.Redirect(w, r, dest, http.StatusFound)
}

// isSafeAdTarget yalnızca mutlak http(s) URL'lerini kabul eder.
func isSafeAdTarget(u string) bool {
	u = strings.TrimSpace(u)
	if u == "" {
		return false
	}
	p, err := url.Parse(u)
	if err != nil {
		return false
	}
	return (p.Scheme == "http" || p.Scheme == "https") && p.Host != ""
}

// sameSitePath yalnızca site-içi göreli yolları döndürür (açık yönlendirme koruması).
func sameSitePath(u string) string {
	u = strings.TrimSpace(u)
	// Ters bölü: bazı tarayıcılar "\" karakterini "/" gibi yorumlar; "/\evil.com"
	// protokol-göreli adres sayılabilir. Hiç kabul etmiyoruz.
	if strings.HasPrefix(u, "/") && !strings.HasPrefix(u, "//") &&
		!strings.ContainsAny(u, "\r\n\\") {
		return u
	}
	return ""
}

// handleAdView gösterim pikseli/API'si (opsiyonel client ölçümü).
func (s *Server) handleAdView(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	site := s.siteByDomain(ctx, r.Host)
	_ = r.ParseForm()
	id := parseID(r.FormValue("id"))
	if id > 0 {
		_ = repo.RecordAdView(ctx, s.pool, site.ID, id, 1)
	}
	w.Header().Set("Content-Type", "image/gif")
	w.Write([]byte("\x47\x49\x46\x38\x39\x61\x01\x00\x01\x00\x80\x00\x00\x00\x00\x00\xff\xff\xff\x21\xf9\x04\x01\x00\x00\x00\x00\x2c\x00\x00\x00\x00\x01\x00\x01\x00\x00\x02\x02\x44\x01\x00\x3b"))
}
