package aieditor

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"theblock/site/internal/ai"
	"theblock/site/internal/repo"
)

// SeoAyarlar otomatik SEO doldurmanın açık/kapalı ve hız bilgisidir.
type SeoAyarlar struct {
	Aktif  bool
	Dakika int
}

const SeoVarsayilanDakika = 20

const SeoEszamanli = 3

// SeoMaksDakika panelden girilebilecek üst sınır.
const SeoMaksDakika = 120

func SeoAyarlariOku(ctx context.Context, pool *pgxpool.Pool) SeoAyarlar {
	a := SeoAyarlar{Aktif: true, Dakika: SeoVarsayilanDakika}
	if b, err := repo.GetSetting(ctx, pool, "bot", "seo_otomatik"); err == nil {
		a.Aktif = strings.Trim(string(b), `" `) == "true"
	}
	// Yeni anahtar: dakika. Eski "seo_dilim" (tur başına) değeri varsa
	// dakikaya çevrilir (2 dakikalık tur varsayımıyla ×30).
	if b, err := repo.GetSetting(ctx, pool, "bot", "seo_dakika"); err == nil {
		if n, e := strconv.Atoi(strings.Trim(string(b), `" `)); e == nil && n > 0 {
			a.Dakika = sinirDakika(n)
		}
	} else if b, err := repo.GetSetting(ctx, pool, "bot", "seo_dilim"); err == nil {
		if n, e := strconv.Atoi(strings.Trim(string(b), `" `)); e == nil && n > 0 {
			a.Dakika = sinirDakika(n * 30)
		}
	}
	return a
}

// SeoDoldurAyarlari yazar (panel formundan).
func SeoDoldurAyarlari(ctx context.Context, pool *pgxpool.Pool, aktif bool, dakika int) error {
	v := "false"
	if aktif {
		v = "true"
	}
	if err := repo.SetSetting(ctx, pool, "bot", "seo_otomatik", []byte(v)); err != nil {
		return err
	}
	return repo.SetSetting(ctx, pool, "bot", "seo_dakika",
		[]byte(strconv.Itoa(sinirDakika(dakika))))
}

// sinirDakika hızı geçerli aralığa sıkıştırır.
func sinirDakika(n int) int {
	if n <= 0 {
		return SeoVarsayilanDakika
	}
	if n > SeoMaksDakika {
		return SeoMaksDakika
	}
	return n
}

type seoDeneme struct {
	mu   sync.Mutex
	sayi map[int64]int
}

func (d *seoDeneme) asildi(id int64) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.sayi[id] >= 3
}

func (d *seoDeneme) basarisiz(id int64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.sayi[id]++
}

func (d *seoDeneme) basarili(id int64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.sayi, id)
}

func SeoDoldurmaTuru(ctx context.Context, pool *pgxpool.Pool, aiclient *ai.Client, logger *slog.Logger) int {
	return seoDoldurmaTuru(ctx, pool, aiclient, logger, &seoDeneme{sayi: map[int64]int{}}, 10)
}

// seoTurLimiti tek turda işlenecek en fazla içerik. Hız ayarı (dakika)
// döngüde uygulanır; bu, tek turun süresini sınırlar.
const seoTurLimiti = 10

func seoDoldurmaTuru(ctx context.Context, pool *pgxpool.Pool, aiclient *ai.Client, logger *slog.Logger, d *seoDeneme, limit int) (doldurulan int) {
	ayar := SeoAyarlariOku(ctx, pool)
	if !ayar.Aktif {
		return 0
	}
	if aiclient == nil || !aiclient.Enabled() {
		return 0
	}
	if limit <= 0 {
		limit = seoTurLimiti
	}
	// Vazgeçilenleri de doldurabilmek için biraz fazla isteyelim.
	liste, err := SeoBosTaslaklar(ctx, pool, limit*2)
	if err != nil {
		logger.Warn("seo: boş taslaklar okunamadı", "error", err)
		return 0
	}
	// Aday listesini sınırla: eşzamanlı işlemede fazlasını başlatmayalım.
	adaylar := liste
	if len(adaylar) > limit {
		adaylar = adaylar[:limit]
	}

	// Eşzamanlı işleme: sağlayıcı 20-83 sn yanıt verdiği için sıralı
	// işleme kuyruğu eritmiyordu (ölçüldü: 1-2/dk).
	tur := make(chan struct{}, SeoEszamanli)
	sonuc := make(chan bool, len(adaylar))
	baslatilan := 0
	for _, c := range adaylar {
		if ctx.Err() != nil {
			break
		}
		if d.asildi(c.ID) {
			continue
		}
		tur <- struct{}{}
		baslatilan++
		go func(c repo.Content) {
			defer func() { <-tur }()
			oneri, err := seoUretTekrar(ctx, aiclient, c, logger)
			if err != nil {
				logger.Warn("seo: üretilemedi", "içerik", c.ID, "error", err)
				d.basarisiz(c.ID)
				sonuc <- false
				return
			}
			// force=false: editörün elle yazdığı alan korunur.
			n, err := SeoYaz(ctx, pool, c.ID, oneri, false)
			if err != nil {
				logger.Warn("seo: yazılamadı", "içerik", c.ID, "error", err)
				d.basarisiz(c.ID)
				sonuc <- false
				return
			}
			d.basarili(c.ID)
			if n > 0 {
				logger.Info("seo: otomatik dolduruldu", "içerik", c.ID,
					"alan", n, "sağlayıcı", oneri.Provider, "model", oneri.Model)
				sonuc <- true
				return
			}
			sonuc <- false
		}(c)
	}
	// Yalnız BAŞLATILAN goroutine sayısı kadar sonuç beklenir; atlananlar
	// (asildi) sayıya katılırsa döngü sonsuza kadar beklerdi.
	for i := 0; i < baslatilan; i++ {
		select {
		case ok := <-sonuc:
			if ok {
				doldurulan++
			}
		case <-ctx.Done():
			return doldurulan
		}
	}
	return doldurulan
}

func seoUretTekrar(ctx context.Context, aiclient *ai.Client, c repo.Content, logger *slog.Logger) (SeoOneri, error) {
	oneri, err := SeoUret(ctx, aiclient, c)
	if err == nil {
		return oneri, nil
	}
	if ctx.Err() != nil {
		return SeoOneri{}, err
	}
	logger.Info("seo: yeniden deneniyor", "içerik", c.ID, "ilk_hata", err.Error())
	oneri2, err2 := SeoUret(ctx, aiclient, c)
	if err2 != nil {
		return SeoOneri{}, err2
	}
	return oneri2, nil
}

func RunSeoDoldurma(ctx context.Context, pool *pgxpool.Pool, aiclient *ai.Client, logger *slog.Logger, _ time.Duration) {
	deneme := &seoDeneme{sayi: map[int64]int{}}
	logger.Info("seo doldurma başladı", "varsayılan_dakika", SeoVarsayilanDakika)
	for {
		if ctx.Err() != nil {
			logger.Info("seo doldurma durdu")
			return
		}
		ayar := SeoAyarlariOku(ctx, pool)
		if !ayar.Aktif || aiclient == nil || !aiclient.Enabled() {
			if !bekle(ctx, time.Minute) {
				return
			}
			continue
		}
		bas := time.Now()
		n := seoDoldurmaTuru(ctx, pool, aiclient, logger, deneme, seoTurLimiti)
		if n > 0 {
			logger.Info("seo doldurma turu", "doldurulan", n, "dakika_hedefi", ayar.Dakika)
		}
		if kalan := seoBekleme(ayar.Dakika, n, time.Since(bas)); kalan > 0 {
			if !bekle(ctx, kalan) {
				return
			}
		} else if n == 0 {
			// Boş kuyruk: dakikada bir kontrol et, gereksiz sorgu/AI çağrısı
			// yapma.
			if !bekle(ctx, time.Minute) {
				return
			}
		}
	}
}

func seoBekleme(dakika, islenen int, gecen time.Duration) time.Duration {
	if islenen <= 0 {
		return 0
	}
	hedef := time.Duration(float64(time.Minute) * float64(islenen) /
		float64(sinirDakika(dakika)))
	if hedef <= gecen {
		return 0
	}
	return hedef - gecen
}

// bekle ctx iptalinde false döner.
func bekle(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}
