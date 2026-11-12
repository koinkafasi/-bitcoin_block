package repo

import (
	"context"
	"strconv"
	"strings"
	"unicode"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ---- İç link önerisi: seçili metinden ilgili yazıyı bulma ----
//
// NEDEN YENİ ARAMA: eski sorgu `title ILIKE '%tüm ifade%'` yapıyordu.
// Editörde bir cümle parçası seçilip bağlantı aranınca ("Bitcoin ETF
// onayı") tam alt dize eşleşmesi beklendiği için sonuç çıkmıyordu —
// başlık "SEC, Bitcoin ETF'lerini onayladı" olsa bile. Yazarın her
// seferinde başlığı tahmin edip elle yazması gerekiyordu.
//
// YENİ DAVRANIŞ: ifade kelimelere bölünür, anlamsız bağlaçlar atılır,
// kalan kelimelerin HEPSİNİ taşıyan yazılar bulunur ve alaka sırasına
// göre dizilir. Hiç sonuç çıkmazsa kelime sayısı azaltılarak tekrar
// denenir (en az bir kelime kalana dek).

// icLinkStopWords aramada taşıyıcı olmayan kelimeler. Bunları sorguya
// katmak, neredeyse her başlığı eşleştirip sıralamayı anlamsızlaştırıyor.
var icLinkStopWords = map[string]bool{
	"ve": true, "ile": true, "için": true, "bir": true, "bu": true,
	"şu": true, "o": true, "da": true, "de": true, "ki": true,
	"mi": true, "mı": true, "mu": true, "mü": true, "ne": true,
	"çok": true, "daha": true, "en": true, "gibi": true, "kadar": true,
	"sonra": true, "önce": true, "ama": true, "fakat": true, "ancak": true,
	"olarak": true, "olan": true, "oldu": true, "the": true, "and": true,
	"for": true, "with": true, "from": true,
}

// icLinkKelimeler ifadeyi aranabilir kelimelere ayırır.
//
// Üç harften kısa kelimeler atılır: "ab", "io" gibi parçalar ILIKE'ta
// her yere uyuyor. Noktalama ve kesme işareti temizlenir ki "Bitcoin'in"
// ile "Bitcoin" aynı kelime sayılsın.
func icLinkKelimeler(ifade string) []string {
	alanlar := strings.FieldsFunc(ifade, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	var out []string
	gorulen := map[string]bool{}
	for _, k := range alanlar {
		k = strings.ToLower(strings.TrimSpace(k))
		if len([]rune(k)) < 3 || icLinkStopWords[k] || gorulen[k] {
			continue
		}
		gorulen[k] = true
		out = append(out, k)
		if len(out) == 6 {
			break // altıdan fazlası sorguyu daraltıp sonuçsuz bırakıyor
		}
	}
	return out
}

// OneriliContentLinks seçili metne en yakın yayınları verir.
//
// Sıralama: (1) tam ifade başlıkta geçiyorsa en üstte, (2) eşleşen
// kelime sayısı, (3) tazelik. Böylece hem birebir eşleşme hem de konu
// yakınlığı aynı listede toplanıyor.
func OneriliContentLinks(ctx context.Context, pool *pgxpool.Pool, siteID int64, ifade string, limit int) ([]ContentLink, error) {
	ifade = strings.TrimSpace(ifade)
	if ifade == "" {
		return nil, nil
	}
	if siteID <= 0 {
		siteID = 1
	}
	if limit <= 0 || limit > 30 {
		limit = 12
	}
	kelimeler := icLinkKelimeler(ifade)
	if len(kelimeler) == 0 {
		// Yalnız bağlaçlardan oluşan seçim: eski davranışa düş.
		return SearchPublishedContentLinks(ctx, pool, siteID, ifade, limit)
	}

	// Kelime sayısını azaltarak dene: önce hepsi, sonra ilk n-1…
	for n := len(kelimeler); n >= 1; n-- {
		out, err := icLinkSorgu(ctx, pool, siteID, ifade, kelimeler[:n], limit)
		if err != nil {
			return nil, err
		}
		if len(out) > 0 {
			return out, nil
		}
	}
	return nil, nil
}

func icLinkSorgu(ctx context.Context, pool *pgxpool.Pool, siteID int64, ifade string, kelimeler []string, limit int) ([]ContentLink, error) {
	// $1 site, $2 tam ifade, $3 limit; kelimeler $4'ten başlar.
	args := []any{siteID, ifade, limit}
	var kosullar, skorlar []string
	for i, k := range kelimeler {
		p := "$" + strconv.Itoa(len(args)+1)
		args = append(args, "%"+k+"%")
		kosullar = append(kosullar,
			"(c.title ILIKE "+p+" OR COALESCE(c.spot,'') ILIKE "+p+")")
		// Başlıkta geçen kelime, spotta geçenden değerli.
		skorlar = append(skorlar,
			"(CASE WHEN c.title ILIKE "+p+" THEN 10 ELSE 0 END)",
			"(CASE WHEN COALESCE(c.spot,'') ILIKE "+p+" THEN 3 ELSE 0 END)")
		_ = i
	}

	sql := `
		SELECT c.id, c.title, COALESCE(c.slug,''), c.locale
		FROM contents c
		WHERE c.site_id = $1
		  AND c.status = 'yayinda'
		  AND (` + strings.Join(kosullar, " AND ") + `)
		ORDER BY
		  (CASE WHEN c.title ILIKE '%' || $2 || '%' THEN 100 ELSE 0 END) +
		  ` + strings.Join(skorlar, " + ") + ` DESC,
		  c.published_at DESC NULLS LAST,
		  c.id DESC
		LIMIT $3`

	rows, err := pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ContentLink
	for rows.Next() {
		var l ContentLink
		if err := rows.Scan(&l.ID, &l.Title, &l.Slug, &l.Locale); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}
