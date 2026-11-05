package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"theblock/site/internal/piyasa"
	"theblock/site/internal/platform"
	"theblock/site/internal/repo"
)

// readingMinutes içerik metninden yaklaşık okuma süresini (dakika, 200 kelime/dk) hesaplar.
func readingMinutes(htmlBody, spot string) int {
	txt := stripTags(htmlBody) + " " + spot
	words := len(strings.Fields(txt))
	m := (words + 199) / 200
	if m < 1 {
		m = 1
	}
	return m
}

// ---- Modüler sayfa tasarımcısı (spec 08) ----

// blockType palette öğesi.
type blockType struct {
	Type string `json:"type"`
	Name string `json:"name"`
	Desc string `json:"desc"`
	Icon string `json:"icon"`
}

var builderTypes = []blockType{
	{"kategori_liste", "Kategori Haber Listesi", "Kategori sayfasının ana liste gövdesi (görünüm ayarıyla)", "list"},
	// Kategori gövdesini bağımsız modüllere ayıran tipler. Her biri kendi
	// haber aralığını, görünümünü ve satır yerleşimini taşır; böylece slider,
	// seçimli manşetler, 4'lü ızgara ve akış ayrı ayrı taşınıp ayarlanabilir.
	{"kategori_slider", "Kategori Slider", "Kategori manşet slider'ı (bağımsız modül)", "image"},
	{"kategori_secimli", "Seçimli Manşetler", "Seçili manşet kartları (sol/sağ ray)", "layout-grid"},
	{"kategori_4lu", "4'lü Manşet", "4 haberlik kart ızgarası", "layout-grid"},
	{"kategori_akis", "Haber Akışı", "Sonsuz akış (12'lik enerji blokları)", "list"},
	{"slider", "Slider", "Otomatik geçişli görsel slider", "image"},
	{"manset6", "6'lı Manşet", "Mynet tarzı 6 slaytlı tam genişlik manşet", "image"},
	{"manset2", "Manşet II", "Yarım görselli, renkli arka planlı 23 slaytlı manşet", "image"},
	{"son_dakika", "Son Dakika", "Kayan son dakika bandı", "bolt"},
	{"manset", "Manşet", "1 büyük + yandakiler ızgarası", "layout-grid"},
	{"kategori_sekmeli", "Kategori Sekmeli", "Sekmeli kategori bölümleri", "folder"},
	{"trend_numarali", "Trend Haberler", "Numaralı trend listesi", "flame"},
	{"galeri_foto", "Foto Galeri", "Fotoğraf galerisi", "photo"},
	{"galeri_video", "Video Galeri", "Video galerisi", "video"},
	{"one_cikanlar", "Öne Çıkanlar", "Kart ızgarası", "star"},
	{"kategori_izgara", "Kategori Izgarası", "İkonlu kategori kartları", "folder"},
	{"trend", "Trend", "Gündem etiketleri", "flame"},
	{"reklam", "Reklam", "Reklam yerleşimi (slot)", "bell"},
	{"baslik", "Başlık", "Bölüm başlığı / ayraç", "heading"},
	{"metin", "Zengin Metin", "Biçimlendirilmiş metin (kalın, liste, bağlantı…)", "align-left"},
	{"servis_bandi", "Servis Bandı", "Hava + piyasa bandı", "sun"},
	{"kur_borsa", "Kur & Borsa", "Döviz, altın, borsa göstergeleri (canlı)", "chart"},
	{"hisseler", "Hisseler", "BIST hisse şeridi: kod, fiyat, yön (canlı)", "chart"},
	{"kripto_para", "Kripto Para", "Kripto fiyat ızgarası (ikon, fiyat, %24s)", "chart"},
	{"kur_cevirici", "Kur Çevirici", "Döviz çevirici (12+ para birimi, canlı kur)", "chart"},
	// Düzenlerde kullanılan ama palette eksik olan tipler
	{"hikayeler", "Hikayeler", "Dairesel hikaye şeridi", "star"},
	{"ust_manset", "Üst Manşet", "Yatay kaydırmalı kart şeridi", "layout-grid"},
	{"canli_yayin", "Canlı Yayın", "Kırmızı canlı yayın şeridi", "video"},
	{"kategori_panel", "Kategori Paneli", "Tek kategori vitrini (büyük manşet + kartlar)", "folder"},
	{"puan_durumu", "Puan Durumu (Takım)", "Süper Lig puan durumu (takım logolu)", "layout-grid"},
	{"puan_serit", "Puan Durumu Şeridi", "Takım logosu + puan, yatay şerit (Habertürk spor yazısı düzeni)", "layout-grid"},
	{"gazete_manset", "Gazete Manşetleri", "Günün gazete ön sayfaları (sağ sütunda tek slayt)", "news"},
}

// builderTypesFor sayfaya uygun blok paletini döndürür.
// Yazı sayfasında sayfa içine özgü bloklar (hikaye şeridi, slider vb.)
// anlamsız oldukları için sunulmaz.
func builderTypesFor(page string) []blockType {
	if isSayfaPage(page) {
		// Kurumsal sayfa paleti (Faz 143): gövde + ek metin + kenar modülleri.
		return []blockType{
			{"sayfa_govde", "Sayfa İçeriği", "Sayfanın kendi metni — gövde yerine geçer", "align-left"},
			{"metin", "Zengin Metin", "Biçimlendirilmiş metin (kalın, liste, bağlantı…)", "align-left"},
			{"baslik", "Başlık", "Bölüm başlığı / ayraç", "heading"},
			{"reklam", "Reklam", "Reklam yerleşimi (slot)", "bell"},
			{"one_cikanlar", "Son Haberler", "En yeni haber kartları", "star"},
			{"trend_numarali", "Trend Haberler", "Numaralı trend listesi", "flame"},
			{"kategori_izgara", "Kategori Izgarası", "İkonlu kategori kartları", "folder"},
			{"galeri_foto", "Foto Galeri", "Fotoğraf galerisi", "photo"},
			{"galeri_video", "Video Galeri", "Video galerisi", "video"},
			{"servis_bandi", "Servis Bandı", "Hava + piyasa bandı", "sun"},
		}
	}
	if page != "article" {
		if isKategoriPage(page) {
			return builderTypes
		}
		// Kategori modülleri yalnız kategori sayfasında anlamlıdır; ana
		// sayfada eklendiğinde veri kaynağı (Category) olmadığı için boş
		// kalırlardı. Paleti sayfaya göre daralt.
		out := make([]blockType, 0, len(builderTypes))
		for _, t := range builderTypes {
			if !isCategoryModuleType(t.Type) {
				out = append(out, t)
			}
		}
		return out
	}
	return []blockType{
		{"yazi_govde", "Yazı Gövdesi", "Yazının kendi metni — gövde yerine geçer", "align-left"},
		{"metin", "Zengin Metin", "Biçimlendirilmiş metin (kalın, liste, bağlantı…)", "align-left"},
		{"canli_yayin", "Canlı Yayın", "Sağ sütunda kırmızı şerit", "video"},
		{"trend_numarali", "Trend Haberler", "Numaralı trend listesi", "flame"},
		{"one_cikanlar", "Son Haberler", "En yeni haber kartları", "star"},
		{"kategori_izgara", "Kategori Izgarası", "İkonlu kategori kartları", "folder"},
		{"galeri_foto", "Foto Galeri", "Fotoğraf galerisi", "photo"},
		{"galeri_video", "Video Galeri", "Video galerisi", "video"},
		{"servis_bandi", "Servis Bandı", "Hava + piyasa bandı", "sun"},
		{"kur_borsa", "Kur & Borsa", "Döviz, altın, borsa göstergeleri (canlı)", "chart"},
		{"hisseler", "Hisseler", "BIST hisse şeridi: kod, fiyat, yön (canlı)", "chart"},
		{"kripto_para", "Kripto Para", "Kripto fiyat ızgarası (ikon, fiyat, %24s)", "chart"},
		{"kur_cevirici", "Kur Çevirici", "Döviz çevirici (12+ para birimi, canlı kur)", "chart"},
		{"puan_serit", "Puan Durumu Şeridi", "Takım logosu + puan, yatay şerit (spor yazıları)", "layout-grid"},
		{"reklam", "Reklam", "Reklam yerleşimi (slot)", "bell"},
		{"baslik", "Başlık", "Bölüm başlığı / ayraç", "heading"},
	}
}

// isCategoryModuleType kategori gövdesini bağımsız parçalara ayıran blok
// tiplerinden biri mi? Bu tipler yalnız kategori sayfasında veri taşır.
func isCategoryModuleType(typ string) bool {
	switch typ {
	case "kategori_slider", "kategori_secimli", "kategori_4lu", "kategori_akis":
		return true
	}
	return false
}

type builderData struct {
	Blocks     template.JS
	Types      []blockType
	Page       string
	Zones      []builderZone
	Bg         *bgView
	Layout     *layoutView
	PreviewURL string
	// Designs Manşet Editörü kapak şablonları (blok başına kapak eşlemesi, Faz 3).
	Designs []repo.HeadlineDesign
	// Versions sayfa düzeninin sürüm geçmişi (Faz 4).
	Versions []repo.LayoutVersion
	// AB sayfa düzeni A/B testi durumu (Faz 7).
	AB *repo.LayoutAB
	// CatView kategori listeleme görünüm şablonu (Faz 9): klasik|liste|izgara|mozaik.
	CatView string
	// Kategoriye özel tasarım: CatMode true ise sayfa kategori editörüdür;
	// SelectedCat doluysa "kategori:<slug>" düzenlenir, boşsa genel düzen.
	CatMode     bool
	SelectedCat string
	Categories  []repo.Category
	// AllCategories blok ayarlarındaki kategori çoklu seçimi için tüm
	// yayındaki kategoriler (kategori modu dışında da dolu).
	AllCategories []repo.Category
	UsingGlobal   bool
	// PageLayout bu sayfaya özel sütun şablonu (sağ sütun / genişlikler).
	PageLayout *pageLayoutView
	// PageTexts / PageTextLocale düzenlenebilir şablon metinleri (Faz 142).
	PageTexts      []pageTextItem
	PageTextLocale string
	// Kurumsal sayfalar (Faz 143): SayfaMode true ise sayfa düzeni düzenlenir,
	// SelectedSayfa seçili sayfanın slug'ıdır; Pages seçim listesidir.
	SayfaMode     bool
	SelectedSayfa string
	Pages         []repo.Page
	// Piyasa Kur & Borsa modülü göstergeleri (önbellekten; istek başına bir
	// kez okunur, aynı sayfadaki birden çok modül paylaşır).
	Piyasa *piyasa.Sonuc
} // builderZone tasarımcıda seçilebilir konum.
type builderZone struct {
	Key  string
	Name string
	Hint string
}

// builderZonesFor sayfa başına seçilebilir bölgeler.
func builderZonesFor(page string) []builderZone {
	if page == "article" {
		return []builderZone{
			{"ust", "Üst", "Sayfanın en üstü — başlıktan önce, tam genişlik"},
			{"ana", "Gövde", "Başlık/spot altında — yazının gövde metninin yeri"},
			{"yan", "Sağ Sütun", "Yazının yanındaki dikey kolon"},
			{"genis", "İçerik Altı", "Yazı gövdesinin hemen altı"},
		}
	}
	if isKategoriPage(page) {
		return []builderZone{
			{"ust", "Üst", "Liste başlığının altında, tam genişlik"},
			{"govde", "Gövde", "Haber listesinin olduğu ana alan — modüler bloklarla düzenlenir"},
			{"yan", "Sağ Sütun", "Haber listesinin yanındaki dikey kolon"},
			{"genis", "Alt", "Liste ve sayfalama altında, tam genişlik"},
		}
	}
	if isSayfaPage(page) {
		return []builderZone{
			{"ust", "Üst", "Başlıktan önce, tam genişlik"},
			{"ana", "Gövde", "Sayfa içeriğinin yeri — içerik bloğu buraya gelir"},
			{"genis", "Alt", "Sayfa içeriğinin altında, tam genişlik"},
		}
	}
	return []builderZone{
		{"ust", "Üst", "Sayfa tepesi — tam genişlik"},
		{"ana", "Ana (Sol)", "Sol ana kolon"},
		{"yan", "Sağ Sütun", "Sağ dikey kolon"},
		{"genis", "Tam Genişlik", "İki kolonun altında, tam genişlikte"},
	}
}

// builderPage istekten sayfa anahtarını okur (varsayılan home).
// Kategori sayfası için isteğe bağlı "kategori" parametresi verilirse o
// kategoriye özel anahtar ("kategori:<slug>") döner.
func builderPage(r *http.Request) string {
	v := strings.TrimSpace(r.URL.Query().Get("sayfa"))
	if strings.HasPrefix(v, "kategori:") {
		slug := strings.TrimSpace(strings.TrimPrefix(v, "kategori:"))
		if validCatSlug(slug) {
			return "kategori:" + slug
		}
		return "kategori"
	}
	switch v {
	case "article":
		return "article"
	case "kategori", "category":
		if slug := strings.TrimSpace(r.URL.Query().Get("kategori")); slug != "" && validCatSlug(slug) {
			return "kategori:" + slug
		}
		return "kategori"
	case "sayfa":
		if slug := strings.TrimSpace(r.URL.Query().Get("sayfa_slug")); slug != "" && validCatSlug(slug) {
			return "sayfa:" + slug
		}
		return "sayfa"
	}
	if strings.HasPrefix(v, "sayfa:") {
		slug := strings.TrimSpace(strings.TrimPrefix(v, "sayfa:"))
		if validCatSlug(slug) {
			return "sayfa:" + slug
		}
		return "sayfa"
	}
	return "home"
}

// validCatSlug builder sayfa anahtarına giren kategori slug'ını doğrular.
func validCatSlug(slug string) bool {
	if slug == "" || len(slug) > 80 {
		return false
	}
	for _, r := range slug {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
			return false
		}
	}
	return true
}

// isKategoriPage genel ve kategoriye özel tasarımcı anahtarlarını kapsar.
func isKategoriPage(page string) bool {
	return page == "kategori" || strings.HasPrefix(page, "kategori:")
}

// isSayfaPage kurumsal sayfa tasarımcısı anahtarlarını kapsar (Faz 143).
func isSayfaPage(page string) bool {
	return page == "sayfa" || strings.HasPrefix(page, "sayfa:")
}

// sayfaSlug "sayfa:<slug>" anahtarından slug'ı çıkarır (genel anahtarda boş).
func sayfaSlug(page string) string {
	if strings.HasPrefix(page, "sayfa:") {
		return strings.TrimPrefix(page, "sayfa:")
	}
	return ""
}

// layoutEmpty boş/varsayılan düzen yanıtını ayırt eder.
func layoutEmpty(b []byte) bool {
	t := strings.TrimSpace(string(b))
	return t == "" || t == "[]" || t == "null"
}

// handleBuilderGet sayfa tasarımcısı editörü.
func (s *Server) handleBuilderGet(w http.ResponseWriter, r *http.Request) {
	// Sayfa tasarımcısının betiği (ve SortableJS) gövdenin SONUNDA,
	// .admin-main dışındadır. Panelin kabuk korumalı gezinmesi yalnız
	// .admin-main'i değiştirdiği için boost'lu geçişte tuval hiç kurulmuyor
	// ve sayfa boş geliyordu (F5 ile düzeliyordu). Buradan tam sayfa
	// yüklemeye zorlanır; birincil koruma menü bağlantılarındadır.
	if IsHTMX(r) {
		w.Header().Set("HX-Refresh", "true")
	}
	ctx := r.Context()
	sid := s.curSiteID(r)
	page := builderPage(r)
	catMode := isKategoriPage(page)
	sayfaMode := isSayfaPage(page)
	selectedCat := ""
	if strings.HasPrefix(page, "kategori:") {
		selectedCat = strings.TrimPrefix(page, "kategori:")
	}
	selectedSayfa := sayfaSlug(page)
	d := s.baseView("tr", "/admin", "Sayfa Tasarımcısı")
	d.User = ptr(userFromContext(ctx))
	d.Flash = s.consumeFlash(r)
	raw, _ := repo.GetLayout(ctx, s.pool, sid, page)
	usingGlobal := false
	// Kategoriye özel düzen yoksa genel kategori düzenini başlangıç olarak
	// göster; kullanıcı kaydedince yalnızca bu kategoriye özel düzen oluşur.
	if selectedCat != "" && layoutEmpty(raw) {
		if genel, err := repo.GetLayout(ctx, s.pool, sid, "kategori"); err == nil && !layoutEmpty(genel) {
			raw = genel
			usingGlobal = true
		}
	}
	raw = platform.NormalizeDesign(raw) // blok dizisi JSON
	if len(raw) == 0 {
		raw = []byte("[]")
	}
	// Eski tek bloklu kategori panelleri: tasarımcıda panel başına ayrı
	// modül olarak göster. Kaydedince yeni "kategori_panel" blokları kalıcı
	// olur; public tarafta eski blok geriye dönük uyumlu render edilir.
	if strings.Contains(string(raw), `"kategori_bloklari"`) {
		raw = expandCategoryPanelBlocks(raw, s.panelsViewFor(ctx, sid))
	}
	designs, _ := repo.ListHeadlineDesigns(ctx, s.pool)
	versions, _ := repo.ListLayoutVersions(ctx, s.pool, sid, page, 30)
	ab, _ := repo.GetLayoutAB(ctx, s.pool, sid, page)
	var catOpts, allCatOpts []repo.Category
	if cats, err := repo.ListCategories(ctx, s.pool, sid); err == nil {
		for _, c := range cats {
			if c.Status == "yayinda" {
				allCatOpts = append(allCatOpts, c)
				if catMode {
					catOpts = append(catOpts, c)
				}
			}
		}
	}
	var pages []repo.Page
	if sayfaMode {
		pages, _ = repo.ListPages(ctx, s.pool)
		// Genel "sayfa" anahtarı seçili sayfa yoksa ilk sayfayı düzenler.
		if selectedSayfa == "" && len(pages) > 0 {
			selectedSayfa = pages[0].Slug
			page = "sayfa:" + selectedSayfa
			if layoutEmpty(raw) {
				raw, _ = repo.GetLayout(ctx, s.pool, sid, page)
			}
		}
	}
	d.Builder = &builderData{
		Blocks:         template.JS(raw),
		Types:          builderTypesFor(page),
		Page:           page,
		Zones:          builderZonesFor(page),
		Bg:             s.bgViewFor(ctx),
		Layout:         s.layoutViewFor(ctx),
		PreviewURL:     s.builderPreviewURL(ctx, page),
		Designs:        designs,
		Versions:       versions,
		AB:             &ab,
		CatView:        s.categoryViewForPage(ctx, sid, page),
		CatMode:        catMode,
		SelectedCat:    selectedCat,
		Categories:     catOpts,
		AllCategories:  allCatOpts,
		UsingGlobal:    usingGlobal,
		PageLayout:     s.pageLayoutFor(ctx, page),
		PageTexts:      builderPageTextItems(ctx, s, sid, page, s.defaultLang(ctx)),
		PageTextLocale: s.defaultLang(ctx),
		SayfaMode:      sayfaMode,
		SelectedSayfa:  selectedSayfa,
		Pages:          pages,
	}
	s.render(w, r, "tasarimci", d)
}

// builderPreviewURL tasarımcıdaki "Önizle" bağlantısı için uygun public
// adresi döndürür (home → /, article → örnek yazı sayfası, kategori → ilk
// yayındaki kategori listesi).
func (s *Server) builderPreviewURL(ctx context.Context, page string) string {
	if isSayfaPage(page) {
		if slug := sayfaSlug(page); slug != "" {
			return "/" + slug
		}
		if pages, err := repo.ListPages(ctx, s.pool); err == nil && len(pages) > 0 {
			return "/" + pages[0].Slug
		}
		return "/"
	}
	if strings.HasPrefix(page, "kategori:") {
		return "/kategori/" + strings.TrimPrefix(page, "kategori:")
	}
	switch page {
	case "article":
		return "/iletisim"
	case "kategori":
		if cats := s.navCategories(ctx); len(cats) > 0 && cats[0].Slug != "" {
			return "/kategori/" + cats[0].Slug
		}
	}
	return "/"
}

// handleBuilderSave düzeni kaydeder.
func (s *Server) handleBuilderSave(w http.ResponseWriter, r *http.Request) {
	page := builderPage(r)
	var req struct {
		Blocks json.RawMessage `json:"blocks"`
	}
	if IsHTMX(r) {
		_ = r.ParseForm()
		req.Blocks = json.RawMessage(r.FormValue("blocks"))
	} else if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "bloklar yok"})
		return
	}
	if len(req.Blocks) == 0 {
		if IsHTMX(r) {
			s.writeHTMXStatus(w, "Kaydedilecek blok yok.", "error")
			return
		}
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "bloklar yok"})
		return
	}
	norm, arr, err := s.normalizeBlocksForPage(page, req.Blocks)
	if err != nil {
		if IsHTMX(r) {
			s.writeHTMXStatus(w, err.Error(), "error")
			return
		}
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	// Faz 4: her kayıtta bir önceki aktif düzen sürüm geçmişine arşivlenir.
	oldBlocks, _ := repo.GetLayout(r.Context(), s.pool, s.curSiteID(r), page)
	var versionNote string
	if len(oldBlocks) > 2 {
		versionNote = page + " düzeni güncellendi (" + itoa(len(arr)) + " blok)"
	}
	if err := repo.SaveLayout(r.Context(), s.pool, s.curSiteID(r), page, norm); err != nil {
		if IsHTMX(r) {
			s.writeHTMXStatus(w, "Kaydedilemedi: "+err.Error(), "error")
			return
		}
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	uid := userFromContext(r.Context()).ID
	if len(oldBlocks) > 2 && uid >= 0 {
		if _, err := repo.CreateLayoutVersion(r.Context(), s.pool, s.curSiteID(r), page, oldBlocks, versionNote, uid); err != nil {
			s.logger.Warn("layout version yazılamadı", "err", err)
		}
	}
	s.audit(r, "tasarim_kaydet", page+": "+itoa(len(arr))+" blok")
	if IsHTMX(r) {
		label := "Ana sayfa"
		if page == "article" {
			label = "Yazı sayfası"
		} else if isKategoriPage(page) {
			if strings.HasPrefix(page, "kategori:") {
				label = "Kategori sayfası (" + strings.TrimPrefix(page, "kategori:") + ")"
			} else {
				label = "Kategori sayfası"
			}
		}
		s.writeHTMXStatus(w, "Kaydedildi ("+itoa(len(arr))+" modül). "+label+" güncellendi.", "ok")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "count": len(arr), "page": page})
}

// normalizeBlocksForPage blok dizisini bu sayfanın izin verdiği zonlara sabitler
// (Faz 8: kopyalama dahil paylaşılan yol). Geçersiz zonlar sayfanın ilk geçerli
// zonuna düşer. Dönüş: normalize edilmiş JSON, dizi, hata.
func (s *Server) normalizeBlocksForPage(page string, raw json.RawMessage) ([]byte, []map[string]any, error) {
	blocks := platform.NormalizeDesign(raw)
	var arr []map[string]any
	if err := json.Unmarshal(blocks, &arr); err != nil {
		return nil, nil, err
	}
	valid := map[string]bool{}
	zones := builderZonesFor(page)
	for _, z := range zones {
		valid[z.Key] = true
	}
	// Tanınmayan blok nereye düşer? Hedefin ilk bölgesi genelde doğrudur,
	// ama yazı sayfasında ilk bölge "ust" tam genişlik şerittir; oraya
	// rastgele blok koymak düzeni bozar. Yazıda güvenli yer sağ sütun.
	fallback := zones[0].Key
	if page == "article" {
		fallback = "yan"
	}
	if isSayfaPage(page) {
		fallback = "genis"
	}
	for _, b := range arr {
		opts, ok := b["opts"].(map[string]any)
		if !ok {
			opts = map[string]any{}
			b["opts"] = opts
		}
		z, _ := opts["zone"].(string)
		if !valid[z] {
			opts["zone"] = fallback
		}
	}
	// Gövde blokları ("yazi_govde" / "sayfa_govde") yalnız kendi sayfa
	// türünde ve yalnız bir kez anlamlıdır (Faz 142/143): sabit gövdenin
	// yerini alır. Başka sayfada tutulursa sessizce düşürülür; aynı sayfada
	// ikincisi yok sayılır. Bölgesi her zaman "ana"dır.
	govdeType := ""
	switch {
	case page == "article":
		govdeType = "yazi_govde"
	case isSayfaPage(page):
		govdeType = "sayfa_govde"
	}
	seenGovde := false
	out := arr[:0]
	for _, b := range arr {
		typ, _ := b["type"].(string)
		isGovde := typ == "yazi_govde" || typ == "sayfa_govde"
		if !isGovde {
			out = append(out, b)
			continue
		}
		if govdeType == "" || typ != govdeType || seenGovde {
			continue
		}
		seenGovde = true
		opts, _ := b["opts"].(map[string]any)
		if opts == nil {
			opts = map[string]any{}
			b["opts"] = opts
		}
		opts["zone"] = "ana"
		out = append(out, b)
	}
	arr = out
	norm, err := json.Marshal(arr)
	if err != nil {
		return nil, nil, err
	}
	return norm, arr, nil
}

// handleBuilderReset düzeni varsayılana sıfırlar.
func (s *Server) handleBuilderReset(w http.ResponseWriter, r *http.Request) {
	page := builderPage(r)
	if err := repo.DeleteLayout(r.Context(), s.pool, s.curSiteID(r), page); err != nil {
		s.flashRedirect(w, r, "/admin/tasarim?sayfa="+page, "error", "Sıfırlanamadı.")
		return
	}
	s.audit(r, "tasarim_sifirla", page)
	s.flashRedirect(w, r, "/admin/tasarim?sayfa="+page, "info", "Düzen varsayılana sıfırlandı.")
}

// handleBuilderCopy mevcut sayfanın düzenini başka bir sayfa anahtarına
// kopyalar (Faz 8). Hedefin zonlarına uyarlama normalizeBlocksForPage ile
// yapılır; hedefin önceki düzeni sürüm geçmişine arşivlenir.
func (s *Server) handleBuilderCopy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sid := s.curSiteID(r)
	page := builderPage(r)
	var req struct {
		Hedef string `json:"hedef"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Hedef == "" {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "hedef sayfa yok"})
		return
	}
	hedef := req.Hedef
	if hedef != "home" && hedef != "article" && hedef != "kategori" {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "geçersiz hedef sayfa"})
		return
	}
	if hedef == page {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "hedef kaynakla aynı olmamalı"})
		return
	}
	src, err := repo.GetLayout(ctx, s.pool, sid, page)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "kaynak okunamadı"})
		return
	}
	norm, arr, err := s.normalizeBlocksForPage(hedef, src)
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	// hedefin önceki düzenini arşivle (Faz 4 geçmişi)
	oldBlocks, _ := repo.GetLayout(ctx, s.pool, sid, hedef)
	var note string
	if len(oldBlocks) > 2 {
		note = hedef + " düzeni " + page + " düzeninden kopyalandı (" + itoa(len(arr)) + " blok)"
	}
	if err := repo.SaveLayout(ctx, s.pool, sid, hedef, norm); err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	uid := userFromContext(ctx).ID
	if len(oldBlocks) > 2 && uid >= 0 {
		if _, err := repo.CreateLayoutVersion(r.Context(), s.pool, sid, hedef, oldBlocks, note, uid); err != nil {
			s.logger.Warn("layout version yazılamadı", "err", err)
		}
	}
	s.audit(r, "tasarim_kopyala", page+" → "+hedef+": "+itoa(len(arr))+" blok")
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "kaynak": page, "hedef": hedef, "count": len(arr)})
}

// handleBuilderHistory düzenin sürüm geçmişini JSON olarak döndürür.
func (s *Server) handleBuilderHistory(w http.ResponseWriter, r *http.Request) {
	page := builderPage(r)
	vs, err := repo.ListLayoutVersions(r.Context(), s.pool, s.curSiteID(r), page, 50)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	type vOut struct {
		ID        int64    `json:"id"`
		Note      string   `json:"note"`
		CreatedAt string   `json:"created_at"`
		Count     int      `json:"count"`
		Types     []string `json:"types"`
	}
	out := make([]vOut, 0, len(vs))
	for _, v := range vs {
		types := layoutBlockTypes(v.Blocks)
		out = append(out, vOut{
			ID: v.ID, Note: v.Note,
			CreatedAt: v.CreatedAt.Format("02.01.2006 15:04"),
			Count:     len(types), Types: types,
		})
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "page": page, "versions": out})
}

// handleBuilderRestore istenen sürümü geri alır; geri alma da geçmişe yazılır.
func (s *Server) handleBuilderRestore(w http.ResponseWriter, r *http.Request) {
	page := builderPage(r)
	id := parseID(r.PathValue("id"))
	if id <= 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "sürüm yok"})
		return
	}
	if err := repo.RestoreLayoutVersion(r.Context(), s.pool, s.curSiteID(r), page, id, "", userFromContext(r.Context()).ID); err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	s.audit(r, "tasarim_geri_al", itoa(int(id)))
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "page": page, "version": id})
}

// layoutBlockTypes blok dizisi JSON'undaki blok tiplerini döndürür (geçmiş görünümü).
func layoutBlockTypes(blocks []byte) []string {
	var arr []struct {
		Type string `json:"type"`
	}
	if json.Unmarshal(platform.NormalizeDesign(blocks), &arr) != nil {
		return nil
	}
	var out []string
	for _, b := range arr {
		if b.Type != "" {
			out = append(out, b.Type)
		}
	}
	return out
}

// categoryListView kategori listeleme görünüm şablon adını döndürür (Faz 9).
// Değer: klasik | liste | izgara | mozaik (geçersiz → klasik).
func (s *Server) categoryListView(ctx context.Context) string {
	b, err := repo.GetSetting(ctx, s.pool, "tasarim", "kategori_gorunum")
	if err != nil {
		return "klasik"
	}
	var v string
	if json.Unmarshal(b, &v) == nil && v != "" {
		return v
	}
	return strings.Trim(string(b), `"`)
}

// setCategoryListView kategori görünümünü kaydeder.
func (s *Server) setCategoryListView(ctx context.Context, v string) {
	switch v {
	case "klasik", "liste", "izgara", "mozaik":
	default:
		v = "klasik"
	}
	b, _ := json.Marshal(v)
	_ = repo.SetSetting(ctx, s.pool, "tasarim", "kategori_gorunum", b)
}

// categoryBySlug yayındaki kategoriyi slug ile bulur.
func (s *Server) categoryBySlug(ctx context.Context, siteID int64, slug string) (repo.Category, bool) {
	cats, err := repo.ListCategories(ctx, s.pool, siteID)
	if err != nil {
		return repo.Category{}, false
	}
	for _, c := range cats {
		if c.Slug == slug {
			return c, true
		}
	}
	return repo.Category{}, false
}

// categoryViewForPage seçili kategoriye özel liste görünümünü döndürür;
// özel görünüm yoksa genel kategori görünümüne düşer.
func (s *Server) categoryViewForPage(ctx context.Context, siteID int64, page string) string {
	global := s.categoryListView(ctx)
	slug := strings.TrimPrefix(page, "kategori:")
	if slug == page || slug == "" {
		return global
	}
	if c, ok := s.categoryBySlug(ctx, siteID, slug); ok {
		if validCategoryView(c.ListView) && c.ListView != "" {
			return c.ListView
		}
	}
	return global
}

// handleBuilderCategoryViewSave tasarımcıdan kategori görünümünü kaydeder.
// Genel düzende genel ayarı, kategoriye özel düzende yalnız o kategorinin
// liste görünümünü günceller.
func (s *Server) handleBuilderCategoryViewSave(w http.ResponseWriter, r *http.Request) {
	var req struct {
		View string `json:"view"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	page := builderPage(r)
	if !validCategoryView(req.View) {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "geçersiz görünüm"})
		return
	}
	slug := strings.TrimPrefix(page, "kategori:")
	if slug != page && slug != "" {
		c, ok := s.categoryBySlug(r.Context(), s.curSiteID(r), slug)
		if !ok {
			s.writeJSON(w, http.StatusNotFound, map[string]any{"ok": false, "error": "kategori bulunamadı"})
			return
		}
		if err := repo.SetCategoryDesign(r.Context(), s.pool, c.ID, c.LayoutPreset, req.View); err != nil {
			s.writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error()})
			return
		}
	} else {
		s.setCategoryListView(r.Context(), req.View)
	}
	s.audit(r, "tasarim_kategori_gorunum", page+": "+req.View)
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "page": page, "view": req.View})
}

// handleBuilderCategoryViewGet mevcut kategori görünümünü + seçenekleri döndürür.
func (s *Server) handleBuilderCategoryViewGet(w http.ResponseWriter, r *http.Request) {
	page := builderPage(r)
	s.writeJSON(w, http.StatusOK, map[string]any{
		"ok":   true,
		"page": page,
		"view": s.categoryViewForPage(r.Context(), s.curSiteID(r), page),
		"views": []map[string]string{
			{"id": "klasik", "name": "Klasik (büyük + kartlar)"},
			{"id": "liste", "name": "Sıkı Liste"},
			{"id": "izgara", "name": "Izgara Kartlar"},
			{"id": "mozaik", "name": "Mozaik"},
		},
	})
}

// handleBuilderABGet A/B testi durumunu + son günlerin hit sayılarını döndürür.
func (s *Server) handleBuilderABGet(w http.ResponseWriter, r *http.Request) {
	page := builderPage(r)
	sid := s.curSiteID(r)
	ab, err := repo.GetLayoutAB(r.Context(), s.pool, sid, page)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	hits := s.abHitsView(r.Context(), sid, page, 14)
	type hOut struct {
		Day     string `json:"day"`
		Variant string `json:"variant"`
		Hits    int64  `json:"hits"`
	}
	ho := make([]hOut, 0, len(hits))
	for _, h := range hits {
		ho = append(ho, hOut{Day: h.Day, Variant: h.Variant, Hits: h.Hits})
	}
	s.writeJSON(w, http.StatusOK, map[string]any{
		"ok": true, "page": page,
		"enabled":   ab.Enabled,
		"weight_b":  ab.WeightB,
		"b_count":   len(layoutBlockTypes(ab.BBlocks)),
		"hits_list": ho,
	})
}

// handleBuilderABSave A/B ayarını kaydeder. mode="set-b" ise b_blocks'u isteğin
// blocks'u ile günceller; diğer türlü varsayılana döner.
func (s *Server) handleBuilderABSave(w http.ResponseWriter, r *http.Request) {
	page := builderPage(r)
	sid := s.curSiteID(r)
	var req struct {
		Mode    string          `json:"mode"`
		Enabled *bool           `json:"enabled"`
		WeightB int             `json:"weight_b"`
		Blocks  json.RawMessage `json:"blocks"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "geçersiz istek: " + err.Error()})
		return
	}
	ab, _ := repo.GetLayoutAB(r.Context(), s.pool, sid, page)
	if req.Enabled != nil {
		ab.Enabled = *req.Enabled
	}
	ab.WeightB = req.WeightB
	if req.Mode == "set-b" || len(req.Blocks) > 0 {
		ab.BBlocks = platform.NormalizeDesign(req.Blocks)
	}
	if err := repo.SaveLayoutAB(r.Context(), s.pool, sid, page, ab.Enabled, ab.WeightB, ab.BBlocks); err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	s.audit(r, "tasarim_ab_kaydet", page+": "+itoa(ab.WeightB)+"% B")
	s.writeJSON(w, http.StatusOK, map[string]any{
		"ok": true, "page": page, "enabled": ab.Enabled, "weight_b": ab.WeightB,
	})
}

// ---- Public render için tipli blok görünümü ----

// homeBlock public ana sayfada render edilen tipli blok. D alanı kök görünüme
// geri referanstır; şablonlarda $.D.Home.Hava / $.D.Ads gibi paylaşılan alanlara erişim içindir.
type homeBlock struct {
	ID       string
	Type     string
	Visible  bool
	Count    int
	Autoplay bool
	Interval int
	Dots     bool
	Arrows   bool
	Slot     string
	Label    string
	Text     string
	// RichHTML biçimlendirilmiş metin bloğu içeriği (Faz 6). Depoda güvenli
	// HTML alt kümesine indirgenmiş halde tutulur; render'da template.HTML.
	RichHTML template.HTML
	// ListView kategori_liste bloğunun kendi görünümü (Faz 11). Boşsa o sayfanın
	// genel kategori_view ayarı kullanılır; doluysa bu blok için geçerli olur.
	ListView string
	// Kategori gövdesi manşet düzeni ve ölçüleri (kategori_liste bloğu):
	// CatHeroStyle "klasik" veya "portre"; portre 5 slaytlı dikey slider.
	CatHeroStyle string
	CatHeroCount int
	CatHeroWidth int
	CatHeroRatio string
	CatListCols  int
	CatSlug      string
	// Kategori paneli (kategori_panel): tek kategori vitrininin düzeni ve
	// rengi. Eski kategori_bloklari bloğu bunları panel ayarından alırdı.
	PanelDuzen string
	PanelTone  string
	// Kategori gövdesini bağımsız modüllere ayıran bloklar için:
	//   RowLayout  hero-left | hero-center | hero-right (boş = kendi satırı)
	//   RowSlot    left | center | right (CSS sınıfı için türetilir)
	//   ItemOffset haber aralığının başlangıcı; -1 = otomatik sıra
	//   CatItems   bu modülün tükettiği haberler (kopya kategori görünümü)
	//   AkisStart  akışın sayfa içindeki mutlak başlangıcı; -1 = eski hesap
	RowLayout    string
	RowSlot      string
	RowSlotClass string
	ItemOffset   int
	CatItems     []repo.Content
	AkisStart    int
	AkisStartSet bool
	RailStyle    string
	// HeroShadow kategori manşet slider'ındaki başlığa gölge ekler
	// (portre/üçlü). Slider modülündeki başlık gölgesi karşılığıdır.
	HeroShadow bool
	// Kategori akış vitrini (coinhit/enerji modeli): ilk 4 haber 4'lü kart,
	// kalanı 3 sütun × 4 kart enerji düzeniyle akar. AkisLimit toplam haber
	// sınırı, AkisPer her yüklemede eklenecek haber sayısı, AkisAuto kaydırınca
	// otomatik yükleme.
	AkisLimit int
	AkisPer   int
	AkisAuto  bool
	// AkisHero akışın üstündeki manşet alanını (portre/üçlü) tutar; varsayılan
	// açıktır, istenirse kapatılır.
	AkisHero bool
	// AkisFont akış kartlarının (ilk 4 kart + enerji blokları) yazı ailesidir;
	// tüm akış tek font kullanır. Boşsa coinhit varsayılanı serif'tir.
	// Değerler slider'daki title_font ile aynıdır (system/serif/sans/...).
	AkisFont string
	// Kategori sekmeli blok: blok bazında seçilen kategori slug'ları ve
	// sekme stili/renkleri. CatTabs boşsa yönetimdeki genel sekme seçimi
	// (yoksa menüdeki ilk 8 kategori) kullanılır.
	CatTabs        []string
	TabStyle       string
	TabBg          string
	TabColor       string
	TabActiveBg    string
	TabActiveColor string
	TabRadius      int
	// TabNav sekme içinde sağ/sol oklarla sayfa geçişi (varsayılan açık).
	TabNav bool
	// TabCSS sekme çubuğu için güvenli CSS değişkenleri (renk/yarıçap).
	TabCSS template.CSS
	// Üst manşet kart görünümü: düzen (şerit/ızgara), kart stili ve renkler.
	UMLayout      string
	UMCols        int
	UMStyle       string
	UMTitleColor  string
	UMTitleBg     string
	UMTitleSize   int
	UMGradColor   string
	UMGradOpacity int
	UMShadow      bool
	// Üst manşet kart kabuğu: açık/koyu tema arka planı, gölge ve başlık
	// yatay iç boşluğu. UMTitlePad < 0 = ayarlanmadı (stilin kendi dolgusu).
	UMCardBg     string
	UMCardDarkBg string
	UMCardShadow bool
	UMTitlePad   int
	UMStyleCSS   template.CSS
	// KapakTasarimID blok başına kapak şablonu eşlemesi (Faz 3). 0 = kapak
	// tasarımı yok (içeriğin kendi görseli kullanılır); >0 = Manşet Editörü
	// şablonu id'si → bloğun içerikleri o şablonla rayına dizilir.
	KapakTasarimID int64
	// Slider/manşet tasarım kapaklarında HTML başlık/etiket/spot üst yazısını
	// görsel olarak gizler; metinler ekran okuyucu ve SEO için DOM'da kalır.
	ShowTitle          bool
	ShowBadge          bool
	ShowSpot           bool
	TitleColor         string
	TitleFont          string
	TitleLineHeight    float64
	TitleLetterSpacing int
	TitleWordSpacing   int
	// TitleCase: uppercase | capitalize | sentence | lowercase (boş = stil varsayılanı).
	TitleCase  string
	TitleStyle template.CSS
	// Slider/manşet başlık arkasındaki renk, saydamlık, geçiş ve blur.
	OverlayColor    string
	OverlayOpacity  int
	OverlayGradient string
	OverlayHeight   int
	OverlayBlur     int
	OverlayStyle    template.CSS
	CatColor        string
	CatStyle        template.CSS
	CatPosition     string
	// CatShape kategori etiketinin görsel biçimi: badge, pin, pin2, pill, outline, corner.
	CatShape string
	// CatClip / CatLeft / CatBottom seçili etiket stilinin gelişmiş
	// özelleştirmeleridir. Boşsa ilgili stilin CSS varsayılanı kullanılır.
	CatClip   string
	CatLeft   string
	CatBottom string
	// HeadingStyle modül başlığı işaretleyicisi: line, bar, underline, pill, dot,
	// corner, tab, boxed, gradient, none.
	HeadingStyle string
	// HideHeading true ise modül başlığı görsel olarak gizlenir; başlık metni
	// ekran okuyucu ve SEO için .sr-only olarak DOM'da kalır.
	HideHeading bool
	// HideBadge true ise kategori etiketleri görsel olarak gizlenir; etiket
	// metni .sr-only olarak DOM'da kalır.
	HideBadge bool
	// CardStyle paylaşımlı kart stili (kategori ızgarası, üst manşet, öne
	// çıkanlar). Boş = klasik. Değerler: bloomberg, bloomberg-cizgi,
	// bant-alt, vurgu, cerceve, kose, cam, alt-cizgi.
	CardStyle string
	// CardCSS kart stilinin renk/ölçü değişkenleri (CSS özel özellikleri).
	CardCSS template.CSS
	// ModuleStyle modül sarmalayıcısının görünüm şablonu: boxed, soft, accent, glass.
	ModuleStyle string
	// HideMobile true ise modül mobil görünümde gizlenir; DOM/SEO içeriği korunur.
	HideMobile  bool
	PagerType   string
	PagerLayout string
	PagerShape  string
	PagerPos    string
	PagerColor  string
	ShowAll     bool
	// Slider görünüm şablonu (edirne, minimal, magazin, sinema...).
	SliderStyle string
	// PagerSize sayı/nokta göstergesinin piksel boyutudur.
	PagerSize int
	// AllowOver20 true ise slider/6'lı manşet 20'nin üzerinde içerik gösterebilir.
	AllowOver20 bool
	// Manşet II: yarım görsel + renkli arka plan + 23'lü numaralı gezinme.
	M2Bg          string
	M2Accent      string
	M2Fade        string
	M2NumBg       string
	M2NumActiveBg string
	M2Side        string
	M2Height      int
	// Zone yerleşim bölgesi:
	//   "ust"  → tam genişlik üst alan (hikayeler, üst manşet, slider, servis)
	//   "ana"  → sol ana kolon (varsayılan)
	//   "yan"  → sağ sütun
	//   "genis"→ iki kolonlu düzen BİTTİKTEN SONRA tam genişlik
	//            (haberpi'de kategori panelleri ve foto/video galeriler
	//             sol kolonda değil, burada tam genişlikte durur)
	Zone string

	// Modül başına görünüm/ölçü ayarları (blok opts: st*).
	// Uzunluk alanları CSS birimi kabul eder (px/rem/em/%/vw/vh/calc);
	// padding ve margin 1-4 değerli CSS kısayolu olarak saklanabilir.
	StPad         string
	StMargin      string
	StGapX        string // kartlar arası yatay (sütun) boşluk
	StGapY        string // kartlar arası dikey (satır) boşluk
	StGapTop      string
	StGapBottom   string
	StBorderW     int
	StBorderColor string
	StRadius      int
	// StRadiusSet "Köşe yuvarlaklığı" alanının açıkça doldurulduğunu gösterir.
	// 0 geçerli bir değerdir (tamamen köşeli); boş alan tema yarıçapını korur.
	StRadiusSet bool
	StBg        string
	// Serbest (absolute) konumlandırma — Faz 5. StAbs true ise blok akıştan
	// çıkar ve .blk-st sarmalayıcısına position:absolute + konum yazılır;
	// bloğun zon container'ı (position:relative) referans alınır.
	StAbs bool
	StX   int
	StY   int
	StW   int
	StZ   int
	Wrap  bool // stil varsa modül bir sarmalayıcıya alınır
	Style template.CSS

	D      *tmplData
	Slides []repo.Content
	// Rail slider sağındaki iki sabit kart; slider sayısından etkilenmez.
	Rail        []repo.Content
	Items       []repo.Content
	CatSections []catSection
	Categories  []repo.Category
	Trends      []repo.CatTrend
	// Puan durumu (puan_durumu bloğu)
	Standings []repo.StandingRow
	// Puan durumu şeridi (puan_serit bloğu) — Habertürk spor yazısı düzeni.
	PuanBaslik string
	PuanLigAdi string
	// Gazete manşetleri (gazete_manset bloğu): günün gazete ön sayfaları.
	// GazeteDar sağ sütunda (yan) true olur → şerit değil TEK SLAYT düzeni.
	GazeteKapak  []repo.NewspaperPage
	GazeteGun    string
	GazeteBaslik string
	GazeteDar    bool
	// GazeteAralik otomatik geçiş aralığı (sn); 0 = otomatik geçiş kapalı.
	GazeteAralik int
	// Piyasa Kur & Borsa modülü göstergeleri (blok ayarlarına göre süzülmüş).
	Piyasa []piyasa.Kalem
	// PiyasaGuncel verinin çekildiği an ("son güncelleme" satırı).
	PiyasaGuncel time.Time
	// PiyasaDemo önbellek boşsa true (modül hiç basılmaz, rozet gösterilir).
	PiyasaDemo bool
	// PiyasaStil modül şablonu (referans|sade|kart|dar).
	PiyasaStil string
	// PiyasaOk yön oku tasarımı (noktali|ucgen|ok|uzun|yok).
	PiyasaOk string
	// PiyasaBg arka plan seçimi (yok|zemin|ozel). "yok" saydam demektir.
	PiyasaBg string
	// PiyasaBgCSS arka plan için basılacak stil parçası (boşsa kural yok).
	PiyasaBgCSS template.CSS
	// Hisseler modülü (BIST hisse şeridi): kod + fiyat + yön.
	// Görünüm sözlüğü Kur & Borsa ile aynıdır (bkz. hisse_view.go).
	Hisseler      []repo.HisseRow
	HisseYerlesim string // serit | izgara
	HisseStil     string // referans | sade | kart | dar
	HisseArrow    string // noktali | ucgen | ok | uzun | yok
	HisseBg       string // kart | yok | zemin | ozel
	HisseBgCSS    template.CSS
	HisseSirala   string // kod | yukselen | dusen
	HisseEtkin    string // bölgeye göre çözülmüş stil (şablona bu gider)
	HisseDar      bool   // dar bölgede mi (sağ ray)
	HisseGecis    string // surekli (Habertürk) | adim (aralıklı)
	HisseHiz      int    // px/sn (sürekli model)
	HisseBekle    int    // sn (aralıklı model beklemesi)
	HisseKayma    int    // ms (aralıklı model geçiş süresi)
	HisseSure     int    // CSS yedeği için tur süresi (sn)
	HisseAra      bool   // modül altında hisse arama kutusu
	HisseDegisim  bool   // yüzde değişimi göster
	HisseGuncel   time.Time
	HisseCSS      template.CSS
	// Kur Çevirici modülü (Kur tablosu önbelleğinden).
	KurCevirici *kurCeviriciView
	// Kur Çevirici ayarları (görünüm build sırasında kurulur).
	KCBaslik string
	KCFrom   string
	KCTo     string
	KCMiktar int

	// Kur Çevirici arka plan seçimi (kart|yok|zemin|ozel) ve stil parçası.
	KCBg    string
	KCBgCSS template.CSS
	// Kripto Para modülü.
	Kripto        []repo.KriptoRow
	KriptoKolon   int
	KriptoDegisim bool
	KriptoIkon    bool
	KriptoSirala  string
	// Kripto modülü görünüm/geçiş — Hisseler ile aynı sözlük.
	KriptoYerlesim string // izgara | serit
	KriptoStil     string // otomatik | referans | sade | kart | dar | rail | mini
	KriptoEtkin    string // bölgeye göre çözülmüş stil (şablona bu gider)
	KriptoDar      bool   // dar bölgede mi (sağ ray)
	KriptoListe    bool   // rail/mini (liste tabanlı şablon)
	KriptoGecis    string // surekli | adim
	KriptoHiz      int    // px/sn
	KriptoBekle    int    // sn (adım modeli)
	KriptoKayma    int    // ms (adım modeli)
	KriptoSure     int    // CSS yedeği (sn)
	KriptoCSS      template.CSS
	KriptoGuncel   time.Time
	// Kripto arka plan seçimi (kart|yok|zemin|ozel) ve stil parçası.
	KriptoBg    string
	KriptoBgCSS template.CSS
	// Kari ana sütun mozaik satırları (haberpi düzeni)
	Kari []kariRow
	// Servis bandı için blok bazında seçilmiş kur kalemleri.
	ServisKurlar []repo.Kur
	// Servis bandının hava durumu + il listesi. Ana sayfa dışındaki
	// sayfalarda (yazı, kategori, kurumsal) veri havuzunda hava/şehir
	// yoktur; bandın buralarda da basılabilmesi için blok üzerinde taşınır.
	ServisHava   *WeatherS
	ServisCities []repo.City
	// Servis bandı arka plan seçimi (kart|yok|zemin|ozel) ve stil parçası.
	ServisBg    string
	ServisBgCSS template.CSS

	// Trend Haberler modülü — kartın SAĞINDAKİ sayısal alan.
	//
	// Kullanıcı isteği: "Trend Haberler modülünde kartların sağındaki alanda
	// sayısal alan köşeleri düzleştirme ayarı ekle, stil türleri de ekle."
	//
	// TrendNumStil sayısal alanın görünümüdür (blok|duz|serit|rozet|buyuk|kontur).
	// TrendNumRadius köşe yarıçapıdır; -1 = tema varsayılanı (kartın köşesine
	// uyar), 0 = tamamen düz köşe.
	TrendNumStil   string
	TrendNumRadius int
	TrendNumCSS    template.CSS
	// Opts bloğun ham ayarlarıdır; servis bandı gibi liste filtreleri için
	// render sırasında kullanılır.
	Opts map[string]any
	// Kategori panelleri (haberpi ana sayfa panel blokları)
	CatBlocks []catBlock
	// Hikaye şeridi için zuck.js uyumlu JSON dizisi (haberpi initHikaye ile birebir)
	StoriesJSON template.JS
}

// catRow kategori gövdesindeki görsel satır: Group true ise ardışık
// hero-left/hero-center/hero-right modülleri tek .cat-hero ızgarasında
// birleştirilir; false ise blok kendi satırında basılır.
type catRow struct {
	Group  bool
	Class  string
	Style  template.CSS
	Blocks []homeBlock
}

// kariRow mozaik ızgarasının tek satırı; Horizontal ise kartlar yatay
// (görsel solda, başlık sağda) yerleşir.
type kariRow struct {
	Horizontal bool
	Items      []repo.Content
}

// buildKariRows haberleri haberpi ana sütun desenine göre böler:
// 2 dikey, 3 dikey, 2 yatay, 3 dikey, 2 yatay, 3 dikey, ...
func buildKariRows(items []repo.Content, maxRows int) []kariRow {
	var rows []kariRow
	i := 0
	for r := 0; i < len(items) && (maxRows <= 0 || len(rows) < maxRows); r++ {
		size, horiz := 3, false
		if r == 0 {
			size = 2
		} else if r%2 == 0 {
			size, horiz = 2, true
		}
		if i+size > len(items) {
			size = len(items) - i
		}
		if size <= 0 {
			break
		}
		// Son grupta tek kart kalıyorsa (ör. 7 içerik -> 2+3+2, 9 -> 2+3+2+2),
		// tek kartlı dev satır yerine önceki satıra ekle.
		if size == 1 && len(rows) > 0 {
			rows[len(rows)-1].Items = append(rows[len(rows)-1].Items, items[i])
			break
		}
		rows = append(rows, kariRow{Horizontal: horiz, Items: items[i : i+size]})
		i += size
	}
	return rows
}

// firstContent ilk n öğeyi döndürür (n<=0 veya n>len → tümü).
func firstContent(items []repo.Content, n int) []repo.Content {
	if n <= 0 || n > len(items) {
		n = len(items)
	}
	return items[:n]
}

// catItemSlice haber listesinden [start, start+count) aralığını güvenli
// biçimde döndürür. Aralık dışına taşarsa eldeki kadarını verir.
func catItemSlice(items []repo.Content, start, count int) []repo.Content {
	if start < 0 {
		start = 0
	}
	if start >= len(items) || count <= 0 {
		return nil
	}
	end := start + count
	if end > len(items) {
		end = len(items)
	}
	return items[start:end]
}

// assignCategoryModuleItems kategori gövdesindeki bağımsız modüllere haber
// aralıklarını atar. Öncelik açık item_offset değerlerindedir; boş bırakılan
// modüller sırayla ilk kullanılmamış haberlerden beslenir. Akış modülü kalan
// haberleri tüketir ve AkisStart ile AJAX ucuna mutlak başlangıç verir.
func assignCategoryModuleItems(blocks []homeBlock, items []repo.Content) {
	if len(blocks) == 0 || len(items) == 0 {
		return
	}
	used := make([]bool, len(items))
	mark := func(start, end int) {
		for i := start; i < end && i < len(used); i++ {
			used[i] = true
		}
	}
	// 1) Açık aralıklar.
	for i := range blocks {
		b := &blocks[i]
		if !isCategoryModuleType(b.Type) || b.ItemOffset < 0 {
			continue
		}
		if b.Type == "kategori_akis" {
			b.AkisStart = b.ItemOffset
			b.AkisStartSet = true
			// Akış kalan aralığı tüketir; sonraki otomatik modüller
			// akışla aynı haberleri tekrar basmasın.
			mark(b.ItemOffset, len(items))
			continue
		}
		b.CatItems = catItemSlice(items, b.ItemOffset, b.Count)
		if b.CatItems == nil {
			// Aralık dışı: modül boş kalmalı, tam liste ile dolmamalı.
			// (withCatBlock yalnız nil olmayan dilimi uygular.)
			b.CatItems = []repo.Content{}
		}
		mark(b.ItemOffset, b.ItemOffset+len(b.CatItems))
	}
	// 2) Otomatik modüller: sırayla ilk kullanılmamış aralık.
	cursor := 0
	for i := range blocks {
		b := &blocks[i]
		if !isCategoryModuleType(b.Type) || b.ItemOffset >= 0 {
			continue
		}
		if b.Type == "kategori_akis" {
			// Akıştan sonra gelen modüller boş kalır; akış son sırada
			// kullanılmalıdır. Akış bitişik bir kuyruk tükettiği için açık
			// aralıklı modüllerin son kullandığı haberden sonra başlar;
			// aksi halde aradaki haberler akışta tekrar basılırdı.
			start := cursor
			for i := start; i < len(items); i++ {
				if used[i] {
					start = i + 1
				}
			}
			b.AkisStart = start
			b.AkisStartSet = true
			cursor = len(items)
			continue
		}
		start := cursor
		for start < len(items) && used[start] {
			start++
		}
		n := b.Count
		if n < 1 {
			n = 1
		}
		end := start
		for end < len(items) && end-start < n {
			if !used[end] {
				end++
			} else {
				break
			}
		}
		b.CatItems = items[start:end]
		mark(start, end)
		cursor = end
	}
}

// groupCatRows kategori gövdesi bloklarını görsel satırlara böler. Ardışık
// hero-left/hero-center/hero-right modülleri tek satırda toplanır; aynı yuva
// ikinci kez gelirse yeni satır başlar. Satır genişliği orta slider'ın
// hero_width değerinden, oran sınıfı da hero_ratio değerinden türetilir.
func groupCatRows(blocks []homeBlock) []catRow {
	var rows []catRow
	var cur []homeBlock
	curSlots := map[string]bool{}
	flush := func() {
		if len(cur) == 0 {
			return
		}
		row := catRow{Group: true, Blocks: cur}
		slots := map[string]bool{}
		for _, b := range cur {
			slots[b.RowSlot] = true
		}
		switch {
		case slots["left"] && slots["center"] && slots["right"]:
			row.Class = "cat-row-lcr"
		case slots["left"] && slots["center"]:
			row.Class = "cat-row-lc"
		case slots["center"] && slots["right"]:
			row.Class = "cat-row-cr"
		case slots["left"] && slots["right"]:
			row.Class = "cat-row-lr"
		case slots["center"]:
			row.Class = "cat-row-c"
		case slots["left"]:
			row.Class = "cat-row-l"
		case slots["right"]:
			row.Class = "cat-row-r"
		}
		width := 0
		ratio := ""
		for _, b := range cur {
			if b.RowSlot == "center" {
				width = b.CatHeroWidth
				ratio = b.CatHeroRatio
			}
		}
		if width < 1 {
			width = 44
		}
		if ratio == "" {
			ratio = "4/5"
		}
		row.Style = template.CSS("--cat-hero-w:" + strconv.Itoa(width) + "%;")
		switch ratio {
		case "3/4":
			row.Class += " cat-ratio-3-4"
		case "1/1":
			row.Class += " cat-ratio-1-1"
		case "16/9":
			row.Class += " cat-ratio-16-9"
		default:
			row.Class += " cat-ratio-4-5"
		}
		rows = append(rows, row)
		cur = nil
		curSlots = map[string]bool{}
	}
	for _, b := range blocks {
		if !b.Visible || b.Zone != "govde" {
			continue
		}
		if b.RowSlot == "" {
			flush()
			rows = append(rows, catRow{Blocks: []homeBlock{b}})
			continue
		}
		if curSlots[b.RowSlot] {
			flush()
		}
		cur = append(cur, b)
		curSlots[b.RowSlot] = true
	}
	flush()
	return rows
}

// excludeContent items listesinden, exclude listelerinde yer alan kimlikleri çıkarır.
func excludeContent(items []repo.Content, lists ...[]repo.Content) []repo.Content {
	if len(items) == 0 || len(lists) == 0 {
		return items
	}
	seen := make(map[int64]bool)
	for _, l := range lists {
		for _, c := range l {
			seen[c.ID] = true
		}
	}
	out := items[:0]
	for _, c := range items {
		if !seen[c.ID] {
			out = append(out, c)
		}
	}
	return out
}

func firstCat(items []repo.Category, n int) []repo.Category {
	if n <= 0 || n > len(items) {
		n = len(items)
	}
	return items[:n]
}

func firstTrend(items []repo.CatTrend, n int) []repo.CatTrend {
	if n <= 0 || n > len(items) {
		n = len(items)
	}
	return items[:n]
}

func firstSection(items []catSection, n int) []catSection {
	if n <= 0 || n > len(items) {
		n = len(items)
	}
	return items[:n]
}

// firstStd puan durumu satırlarından ilk n tanesini döndürür.
func firstStd(items []repo.StandingRow, n int) []repo.StandingRow {
	if n <= 0 || n > len(items) {
		n = len(items)
	}
	return items[:n]
}

// buildBlocks ham blok dizisini, blok başına hazırlanmış veri yükleriyle döndürür.
func (s *Server) buildBlocks(ctx context.Context, d *tmplData, raw []byte) []homeBlock {
	if d.Home == nil {
		return nil
	}
	return s.buildBlocksFrom(ctx, d, raw, d.Home)
}

// buildBlocksFrom verilen veri kaynağından blokları doldurur. Yazı sayfası
// kendi havuzunu (homeData) kurar ve aynı fonksiyonu kullanır.
func (s *Server) buildBlocksFrom(ctx context.Context, d *tmplData, raw []byte, h *homeData) []homeBlock {
	blocks := parseHomeBlocks(raw)
	if len(blocks) == 0 || h == nil {
		return nil
	}
	// Kategori görünürlüğü: yazı sayfasında blok yalnız seçilen
	// kategorilerde basılır. Filtre döngüden ÖNCE uygulanır ki elenen blok
	// için veri de yüklenmesin.
	if d != nil && d.Article != nil {
		gorunen := blocks[:0]
		for _, b := range blocks {
			if blokYazidaGorunur(b.Opts, d) {
				gorunen = append(gorunen, b)
			}
		}
		blocks = gorunen
		if len(blocks) == 0 {
			return nil
		}
	}
	for i := range blocks {
		b := &blocks[i]
		b.D = d
		// Blok bazında kategori seçimi: seçili kategorinin en yeni içerikleri
		// varsayılan veri kaynağının yerine kullanılır. Kaynak yoksa (silinmiş
		// kategori vb.) blok normal davranışına döner.
		var src []repo.Content
		if b.CatSlug != "" {
			need := b.Count
			if b.Type == "kategori_panel" {
				need++ // büyük manşet + count kadar kart
			}
			if need < 12 {
				need = 12
			}
			src = s.blockCatItems(ctx, h, b.CatSlug, need)
		}
		switch b.Type {
		case "slider":
			slides := h.Slides
			if len(src) > 0 {
				slides = src
			}
			b.Slides = firstContent(slides, b.Count)
			// Sağ ray slider sayısından bağımsız sabit kartlardır. İlk
			// slaytlarla tekrar etmemesi için 3. içerikten başlanır.
			//
			// "sayili" (Habertürk spor düzeni) 2×2 ızgara kullanır: 4 kart.
			// Diğer stiller 2 kartlık dikey raydır.
			railCount := 2
			if b.SliderStyle == "sayili" {
				railCount = 4
			}
			if len(slides) > 2 {
				b.Rail = firstContent(slides[2:], railCount)
			} else {
				b.Rail = firstContent(slides, railCount)
			}
		case "manset6":
			// mynet tarzı 6'lı manşet: güncel öne çıkan içeriklerden
			slides := h.Slides
			if len(src) > 0 {
				slides = src
			}
			b.Slides = firstContent(slides, b.Count)
		case "manset2":
			// Yarım görselli, renkli arka planlı 23'lü Manşet II.
			slides := h.Slides
			if len(src) > 0 {
				slides = src
			}
			b.Slides = firstContent(slides, b.Count)
		case "manset":
			items := h.Lead
			if len(src) > 0 {
				items = src
			}
			b.Items = firstContent(items, b.Count)
		case "one_cikanlar":
			if len(src) > 0 {
				b.Items = firstContent(src, b.Count)
			} else {
				// manşet ve son dakika haberleri tekrar görünmesin
				b.Items = firstContent(excludeContent(h.Recent, h.Lead, h.Breaking), b.Count)
			}
		case "son_dakika":
			items := h.Breaking
			if len(src) > 0 {
				items = src
			}
			b.Items = firstContent(items, b.Count)
		case "trend_numarali":
			items := h.TopRead
			if len(src) > 0 {
				items = src
			}
			b.Items = firstContent(items, b.Count)
		case "kategori_sekmeli":
			// Blok kendi kategori listesini taşıyorsa onu kullan; yoksa
			// yönetimdeki genel sekme seçimine/menüye düş.
			b.CatSections = s.catSectionsFor(ctx, d, h, b)
			if !h.TabCatsSelected && len(b.CatTabs) == 0 {
				b.CatSections = firstSection(b.CatSections, b.Count)
			}
		case "kategori_izgara":
			// Tüm yayındaki kategoriler (yeni eklenenler dahil) — eksik kalmasın.
			var cs []repo.Category
			for _, c := range h.Categories {
				if c.Slug == "" || c.Status != "yayinda" {
					continue
				}
				cs = append(cs, c)
			}
			b.Categories = cs
		case "trend":
			b.Trends = firstTrend(h.Trends, b.Count)
		case "galeri_foto":
			b.Items = firstContent(h.GalleryPhotos, b.Count)
		case "galeri_video":
			b.Items = firstContent(h.GalleryVideos, b.Count)
		case "hikayeler":
			if len(src) > 0 {
				b.Items = firstContent(withCover(src), b.Count)
			} else {
				// hikayeler görselli taze içeriklerden; kategori çeşitliliği
				// için sırayla birer tane seçilerek doldurulur
				b.Items = firstContent(diversify(withCover(h.Recent)), b.Count)
			}
			b.StoriesJSON = s.storiesJSON(d, b.Items)
		case "ust_manset":
			if len(src) > 0 {
				b.Items = firstContent(src, b.Count)
			} else {
				// üst manşet şeridi: slider'da kullanılmayan taze haberler
				b.Items = firstContent(excludeContent(h.Recent, nil), b.Count)
			}
			if len(b.Items) > 8 {
				b.Items = b.Items[:8]
			}
		case "kategori_bloklari":
			b.CatBlocks = h.CatBlocks
		case "kategori_panel":
			// Tek kategori paneli: kendi kategorisinin haberlerinden bir
			// büyük manşet + count kadar kart üretir.
			cat, ok := categoryBySlug(h.Categories, b.CatSlug)
			if ok {
				if cb, ok := catBlockFromItems(cat.Name, cat.Slug, src, b.PanelDuzen, b.PanelTone, b.Count); ok {
					b.CatBlocks = []catBlock{cb}
				}
			}
		case "kari_kartlar":
			items := excludeContent(h.Recent, h.Lead, h.Breaking)
			if len(src) > 0 {
				items = src
			}
			b.Kari = buildKariRows(items, 0)
		case "puan_durumu":
			if d.Puan == nil {
				pd := repo.StandingsOf(context.Background(), s.pool)
				d.Puan = &pd
			}
			b.Standings = firstStd(d.Puan.Rows, b.Count)
		case "gazete_manset":
			// Günün gazete kapakları — önbellekten (DB), ağa çıkılmaz.
			//
			// Kaynak boşsa (henüz çekilmemiş) modül hiç basılmaz; boş kutu
			// göstermek yerine bölüm tamamen yok olur.
			sid := int64(1)
			if h != nil && h.SiteID > 0 {
				sid = h.SiteID
			}
			kapak, gun, err := repo.LatestNewspaperPages(ctx, s.pool, sid, b.Count)
			if err != nil || len(kapak) == 0 {
				b.GazeteKapak = nil
				continue
			}
			b.GazeteKapak = kapak
			b.GazeteGun = gun
			b.GazeteDar = darBolge(b.Zone)
		case "puan_serit":
			// Habertürk spor yazısı düzeni: takım kartları şeridi.
			// Lig blok ayarından seçilir; önbellekten okunur (ağa çıkılmaz).
			lg := repo.LeagueByKey(blockOptString(b.Opts, "lig", "superlig"))
			sd := repo.StandingsOfLeague(context.Background(), s.pool, lg)
			b.Standings = firstStd(sd.Rows, b.Count)
			b.PuanBaslik = blockOptString(b.Opts, "puan_baslik", "PUAN DURUMU")
			b.PuanLigAdi = lg.Name
			if len(b.Standings) == 0 {
				// Puan verisi yoksa modül hiç basılmaz (boş kutu göstermeyiz).
				b.PuanBaslik = ""
			}
		case "kripto_para":
			// Kripto fiyatları önbellekten; veri yoksa modül basılmaz.
			if d.Kripto == nil {
				kd := repo.KriptoOf(context.Background(), s.pool)
				d.Kripto = &kd
			}
			b.Kripto = kriptoSatirlar(d.Kripto.Rows, b.KriptoSirala, b.Count)
			b.KriptoGuncel = d.Kripto.Updated
			b.KriptoCSS = template.CSS("--krp-dur:" + itoa(b.KriptoSure) + "s")
			switch b.KriptoBg {
			case "kart":
				// Modülün kendi kart/çerçeve yüzeyi (varsayılan).
			case "yok":
				// Saydam: iç kartların yüzeyi CSS ile kaldırılır.
			case "zemin":
				b.KriptoBgCSS = template.CSS(kbArkaPlanCSS("zemin", ""))
			case "ozel":
				b.KriptoBgCSS = template.CSS(kbArkaPlanCSS("ozel", b.StBg))
			}
		case "kur_cevirici":
			// Kur tablosu önbellekten; tablo boşsa modül basılmaz.
			if d.KurTablo == nil {
				kt := repo.KurlarOf(context.Background(), s.pool)
				d.KurTablo = &kt
			}
			b.KurCevirici = kurCeviriciOlustur(*d.KurTablo, b.KCBaslik, b.KCFrom, b.KCTo, float64(b.KCMiktar))
			switch b.KCBg {
			case "kart":
				// Modülün kendi yüzeyi (varsayılan).
			case "yok":
				// Saydam: .kcr-bg-yok sınıfı yüzeyi/kenarlığı kaldırır.
			case "zemin":
				b.KCBgCSS = template.CSS(kbArkaPlanCSS("zemin", ""))
			case "ozel":
				b.KCBgCSS = template.CSS(kbArkaPlanCSS("ozel", b.StBg))
			}
		case "servis_bandi":
			// Yazı/kategori/kurumsal sayfalarda veri havuzunda kur ve hava
			// yoktur; bant yalnız ana sayfada doluyordu. Önbellekten tamamla.
			kurlar := h.Kurlar
			if len(kurlar) == 0 {
				kurlar, _ = servisKurlar(ctx, s.pool)
			}
			b.ServisKurlar = filterServisKurlar(kurlar, b.Opts)
			b.ServisHava = h.Hava
			if b.ServisHava == nil {
				b.ServisHava = s.havaFromCache(ctx)
			}
			if len(d.Cities) == 0 {
				d.Cities, _ = repo.ListCities(ctx, s.pool)
			}
			b.ServisCities = d.Cities
			switch b.ServisBg {
			case "kart":
				// .tk-inner varsayılan yüzeyi.
			case "yok":
				// Saydam: .sv-bg-yok sınıfı yüzeyi/kenarlığı kaldırır.
			case "zemin":
				b.ServisBgCSS = template.CSS("--sv-bg:var(--hp-surface-2,#f4f4f5)")
			case "ozel":
				if c := sanitizeHexColor(b.StBg); c != "" {
					b.ServisBgCSS = template.CSS("--sv-bg:" + c)
				}
			}
		case "hisseler":
			// Fiyatlar önbellekten (istek yolunda ağa çıkılmaz). Veri yoksa
			// modül hiç basılmaz: sahte/demo fiyat gösterilmez.
			if d.Hisseler == nil {
				hd := repo.HisselerOf(context.Background(), s.pool)
				d.Hisseler = &hd
			}
			b.Hisseler = hisseRows(d.Hisseler.Rows, b.HisseSirala, b.Count)
			b.HisseGuncel = d.Hisseler.Updated
			b.HisseCSS = template.CSS("--hss-dur:" + itoa(b.HisseSure) + "s")
			// Arka plan: Kur & Borsa ile aynı kurallar. "yok" saydamdır;
			// zemin/özel renk modülün kendi bölümüne uygulanır ve ortak
			// "Arka plan rengi" alanı susturulur (çift dolgu olmasın).
			switch b.HisseBg {
			case "kart":
				// Modülün kendi çerçevesi (varsayılan görünüm).
			case "yok":
				// Saydam: .hss-bg-yok çerçevenin yüzeyini/kenarlığını kaldırır.
			case "zemin":
				b.HisseBgCSS = template.CSS(kbArkaPlanCSS("zemin", ""))
			case "ozel":
				b.HisseBgCSS = template.CSS(kbArkaPlanCSS("ozel", b.StBg))
			}
		case "kur_borsa":
			// Piyasa göstergeleri önbellekten (istek yolunda ağa çıkılmaz).
			if d.Piyasa == nil {
				ps := repo.PiyasaOf(context.Background(), s.pool)
				d.Piyasa = &ps
			}
			b.Piyasa = filterPiyasa(d.Piyasa.Kalemler, b.Opts)
			b.PiyasaGuncel = d.Piyasa.Guncel
			b.PiyasaDemo = len(d.Piyasa.Kalemler) == 0
			b.PiyasaStil = kbStilOf(b.Opts)
			b.PiyasaOk = kbOkOf(b.Opts)
			b.PiyasaBg = kbBgOf(b.Opts)
			switch b.PiyasaBg {
			case "kart":
				b.PiyasaBgCSS = template.CSS(kbArkaPlanCSS("kart", ""))
			case "yok":
				// Saydam.
			case "zemin":
				b.PiyasaBgCSS = template.CSS(kbArkaPlanCSS("zemin", ""))
			case "ozel":
				b.PiyasaBgCSS = template.CSS(kbArkaPlanCSS("ozel", b.StBg))
			}
		case "canli_yayin":
			// video varsa onu, yoksa en taze haberi gösterir
			if len(src) > 0 {
				b.Items = firstContent(src, 1)
			} else {
				b.Items = firstContent(h.GalleryVideos, 1)
				if len(b.Items) == 0 {
					b.Items = firstContent(h.Recent, 1)
				}
			}
		}
		s.applyDesignCover(ctx, b)
	}
	// Kategori gövdesi bağımsız modülleri: slider / seçimli manşetler /
	// 4'lü ızgara / akış aynı haber havuzunu sırayla paylaşır.
	if d.Category != nil {
		assignCategoryModuleItems(blocks, d.Category.Items)
	}
	// Video süreleri: blok kartlarında gösterilecek videolar için TEK sorgu
	// (her blok için sorgu yapılmaz — N+1 yok). Faz 21.
	s.fillBlockVideoDurations(ctx, blocks)
	return blocks
}

// filterServisKurlar servis bandındaki kur kalemlerini blok ayarlarına göre
// süzer.
//
// Varsayılan AÇIK: panelde bir kalemin kutusu işaretli görünüyorsa (anahtar
// hiç kaydedilmemişse de işaretli görünür), bantta da görünmelidir. Kullanıcı
// kutuyu kaldırıp kaydederse anahtar `false` olarak yazılır ve kalem elenir.
// Böylece panel ile bant aynı şeyi gösterir.
func filterServisKurlar(items []repo.Kur, opts map[string]any) []repo.Kur {
	keys := map[string]string{
		"BIST": "kur_bist", "USD": "kur_usd", "EUR": "kur_eur",
		"ALTIN": "kur_altin", "BTC": "kur_btc", "ONS": "kur_ons",
	}
	out := make([]repo.Kur, 0, len(items))
	for _, k := range items {
		key, known := keys[k.Name]
		if known && !blockOptBool(opts, key, true) {
			continue
		}
		out = append(out, k)
	}
	return out
}

func blockOptBool(opts map[string]any, key string, def bool) bool {
	if v, ok := opts[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return def
}

// blockOptString blok ayarından metin okur.
func blockOptString(opts map[string]any, key, def string) string {
	if v, ok := opts[key].(string); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return def
}

// blokKategoriGorunumu blok ayarındaki `gorunum_kats` listesini döndürür.
//
// Kullanıcı isteği: "hangi yazı kategorilerinde yazılarda olması için seçenek".
// Virgülle ayrılmış kategori slug'ları (builder'daki cats-multi alanı).
// Boş = kısıt yok (her sayfada görünür — geriye dönük uyumlu).
func blokKategoriGorunumu(opts map[string]any) []string {
	raw := strings.TrimSpace(blockOptString(opts, "gorunum_kats", ""))
	if raw == "" {
		return nil
	}
	out := make([]string, 0, 6)
	for _, s := range strings.Split(raw, ",") {
		// Değer JSON'a yazılırken tırnaklanmış olabilir (""spor"" gibi).
		// Tırnaklar temizlenir; aksi hâlde "spor" != spor olur ve filtre
		// hiçbir yazıyla eşleşmez.
		s = strings.Trim(strings.TrimSpace(s), `"'`)
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// blokYazidaGorunur mu? Yalnız yazı (makale) sayfalarında kısıt uygulanır:
//
//   - `gorunum_kats` boşsa → her yerde görünür.
//   - Doluysa ve sayfa bir yazı sayfasıysa → yazının kategorisi listede
//     olmalı; değilse blok hiç basılmaz (boşluk da bırakmaz).
//   - Yazı sayfası değilse (ana sayfa/kategori) → kısıt uygulanmaz; oradaki
//     düzen ayrı bir karardır, modülü orada görmek isteyip istemediğini
//     kullanıcı o düzeni düzenleyerek belirler.
func blokYazidaGorunur(opts map[string]any, d *tmplData) bool {
	kats := blokKategoriGorunumu(opts)
	if len(kats) == 0 {
		return true
	}
	if d == nil || d.Article == nil || d.Article.Content.CategorySlug == nil {
		return true
	}
	slug := strings.TrimSpace(*d.Article.Content.CategorySlug)
	if slug == "" {
		return true
	}
	for _, k := range kats {
		if strings.EqualFold(k, slug) {
			return true
		}
	}
	return false
}

// filterPiyasa Kur & Borsa modülündeki göstergeleri blok ayarlarına göre
// süzer ve sıralar.
//
// Ayar anahtarı gösterge kimliğidir (`piyasa_usd` gibi), görünen ad değil:
// ad değişse de (ör. "Dolar" → "USD/TRY") düzen bozulmaz. Ayar yoksa tümü
// görünür — eski düzenler ve ilk ekleme için doğru varsayılan.
func filterPiyasa(items []piyasa.Kalem, opts map[string]any) []piyasa.Kalem {
	if len(items) == 0 {
		return nil
	}
	out := make([]piyasa.Kalem, 0, len(items))
	for _, k := range items {
		if !blockOptBool(opts, "piyasa_"+k.Anahtar, true) {
			continue
		}
		out = append(out, k)
	}
	// Sıra: gösterge listesi ekranda referans düzene göre sabittir
	// (BIST, Dolar, Euro, Gram Altın, …). Özel sıra istenirse burada
	// opts["sira"] okunabilir; şimdilik kaynak sırası korunur.
	if limit := blockOptInt(opts, "count", 0); limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

// blockOptInt blok ayarından tam sayı okur (JSON'dan float64 gelir).
func blockOptInt(opts map[string]any, key string, def int) int {
	v, ok := opts[key]
	if !ok {
		return def
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case string:
		if i, err := strconv.Atoi(n); err == nil {
			return i
		}
	}
	return def
}

// withCover yalnızca görseli olan içerikleri döndürür (hikaye dairesi için).
func withCover(items []repo.Content) []repo.Content {
	out := items[:0]
	for _, c := range items {
		if c.CoverMediaID != nil && *c.CoverMediaID > 0 {
			out = append(out, c)
		}
	}
	return out
}

// diversify her kategoriden sırayla birer içerik alır: ilk turda tüm
// kategorilerden birer tane, sonraki turlarda her kategoriden birer tane daha.
// Böylece hikaye şeridinde tek kategorinin (ör. teknoloji) üst üste düşmez.
func diversify(items []repo.Content) []repo.Content {
	groups := make([][]repo.Content, 0, 12)
	index := map[int64]int{}
	for _, c := range items {
		var key int64 = -1
		if c.CategoryID != nil {
			key = *c.CategoryID
		}
		i, ok := index[key]
		if !ok {
			i = len(groups)
			index[key] = i
			groups = append(groups, nil)
		}
		groups[i] = append(groups[i], c)
	}
	var out []repo.Content
	for round := 0; ; round++ {
		added := false
		for _, g := range groups {
			if round < len(g) {
				out = append(out, g[round])
				added = true
			}
		}
		if !added {
			return out
		}
	}
}

// storiesJSON hikaye şeridi için haberpi/zuck.js uyumlu "window.hikayeData"
// JSON dizisini üretir. Alan yapısı haberpi main.js initHikaye() ile birebirdir;
// zuck.js Snapgram skin tıklama, açılış animasyonu ve geçişleri aynı şekilde kurar.
func (s *Server) storiesJSON(d *tmplData, items []repo.Content) template.JS {
	if len(items) == 0 {
		return ""
	}
	// haberpi hikaye verisi şekli: id/photo/name/link/lastUpdated/baslik/itemId/time
	//
	// Photo ve PhotoBuyuk bilerek AYRI: daire 86x86 (retinada 172px yeter),
	// hikâye açılınca gösterilen kare ise ~362px (retinada 724px ister).
	// Önceden ikisi de 480x480 thumb'dı: daire için fazla, büyük görünüm için
	// az. Daire artık 320 varyantından (kullanılabilir kare 180px) gelir —
	// ölçüm: anasayfada 12 dairede kayda değer tasarruf. Büyük görsel yalnız
	// kullanıcı hikâyeyi açarsa indirilir.
	type storyJSON struct {
		ID          string `json:"id"`
		Photo       string `json:"photo"`
		PhotoBuyuk  string `json:"photoBuyuk"`
		Name        string `json:"name"`
		Link        string `json:"link"`
		LastUpdated int64  `json:"lastUpdated"`
		Baslik      string `json:"baslik"`
		ItemID      string `json:"itemId"`
		Time        int64  `json:"time"`
	}
	arr := make([]storyJSON, 0, len(items))
	for _, c := range items {
		if c.PublishedAt == nil {
			continue
		}
		name := "Haber"
		if c.CategoryName != nil && *c.CategoryName != "" {
			name = *c.CategoryName
		}
		arr = append(arr, storyJSON{
			ID:          fmt.Sprintf("hikaye-%d", c.ID),
			Photo:       s.mediaVariantURL(c.CoverMediaID, 320),
			PhotoBuyuk:  s.mediaVariantURL(c.CoverMediaID, 768),
			Name:        name,
			Link:        fmt.Sprintf("%s/%s-%d", d.LangBase, c.DisplaySlug(), c.ID),
			LastUpdated: c.PublishedAt.Unix(),
			Baslik:      c.Title,
			ItemID:      fmt.Sprintf("item-%d", c.ID),
			Time:        c.PublishedAt.Unix(),
		})
	}
	raw, err := json.Marshal(arr)
	if err != nil {
		return ""
	}
	return template.JS(string(raw))
}

// parseCatTabs kategori sekmeli bloktaki "sekmeler" ayarını okur.
// Değer JSON string ("a,b") veya dizi olabilir; slug'lar doğrulanır,
// tekrarlar atılır ve en fazla 12 sekme kabul edilir.
func parseCatTabs(v any) []string {
	var raw []string
	switch t := v.(type) {
	case string:
		raw = strings.Split(t, ",")
	case []any:
		for _, x := range t {
			if s, ok := x.(string); ok {
				raw = append(raw, s)
			}
		}
	case []string:
		raw = t
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(raw))
	for _, slug := range raw {
		slug = strings.TrimSpace(slug)
		if slug == "" || !validCatSlug(slug) || seen[slug] {
			continue
		}
		seen[slug] = true
		out = append(out, slug)
		if len(out) >= 12 {
			break
		}
	}
	return out
}

// blockCatItems blok ayarlarında seçilen kategori için en yeni içerikleri
// döndürür. Aynı kategoriyi kullanan bloklar için sorgu sayfa başına BİR kez
// yapılır (homeData.catItemsCache). Slug geçersiz/silinmişse nil döner ve
// blok varsayılan veri kaynağını kullanmaya devam eder.
func (s *Server) blockCatItems(ctx context.Context, h *homeData, slug string, want int) []repo.Content {
	if h == nil || h.SiteID == 0 || slug == "" {
		return nil
	}
	if h.catItemsCache != nil {
		if cached, ok := h.catItemsCache[slug]; ok {
			if len(cached) > want {
				return cached[:want]
			}
			return cached
		}
	}
	var catID int64
	for _, c := range h.Categories {
		if c.Slug == slug && c.Status == "yayinda" {
			catID = c.ID
			break
		}
	}
	if catID == 0 {
		return nil
	}
	per := want
	if per < 12 {
		per = 12
	}
	if per > 48 {
		per = 48
	}
	items, _, err := repo.FeedPage(ctx, s.pool, repo.FeedQuery{
		SiteID: h.SiteID, CategoryID: catID, Page: 1, PerPage: per,
	})
	if err != nil {
		return nil
	}
	if h.catItemsCache == nil {
		h.catItemsCache = map[string][]repo.Content{}
	}
	h.catItemsCache[slug] = items
	if len(items) > want {
		return items[:want]
	}
	return items
}

// catSectionsFor kategori sekmeli blok için sekmeleri doldurur.// Öncelik: bloktaki seçim → yönetimdeki genel sekme seçimi → menü kategorileri.
// Tüm sekmelerin içerikleri TEK sorguda çekilir (FeedByCategories).
func (s *Server) catSectionsFor(ctx context.Context, d *tmplData, h *homeData, b *homeBlock) []catSection {
	if h == nil || h.SiteID == 0 || b == nil {
		return nil
	}
	cats := h.Categories
	if len(cats) == 0 && d != nil {
		cats = d.NavCategories
	}
	bySlug := make(map[string]repo.Category, len(cats))
	for _, c := range cats {
		if c.Slug != "" {
			bySlug[c.Slug] = c
		}
	}
	var chosen []repo.Category
	if len(b.CatTabs) > 0 {
		for _, slug := range b.CatTabs {
			if c, ok := bySlug[slug]; ok && c.Status == "yayinda" {
				chosen = append(chosen, c)
			}
		}
	} else if slugs, ok := tabCatSlugs(ctx, s); ok && len(slugs) > 0 {
		// Genel sekme seçimi: kategori listesi zaten bellekte (h.Categories),
		// ekstra ListCategories sorgusu yapılmaz.
		for _, slug := range slugs {
			if c, found := bySlug[slug]; found && c.Status == "yayinda" {
				chosen = append(chosen, c)
			}
		}
	}
	if len(chosen) == 0 {
		if d != nil {
			chosen = d.NavCategories
		} else {
			chosen = cats
		}
	}
	if len(chosen) > 12 {
		chosen = chosen[:12]
	}
	if len(b.CatTabs) == 0 && len(chosen) > 8 {
		// Genel/menü tabanlı eski davranış: en fazla 8 sekme.
		chosen = chosen[:8]
	}
	per := b.Count
	if per <= 0 {
		per = 5
	}
	if per > 20 {
		per = 20
	}
	pageSize := per
	if b.TabNav {
		// Oklarla ilerlemek için 3 sayfalık haber beslenir (en fazla 20).
		per = per * 3
		if per > 20 {
			per = 20
		}
	}
	ids := make([]int64, 0, len(chosen))
	keyParts := make([]string, 0, len(chosen)+3)
	for _, c := range chosen {
		ids = append(ids, c.ID)
		keyParts = append(keyParts, c.Slug)
	}
	keyParts = append(keyParts, strconv.Itoa(per), strconv.Itoa(pageSize), strconv.FormatBool(b.TabNav))
	key := strings.Join(keyParts, ",")
	if h.catCache != nil {
		if cached, ok := h.catCache[key]; ok {
			return cached
		}
	}
	itemsByCat, err := repo.FeedByCategories(ctx, s.pool, h.SiteID, ids, per)
	if err != nil {
		return nil
	}
	out := make([]catSection, 0, len(chosen))
	for _, c := range chosen {
		items := itemsByCat[c.ID]
		if len(items) == 0 {
			continue
		}
		sec := catSection{Name: c.Name, Slug: c.Slug, Items: items}
		if b.TabNav {
			for i := 0; i < len(items); i += pageSize {
				end := i + pageSize
				if end > len(items) {
					end = len(items)
				}
				sec.Pages = append(sec.Pages, items[i:end])
			}
		} else {
			if len(items) > pageSize {
				items = items[:pageSize]
				sec.Items = items
			}
			sec.Pages = [][]repo.Content{items}
		}
		out = append(out, sec)
	}
	if h.catCache == nil {
		h.catCache = map[string][]catSection{}
	}
	h.catCache[key] = out
	return out
}

// parseHomeBlocks ham blok dizisini tipli görünüme çevirir.
func parseHomeBlocks(raw []byte) []homeBlock {
	raw = platform.NormalizeDesign(raw)
	if len(raw) == 0 {
		return nil
	}
	var arr []struct {
		ID      string         `json:"id"`
		Type    string         `json:"type"`
		Visible *bool          `json:"visible"`
		Opts    map[string]any `json:"opts"`
	}
	if json.Unmarshal(raw, &arr) != nil {
		return nil
	}
	var out []homeBlock
	for _, b := range arr {
		bv := homeBlock{ID: b.ID, Type: b.Type, Visible: true}
		if b.Visible != nil {
			bv.Visible = *b.Visible
		}
		get := func(k string) string {
			if v, ok := b.Opts[k]; ok {
				if s, ok := v.(string); ok {
					return s
				}
			}
			return ""
		}
		intOf := func(k string, def int) int {
			if v, ok := b.Opts[k].(float64); ok {
				return int(v)
			}
			return def
		}
		// intOpt sayısal değerleri JSON number (float64) veya formdan gelen
		// sayısal string olarak kabul eder; seçim alanları string saklar.
		intOpt := func(k string, def int) int {
			if n := intOf(k, 0); n != 0 {
				return n
			}
			if raw := get(k); raw != "" {
				if n, err := strconv.Atoi(strings.TrimSpace(raw)); err == nil {
					return n
				}
			}
			return def
		}
		floatOf := func(k string, def float64) float64 {
			if v, ok := b.Opts[k].(float64); ok {
				return v
			}
			return def
		}
		boolOf := func(k string, def bool) bool {
			if v, ok := b.Opts[k].(bool); ok {
				return v
			}
			return def
		}
		// lenOf uzunluk alanlarını hem string ("1rem") hem eski JSON
		// sayısı (12) olarak kabul eder; birimsiz sayı sanitizer'da px olur.
		lenOf := func(k string) string {
			switch v := b.Opts[k].(type) {
			case string:
				return v
			case float64:
				// Eski düzenlerde 0 "ayar yok" anlamına geliyordu; gereksiz
				// sarmalayıcı/boşluk üretmemek için 0'ı yok say.
				if v == 0 {
					return ""
				}
				return strconv.FormatFloat(v, 'f', -1, 64)
			}
			return ""
		}
		bv.Opts = b.Opts
		bv.Count = intOf("count", 6)
		if bv.Type == "trend_numarali" {
			// Sayısal alan (kartın sağındaki numara kutusu) görünümü.
			//
			// Kullanıcı isteği: "kartların sağındaki alanda sayısal alan
			// köşeleri düzleştirme ayarı ekle, stil türleri de ekle."
			bv.TrendNumStil = get("tn_stil")
			switch bv.TrendNumStil {
			case "blok", "duz", "serit", "rozet", "buyuk", "kontur":
				// geçerli
			default:
				bv.TrendNumStil = "blok" // mevcut görünüm
			}
			// -1 = tema (kartın köşesine uyar), 0 = tamamen düz köşe.
			// 0 geçerli bir değer olduğu için intOpt değil ham okuma.
			bv.TrendNumRadius = -1
			if n, ok := b.Opts["tn_radius"].(float64); ok {
				bv.TrendNumRadius = int(n)
			} else if raw := get("tn_radius"); strings.TrimSpace(raw) != "" {
				if n, err := strconv.Atoi(strings.TrimSpace(raw)); err == nil {
					bv.TrendNumRadius = n
				}
			}
			if bv.TrendNumRadius > 40 {
				bv.TrendNumRadius = 40
			}
			if bv.TrendNumRadius < 0 {
				bv.TrendNumRadius = -1
			}
			var tnVars []string
			// Düz köşe isteniyorsa kutu KENDİ yarıçapını alır (0px); kart
			// taşma kırpması aksi hâlde üst köşeyi karttan miras bırakırdı.
			if bv.TrendNumRadius >= 0 {
				tnVars = append(tnVars, "--tn-radius:"+strconv.Itoa(bv.TrendNumRadius)+"px")
			}
			if c := sanitizeHexColor(get("tn_bg")); c != "" {
				tnVars = append(tnVars, "--tn-bg:"+c)
			}
			if c := sanitizeHexColor(get("tn_fg")); c != "" {
				tnVars = append(tnVars, "--tn-fg:"+c)
			}
			if len(tnVars) > 0 {
				bv.TrendNumCSS = template.CSS(strings.Join(tnVars, ";"))
			}
		}
		if bv.Type == "gazete_manset" {
			// 32 gazete var; varsayılan 12 kapak (sağ sütunda tek slayt
			// döner, 12 yeterli bir tur).
			bv.Count = intOf("count", 12)
			if bv.Count < 1 {
				bv.Count = 1
			}
			if bv.Count > 32 {
				bv.Count = 32
			}
			bv.GazeteBaslik = strings.TrimSpace(get("gazete_baslik"))
			// 0 geçerli: otomatik geçiş kapalı (kullanıcı isteyerek kapatır).
			bv.GazeteAralik = 6
			if v, ok := b.Opts["gazete_aralik"]; ok {
				switch t := v.(type) {
				case float64:
					bv.GazeteAralik = int(t)
				case string:
					if n, err := strconv.Atoi(strings.TrimSpace(t)); err == nil {
						bv.GazeteAralik = n
					}
				}
			}
			if bv.GazeteAralik < 0 {
				bv.GazeteAralik = 0
			}
			if bv.GazeteAralik > 30 {
				bv.GazeteAralik = 30
			}
		}
		if bv.Type == "puan_durumu" {
			bv.Count = intOf("count", 10) // varsayılan 10 takım (haberpi ile aynı)
		}
		if bv.Type == "manset2" {
			bv.Count = intOf("count", 23)
			if bv.Count < 1 {
				bv.Count = 1
			}
			if bv.Count > 48 {
				bv.Count = 48
			}
		}
		bv.AllowOver20 = boolOf("allow_over_20", false)
		if bv.Type == "slider" || bv.Type == "manset6" {
			// Varsayılan üst sınır 20; isteğe bağlı olarak 20 üzerine
			// çıkılabilir. Veri havuzu en fazla 48 içerik besler.
			maxCount := 20
			if bv.AllowOver20 {
				maxCount = 48
			}
			if bv.Count > maxCount {
				bv.Count = maxCount
			}
			if bv.Count < 1 {
				bv.Count = 1
			}
		}
		bv.Autoplay = boolOf("autoplay", true)
		bv.Interval = intOf("interval", 6)
		bv.Dots = boolOf("dots", true)
		bv.Arrows = boolOf("arrows", true)
		bv.Slot = get("slot")
		bv.Label = get("label")
		bv.Text = get("text")
		bv.CatSlug = get("kategori")
		if bv.Type == "kategori_panel" {
			// Tek kategori paneli: manşet dışı haber sayısı, vitrin düzeni
			// ve renk artık modülün kendi ayarlarıdır.
			bv.Count = intOpt("count", 9)
			if bv.Count < 3 {
				bv.Count = 3
			}
			if bv.Count > 30 {
				bv.Count = 30
			}
			bv.PanelDuzen = get("duzen")
			if !validPanelDuzen(bv.PanelDuzen) {
				bv.PanelDuzen = defaultPanelDuzen(bv.CatSlug)
			}
			bv.PanelTone = get("tone")
			if !validPanelTone(bv.PanelTone) {
				bv.PanelTone = catTone(bv.CatSlug)
			}
		}
		bv.ListView = get("view")
		if bv.Type == "servis_bandi" {
			bv.ServisBg = servisBgOf(get("sv_bg"))
		}
		if bv.Type == "kripto_para" {
			// Kripto ızgarası/şeridi: kaç coin, sütun, görünüm, geçiş.
			bv.Count = intOpt("count", kriptoSayiVarsayilan)
			if bv.Count < 2 {
				bv.Count = 2
			}
			if bv.Count > kriptoSayiAzami {
				bv.Count = kriptoSayiAzami
			}
			bv.KriptoKolon = intOpt("kp_kolon", kriptoKolonVarsayilan)
			if bv.KriptoKolon < kriptoKolonEnAz {
				bv.KriptoKolon = kriptoKolonEnAz
			}
			if bv.KriptoKolon > kriptoKolonEnCok {
				bv.KriptoKolon = kriptoKolonEnCok
			}
			bv.KriptoDegisim = boolOf("kp_degisim", true)
			bv.KriptoIkon = boolOf("kp_ikon", true)
			bv.KriptoSirala = kriptoSiralaOf(get("kp_sirala"))
			// Görünüm/geçiş: Hisseler modülüyle aynı ayar adları ve mantık.
			bv.KriptoYerlesim = kriptoYerlesimOf(get("kp_layout"))
			bv.KriptoStil = kriptoStilOf(get("kp_style"))
			bv.KriptoBg = kriptoBgOf(get("kp_bg"))
			bv.KriptoGecis = kriptoGecisOf(get("kp_gecis"))
			bv.KriptoHiz = hisseHizCoz(get("kp_hiz_secim"), intOpt("kp_speed", 0), intOpt("kp_hiz", 0), bv.Count)
			bv.KriptoBekle = intOpt("kp_bekle", hisseBekleVarsayilan)
			if bv.KriptoBekle < 1 {
				bv.KriptoBekle = 1
			}
			if bv.KriptoBekle > 30 {
				bv.KriptoBekle = 30
			}
			bv.KriptoKayma = intOpt("kp_kayma", hisseKaymaVarsayilan)
			if bv.KriptoKayma < 200 {
				bv.KriptoKayma = 200
			}
			if bv.KriptoKayma > 3000 {
				bv.KriptoKayma = 3000
			}
			bv.KriptoSure = hisseSureTur(bv.Count, bv.KriptoHiz)
		}
		if bv.Type == "kur_cevirici" {
			// Çevirici: başlık, varsayılan kaynak/hedef ve tutar.
			bv.KCBaslik = get("kc_baslik")
			bv.KCFrom = get("kc_from")
			bv.KCTo = get("kc_to")
			bv.KCBg = kurCeviriciBgOf(get("kc_bg"))
			bv.KCMiktar = intOpt("kc_miktar", kurCeviriciMiktarVarsayilan)
			if bv.KCMiktar <= 0 {
				bv.KCMiktar = kurCeviriciMiktarVarsayilan
			}
			if bv.KCMiktar > 1000000000 {
				bv.KCMiktar = 1000000000
			}
		}
		if bv.Type == "hisseler" {
			// BIST hisse şeridi: kaç hisse, hangi yerleşim/görünüm,
			// sıralama ve şerit turu modülün kendi ayarındadır.
			// Görünüm sözlüğü Kur & Borsa ile aynıdır (hs_style/hs_arrow/hs_bg).
			bv.Count = intOpt("count", hisseSayiVarsayilan)
			if bv.Count < 3 {
				bv.Count = 3
			}
			if bv.Count > hisseSayiAzami {
				bv.Count = hisseSayiAzami
			}
			bv.HisseYerlesim = hisseYerlesimOf(get("hs_layout"), get("hs_style"))
			bv.HisseStil = hisseStilOf(get("hs_style"))
			// hs_ok ilk sürümün bool'u; eski düzenlerde kapatılmış olabilir.
			var eskiOk *bool
			if v, ok := b.Opts["hs_ok"].(bool); ok {
				eskiOk = &v
			}
			bv.HisseArrow = hisseOkOf(get("hs_arrow"), eskiOk)
			bv.HisseBg = hisseBgOf(get("hs_bg"))
			bv.HisseSirala = hisseSiralaOf(get("hs_sirala"))
			// Geçiş modeli + hız: sürekli (Habertürk) veya aralıklı.
			bv.HisseGecis = hisseGecisOf(get("hs_gecis"))
			bv.HisseHiz = hisseHizCoz(get("hs_hiz_secim"), intOpt("hs_speed", 0), intOpt("hs_hiz", 0), bv.Count)
			bv.HisseBekle = intOpt("hs_bekle", hisseBekleVarsayilan)
			if bv.HisseBekle < 1 {
				bv.HisseBekle = 1
			}
			if bv.HisseBekle > 30 {
				bv.HisseBekle = 30
			}
			bv.HisseKayma = intOpt("hs_kayma", hisseKaymaVarsayilan)
			if bv.HisseKayma < 200 {
				bv.HisseKayma = 200
			}
			if bv.HisseKayma > 3000 {
				bv.HisseKayma = 3000
			}
			bv.HisseSure = hisseSureTur(bv.Count, bv.HisseHiz)
			bv.HisseAra = boolOf("hs_ara", true)
			bv.HisseDegisim = boolOf("hs_degisim", false)
		}
		// Kategori modülleri: satır yerleşimi ve haber aralığı.
		bv.RowLayout = get("row_layout")
		switch bv.RowLayout {
		case "hero-left":
			bv.RowSlot, bv.RowSlotClass = "left", "cat-slot-left"
		case "hero-center":
			bv.RowSlot, bv.RowSlotClass = "center", "cat-slot-center"
		case "hero-right":
			bv.RowSlot, bv.RowSlotClass = "right", "cat-slot-right"
		default:
			bv.RowLayout = ""
		}
		// 0 geçerli bir başlangıçtır; -1 "otomatik" anlamına gelir.
		bv.ItemOffset = -1
		if n, ok := b.Opts["item_offset"].(float64); ok {
			bv.ItemOffset = int(n)
		} else if raw := get("item_offset"); raw != "" {
			if n, err := strconv.Atoi(strings.TrimSpace(raw)); err == nil {
				bv.ItemOffset = n
			}
		}
		if bv.ItemOffset < 0 {
			bv.ItemOffset = -1
		}
		if bv.ItemOffset > 500 {
			bv.ItemOffset = 500
		}
		bv.AkisStart = -1
		bv.RailStyle = get("rail_style")
		if bv.RailStyle != "yatay" {
			bv.RailStyle = "overlay"
		}
		if bv.Type == "kategori_slider" {
			bv.Count = intOpt("count", 5)
			if bv.Count < 1 {
				bv.Count = 1
			}
			if bv.Count > 20 {
				bv.Count = 20
			}
		}
		if bv.Type == "kategori_secimli" {
			bv.Count = intOpt("count", 3)
			if bv.Count < 1 {
				bv.Count = 1
			}
			if bv.Count > 12 {
				bv.Count = 12
			}
		}
		if bv.Type == "kategori_4lu" {
			bv.Count = intOpt("count", 4)
			if bv.Count < 1 {
				bv.Count = 1
			}
			if bv.Count > 12 {
				bv.Count = 12
			}
		}
		// Kategori gövdesi manşet düzeni:
		//   portre = solda 5 slaytlı dikey slider, sağda 2 sabit manşet
		//   uclu   = solda 3, ortada slider, sağda 2 manşet
		//   klasik = 1 büyük + 2 yan (eski düzen)
		bv.CatHeroStyle = get("hero_style")
		switch bv.CatHeroStyle {
		case "klasik", "portre", "uclu":
		default:
			bv.CatHeroStyle = "portre"
		}
		bv.CatHeroCount = intOpt("hero_count", 5)
		if bv.CatHeroCount < 2 {
			bv.CatHeroCount = 2
		}
		if bv.CatHeroCount > 10 {
			bv.CatHeroCount = 10
		}
		// Üçlü düzende genişlik ortadaki slider'a aittir; yan raylara da
		// yer kalması için daha dar aralıkta tutulur. Bağımsız slider
		// modülü manşet satırının ortasındaysa aynı dar aralık geçerlidir.
		heroDefault, heroMin, heroMax := 58, 45, 75
		if bv.CatHeroStyle == "uclu" || bv.RowLayout == "hero-center" {
			heroDefault, heroMin, heroMax = 44, 30, 60
		}
		bv.CatHeroWidth = intOpt("hero_width", heroDefault)
		if bv.CatHeroWidth < heroMin {
			bv.CatHeroWidth = heroMin
		}
		if bv.CatHeroWidth > heroMax {
			bv.CatHeroWidth = heroMax
		}
		bv.CatHeroRatio = get("hero_ratio")
		switch bv.CatHeroRatio {
		case "3/4", "1/1", "16/9", "4/5":
		default:
			bv.CatHeroRatio = "4/5"
		}
		// Tek başına eklenen slider tam genişlikte yatay görünür; manşet
		// satırına katıldığında dikey 4:5 varsayılanı korunur.
		if bv.Type == "kategori_slider" && bv.RowLayout == "" && get("hero_ratio") == "" {
			bv.CatHeroRatio = "16/9"
		}
		bv.CatListCols = intOpt("list_cols", 0)
		if bv.Type == "kategori_4lu" && get("list_cols") == "" {
			bv.CatListCols = 4
		}
		if bv.CatListCols != 2 && bv.CatListCols != 3 && bv.CatListCols != 4 {
			// Üçlü manşet şablonu 4'lü kart ızgarasıyla gelir; diğer
			// düzenlerde alt kartlar iki sütundur.
			if bv.CatHeroStyle == "uclu" {
				bv.CatListCols = 4
			} else {
				bv.CatListCols = 2
			}
		}
		// Akış vitrini (coinhit/enerji): toplam haber sınırı, her yüklemede
		// eklenecek haber sayısı ve otomatik yükleme.
		bv.AkisLimit = intOpt("akis_limit", 60)
		if bv.AkisLimit < 8 {
			bv.AkisLimit = 8
		}
		if bv.AkisLimit > 200 {
			bv.AkisLimit = 200
		}
		bv.AkisPer = intOpt("akis_per", 12)
		if bv.AkisPer < 12 {
			bv.AkisPer = 12
		}
		if bv.AkisPer > 36 {
			bv.AkisPer = 36
		}
		// Enerji ızgarası 12'lik (3 sütun × 4 kart) çalışır; ara değerleri
		// en yakın 12'nin katına yuvarla.
		if bv.AkisPer%12 != 0 {
			bv.AkisPer = ((bv.AkisPer + 11) / 12) * 12
		}
		bv.AkisAuto = boolOf("akis_auto", true)
		// Akışın üstündeki manşet alanı (portre/üçlü) varsayılan olarak görünür;
		// istenirse kapatılabilir.
		bv.AkisHero = boolOf("akis_hero", true)
		// Akış kartları tek yazı ailesi kullanır; boş bırakılırsa coinhit
		// varsayılanı (serif) kalır. Anahtarlar slider'daki title_font ile aynı.
		bv.AkisFont = get("akis_font")
		if _, ok := titleFontFamilies[bv.AkisFont]; !ok {
			bv.AkisFont = ""
		}
		// Kategori manşet slider'ı (portre/üçlü): slider modülündeki başlık
		// arkası, sayı göstergesi ve gölge seçenekleri bu blokta da geçerli.
		bv.HeroShadow = boolOf("hero_shadow", false)
		// Kategori sekmeli blok: kategori listesi, stil ve renkler.
		bv.CatTabs = parseCatTabs(b.Opts["sekmeler"])
		bv.TabStyle = get("tab_style")
		switch bv.TabStyle {
		case "segmented", "square", "pill", "underline", "boxed", "minimal", "tab", "tabflat":
		default:
			bv.TabStyle = "segmented"
		}
		bv.TabNav = boolOf("tab_nav", true)
		bv.TabBg = sanitizeHexColor(get("tab_bg"))
		bv.TabColor = sanitizeHexColor(get("tab_color"))
		bv.TabActiveBg = sanitizeHexColor(get("tab_active_bg"))
		bv.TabActiveColor = sanitizeHexColor(get("tab_active_color"))
		bv.TabRadius = -1
		// 0 geçerli bir değer (tamamen düz köşe) olduğu için intOpt yerine
		// ham değer okunur.
		if n, ok := b.Opts["tab_radius"].(float64); ok {
			bv.TabRadius = int(n)
		} else if raw := get("tab_radius"); raw != "" {
			if n, err := strconv.Atoi(strings.TrimSpace(raw)); err == nil {
				bv.TabRadius = n
			}
		}
		if bv.TabRadius > 28 {
			bv.TabRadius = 28
		}
		if bv.TabRadius < 0 {
			bv.TabRadius = -1
		}
		var tabVars []string
		if bv.TabBg != "" {
			tabVars = append(tabVars, "--kt-bg:"+bv.TabBg)
		}
		if bv.TabColor != "" {
			tabVars = append(tabVars, "--kt-fg:"+bv.TabColor)
		}
		if bv.TabActiveBg != "" {
			tabVars = append(tabVars, "--kt-active-bg:"+bv.TabActiveBg)
		}
		if bv.TabActiveColor != "" {
			tabVars = append(tabVars, "--kt-active-fg:"+bv.TabActiveColor)
		}
		if bv.TabRadius >= 0 {
			tabVars = append(tabVars, "--kt-radius:"+strconv.Itoa(bv.TabRadius)+"px")
		}
		if len(tabVars) > 0 {
			bv.TabCSS = template.CSS(strings.Join(tabVars, ";"))
		}
		if bv.Type == "kategori_sekmeli" && get("count") == "" {
			bv.Count = 5
		}
		// Üst manşet kart görünümü (başlık görsel üstünde stiller).
		bv.UMLayout = get("um_layout")
		switch bv.UMLayout {
		case "serit", "izgara":
		default:
			bv.UMLayout = "serit"
		}
		bv.UMCols = intOpt("um_cols", 4)
		if bv.UMCols < 2 || bv.UMCols > 4 {
			bv.UMCols = 4
		}
		bv.UMStyle = get("um_style")
		switch bv.UMStyle {
		case "klasik", "overlay", "overlay-top", "band", "frame", "corner":
		default:
			bv.UMStyle = "klasik"
		}
		bv.UMTitleColor = sanitizeHexColor(get("um_title_color"))
		bv.UMTitleBg = sanitizeHexColor(get("um_title_bg"))
		bv.UMTitleSize = intOpt("um_title_size", 0)
		if bv.UMTitleSize < 10 {
			bv.UMTitleSize = 0
		}
		if bv.UMTitleSize > 40 {
			bv.UMTitleSize = 40
		}
		bv.UMGradColor = sanitizeHexColor(get("um_grad_color"))
		bv.UMGradOpacity = intOpt("um_grad_opacity", -1)
		if bv.UMGradOpacity > 100 {
			bv.UMGradOpacity = 100
		}
		bv.UMShadow = boolOf("um_shadow", true)
		bv.UMCardBg = sanitizeHexColor(get("um_card_bg"))
		bv.UMCardDarkBg = sanitizeHexColor(get("um_card_dark_bg"))
		bv.UMCardShadow = boolOf("um_card_shadow", true)
		// 0 geçerli (başlık kutusunda yan boşluk yok); -1 = ayarlanmadı.
		bv.UMTitlePad = -1
		if n, ok := b.Opts["um_title_pad"].(float64); ok {
			bv.UMTitlePad = int(n)
		} else if raw := get("um_title_pad"); raw != "" {
			if n, err := strconv.Atoi(strings.TrimSpace(raw)); err == nil {
				bv.UMTitlePad = n
			}
		}
		if bv.UMTitlePad > 60 {
			bv.UMTitlePad = 60
		}
		if bv.UMTitlePad < 0 {
			bv.UMTitlePad = -1
		}
		umCSS := ""
		if bv.UMTitleColor != "" {
			umCSS += "--um-title-color:" + bv.UMTitleColor + ";"
		}
		if bv.UMTitleBg != "" {
			umCSS += "--um-title-bg:" + hexToRGBA(bv.UMTitleBg, 0.82) + ";"
		}
		if bv.UMTitleSize > 0 {
			umCSS += "--um-title-size:" + px(bv.UMTitleSize) + ";"
		}
		if bv.UMGradColor != "" {
			umCSS += "--um-grad-color:" + bv.UMGradColor + ";"
		}
		if bv.UMGradOpacity >= 0 {
			umCSS += "--um-grad-opacity:" + strconv.FormatFloat(float64(bv.UMGradOpacity)/100, 'f', 2, 64) + ";"
		}
		if !bv.UMShadow {
			umCSS += "--um-shadow:none;"
		}
		if bv.UMCardBg != "" {
			umCSS += "--um-card-bg-light:" + bv.UMCardBg + ";"
		}
		if bv.UMCardDarkBg != "" {
			umCSS += "--um-card-bg-dark:" + bv.UMCardDarkBg + ";"
		}
		if !bv.UMCardShadow {
			umCSS += "--um-card-shadow:none;--um-card-shadow-hover:none;"
		}
		if bv.UMTitlePad >= 0 {
			umCSS += "--um-title-pad-x:" + px(bv.UMTitlePad) + ";"
		}
		if umCSS != "" {
			bv.UMStyleCSS = template.CSS(umCSS)
		}
		if h := get("html"); h != "" {
			bv.RichHTML = template.HTML(platform.SanitizeRichHTML(h))
		}
		bv.KapakTasarimID = int64(intOf("kapak_tasarim", 0))
		showOverlayDefault := bv.KapakTasarimID == 0
		bv.ShowTitle = boolOf("overlay_title", showOverlayDefault)
		bv.ShowBadge = boolOf("overlay_badge", showOverlayDefault)
		bv.ShowSpot = boolOf("overlay_spot", showOverlayDefault)
		bv.TitleColor = sanitizeHexColor(get("title_color"))
		bv.TitleFont = get("title_font")
		if _, ok := titleFontFamilies[bv.TitleFont]; !ok {
			bv.TitleFont = ""
		}
		bv.TitleLineHeight = floatOf("title_line_height", 0)
		if bv.TitleLineHeight < 0.8 || bv.TitleLineHeight > 2.5 {
			bv.TitleLineHeight = 0
		}
		bv.TitleLetterSpacing = intOf("title_letter_spacing", 0)
		if bv.TitleLetterSpacing < -5 || bv.TitleLetterSpacing > 20 {
			bv.TitleLetterSpacing = 0
		}
		bv.TitleWordSpacing = intOf("title_word_spacing", 0)
		if bv.TitleWordSpacing < -5 || bv.TitleWordSpacing > 40 {
			bv.TitleWordSpacing = 0
		}
		bv.TitleCase = get("title_case")
		switch bv.TitleCase {
		case "none", "uppercase", "capitalize", "sentence", "lowercase":
		default:
			bv.TitleCase = ""
		}
		bv.OverlayColor = sanitizeHexColor(get("overlay_color"))
		opacityPreset := get("overlay_opacity_preset")
		if opacityPreset != "" && opacityPreset != "custom" {
			if n, err := strconv.Atoi(opacityPreset); err == nil {
				bv.OverlayOpacity = n
			} else {
				bv.OverlayOpacity = -1
			}
		} else {
			bv.OverlayOpacity = intOf("overlay_opacity", -1)
		}
		if bv.OverlayOpacity < -1 {
			bv.OverlayOpacity = -1
		} else if bv.OverlayOpacity > 100 {
			bv.OverlayOpacity = 100
		}
		bv.OverlayGradient = get("overlay_gradient")
		switch bv.OverlayGradient {
		case "none", "solid", "bottom", "left", "right", "diagonal":
		default:
			bv.OverlayGradient = ""
		}
		bv.OverlayHeight = intOf("overlay_height", -1)
		if bv.OverlayHeight < -1 {
			bv.OverlayHeight = -1
		} else if bv.OverlayHeight > 100 {
			bv.OverlayHeight = 100
		}
		bv.OverlayBlur = intOf("overlay_blur", -1)
		if bv.OverlayBlur < -1 {
			bv.OverlayBlur = -1
		} else if bv.OverlayBlur > 30 {
			bv.OverlayBlur = 30
		}
		overlayCSS := ""
		if bv.OverlayColor != "" || bv.OverlayOpacity >= 0 || bv.OverlayGradient != "" ||
			bv.OverlayHeight >= 0 || bv.OverlayBlur >= 0 {
			color := bv.OverlayColor
			if color == "" {
				color = "#0b0a11"
			}
			alpha := 0.62
			if bv.OverlayOpacity >= 0 {
				alpha = float64(bv.OverlayOpacity) / 100
			}
			rgba := hexToRGBA(color, alpha)
			switch bv.OverlayGradient {
			case "none":
				overlayCSS += "background:transparent;"
			case "bottom":
				overlayCSS += "background:linear-gradient(to top," + rgba + ",transparent);"
			case "left":
				overlayCSS += "background:linear-gradient(to right," + rgba + ",transparent);"
			case "right":
				overlayCSS += "background:linear-gradient(to left," + rgba + ",transparent);"
			case "diagonal":
				overlayCSS += "background:linear-gradient(135deg," + rgba + ",transparent);"
			case "solid":
				overlayCSS += "background:" + rgba + ";"
			case "":
				// Renk/saydamlık varsa düz renk; yalnız blur/yükseklik
				// seçildiyse stilin kendi arka planı korunur.
				if bv.OverlayColor != "" || bv.OverlayOpacity >= 0 {
					overlayCSS += "background:" + rgba + ";"
				}
			}
			if bv.OverlayHeight >= 0 {
				// Yükseklik bir TABAN'dır, sabit tavan değil: uzun başlık/spot
				// bandın üstünden taşıp kırpılmasın. İçerik bandı yukarı doğru
				// büyütür; kısa içerikte seçilen oran korunur.
				overlayCSS += "top:auto;bottom:0;left:0;right:0;min-height:" +
					strconv.Itoa(bv.OverlayHeight) + "%;height:auto;display:flex;align-items:flex-end;" +
					"box-sizing:border-box;"
			} else if bv.OverlayGradient == "bottom" || bv.OverlayGradient == "left" ||
				bv.OverlayGradient == "right" || bv.OverlayGradient == "diagonal" {
				overlayCSS += "inset:0;display:flex;align-items:flex-end;box-sizing:border-box;"
			}
		}
		if bv.OverlayBlur >= 0 {
			overlayCSS += "-webkit-backdrop-filter:blur(" + px(bv.OverlayBlur) + ");backdrop-filter:blur(" + px(bv.OverlayBlur) + ");"
		}
		if overlayCSS != "" {
			bv.OverlayStyle = template.CSS(overlayCSS)
		}
		titleCSS := ""
		if bv.TitleColor != "" {
			titleCSS += "color:" + bv.TitleColor + ";"
		}
		if bv.TitleFont != "" {
			titleCSS += "font-family:" + titleFontFamilies[bv.TitleFont] + ";"
		}
		if bv.TitleLineHeight > 0 {
			titleCSS += "line-height:" + strconv.FormatFloat(bv.TitleLineHeight, 'f', 2, 64) + ";"
		}
		if bv.TitleLetterSpacing != 0 {
			titleCSS += "letter-spacing:" + px(bv.TitleLetterSpacing) + ";"
		}
		if bv.TitleWordSpacing != 0 {
			titleCSS += "word-spacing:" + px(bv.TitleWordSpacing) + ";"
		}
		if titleCSS != "" {
			bv.TitleStyle = template.CSS(titleCSS)
		}
		catSize := intOf("cat_size", 0)
		if catSize < 10 {
			catSize = 0
		}
		if catSize > 40 {
			catSize = 40
		}
		catCSS := ""
		bv.CatColor = sanitizeHexColor(get("cat_color"))
		if bv.CatColor != "" {
			catCSS += "background-color:" + bv.CatColor + ";"
		}
		if catSize > 0 {
			catCSS += "font-size:" + px(catSize) + ";"
		}
		catPos := get("cat_pos")
		switch catPos {
		case "inline", "top-left", "top-right", "bottom-left", "bottom-center", "bottom-right":
			bv.CatPosition = catPos
		default:
			bv.CatPosition = "inline"
		}
		// Üst manşet ve karma kartlarda kategori, görseli saran .um-media /
		// .ky-media katmanının içinde konumlanır. Bu sayede başlık metninin
		// üstüne binmez; inline absolute koordinatları kullanılmaz.
		catInMedia := bv.Type == "ust_manset" || bv.Type == "kari_kartlar"
		// Öne çıkanlar kartında kategori başlık altındaki metin akışındadır;
		// pin biçimi burada mutlak konuma taşınmamalıdır.
		catInlineOnly := bv.Type == "one_cikanlar" && catPos == ""
		if !catInMedia && !catInlineOnly {
			switch bv.CatPosition {
			case "top-left":
				catCSS += "position:absolute;bottom:auto;top:12px;left:16px;z-index:3;"
			case "top-right":
				catCSS += "position:absolute;bottom:auto;top:12px;right:16px;z-index:3;"
			case "bottom-left":
				catCSS += "position:absolute;top:auto;bottom:12px;left:16px;z-index:3;"
			case "bottom-center":
				catCSS += "position:absolute;top:auto;bottom:12px;left:50%;transform:translateX(-50%);z-index:3;"
			case "bottom-right":
				catCSS += "position:absolute;top:auto;bottom:12px;right:16px;z-index:3;"
			}
		}
		bv.CatShape = get("cat_shape")
		switch bv.CatShape {
		case "badge", "pin", "pin2", "pill", "outline", "corner":
		default:
			bv.CatShape = "badge"
		}
		// Harita pin biçimleri varsayılan olarak görselin altına yerleşir.
		// Kullanıcı cat_pos ile açıkça başka bir konum seçtiyse o korunur.
		if (bv.CatShape == "pin" || bv.CatShape == "pin2") && get("cat_pos") == "" && !catInlineOnly {
			if bv.CatShape == "pin2" {
				bv.CatPosition = "bottom-center"
			} else {
				bv.CatPosition = "bottom-left"
			}
			if !catInMedia {
				if bv.CatShape == "pin2" {
					catCSS += "position:absolute;top:auto;bottom:-8px;left:50%;transform:translateX(-50%);z-index:3;"
				} else {
					catCSS += "position:absolute;top:auto;bottom:-8px;left:16px;z-index:3;"
				}
			}
		}
		bv.CatClip = sanitizeCatClip(get("cat_clip"))
		bv.CatLeft = sanitizeCatLength(get("cat_left"))
		bv.CatBottom = sanitizeCatLength(get("cat_bottom"))
		if bv.CatClip != "" {
			catCSS += "clip-path:" + bv.CatClip + ";"
		}
		if !catInlineOnly {
			if catInMedia {
				if bv.CatLeft != "" {
					catCSS += "--cat-left:" + bv.CatLeft + ";"
				}
				if bv.CatBottom != "" {
					catCSS += "--cat-top:auto;--cat-bottom:" + bv.CatBottom + ";"
				}
			} else {
				// Konum seçilmemiş olsa bile özel left/bottom girildiyse
				// etiketi mutlak konuma al; aksi halde satır içi kalır ve
				// koordinatlar uygulanmaz.
				if bv.CatPosition == "inline" && (bv.CatLeft != "" || bv.CatBottom != "") {
					bv.CatPosition = "bottom-left"
					catCSS += "position:absolute;top:auto;bottom:12px;z-index:3;"
				}
				if bv.CatPosition != "inline" {
					if bv.CatLeft != "" && bv.CatPosition != "top-right" && bv.CatPosition != "bottom-right" {
						catCSS += "left:" + bv.CatLeft + ";"
					}
					if bv.CatBottom != "" && strings.HasPrefix(bv.CatPosition, "bottom") {
						catCSS += "bottom:" + bv.CatBottom + ";"
					}
				}
			}
		}
		if catCSS != "" {
			bv.CatStyle = template.CSS(catCSS)
		}
		bv.HeadingStyle = get("head_style")
		switch bv.HeadingStyle {
		case "line", "bar", "underline", "pill", "dot",
			"corner", "tab", "boxed", "gradient", "none":
		default:
			bv.HeadingStyle = "line"
		}
		bv.HideHeading = boolOf("hide_heading", false)
		bv.HideBadge = boolOf("hide_badge", false)
		// Paylaşımlı kart stili (Bloomberg vb.) — kategori ızgarası, üst manşet
		// ve öne çıkanlar modüllerinde ortak. Renkler açık/koyu tema için ayrı
		// verilir; böylece dark mode'da bant arka planı istenen renge döner.
		bv.CardStyle = get("card_style")
		switch bv.CardStyle {
		case "bloomberg", "bloomberg-cizgi", "bant-alt", "vurgu",
			"cerceve", "kose", "cam", "alt-cizgi":
		default:
			bv.CardStyle = ""
		}
		if bv.CardStyle != "" {
			cardCSS := ""
			if v := sanitizeHexColor(get("card_title_bg")); v != "" {
				cardCSS += "--cs-light-bg:" + v + ";"
			}
			if v := sanitizeHexColor(get("card_title_fg")); v != "" {
				cardCSS += "--cs-light-fg:" + v + ";"
			}
			if v := sanitizeHexColor(get("card_dark_bg")); v != "" {
				cardCSS += "--cs-dark-bg:" + v + ";"
			}
			if v := sanitizeHexColor(get("card_dark_fg")); v != "" {
				cardCSS += "--cs-dark-fg:" + v + ";"
			}
			if v := sanitizeHexColor(get("card_accent")); v != "" {
				cardCSS += "--cs-accent:" + v + ";"
			}
			if n := intOpt("card_title_size", 0); n >= 10 && n <= 40 {
				cardCSS += "--cs-size:" + px(n) + ";"
			}
			// Kart gövdesi (kabuk) arka planı: bant renginden bağımsız.
			// Açık/koyu tema ayrı değişkenlere yazılır; dark mode kuralı
			// inline değeri ezmeden doğru rengi seçebilsin.
			if v := sanitizeHexColor(get("card_box_bg")); v != "" {
				cardCSS += "--cs-card-light-bg:" + v + ";"
			}
			if v := sanitizeHexColor(get("card_box_dark_bg")); v != "" {
				cardCSS += "--cs-card-dark-bg:" + v + ";"
			}
			// Gölge kapatma: hem normal hem hover gölgesi kalkar (ana
			// sayfadaki um_card_shadow ile aynı davranış).
			if !boolOf("card_shadow", true) {
				cardCSS += "--cs-shadow:none;--cs-shadow-hover:none;"
			}
			if cardCSS != "" {
				bv.CardCSS = template.CSS(cardCSS)
			}
		}
		bv.ModuleStyle = get("mod_style")
		switch bv.ModuleStyle {
		case "boxed", "soft", "accent", "glass":
		default:
			bv.ModuleStyle = ""
		}
		bv.HideMobile = boolOf("hide_mobile", false)
		if bv.HideMobile {
			bv.Wrap = true
		}
		pagerType := get("pager_type")
		if pagerType != "dots" && pagerType != "numbers" && pagerType != "none" {
			if bv.Type == "manset6" {
				pagerType = "numbers"
			} else if boolOf("dots", true) {
				pagerType = "dots"
			} else {
				pagerType = "none"
			}
		}
		bv.PagerType = pagerType
		bv.PagerLayout = "vertical"
		if get("pager_layout") == "horizontal" {
			bv.PagerLayout = "horizontal"
		}
		bv.PagerColor = sanitizeHexColor(get("pager_color"))
		bv.PagerShape = get("pager_shape")
		if bv.PagerShape != "square" && bv.PagerShape != "round" && bv.PagerShape != "outline" {
			if bv.Type == "manset6" || bv.Type == "kategori_liste" {
				// Kategori manşet göstergesi tasarım gereği yuvarlaktır.
				bv.PagerShape = "round"
			} else {
				bv.PagerShape = "square"
			}
		}
		bv.PagerPos = get("pager_pos")
		if bv.PagerPos != "left" && bv.PagerPos != "center" && bv.PagerPos != "right" {
			if bv.Type == "manset6" && bv.PagerLayout == "vertical" {
				bv.PagerPos = "left"
			} else {
				bv.PagerPos = "right"
			}
		}
		bv.ShowAll = boolOf("show_all", false)
		if bv.Type == "manset2" || bv.Type == "slider" {
			bv.M2Bg = sanitizeHexColor(get("m2_bg"))
			if bv.M2Bg == "" {
				bv.M2Bg = "#0c1021"
			}
			bv.M2Accent = sanitizeHexColor(get("m2_accent"))
			if bv.M2Accent == "" {
				bv.M2Accent = "#c90914"
			}
			bv.M2Fade = sanitizeHexColor(get("m2_fade"))
			bv.M2NumBg = sanitizeHexColor(get("m2_num_bg"))
			bv.M2NumActiveBg = sanitizeHexColor(get("m2_num_active_bg"))
			bv.M2Side = get("m2_side")
			switch bv.M2Side {
			case "alternate", "right", "left":
			default:
				bv.M2Side = "alternate"
			}
			bv.M2Height = intOf("m2_height", 330)
			if bv.M2Height < 220 {
				bv.M2Height = 220
			}
			if bv.M2Height > 600 {
				bv.M2Height = 600
			}
		}
		bv.SliderStyle = get("slider_style")
		validSliderStyle := bv.SliderStyle == "edirne" || bv.SliderStyle == "minimal" ||
			bv.SliderStyle == "magazin" || bv.SliderStyle == "sinema" || bv.SliderStyle == "stil5" ||
			bv.SliderStyle == "sayili"
		if bv.SliderStyle != "default" && !validSliderStyle {
			bv.SliderStyle = "default"
		}
		// Sayılı şerit stili (Habertürk spor düzeni) numaralı göstergenin
		// parçasıdır: şerit olmadan stil eksik kalır. Bu yüzden gösterge
		// ayarı ne olursa olsun numaralı şerit basılır; son hücre "+" olur.
		if bv.SliderStyle == "sayili" {
			bv.PagerType = "numbers"
			bv.ShowAll = true
		}
		bv.PagerSize = intOf("pager_size", 0)
		if bv.PagerSize > 0 && bv.PagerSize < 14 {
			bv.PagerSize = 14
		}
		if bv.PagerSize > 48 {
			bv.PagerSize = 48
		}
		bv.Zone = get("zone")
		bv.StPad = sanitizeBoxShorthand(lenOf("stPad"), false, false)
		bv.StMargin = sanitizeBoxShorthand(lenOf("stMargin"), true, true)
		bv.StGapTop = sanitizeLengthValue(lenOf("stGapTop"), true, true)
		bv.StGapBottom = sanitizeLengthValue(lenOf("stGapBottom"), true, true)
		bv.StGapX = sanitizeLengthValue(lenOf("stGapX"), false, false)
		bv.StGapY = sanitizeLengthValue(lenOf("stGapY"), false, false)
		bv.StBorderW = intOf("stBorderW", 0)
		bv.StRadius = intOpt("stRadius", 0)
		if v, ok := b.Opts["stRadius"]; ok && v != nil {
			// 0 "köşeli" demektir; yalnız alan hiç yoksa/boşsa tema
			// yarıçapı geçerli. null ve boş string "boş" sayılır.
			if s, isStr := v.(string); !isStr || strings.TrimSpace(s) != "" {
				bv.StRadiusSet = true
			}
		}
		bv.StBorderColor = get("stBorderColor")
		bv.StBg = get("stBg")
		bv.StAbs = boolOf("stAbs", false)
		bv.StX = intOf("stX", 0)
		bv.StY = intOf("stY", 0)
		bv.StW = intOf("stW", 0)
		bv.StZ = intOf("stZ", 0)
		bv.buildStyle()
		if bv.ModuleStyle != "" {
			bv.Wrap = true
		}
		switch bv.Zone {
		case "ust", "ana", "yan", "genis", "govde":
			// geçerli
		default:
			bv.Zone = "ana" // varsayılan: sol ana kolon
		}
		// "Otomatik" şablon bölgeye göre çözülür: sağ raya (yan) taşınan
		// modül kendiliğinden kompakt görünüme geçer. Kullanıcı açıkça bir
		// şablon seçtiyse ona dokunulmaz.
		if bv.Type == "kripto_para" {
			bv.KriptoDar = darBolge(bv.Zone)
			bv.KriptoEtkin = kriptoEtkinStil(bv.KriptoStil, bv.KriptoDar)
			bv.KriptoListe = kriptoListeStili(bv.KriptoEtkin)
		}
		if bv.Type == "hisseler" {
			bv.HisseDar = darBolge(bv.Zone)
			bv.HisseEtkin = hisseEtkinStil(bv.HisseStil, bv.HisseDar)
		}
		if bv.Type == "" {
			continue
		}
		out = append(out, bv)
	}
	return out
}

// hexToRGBA, #rrggbb/#rgb renklerini CSS rgba() biçimine çevirir.
func hexToRGBA(hex string, alpha float64) string {
	if alpha < 0 {
		alpha = 0
	}
	if alpha > 1 {
		alpha = 1
	}
	h := strings.TrimPrefix(strings.TrimSpace(hex), "#")
	if len(h) == 3 {
		h = string([]byte{h[0], h[0], h[1], h[1], h[2], h[2]})
	}
	n, err := strconv.ParseUint(h, 16, 32)
	if err != nil || len(h) != 6 {
		return fmt.Sprintf("rgba(11,10,17,%.2f)", alpha)
	}
	return fmt.Sprintf("rgba(%d,%d,%d,%.2f)", (n>>16)&0xff, (n>>8)&0xff, n&0xff, alpha)
}

var titleFontFamilies = map[string]string{
	"system":    `system-ui, -apple-system, Segoe UI, Roboto, Arial, sans-serif`,
	"serif":     `Georgia, Times New Roman, Times, serif`,
	"sans":      `Arial, Helvetica, sans-serif`,
	"condensed": `Arial Narrow, Roboto Condensed, Impact, sans-serif`,
	"display":   `Trebuchet MS, Segoe UI, Arial, sans-serif`,
	"mono":      `ui-monospace, SFMono-Regular, Menlo, Consolas, monospace`,
}

var (
	catClipPathRe = regexp.MustCompile(`(?i)^(polygon|inset|circle|ellipse)\([0-9a-z%.,\s\-()]+\)$`)
	catLengthRe   = regexp.MustCompile(`^-?(?:\d+|\d*\.\d+)(?:px|%|rem|em|vw|vh)$`)
	catNumberRe   = regexp.MustCompile(`^-?(?:\d+|\d*\.\d+)$`)
	catCalcRe     = regexp.MustCompile(`(?i)^calc\([0-9a-z%.,\s\-+*/()]+\)$`)
)

// sanitizeCatClip yalnızca güvenli CSS clip-path fonksiyonlarını kabul eder.
func sanitizeCatClip(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	if strings.EqualFold(v, "none") {
		return "none"
	}
	if catClipPathRe.MatchString(v) {
		return v
	}
	return ""
}

// sanitizeCatLength left/bottom için px, %, rem, em, vw/vh, auto ve calc()
// değerlerini kabul eder; birimsiz sayıları px'e çevirir.
func sanitizeCatLength(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	if strings.EqualFold(v, "auto") {
		return "auto"
	}
	if catLengthRe.MatchString(v) || catCalcRe.MatchString(v) {
		return v
	}
	if catNumberRe.MatchString(v) {
		return v + "px"
	}
	return ""
}

// sanitizeLengthValue tek bir CSS uzunluğunu doğrular. Modül iç/dış
// boşluklarında px/rem/em/%/vw/vh/calc() ve birimsiz sayı (px) kabul edilir.
// allowAuto margin için, allowNegative ise negatif margin için kullanılır.
func sanitizeLengthValue(v string, allowAuto, allowNegative bool) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	if allowAuto && strings.EqualFold(v, "auto") {
		return "auto"
	}
	if !allowNegative && strings.HasPrefix(v, "-") {
		return ""
	}
	if catLengthRe.MatchString(v) || catCalcRe.MatchString(v) {
		return v
	}
	if catNumberRe.MatchString(v) {
		return v + "px"
	}
	return ""
}

// splitCSSValues, CSS kısayol değerini boşluklardan ayırır; calc() gibi
// parantezli ifadelerin içindeki boşlukları korur.
func splitCSSValues(v string) []string {
	var out []string
	var b strings.Builder
	depth := 0
	for _, r := range v {
		switch r {
		case '(':
			depth++
			b.WriteRune(r)
		case ')':
			if depth > 0 {
				depth--
			}
			b.WriteRune(r)
		case ' ', '\t', '\n', '\r':
			if depth == 0 {
				if b.Len() > 0 {
					out = append(out, b.String())
					b.Reset()
				}
			} else {
				b.WriteRune(r)
			}
		default:
			b.WriteRune(r)
		}
	}
	if b.Len() > 0 {
		out = append(out, b.String())
	}
	return out
}

// sanitizeBoxShorthand padding/margin için 1-4 değerli CSS kısayolunu
// doğrular (örn. "1rem", "10px 20px", "0 auto"). Geçersiz değer tümden
// reddedilir; böylece kısmen uygulanmış stil oluşmaz.
func sanitizeBoxShorthand(v string, allowAuto, allowNegative bool) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	parts := splitCSSValues(v)
	if len(parts) == 0 || len(parts) > 4 {
		return ""
	}
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		s := sanitizeLengthValue(p, allowAuto, allowNegative)
		if s == "" {
			return ""
		}
		out = append(out, s)
	}
	return strings.Join(out, " ")
}

// buildStyle modül başına görünüm/ölçü ayarlarından (st* opts) güvenli bir
// inline stil üretir. Yalnızca geçerli/kırpılmış değerler kullanılır; hiçbir
// ayar yoksa Wrap=false kalır ve public çıktı değişmez.
func (b *homeBlock) buildStyle() {
	var p []string
	if b.StPad != "" {
		p = append(p, "padding:"+b.StPad)
	}
	if b.StMargin != "" {
		p = append(p, "margin:"+b.StMargin)
	}
	if b.StGapTop != "" {
		p = append(p, "margin-top:"+b.StGapTop)
	}
	if b.StGapBottom != "" {
		p = append(p, "margin-bottom:"+b.StGapBottom)
	}
	// Kart ızgaraları bu değişkenleri tüketir (public.css). Değer yoksa
	// modülün kendi varsayılan boşluğu geçerli kalır.
	if b.StGapX != "" {
		p = append(p, "--blk-gap-x:"+b.StGapX)
	}
	if b.StGapY != "" {
		p = append(p, "--blk-gap-y:"+b.StGapY)
	}
	if b.StRadiusSet {
		// Yerel değişkenler: finans modüllerinin iç kartları (public.css)
		// --hp-radius / --hp-radius-sm tüketir. 0 verildiğinde modülün
		// tamamı köşeli olur; alan boşsa tema değeri miras alınır.
		r := clampInt(b.StRadius, 0, 60)
		sm := r - 2
		if sm < 0 {
			sm = 0
		}
		p = append(p, "--hp-radius:"+px(r), "--hp-radius-sm:"+px(sm))
		p = append(p, "border-radius:"+px(r))
	}
	if b.StBorderW > 0 {
		col := "var(--hp-border)"
		if c := sanitizeHexColor(b.StBorderColor); c != "" {
			col = c
		}
		p = append(p, "border:"+px(clampInt(b.StBorderW, 0, 12))+" solid "+col)
	}
	// Finans modüllerinde arka plan modülün kendi seçimiyle (kb_bg/hs_bg/
	// kp_bg/kc_bg) yönetilir; ortak "Arka plan rengi" alanı sarmalayıcıya
	// basılmaz. Aksi hâlde "yok" (saydam) seçilse bile kutu görünürdü.
	suppressBg := false
	switch b.Type {
	case "kur_borsa", "hisseler", "kripto_para", "kur_cevirici", "servis_bandi":
		suppressBg = true
	}
	if c := sanitizeHexColor(b.StBg); c != "" && !suppressBg {
		p = append(p, "background:"+c)
	}
	if b.StAbs {
		p = append(p, "position:absolute")
		p = append(p, "left:"+px(clampInt(b.StX, -2000, 4000)))
		p = append(p, "top:"+px(clampInt(b.StY, -2000, 4000)))
		if b.StW > 0 {
			p = append(p, "width:"+px(clampInt(b.StW, 40, 4000)))
		}
		if b.StZ != 0 {
			p = append(p, "z-index:"+itoa(clampInt(b.StZ, -10, 100)))
		}
	}
	if len(p) == 0 {
		return
	}
	b.Wrap = true
	b.Style = template.CSS(strings.Join(p, ";"))
}
