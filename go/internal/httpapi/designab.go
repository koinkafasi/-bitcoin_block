package httpapi

import (
	"context"
	"math/rand/v2"
	"net/http"

	"theblock/site/internal/platform"
	"theblock/site/internal/repo"
)

// ---- Sayfa düzeni A/B testi (Faz 7) ----
//
// A varyantı = aktif düzen (page_layouts.blocks), B varyantı = layout_ab.b_blocks
// (yönetici tasarımcıdan "B olarak ata" der). Ziyaretçi çerez (tb_ab) ile ayrıştırılır;
// çerez yoksa weight_b olasılığıyla B atanır ve çerez yazılır. Günlük hit sayılır.

const abCookie = "tb_ab"

// applyAB (varsa) A/B testini uygular: varyant seçer, çerez yazar, günlük hit
// kaydeder ve B seçildiyse d.Blocks'u B bloğundan kurar. AB kapalıysa dokunulmaz.
func (s *Server) applyAB(w http.ResponseWriter, r *http.Request, d *tmplData, pageKey string) {
	if d == nil || d.Home == nil {
		return
	}
	ctx := r.Context()
	sid := s.curSiteID(r)
	ab, err := repo.GetLayoutAB(ctx, s.pool, sid, pageKey)
	if err != nil || !ab.Enabled {
		return
	}

	variant := s.abVariantFromCookie(r, pageKey)
	if variant == "" {
		variant = rollABVariant(ab.WeightB)
		s.setABCookie(w, pageKey, variant)
	}
	_ = repo.RecordABHit(ctx, s.pool, sid, pageKey, variant) // sayaç hatayı bozmaz

	if variant == "b" {
		bRaw := platform.NormalizeDesign(ab.BBlocks)
		if len(bRaw) > 2 {
			d.Blocks = s.buildBlocks(ctx, d, bRaw)
		}
	}
}

// abVariantFromCookie çerezden daha önce atanan varyantı okur.
func (s *Server) abVariantFromCookie(r *http.Request, pageKey string) string {
	c, err := r.Cookie(abCookie + "_" + pageKey)
	if err != nil {
		return ""
	}
	if c.Value == "a" || c.Value == "b" {
		return c.Value
	}
	return ""
}

// rollABVariant weightB (0-100) olasılığıyla "b" veya "a" döndürür.
func rollABVariant(weightB int) string {
	switch {
	case weightB >= 100:
		return "b"
	case weightB <= 0:
		return "a"
	}
	// #nosec G404 -- A/B varyant kovası: güvenlik kararı değil, ölçüm
	// dağıtımı. Kriptografik RNG burada gereksiz ve yavaş olurdu.
	if rand.IntN(100) < weightB { // nosemgrep: go.lang.security.audit.crypto.math_random.math-random-used -- A/B kova seçimi; kriptografik rastgelelik gerekmiyor.
		return "b"
	}
	return "a"
}

// setABCookie varyantı 90 günlük çerez olarak yazar (aynı ziyaretçi sabit kalır).
func (s *Server) setABCookie(w http.ResponseWriter, pageKey, variant string) {
	http.SetCookie(w, &http.Cookie{
		Name: abCookie + "_" + pageKey, Value: variant, Path: "/",
		MaxAge: 90 * 24 * 3600, HttpOnly: true, Secure: s.secureCookie(), SameSite: http.SameSiteLaxMode,
	})
}

// abHitsView panel için son günlerin A/B sayılarını döndürür.
func (s *Server) abHitsView(ctx context.Context, sid int64, pageKey string, days int) []repo.LayoutABHits {
	h, err := repo.ListLayoutABHits(ctx, s.pool, sid, pageKey, days)
	if err != nil {
		return nil
	}
	return h
}
