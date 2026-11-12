package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"theblock/site/internal/repo"
)

// HeaderButton header sağ tarafındaki isteğe bağlı CTA düğmesi.
type HeaderButton struct {
	Label string `json:"label"`
	URL   string `json:"url"`
	Style string `json:"style"` // primary | ghost
}

type HeaderConfig struct {
	Model       string `json:"model"`
	Skin        string `json:"skin"`  // auto | dark | light | accent
	Width       string `json:"width"` // normal | wide
	Sticky      bool   `json:"sticky"`
	Topbar      bool   `json:"topbar"`
	TopbarText  string `json:"topbar_text"`
	ShowTagline bool   `json:"show_tagline"`
	ShowSearch  bool   `json:"show_search"`
	ShowTheme   bool   `json:"show_theme"`
	ShowMenu    bool   `json:"show_menu"`
	ShowHome    bool   `json:"show_home"`
	// HideMember üye giriş/hesap simgesini gizler (varsayılan: görünür).
	// Ters anlamlı alan eski kayıtlı JSON'larda geriye dönük uyumluluk sağlar.
	HideMember bool           `json:"hide_member"`
	Buttons    []HeaderButton `json:"buttons"`
}

type headerEditData struct {
	Config *HeaderConfig
}

func (c *HeaderConfig) normalize() {
	switch c.Model {
	case "model1", "model2", "model3", "model4":
	default:
		c.Model = "model1"
	}
	switch c.Skin {
	case "auto", "dark", "light", "accent":
	default:
		c.Skin = "auto"
	}
	if c.Width != "wide" {
		c.Width = "normal"
	}
	if len(c.Buttons) > 4 {
		c.Buttons = c.Buttons[:4]
	}
	for i := range c.Buttons {
		c.Buttons[i].Label = trimSpace(c.Buttons[i].Label)
		c.Buttons[i].URL = trimSpace(c.Buttons[i].URL)
		c.Buttons[i].URL = normalizePageURL(c.Buttons[i].URL)
		if c.Buttons[i].Style != "ghost" {
			c.Buttons[i].Style = "primary"
		}
	}
}

func (s *Server) headerConfig(ctx context.Context) *HeaderConfig {
	if v := s.cachedSetting(ctx, "header", "ana"); len(v) > 0 {
		var c HeaderConfig
		if json.Unmarshal(v, &c) == nil {
			c.normalize()
			return &c
		}
	}
	c := &HeaderConfig{
		Model: "model1", Skin: "auto", Width: "normal",
		Sticky: true, ShowSearch: true, ShowTheme: true, ShowMenu: true,
	}
	return c
}

func (s *Server) handleHeaderGet(w http.ResponseWriter, r *http.Request) {
	d := s.buildHeaderData(r)
	if IsHTMX(r) {
		s.renderHeaderPanelV2(w, r, d)
		return
	}
	s.renderHeaderPageV2(w, r, d)
}

func (s *Server) buildHeaderData(r *http.Request) *tmplData {
	d := s.baseView("tr", "/admin", "Header Yönetimi")
	d.User = ptr(userFromContext(r.Context()))
	d.Flash = s.consumeFlash(r)
	d.HeaderEdit = &headerEditData{Config: s.headerConfig(r.Context())}
	return d
}

func (s *Server) renderHeaderPanel(w http.ResponseWriter, r *http.Request, kind, text string) {
	d := s.buildHeaderData(r)
	if kind != "" {
		d.Flash = &flashMsg{Kind: kind, Text: text}
	}
	s.renderHeaderPanelV2(w, r, d)
}

func (s *Server) handleHeaderSave(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var cfg HeaderConfig
	if IsHTMX(r) {
		_ = r.ParseForm()
		raw := r.FormValue("config")
		if raw == "" || json.Unmarshal([]byte(raw), &cfg) != nil {
			s.renderHeaderPanel(w, r, "error", "Header verisi geçersiz.")
			return
		}
	} else {
		defer r.Body.Close()
		var body struct {
			Config HeaderConfig `json:"config"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 2<<20)).Decode(&body); err != nil {
			s.writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "JSON geçersiz: " + err.Error()})
			return
		}
		cfg = body.Config
	}
	cfg.normalize()
	raw, err := json.Marshal(cfg)
	if err != nil {
		s.renderHeaderPanel(w, r, "error", err.Error())
		return
	}
	if err := repo.SetSetting(ctx, s.pool, "header", "ana", raw); err != nil {
		s.renderHeaderPanel(w, r, "error", err.Error())
		return
	}
	s.invalidateContentCache()
	if IsHTMX(r) {
		s.renderHeaderPanel(w, r, "info", "Header kaydedildi.")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
