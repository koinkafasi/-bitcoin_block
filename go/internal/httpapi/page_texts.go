package httpapi

import (
	"context"
	"encoding/json"
	"html/template"
	"net/http"
	"strings"

	"theblock/site/internal/platform"
	"theblock/site/internal/repo"
)

// pageTextDef tasarımcıda düzenlenebilen tek şablon metninin tanımı.
type pageTextDef struct {
	Key       string
	Label     string
	Hint      string
	Default   string
	Multiline bool
	// Raw true ise {…} yer tutucuları değiştirilmez (HTML gövde gibi).
	Raw bool
}

// pageTextItem tasarımcı formuna basılan metin satırı. Value yalnız bu dil
// için kaydedilmiş override'dır; boşsa Default (yer tutucu) gösterilir.
type pageTextItem struct {
	Key       string
	Label     string
	Hint      string
	Value     string
	Default   string
	Multiline bool
	// Rich true ise alan zengin metin editörüyle (contenteditable) basılır.
	Rich bool
	// RichValue zengin alanın başlangıç HTML'idir (kaçışsız basılır).
	RichValue template.HTML
}

// Kategori/yazı metinlerinin kod varsayılanları.
const (
	defKategoriBaslik    = "{kategori} Haberleri"
	defKategoriTumu      = "Tümü"
	defKategoriBosBaslik = "Bu kategoride henüz yayınlanmış haber yok."
	defKategoriBosMetin  = "Yeni içerikler yayınlandıkça burada listelenecek."
	defArticleOkuma      = "{dk} dk okuma"
	defArticleGoruntule  = "{sayi} görüntülenme"
	defArticleEditor     = "Editör"
	defArticleAIOzet     = "AI ÖZET"
	defArticleAnasayfa   = "Anasayfa"
)

func pageTextDefsFor(page string) []pageTextDef {
	switch {
	case isKategoriPage(page):
		return []pageTextDef{
			{Key: "kategori.baslik", Label: "Kategori başlığı", Hint: "{kategori} yerine kategori adı yazılır.", Default: defKategoriBaslik},
			{Key: "kategori.intro", Label: "Kategori açıklaması", Hint: "Başlığın altındaki SEO metni. Boşsa kategorinin hazır açıklaması kullanılır.", Multiline: true},
			{Key: "kategori.tumu", Label: "“Tümü” sekmesi", Default: defKategoriTumu},
			{Key: "kategori.bos_baslik", Label: "Boş kategori başlığı", Default: defKategoriBosBaslik},
			{Key: "kategori.bos_metin", Label: "Boş kategori metni", Default: defKategoriBosMetin, Multiline: true},
		}
	case page == "article":
		return []pageTextDef{
			{Key: "article.anasayfa", Label: "Kırıntı: Anasayfa", Default: defArticleAnasayfa},
			{Key: "article.editor", Label: "Editör etiketi", Default: defArticleEditor},
			{Key: "article.okuma_suresi", Label: "Okuma süresi", Hint: "{dk} okuma dakikasıyla değişir.", Default: defArticleOkuma},
			{Key: "article.goruntulenme", Label: "Görüntülenme", Hint: "{sayi} görüntülenme sayısıyla değişir.", Default: defArticleGoruntule},
			{Key: "article.ai_ozet", Label: "AI özet etiketi", Default: defArticleAIOzet},
		}
	case isSayfaPage(page):
		// Kurumsal sayfalar (Faz 143): başlık + gövde metni.
		return []pageTextDef{
			{Key: "sayfa.baslik", Label: "Sayfa başlığı", Hint: "Boşsa sayfanın kayıtlı adı kullanılır."},
			{Key: "sayfa.govde", Label: "Sayfa içeriği", Hint: "Boşsa sayfanın kendi içeriği basılır. Buradaki metin kaydedildiğinde öncelik kazanır.", Multiline: true, Raw: true},
		}
	}
	return nil
}

// categoryTextDefs kategori metinlerini bu kategoriye özel varsayılanlarla
// döndürür (intro için hazır kategori açıklaması).
func categoryTextDefs(slug string) []pageTextDef {
	defs := pageTextDefsFor("kategori")
	intro := categoryIntro(slug)
	for i := range defs {
		if defs[i].Key == "kategori.intro" {
			defs[i].Default = intro
		}
	}
	return defs
}

// pageTextVars yer tutucuları (ör. {kategori}) doldurur.
func pageTextVars(s string, vars map[string]string) string {
	if s == "" || len(vars) == 0 {
		return s
	}
	for k, v := range vars {
		s = strings.ReplaceAll(s, "{"+k+"}", v)
	}
	return s
}

// pageTextBaseKey kategoriye özel sayfaların genel kategori anahtarını verir.
func pageTextBaseKey(page string) string {
	if strings.HasPrefix(page, "kategori:") {
		return "kategori"
	}
	return page
}

// resolvePageTexts saf çözümleyici: verilen override satırlarından metinleri
// üretir. Test edilebilirlik için DB/HTTP'den bağımsızdır.
func resolvePageTexts(pageKey, baseKey string, rows []repo.PageText, locale, defLocale string, defs []pageTextDef, vars map[string]string) map[string]string {
	lookup := func(key string) (string, bool) {
		for _, loc := range []string{locale, defLocale} {
			if loc == "" {
				continue
			}
			for _, pk := range []string{pageKey, baseKey} {
				if pk == "" {
					continue
				}
				for _, r := range rows {
					if r.PageKey != pk || r.TextKey != key || r.Locale != loc {
						continue
					}
					if strings.TrimSpace(r.Value) == "" {
						continue
					}
					return r.Value, true
				}
			}
		}
		return "", false
	}
	out := make(map[string]string, len(defs))
	for _, d := range defs {
		v, ok := lookup(d.Key)
		if !ok {
			v = d.Default
		}
		if !d.Raw {
			v = pageTextVars(v, vars)
		}
		out[d.Key] = v
	}
	return out
}

// pageTextsFor bir sayfanın çözümlenmiş metinlerini döndürür. Okuma hatasında
// kod varsayılanlarıyla devam edilir: metin yüzünden sayfa düşmemeli.
func (s *Server) pageTextsFor(ctx context.Context, siteID int64, pageKey, locale string, defs []pageTextDef, vars map[string]string) map[string]string {
	defLocale := s.defaultLang(ctx)
	baseKey := pageTextBaseKey(pageKey)
	var rows []repo.PageText
	if r, err := repo.ListPageTexts(ctx, s.pool, siteID, pageKey); err == nil {
		rows = append(rows, r...)
	}
	if baseKey != "" && baseKey != pageKey {
		if r, err := repo.ListPageTexts(ctx, s.pool, siteID, baseKey); err == nil {
			rows = append(rows, r...)
		}
	}
	return resolvePageTexts(pageKey, baseKey, rows, locale, defLocale, defs, vars)
}

// builderPageTextItems tasarımcı formu için bu sayfanın metin satırlarını
// hazırlar. Value yalnız istenen dildeki override'dır (yoksa boş).
func builderPageTextItems(ctx context.Context, s *Server, siteID int64, page, locale string) []pageTextItem {
	defs := pageTextDefsFor(page)
	// Kategoriye özel sayfada intro yer tutucusu o kategorinin hazır
	// açıklamasını göstersin.
	if slug := strings.TrimPrefix(page, "kategori:"); slug != page && slug != "" {
		defs = categoryTextDefs(slug)
	}
	// Kurumsal sayfa: başlık yer tutucusu sayfanın adı, gövde alanı ise
	// sayfanın mevcut içeriğidir (kaydedilene kadar değişmez).
	var sayfa *repo.Page
	if slug := strings.TrimPrefix(page, "sayfa:"); slug != page && slug != "" {
		if p, err := repo.GetPageBySlug(ctx, s.pool, slug, false); err == nil {
			sayfa = &p
			for i := range defs {
				if defs[i].Key == "sayfa.baslik" {
					defs[i].Default = p.Title
				}
			}
		}
	}
	baseKey := pageTextBaseKey(page)
	var rows []repo.PageText
	if r, err := repo.ListPageTexts(ctx, s.pool, siteID, page); err == nil {
		rows = append(rows, r...)
	}
	if baseKey != "" && baseKey != page {
		if r, err := repo.ListPageTexts(ctx, s.pool, siteID, baseKey); err == nil {
			rows = append(rows, r...)
		}
	}
	items := make([]pageTextItem, 0, len(defs))
	for _, d := range defs {
		val := ""
		for _, r := range rows {
			if r.TextKey == d.Key && r.Locale == locale && strings.TrimSpace(r.Value) != "" {
				val = r.Value
				break
			}
		}
		rich := false
		richHTML := template.HTML("")
		// Kurumsal sayfa gövdesi: override yoksa mevcut içerik editörde
		// görünsün (kaydedilince override olur, aksi halde sabit içerik kalır).
		if sayfa != nil && d.Key == "sayfa.govde" {
			rich = true
			if strings.TrimSpace(val) == "" {
				// Sunucu üretimi (bloklardan): zaten kaçışlı.
				val = string(s.blocksToHTML(sayfa.Detail, nil))
				richHTML = template.HTML(val)
			} else {
				// Kayıtta süzülüyor; eski satırlar ham olabilir → basımda da
				// süzülür (savunma derinliği, kalıcı XSS).
				richHTML = template.HTML(platform.SanitizeRichHTML(val))
			}
		}
		items = append(items, pageTextItem{
			Key: d.Key, Label: d.Label, Hint: d.Hint,
			Value: val, Default: d.Default, Multiline: d.Multiline,
			Rich: rich, RichValue: richHTML,
		})
	}
	return items
}

// handleBuilderPageTextSave tasarımcıdaki "Sayfa Metinleri" kartını kaydeder.
// Boş değer ilgili override'ı siler (varsayılana dönüş).
func (s *Server) handleBuilderPageTextSave(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	page := builderPage(r)
	defs := pageTextDefsFor(page)
	if len(defs) == 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "bu sayfada düzenlenebilir metin yok"})
		return
	}
	var req struct {
		Locale string            `json:"locale"`
		Texts  map[string]string `json:"texts"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "geçersiz istek: " + err.Error()})
		return
	}
	locale := strings.TrimSpace(req.Locale)
	if locale == "" {
		locale = s.defaultLang(ctx)
	}
	known := make(map[string]bool, len(defs))
	for _, d := range defs {
		known[d.Key] = true
	}
	uid := userFromContext(ctx).ID
	saved := 0
	for k, v := range req.Texts {
		if !known[k] {
			continue
		}
		// Kurumsal sayfa gövdesi zengin metindir; güvenli alt kümeye indirgenir.
		if k == "sayfa.govde" {
			v = platform.SanitizeRichHTML(v)
		}
		if err := repo.SavePageText(ctx, s.pool, s.curSiteID(r), page, k, locale, v, uid); err != nil {
			s.writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		saved++
	}
	s.audit(r, "tasarim_sayfa_metni", page+": "+itoa(saved)+" metin ("+locale+")")
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "saved": saved, "locale": locale})
}
