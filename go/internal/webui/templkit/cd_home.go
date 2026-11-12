package templkit

import "strings"

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
	// Gri true ise bu kartin gorseli siyah-beyaz basilir. Yazi
	// sayfasinda uygulanmaz: okuyucu habere girince gorsel renkli acilir.
	Gri bool
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

type CDDusunce struct {
	Baslik  string
	Sponsor string
	Image   string
	Href    string
}

type CDVideo struct {
	Etiket string
	Baslik string
	Image  string
	Href   string
	Sure   string
	// Srcset 16:9 kapak için genişlik adayları (tarayıcı seçer).
	Srcset string
	// ImageYedek kapak yüklenemezse denenecek adres ZİNCİRİ (virgüllü).
	ImageYedek string
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
	// ImageYedek dikey kart yedek ZİNCİRİ (oar2 → maxres → hq).
	ImageYedek string
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
	Arastirma []CDArastirma
	Klipler   []CDKlip
	// Canli sol kolonun EN ÜSTÜNDEKİ canlı yayın modülü (Son Haberler'in
	// üstünde). VideoID boşsa modül basılmaz.
	Canli    CDCanli
	MostRead []CDStory    // §21 — 1 büyük + 5 satır
	Opinion  []CDUzman    // §20 — uzman adı + unvan + avatar
	Prices   []CDPriceRow // §14
	Sections []CDSection
	// Hreflang gerçek çeviriler (dil → mutlak URL). Yalnız çevirisi
	// olanlar; boşsa x-default basılır.
	Hreflang map[string]string
	// JSONLD Go tarafında üretilmiş yapısal veri (script etiketiyle).
	// templ `<script>` içeriğini ham saydığı için burada taşınır.
	JSONLD string
	// CoinDesk SEO sınıflandırması (bkz. templkit.CDHead).
	PageCategory    string
	ContentCategory string
	Locale          string
	// BaseURL og:image mutlaklaştırması için kök adres.
	BaseURL string
	// Piyasa Nabzı (yalnız /markets): gerçek zincir-üstü veri.
	NabizFiyat *CDGrafikKart
	Nabiz      []CDGrafikKart
	// Analiz akışı: Piyasa Nabzı'nın üstünde akan üç kolon.
	// Boşsa modül hiç basılmaz.
	Analiz []CDAnalizKart
	// AnalizSayi rozetteki gerçek sayı (son 7 günün analiz adedi).
	AnalizSayi int
	// Explore "Explore more from CoinDesk" yatay şeridi (bölüm
	// panelleriyle). Boşsa modül basılmaz.
	Explore CDExplore
	// Giris: ana sayfanın GİRİŞ HERO'su (11 Kasım 2026) — şerit ile
	// üç kolon manşet arasında durur. Veri yoksa (Hazir=false) basılmaz.
	Giris    CDGiris
	Moduller []CDModul
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
	// Kilitli: içerik abonelik kademesi gerektiriyor ve ziyaretçinin
	// erişimi yok. Gövde kesilmiş olarak basılır, altına kilit kutusu
	// gelir.
	Kilitli bool
	// KilitKatman gereken katmanın adı ("Gümüş"). Boşsa genel ifade.
	KilitKatman string
	// Yorum makale altındaki yorum bölümü (üyelik zorunlu).
	Yorum CDYorumBolumu
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
	// BreadcrumbJSONLD ikinci yapısal veri bloğu (BreadcrumbList).
	BreadcrumbJSONLD string
	// ContentID tepki çubuğunun hedefi (yayınsa dolu). Metin olarak
	// tutuluyor: şablon sayı biçimlemez ve boşsa blok hiç basılmaz.
	ContentID string
	// FAQJSONLD gövdedeki gerçek "Sık sorulan sorular" bölümünden
	// çıkarılmış FAQPage bloğu. Bölüm yoksa boştur ve hiç basılmaz.
	FAQJSONLD string
	// Hreflang gerçek çeviriler (dil → mutlak URL).
	Hreflang      map[string]string
	Dil           CDMakaleDil
	CeviriEtiketi string
	// CeviriHref etiketin bağlandığı ORİJİNAL makale adresi.
	CeviriHref string
	// PageCategory/ContentCategory/Locale/ImageAlt/Metas: head'e geçen
	// CoinDesk SEO alanları (bkz. templkit.CDHead).
	PageCategory    string
	ContentCategory string
	Locale          string
	ImageAlt        string
	Metas           []CDMeta
	// ShareURL paylaşım bağlantıları için mutlak adres.
	ShareURL string
	// ShareX / ShareWhatsApp / ShareLinkedIn hazır paylaşım adresleri.
	ShareX        string
	ShareWhatsApp string
	ShareLinkedIn string
}

type CDMakaleDil struct {
	Yazan       string
	Yayinlandi  string
	Guncellendi string
	// Ceviri etiket metni ("AI tarafından çevrildi").
	Ceviri string
	// CeviriIpucu etiketin aria-label'i (ekran okuyucu için tam cümle).
	CeviriIpucu string
}

// CDMakaleDilTR Türkçe varsayılan iskelet metinleri.
func CDMakaleDilTR() CDMakaleDil {
	return CDMakaleDil{
		Yazan:       "Yazan",
		Yayinlandi:  "Yayınlandı",
		Guncellendi: "Güncellendi",
		Ceviri:      "AI tarafından çevrildi",
		CeviriIpucu: "Bu sayfa yapay zekâ ile Türkçeye çevrildi; orijinalini görüntüle",
	}
}

// CDMakaleDilEN İngilizce iskelet metinleri.
func CDMakaleDilEN() CDMakaleDil {
	return CDMakaleDil{
		Yazan:       "By",
		Yayinlandi:  "Published",
		Guncellendi: "Updated",
		Ceviri:      "Translated by AI",
		CeviriIpucu: "This page was translated into English by AI; view the original",
	}
}

// CDMakaleDilIcin dil koduna göre iskelet metinleri (bilinmeyen → TR).
func CDMakaleDilIcin(locale string) CDMakaleDil {
	switch strings.ToLower(strings.TrimSpace(locale)) {
	case "en":
		return CDMakaleDilEN()
	default:
		return CDMakaleDilTR()
	}
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
	// Alt öne çıkan 4'lü ızgara (239 px kartlar).
	Alt []CDStory
	// Sağ ray: görselsiz başlık + saat (6 kalem).
	Ray []CDStory
	// Side eski 3'lü yan sütun — geriye dönük alan, yeni şablonda
	// KULLANILMIYOR (CoinDesk kategori sayfasında böyle bir sütun yok).
	Side     []CDStory
	Stories  []CDStory
	MostRead []CDStory
	// MoreBaslik "More From {bölüm}" etiketi (CoinDesk deseni).
	MoreBaslik string
	Page       int
	PrevHref   string
	NextHref   string
	// JSONLD Go tarafında üretilmiş yapısal veri.
	JSONLD string
	// CoinDesk SEO sınıflandırması (bkz. templkit.CDHead).
	PageCategory    string
	ContentCategory string
	Locale          string
	// BaseURL og:image mutlaklaştırması için kök adres.
	BaseURL string
	// Piyasa Nabzı (yalnız /markets): gerçek zincir-üstü veri.
	NabizFiyat *CDGrafikKart
	Nabiz      []CDGrafikKart
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

type CDAnalizKart struct {
	Baslik   string
	Spot     string
	Kategori string
	Href     string
	Tarih    string
}
