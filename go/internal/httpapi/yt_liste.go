package httpapi

import (
	"context"
	"time"

	"theblock/site/internal/repo"
)

const ytListeTTL = 10 * time.Second

const ytListeTavan = 12

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
