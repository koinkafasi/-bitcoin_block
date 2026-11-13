package aieditor

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"theblock/site/internal/ai"
	"theblock/site/internal/auth"
	"theblock/site/internal/repo"
)

// YazimAyarlar otomatik AI ile yazımın açık/kapalı ve hız bilgisidir.
type YazimAyarlar struct {
	Aktif  bool
	Dakika int
}

// YazimVarsayilanDakika varsayılan hızdır (dakikada iş).
const YazimVarsayilanDakika = 3

// YazimMaksDakika panelden girilebilecek üst sınır.
const YazimMaksDakika = 60

// YazimTurLimiti tek turda ele alınacak en fazla taslak.
const YazimTurLimiti = 6

func YazimAyarlariOku(ctx context.Context, pool *pgxpool.Pool) YazimAyarlar {
	a := YazimAyarlar{Aktif: false, Dakika: YazimVarsayilanDakika}
	if b, err := repo.GetSetting(ctx, pool, "bot", "yazim_otomatik"); err == nil {
		a.Aktif = strings.Trim(string(b), `" `) == "true"
	}
	if b, err := repo.GetSetting(ctx, pool, "bot", "yazim_dakika"); err == nil {
		if n, e := strconv.Atoi(strings.Trim(string(b), `" `)); e == nil && n > 0 {
			a.Dakika = sinirYazimDakika(n)
		}
	}
	return a
}

// YazimAyarlariYaz panel formundan yazar.
func YazimAyarlariYaz(ctx context.Context, pool *pgxpool.Pool, aktif bool, dakika int) error {
	v := "false"
	if aktif {
		v = "true"
	}
	if err := repo.SetSetting(ctx, pool, "bot", "yazim_otomatik", []byte(v)); err != nil {
		return err
	}
	return repo.SetSetting(ctx, pool, "bot", "yazim_dakika",
		[]byte(strconv.Itoa(sinirYazimDakika(dakika))))
}

func sinirYazimDakika(n int) int {
	if n <= 0 {
		return YazimVarsayilanDakika
	}
	if n > YazimMaksDakika {
		return YazimMaksDakika
	}
	return n
}

type yazimDeneme struct {
	mu   sync.Mutex
	sayi map[int64]int
}

func (d *yazimDeneme) asildi(id int64) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.sayi[id] >= 3
}

func (d *yazimDeneme) basarisiz(id int64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.sayi[id]++
}

func (d *yazimDeneme) basarili(id int64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.sayi, id)
}

func RunYazimDoldurma(ctx context.Context, pool *pgxpool.Pool, aiclient *ai.Client, logger *slog.Logger, every time.Duration) {
	if every <= 0 {
		every = time.Minute
	}
	logger.Info("ai yazım hattı başladı", "her", every.String())
	deneme := &yazimDeneme{sayi: map[int64]int{}}
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			logger.Info("ai yazım hattı durdu")
			return
		case <-ticker.C:
			yazimTuru(ctx, pool, aiclient, logger, deneme)
		}
	}
}

func yazimTuru(ctx context.Context, pool *pgxpool.Pool, aiclient *ai.Client, logger *slog.Logger, d *yazimDeneme) {
	ayar := YazimAyarlariOku(ctx, pool)
	if !ayar.Aktif {
		return
	}
	if aiclient == nil || !aiclient.Enabled() {
		return
	}

	// 1) Hazır sonuçları uygula. Kuyruğa yazmadan ÖNCE: biriken sonuçlar
	//    içeriğe dönsün, editör görsün.
	uygulanan, err := yazimSonuclariUygula(ctx, pool, logger, 20)
	if err != nil {
		logger.Warn("yazım: sonuçlar uygulanamadı", "error", err)
	}
	if uygulanan > 0 {
		logger.Info("yazım: AI sonuçları içeriğe yazıldı", "adet", uygulanan)
	}

	// 2) Hız sınırı: son 1 dakikada eklenen iş sayısı.
	//
	// GECE PENCERESİ: sağlayıcı kotası 03:00'te sıfırlanıyor ve ölçümde
	// (13 Kasım 2026) 03:00-09:00 arası hiç 429 görülmedi — o pencerede
	// dakika hedefi GeceCarpani katına çıkar. Aynı ölçümde o saatlerde
	// hat saatte 2-8 iş üretiyordu, akşam 81; bedava kapasite boş
	// geçiyordu.
	dakikaHedefi := ayar.Dakika
	if ai.KotaBolPencere(time.Now()) {
		dakikaHedefi *= YazimGeceCarpani
	}
	kota := dakikaHedefi - repo.YazimSayaciSonDakika(ctx, pool, 1)
	if kota <= 0 {
		return
	}
	if kota > YazimTurLimiti*catBoyutCarpani(dakikaHedefi, ayar.Dakika) {
		kota = YazimTurLimiti * catBoyutCarpani(dakikaHedefi, ayar.Dakika)
	}

	liste, err := repo.YazimBekleyenTaslaklar(ctx, pool, 1, kota)
	if err != nil {
		logger.Warn("yazım: bekleyen taslaklar okunamadı", "error", err)
		return
	}
	if len(liste) == 0 {
		return
	}
	svc := NewService(pool, aiclient, logger)
	eklenen := 0
	for _, c := range liste {
		if ctx.Err() != nil {
			return
		}
		if d.asildi(c.ID) {
			continue
		}
		act, ok := ActionByID("news")
		if !ok {
			return
		}
		req := Request{
			ContentID: c.ID,
			User:      auth.User{ID: c.AuthorID},
			Action:    "news",
			Instruction: "Kaynak özetlerinden özgün, doğrulanabilir bir Türkçe haber " +
				"taslağı oluştur. Kaynakta olmayan olgu ekleme.",
		}
		if _, err := svc.createJob(ctx, act, c, req, "queued"); err != nil {
			logger.Warn("yazım: iş kuyruğa yazılamadı", "içerik", c.ID, "error", err)
			d.basarisiz(c.ID)
			continue
		}
		d.basarili(c.ID)
		eklenen++
	}
	if eklenen > 0 {
		logger.Info("yazım: taslaklar AI kuyruğuna alındı", "adet", eklenen)
	}
}

// yazimSonuclariUygula tamamlanmış, uygulanmamış AI sonuçlarını içeriğe yazar.
func yazimSonuclariUygula(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger, limit int) (int, error) {
	isler, err := repo.YazimUygulanacakIsler(ctx, pool, 1, limit)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, is := range isler {
		if ctx.Err() != nil {
			//nolint:nilerr // iş listesi okunamazsa bu tur boş geçer; kuyruk bir sonraki turda yeniden denenir
			return n, nil
		}
		res, err := DecodeResult(is.Result)
		if err != nil {
			logger.Warn("yazım: sonuç okunamadı", "iş", is.JobID, "error", err)
			// Okunamayan sonucu işaretle: her turda yeniden denenmesin.
			_ = repo.MarkAIJobApplied(ctx, pool, is.JobID)
			continue
		}
		y, ok := yazimSonucu(res)
		if !ok {
			logger.Warn("yazım: sonuç eksik (başlık/gövde yok)", "iş", is.JobID)
			_ = repo.MarkAIJobApplied(ctx, pool, is.JobID)
			continue
		}
		// Sürüm denetimi: iş kuyruğa girdikten sonra editör içeriği
		// kaydettiyse AI sonucu onun değişikliğini EZMEZ.
		job, jerr := repo.GetAIJob(ctx, pool, is.JobID)
		surum := int64(0)
		if jerr == nil {
			surum = int64(job.ArticleVersion)
		}
		if err := repo.SetContentYazim(ctx, pool, is.ContentID, surum, y); err != nil {
			if errors.Is(err, repo.ErrConflict) {
				logger.Info("yazım: içerik bu arada değişti, sonuç yazılmadı",
					"iş", is.JobID, "içerik", is.ContentID)
				_ = repo.MarkAIJobApplied(ctx, pool, is.JobID)
				continue
			}
			logger.Warn("yazım: içeriğe yazılamadı", "iş", is.JobID, "içerik", is.ContentID, "error", err)
			continue
		}
		if err := repo.MarkAIJobApplied(ctx, pool, is.JobID); err != nil {
			logger.Warn("yazım: iş işaretlenemedi", "iş", is.JobID, "error", err)
		}
		n++
		logger.Info("yazım: taslak AI metniyle güncellendi",
			"içerik", is.ContentID, "iş", is.JobID,
			"paragraf", len(y.Paragraf), "uyarı", len(res.Warnings))
	}
	return n, nil
}

func yazimSonucu(res Result) (repo.YazimSonucu, bool) {
	title := strings.TrimSpace(res.Fields["title"])
	if title == "" {
		title = strings.TrimSpace(res.Title)
	}
	spot := strings.TrimSpace(res.Fields["spot"])
	body := strings.TrimSpace(res.Fields["body"])
	if body == "" {
		for _, sec := range res.Sections {
			if sec.Title == "Gövde" {
				body = strings.TrimSpace(sec.Body)
				break
			}
		}
	}
	if title == "" || body == "" {
		return repo.YazimSonucu{}, false
	}
	paragraflar := bodyParagraflari(body)
	if len(paragraflar) == 0 {
		return repo.YazimSonucu{}, false
	}
	// DÖNGÜ KAPISI — buradan geçmeyen metin içeriğe YAZILMAZ.
	//
	// Üretimde görüldü (içerik 10764, 13 Kasım 2026): model 4 paragraf
	// üretip bunları 12 KEZ tekrarlamış; 48 blok, 3.000 kelime, hepsi
	// aynı dört cümle. Aynı arıza 12 Kasım'da da olmuş (içerik 10169),
	// yani uzunluk hedefinden ÖNCE de vardı — hedef yalnız baskıyı
	// artırıyor.
	//
	// yapi.go zaten tekrarı yakalıyordu ama UYARI olarak: uyarı yazmayı
	// engellemiyor ve bozuk metin taslağa giriyordu. Burası deterministik
	// bir kapı.
	temiz, yinelenen := paragraflariTekille(paragraflar)
	if len(temiz) == 0 {
		return repo.YazimSonucu{}, false
	}
	// Yarısından fazlası tekrarsa model döngüye girmiştir; kalanı
	// kurtarmak yerine sonucu REDDEDİYORUZ — iş kuyrukta yeniden
	// denenir. Yarım kurtarılmış metin, editöre "bu iyi" sinyali verir.
	if yinelenen*2 >= len(paragraflar) {
		return repo.YazimSonucu{}, false
	}
	paragraflar = temiz
	y := repo.YazimSonucu{
		Title:    title,
		Spot:     spot,
		SeoTitle: strings.TrimSpace(res.Fields["seo_title"]),
		SeoDesc:  strings.TrimSpace(res.Fields["meta_description"]),
		// SEO alanları da yazma sınırından geçer (70/165 rune).
		Paragraf: paragraflar,
		Ozet:     strings.TrimSpace(res.Fields["ai_summary"]),
	}
	return y, true
}

// bodyParagraflari gövdeyi paragraf listesine böler (boş satır veya satır
// sonu sınırları). Boş paragraflar atılır.
func bodyParagraflari(body string) []string {
	body = strings.ReplaceAll(body, "\r\n", "\n")
	var out []string
	for _, p := range strings.Split(body, "\n") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// paragraflariTekille birebir tekrar eden paragrafları atar ve kaç
// tanesinin atıldığını döndürür.
//
// Karşılaştırma normalize edilmiş metin üzerinden: model aynı cümleyi
// farklı boşluk veya büyük/küçük harfle yineleyebiliyor ve ham
// karşılaştırma bunu kaçırır. Noktalama KORUNUYOR — iki farklı cümle
// yalnız noktalamayla ayrışabilir.
func paragraflariTekille(in []string) (out []string, yinelenen int) {
	gorulen := make(map[string]bool, len(in))
	out = make([]string, 0, len(in))
	for _, p := range in {
		anahtar := strings.ToLower(strings.Join(strings.Fields(p), " "))
		// Çok kısa satırlar (ara başlık, tek kelime) meşru olarak
		// tekrarlanabilir; yalnız gövde uzunluğundakileri eliyoruz.
		if len([]rune(anahtar)) < 40 {
			out = append(out, p)
			continue
		}
		if gorulen[anahtar] {
			yinelenen++
			continue
		}
		gorulen[anahtar] = true
		out = append(out, p)
	}
	return out, yinelenen
}

// YazimGeceCarpani kota penceresinde dakika hedefinin çarpanıdır.
//
// ceviri.GeceCarpani ile aynı gerekçe ve aynı değer: ölçülen boş pencereyi
// kapatmaya yetiyor, sağlayıcıyı yeniden 429'a sokmayacak kadar ölçülü.
// Kota yine biterse ai.Client kısa devresi devreye girer ve iş kuyrukta
// bekler (kaybolmaz).
const YazimGeceCarpani = 4

// catBoyutCarpani tur limitinin gece penceresinde kaça katlanacağını verir.
//
// Dakika hedefi 4 katına çıkınca tur limiti de büyümeli, yoksa döngü
// hedefe ulaşmadan limite takılır ve pencere yine boş geçer. Ama parti
// boyutunu sınırsız büyütmek tek turda çok iş açıp belleği ve sağlayıcıyı
// zorlardı; bu yüzden orana bağlı ve en çok YazimGeceCarpani kadar.
func catBoyutCarpani(hedef, temel int) int {
	if temel <= 0 || hedef <= temel {
		return 1
	}
	kat := hedef / temel
	if kat > YazimGeceCarpani {
		kat = YazimGeceCarpani
	}
	if kat < 1 {
		kat = 1
	}
	return kat
}
