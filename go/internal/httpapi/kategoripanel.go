package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"theblock/site/internal/repo"
)

// ---- Kategori Panelleri (haberpi tarzı alt bölümler) yönetimi ----
//
// Ana sayfadaki "Güncel / Siyaset / Ekonomi ..." panelleri artık her biri
// ayrı bir "kategori_panel" modülüdür. Modül ayarlarında kategori, vitrin
// düzeni (iki büyük | mozaik | büyük+liste …), renk ve haber sayısı bulunur;
// sıralama/görünürlük diğer modüller gibi Sayfa Tasarımcısı'ndan yönetilir.
//
// Eski tek bloklu "kategori_bloklari" düzeni geriye dönük uyumluluk için
// hâlâ render edilir. Tasarımcı açıldığında bu blok, kayıtlı panel ayarına
// göre tek tek "kategori_panel" bloklarına genişletilir (expandCategoryPanelBlocks).

const (
	panelsSection = "kategori_panelleri"
	panelsKey     = "ana"
)

// panelAyar tek kategori panelinin ayarı.
type panelAyar struct {
	Slug  string `json:"slug"`
	Duzen string `json:"duzen"` // iki | ust3alt2 | mozaik | solbuyuk | buyuk | ekonomi
	Tone  string `json:"tone"`  // yesil | kirmizi
	Show  bool   `json:"show"`
}

type panelConfig struct {
	Adet   int         `json:"adet"`
	Panels []panelAyar `json:"panels"`
}

// panelConfigOf ayarı okur; yoksa (nil,false).
func (s *Server) panelConfigOf(ctx context.Context) (*panelConfig, bool) {
	v := s.cachedSetting(ctx, panelsSection, panelsKey)
	if len(v) == 0 {
		return nil, false
	}
	var c panelConfig
	if json.Unmarshal(v, &c) != nil || len(c.Panels) == 0 {
		return nil, false
	}
	if c.Adet <= 0 {
		c.Adet = 9
	}
	return &c, true
}

// defaultPanelDuzen slug'a göre haberpi eşlemesi.
func defaultPanelDuzen(slug string) string {
	switch slug {
	case "guncel", "gundem":
		return "ust3alt2"
	case "ekonomi":
		// Habertürk Ekonomi vitrini: solda büyük kare kart, sağda 2x2 ızgara.
		return "ekonomi"
	case "gastro":
		// Habertürk GASTRO vitrini: yatay kaydırmalı 1:2 portre kartlar.
		return "gastro"
	case "siyaset":
		return "solbuyuk"
	default:
		return "mozaik"
	}
}

func validPanelDuzen(d string) bool {
	return d == "iki" || d == "mozaik" || d == "buyuk" || d == "solbuyuk" ||
		d == "ust3alt2" || d == "ekonomi" || d == "gastro"
}

// panelSatir tasarımcı arayüzü için tek kategori satırı.
type panelSatir struct {
	Slug  string
	Name  string
	Duzen string
	Tone  string
	Show  bool
}

// panelTon seçilebilir renk.
type panelTon struct {
	Key   string
	Label string
}

// panelTonlar seçilebilir renkler.
var panelTonlar = []panelTon{
	{"kirmizi", "Kırmızı (tema)"},
	{"yesil", "Yeşil"},
	{"mavi", "Mavi"},
	{"turuncu", "Turuncu"},
	{"mor", "Mor"},
	{"sari", "Sarı"},
	{"lacivert", "Lacivert"},
	{"gri", "Gri"},
}

func validPanelTone(t string) bool {
	for _, x := range panelTonlar {
		if x.Key == t {
			return true
		}
	}
	return false
}

type panelsView struct {
	Adet      int
	Satirlar  []panelSatir
	Tonlar    []panelTon
	HasCustom bool
}

// panelsViewFor tasarımcı kartı için mevcut durumu (tüm kategoriler, ayarlı
// sırayla) hazırlar.
func (s *Server) panelsViewFor(ctx context.Context, siteID int64) *panelsView {
	cats, _ := repo.ListCategories(ctx, s.pool, siteID)
	bySlug := map[string]repo.Category{}
	for _, c := range cats {
		bySlug[c.Slug] = c
	}
	v := &panelsView{Adet: 9, Tonlar: panelTonlar}
	cfg, ok := s.panelConfigOf(ctx)
	if ok {
		v.HasCustom = true
		if cfg.Adet > 0 {
			v.Adet = cfg.Adet
		}
		seen := map[string]bool{}
		for _, p := range cfg.Panels {
			c, exists := bySlug[p.Slug]
			if !exists {
				continue
			}
			seen[p.Slug] = true
			du := p.Duzen
			if !validPanelDuzen(du) {
				du = defaultPanelDuzen(p.Slug)
			}
			tone := p.Tone
			if !validPanelTone(tone) {
				tone = catTone(p.Slug)
			}
			v.Satirlar = append(v.Satirlar, panelSatir{Slug: p.Slug, Name: c.Name, Duzen: du, Tone: tone, Show: p.Show})
		}
		// ayarda olmayan yayındaki kategorileri sona ekle (gizli)
		for _, c := range cats {
			if c.Status == "yayinda" && !seen[c.Slug] {
				v.Satirlar = append(v.Satirlar, panelSatir{Slug: c.Slug, Name: c.Name, Duzen: defaultPanelDuzen(c.Slug), Tone: catTone(c.Slug), Show: false})
			}
		}
	} else {
		for _, c := range cats {
			if c.Status != "yayinda" {
				continue
			}
			v.Satirlar = append(v.Satirlar, panelSatir{Slug: c.Slug, Name: c.Name, Duzen: defaultPanelDuzen(c.Slug), Tone: catTone(c.Slug), Show: true})
		}
	}
	return v
}

// handlePanelsSave POST /admin/kategori-panelleri/kaydet — kart formunu kaydeder.
func (s *Server) handlePanelsSave(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	back := "/admin/tasarim?sayfa=" + builderPage(r)
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/") {
		r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
		_ = r.ParseMultipartForm(1 << 20)
	} else {
		_ = r.ParseForm()
	}

	if r.FormValue("reset") != "" {
		_ = repo.SetSetting(ctx, s.pool, panelsSection, panelsKey, []byte("null"))
		s.invalidateContentCache()
		s.flashRedirect(w, r, back, "info", "Kategori panelleri varsayılana döndürüldü.")
		return
	}

	adet := 9
	if n := parseIntPtr(r.FormValue("adet")); n != nil && *n > 0 {
		adet = *n
	}
	if adet > 30 {
		adet = 30
	}

	var panels []panelAyar
	seen := map[string]bool{}
	for _, slug := range strings.Split(r.FormValue("sira"), ",") {
		slug = strings.TrimSpace(slug)
		if slug == "" || seen[slug] {
			continue
		}
		seen[slug] = true
		du := r.FormValue("duzen_" + slug)
		if !validPanelDuzen(du) {
			du = defaultPanelDuzen(slug)
		}
		tone := r.FormValue("tone_" + slug)
		if !validPanelTone(tone) {
			tone = catTone(slug)
		}
		panels = append(panels, panelAyar{
			Slug:  slug,
			Duzen: du,
			Tone:  tone,
			Show:  r.FormValue("show_"+slug) != "",
		})
	}

	raw, err := json.Marshal(panelConfig{Adet: adet, Panels: panels})
	if err != nil || len(panels) == 0 {
		// hiç satır yoksa varsayılana dön
		_ = repo.SetSetting(ctx, s.pool, panelsSection, panelsKey, []byte("null"))
	} else if err := repo.SetSetting(ctx, s.pool, panelsSection, panelsKey, raw); err != nil {
		s.flashRedirect(w, r, back, "error", "Kaydedilemedi.")
		return
	}
	s.invalidateContentCache()
	s.flashRedirect(w, r, back, "info", "Kategori panelleri kaydedildi.")
}

// categoryBySlug yayındaki kategoriyi slug ile bulur (panel adı ve rengi
// için). blockCatItems da aynı kontrolü yapar; burada adı da gerekiyor.
func categoryBySlug(cats []repo.Category, slug string) (repo.Category, bool) {
	for _, c := range cats {
		if c.Slug == slug && c.Status == "yayinda" {
			return c, true
		}
	}
	return repo.Category{}, false
}

// catBlockFromItems tek kategori panelini haber listesinden üretir.
// Kapak görseli olan ilk haber büyük manşet olur; kalanlardan en fazla adet
// tanesi ızgaraya/listeye eklenir. Eski homeCatBlocks ve yeni
// kategori_panel modülü aynı üretim yolunu kullanır.
func catBlockFromItems(name, slug string, items []repo.Content, duzen, tone string, adet int) (catBlock, bool) {
	if len(items) < 3 {
		return catBlock{}, false
	}
	if adet < 1 {
		adet = 9
	}
	lead := -1
	for i, it := range items {
		if it.CoverMediaID != nil {
			lead = i
			break
		}
	}
	if lead < 0 {
		return catBlock{}, false
	}
	b := catBlock{Name: name, Slug: slug, Tone: tone, Duzen: duzen, Lead: items[lead]}
	for i, it := range items {
		if i == lead {
			continue
		}
		b.Items = append(b.Items, it)
		if len(b.Items) >= adet {
			break
		}
	}
	if len(b.Items) < 2 {
		return catBlock{}, false
	}
	return b, true
}

// expandCategoryPanelBlocks eski tek bloklu "kategori_bloklari" düzenini,
// kayıtlı panel ayarına göre tek tek "kategori_panel" bloklarına genişletir.
// Tasarımcı bu sayede eski düzeni de parçalanmış modüller olarak gösterir;
// kullanıcı kaydettiğinde yeni bloklar kalıcı olur. Yeni blok zaten varsa
// (geçiş tamamlanmışsa) düzen olduğu gibi bırakılır.
func expandCategoryPanelBlocks(raw []byte, v *panelsView) []byte {
	if len(raw) == 0 || v == nil {
		return raw
	}
	var arr []map[string]any
	if json.Unmarshal(raw, &arr) != nil {
		return raw
	}
	for _, b := range arr {
		if t, _ := b["type"].(string); t == "kategori_panel" {
			return raw
		}
	}
	limit := 6
	if v.HasCustom {
		// Yönetimden seçilen paneller kırpılmaz (homeCatBlocks ile aynı).
		limit = 20
	}
	rows := make([]panelSatir, 0, len(v.Satirlar))
	for _, r := range v.Satirlar {
		if r.Show {
			rows = append(rows, r)
		}
	}
	if len(rows) > limit {
		rows = rows[:limit]
	}
	adet := v.Adet
	if adet <= 0 {
		adet = 9
	}
	used := map[string]bool{}
	for _, b := range arr {
		if id, ok := b["id"].(string); ok {
			used[id] = true
		}
	}
	out := make([]map[string]any, 0, len(arr)+len(rows))
	for _, b := range arr {
		t, _ := b["type"].(string)
		if t != "kategori_bloklari" {
			out = append(out, b)
			continue
		}
		baseID, _ := b["id"].(string)
		if baseID == "" {
			baseID = "kat-panel"
		}
		visible := true
		if bv, ok := b["visible"].(bool); ok {
			visible = bv
		}
		opts := map[string]any{}
		if m, ok := b["opts"].(map[string]any); ok {
			for k, val := range m {
				opts[k] = val
			}
		}
		for _, r := range rows {
			childOpts := make(map[string]any, len(opts)+4)
			for k, val := range opts {
				childOpts[k] = val
			}
			childOpts["kategori"] = r.Slug
			childOpts["duzen"] = r.Duzen
			childOpts["tone"] = r.Tone
			childOpts["count"] = adet
			id := baseID + "-" + r.Slug
			for n := 2; used[id]; n++ {
				id = baseID + "-" + r.Slug + "-" + strconv.Itoa(n)
			}
			used[id] = true
			out = append(out, map[string]any{
				"id":      id,
				"type":    "kategori_panel",
				"visible": visible,
				"folded":  false,
				"locked":  false,
				"opts":    childOpts,
			})
		}
	}
	enc, err := json.Marshal(out)
	if err != nil {
		return raw
	}
	return enc
}
