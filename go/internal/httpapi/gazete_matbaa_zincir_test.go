package httpapi

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGazeteMatbaaZinciri(t *testing.T) {
	if os.Getenv("GAZETE_MATBAA_TEST") == "" {
		t.Skip("GAZETE_MATBAA_TEST ayarlı değil")
	}
	chrome := findChrome()
	if chrome == "" {
		t.Skip("Chromium bulunamadı")
	}
	gsYol, err := exec.LookPath("gs")
	if err != nil {
		t.Skip("Ghostscript bulunamadı")
	}

	const tasma = 3.0
	boyut := gazeteBoyut("A4")
	trimW, trimH := gazeteMmSayi(boyut.W), gazeteMmSayi(boyut.H)
	kenarMm := gazeteIsaretAlaniMm + tasma
	medyaW, medyaH := trimW+2*kenarMm, trimH+2*kenarMm

	blok := gazeteBlockHTML(gazeteBlock{
		ID: "m", Tip: "manset", X: 0, Y: 0, W: 100, H: 60,
		Icerik: &gazeteIcerik{Haberler: []gazeteArticle{{
			Baslik: "Matbaa denemesi", Spot: "Kesim ve taşma payı sınaması.",
			Kategori: "Deneme", Kaynak: "theblock", Tarih: "30.10.2026"}}},
	})

	doc := fmt.Sprintf(`<!doctype html><html lang="tr"><head><meta charset="utf-8"><style>
@page { size: %.3fmm %.3fmm; margin: 0; }
body{margin:0}
:root{--gz-sayfa-w:%s;--gz-sayfa-h:%s;--gz-kenar:%s;--gz-tip-tab:%s;
      --gz-tip-olcek:1;--gz-yazi:%s;--gz-matbaa-kenar:%.3fmm;}
%s
%s
</style></head><body>
<div class="gz-matbaa-sayfa">%s<div class="gazete-sayfa">%s</div></div>
</body></html>`,
		medyaW, medyaH, boyut.W, boyut.H, boyut.Kenar, boyut.TipTab,
		gazeteYaziTipiCSS(""), kenarMm,
		gazeteBaskiCSS(), gazeteMatbaaCSS,
		gazeteKesimIsaretleri(), blok)

	dir := t.TempDir()
	htmlPath := filepath.Join(dir, "matbaa.html")
	if err := os.WriteFile(htmlPath, []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	hamPDF := filepath.Join(dir, "ham.pdf")
	cmd := exec.Command(chrome,
		"--headless=new", "--disable-gpu", "--no-sandbox", "--disable-dev-shm-usage",
		"--no-pdf-header-footer", "--virtual-time-budget=10000",
		"--print-to-pdf="+hamPDF, "file://"+htmlPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Chromium: %v (%s)", err, firstLine(string(out), 200))
	}

	ayar := gazeteMatbaaAyar{PDFX: "X-3", Kosul: "CGATS TR 001",
		ICCYolu: gazeteVarsayilanCMYKProfili()}
	if ayar.ICCYolu == "" {
		t.Skip("CMYK ICC profili bulunamadı")
	}
	t.Logf("ICC profili: %s", ayar.ICCYolu)

	defPath := filepath.Join(dir, "pdfx_def.ps")
	if err := os.WriteFile(defPath,
		[]byte(gazetePDFXTanimi(ayar, trimW, trimH, tasma)), 0o644); err != nil {
		t.Fatal(err)
	}
	cikti := filepath.Join(dir, "matbaa.pdf")
	args := []string{
		"-dBATCH", "-dNOPAUSE", "-dSAFER", "-dQUIET",
		"-sDEVICE=pdfwrite", "-dPDFX", "-dPDFSETTINGS=/prepress",
		"-dCompatibilityLevel=1.6",
		"-sColorConversionStrategy=CMYK", "-dProcessColorModel=/DeviceCMYK",
		"-dAutoRotatePages=/None",
		"-sOutputICCProfile=" + ayar.ICCYolu,
		"-sOutputFile=" + cikti,
		defPath, hamPDF,
	}
	if out, err := exec.Command(gsYol, args...).CombinedOutput(); err != nil {
		t.Fatalf("Ghostscript: %v\n%s", err, string(out))
	}

	s := &Server{}
	rap := s.gazeteMatbaaPreflight(context.Background(), cikti, gazeteMatbaaRapor{ICC: ayar.ICCYolu})

	t.Logf("sayfa boyu: %s", rap.SayfaBoyu)
	t.Logf("TrimBox: %v · BleedBox: %v · OutputIntent: %v",
		rap.TrimBox, rap.BleedBox, rap.CiktiAmaci)
	if len(rap.Uyarilar) > 0 {
		t.Logf("uyarılar: %s", strings.Join(rap.Uyarilar, " | "))
	}

	// Matbaa dosyasının olmazsa olmazları.
	if !rap.CiktiAmaci {
		t.Error("OutputIntent yok: dosya PDF/X sayılmaz")
	}
	if !rap.TrimBox {
		t.Error("TrimBox yok: matbaa kesimin nereden geçeceğini bilemez")
	}
	if !rap.BleedBox {
		t.Error("BleedBox yok: taşma payı tanımsız")
	}
	if len(rap.GomulmeyenF) > 0 {
		t.Errorf("gömülü olmayan font: %v", rap.GomulmeyenF)
	}
	gotW, gotH := sayfaBoyuPt(t, rap.SayfaBoyu)
	beklW := (trimW + 2*kenarMm) * mmPt
	beklH := (trimH + 2*kenarMm) * mmPt
	if fark(gotW, beklW) > 2 || fark(gotH, beklH) > 2 {
		t.Errorf("sayfa kutusu %.2f×%.2f pt; beklenen ≈%.2f×%.2f pt",
			gotW, gotH, beklW, beklH)
	}
}

// sayfaBoyuPt "658.08 x 904.08 pts" biçimini sayıya çevirir.
func sayfaBoyuPt(t *testing.T, s string) (float64, float64) {
	t.Helper()
	alan := strings.Fields(s)
	if len(alan) < 3 {
		t.Fatalf("sayfa boyu çözülemedi: %q", s)
	}
	var w, h float64
	if _, err := fmt.Sscanf(alan[0], "%f", &w); err != nil {
		t.Fatalf("genişlik çözülemedi: %q", alan[0])
	}
	if _, err := fmt.Sscanf(alan[2], "%f", &h); err != nil {
		t.Fatalf("yükseklik çözülemedi: %q", alan[2])
	}
	return w, h
}

func fark(a, b float64) float64 {
	if a > b {
		return a - b
	}
	return b - a
}
