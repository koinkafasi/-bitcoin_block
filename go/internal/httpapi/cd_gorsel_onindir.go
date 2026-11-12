package httpapi

import (
	"context"
	"time"

	"theblock/site/internal/repo"
)

// ---- Kaynak görsellerini önden indirme işçisi ----
//
// `/kg/<hash>` tembel çalışıyor: görsel ancak biri sayfayı açınca
// iniyor. Ölçüm (12 Kasım 2026): 4.918 kayıttan 136'sı diskte (%3).
// Kalan %97 uzak sunucuya bağlı ve o sunucu görseli taşıyınca ya da
// silince içerik kalıcı olarak görselsiz kalıyor — bu daha önce eski
// WordPress görsellerinde yaşandı (cdn.bitcosar.com artık 403).
//
// İşçi aralıklarla küçük gruplar indirir. Amacı hızlı bitirmek değil,
// sürekli ilerlemek: sunucuları boğmadan arşivi kendi diskimize almak.

const (
	// gorselOnIndirAralik: iki tur arası. Sık olması gerekmiyor —
	// kuyruk uzun ömürlü, acelesi olan görsel zaten istekte iniyor.
	gorselOnIndirAralik = 3 * time.Minute

	// gorselOnIndirGrup: tur başına görsel. Küçük tutuldu; büyük grup
	// tek turda çok sayıda dış isteğe yol açar ve kaynak sunucular hız
	// sınırına takılır (sonra o adresler "başarısız" sayılırdı).
	gorselOnIndirGrup = 12

	// gorselOnIndirBekleme: ardışık iki indirme arası nezaket aralığı.
	gorselOnIndirBekleme = 700 * time.Millisecond
)

// GorselOnIndiriciBaslat arka planda kaynak görsellerini indirir.
// Bağlam iptal edilince durur.
func (s *Server) GorselOnIndiriciBaslat(ctx context.Context) {
	go func() {
		// İlk tur için kısa gecikme: açılışta site zaten DB, şablon ve
		// zamanlayıcı kuruyor; dış indirme o yoğunluğa eklenmesin.
		select {
		case <-ctx.Done():
			return
		case <-time.After(30 * time.Second):
		}

		tik := time.NewTicker(gorselOnIndirAralik)
		defer tik.Stop()
		for {
			s.gorselOnIndirTur(ctx)
			select {
			case <-ctx.Done():
				return
			case <-tik.C:
			}
		}
	}()
}

// gorselOnIndirTur tek tur: en çok `gorselOnIndirGrup` görsel indirir.
func (s *Server) gorselOnIndirTur(ctx context.Context) {
	if s.pool == nil {
		return
	}
	liste, err := repo.IndirilecekGorseller(ctx, s.pool, gorselOnIndirGrup)
	if err != nil {
		s.logger.Warn("görsel ön indirme: kuyruk okunamadı", "error", err)
		return
	}
	if len(liste) == 0 {
		return
	}

	basarili := 0
	for _, g := range liste {
		select {
		case <-ctx.Done():
			return
		default:
		}

		b, ct, err := kaynakGorselIndirCtx(ctx, g.URL)
		if err != nil {
			// Sayaç artar; 5 denemeden sonra bu kayıt bırakılır.
			repo.GorselIndirmeBasarisiz(ctx, s.pool, g.Hash, err.Error())
		} else {
			kaynakGorselYaz(g.Hash, "", b)
			repo.KaynakGorselIsaretle(ctx, s.pool, g.Hash, ct, int64(len(b)))
			basarili++
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(gorselOnIndirBekleme):
		}
	}

	if basarili > 0 {
		ozet, _ := repo.GorselIndirmeDurumu(ctx, s.pool)
		s.logger.Info("görsel ön indirme turu",
			"indirilen", basarili, "deneme", len(liste),
			"toplam_indirilmis", ozet.Indirilmis, "bekleyen", ozet.Bekleyen)
	}
}
