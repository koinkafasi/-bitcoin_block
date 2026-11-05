package repo

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Resmî ilanlar — detay sayfası, filtreleme ve site haritası desteği.
//
// Neden ayrı dosya: notices.go yönetim panelinin ihtiyaç duyduğu temel
// işlemleri (ekle/sil/listele) tutuyor. Buradakiler public tarafın kalıcı
// ilan adresleri ve BİK tebliğinin site haritası beklentisi için eklendi.

// NoticeFilter public ilan listesinin daralmasını tarif eder.
//
// Date boşsa en son ilan günü kullanılır (eski davranış korunur). Tüm
// tarihlerde arama yapmak için Date = NoticeAllDates verilir; kurum veya
// şehir filtresi tek güne hapsolursa kullanıcı aradığını bulamaz.
type NoticeFilter struct {
	Date        string
	Institution string
	City        string
	Query       string
	Limit       int
}

// NoticeAllDates, NoticeFilter.Date için "tarih ayrımı yapma" değeridir.
const NoticeAllDates = "tumu"

// noticeCols tek yerde tutulur: üç sorgu da aynı sırayı taramak zorunda.
const noticeCols = `id, site_id, title, COALESCE(institution,''), COALESCE(city,''),
	notice_date::text, COALESCE(url,''), COALESCE(body,''), sort, updated_at`

func scanNotices(rows interface {
	Next() bool
	Scan(...any) error
	Err() error
	Close()
}) ([]OfficialNotice, error) {
	defer rows.Close()
	var out []OfficialNotice
	for rows.Next() {
		var n OfficialNotice
		if err := rows.Scan(&n.ID, &n.SiteID, &n.Title, &n.Institution, &n.City,
			&n.NoticeDate, &n.URL, &n.Body, &n.Sort, &n.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

// FilterNotices public liste için ilanları süzer ve kullanılan tarihi döndürür.
//
// Dönen ikinci değer, listenin hangi güne ait olduğudur; tüm tarihlerde arama
// yapıldıysa boştur (başlıkta tek bir tarih göstermek yanıltıcı olurdu).
func FilterNotices(ctx context.Context, pool *pgxpool.Pool, siteID int64, f NoticeFilter) ([]OfficialNotice, string, error) {
	limit := f.Limit
	if limit <= 0 || limit > 500 {
		limit = 300
	}
	tumu := strings.EqualFold(strings.TrimSpace(f.Date), NoticeAllDates)

	// Tarih koşulu: tümü istenmediyse verilen gün, verilmediyse en son gün.
	tarihKosulu := `notice_date = COALESCE(NULLIF($2,'')::date,
		(SELECT max(notice_date) FROM official_notices WHERE site_id=$1))`
	dateArg := strings.TrimSpace(f.Date)
	if tumu {
		tarihKosulu = `($2 = $2)` // her zaman doğru; argüman sırası bozulmasın
		dateArg = NoticeAllDates
	}

	q := `SELECT ` + noticeCols + ` FROM official_notices
		WHERE site_id=$1 AND ` + tarihKosulu + `
		  AND ($3 = '' OR institution = $3)
		  AND ($4 = '' OR city = $4)
		  AND ($5 = '' OR title ILIKE '%' || $5 || '%' OR COALESCE(body,'') ILIKE '%' || $5 || '%')
		ORDER BY notice_date DESC, sort, institution, title
		LIMIT $6`

	rows, err := pool.Query(ctx, q, siteID, dateArg,
		strings.TrimSpace(f.Institution), strings.TrimSpace(f.City),
		strings.TrimSpace(f.Query), limit)
	if err != nil {
		return nil, "", err
	}
	out, err := scanNotices(rows)
	if err != nil {
		return nil, "", err
	}
	if tumu {
		return out, "", nil
	}
	used := strings.TrimSpace(f.Date)
	if used == "" && len(out) > 0 {
		used = out[0].NoticeDate
	}
	return out, used, nil
}

// NoticeByID tek ilanı döndürür. siteID ile birlikte aranır: başka bir sitenin
// ilanı kimlik tahminiyle okunamasın.
func NoticeByID(ctx context.Context, pool *pgxpool.Pool, siteID, id int64) (OfficialNotice, error) {
	var n OfficialNotice
	err := pool.QueryRow(ctx, `SELECT `+noticeCols+`
		FROM official_notices WHERE site_id=$1 AND id=$2`, siteID, id).
		Scan(&n.ID, &n.SiteID, &n.Title, &n.Institution, &n.City,
			&n.NoticeDate, &n.URL, &n.Body, &n.Sort, &n.UpdatedAt)
	return n, err
}

// NoticeFacets filtre açılır listelerini besler: ilan bulunan kurumlar ve
// şehirler. Boş değerler elenir ki listede boş satır çıkmasın.
func NoticeFacets(ctx context.Context, pool *pgxpool.Pool, siteID int64) (kurumlar, sehirler []string, err error) {
	rows, err := pool.Query(ctx, `
		SELECT DISTINCT 'k' AS tur, institution AS deger FROM official_notices
		  WHERE site_id=$1 AND COALESCE(institution,'') <> ''
		UNION
		SELECT DISTINCT 's', city FROM official_notices
		  WHERE site_id=$1 AND COALESCE(city,'') <> ''
		ORDER BY 1, 2`, siteID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var tur, deger string
		if err := rows.Scan(&tur, &deger); err != nil {
			return nil, nil, err
		}
		if tur == "k" {
			kurumlar = append(kurumlar, deger)
		} else {
			sehirler = append(sehirler, deger)
		}
	}
	return kurumlar, sehirler, rows.Err()
}

// NoticesForSitemap site haritasına girecek ilanları döndürür.
//
// BİK tebliği, içeriklere ait tam URL adreslerinin ve son güncelleme
// zamanlarının XML site haritasında bulunmasını istiyor; resmî ilanlar da
// yayımlanan içeriktir.
func NoticesForSitemap(ctx context.Context, pool *pgxpool.Pool, siteID int64, limit int) ([]OfficialNotice, error) {
	if limit <= 0 || limit > 5000 {
		limit = 2000
	}
	rows, err := pool.Query(ctx, `SELECT `+noticeCols+`
		FROM official_notices WHERE site_id=$1
		ORDER BY notice_date DESC, id DESC LIMIT $2`, siteID, limit)
	if err != nil {
		return nil, err
	}
	return scanNotices(rows)
}

// NoticeLastMod site haritası için en son güncelleme zamanını döndürür.
func NoticeLastMod(ctx context.Context, pool *pgxpool.Pool, siteID int64) (time.Time, error) {
	var t *time.Time
	if err := pool.QueryRow(ctx,
		`SELECT max(updated_at) FROM official_notices WHERE site_id=$1`, siteID).Scan(&t); err != nil {
		return time.Time{}, err
	}
	if t == nil {
		return time.Time{}, nil
	}
	return *t, nil
}
