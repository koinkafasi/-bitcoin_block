package repo

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// YouTube video arşivi (bkz. migrations/0122_youtube_icerik).
//
// Tablo, scheduler'ın periyodik çektiği YouTube içeriğini tutar; sayfalar
// buradan okur. Yönetim bayrakları (`one_cikan`, `canli`, `gizli`, `sira`)
// yalnız panelden değişir — senkronizasyon bunları EZMEZ.

// YoutubeKayit tek bir arşiv satırı.
type YoutubeKayit struct {
	VideoID   string
	Tur       string // video | short | stream
	Baslik    string
	Aciklama  string
	Kanal     string
	Gorsel    string
	URL       string
	SureSn    int
	Izlenme   int64
	YayinAt   *time.Time
	Canli     bool
	OneCikan  bool
	Gizli     bool
	Sira      int
	CreatedAt time.Time
	UpdatedAt time.Time
}

const youtubeKolonlar = `video_id, tur, baslik, aciklama, kanal, gorsel, url,
	sure_sn, izlenme, yayin_at, canli, one_cikan, gizli, sira, created_at, updated_at`

func youtubeTara(row pgx.Row) (YoutubeKayit, error) {
	var k YoutubeKayit
	err := row.Scan(&k.VideoID, &k.Tur, &k.Baslik, &k.Aciklama, &k.Kanal, &k.Gorsel,
		&k.URL, &k.SureSn, &k.Izlenme, &k.YayinAt, &k.Canli, &k.OneCikan, &k.Gizli,
		&k.Sira, &k.CreatedAt, &k.UpdatedAt)
	return k, err
}

// YoutubeKaydet çekilen kayıtları ekler/günceller.
//
// YALNIZ YouTube'dan gelen alanlar yazılır: başlık, açıklama, görsel, süre,
// izlenme, yayın zamanı. `canli/one_cikan/gizli/sira` DOKUNULMAZ — editörün
// "öne çıkar" veya "gizle" kararı her senkronizasyonda korunmalı.
// (Canlı işareti ayrı çağrıyla, YoutubeCanliAyarla ile yazılır.)
func YoutubeKaydet(ctx context.Context, pool DB, items []YoutubeKayit) (int, error) {
	n := 0
	for i, it := range items {
		if it.VideoID == "" {
			continue
		}
		_, err := pool.Exec(ctx, `
			INSERT INTO youtube_icerik
				(video_id, tur, baslik, aciklama, kanal, gorsel, url, sure_sn, izlenme, yayin_at, sira)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
			ON CONFLICT (video_id) DO UPDATE SET
				tur       = EXCLUDED.tur,
				baslik    = EXCLUDED.baslik,
				aciklama  = EXCLUDED.aciklama,
				kanal     = EXCLUDED.kanal,
				gorsel    = EXCLUDED.gorsel,
				url       = EXCLUDED.url,
				sure_sn   = EXCLUDED.sure_sn,
				izlenme   = EXCLUDED.izlenme,
				yayin_at  = COALESCE(EXCLUDED.yayin_at, youtube_icerik.yayin_at),
				-- sira yalnız ilk yazımda anlamlı; sonraki çekmelerde
				-- "yeni video başa" davranışı yayin_at sıralamasıyla gelir.
				sira      = youtube_icerik.sira,
				updated_at = now()`,
			it.VideoID, it.Tur, it.Baslik, it.Aciklama, it.Kanal, it.Gorsel, it.URL,
			it.SureSn, it.Izlenme, it.YayinAt, i)
		if err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

// YoutubeListele türe göre görünür kayıtları döndürür (yeni → eski).
func YoutubeListele(ctx context.Context, pool DB, tur string, limit int) ([]YoutubeKayit, error) {
	if limit <= 0 {
		limit = 30
	}
	rows, err := pool.Query(ctx, `
		SELECT `+youtubeKolonlar+` FROM youtube_icerik
		WHERE tur = $1 AND NOT gizli
		ORDER BY yayin_at DESC NULLS LAST, sira ASC
		LIMIT $2`, tur, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []YoutubeKayit
	for rows.Next() {
		k, err := youtubeTara(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

// YoutubeListeleHepsi panel listesi: gizli olanlar da, tür süzgeciyle.
func YoutubeListeleHepsi(ctx context.Context, pool DB, tur string, limit int) ([]YoutubeKayit, error) {
	if limit <= 0 {
		limit = 100
	}
	sorgu := `SELECT ` + youtubeKolonlar + ` FROM youtube_icerik
		WHERE ($1 = '' OR tur = $1)
		ORDER BY one_cikan DESC, yayin_at DESC NULLS LAST, sira ASC
		LIMIT $2`
	rows, err := pool.Query(ctx, sorgu, tur, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []YoutubeKayit
	for rows.Next() {
		k, err := youtubeTara(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

// YoutubeOne tek kayıt (video sayfası).
func YoutubeOne(ctx context.Context, pool DB, videoID string) (*YoutubeKayit, error) {
	row := pool.QueryRow(ctx, `SELECT `+youtubeKolonlar+` FROM youtube_icerik WHERE video_id = $1`, videoID)
	k, err := youtubeTara(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &k, nil
}

// YoutubeOneCikan öne çıkarılan kaydı döndürür (yoksa ErrNotFound).
func YoutubeOneCikan(ctx context.Context, pool DB, tur string) (*YoutubeKayit, error) {
	row := pool.QueryRow(ctx, `
		SELECT `+youtubeKolonlar+` FROM youtube_icerik
		WHERE tur = $1 AND one_cikan AND NOT gizli
		ORDER BY updated_at DESC LIMIT 1`, tur)
	k, err := youtubeTara(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &k, nil
}

// YoutubeCanliAyarla yalnız verilen video "canlı" kalacak biçimde işaretler
// (boş kimlik: hepsi kapanır).
func YoutubeCanliAyarla(ctx context.Context, pool DB, videoID string) error {
	_, err := pool.Exec(ctx, `
		UPDATE youtube_icerik
		SET canli = (video_id = $1), updated_at = now()
		WHERE canli <> (video_id = $1)`, videoID)
	return err
}

// YoutubeBayrak tek bir bayrağı açar/kapatır. alan adı SABİT listeden gelir
// (SQL'e kullanıcı metni eklenmez).
func YoutubeBayrak(ctx context.Context, pool DB, videoID, alan string, deger bool) error {
	kolon := ""
	switch alan {
	case "one_cikan":
		kolon = "one_cikan"
	case "gizli":
		kolon = "gizli"
	case "canli":
		kolon = "canli"
	default:
		return ErrNotFound
	}
	if kolon == "one_cikan" && deger {
		// Aynı türde tek öne çıkan: eskisi kapanır (belirsizlik kalmasın).
		if _, err := pool.Exec(ctx, `
			UPDATE youtube_icerik SET one_cikan = false, updated_at = now()
			WHERE one_cikan AND tur = (SELECT tur FROM youtube_icerik WHERE video_id = $1)`,
			videoID); err != nil {
			return err
		}
	}
	_, err := pool.Exec(ctx, `UPDATE youtube_icerik SET `+kolon+` = $2, updated_at = now() WHERE video_id = $1`,
		videoID, deger)
	return err
}

// YoutubeSil kaydı arşivden kaldırır. Senkronizasyon bir sonraki turda
// YouTube'da hâlâ duruyorsa yeniden ekler; kalıcı silmek için panelden
// "gizle" kullanılır (bkz. panel açıklaması).
func YoutubeSil(ctx context.Context, pool DB, videoID string) error {
	_, err := pool.Exec(ctx, `DELETE FROM youtube_icerik WHERE video_id = $1`, videoID)
	return err
}

// YoutubeSayilar tür başına görünür kayıt sayısı (panel özeti).
func YoutubeSayilar(ctx context.Context, pool DB) (map[string]int, error) {
	rows, err := pool.Query(ctx, `SELECT tur, count(*) FROM youtube_icerik WHERE NOT gizli GROUP BY tur`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var tur string
		var n int
		if err := rows.Scan(&tur, &n); err != nil {
			return nil, err
		}
		out[tur] = n
	}
	return out, rows.Err()
}
