package repo

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ZİYARETÇİ COĞRAFYASI — veri katmanı.
//
// Tablo: visitor_locations (migration 0130). Satır BAŞINA BİR ZİYARETÇİ
// (visitor_id PK), ziyaret başına değil: aynı tarayıcı 50 kez gelse bile
// tek satır güncellenir. Böylece tablo trafikle doğrusal büyümez.

// ZiyaretciKonum kaydedilecek tek ziyaretçi satırı.
//
// Boş alanlar NULL yazılır; "bilinmiyor" ile "0" karışmasın diye
// pointer yerine değer + Has* bayrağı yerine nil kontrolü kullanılıyor:
// enlem/boylam null olabilir, 0.0 geçerli bir koordinattır (Gine Körfezi).
type ZiyaretciKonum struct {
	VisitorID string   // UUID metni (çerezden)
	IP        string   // boş olabilir
	Ulke      string   // ISO-2, boş olabilir
	Sehir     string   // şimdilik yalnız tarayıcıdan/ileride
	Enlem     *float64 // nil = yok
	Boylam    *float64
	DogrulukM *int
	Kaynak    string // "ip" | "tarayici"
	Izin      bool   // tarayıcı izni verildi mi
	UserAgent string
}

// ZiyaretciKaydet satırı ekler ya da günceller (visitor_id anahtar).
//
// KISMİ GÜNCELLEME BİLİNÇLİ: IP kaydı geldiğinde tarayıcının daha önce
// yazdığı KOORDİNAT SİLİNMEZ. `COALESCE` ile yalnız yeni değer varsa
// üzerine yazılır; yoksa mevcut korunur. Aksi hâlde her sayfa görüntülemesi
// (IP kaydı) kullanıcının izinle verdiği konumu sıfırlardı.
func ZiyaretciKaydet(ctx context.Context, pool *pgxpool.Pool, z ZiyaretciKonum) error {
	if pool == nil || z.VisitorID == "" {
		return nil
	}
	_, err := pool.Exec(ctx, `
		INSERT INTO visitor_locations AS v
			(visitor_id, ip, ulke, sehir, enlem, boylam, dogruluk_m,
			 kaynak, izin, user_agent, ilk_görülme, son_görülme)
		VALUES ($1, NULLIF($2,'')::inet, NULLIF($3,''), NULLIF($4,''),
		        $5, $6, $7, $8, $9, NULLIF($10,''), now(), now())
		ON CONFLICT (visitor_id) DO UPDATE SET
			ip          = COALESCE(EXCLUDED.ip,          v.ip),
			ulke        = COALESCE(EXCLUDED.ulke,        v.ulke),
			sehir       = COALESCE(EXCLUDED.sehir,       v.sehir),
			enlem       = COALESCE(EXCLUDED.enlem,       v.enlem),
			boylam      = COALESCE(EXCLUDED.boylam,      v.boylam),
			dogruluk_m  = COALESCE(EXCLUDED.dogruluk_m,  v.dogruluk_m),
			-- kaynak/izin YALNIZ yükseltilir: tarayıcı izni verildiyse
			-- sonraki IP kaydı onu 'ip'ye geri düşürmemeli.
			kaynak      = CASE WHEN v.kaynak = 'tarayici' THEN v.kaynak
			                   ELSE EXCLUDED.kaynak END,
			izin        = v.izin OR EXCLUDED.izin,
			user_agent  = COALESCE(EXCLUDED.user_agent,  v.user_agent),
			son_görülme = now()`,
		z.VisitorID, z.IP, z.Ulke, z.Sehir, z.Enlem, z.Boylam, z.DogrulukM,
		z.Kaynak, z.Izin, z.UserAgent)
	return err
}

// ZiyaretciEskiSil son görülmesi `gun` günden eski satırları siler ve silinen
// satır sayısını döndürür.
//
// NEDEN: KVKK md. 4 ve 7 — kişisel veri, işlendiği amaç için gereken süre
// kadar tutulur. Aydınlatma metni konum/IP kaydı için 12 ay süre bildiriyor;
// bu işlev o sözü uygular (bkz. httpapi KonumTemizlikBaslat).
func ZiyaretciEskiSil(ctx context.Context, pool *pgxpool.Pool, gun int) (int64, error) {
	if pool == nil || gun <= 0 {
		return 0, nil
	}
	t, err := pool.Exec(ctx,
		`DELETE FROM visitor_locations WHERE son_görülme < now() - make_interval(days => $1)`, gun)
	if err != nil {
		return 0, err
	}
	return t.RowsAffected(), nil
}

// TrafikRiskEskiSil trafik güvenliği kayıtlarında (visitor_risk, migration
// 0131) son görülmesi `gun` günden eski satırları siler. Aynı saklama
// süresi: bu tablo da IP ve tarayıcı kimliği tutuyor.
func TrafikRiskEskiSil(ctx context.Context, pool *pgxpool.Pool, gun int) (int64, error) {
	if pool == nil || gun <= 0 {
		return 0, nil
	}
	t, err := pool.Exec(ctx,
		`DELETE FROM visitor_risk WHERE son_gorulme < now() - make_interval(days => $1)`, gun)
	if err != nil {
		return 0, err
	}
	return t.RowsAffected(), nil
}

// ZiyaretciOzeti panelin tek seferde okuduğu toplu görünüm.
type ZiyaretciOzeti struct {
	Toplam         int64 // benzersiz ziyaretçi (satır)
	Son24          int64
	Konumlu        int64 // koordinatı olan (izin veren)
	UlkeSayisi     int
	UlkeDagilim    []ZiyaretciUlke
	SehirDagilim   []ZiyaretciSehir
	SonGelenler    []ZiyaretciSatir
	Konumlular     []ZiyaretciSatir // koordinatlılar, son görülene göre
	IzinOraniYuzde int
}

type ZiyaretciUlke struct {
	Kod  string
	Adet int64
	Pay  float64 // 0..100
}

type ZiyaretciSehir struct {
	Ulke  string
	Sehir string
	Adet  int64
}

type ZiyaretciSatir struct {
	VisitorID  string
	Ulke       string
	Sehir      string
	Enlem      *float64
	Boylam     *float64
	DogrulukM  *int
	Kaynak     string
	Izin       bool
	SonGorulme time.Time
	MaskeliIP  string
}

// ZiyaretciOzet son `gun` günün coğrafya özetini üretir.
//
// TEK TURDA OKUMA: panel açılışında 6 ayrı sorgu yerine birkaç ardışık
// sorgu; hepsi indeksli (ix_visitor_konum_son / _ulke / _koord).
func ZiyaretciOzet(ctx context.Context, pool *pgxpool.Pool, gun int) (ZiyaretciOzeti, error) {
	var o ZiyaretciOzeti
	if pool == nil {
		return o, nil
	}
	if gun <= 0 {
		gun = 30
	}

	if err := pool.QueryRow(ctx, `
		SELECT count(*),
		       count(*) FILTER (WHERE son_görülme >= now() - interval '24 hours'),
		       count(*) FILTER (WHERE enlem IS NOT NULL AND boylam IS NOT NULL),
		       count(DISTINCT ulke) FILTER (WHERE ulke IS NOT NULL)
		  FROM visitor_locations
		 WHERE son_görülme >= now() - make_interval(days => $1)`,
		gun).Scan(&o.Toplam, &o.Son24, &o.Konumlu, &o.UlkeSayisi); err != nil {
		return o, err
	}
	if o.Toplam > 0 {
		o.IzinOraniYuzde = int((o.Konumlu*100 + o.Toplam/2) / o.Toplam)
	}

	// Ülke kırılımı — en kalabalık 12.
	rows, err := pool.Query(ctx, `
		SELECT COALESCE(ulke,'??') AS kod, count(*) AS adet
		  FROM visitor_locations
		 WHERE son_görülme >= now() - make_interval(days => $1)
		 GROUP BY 1
		 ORDER BY 2 DESC
		 LIMIT 12`, gun)
	if err != nil {
		return o, err
	}
	for rows.Next() {
		var u ZiyaretciUlke
		if err := rows.Scan(&u.Kod, &u.Adet); err == nil {
			if o.Toplam > 0 {
				u.Pay = float64(u.Adet) * 100 / float64(o.Toplam)
			}
			o.UlkeDagilim = append(o.UlkeDagilim, u)
		}
	}
	rows.Close()

	// Şehir kırılımı — yalnız dolu şehirler.
	rows, err = pool.Query(ctx, `
		SELECT COALESCE(ulke,'??'), sehir, count(*)
		  FROM visitor_locations
		 WHERE son_görülme >= now() - make_interval(days => $1)
		   AND sehir IS NOT NULL AND sehir <> ''
		 GROUP BY 1,2
		 ORDER BY 3 DESC
		 LIMIT 12`, gun)
	if err != nil {
		return o, err
	}
	for rows.Next() {
		var s ZiyaretciSehir
		if err := rows.Scan(&s.Ulke, &s.Sehir, &s.Adet); err == nil {
			o.SehirDagilim = append(o.SehirDagilim, s)
		}
	}
	rows.Close()

	oku := func(sql string) ([]ZiyaretciSatir, error) {
		rs, err := pool.Query(ctx, sql, gun)
		if err != nil {
			return nil, err
		}
		defer rs.Close()
		var out []ZiyaretciSatir
		for rs.Next() {
			var s ZiyaretciSatir
			var ip *string
			if err := rs.Scan(&s.VisitorID, &s.Ulke, &s.Sehir, &s.Enlem, &s.Boylam,
				&s.DogrulukM, &s.Kaynak, &s.Izin, &s.SonGorulme, &ip); err != nil {
				continue
			}
			if ip != nil {
				s.MaskeliIP = *ip
			}
			out = append(out, s)
		}
		return out, rs.Err()
	}

	o.SonGelenler, _ = oku(`
		SELECT visitor_id::text, COALESCE(ulke,''), COALESCE(sehir,''),
		       enlem, boylam, dogruluk_m, kaynak, izin, son_görülme, host(ip)
		  FROM visitor_locations
		 WHERE son_görülme >= now() - make_interval(days => $1)
		 ORDER BY son_görülme DESC
		 LIMIT 40`)

	o.Konumlular, _ = oku(`
		SELECT visitor_id::text, COALESCE(ulke,''), COALESCE(sehir,''),
		       enlem, boylam, dogruluk_m, kaynak, izin, son_görülme, host(ip)
		  FROM visitor_locations
		 WHERE son_görülme >= now() - make_interval(days => $1)
		   AND enlem IS NOT NULL AND boylam IS NOT NULL
		 ORDER BY son_görülme DESC
		 LIMIT 40`)

	return o, nil
}

// ZiyaretciVar mı — panel kapısı için (tek ziyaretçi).
func ZiyaretciVar(ctx context.Context, pool *pgxpool.Pool, visitorID string) (izn bool, ulke string, err error) {
	if pool == nil || visitorID == "" {
		return false, "", nil
	}
	err = pool.QueryRow(ctx,
		`SELECT izin, COALESCE(ulke,'') FROM visitor_locations WHERE visitor_id = $1::uuid`,
		visitorID).Scan(&izn, &ulke)
	return izn, ulke, err
}
