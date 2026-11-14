package httpapi

import (
	"context"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"theblock/site/internal/repo"
)

// TRAFİK GÜVENLİĞİ MIDDLEWARE — zincirin EN DIŞINDA.
//
// NEDEN EN DIŞTA: engelleme kararı sayfa önbelleğinden, gzip'ten ve
// yönlendirmelerden ÖNCE verilmeli. İçeride olsaydı, önbellekte oturan bir
// sayfa botlara servis edilir ve sayaç yine kirlenirdi (ölçülen sızıntı:
// sayfa_onbellek.go isabette `okunmaTamponu.ekle` sahibine bakmıyordu).
//
// İLK İSTEK ÖDÜNLEŞİMİ (dürüstçe): yeni bir IP'de ASN sorgusu 120 ms
// bütçesini aşarsa sınıflandırma UA/başlık katmanıyla yapılır ve
// zenginleştirme arka planda tamamlanır. Yani HOSTING'DEN gelen bir bot
// ilk isteğinde UA'sı temizse "temiz" sayılabilir; sonraki istekte doğru
// sınıflanır. Sayfa hızını DNS'e feda etmemek bilinçli bir tercih.

const (
	// hizPencere / hizSinir: insan davranışının üstündeki istek hızı.
	// 30 saniyede 40 sayfa: hızlı okuyucu bile bunu aşmaz; crawler aşar.
	hizPencere = 30 * time.Second
	hizSinir   = 40
	// riskYazmaAralik aynı ziyaretçi için DB yazma sıklığı.
	riskYazmaAralik = 10 * time.Minute
	// durumSinir bellek üst sınırı (aktif ziyaretçi izleme).
	durumSinir = 20000
)

// ziyaretciDurum ziyaretçi başına kısa ömürlü bellek durumu.
type ziyaretciDurum struct {
	IPler      map[string]time.Time
	Ulkeler    map[string]time.Time
	Istekler   []time.Time
	SonYazma   time.Time
	SonSinif   string
	SonGorulme time.Time
	WebRTCIP   string
	// Beacon'dan gelen ve hesaba katılacak bayraklar.
	Webdriver     bool
	HeadlessGPU   bool
	TZUyusmazlik  bool
	DilUyusmazlik bool
	WebRTCFarkli  bool
	EtkilesimYok  bool
}

type guvenlikDurum struct {
	mu       sync.Mutex
	m        map[string]*ziyaretciDurum
	temizlik time.Time
}

var guvenlikDurumlari = &guvenlikDurum{m: map[string]*ziyaretciDurum{}}

// al ziyaretçi durumunu döndürür, gerekirse oluşturur.
func (g *guvenlikDurum) al(anahtar string, ip, ulke string) *ziyaretciDurum {
	g.mu.Lock()
	defer g.mu.Unlock()
	now := time.Now()
	if now.Sub(g.temizlik) > 5*time.Minute {
		g.temizleKilitsiz(now)
	}
	d, ok := g.m[anahtar]
	if !ok {
		if len(g.m) >= durumSinir {
			// Bellek dolu: en eski kaydı at (kaba ama sınırlı).
			var eskiAnahtar string
			var eskiZaman time.Time
			for k, v := range g.m {
				if eskiAnahtar == "" || v.SonGorulme.Before(eskiZaman) {
					eskiAnahtar, eskiZaman = k, v.SonGorulme
				}
			}
			if eskiAnahtar != "" {
				delete(g.m, eskiAnahtar)
			}
		}
		d = &ziyaretciDurum{
			IPler:   map[string]time.Time{},
			Ulkeler: map[string]time.Time{},
		}
		g.m[anahtar] = d
	}
	d.SonGorulme = now
	if ip != "" {
		d.IPler[ip] = now
	}
	if ulke != "" {
		d.Ulkeler[ulke] = now
	}
	// Pencere dışı istekleri at.
	kes := now.Add(-hizPencere)
	i := 0
	for ; i < len(d.Istekler); i++ {
		if d.Istekler[i].After(kes) {
			break
		}
	}
	if i > 0 {
		d.Istekler = append([]time.Time(nil), d.Istekler[i:]...)
	}
	d.Istekler = append(d.Istekler, now)
	return d
}

// temizleKilitsiz 1 saatten eski kayıtları siler (kilit çağıranda).
func (g *guvenlikDurum) temizleKilitsiz(now time.Time) {
	g.temizlik = now
	kes := now.Add(-time.Hour)
	for k, v := range g.m {
		if v.SonGorulme.Before(kes) {
			delete(g.m, k)
		}
	}
}

// guvenlikDurumKopyasi kilit altında okunacak alanları toplar.
type durumOzeti struct {
	IPRotasyon    int
	UlkeRotasyon  int
	IstekSayisi   int
	HizAsimi      bool
	SonYazma      time.Time
	SonSinif      string
	Webdriver     bool
	HeadlessGPU   bool
	TZUyusmazlik  bool
	DilUyusmazlik bool
	WebRTCFarkli  bool
	WebRTCIP      string
	EtkilesimYok  bool
}

func (g *guvenlikDurum) ozet(anahtar string) durumOzeti {
	g.mu.Lock()
	defer g.mu.Unlock()
	d, ok := g.m[anahtar]
	if !ok {
		return durumOzeti{}
	}
	now := time.Now()
	kes := now.Add(-30 * time.Minute)
	ipSay := 0
	for _, t := range d.IPler {
		if t.After(kes) {
			ipSay++
		}
	}
	ulkeSay := 0
	for _, t := range d.Ulkeler {
		if t.After(kes) {
			ulkeSay++
		}
	}
	return durumOzeti{
		IPRotasyon:    ipSay,
		UlkeRotasyon:  ulkeSay,
		IstekSayisi:   len(d.Istekler),
		HizAsimi:      len(d.Istekler) > hizSinir,
		SonYazma:      d.SonYazma,
		SonSinif:      d.SonSinif,
		Webdriver:     d.Webdriver,
		HeadlessGPU:   d.HeadlessGPU,
		TZUyusmazlik:  d.TZUyusmazlik,
		DilUyusmazlik: d.DilUyusmazlik,
		WebRTCFarkli:  d.WebRTCFarkli,
		WebRTCIP:      d.WebRTCIP,
		EtkilesimYok:  d.EtkilesimYok,
	}
}

func (g *guvenlikDurum) yazmaIsaretle(anahtar, sinif string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if d, ok := g.m[anahtar]; ok {
		d.SonYazma = time.Now()
		d.SonSinif = sinif
	}
}

// sinyalleriKopyala durumdaki bayrakları girdiye taşır.
func (g *guvenlikDurum) bayraklariAl(anahtar string, girdi *guvenlikGirdi) {
	o := g.ozet(anahtar)
	girdi.IPRotasyon = o.IPRotasyon
	girdi.UlkeRotasyon = o.UlkeRotasyon
	girdi.HizAsimi = o.HizAsimi
	girdi.Webdriver = o.Webdriver
	girdi.HeadlessGPU = o.HeadlessGPU
	girdi.TZUyusmazlik = o.TZUyusmazlik
	girdi.DilUyusmazlik = o.DilUyusmazlik
	girdi.WebRTCFarkli = o.WebRTCFarkli
	girdi.WebRTCIP = o.WebRTCIP
	girdi.EtkilesimYok = o.EtkilesimYok
}

// withGuvenlik trafik güvenliği katmanı.
//
// OLAY KAYDI: her istek visitor_events'e yazılır (14 gün). Karar verildikten
// SONRA, yanıtın gerçek durum koduyla birlikte — "403 aldı" bilgisi ancak
// handler çalışınca bilinir. Yazma kuyruğa bırakılır, istek beklemez.
func (s *Server) withGuvenlik(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		pol := s.guvenlikPolitikasi(ctx)
		if !pol.Aktif {
			next.ServeHTTP(w, r)
			return
		}
		p := r.URL.Path
		// ATLANACAK YOLLAR: statik varlık, admin paneli, smoke ve beacon.
		// Admin panelini engellemek operatörü kendi sitesinden kilitler;
		// bu katman okuyucu trafiği içindir.
		if strings.HasPrefix(p, "/static/") || strings.HasPrefix(p, "/admin") ||
			strings.HasPrefix(p, "/api/guvenlik") || strings.HasPrefix(p, "/_") ||
			r.Header.Get("X-Tb-Smoke") != "" {
			next.ServeHTTP(w, r)
			return
		}

		ip := clientIP(r)
		if ip == "" || isPrivateIP(ip) || s.isAllowlistedIP(ctx, ip) {
			next.ServeHTTP(w, r)
			return
		}

		ulke := cfUlke(r)
		anahtar := s.ziyaretciID(w, r, false)
		if anahtar == "" {
			anahtar = "ip:" + ip
		}

		guvenlikDurumlari.al(anahtar, ip, ulke)
		girdi := guvenlikGirdi{
			Anahtar:    anahtar,
			IP:         ip,
			Ulke:       ulke,
			EsikYuksek: s.ayarSayi(ctx, "guvenlik", "esik_yuksek", 60, 30, 100),
			EsikBot:    s.ayarSayi(ctx, "guvenlik", "esik_bot", 85, 50, 100),
			Politika:   pol,
		}

		// UA
		ua := strings.TrimSpace(r.UserAgent())
		girdi.UserAgent = ua
		if ua == "" {
			girdi.UABos = true
		} else if oto, neden := otomatikMi(ua); oto {
			girdi.UAOtomasyon = true
			girdi.UANeden = neden
		}

		// Arama motoru: UA iddiası + ters DNS doğrulaması.
		if crawler, _ := aramaMotoruMu(ua); crawler {
			dogrulandi, kesin := s.crawlerDogrula(r, ip)
			switch {
			case dogrulandi:
				girdi.AramaMotoru = true
			case kesin:
				// İddia var, DNS başka ağ gösteriyor → sahte crawler.
				girdi.SahteCrawler = true
			}
			// DNS hatası: ne arama ne bot. Diğer katmanlar karar verir.
		}

		// Başlıklar
		if v, neden := proxyBasliklariSinyali(r); v {
			girdi.ProxyBaslik = true
			girdi.ProxyNeden = neden
		}
		girdi.XFFCeliski = xffCeliski(r, ip)
		girdi.DilYok = strings.TrimSpace(r.Header.Get("Accept-Language")) == ""
		_, kanitVar, _ := tarayiciGezinmeKaniti(r)
		girdi.SecFetchYok = !kanitVar && isSayfaIstek(r)

		// Tuzak yolu: honeypot. Buraya gelen insan değildir.
		if isTrapPath(p) {
			girdi.Tuzak = true
		}

		// Ağ katmanı
		if b, ok := s.asnSorgula(ctx, ip); ok {
			bb := b
			girdi.ASN = &bb
		}
		// TOR YALNIZ ŞÜPHELİ AĞDA SORULUR — ÖLÇÜLEN KUSUR (14 Kasım 2026):
		// yorum "yalnız hosting ise" diyordu ama kod her istekte çağırıyordu.
		// Tor çıkışları Team Cymru'da sıradan hosting (veya VPN) görünür;
		// "konut/mobil" sınıfında Tor olma ihtimali yok denecek kadar az.
		// Her isteğe 60 ms DNS eklemek, sıfır bilgi karşılığında sayfa
		// gecikmesi demekti. Artık yalnız ağ katmanı şüpheliyse sorulur.
		if girdi.ASN != nil && (girdi.ASN.Tur == "hosting" || girdi.ASN.Tur == "vpn") {
			girdi.Tor = isTorExit(ip)
		}

		// İstemci sinyalleri (varsa)
		guvenlikDurumlari.bayraklariAl(anahtar, &girdi)

		karar := guvenlikHesapla(girdi)

		// DB yazımı: sınıf değiştiyse veya 10 dk geçtiyse.
		//
		// ÇEREZSİZ SATIR YAZILMAZ (sayılan istisna hariç) — ÖLÇÜLEN KUSUR
		// (14 Kasım 2026): ilk istekte çerez henüz yok (withKonum onu
		// zincirin İÇİNDE yazıyor), bu yüzden anahtar "ip:<adres>" oluyor;
		// ikinci istekte çerez gelince ayrı bir UUID satırı doğuyordu.
		// Aynı ziyaretçi panelde İKİ satır görünüp "toplam profil"i
		// şişiriyordu. Kural:
		//   · çerez varsa            → yaz (kimlik kararlı)
		//   · sayılmayan sınıf       → yaz (bot/arama/VPN çerez almaz,
		//                              panelde görünmeleri gerekir)
		//   · çerezsiz + sayılan     → YAZMA; birazdan çerezle gelecek
		yaz := gecerliUUID(anahtar) || !karar.Sayilir
		o := guvenlikDurumlari.ozet(anahtar)
		if yaz && (time.Since(o.SonYazma) > riskYazmaAralik || o.SonSinif != karar.Sinif) {
			guvenlikDurumlari.yazmaIsaretle(anahtar, karar.Sinif)
			s.riskArkaPlandaYaz(karar)
		}

		if karar.Engelle {
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("X-Robots-Tag", "noindex, nofollow")
			http.Error(w, "403 Forbidden", http.StatusForbidden)
			s.olayKaydet(r, karar, http.StatusForbidden, true)
			return
		}

		dw := &durumYazan{ResponseWriter: w}
		next.ServeHTTP(dw, r.WithContext(context.WithValue(ctx, guvenlikKarariAnahtari{}, karar)))
		durum := dw.durum
		if durum == 0 {
			durum = http.StatusOK
		}
		s.olayKaydet(r, karar, durum, false)
	})
}

// isSayfaIstek tarayıcı sayfa gezinmesi mi (beacon/XHR değil).
func isSayfaIstek(r *http.Request) bool {
	dest := r.Header.Get("Sec-Fetch-Dest")
	if dest == "document" || dest == "iframe" || dest == "" {
		if r.Method != http.MethodGet {
			return false
		}
		// HTML bekleyen istek
		acc := r.Header.Get("Accept")
		return acc == "" || strings.Contains(acc, "text/html")
	}
	return false
}

// xffCeliski: XFF ile CF-Connecting-IP farklı ağları gösteriyor mu.
//
// GERÇEK ÇELİŞKİ ÖLÇÜTÜ: ilk XFF atlaması ile CF başlığı aynı değilse ve
// XFF özel/ayrı bir ağdaysa. Cloudflare normalde XFF'i kendisi yazar
// (tek atlama, CF-Connecting-IP ile aynı). Farklıysa araya bir aracı
// girmiş demektir.
func xffCeliski(r *http.Request, cfIP string) bool {
	xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if xff == "" || cfIP == "" {
		return false
	}
	ilk := xff
	if i := strings.IndexByte(xff, ','); i > 0 {
		ilk = strings.TrimSpace(xff[:i])
	}
	if ilk == "" {
		return false
	}
	if a, b := net.ParseIP(ilk), net.ParseIP(cfIP); a != nil && b != nil && a.Equal(b) {
		return false
	}
	return true
}

// riskArkaPlandaYaz kararı DB'ye yazar; isteği bekletmez.
func (s *Server) riskArkaPlandaYaz(k *guvenlikKarari) {
	if s.pool == nil {
		return
	}
	satir := riskSatiri(k)
	go func() {
		ctx, iptal := context.WithTimeout(context.Background(), 4*time.Second)
		defer iptal()
		if err := repo.RiskKaydet(ctx, s.pool, satir); err != nil {
			s.logger.Debug("risk satırı yazılamadı", "error", err)
		}
	}()
}

// riskSatiri kararı DB satırına çevirir.
func riskSatiri(k *guvenlikKarari) repo.RiskSatir {
	vid := ""
	if gecerliUUID(k.Anahtar) {
		vid = k.Anahtar
	}
	return repo.RiskSatir{
		Anahtar:     k.Anahtar,
		VisitorID:   vid,
		IP:          k.IP,
		IPAgg:       k.IPAgg,
		ASN:         k.ASN,
		ASNAd:       k.ASNAd,
		ASNUlke:     "",
		ASNTur:      turVeya(k.ASNTur, "bilinmiyor"),
		Ulke:        k.Ulke,
		Puan:        k.Puan,
		Sinif:       k.Sinif,
		VPN:         k.sinifBayragi("asn_vpn"),
		Proxy:       k.sinifBayragi("baslik_proxy") || k.sinifBayragi("xff_celiski"),
		Hosting:     k.sinifBayragi("asn_hosting"),
		Tor:         k.sinifBayragi("tor"),
		Bot:         k.Sinif == "bot",
		Sinyaller:   k.sinyallerJSON(),
		IPDegisim:   0,
		IstekSayisi: 1,
		UserAgent:   k.UA,
	}
}

func turVeya(v, d string) string {
	if strings.TrimSpace(v) == "" {
		return d
	}
	return v
}

// sinifBayragi karar sinyallerinde kod var mı.
func (k *guvenlikKarari) sinifBayragi(kod string) bool {
	for _, s := range k.Sinyaller {
		if s.K == kod {
			return true
		}
	}
	return false
}

// guvenlikPolitikasi ayarları okur.
func (s *Server) guvenlikPolitikasi(ctx context.Context) guvenlikPolitikasi {
	return guvenlikPolitikasi{
		Aktif:        s.ayarAcik(ctx, "guvenlik", "aktif", true),
		VPNEngel:     s.ayarAcik(ctx, "guvenlik", "vpn_engel", false),
		HostingEngel: s.ayarAcik(ctx, "guvenlik", "hosting_engel", true),
		BotEngel:     s.ayarAcik(ctx, "guvenlik", "bot_engel", true),
		WebRTC:       s.ayarAcik(ctx, "guvenlik", "webrtc", true),
	}
}

// tersDNSAdlari IP'nin PTR kayıtlarını kısa bütçeyle döndürür.
func tersDNSAdlari(r *http.Request, ip string) ([]string, error) {
	ctx, iptal := context.WithTimeout(r.Context(), 150*time.Millisecond)
	defer iptal()
	return net.DefaultResolver.LookupAddr(ctx, ip)
}

// isTorExit Tor çıkış listesini sorgular.
//
// NEDEN AYRI: Tor trafiği ASN olarak sıradan bir hosting'ten gelir
// (ör. 45.x). Çıkış listesi `check.torproject.org` üzerinden DNS ile
// alınabilir ama her istekte sorgulamak maliyetli. Bu yüzden:
//   - IP, ASN sınıfı "hosting" ise ayrıca Tor listesine bakılır
//   - Sonuç 10 dk bellek önbelleğinde tutulur.
func isTorExit(ip string) bool {
	return torOnbellek.al(ip)
}

type torListe struct {
	mu sync.Mutex
	m  map[string]torKayit
}

type torKayit struct {
	acik bool
	at   time.Time
}

var torOnbellek = &torListe{m: map[string]torKayit{}}

func (t *torListe) al(ip string) bool {
	t.mu.Lock()
	v, ok := t.m[ip]
	t.mu.Unlock()
	if ok && time.Since(v.at) < 10*time.Minute {
		return v.acik
	}
	// Sorgu: <ters>.<port>.exitlist.torproject.org — DNS A kaydı varsa çıkış.
	// Bilinen portlar 80/443. Yalnız 443 denenir (tek sorgu, en yaygın).
	ters := tersIP(ip)
	if ters == "" {
		return false
	}
	ctx, iptal := context.WithTimeout(context.Background(), 60*time.Millisecond)
	defer iptal()
	acik := false
	if _, err := net.DefaultResolver.LookupHost(ctx, ters+".443.exitlist.torproject.org"); err == nil {
		acik = true
	}
	t.mu.Lock()
	if len(t.m) > 5000 {
		t.m = map[string]torKayit{}
	}
	t.m[ip] = torKayit{acik: acik, at: time.Now()}
	t.mu.Unlock()
	return acik
}
