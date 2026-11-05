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
	Aktif bool
	// Dakika: DAKİKADA işlenecek içerik sayısı (AI maliyeti/latency sınırı).
	//
	// Ölçüm (canlı): bot günde ~3.593 taslak ekliyor (~150/saat). Eski ayar
	// "2 dakikada 5" = 150/saat idi — akış hızına TAM EŞİT, yani 10.400
	// kişilik birikmiş kuyruk hiç erimiyordu. Hız artık dakika cinsinden
	// veriliyor ki panelden anlaşılır olsun.
	Dakika int
}

// SeoVarsayilanDakika varsayılan hızdır (dakikada içerik).
//
// Ölçüm: aynı sağlayıcıda tek çağrı 20–83 sn (auto/best-chat farklı
// modellere yönlendiriyor). Sıralı işlemede pratik hız 1–2/dk'ya düşüyor,
// 10 binlik kuyruk hiç erimiyor.
const SeoVarsayilanDakika = 20

// SeoEszamanli aynı anda uçuşta olacak AI isteği sayısı.
//
// 3 paralel istek sağlayıcıda ölçüldü (3 istek 31 sn'de döndü, hatasız).
const SeoEszamanli = 3

// SeoMaksDakika panelden girilebilecek üst sınır.
const SeoMaksDakika = 120

// SeoAyarlariOku bot ayarlarından okur.
//
// Varsayılan AÇIK: otomatik haberlerin SEO alanları doldurulsun istendi.
// Ayar hiç yazılmamışsa açık kabul edilir; panelden kapatılabilir.
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

// seoDeneme bir içeriğin kaç kez başarısız olduğunu tutar.
//
// Başarısız içerik her turda yeniden denenirse her 2 dakikada bir AI
// çağrısı (ve ücret) harcanır. Üç denemeden sonra vazgeçilir; süreç
// yeniden başlarsa sayaç sıfırlanır (içerik düzelmiş olabilir).
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

// SeoDoldurmaTuru tek bir turda (en çok 10 içerik) boş SEO'lu taslakları
// doldurur ve doldurulan sayısını verir.
//
// Bir içerik hata verirse tur devam eder: tek bir bozuk kayıt tüm kuyruğu
// durdurmamalı.
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

// seoUretTekrar SEO üretimini en çok iki kez dener.
//
// Neden: sağlayıcı (auto/best-chat) isteklerin bir kısmında görünmeyen akıl
// yürütmeye token harcayıp içeriği hiç yazmıyor (ölçüm: 1000 token'da
// finish=length ve 14 karakter). Bu yanıt ne JSON ne kurtarılabilir; tek
// denemede bırakmak içeriği kalıcı boş bırakıyordu. İkinci deneme
// gözlemlenen başarı oranını belirgin biçimde yükseltiyor.
//
// Yalnız İKİ deneme: her deneme AI maliyeti; sürekli başarısız kayıt
// kuyruğu tıkamamalı (3 başarısızlıktan sonra tur dışı bırakılır).
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

// RunSeoDoldurma otomatik haberlerin SEO alanlarını SÜREKLİ, ayarlanan
// dakikalık hızda doldurur.
//
// Eski yapı "her 2 dakikada bir tur" idi ve tur başına 5 içerik işliyordu:
// saatte 150. Bot ise saatte ~150 taslak üretiyor (ölçüm: 24 saatte 3.593).
// Yani akış hızına tam eşit → 10.400 kişilik birikmiş kuyruk hiç erimiyordu.
//
// Yeni yapı: tur biter bitmez bir sonraki tura geçilir, iki tur arasında
// yalnız hedef aralığı (60 sn / dakika) tamamlayacak kadar beklenir. Böylece
// hız panelden anlaşılır bir birimle (dakikada N) verilir ve kuyruk erir.
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
		// Tempo: dakikada `Dakika` içerik ⇒ öğe başına 60 sn / Dakika.
		//
		// İşlenen öğe sayısına göre hedef süre hesaplanır; tur zaten bu
		// süreden uzun sürdüyse beklenmez (AI çağrısı hızı zaten sınırlar).
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

// seoBekleme, hedef tempoyu tutturmak için beklenecek süreyi verir.
//
//	dakika: dakikada hedeflenen içerik sayısı
//	islenen: bu turda işlenen içerik sayısı
//	gecen: turun gerçek süresi
//
// Örnek: dakika=20, islenen=10 ⇒ hedef 30 sn. Tur 12 sn sürdüyse 18 sn beklenir.
// Tur 40 sn sürdüyse bekleme yok (0): yavaş sağlayıcı zaten tempoyu tutuyor.
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
