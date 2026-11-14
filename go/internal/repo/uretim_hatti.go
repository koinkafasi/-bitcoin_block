package repo

import (
	"context"
	"time"
)

// Üretim hattı ölçümleri — yönetim panelindeki canlı gösterge.
//
// NEDEN VAR: hattın çalışıp çalışmadığı bugüne kadar ancak veritabanına
// elle sorgu atarak anlaşılabiliyordu. Bu oturumda art arda ÜÇ sessiz
// arıza çıktı ve üçü de yalnız elle ölçümle görüldü:
//   - medya kaydı yabancı anahtar hatası (iki işçi aylarca hiçbir şey
//     kaydedememiş, hata WARN olarak loglanıp tur devam ediyordu),
//   - ölü adresler kuyruğu tıkıyordu (49 kayıt 2012'lik kuyruğu
//     durdurdu),
//   - sağlayıcı zaman aşımları (günde ~103 iş kaybı, akşam saatlerinde).
//
// Üçü de "her şey yeşil görünüyor ama hiçbir şey ilerlemiyor" türünden.
// Bu sorgular o körlüğü kapatıyor: panelde İLERLEME ve KAYIP yan yana.

// UretimOzet hattın tek bakışta durumu.
type UretimOzet struct {
	// Calisan şu an işlenen işler (status='running').
	Calisan int
	// Kuyrukta sıradaki işler (status='queued').
	Kuyrukta int
	// BekleyenTaslak gövdesi yazılmayı bekleyen içerik.
	BekleyenTaslak int

	// Son24Tamam / Son24Hata son 24 saatteki iş sonuçları.
	Son24Tamam int
	Son24Hata  int

	// Son24KuyrugaGiren son 24 saatte kuyruğa GİREN iş.
	//
	// NEDEN GEREKLİ: panelin ilk sürümü "kuyruk boş + bekleyen taslak
	// çok" durumunu "besleme kesik" diye yorumluyordu. 14 Kasım 2026
	// ölçümü bunu çürüttü: kuyruk akıyordu (running 2, queued 1) ve 24
	// saatte 371 iş girmişti — kuyruk kısa görünüyordu çünkü işler
	// girdiği hızda tüketiliyor. Taslaklar AI'ı değil YAYINI bekliyordu.
	// Gerçek açlıkla kısa kuyruğu ayırmak için girişi de saymak gerekti.
	Son24KuyrugaGiren int

	// Son24Uretilen son 24 saatte üretilen özgün içerik.
	Son24Uretilen int
	// Son24MedyanKelime üretilen içeriğin medyan uzunluğu.
	//
	// DOĞRU SAYIM: `strict $.**.text`. Lax modda jsonpath diziyi
	// otomatik açıp her metni İKİ KEZ sayıyor; bu oturumda tüm uzunluk
	// ölçümleri bir kez bu yüzden iki kat yüksek çıktı.
	Son24MedyanKelime int
}

// UretimSaat saatlik üretim/kayıp satırı.
type UretimSaat struct {
	Saat   time.Time
	Tamam  int
	Hata   int
	Kelime int // o saatte üretilen içeriğin medyan uzunluğu
}

// UretimHata gruplanmış başarısızlık nedeni.
type UretimHata struct {
	Neden string
	Adet  int
	Son   time.Time
}

// UretimCalisan panelde "şu an ne işleniyor" satırı.
type UretimCalisan struct {
	JobID     int64
	ContentID int64
	Baslik    string
	Eylem     string
	Baslangic time.Time
}

// UretimOzetOku tek sorgu turuyla özeti doldurur.
func UretimOzetOku(ctx context.Context, pool DB) (UretimOzet, error) {
	var o UretimOzet

	if err := pool.QueryRow(ctx, `
		SELECT
		  count(*) FILTER (WHERE status = 'running'),
		  count(*) FILTER (WHERE status = 'queued'),
		  count(*) FILTER (WHERE created_at > now() - interval '24 hours')
		FROM ai_jobs`).Scan(&o.Calisan, &o.Kuyrukta, &o.Son24KuyrugaGiren); err != nil {
		return o, err
	}

	if err := pool.QueryRow(ctx, `
		SELECT
		  count(*) FILTER (WHERE status = 'completed'),
		  count(*) FILTER (WHERE status = 'failed')
		FROM ai_jobs
		WHERE created_at > now() - interval '24 hours'`).
		Scan(&o.Son24Tamam, &o.Son24Hata); err != nil {
		return o, err
	}

	// Bekleyen taslak: gövdesi olmayan ya da AI'ya hiç girmemiş içerik
	// sayısı panelin "besleme var mı" sorusunu yanıtlıyor. Kuyruk boş
	// VE bekleyen çoksa hat işsiz değil, besleme kesik demektir.
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM contents
		 WHERE status <> 'yayinda'`).Scan(&o.BekleyenTaslak); err != nil {
		return o, err
	}

	if err := pool.QueryRow(ctx, `
		WITH g AS (
		  SELECT array_length(string_to_array(regexp_replace(trim(COALESCE(
		           (SELECT string_agg(x #>> '{}', ' ')
		              FROM jsonb_array_elements(
		                     jsonb_path_query_array(c.detail, 'strict $.**.text')) x), '')),
		           '\s+', ' ', 'g'), ' '), 1) AS kelime
		    FROM contents c
		   WHERE c.translation_of IS NULL
		     AND c.created_at > now() - interval '24 hours'
		)
		SELECT count(*),
		       COALESCE(percentile_disc(0.5) WITHIN GROUP (ORDER BY kelime), 0)
		  FROM g WHERE kelime IS NOT NULL`).
		Scan(&o.Son24Uretilen, &o.Son24MedyanKelime); err != nil {
		return o, err
	}
	return o, nil
}

// UretimSaatlikOku son 24 saatin saatlik dökümü.
//
// SAATLİK KIRILIM ŞART: bu oturumda sağlayıcı zaman aşımlarının
// yalnız AKŞAM saatlerinde toplandığı (18:00-23:00 arası 87 hata,
// 00:00-09:00 arası sıfır) ancak saatlik dağılıma bakınca görüldü.
// Günlük toplam bunu gizliyordu.
func UretimSaatlikOku(ctx context.Context, pool DB) ([]UretimSaat, error) {
	rows, err := pool.Query(ctx, `
		SELECT date_trunc('hour', created_at) AS saat,
		       count(*) FILTER (WHERE status = 'completed'),
		       count(*) FILTER (WHERE status = 'failed')
		  FROM ai_jobs
		 WHERE created_at > now() - interval '24 hours'
		 GROUP BY 1 ORDER BY 1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []UretimSaat
	for rows.Next() {
		var s UretimSaat
		if err := rows.Scan(&s.Saat, &s.Tamam, &s.Hata); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// UretimHatalariOku başarısızlık nedenlerini gruplar.
//
// NEDEN İLK 80 KARAKTER: hata metinleri adres ve zaman damgası
// taşıyor, tam metinle gruplamak her satırı ayrı gösterirdi. 80
// karakter sağlayıcı/hata türünü ayırmaya yetiyor.
func UretimHatalariOku(ctx context.Context, pool DB, limit int) ([]UretimHata, error) {
	if limit <= 0 {
		limit = 6
	}
	rows, err := pool.Query(ctx, `
		SELECT left(COALESCE(error, '(boş)'), 80) AS neden,
		       count(*), max(created_at)
		  FROM ai_jobs
		 WHERE status = 'failed'
		   AND created_at > now() - interval '24 hours'
		 GROUP BY 1 ORDER BY 2 DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []UretimHata
	for rows.Next() {
		var h UretimHata
		if err := rows.Scan(&h.Neden, &h.Adet, &h.Son); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

// UretimCalisanlarOku şu an işlenen işleri içerik başlığıyla verir.
func UretimCalisanlarOku(ctx context.Context, pool DB, limit int) ([]UretimCalisan, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := pool.Query(ctx, `
		SELECT j.id, COALESCE(j.content_id, 0), COALESCE(c.title, '(içerik yok)'),
		       COALESCE(j.action, ''), j.started_at
		  FROM ai_jobs j
		  LEFT JOIN contents c ON c.id = j.content_id
		 WHERE j.status = 'running'
		 ORDER BY j.started_at ASC NULLS LAST
		 LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []UretimCalisan
	for rows.Next() {
		var c UretimCalisan
		var bas *time.Time
		if err := rows.Scan(&c.JobID, &c.ContentID, &c.Baslik, &c.Eylem, &bas); err != nil {
			return nil, err
		}
		if bas != nil {
			c.Baslangic = *bas
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// UretimGorselDurumu medya aktarımının ilerlemesi.
func UretimGorselDurumu(ctx context.Context, pool DB) (aktarilan, toplam int, err error) {
	err = pool.QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE media_id IS NOT NULL), count(*)
		  FROM source_images WHERE bytes > 0`).Scan(&aktarilan, &toplam)
	return
}

// UretimKalite üretilen metnin YAYINLANABİLİR olup olmadığını ölçer.
//
// NEDEN AYRI BİR ÖLÇÜM: panelin ilk sürümü yalnız "kaç iş bitti" ve
// "medyan kaç kelime" gösteriyordu. 14 Kasım 2026'da ölçtüğümde asıl
// sorun bu ikisinin arasından sızdı:
//
//	BEKLEYEN TASLAK (2003) : medyan  71 kelime — 1868'i 150'nin ALTINDA
//	YAYINDA (bu ay, 1105)  : medyan  98 kelime —  675'i 150'nin ALTINDA
//
// Yani hat çalışıyordu, iş sayıları yeşildi, ama ürettiği şeyin %93'ü
// yayınlanabilir değildi — ve 675 ince içerik çoktan yayınlanmıştı.
// Arama sıralamasında öne çıkma hedefiyle 70-100 kelimelik yazıları
// toplu yayınlamak birbirini götüren iki iş; Google bunu "scaled
// content abuse" başlığında değerlendiriyor. Bu yüzden panelde iş
// sayısının YANINDA ne ürettiği de durmalı.
type UretimKalite struct {
	// Taslak bekleyen taslak sayısı, InceTaslak bunların kaçının
	// eşiğin altında kaldığı, HedefteTaslak kaçının hedef bandı tuttuğu.
	Taslak        int
	InceTaslak    int
	HedefteTaslak int
	MedyanTaslak  int

	// YayindaAy bu ay yayınlanan, InceYayin bunların kaçının eşiğin
	// altında olduğu. Yayınlanmış ince içerik taslaktan daha acil:
	// taslak düzeltilebilir, yayınlanan zarar vermeye başlamıştır.
	YayindaAy   int
	InceYayin   int
	MedyanYayin int
}

// InceEsik altında kalan metni "ince içerik" sayan kelime sınırı.
// 150, haber bandının (350-550) alt sınırının epey altında bilinçli
// seçildi: buranın altı tartışmasız ince, arası tartışılır.
const InceEsik = 150

// uretimKelimeSayimi bir içeriğin kelime sayısını veren alt sorgu.
//
// `strict` ŞART: lax modda jsonpath diziyi otomatik açıp her metin
// düğümünü İKİ KEZ sayıyor. Bu oturumda tüm uzunluk ölçümleri bir kez
// bu yüzden iki kat yüksek çıktı ve bantlar yanlış ayarlandı.
const uretimKelimeSayimi = `
	(SELECT count(*) FROM (
	   SELECT regexp_split_to_table(string_agg(t::text, ' '), '\s+') AS w
	   FROM jsonb_path_query(c.detail, 'strict $.**.text') t
	 ) q WHERE w <> '')`

// UretimKaliteOku taslak ve yayın tarafının uzunluk dağılımını okur.
func UretimKaliteOku(ctx context.Context, pool DB) (UretimKalite, error) {
	var k UretimKalite

	sorgu := `
		WITH t AS (
		  SELECT ` + uretimKelimeSayimi + ` AS kelime
		  FROM contents c
		  WHERE c.status = 'taslak' AND jsonb_typeof(c.detail) IS NOT NULL
		)
		SELECT
		  count(*),
		  count(*) FILTER (WHERE kelime < $1),
		  count(*) FILTER (WHERE kelime >= $2),
		  COALESCE(round(percentile_disc(0.5) WITHIN GROUP (ORDER BY kelime)), 0)
		FROM t`
	if err := pool.QueryRow(ctx, sorgu, InceEsik, UretimHedefAlt).
		Scan(&k.Taslak, &k.InceTaslak, &k.HedefteTaslak, &k.MedyanTaslak); err != nil {
		return k, err
	}

	// YAYIN TARAFI bu ayla sınırlı: eski arşivi bugünün hattına
	// yazmak yanlış olur, panel "şu an ne yapıyoruz" sorusunu yanıtlıyor.
	sorgu = `
		WITH y AS (
		  SELECT ` + uretimKelimeSayimi + ` AS kelime
		  FROM contents c
		  WHERE c.status = 'yayinda'
		    AND COALESCE(c.published_at, c.created_at) >= date_trunc('month', now())
		    AND jsonb_typeof(c.detail) IS NOT NULL
		)
		SELECT
		  count(*),
		  count(*) FILTER (WHERE kelime < $1),
		  COALESCE(round(percentile_disc(0.5) WITHIN GROUP (ORDER BY kelime)), 0)
		FROM y`
	if err := pool.QueryRow(ctx, sorgu, InceEsik).
		Scan(&k.YayindaAy, &k.InceYayin, &k.MedyanYayin); err != nil {
		return k, err
	}
	return k, nil
}

// UretimHedefAlt haber bandının alt sınırı (aieditor.UzunlukHedefi ile
// aynı sayı). repo katmanı aieditor'ü içe alamaz — döngü olur — bu
// yüzden sabit burada TEKRARLANIYOR. İkisinin ayrışmasını önleyen test
// httpapi tarafında (TestUretimHedefAltAieditorIleAyni): orası hem
// repo'yu hem aieditor'ü görebiliyor.
const UretimHedefAlt = 350
