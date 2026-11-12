package httpapi

import (
	"context"
	"encoding/json"
	"html/template"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"theblock/site/internal/repo"
)

const (
	layoutSection = "olcu"
	layoutKey     = "ana"
)

type LayoutConfig struct {
	// SiteWidth üst menü, gövde ve footer için ortak azami içerik genişliği.
	// Boşsa public.css fallback'i 1280px kullanılır.
	SiteWidth   *int   `json:"siteWidth"`
	GapGrid     *int   `json:"gapGrid"`
	GapColumn   *int   `json:"gapColumn"`
	GapSide     *int   `json:"gapSide"`
	GapSmall    *int   `json:"gapSmall"`
	SidePad     *int   `json:"sidePad"`   // sağ sütun sol dolgusu (px)
	SideLineW   *int   `json:"sideLineW"` // sağ sütun dikey ayraç çizgisi (px)
	BorderW     *int   `json:"borderWidth"`
	Radius      *int   `json:"radius"`
	BorderColor string `json:"borderColor"`
}

// layoutConfig ayardan okur; yoksa boş (tüm varsayılan) döndürür.
func (s *Server) layoutConfig(ctx context.Context) *LayoutConfig {
	c := &LayoutConfig{}
	if v := s.cachedSetting(ctx, layoutSection, layoutKey); len(v) > 0 {
		_ = json.Unmarshal(v, c)
	}
	return c
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func px(n int) string { return strconv.Itoa(n) + "px" }

var hexColorRe = regexp.MustCompile(`^#([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$`)

// sanitizeHexColor yalnızca geçerli hex renkleri kabul eder (CSS enjeksiyonu
// imkânsız); geçersiz/boş değer "" döner.
func sanitizeHexColor(v string) string {
	v = strings.TrimSpace(v)
	if hexColorRe.MatchString(v) {
		return v
	}
	return ""
}

// layoutTokensCSS public_head'te :root üzerine yazılacak CSS değişkenlerini
// üretir. Hiçbir alan ayarlanmamışsa boş döner (public.css fallback'leri).
func (s *Server) layoutTokensCSS(ctx context.Context) template.CSS {
	c := s.layoutConfig(ctx)
	var p []string
	if c.SiteWidth != nil {
		p = append(p, "--site-width:"+px(clampInt(*c.SiteWidth, 960, 1600)))
	}
	if c.GapGrid != nil {
		p = append(p, "--tb-gap:"+px(clampInt(*c.GapGrid, 0, 80)))
	}
	if c.GapColumn != nil {
		p = append(p, "--tb-gap-col:"+px(clampInt(*c.GapColumn, 0, 80)))
	}
	if c.GapSide != nil {
		p = append(p, "--tb-gap-side:"+px(clampInt(*c.GapSide, 0, 80)))
	}
	if c.GapSmall != nil {
		p = append(p, "--tb-gap-sm:"+px(clampInt(*c.GapSmall, 0, 80)))
	}
	if c.SidePad != nil {
		p = append(p, "--tb-side-pad:"+px(clampInt(*c.SidePad, 0, 80)))
	}
	if c.SideLineW != nil {
		p = append(p, "--tb-side-bw:"+px(clampInt(*c.SideLineW, 0, 8)))
	}
	if c.BorderW != nil {
		p = append(p, "--tb-bw:"+px(clampInt(*c.BorderW, 0, 8)))
	}
	if c.Radius != nil {
		r := clampInt(*c.Radius, 0, 40)
		sm := r - 2
		if sm < 0 {
			sm = 0
		}
		p = append(p, "--hp-radius:"+px(r), "--hp-radius-sm:"+px(sm))
	}
	if bc := sanitizeHexColor(c.BorderColor); bc != "" {
		p = append(p, "--hp-border:"+bc, "--hp-border-2:"+bc)
	}
	if len(p) == 0 {
		return ""
	}
	return template.CSS(":root{" + strings.Join(p, ";") + ";}")
}

// layoutView tasarımcı arayüzü için ölçü/kenarlık verisi (form ön-doldurma).
type layoutView struct {
	Config      *LayoutConfig
	SiteWidth   int
	GapGrid     int
	GapColumn   int
	GapSide     int
	GapSmall    int
	SidePad     int
	SideLineW   int
	BorderW     int
	Radius      int
	BorderColor string
	HasColor    bool
	HasCustom   bool
}

// layoutViewFor efektif değerleri (ayar yoksa varsayılanlar) hesaplar.
func (s *Server) layoutViewFor(ctx context.Context) *layoutView {
	c := s.layoutConfig(ctx)
	v := &layoutView{
		Config: c, SiteWidth: 1280, GapGrid: 12, GapColumn: 12, GapSide: 10, GapSmall: 8,
		SidePad: 12, SideLineW: 1, BorderW: 1, Radius: 8,
	}
	set := func(dst *int, src *int) {
		if src != nil {
			*dst = *src
			v.HasCustom = true
		}
	}
	set(&v.SiteWidth, c.SiteWidth)
	set(&v.GapGrid, c.GapGrid)
	set(&v.GapColumn, c.GapColumn)
	set(&v.GapSide, c.GapSide)
	set(&v.GapSmall, c.GapSmall)
	set(&v.SidePad, c.SidePad)
	set(&v.SideLineW, c.SideLineW)
	set(&v.BorderW, c.BorderW)
	set(&v.Radius, c.Radius)
	if c.BorderColor != "" {
		v.BorderColor = c.BorderColor
		v.HasColor = true
		v.HasCustom = true
	}
	return v
}

// parseIntPtr boş metinde nil, aksi halde ayrıştırılmış tamsayı döndürür.
func parseIntPtr(s string) *int {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return nil
	}
	return &n
}

// handleLayoutSave POST /admin/olcu/kaydet — ölçü/kenarlık ayarlarını kaydeder.
func (s *Server) handleLayoutSave(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	back := "/admin/tasarim?sayfa=" + builderPage(r)
	// Hem urlencoded (arayüz formu) hem multipart (test) gövdeyi oku:
	// FormValue gerekirse ParseMultipartForm'u kendisi çağırır.
	r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/") {
		_ = r.ParseMultipartForm(1 << 20)
	} else {
		_ = r.ParseForm()
	}

	cfg := &LayoutConfig{}
	if r.FormValue("reset") == "" {
		cfg.SiteWidth = parseIntPtr(r.FormValue("siteWidth"))
		cfg.GapGrid = parseIntPtr(r.FormValue("gapGrid"))
		cfg.GapColumn = parseIntPtr(r.FormValue("gapColumn"))
		cfg.GapSide = parseIntPtr(r.FormValue("gapSide"))
		cfg.GapSmall = parseIntPtr(r.FormValue("gapSmall"))
		cfg.SidePad = parseIntPtr(r.FormValue("sidePad"))
		cfg.SideLineW = parseIntPtr(r.FormValue("sideLineW"))
		cfg.BorderW = parseIntPtr(r.FormValue("borderWidth"))
		cfg.Radius = parseIntPtr(r.FormValue("radius"))
		if r.FormValue("borderColorEnabled") != "" {
			cfg.BorderColor = sanitizeHexColor(r.FormValue("borderColor"))
		}
	}

	raw, err := json.Marshal(cfg)
	if err != nil {
		s.flashRedirect(w, r, back, "error", "Kaydedilemedi: "+err.Error())
		return
	}
	if err := repo.SetSetting(ctx, s.pool, layoutSection, layoutKey, raw); err != nil {
		s.flashRedirect(w, r, back, "error", "Kaydedilemedi.")
		return
	}
	s.audit(r, "olcu_guncelle", "olcu")
	s.invalidateContentCache()

	msg := "Ölçü ve kenarlık ayarları kaydedildi."
	if r.FormValue("reset") != "" {
		msg = "Ölçü ve kenarlık ayarları varsayılana döndürüldü."
	}
	s.flashRedirect(w, r, back, "info", msg)
}

func (s *Server) handleLayoutPartial(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_ = r.ParseForm()
	key := strings.TrimSpace(r.FormValue("key"))
	val := strings.TrimSpace(r.FormValue("value"))

	cfg := s.layoutConfig(ctx)
	num := func(dst **int, lo, hi int) {
		if val == "" {
			*dst = nil // boş = varsayılan
			return
		}
		n, err := strconv.Atoi(val)
		if err != nil {
			return
		}
		n = clampInt(n, lo, hi)
		*dst = &n
	}
	switch key {
	case "reset":
		cfg = &LayoutConfig{} // tümü varsayılan
	case "borderColor":
		if val == "" {
			cfg.BorderColor = ""
		} else if c := sanitizeHexColor(val); c != "" {
			cfg.BorderColor = c
		}
	case "siteWidth":
		num(&cfg.SiteWidth, 960, 1600)
	case "gapGrid":
		num(&cfg.GapGrid, 0, 80)
	case "gapColumn":
		num(&cfg.GapColumn, 0, 80)
	case "gapSide":
		num(&cfg.GapSide, 0, 80)
	case "gapSmall":
		num(&cfg.GapSmall, 0, 80)
	case "sidePad":
		num(&cfg.SidePad, 0, 80)
	case "sideLineW":
		num(&cfg.SideLineW, 0, 8)
	case "borderWidth":
		num(&cfg.BorderW, 0, 8)
	case "radius":
		num(&cfg.Radius, 0, 40)
	default:
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "geçersiz anahtar"})
		return
	}

	raw, err := json.Marshal(cfg)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	if err := repo.SetSetting(ctx, s.pool, layoutSection, layoutKey, raw); err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "kaydedilemedi"})
		return
	}
	s.invalidateContentCache()
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
