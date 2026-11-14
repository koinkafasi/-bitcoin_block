package repo

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TRAFİK GÜVENLİĞİ — veri katmanı (migration 0131).
//
// İki tablo:
//   visitor_risk → ziyaretçi başına TEK satır (anahtar PK). Tüm istekler
//                  satırı günceller; tablo trafikle doğrusal büyümez.
//   ip_asn       → ASN önbelleği, /24 değil GERÇEK önek anahtarıyla.
//
// YAZMA SIKLIĞI ÇAĞIRANDA: middleware her istekte DB'ye yazmaz (saniyede
// yüzlerce istek = yüzlerce upsert). Ziyaretçi başına 10 dakikada bir
// yazılır; arada biriken sayaçlar bellekte tutulur.

// RiskSatir panelin ve yazımın ortak satır tipi.
type RiskSatir struct {
	Anahtar     string // tb_vid ya da "ip:<adres>"
	VisitorID   string // çerez varsa UUID, yoksa ""
	IP          string
	IPAgg       string
	ASN         int
	ASNAd       string
	ASNUlke     string
	ASNTur      string // hosting|vpn|tor|mobil|kurumsal|konut|bilinmiyor
	Ulke        string
	Puan        int
	Sinif       string // temiz|supheli|yuksek|bot|arama
	VPN         bool
	Proxy       bool
	Hosting     bool
	Tor         bool
	Bot         bool
	Sinyaller   string // JSON dizi metni
	WebRTCIP    string
	IPDegisim   int
	UlkeDegisim int
	IstekSayisi int64
	IlkGorulme  time.Time
	SonGorulme  time.Time
}

// RiskKaydet satırı ekler/günceller.
//
// BAYRAKLAR `OR` İLE BİRLEŞİR: bir ziyaretçi bir kez VPN'den görüldüyse
// bu kanıt korunur (sonraki temiz IP onu silmez). PUAN ise SON değerdir:
// panel "şu anki risk"i göstermeli, geçmişin en yükseğini değil — aksi
// hâlde VPN'den çıkıp temiz bağlantıya geçen okuyucu sonsuza dek
// "yüksek risk" görünürdü.
func RiskKaydet(ctx context.Context, pool *pgxpool.Pool, r RiskSatir) error {
	if pool == nil || r.Anahtar == "" {
		return nil
	}
	if r.Sinyaller == "" {
		r.Sinyaller = "[]"
	}
	_, err := pool.Exec(ctx, `
		INSERT INTO visitor_risk AS v
			(anahtar, visitor_id, ip, ip_ag, asn, asn_ad, asn_ulke, asn_tur,
			 ulke, puan, sinif, vpn, proxy, hosting, tor, bot, sinyaller,
			 webrtc_ip, ip_degisim, ulke_degisim, istek_sayisi,
			 ilk_gorulme, son_gorulme)
		VALUES ($1, NULLIF($2,'')::uuid, NULLIF($3,'')::inet, NULLIF($4,''),
		        NULLIF($5,0), NULLIF($6,''), NULLIF($7,''), $8,
		        NULLIF($9,''), $10, $11, $12, $13, $14, $15, $16, $17::jsonb,
		        NULLIF($18,'')::inet, 0, 0, $19, now(), now())
		ON CONFLICT (anahtar) DO UPDATE SET
			visitor_id   = COALESCE(EXCLUDED.visitor_id, v.visitor_id),
			-- IP/ülke DEĞİŞİMİ: VPN düğüm değiştiren ziyaretçinin en güçlü
			-- sinyali. Sayaç, art arda gelen farklı değerlerde artar.
			ip_degisim   = v.ip_degisim + CASE
				WHEN EXCLUDED.ip IS NOT NULL AND v.ip IS NOT NULL
				     AND EXCLUDED.ip <> v.ip THEN 1 ELSE 0 END,
			ulke_degisim = v.ulke_degisim + CASE
				WHEN EXCLUDED.ulke IS NOT NULL AND v.ulke IS NOT NULL
				     AND EXCLUDED.ulke <> v.ulke THEN 1 ELSE 0 END,
			ip           = COALESCE(EXCLUDED.ip, v.ip),
			ip_ag        = COALESCE(EXCLUDED.ip_ag, v.ip_ag),
			asn          = COALESCE(EXCLUDED.asn, v.asn),
			asn_ad       = COALESCE(EXCLUDED.asn_ad, v.asn_ad),
			asn_ulke     = COALESCE(EXCLUDED.asn_ulke, v.asn_ulke),
			-- 'bilinmiyor' bilgi DEĞİLDİR: bilinen sınıfı ezmemeli.
			asn_tur      = CASE WHEN EXCLUDED.asn_tur = 'bilinmiyor'
			                    THEN v.asn_tur ELSE EXCLUDED.asn_tur END,
			ulke         = COALESCE(EXCLUDED.ulke, v.ulke),
			puan         = EXCLUDED.puan,
			sinif        = EXCLUDED.sinif,
			vpn          = v.vpn     OR EXCLUDED.vpn,
			proxy        = v.proxy   OR EXCLUDED.proxy,
			hosting      = v.hosting OR EXCLUDED.hosting,
			tor          = v.tor     OR EXCLUDED.tor,
			bot          = v.bot     OR EXCLUDED.bot,
			sinyaller    = EXCLUDED.sinyaller,
			webrtc_ip    = COALESCE(EXCLUDED.webrtc_ip, v.webrtc_ip),
			istek_sayisi = v.istek_sayisi + EXCLUDED.istek_sayisi,
			son_gorulme  = now()`,
		r.Anahtar, r.VisitorID, r.IP, r.IPAgg, r.ASN, r.ASNAd, r.ASNUlke, r.ASNTur,
		r.Ulke, r.Puan, r.Sinif, r.VPN, r.Proxy, r.Hosting, r.Tor, r.Bot, r.Sinyaller,
		r.WebRTCIP, r.IstekSayisi)
	return err
}

// RiskIstemciGuncelle tarayıcı beacon'ının sonucunu yazar.
//
// PUAN YALNIZ YÜKSELİR (`GREATEST`): istemci sinyalleri sunucunun verdiği
// kararı düşüremez. Aksi hâlde bot, "ben temizim" diyen bir beacon ile
// kendi puanını sıfırlardı. Sınıf da yalnız YÜKSELİR (temiz<supheli<
// yuksek<bot): beacon kararı 100 puansa sınıf "bot" olmalı, "yuksek"
// değil — ölçüldü 14 Kasım 2026, SQL eşiği sabit 60'a bakıyordu.
//
// VPN/PROXY BAYRAĞI BURADA YAZILMAZ: bu bayraklar AĞ/başlık kanıtıdır
// (ASN, proxy başlığı). İstemci puanı yüksek diye bir ziyaretçiyi "VPN"
// diye etiketlemek yanlış olurdu — puan başka sinyallerden de gelir.
func RiskIstemciGuncelle(ctx context.Context, pool *pgxpool.Pool,
	anahtar, webrtcIP string, puan int, sinif, sinyaller string) error {
	if pool == nil || anahtar == "" {
		return nil
	}
	if sinyaller == "" {
		sinyaller = "[]"
	}
	_, err := pool.Exec(ctx, `
		UPDATE visitor_risk SET
			puan      = LEAST(100, GREATEST(puan, $2)),
			sinif     = CASE
				WHEN $3 = 'bot' THEN 'bot'
				WHEN $3 = 'yuksek' AND sinif IN ('temiz','supheli') THEN 'yuksek'
				WHEN $3 = 'supheli' AND sinif = 'temiz' THEN 'supheli'
				ELSE sinif END,
			webrtc_ip = COALESCE(NULLIF($4,'')::inet, webrtc_ip),
			sinyaller = $5::jsonb,
			son_gorulme = now()
		WHERE anahtar = $1`,
		anahtar, puan, sinif, webrtcIP, sinyaller)
	return err
}

// RiskOzeti panelin tek seferde okuduğu görünüm.
type RiskOzeti struct {
	Toplam     int64
	Gercek     int64
	Supheli    int64
	Yuksek     int64
	Bot        int64
	Arama      int64
	VPN        int64
	Hosting    int64
	Tor        int64
	ASNDagilim []RiskASN
	Sinyaller  []RiskSinyal
	Satirlar   []RiskSatir
}

type RiskASN struct {
	ASN  int
	Ad   string
	Tur  string
	Adet int64
}

type RiskSinyal struct {
	Kod  string
	Adet int64
}

// RiskOzet son `gun` günün risk özetini üretir.
func RiskOzet(ctx context.Context, pool *pgxpool.Pool, gun int) (RiskOzeti, error) {
	var o RiskOzeti
	if pool == nil {
		return o, nil
	}
	if gun <= 0 {
		gun = 30
	}

	if err := pool.QueryRow(ctx, `
		SELECT count(*),
		       count(*) FILTER (WHERE sinif = 'temiz'),
		       count(*) FILTER (WHERE sinif = 'supheli'),
		       count(*) FILTER (WHERE sinif = 'yuksek'),
		       count(*) FILTER (WHERE sinif = 'bot'),
		       count(*) FILTER (WHERE sinif = 'arama'),
		       count(*) FILTER (WHERE vpn OR proxy),
		       count(*) FILTER (WHERE hosting),
		       count(*) FILTER (WHERE tor)
		  FROM visitor_risk
		 WHERE son_gorulme >= now() - make_interval(days => $1)`,
		gun).Scan(&o.Toplam, &o.Gercek, &o.Supheli, &o.Yuksek, &o.Bot,
		&o.Arama, &o.VPN, &o.Hosting, &o.Tor); err != nil {
		return o, err
	}

	rows, err := pool.Query(ctx, `
		SELECT asn, COALESCE(asn_ad,''), COALESCE(asn_tur,'bilinmiyor'), count(*)
		  FROM visitor_risk
		 WHERE son_gorulme >= now() - make_interval(days => $1)
		   AND asn IS NOT NULL
		 GROUP BY 1,2,3
		 ORDER BY 4 DESC
		 LIMIT 15`, gun)
	if err != nil {
		return o, err
	}
	for rows.Next() {
		var a RiskASN
		if err := rows.Scan(&a.ASN, &a.Ad, &a.Tur, &a.Adet); err == nil {
			o.ASNDagilim = append(o.ASNDagilim, a)
		}
	}
	rows.Close()

	rows, err = pool.Query(ctx, `
		SELECT s->>'k' AS kod, count(*) AS adet
		  FROM visitor_risk, jsonb_array_elements(sinyaller) AS s
		 WHERE son_gorulme >= now() - make_interval(days => $1)
		 GROUP BY 1
		 ORDER BY 2 DESC
		 LIMIT 20`, gun)
	if err != nil {
		return o, err
	}
	for rows.Next() {
		var s RiskSinyal
		if err := rows.Scan(&s.Kod, &s.Adet); err == nil {
			o.Sinyaller = append(o.Sinyaller, s)
		}
	}
	rows.Close()

	rs, err := pool.Query(ctx, `
		SELECT anahtar, COALESCE(visitor_id::text,''), host(ip), COALESCE(ip_ag,''),
		       COALESCE(asn,0), COALESCE(asn_ad,''), COALESCE(asn_ulke,''),
		       COALESCE(asn_tur,'bilinmiyor'), COALESCE(ulke,''), puan,
		       COALESCE(sinif,'temiz'), vpn, proxy, hosting, tor, bot,
		       sinyaller::text, COALESCE(host(webrtc_ip),''),
		       ip_degisim, ulke_degisim, istek_sayisi, ilk_gorulme, son_gorulme
		  FROM visitor_risk
		 WHERE son_gorulme >= now() - make_interval(days => $1)
		 ORDER BY puan DESC, son_gorulme DESC
		 LIMIT 120`, gun)
	if err != nil {
		return o, err
	}
	defer rs.Close()
	for rs.Next() {
		var s RiskSatir
		if err := rs.Scan(&s.Anahtar, &s.VisitorID, &s.IP, &s.IPAgg, &s.ASN,
			&s.ASNAd, &s.ASNUlke, &s.ASNTur, &s.Ulke, &s.Puan, &s.Sinif,
			&s.VPN, &s.Proxy, &s.Hosting, &s.Tor, &s.Bot, &s.Sinyaller,
			&s.WebRTCIP, &s.IPDegisim, &s.UlkeDegisim, &s.IstekSayisi,
			&s.IlkGorulme, &s.SonGorulme); err == nil {
			o.Satirlar = append(o.Satirlar, s)
		}
	}
	return o, rs.Err()
}

// RiskTemizle eski satırları siler; tablo sonsuz büyümesin.
// Dönen: silinen satır sayısı.
func RiskTemizle(ctx context.Context, pool *pgxpool.Pool, gun int) (int64, error) {
	if pool == nil {
		return 0, nil
	}
	if gun < 30 {
		gun = 180
	}
	tag, err := pool.Exec(ctx,
		`DELETE FROM visitor_risk WHERE son_gorulme < now() - make_interval(days => $1)`, gun)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// ---- ASN önbelleği ----

// ASNSatir ip_asn tablosunun tek satırı.
type ASNSatir struct {
	Ag     string
	ASN    int
	Ad     string
	Ulke   string
	Tur    string
	Kaynak string
	Guncel time.Time
}

// ASNKaydet önbellek satırını yazar (ag PK).
func ASNKaydet(ctx context.Context, pool *pgxpool.Pool, a ASNSatir) error {
	if pool == nil || a.Ag == "" || a.ASN <= 0 {
		return nil
	}
	_, err := pool.Exec(ctx, `
		INSERT INTO ip_asn (ag, asn, asn_ad, ulke, tur, kaynak, guncelleme)
		VALUES ($1, $2, COALESCE($3,''), NULLIF($4,''), COALESCE($5,'bilinmiyor'), COALESCE($6,'cymru'), now())
		ON CONFLICT (ag) DO UPDATE SET
			asn = EXCLUDED.asn, asn_ad = EXCLUDED.asn_ad, ulke = EXCLUDED.ulke,
			tur = EXCLUDED.tur, kaynak = EXCLUDED.kaynak, guncelleme = now()`,
		a.Ag, a.ASN, a.Ad, a.Ulke, a.Tur, a.Kaynak)
	return err
}

// ASNListele açılış ön ısıtması için son kullanılan önekleri döndürür.
func ASNListele(ctx context.Context, pool *pgxpool.Pool, limit int) ([]ASNSatir, error) {
	if pool == nil {
		return nil, nil
	}
	if limit <= 0 || limit > 5000 {
		limit = 500
	}
	rows, err := pool.Query(ctx, `
		SELECT ag, asn, COALESCE(asn_ad,''), COALESCE(ulke,''), COALESCE(tur,'bilinmiyor'),
		       COALESCE(kaynak,'cymru'), guncelleme
		  FROM ip_asn ORDER BY guncelleme DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ASNSatir
	for rows.Next() {
		var a ASNSatir
		if err := rows.Scan(&a.Ag, &a.ASN, &a.Ad, &a.Ulke, &a.Tur, &a.Kaynak, &a.Guncel); err == nil {
			out = append(out, a)
		}
	}
	return out, rows.Err()
}

// ASNTemizle eski önbellek satırlarını siler (30 gün).
func ASNTemizle(ctx context.Context, pool *pgxpool.Pool, gun int) (int64, error) {
	if pool == nil {
		return 0, nil
	}
	if gun < 1 {
		gun = 30
	}
	tag, err := pool.Exec(ctx,
		`DELETE FROM ip_asn WHERE guncelleme < now() - make_interval(days => $1)`, gun)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
