package httpapi

import (
	"context"
	"sync"
	"time"

	"theblock/site/internal/repo"
)

const cdCacheTTL = 10 * time.Second

type cdShellData struct {
	Tags   []repo.Etiket
	Coins  []repo.FiyatSatiri
	Dolu   map[string]bool
	DoluOk bool
}

func (s *Server) cdShellVeri(ctx context.Context) cdShellData {
	v, ok := s.cdShellCache.Load(ctx, "shell", func(ctx context.Context) (cdShellData, bool) {
		var d cdShellData
		var wg sync.WaitGroup
		wg.Add(3)
		go func() { defer wg.Done(); d.Tags, _ = repo.PopulerEtiketler(ctx, s.pool, 6) }()
		go func() { defer wg.Done(); d.Coins = repo.Fiyatlar(ctx, s.pool) }()
		go func() {
			defer wg.Done()
			dolu, err := repo.DoluBolumler(ctx, s.pool, 1)
			if err == nil {
				d.Dolu, d.DoluOk = dolu, true
			}
		}()
		wg.Wait()
		// "İçerik yok" geçerli bir sonuçtur: boş etiket/section listesi de
		// önbelleğe yazılır, aksi hâlde her istek yeniden sorgular.
		return d, true
	})
	if !ok {
		return cdShellData{}
	}
	return v
}

func (s *Server) cdHomeIcerikleri(ctx context.Context) []repo.Content {
	v, ok := s.cdHomeCache.Load(ctx, "home:tr", func(ctx context.Context) ([]repo.Content, bool) {
		items, _, err := repo.ListContents(ctx, s.pool, repo.ListFilter{
			Status:         "yayinda",
			SiteID:         1,
			Locale:         "tr",
			Page:           1,
			PerPage:        60,
			ExcludeGallery: true,
			Sort:           "yayin",
		})
		if err != nil {
			// Hata ÖNBELLEĞE YAZILMAZ: geçici DB hatası 10 sn boyunca boş
			// sayfa olarak donmasın, sonraki istek yeniden dener.
			s.logger.Warn("cd ana sayfa: içerik listesi", "error", err)
			return nil, false
		}
		return items, true
	})
	if !ok {
		return nil
	}
	return v
}

// cdListeSonuc liste sayfası önbellek değeri (kayıtlar + toplam sayı).
type cdListeSonuc struct {
	Kayitlar []repo.Content
	Toplam   int
}

// cdListeIcerikleri bölüm/liste sayfası içeriğini önbellekten getirir.
// Anahtar: bölüm + sayfa numarası (farklı sayfalar birbirini ezmez).
func (s *Server) cdListeIcerikleri(ctx context.Context, slug string, sayfa int) ([]repo.Content, int) {
	key := slug + ":" + itoa(sayfa)
	v, ok := s.cdListeCache.Load(ctx, key, func(ctx context.Context) (cdListeSonuc, bool) {
		f := repo.ListFilter{
			Status:         "yayinda",
			SiteID:         1,
			Locale:         "tr",
			Page:           sayfa,
			PerPage:        24,
			ExcludeGallery: true,
			Sort:           "yayin",
		}
		if slug != "latest-crypto-news" {
			if id := s.kategoriID(ctx, slug); id > 0 {
				f.CategoryID = id
			}
		}
		items, toplam, err := repo.ListContents(ctx, s.pool, f)
		if err != nil {
			s.logger.Warn("cd bölüm: liste", "bolum", slug, "error", err)
			return cdListeSonuc{}, false
		}
		return cdListeSonuc{Kayitlar: items, Toplam: toplam}, true
	})
	if !ok {
		return nil, 0
	}
	return v.Kayitlar, v.Toplam
}
