package httpapi

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"theblock/site/internal/repo"
)

const okunmaAraligi = 2 * time.Second

// okunmaBiriktirici içerik başına okunma sayısını bellekte biriktirir.
type okunmaBiriktirici struct {
	mu sync.Mutex
	m  map[int64]int64
}

func newOkunmaBiriktirici() *okunmaBiriktirici {
	return &okunmaBiriktirici{m: map[int64]int64{}}
}

func (b *okunmaBiriktirici) ekle(id int64) {
	if id <= 0 {
		return
	}
	b.mu.Lock()
	b.m[id]++
	b.mu.Unlock()
}

// al birikeni boşaltıp döndürür (yazım için). Boşsa nil.
func (b *okunmaBiriktirici) al() map[int64]int64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.m) == 0 {
		return nil
	}
	m := b.m
	b.m = make(map[int64]int64, len(m))
	return m
}

func (b *okunmaBiriktirici) boyut() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.m)
}

var okunmaTamponu = newOkunmaBiriktirici()

// okunmaIsaretAnahtari istek bağlamına konan sayaç işaretinin anahtarı.
type okunmaIsaretAnahtari struct{}

// okunmaSaymaAnahtari : "bu istekte okunma sayılmasın" (arka plan yenilemesi).
type okunmaSaymaAnahtari struct{}

// okunmaIsareti handler'ın bildirdiği içerik kimliğini taşır.
type okunmaIsareti struct {
	mu sync.Mutex
	id int64
}

func (o *okunmaIsareti) al() int64 {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.id
}

func (o *okunmaIsareti) koy(id int64) {
	o.mu.Lock()
	if o.id == 0 {
		o.id = id
	}
	o.mu.Unlock()
}

func okunmayiIsaretle(ctx context.Context, id int64) bool {
	v, ok := ctx.Value(okunmaIsaretAnahtari{}).(*okunmaIsareti)
	if !ok || v == nil || id <= 0 {
		return false
	}
	v.koy(id)
	return true
}

func okunmaSayilmasin(ctx context.Context) context.Context {
	return context.WithValue(ctx, okunmaSaymaAnahtari{}, true)
}

// okunmakaydet handler'ın tek çağrı noktası.
func (s *Server) okunmaKaydet(ctx context.Context, id int64) {
	if ctx.Value(okunmaSaymaAnahtari{}) != nil {
		return // arka plan yenilemesi: kullanıcı okuması değil
	}
	if !okunmaSayilirCtx(ctx) {
		return // güvenlik kararı: bot/VPN/şüpheli → sayılmaz
	}
	if okunmayiIsaretle(ctx, id) {
		return // sayfa önbelleği katmanı sayacak
	}
	okunmaTamponu.ekle(id)
}

// okunmaSayilirCtx okunmanın sayılıp sayılmayacağını güvenlik kararından
// okur. Karar yoksa (güvenlik katmanı kapalı) SAYILIR — eski davranış.
//
// NEDEN CTX: okunmaKaydet yalnız bağlam alıyor; karar middleware tarafından
// aynı bağlama konuyor. Ayrı bir parametre geçirmek tüm çağrı zincirini
// değiştirirdi.
func okunmaSayilirCtx(ctx context.Context) bool {
	k, ok := ctx.Value(guvenlikKarariAnahtari{}).(*guvenlikKarari)
	if !ok || k == nil {
		return true
	}
	return k.Sayilir
}

func OkunmaSayaciBaslat(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger, aralik time.Duration) {
	if aralik <= 0 {
		aralik = okunmaAraligi
	}
	yaz := func() {
		m := okunmaTamponu.al()
		if len(m) == 0 {
			return
		}
		// İstek bağlamına bağlanmaz: kapanışta da yazılmalı.
		yctx, iptal := context.WithTimeout(context.Background(), 5*time.Second)
		defer iptal()
		if err := repo.IncrementHits(yctx, pool, m); err != nil {
			// Kaybedilen sayıyı geri koyma: yazım hatası kalıcıysa bellek
			// şişerdi. Loglanır, sonraki turda devam edilir.
			if logger != nil {
				logger.Warn("okunma sayacı yazılamadı", "içerik", len(m), "error", err)
			}
		}
	}

	go func() {
		t := time.NewTicker(aralik)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				yaz()
				return
			case <-t.C:
				yaz()
			}
		}
	}()
}

// okunmaSayaciDurdur testler için: birikeni yazmadan boşaltır.
func okunmaSayaciBosalt() map[int64]int64 { return okunmaTamponu.al() }
