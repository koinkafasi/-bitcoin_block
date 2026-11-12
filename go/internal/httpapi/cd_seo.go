package httpapi

import (
	"encoding/json"
	"html/template"
	"strings"

	"theblock/site/internal/webui/templkit"
)

// cdLDItem liste öğesi (bölüm/etiket/fiyat listeleri).
type cdLDItem struct {
	Headline  string // haber başlığı (fiyat listesinde varlık adı)
	URL       string
	Author    string
	Published string // ISO-8601
	Type      string // "NewsArticle" (varsayılan) | "FinancialProduct"
}

func cdCollectionJSONLD(siteName, base, name, canonical string, items []cdLDItem) template.HTML {
	liste := make([]any, 0, len(items))
	for i, it := range items {
		tip := it.Type
		if tip == "" {
			tip = "NewsArticle"
		}
		o := map[string]any{
			"@type": tip,
			"name":  it.Headline,
			"url":   it.URL,
		}
		if tip == "NewsArticle" {
			// CoinDesk anahtarı `headline`; FinancialProduct'ta `name`.
			delete(o, "name")
			o["headline"] = it.Headline
			o["inLanguage"] = "tr"
			if it.Author != "" {
				o["author"] = []any{map[string]any{"@type": "Person", "name": it.Author}}
			}
			if it.Published != "" {
				o["datePublished"] = it.Published
			}
		}
		liste = append(liste, map[string]any{
			"@type":    "ListItem",
			"position": i + 1,
			"item":     o,
		})
	}
	return jsonldScript(map[string]any{
		"@context":    "http://schema.org",
		"@type":       "CollectionPage",
		"name":        name,
		"url":         canonical,
		"description": cdKoleksiyonAciklama(name, canonical),
		"publisher": map[string]any{
			"@type": "NewsMediaOrganization",
			"name":  siteName,
			"logo":  "",
			"url":   base,
		},
		"mainEntity": map[string]any{
			"@type":           "ItemList",
			"itemListElement": liste,
		},
	})
}

// cdKoleksiyonAciklama CoinDesk deseni: bölüm adı açıklama alanına yazılır.
func cdKoleksiyonAciklama(name, canonical string) string {
	if name != "" {
		return name
	}
	return canonical
}

func cdCoinJSONLD(siteName, canonical, name, sym, price, change string) template.HTML {
	props := []any{
		map[string]any{"@type": "PropertyValue", "name": "Ticker Symbol", "value": sym},
		map[string]any{"@type": "PropertyValue", "name": "Price", "value": price, "unitText": "$"},
	}
	// Değişim yüzdesi: "+2,17" biçiminde metin geliyor; sayıya çevrilebiliyorsa
	// yazılır, çevrilemiyorsa alan atlanır.
	if v, ok := cdYuzdeSayi(change); ok {
		props = append([]any{
			map[string]any{"@type": "PropertyValue", "name": "Price Change (24 hours)", "value": v, "unitText": "%"},
		}, props...)
	}
	return jsonldScript(map[string]any{
		"@context":           "http://schema.org",
		"@type":              []any{"WebPage", "FinancialProduct"},
		"url":                canonical,
		"name":               name,
		"additionalProperty": props,
		"publisher": map[string]any{
			"@type": "NewsMediaOrganization",
			"name":  siteName,
			"logo":  "",
			"url":   canonical[:cdKokSonu(canonical)],
		},
	})
}

// cdKokSonu adresin şema+host kısmının bitiş indeksini verir.
func cdKokSonu(u string) int {
	if i := strings.Index(u, "://"); i >= 0 {
		if j := strings.Index(u[i+3:], "/"); j >= 0 {
			return i + 3 + j
		}
	}
	return len(u)
}

// cdFiyatListesiJSONLD /price ve /price/{coin} altındaki "diğer varlıklar"
// listesi (CoinDesk "Cryptocurrency Market Cap Rankings").
func cdFiyatListesiJSONLD(name string, items []cdLDItem) template.HTML {
	liste := make([]any, 0, len(items))
	for i, it := range items {
		liste = append(liste, map[string]any{
			"@type":    "ListItem",
			"position": i + 1,
			"item": map[string]any{
				"@type": "FinancialProduct",
				"name":  it.Headline,
				"url":   it.URL,
			},
		})
	}
	if len(liste) == 0 {
		return ""
	}
	return jsonldScript(map[string]any{
		"@context":        "https://schema.org",
		"@type":           "ItemList",
		"name":            name,
		"itemListElement": liste,
	})
}

func cdKisiJSONLD(siteName, canonical, name, rol, aciklama, gorsel string, sameAs []string) template.JS {
	kisi := map[string]any{
		"@type": "Person",
		"name":  name,
		"worksFor": map[string]any{
			"@type": "Organization",
			"name":  siteName,
		},
		"url": canonical,
	}
	if rol != "" {
		kisi["jobTitle"] = rol
	}
	if aciklama != "" {
		kisi["description"] = aciklama
	}
	if gorsel != "" {
		kisi["image"] = map[string]any{"@type": "ImageObject", "url": gorsel}
	}
	if len(sameAs) > 0 {
		kisi["sameAs"] = sameAs
	}
	blok := map[string]any{
		"@context": "http://schema.org",
		"@type":    "WebPage",
		"url":      canonical,
		"publisher": map[string]any{
			"@type": "NewsMediaOrganization",
			"name":  siteName,
			"logo":  "",
			"url":   canonical[:cdKokSonu(canonical)],
		},
		"mainEntity": kisi,
	}
	b, err := json.Marshal(blok)
	if err != nil {
		return ""
	}
	return template.JS(b)
}

// cdYuzdeSayi "+2,17" / "-0.20%" → 2.17 / -0.2. Ondalık virgül de kabul.
func cdYuzdeSayi(s string) (float64, bool) {
	s = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(s), "%"))
	s = strings.ReplaceAll(s, "+", "")
	s = strings.ReplaceAll(s, ",", ".")
	// Elle ayrıştırma: strconv + hata yolu tek satırda daha kısa olurdu ama
	// binlik ayırıcı (1.234,56) burada yok; güvenli olan tek biçim bu.
	var tam, kesir float64
	var kesirBas float64 = 1
	isaret := 1.0
	if strings.HasPrefix(s, "-") {
		isaret = -1
		s = s[1:]
	}
	nokta := false
	rakam := false
	for _, c := range s {
		switch {
		case c >= '0' && c <= '9':
			rakam = true
			if nokta {
				kesirBas *= 10
				kesir += float64(c-'0') / kesirBas
			} else {
				tam = tam*10 + float64(c-'0')
			}
		case c == '.':
			if nokta {
				return 0, false
			}
			nokta = true
		default:
			return 0, false
		}
	}
	if !rakam {
		return 0, false
	}
	return isaret * (tam + kesir), true
}

func cdMakaleMetas(siteName string, v makaleMetaGirdi) []templkit.CDMeta {
	var out []templkit.CDMeta
	ekle := func(name, content string) {
		if strings.TrimSpace(content) != "" {
			out = append(out, templkit.CDMeta{Name: name, Content: content})
		}
	}
	ekle("content_id", v.ID)
	ekle("content_title", v.Baslik)
	ekle("content_type", "News")
	ekle("content_vertical", v.BolumSlug)
	ekle("content_language", "tr")
	ekle("language_selection", "tr")
	ekle("publish_date", v.YayinTarih)
	ekle("publish_time", v.YayinSaat)
	ekle("display_date", v.YayinTarih)
	ekle("display_time", v.YayinSaat)
	ekle("create_date", v.YayinTarih)
	ekle("create_time", v.YayinSaat)
	ekle("reg_enabled", "false")
	ekle("sponsored", "false")
	ekle("featured_media_type", "basic")
	if v.Gorsel != "" {
		ekle("featured_media_key", "image")
		ekle("thumbnail", v.Gorsel)
	}
	if v.YazarSlug != "" {
		ekle("authors", v.YazarSlug)
		ekle("author_1", v.YazarSlug)
	}
	if len(v.Etiketler) > 0 {
		ekle("tags", strings.Join(v.Etiketler, ","))
		for i, e := range v.Etiketler {
			if i >= 5 {
				break
			}
			ekle("tag_"+itoaKP(i+1), e)
		}
	}
	// Parse.ly bloğu (CoinDesk birebir aynı adları kullanıyor).
	ekle("parsely-title", v.Baslik)
	ekle("parsely-link", v.Kanonik)
	ekle("parsely-type", "post")
	if v.Gorsel != "" {
		ekle("parsely-image-url", v.Gorsel)
	}
	ekle("parsely-pub-date", v.YayinISO)
	ekle("parsely-section", v.Bolum)
	ekle("parsely-author", v.Yazar)
	if len(v.Etiketler) > 0 {
		ekle("parsely-tags", strings.Join(v.Etiketler, ","))
	}
	ekle("parsely-post-id", v.ID)
	return out
}

// makaleMetaGirdi cdMakaleMetas girdisi.
type makaleMetaGirdi struct {
	ID         string
	Baslik     string
	Bolum      string
	BolumSlug  string
	Yazar      string
	YazarSlug  string
	Kanonik    string
	Gorsel     string
	YayinISO   string
	YayinTarih string // YYYYMMDD
	YayinSaat  string // HH:MM
	Etiketler  []string
}

// itoaKP küçük sayı → metin (strconv bağımlılığı olmadan).
func itoaKP(n int) string {
	if n <= 0 {
		return "0"
	}
	var b [8]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
