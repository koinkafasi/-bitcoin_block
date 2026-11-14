package templkit

import (
	"strings"
	"sync/atomic"
)

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
	// KulupURL ana sayfadaki "Milyonerler Kulübü" kartının hedefi. Boşsa
	// CDKulupURLVarsayilan kullanılır.
	//
	// NEDEN AYARLANABİLİR: kartın hedefi zamanla değişiyor — üye paneli
	// (traders.bitcosar.com) ile sitenin kendi satış sayfası (/funnel)
	// arasında gidip geliyor. Sabit tutulduğunda her değişiklik kod
	// değişikliği + imaj sevki demekti; metin panelden düzenlenebilirken
	// bağlantının kodda kalması tutarsızdı.
	KulupURL string
}

// CDKulupURLVarsayilan panelden adres girilmediğinde kullanılan hedef.
//
// ESKİ DAVRANIŞ KORUNUYOR: ayar boşken kart bugüne kadar olduğu yere
// (üye paneli) gitmeye devam eder; kimse ayar girmezse hiçbir şey bozulmaz.
const CDKulupURLVarsayilan = "https://traders.bitcosar.com/"

// CDKulupHref kartın basılacak hedefi (ayar boşsa varsayılan).
//
// SAVUNMA KATMANI: adres kaydedilirken httpapi tarafında doğrulanıyor.
// Burada ikinci bir kapı var çünkü templkit'e değer başka bir yoldan
// (ileride eklenecek bir çağıran) gelebilir; `javascript:` şeması bir kez
// basılırsa XSS olur. Yalnız "/" ile başlayan yol ya da https kabul edilir.
func CDKulupHref() string {
	e := cdEnt.Load()
	if e == nil {
		return CDKulupURLVarsayilan
	}
	u := e.KulupURL
	if u == "" {
		return CDKulupURLVarsayilan
	}
	if strings.HasPrefix(u, "//") || strings.HasPrefix(u, "/\\") {
		return CDKulupURLVarsayilan
	}
	if strings.HasPrefix(u, "/") || strings.HasPrefix(u, "https://") {
		return u
	}
	return CDKulupURLVarsayilan
}

// CDKulupDis true ise kart yeni sekmede açılmalı.
//
// YALNIZ DIŞ ADRESLER: site içi bir yol (/funnel) yeni sekmede açılırsa
// ziyaretçi ana sayfayı kaybetmiş gibi hisseder; aynı sekmede ilerlemek
// beklenen davranış.
func CDKulupDis() bool {
	return strings.HasPrefix(CDKulupHref(), "http://") ||
		strings.HasPrefix(CDKulupHref(), "https://")
}

var cdEnt atomic.Pointer[CDEntegrasyon]

// SetCDEntegrasyon entegrasyon kimliklerini yayına alır.
// Boş/nil geçmek "hiçbir entegrasyon basma" demektir.
func SetCDEntegrasyon(e *CDEntegrasyon) { cdEnt.Store(e) }

// CDEntegrasyonOku mevcut kimlikleri döndürür (yoksa nil).
func CDEntegrasyonOku() *CDEntegrasyon { return cdEnt.Load() }
