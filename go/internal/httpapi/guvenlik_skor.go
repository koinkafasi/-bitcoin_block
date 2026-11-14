package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
)

// RİSK SKORU — tek karar noktası.
//
// TASARIM İLKESİ: sunucu tarafından VPN arkasındaki GERÇEK IP
// kriptografik olarak çıkarılamaz. Elimizde yalnız kanıt katmanları var:
//
//	1) AĞ      — ASN sağlayıcı sınıfı (hosting/vpn/tor). En güçlü sinyal.
//	2) BAŞLIK  — proxy başlıkları, XFF çelişkisi, Sec-Fetch eksikliği.
//	3) DAVRANIŞ— hız aşımı, tuzak yolu, IP/ülke rotasyonu.
//	4) İSTEMCİ — WebRTC sızıntısı, webdriver, saat dilimi uyuşmazlığı.
//	             YALNIZ PUANI YÜKSELTİR (istemciye güvenilmez).
//
// SINIFLAR (eşikler ayarlanabilir):
//
//	temiz    < 30   → sayılır, reklam gösterilir
//	supheli  ≥ 30   → SAYILMAZ, reklam gizlenir (okuma deneyimi bozulmaz)
//	yuksek   ≥ 60   → SAYILMAZ, reklam gizlenir
//	bot      ≥ 85   → SAYILMAZ, engellenir (403)
//	arama           → doğrulanmış arama motoru: engellenmez, SAYILMAZ
//
// NEDEN "sayılma" eşiği 30'da bu kadar düşük: bir okunma sayısının kirli
// olması, temiz bir okunmanın kaçırılmasından daha maliyetli. 30 puan
// tek başına bir sinyalle (proxy başlığı 35) aşılabilir; ama tek başına
// "dil yok" (10) veya "etkileşim yok" (20) aşamaz — yani tek zayıf sinyal
// kimseyi dışlamaz.

// guvenlikSinyali tek bir kanıt parçası.
type guvenlikSinyali struct {
	K string `json:"k"` // kısa kod (panelde gruplanır)
	P int    `json:"p"` // puan katkısı
	N string `json:"n"` // insan okur açıklama
}

// Sinyal ağırlıkları — tek yerde, test edilebilir.
//
// pUAOtomasyon 90 (eşik 85'in ÜSTÜNDE): curl/headless UA'sı TEK BAŞINA bot
// sayılmalı. 80 verilseydi otomasyon aracı yalnız "yüksek" sınıfa düşer,
// engellenmez ve sayaç yine kirlenirdi — test yakaladı (14 Kasım 2026).
const (
	pUAOtomasyon   = 90
	pUABos         = 40
	pASNHosting    = 45
	pASNVPN        = 60
	pASNTor        = 70
	pBasliKProxy   = 35
	pXFFCeliski    = 40
	pSecFetchYok   = 25
	pDilYok        = 10
	pIPRotasyon    = 50
	pUlkeRotasyon  = 40
	pHizAsimi      = 55
	pTuzak         = 90
	pWebdriver     = 80
	pHeadlessGPU   = 55
	pTZUyusmazlik  = 40
	pDilUyusmazlik = 15
	pWebRTCFarkli  = 45
	pEtkilesimYok  = 20
	pSahteCrawler  = 85
)

// guvenlikKarari middleware'in context'e koyduğu karar.
type guvenlikKarari struct {
	Anahtar   string
	IP        string
	Ulke      string
	ASN       int
	ASNAd     string
	ASNTur    string
	IPAgg     string
	Puan      int
	Sinif     string // temiz|supheli|yuksek|bot|arama
	Sayilir   bool   // okunma sayılsın mı
	Reklam    bool   // reklam gösterilsin mi
	Engelle   bool   // 403 mü
	Sinyaller []guvenlikSinyali
	// Siniflandirildi: ASN bilgisi geldi mi? Gelmediyse sınıf UA/başlık
	// katmanıyla verilmiş demektir (ilk istek ödünleşimi).
	Siniflandirildi bool
}

// guvenlikKarariAnahtari kararı istek bağlamında taşır.
type guvenlikKarariAnahtari struct{}

func guvenlikKarariAl(r *http.Request) (*guvenlikKarari, bool) {
	v, ok := r.Context().Value(guvenlikKarariAnahtari{}).(*guvenlikKarari)
	return v, ok && v != nil
}

// sayilirMi: okunma sayılsın mı? Karar yoksa (middleware kapalı) EVET —
// güvenlik katmanı kapatıldığında eski davranış korunur.
func sayilirMi(r *http.Request) bool {
	k, ok := guvenlikKarariAl(r)
	if !ok {
		return true
	}
	return k.Sayilir
}

// reklamGosterilir mi? Karar yoksa evet.
func reklamGosterilir(r *http.Request) bool {
	k, ok := guvenlikKarariAl(r)
	if !ok {
		return true
	}
	return k.Reklam
}

// aramaMotoruMu UA'dan arama motoru botu tanır.
//
// AYRI SINIF: Googlebot/Bingbot engellenmemeli ve "gerçek ziyaretçi"
// sayılmamalı. Ama sahte Googlebot çok yaygın; bu yüzden crawler UA'sı
// TEK BAŞINA yetmez — middleware ters DNS doğrulaması ister
// (bkz. crawlerDogrula).
func aramaMotoruMu(ua string) (bool, string) {
	u := strings.ToLower(strings.TrimSpace(ua))
	if u == "" {
		return false, ""
	}
	for _, imza := range []string{
		"googlebot", "google-inspectiontool", "bingbot", "bingpreview",
		"yandexbot", "duckduckbot", "baiduspider", "applebot",
		"yeti", "naver", "seznam", "sogou", "exabot",
	} {
		if strings.Contains(u, imza) {
			return true, imza
		}
	}
	return false, ""
}

// crawlerDogrula iddia edilen arama motoru botunun gerçekten o ağdan
// gelip gelmediğini ters DNS ile doğrular.
//
// ÜÇ DURUM (fail-open tasarımı):
//
//	doğrulandı    → arama sınıfı, engellenmez
//	reddedildi    → sahte crawler → bot sınıfı
//	DNS hatası    → DOĞRULANAMADI: bot sayılmaz (yanlış pozitif Googlebot'u
//	                engellemek SEO'yu bozar; risk asimetrik).
func (s *Server) crawlerDogrula(r *http.Request, ip string) (dogrulandi bool, kesin bool) {
	// Google/Bing kendi CIDR'lerini yayınlıyor ama liste değişken; ters DNS
	// asgari ve yeterli doğrulama: PTR *.googlebot.com / *.search.msn.com.
	adlar, err := tersDNSAdlari(r, ip)
	if err != nil || len(adlar) == 0 {
		return false, false
	}
	for _, ad := range adlar {
		a := strings.ToLower(strings.TrimSuffix(ad, "."))
		if strings.HasSuffix(a, ".googlebot.com") ||
			strings.HasSuffix(a, ".google.com") ||
			strings.HasSuffix(a, ".search.msn.com") ||
			strings.HasSuffix(a, ".yandex.com") ||
			strings.HasSuffix(a, ".yandex.ru") ||
			strings.HasSuffix(a, ".applebot.apple.com") {
			return true, true
		}
	}
	return false, true
}

// guvenlikHesapla kararı üretir. Saf fonksiyon (DNS hariç): test edilebilir.
//
// `asn` nil ise ağ katmanı yoktur; skor yalnız UA/başlık/davranıştan gelir.
func guvenlikHesapla(girdi guvenlikGirdi) *guvenlikKarari {
	k := &guvenlikKarari{
		Anahtar:   girdi.Anahtar,
		IP:        girdi.IP,
		Ulke:      girdi.Ulke,
		Sinyaller: []guvenlikSinyali{},
	}
	ekle := func(kod string, puan int, neden string) {
		if puan <= 0 {
			return
		}
		k.Sinyaller = append(k.Sinyaller, guvenlikSinyali{K: kod, P: puan, N: neden})
		k.Puan += puan
	}

	// ---- Arama motoru: ayrı sınıf, engellenmez ----
	if girdi.AramaMotoru {
		k.Sinif = "arama"
		k.Sayilir = false
		k.Reklam = false
		k.Siniflandirildi = true
		return k
	}

	// ---- 1) AĞ KATMANI ----
	if girdi.ASN != nil {
		k.ASN = girdi.ASN.ASN
		k.ASNAd = girdi.ASN.Ad
		k.ASNTur = girdi.ASN.Tur
		k.IPAgg = girdi.ASN.Ag
		k.Siniflandirildi = true
		etiket := asnSaglayiciEtiket(girdi.ASN.Ad)
		if etiket == "" {
			etiket = "AS" + itoa(girdi.ASN.ASN)
		}
		switch girdi.ASN.Tur {
		case "hosting":
			ekle("asn_hosting", pASNHosting, "veri merkezi: "+etiket)
		case "vpn":
			ekle("asn_vpn", pASNVPN, "ticari VPN: "+etiket)
		case "tor":
			ekle("tor", pASNTor, "Tor çıkışı")
		}
	}
	if girdi.Tor {
		ekle("tor", pASNTor, "Tor çıkış düğümü")
	}

	// ---- 2) UA ----
	if girdi.UAOtomasyon {
		ekle("ua_bot", pUAOtomasyon, "otomasyon aracı: "+girdi.UANeden)
	} else if girdi.UABos {
		ekle("ua_bos", pUABos, "boş user-agent")
	}

	// ---- 3) BAŞLIK ANOMALİLERİ ----
	if girdi.ProxyBaslik {
		ekle("baslik_proxy", pBasliKProxy, girdi.ProxyNeden)
	}
	if girdi.XFFCeliski {
		ekle("xff_celiski", pXFFCeliski, "X-Forwarded-For ile CF-Connecting-IP çelişiyor")
	}
	if girdi.SecFetchYok && !girdi.APIIstek {
		// API/beacon istekleri Sec-Fetch taşımaz (fetch); yalnız sayfa
		// gezinmesinde aranır. Aksi hâlde kendi beacon'ımız bizi cezalandırırdı.
		ekle("secfetch_yok", pSecFetchYok, "tarayıcı gezinme başlıkları yok")
	}
	if girdi.DilYok && !girdi.APIIstek {
		ekle("dil_yok", pDilYok, "Accept-Language yok")
	}

	// ---- 4) DAVRANIŞ ----
	if girdi.IPRotasyon >= 3 {
		ekle("ip_rotasyon", pIPRotasyon, "kısa sürede "+itoa(girdi.IPRotasyon)+" farklı IP")
	}
	if girdi.UlkeRotasyon >= 2 {
		ekle("ulke_rotasyon", pUlkeRotasyon, "kısa sürede "+itoa(girdi.UlkeRotasyon)+" farklı ülke")
	}
	if girdi.HizAsimi {
		ekle("hiz_asimi", pHizAsimi, "istek hızı insan sınırının üstünde")
	}
	if girdi.Tuzak {
		ekle("tuzak", pTuzak, "bot tuzağı yolu ziyaret edildi")
	}

	// ---- 5) İSTEMCİ BEACON SINYALLERİ (yalnız yükseltir) ----
	if girdi.Webdriver {
		ekle("webdriver", pWebdriver, "navigator.webdriver=true")
	}
	if girdi.HeadlessGPU {
		ekle("headless_gpu", pHeadlessGPU, "yazılım (SwiftShader) grafik birimi")
	}
	if girdi.TZUyusmazlik {
		ekle("tz_uyusmazlik", pTZUyusmazlik, "saat dilimi ile IP ülkesi uyuşmuyor")
	}
	if girdi.DilUyusmazlik {
		ekle("dil_uyusmazlik", pDilUyusmazlik, "tarayıcı dili ile IP ülkesi uyuşmuyor")
	}
	if girdi.WebRTCFarkli {
		ekle("webrtc_farkli", pWebRTCFarkli, "WebRTC farklı genel IP sızdırdı: "+girdi.WebRTCIP)
	}
	if girdi.EtkilesimYok {
		ekle("etkilesim_yok", pEtkilesimYok, "sayfada hiç etkileşim yok")
	}
	if girdi.SahteCrawler {
		ekle("sahte_crawler", pSahteCrawler, "arama motoru botu iddiası doğrulanamadı")
	}

	// PUAN 100'DE KIRPILIR: sinyaller toplanınca 265 gibi değerler çıkıyor
	// (ölçüldü 14 Kasım 2026: webdriver 80 + gpu 55 + tz 40 + rtc 45 + 20).
	// Ham toplamı hem DB'ye hem panele yazmak, çubuğu %100'de dururken
	// "265" yazmak demekti — tutarsız ve anlamsız. Eşikler 0..100 ölçeğinde
	// tanımlı; puan da o ölçekte kalmalı.
	if k.Puan > 100 {
		k.Puan = 100
	}

	k.Sinif, k.Sayilir, k.Reklam, k.Engelle = sinifla(k.Puan, girdi.EsikYuksek, girdi.EsikBot, girdi.Politika)
	// POLİTİKA ENGELİ — sınıftan BAĞIMSIZ: hosting 45 puanla yalnız
	// "şüpheli" olur, ama yönetici "veri merkezini engelle" dediyse 45
	// puan yeterli değildir diye engellememek yanlış olurdu. Karar
	// politikadan gelir, puandan değil.
	if girdi.ASN != nil {
		if girdi.ASN.Tur == "hosting" && girdi.Politika.HostingEngel {
			k.Engelle = true
		}
		if girdi.ASN.Tur == "vpn" && girdi.Politika.VPNEngel {
			k.Engelle = true
		}
	}
	if girdi.Tor && (girdi.Politika.VPNEngel || girdi.Politika.HostingEngel) {
		k.Engelle = true
	}
	return k
}

// sinifla puanı sınıfa çevirir ve politika bayraklarını uygular. Saf.
func sinifla(puan, esikYuksek, esikBot int, p guvenlikPolitikasi) (sinif string, sayilir, reklam, engelle bool) {
	if esikYuksek <= 0 {
		esikYuksek = 60
	}
	if esikBot <= 0 || esikBot <= esikYuksek {
		esikBot = 85
	}
	switch {
	case puan >= esikBot:
		return "bot", false, false, p.BotEngel
	case puan >= esikYuksek:
		return "yuksek", false, false, false
	case puan >= 30:
		return "supheli", false, false, false
	}
	return "temiz", true, true, false
}

// guvenlikGirdi hesaplamaya giren tüm kanıtlar.
type guvenlikGirdi struct {
	Anahtar string
	IP      string
	Ulke    string

	ASN          *asnBilgi
	Tor          bool
	AramaMotoru  bool
	SahteCrawler bool

	UAOtomasyon bool
	UABos       bool
	UANeden     string

	ProxyBaslik bool
	ProxyNeden  string
	XFFCeliski  bool
	SecFetchYok bool
	DilYok      bool
	APIIstek    bool

	IPRotasyon   int
	UlkeRotasyon int
	HizAsimi     bool
	Tuzak        bool

	Webdriver     bool
	HeadlessGPU   bool
	TZUyusmazlik  bool
	DilUyusmazlik bool
	WebRTCFarkli  bool
	WebRTCIP      string
	EtkilesimYok  bool

	EsikYuksek int
	EsikBot    int
	Politika   guvenlikPolitikasi
}

// guvenlikPolitikasi panelden yönetilen engelleme kararları.
type guvenlikPolitikasi struct {
	Aktif        bool
	VPNEngel     bool
	HostingEngel bool
	BotEngel     bool
	WebRTC       bool
}

// sinyallerJSON panelde saklanacak biçim.
func (k *guvenlikKarari) sinyallerJSON() string {
	if len(k.Sinyaller) == 0 {
		return "[]"
	}
	b, err := json.Marshal(k.Sinyaller)
	if err != nil {
		return "[]"
	}
	return string(b)
}
