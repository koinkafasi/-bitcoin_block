package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
)

// ---- Skor eşiğine göre otomatik yayın ----
//
// AMAÇ: Google Trend'de yükselen ve güven skoru yüksek içerikleri insan
// onayı beklemeden yayına almak; arama motorunda erken görünmek.
//
// ÖLÇÜM (13 Kasım 2026, canlı veritabanı):
//
//	taslak score_guven     min 0   / ort 62.6 / maks 95
//	taslak score_trend     min 0   / ort 10.0 / maks 100
//	taslak ai_score        min 0   / ort 80.8 / maks 95
//	YAYINDAKİ score_guven  ort 51.4 / medyan 50.0
//
//	güven dilimleri (1869 taslak):
//	  80+   : 107
//	  70-79 : 324
//	  60-69 : 1281   <- kütle burada
//	  50-59 : 161
//	  <50   : 5
//
// İki karar bu ölçümden çıktı:
//
//  1. EŞİK 70. Yayındaki içeriğin güven ortalaması 51,4; yani 70 eşiği
//     mevcut yayın çizgisinin belirgin ÜSTÜNDE. 60'a indirmek havuzu
//     1281 taslak büyütürdü — kalite çizgisi yayındakilere yaklaşırdı.
//
//  2. TREND ŞART DEĞİL, BONUS. score_trend ortalaması 10: çoğu içerikte
//     trend eşleşmesi yok. "güven>=70 VE trend>=60" aday havuzunu 429'dan
//     39'a düşürüyordu. Trend artık yalnız SIRALAMADA kullanılıyor:
//     trendli içerik önce çıkar, trendsiz olan da sırasını bekler.

// OtomatikYayinAyar panelden okunan otomatik yayın ayarlarıdır.
type OtomatikYayinAyar struct {
	// Aktif kapalıysa hiçbir şey yayınlanmaz. VARSAYILAN KAPALI:
	// içeriğin insan onayı olmadan siteye çıkması bir politika kararıdır,
	// kodun varsayılanı olmamalı.
	Aktif bool
	// GuvenEsigi bu değerin altındaki içerik yayınlanmaz.
	GuvenEsigi float64
	// GunlukLimit bir günde en çok kaç içerik otomatik yayınlanır.
	//
	// NEDEN SINIR: eşiği geçen 429 taslak var. Hepsi bir anda yayına
	// çıksa hem okuyucu akışı anlamsız görünür hem arama motoruna ani
	// kütle gider (kalitesiz toplu üretim sinyali). Kademeli akış hem
	// doğal görünür hem her partinin etkisi ölçülebilir.
	GunlukLimit int
	// SeoZorunlu açıkken SEO başlığı/açıklaması boş olan içerik
	// yayınlanmaz. Boş meta ile yayınlamak amacın tam tersi olurdu.
	SeoZorunlu bool
}

// OtomatikYayinVarsayilan güvenli varsayılanları verir.
func OtomatikYayinVarsayilan() OtomatikYayinAyar {
	return OtomatikYayinAyar{
		Aktif:       false, // bilinçli: politika kararı kullanıcıya ait
		GuvenEsigi:  70,    // yayındaki ortalama 51.4'ün belirgin üstü
		GunlukLimit: 12,    // ~2 saatte bir içerik; doğal akış
		SeoZorunlu:  true,
	}
}

// OtomatikYayinAyarOku ayarları veritabanından okur; eksik alanlar
// varsayılanda kalır.
func OtomatikYayinAyarOku(ctx context.Context, pool DB) OtomatikYayinAyar {
	a := OtomatikYayinVarsayilan()

	oku := func(anahtar string) (string, bool) {
		b, err := GetSetting(ctx, pool, "otomatik_yayin", anahtar)
		if err != nil || len(b) == 0 {
			return "", false
		}
		var s string
		if json.Unmarshal(b, &s) == nil {
			return s, true
		}
		return string(b), true
	}

	if v, ok := oku("aktif"); ok {
		a.Aktif = v == "1" || v == "true"
	}
	if v, ok := oku("guven_esigi"); ok {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 && f <= 100 {
			a.GuvenEsigi = f
		}
	}
	if v, ok := oku("gunluk_limit"); ok {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 && n <= 200 {
			a.GunlukLimit = n
		}
	}
	if v, ok := oku("seo_zorunlu"); ok {
		a.SeoZorunlu = v == "1" || v == "true"
	}
	return a
}

// OtomatikYayinAday yayına alınmaya hazır bir taslaktır.
type OtomatikYayinAday struct {
	ID        int64
	Title     string
	Locale    string
	GuvenSkor float64
	TrendSkor float64
	AISkor    float64
}

// OtomatikYayinAdaylari eşiği geçen taslakları TREND SKORU YÜKSEK OLAN
// ÖNCE olacak şekilde döndürür.
//
// Sıralama gerekçesi: trend eşleşmesi olan içerik arama hacminin zirvesine
// yakın; onu bekletmek fırsatı kaçırmak olur. Trendsiz içerik (ortalamanın
// gösterdiği gibi çoğunluk) güven skoruna göre sıralanır ve sırasını
// bekler.
func OtomatikYayinAdaylari(ctx context.Context, pool DB, a OtomatikYayinAyar, limit int) ([]OtomatikYayinAday, error) {
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	seoKosul := ""
	if a.SeoZorunlu {
		seoKosul = `
		   AND btrim(COALESCE(seo_title, '')) <> ''
		   AND btrim(COALESCE(seo_description, '')) <> ''`
	}

	sorgu := fmt.Sprintf(`
		SELECT id, title, COALESCE(locale, 'tr'),
		       COALESCE(score_guven, 0), COALESCE(score_trend, 0), COALESCE(ai_score, 0)
		  FROM contents
		 WHERE status = 'taslak'
		   AND COALESCE(score_guven, 0) >= $1%s
		 ORDER BY COALESCE(score_trend, 0) DESC,
		          COALESCE(score_guven, 0) DESC,
		          created_at DESC
		 LIMIT $2`, seoKosul)

	rows, err := pool.Query(ctx, sorgu, a.GuvenEsigi, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []OtomatikYayinAday
	for rows.Next() {
		var d OtomatikYayinAday
		if err := rows.Scan(&d.ID, &d.Title, &d.Locale,
			&d.GuvenSkor, &d.TrendSkor, &d.AISkor); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// OtomatikYayinBugunSayisi bugün otomatik yayınlanan içerik sayısını verir.
//
// Günlük limiti uygulamak için: sayaç ayrı bir tabloda değil, içeriğin
// kendi `auto_published_at` damgasında tutulur — tek kaynak, kayma yok.
func OtomatikYayinBugunSayisi(ctx context.Context, pool DB) (int, error) {
	var n int
	err := pool.QueryRow(ctx, `
		SELECT count(*) FROM contents
		 WHERE auto_published_at IS NOT NULL
		   AND auto_published_at >= date_trunc('day', now())`).Scan(&n)
	return n, err
}

// OtomatikYayinla bir taslağı yayına alır ve otomatik yayın damgasını
// yazar.
//
// Damga iki işe yarar: günlük limit sayacı ve denetim izi — hangi içeriğin
// insan onayı olmadan çıktığı sonradan görülebilir.
func OtomatikYayinla(ctx context.Context, pool DB, id int64) error {
	_, err := pool.Exec(ctx, `
		UPDATE contents
		   SET status            = 'yayinda',
		       published_at      = COALESCE(published_at, now()),
		       auto_published_at = now(),
		       updated_at        = now()
		 WHERE id = $1
		   AND status = 'taslak'`, id)
	return err
}
