package httpapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"io/fs"
)

// TestCSSYorumlariSil temel davranış: yorum gider, kural kalır.
func TestCSSYorumlariSil(t *testing.T) {
	girdi := `/* başlık */
.cd-x { color: red; } /* satır sonu */
/* çok
   satırlı */
.cd-y {
  /* iç */
  margin: 0;
}
`
	got := string(cssYorumlariSil([]byte(girdi)))
	if strings.Contains(got, "/*") || strings.Contains(got, "*/") {
		t.Fatalf("yorum kaldı:\n%s", got)
	}
	for _, beklenen := range []string{".cd-x { color: red; }", ".cd-y {", "margin: 0;"} {
		if !strings.Contains(got, beklenen) {
			t.Fatalf("%q kayboldu:\n%s", beklenen, got)
		}
	}
}

// TestCSSYorumlariSilTuzaklar: yorum benzeri diziler string/url içinde
// BOZULMAMALI. Naif uygulamalar tam burada patlar.
func TestCSSYorumlariSilTuzaklar(t *testing.T) {
	haller := []struct{ ad, girdi, beklenen string }{
		{
			ad:       "content içinde yorum dizisi",
			girdi:    `.a::before { content: "/* bu yorum değil */"; }`,
			beklenen: `content: "/* bu yorum değil */"`,
		},
		{
			ad:       "tek tırnak",
			girdi:    `.b { content: '/* x */'; }`,
			beklenen: `content: '/* x */'`,
		},
		{
			ad:       "url() içinde çift eğik",
			girdi:    `.c { background: url("a/*b.png"); }`,
			beklenen: `url("a/*b.png")`,
		},
		{
			ad:       "tırnaksız url",
			girdi:    `.d { background: url(a/*b.png); }`,
			beklenen: `url(a/*b.png)`,
		},
		{
			ad:       "data URI içinde",
			girdi:    `.e { background: url(data:image/svg+xml;utf8,<svg>/*</svg>); }`,
			beklenen: `data:image/svg+xml;utf8,<svg>/*</svg>`,
		},
		{
			ad:       "kaçışlı tırnak",
			girdi:    `.f { content: "a\"/*b*/"; }`,
			beklenen: `content: "a\"/*b*/"`,
		},
		{
			ad:    "gerçek yorum araya giriyor",
			girdi: `.g/* x */{ color: blue; }`,
			// Boşluk EKLENMEZ: `{` ayırıcı olduğu için `.g{ ... }` yeterli.
			beklenen: `.g{ color: blue; }`,
		},
		{
			ad:    "yorum iki sözcüğü birleştirmemeli",
			girdi: `.a/* x */b { color: red; }`,
			// İki taraf da sözcük karakteri: araya boşluk ŞART, aksi hâlde
			// `.ab` olur ve seçici ANLAM değiştirir.
			beklenen: `.a b { color: red; }`,
		},
	}
	for _, h := range haller {
		got := string(cssYorumlariSil([]byte(h.girdi)))
		if !strings.Contains(got, h.beklenen) {
			t.Errorf("%s: %q kayboldu, gelen:\n%s", h.ad, h.beklenen, got)
		}
	}
}

// TestCSSYorumlariSilKapanmamisYorum: yarısı yorum olan geçersiz CSS
// sunulmasın (kalan atılır).
func TestCSSYorumlariSilKapanmamisYorum(t *testing.T) {
	got := string(cssYorumlariSil([]byte(".a{color:red}\n/* kapanmadı .b{color:blue}")))
	if strings.Contains(got, ".b{color:blue}") {
		t.Fatalf("kapanmamış yorumun içi sunuldu:\n%s", got)
	}
	if !strings.Contains(got, ".a{color:red}") {
		t.Fatalf("yorumdan önceki kural kayboldu:\n%s", got)
	}
}

// TestCSSYorumlariSilBosSatir: yorum silinince kalan boş satır yığını
// temizlenir, ama satır içi boşluk korunur.
func TestCSSYorumlariSilBosSatir(t *testing.T) {
	got := string(cssYorumlariSil([]byte(".a {\n/* 1 */\n/* 2 */\n/* 3 */\n  color: red;\n}\n")))
	if strings.Contains(strings.ReplaceAll(got, "color", ""), "\n\n\n") {
		t.Fatalf("ardışık boş satırlar temizlenmedi:\n%q", got)
	}
	if !strings.Contains(got, "  color: red;") {
		t.Fatalf("satır içi girinti bozuldu:\n%q", got)
	}
}

// TestCSSSunucuYorumsuzVeETag: HTTP katmanı — gövdede yorum yok, ETag var,
// ikinci istek 304.
func TestCSSSunucuYorumsuzVeETag(t *testing.T) {
	fsys := fstest.MapFS{
		"cd.css": &fstest.MapFile{Data: []byte("/* iç not */\n.a { color: red; }\n")},
		"x.png":  &fstest.MapFile{Data: []byte{0x89, 'P', 'N', 'G'}},
	}
	s := newCSSSunucu(fsys)

	ist := func(basliklar map[string]string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodGet, "/cd.css", nil)
		for k, v := range basliklar {
			r.Header.Set(k, v)
		}
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		return w
	}

	w := ist(nil)
	if w.Code != http.StatusOK {
		t.Fatalf("kod %d", w.Code)
	}
	if strings.Contains(w.Body.String(), "iç not") {
		t.Fatalf("yorum dışarı sızdı:\n%s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), ".a { color: red; }") {
		t.Fatalf("kural kayboldu:\n%s", w.Body.String())
	}
	etag := w.Header().Get("ETag")
	if etag == "" {
		t.Fatal("ETag yok")
	}
	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/css") {
		t.Fatalf("Content-Type %q", ct)
	}
	if w.Header().Get("Content-Length") != itoa(w.Body.Len()) {
		t.Fatalf("Content-Length yanlış: %q vs %d", w.Header().Get("Content-Length"), w.Body.Len())
	}

	// Aynı ETag ile ikinci istek: 304 ve gövde yok.
	w2 := ist(map[string]string{"If-None-Match": etag})
	if w2.Code != http.StatusNotModified {
		t.Fatalf("304 beklenirdi, gelen %d", w2.Code)
	}
	if w2.Body.Len() != 0 {
		t.Fatalf("304 gövdesi boş olmalı, %d bayt", w2.Body.Len())
	}

	// CSS olmayan dosya eski davranışta (dosya olarak) sunulur.
	r := httptest.NewRequest(http.MethodGet, "/x.png", nil)
	w3 := httptest.NewRecorder()
	s.ServeHTTP(w3, r)
	if w3.Code != http.StatusOK || !bytes.HasPrefix(w3.Body.Bytes(), []byte{0x89, 'P', 'N', 'G'}) {
		t.Fatalf("png sunulmadı: %d %q", w3.Code, w3.Body.Bytes())
	}
}

// TestCSSSunucuOnbellek: aynı dosya ikinci kez okununca FS'e gidilmez
// (dönüşüm süreç başına bir kez).
func TestCSSSunucuOnbellek(t *testing.T) {
	sayac := 0
	fsys := sayanFS{ic: fstest.MapFS{
		"a.css": &fstest.MapFile{Data: []byte("/* x */ .a{color:red}")},
	}, sayac: &sayac}
	s := newCSSSunucu(fsys)
	for i := 0; i < 3; i++ {
		r := httptest.NewRequest(http.MethodGet, "/a.css", nil)
		s.ServeHTTP(httptest.NewRecorder(), r)
	}
	if sayac != 1 {
		t.Fatalf("FS %d kez okundu, 1 beklenirdi", sayac)
	}
}

type sayanFS struct {
	ic    fstest.MapFS
	sayac *int
}

func (f sayanFS) Open(name string) (fs.File, error) {
	*f.sayac++
	return f.ic.Open(name)
}

// ReadFile AÇIKÇA sayılır: fstest.MapFS ReadFileFS uyguluyor ve fs.ReadFile
// onu tercih ediyor; yalnız Open'ı saymak testi yanlış yeşil gösterirdi.
func (f sayanFS) ReadFile(name string) ([]byte, error) {
	*f.sayac++
	return f.ic.ReadFile(name)
}
