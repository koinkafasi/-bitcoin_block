package repo

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TranskriptAramaSonucu tam metin aramasında bulunan tek transkripttir.
type TranskriptAramaSonucu struct {
	TranscriptID int64
	ContentID    *int64
	MediaID      int64
	ContentTitle string
	Language     string
	DurationMS   int64
	// Snippet eşleşen bölümün vurgulanmış hâli (ts_headline).
	Snippet string
	Rank    float64
}

func SearchAudioTranscripts(ctx context.Context, pool *pgxpool.Pool, siteID int64, query string, limit int) ([]TranskriptAramaSonucu, error) {
	q := strings.TrimSpace(query)
	if len([]rune(q)) < 2 {
		return nil, nil
	}
	if limit <= 0 || limit > 100 {
		limit = 30
	}

	const sel = `
		SELECT t.id, t.content_id, t.media_id, COALESCE(c.title, '') AS baslik,
		       t.language, t.duration_ms,
		       ts_headline('simple', t.text, websearch_to_tsquery('simple', $2),
		                   'StartSel=<mark>, StopSel=</mark>, MaxWords=28, MinWords=12, MaxFragments=1') AS snippet,
		       ts_rank(t.search, websearch_to_tsquery('simple', $2)) AS rank
		FROM audio_transcripts t
		LEFT JOIN contents c ON c.id = t.content_id
		WHERE t.site_id = $1
		  AND t.search @@ websearch_to_tsquery('simple', $2)
		ORDER BY rank DESC, t.updated_at DESC
		LIMIT $3`

	rows, err := pool.Query(ctx, sel, siteIDOr1(siteID), q, limit)
	if err != nil {
		// websearch_to_tsquery geçersiz sözdiziminde hata verebilir; sessizce
		// ILIKE'a düş (kullanıcıya hata göstermek yerine sonuç ver).
		return searchTranscriptsFallback(ctx, pool, siteID, q, limit)
	}
	defer rows.Close()

	out, err := scanTranscriptHits(rows)
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		// tsvector boş olabilir (eski kayıtlar/trigger öncesi); ILIKE dene.
		return searchTranscriptsFallback(ctx, pool, siteID, q, limit)
	}
	return out, nil
}

// searchTranscriptsFallback basit ILIKE aramasıdır. Tam metin indeksi
// kullanılamadığında (geçersiz sorgu, boş vektör) devreye girer.
func searchTranscriptsFallback(ctx context.Context, pool *pgxpool.Pool, siteID int64, q string, limit int) ([]TranskriptAramaSonucu, error) {
	const sel = `
		SELECT t.id, t.content_id, t.media_id, COALESCE(c.title, '') AS baslik,
		       t.language, t.duration_ms,
		       substring(t.text FROM greatest(1, strpos(lower(t.text), lower($2)) - 80) FOR 200) AS snippet,
		       0.0 AS rank
		FROM audio_transcripts t
		LEFT JOIN contents c ON c.id = t.content_id
		WHERE t.site_id = $1 AND t.text ILIKE '%' || $2 || '%'
		ORDER BY t.updated_at DESC
		LIMIT $3`
	rows, err := pool.Query(ctx, sel, siteIDOr1(siteID), q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTranscriptHits(rows)
}

func scanTranscriptHits(rows pgx.Rows) ([]TranskriptAramaSonucu, error) {
	var out []TranskriptAramaSonucu
	for rows.Next() {
		var r TranskriptAramaSonucu
		if err := rows.Scan(&r.TranscriptID, &r.ContentID, &r.MediaID, &r.ContentTitle,
			&r.Language, &r.DurationMS, &r.Snippet, &r.Rank); err != nil {
			return nil, err
		}
		r.Snippet = strings.Join(strings.Fields(r.Snippet), " ")
		out = append(out, r)
	}
	return out, rows.Err()
}
