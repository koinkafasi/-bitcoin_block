package httpapi

import (
	"strings"
	"testing"

	"theblock/site/internal/repo"
)

// ZİYARET NİYETİ TESTLERİ — sınıflandırma, maskeleme ve çıkarım.
//
// Hepsi SAF fonksiyon: DNS/DB yok. Asıl sınanan üç şey:
//  1) sınıflandırma sırası (yönetim > zafiyet > besleme > arama > giriş),
//  2) sorgu/yol maskelemesi (jeton bu tabloya düşmemeli),
//  3) niyet eşikleri (tek zayıf sinyal "tehdit" dememeli).

func TestOlayTuru(t *testing.T) {
	ornekler := map[string]string{
		"/":                       turIcerik,
		"/haber/bitcoin-100-bin":  turIcerik,
		"/kripto/ethereum":        turIcerik,
		"/rss.xml":                turBesleme,
		"/sitemap.xml":            turBesleme,
		"/robots.txt":             turBesleme,
		"/news-sitemap":           turBesleme,
		"/ara":                    turArama,
		"/ara?q=bitcoin":          turArama,
		"/api/piyasa":             turAPI,
		"/admin":                  turYonetim,
		"/admin/trafik-guvenligi": turYonetim,
		"/admin/giris":            turYonetim, // panelin kendi girişi
		"/giris":                  turGiris,
		"/kayit":                  turGiris,
		"/sifre-sifirla":          turGiris,
		"/wp-login.php":           turZafiyet,
		"/blog/wp-login.php":      turZafiyet,
		"/.env":                   turZafiyet,
		"/.git/config":            turZafiyet,
		"/phpmyadmin/index.php":   turZafiyet,
		"/xmlrpc.php":             turZafiyet,
		"/cgi-bin/luci":           turZafiyet,
		"/vendor/phpunit/x.php":   turZafiyet,
		// Tuzak yolu en önce: honeypot'a düşen istek başka bir şey değil.
		trapYoluTest(): turTuzak,
		"":             turDiger,
	}
	for yol, bekle := range ornekler {
		if got := olayTuru(yol); got != bekle {
			t.Errorf("olayTuru(%q) = %q, beklenen %q", yol, got, bekle)
		}
	}
}

// trapYoluTest isTrapPath'in tanıdığı gerçek bir yolu döndürür. Sabit
// yazmak yerine fonksiyondan alınıyor: honeypot listesi değişirse test
// sessizce yanlış şeyi sınamasın.
func trapYoluTest() string {
	for _, p := range []string{
		"/arsiv/dizin", "/gizli-dizin", "/tum-haberler-dizin",
	} {
		if isTrapPath(p) {
			return p
		}
	}
	return "/arsiv/dizin"
}

func TestOlayKaynagi(t *testing.T) {
	const host = "bitcosar.com"
	ornekler := []struct{ ref, bekle string }{
		{"", kaynakYok},
		{"https://bitcosar.com/haber/x", kaynakIc},
		{"https://www.bitcosar.com/", kaynakIc},
		{"https://bitcosar.com.tr/x", kaynakDis}, // benzer ad, farklı site
		{"https://www.google.com/search?q=x", kaynakArama},
		{"https://www.bing.com/search?q=x", kaynakArama},
		{"https://duckduckgo.com/", kaynakArama},
		{"https://news.google.com/", kaynakArama},
		{"https://t.co/abc", kaynakSosyal},
		{"https://x.com/kullanici/status/1", kaynakSosyal},
		{"https://www.reddit.com/r/bitcoin", kaynakSosyal},
		{"https://baska-bir-site.example/makale", kaynakDis},
		{"çöp", kaynakDis},
	}
	for _, o := range ornekler {
		if got := olayKaynagi(o.ref, host); got != o.bekle {
			t.Errorf("olayKaynagi(%q) = %q, beklenen %q", o.ref, got, o.bekle)
		}
	}
}

func TestSorguTemizle(t *testing.T) {
	// Jeton değeri ASLA saklanmamalı.
	got := sorguTemizle("token=abcdef123&sayfa=2")
	if strings.Contains(got, "abcdef123") {
		t.Fatalf("jeton maskelenmedi: %q", got)
	}
	if !strings.Contains(got, "token=***") || !strings.Contains(got, "sayfa=2") {
		t.Fatalf("beklenen biçim değil: %q", got)
	}
	// Parola sıfırlama bağlantısı tipik biçimi.
	got = sorguTemizle("kod=9f8e7d6c5b&eposta=ali@example.com")
	if strings.Contains(got, "9f8e7d6c5b") || strings.Contains(got, "ali@example.com") {
		t.Fatalf("hassas değer sızdı: %q", got)
	}
	// Uzun serbest değer kırpılır (jeton görünümlü, adı hassas değil).
	uzun := strings.Repeat("A", 60)
	got = sorguTemizle("q=" + uzun)
	if len(got) > 60 || !strings.Contains(got, "…") {
		t.Fatalf("uzun değer kırpılmadı: %q", got)
	}
	// Toplam uzunluk sınırı.
	got = sorguTemizle(strings.Repeat("a=1&", 200))
	if len(got) > 210 {
		t.Fatalf("sorgu kırpılmadı: %d karakter", len(got))
	}
	// Boş ve soru işaretli girdi.
	if sorguTemizle("") != "" || sorguTemizle("?") != "" {
		t.Fatal("boş sorgu boş dönmeli")
	}
}

func TestYolTemizle(t *testing.T) {
	// Parola sıfırlama jetonu yol parçası olarak: maskelenmeli.
	jeton := "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6"
	got := yolTemizle("/sifre-sifirla/" + jeton)
	if strings.Contains(got, jeton) {
		t.Fatalf("yol jetonu maskelenmedi: %q", got)
	}
	if !strings.Contains(got, "***") {
		t.Fatalf("maske işareti yok: %q", got)
	}
	// Kısa ve anlamlı parçalar korunur.
	if got := yolTemizle("/haber/bitcoin-ethereum-analiz"); got != "/haber/bitcoin-ethereum-analiz" {
		t.Errorf("normal yol değiştirildi: %q", got)
	}
	if got := yolTemizle(""); got != "/" {
		t.Errorf("boş yol / olmalı: %q", got)
	}
	// Uzun ama rakamsız (slug) parça maskelenmez: yalnız jeton görünümlü.
	slug := strings.Repeat("uzun-slug-", 4)
	if got := yolTemizle("/" + slug); strings.Contains(got, "***") {
		t.Errorf("rakamsız uzun slug maskelendi: %q", got)
	}
}

func TestNiyetCikar(t *testing.T) {
	// 1) Zafiyet her zaman tehdit — tek istek yeter.
	n := NiyetCikar(map[string]int{turZafiyet: 1, turIcerik: 40}, 41, 0)
	if n.Baslik != "Zafiyet taraması" || n.Seviye != "tehdit" {
		t.Fatalf("zafiyet tehdit olmalı: %+v", n)
	}

	// 2) Tuzak: zafiyetten sonra, tuzak da tehdit.
	n = NiyetCikar(map[string]int{turTuzak: 2, turIcerik: 30}, 32, 0)
	if n.Baslik != "Bot tuzağına düştü" || n.Seviye != "tehdit" {
		t.Fatalf("tuzak tehdit olmalı: %+v", n)
	}

	// 3) Kimlik denemesi: 3+ giriş VE toplamın ≥%20'si.
	n = NiyetCikar(map[string]int{turGiris: 5, turIcerik: 5}, 10, 0)
	if n.Baslik != "Kimlik denemesi" || n.Seviye != "tehdit" {
		t.Fatalf("giriş yoğunluğu yakalanmadı: %+v", n)
	}
	// 2 giriş 100 içerik arasında: kimlik denemesi DENMEZ.
	n = NiyetCikar(map[string]int{turGiris: 2, turIcerik: 100}, 102, 0)
	if n.Baslik == "Kimlik denemesi" {
		t.Fatalf("tek tük giriş kimlik denemesi sayılmamalı: %+v", n)
	}

	// 4) Besleme toplama.
	n = NiyetCikar(map[string]int{turBesleme: 10, turIcerik: 90}, 100, 0)
	if n.Baslik != "Besleme / harita toplama" || n.Seviye != "dikkat" {
		t.Fatalf("besleme yakalanmadı: %+v", n)
	}

	// 5) İçerik kazıma: hacim + pay eşiği; seviye "dikkat", "tehdit" DEĞİL.
	n = NiyetCikar(map[string]int{turIcerik: 40, turArama: 1}, 41, 0)
	if n.Baslik != "İçerik kazıma (olası)" || n.Seviye != "dikkat" {
		t.Fatalf("içerik kazıma yakalanmadı: %+v", n)
	}
	// 3 içerik: hacim eşiğinin altında → olağan gezinme.
	n = NiyetCikar(map[string]int{turIcerik: 3}, 3, 0)
	if n.Baslik != "Olağan gezinme" || n.Seviye != "bilgi" {
		t.Fatalf("az istek olağan sayılmalı: %+v", n)
	}

	// 6) Kayıt yok.
	n = NiyetCikar(nil, 0, 0)
	if n.Baslik != "Kayıt yok" {
		t.Fatalf("boş sayım 'Kayıt yok' dönmeli: %+v", n)
	}

	// 7) Engellenen sayısı açıklamaya girer.
	n = NiyetCikar(map[string]int{turIcerik: 3}, 3, 2)
	if !strings.Contains(n.Aciklama, "2 istek engellendi") {
		t.Fatalf("engellenen sayısı açıklamada yok: %q", n.Aciklama)
	}
}

func TestUAOzet(t *testing.T) {
	ornekler := []struct{ ua, bekle string }{
		{"", "boş"},
		{"curl/8.4.0", "curl"},
		{"python-requests/2.31.0", "python-requests"},
		{"Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)", "Googlebot"},
		{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/120.0 Safari/537.36", "Chrome · Windows"},
		{"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 Version/17.0 Safari/605.1.15", "Safari · macOS"},
		{"Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 Version/17.0 Mobile/15E148 Safari/604.1", "Safari · iOS"},
		{"Mozilla/5.0 (X11; Linux x86_64) Gecko/20100101 Firefox/121.0", "Firefox · Linux"},
		{"Mozilla/5.0 (Windows NT 10.0) AppleWebKit/537.36 Chrome/120.0 Safari/537.36 Edg/120.0", "Edge · Windows"},
		// HeadlessChrome, Chrome'dan ÖNCE yakalanmalı.
		{"Mozilla/5.0 HeadlessChrome/120.0", "HeadlessChrome"},
	}
	for _, o := range ornekler {
		if got := uaOzet(o.ua); got != o.bekle {
			t.Errorf("uaOzet(%q) = %q, beklenen %q", o.ua, got, o.bekle)
		}
	}
	// Chrome UA'sı "Safari/" içerir; naif sıralama Safari derdi.
	if got := uaOzet("Mozilla/5.0 Chrome/120.0 Safari/537.36"); strings.HasPrefix(got, "Safari") {
		t.Errorf("Chrome Safari sanıldı: %q", got)
	}
}

// repoOlay test için tek satır üretir (yalnız anahtar anlamlı).
func repoOlay(anahtar string) repo.ZiyaretOlay {
	return repo.ZiyaretOlay{Anahtar: anahtar, Yol: "/", Tur: turIcerik}
}

func TestOlayKuyrukDusurme(t *testing.T) {
	// Kuyruk dolduğunda olayYaz BLOKLAMAMALI; düşen sayılır.
	// Kanal global: testte doldurup geri boşaltıyoruz.
	k := olaylariBaslat()
	for len(k) > 0 {
		<-k
	}
	basla := olayDusen()
	for i := 0; i < olayKuyrukBoyu; i++ {
		olayYaz(repoOlay("a"))
	}
	if len(k) != olayKuyrukBoyu {
		t.Fatalf("kuyruk dolmadı: %d", len(k))
	}
	olayYaz(repoOlay("a")) // bu düşmeli
	if olayDusen() != basla+1 {
		t.Fatalf("düşen olay sayılmadı: %d", olayDusen()-basla)
	}
	// Anahtarsız olay hiç kuyruğa girmez.
	once := len(k)
	olayYaz(repoOlay(""))
	if len(k) != once {
		t.Fatal("anahtarsız olay kuyruğa girdi")
	}
	for len(k) > 0 {
		<-k
	}
}
