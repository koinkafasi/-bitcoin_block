package repo

import (
	"context"
	"sort"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Birleşik gelen kutusu (Faz 167).
//
// NEDEN: iletişim mesajları, içerik geri bildirimi ve yorumlar üç ayrı
// ekranda duruyordu; "bugün ne geldi?" sorusunun cevabı üç yere bakmayı
// gerektiriyordu. Pacely'nin /messages ekranı tek listede topluyor.
//
// Bu görünüm SALT OKUMADIR ve yıkıcı eylem içermez: her öğe kendi ekranına
// yönlendirir. İşlem yapmak isteyen kullanıcı kaynağın kendi sayfasına
// gider; böylece yetki ve iş kuralları tek yerde kalır.
type GelenOge struct {
	Tur    string // "iletisim" | "bildirim" | "yorum"
	TurAd  string
	Ikon   string
	ID     int64
	Baslik string
	Ozet   string
	Kimden string
	Durum  string // yeni | okundu | arsiv | bekliyor | cozuldu
	Href   string
	Zaman  time.Time
}

// GelenKutusu üç kaynağı tek listede birleştirir.
//
// `limit` toplam üst sınırdır, kaynak başına değil: kullanıcı "son 50 olay"
// görür, "her kaynaktan 50" değil. Kaynak başına limit/2+1 çekilip birleşim
// kesiliyor; böylece tek bir kaynak listeyi doldurup diğerlerini ezmiyor.
func GelenKutusu(ctx context.Context, pool *pgxpool.Pool, siteID int64, limit int) ([]GelenOge, error) {
	if limit <= 0 {
		limit = 40
	}
	pay := limit/2 + 1

	yeniMesajlar, err := GelenIletisim(ctx, pool, siteID, pay)
	if err != nil {
		return nil, err
	}
	yeniBildirimler, err := GelenBildirim(ctx, pool, siteID, pay)
	if err != nil {
		return nil, err
	}
	yeniYorumlar, err := GelenYorum(ctx, pool, siteID, pay)
	if err != nil {
		return nil, err
	}

	hepsi := make([]GelenOge, 0, len(yeniMesajlar)+len(yeniBildirimler)+len(yeniYorumlar))
	hepsi = append(hepsi, yeniMesajlar...)
	hepsi = append(hepsi, yeniBildirimler...)
	hepsi = append(hepsi, yeniYorumlar...)

	// En yeni önce. Basit seçim sıralaması yerine sort.Slice kullanılıyor:
	// üç kaynak birleşince 120 öğeye kadar çıkabilir.
	siralaZaman(hepsi)
	if len(hepsi) > limit {
		hepsi = hepsi[:limit]
	}
	return hepsi, nil
}

// GelenIletisim iletişim formundan gelen mesajlar.
func GelenIletisim(ctx context.Context, pool *pgxpool.Pool, siteID int64, limit int) ([]GelenOge, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, COALESCE(NULLIF(subject,''),'(konu yok)'), message,
		       COALESCE(NULLIF(name,''),'İsimsiz'), COALESCE(email,''), status, created_at
		FROM contact_messages
		WHERE site_id = $1
		ORDER BY created_at DESC LIMIT $2`, siteID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []GelenOge
	for rows.Next() {
		var o GelenOge
		var govde, eposta string
		if err := rows.Scan(&o.ID, &o.Baslik, &govde, &o.Kimden, &eposta, &o.Durum, &o.Zaman); err != nil {
			return nil, err
		}
		o.Tur = "iletisim"
		o.TurAd = "İletişim"
		o.Ikon = "mail"
		if eposta != "" {
			o.Kimden = o.Kimden + " · " + eposta
		}
		o.Ozet = kisalt(govde, 140)
		o.Href = "/admin/iletisim"
		out = append(out, o)
	}
	return out, rows.Err()
}

// GelenBildirim okuyucu içerik geri bildirimi.
func GelenBildirim(ctx context.Context, pool *pgxpool.Pool, siteID int64, limit int) ([]GelenOge, error) {
	rows, err := pool.Query(ctx, `
		SELECT f.id, COALESCE(c.title,'(içerik silinmiş)'), COALESCE(f.detail,''),
		       COALESCE(NULLIF(f.reporter,''),'Okuyucu'), f.kind, f.status, f.created_at
		FROM content_feedback f
		LEFT JOIN contents c ON c.id = f.content_id
		WHERE f.site_id = $1
		ORDER BY f.created_at DESC LIMIT $2`, siteID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []GelenOge
	for rows.Next() {
		var o GelenOge
		var detay, tur string
		if err := rows.Scan(&o.ID, &o.Baslik, &detay, &o.Kimden, &tur, &o.Durum, &o.Zaman); err != nil {
			return nil, err
		}
		o.Tur = "bildirim"
		o.TurAd = "Geri bildirim"
		o.Ikon = "message-dots"
		o.Kimden = o.Kimden + " · " + bildirimTuru(tur)
		o.Ozet = kisalt(detay, 140)
		o.Href = "/admin/geri-bildirim"
		out = append(out, o)
	}
	return out, rows.Err()
}

// GelenYorum onay bekleyen ve son yorumlar.
func GelenYorum(ctx context.Context, pool *pgxpool.Pool, siteID int64, limit int) ([]GelenOge, error) {
	rows, err := pool.Query(ctx, `
		SELECT cm.id, COALESCE(c.title,'(içerik silinmiş)'), COALESCE(cm.body,''),
		       COALESCE(NULLIF(cm.author_name,''),'Ziyaretçi'), cm.status, cm.created_at
		FROM comments cm
		LEFT JOIN contents c ON c.id = cm.content_id
		WHERE c.site_id = $1
		ORDER BY cm.created_at DESC LIMIT $2`, siteID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []GelenOge
	for rows.Next() {
		var o GelenOge
		var govde string
		if err := rows.Scan(&o.ID, &o.Baslik, &govde, &o.Kimden, &o.Durum, &o.Zaman); err != nil {
			return nil, err
		}
		o.Tur = "yorum"
		o.TurAd = "Yorum"
		o.Ikon = "message-circle"
		o.Ozet = kisalt(govde, 140)
		o.Href = "/admin/yorumlar"
		out = append(out, o)
	}
	return out, rows.Err()
}

// GelenSayilar kaynak başına toplam sayı (üst kartlar için).
func GelenSayilar(ctx context.Context, pool *pgxpool.Pool, siteID int64) (map[string]int, error) {
	out := map[string]int{}
	rows, err := pool.Query(ctx, `
		SELECT 'iletisim', count(*) FROM contact_messages WHERE site_id=$1 AND status='yeni'
		UNION ALL
		SELECT 'bildirim', count(*) FROM content_feedback WHERE site_id=$1 AND status='bekliyor'
		UNION ALL
		SELECT 'yorum', count(*) FROM comments cm JOIN contents c ON c.id=cm.content_id
		  WHERE c.site_id=$1 AND cm.status='bekliyor'`, siteID)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var ad string
		var n int
		if err := rows.Scan(&ad, &n); err != nil {
			return out, err
		}
		out[ad] = n
	}
	return out, rows.Err()
}

func bildirimTuru(v string) string {
	switch v {
	case "hata":
		return "Hata bildirimi"
	case "oneri":
		return "Öneri"
	case "telif":
		return "Telif"
	case "spam":
		return "Spam"
	case "":
		return "Bildirim"
	}
	return v
}

func kisalt(v string, n int) string {
	r := []rune(v)
	if len(r) <= n {
		return v
	}
	return string(r[:n]) + "…"
}

// siralaZaman öğeleri en yeni önce sıralar.
func siralaZaman(v []GelenOge) {
	sort.SliceStable(v, func(i, j int) bool { return v[i].Zaman.After(v[j].Zaman) })
}
