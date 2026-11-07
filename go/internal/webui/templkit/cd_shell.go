package templkit

import "strconv"

// CoinDesk tasarım kabuğu (bitcosar).
// Kaynak: docs/COINDESK-SPEC.md §3 (üst gezinme + fiyat şeridi) ve §6
// (ana sayfa modül sırası). Değerler ölçüldü: nav sırası, şerit içeriği,
// alt bilgi sütunları CoinDesk'in kendi HTML'inden alındı.
//
// TASARIM İLKESİ: gölge yok, ayırma çizgiyle; köşe 2 px; JS yalnız
// etkileşim için (şerit sekmesi). Makale sayfası JS'siz okunur.

// CDTickerItem fiyat şeridindeki tek kalem.
type CDTickerItem struct {
	Sym    string
	Slug   string // kanonik yol parçası (tam ad): "bitcoin"
	Price  string
	Change string // "+%3.48" gibi, işaret dahil
	Up     bool
}

// CDNavLink üst gezinmedeki tek bağlantı.
type CDNavLink struct {
	Label  string
	Href   string
	Active bool
}

// CDDil dil seçicideki tek dil.
type CDDil struct {
	Kod   string
	Ad    string
	Aktif bool
	Href  string
}

// CDShell, CoinDesk kabuğunun sayfa verisidir.
type CDShell struct {
	SiteName   string
	LangBase   string
	Nav        []CDNavLink
	Ticker     []CDTickerItem
	TickerTab  string // "markets" | "stocks"
	TickerLive string // "Markets Live" / "Crypto Stocks"
	SearchHref string
	LoginHref  string
	SignupHref string
	Locale     string
	// Diller dil seçicisi (yalnız gerçek diller).
	Diller []CDDil
	// MenuTags mega menüdeki etiket sütunu (yalnız içeriği olan etiketler).
	MenuTags []CDMenuItem
	// MenuCoins mega menüdeki canlı fiyat satırları (ilk 5 coin).
	MenuCoins []CDMenuItem
	// MenuBolumler YAYINDA İÇERİĞİ OLAN bölüm slug'ları.
	//
	// NEDEN: menü sabit tanımlı, ama Focus/Videos/Podcasts/News Analysis/
	// Press Releases bölümlerinde yayında içerik yok (ölçüm 6 Kasım 2026:
	// 0). Boş bölüme bağlantı vermek kullanıcıyı "içerik yok" sayfasına
	// götürüyordu — menü sitenin tamamını vaat edip yarısını vermiyor.
	// Bu harita doluysa yalnız içeriği olan bölümler basılır; boşsa (panel
	// sayfaları, testler) filtre uygulanmaz ve menü eskisi gibi çalışır.
	MenuBolumler map[string]bool
	// Footer sütunları ölçülen yapıya göre sabittir (aşağıda).
	// BaseURL og:image mutlaklaştırması ve yapısal veri için kök adres
	// (sayfa görünümlerinden doldurulur; boşsa göreli kalır).
	BaseURL string
}

// cdFooterColumns CoinDesk alt bilgi sütunları (ölçüm §3).
//
// YAPI (CoinDesk /tr, 7 Kasım 2026): DÖRT sütun (her biri 237 px, gap 24),
// her sütunda 2-3 BÖLÜM başlığı ÜST ÜSTE. Başlık 14/22 w500 `capitalize`,
// bağlantı 12/16 w400 sönük, bölümler arası 38 px.
//
// BAĞLANTILAR DOĞRULANDI (curl, 7 Kasım 2026): her yol 200 döndü. Ölü
// olanlar (blog · erisilebilirlik · reklam · site-haritasi · index · api ·
// web-seminerleri) LİSTEYE ALINMADI — alt bilgi 404 vermemeli.
//
// SÜZGEÇ KALDIRILDI: önceki sürüm `MenuBolumler` ile satır eliyordu, ama o
// harita yalnız İÇERİK bölümlerini sayıyor; /hakkimizda gibi statik
// sayfalar haritada olmadığı için alt bilgi 942 px yerine 497 px'e
// düşüyordu ve sütunlar seyreliyordu. Liste artık elle doğrulanmış.
var cdFooterColumns = []cdFooterBolum{
	{Title: "Hakkımızda", Links: [][2]string{
		{"Hakkımızda", "/hakkimizda"},
		{"Kariyer", "/kariyer"},
	}},
	{Title: "İletişim", Links: [][2]string{
		{"Bize Ulaşın", "/iletisim"},
		{"Gizlilik", "/gizlilik-politikasi"},
		{"Kullanım Koşulları", "/kullanim-kosullari"},
	}},
	{Title: "Newsletters", Links: [][2]string{
		{"Bültenler", "/newsletters"},
	}},
	{Title: "Haberler", Links: [][2]string{
		{"Son Haberler", "/latest-crypto-news"},
		{"Piyasalar", "/markets"},
		{"Finans", "/business"},
		{"Teknoloji", "/tech"},
		{"Politika", "/policy"},
		{"Haber Analizi", "/news-analysis"},
	}},
	{Title: "Videolar", Links: [][2]string{
		{"Video", "/videos"},
	}},
	{Title: "Podcasts", Links: [][2]string{
		{"Podcastlar", "/podcasts"},
	}},
	{Title: "Kripto paralar", Links: [][2]string{
		{"Tüm fiyatlar", "/price"},
		{"Bitcoin (BTC)", "/price/bitcoin"},
		{"Ethereum (ETH)", "/price/ethereum"},
		{"XRP", "/price/xrp"},
		{"Solana (SOL)", "/price/solana"},
	}},
	{Title: "Veri ve Endeksler", Links: [][2]string{
		{"Piyasa Verileri", "/price"},
	}},
	{Title: "Fikir Birliği", Links: [][2]string{
		{"Etkinlikler", "/etkinlikler"},
	}},
	{Title: "Sponsorlu", Links: [][2]string{
		{"Düşünce Liderliği", "/sponsored-content"},
		{"Basın Bültenleri", "/press-release"},
	}},
	{Title: "Araştırma", Links: [][2]string{
		{"Araştırma Merkezi", "/research"},
		{"Görüş", "/opinion"},
		{"Odak", "/focus"},
	}},
}

// cdFooterSutunlari bölümleri DÖRT sütuna sırayla dağıtır (CoinDesk
// `grid-cols-4`). Sıra korunur; sütun yükseklikleri dengeli kalır.
func cdFooterSutunlari(s CDShell) [][]cdFooterBolum {
	const sutunSayisi = 4
	out := make([][]cdFooterBolum, sutunSayisi)
	for i, b := range cdFooterColumns {
		out[i%sutunSayisi] = append(out[i%sutunSayisi], b)
	}
	return out
}

// cdFooterBolum alt bilgide tek başlık + bağlantıları.
type cdFooterBolum struct {
	Title string
	Links [][2]string
}

// cdFooterSutunlari alt bilgi sütunlarını döndürür; içeriği olmayan bölüm
// satırları elenir.
//
// NEDEN: alt bilgi de bir gezinmedir. Ölçüm (6 Kasım 2026): Focus · Video ·
// Podcast · Analiz · Basın Bültenleri bölümlerinde yayında içerik yoktu,
// ama alt bilgi beş ölü bağlantı veriyordu. Kural mega menüyle aynı:
// CDSekme etiketi: "markets" → "Markets Live".
func cdSekmeEtiketi(tab string) string {
	if tab == "stocks" {
		return "Hisse Senetleri"
	}
	return "Canlı Piyasalar"
}

// cdSiraya numarayı 1-tabanlı metne çevirir (0 → "1").
func cdSiraya(i int) string {
	return strconv.Itoa(i + 1)
}

// cdSayiMetni sayacı metne çevirir (hesap menüsü/kutuları).
func cdSayiMetni(n int) string {
	return strconv.Itoa(n)
}

// cdOkumaMetni okuma süresi metni.
func cdOkumaMetni(dk int) string {
	if dk <= 0 {
		dk = 1
	}
	return strconv.Itoa(dk) + " dk okuma"
}

// cdSlug sembolü fiyat yolu parçasına çevirir (BTC → btc).
//
// DİKKAT: kanonik fiyat adresi artık TAM ADDIR (`/price/bitcoin`); bu
// yardımcı yalnız sembol karşılaştırması ve 301 yönlendirmesi için kalır.
func cdSlug(sym string) string {
	out := make([]rune, 0, len(sym))
	for _, r := range sym {
		switch {
		case r >= 'A' && r <= 'Z':
			out = append(out, r+('a'-'A'))
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			out = append(out, r)
		}
	}
	return string(out)
}
