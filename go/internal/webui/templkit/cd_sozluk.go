package templkit

import "strings"

// ÖN YÜZ ÇEVİRİ SÖZLÜĞÜ — CoinDesk temasının arayüz metinleri.
//
// NEDEN BU DOSYA VAR (ölçüm, 13 Kasım 2026): dil seçici eklendi, `/en/`
// sayfaları 200 dönüyor ve `<html lang="en">` basıyordu ama İÇERİK Türkçe
// kalıyordu. Sebep: templkit'te i18n kullanımı SIFIRDI (grep: 0 eşleşme).
// DB'de 179 çeviri anahtarı vardı, şablonların hiçbiri onları okumuyordu.
// Yani çeviri altyapısı vardı, arayüz onu hiç sormuyordu.
//
// NEDEN ANAHTAR = TÜRKÇE METNİN KENDİSİ: yönetici panelinde ~1400 metin
// var; hepsine anahtar uydurmak bakım borcu üretir ve şablon okunmaz olur.
// Türkçe metin anahtar olunca:
//   · şablon okunur kalır — `cdT(s.Locale, "Oturum Aç")`
//   · çevirisi unutulan metin sessizce Türkçe kalır (panik yok, kırık yok)
//   · yeni metin eklemek tek satır: sözlüğe karşılığını yaz.
//
// YALNIZ ÖN YÜZ. Yönetim paneli (icerikform, gazete, sistem…) Türkçe kalır:
// editörler Türkçe çalışıyor ve o metinler okuyucuya görünmüyor.
//
// KAPSAM DIŞI (bilinçli): DB'den gelen içerik (haber başlığı, kategori adı,
// yazar adı) çevrilmez — onun çevirisi içerik hattının işi (translate). Bu
// sözlük YALNIZ arayüz kabuğunu çevirir.

// cdCeviriS ozluk dil kodu → (Türkçe metin → o dildeki karşılığı).
// İngilizce dışındaki diller için bugün yalnız Türkçe metin döner; yeni dil
// eklendiğinde buraya bir sözlük daha eklenir (yapı hazır).
var cdCeviriSozluk = map[string]map[string]string{
	"en": cdSozlukEN,
}

// cdT arayüz metnini istenen dile çevirir.
//
// Davranış: bilinmeyen metin → Türkçe (varsayılan) döner. Kısmi çeviri
// güvenlidir: eksik satır Türkçe görünür, sayfa bozulmaz.
func cdT(dil, tr string) string {
	dil = strings.TrimSpace(dil)
	if dil == "" || dil == "tr" || strings.HasPrefix(dil, "tr-") {
		return tr
	}
	if m, ok := cdCeviriSozluk[dil]; ok {
		if v, ok := m[tr]; ok {
			return v
		}
	}
	// Bölgesel kod (en-US) → ana dil (en).
	if i := strings.IndexByte(dil, '-'); i > 0 {
		if m, ok := cdCeviriSozluk[dil[:i]]; ok {
			if v, ok := m[tr]; ok {
				return v
			}
		}
	}
	return tr
}

// CDT arayüz metnini çevirir — dışa açık sarmalayıcı.
//
// httpapi paketi (kabuk verisi: gezinme etiketleri) aynı sözlüğü kullanır;
// böylece menü etiketi TEK yerden çevrilir, şablonda ikinci kez çevrilmez
// (çift çeviri = Türkçe metnin İngilizcesini tekrar aramak, zararsız ama
// gereksiz).
func CDT(dil, tr string) string { return cdT(dil, tr) }

// cdSozlukEN İngilizce arayüz karşılıkları.
//
// Kaynak: Türkçe metnin BİREBİR kendisi (bkz. üstteki gerekçe). Aynı metin
// birden çok yerde geçiyorsa tek satır yeter.
var cdSozlukEN = map[string]string{
	// ---- kabuk: üst çubuk, oturum, arama ----
	"Oturum Aç":          "Sign In",
	"Hesabım":            "My Account",
	"Çıkış":              "Sign Out",
	"Çıkış yap":          "Sign Out",
	"Üye ol":             "Join",
	"Giriş yap":          "Sign In",
	"Tümü":               "All",
	"Tümü →":             "All →",
	"Bölümler":           "Sections",
	"Kullanım Koşulları": "Terms of Use",
	"Çerez Politikası":   "Cookie Policy",
	"Reklam Politikası":  "Advertising Policy",
	"İletişim":           "Contact",
	"© 2026 bitcosar — Tüm hakları saklıdır.":               "© 2026 bitcosar — All rights reserved.",
	"Kaynaklar atıf ile gösterilir; tam metin yayınlanmaz.": "Sources are credited; full text is not republished.",
	"Arşiv dizini":             "Archive index",
	"Tüm arşiv":                "Full archive",
	"Tüm haberler dizini":      "All news index",
	"Piyasa şeridi":            "Market ticker",
	"Aramayı aç":               "Open search",
	"Aramayı kapat":            "Close search",
	"Arama türü":               "Search type",
	"Karanlık moda geç":        "Switch to dark mode",
	"Açık mod":                 "Light mode",
	"Oturum aç veya kaydol":    "Sign in or sign up",
	"Ana menü":                 "Main menu",
	"Menü":                     "Menu",
	"Piyasa şeridi (fiyatlar)": "Market ticker (prices)",
	"Kripto":                   "Crypto",
	"Hisse Senetleri":          "Stocks",
	"Hisse":                    "Stocks",

	// ---- ana sayfa modülleri ----
	"Düşünce Liderliği":     "Thought Leadership",
	"Basın Bültenleri":      "Press Releases",
	"Önceki bülten":         "Previous release",
	"Sonraki bülten":        "Next release",
	"Bugünün Videoları":     "Today's Videos",
	"Önceki video":          "Previous video",
	"Daha Fazlasını Keşfet": "Explore More",
	"Öne Çıkan Araştırma":   "Featured Research",
	"Önceki araştırma":      "Previous research",
	"Sonraki araştırma":     "Next research",
	"Bugünün Klipleri":      "Today's Clips",
	"Önceki klip":           "Previous clip",
	"En Çok Okunan":         "Most Read",
	"Görüş":                 "Opinion",
	"Günlük bülten":         "Daily newsletter",
	"Günün en önemli kripto haberleri, her sabah e-postanızda. Ücretsiz.":               "The day's top crypto news, in your inbox every morning. Free.",
	"Kayıt sonrası onay e-postası gönderilir. Adresiniz yalnız bülten için kullanılır;": "A confirmation email is sent after sign-up. Your address is used for the newsletter only;",
	"gizlilik politikası": "privacy policy",
	"Varlık":              "Asset",
	"Piyasa değeri":       "Market cap",
	"Tür":                 "Type",
	"Günlük":              "Daily",

	// ---- bölüm / liste sayfaları ----
	"← Önceki":                            "← Previous",
	"Bu bölümde henüz haber yok.":         "No stories in this section yet.",
	"Bu etikette henüz haber yok.":        "No stories with this tag yet.",
	"Tümünü gör →":                        "See all →",
	"Öne çıkanlar":                        "Highlights",
	"Gönder":                              "Send",
	"Arama yapmak için bir sözcük yazın.": "Type a word to search.",

	// ---- makale ----
	"Paylaş":                        "Share",
	"X'te paylaş":                   "Share on X",
	"WhatsApp'ta paylaş":            "Share on WhatsApp",
	"LinkedIn'de paylaş":            "Share on LinkedIn",
	"App'ta paylaş":                 "Share in app",
	"Özet":                          "Summary",
	"Göster":                        "Show",
	"Abonelere özel":                "Subscriber exclusive",
	"Yazının devamı abonelere açık": "The rest of this story is for subscribers",
	"Bu içeriği okumak için":        "To read this story you need",
	"ve üzeri bir aboneliğiniz olması gerekiyor.":              "or above.",
	"Bu içeriği okumak için aboneliğiniz olması gerekiyor.":    "A subscription is required to read this story.",
	"Abonelik seçeneklerine bak":                               "See subscription options",
	"Zaten aboneyim, giriş yap":                                "I'm already a subscriber, sign in",
	"Bu haber işine yaradı mı?":                                "Was this story useful?",
	"Evet, işime yaradı":                                       "Yes, it was",
	"Pek değil":                                                "Not really",
	"Teşekkürler — bu geri bildirim önerilerimizi düzeltiyor.": "Thanks — this feedback helps us improve.",
	"İlgili Haberler":                                          "Related Stories",
	"Özeti kapat":                                              "Close summary",
	"iletişim":                                                 "contact",

	// ---- fiyat / piyasa ----
	"Kripto Fiyatları":      "Crypto Prices",
	"Piyasa değerine göre":  "By market cap",
	"Fiyat grafiği":         "Price chart",
	"Zincir üstü metrikler": "On-chain metrics",
	"Diğer varlıklar":       "Other assets",
	"Piyasa Nabzı":          "Market Pulse",
	"Fiyat verisi şu an alınamıyor; birkaç dakika sonra yeniden deneyin.":                "Price data is unavailable right now; please try again in a few minutes.",
	"Bu hisse için grafik verisi şu an alınamıyor; birkaç dakika sonra yeniden deneyin.": "Chart data for this stock is unavailable right now; please try again in a few minutes.",
	"Zaman aralığı":                          "Time range",
	"Veri yok — planlı yenileme bekleniyor.": "No data — awaiting scheduled refresh.",
	"Tüm Hisseler":                           "All Stocks",
	"TÜMÜ":                                   "ALL",
	"Hisse adı":                              "Stock name",
	"Bu aramaya uygun hisse yok. Kodu (THYAO) veya şirket adını deneyin.":                  "No stock matches this search. Try the ticker (THYAO) or the company name.",
	"Hisse ara (THYAO veya Türk Hava Yolları)":                                             "Search stocks (THYAO or Turkish Airlines)",
	"BIST endeksi ve hisseleri ile önde gelen kripto paralar.":                             "The BIST index and stocks, plus leading cryptocurrencies.",
	"BIST'te işlem gören hisselerin kod, ad, fiyat ve günlük değişimi.":                    "Ticker, name, price and daily change for stocks traded on BIST.",
	"Bitcoin, Ethereum ve diğer kripto paraların TL fiyatı, 24 saat ve 7 günlük değişimi.": "TL prices for Bitcoin, Ethereum and other cryptocurrencies with 24-hour and 7-day change.",

	// ---- video ----
	"Videoyu YouTube'da aç": "Open video on YouTube",
	"Sıradaki":              "Up next",
	"Canlı Yayınlar":        "Live Streams",
	"Açıklama":              "Description",
	"Canlı Yayın":           "Live",
	"Ses düzeyi":            "Volume",
	"Oynatma hızı":          "Playback speed",
	"Video bölümleri":       "Video chapters",
	"Sıradaki videolar (tekerlekle kaydırılır)": "Up next (scroll with wheel)",

	// ---- yazar ----
	"Yazarlar ve Editörler":                                    "Authors and Editors",
	"Haberleri kaleme alan yazarlar, muhabirler ve editörler.": "The writers, reporters and editors behind our stories.",
	"Yazar listesi şu an alınamıyor.":                          "The author list is unavailable right now.",
	"Son yazılar":                                              "Latest stories",
	"Önceki":                                                   "Previous",

	// ---- bülten ----
	"Bültenler": "Newsletters",
	"Günün kripto haberleri ve piyasa özeti, her sabah e-postanızda.": "The day's crypto news and market recap, in your inbox every morning.",
	"Bülten arşivi":                 "Newsletter archive",
	"Gönderilen bültenlerin arşivi": "Archive of sent newsletters",
	"günlük özet arşivinde":         "in the daily digest archive",

	// ---- kurumsal / etkinlik / yorum ----
	"Genel iletişim":                      "General enquiries",
	"Haber ihbarı":                        "News tip",
	"Reklam & iş birliği":                 "Advertising & partnerships",
	"Düzeltme talebi":                     "Correction request",
	"Çalışma saatleri":                    "Office hours",
	"Adınız":                              "Your name",
	"Telefon (isteğe bağlı)":              "Phone (optional)",
	"Mesajınız":                           "Your message",
	"Yaklaşan etkinlikler ve takvim.":     "Upcoming events and calendar.",
	"Yaklaşan etkinlik yok.":              "No upcoming events.",
	"Yorumlar yayımlanmadan önce okunur.": "Comments are reviewed before publishing.",
	"Bu yazı hakkında ne düşünüyorsun?":   "What do you think about this story?",
	"Bu yoruma katılıyorum":               "I agree with this comment",

	// ---- 404 / akış ----
	"Aradığınız sayfa bulunamadı": "Page not found",
	"Bağlantı taşınmış, silinmiş ya da hiç var olmamış olabilir.\n\t\t\t\t\tAşağıdan arama yapabilir veya ana sayfaya dönebilirsiniz.": "The link may have moved, been deleted, or never existed.\n\t\t\t\t\tSearch below or return to the home page.",
	"Akış filtreleri": "Feed filters",

	// ---- giriş hero (piyasa) ----
	"24S HACİM":                           "24H VOLUME",
	"BTC DOMİNANS":                        "BTC DOMINANCE",
	"Korku & Açgözlülük Endeksi":          "Fear & Greed Index",
	"Dün:":                                "Yesterday:",
	"Geçen Hafta:":                        "Last week:",
	"Bitcoin RSI (14 gün)":                "Bitcoin RSI (14 days)",
	"Chrome Eklentisi":                    "Chrome Extension",
	"Kripto haberleri ve fiyat uyarıları": "Crypto news & price alerts",
	"Milyonerler Kulübü":                  "Millionaires Club",
	"Kripto sinyalleri · Üye girişi":      "Crypto signals · Member login",
	"Aşırı Satım: <30":                    "Oversold: <30",
	"Aşırı Alım: >70":                     "Overbought: >70",
	"Verinin sunucudaki yaşı":             "Age of the data on the server",
	"Bitcoin son 7 gün fiyat grafiği":     "Bitcoin 7-day price chart",

	// ---- abonelik ----
	"Ödeme": "Payment",
	"Ödeme tamamlandığında aboneliğiniz otomatik etkinleşir.":                                                   "Your subscription activates automatically once payment completes.",
	"Katmanınıza göre ayrıcalıklı haberlere, analizlere ve\n\t\t\t\t\t\t\t\treklamsız okuma deneyimine erişin.": "Get exclusive stories and analysis plus an ad-free reading experience, based on your tier.",
	"Aylık":                              "Monthly",
	"Yıllık":                             "Yearly",
	"Henüz tanımlı abonelik paketi yok.": "No subscription plan is defined yet.",
	"Mevcut planınız":                    "Your current plan",
	"Giriş yapıp abone ol":               "Sign in and subscribe",
	"Ödeme bilgileri":                    "Payment details",
	"Şehir":                              "City",
	"İndirim kodu (varsa)":               "Discount code (if any)",
	"Ön Bilgilendirme Formu":             "Preliminary Information Form",
	"Mesafeli Satış Sözleşmesi":          "Distance Sales Agreement",
	"KVKK Aydınlatma Metni":              "Data Protection Notice",
	"Gizlilik Politikası":                "Privacy Policy",
	"Ödemeye geç":                        "Continue to payment",
	"Sık sorulan sorular":                "Frequently asked questions",
	"Fatura dönemi":                      "Billing period",

	// ---- nav / bölüm adları ----
	"Son Haberler":  "Latest News",
	"Piyasalar":     "Markets",
	"Finans":        "Business",
	"Teknoloji":     "Tech",
	"Politika":      "Policy",
	"Kripto 101":    "Learn",
	"Fiyatlar":      "Prices",
	"Video":         "Video",
	"Haber Analizi": "News Analysis",
	"Odak":          "Focus",
	"Araştırma":     "Research",

	// ---- nav / menü ek etiketleri ----
	"Haberler":             "News",
	"Son Kripto Haberleri": "Latest Crypto News",
	"Videolar":             "Videos",
	"Podcastlar":           "Podcasts",
	"BIST hisse fiyatları": "BIST stock prices",
	"Tüm hisseleri gör":    "See all stocks",
	"Sponsorlu":            "Sponsored",
	"Focus":                "Focus",

	// ---- alt bilgi sütunları ----
	"Hakkımızda":        "About",
	"Kariyer":           "Careers",
	"Bize Ulaşın":       "Contact Us",
	"Gizlilik":          "Privacy",
	"Newsletters":       "Newsletters",
	"Kripto paralar":    "Cryptocurrencies",
	"Tüm fiyatlar":      "All prices",
	"Veri ve Endeksler": "Data & Indices",
	"Piyasa Verileri":   "Market Data",
	"Fikir Birliği":     "Consensus",
	"Etkinlikler":       "Events",
	"Araştırma Merkezi": "Research Hub",

	// ---- mobil / genel düğmeler ----
	"Ara":           "Search",
	"Dil":           "Language",
	"Tema":          "Theme",
	"Koyu mod":      "Dark mode",
	"Kaydol":        "Sign Up",
	"Kaydedilenler": "Saved",
	"Bildirimler":   "Notifications",
	"dk okuma":      "min read",
	// ---- ana sayfa hero (giriş) ----
	"Kripto dünyasından en güncel bilgiler": "The latest from the world of crypto",
	"Türkiye'nin En Güvenilir ":             "The Most Trusted ",
	"Dijital Varlık Bilgi":                  "Digital Asset Information",
	" Platformu":                            " Platform",
	"Bitcoin, Ethereum, altcoinler ve blockchain dünyasından anlık haberler, derinlemesine analizler ve güvenilir veriler.": "Breaking news, in-depth analysis and reliable data from the world of Bitcoin, Ethereum, altcoins and blockchain.",
	"Son Haberleri Keşfet": "Explore Latest News",
	"Piyasaları İncele":    "Explore Markets",
	"Anlık Piyasa Verisi":  "Live Market Data",
	"Derinlemesine Analiz": "In-Depth Analysis",
	"Doğrulanmış Kaynak":   "Verified Sources",

	// ---- video sayfası ek ----
	"Son Videolar": "Latest Videos",
	"Shorts":       "Shorts",
	// ---- kalan arayüz metinleri (13 Kasım 2026 taraması) ----
	"E-posta adresiniz": "Your email address",
	"Sonraki":           "Next",
	"Kişisel verilerimin mesajımın yanıtlanması amacıyla işlenmesine ilişkin aydınlatma metnini okudum.":                                      "I have read the data protection notice on the processing of my personal data for the purpose of responding to my message.",
	"Bağlantı taşınmış, silinmiş ya da hiç var olmamış olabilir. Aşağıdan arama yapabilir veya ana sayfaya dönebilirsiniz.":                   "The link may have moved, been deleted, or never existed. Search below or return to the home page.",
	"Canlı yayın oynatıcısı hazırlanıyor… Yayın yoksa YouTube çerçevesi bilgi verir.":                                                         "The live player is loading… If there is no stream, the YouTube frame will say so.",
	"Bu içerik, kaynağında yayımlanan haberin özetidir; tam metin kaynağa aittir ve yukarıda bağlantısı verilmiştir. Düzeltme talepleri için": "This story summarises a report published at its source; the full text belongs to the source, linked above. For correction requests contact",
	"Kaynak: CoinGecko — TRY. Bilgilendirme amaçlıdır, yatırım tavsiyesi değildir.":                                                           "Source: CoinGecko — TRY. For information only, not investment advice.",
	"Kaynaklar: BIST (Yahoo Finance) · kripto (CoinGecko) — TRY. Bilgilendirme amaçlıdır, yatırım tavsiyesi değildir.":                        "Sources: BIST (Yahoo Finance) · crypto (CoinGecko) — TRY. For information only, not investment advice.",
	"Kaynak: CryptoQuant · borsa akışları Binance · günlük veri · 10 dakika önbellek":                                                         "Source: CryptoQuant · Binance exchange flows · daily data · 10-minute cache",
	"Kaynak: CryptoQuant · BTC · Binance akışları · günlük veri · 10 dakika önbellek":                                                         "Source: CryptoQuant · BTC · Binance flows · daily data · 10-minute cache",
	// ---- mega menü açıklamaları ----
	"Araştırma raporları ve veri analizleri": "Research reports and data analysis",
	"Derinlemesine haber analizleri":         "In-depth news analysis",
	"Odak dosyaları ve uzun okumalar":        "Focus files and long reads",
	"Son Kripto Fiyatları":                   "Latest Crypto Prices",
	"Tüm kripto fiyatlarını gör":             "See all crypto prices",
	"Kripto Haberleri":                       "Crypto News",
	"haberleri":                              "news",
	"daha fazlası":                           "more from us",
	"Son haberler ve analizler.":             "Latest stories and analysis.",
	"3000+ kripto varlığın canlı fiyatı, piyasa değeri ve 24 saatlik değişimi.": "Live prices, market cap and 24-hour change for 3,000+ crypto assets.",
	"Sonraki klip": "Next clip",
	// ---- göreli zaman (sayı ayrı basılır, birim çevrilir) ----
	"az önce":      "just now",
	" dakika önce": " minutes ago",
	" saat önce":   " hours ago",
	" gün önce":    " days ago",
	// ---- sayfa başlıkları (title / h1) ----
	"Sayfa bulunamadı":                        "Page not found",
	"Piyasa & Borsa":                          "Markets & Stocks",
	"BIST ve kripto fiyatları":                "BIST and crypto prices",
	"BIST hisse fiyatları ve arama":           "BIST stock prices and search",
	"Kripto Para":                             "Crypto",
	"canlı fiyatlar, 24s/7g değişim ve hacim": "live prices, 24h/7d change and volume",
	"Basın bülteni":                           "Press release",
	"Klip":                                    "Clip",
	"Podcast":                                 "Podcast",
	"Sponsorlu İçerik":                        "Sponsored Content",
	"Kripto ve blockchain üzerine sohbetler, analizler ve röportajlar.":                "Conversations, analysis and interviews on crypto and blockchain.",
	"İş ortaklarımızın hazırladığı içerikler. Sponsorlu içerikler açıkça etiketlenir.": "Content produced by our partners. Sponsored content is clearly labelled.",
	"Şu an yayında sponsorlu içerik yok.":                                              "No sponsored content is live right now.",
	"Henüz podcast bölümü yayınlanmadı. Bölümler eklendikçe burada listelenir.":        "No podcast episodes yet. Episodes appear here as they are added.",
	// ---- sayfa başlık/açıklama (title, meta description) ----
	"Kripto Haberleri ve Piyasa Verisi": "Crypto News and Market Data",
	"Bitcoin, Ethereum ve kripto piyasalarından son haberler, analizler ve canlı fiyatlar.": "The latest Bitcoin, Ethereum and crypto market news, analysis and live prices.",
	"Haberleri": "News",
	"Fiyatı":    "Price",
	"Canlı fiyat, 24 saatlik değişim ve piyasa sıralaması.": "Live price, 24-hour change and market ranking.",
	"Hisse Fiyatı ve Grafiği":                               "Stock Price and Chart",
	"bölümünden daha fazlası":                               "more from this section",
	// ---- konum izni (14 Kasım 2026) ----
	"Konumunuzu paylaşın": "Share your location",
	"Bulunduğunuz bölgeye göre yerel piyasa verisi ve içerik gösterebilmemiz için konumunuza ihtiyacımız var.": "We need your location to show local market data and content for your region.",
	"Konum yalnız bölge istatistiği için kullanılır.":                                                          "Location is used for regional statistics only.",
	"Yaklaşık 11 metreye yuvarlanır; adresiniz kaydedilmez.":                                                   "Rounded to about 11 metres; your address is not stored.",
	"Üçüncü taraflarla paylaşılmaz.":                                                                           "Not shared with third parties.",
	"Şimdi değil":                                                                                              "Not now",
	"Konumu etkinleştir":                                                                                       "Enable location",
	"İzni tarayıcı ayarlarından her zaman geri alabilirsiniz.":                                                 "You can revoke permission in your browser settings at any time.",
	"Konum alınamadı. Tarayıcı izinlerini kontrol edin.":                                                       "Could not get your location. Check your browser permissions.",
	"Konum izni gerekli":                                                                                       "Location permission required",
	"Üyelik panelini kullanmak için konum izni gerekiyor.":                                                     "Location permission is required to use the member panel.",
	"Üyelik paneli için konum izni":                                                                            "Location permission for the member panel",
	"Hesabınızı güvende tutmak ve size bulunduğunuz bölgeye uygun içerik göstermek için konumunuza ihtiyacımız var. İzin vermeden panel ve kayıt işlemleri açılmıyor.": "We need your location to keep your account secure and to show content for your region. The panel and registration stay locked without it.",
	"Konum yaklaşık 11 metreye yuvarlanır, adresiniz kaydedilmez ve üçüncü taraflarla paylaşılmaz. İzni tarayıcı ayarlarından geri alabilirsiniz.":                     "Location is rounded to about 11 metres, your address is not stored and it is not shared with third parties. You can revoke permission in your browser settings.",
	"Ana sayfaya dön": "Back to home",
}
