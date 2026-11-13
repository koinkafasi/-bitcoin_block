package httpapi

import (
	"strings"
	"testing"
)

// TestAdsTxtSatirGecerli: geçerli/geçersiz satır ayrımı.
//
// Bu doğrulama GÜVENLİK sınırı: panel ele geçse bile ads.txt'ye HTML ya da
// bozuk satır yazılamamalı.
func TestAdsTxtSatirGecerli(t *testing.T) {
	gecerli := []string{
		"google.com, pub-1234567890123456, DIRECT, f08c47fec0942fa0",
		"ads.example.com, 12345, RESELLER",
		"exchange.example.net, publisher-7, DIRECT",
		"# kısa açıklama",
	}
	for _, s := range gecerli {
		if !adsTxtSatirGecerli(s) {
			t.Errorf("geçerli sayılmalıydı: %q", s)
		}
	}

	gecersiz := []string{
		"",
		"   ",
		"google.com, pub-1",         // eksik alan
		"google.com, pub-1, MAYBE",  // geçersiz ilişki
		"<script>alert(1)</script>", // HTML
		"google.com, pub-1, DIRECT, f08c47fec0942fa0<script>",   // HTML sonda
		"https://google.com, pub-1, DIRECT",                     // şema yasak
		"google.com/path, pub-1, DIRECT",                        // yol yasak
		"noktasiz, pub-1, DIRECT",                               // nokta yok
		`google.com, pub"1, DIRECT`,                             // tırnak
		"google.com, pub-1, DIRECT; DROP",                       // noktalı virgül
		"google.com\\, pub-1, DIRECT",                           // ters bölü
		"google.com, pub 1, DIRECT",                             // boşluklu id
		"google.com, pub-1, DIRECT, ZZZZ",                       // hex olmayan sertifika
		"google.com, pub-1, DIRECT, f08c47fec0942fa0, fazladan", // fazla alan
	}
	for _, s := range gecersiz {
		if adsTxtSatirGecerli(s) {
			t.Errorf("REDDEDİLMELİYDİ: %q", s)
		}
	}
}

// TestAdsTxtGovdeBeyanYokken: hiç ayar yoksa biçimsel olarak geçerli,
// yorumdan ibaret bir dosya üretilir (boş dosya değil).
func TestAdsTxtGovdeBeyanYokken(t *testing.T) {
	s := &Server{logger: testLogger(t), pool: nil}
	// Havuz yok: cachedSetting boş döner.
	g := s.adsTxtGovde(t.Context())
	if strings.TrimSpace(g) == "" {
		t.Fatal("ads.txt tamamen boş olmamalı")
	}
	if !strings.HasPrefix(g, "#") {
		t.Fatalf("beyan yokken yalnız yorum beklenirdi, gelen:\n%s", g)
	}
	// HTML sızmamalı.
	if strings.ContainsAny(g, "<>") {
		t.Fatalf("ads.txt içinde HTML var:\n%s", g)
	}
}

// TestAdsTxtGovdeAdsenseSatiri: geçerli yayıncı kimliği doğru satıra çevrilir.
//
// `ca-pub-X` → `pub-X` dönüşümü AdSense ads.txt biçimidir; yanlış yazılırsa
// satır geçersiz olur ve gelir doğrulaması düşer.
//
// Önek KURALI: yayıncı kimliği `ca-pub-` ile gelir (AdSense panelinden
// kopyalanan değer bu). Çıplak `pub-…` KABUL EDİLMEZ — aynı doğrulayıcı
// `<meta name="google-adsense-account">` için de kullanılıyor ve orada
// `ca-pub-` zorunlu; gevşetmek yanlış meta basılmasına yol açardı.
func TestAdsTxtGovdeAdsenseSatiri(t *testing.T) {
	const girdi = "ca-pub-1234567890123456"
	const beklenen = "google.com, pub-1234567890123456, DIRECT, f08c47fec0942fa0"

	pub := validAdsensePublisher(girdi)
	if pub == "" {
		t.Fatalf("%q geçerli sayılmalıydı", girdi)
	}
	// adsTxtGovde'nin yaptığı dönüşümün aynısı:
	satir := "google.com, " + strings.TrimPrefix(pub, "ca-") + ", DIRECT, f08c47fec0942fa0"
	if satir != beklenen {
		t.Errorf("%q için satır %q, beklenen %q", girdi, satir, beklenen)
	}
	if !adsTxtSatirGecerli(satir) {
		t.Errorf("üretilen satır kendi doğrulamamızdan geçmedi: %q", satir)
	}

	// Çıplak `pub-` reddedilmeli (meta etiketiyle ortak doğrulayıcı).
	if validAdsensePublisher("pub-1234567890123456") != "" {
		t.Error("çıplak `pub-…` kabul edilmemeli: ca-pub- öneki zorunlu")
	}
}

// TestAdsTxtGovdeGecersizSatirAtlanir: ek satırlarda bozuk olan atlanır,
// sağlam olan yazılır. Tek bozuk satır tüm dosyayı düşürmemeli.
func TestAdsTxtGovdeGecersizSatirAtlanir(t *testing.T) {
	// adsTxtGovde ayarları okuyor; doğrulama mantığını doğrudan sınıyoruz.
	girdiler := "ads.example.com, 111, DIRECT\n<script>kötü</script>\nexchange.example.net, 222, RESELLER"
	var yazilan []string
	for _, satir := range strings.Split(girdiler, "\n") {
		if adsTxtSatirGecerli(satir) {
			yazilan = append(yazilan, satir)
		}
	}
	if len(yazilan) != 2 {
		t.Fatalf("2 geçerli satır beklenirdi, %d bulundu: %v", len(yazilan), yazilan)
	}
	for _, s := range yazilan {
		if strings.Contains(s, "script") {
			t.Fatalf("bozuk satır sızdı: %q", s)
		}
	}
}
