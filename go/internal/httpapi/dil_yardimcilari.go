package httpapi

import (
	"context"
	"strings"
)

func (s *Server) yerelIcerikVar(ctx context.Context, siteID int64, locale string) bool {
	if locale == "" {
		return false
	}
	var bir int
	err := s.pool.QueryRow(ctx,
		`SELECT 1 FROM contents WHERE site_id=$1 AND locale=$2 AND status='yayinda' LIMIT 1`,
		siteIDOr1(siteID), locale).Scan(&bir)
	return err == nil
}

func (s *Server) icerikDili(ctx context.Context, siteID int64, istenen string) (string, bool) {
	varsayilan := s.i18n.Default()
	if istenen == "" || istenen == varsayilan {
		return varsayilan, false
	}
	if s.yerelIcerikVar(ctx, siteID, istenen) {
		return istenen, false
	}
	return varsayilan, true
}

func (s *Server) yerelHreflang(ctx context.Context, absBase string, siteID int64, istenen string, yolBicimi func(locale string) string) []langLink {
	taban := strings.TrimRight(absBase, "/")
	kendi := istenen
	if kendi == "" {
		kendi = s.i18n.Default()
	}
	out := []langLink{{
		Code: kendi,
		URL:  taban + yolBicimi(kendi),
	}}
	for _, l := range s.i18n.Enabled() {
		if l.Code == kendi || l.Code == "tr" {
			continue
		}
		if !s.yerelIcerikVar(ctx, siteID, l.Code) {
			continue
		}
		out = append(out, langLink{Code: l.Code, URL: taban + yolBicimi(l.Code)})
	}
	// tr varsayılan dil: her zaman alternatif olarak listelenir (kendi dil
	// değilse) çünkü sayfa ona düşebiliyor.
	if kendi != s.i18n.Default() {
		out = append(out, langLink{Code: s.i18n.Default(), URL: taban + yolBicimi(s.i18n.Default())})
	}
	if len(out) < 2 {
		// Tek dil: hreflang anlamsız.
		return nil
	}
	out = append(out, langLink{Code: "x-default", URL: taban + yolBicimi(s.i18n.Default())})
	return out
}

// siteIDOr1 nil/0 site kimliğini 1'e çeker (repo katmanıyla aynı kural).
func siteIDOr1(id int64) int64 {
	if id <= 0 {
		return 1
	}
	return id
}

func (s *Server) categoryLangName(ctx context.Context, locale, slug, yedek string) string {
	if s.i18n == nil || locale == "" || locale == "tr" || slug == "" {
		return yedek
	}
	anahtar := "category." + slug
	v := s.i18n.T(locale, anahtar)
	if v == anahtar || strings.TrimSpace(v) == "" {
		return yedek
	}
	return v
}

func (s *Server) localeText(locale, anahtar, yedek string) string {
	if s.i18n == nil || locale == "" || locale == "tr" {
		return yedek
	}
	v := s.i18n.T(locale, anahtar)
	if v == anahtar || strings.TrimSpace(v) == "" {
		return yedek
	}
	return v
}

func (s *Server) kategoriIcerikDili(ctx context.Context, siteID int64, slug, istenen string) (string, bool) {
	varsayilan := s.i18n.Default()
	if istenen == "" || istenen == varsayilan {
		return varsayilan, false
	}
	var bir int
	err := s.pool.QueryRow(ctx, `
		SELECT 1 FROM contents c
		JOIN categories kat ON kat.id = c.category_id
		WHERE c.site_id = $1 AND c.locale = $2 AND c.status = 'yayinda'
		  AND (kat.slug = $3
		       OR kat.parent_id = (SELECT id FROM categories WHERE slug = $3 LIMIT 1))
		LIMIT 1`, siteIDOr1(siteID), istenen, slug).Scan(&bir)
	if err == nil {
		return istenen, false
	}
	return varsayilan, true
}
