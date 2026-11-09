package httpapi

import (
	"crypto/sha256"
	"sync"
)

// cspHashCacheSayisi önbellekte tutulacak en fazla CSP başlığı.
//
// Her giriş ~1 KB (başlık metni). 512 giriş ≈ 0,5 MB; süreç ömrü boyunca
// sabit kalır. Sayfa çeşidi (bölüm + sayfa + makale) bundan çok daha az
// olduğu için pratikte tüm sıcak sayfalar önbellekte durur.
const cspHashCacheSayisi = 512

// cspHashCache gövde özetinden CSP başlığına giden süreç içi önbellek.
//
// NEDEN: `buildCSP` her HTML yanıtında gövdeyi iki kez tarıyor (script ve
// style blokları) ve bulduğu her blok için sha256 hesaplıyor. Profil
// (8 Kasım 2026, ana sayfa yükü): `indexFold` tek başına CPU'nun ~%9'u —
// yani istek yolunda sorgulardan sonra en pahalı saf-CPU işi.
//
// ANAHTAR GÖVDENİN KENDİSİ: CSP hash'leri yalnız gövdeye bağlı olduğu için
// (collab/rapor adresleri süreç başına sabit) aynı gövde → aynı başlık.
// Gövde özeti, taramadan çok daha ucuz: 80 KB için ~30 µs'ye karşı iki tam
// tarama + N adet sha256.
//
// YANLIŞ POZİTİF RİSKİ YOK: anahtar 256 bit özet; farklı gövde aynı anahtara
// düşerse CSP yanlış olurdu, ama bu olasılık kriptografik olarak yok sayılır
// (ikinci ön-kare saldırısı gerekir).
type cspHashCache struct {
	mu    sync.Mutex
	m     map[[32]byte]string
	sira  [][32]byte // FIFO tahliye kuyruğu
	limit int
}

func newCSPHashCache() *cspHashCache {
	return &cspHashCache{m: map[[32]byte]string{}, limit: cspHashCacheSayisi}
}

// Get gövde için önbellekteki başlığı döndürür.
func (c *cspHashCache) Get(govde string) (string, bool) {
	k := sha256.Sum256([]byte(govde))
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.m[k]
	return v, ok
}

// Set gövde → başlık eşlemesini saklar; sınır aşılırsa en eski giriş düşer.
func (c *cspHashCache) Set(govde, baslik string) {
	k := sha256.Sum256([]byte(govde))
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, var_ := c.m[k]; var_ {
		return // zaten var; kuyruğu şişirme
	}
	for len(c.sira) >= c.limit {
		eski := c.sira[0]
		c.sira = c.sira[1:]
		delete(c.m, eski)
	}
	c.m[k] = baslik
	c.sira = append(c.sira, k)
}

// Len testler için giriş sayısı.
func (c *cspHashCache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.m)
}
