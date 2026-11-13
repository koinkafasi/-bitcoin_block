package httpapi

import (
	"math"
	"strings"
	"testing"
)

func ornekBirimler() []CeviriciBirim {
	return []CeviriciBirim{
		{Kod: "BTC", Ad: "Bitcoin", USD: 60000, Kripto: true},
		{Kod: "ETH", Ad: "Ethereum", USD: 3000, Kripto: true},
		{Kod: "USD", Ad: "Amerikan Doları", USD: 1},
		{Kod: "TRY", Ad: "Türk Lirası", USD: 0.025}, // 1 USD = 40 TRY
	}
}

func yakin(t *testing.T, got, istek float64, ad string) {
	t.Helper()
	if math.Abs(got-istek) > istek*1e-9 {
		t.Errorf("%s = %v, beklenen %v", ad, got, istek)
	}
}

// ÇEVİRİ MATEMATİĞİ: her iki birim de USD üzerinden geçiyor. Yanlış
// sonuç kullanıcıyı doğrudan yanıltır, o yüzden köşeler dahil sınandı.
func TestCeviriciCevir(t *testing.T) {
	b := ornekBirimler()
	btc, _ := ceviriciBul("BTC", b)
	eth, _ := ceviriciBul("ETH", b)
	usd, _ := ceviriciBul("USD", b)
	try, _ := ceviriciBul("TRY", b)

	yakin(t, ceviriciCevir(1, btc, usd), 60000, "1 BTC -> USD")
	yakin(t, ceviriciCevir(1, btc, eth), 20, "1 BTC -> ETH")
	yakin(t, ceviriciCevir(1, btc, try), 2400000, "1 BTC -> TRY")
	yakin(t, ceviriciCevir(1, usd, try), 40, "1 USD -> TRY")
	yakin(t, ceviriciCevir(40, try, usd), 1, "40 TRY -> USD")
	// Aynı birim: miktar değişmemeli.
	yakin(t, ceviriciCevir(7, btc, btc), 7, "7 BTC -> BTC")
}

// Hedefin kuru yoksa SIFIR dönmeli; bölme hatası veya NaN basmak
// sayfada "NaN TRY" gösterirdi.
func TestCeviriciCevirKursuzHedef(t *testing.T) {
	kaynak := CeviriciBirim{Kod: "BTC", USD: 60000}
	for _, hedef := range []CeviriciBirim{{Kod: "X", USD: 0}, {Kod: "Y", USD: -1}} {
		if got := ceviriciCevir(1, kaynak, hedef); got != 0 {
			t.Errorf("%s hedefinde sonuç %v, 0 beklenir", hedef.Kod, got)
		}
	}
}

// Girdi temizliği: Türkçe klavyede ondalık ayırıcı VİRGÜL.
func TestCeviriciMiktar(t *testing.T) {
	testler := []struct {
		ham   string
		istek float64
	}{
		{"1", 1},
		{"2.5", 2.5},
		{"2,5", 2.5}, // virgül kabul edilmeli
		{"  3  ", 3}, // kenar boşluğu
		{"", 1},      // boş -> varsayılan
		{"abc", 1},   // çöp -> varsayılan
		{"0", 1},     // sıfır sayfayı boşaltırdı
		{"-5", 1},    // negatif anlamsız
		{"1e13", 1},  // üst sınır aşımı
	}
	for _, tt := range testler {
		if got := ceviriciMiktar(tt.ham); got != tt.istek {
			t.Errorf("ceviriciMiktar(%q) = %v, beklenen %v", tt.ham, got, tt.istek)
		}
	}
}

// Tanınmayan birim kodu sessizce varsayılana düşmeli: adres
// kurcalanmış olabilir, sayfa yine çalışmalı.
func TestCeviriciSec(t *testing.T) {
	b := ornekBirimler()
	if got := ceviriciSec("eth", "BTC", b); got != "ETH" {
		t.Errorf("küçük harf kabul edilmedi: %q", got)
	}
	if got := ceviriciSec("YOKBOYLE", "BTC", b); got != "BTC" {
		t.Errorf("tanınmayan kod varsayılana düşmedi: %q", got)
	}
	if got := ceviriciSec("", "TRY", b); got != "TRY" {
		t.Errorf("boş kod varsayılana düşmedi: %q", got)
	}
}

// ONDALIK BASAMAK BÜYÜKLÜĞE GÖRE: 0,00001234 BTC'yi iki basamakla
// göstermek "0,00" eder ve aracı işe yaramaz kılar.
func TestCeviriciSayi(t *testing.T) {
	testler := []struct {
		n     float64
		ister string
	}{
		{0, "0"},
		{0.00001234, "0.00001234"},
		{0.5, "0.500000"},
		{12.3456, "12.3456"},
		{2400000, "2400000.00"},
	}
	for _, tt := range testler {
		if got := ceviriciSayi(tt.n); got != tt.ister {
			t.Errorf("ceviriciSayi(%v) = %q, beklenen %q", tt.n, got, tt.ister)
		}
	}
}

// Gövde JS KAPALIYKEN de sonuç göstermeli: form GET gönderiyor ve
// hesap sunucuda yapılıyor.
func TestCeviriciGovdeSunucuSonucu(t *testing.T) {
	h := ceviriciGovde(ornekBirimler(), "BTC", "TRY", 2)
	if !strings.Contains(h, "method=\"get\"") {
		t.Error("form GET göndermiyor; JS kapalıyken çalışmaz")
	}
	if !strings.Contains(h, "2400000.00") && !strings.Contains(h, "4800000.00") {
		t.Errorf("sunucu sonucu basılmamış (gövde %d bayt)", len(h))
	}
	// Birim kur satırı ayrıca olmalı.
	if !strings.Contains(h, "1 BTC = ") {
		t.Error("birim kur satırı yok")
	}
	// Yatırım tavsiyesi olmadığı belirtilmeli (YMYL).
	if !strings.Contains(h, "yatırım tavsiyesi değildir") {
		t.Error("uyarı metni yok")
	}
}

// Veri yoksa sayfa ÇÖKMEMELİ, açıklayıcı mesaj basmalı.
func TestCeviriciGovdeVerisiz(t *testing.T) {
	h := ceviriciGovde(nil, "BTC", "TRY", 1)
	if !strings.Contains(h, "alınamadı") {
		t.Errorf("verisiz durumda açıklama yok: %q", h)
	}
	if strings.Contains(h, "<form") {
		t.Error("verisizken boş form basılmış")
	}
}

// Gömülü kur verisi script etiketini erken KAPATMAMALI.
func TestCeviriciGomuluVeriGuvenli(t *testing.T) {
	b := append(ornekBirimler(), CeviriciBirim{Kod: "XX", Ad: "</script><script>alert(1)", USD: 1})
	h := ceviriciGovde(b, "BTC", "TRY", 1)
	if strings.Contains(h, "</script><script>alert") {
		t.Error("gömülü veri script etiketini kapatabiliyor")
	}
}

// Açılır listede kripto ve döviz AYRI gruplarda olmalı.
func TestCeviriciSecimGruplari(t *testing.T) {
	s := ceviriciSecim("kaynak", "Kaynak birim", "BTC", ornekBirimler())
	if !strings.Contains(s, `<optgroup label="Kripto">`) {
		t.Error("kripto grubu yok")
	}
	if !strings.Contains(s, `<optgroup label="Döviz">`) {
		t.Error("döviz grubu yok")
	}
	if !strings.Contains(s, `value="BTC" selected`) {
		t.Error("seçili birim işaretlenmemiş")
	}
}
