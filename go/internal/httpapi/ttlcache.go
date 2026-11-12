package httpapi

import (
	"context"
	"sync"
	"time"
)

type ttlcache[T any] struct {
	ttl time.Duration

	mu    sync.Mutex
	entry map[string]ttlEntry[T]
	// calls, aynı ana için uçuştaki yüklemeleri tutar.
	calls map[string]*ttlCall[T]
}

type ttlEntry[T any] struct {
	at  time.Time
	val T
	ok  bool
}

type ttlCall[T any] struct {
	done chan struct{}
	val  T
	ok   bool
}

func newTTLCache[T any](ttl time.Duration) *ttlcache[T] {
	return &ttlcache[T]{
		ttl:   ttl,
		entry: map[string]ttlEntry[T]{},
		calls: map[string]*ttlCall[T]{},
	}
}

// Get anahtar için taze değer varsa döndürür.
func (c *ttlcache[T]) Get(key string) (T, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.entry[key]; ok && e.ok && time.Since(e.at) < c.ttl {
		return e.val, true
	}
	var zero T
	return zero, false
}

// Set değeri yazar (yükleyici dışından elle tazeleme için).
func (c *ttlcache[T]) Set(key string, v T) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entry[key] = ttlEntry[T]{at: time.Now(), val: v, ok: true}
}

// Clear önbelleği boşaltır (ayar/içerik değişince çağrılır).
func (c *ttlcache[T]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entry = map[string]ttlEntry[T]{}
}

func (c *ttlcache[T]) Load(ctx context.Context, key string, load func(context.Context) (T, bool)) (T, bool) {
	if v, ok := c.Get(key); ok {
		return v, true
	}

	c.mu.Lock()
	// Bu arada başka bir çağıran doldurmuş olabilir.
	if e, ok := c.entry[key]; ok && e.ok && time.Since(e.at) < c.ttl {
		v := e.val
		c.mu.Unlock()
		return v, true
	}
	if cl, exists := c.calls[key]; exists {
		c.mu.Unlock()
		select {
		case <-cl.done:
			return cl.val, cl.ok
		case <-ctx.Done():
			var zero T
			return zero, false
		}
	}
	cl := &ttlCall[T]{done: make(chan struct{})}
	c.calls[key] = cl
	c.mu.Unlock()

	// Yükleme kilidin DIŞINDA yapılır: uzun sorgu diğer anahtarları bloklamaz.
	v, ok := load(ctx)

	c.mu.Lock()
	delete(c.calls, key)
	if ok {
		c.entry[key] = ttlEntry[T]{at: time.Now(), val: v, ok: true}
	}
	c.mu.Unlock()

	cl.val, cl.ok = v, ok
	close(cl.done)
	return v, ok
}
