package httpapi

import (
	"strings"
	"testing"

	"theblock/site/internal/repo"
)

// Model yalnız ortam değişkeninden okunmalı. DB'den okumak yanlış cevabı
// güvenle göstermek olurdu: ai.NewClient de yalnız env okuyor, paneldeki
// `yapay_zeka` ayarları o yolu etkilemiyor.
func TestUretimModelEnvdenOkur(t *testing.T) {
	t.Setenv("OMNIROUTE_MODEL", "evren/qwen3-vl-30b")
	t.Setenv("OMNIROUTE_REASONING_EFFORT", "")
	g := uretimModel()
	if !strings.Contains(g, "evren/qwen3-vl-30b") {
		t.Fatalf("model yazılmamış: %s", g)
	}
	if !strings.Contains(g, "sabit") {
		t.Errorf("sabit model notu yok: %s", g)
	}
	if strings.Contains(g, "otomatik birleşim") {
		t.Errorf("sabit model 'otomatik birleşim' diye anlatılıyor: %s", g)
	}
}

// auto/* uyarısı ŞART: ölçülen kaybın tek sebebi buydu ve panel bunu
// söylemezse okuyan sayıları doğru, sebebi yanlış okur.
func TestUretimModelAutoBirlesimiUyarir(t *testing.T) {
	t.Setenv("OMNIROUTE_MODEL", "auto/best-chat")
	g := uretimModel()
	if !strings.Contains(g, "otomatik birleşim") {
		t.Fatalf("auto/* uyarısı yok: %s", g)
	}
	if !strings.Contains(g, "minTier=free") {
		t.Errorf("uyarı ölçülen sebebi söylemiyor: %s", g)
	}
}

// Boş env'de "model yok" ile "varsayılana düştü" ayrılmalı: ai.NewClient
// sessizce gpt-4o-mini'ye düşüyor ve bu paneldeki en yanıltıcı boşluk
// olurdu.
func TestUretimModelBosEnvVarsayilaniSoyler(t *testing.T) {
	t.Setenv("OMNIROUTE_MODEL", "")
	t.Setenv("OMNIROUTE_DEFAULT_MODEL", "")
	g := uretimModel()
	if !strings.Contains(g, "varsayılan") {
		t.Fatalf("varsayılana düşüldüğü yazılmamış: %s", g)
	}
	if !strings.Contains(g, "gpt-4o-mini") {
		t.Errorf("hangi varsayılan olduğu yazılmamış: %s", g)
	}
}

func TestUretimModelDusunmeCabasiniGosterir(t *testing.T) {
	t.Setenv("OMNIROUTE_MODEL", "evren/deepseek-v4.1-flash")
	t.Setenv("OMNIROUTE_REASONING_EFFORT", "none")
	g := uretimModel()
	if !strings.Contains(g, "Düşünme çabası") || !strings.Contains(g, "none") {
		t.Fatalf("düşünme çabası gösterilmiyor: %s", g)
	}
}

// Sabit model notu zaman aşımı imzasına atıf yapıyor; imza adı
// repo'da değişirse metin sessizce anlamsızlaşmasın.
func TestUretimModelImzaAdiReposuylaAyni(t *testing.T) {
	t.Setenv("OMNIROUTE_MODEL", "evren/qwen3-vl-30b")
	if !strings.Contains(uretimModel(), repo.ImzaZamanAsimi) {
		t.Fatal("sabit model notu repo.ImzaZamanAsimi'yi içermiyor")
	}
}
