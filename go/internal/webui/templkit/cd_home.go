package templkit

// CoinDesk ana sayfa — veri tipleri.
//
// Modül sırası docs/COINDESK-SPEC.md §6'dan BİREBİR alınmıştır; sıra
// değiştirilmez. Boş modül basılmaz (boş kutu tasarımı bozuk gösterir),
// ama sıra korunur.

// CDStory kart/liste öğesi.
type CDStory struct {
	Title     string
	Spot      string
	Href      string
	Image     string
	Time      string
	Source    string
	SourceURL string
	Kicker    string
	Tags      []string
}

// CDSection ana sayfadaki bölüm bloğu (Markets / Finance / Tech / Policy).
// CoinDesk düzeni: 1 büyük görselli haber + 2 alt başlık + 3 satır.
type CDSection struct {
	Title  string
	Href   string
	Lead   CDStory
	Second []CDStory
	Rows   []CDStory
}

// CDPriceRow fiyat tablosu satırı (CoinDesk "Prices" modülü).
type CDPriceRow struct {
	Rank   int
	Sym    string
	Name   string
	Price  string
	Change string
	Up     bool
	Href   string
}

// CDHomeView ana sayfa şablon verisi.
type CDHomeView struct {
	Shell     CDShell
	Title     string
	Desc      string
	Canonical string

	// §6 sırası:
	//  3 promo banner · 4 today's videos · 5 latest news · 6 thought leadership
	//  7 newsletters · 8 manşet bloğu · 9 press releases · 10 research
	//  12 clips · 13 reklam · 14 prices · 15 stocks · 16 exchange benchmark
	//  17 videos · 18 indices · 19 podcasts · 20 opinion · 21 most read
	//  22 bölüm blokları
	Promo    string
	Lead     CDStory
	Side     []CDStory
	Latest   []CDStory    // §5 — saat damgalı düz liste
	MostRead []CDStory    // §21 — 1 büyük + 5 satır
	Opinion  []CDStory    // §20 — yazar avatarlı
	Prices   []CDPriceRow // §14
	Sections []CDSection
	// Hreflang gerçek çeviriler (dil → mutlak URL). Yalnız çevirisi
	// olanlar; boşsa x-default basılır.
	Hreflang map[string]string
	// JSONLD Go tarafında üretilmiş yapısal veri (script etiketiyle).
	// templ `<script>` içeriğini ham saydığı için burada taşınır.
	JSONLD string
}

// CDArticleView makale sayfası (§7).
type CDArticleView struct {
	Shell       CDShell
	Title       string
	Desc        string
	Canonical   string
	Kicker      string
	SectionName string
	SectionHref string
	Headline    string
	Spot        string
	Author      string
	PublishedAt string
	ReadMinutes int
	BodyHTML    string // güvenilir, sunucuda üretilmiş
	Image       string
	ImageCredit string
	SourceName  string
	SourceURL   string
	Tags        []CDTag
	Related     []CDStory
	PrevHref    string
	PrevTitle   string
	NextHref    string
	NextTitle   string
	// BaseURL yapısal veri (breadcrumb) için mutlak kök.
	BaseURL string
	// JSONLD Go tarafında üretilmiş NewsArticle + BreadcrumbList.
	JSONLD string
}

// CDTag makale etiketi.
type CDTag struct {
	Ad   string
	Slug string
}

// CDSectionPage bölüm sayfası (§8).
type CDSectionPage struct {
	Shell     CDShell
	Title     string
	Desc      string
	Canonical string
	Section   string
	Href      string
	Lead      CDStory
	Side      []CDStory
	Stories   []CDStory
	MostRead  []CDStory
	Page      int
	PrevHref  string
	NextHref  string
	// JSONLD Go tarafında üretilmiş yapısal veri.
	JSONLD string
}
