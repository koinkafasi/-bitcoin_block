package templkit

import (
	"strconv"
	"strings"
)

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
	Href   string // tıklama hedefi; boşsa Slug'tan /price/ yolu kurulur
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
	// Oturum durumu (üst çubuk): giriş YOKKEN "Oturum Aç" + "Kaydol",
	// VARKEN ikisi de kalkar yerine kullanıcı simgesi + hesap menüsü gelir
	// (kullanıcı isteği, 8 Kasım 2026: "giriş yapmışken neden hâlâ oturum
	// aç ve kaydol gösteriyor").
	//
	// cdShellCtx çerezden çözer; istek yoksa (r == nil) alanlar sıfır
	// kalır ve şablon bağlantıları basar — anonim davranış değişmez.
	LoggedIn    bool
	UserName    string
	UserInitial string
	UserHref    string // hesap merkezi ("/hesabim")
	UserUnread  int    // okunmamış bildirim (0 = rozet basılmaz)
	Locale      string
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

	// ---- MARKA (11 Kasım 2026) ----
	//
	// NEDEN: admin panelden logo yüklenebiliyordu ama cd kabuğu logoyu
	// SABİT `/static/img/logo-bitcosar-2x.png` olarak basıyordu — yüklenen
	// görsel hiçbir yerde görünmüyordu (kullanıcı bildirimi: "svg dosya ile
	// değiştirebilmek için admin panelden denedim").
	//
	// Artık: LogoURL doluysa O kullanılır; boşsa eski statik PNG/SVG
	// üçlemesine düşülür (geri uyumlu, dosya silinmişse kırık görsel yok).
	LogoURL string

	// LogoAcikURL açık tema için ayrı logo (markanın koyu renkli sürümü).
	// Boşsa LogoURL iki temada da kullanılır.
	LogoAcikURL string

	// LogoGenislik / LogoYukseklik yüklenen dosyanın GERÇEK ölçüsü.
	// CLS için width/height özniteliği gerekiyor; ölçü bilinmiyorsa
	// (SVG'de viewBox yok) varsayılan 146×32 korunur.
	LogoGenislik  int
	LogoYukseklik int

	// LogoHedefYukseklik panelden ayarlanan ÇİZİM yüksekliği (px).
	// 0 = 32 (varsayılan). `width/height` özniteliği değil CSS
	// yüksekliği: yatay logo doğal oranında uzasın.
	LogoHedefYukseklik int
	// LogoAzamiGenislik panelden ayarlanan en fazla genişlik (px).
	// 0 = 260.
	LogoAzamiGenislik int
	// SloganGirinti sloganın soldan girintisi (px). Logo dosyasında
	// simge solda, yazı sağda olduğu için sloganı yazının başladığı
	// harfin altına oturtmak panelden ayarlanıyor.
	SloganGirinti int

	// Tagline logonun altındaki slogan (marka ayarından).
	Tagline string

	// FaviconURL yüklenen favicon; boşsa statik simge kullanılır.
	FaviconURL string
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

// cdSiraya numarayı 1-tabanlı metne çevirir (0 → "1").
func cdSiraya(i int) string {
	return strconv.Itoa(i + 1)
}

// cdSayiMetni sayacı metne çevirir (hesap menüsü/kutuları).
func cdSayiMetni(n int) string {
	return strconv.Itoa(n)
}

// cdBildirimSayi bildirim ROZETİ metni — 9 üstü "9+" (eski header ile
// aynı sınır; rozet kutusu 3 haneli sayıya şişmesin).
func cdBildirimSayi(n int) string {
	if n > 9 {
		return "9+"
	}
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

// cdLogoOlcu logo width/height özniteliği için güvenli ölçü döndürür.
//
// CLS kuralı: <img> width/height taşımalı ki tarayıcı yer ayırsın.
// Yüklenen dosyanın ölçüsü bilinmiyorsa (SVG'de viewBox yok, ya da medya
// kaydında ölçü boş) VARSAYILANA düşülür — 0 yazmak düzeni bozar.
// templ yalnız string/attribute basabildiği için strconv kullanılıyor.
func cdLogoOlcu(deger, varsayilan int) string {
	if deger <= 0 || deger > 4000 {
		deger = varsayilan
	}
	return strconv.Itoa(deger)
}

// cdDolu metin GERÇEKTEN dolu mu (yalnız boşluk değil)?
//
// templ'de `if s != ""` yalnız boşluk taşıyan değeri de doğru sayıyor ve
// boş `<span>` basılıyordu (test yakaladı, 11 Kasım 2026). Slogan gibi
// kullanıcı girdisi alanlar için trim'li kontrol gerekiyor.
func cdDolu(s string) bool { return strings.TrimSpace(s) != "" }

// cdKirp baştaki/sondaki boşluğu atar (templ'den çağrılıyor).
func cdKirp(s string) string { return strings.TrimSpace(s) }

// cdLogoStil logo ölçüsünü CSS değişkenlerine çevirir.
//
// NEDEN DEĞİŞKEN: yükseklik hem masaüstünde hem mobilde aynı kalsın ama
// yatay logo doğal oranında uzasın. width/height ÖZNİTELİĞİ sabit oran
// dayatır; CSS `height` + `width:auto` doğru davranış.
//
// Değerler panelden geliyor ve sunucuda kırpılıyor (16..120 / 40..600);
// yine de burada son bir kez sınırlanıyor — şablon enjeksiyonu değil ama
// bozuk bir ayar satırı tüm üst çubuğu bozmasın.
func cdLogoStil(h, w, girinti int) string {
	if h <= 0 || h > 120 {
		h = 32
	}
	if w <= 0 || w > 600 {
		w = 260
	}
	if girinti < 0 || girinti > 600 {
		girinti = 0
	}
	return "--cd-logo-h:" + strconv.Itoa(h) + "px;--cd-logo-max:" + strconv.Itoa(w) +
		"px;--cd-logo-indent:" + strconv.Itoa(girinti) + "px"
}

// itoaOlcu form alanı için ölçü yazar; 0 = "boş" (varsayılan gösterilir).
//
// NEDEN: `0` yazmak kullanıcıyı "ölçü sıfır mı?" diye düşündürüyor ve
// kaydedince 0 gönderip varsayılana düşürüyordu. Boş bırakmak doğru:
// placeholder varsayılanı gösterir, sunucu boşu varsayılan sayar.
func itoaOlcu(n int) string {
	if n <= 0 {
		return ""
	}
	return strconv.Itoa(n)
}
