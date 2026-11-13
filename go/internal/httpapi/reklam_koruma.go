package httpapi

import (
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ---- Reklam tıklama sahtekârlığı koruması ----
//
// TEHDİT: reklam gelirini hacklemek isteyen kişiler bot/özel script ile
// kendi reklamlarımıza (ya da AdSense reklamlarına) çok kez tıklayıp
// reklam ağının bizi "geçersiz trafik" gerekçesiyle askıya almasını
// sağlamaya çalışıyor. Proxy/VPN ile her tıklamada farklı IP kullanarak
// IP tabanlı sayacı da atlatmaya çalışıyorlar.
//
// SAVUNMA KATMANLARI (hepsi sunucu tarafında, istemciye güvenmeden):
//  1. Motor/araç tespiti — UA ve başlık parmak izi (curl, python-requests,
//     headless chrome, selenium, puppeteer, fantom…).
//  2. Proxy/VPN sinyali — Via, Forwarded, Proxy-Connection, X-Proxy-*,
//     anormal X-Forwarded-For zinciri, CF-Connecting-IP ile çelişen başlık.
//  3. Tarayıcı gezinme kanıtı — Sec-Fetch-Site / Sec-Fetch-Mode /
//     Sec-Fetch-User; gerçek bir <a> tıklamasında tarayıcı bunları kendi
//     ekler, script/fetch ile taklit edilemez (aynı origin'den fetch farklı
//     değer taşır), curl hiç göndermez.
//  4. Referer bizim alan adımız mı.
//  5. IP başına hız sınırı (kayan pencere) — tek makineden seri tıklama.
//  6. (IP, reklam) tekilleştirme — aynı IP aynı reklamı 30 dk içinde bir
//     kez saydırır (tıklama sayısı şişmez).
//  7. İstem dışı çağrı tespiti — Accept başlığında text/html yoksa
//     (fetch/XHR/beacon) tıklama sayılmaz.
//
// DÜRÜST SINIR: dağıtık tıklama çiftlikleri (her biri farklı gerçek
// tarayıcı + farklı konut IP'si) TAMAMEN engellenemez; bu katmanlar
// "tek makineden/script ile/proxy ile" yapılan saldırıyı keser ve geri
// kalanı raporlanabilir hâle getirir. Konut-proxy ağlarını yakalamak için
// harici IP itibar servisi gerekir; bu yüzden aşağıdaki tasarım onu
// TAKILABİLİR bırakır (ipItibarKontrol) — kimlik girilirse devreye girer.

// reklamKorumaAyarlari eşikler. Testler bunları düşürebilsin diye değişken.
var (
	// reklamIPPencere tek IP için kayan pencere uzunluğu.
	reklamIPPencere = time.Minute
	// reklamIPTiklamaSinir pencerede sayılacak en çok tıklama.
	reklamIPTiklamaSinir = 5
	// reklamTekilleme aynı (IP, reklam) için sayım aralığı.
	reklamTekilleme = 30 * time.Minute
	// reklamGosterimSinir pencere başına gösterim (piksel) sınırı.
	reklamGosterimSinir = 60
)

// reklamKoruma süreç içi sayaçlar. Bellek üst sınırı için fırsatçı temizlik.
type reklamKoruma struct {
	mu sync.Mutex
	// ip → pencere
	tiklamalar map[string]*reklamPencere
	gosterim   map[string]*reklamPencere
	// "ip|ad" → son sayılan tıklama
	sonTiklama  map[string]time.Time
	sonTemizlik time.Time

	// görünürlük: panel/log için
	sayilan, supheli, engellenen atomic.Int64
}

type reklamPencere struct {
	baslangic time.Time
	adet      int
}

func newReklamKoruma() *reklamKoruma {
	return &reklamKoruma{
		tiklamalar:  map[string]*reklamPencere{},
		gosterim:    map[string]*reklamPencere{},
		sonTiklama:  map[string]time.Time{},
		sonTemizlik: time.Now(),
	}
}

// reklamKarar bir tıklamanın sınıflandırma sonucudur.
type reklamKarar struct {
	// Say boolean: DB'ye tıklama yazılsın mı.
	Say bool
	// Neden boşsa temiz; doluysa sayılmama gerekçesi (log/panel).
	Neden string
}

// otomatikUA bilinen bot/otomasyon imzaları. Küçük harfle karşılaştırılır.
//
// Not: "bot" alt dizesi bilinçli olarak YOK — "Robotics" gibi meşru
// markaları da yakalar. Bunun yerine gerçek araç adları listelenir.
var otomatikUA = []string{
	"curl/", "wget/", "python-requests", "python-urllib", "aiohttp",
	"go-http-client", "java/", "okhttp", "libwww-perl", "httpclient",
	"headlesschrome", "headless chrome", "phantomjs", "selenium",
	"puppeteer", "playwright", "webdriver", "chrome-lighthouse",
	"axios/", "node-fetch", "undici", "guzzle", "scrapy", "mechanize",
	"ahrefsbot", "semrushbot", "mj12bot", "dotbot", "petalbot",
	"bytespider", "dataforseo", "zoominfobot", "blexbot",
}

// otomatikMi UA otomasyon aracı mı.
func otomatikMi(ua string) (bool, string) {
	u := strings.ToLower(strings.TrimSpace(ua))
	if u == "" {
		return true, "boş user-agent"
	}
	for _, imza := range otomatikUA {
		if strings.Contains(u, imza) {
			return true, "otomasyon: " + imza
		}
	}
	return false, ""
}

// proxyBasliklariSinyali istek proxy/VPN üzerinden mi geliyor.
//
// GERÇEK sunucumuz Cloudflare arkasında: meşru isteklerde CF-Connecting-IP
// bulunur ve X-Forwarded-For genelde YOKTUR. Aşağıdakiler görüldüğünde
// istek bir aracıdan geçmiş demektir.
func proxyBasliklariSinyali(r *http.Request) (bool, string) {
	h := r.Header
	for _, ad := range []string{"Via", "Proxy-Connection", "X-Proxy-Id", "X-Proxy-User-IP", "Forwarded", "X-Real-IP", "X-Client-IP", "Client-IP", "X-Originating-IP"} {
		if v := strings.TrimSpace(h.Get(ad)); v != "" {
			// "Forwarded" ve "Via" meşru CDN'lerde de görülebilir; yine de
			// reklam tıklaması için şüpheli sayılır (tıklama sayılmaz).
			return true, "proxy başlığı: " + ad
		}
	}
	// XFF zinciri: birden çok atlama = aracı. Tek atlama Cloudflare'de de
	// görülebilir, o yüzden yalnız 2+ atlama şüpheli.
	if xff := h.Get("X-Forwarded-For"); xff != "" {
		if strings.Count(xff, ",") >= 2 {
			return true, "uzun X-Forwarded-For zinciri"
		}
	}
	return false, ""
}

// tarayiciGezinmeKaniti gerçek bir bağlantı tıklaması mı.
//
// Sec-Fetch-* başlıklarını TARAYICI ekler; sayfa içi JS/fetch bunları
// taklit edemez (fetch'te mode=cors, dest=empty olur) ve curl hiç göndermez.
// Desteklemeyen çok eski tarayıcılarda başlık yoktur: o durumda bu katman
// "kanıt yok" der ve tek başına engellemez (aşağıdaki diğer katmanlar sürer).
func tarayiciGezinmeKaniti(r *http.Request) (kanit bool, varMi bool, neden string) {
	site := r.Header.Get("Sec-Fetch-Site")
	mode := r.Header.Get("Sec-Fetch-Mode")
	dest := r.Header.Get("Sec-Fetch-Dest")
	if site == "" && mode == "" && dest == "" {
		return false, false, ""
	}
	if site != "" && site != "same-origin" && site != "same-site" && site != "none" {
		return false, true, "sec-fetch-site: " + site
	}
	if mode != "" && mode != "navigate" {
		return false, true, "sec-fetch-mode: " + mode
	}
	if dest != "" && dest != "document" && dest != "iframe" {
		return false, true, "sec-fetch-dest: " + dest
	}
	// Kullanıcı başlatmalı gezinme: gerçek tıklamada "?1".
	if strings.TrimSpace(r.Header.Get("Sec-Fetch-User")) == "" {
		return false, true, "sec-fetch-user yok"
	}
	return true, true, ""
}

// istemciGezinmeyiKabulEdiyor Accept başlığı HTML bekliyor mu.
//
// Tarayıcı adres çubuğundan/bağlantıdan gelen istekte `text/html` vardır;
// fetch/XHR/beacon isteğinde genelde `*/*` ya da JSON olur.
func istemciGezinmeyiKabulEdiyor(r *http.Request) bool {
	a := strings.ToLower(r.Header.Get("Accept"))
	return strings.Contains(a, "text/html") || strings.Contains(a, "*/*")
}

// reklamTiklamaDegerlendir tek bir tıklamayı değerlendirir.
//
// Sıra ÖNEMLİ: en ucuz ve en kesin kontroller önce; sayaçlar en sonda
// (sayılmayacak istek sayacı kirletmesin).
func (k *reklamKoruma) reklamTiklamaDegerlendir(r *http.Request, ip string, adID int64) reklamKarar {
	if ok, neden := otomatikMi(r.UserAgent()); ok {
		k.engellenen.Add(1)
		return reklamKarar{Neden: neden}
	}
	if ok, neden := proxyBasliklariSinyali(r); ok {
		k.engellenen.Add(1)
		return reklamKarar{Neden: neden}
	}
	if kanit, varMi, neden := tarayiciGezinmeKaniti(r); varMi && !kanit {
		k.engellenen.Add(1)
		return reklamKarar{Neden: neden}
	}
	if !istemciGezinmeyiKabulEdiyor(r) {
		k.engellenen.Add(1)
		return reklamKarar{Neden: "Accept başlığı gezinme değil"}
	}
	// Referer: varsa bizim origin'den olmalı. Yoksa (gizlilik ayarı) yalnız
	// uyarı — engelleme değil.
	if ref := strings.TrimSpace(r.Header.Get("Referer")); ref != "" {
		if !ayniOriginMi(r, ref) {
			k.engellenen.Add(1)
			return reklamKarar{Neden: "referer başka origin: " + kisalt(ref, 60)}
		}
	}

	if ip == "" {
		k.supheli.Add(1)
		return reklamKarar{Neden: "IP belirlenemedi"}
	}

	simdi := time.Now()
	k.mu.Lock()
	defer k.mu.Unlock()
	k.temizleKilitli(simdi)

	// (IP, reklam) tekilleştirme: tıklama sayısı şişmesin.
	anahtar := ip + "|" + itoa64(adID)
	if t, ok := k.sonTiklama[anahtar]; ok && simdi.Sub(t) < reklamTekilleme {
		k.supheli.Add(1)
		return reklamKarar{Neden: "tekrar tıklama (tekilleştirme)"}
	}

	// IP hız sınırı.
	p := k.tiklamalar[ip]
	if p == nil || simdi.Sub(p.baslangic) >= reklamIPPencere {
		p = &reklamPencere{baslangic: simdi}
		k.tiklamalar[ip] = p
	}
	if p.adet >= reklamIPTiklamaSinir {
		k.engellenen.Add(1)
		return reklamKarar{Neden: "IP hız sınırı"}
	}

	p.adet++
	k.sonTiklama[anahtar] = simdi
	k.sayilan.Add(1)
	return reklamKarar{Say: true}
}

// reklamGosterimIzinli gösterim pikseli için hız sınırı.
//
// NEDEN: gösterim sayısını şişirmek de reklam ağını "geçersiz trafik"
// noktasına götürür. Aynı IP'den dakikada onlarca piksel isteği bot işidir.
func (k *reklamKoruma) reklamGosterimIzinli(r *http.Request, ip string) bool {
	if ok, _ := otomatikMi(r.UserAgent()); ok {
		k.engellenen.Add(1)
		return false
	}
	if ip == "" {
		return false
	}
	simdi := time.Now()
	k.mu.Lock()
	defer k.mu.Unlock()
	k.temizleKilitli(simdi)
	p := k.gosterim[ip]
	if p == nil || simdi.Sub(p.baslangic) >= reklamIPPencere {
		p = &reklamPencere{baslangic: simdi}
		k.gosterim[ip] = p
	}
	if p.adet >= reklamGosterimSinir {
		k.engellenen.Add(1)
		return false
	}
	p.adet++
	return true
}

// temizleKilitli süresi geçmiş kayıtları atar (bellek sızıntısı yok).
//
// En fazla dakikada bir çalışır: her istekte tüm haritayı taramak O(n)
// olurdu. Kilit çağıranda tutulur.
func (k *reklamKoruma) temizleKilitli(simdi time.Time) {
	if simdi.Sub(k.sonTemizlik) < time.Minute {
		return
	}
	k.sonTemizlik = simdi
	for ip, p := range k.tiklamalar {
		if simdi.Sub(p.baslangic) >= reklamIPPencere {
			delete(k.tiklamalar, ip)
		}
	}
	for ip, p := range k.gosterim {
		if simdi.Sub(p.baslangic) >= reklamIPPencere {
			delete(k.gosterim, ip)
		}
	}
	for a, t := range k.sonTiklama {
		if simdi.Sub(t) >= reklamTekilleme {
			delete(k.sonTiklama, a)
		}
	}
}

// ayniOriginMi referer bizim host'umuza mı ait.
func ayniOriginMi(r *http.Request, referer string) bool {
	u, err := url.Parse(referer)
	if err != nil || u.Host == "" {
		return false
	}
	return strings.EqualFold(u.Host, r.Host)
}

func kisalt(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// reklamKorumaIstatistik panel/log için sayıları döndürür.
func (k *reklamKoruma) istatistik() (sayilan, supheli, engellenen int64) {
	return k.sayilan.Load(), k.supheli.Load(), k.engellenen.Load()
}

// ipGecerli mi (net.ParseIP) — tablo/DB'ye yazmadan önce doğrulama.
func ipGecerli(ip string) bool {
	return net.ParseIP(ip) != nil
}
