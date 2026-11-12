package scheduler

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"theblock/site/internal/repo"
)

// --- blocksPlainText ---

func TestBlocksPlainTextBlokMetinleriniBirlestirir(t *testing.T) {
	detail := []byte(`[{"text":"Birinci"},{"text":"  İkinci  "},{"text":""},{"text":"Üçüncü"}]`)
	if got := blocksPlainText(detail); got != "Birinci İkinci Üçüncü" {
		t.Fatalf("got = %q", got)
	}
}

// Bütün bloklar boşsa sonuç boş olmalı (boşluk yığını değil).
func TestBlocksPlainTextBosBloklar(t *testing.T) {
	if got := blocksPlainText([]byte(`[{"text":"   "},{"text":""}]`)); got != "" {
		t.Fatalf("got = %q; boş bekleniyordu", got)
	}
	if got := blocksPlainText(nil); got != "" {
		t.Fatalf("nil için got = %q", got)
	}
}

// Bozuk JSON'da ham gövde kırpılıp döner: indeksleme tamamen durmamalı.
func TestBlocksPlainTextBozukJSONHamGovde(t *testing.T) {
	if got := blocksPlainText([]byte("  düz metin gövde  ")); got != "düz metin gövde" {
		t.Fatalf("got = %q", got)
	}
}

// `text` alanı olmayan bloklar (görsel/video) atlanır, paniğe yol açmaz.
func TestBlocksPlainTextAlansizBloklar(t *testing.T) {
	if got := blocksPlainText([]byte(`[{"tip":"gorsel"},{"text":"Var"}]`)); got != "Var" {
		t.Fatalf("got = %q", got)
	}
}

// --- slugURL ---

func TestSlugURL(t *testing.T) {
	kasalar := []struct {
		slug string
		id   int64
		want string
	}{
		{"deprem", 42, "/deprem-42"},
		{"", 7, "/icerik-7"},
		{"   ", 7, "/icerik-7"},
		{"  bosluklu  ", 9, "/bosluklu-9"},
	}
	for _, k := range kasalar {
		if got := slugURL(k.slug, k.id); got != k.want {
			t.Errorf("slugURL(%q,%d) = %q; %q bekleniyordu", k.slug, k.id, got, k.want)
		}
	}
}

// --- catSlugOrEmpty ---

func TestCatSlugOrEmpty(t *testing.T) {
	if got := catSlugOrEmpty(nil); got != "" {
		t.Errorf("nil için %q", got)
	}
	s := "ekonomi"
	if got := catSlugOrEmpty(&s); got != "ekonomi" {
		t.Errorf("got = %q", got)
	}
	bos := ""
	if got := catSlugOrEmpty(&bos); got != "" {
		t.Errorf("boş işaretçi için %q", got)
	}
}

// --- truncateText ---

// Kesim RUNE sınırında olmalı: bayt kesimi Türkçe karakteri bozar.
func TestTruncateTextRuneSinirinda(t *testing.T) {
	kasalar := []struct {
		in   string
		n    int
		want string
	}{
		{"kısa", 10, "kısa"},
		{"tam", 3, "tam"},
		{"Çok daha uzun bir başlık", 0, ""},
		{"Çok daha uzun bir başlık", 3, "Çok"},
		{"şşşşş", 2, "şş"},
		{"", 5, ""},
	}
	for _, k := range kasalar {
		if got := truncateText(k.in, k.n); got != k.want {
			t.Errorf("truncateText(%q,%d) = %q; %q bekleniyordu", k.in, k.n, got, k.want)
		}
	}
}

// --- trendcenter saf kararları ---

func TestTaxonomyWordsKategoriVeEtiket(t *testing.T) {
	tax := repo.TrendTaxonomy{
		Categories: map[int64]string{1: "Ekonomi", 2: "Dünya"},
		Tags:       []string{"Deprem", "Ekonomi"},
	}
	w := taxonomyWords(tax)
	for _, beklenen := range []string{"ekonomi", "dünya", "deprem"} {
		if !w[beklenen] {
			t.Errorf("%q kümede olmalı; küme=%v", beklenen, w)
		}
	}
	// Yinelenen ad tek kelime üretir.
	sayac := 0
	for k := range w {
		if k == "ekonomi" {
			sayac++
		}
	}
	if sayac != 1 {
		t.Errorf("ekonomi %d kez; 1 bekleniyordu", sayac)
	}
}

func TestTaxonomyWordsBos(t *testing.T) {
	if got := taxonomyWords(repo.TrendTaxonomy{}); len(got) != 0 {
		t.Fatalf("boş taksonomi %d kelime üretti", len(got))
	}
}

func TestWordCoverage(t *testing.T) {
	tax := map[string]bool{"ekonomi": true, "enflasyon": true}
	kasalar := []struct {
		ad    string
		terms []string
		want  float64
	}{
		{"tam", []string{"ekonomi", "enflasyon"}, 1},
		{"yarım", []string{"ekonomi", "deprem"}, 0.5},
		{"hiç", []string{"deprem", "sel"}, 0},
		{"boş", nil, 0},
	}
	for _, k := range kasalar {
		if got := wordCoverage(k.terms, tax); got != k.want {
			t.Errorf("%s: got = %v; %v bekleniyordu", k.ad, got, k.want)
		}
	}
}

func TestLogNorm(t *testing.T) {
	// 0 ve negatif → 0
	if got := logNorm(0, 10); got != 0 {
		t.Errorf("logNorm(0) = %v", got)
	}
	if got := logNorm(-5, 10); got != 0 {
		t.Errorf("logNorm(-5) = %v", got)
	}
	// üst sınırda → 1
	if got := logNorm(10, 10); got != 1 {
		t.Errorf("logNorm(10,10) = %v; 1 bekleniyordu", got)
	}
	// üst sınırı aşan da 1 (kırpılır)
	if got := logNorm(1000, 10); got != 1 {
		t.Errorf("logNorm(1000,10) = %v; 1 bekleniyordu", got)
	}
	// ara değer 0..1 arasında ve artan
	a := logNorm(1, 100)
	b := logNorm(50, 100)
	if !(a > 0 && a < b && b < 1) {
		t.Errorf("monotonluk bozuk: a=%v b=%v", a, b)
	}
}

// --- Run: context iptali ---

// Run ctx iptalinde döngüden çıkmalı (havuz nil olsa da tick çağrılmaz).
func TestRunContextIptalindeDoner(t *testing.T) {
	ctx, iptal := context.WithCancel(context.Background())
	logger := slog.New(slog.NewTextHandler(&yazici{}, &slog.HandlerOptions{Level: slog.LevelError}))
	done := make(chan struct{})
	go func() {
		Run(ctx, nil, logger, time.Hour) // saatlik tik: hiç tetiklenmez
		close(done)
	}()
	iptal()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("Run context iptalinde dönmedi (sızıntı)")
	}
}

// every<=0 varsayılana düşmeli; sıfır aralık ticker paniği üretmemeli.
func TestRunSifirAralikPanikYapmaz(t *testing.T) {
	ctx, iptal := context.WithCancel(context.Background())
	logger := slog.New(slog.NewTextHandler(&yazici{}, &slog.HandlerOptions{Level: slog.LevelError}))
	done := make(chan struct{})
	go func() {
		Run(ctx, nil, logger, 0)
		close(done)
	}()
	// Varsayılan 30 sn olduğu için tick çalışmaz; iptal edip çıkıyoruz.
	time.Sleep(50 * time.Millisecond)
	iptal()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("Run dönmedi")
	}
}

// yazici, slog çıktısını yutan minimal io.Writer.
type yazici struct{ sb strings.Builder }

func (y *yazici) Write(p []byte) (int, error) { return y.sb.Write(p) }
