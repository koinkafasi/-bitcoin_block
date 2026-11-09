package httpapi

// WordPress kalıntısı adresler.
//
// NEDEN: bitcosar.com 2019–2025 arası WordPress ile yayınlandı; permalink
// yapısı `/%postname%/` idi ve arşivde 3.235 yazı, 35 kategori, 3.808 etiket,
// 5 yazar var. Site Go'ya taşındığında adres deseni değişti
// (`/<bölüm>/<YYYY>/<MM>/<DD>/<slug>`). Eski adresler 404 dönerse Google'da
// birikmiş bağlantı değeri çöpe gider ve indekslenmiş URL'ler düşer.
//
// Burada üç sınıf ele alınır:
//   - yazı adresinin BÖLÜMÜ yanlış ya da hiç yok → kanonik adrese 301
//   - tarih arşivi (/2019/12, /2019/12/11) → /arsiv
//   - /?p=<WordPress kimliği> → yazının kanonik adresi
//
// Kategori/etiket/yazar eski adresleri `redirects` tablosunda tutulur
// (veriyle üretilir, kodda sabit liste yok).

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

// wpKimlikSorgusu WordPress'in kimlikle erişim parametrelerini okur.
// `?p=` yazı, `?page_id=` sayfa demektir; sayfalar taşınmadığı için
// yalnız `p` ele alınır ve geçersizse 0 döner.
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

// eskiKimlikYonlendir `/?p=<wp kimliği>` adresini kanonik yazı adresine taşır.
// WordPress kimliği `contents_extra.extra->>'wp_id'` alanında saklıdır
// (aktarım betiği yazar); eşleşme yoksa 404 verilir.
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
	// `p` parametresi hedefe TAŞINMAZ: taşınırsa adres yine `?p=` içerdiği
	// için aynı kural tetiklenir ve yönlendirme kendine döner (sonsuz döngü).
	// Diğer parametreler (utm_*) korunur.
	if q := r.URL.Query(); len(q) > 0 {
		q.Del("p")
		q.Del("page_id")
		if kalan := q.Encode(); kalan != "" {
			hedef += "?" + kalan
		}
	}
	http.Redirect(w, r, hedef, http.StatusMovedPermanently)
}
