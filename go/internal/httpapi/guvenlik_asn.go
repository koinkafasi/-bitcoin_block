package httpapi

import (
	"context"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"theblock/site/internal/repo"
)

// ASN / SAĞLAYICI SINIFLANDIRMASI — Team Cymru DNS.
//
// NEDEN BU KAYNAK: VPN/proxy tespitinde asıl soru "bu IP kimin?".
// Team Cymru'nun `origin.asn.cymru.com` TXT kaydı ücretsiz, anahtarsız ve
// kayıt defteri verisinden üretiliyor:
//
//	<ters-ip>.origin.asn.cymru.com → "15169 | 8.8.8.0/24 | US | arin | 2023-12"
//	AS<no>.asn.cymru.com           → "15169 | US | arin | 2000-03-30 | GOOGLE - Google LLC, US"
//
// ÖLÇÜLDÜ (14 Kasım 2026, bu makine): süreç içi `net.LookupTXT` ortanca
// **1.4 ms**, en kötü 234 ms (soğuk DNS). `dig` ile ölçüm yanıltıcıydı
// (190 ms) çünkü süreç başlatma maliyeti ölçüme karışıyordu.
//
// Bu yüzden: önbellek isabetsizliğinde **120 ms bütçe** ile eşzamanlı
// sorgu yapılır. Bütçe aşılırsa istek BEKLETİLMEZ; sınıflandırma UA/başlık
// katmanıyla yapılır ve zenginleştirme arka planda tamamlanır. Yani ilk
// istek "bilinmiyor" olabilir, sonraki istekler doğru sınıflanır. Bu
// bilinçli bir ödünleşim: sayfa hızını bir DNS sorgusuna feda etmiyoruz.

const (
	// asnButce tek istekte ASN için harcanabilecek en uzun süre.
	asnButce = 120 * time.Millisecond
	// asnTTL önbellek ömrü. ASN atamaları gün içinde değişmez.
	asnTTL = 6 * time.Hour
	// asnOnbellekSinir bellek üst sınırı (~50k /24 ≈ birkaç MB).
	asnOnbellekSinir = 50000
)

// asnBilgi tek bir IP aralığının sağlayıcı bilgisi.
type asnBilgi struct {
	ASN  int
	Ad   string // "HETZNER-AS - Hetzner Online GmbH, DE"
	Ulke string // AS kayıt ülkesi
	Tur  string // hosting|vpn|tor|mobil|kurumsal|konut|bilinmiyor
	Ag   string // "5.9.0.0/16"
	at   time.Time
}

type asnOnbellek struct {
	mu sync.Mutex
	m  map[string]asnBilgi
}

var asnCache = &asnOnbellek{m: map[string]asnBilgi{}}

func (c *asnOnbellek) al(k string) (asnBilgi, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.m[k]
	if !ok || time.Since(v.at) > asnTTL {
		return asnBilgi{}, false
	}
	return v, true
}

func (c *asnOnbellek) koy(k string, v asnBilgi) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.m) >= asnOnbellekSinir {
		// Fırsatçı temizlik: süresi dolmuşları at; hâlâ doluysa yarısını sil.
		for kk, vv := range c.m {
			if time.Since(vv.at) > asnTTL {
				delete(c.m, kk)
			}
		}
		if len(c.m) >= asnOnbellekSinir {
			n := 0
			for kk := range c.m {
				delete(c.m, kk)
				if n++; n > asnOnbellekSinir/2 {
					break
				}
			}
		}
	}
	c.m[k] = v
}

// asnArkaPlanKuyruk: bütçesi aşılan sorgular burada tamamlanır.
// Kanal doluysa istek DÜŞÜRÜLÜR — yavaş DNS yüzünden goroutine patlaması
// olmasın. Düşen sorgu bir sonraki istekte yeniden denenir.
var asnArkaPlanKuyruk = make(chan string, 64)

// asnAnahtar IP'yi önbellek anahtarına çevirir: IPv4'te /24, IPv6'da /48.
//
// NEDEN /24: Team Cymru öneği çoğu zaman /24'ten küçük değil (bir /16 tek
// sağlayıcı). /24 anahtarı, aynı sağlayıcının komşu bloklarını ayrı ayrı
// sorgulamamızı engellemez ama sıcak IP'leri tek girdide tutar. Gerçek önek
// (`Ag`) ayrıca saklanır ve ip_asn tablosuna yazılır.
func asnAnahtar(ip string) string {
	p := net.ParseIP(strings.TrimSpace(ip))
	if p == nil {
		return ""
	}
	if v4 := p.To4(); v4 != nil {
		return strconv.Itoa(int(v4[0])) + "." + strconv.Itoa(int(v4[1])) + "." +
			strconv.Itoa(int(v4[2])) + ".0/24"
	}
	// IPv6: ilk 3 grup (48 bit).
	h := p.String()
	parca := strings.Split(h, ":")
	for len(parca) < 3 {
		return h
	}
	return strings.Join(parca[:3], ":") + "::/48"
}

func tersIP(ip string) string {
	p := net.ParseIP(strings.TrimSpace(ip))
	if p == nil {
		return ""
	}
	if v4 := p.To4(); v4 != nil {
		return strconv.Itoa(int(v4[3])) + "." + strconv.Itoa(int(v4[2])) + "." +
			strconv.Itoa(int(v4[1])) + "." + strconv.Itoa(int(v4[0]))
	}
	// IPv6 ters çevirme (nibble): ptr.ip6.arpa
	full := p.String()
	if i := strings.IndexByte(full, ':'); i > 0 && strings.Contains(full, "::") {
		// Basitleştirilmiş: kütüphane yok; yalnız tam yazılmış IPv6 denenir.
		return ""
	}
	hex := strings.ReplaceAll(strings.TrimRight(full, ":"), ":", "")
	if len(hex) != 32 {
		return ""
	}
	var b strings.Builder
	for i := len(hex) - 1; i >= 0; i-- {
		b.WriteByte(hex[i])
		b.WriteByte('.')
	}
	b.WriteString("ip6.arpa")
	return b.String()
}

// asnCoz Team Cymru TXT cevabını ayrıştırır. Saf fonksiyon: test edilebilir.
//
// Biçim: "15169 | 8.8.8.0/24 | US | arin | 2023-12-01"
func asnCoz(txt string) (asn int, ag, ulke string, ok bool) {
	parca := strings.Split(txt, "|")
	if len(parca) < 2 {
		return 0, "", "", false
	}
	n, err := strconv.Atoi(strings.TrimSpace(parca[0]))
	if err != nil || n <= 0 {
		return 0, "", "", false
	}
	ag = strings.TrimSpace(parca[1])
	if len(parca) >= 3 {
		ulke = strings.TrimSpace(parca[2])
	}
	return n, ag, ulke, true
}

// asnAdCoz AS adı cevabını ayrıştırır.
//
// Biçim: "15169 | US | arin | 2000-03-30 | GOOGLE - Google LLC, US"
func asnAdCoz(txt string) string {
	parca := strings.Split(txt, "|")
	if len(parca) < 5 {
		return ""
	}
	return strings.TrimSpace(parca[len(parca)-1])
}

// asnSorgula IP'nin ASN bilgisini döndürür. `tamam` false ise bilgi yok
// (bütçe aşıldı, özel IP, DNS hatası). Çağıran FAIL-OPEN davranmalı.
func (s *Server) asnSorgula(ctx context.Context, ip string) (asnBilgi, bool) {
	if isPrivateIP(ip) {
		return asnBilgi{}, false
	}
	anahtar := asnAnahtar(ip)
	if anahtar == "" {
		return asnBilgi{}, false
	}
	if v, ok := asnCache.al(anahtar); ok {
		return v, true
	}

	sctx, iptal := context.WithTimeout(ctx, asnButce)
	defer iptal()

	ters := tersIP(ip)
	if ters == "" {
		return asnBilgi{}, false
	}
	txts, err := net.DefaultResolver.LookupTXT(sctx, ters+".origin.asn.cymru.com")
	if err != nil || len(txts) == 0 {
		if sctx.Err() != nil {
			s.asnArkaPlanaAt(ip)
		}
		return asnBilgi{}, false
	}
	asn, ag, ulke, ok := asnCoz(txts[0])
	if !ok {
		return asnBilgi{}, false
	}

	ad := ""
	if ntxts, err := net.DefaultResolver.LookupTXT(sctx, "AS"+strconv.Itoa(asn)+".asn.cymru.com"); err == nil && len(ntxts) > 0 {
		ad = asnAdCoz(ntxts[0])
	}

	b := asnBilgi{ASN: asn, Ad: ad, Ulke: ulke, Ag: ag, Tur: asnTur(ad), at: time.Now()}
	asnCache.koy(anahtar, b)
	if ag != "" {
		asnCache.koy(ag, b)
	}
	// Kalıcı önbellek: süreç yeniden başlarsa soğuk başlamasın. Yazma
	// isteği bekletmez (arka plan); tablo anahtar başına tek satır.
	s.asnDBYaz(b)
	return b, true
}

// asnArkaPlanaAt bütçesi aşılan IP'yi kuyruğa koyar (engellemeden).
func (s *Server) asnArkaPlanaAt(ip string) {
	select {
	case asnArkaPlanKuyruk <- ip:
	default:
		// Kuyruk dolu: bu IP bu turda sınıflandırılmaz. Sonraki istek dener.
	}
}

// ASNArkaPlanBaslat kuyruğu işleyen tek bir işçi başlatır.
//
// NEDEN TEK İŞÇİ: DNS sorgusu zaten milisaniyeler sürüyor; 64'lük kuyruk
// tek işçiyle saniyede yüzlerce IP'yi tamamlar. Çok işçi, aynı /24 için
// yarış (ve gereksiz sorgu) üretirdi.
func (s *Server) ASNArkaPlanBaslat(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case ip := <-asnArkaPlanKuyruk:
				// İstek bağlamından bağımsız: kapanışta da çalışsın.
				bctx, iptal := context.WithTimeout(context.Background(), 2*time.Second)
				s.asnSorgulaZorla(bctx, ip)
				iptal()
			}
		}
	}()
}

// asnSorgulaZorla önbelleği atlayarak sorgular (arka plan yolu).
func (s *Server) asnSorgulaZorla(ctx context.Context, ip string) {
	anahtar := asnAnahtar(ip)
	if anahtar == "" {
		return
	}
	if _, ok := asnCache.al(anahtar); ok {
		return
	}
	ters := tersIP(ip)
	if ters == "" {
		return
	}
	txts, err := net.DefaultResolver.LookupTXT(ctx, ters+".origin.asn.cymru.com")
	if err != nil || len(txts) == 0 {
		return
	}
	asn, ag, ulke, ok := asnCoz(txts[0])
	if !ok {
		return
	}
	ad := ""
	if ntxts, err := net.DefaultResolver.LookupTXT(ctx, "AS"+strconv.Itoa(asn)+".asn.cymru.com"); err == nil && len(ntxts) > 0 {
		ad = asnAdCoz(ntxts[0])
	}
	b := asnBilgi{ASN: asn, Ad: ad, Ulke: ulke, Ag: ag, Tur: asnTur(ad), at: time.Now()}
	asnCache.koy(anahtar, b)
	if ag != "" {
		asnCache.koy(ag, b)
	}
	s.asnDBYaz(b)
}

// asnDBYaz kalıcı önbelleğe yazar (istek yolunu bekletmez).
func (s *Server) asnDBYaz(b asnBilgi) {
	if s.pool == nil || b.Ag == "" {
		return
	}
	go func() {
		ctx, iptal := context.WithTimeout(context.Background(), 3*time.Second)
		defer iptal()
		if err := repo.ASNKaydet(ctx, s.pool, repo.ASNSatir{
			Ag: b.Ag, ASN: b.ASN, Ad: b.Ad, Ulke: b.Ulke, Tur: b.Tur, Kaynak: "cymru",
		}); err != nil {
			s.logger.Debug("asn önbelleği yazılamadı", "error", err)
		}
	}()
}

// ASNIsit açılışta kalıcı önbelleği belleğe yükler.
func (s *Server) ASNIsit(ctx context.Context) {
	satirlar, err := repo.ASNListele(ctx, s.pool, 2000)
	if err != nil {
		s.logger.Warn("asn ön ısıtma", "error", err)
		return
	}
	for _, a := range satirlar {
		asnCache.koy(a.Ag, asnBilgi{
			ASN: a.ASN, Ad: a.Ad, Ulke: a.Ulke, Tur: a.Tur, Ag: a.Ag,
			at: time.Now(),
		})
	}
	if len(satirlar) > 0 {
		s.logger.Info("asn önbelleği ısıtıldı", "adet", len(satirlar))
	}
}

// ---- Sağlayıcı sınıflandırması ----

// vpnAnahtar: ticari VPN / gizlilik ağları. ARKASINDA GERÇEK İNSAN olabilir;
// bu yüzden "engelle" değil "sayma + işaretle" varsayılanı.
var vpnAnahtar = []string{
	"TEFINCOM", "NORDVPN", "SECFIREWALL", "MULLVAD", "31173 SERVICES",
	"LONDON TRUST MEDIA", "PRIVATE INTERNET ACCESS", "IPVANISH",
	"ZENLAYER", "PACKET HUB", "DATACAMP", "CDN77", "SURFSHARK",
	"EXPRESSVPN", "EXPRESSTECH", "CYBERGHOST", "PUREVPN", "STRONGVPN",
	"ATLAS VPN", "HOTSPOT SHIELD", "ANCHORFREE", "TUNNELBEAR", "MCFAITH",
	"PROTON AG", "PROTONVPN", "WINDSCRIBE", "WYPRSTREAM", "VYPRVPN",
	"HIDEMYASS", "AVAST SOFTWARE", "PIA", "PERFECT PRIVACY",
	"CLOUDFLARENET", "WARP", "OVPN", "AZIREVPN", "IVACY", "PURE",
	"SPEEDIFY", "HOLA", "BRIGHT DATA", "LUMINATI", "OXYSQL", "GEONODE",
	"SMARTPROXY", "PROXYSCRAPE", "WEBSHARE", "IPROYAL",
	// M247 bir barındırma sağlayıcısı ama müşterisi neredeyse tamamen VPN
	// satıcıları: hosting saymak puanı 45'te bırakıp gerçek VPN trafiğini
	// hafife alırdı. Ölçüldü: 45.83.89.1 → "M247 Europe SRL".
	"M247",
}

// hostingAnahtar: veri merkezi/bulut. Gerçek okuyucu neredeyse hiç gelmez.
var hostingAnahtar = []string{
	"HETZNER", "DIGITALOCEAN", "AMAZON", "AMAZON-AES", "AMAZON-02",
	"OVH", "LINODE", "AKAMAI", "VULTR", "CHOOPA", "CONTABO", "SCALEWAY",
	"ONLINE S.A.S", "LEASE WEB", "LEASEWEB", "KAMATERA", "INTERSERVER",
	"HOSTINGER", "GODADDY", "NAME-CHEAP", "ALIBABA", "ALICLOUD",
	"TENCENT", "HUAWEI CLOUD", "SOFTLAYER", "IBM CLOUD", "ORACLE",
	"MICROSOFT", "AZURE", "GOOGLE CLOUD", "GOOGLE-", "GCP",
	"COLO", "DATACENTER", "DATA CENTER", "HOSTING", "SERVER", "CLOUD",
	"VPN", "PROXY", "BULLETPROOF", "UFO-TECHNOLOGIES",
	"ESAB", "ZSCALER", "NEXUSGUARD", "INCAPSULA", "IMPERVA",
	"FASTLY", "STACKPATH", "NETLIFY", "VERCEL", "HEROKU", "RENDER",
}

// mobilAnahtar: mobil operatörler. Yalnız bilgi amaçlı sınıf; puan düşürmez.
var mobilAnahtar = []string{
	"MOBILE", "CELL", "CELLULAR", "WIRELESS", "TURKCELL", "VODAFONE",
	"TURK TELEKOM", "TTNET", "SUPERONLINE", "TELEFONICA", "ORANGE",
	"T-MOBILE", "VODACOM", "AIRTEL", "MTN", "ETISALAT", "ZAIN",
	"SAUDI TELECOM", "STC", "BELL CANADA", "ROGERS", "TELUS",
	"VERIZON", "AT&T", "SPRINT", "COMCAST", "CHARTER", "COX",
}

// asnTur AS adından sağlayıcı sınıfı çıkarır. Saf fonksiyon.
//
// SIRA ÖNEMLİ: VPN anahtarları önce denenir. M247 hem hosting hem VPN
// satıyor; VPN sayılması daha doğru çünkü müşterisi çoğunlukla VPN.
func asnTur(ad string) string {
	u := strings.ToUpper(strings.TrimSpace(ad))
	if u == "" {
		return "bilinmiyor"
	}
	for _, k := range vpnAnahtar {
		if strings.Contains(u, k) {
			return "vpn"
		}
	}
	for _, k := range hostingAnahtar {
		if strings.Contains(u, k) {
			return "hosting"
		}
	}
	for _, k := range mobilAnahtar {
		if strings.Contains(u, k) {
			return "mobil"
		}
	}
	return "konut"
}

// asnSaglayiciEtiket panelde gösterilecek kısa ad.
func asnSaglayiciEtiket(ad string) string {
	ad = strings.TrimSpace(ad)
	if ad == "" {
		return ""
	}
	// "15169 | ..." biçiminde gelirse son parçayı al.
	if i := strings.LastIndex(ad, "|"); i >= 0 {
		ad = strings.TrimSpace(ad[i+1:])
	}
	// " - " ile ayrılmışsa marka kısmı önce gelir.
	if i := strings.Index(ad, " - "); i > 0 {
		ad = ad[:i]
	}
	if i := strings.Index(ad, ","); i > 0 {
		ad = ad[:i]
	}
	if len(ad) > 48 {
		ad = ad[:48]
	}
	return strings.TrimSpace(ad)
}
