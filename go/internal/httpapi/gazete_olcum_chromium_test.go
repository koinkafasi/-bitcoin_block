package httpapi

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGazeteOlcumChromium(t *testing.T) {
	if os.Getenv("GAZETE_CHROMIUM_TEST") == "" {
		t.Skip("GAZETE_CHROMIUM_TEST ayarlı değil")
	}
	chrome := findChrome()
	if chrome == "" {
		t.Skip("Chromium bulunamadı")
	}

	// Biri kasten taşan, biri rahat sığan iki blok.
	tasan := gazeteBlock{
		ID: "tasan", Tip: "metin", X: 2, Y: 2, W: 40, H: 6,
		Metin: strings.Repeat("Bu metin bloğa kesinlikle sığmayacak kadar uzun. ", 40),
	}
	sigan := gazeteBlock{
		ID: "sigan", Tip: "metin", X: 50, Y: 2, W: 40, H: 40,
		Metin: "Kısa metin.",
	}

	boyut := gazeteBoyut("A4")
	doc := `<!doctype html><html lang="tr"><head><meta charset="utf-8"><style>` +
		gazeteSayfaDegiskenleri(boyut, 1) + "\n" + gazeteBaskiCSS() +
		`</style></head><body><div class="gazete-sayfa" data-sayfa="1">` +
		gazeteBlockHTML(tasan) + gazeteBlockHTML(sigan) +
		`</div>` + olcumBetigi(true) + `</body></html>`

	dir := t.TempDir()
	yol := filepath.Join(dir, "olcum.html")
	if err := os.WriteFile(yol, []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(chrome,
		"--headless=new", "--disable-gpu", "--no-sandbox", "--disable-dev-shm-usage",
		"--virtual-time-budget=10000", "--dump-dom", "file://"+yol)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("Chromium çalıştırılamadı: %v", err)
	}

	tasmalar, err := gazeteTasmaAyikla(string(out))
	if err != nil {
		t.Fatalf("rapor ayıklanamadı: %v", err)
	}

	// Yalnız taşan blok bildirilmeli; sığan blok rapora girmemeli.
	if len(tasmalar) != 1 {
		t.Fatalf("1 taşma beklenirken %d: %+v", len(tasmalar), tasmalar)
	}
	g := tasmalar[0]
	if g.BlokID != "tasan" {
		t.Errorf("yanlış blok bildirildi: %q", g.BlokID)
	}
	if g.Sayfa != 1 {
		t.Errorf("sayfa numarası yanlış: %d", g.Sayfa)
	}
	if g.Dikey <= 0 {
		t.Errorf("dikey taşma ölçülmedi: %+v", g)
	}
	if g.Oneri == "" {
		t.Error("öneri üretilmedi")
	}
	t.Logf("ölçüldü: %s sayfa %d · dikey %.1fpx · kutu %.0fpx · %s",
		g.BlokID, g.Sayfa, g.Dikey, g.KutuY, g.Oneri)
}
