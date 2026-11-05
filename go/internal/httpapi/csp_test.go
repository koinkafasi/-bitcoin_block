package httpapi

import (
	"crypto/sha256"
	"encoding/base64"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// hashOf testte beklenen CSP hash'ini üretir.
func hashOf(icerik string) string {
	sum := sha256.Sum256([]byte(icerik))
	return "'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) + "'"
}

// TestCSPScriptHashes satır içi script'lerin hash'lenmesi.
func TestCSPScriptHashes(t *testing.T) {
	html := `<html><head>` +
		`<script>var a = 1;</script>` +
		`<script type="application/json">{"x":1}</script>` +
		`<script type="application/ld+json">{"@type":"NewsArticle"}</script>` +
		`<script src="/static/app.js" defer></script>` +
		`<script>var b = 2;</script>` +
		`</head></html>`

	got := cspScriptHashes(html)
	if len(got) != 2 {
		t.Fatalf("2 hash beklenirdi (veri blokları ve src'li atlanır), alınan %d: %v", len(got), got)
	}
	want1 := hashOf("var a = 1;")
	want2 := hashOf("var b = 2;")
	if got[0] != want1 {
		t.Errorf("1. hash yanlış:\n alınan %s\n beklenen %s", got[0], want1)
	}
	if got[1] != want2 {
		t.Errorf("2. hash yanlış:\n alınan %s\n beklenen %s", got[1], want2)
	}
}

// TestCSPScriptHashes_Tekrarlanan aynı içerikli script'ler tek hash üretir.
func TestCSPScriptHashes_Tekrarlanan(t *testing.T) {
	html := `<script>ayni()</script><p>x</p><script>ayni()</script>`
	if got := cspScriptHashes(html); len(got) != 1 {
		t.Fatalf("tek hash beklenirdi, alınan %d: %v", len(got), got)
	}
}

// TestCSPScriptHashes_SatirSonu tarayıcı CRLF'i LF'e indirger; hash eşleşmeli.
func TestCSPScriptHashes_SatirSonu(t *testing.T) {
	crlf := "<script>a();\r\nb();</script>"
	lf := "<script>a();\nb();</script>"
	h1 := cspScriptHashes(crlf)
	h2 := cspScriptHashes(lf)
	if len(h1) != 1 || len(h2) != 1 {
		t.Fatalf("hash üretilemedi: %v / %v", h1, h2)
	}
	if h1[0] != h2[0] {
		t.Errorf("CRLF ve LF aynı hash'i vermeli:\n %s\n %s", h1[0], h2[0])
	}
}

// TestCSPScriptHashes_Bos bos/beyaz script'ler hash üretmez.
func TestCSPScriptHashes_Bos(t *testing.T) {
	for _, in := range []string{"", "<div></div>", "<script></script>", "<script>   \n </script>"} {
		if got := cspScriptHashes(in); len(got) != 0 {
			t.Errorf("%q için hash beklenmezdi, alınan %v", in, got)
		}
	}
}

// TestCSPScriptHashes_BuyukHarf kapanış etiketi büyük harfle de bulunmalı.
func TestCSPScriptHashes_BuyukHarf(t *testing.T) {
	got := cspScriptHashes("<script>x()</SCRIPT>")
	if len(got) != 1 || got[0] != hashOf("x()") {
		t.Fatalf("büyük harfli kapanış tanınmadı: %v", got)
	}
}

// TestBuildCSP zorunlu politikanın güvenlik özellikleri.
// TestCSPPolitikadaYasakliTasima, üretilen politikanın TAMAMINDA
// `'unsafe-eval'` ve script tarafında `'unsafe-inline'` bulunmadığını
// doğrular — politika varyantlarının hepsinde.
//
// NEDEN AYRI TEST: CI'daki kapı bunu `grep csp.go` ile arıyordu ve
// csp.go'daki bir YORUM satırını ("// 'unsafe-inline' ve 'unsafe-eval'
// YOK.") yakalayıp işi kırdı (5 Kasım 2026). Kaynak metni değil, ÜRETİLEN
// politikayı sınamak doğrusu: yorum değişse de test anlamlı kalır.
func TestCSPPolitikadaYasakliTasima(t *testing.T) {
	h := hashOf("var a = 1;")
	varyantlar := map[string]string{
		"boş":             buildCSP(nil, nil, "", ""),
		"hash'li":         buildCSP([]string{h}, []string{h}, "", ""),
		"collab+rapor":    buildCSP([]string{h}, []string{h}, "wss://collab.example.com/ws", "https://theblock.com.tr/csp-rapor"),
		"çok hash":        buildCSP([]string{h, hashOf("var b = 2;")}, []string{h}, "", ""),
		"geçersiz collab": buildCSP(nil, nil, "://bozuk", ""),
	}
	for ad, csp := range varyantlar {
		if strings.Contains(csp, "'unsafe-eval'") {
			t.Errorf("%s: 'unsafe-eval' politikada olmamalı: %s", ad, csp)
		}
		// 'unsafe-inline' YALNIZ style-src-attr'de olabilir (öznitelikler
		// hash'lenemez). script-src ve style-src'te bulunmamalı.
		for _, direktif := range strings.Split(csp, ";") {
			d := strings.TrimSpace(direktif)
			if strings.HasPrefix(d, "style-src-attr") {
				continue
			}
			if strings.Contains(d, "'unsafe-inline'") {
				t.Errorf("%s: 'unsafe-inline' yalnız style-src-attr'de olmalı, bulundu: %q", ad, d)
			}
		}
	}
}

func TestBuildCSP(t *testing.T) {
	h := hashOf("var a = 1;")
	csp := buildCSP([]string{h}, nil, "", "")

	for _, gerekli := range []string{
		"default-src 'self'",
		"script-src 'self' " + h,
		"object-src 'none'",
		"base-uri 'self'",
		"frame-ancestors 'self'",
		"form-action 'self'",
	} {
		if !strings.Contains(csp, gerekli) {
			t.Errorf("%q eksik: %s", gerekli, csp)
		}
	}
	for _, yasak := range []string{"'unsafe-eval'", "script-src 'self' 'unsafe-inline'", "nonce-"} {
		if strings.Contains(csp, yasak) {
			t.Errorf("%q politikada olmamalı: %s", yasak, csp)
		}
	}

	// Hash yoksa script-src yalnız 'self' + izinli dış kaynaklar.
	bos := buildCSP(nil, nil, "", "")
	if strings.Contains(bos, "'unsafe-inline'; https") || strings.Contains(bos, "script-src 'self' 'unsafe-inline'") {
		t.Errorf("hash yokken unsafe-inline'a düşülmemeli: %s", bos)
	}
	if !strings.Contains(bos, "script-src 'self' https://www.googletagmanager.com") {
		t.Errorf("hash yokken script-src 'self' ile başlamalı: %s", bos)
	}
}

// TestCSPOnbellekGuvenli CSP başlığı gövdeden ÜRETİLMELİ ve HTML'de nonce
// bulunmamalı: public sayfalar CDN'de önbelleklendiği için nonce tekilliği
// bozulurdu (bkz. csp.go başlığındaki gerekçe).
func TestCSPOnbellekGuvenli(t *testing.T) {
	html := `<script>var a=1;</script>`
	csp := buildCSP(cspScriptHashes(html), cspStyleHashes(html), "", "")
	if strings.Contains(csp, "nonce-") {
		t.Errorf("önbelleklenen yanıtlarda nonce KULLANILMAMALI: %s", csp)
	}
	if !strings.Contains(csp, hashOf("var a=1;")) {
		t.Errorf("satır içi script hash'i politikada olmalı: %s", csp)
	}
	// Gövde değişmez: nonce eklenmez.
	if strings.Contains(html, "nonce") {
		t.Error("HTML gövdesine nonce eklenmemeli (önbellek)")
	}
}

// TestCSPScriptHashes_TurkceMetin Türkçe karakterler (İ, ğ, ş) hash
// çıkarımını kaydırmamalı. Regresyon: strings.ToLower ile arama yapıldığında
// 'İ' bayt uzunluğunu değiştiriyor, indeksler kayıyor ve hash'ler gövdeyle
// uyuşmuyordu (tarayıcıda CSP ihlali).
func TestCSPScriptHashes_TurkceMetin(t *testing.T) {
	html := `<h1>İSTANBUL'DA GÜNDEM: ŞİDDET VE İKLİM</h1>` +
		`<p>ĞÜŞİÖÇ ğüşiöç</p>` +
		`<script>var x = "İstanbul";</script>`
	got := cspScriptHashes(html)
	if len(got) != 1 {
		t.Fatalf("1 hash beklenirdi, alınan %d: %v", len(got), got)
	}
	if want := hashOf(`var x = "İstanbul";`); got[0] != want {
		t.Fatalf("Türkçe metinden sonra hash kaydı:\n alınan %s\n beklenen %s", got[0], want)
	}
}

// TestIndexFold_BoyutKoruyucu indexFold ASCII katlaması yapar ve indeks
// orijinal metne göre doğru kalır.
func TestIndexFold_BoyutKoruyucu(t *testing.T) {
	s := "İĞÜŞ<script>"
	if got := indexFold(s, "<SCRIPT"); got != len("İĞÜŞ") {
		t.Fatalf("indexFold = %d, beklenen %d (Türkçe önek kaydırmamalı)", got, len("İĞÜŞ"))
	}
	if got := indexFold("abc", "xyz"); got != -1 {
		t.Fatalf("bulunmamalıydı, alınan %d", got)
	}
}

// TestSablonlardaSatirIciOlayYok: zorunlu CSP satır içi olay özniteliklerini
// çalıştırmaz. Şablonlara yeniden on*= eklenirse site sessizce bozulur —
// bu test onu derleme/test aşamasında yakalar.
func TestSablonlardaSatirIciOlayYok(t *testing.T) {
	kok := filepath.Join("..", "webui")
	desen := regexp.MustCompile(`\son[a-z]+\s*=\s*["']`)
	var ihlaller []string
	for _, alt := range []string{"templates", "templkit"} {
		dir := filepath.Join(kok, alt)
		err := filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			ad := info.Name()
			if !strings.HasSuffix(ad, ".html") && !strings.HasSuffix(ad, ".templ") {
				return nil
			}
			b, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			for i, satir := range strings.Split(string(b), "\n") {
				if desen.MatchString(satir) {
					kisa := strings.TrimSpace(satir)
					if len(kisa) > 120 {
						kisa = kisa[:120]
					}
					ihlaller = append(ihlaller, p+":"+strconv.Itoa(i+1)+": "+kisa)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("%s taranamadı: %v", dir, err)
		}
	}
	if len(ihlaller) > 0 {
		t.Errorf("satır içi olay özniteliği bulundu (CSP bunları bloklar, data-* kullanın):\n%s",
			strings.Join(ihlaller, "\n"))
	}
}
