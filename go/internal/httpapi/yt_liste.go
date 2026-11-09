package httpapi

import (
	"context"
	"time"

	"theblock/site/internal/repo"
)

// ytListeTTL YouTube listelerinin süreç içi ömrü.
//
// NEDEN: ana sayfa render'ı AYNI listeyi farklı modüller için 6 kez
// çekiyordu (video · short · stream × ana sayfa + "Explore"). Ölçüm
// (10 Kasım 2026, soğuk ana sayfa): 29 sorgunun 6'sı bu tabloya gidiyordu,
// oysa yalnız 3 ayrı tür var. Tablo scheduler tarafından dakikalar
// mertebesinde güncellendiği için 10 sn'lik önbellek tazelik kaybı
// yaratmaz; sayfa yanıtı zaten `max-age=60, s-maxage=300`.
const ytListeTTL = 10 * time.Second

// ytListeTavan önbellekte tutulan en uzun liste. Modüller 3/5/6/12 satır
// isteyebiliyor; en uzunu bir kez çekilip dilimlenir. Böylece aynı tür için
// farklı `LIMIT` değerleri ayrı önbellek girdisi ÜRETMEZ.
const ytListeTavan = 12

// ytListe tür bazında YouTube listesini döndürür (önbellekli).
//
// Hata ÖNBELLEĞE YAZILMAZ: geçici DB hatası 10 sn boyunca boş modül olarak
// donmasın, sonraki istek yeniden dener (ttlcache `ok=false` sözleşmesi).
func (s *Server) ytListe(ctx context.Context, tur string) []repo.YoutubeKayit {
	if s.ytListeCache == nil {
		kayitlar, err := repo.YoutubeListele(ctx, s.pool, tur, ytListeTavan)
		if err != nil {
			return nil
		}
		return kayitlar
	}
	v, ok := s.ytListeCache.Load(ctx, tur, func(ctx context.Context) ([]repo.YoutubeKayit, bool) {
		kayitlar, err := repo.YoutubeListele(ctx, s.pool, tur, ytListeTavan)
		if err != nil {
			s.logger.Warn("youtube listesi", "tur", tur, "error", err)
			return nil, false
		}
		return kayitlar, true
	})
	if !ok {
		return nil
	}
	return v
}

// ytListeDilim önbellekten gelen listeyi istenen uzunlukta keser.
func ytListeDilim(kayitlar []repo.YoutubeKayit, adet int) []repo.YoutubeKayit {
	if adet <= 0 {
		return nil
	}
	if len(kayitlar) > adet {
		return kayitlar[:adet]
	}
	return kayitlar
}
