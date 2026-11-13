package httpapi

import (
	"os"
	"strings"
	"testing"
)

// SIRA TUZAĞI: v.Locale, makaleJSONLD çağrısından ÖNCE kurulmalı.
//
// Üretimde ölçüldü (13 Kasım 2026): İngilizce makale <html lang="en">
// ile basılıyor ama yapısal veride inLanguage="tr" diyordu. Sebep
// mantık değil SIRAYDI — v.Locale satır ~360'ta atanıyordu, makaleJSONLD
// ise satır ~300'de çağrılıyordu, yani boş locale görüp "tr"ye
// düşüyordu.
//
// Bu testin kaynak metnine bakmasının sebebi: makaleJSONLD'nin birim
// testi Locale'i DOĞRUDAN verdiği için geçiyordu ve hatayı göremiyordu.
// Hata çağıranın sırasındaydı; denetlenmesi gereken de o.
func TestCDArticleLocaleJSONLDdenOnceKurulur(t *testing.T) {
	ham, err := os.ReadFile("cd_article.go")
	if err != nil {
		t.Fatalf("cd_article.go okunamadı: %v", err)
	}
	src := string(ham)

	iLocale := strings.Index(src, "v.Locale = ")
	if iLocale < 0 {
		t.Fatal("v.Locale ataması bulunamadı; alan adı değiştiyse bu testi güncelleyin")
	}
	iLD := strings.Index(src, "makaleJSONLD(")
	if iLD < 0 {
		t.Fatal("makaleJSONLD çağrısı bulunamadı")
	}
	if iLocale > iLD {
		t.Error("v.Locale makaleJSONLD'den SONRA atanıyor: " +
			"yapısal veri boş locale görüp \"tr\"ye düşer (İngilizce sayfada yanlış dil sinyali)")
	}

	// Tek atama olmalı: ikinci bir atama sıranın yeniden bozulması demek.
	if n := strings.Count(src, "v.Locale = "); n != 1 {
		t.Errorf("v.Locale %d kez atanıyor; tek noktada kalmalı", n)
	}
}

// Aynı tuzak dil sözlüğü için de geçerli: v.Dil şablona gidiyor ve
// kaynak satırının dilini belirliyor.
func TestCDArticleDilSozluguErkenKurulur(t *testing.T) {
	ham, err := os.ReadFile("cd_article.go")
	if err != nil {
		t.Fatalf("cd_article.go okunamadı: %v", err)
	}
	src := string(ham)

	iDil := strings.Index(src, "v.Dil = ")
	iLD := strings.Index(src, "makaleJSONLD(")
	if iDil < 0 || iLD < 0 {
		t.Fatal("v.Dil ataması veya makaleJSONLD çağrısı bulunamadı")
	}
	if iDil > iLD {
		t.Error("v.Dil makaleJSONLD'den sonra atanıyor")
	}
}
