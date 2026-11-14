package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"theblock/site/internal/platform"
	"theblock/site/internal/repo"
)

type BrandConfig struct {
	Mode   string `json:"mode"`
	Text   string `json:"text"`
	IconID *int64 `json:"iconId"`
	// IconYukseklik simgenin çizim yüksekliği (px). 0 → LogoYukseklik.
	IconYukseklik int `json:"iconYukseklik,omitempty"`
	// ImageID KOYU tema logosu (varsayılan).
	ImageID     *int64 `json:"imageId"`
	ImageAcikID *int64 `json:"imageAcikId"`
	FaviconID   *int64 `json:"faviconId"`
	OGImageID   *int64 `json:"ogImageId"`
	Tagline     string `json:"tagline"`

	LogoYukseklik     int `json:"logoYukseklik,omitempty"`
	SloganGirinti     int `json:"sloganGirinti,omitempty"`
	LogoGenislikAzami int `json:"logoGenislikAzami,omitempty"`
}

type brandEditData struct {
	Config *BrandConfig
	// Site kimliği — başlık/slogan (Ayarlar→Genel'den taşındı, tek merkez).
	SiteBaslik string
	Slogan     string
}

// brandConfig ayardan okur; yoksa varsayılan (metin logo).
func (s *Server) brandConfig(ctx context.Context) *BrandConfig {
	if v := s.cachedSetting(ctx, "brand", "ana"); len(v) > 0 {
		var c BrandConfig
		if json.Unmarshal(v, &c) == nil {
			if c.Mode == "" {
				c.Mode = "text"
			}
			return &c
		}
	}
	return &BrandConfig{Mode: "text"}
}

// brandSiteIdentity site başlığı/slogan'ını okur (settings genel).
func (s *Server) brandSiteIdentity(ctx context.Context) (string, string) {
	raw, err := repo.GetSetting(ctx, s.pool, "genel", "site_baslik")
	baslik := ""
	if err == nil {
		var m map[string]string
		if json.Unmarshal(raw, &m) == nil {
			baslik = m["tr"]
		}
	}
	raw, err = repo.GetSetting(ctx, s.pool, "genel", "slogan")
	slogan := ""
	if err == nil {
		var m map[string]string
		if json.Unmarshal(raw, &m) == nil {
			slogan = m["tr"]
		}
	}
	return baslik, slogan
}

func (s *Server) saveBrandImage(r *http.Request, field string) (int64, error) {
	f, hdr, err := r.FormFile(field)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	data := make([]byte, hdr.Size)
	if _, err := readFull(f, data); err != nil {
		return 0, err
	}
	if len(data) == 0 {
		return 0, errors.New("dosya boş")
	}
	var width, height int
	if strings.HasSuffix(strings.ToLower(hdr.Filename), ".svg") {
		// SVG: vektör. Beyaz liste temizliği ZORUNLU; başarısızsa reddedilir.
		temiz, w, h, err := platform.SanitizeSVGBoyut(data)
		if err != nil {
			return 0, errors.New("SVG reddedildi: " + err.Error())
		}
		data = temiz
		width, height = w, h // 0 olabilir: belgede ölçü yok
	} else {
		decoded, err := platform.DecodeImage(data)
		if err == nil {
			b := decoded.Bounds()
			width, height = b.Dx(), b.Dy()
		} else if !isKnownImageExt(hdr.Filename) {
			return 0, errors.New("desteklenmeyen görsel (jpg/png/webp/svg)")
		} else {
			// Bilinen uzantı ama çözülemedi (ör. animasyonlu/egzotik webp):
			// dosyayı sakla, ölçüleri boş bırak.
			s.logger.Warn("marka görseli çözülemedi, ölçüsüz saklanıyor", "alan", field, "error", err)
		}
	}
	storedPath, err := platform.SaveMedia(data, hdr.Filename)
	if err != nil {
		return 0, err
	}
	m, _ := platformFileExt(hdr.Filename)
	size := int64(len(data))
	u := userFromContext(r.Context())
	mid, err := repo.CreateMedia(r.Context(), s.pool, &repo.Media{
		SiteID: s.curSiteID(r), UserID: u.ID, Kind: "image", Filename: hdr.Filename, Path: storedPath,
		Mime: m, SizeBytes: &size, Width: &width, Height: &height, Source: "brand",
	})
	if err != nil {
		return 0, err
	}
	return mid, nil
}

// isKnownImageExt dosya adı bilinen görsel uzantısı mı?
func isKnownImageExt(name string) bool {
	lower := strings.ToLower(name)
	// .svg: vektör logo (kullanıcı isteği, 11 Kasım 2026). Yükleme yolunda
	// platform.SanitizeSVG beyaz liste temizliğinden geçer.
	for _, ext := range []string{".jpg", ".jpeg", ".png", ".webp", ".gif", ".avif", ".svg"} {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return false
}

// handleBrandGet /admin/marka — marka/logo yöneticisi.
func (s *Server) handleBrandGet(w http.ResponseWriter, r *http.Request) {
	d := s.buildBrandData(r)
	if IsHTMX(r) {
		s.renderMarkaPanel(w, r, d)
		return
	}
	s.renderMarkaPage(w, r, d)
}

func (s *Server) buildBrandData(r *http.Request) *tmplData {
	d := s.baseView("tr", "/admin", "Marka ve Logo")
	d.User = ptr(userFromContext(r.Context()))
	d.Flash = s.consumeFlash(r)
	baslik, slogan := s.brandSiteIdentity(r.Context())
	d.BrandEdit = &brandEditData{Config: s.brandConfig(r.Context()), SiteBaslik: baslik, Slogan: slogan}
	g := gorunumLoad(r.Context(), s.pool)
	d.Gorunum = &g
	return d
}

func (s *Server) renderBrandPanel(w http.ResponseWriter, r *http.Request, kind, text string) {
	d := s.buildBrandData(r)
	if kind != "" {
		d.Flash = &flashMsg{Kind: kind, Text: text}
	}
	s.renderMarkaPanel(w, r, d)
}

func (s *Server) brandResult(w http.ResponseWriter, r *http.Request, kind, text string) {
	if IsHTMX(r) {
		s.renderBrandPanel(w, r, kind, text)
		return
	}
	s.flashRedirect(w, r, "/admin/marka", kind, text)
}

// handleBrandSave POST /admin/marka/kaydet — multipart (logo/favicon/og + alanlar).
func (s *Server) handleBrandSave(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	r.Body = http.MaxBytesReader(w, r.Body, 50<<20)
	if err := r.ParseMultipartForm(60 << 20); err != nil {
		s.brandResult(w, r, "error", "Form okunamadı (dosya çok büyük olabilir).")
		return
	}
	cfg := s.brandConfig(ctx)
	mode := r.FormValue("mode")
	if mode != "image" {
		mode = "text"
	}
	cfg.Mode = mode
	cfg.Text = strings.TrimSpace(r.FormValue("text"))
	cfg.Tagline = strings.TrimSpace(r.FormValue("tagline"))

	// LOGO ÖLÇÜSÜ (kullanıcı isteği, 11 Kasım 2026). Boş/bozuk giriş
	// varsayılana döner; aralık dışı değer KIRPILIR (uydurma değer yok).
	if r.Form.Has("logo_yukseklik") {
		cfg.LogoYukseklik = sinirlaInt(r.FormValue("logo_yukseklik"), 16, 120, 0)
	}
	if r.Form.Has("logo_genislik") {
		cfg.LogoGenislikAzami = sinirlaInt(r.FormValue("logo_genislik"), 40, 600, 0)
	}
	if r.Form.Has("slogan_girinti") {
		cfg.SloganGirinti = sinirlaInt(r.FormValue("slogan_girinti"), 0, 600, 0)
	}
	if r.Form.Has("icon_yukseklik") {
		cfg.IconYukseklik = sinirlaInt(r.FormValue("icon_yukseklik"), 16, 120, 0)
	}

	if r.FormValue("remove_logo") != "" {
		cfg.ImageID = nil
	}
	if r.FormValue("remove_icon") != "" {
		cfg.IconID = nil
	}
	if r.FormValue("remove_logo_acik") != "" {
		cfg.ImageAcikID = nil
	}
	if r.FormValue("remove_favicon") != "" {
		cfg.FaviconID = nil
	}
	if r.FormValue("remove_og") != "" {
		cfg.OGImageID = nil
	}

	var hatalar []string
	ustlendiLogo := false
	set := func(field string, dst **int64) {
		id, err := s.saveBrandImage(r, field)
		if err == nil {
			*dst = &id
			if field == "logo" || field == "icon" {
				ustlendiLogo = true
			}
			return
		}
		if !errors.Is(err, http.ErrMissingFile) {
			// Sessizce yutma: kullanıcı hatayı görsün (eski davranış: log + "kaydedildi").
			s.logger.Warn("marka görsel yüklenemedi", "alan", field, "error", err)
			hatalar = append(hatalar, field+": "+err.Error())
		}
	}
	set("icon", &cfg.IconID)
	set("logo", &cfg.ImageID)
	set("logo_acik", &cfg.ImageAcikID)
	set("favicon", &cfg.FaviconID)
	set("og", &cfg.OGImageID)

	// Yeni logo yüklendiyse mod otomatik "görsel"e geçer — aksi halde
	// kullanıcı dosyayı yükler ama metin logo görünmeye devam ederdi.
	if ustlendiLogo && cfg.Mode != "image" {
		cfg.Mode = "image"
	}
	// Görsel modu seçili ama görsel yoksa metin moduna düş (resim yokken yazı logo)
	if cfg.Mode == "image" && cfg.ImageID == nil && cfg.IconID == nil {
		cfg.Mode = "text"
	}

	raw, err := json.Marshal(cfg)
	if err != nil {
		s.brandResult(w, r, "error", "Kaydedilemedi: "+err.Error())
		return
	}
	if err := repo.SetSetting(ctx, s.pool, "brand", "ana", raw); err != nil {
		s.brandResult(w, r, "error", "Kaydedilemedi: "+err.Error())
		return
	}
	// Site kimliği: başlık + slogan (Ayarlar→Genel'den taşındı; tek merkez).
	if r.Form.Has("site_baslik") {
		s.siteBaslikKaydet(ctx, r.FormValue("site_baslik"))
	}
	if slogan := strings.TrimSpace(r.FormValue("slogan")); r.Form.Has("slogan") {
		_ = repo.SetSetting(ctx, s.pool, "genel", "slogan", mustJSON(map[string]string{"tr": slogan}))
	}
	s.audit(r, "marka_guncelle", "brand")
	s.invalidateContentCache()
	s.cdShellCache.Clear()
	if s.sayfaCache != nil {
		s.sayfaCache.temizle()
	}
	if len(hatalar) > 0 {
		s.brandResult(w, r, "error", "Kısmen kaydedildi — yüklenemeyen: "+strings.Join(hatalar, "; "))
		return
	}
	not := "Marka ve logo ayarları kaydedildi."
	if ustlendiLogo {
		not += " Görsel logo modu otomatik açıldı; site anında güncellendi (sayfayı yenilemeniz yeterli)."
	}
	s.brandResult(w, r, "info", not)
}

// mustJSON JSON'a çevirir; hata durumunda boş nesne döner.
func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte("{}")
	}
	return b
}

func sinirlaInt(ham string, alt, ust, varsayilan int) int {
	ham = strings.TrimSpace(ham)
	if ham == "" {
		return varsayilan
	}
	n, err := strconv.Atoi(ham)
	if err != nil {
		return varsayilan
	}
	if n < alt {
		return alt
	}
	if n > ust {
		return ust
	}
	return n
}
