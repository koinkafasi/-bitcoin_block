package aieditor

import (
	"context"
	"log/slog"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgxpool"

	"theblock/site/internal/ai"
	"theblock/site/internal/repo"
)

// ---- Sınır dışı SEO başlıklarının onarımı ----
//
// ÖLÇÜM (13 Kasım 2026, canlı veritabanı — 3836 yayın içeriği):
//
//	seo_title uzunluk dilimleri
//	  <=50 kisa         : 1323
//	  51-60 ideal       :  971
//	  61-70 kesilir     :  703
//	  70+  fena kesilir :  788   (en uzunu 141 karakter)
//
// Yani 1491 içerik (%38) arama sonucunda YARIM başlıkla görünüyor.
//
// TEŞHİS: bu bir kod kusuru değil. Üretim zinciri doğru çalışıyor —
// SeoBaslikSertMax 70 ve son haftanın 460 başlığından 458'i 60 karakterin
// altında. 70+ olanların TAMAMI 90 günden eski ve 782'sinde
// `seo_title = title`; yani eski WordPress göçünde haber başlığı olduğu
// gibi SEO alanına kopyalanmış ("Crypto Gibraltar 2023..." gibi).
//
// Mevcut SeoBosTaslaklar bunları göremiyor: yalnız `status='taslak'` ve
// alanı BOŞ olanları alıyor. Dolu ama sınır dışı olana kimse dokunmuyordu.
//
// Bu dosya o boşluğu kapatır: yayındaki uzun başlıkları AI ile yeniden
// üretir — kırpmaz. Kırpmak cümleyi ortadan keser ("...Kripto Para
// Yatırımcılarının "Tüm Paralarını Kayb"); yeniden üretim ise 60 karaktere
// sığan anlamlı bir başlık yazar.

// SeoOnarimGunlukLimit bir günde en çok kaç yayın başlığı onarılır.
//
// NEDEN SINIRLI: yayındaki içeriğin meta başlığını değiştirmek arama
// görünürlüğünü etkiler. 1491 içeriği tek seferde değiştirmek, etkisi
// ölçülemeyen büyük bir hareket olurdu. Günde 20 ile ~75 günde tamamlanır
// ve her hafta Search Console'da etkisi izlenebilir.
const SeoOnarimGunlukLimit = 20

// SeoOnarimTurLimiti tek turda işlenecek içerik sayısı.
const SeoOnarimTurLimiti = 5

// SeoSinirDisiYayinlar SEO başlığı sert sınırı aşan YAYINDA içerikleri
// döndürür.
//
// En uzundan başlar: 141 karakterlik başlık 71 karakterlikten çok daha
// fazla kaybediyor, önce o düzelsin.
func SeoSinirDisiYayinlar(ctx context.Context, pool *pgxpool.Pool, limit int) ([]repo.Content, error) {
	if limit <= 0 || limit > 50 {
		limit = SeoOnarimTurLimiti
	}
	rows, err := pool.Query(ctx, `
		SELECT id FROM contents
		 WHERE status = 'yayinda'
		   AND length(seo_title) > $1
		   AND COALESCE(title, '') <> ''
		   AND (seo_onarim_at IS NULL)
		 ORDER BY length(seo_title) DESC
		 LIMIT $2`, SeoBaslikSertMax, limit)
	if err != nil {
		return nil, err
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]repo.Content, 0, len(ids))
	for _, id := range ids {
		c, err := repo.GetContent(ctx, pool, id)
		if err != nil {
			continue
		}
		out = append(out, c)
	}
	return out, nil
}

// SeoOnarimBugunSayisi bugün onarılan başlık sayısını verir.
func SeoOnarimBugunSayisi(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	var n int
	err := pool.QueryRow(ctx, `
		SELECT count(*) FROM contents
		 WHERE seo_onarim_at >= date_trunc('day', now())`).Scan(&n)
	return n, err
}

// seoOnarimDamgala onarım damgasını yazar.
//
// Damga iki işe yarar: aynı içeriğin tekrar tekrar onarılmasını önler
// (AI her çağrıda biraz farklı başlık üretir, sonsuz döngü olurdu) ve
// günlük limit sayacını besler.
func seoOnarimDamgala(ctx context.Context, pool *pgxpool.Pool, id int64) error {
	_, err := pool.Exec(ctx, `
		UPDATE contents SET seo_onarim_at = now() WHERE id = $1`, id)
	return err
}

// RunSeoOnarim sınır dışı SEO başlıklarını kademeli olarak onarır.
//
// Kota penceresine duyarlı: gece kotası tazeyken daha sık tur atar, kota
// tükendiğinde hiç çalışmaz (bkz. internal/ai/kota.go).
func RunSeoOnarim(ctx context.Context, pool *pgxpool.Pool, aiclient *ai.Client, logger *slog.Logger, every time.Duration) {
	if every <= 0 {
		every = 15 * time.Minute
	}
	logger.Info("seo onarım işçisi başladı", "her", every.String(),
		"gunluk_limit", SeoOnarimGunlukLimit, "sert_max", SeoBaslikSertMax)

	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			logger.Info("seo onarım işçisi durdu")
			return
		case <-ticker.C:
			seoOnarimTuru(ctx, pool, aiclient, logger)
		}
	}
}

func seoOnarimTuru(ctx context.Context, pool *pgxpool.Pool, aiclient *ai.Client, logger *slog.Logger) {
	if aiclient == nil || !aiclient.Enabled() {
		return
	}
	// Kota tükendiyse hiç deneme: 429 alıp boşa iş üretmek yerine
	// sıfırlamayı bekle.
	if aiclient.KotaTukendi() {
		return
	}

	bugun, err := SeoOnarimBugunSayisi(ctx, pool)
	if err != nil {
		logger.Warn("seo onarım: günlük sayaç okunamadı", "error", err)
		return
	}
	if bugun >= SeoOnarimGunlukLimit {
		return
	}

	kalan := SeoOnarimGunlukLimit - bugun
	if kalan > SeoOnarimTurLimiti {
		kalan = SeoOnarimTurLimiti
	}

	liste, err := SeoSinirDisiYayinlar(ctx, pool, kalan)
	if err != nil {
		logger.Warn("seo onarım: sınır dışı başlıklar okunamadı", "error", err)
		return
	}
	if len(liste) == 0 {
		return
	}

	onarilan := 0
	for _, c := range liste {
		if ctx.Err() != nil {
			return
		}
		oneri, err := SeoUret(ctx, aiclient, c)
		if err != nil {
			// Kota hatasıysa turu bitir: sıradakiler de aynı hatayı alır.
			if aiclient.KotaTukendi() {
				logger.Info("seo onarım: kota doldu, tur bitti", "onarilan", onarilan)
				return
			}
			logger.Warn("seo onarım: üretilemedi", "icerik", c.ID, "error", err)
			continue
		}
		oneri = SeoDogrula(oneri)
		if oneri.Baslik == "" {
			continue
		}
		// force=true: alan DOLU ama sınır dışı; üzerine yazmak işin özü.
		if _, err := SeoYaz(ctx, pool, c.ID, oneri, true); err != nil {
			logger.Warn("seo onarım: yazılamadı", "icerik", c.ID, "error", err)
			continue
		}
		if err := seoOnarimDamgala(ctx, pool, c.ID); err != nil {
			logger.Warn("seo onarım: damgalanamadı", "icerik", c.ID, "error", err)
		}
		onarilan++
		eskiUzunluk := 0
		if c.SeoTitle != nil {
			eskiUzunluk = utf8.RuneCountInString(*c.SeoTitle)
		}
		logger.Info("seo başlığı onarıldı", "icerik", c.ID,
			"eski_uzunluk", eskiUzunluk,
			"yeni_uzunluk", utf8.RuneCountInString(oneri.Baslik))
	}
	if onarilan > 0 {
		logger.Info("seo onarım turu", "onarilan", onarilan, "bugun", bugun+onarilan,
			"gunluk_limit", SeoOnarimGunlukLimit)
	}
}
