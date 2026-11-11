package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"html/template"
	"net/http"
	"path/filepath"
	"strings"

	"theblock/site/internal/platform"
	"theblock/site/internal/repo"
)

// ---- Site arka planı (Sayfa Tasarımcısı içinde yönetilir) ----
//
// Public sayfalarda gövde (body) arka plan görseli varsayılan olarak
// statik "ana-bg.svg" desenidir (bkz. partials.html). Bu ayar doluysa
// yöneticinin yüklediği görsel — SVG veya raster (jpg/png/webp/gif) —
// onun yerine kullanılır.

const (
	bgSection = "arkaplan"
	bgKey     = "ana"
)

// BackgroundConfig arka plan görseli ve yerleşim ayarları.
//
//	MediaID: yüklenen görselin medya kimliği (nil = varsayılan desen)
//	Disabled: true ise arka plan görseli tamamen kapalıdır (desen ve özel
//	          görsel gösterilmez; yalnızca tema renk zemini kalır)
//	Size/Repeat/Position/Attach: CSS background-* değerleri (allowlist)
type BackgroundConfig struct {
	MediaID  *int64 `json:"mediaId"`
	Disabled bool   `json:"disabled"`
	Size     string `json:"size"`
	Repeat   string `json:"repeat"`
	Position string `json:"position"`
	Attach   string `json:"attach"`
}

// backgroundConfig ayardan okur; yoksa boş (varsayılan desen) döndürür.
func (s *Server) backgroundConfig(ctx context.Context) *BackgroundConfig {
	c := &BackgroundConfig{}
	if v := s.cachedSetting(ctx, bgSection, bgKey); len(v) > 0 {
		_ = json.Unmarshal(v, c)
	}
	c.normalize()
	return c
}

// normalize geçersiz/boş değerleri güvenli varsayılanlara çeker. Değerler
// doğrudan CSS'e gömüldüğü için yalnızca allowlist'teki değerler geçer
// (serbest metin CSS enjeksiyonuna yol açamaz).
func (c *BackgroundConfig) normalize() {
	c.Size = bgPick(c.Size, []string{"cover", "contain", "auto", "100% 100%"}, "cover")
	c.Repeat = bgPick(c.Repeat, []string{"no-repeat", "repeat", "repeat-x", "repeat-y"}, "no-repeat")
	c.Position = bgPick(c.Position, []string{"center top", "center", "top", "bottom", "left center", "right center"}, "center top")
	c.Attach = bgPick(c.Attach, []string{"fixed", "scroll"}, "fixed")
}

func bgPick(v string, allowed []string, def string) string {
	v = strings.TrimSpace(v)
	for _, a := range allowed {
		if v == a {
			return v
		}
	}
	return def
}

// siteBgCSS public sayfalarda body arka planı için inline <style> içeriğini
// üretir. Üç durum vardır:
//   - kapalı (Disabled): görsel yok (background-image:none)
//   - özel görsel (MediaID): yöneticinin yüklediği görsel
//   - aksi halde boş döner; şablon varsayılan desene (ana-bg.svg) düşer.
//
// Dönüş tipi template.CSS'tir; html/template yalnızca bu türü <style>
// (CSS bağlamı) içinde güvenli kabul eder.
func (s *Server) siteBgCSS(ctx context.Context) template.CSS {
	c := s.backgroundConfig(ctx)
	if c.Disabled {
		return template.CSS("body{background-image:none;}")
	}
	if c.MediaID == nil || *c.MediaID <= 0 {
		return ""
	}
	u := s.mediaURL(c.MediaID)
	if u == "" || u == "/media/none.svg" {
		return ""
	}
	// URL yol biçimindedir (/media/yyyymm/dosya); yine de tırnak/kapanış
	// gibi CSS'i bozabilecek karakterler kaçırılır.
	u = strings.NewReplacer(`"`, "%22", `'`, "%27", `(`, "%28", `)`, "%29").Replace(u)
	css := "body{background-image:url(\"" + u + "\");" +
		"background-size:" + c.Size + ";" +
		"background-repeat:" + c.Repeat + ";" +
		"background-position:" + c.Position + ";" +
		"background-attachment:" + c.Attach + ";}"
	return template.CSS(css) // #nosec G203 - değerler allowlist + kaçışlanmış yol
}

// bgView tasarımcı arayüzü için arka plan önizleme verisi.
type bgView struct {
	Config    *BackgroundConfig
	URL       string // özel görsel varsa önizleme adresi
	HasCustom bool
}

// bgViewFor tasarımcı sayfası için mevcut arka planı hazırlar.
func (s *Server) bgViewFor(ctx context.Context) *bgView {
	c := s.backgroundConfig(ctx)
	v := &bgView{Config: c}
	if c.MediaID != nil && *c.MediaID > 0 {
		if u := s.mediaURL(c.MediaID); u != "" && u != "/media/none.svg" {
			v.URL = u
			v.HasCustom = true
		}
	}
	return v
}

// detectSVG yüklenen verinin SVG olup olmadığını belirler (uzantı veya
// içerik ipucu ile).
func detectSVG(ext string, data []byte) bool {
	if strings.EqualFold(ext, ".svg") {
		return true
	}
	head := data
	if len(head) > 512 {
		head = head[:512]
	}
	ls := strings.ToLower(string(head))
	return strings.Contains(ls, "<svg")
}

// isGIF verinin GIF imzası ("GIF87a"/"GIF89a") ile başladığını doğrular.
func isGIF(data []byte) bool {
	return len(data) >= 6 && string(data[:4]) == "GIF8"
}

// saveBackgroundImage yüklenen arka plan görselini medyaya kaydeder.
// Raster (jpg/png/webp/gif) ve SVG desteklenir; SVG etkin içerikten
// arındırılır (platform.SanitizeSVG).
func (s *Server) saveBackgroundImage(r *http.Request, data []byte, filename string) (int64, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	var mime string
	width, height := 0, 0

	if detectSVG(ext, data) {
		clean, err := platform.SanitizeSVG(data)
		if err != nil {
			return 0, err
		}
		data = clean
		mime = "image/svg+xml"
		if ext != ".svg" {
			filename = strings.TrimSuffix(filename, filepath.Ext(filename)) + ".svg"
		}
	} else if ext == ".gif" || isGIF(data) {
		// GIF'ler (animasyonlu dahil) çözülmeden olduğu gibi saklanır;
		// animasyon korunur. Yalnızca imza doğrulanır.
		if !isGIF(data) {
			return 0, errors.New("geçersiz GIF dosyası")
		}
		mime = "image/gif"
		if ext != ".gif" {
			filename = strings.TrimSuffix(filename, filepath.Ext(filename)) + ".gif"
		}
	} else {
		decoded, err := platform.DecodeImage(data)
		if err != nil {
			return 0, errors.New("desteklenmeyen görsel (jpg, png, webp, gif veya svg yükleyin)")
		}
		b := decoded.Bounds()
		width, height = b.Dx(), b.Dy()
		m, _ := platformFileExt(filename)
		mime = m
	}

	storedPath, err := platform.SaveMedia(data, filename)
	if err != nil {
		return 0, errors.New("depolama hatası")
	}
	size := int64(len(data))
	u := userFromContext(r.Context())
	var wp, hp *int
	if width > 0 {
		wp = &width
	}
	if height > 0 {
		hp = &height
	}
	mid, err := repo.CreateMedia(r.Context(), s.pool, &repo.Media{
		SiteID: s.curSiteID(r), UserID: u.ID, Kind: "image", Filename: filename, Path: storedPath,
		Mime: mime, SizeBytes: &size, Width: wp, Height: hp, Source: "upload",
	})
	if err != nil {
		s.logger.Error("arka plan medya kaydı", "error", err)
		return 0, errors.New("medya kaydedilemedi")
	}
	return mid, nil
}

// handleBackgroundSave POST /admin/arka-plan/kaydet — multipart form.
// Görsel (isteğe bağlı), yerleşim ayarları ve "varsayılana dön" bayrağını
// işler; tasarımcı sayfasına geri yönlendirir.
func (s *Server) handleBackgroundSave(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	back := "/admin/tasarim?sayfa=" + builderPage(r)
	r.Body = http.MaxBytesReader(w, r.Body, 30<<20)
	if err := r.ParseMultipartForm(40 << 20); err != nil {
		s.flashRedirect(w, r, back, "error", "Form okunamadı (dosya çok büyük olabilir).")
		return
	}

	cfg := s.backgroundConfig(ctx)
	cfg.Size = r.FormValue("size")
	cfg.Repeat = r.FormValue("repeat")
	cfg.Position = r.FormValue("position")
	cfg.Attach = r.FormValue("attach")
	cfg.normalize()

	if r.FormValue("remove_bg") != "" {
		cfg.MediaID = nil
		cfg.Disabled = false
	}
	if r.FormValue("disable_bg") != "" {
		cfg.MediaID = nil
		cfg.Disabled = true
	}

	f, hdr, err := r.FormFile("arka")
	switch {
	case err == nil:
		defer f.Close()
		data := make([]byte, hdr.Size)
		if _, err := readFull(f, data); err != nil {
			s.flashRedirect(w, r, back, "error", "Dosya okunamadı.")
			return
		}
		id, err := s.saveBackgroundImage(r, data, hdr.Filename)
		if err != nil {
			s.flashRedirect(w, r, back, "error", err.Error())
			return
		}
		cfg.MediaID = &id
		cfg.Disabled = false
	case errors.Is(err, http.ErrMissingFile):
		// dosya seçilmedi: yalnızca ayarlar güncellenir
	default:
		s.flashRedirect(w, r, back, "error", "Dosya alınamadı.")
		return
	}

	raw, err := json.Marshal(cfg)
	if err != nil {
		s.flashRedirect(w, r, back, "error", "Kaydedilemedi: "+err.Error())
		return
	}
	if err := repo.SetSetting(ctx, s.pool, bgSection, bgKey, raw); err != nil {
		s.flashRedirect(w, r, back, "error", "Kaydedilemedi.")
		return
	}
	s.audit(r, "arka_plan_guncelle", "arkaplan")
	s.invalidateContentCache()

	msg := "Arka plan kaydedildi."
	switch {
	case cfg.Disabled:
		msg = "Arka plan kapatıldı (görsel gösterilmiyor)."
	case cfg.MediaID == nil:
		msg = "Arka plan varsayılan desene döndürüldü."
	}
	s.flashRedirect(w, r, back, "info", msg)
}
