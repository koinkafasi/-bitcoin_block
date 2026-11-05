package httpapi

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ---- Kimlik doğrulama hız sınırları ----
//
// Panel girişinde (/admin/giris) hiç hız sınırı yoktu: 8+ karakterli her
// parola sınırsız denenebiliyordu. Okuyucu girişinde ise yalnız e-posta
// bazlı sayaç vardı; bu da bilinen bir e-postayı 5 hatalı denemeyle
// kilitleyerek hizmet engellemeye (DoS) izin veriyordu.
//
// Bu sayaç süreç içidir (yeniden başlatmada sıfırlanır) ve yalnız
// BAŞARISIZ denemeleri sayar:
//   - IP başına genel üst sınır → tek kaynaktan toplu deneme,
//   - (IP + hesap) çifti başına sınır → tek hesaba hedefli deneme.
// Başarılı giriş sayaçları temizler; meşru kullanıcı yanlış parola yazdığında
// kendi IP'sini kalıcı olarak kilitlemez.

type denemeSayaci struct {
	n   int
	ilk time.Time
}

var authDenemeleri = struct {
	mu sync.Mutex
	m  map[string]*denemeSayaci
}{m: map[string]*denemeSayaci{}}

const (
	// authPencere sayaçların kaydığı zaman aralığı.
	authPencere = 10 * time.Minute
	// authIPAzami tek IP'den 10 dakikada kabul edilen başarısız deneme.
	authIPAzami = 30
	// authHesapAzami tek (IP + hesap) çifti için başarısız deneme.
	authHesapAzami = 6
)

// authParcala anahtarı pencereye göre sıfırlayıp sayaç döndürür.
// Kilit çağıran tarafta tutulur.
func authParcala(key string, simdi time.Time) *denemeSayaci {
	c := authDenemeleri.m[key]
	if c == nil || simdi.Sub(c.ilk) > authPencere {
		c = &denemeSayaci{ilk: simdi}
		authDenemeleri.m[key] = c
	}
	return c
}

// authIPIzinli IP'nin genel sınırı aşıp aşmadığını söyler.
func authIPIzinli(ip string) bool {
	simdi := time.Now()
	authDenemeleri.mu.Lock()
	defer authDenemeleri.mu.Unlock()
	c := authParcala("ip:"+ip, simdi)
	return c.n < authIPAzami
}

// authHesapIzinli (IP + hesap) çiftinin sınırını denetler.
func authHesapIzinli(ip, hesap string) bool {
	simdi := time.Now()
	authDenemeleri.mu.Lock()
	defer authDenemeleri.mu.Unlock()
	c := authParcala("hesap:"+ip+"|"+strings.ToLower(hesap), simdi)
	return c.n < authHesapAzami
}

// authBasarisiz bir başarısız denemeyi iki sayaca da yazar.
func authBasarisiz(ip, hesap string) {
	simdi := time.Now()
	authDenemeleri.mu.Lock()
	defer authDenemeleri.mu.Unlock()
	authParcala("ip:"+ip, simdi).n++
	if hesap != "" {
		authParcala("hesap:"+ip+"|"+strings.ToLower(hesap), simdi).n++
	}
	// Haritanın sınırsız büyümesini önle.
	if len(authDenemeleri.m) > 5000 {
		for k, v := range authDenemeleri.m {
			if simdi.Sub(v.ilk) > authPencere {
				delete(authDenemeleri.m, k)
			}
		}
	}
}

// authBasarili başarılı girişte ilgili sayaçları temizler.
func authBasarili(ip, hesap string) {
	authDenemeleri.mu.Lock()
	defer authDenemeleri.mu.Unlock()
	delete(authDenemeleri.m, "ip:"+ip)
	if hesap != "" {
		delete(authDenemeleri.m, "hesap:"+ip+"|"+strings.ToLower(hesap))
	}
}

// authDenemeReddi kullanıcıya dönen ortak ileti.
const authDenemeReddi = "Çok fazla başarısız deneme. Lütfen 10 dakika sonra tekrar deneyin."

// ---- Kayıt hız sınırı ----

const (
	kayitPencere = time.Hour
	kayitAzami   = 5
)

// kayitIzinli aynı IP'den saatte en fazla kayitAzami hesap açılmasına izin verir.
func kayitIzinli(ip string) bool {
	simdi := time.Now()
	authDenemeleri.mu.Lock()
	defer authDenemeleri.mu.Unlock()
	k := "kayit:" + ip
	c := authDenemeleri.m[k]
	if c == nil || simdi.Sub(c.ilk) > kayitPencere {
		return true
	}
	return c.n < kayitAzami
}

// kayitSay başarılı/başarısız kayıt denemesini sayar.
func kayitSay(ip string) {
	simdi := time.Now()
	authDenemeleri.mu.Lock()
	defer authDenemeleri.mu.Unlock()
	k := "kayit:" + ip
	c := authDenemeleri.m[k]
	if c == nil || simdi.Sub(c.ilk) > kayitPencere {
		authDenemeleri.m[k] = &denemeSayaci{n: 1, ilk: simdi}
		return
	}
	c.n++
}

// ---- İstemci IP'si ----

// clientIP istemci IP'sini döndürür.
//
// Cloudflare/tünel başlıklarına YALNIZ istek yerel/özel bir adresten
// (127.0.0.1, 10.x, 172.16-31.x, 192.168.x) geldiğinde güvenilir. Aksi hâlde
// doğrudan bağlanan bir istemci X-Forwarded-For uydurup hız sınırlarını ve
// bot tuzağını atlatabilirdi.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if isPrivateIP(host) {
		if cf := strings.TrimSpace(r.Header.Get("CF-Connecting-IP")); cf != "" {
			return cf
		}
		if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
			if i := strings.IndexByte(xff, ','); i > 0 {
				return strings.TrimSpace(xff[:i])
			}
			return xff
		}
	}
	return host
}
