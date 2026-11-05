package httpapi

import (
	"context"
	"time"

	"theblock/site/internal/repo"
)

// İçerik/ayar önbelleği: menü + footer + marka her istekte aynı kategorileri ve
// ayarları okuduğu için kısa TTL ile tek sorguya indirilir (performans).
const contentCacheTTL = 10 * time.Second

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
	s.siteCfgMu.Unlock()
}
