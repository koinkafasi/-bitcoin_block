package templkit

import (
	"os"
	"strings"
	"testing"
)

// Yazar sayfaları (12 Kasım 2026) — CoinDesk yazar profili ölçümüyle
// hizalandı: içerik 1032 px ORTALANMIŞ, avatar 240 px SAĞ sütunda,
// h1 36/44, bio 22/30, akış başlığı 16/32 sans.
//
// Bu testler tasarımın sessizce eski hâline (sola yapışık tek sütun)
// dönmesini engeller.

func cdYazarCSSOku(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("../static/cd-yazar.css")
	if err != nil {
		t.Fatalf("cd-yazar.css okunamadı: %v", err)
	}
	return string(b)
}

func TestCDYazarIcerikOrtali(t *testing.T) {
	css := cdYazarCSSOku(t)
	if !strings.Contains(css, "max-width: 1032px") {
		t.Error("yazar içeriği 1032 px olmalı (CoinDesk ölçümü)")
	}
	if !strings.Contains(css, "margin: 0 auto") {
		t.Error("yazar içeriği ORTALANMALI (margin: 0 auto)")
	}
}

func TestCDYazarKimlikIzgarasi(t *testing.T) {
	css := cdYazarCSSOku(t)
	// Sol metin esnek, sağ görsel 240 px sabit.
	if !strings.Contains(css, "grid-template-columns: minmax(0, 1fr) 240px") {
		t.Error("kimlik bloğu metin + 240 px görsel ızgarası olmalı")
	}
	if !strings.Contains(css, ".cd-yzp-avatar") || !strings.Contains(css, "width: 240px") {
		t.Error("avatar 240 px olmalı")
	}
	// Akış satırı: metin + 327 px kapak.
	if !strings.Contains(css, "grid-template-columns: minmax(0, 1fr) 327px") {
		t.Error("akış satırı metin + 327 px kapak ızgarası olmalı")
	}
}

func TestCDYazarTipografi(t *testing.T) {
	css := cdYazarCSSOku(t)
	for _, kural := range []string{
		"font-size: 36px", // h1
		"line-height: 44px",
		"font-size: 22px", // bio
		"line-height: 30px",
		"font-size: 24px", // satır başlığı
		"font-size: 18px", // spot
	} {
		if !strings.Contains(css, kural) {
			t.Errorf("tipografi kuralı eksik: %s", kural)
		}
	}
}

func TestCDYazarMobilTekSutun(t *testing.T) {
	css := cdYazarCSSOku(t)
	i := strings.Index(css, "@media (max-width: 767px)")
	if i < 0 {
		t.Fatal("mobil kırılma noktası 767px olmalı (site geneli)")
	}
	mobil := css[i:]
	if !strings.Contains(mobil, ".cd-yzp-kimlik") {
		t.Error("mobilde kimlik bloğu tek sütuna inmeli")
	}
	if !strings.Contains(mobil, "order: -1") {
		t.Error("mobilde avatar metnin ÜSTÜNE geçmeli (order: -1)")
	}
}

// Görselsiz satır tek sütuna inmeli: depoda kapak çoğu içerikte yok
// (ölçüldü 12 Kasım 2026: 3814 yayındaki içerikten yalnız 8'inde kapak).
func TestCDYazarGorselsizSatir(t *testing.T) {
	css := cdYazarCSSOku(t)
	if !strings.Contains(css, ".cd-yzp-satir--gorselsiz") {
		t.Fatal("görselsiz satır sınıfı tanımlı olmalı")
	}
	tpl := cdYazarTemplOku(t)
	if !strings.Contains(tpl, `templ.KV("cd-yzp-satir--gorselsiz", y.Image == "")`) {
		t.Error("şablon görselsiz satıra sınıf eklemeli")
	}
}

func cdYazarTemplOku(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("cd_yazar.templ")
	if err != nil {
		t.Fatalf("cd_yazar.templ okunamadı: %v", err)
	}
	return string(b)
}

// JSON-LD sayfanın ÜSTÜNDE düz metin olarak görünmemeli: cdHead alanı
// `@templ.Raw` ile basıldığı için üretici <script> sarmalayıcısını
// kendisi eklemek zorunda (12 Kasım 2026 hatası).
func TestCDYazarJSONLDSarmalayici(t *testing.T) {
	b, err := os.ReadFile("../../httpapi/cd_seo.go")
	if err != nil {
		t.Fatalf("cd_seo.go okunamadı: %v", err)
	}
	src := string(b)
	i := strings.Index(src, "func cdKisiJSONLD(")
	if i < 0 {
		t.Fatal("cdKisiJSONLD bulunamadı")
	}
	govde := src[i:]
	if strings.Contains(govde, "template.JS(b)") {
		t.Error("cdKisiJSONLD sarmalayıcısız JSON döndürmemeli (sayfada düz metin görünür)")
	}
	if !strings.Contains(govde, "return jsonldScript(blok)") {
		t.Error("cdKisiJSONLD jsonldScript ile <script> sarmalayıcısı üretmeli")
	}
}
