package httpapi

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

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
