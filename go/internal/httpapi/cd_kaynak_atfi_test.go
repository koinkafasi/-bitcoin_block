package httpapi

import (
	"encoding/json"
	"strings"
	"testing"

	"theblock/site/internal/webui/templkit"
)

// ldCoz NewsArticle bloğunu <script> sarmalından çıkarıp ayrıştırır.
func ldCoz(t *testing.T, g makaleLDGirdi) map[string]any {
	t.Helper()
	ham := string(makaleJSONLD(g))
	i := strings.Index(ham, ">")
	j := strings.LastIndex(ham, "<")
	if i < 0 || j <= i {
		t.Fatalf("script sarmalı beklenen biçimde değil: %q", ham)
	}
	var d map[string]any
	if err := json.Unmarshal([]byte(ham[i+1:j]), &d); err != nil {
		t.Fatalf("JSON-LD ayrıştırılamadı: %v\n%s", err, ham[i+1:j])
	}
	return d
}

func temelGirdi() makaleLDGirdi {
	return makaleLDGirdi{
		View: templkit.CDArticleView{
			Headline:  "Başlık",
			Canonical: "https://bitcosar.com/x",
			BaseURL:   "https://bitcosar.com",
			Locale:    "tr",
		},
		YayinISO: "2026-11-13T10:00:00Z",
	}
}

// Sabit "tr" iken İngilizce çeviriler <html lang="en"> ile basılıp
// yapısal veride "tr" diyordu: canlıda ölçüldü, aynı sayfa Google'a iki
// farklı dil sinyali veriyordu.
func TestMakaleJSONLDDilSayfadanGelir(t *testing.T) {
	testler := []struct{ locale, istek string }{
		{"tr", "tr"},
		{"en", "en"},
		{"EN", "en"},
		{" en ", "en"},
		{"", "tr"}, // eski kayıtlarda alan boş ve o içerikler Türkçe
		{"tr-TR", "tr-tr"},
	}
	for _, tt := range testler {
		g := temelGirdi()
		g.View.Locale = tt.locale
		if got := ldCoz(t, g)["inLanguage"]; got != tt.istek {
			t.Errorf("locale %q → inLanguage %v, beklenen %q", tt.locale, got, tt.istek)
		}
	}
}

// Kaynak adresi elimizdeydi ve sayfada görünür basılıyordu ama yapısal
// veriye hiç girmiyordu. E-E-A-T'nin taşıyıcısı bu alan.
func TestMakaleJSONLDKaynakAtfi(t *testing.T) {
	g := temelGirdi()
	g.View.SourceURL = "https://cryptobriefing.com/haber"
	g.View.SourceName = "CryptoBriefing"
	d := ldCoz(t, g)

	if d["isBasedOn"] != "https://cryptobriefing.com/haber" {
		t.Errorf("isBasedOn = %v", d["isBasedOn"])
	}
	atiflar, ok := d["citation"].([]any)
	if !ok || len(atiflar) != 1 {
		t.Fatalf("citation listesi bekleniyordu: %#v", d["citation"])
	}
	atif, _ := atiflar[0].(map[string]any)
	if atif["@type"] != "CreativeWork" {
		t.Errorf("citation.@type = %v", atif["@type"])
	}
	if atif["url"] != "https://cryptobriefing.com/haber" {
		t.Errorf("citation.url = %v", atif["url"])
	}
	if atif["name"] != "CryptoBriefing" {
		t.Errorf("citation.name = %v", atif["name"])
	}
}

// Kaynağı olmayan içerik (kendi rehberlerimiz) uydurma atıf almamalı.
func TestMakaleJSONLDKaynaksizIcerikteAtifYok(t *testing.T) {
	d := ldCoz(t, temelGirdi())
	if _, varMi := d["isBasedOn"]; varMi {
		t.Error("kaynaksız içerikte isBasedOn yazılmamalı")
	}
	if _, varMi := d["citation"]; varMi {
		t.Error("kaynaksız içerikte citation yazılmamalı")
	}
}

// Adı olmayan kaynakta bile adres atfı yazılmalı: eksik etiket, atfı
// tümden düşürmek için gerekçe değil.
func TestMakaleJSONLDAdsizKaynak(t *testing.T) {
	g := temelGirdi()
	g.View.SourceURL = "https://ornek.com/x"
	d := ldCoz(t, g)
	if d["isBasedOn"] != "https://ornek.com/x" {
		t.Fatalf("isBasedOn = %v", d["isBasedOn"])
	}
	atif := d["citation"].([]any)[0].(map[string]any)
	if _, varMi := atif["name"]; varMi {
		t.Error("ad yokken citation.name yazılmamalı")
	}
}

// Boş dize, alanı atlamaktan daha kötü: doğrulayıcı onu geçersiz değer
// sayıp NewsArticle'ı hatalı işaretliyordu.
func TestMakaleJSONLDYayinciLogosu(t *testing.T) {
	g := temelGirdi()
	if _, varMi := ldCoz(t, g)["publisher"].(map[string]any)["logo"]; varMi {
		t.Error("logo yokken alan hiç yazılmamalı")
	}

	g.Logo = "https://bitcosar.com/static/logo.png"
	logo, ok := ldCoz(t, g)["publisher"].(map[string]any)["logo"].(map[string]any)
	if !ok {
		t.Fatal("logo ImageObject olmalı")
	}
	if logo["@type"] != "ImageObject" || logo["url"] != g.Logo {
		t.Errorf("logo = %#v", logo)
	}
}

func TestMakaleJSONLDBaglamHTTPS(t *testing.T) {
	if got := ldCoz(t, temelGirdi())["@context"]; got != "https://schema.org" {
		t.Errorf("@context = %v", got)
	}
}
