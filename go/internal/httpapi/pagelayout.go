package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"theblock/site/internal/repo"
)

// ---- Sayfa şablonu (sütun düzeni) ----
//
// Sayfa Tasarımcısı'nda her sayfa (ana sayfa, yazı sayfası, kategori ve
// kategori:<slug>) için sağ sütunun gösterilip gösterilmeyeceği, sağ sütun
// genişliği ve tek sütun içerik genişliği ayrı ayrı saklanır. Ayarlar global
// ölçü token'larından (olcu/ana) bağımsızdır; böylece aynı sitede ana sayfa
// iki sütun, yazı sayfası tek sütun olabilir.

const pageLayoutSection = "sayfa_duzen"

// pageLayoutConfig settings tablosunda saklanan ham şablon ayarıdır.
// Nil alan "varsayılanı kullan" anlamına gelir.
type pageLayoutConfig struct {
	Sidebar      *bool `json:"sidebar,omitempty"`
	SidebarWidth *int  `json:"sidebarWidth,omitempty"`
	ContentWidth *int  `json:"contentWidth,omitempty"`
	// Kategori sayfası üst alanı (sayfaya özel): çerçeveli başlık, açıklama
	// ve kırıntı görsel olarak gizlenebilir. Metin SEO/erişilebilirlik için
	// DOM'da kalır (sr-only).
	HideTitle      *bool `json:"hideTitle,omitempty"`
	HideDesc       *bool `json:"hideDesc,omitempty"`
	HideBreadcrumb *bool `json:"hideBreadcrumb,omitempty"`
	// BlockGap modüller arası dikey boşluk (CSS uzunluğu: 12px, 1rem…).
	// Boş/nil = modüllerin kendi kenar boşlukları kullanılır.
	BlockGap *string `json:"blockGap,omitempty"`
}

// pageLayoutView şablon ve yönetim arayüzü için çözümlenmiş değerler.
type pageLayoutView struct {
	Sidebar      bool
	SidebarWidth int
	MainWidth    int
	ContentWidth int
	// Kategori sayfası üst alanı görünürlüğü (yalnız category şablonunda
	// etkilidir): true = ilgili alan görsel gizli, DOM'da sr-only kalır.
	HideTitle      bool
	HideDesc       bool
	HideBreadcrumb bool
	// BlockGap modüller arası dikey boşluk (boş = varsayılan).
	BlockGap string
	// HasCustom bu sayfa anahtarında kayıtlı bir ayar var mı?
	HasCustom bool
	// UsingGlobal kategori:<slug> için genel kategori ayarı kullanılıyorsa true.
	UsingGlobal bool
}

// pageLayoutBaseKey kategori:<slug> için varsayılan/genel anahtarı döndürür.
func pageLayoutBaseKey(page string) string {
	if strings.HasPrefix(page, "kategori:") {
		return "kategori"
	}
	return page
}

// defaultSingleWidth tek sütun görünümünde içeriğin rahat okunabilir azami
// genişliği. Yazı sayfasında 960 px; ana sayfa/kategoride site genişliği.
func defaultSingleWidth(page string) int {
	if pageLayoutBaseKey(page) == "article" {
		return 960
	}
	return 1280
}

// readPageLayout kayıtlı ayarı okur; boş/{} kayıt yok sayılır.
func (s *Server) readPageLayout(ctx context.Context, page string) (pageLayoutConfig, bool) {
	var c pageLayoutConfig
	raw := s.cachedSetting(ctx, pageLayoutSection, page)
	if len(raw) == 0 || strings.TrimSpace(string(raw)) == "{}" {
		return c, false
	}
	if err := json.Unmarshal(raw, &c); err != nil {
		return pageLayoutConfig{}, false
	}
	return c, true
}

// pageLayoutViewFrom ham ayarı doğrulanmış görünüme çevirir. Saf fonksiyon:
// varsayılanlar, sınırlar ve türetilmiş ana sütun genişliği burada uygulanır.
func pageLayoutViewFrom(page string, cfg pageLayoutConfig, custom, usingGlobal bool) *pageLayoutView {
	v := &pageLayoutView{
		Sidebar:      true,
		SidebarWidth: 33,
		ContentWidth: defaultSingleWidth(page),
		HasCustom:    custom,
		UsingGlobal:  usingGlobal,
	}
	if cfg.Sidebar != nil {
		v.Sidebar = *cfg.Sidebar
	}
	if cfg.SidebarWidth != nil {
		v.SidebarWidth = clampInt(*cfg.SidebarWidth, 20, 50)
	}
	if cfg.ContentWidth != nil {
		v.ContentWidth = clampInt(*cfg.ContentWidth, 680, 1600)
	}
	if cfg.HideTitle != nil {
		v.HideTitle = *cfg.HideTitle
	}
	if cfg.HideDesc != nil {
		v.HideDesc = *cfg.HideDesc
	}
	if cfg.HideBreadcrumb != nil {
		v.HideBreadcrumb = *cfg.HideBreadcrumb
	}
	if cfg.BlockGap != nil {
		v.BlockGap = sanitizeLengthValue(*cfg.BlockGap, false, false)
	}
	if !v.Sidebar {
		v.SidebarWidth = 0
	}
	if v.SidebarWidth > 0 {
		v.MainWidth = 100 - v.SidebarWidth
	} else {
		v.MainWidth = 100
	}
	return v
}

// mergePageLayout alan bazlı miras: yalnız override'da (kategoriye özel
// kayıt) bulunan alanlar tabanın (genel kategori düzeni) yerine geçer;
// olmayan alanlar tabandan gelir. Böylece genel düzende yapılan bir gizleme
// tüm kategorilere yansır; kategori yalnız gerçekten değiştirdiği alanda
// istisna olur.
func mergePageLayout(base, over pageLayoutConfig) pageLayoutConfig {
	m := base
	if over.Sidebar != nil {
		m.Sidebar = over.Sidebar
	}
	if over.SidebarWidth != nil {
		m.SidebarWidth = over.SidebarWidth
	}
	if over.ContentWidth != nil {
		m.ContentWidth = over.ContentWidth
	}
	if over.HideTitle != nil {
		m.HideTitle = over.HideTitle
	}
	if over.HideDesc != nil {
		m.HideDesc = over.HideDesc
	}
	if over.HideBreadcrumb != nil {
		m.HideBreadcrumb = over.HideBreadcrumb
	}
	if over.BlockGap != nil {
		m.BlockGap = over.BlockGap
	}
	return m
}

// pageLayoutFor efektif sayfa şablonunu döndürür. Kategoriye özel ayar yoksa
// genel kategori ayarına düşer; kısmi kayıt varsa alan bazlı miraslanır.
func (s *Server) pageLayoutFor(ctx context.Context, page string) *pageLayoutView {
	cfg, custom := s.readPageLayout(ctx, page)
	usingGlobal := false
	if strings.HasPrefix(page, "kategori:") {
		if base, ok := s.readPageLayout(ctx, "kategori"); ok {
			if custom {
				cfg = mergePageLayout(base, cfg)
			} else {
				cfg = base
				usingGlobal = true
			}
		}
	}
	return pageLayoutViewFrom(page, cfg, custom, usingGlobal)
}

// handleBuilderPageLayoutGet sayfa şablonunun ham kaydını döndürür (yedekleme
// ve arayüz için). Alan bazlı miras açıkça belirtilir: config yalnız bu
// sayfada kayıtlı alanları içerir.
func (s *Server) handleBuilderPageLayoutGet(w http.ResponseWriter, r *http.Request) {
	page := builderPage(r)
	cfg, custom := s.readPageLayout(r.Context(), page)
	raw, err := json.Marshal(cfg)
	if err != nil {
		raw = []byte("{}")
	}
	s.writeJSON(w, http.StatusOK, map[string]any{
		"ok": true, "page": page,
		"hasCustom":   custom,
		"usingGlobal": strings.HasPrefix(page, "kategori:") && !custom,
		"config":      json.RawMessage(raw),
		"layout":      s.pageLayoutFor(r.Context(), page),
	})
}

// handleBuilderPageLayoutSave sayfa şablonu ayarını kaydeder. Gelen alanlar
// mevcut kaydın üzerine yamalanır (alan bazlı miras korunur); böylece tek bir
// ayarı değiştirmek diğer alanları genel düzenden kopardığı için sabitlemez.
func (s *Server) handleBuilderPageLayoutSave(w http.ResponseWriter, r *http.Request) {
	page := builderPage(r)
	var req struct {
		Reset bool `json:"reset"`
		pageLayoutConfig
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "geçersiz istek: " + err.Error()})
		return
	}

	var raw []byte
	if req.Reset {
		raw = []byte("{}")
	} else {
		cur, _ := s.readPageLayout(r.Context(), page)
		cfg := mergePageLayout(cur, req.pageLayoutConfig)
		if cfg.SidebarWidth != nil {
			v := clampInt(*cfg.SidebarWidth, 20, 50)
			cfg.SidebarWidth = &v
		}
		if cfg.ContentWidth != nil {
			v := clampInt(*cfg.ContentWidth, 680, 1600)
			cfg.ContentWidth = &v
		}
		if cfg.BlockGap != nil {
			v := sanitizeLengthValue(*cfg.BlockGap, false, false)
			cfg.BlockGap = &v
		}
		var err error
		raw, err = json.Marshal(cfg)
		if err != nil {
			s.writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error()})
			return
		}
	}
	if err := repo.SetSetting(r.Context(), s.pool, pageLayoutSection, page, raw); err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	s.invalidateContentCache()
	s.audit(r, "sayfa_sablonu_kaydet", page)

	v := s.pageLayoutFor(r.Context(), page)
	s.writeJSON(w, http.StatusOK, map[string]any{
		"ok": true, "page": page,
		"layout": map[string]any{
			"sidebar":        v.Sidebar,
			"sidebarWidth":   v.SidebarWidth,
			"mainWidth":      v.MainWidth,
			"contentWidth":   v.ContentWidth,
			"hideTitle":      v.HideTitle,
			"hideDesc":       v.HideDesc,
			"hideBreadcrumb": v.HideBreadcrumb,
			"blockGap":       v.BlockGap,
			"hasCustom":      v.HasCustom,
			"usingGlobal":    v.UsingGlobal,
		},
	})
}
