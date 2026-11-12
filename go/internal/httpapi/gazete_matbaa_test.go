package httpapi

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Faz 7 koruyucu testleri — specs/13-gazete-sayfa-duzenleyici.md K8

func TestGazeteMatbaaGeometrisi(t *testing.T) {
	g := gazeteMatbaaGeometrisi(210, 297, 3)

	// Kesim kutusu tam A4 olmalı.
	if f := fark(g.TrimW, 210*mmPt); f > 0.01 {
		t.Errorf("kesim genişliği yanlış: %.3f pt (sapma %.3f)", g.TrimW, f)
	}
	if f := fark(g.TrimH, 297*mmPt); f > 0.01 {
		t.Errorf("kesim yüksekliği yanlış: %.3f pt", g.TrimH)
	}
	// Sayfa kutusu her kenardan (3+8)=11mm büyük olmalı.
	if f := fark(g.MediaW, (210+22)*mmPt); f > 0.01 {
		t.Errorf("sayfa kutusu genişliği yanlış: %.3f pt", g.MediaW)
	}
	if f := fark(g.MediaH, (297+22)*mmPt); f > 0.01 {
		t.Errorf("sayfa kutusu yüksekliği yanlış: %.3f pt", g.MediaH)
	}
	// Kesim kutusu ortalanmış olmalı.
	if f := fark(g.TrimX0, (8+3)*mmPt); f > 0.01 {
		t.Errorf("kesim kutusu ortalanmamış: x0=%.3f", g.TrimX0)
	}
	// Taşma kutusu kesimi sarmalı, sayfa kutusunu aşmamalı.
	if g.BleedX0 >= g.TrimX0 {
		t.Error("taşma kutusu kesim kutusunun içinde kalmış")
	}
	if g.BleedX0+g.BleedW > g.MediaW+0.01 {
		t.Error("taşma kutusu sayfa kutusunu aşıyor")
	}

	// Taşma payı sıfır: kesim ve taşma kutuları çakışır, işaret alanı kalır.
	s := gazeteMatbaaGeometrisi(210, 297, 0)
	if fark(s.TrimX0, s.BleedX0) > 0.01 {
		t.Errorf("taşma sıfırken kutular ayrışmış: %.3f vs %.3f", s.TrimX0, s.BleedX0)
	}
	if fark(s.MediaW, (210+16)*mmPt) > 0.01 {
		t.Errorf("işaret alanı uygulanmamış: %.3f", s.MediaW)
	}
}

// TestGazetePDFXAnahtari sürüm eşlemesini sabitler. Yanlış anahtar, matbaanın
// istediğinden başka bir standartta dosya üretir.
func TestGazetePDFXAnahtari(t *testing.T) {
	for giris, beklenen := range map[string]string{
		"X-1a":     "/PDF/X-1a:2001",
		"x1a":      "/PDF/X-1a:2001",
		"PDF/X-1a": "/PDF/X-1a:2001",
		"X-4":      "/PDF/X-4",
		"x4":       "/PDF/X-4",
		"X-3":      "/PDF/X-3:2002",
		"":         "/PDF/X-3:2002",
		"bilinmez": "/PDF/X-3:2002",
	} {
		if got := gazetePDFXAnahtari(giris); got != beklenen {
			t.Errorf("%q: %q beklenirken %q", giris, beklenen, got)
		}
	}
}

// TestGazetePDFXTanimi PostScript tanımının çıktı amacını ve kutuları
// bildirdiğini doğrular.
func TestGazetePDFXTanimi(t *testing.T) {
	a := gazeteMatbaaAyar{PDFX: "X-3", Kosul: "CGATS TR 001", ICCYolu: "/tmp/x.icc"}
	ps := gazetePDFXTanimi(a, 210, 297, 3)

	for _, gerek := range []string{
		"/Type /OutputIntent", "/S /GTS_PDFX", "DestOutputProfile",
		"/TrimBox", "/BleedBox", "setpagedevice", "/tmp/x.icc",
	} {
		if !strings.Contains(ps, gerek) {
			t.Errorf("tanımda %q yok", gerek)
		}
	}
	// ICC yoksa çıktı amacı yazılmamalı; kutular yine yazılmalı.
	bos := gazetePDFXTanimi(gazeteMatbaaAyar{PDFX: "X-3"}, 210, 297, 3)
	if strings.Contains(bos, "OutputIntent") {
		t.Error("ICC yokken çıktı amacı yazıldı; dosya geçersiz PDF/X olurdu")
	}
	if !strings.Contains(bos, "/TrimBox") {
		t.Error("ICC yokken kutular da yazılmamış")
	}
}

func TestGazetePSKacir(t *testing.T) {
	for giris, beklenen := range map[string]string{
		`/tmp/a(b).icc`: `/tmp/a\(b\).icc`,
		`a\b`:           `a\\b`,
		`düz`:           `düz`,
	} {
		if got := gazetePSKacir(giris); got != beklenen {
			t.Errorf("%q: %q beklenirken %q", giris, beklenen, got)
		}
	}
	// Kaçırılmış dizgi tanıma konunca parantez dengesi bozulmamalı.
	a := gazeteMatbaaAyar{ICCYolu: `/tmp/a(b).icc`, Kosul: "X)Y"}
	ps := gazetePDFXTanimi(a, 210, 297, 3)
	if strings.Contains(ps, `(/tmp/a(b).icc)`) {
		t.Error("ICC yolundaki parantez kaçırılmadı")
	}
}

// TestGazeteMatbaaPreflightYakalar preflight'ın eksikleri sessiz geçmediğini
// doğrular. Geçerse matbaa dosyayı geri çevirir ve bunu biz görmeliyiz.
func TestGazeteMatbaaPreflightYakalar(t *testing.T) {
	s := &Server{}
	// Var olmayan dosya: her denetim başarısız olmalı ve uyarı üretmeli.
	rap := s.gazeteMatbaaPreflight(context.Background(),
		filepath.Join(t.TempDir(), "yok.pdf"), gazeteMatbaaRapor{})
	if rap.Hazir() {
		t.Error("eksik dosya için hazır bildirildi")
	}
	birlesik := strings.Join(rap.Uyarilar, " | ")
	for _, gerek := range []string{"TrimBox", "BleedBox", "OutputIntent"} {
		if !strings.Contains(birlesik, gerek) {
			t.Errorf("%q eksikliği bildirilmedi: %s", gerek, birlesik)
		}
	}

	// ICC ayarlı değilse ayrıca uyarılmalı: genel CMYK ile baskı renk
	// kayması demek.
	if !strings.Contains(birlesik, "ICC") {
		t.Errorf("ICC uyarısı yok: %s", birlesik)
	}
}

// TestGazeteMatbaaDijitaliBozmaz matbaa dosyasının dijital nüshadan ayrı bir
// anahtarla saklandığını doğrular (SPEC K8).
func TestGazeteMatbaaDijitaliBozmaz(t *testing.T) {
	if gazeteMatbaaDosya == "gazete.pdf" {
		t.Fatal("matbaa dosyası dijital nüshanın üstüne yazıyor")
	}
	if !strings.HasSuffix(gazeteMatbaaDosya, ".pdf") {
		t.Errorf("matbaa dosyası adı beklenmedik: %q", gazeteMatbaaDosya)
	}
	// Üretim kodu dijital nüshanın yolunu güncelleyen çağrıyı içermemeli.
	b, err := os.ReadFile("gazete_matbaa.go")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "SetGazeteIssuePDF") {
		t.Error("matbaa üretimi dijital nüshanın pdf_path kaydını değiştiriyor")
	}
}

// TestGazeteKesimIsaretleri dört köşede de işaret çifti üretildiğini doğrular.
func TestGazeteKesimIsaretleri(t *testing.T) {
	h := gazeteKesimIsaretleri()
	if n := strings.Count(h, "gz-kesim-im"); n != 8 {
		t.Errorf("4 köşe × 2 işaret = 8 beklenirken %d", n)
	}
	if strings.Count(h, "yatay") != 4 || strings.Count(h, "dikey") != 4 {
		t.Errorf("yatay/dikey dengesi bozuk: %s", h)
	}
	// İşaretler kesim çizgisinden boşlukla ayrılmalı.
	if !strings.Contains(h, "2mm") {
		t.Error("işaretler kesim çizgisine bitişik; operatör çizgiyi göremez")
	}
}
