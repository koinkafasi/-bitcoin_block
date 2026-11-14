package httpapi

import (
	"context"
	"time"

	"theblock/site/internal/repo"
)

// ZİYARETÇİ KAYITLARI — SAKLAMA SÜRESİ.
//
// Aydınlatma metni (/kvkk) ve çerez politikası, ziyaretçi konum kaydının
// (visitor_locations) ve trafik güvenliği kaydının (visitor_risk) son
// ziyaretten itibaren 12 ay tutulduğunu bildiriyor. Süre yalnız metinde
// kalmasın diye burada uygulanıyor: günde bir kez süresi dolan satırlar
// silinir. Süre değişirse METİNLER DE güncellenmeli (migration 0132).
const (
	konumSaklamaGun      = 365
	konumTemizlikAraligi = 24 * time.Hour
)

// KonumTemizlikBaslat süresi dolan ziyaretçi kayıtlarını siler: açılışta
// bir kez, sonra günde bir.
func (s *Server) KonumTemizlikBaslat(ctx context.Context) {
	sil := func() {
		sctx, iptal := context.WithTimeout(ctx, 30*time.Second)
		defer iptal()
		konum, err := repo.ZiyaretciEskiSil(sctx, s.pool, konumSaklamaGun)
		if err != nil {
			s.logger.Warn("ziyaretçi konumu saklama temizliği", "error", err)
		}
		risk, err := repo.TrafikRiskEskiSil(sctx, s.pool, konumSaklamaGun)
		if err != nil {
			s.logger.Warn("trafik güvenliği saklama temizliği", "error", err)
		}
		if konum+risk > 0 {
			s.logger.Info("ziyaretçi kayıtları: süresi dolanlar silindi", "konum", konum, "risk", risk)
		}
	}
	go func() {
		sil()
		t := time.NewTicker(konumTemizlikAraligi)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				sil()
			}
		}
	}()
}
