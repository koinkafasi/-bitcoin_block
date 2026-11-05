// Package classify bot hattına düşen içeriğin kategorisini tahmin eder.
//
// Neden gerekli: 39.211 taslağın 30.021'i (%77) "Gündem"de toplanıyordu.
// Nedeni bir hata değil, tek sinyalin yokluğuydu — feed'in kendi kategorisi
// atanıyor, o da agregatörlerde (MSN, Google News, "Son Dakika" akışları)
// "Gündem" oluyordu. Sonuç: spor haberi de, borsa haberi de aynı kutuya
// düşüyor, kategori sayfaları ve manset seçimi anlamsızlaşıyordu.
//
// Yöntem üç kademeli, en güvenilirden en zayıfa:
//  1. Kaynağın kendi bölümü (RSS <category> — yayıncının kendi sınıflandırması)
//  2. Feed'in panelde editör tarafından atanmış kategorisi (Gündem dışıysa:
//     "GZT Spor" akışındaki içerik spordur, anahtar sözcük tartışması gereksiz)
//  3. Başlık+spot anahtar sözcükleri (yerel, deterministik, açıklanabilir)
//  4. Hiçbiri yoksa "gundem"
//
// Model tabanlı sınıflandırma ileride eklenebilir; bu paket anahtarsız çalışır
// ve her kararın gerekçesi tablodan okunabilir.
package classify

import (
	"sort"
	"strings"
)

// Doc sınıflandırılacak içeriktir.
type Doc struct {
	Title    string
	Spot     string
	Sections []string // RSS <category> değerleri
}

// Slug döndürür. feedSlug feed'in panelde atanmış kategorisidir (boş olabilir).
func Slug(doc Doc, feedSlug string) string {
	slug, _ := Classify(doc, feedSlug)
	return slug
}

// Classify slug ve kararın gerekçesini döndürür (panel/log için).
func Classify(doc Doc, feedSlug string) (slug, reason string) {
	if s := fromSections(doc.Sections); s != "" {
		return s, "kaynak bölümü"
	}
	if feedSlug != "" && feedSlug != "gundem" {
		return feedSlug, "feed kategorisi"
	}
	if s, kw := fromKeywordsHit(doc.Title + " " + doc.Spot); s != "" {
		return s, "anahtar sözcük: " + kw
	}
	return "gundem", "varsayılan"
}

// sectionMap yayıncıların bölüm adlarını bizim slug'lara eşler. Türkçe
// normalize edilmiş (küçük harf, noktalama yok) biçimde tutulur.
var sectionMap = map[string]string{
	"gundem": "gundem", "turkiye": "gundem", "son dakika": "gundem",
	"ekonomi": "ekonomi", "is dunyasi": "is-dunyasi", "finans": "finans",
	"piyasalar": "piyasalar", "borsa": "borsa", "doviz": "doviz",
	"altin": "altin", "emtia": "emtia", "faiz": "faiz", "enflasyon": "enflasyon",
	"kripto": "kripto-para", "kripto para": "kripto-para", "bitcoin": "kripto-para",
	"dunya": "dunya", "world": "dunya", "dis haberler": "dunya",
	"spor": "spor", "futbol": "futbol", "basketbol": "basketbol",
	"teknoloji": "teknoloji", "bilim": "bilim", "bilim teknoloji": "bilim",
	"oyun": "oyun", "oyunlar": "oyun", "espor": "e-spor", "e-spor": "e-spor",
	"yasam": "yasam", "hayat": "yasam", "saglik": "saglik", "saglik yasam": "saglik",
	"kultur": "egitim-kultur", "egitim": "egitim-kultur", "egitim kultur": "egitim-kultur",
	"sanat": "sanat", "sinema": "sinema", "muzik": "muzik", "kitap": "kitap",
	"edebiyat": "edebiyat", "tarih": "tarih", "mimari": "mimari",
	"turizm": "turizm", "gezi": "gezi-rotalari", "seyahat": "seyahat-rehberi",
	"otomotiv": "otomotiv", "enerji": "enerji", "emlak": "emlak",
	"tarim": "tarim", "sigorta": "finans", "bankacilik": "bankacilik",
	"asayis": "asayis", "polis adliye": "asayis", "3 sayfa": "asayis",
	"aktuel": "aktuel", "toplum": "toplum", "kadin": "kadin", "genclik": "genclik",
	"magazin": "medya-dijital-kultur", "medya": "medya-dijital-kultur",
	"yerel yonetim": "gundem", "sehir": "sehir-rehberi",
	"afrika": "afrika", "amerika": "amerika", "asya": "asya", "avrupa": "avrupa",
	"orta dogu": "ortadogu", "orta dogu haberleri": "ortadogu",
}

// keywordRules kategori → anahtar sözcükler. Sıra önemlidir: eşit sayıda
// eşleşmede önce gelen kazanır (daha özgül kategoriler üstte).
var keywordRules = []struct {
	slug  string
	words []string
}{
	{"borsa", []string{"borsa istanbul", "bist", "bist 100", "endeks", "hisse senedi", "hisse"}},
	{"doviz", []string{"dolar", "euro", "döviz", "kur", "sterlin", "parite", "forex"}},
	{"altin", []string{"altın", "gram altın", "çeyrek altın", "ons altın", "sarrafiye"}},
	{"kripto-para", []string{"bitcoin", "ethereum", "kripto", "kripto para", "btc", "blockchain", "stablecoin"}},
	{"enflasyon", []string{"enflasyon", "tüfe", "üfe", "fiyat artışı", "hayat pahalılığı"}},
	{"faiz", []string{"faiz", "politika faizi", "merkez bankası", "tcmb", "ppk"}},
	{"ekonomi", []string{"ekonomi", "ekonomik", "gsyh", "büyüme", "ihracat", "ithalat", "cari açık", "bütçe", "vergi", "zam", "asgari ücret", "işsizlik", "piyasa", "zam oranı"}},
	{"enerji", []string{"enerji", "doğal gaz", "elektrik", "petrol", "benzin", "motorin", "akaryakıt", "güneş enerjisi", "rüzgar enerjisi"}},
	{"otomotiv", []string{"otomobil", "araç", "otomotiv", "sıfır araç", "ikinci el araç", "tesla", "tofaş", "renault", "volkswagen", "kış lastiği"}},
	{"emlak", []string{"konut", "emlak", "kira", "konut satışı", "mortgage", "arsa", "daire fiyat"}},
	{"finans", []string{"bankacılık", "kredi", "mevduat", "sigorta", "yatırım", "halka arz"}},
	{"afet-kaza", []string{"deprem", "sel", "yangın", "kaza", "heyelan", "fırtına", "afet", "çöktü", "çökme", "enkaz", "yaralı", "çarpıştı", "devrildi", "patlama"}},
	{"futbol", []string{"futbol", "maç", "gol", "süper lig", "lig", "takım", "teknik direktör", "transfer", "hakem", "penaltı", "derbi", "uefa", "fifa", "milli takım", "sakatlık", "sakat"}},
	{"basketbol", []string{"basketbol", "nba", "euroleague", "potas"}},
	{"e-spor", []string{"e-spor", "espor", "valorant", "league of legends", "counter-strike"}},
	{"spor", []string{"spor", "sporcu", "antrenman", "atletizm", "güreş", "voleybol", "yüzme", "olimpiyat", "şampiyona", "kupası"}},
	{"saglik", []string{"sağlık", "hastane", "doktor", "hastalık", "virüs", "grip", "aşı", "kanser", "obezite", "reçete", "ilaç", "salgın"}},
	{"teknoloji", []string{"teknoloji", "yapay zekâ", "yapay zeka", "yazılım", "iphone", "android", "samsung", "google", "microsoft", "apple", "siber"}},
	{"bilim", []string{"bilim", "araştırma", "nasa", "uzay", "gezegen", "iklim", "deprem bilim", "arkeoloji"}},
	{"oyun", []string{"oyun", "playstation", "xbox", "nintendo", "steam", "konsol"}},
	{"sinema", []string{"film", "sinema", "dizi", "oscar", "festival", "gişe", "oyuncu"}},
	{"muzik", []string{"müzik", "konser", "albüm", "şarkı", "sanatçı"}},
	{"sanat", []string{"sergi", "resim", "heykel", "sanat", "müze"}},
	{"edebiyat", []string{"kitap", "roman", "şiir", "yazar", "edebiyat", "öykü"}},
	{"tarih", []string{"tarih", "osmanlı", "cumhuriyet tarihi", "antik", "kazı"}},
	{"mimari", []string{"mimari", "mimar", "kentsel dönüşüm", "inşaat", "istinat duvarı"}},
	{"egitim-kultur", []string{"eğitim", "okul", "üniversite", "öğrenci", "öğretmen", "sınav", "yks", "lgs", "meb", "yök", "burs"}},
	{"turizm", []string{"turizm", "tatil", "otel", "rezervasyon", "uçak bileti"}},
	{"gezi-rotalari", []string{"gezi", "rota", "seyahat", "keşif"}},
	{"yasam", []string{"yaşam", "günlük hayat", "beslenme", "moda", "dekorasyon", "ev", "aile", "çocuk", "ilişki", "psikoloji", "hobi", "yemek", "tarif"}},
	{"kadin", []string{"kadın", "kadın hakları", "annelik"}},
	{"genclik", []string{"gençlik", "genç", "staj"}},
	{"toplum", []string{"toplum", "sosyal", "dayanışma", "dernek", "vakıf"}},
	{"asayis", []string{"cinayet", "tutuklama", "gözaltı", "operasyon", "polis", "jandarma", "uyuşturucu", "kaçakçılık", "dolandırıcılık", "soruşturma", "dava", "ceset", "kayıp"}},
	{"tarim", []string{"tarım", "çiftçi", "buğday", "hayvancılık", "rekolte", "hasat"}},
	{"dunya", []string{"abd", "amerika", "avrupa", "avrupa birliği", "nato", "rusya", "ukrayna", "çin", "iran", "israil", "filistin", "gazze", "suriye", "almanya", "fransa", "ingiltere", "hindistan", "japonya", "dünya", "birleşmiş milletler", "trump", "putin", "bm"}},
	{"siyaset", []string{"cumhurbaşkanı", "meclis", "tbmm", "bakan", "parti", "seçim", "chp", "ak parti", "mhp", "hdp", "dem parti", "siyaset", "muhalefet"}},
}

// broadRules tek eşleşmeyle karar vermemesi gereken geniş kategorilerdir.
//
// Gerekçe somut: "Yeni spor salonu açıldı" cümlesinde tek başına "spor"
// kelimesi var diye haberi Spor'a atmak yanlış (o haber Yaşam'dır). Geniş
// kategorilerde en az iki bağımsız sinyal aranır; özgül kategoriler
// (futbol, borsa, altın, faiz…) tek sinyalle karar verebilir.
var broadRules = map[string]int{
	"spor": 2, "ekonomi": 2, "yasam": 2, "dunya": 2,
	"toplum": 2, "finans": 2, "mimari": 2, "sanat": 2, "edebiyat": 2, "e-spor": 2,
	"kadin": 2, "genclik": 2,
}

// fromSections kaynak bölümlerinden slug üretir.
func fromSections(sections []string) string {
	for _, s := range sections {
		if slug, ok := sectionMap[normalize(s)]; ok && slug != "gundem" {
			return slug
		}
	}
	for _, s := range sections {
		if slug, ok := sectionMap[normalize(s)]; ok {
			return slug
		}
	}
	return ""
}

// fromKeywordsHit en çok eşleşen kuralı döndürür; eşitlikte tablo sırası.
func fromKeywordsHit(text string) (slug, hit string) {
	norm := " " + normalize(text) + " "
	type scored struct {
		slug string
		hits []string
	}
	var best scored
	for _, rule := range keywordRules {
		var hits []string
		for _, w := range rule.words {
			if strings.Contains(norm, " "+normalize(w)+" ") {
				hits = append(hits, w)
			}
		}
		if len(hits) == 0 {
			continue
		}
		// Geniş kategorilerde tek sinyal yetmez (bkz. broadRules).
		if min := broadRules[rule.slug]; min > 0 && len(hits) < min {
			continue
		}
		// Eşitlikte tablo sırası korunur: yalnız daha çok eşleşme kazanır.
		if best.slug == "" || len(hits) > len(best.hits) {
			best = scored{slug: rule.slug, hits: hits}
		}
	}
	if best.slug == "" {
		return "", ""
	}
	if len(best.hits) > 2 {
		best.hits = best.hits[:2]
	}
	return best.slug, strings.Join(best.hits, ", ")
}

// normalize Türkçe'ye duyarlı küçük harfe indirir ve noktalama temizler.
//
// strings.ToLower tek başına yetmez: 'İ' → "i̇" (i + birleşik nokta) üretir ve
// "İSTANBUL" ile "istanbul" eşleşmez. Bu yüzden büyük harfler önce Türkçe
// kurallarına göre çevrilir.
func normalize(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case 'İ':
			b.WriteRune('i')
		case 'I':
			b.WriteRune('ı')
		default:
			b.WriteRune(r)
		}
	}
	low := strings.ToLower(b.String())
	var out strings.Builder
	space := true
	for _, r := range low {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == 'ç' || r == 'ğ' || r == 'ı' || r == 'ö' || r == 'ş' || r == 'ü' || r == 'â' || r == 'î' || r == 'û' {
			out.WriteRune(r)
			space = false
		} else if !space {
			out.WriteRune(' ')
			space = true
		}
	}
	return strings.TrimSpace(out.String())
}

// Slugs testlerde ve raporlarda kullanılmak üzere tüm kuralların slug listesi.
func Slugs() []string {
	seen := map[string]bool{}
	for _, r := range keywordRules {
		seen[r.slug] = true
	}
	for _, s := range sectionMap {
		seen[s] = true
	}
	out := make([]string, 0, len(seen))
	for s := range seen {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}
