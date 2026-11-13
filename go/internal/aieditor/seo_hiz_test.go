package aieditor

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSeoBeklemeTempo(t *testing.T) {
	// dakika=20 → öğe başına 3 sn. 10 öğe = 30 sn hedef.
	// Tur 12 sn sürdüyse 18 sn beklenir.
	assert.Equal(t, 18*time.Second, seoBekleme(20, 10, 12*time.Second))

	// Tur hedeften uzun sürdüyse bekleme yok.
	assert.Equal(t, time.Duration(0), seoBekleme(20, 10, 40*time.Second))
	assert.Equal(t, time.Duration(0), seoBekleme(20, 10, 30*time.Second))

	// İş yoksa bekleme hesabı yapılmaz (çağıran dakikada bir bekler).
	assert.Equal(t, time.Duration(0), seoBekleme(20, 0, 0))

	// Yavaş ayar daha çok bekletir; hızlı ayar daha az.
	yavas := seoBekleme(6, 5, time.Second)
	hizli := seoBekleme(60, 5, time.Second)
	assert.Greater(t, yavas, hizli)
}

func TestSeoDakikaSinirlari(t *testing.T) {
	// 0/negatif → varsayılan.
	assert.Equal(t, SeoVarsayilanDakika, sinirDakika(0))
	assert.Equal(t, SeoVarsayilanDakika, sinirDakika(-5))
	// Üst sınır.
	assert.Equal(t, SeoMaksDakika, sinirDakika(10000))
	// Geçerli değer korunur.
	assert.Equal(t, 15, sinirDakika(15))
}

func TestSeoTempoAkisiGecer(t *testing.T) {
	const saatlikAkis = 150 // ölçüm: 24 saatte 3.593 taslak
	saatlikKapasite := SeoVarsayilanDakika * 60
	assert.Greater(t, saatlikKapasite, saatlikAkis,
		"varsayılan hız saatlik akışı geçmeli, yoksa kuyruk hiç erimez")
}

// TestSeoHizAyarAnahtari: panel yeni anahtarı kullanmalı (eski "tur
// başına" anlaşılmıyordu).
func TestSeoHizAyarAnahtari(t *testing.T) {
	src := mustReadAieditor(t, "seo_worker.go")
	assert.Contains(t, src, `"bot", "seo_dakika"`)
	// Eski anahtar yalnız geriye dönük okuma için kalmalı.
	assert.Contains(t, src, `"bot", "seo_dilim"`)

	// Çalışma döngüsü sürekli olmalı; sabit aralıklı ticker kuyruğu
	// eritmiyordu.
	assert.NotContains(t, src, "time.NewTicker",
		"sabit ticker yerine tempo tabanlı sürekli döngü olmalı")
	// ÇAĞRI yeri önemli: yalnız fonksiyonun tanımlı olması yetmez
	// (mutasyonda çağrı devre dışı bırakıldığında test yeşil kalıyordu).
	assert.Contains(t, src, "seoBekleme(ayar.Dakika, n, time.Since(bas))",
		"tempo döngüde uygulanmalı")
}

func TestSeoEszamanliIsleme(t *testing.T) {
	src := mustReadAieditor(t, "seo_worker.go")

	assert.Contains(t, src, "const SeoEszamanli = 3")
	assert.Contains(t, src, "go func(c repo.Content)",
		"içerikler ayrı goroutine'de üretilmeli")
	// Semafor: eşzamanlılık sabit ve sınırlı olmalı (sağlayıcıyı boğmamak).
	assert.Contains(t, src, "make(chan struct{}, SeoEszamanli)")
	// Sonuç toplama BAŞLATILAN sayısına göre olmalı; atlananlar sayılırsa
	// döngü sonsuza kadar bekler (kilitlenme).
	assert.Contains(t, src, "baslatilan++")
	assert.Contains(t, src, "i < baslatilan")
}

func TestSeoMaxTokenYeterli(t *testing.T) {
	src := mustReadAieditor(t, "seo.go")
	assert.NotContains(t, src, "MaxTokens:   400")
	assert.Contains(t, src, "MaxTokens:   700")
}

// TestAIZamanAsimiYeterli: sınır İKİ KEZ yükseltildi, iki kez de ölçümle.
// 60 sn → 120 sn (sağlayıcı 80 sn'ye kadar yanıt veriyordu), 120 sn → 180 sn
// (14 Kasım 2026 ölçümü: BAŞARILI işlerin p90'ı 194.7 sn, en uzunu 239.1 sn;
// 120 sn sağlıklı yanıtları kesiyordu).
func TestAIZamanAsimiYeterli(t *testing.T) {
	src := mustReadAieditor(t, "../ai/ai.go")
	assert.Contains(t, src, "aiHTTPTimeout()")
	assert.Contains(t, src, "AI_HTTP_TIMEOUT_SEC")
	assert.Contains(t, src, "return 180 * time.Second")
	assert.NotContains(t, src, "return 120 * time.Second",
		"120 sn p90'ın (194.7 sn) altında")
	assert.NotContains(t, src, "Timeout: 60 * time.Second",
		"60 sn sağlıklı yanıtları kesiyordu")
}

func TestSeoUretTekrar(t *testing.T) {
	src := mustReadAieditor(t, "seo_worker.go")

	assert.Contains(t, src, "func seoUretTekrar(")
	assert.Contains(t, src, "oneri, err := seoUretTekrar(ctx, aiclient, c, logger)",
		"tur, tekrar denemeli üreticiyi kullanmalı")

	// En çok iki deneme: her deneme AI maliyeti.
	i := strings.Index(src, "func seoUretTekrar(")
	require.Greater(t, i, 0)
	son := strings.Index(src[i:], "\n}")
	require.Greater(t, son, 0)
	govde := src[i : i+son]
	assert.Equal(t, 2, strings.Count(govde, "SeoUret(ctx, aiclient, c)"),
		"tam iki deneme olmalı (sonsuz değil)")

	// ctx iptalinde ikinci deneme yapılmamalı.
	assert.Contains(t, govde, "if ctx.Err() != nil {")
}
