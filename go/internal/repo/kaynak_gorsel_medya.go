package repo

import (
	"context"
)

// ---- Kaynak görsellerini medya kütüphanesine alma ----
//
// DURUM (13 Kasım 2026): RSS görselleri `source_images` tablosuna ve
// `{depo}/kaynak/` ağacına iniyor — 712 görsel, 152 MB. Ama medya
// kütüphanesi (`media_files`) yalnız 39 kayıt tutuyor: brand 30, upload 9.
//
// Yani görseller BİZİM DİSKİMİZDE ama BİZİM KÜTÜPHANEMİZDE değil. Sonuç:
//   · alt metni yazılamıyor (SEO'da görsel aramanın tek girdisi),
//   · editör panelinden düzenlenemiyor,
//   · AI görsel üretici geldiğinde "şunları yeniden üret" denemiyor.
//
// Aktarımdan sonra `media_files.source = 'internet'` işareti taşırlar.
// O değer şemada zaten kabul ediliyor:
//   CHECK (source IN ('upload','library','internet','render','generate','pexels','ai'))
// Yani yeni bir işaret alanı GEREKMEDİ — var olan sözleşme kullanıldı.
// Görsel üretici gelince `source='internet'` olanlar kuyruğa alınır ve
// yeniden üretilenler `source='ai'` olur.

// AktarilacakGorsel medya kütüphanesine alınmayı bekleyen kaynak görseli.
type AktarilacakGorsel struct {
	Hash        string
	URL         string
	ContentType string
	Bytes       int64
}

// AktarilacakGorseller indirilmiş ama medyaya alınmamış görselleri verir.
//
// Sıra YENİDEN ESKİYE: yeni haberin görseli daha çok okunuyor, önce o
// kütüphaneye girsin ve alt metni yazılabilir olsun.
func AktarilacakGorseller(ctx context.Context, pool DB, limit int) ([]AktarilacakGorsel, error) {
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	rows, err := pool.Query(ctx, `
		SELECT hash, url, COALESCE(content_type, ''), COALESCE(bytes, 0)
		  FROM source_images
		 WHERE media_id IS NULL
		   AND bytes > 0
		 ORDER BY created_at DESC
		 LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []AktarilacakGorsel
	for rows.Next() {
		var g AktarilacakGorsel
		if err := rows.Scan(&g.Hash, &g.URL, &g.ContentType, &g.Bytes); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// GorselMedyayaBagla aktarımı kaydeder.
func GorselMedyayaBagla(ctx context.Context, pool DB, hash string, mediaID int64) error {
	_, err := pool.Exec(ctx, `
		UPDATE source_images SET media_id = $2 WHERE hash = $1`, hash, mediaID)
	return err
}

// GorselKullananIcerikler bir kaynak görselini kapak olarak kullanan
// içeriklerin kimliklerini verir.
//
// `image_url` alanı `/kg/<hash>` biçiminde tutuluyor (bkz.
// KaynakGorselYol). Aktarımdan sonra bu içeriklerin `cover_media_id`'si
// bağlanır ki görsel artık kütüphane kaydı üzerinden sunulsun.
func GorselKullananIcerikler(ctx context.Context, pool DB, hash string) ([]int64, error) {
	rows, err := pool.Query(ctx, `
		SELECT id FROM contents
		 WHERE image_url = $1
		   AND cover_media_id IS NULL`, KaynakGorselYol(hash))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// IcerikKapakMedyasiAta içeriğin kapak medyasını bağlar.
//
// `image_url` KORUNUR, silinmez: eski bağlantı hem geri dönüş yolu hem
// de atıf kaydı (görselin hangi kaynaktan geldiği `image_credit` ile
// birlikte izlenebilir kalsın).
func IcerikKapakMedyasiAta(ctx context.Context, pool DB, contentID, mediaID int64) error {
	_, err := pool.Exec(ctx, `
		UPDATE contents
		   SET cover_media_id = $2, updated_at = now()
		 WHERE id = $1
		   AND cover_media_id IS NULL`, contentID, mediaID)
	return err
}

// GorselAktarimOzeti panel/günlük için tek satırlık durum.
type GorselAktarimOzeti struct {
	Toplam    int // indirilmiş kaynak görsel
	Aktarilan int // medyaya alınmış
	Bekleyen  int
}

// GorselAktarimDurumu aktarım ilerleyişini verir.
func GorselAktarimDurumu(ctx context.Context, pool DB) (GorselAktarimOzeti, error) {
	var o GorselAktarimOzeti
	err := pool.QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE bytes > 0),
		       count(*) FILTER (WHERE bytes > 0 AND media_id IS NOT NULL),
		       count(*) FILTER (WHERE bytes > 0 AND media_id IS NULL)
		  FROM source_images`).
		Scan(&o.Toplam, &o.Aktarilan, &o.Bekleyen)
	return o, err
}

// DisKaynakliMedyalar `source='internet'` olan medya kayıtlarını verir.
//
// AI görsel üretici devreye girdiğinde yeniden üretim kuyruğu budur:
// dış kaynaklı görsel yerine kendi ürettiğimiz görsel konur ve kayıt
// `source='ai'` olarak güncellenir.
func DisKaynakliMedyalar(ctx context.Context, pool DB, limit int) ([]int64, error) {
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	rows, err := pool.Query(ctx, `
		SELECT id FROM media_files
		 WHERE source = 'internet'
		 ORDER BY created_at DESC
		 LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
