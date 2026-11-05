package ceviri

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"theblock/site/internal/ai"
	"theblock/site/internal/platform"
	"theblock/site/internal/repo"
)

// Editörden tetiklenen AI çeviri istekleri.
//
// Neden kuyruk: AI çeviri 20–160 sn sürüyor. İsteği HTTP yanıtı içinde
// bekletmek iki sert sınırı aşıyordu (ölçüm: 28 paragraflık haber 160 sn,
// istek HTTP 000 ile düştü):
//   - sunucu WriteTimeout: 60 sn
//   - Cloudflare proxy: ~100 sn (524)
//
// Düğme artık yalnız `ceviri_istek` satırı yazar; ceviri.Run döngüsü bu
// satırları otomatik kuyruktan ÖNCE işler; panel 2 sn'de bir durumu yoklar.

// İstek durumları (DB'de metin olarak durur).
const (
	IstekBekliyor  = "bekliyor"
	IstekCalisiyor = "calisiyor"
	IstekBitti     = "bitti"
	IstekHata      = "hata"
)

// Istek tek bir editör çeviri isteğidir.
type Istek struct {
	ID        int64
	ContentID int64
	Locale    string
	Durum     string
	SonucID   int64
	Hata      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// IstekEkle içerik+dil için çeviri isteği yazar.
//
// Aynı çift için tek satır vardır (benzersiz indeks). Bekleyen/çalışan bir
// istek varsa dokunulmaz: düğmeye iki kez basmak iki çeviri üretmemeli.
// Bitmiş/hatalı istek yeniden denenebilsin diye sıfırlanır.
func IstekEkle(ctx context.Context, pool *pgxpool.Pool, contentID int64, locale string) error {
	if pool == nil {
		return errors.New("veritabanı yok")
	}
	if contentID <= 0 {
		return errors.New("geçersiz içerik")
	}
	locale = strings.TrimSpace(locale)
	if locale == "" {
		return errors.New("dil belirtilmedi")
	}
	_, err := pool.Exec(ctx, `
		INSERT INTO ceviri_istek (content_id, locale, durum)
		VALUES ($1, $2, 'bekliyor')
		ON CONFLICT (content_id, locale) DO UPDATE
		   SET durum = 'bekliyor', hata = NULL, sonuc_id = NULL, updated_at = now()
		 WHERE ceviri_istek.durum IN ('bitti', 'hata')`, contentID, locale)
	return err
}

// IstekDurum içerik+dil için isteğin son durumunu döndürür.
func IstekDurum(ctx context.Context, pool *pgxpool.Pool, contentID int64, locale string) (Istek, bool, error) {
	var i Istek
	if pool == nil {
		return i, false, errors.New("veritabanı yok")
	}
	err := pool.QueryRow(ctx, `
		SELECT id, content_id, locale, durum, COALESCE(sonuc_id, 0),
		       COALESCE(hata, ''), created_at, updated_at
		FROM ceviri_istek
		WHERE content_id = $1 AND locale = $2`, contentID, locale).
		Scan(&i.ID, &i.ContentID, &i.Locale, &i.Durum, &i.SonucID, &i.Hata, &i.CreatedAt, &i.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return i, false, nil
	}
	if err != nil {
		return i, false, err
	}
	return i, true, nil
}

// istekAl sıradaki bekleyen isteği sahiplenir.
//
// SKIP LOCKED: süreç iki kez başlarsa (deploy sırasında kısa örtüşme) aynı
// istek iki kez işlenmez.
func istekAl(ctx context.Context, pool *pgxpool.Pool) (*Istek, error) {
	var i Istek
	err := pool.QueryRow(ctx, `
		UPDATE ceviri_istek
		   SET durum = 'calisiyor', updated_at = now()
		 WHERE id = (
			SELECT id FROM ceviri_istek
			 WHERE durum = 'bekliyor'
			 ORDER BY id
			 LIMIT 1
			 FOR UPDATE SKIP LOCKED
		 )
		RETURNING id, content_id, locale`).
		Scan(&i.ID, &i.ContentID, &i.Locale)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &i, nil
}

// istekIsle tek isteği işler; başarıda true.
func istekIsle(ctx context.Context, pool *pgxpool.Pool, aiclient *ai.Client, logger *slog.Logger, is Istek, durum string) bool {
	src, err := repo.GetContent(ctx, pool, is.ContentID)
	if err != nil {
		istekBitir(ctx, pool, is.ID, IstekHata, 0, "Kaynak içerik bulunamadı (silinmiş olabilir).")
		logger.Warn("ceviri: istek kaynağı okunamadı", "istek", is.ID, "içerik", is.ContentID, "error", err)
		return false
	}
	// Zaten çeviri varsa AI çağrısı yapılmaz: sonuç mevcut kayda bağlanır.
	if trs, err := repo.TranslationsOf(ctx, pool, src.TranslationRootID()); err == nil {
		for _, t := range trs {
			if t.Locale == is.Locale && t.ID != src.ID {
				istekBitir(ctx, pool, is.ID, IstekBitti, t.ID, "")
				return true
			}
		}
	}
	cev, err := UretTekrar(ctx, aiclient, src, is.Locale, logger)
	if err != nil {
		istekBitir(ctx, pool, is.ID, IstekHata, 0, hataMetni(err))
		logger.Warn("ceviri: istek üretilemedi", "istek", is.ID, "içerik", is.ContentID,
			"dil", is.Locale, "error", err)
		return false
	}
	// Üretim sürerken otomatik tur aynı çeviriyi yazmış olabilir (ölçüm:
	// 3,5 dk süren istek sırasında otomatik tur 554'ü çevirdi). Kaydetmeden
	// önce son kontrol: kopya çeviri oluşturmayalım.
	if trs, err := repo.TranslationsOf(ctx, pool, src.TranslationRootID()); err == nil {
		for _, t := range trs {
			if t.Locale == is.Locale && t.ID != src.ID {
				istekBitir(ctx, pool, is.ID, IstekBitti, t.ID, "")
				logger.Info("ceviri: bu arada üretilmiş çeviri bulundu",
					"istek", is.ID, "yeni", t.ID, "dil", is.Locale)
				return true
			}
		}
	}

	nid, err := Kaydet(ctx, pool, src, is.Locale, cev, durum)
	if err != nil {
		// Kayıt düştüyse nedeni "çeviri zaten var" olabilir: contents
		// tablosunda (translation_of, locale) benzersiz kısıtı YOK, ama
		// otomatik tur aynı çeviriyi bizimle aynı anda yazmış olabilir.
		// Kullanıcıya hata göstermek yanlış olur — çeviri VAR. Mevcut
		// kayda bağla.
		if trs, terr := repo.TranslationsOf(ctx, pool, src.TranslationRootID()); terr == nil {
			for _, t := range trs {
				if t.Locale == is.Locale && t.ID != src.ID {
					istekBitir(ctx, pool, is.ID, IstekBitti, t.ID, "")
					logger.Info("ceviri: istek zaten üretilmiş çeviriye bağlandı",
						"istek", is.ID, "yeni", t.ID, "dil", is.Locale)
					return true
				}
			}
		}
		istekBitir(ctx, pool, is.ID, IstekHata, 0, kayitHataMetni(err))
		logger.Warn("ceviri: istek kaydedilemedi", "istek", is.ID, "içerik", is.ContentID,
			"dil", is.Locale, "error", err)
		return false
	}
	// Etiketler kaynaktan kopyalanır: çeviri aynı konu kümesinde kalmalı.
	// Kök içerik esastır; çevirinin etiketleri kendi kaynağından gelir.
	kaynakID := src.TranslationRootID()
	if tags, err := repo.ContentTagNames(ctx, pool, kaynakID); err == nil && len(tags) > 0 {
		if err := repo.ReplaceContentTags(ctx, pool, nid, tags, platform.Slugify); err != nil {
			logger.Warn("ceviri: etiketler kopyalanamadı", "yeni", nid, "error", err)
		}
	}
	istekBitir(ctx, pool, is.ID, IstekBitti, nid, "")
	logger.Info("ceviri: istek tamamlandı", "istek", is.ID, "kaynak", src.ID, "yeni", nid,
		"dil", is.Locale, "durum", durum, "sağlayıcı", cev.Provider, "model", cev.Model)
	return true
}

// istekBitir isteğin son durumunu yazar.
func istekBitir(ctx context.Context, pool *pgxpool.Pool, id int64, durum string, sonuc int64, hata string) {
	var sonucPtr *int64
	if sonuc > 0 {
		sonucPtr = &sonuc
	}
	if _, err := pool.Exec(ctx, `
		UPDATE ceviri_istek
		   SET durum = $2, sonuc_id = $3, hata = NULLIF($4, ''), updated_at = now()
		 WHERE id = $1`, id, durum, sonucPtr, hata); err != nil {
		// İsteğin kendisi güncellenemezse panel sonsuza kadar "çevriliyor"
		// gösterir; en azından günlükte görünmeli.
		slog.Warn("ceviri: istek durumu yazılamadı", "istek", id, "error", err)
	}
}

// hataMetni AI/üretim hatasını editöre gösterilecek metne çevirir.
func hataMetni(err error) string {
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "yapılandırılmadı"):
		return "AI sağlayıcısı yapılandırılmadı. Yönetim panelinden AI ayarlarını kontrol edin."
	case strings.Contains(msg, "deadline"), strings.Contains(msg, "timeout"), strings.Contains(msg, "context canceled"):
		return "AI sağlayıcısı zamanında yanıt vermedi. Birkaç dakika sonra yeniden deneyin."
	case strings.Contains(msg, "json"):
		return "AI yanıtı okunamadı (uzun metin kesilmiş olabilir). Yeniden deneyin."
	case strings.Contains(msg, "paragraf sayısı"):
		return "AI paragraf sayısını koruyamadı; çeviri yarım kalmasın diye kaydedilmedi. Yeniden deneyin."
	}
	return "Çeviri üretilemedi. Ayrıntı sunucu günlüğüne yazıldı."
}

// kayitHataMetni veritabanı hatasını editöre gösterilecek metne çevirir.
//
// Ham SQL metni gösterilmez: FK ihlali doğrudan sızıyordu
// ("insert or update on table contents violates foreign key constraint").
func kayitHataMetni(err error) string {
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "foreign key"):
		return "Çeviri kaydedilemedi: kaynak içeriğin kategori/görsel bağlantısı geçersiz. " +
			"Kaynak içerikteki kategori ve görseli kontrol edip yeniden deneyin."
	case strings.Contains(msg, "duplicate key"), strings.Contains(msg, "unique"):
		return "Bu dil için çeviri zaten var. Sayfayı yenileyip mevcut çeviriyi düzenleyin."
	}
	return "Çeviri kaydedilemedi. Ayrıntı sunucu günlüğüne yazıldı."
}
