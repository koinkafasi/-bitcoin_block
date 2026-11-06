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
	// Paylasim/sosyal: makale altindaki dugmeler icin.
	ShareURL string
}

// CDSection ana sayfadaki bölüm hücresi (Markets / Policy / Tech / Finance).
// CoinDesk düzeni (ölçüldü): 1 görselli haber + 3 saat damgalı satır.
type CDSection struct {
	Title string
	Href  string
	Lead  CDStory
	Rows  []CDStory
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

// CDRailModul hero yan rayındaki tek modül (bölüm kartı).
//
// CoinDesk sağ rayı üç modül yığar (Bugünün Videoları · Öne Çıkan
// Araştırma · Bugünün Klipleri). Bizde o içerik türleri yok; aynı
// yapıyı gerçek veriyle dolduruyoruz: her bölümden bir kart.
type CDRailModul struct {
	Title string
	Href  string
	Story CDStory
}

// CDHomeView ana sayfa şablon verisi.
type CDHomeView struct {
	Shell     CDShell
	Title     string
	Desc      string
	Canonical string

	// §6 sırası:
	//  4 today's videos · 5 latest news · 6 thought leadership
	//  8 manşet bloğu · 9 press releases · 10 research
	//  12 clips · 13 reklam · 14 prices · 15 stocks · 16 exchange benchmark
	//  17 videos · 18 indices · 19 podcasts · 20 opinion · 21 most read
	//  22 bölüm blokları
	//
	// §3 promo bandı ve §7 bülten formu ana sayfada YOK: CoinDesk TR ana
	// sayfasında ne promo şeridi ne de form var (6 Kasım 2026 ölçümü).
	// Bülten bileşeni /bulten sayfasında duruyor.
	Lead      CDStory
	Side      []CDStory
	Latest    []CDStory     // §5 — saat damgalı düz liste
	RailLeft  []CDStory     // §7A sol ray — "Son Haberler" (başlık + saat)
	RailRight []CDRailModul // §7A sağ ray — bölüm modülleri
	MostRead  []CDStory     // §21 — 1 büyük + 5 satır
	Opinion   []CDStory     // §20 — yazar avatarlı
	Prices    []CDPriceRow  // §14
	Sections  []CDSection
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
	// AuthorURL yazar sayfası (varsa). CoinDesk yazar adını ALT ÇİZİLİ
	// bağlantı yapıyor — sitede alt çizginin nadir istisnası (spec §10.2).
	AuthorURL   string
	PublishedAt string
	// PublishedISO <time datetime> için ISO-8601 (makine okur).
	PublishedISO string
	// UpdatedAt yalnız gerçekten sonradan düzenlendiyse dolar; boşsa
	// "Güncellendi" satırı BASILMAZ (uydurma güncelleme tarihi yanıltır).
	UpdatedAt   string
	UpdatedISO  string
	ReadMinutes int
	// Summary AI özeti (ai_summary). Boşsa özet kutusu hiç basılmaz.
	Summary string
	// SummaryIlkSatir kapalıyken görünen tek satır önizleme.
	SummaryIlkSatir string
	// SummaryMaddeler açıldığında listelenen maddeler (cümlelere bölünmüş).
	SummaryMaddeler []string
	BodyHTML        string // güvenilir, sunucuda üretilmiş
	Image           string
	ImageCredit     string
	SourceName      string
	SourceURL       string
	Tags            []CDTag
	Related         []CDStory
	PrevHref        string
	PrevTitle       string
	NextHref        string
	NextTitle       string
	// BaseURL yapısal veri (breadcrumb) için mutlak kök.
	BaseURL string
	// JSONLD Go tarafında üretilmiş NewsArticle + BreadcrumbList.
	JSONLD string
	// ShareURL paylaşım bağlantıları için mutlak adres.
	ShareURL string
	// ShareX / ShareWhatsApp / ShareLinkedIn hazır paylaşım adresleri.
	ShareX        string
	ShareWhatsApp string
	ShareLinkedIn string
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
