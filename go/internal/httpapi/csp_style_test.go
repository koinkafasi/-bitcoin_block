package httpapi

import (
	"strings"
	"testing"
)

func TestCSPStyleHashesBos(t *testing.T) {
	for _, html := range []string{"", "<p>yok</p>", "<script>x</script>"} {
		if got := cspStyleHashes(html); len(got) != 0 {
			t.Errorf("cspStyleHashes(%q) = %v; boş bekleniyordu", html, got)
		}
	}
}

func TestCSPStyleHashesTek(t *testing.T) {
	html := `<style>body{color:red}</style>`
	got := cspStyleHashes(html)
	if len(got) != 1 {
		t.Fatalf("%d hash; 1 bekleniyordu: %v", len(got), got)
	}
	if !strings.HasPrefix(got[0], "'sha256-") || !strings.HasSuffix(got[0], "'") {
		t.Errorf("CSP biçimi yanlış: %s", got[0])
	}
	// Aynı içerik → aynı hash (deterministik; önbellek güvenli).
	if ikinci := cspStyleHashes(html); len(ikinci) != 1 || ikinci[0] != got[0] {
		t.Errorf("deterministik değil: %v vs %v", got, ikinci)
	}
}

// Aynı içerikli iki blok tek hash üretir; farklı içerik iki hash.
func TestCSPStyleHashesTekilleştirir(t *testing.T) {
	ayni := `<style>a{}</style><style>a{}</style>`
	if got := cspStyleHashes(ayni); len(got) != 1 {
		t.Errorf("aynı bloklar %d hash üretti; 1 bekleniyordu: %v", len(got), got)
	}
	farkli := `<style>a{}</style><style>b{}</style>`
	if got := cspStyleHashes(farkli); len(got) != 2 {
		t.Errorf("farklı bloklar %d hash üretti; 2 bekleniyordu: %v", len(got), got)
	}
}

// Boş/yalnız boşluk blok hash üretmemeli (tarayıcı da yok sayar).
func TestCSPStyleHashesBosBlok(t *testing.T) {
	html := "<style>   \n\t </style><style>a{}</style>"
	if got := cspStyleHashes(html); len(got) != 1 {
		t.Errorf("%d hash; yalnız dolu blok sayılmalı: %v", len(got), got)
	}
}

// CRLF ve LF aynı hash'i vermeli: tarayıcı satır sonlarını LF'e indirger,
// aksi hâlde Windows'ta üretilen yanıt kendi hash'iyle eşleşmezdi.
func TestCSPStyleHashesSatirSonu(t *testing.T) {
	lf := cspStyleHashes("<style>a{}\nb{}</style>")
	crlf := cspStyleHashes("<style>a{}\r\nb{}</style>")
	cr := cspStyleHashes("<style>a{}\rb{}</style>")
	if len(lf) != 1 || len(crlf) != 1 || len(cr) != 1 {
		t.Fatalf("beklenmeyen hash sayısı: lf=%v crlf=%v cr=%v", lf, crlf, cr)
	}
	if lf[0] != crlf[0] || lf[0] != cr[0] {
		t.Errorf("satır sonu hash'i değiştiriyor:\n lf=%s\n crlf=%s\n cr=%s", lf[0], crlf[0], cr[0])
	}
}

// Büyük harfli etiket de tanınmalı (HTML etiket adları duyarsızdır).
func TestCSPStyleHashesBuyukHarf(t *testing.T) {
	if got := cspStyleHashes(`<STYLE>a{}</STYLE>`); len(got) != 1 {
		t.Errorf("<STYLE> tanınmadı: %v", got)
	}
}

// style özniteliği olan bir etiket <style> sanılmamalı.
func TestCSPStyleHashesOznitelikKarismaz(t *testing.T) {
	html := `<div style="color:red">x</div>`
	if got := cspStyleHashes(html); len(got) != 0 {
		t.Errorf("style= özniteliği hash üretti: %v", got)
	}
}

// --- politika düzeyi ---

// style-src'de 'unsafe-inline' BULUNMAMALI; style-src-attr'de bilinçli olarak
// bulunmalı. Bu ayrım Faz 211'in tüm amacıdır.
func TestBuildCSPStyleAyrimi(t *testing.T) {
	csp := buildCSP(nil, nil, "", "")

	styleSrc := direktif(csp, "style-src ")
	if styleSrc == "" {
		t.Fatalf("style-src direktifi yok: %s", csp)
	}
	if strings.Contains(styleSrc, "'unsafe-inline'") {
		t.Errorf("style-src 'unsafe-inline' içermemeli: %s", styleSrc)
	}
	if !strings.Contains(styleSrc, "'self'") {
		t.Errorf("style-src 'self' içermeli: %s", styleSrc)
	}

	attr := direktif(csp, "style-src-attr ")
	if attr == "" {
		t.Fatalf("style-src-attr direktifi yok (öznitelikler için şart): %s", csp)
	}
	if !strings.Contains(attr, "'unsafe-inline'") {
		t.Errorf("style-src-attr 'unsafe-inline' içermeli (öznitelikler): %s", attr)
	}
}

// Sayfadaki <style> hash'i politikaya girmeli.
func TestBuildCSPStyleHashPolitikada(t *testing.T) {
	html := `<style>body{--x:1}</style>`
	csp := buildCSP(nil, cspStyleHashes(html), "", "")
	styleSrc := direktif(csp, "style-src ")
	if !strings.Contains(styleSrc, "'sha256-") {
		t.Errorf("style-src hash içermeli: %s", styleSrc)
	}
}

// style-src-attr satır içi style'ı ETKİLEMEZ: iki direktif ayrı kalmalı,
// yoksa öznitelikler sessizce ölür (panel görünümü bozulur).
func TestBuildCSPStyleAttrAyri(t *testing.T) {
	csp := buildCSP(nil, nil, "", "")
	a := direktif(csp, "style-src ")
	b := direktif(csp, "style-src-attr ")
	if a == b {
		t.Fatal("style-src ve style-src-attr aynı olmamalı")
	}
	// style-src'te hash olsa bile attr'de olmamalı (attr hash kabul etmez).
	withHash := buildCSP(nil, cspStyleHashes("<style>a{}</style>"), "", "")
	if strings.Contains(direktif(withHash, "style-src-attr "), "sha256") {
		t.Error("style-src-attr hash içermemeli (geçersiz direktif olurdu)")
	}
}

// direktif, CSP dizesinden adı verilen direktifin değerini çeker.
func direktif(csp, ad string) string {
	for _, p := range strings.Split(csp, ";") {
		p = strings.TrimSpace(p)
		if strings.HasPrefix(p, ad) {
			return p
		}
	}
	return ""
}
