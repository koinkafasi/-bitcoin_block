package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"theblock/site/internal/platform"
	"theblock/site/internal/repo"
)

// BrandConfig marka/logo ayarları.
//
//	Mode: "text" (metin logo) | "image" (görsel logo)
//	Text: metin logo (boşsa "theblock" varsayılanı)
//	ImageID/FaviconID/OGImageID: medya id'leri (nil = yok)
//	Tagline: header/footer altı slogan
type BrandConfig struct {
	Mode      string `json:"mode"`
	Text      string `json:"text"`
	ImageID   *int64 `json:"imageId"`
	FaviconID *int64 `json:"faviconId"`
	OGImageID *int64 `json:"ogImageId"`
	Tagline   string `json:"tagline"`
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

// saveBrandImage yüklenen görseli medyaya kaydeder ve medya id'sini döndürür.
// Görsel çözülebilirse ölçüleri kaydedilir; çözülemese de (egzotik webp vb.)
// dosya saklanır — logo kaybı sessizce yaşanmaz. Bilinen uzantı + okunabilir
// içerik yeterlidir.
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
	decoded, err := platform.DecodeImage(data)
	if err == nil {
		b := decoded.Bounds()
		width, height = b.Dx(), b.Dy()
	} else if !isKnownImageExt(hdr.Filename) {
		return 0, errors.New("desteklenmeyen görsel (jpg/png/webp)")
	} else {
		// Bilinen uzantı ama çözülemedi (ör. animasyonlu/egzotik webp):
		// dosyayı sakla, ölçüleri boş bırak.
		s.logger.Warn("marka görseli çözülemedi, ölçüsüz saklanıyor", "alan", field, "error", err)
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
	for _, ext := range []string{".jpg", ".jpeg", ".png", ".webp", ".gif", ".avif"} {
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

	if r.FormValue("remove_logo") != "" {
		cfg.ImageID = nil
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
			if field == "logo" {
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
	set("logo", &cfg.ImageID)
	set("favicon", &cfg.FaviconID)
	set("og", &cfg.OGImageID)

	// Yeni logo yüklendiyse mod otomatik "görsel"e geçer — aksi halde
	// kullanıcı dosyayı yükler ama metin logo görünmeye devam ederdi.
	if ustlendiLogo && cfg.Mode != "image" {
		cfg.Mode = "image"
	}
	// Görsel modu seçili ama görsel yoksa metin moduna düş (resim yokken yazı logo)
	if cfg.Mode == "image" && cfg.ImageID == nil {
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
	if baslik := strings.TrimSpace(r.FormValue("site_baslik")); r.Form.Has("site_baslik") {
		_ = repo.SetSetting(ctx, s.pool, "genel", "site_baslik", mustJSON(map[string]string{"tr": baslik}))
	}
	if slogan := strings.TrimSpace(r.FormValue("slogan")); r.Form.Has("slogan") {
		_ = repo.SetSetting(ctx, s.pool, "genel", "slogan", mustJSON(map[string]string{"tr": slogan}))
	}
	s.audit(r, "marka_guncelle", "brand")
	s.invalidateContentCache()
	if len(hatalar) > 0 {
		s.brandResult(w, r, "error", "Kısmen kaydedildi — yüklenemeyen: "+strings.Join(hatalar, "; "))
		return
	}
	not := "Marka ve logo ayarları kaydedildi."
	if ustlendiLogo {
		not += " Görsel logo modu otomatik açıldı. Site tarafında ~1 dakika içinde görünür (önbellek)."
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
