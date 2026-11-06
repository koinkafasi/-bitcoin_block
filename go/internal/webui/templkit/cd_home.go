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

// CDDusunce "Düşünce Liderliği" modülü (sol ray).
//
// CoinDesk'te sponsorlu içerik: görsel + başlık + "Sponsored by …".
// Kaynak gerçek CoinDesk sponsorlu içeriği; kendi sponsorlu arşivimiz
// dolduğunda aynı bileşen oradan beslenecek.
type CDDusunce struct {
	Baslik  string
	Sponsor string
	Image   string
	Href    string
}

// CDVideo "Bugünün Videoları" modülü (sağ ray).
//
// Oynatıcı yerine kapak + başlık: video altyapısı gelene kadar kart
// kaynağa gider (kullanıcı isteği: geçici olarak CoinDesk video linki).
type CDVideo struct {
	Etiket string
	Baslik string
	Image  string
	Href   string
	Sure   string
}

// CDArastirma "Öne Çıkan Araştırma" metrik kartı (sağ ray).
type CDArastirma struct {
	Metrik    string
	Baslik    string
	Aciklama  string
	Rapor     string
	Etiketler []string
	Href      string
}

// CDKlip "Bugünün Klipleri" dikey (9/16) video kartı.
type CDKlip struct {
	Baslik string
	Image  string
	Href   string
}

// CDBasin "Basın Bültenleri" slider slaytı.
type CDBasin struct {
	Baslik  string
	Sponsor string
	Href    string
}

// CDUzman "Görüş" satırındaki uzman/editoryal profil.
type CDUzman struct {
	Ad     string
	Unvan  string
	Avatar string
	Makale string
	Href   string
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
	Lead CDStory
	// §7A sol ray: "Son Haberler" + Düşünce Liderliği + tek görselli manşet.
	RailLeft  []CDStory
	Dusunce   *CDDusunce
	TekGorsel CDStory
	// §7A orta kolon: manşet + 8 görselli haber + Basın Bültenleri slider.
	Akis  []CDStory
	Basin []CDBasin
	// §7A sağ ray: video · araştırma · klipler (1284 altında gizli).
	Videolar  []CDVideo
	Arastirma *CDArastirma
	Klipler   []CDKlip
	MostRead  []CDStory    // §21 — 1 büyük + 5 satır
	Opinion   []CDUzman    // §20 — uzman adı + unvan + avatar
	Prices    []CDPriceRow // §14
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

// cdBasinGeri / cdBasinIleri JS'siz slider'ın halka gezinmesi.
// i 0-tabanlı; dönen değer de 0-tabanlı (şablon cdSiraya ile 1'e çevirir).
func cdBasinGeri(i, n int) int {
	if n <= 0 {
		return 0
	}
	return (i - 1 + n) % n
}

func cdBasinIleri(i, n int) int {
	if n <= 0 {
		return 0
	}
	return (i + 1) % n
}
