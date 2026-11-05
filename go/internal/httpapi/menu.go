package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"theblock/site/internal/repo"
)

// MenuRow tek menü satırı.
//
//	Slug  -> kategori slug'ı (boşsa özel link öğesidir, URL kullanılır)
//	URL   -> özel link hedefi (yalnızca Slug boşken)
//	Label -> menüde GÖRÜNEN isim (boşsa Slug'un kategori adı kullanılır)
//	Sub   -> true: header "Diğer" alt menüsünde; false: ana menüde
type MenuRow struct {
	Slug  string `json:"slug"`
	URL   string `json:"url,omitempty"`
	Label string `json:"label,omitempty"`
	Sub   bool   `json:"sub"`
	Name  string `json:"-"` // çözümlenen kategori adı (şablon + admin ipucu)
	// Children yalnızca şablon içindir: üst kategoriye bağlı, menüde
	// gösterilecek alt kategoriler. Ayar JSON'una yazılmaz.
	Children []MenuRow `json:"-"`
}

// menuEditData yönetim paneli "Menü" sayfası verisi.
type menuEditData struct {
	Categories    []repo.Category
	Rows          []MenuRow
	MegaDesign    string
	CatNavDesign  string
	MegaOptions   []menuDesignOption
	CatNavOptions []menuDesignOption
	Bg            *menuBgView
}

// Mega menü panel tasarımları (yönetimden seçilir).
var menuMegaDesigns = []menuDesignOption{
	{Key: "sayfa", Name: "Sayfa Boyu (Tam Ekran)", Desc: "Başlığın altında sayfa yüksekliğinde, alt kategoriler yan yana geniş panel."},
	{Key: "serit", Name: "Sabit Şerit (İkinci Menü)", Desc: "Primary Navigation'ın hemen altında yatay, sabit duran ikinci menü sırası."},
	{Key: "genis", Name: "Geniş Panel", Desc: "Tam genişliğe yakın, daha ferah alt kategori ızgarası."},
	{Key: "kisa", Name: "Kompakt", Desc: "Başlığın altında kısa, 4 sütunlu beyaz panel."},
}

// Kategori sayfası alt menü tasarımları.
var menuCatNavDesigns = []menuDesignOption{
	{Key: "bar", Name: "Yatay Çubuk", Desc: "Alt kategoriler mynet finans gibi yan yana, çizgili yatay menüde."},
	{Key: "pill", Name: "Yuvarlak Etiketler", Desc: "Alt kategoriler yuvarlak hap biçiminde."},
}

type menuDesignOption struct {
	Key  string
	Name string
	Desc string
}

func validMenuDesignOption(list []menuDesignOption, key string) bool {
	for _, o := range list {
		if o.Key == key {
			return true
		}
	}
	return false
}

// menuDesignValues yönetimden seçilen mega menü ve alt menü tasarımlarını
// döndürür. Ayar yoksa yeni varsayılan: sayfa boyu + yatay çubuk.
func (s *Server) menuDesignValues(ctx context.Context) (mega, catnav string) {
	mega = menuSettingString(s.cachedSetting(ctx, "menu", "mega_tasarim"))
	catnav = menuSettingString(s.cachedSetting(ctx, "menu", "alt_tasarim"))
	if !validMenuDesignOption(menuMegaDesigns, mega) {
		mega = "sayfa"
	}
	if !validMenuDesignOption(menuCatNavDesigns, catnav) {
		catnav = "bar"
	}
	return mega, catnav
}

func menuSettingString(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	var v string
	if json.Unmarshal(raw, &v) == nil {
		return strings.TrimSpace(v)
	}
	return strings.TrimSpace(strings.Trim(string(raw), `"`))
}

// publicMenu header menüsünü üretir: ayar yoksa kategorilerden varsayılan.
func (s *Server) publicMenu(ctx context.Context) []MenuRow {
	cats := s.cachedCategories(ctx, siteIDFromCtx(ctx))
	if v := s.cachedSetting(ctx, "menu", "ana"); len(v) > 0 {
		var rows []MenuRow
		if json.Unmarshal(v, &rows) == nil {
			// Eski /sayfa/<slug> bağlantıları kök adrese taşınır (Faz 143).
			for i := range rows {
				rows[i].URL = normalizePageURL(rows[i].URL)
			}
			return s.menuNamed(rows, cats)
		}
	}
	return s.menuNamed(nil, cats)
}

// menuNamed ayar listesini kategori verisiyle çözer: adları doldurur, sırayı
// korur, config'te eksik/yeni görünür kategorileri sona ekler.
func (s *Server) menuNamed(rows []MenuRow, cats []repo.Category) []MenuRow {
	bySlug := map[string]repo.Category{}
	children := map[string][]MenuRow{}
	var order []string
	for _, c := range cats {
		if c.Slug == "" {
			continue
		}
		if _, ok := bySlug[c.Slug]; !ok {
			order = append(order, c.Slug)
		}
		bySlug[c.Slug] = c
		if c.ParentSlug != nil && *c.ParentSlug != "" && c.Status == "yayinda" && (c.InMenu || c.ContentCount > 0) {
			children[*c.ParentSlug] = append(children[*c.ParentSlug], MenuRow{Slug: c.Slug, Name: c.Name})
		}
	}
	seen := map[string]bool{}
	var out []MenuRow
	for _, r := range rows {
		if r.Slug != "" {
			c, ok := bySlug[r.Slug]
			if !ok {
				continue
			}
			// Alt kategoriye taşınmış kayıtlar eski menü ayarında kalmış olsa
			// bile ana menüye geri gelmez.
			if c.ParentID != nil {
				continue
			}
			r.Name = c.Name
		}
		seen[r.Slug+"/"+r.URL] = true
		out = append(out, r)
	}
	for _, slug := range order {
		if seen[slug+"/"] {
			continue
		}
		c := bySlug[slug]
		if c.ParentID != nil || !c.InHeader || c.Status != "yayinda" || c.Slug == "diger" {
			continue
		}
		out = append(out, MenuRow{
			Slug: c.Slug, Name: c.Name,
			Sub: c.Slug == "ilceler" || c.Slug == "sanat",
		})
		seen[slug+"/"] = true
	}
	// Alt kategorileri üst menü satırına bağla (şablon mega panelde kullanır).
	for i := range out {
		if out[i].Slug != "" {
			out[i].Children = children[out[i].Slug]
		}
	}
	return out
}

// handleMenuGet /admin/menu — profesyonel menü düzenleyicisi.
func (s *Server) handleMenuGet(w http.ResponseWriter, r *http.Request) {
	d := s.buildMenuData(r)
	if IsHTMX(r) {
		s.renderMenuPanelV2(w, r, d)
		return
	}
	s.renderMenuPageV2(w, r, d)
}

func (s *Server) buildMenuData(r *http.Request) *tmplData {
	d := s.baseView("tr", "/admin", "Menü Yönetimi")
	d.User = ptr(userFromContext(r.Context()))
	d.Flash = s.consumeFlash(r)
	ctx := r.Context()
	cats, _ := repo.ListCategories(ctx, s.pool, s.curSiteID(r))
	mega, catnav := s.menuDesignValues(ctx)
	d.MenuEdit = &menuEditData{
		Categories: cats, Rows: s.publicMenu(ctx),
		MegaDesign: mega, CatNavDesign: catnav,
		MegaOptions: menuMegaDesigns, CatNavOptions: menuCatNavDesigns,
		Bg: s.menuBgViewFor(ctx),
	}
	return d
}

func (s *Server) renderMenuPanel(w http.ResponseWriter, r *http.Request, kind, text string) {
	d := s.buildMenuData(r)
	if kind != "" {
		d.Flash = &flashMsg{Kind: kind, Text: text}
	}
	s.renderMenuPanelV2(w, r, d)
}

// handleMenuSave POST /admin/menu/kaydet — JSON: {"items":[{slug,url,label,sub}...]}.
func (s *Server) handleMenuSave(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var items []MenuRow
	if IsHTMX(r) {
		_ = r.ParseForm()
		if raw := r.FormValue("items"); raw != "" {
			var body struct {
				Items []MenuRow `json:"items"`
			}
			if json.Unmarshal([]byte(raw), &body) == nil {
				items = body.Items
			} else if json.Unmarshal([]byte(raw), &items) != nil {
				s.renderMenuPanel(w, r, "error", "Menü verisi geçersiz.")
				return
			}
		}
	} else {
		defer r.Body.Close()
		var body struct {
			Items []MenuRow `json:"items"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
			s.writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "JSON geçersiz: " + err.Error()})
			return
		}
		items = body.Items
	}
	if items == nil {
		items = []MenuRow{}
	}
	raw, err := json.Marshal(items)
	if err != nil {
		if IsHTMX(r) {
			s.renderMenuPanel(w, r, "error", err.Error())
			return
		}
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	if err := repo.SetSetting(ctx, s.pool, "menu", "ana", raw); err != nil {
		if IsHTMX(r) {
			s.renderMenuPanel(w, r, "error", err.Error())
			return
		}
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	s.invalidateContentCache()
	if IsHTMX(r) {
		s.renderMenuPanel(w, r, "info", "Menü kaydedildi ("+itoa(len(items))+" öğe).")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "adet": len(items)})
}

// handleMenuDesignSave POST /admin/menu/tasarim — mega menü ve kategori alt
// menüsü tasarımını kaydeder.
func (s *Server) handleMenuDesignSave(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_ = r.ParseForm()
	mega := strings.TrimSpace(r.FormValue("mega"))
	catnav := strings.TrimSpace(r.FormValue("catnav"))
	if !validMenuDesignOption(menuMegaDesigns, mega) || !validMenuDesignOption(menuCatNavDesigns, catnav) {
		if IsHTMX(r) {
			s.renderMenuPanel(w, r, "error", "Tasarım seçimi geçersiz.")
			return
		}
		s.flashRedirect(w, r, "/admin/menu", "error", "Tasarım seçimi geçersiz.")
		return
	}
	megaJSON, _ := json.Marshal(mega)
	catnavJSON, _ := json.Marshal(catnav)
	if err := repo.SetSetting(ctx, s.pool, "menu", "mega_tasarim", megaJSON); err != nil {
		if IsHTMX(r) {
			s.renderMenuPanel(w, r, "error", err.Error())
			return
		}
		s.flashRedirect(w, r, "/admin/menu", "error", err.Error())
		return
	}
	if err := repo.SetSetting(ctx, s.pool, "menu", "alt_tasarim", catnavJSON); err != nil {
		if IsHTMX(r) {
			s.renderMenuPanel(w, r, "error", err.Error())
			return
		}
		s.flashRedirect(w, r, "/admin/menu", "error", err.Error())
		return
	}
	s.invalidateContentCache()
	if IsHTMX(r) {
		s.renderMenuPanel(w, r, "info", "Menü tasarımı kaydedildi.")
		return
	}
	s.flashRedirect(w, r, "/admin/menu", "info", "Menü tasarımı kaydedildi.")
}

// handleMenuReset POST /admin/menu/sifirla — varsayılan menüye döner.
func (s *Server) handleMenuReset(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_ = repo.SetSetting(ctx, s.pool, "menu", "ana", []byte("null"))
	s.invalidateContentCache()
	if IsHTMX(r) {
		s.renderMenuPanel(w, r, "info", "Menü varsayılana sıfırlandı.")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// publicMenuLocale menüyü verilen dilde adlarla döndürür (Faz 189).
//
// Kategoriler tablosunda dil kolonu yok; ad çevirisi `translations`
// tablosunda `category.<slug>` anahtarıyla tutulur. Çeviri yoksa Türkçe ad
// korunur — böylece eksik çeviri menüyü bozmaz, yalnız o satır Türkçe kalır.
func (s *Server) publicMenuLocale(ctx context.Context, locale string) []MenuRow {
	rows := s.publicMenu(ctx)
	if locale == "" || locale == "tr" || s.i18n == nil {
		return rows
	}
	cevir := func(slug, ad string) string {
		if slug == "" {
			return ad
		}
		anahtar := "category." + slug
		v := s.i18n.T(locale, anahtar)
		if v == anahtar || strings.TrimSpace(v) == "" {
			return ad
		}
		return v
	}
	// Slug boş olan satırlar ELLE eklenmiş menü kayıtlarıdır (URL taşır).
	// Bunlarda slug'ı adresten türetiriz: /kategori/teknoloji → teknoloji.
	// Aksi hâlde elle eklenmiş "Donanım" gibi satırlar İngilizce sayfada
	// Türkçe kalıyordu (ölçüldü).
	slugTuret := func(r MenuRow) string {
		if r.Slug != "" {
			return r.Slug
		}
		u := r.URL
		if i := strings.Index(u, "/kategori/"); i >= 0 {
			kalan := u[i+len("/kategori/"):]
			if j := strings.IndexAny(kalan, "/?#"); j >= 0 {
				kalan = kalan[:j]
			}
			return kalan
		}
		return ""
	}
	for i := range rows {
		slug := slugTuret(rows[i])
		rows[i].Name = cevir(slug, rows[i].Name)
		// Şablon `Label`'ı `Name`'den ÖNCE tercih ediyor (menü ayarında elle
		// yazılmış etiket). Yalnız Name'i çevirmek yetmiyordu: ayarda
		// "Donanım" etiketi varsa İngilizce sayfada o görünüyordu (ölçüldü).
		if rows[i].Label != "" {
			rows[i].Label = cevir(slug, rows[i].Label)
		}
		for j := range rows[i].Children {
			cs := slugTuret(rows[i].Children[j])
			rows[i].Children[j].Name = cevir(cs, rows[i].Children[j].Name)
			if rows[i].Children[j].Label != "" {
				rows[i].Children[j].Label = cevir(cs, rows[i].Children[j].Label)
			}
		}
	}
	return rows
}

// navCategoriesLang gezinme kategorilerini verilen dilde adlarla döndürür.
//
// Ad çevirisi `translations` tablosunda `category.<slug>` anahtarındadır;
// yoksa Türkçe ad korunur.
func (s *Server) navCategoriesLang(ctx context.Context, locale string) []repo.Category {
	cats := s.navCategories(ctx)
	if locale == "" || locale == "tr" || s.i18n == nil {
		return cats
	}
	for i := range cats {
		if cats[i].Slug == "" {
			continue
		}
		anahtar := "category." + cats[i].Slug
		if v := s.i18n.T(locale, anahtar); v != anahtar && strings.TrimSpace(v) != "" {
			cats[i].Name = v
		}
	}
	return cats
}
