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

// makaleYolu tarihli makale desenini çözer ve yanıtı yazar.
//
// Dönüş değeri `true` ise yanıt üretilmiştir; çağıran başka iş yapmamalıdır.
// `false` ise yol tarihli makale desenine uymuyordur (ya da içerik yoktur ve
// çağıranın kendi 404'üne düşmesi gerekir).
//
// Bölüm adı ile makale yolunun çakışmasını çözer: `/borsa/2023/04/17/<slug>`
// artık BORSA fiyat sayfasına değil makaleye gider (bkz. çağrı yeri yorumu).
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
