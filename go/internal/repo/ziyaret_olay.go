package repo

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ZİYARET OLAYLARI — istek başına davranış kaydı (migration 0133).
//
// YAZMA YOLU: middleware her istek için tek satır üretir ve bunu bir
// kanala bırakır; işçi satırları TOPLU yazar (bkz. httpapi/ziyaret_olay.go).
// İstek başına tek INSERT, saniyede yüzlerce istekte bağlantı havuzunu
// tıkar ve sayfa gecikmesine dönüşürdü.
//
// SAKLAMA: 14 gün. Olay kaydı doğası gereği büyür; sınır olmadan tablo
// aylar içinde milyonlara çıkar. 14 gün "son iki hafta ne oldu" sorusuna
// yeter; daha eskisi zaten visitor_risk'te özet hâlinde duruyor.

// ZiyaretOlay tek isteğin kaydı.
type ZiyaretOlay struct {
	Anahtar    string
	IP         string
	Zaman      time.Time
	Yontem     string
	Yol        string
	Sorgu      string // maskelenmiş
	Durum      int
	Tur        string
	Kaynak     string
	Referer    string
	UA         string
	Sinif      string
	Engellendi bool
}

// OlayTopluKaydet satırları tek INSERT ile yazar.
//
// TEK SORGU, ÇOK SATIR: 200 satır için 200 gidiş-dönüş yerine 1. `UNNEST`
// pgx'in dizi kodlamasıyla birlikte çalışır; boş dilim hiç sorgu atmaz.
func OlayTopluKaydet(ctx context.Context, pool *pgxpool.Pool, olaylar []ZiyaretOlay) error {
	if pool == nil || len(olaylar) == 0 {
		return nil
	}
	n := len(olaylar)
	anahtar := make([]string, n)
	ip := make([]string, n)
	zaman := make([]time.Time, n)
	yontem := make([]string, n)
	yol := make([]string, n)
	sorgu := make([]string, n)
	durum := make([]int, n)
	tur := make([]string, n)
	kaynak := make([]string, n)
	referer := make([]string, n)
	ua := make([]string, n)
	sinif := make([]string, n)
	engel := make([]bool, n)
	for i, o := range olaylar {
		anahtar[i] = o.Anahtar
		ip[i] = o.IP
		zaman[i] = o.Zaman
		yontem[i] = o.Yontem
		yol[i] = o.Yol
		sorgu[i] = o.Sorgu
		durum[i] = o.Durum
		tur[i] = o.Tur
		kaynak[i] = o.Kaynak
		referer[i] = o.Referer
		ua[i] = o.UA
		sinif[i] = o.Sinif
		engel[i] = o.Engellendi
	}
	_, err := pool.Exec(ctx, `
		INSERT INTO visitor_events
			(anahtar, ip, zaman, yontem, yol, sorgu, durum, tur, kaynak,
			 referer, ua, sinif, engellendi)
		SELECT a, NULLIF(ip,'')::inet, z, y, yo, NULLIF(s,''), d, t, k,
		       NULLIF(r,''), NULLIF(u,''), NULLIF(si,''), e
		  FROM UNNEST($1::text[], $2::text[], $3::timestamptz[], $4::text[],
		              $5::text[], $6::text[], $7::int[], $8::text[], $9::text[],
		              $10::text[], $11::text[], $12::text[], $13::bool[])
		       AS x(a, ip, z, y, yo, s, d, t, k, r, u, si, e)`,
		anahtar, ip, zaman, yontem, yol, sorgu, durum, tur, kaynak,
		referer, ua, sinif, engel)
	return err
}

// Olaylar bir ziyaretçinin son `limit` olayını yeniden eskiye döndürür.
func Olaylar(ctx context.Context, pool *pgxpool.Pool, anahtar string, limit int) ([]ZiyaretOlay, error) {
	if pool == nil || anahtar == "" {
		return nil, nil
	}
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	rows, err := pool.Query(ctx, `
		SELECT anahtar, COALESCE(host(ip),''), zaman, yontem, yol,
		       COALESCE(sorgu,''), durum, tur, kaynak, COALESCE(referer,''),
		       COALESCE(ua,''), COALESCE(sinif,''), engellendi
		  FROM visitor_events
		 WHERE anahtar = $1
		 ORDER BY zaman DESC
		 LIMIT $2`, anahtar, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ZiyaretOlay
	for rows.Next() {
		var o ZiyaretOlay
		if err := rows.Scan(&o.Anahtar, &o.IP, &o.Zaman, &o.Yontem, &o.Yol,
			&o.Sorgu, &o.Durum, &o.Tur, &o.Kaynak, &o.Referer, &o.UA,
			&o.Sinif, &o.Engellendi); err == nil {
			out = append(out, o)
		}
	}
	return out, rows.Err()
}

// OlaySayim anahtar için tur başına istek sayısı. Niyet çıkarımının girdisi.
func OlaySayim(ctx context.Context, pool *pgxpool.Pool, anahtar string) (map[string]int, int, int, error) {
	sayim := map[string]int{}
	if pool == nil || anahtar == "" {
		return sayim, 0, 0, nil
	}
	rows, err := pool.Query(ctx, `
		SELECT tur, count(*), count(*) FILTER (WHERE engellendi)
		  FROM visitor_events
		 WHERE anahtar = $1
		 GROUP BY 1`, anahtar)
	if err != nil {
		return sayim, 0, 0, err
	}
	defer rows.Close()
	toplam, engellenen := 0, 0
	for rows.Next() {
		var tur string
		var n, e int
		if err := rows.Scan(&tur, &n, &e); err == nil {
			sayim[tur] = n
			toplam += n
			engellenen += e
		}
	}
	return sayim, toplam, engellenen, rows.Err()
}

// OlayYolSayim en sık istenen yollar — "neyi yokladı" listesi.
func OlayYolSayim(ctx context.Context, pool *pgxpool.Pool, anahtar string, limit int) ([]OlayYol, error) {
	if pool == nil || anahtar == "" {
		return nil, nil
	}
	if limit <= 0 || limit > 50 {
		limit = 15
	}
	rows, err := pool.Query(ctx, `
		SELECT yol, count(*) AS adet,
		       min(durum), max(durum)
		  FROM visitor_events
		 WHERE anahtar = $1
		 GROUP BY 1
		 ORDER BY 2 DESC, 1
		 LIMIT $2`, anahtar, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []OlayYol
	for rows.Next() {
		var y OlayYol
		if err := rows.Scan(&y.Yol, &y.Adet, &y.DurumMin, &y.DurumMax); err == nil {
			out = append(out, y)
		}
	}
	return out, rows.Err()
}

// OlayYol tek yolun istek sayısı.
type OlayYol struct {
	Yol      string
	Adet     int
	DurumMin int
	DurumMax int
}

// OlayIPAkrabalari aynı IP'yi kullanan DİĞER anahtarları döndürür:
// çerezini silip kimlik değiştiren botun yakalanma yolu (pivot).
func OlayIPAkrabalari(ctx context.Context, pool *pgxpool.Pool, ip, haric string, limit int) ([]OlayAkraba, error) {
	if pool == nil || ip == "" {
		return nil, nil
	}
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	rows, err := pool.Query(ctx, `
		SELECT anahtar, count(*) AS adet, max(zaman)
		  FROM visitor_events
		 WHERE ip = $1::inet AND anahtar <> $2
		 GROUP BY 1
		 ORDER BY 2 DESC
		 LIMIT $3`, ip, haric, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []OlayAkraba
	for rows.Next() {
		var a OlayAkraba
		if err := rows.Scan(&a.Anahtar, &a.Adet, &a.Son); err == nil {
			out = append(out, a)
		}
	}
	return out, rows.Err()
}

// OlayAkraba aynı IP'den görülen başka bir kimlik.
type OlayAkraba struct {
	Anahtar string
	Adet    int
	Son     time.Time
}

// OlayTemizle saklama süresini aşan olayları siler.
func OlayTemizle(ctx context.Context, pool *pgxpool.Pool, gun int) (int64, error) {
	if pool == nil {
		return 0, nil
	}
	if gun < 1 {
		gun = 14
	}
	tag, err := pool.Exec(ctx,
		`DELETE FROM visitor_events WHERE zaman < now() - make_interval(days => $1)`, gun)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// RiskTek tek ziyaretçinin risk satırını okur (profil sayfası).
//
// NEDEN AYRI SORGU: RiskOzet 120 satırı KPI'lar için okuyor; profil
// sayfası tek satırın TÜM alanlarını (maskesiz IP, UA) ister. Paneli
// bunun için genişletmek, her açılışta 120 satırda UA taşımak olurdu.
func RiskTek(ctx context.Context, pool *pgxpool.Pool, anahtar string) (RiskSatir, bool, error) {
	var s RiskSatir
	if pool == nil || anahtar == "" {
		return s, false, nil
	}
	err := pool.QueryRow(ctx, `
		SELECT v.anahtar, COALESCE(v.visitor_id::text,''), host(v.ip), COALESCE(v.ip_ag,''),
		       COALESCE(v.asn, a.asn, 0),
		       COALESCE(NULLIF(v.asn_ad,''), a.asn_ad, ''),
		       COALESCE(NULLIF(v.asn_ulke,''), a.ulke, ''),
		       -- BOŞ DİZE 'bilinmiyor' DEĞİLDİR ama aynı işi görür: RiskKaydet
		       -- asn_tur'u NULLIF'siz yazıyor, ağ bilgisi gelmediğinde sütunda
		       -- '' kalıyordu. Eski koşul (asn_tur <> 'bilinmiyor') boş dizeyi
		       -- "bilinen sınıf" sayıp önbellekten gelen doğru türü EZİYORDU
		       -- (entegrasyon testi yakaladı: tür "" döndü, oysa ip_asn'de
		       -- 'hosting' hazırdı).
		       CASE WHEN COALESCE(v.asn_tur,'') NOT IN ('', 'bilinmiyor')
		            THEN v.asn_tur
		            ELSE COALESCE(NULLIF(a.tur,''), 'bilinmiyor') END,
		       COALESCE(v.ulke,''), v.puan, COALESCE(v.sinif,'temiz'),
		       v.vpn, v.proxy, v.hosting, v.tor, v.bot, v.sinyaller::text,
		       COALESCE(host(v.webrtc_ip),''), v.ip_degisim, v.ulke_degisim,
		       v.istek_sayisi, v.ilk_gorulme, v.son_gorulme,
		       COALESCE(v.user_agent,'')
		  FROM visitor_risk v
		  LEFT JOIN LATERAL (
		      SELECT i.asn, i.asn_ad, i.ulke, i.tur
		        FROM ip_asn i
		       WHERE v.ip IS NOT NULL AND v.ip << i.ag::cidr
		       LIMIT 1
		  ) a ON TRUE
		 WHERE v.anahtar = $1`, anahtar).Scan(
		&s.Anahtar, &s.VisitorID, &s.IP, &s.IPAgg, &s.ASN, &s.ASNAd,
		&s.ASNUlke, &s.ASNTur, &s.Ulke, &s.Puan, &s.Sinif,
		&s.VPN, &s.Proxy, &s.Hosting, &s.Tor, &s.Bot, &s.Sinyaller,
		&s.WebRTCIP, &s.IPDegisim, &s.UlkeDegisim, &s.IstekSayisi,
		&s.IlkGorulme, &s.SonGorulme, &s.UserAgent)
	if err != nil {
		return s, false, err
	}
	return s, true, nil
}

// OlayNiyetleriToptan verilen anahtarlar için tür sayımlarını TEK sorguda
// döndürür: liste sayfası 120 ziyaretçinin niyetini böylece 120 sorguyla
// değil 1 sorguyla çıkarır.
//
// NEDEN ANAHTAR BAŞINA SORGU DEĞİL: 120 açılışta 120 gidiş-dönüş, paneli
// gözle görülür yavaşlatıyordu (ölçüm: ziyaretçi konumu panelinde aynı
// hata 6 ayrı sorguyla yapılmıştı, orada da tek turda okumaya geçildi).
func OlayNiyetleriToptan(ctx context.Context, pool *pgxpool.Pool, anahtarlar []string) (map[string]map[string]int, error) {
	out := map[string]map[string]int{}
	if pool == nil || len(anahtarlar) == 0 {
		return out, nil
	}
	rows, err := pool.Query(ctx, `
		SELECT anahtar, tur, count(*)
		  FROM visitor_events
		 WHERE anahtar = ANY($1::text[])
		 GROUP BY 1, 2`, anahtarlar)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var a, t string
		var n int
		if err := rows.Scan(&a, &t, &n); err == nil {
			if out[a] == nil {
				out[a] = map[string]int{}
			}
			out[a][t] = n
		}
	}
	return out, rows.Err()
}
