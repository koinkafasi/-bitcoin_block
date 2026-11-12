package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"theblock/site/internal/repo"
)

const menuBgKey = "mega_arkaplan"

type MenuBackgroundConfig struct {
	MediaID  *int64 `json:"mediaId"`
	Disabled bool   `json:"disabled"`
	Size     string `json:"size"`
	Repeat   string `json:"repeat"`
	Position string `json:"position"`
	Attach   string `json:"attach"`
	// Overlay 0-100 arası beyaz okunabilirlik katmanı (yüzde).
	Overlay *int `json:"overlay,omitempty"`
}

func (c *MenuBackgroundConfig) normalize() {
	c.Size = bgPick(c.Size, []string{"cover", "contain", "auto", "100% 100%"}, "cover")
	c.Repeat = bgPick(c.Repeat, []string{"no-repeat", "repeat", "repeat-x", "repeat-y"}, "no-repeat")
	c.Position = bgPick(c.Position, []string{"center top", "center", "top", "bottom", "left center", "right center"}, "center")
	c.Attach = bgPick(c.Attach, []string{"scroll", "fixed"}, "scroll")
	if c.Overlay == nil {
		v := 82
		c.Overlay = &v
	} else if *c.Overlay < 0 {
		*c.Overlay = 0
	} else if *c.Overlay > 100 {
		*c.Overlay = 100
	}
}

func (c *MenuBackgroundConfig) overlay() int {
	if c.Overlay == nil {
		return 82
	}
	return *c.Overlay
}

func (s *Server) menuBackgroundConfig(ctx context.Context) *MenuBackgroundConfig {
	c := &MenuBackgroundConfig{}
	if v := s.cachedSetting(ctx, "menu", menuBgKey); len(v) > 0 {
		_ = json.Unmarshal(v, c)
	}
	c.normalize()
	return c
}

type menuBgView struct {
	Config    *MenuBackgroundConfig
	URL       string
	HasCustom bool
}

func (s *Server) menuBgViewFor(ctx context.Context) *menuBgView {
	c := s.menuBackgroundConfig(ctx)
	v := &menuBgView{Config: c}
	if c.MediaID != nil && *c.MediaID > 0 {
		if u := s.mediaURL(c.MediaID); u != "" && u != "/media/none.svg" {
			v.URL = u
			v.HasCustom = true
		}
	}
	return v
}

func (s *Server) menuBgCSS(ctx context.Context) template.CSS {
	c := s.menuBackgroundConfig(ctx)
	if c.Disabled || c.MediaID == nil || *c.MediaID <= 0 {
		return ""
	}
	u := s.mediaURL(c.MediaID)
	if u == "" || u == "/media/none.svg" {
		return ""
	}
	u = strings.NewReplacer(`"`, "%22", `'`, "%27", `(`, "%28", `)`, "%29").Replace(u)
	o := float64(c.overlay()) / 100
	// Beyaz katman metni okunur tutar; 0 seçilirse görsel tamamen görünür.
	css := fmt.Sprintf(`.nav-mega-sayfa .nav-mega-inner{background-image:linear-gradient(rgba(255,255,255,%.2f),rgba(255,255,255,%.2f)),url("%s");background-size:%s;background-repeat:%s;background-position:%s;background-attachment:%s;}`,
		o, o, u, c.Size, c.Repeat, c.Position, c.Attach)
	return template.CSS(css) // #nosec G203 - allowlist + kaçışlanmış medya yolu
}

// handleMenuBackgroundSave POST /admin/menu/tasarim/arka-plan
func (s *Server) handleMenuBackgroundSave(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	back := "/admin/menu"
	r.Body = http.MaxBytesReader(w, r.Body, 30<<20)
	if err := r.ParseMultipartForm(40 << 20); err != nil {
		s.flashRedirect(w, r, back, "error", "Form okunamadı (dosya çok büyük olabilir).")
		return
	}

	cfg := s.menuBackgroundConfig(ctx)
	cfg.Size = r.FormValue("size")
	cfg.Repeat = r.FormValue("repeat")
	cfg.Position = r.FormValue("position")
	cfg.Attach = r.FormValue("attach")
	if n, err := strconv.Atoi(strings.TrimSpace(r.FormValue("overlay"))); err == nil {
		cfg.Overlay = &n
	}
	cfg.normalize()

	if r.FormValue("remove_bg") != "" {
		cfg.MediaID = nil
		cfg.Disabled = false
	}
	if r.FormValue("disable_bg") != "" {
		cfg.MediaID = nil
		cfg.Disabled = true
	}

	f, hdr, err := r.FormFile("mega_arka")
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
		// yalnızca ayarlar güncellenir
	default:
		s.flashRedirect(w, r, back, "error", "Dosya alınamadı.")
		return
	}

	raw, err := json.Marshal(cfg)
	if err != nil {
		s.flashRedirect(w, r, back, "error", "Kaydedilemedi: "+err.Error())
		return
	}
	if err := repo.SetSetting(ctx, s.pool, "menu", menuBgKey, raw); err != nil {
		s.flashRedirect(w, r, back, "error", "Kaydedilemedi.")
		return
	}
	s.audit(r, "menu_arka_plan_guncelle", menuBgKey)
	s.invalidateContentCache()

	msg := "Mega menü arka planı kaydedildi."
	switch {
	case cfg.Disabled:
		msg = "Mega menü arka planı kapatıldı."
	case cfg.MediaID == nil:
		msg = "Mega menü arka planı kaldırıldı (beyaz zemin)."
	}
	s.flashRedirect(w, r, back, "info", msg)
}
