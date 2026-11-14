package templkit

import "sync/atomic"

// CDEntegrasyon — üçüncü taraf entegrasyonların DOĞRULANMIŞ kimlikleri.
//
// NEDEN HAM HTML ALANI YOK: yönetim panelinden serbest `<script>` yapıştırmak
// kalıcı bir XSS kapısıdır. Panel erişimi olan (ya da oturumu çalınan) biri
// siteye HER sayfada çalışan kod enjekte edebilirdi; CSP'nin `script-src 'self'`
// koruması da anlamsız kalırdı çünkü kod zaten kendi kaynağımızdan çıkardı.
//
// ÇÖZÜM: her entegrasyon için TEK bir kimlik alanı tutulur ve httpapi
// tarafında regex ile doğrulanır (G-XXXX, GTM-XXXX, ca-pub-…). `<script>`
// etiketini BİZ üretiriz, kimlik yalnız içine yazılır. Doğrulamayı geçmeyen
// değer boş kabul edilir ve hiçbir şey basılmaz.
//
// Değerler atomik tutulur: sayfa istekleri ile ayar kaydı yarışır, normal
// mutex her isteği serileştirirdi.
type CDEntegrasyon struct {
	// GSCVerification google-site-verification içeriği (yalnız token).
	GSCVerification string
	// GA4ID GA4 ölçüm kimliği: G-XXXXXXXXXX.
	GA4ID string
	// GTMID Google Tag Manager kimliği: GTM-XXXXXXX.
	GTMID string
	// GTMNoScript true ise <body> başında GTM noscript iframe'i de basılır.
	GTMNoScript bool
	// WebSubHubs yayın bildirimi hub adresleri (beyaz liste dışındakiler
	// httpapi tarafında atılır).
	WebSubHubs []string
	// ReklamZone Coinzilla bölge kimliği (örn. 345f78d37c7e19e740).
	ReklamZone string
	// ReklamAktif reklam alanları basılsın mı.
	ReklamAktif bool
	// AdSensePub AdSense yayıncı kimliği: ca-pub-XXXXXXXXXX.
	AdSensePub string
	// CointrafficKey Cointraffic wkey değeri.
	CointrafficKey string
	// ChromeEklentiURL Chrome Web Store'daki eklenti sayfası. Boşsa ana
	// sayfadaki tanıtım kartı basılmaz.
	ChromeEklentiURL string
}

// CDKulupURL Milyonerler Kulübü üye paneli (ana sayfadaki giriş kartı).
const CDKulupURL = "https://traders.bitcosar.com/"

var cdEnt atomic.Pointer[CDEntegrasyon]

// SetCDEntegrasyon entegrasyon kimliklerini yayına alır.
// Boş/nil geçmek "hiçbir entegrasyon basma" demektir.
func SetCDEntegrasyon(e *CDEntegrasyon) { cdEnt.Store(e) }

// CDEntegrasyonOku mevcut kimlikleri döndürür (yoksa nil).
func CDEntegrasyonOku() *CDEntegrasyon { return cdEnt.Load() }
