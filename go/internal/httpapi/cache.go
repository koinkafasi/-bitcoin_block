package httpapi

import (
	"context"
	"time"

	"theblock/site/internal/repo"
	"theblock/site/internal/youtube"
)

// İçerik/ayar önbelleği: menü + footer + marka her istekte aynı kategorileri ve
// ayarları okuduğu için kısa TTL ile tek sorguya indirilir (performans).
const contentCacheTTL = 10 * time.Second

type ytAyarCacheEntry struct {
	at  time.Time
	v   youtube.Ayar
	set bool
}

type catsCacheEntry struct {
	at time.Time
	v  []repo.Category
}

type kvCacheEntry struct {
	at time.Time
	v  []byte
}

// cachedCategories kategori listesini kısa süre önbellekten döndürür.
func (s *Server) cachedCategories(ctx context.Context, siteID int64) []repo.Category {
	s.siteCfgMu.Lock()
	if s.catsCache == nil {
		s.catsCache = map[int64]catsCacheEntry{}
	}
	if e, ok := s.catsCache[siteID]; ok && time.Since(e.at) < contentCacheTTL {
		v := e.v
		s.siteCfgMu.Unlock()
		return v
	}
	s.siteCfgMu.Unlock()

	v, _ := repo.ListCategories(ctx, s.pool, siteID)

	s.siteCfgMu.Lock()
	s.catsCache[siteID] = catsCacheEntry{at: time.Now(), v: v}
	s.siteCfgMu.Unlock()
	return v
}

// cachedSetting ayar değerini kısa süre önbellekten döndürür (yoksa nil).
func (s *Server) cachedSetting(ctx context.Context, section, key string) []byte {
	k := section + "\x00" + key
	s.siteCfgMu.Lock()
	if s.kvCache == nil {
		s.kvCache = map[string]kvCacheEntry{}
	}
	if e, ok := s.kvCache[k]; ok && time.Since(e.at) < contentCacheTTL {
		v := e.v
		s.siteCfgMu.Unlock()
		return v
	}
	s.siteCfgMu.Unlock()

	v, err := repo.GetSetting(ctx, s.pool, section, key)
	if err != nil {
		v = nil
	}

	s.siteCfgMu.Lock()
	s.kvCache[k] = kvCacheEntry{at: time.Now(), v: v}
	s.siteCfgMu.Unlock()
	return v
}

// invalidateContentCache ayar/kategori değişince önbelleği temizler.
func (s *Server) invalidateContentCache() {
	s.siteCfgMu.Lock()
	s.catsCache = nil
	s.kvCache = nil
	s.ytAyarCache = ytAyarCacheEntry{}
	s.siteCfgMu.Unlock()
}

// cachedYtAyar YouTube ayarlarını kısa süre önbellekten döndürür.
//
// Ana sayfada üç ayrı video modülü (video / klipler / canlı) aynı ayarları
// soruyordu; her çağrı settings tablosuna 10 ayrı sorgu ile gidiyordu.
// Ölçüm (8 Kasım 2026, log_statement=all, tek ana sayfa isteği): 54 sorgunun
// 30'u yalnız bu üç çağrıdan geliyordu.
//
// AyarlariOku artık bölümü tek sorguda okuyor; bu önbellek kalan üç katı da
// kaldırır. Panelde kaydetme anında invalidateContentCache çağrıldığı için
// değişiklik en geç bir sonraki istekte görünür.
func (s *Server) cachedYtAyar(ctx context.Context) youtube.Ayar {
	s.siteCfgMu.Lock()
	if e := s.ytAyarCache; e.set && time.Since(e.at) < contentCacheTTL {
		s.siteCfgMu.Unlock()
		return e.v
	}
	s.siteCfgMu.Unlock()

	v := youtube.AyarlariOku(ctx, s.pool)

	s.siteCfgMu.Lock()
	s.ytAyarCache = ytAyarCacheEntry{at: time.Now(), v: v, set: true}
	s.siteCfgMu.Unlock()
	return v
}
