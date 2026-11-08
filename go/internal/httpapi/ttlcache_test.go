package httpapi

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestTTLCacheHitYukleyiciyiTekrarCagirmaz: taze değer varken yükleyici
// çalışmaz. Sayfa başına onlarca DB turunu bu davranış kaldırıyor.
func TestTTLCacheHitYukleyiciyiTekrarCagirmaz(t *testing.T) {
	c := newTTLCache[int](time.Minute)
	ctx := context.Background()
	var n int32
	load := func(context.Context) (int, bool) {
		atomic.AddInt32(&n, 1)
		return 42, true
	}

	v1, ok1 := c.Load(ctx, "k", load)
	v2, ok2 := c.Load(ctx, "k", load)

	if !ok1 || !ok2 || v1 != 42 || v2 != 42 {
		t.Fatalf("değer/ok beklenmedik: %v %v %v %v", v1, ok1, v2, ok2)
	}
	if got := atomic.LoadInt32(&n); got != 1 {
		t.Fatalf("yükleyici %d kez çağrıldı, 1 beklenirdi", got)
	}
}

// TestTTLCacheSuresiDoluncaYenidenYukler: TTL geçince tazelenir.
func TestTTLCacheSuresiDoluncaYenidenYukler(t *testing.T) {
	c := newTTLCache[int](10 * time.Millisecond)
	ctx := context.Background()
	var n int32
	load := func(context.Context) (int, bool) {
		return int(atomic.AddInt32(&n, 1)), true
	}
	if v, _ := c.Load(ctx, "k", load); v != 1 {
		t.Fatalf("ilk değer = %d", v)
	}
	time.Sleep(20 * time.Millisecond)
	if v, _ := c.Load(ctx, "k", load); v != 2 {
		t.Fatalf("TTL sonrası değer = %d, 2 beklenirdi", v)
	}
}

// TestTTLCacheBasarisizSonucOnbellegeYazilmaz: DB hatasında (ok=false)
// değer saklanmaz — geçici hata 10 sn boyunca boş sayfa olarak donmasın.
func TestTTLCacheBasarisizSonucOnbellegeYazilmaz(t *testing.T) {
	c := newTTLCache[int](time.Minute)
	ctx := context.Background()
	var n int32
	fail := func(context.Context) (int, bool) {
		atomic.AddInt32(&n, 1)
		return 0, false
	}
	if _, ok := c.Load(ctx, "k", fail); ok {
		t.Fatal("başarısız yükleme ok=true döndü")
	}
	if _, ok := c.Load(ctx, "k", fail); ok {
		t.Fatal("başarısız yükleme ok=true döndü")
	}
	if got := atomic.LoadInt32(&n); got != 2 {
		t.Fatalf("başarısız sonuç önbelleğe yazılmış: yükleyici %d kez, 2 beklenirdi", got)
	}
}

// TestTTLCacheSingleflight: aynı ana için eşzamanlı çağrılar TEK yükleme
// yapar. Yük testinde (50 eşzamanlı) önbellek boşken oluşan sürüyü bu
// engelliyor.
func TestTTLCacheSingleflight(t *testing.T) {
	c := newTTLCache[int](time.Minute)
	ctx := context.Background()
	var n int32
	release := make(chan struct{})
	load := func(context.Context) (int, bool) {
		atomic.AddInt32(&n, 1)
		<-release // tüm çağıranlar beklemedeyken bitir
		return 7, true
	}

	const N = 20
	var wg sync.WaitGroup
	got := make([]int, N)
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func(i int) {
			defer wg.Done()
			v, _ := c.Load(ctx, "k", load)
			got[i] = v
		}(i)
	}
	// Yükleyicinin başlamasını bekle, sonra serbest bırak.
	time.Sleep(50 * time.Millisecond)
	close(release)
	wg.Wait()

	if n := atomic.LoadInt32(&n); n != 1 {
		t.Fatalf("yükleyici %d kez çağrıldı, 1 beklenirdi (singleflight)", n)
	}
	for i, v := range got {
		if v != 7 {
			t.Fatalf("got[%d] = %d", i, v)
		}
	}
}

// TestTTLCacheIptalBeklemeyiBiraktirir: bekleyen çağıranın bağlamı iptal
// olursa sonsuza kadar beklemez.
func TestTTLCacheIptalBeklemeyiBiraktirir(t *testing.T) {
	c := newTTLCache[int](time.Minute)
	release := make(chan struct{})
	defer close(release)
	load := func(context.Context) (int, bool) {
		<-release
		return 1, true
	}

	// Lider: arka planda yüklüyor.
	go func() { _, _ = c.Load(context.Background(), "k", load) }()
	time.Sleep(30 * time.Millisecond)

	ctx, iptal := context.WithCancel(context.Background())
	iptal()
	bas := time.Now()
	if _, ok := c.Load(ctx, "k", load); ok {
		t.Fatal("iptal edilmiş bağlam ok=true dönmemeli")
	}
	if time.Since(bas) > time.Second {
		t.Fatal("iptal edilmiş bağlam bekledi")
	}
}
