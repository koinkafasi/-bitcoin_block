package httpapi

import (
	"crypto/sha256"
	"sync"
)

const cspHashCacheSayisi = 512

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
