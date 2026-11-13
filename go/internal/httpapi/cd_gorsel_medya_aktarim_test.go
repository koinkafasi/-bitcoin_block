package httpapi

import (
	"strings"
	"testing"

	"theblock/site/internal/repo"
)

// Dosya adı doğrudan depolama yoluna giriyor: yol ayracı, boşluk veya
// Unicode sızması hem dosya sistemi hem URL tarafında sorun çıkarır.
func TestTemizDosyaAdiGuvenliKarakterlerBirakir(t *testing.T) {
	testler := []struct {
		girdi, istek, neden string
	}{
		{"shutterstock_12345", "shutterstock_12345", "alt çizgi korunur"},
		{"Bitcoin Fiyat Analizi", "bitcoin-fiyat-analizi", "boşluk tireye döner"},
		{"../../etc/passwd", "etcpasswd", "yol ayracı ve nokta-nokta düşer"},
		// Go'da ToLower('İ') = 'i' ve 'i' güvenli; diğer Türkçe harfler düşer.
		{"görsel-ÇÖĞÜŞİ", "grsel-i", "ASCII'ye inen harf kalır, diğerleri düşer"},
		{"a/b\\c", "abc", "iki tür ayraç da düşer"},
		{"...", "", "yalnız noktadan ad üretilmez"},
		// Yüzde işaretleri düşer, rakamlar güvenli olduğu için kalır.
		{"%20%00null", "2000null", "yüzde işareti düşer, rakam kalır"},
	}
	for _, tt := range testler {
		if got := temizDosyaAdi(tt.girdi); got != tt.istek {
			t.Errorf("temizDosyaAdi(%q) = %q, beklenen %q (%s)",
				tt.girdi, got, tt.istek, tt.neden)
		}
	}
}

// Ad üretimi hiçbir girdide boş uzantısız ad vermemeli: SaveMedia
// uzantıyı dosya adından okuyor, boş kalırsa ".bin" yazıyor.
func TestGorselDosyaAdiHerZamanUzantiliVeGuvenli(t *testing.T) {
	const hash = "abc123def456"
	testler := []struct {
		ad, url, mime, istekSonek string
	}{
		{"jpeg", "https://x.com/foto.jpg", "image/jpeg", ".jpg"},
		{"webp sorgulu", "https://cdn.x.com/i?id=9", "image/webp", ".webp"},
		{"uzantisiz CDN", "https://cdn.x.com/image/12345", "image/png", ".png"},
		{"mime yok", "https://x.com/a.bin", "", ".jpg"},
		{"avif", "https://x.com/y.avif", "image/avif", ".avif"},
		{"mime parametreli", "https://x.com/z", "image/jpeg; charset=binary", ".jpg"},
	}
	for _, tt := range testler {
		got := gorselDosyaAdi(tt.url, hash, tt.mime)
		if !strings.HasSuffix(got, tt.istekSonek) {
			t.Errorf("%s: gorselDosyaAdi(%q, _, %q) = %q, %q ile bitmeli",
				tt.ad, tt.url, tt.mime, got, tt.istekSonek)
		}
		if strings.ContainsAny(got, `/\ `) {
			t.Errorf("%s: ad güvensiz karakter taşıyor: %q", tt.ad, got)
		}
		if got == tt.istekSonek {
			t.Errorf("%s: ad yalnız uzantıdan oluştu: %q", tt.ad, got)
		}
	}
}

// Kaynak adı kullanılamazsa hash'e düşmeli — adsız kayıt olmamalı.
func TestGorselDosyaAdiHasheDuser(t *testing.T) {
	got := gorselDosyaAdi("https://x.com/...", "deadbeef", "image/jpeg")
	if !strings.HasPrefix(got, "deadbeef") {
		t.Fatalf("kaynak adı kullanılamazken hash'e düşmedi: %q", got)
	}
}

// Alt metin ekran okuyucu ve arama motoru sınırına uymalı, kelime
// ortasından kesilmemeli.
func TestKirpAltMetin(t *testing.T) {
	kisa := "Bitcoin yükseldi"
	if got := kirpAltMetin(kisa, 125); got != kisa {
		t.Errorf("sınır altı metin değişti: %q", got)
	}

	uzun := strings.Repeat("Bitcoin fiyat analizi ", 20) // ~440 karakter
	got := kirpAltMetin(uzun, 125)
	if len([]rune(got)) > 126 { // +1: eklenen "…"
		t.Errorf("kırpılmış metin %d karakter, sınır 125", len([]rune(got)))
	}
	if !strings.HasSuffix(got, "…") {
		t.Errorf("kırpma işareti yok: %q", got)
	}
	// Kelime ortasından kesilmemeli: "…" öncesi harf-sonu olmalı, yani
	// kırpık metin tam kelimeyle bitmeli.
	govde := strings.TrimSuffix(got, "…")
	if strings.HasSuffix(govde, " ") {
		t.Errorf("kırpma sonunda boşluk kaldı: %q", got)
	}
}

// Türkçe karakterli metinde rune sayımı: len() bayt sayar ve erken keserdi.
func TestKirpAltMetinTurkceKarakter(t *testing.T) {
	// Her karakter 2 bayt: len() 125'i 62 karakterde sanırdı.
	metin := strings.Repeat("ğüşıöç", 30) // 180 rune, 360 bayt
	got := kirpAltMetin(metin, 125)
	n := len([]rune(got))
	if n < 100 {
		t.Errorf("Türkçe metin %d karakterde kesildi; rune değil bayt sayılmış olabilir", n)
	}
}

func TestGorselAktarimRaporu(t *testing.T) {
	if got := gorselAktarimRaporu(repo.GorselAktarimOzeti{}); got != "kaynak görsel yok" {
		t.Errorf("boş özet için %q", got)
	}
	got := gorselAktarimRaporu(repo.GorselAktarimOzeti{Toplam: 712, Aktarilan: 178, Bekleyen: 534})
	for _, parca := range []string{"178/712", "25.0%", "534"} {
		if !strings.Contains(got, parca) {
			t.Errorf("rapor %q içermiyor: %q", parca, got)
		}
	}
}

func TestGorselUzanti(t *testing.T) {
	testler := map[string]string{
		"image/jpeg": ".jpg",
		"image/png":  ".png",
		"image/webp": ".webp",
		"image/gif":  ".gif",
		"image/avif": ".avif",
		"":           ".jpg", // bilinmeyen: güvenli varsayılan
	}
	for mimeTipi, istek := range testler {
		if got := gorselUzanti(mimeTipi); got != istek {
			t.Errorf("gorselUzanti(%q) = %q, beklenen %q", mimeTipi, got, istek)
		}
	}
}
