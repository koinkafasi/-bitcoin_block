package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"theblock/site/internal/scoring"
	"theblock/site/internal/trends"
)

// Yayın önceliği eksenlerinin veritabanı tarafı (spec 11 §5).
//
// Sütuna yalnız YAVAŞ DEĞİŞEN eksenler yazılır: score_guven (kaynağın
// feeds.trust değeri değiştiğinde) ve score_trend (trend terimleri
// tazelendiğinde). Tazelik sütuna yazılmaz; created_at'ten sorgu anında
// hesaplanır (scoring.TazelikSQL), böylece hiç bayatlamaz.

// RefreshGuvenScores içeriğin kaynak güvenini tek geçişte yazar.
//
// contents.source_name ↔ feeds.name eşleşmesi üretimde %100 tutuyor
// (34.842/34.845; eşleşmeyen 3 kayıt elle oluşturulmuş, kaynaksız).
// Eşleşmeyenlere 50 yazılır: ne ödüllendirilir ne cezalandırılır —
// scoring.KaynakGuveni ile aynı kural.
func RefreshGuvenScores(ctx context.Context, pool *pgxpool.Pool, siteID int64) (int64, error) {
	guven := scoring.GuvenSQL("f")
	tag, err := pool.Exec(ctx, fmt.Sprintf(`
		UPDATE contents c
		   SET score_guven = %[1]s, score_hesap_at = now()
		  FROM feeds f
		 WHERE f.name = c.source_name
		   AND c.site_id = $1
		   AND c.score_guven IS DISTINCT FROM (%[1]s)::smallint`, guven), siteID)
	if err != nil {
		return 0, err
	}
	n := tag.RowsAffected()

	// Kaynağı bilinmeyen içerik: nötr 50.
	tag2, err := pool.Exec(ctx, `
		UPDATE contents c
		   SET score_guven = 50, score_hesap_at = now()
		 WHERE c.site_id = $1
		   AND c.score_guven <> 50
		   AND NOT EXISTS (SELECT 1 FROM feeds f WHERE f.name = c.source_name)`, siteID)
	if err != nil {
		return n, err
	}
	return n + tag2.RowsAffected(), nil
}

// trendTerimTazelikSaat: bundan eski terimler puanlamaya girmez. Google
// Trends "şu an" demektir; scoring.TrendUyumu da aynı sınırı uygular, bu
// sorgu yalnız gereksiz karşılaştırmayı baştan eler.
const trendTerimTazelikSaat = 48

// TrendSkorSonuc RefreshTrendScores'un raporudur.
type TrendSkorSonuc struct {
	Terim   int // puanlamaya giren taze terim sayısı
	Taranan int
	Eslesen int
	Yazilan int
	Dagilim map[int]int // 10'luk bantlara göre score_trend
	Sure    time.Duration
}

// RefreshTrendScores taslakları taze trend terimleriyle eşleştirip
// contents.score_trend yazar.
//
// trend_matches tablosuna YAZMAZ. O tablo kapsama analizinindir ("bu konuyu
// yayınladık mı?") ve bilerek yalnız yayındaki içeriğe bakar; taslakları
// oraya eklemek coverage_count'u şişirir ve editöre işlenmemiş konuyu
// "kapsandı" gösterirdi. İki soru ayrı, iki depo ayrı.
//
// dry=true iken hiçbir şey yazılmaz, yalnız dağılım raporlanır.
func RefreshTrendScores(ctx context.Context, pool *pgxpool.Pool, siteID int64, batch int, dry bool) (TrendSkorSonuc, error) {
	basladi := time.Now()
	out := TrendSkorSonuc{Dagilim: map[int]int{}}
	if batch <= 0 || batch > 5000 {
		batch = 1000
	}

	rows, err := pool.Query(ctx, fmt.Sprintf(`
		SELECT id, term, last_seen_at, COALESCE(approx_traffic,0)
		  FROM trend_terms
		 WHERE site_id = $1 AND hidden = FALSE
		   AND last_seen_at > now() - interval '%d hours'`, trendTerimTazelikSaat), siteID)
	if err != nil {
		return out, err
	}
	var terimler []trends.HazirTerim
	for rows.Next() {
		var id int64
		var term string
		var son time.Time
		var trafik int
		if err := rows.Scan(&id, &term, &son, &trafik); err != nil {
			rows.Close()
			return out, err
		}
		if h, ok := trends.Hazirla(id, term, son, trafik); ok {
			terimler = append(terimler, h)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return out, err
	}
	out.Terim = len(terimler)
	if len(terimler) == 0 {
		out.Sure = time.Since(basladi)
		return out, nil
	}

	simdi := time.Now()
	lastID := int64(0)
	for {
		// Tazelik penceresinin dışındaki taslaklar zaten 0 tazelikte;
		// trend puanlarını tazelemek sıralamayı değiştirmez, yalnız iş
		// üretir. Sınır scoring'den gelir ki iki taraf ayrışmasın.
		prows, err := pool.Query(ctx, fmt.Sprintf(`
			SELECT id, title, COALESCE(spot,''), COALESCE(score_trend,0)
			  FROM contents
			 WHERE site_id = $1 AND status = 'taslak' AND id > $2
			   AND created_at > now() - interval '%d hours'
			 ORDER BY id LIMIT $3`, scoring.TazelikSinirSaat()), siteID, lastID, batch)
		if err != nil {
			return out, err
		}
		type satir struct {
			id     int64
			metin  string
			mevcut int
		}
		var sayfa []satir
		for prows.Next() {
			var s satir
			var baslik, spot string
			if err := prows.Scan(&s.id, &baslik, &spot, &s.mevcut); err != nil {
				prows.Close()
				return out, err
			}
			s.metin = baslik + " " + spot
			sayfa = append(sayfa, s)
		}
		prows.Close()
		if err := prows.Err(); err != nil {
			return out, err
		}
		if len(sayfa) == 0 {
			break
		}

		for _, s := range sayfa {
			lastID = s.id
			out.Taranan++
			skor := 0
			if en, oran, ok := trends.EnIyiEslesme(trends.HazirlaBelge(s.metin), terimler); ok {
				out.Eslesen++
				skor = scoring.TrendUyumu(oran, en.LastSeenAt, simdi, en.Traffic)
			}
			out.Dagilim[skor/10*10]++
			if skor == s.mevcut {
				continue
			}
			if dry {
				out.Yazilan++
				continue
			}
			if _, err := pool.Exec(ctx,
				`UPDATE contents SET score_trend = $1, score_hesap_at = now() WHERE id = $2`,
				skor, s.id); err != nil {
				return out, err
			}
			out.Yazilan++
		}
	}
	out.Sure = time.Since(basladi)
	return out, nil
}
