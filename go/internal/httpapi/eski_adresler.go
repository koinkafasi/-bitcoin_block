package httpapi

import (
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"theblock/site/internal/repo"
)

// tarihArsivRE yalnız tarih taşıyan yolu yakalar: /2019, /2019/12, /2019/12/11.
// Yazı adresi (/2019/12/11/slug) buraya girmez — ek segment vardır.
var tarihArsivRE = regexp.MustCompile(`^/\d{4}(/\d{2}(/\d{2})?)?/?$`)

func tarihArsivMi(yol string) bool {
	return tarihArsivRE.MatchString(yol)
}

func (s *Server) makaleYolu(w http.ResponseWriter, r *http.Request, segs []string, lang string) bool {
	slugTarih, ok := makaleTarihDeseni(segs)
	if !ok {
		return false
	}
	c, err := repo.ContentBySlug(r.Context(), s.pool, slugTarih)
	if err != nil {
		return false // içerik yok: çağıran bölüm sayfasını denesin
	}
	if c.Status != "yayinda" {
		s.handleNotFound(w, r)
		return true
	}
	// TEK KANONİK ADRES: yazıya bölüm öneki yanlış ya da hiç yokken de
	// ulaşılabiliyordu (aynı içerik iki adreste 200). Kanonik adrese 301.
	if lang == c.Locale {
		if hedef := makaleURL(c); hedef != r.URL.Path {
			if r.URL.RawQuery != "" {
				hedef += "?" + r.URL.RawQuery
			}
			http.Redirect(w, r, hedef, http.StatusMovedPermanently)
			return true
		}
	}
	s.handleCDArticle(w, r, lang, slugTarih, c.ID)
	return true
}

func wpKimlikSorgusu(q url.Values) int64 {
	ham := strings.TrimSpace(q.Get("p"))
	if ham == "" {
		return 0
	}
	id, err := strconv.ParseInt(ham, 10, 64)
	if err != nil || id <= 0 {
		return 0
	}
	return id
}

func (s *Server) eskiKimlikYonlendir(w http.ResponseWriter, r *http.Request, wpID int64) {
	ctx := r.Context()
	var icID int64
	err := s.pool.QueryRow(ctx, `
		SELECT c.id
		  FROM contents c
		  JOIN contents_extra e ON e.content_id = c.id
		 WHERE e.extra->>'wp_id' = $1
		   AND c.site_id = $2
		   AND c.status = 'yayinda'
		 LIMIT 1`, strconv.FormatInt(wpID, 10), s.curSiteID(r)).Scan(&icID)
	if err != nil {
		s.handleNotFound(w, r)
		return
	}
	c, err := repo.GetContent(ctx, s.pool, icID)
	if err != nil {
		s.handleNotFound(w, r)
		return
	}
	hedef := makaleURL(c)
	if q := r.URL.Query(); len(q) > 0 {
		q.Del("p")
		q.Del("page_id")
		if kalan := q.Encode(); kalan != "" {
			hedef += "?" + kalan
		}
	}
	http.Redirect(w, r, hedef, http.StatusMovedPermanently)
}
