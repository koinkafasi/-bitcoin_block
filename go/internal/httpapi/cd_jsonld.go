package httpapi

import (
	"encoding/json"
	"html/template"
	"strings"

	"theblock/site/internal/repo"
	"theblock/site/internal/webui/templkit"
)

func jsonldScript(v any) template.HTML {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return template.HTML(`<script type="application/ld+json">` + string(b) + `</script>`)
}

// orgJSONLD ana sayfa/kurumsal: Organization + WebSite.
func orgJSONLD(siteName, base, canonical string) template.HTML {
	return jsonldScript(map[string]any{
		"@context": "https://schema.org",
		"@graph": []any{
			map[string]any{"@type": "Organization", "name": siteName, "url": base + "/"},
			map[string]any{
				"@type":      "WebSite",
				"name":       siteName,
				"url":        base + "/",
				"inLanguage": "tr",
				"potentialAction": map[string]any{
					"@type":       "SearchAction",
					"target":      base + "/arama?q={search_term_string}",
					"query-input": "required name=search_term_string",
				},
			},
		},
		"url": canonical,
	})
}

// makaleLDGirdi NewsArticle üretimi için girdi.
type makaleLDGirdi struct {
	View      templkit.CDArticleView
	YayinISO  string // datePublished
	GuncelISO string // dateModified (yoksa YayinISO)
	ID        string // content_id / identifier
	YazarSlug string
	Keywords  []string
	GorselAlt string
	// Gorsel og:image / JSON-LD için MUTLAK görsel adresi (göreli
	// `/kg/...` Open Graph'ta geçersizdir).
	Gorsel string
	// Kilitli içerik abonelik gerektiriyor mu. Google, paywall'lu
	// içeriğin BİLDİRİLMESİNİ istiyor: aksi hâlde Googlebot'un gördüğü
	// tam metinle kullanıcının gördüğü kesik metin arasındaki fark
	// "cloaking" sayılabiliyor.
	Kilitli bool
	// Logo yayıncı logosunun MUTLAK adresi. Google'ın NewsArticle
	// doğrulayıcısı `publisher.logo` alanını ImageObject bekliyor; boş
	// dize geçersiz ve zengin sonuç testinde hata veriyor.
	Logo string
}

// ldDilKodu yapısal veri için dil kodu. Boş locale eski kayıtlarda
// görülüyor ve o içerikler Türkçe (bkz. cdAyniDil). Bölgesel varyant
// (tr-TR) schema.org'da geçerli olduğu için olduğu gibi bırakılıyor.
func ldDilKodu(locale string) string {
	l := strings.TrimSpace(locale)
	if l == "" {
		return "tr"
	}
	return strings.ToLower(l)
}

func makaleJSONLD(g makaleLDGirdi) template.HTML {
	v := g.View
	yazar := map[string]any{"@type": "Person", "name": v.Author}
	if g.YazarSlug != "" {
		yazar["url"] = strings.TrimRight(v.BaseURL, "/") + "/yazar/" + g.YazarSlug
	}

	yayinci := map[string]any{
		"@type": "NewsMediaOrganization",
		"name":  v.Shell.SiteName,
		"url":   strings.TrimRight(v.BaseURL, "/"),
	}
	// Logo yoksa alan HİÇ yazılmıyor: boş dize, alanı atlamaktan daha
	// kötü — doğrulayıcı onu geçersiz değer sayıyor.
	if g.Logo != "" {
		yayinci["logo"] = map[string]any{"@type": "ImageObject", "url": g.Logo}
	}

	makale := map[string]any{
		"@type": "NewsArticle",
		// Dil sayfadan okunuyor. Sabit "tr" iken İngilizce çeviriler
		// <html lang="en"> ile basılıp yapısal veride "tr" diyordu:
		// aynı sayfa Google'a iki farklı dil sinyali veriyordu.
		"inLanguage": ldDilKodu(v.Locale),
		"url":        v.Canonical,
		"headline":   v.Headline,
		"abstract":   v.Desc,
		// Ücretsiz erişim durumu HER ZAMAN yazılıyor: `true` olduğunda da
		// açık bir sinyal, Google için belirsizlik bırakmıyor.
		"isAccessibleForFree": !g.Kilitli,
		"mainEntityOfPage":    v.Canonical,
		"articleSection":      v.SectionName,
		"author":              []any{yazar},
		"publisher":           yayinci,
	}

	// KAYNAK ATFI (E-E-A-T). Kripto YMYL'ye yakın bir alan; Google
	// burada "bilgi nereden geliyor" sorusunu ayrıca tartıyor. Kaynak
	// adresi zaten elimizde ve sayfada görünür basılıyordu, yapısal
	// veriye hiç girmiyordu.
	//
	// İki alan birden yazılıyor, çünkü farklı şeyler söylüyorlar:
	//   isBasedOn → bu yazı O çalışmadan TÜRETİLDİ (yeniden yazım)
	//   citation  → bu yazı O çalışmaya ATIFTA BULUNUYOR
	// Toplayıcıda ikisi de doğru ve Google ikisini de okuyor.
	if kaynakURL := strings.TrimSpace(v.SourceURL); kaynakURL != "" {
		atif := map[string]any{"@type": "CreativeWork", "url": kaynakURL}
		if ad := strings.TrimSpace(v.SourceName); ad != "" {
			atif["name"] = ad
			atif["publisher"] = map[string]any{"@type": "Organization", "name": ad}
		}
		makale["isBasedOn"] = kaynakURL
		makale["citation"] = []any{atif}
	}
	if g.ID != "" {
		makale["identifier"] = g.ID
	}
	gorsel := g.Gorsel
	if gorsel == "" {
		gorsel = v.Image
	}
	if gorsel != "" {
		makale["thumbnailUrl"] = gorsel
		makale["image"] = map[string]any{"@type": "ImageObject", "url": gorsel}
	}
	if g.YayinISO != "" {
		makale["datePublished"] = g.YayinISO
		deg := g.GuncelISO
		if deg == "" {
			deg = g.YayinISO
		}
		makale["dateModified"] = deg
	}
	if len(g.Keywords) > 0 {
		makale["keywords"] = g.Keywords
	}

	// https: sitedeki diğer tüm bloklar zaten https kullanıyordu, yalnız
	// makale bloğu http kalmıştı.
	makale["@context"] = "https://schema.org"
	return jsonldScript(makale)
}

func makaleBreadcrumbJSONLD(v templkit.CDArticleView) template.HTML {
	return breadcrumbScript([]crumb{
		{Name: "Ana sayfa", URL: strings.TrimRight(v.BaseURL, "/") + "/"},
		{Name: v.SectionName, URL: strings.TrimRight(v.BaseURL, "/") + v.SectionHref},
		{Name: v.Headline, URL: v.Canonical},
	})
}

func breadcrumbScript(items []crumb) template.HTML {
	js := breadcrumbJSONLD(items)
	if js == "" {
		return ""
	}
	return template.HTML(`<script type="application/ld+json">` + string(js) + `</script>`)
}

// siteAdiJSONLD yardımcı: sadece adı olan kurum bloğu (liste sayfaları).
func siteAdiJSONLD(siteName, canonical string) template.HTML {
	return jsonldScript(map[string]any{
		"@context": "https://schema.org",
		"@type":    "WebSite",
		"name":     siteName,
		"url":      canonical,
	})
}

// makaleYayinISO içerik yayın zamanını ISO-8601'e çevirir (boş olabilir).
// `repo` bağımlılığı burada kullanılmıyor; imza tutarlılığı için alınıyor.
func makaleYayinISO(c repo.Content) string {
	if c.PublishedAt == nil {
		return ""
	}
	return c.PublishedAt.UTC().Format("2006-01-02T15:04:05Z")
}
