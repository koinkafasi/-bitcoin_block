// YouTube senkronizasyon işi.
//
// NEDEN SCHEDULER: sayfalar YouTube'u HER istekte çağırmaz (gecikme + dış
// servis bağımlılığı). Kanal RSS'i + /streams sayfası periyodik çekilir,
// `youtube_icerik` tablosuna yazılır (migrations/0122). Yeni video düştüğünde
// bir sonraki turda tablo güncellenir, sayfa kendiliğinden tazelenir.
package scheduler

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"theblock/site/internal/youtube"
)

// RunYoutube ayar aralığına göre YouTube kanalını çeker.
//
// `every` yalnız YOKLAMA sıklığıdır; gerçek çekme aralığı panelden
// (settings/youtube.dakika) gelir. Böylece panelden "5 dakika" seçilince
// süreç yeniden başlatılmadan uygulanır.
func RunYoutube(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger, every time.Duration) {
	if every <= 0 {
		every = time.Minute
	}
	var son time.Time
	for {
		if ctx.Err() != nil {
			return
		}
		ayar := youtube.AyarlariOku(ctx, pool)
		bekle := time.Duration(ayar.Dakika) * time.Minute
		if bekle <= 0 {
			bekle = 15 * time.Minute
		}
		// İlk turda hemen çek: yeni kurulumda sayfa boş kalmasın.
		if son.IsZero() || time.Since(son) >= bekle {
			if _, err := youtube.Senkron(ctx, pool, logger); err != nil {
				logger.Warn("youtube senkron hatası", "error", err)
			}
			son = time.Now()
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(every):
		}
	}
}
