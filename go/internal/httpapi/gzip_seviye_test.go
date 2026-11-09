package httpapi

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// gzipCoz yardımcı: yanıt gövdesini çözer.
func gzipCoz(t *testing.T, b []byte) string {
	t.Helper()
	zr, err := gzip.NewReader(bytes.NewReader(b))
	if err != nil {
		t.Fatalf("gzip reader: %v", err)
	}
	defer zr.Close()
	out, err := io.ReadAll(zr)
	if err != nil {
		t.Fatalf("gzip read: %v", err)
	}
	return string(out)
}

// testGovdeCesitli sıkıştırma seviyeleri arasında fark yaratacak, tekrarsız
// ama gerçekçi bir HTML gövdesi üretir (deterministik).
func testGovdeCesitli(n int) string {
	var b strings.Builder
	kelimeler := []string{"bitcoin", "ethereum", "piyasa", "analiz", "hacim",
		"kurumsal", "talep", "borsa", "düzenleme", "rapor", "fiyat", "yatırım"}
	for i := 0; i < n; i++ {
		b.WriteString(`<article class="kart"><h2>`)
		b.WriteString(kelimeler[(i*7)%len(kelimeler)])
		b.WriteString(` `)
		b.WriteString(kelimeler[(i*13+3)%len(kelimeler)])
		b.WriteString(`</h2><p>`)
		// i'ye bağlı sayılar tekrarı kırar, sıkıştırma seviyeleri ayrışır.
		b.WriteString(strings.Repeat(kelimeler[(i*5)%len(kelimeler)]+" ", 3+i%11))
		b.WriteString(`</p></article>`)
	}
	return b.String()
}

// TestGzipSeviyeAyaruUygulanir: SetGzipLevel gerçekten sıkıştırmayı etkiler
// (yanıt çözülebilir olmalı) ve aralık dışı değer yok sayılır.
func TestGzipSeviyeAyaruUygulanir(t *testing.T) {
	eski := GzipLevel()
	defer SetGzipLevel(eski)

	govde := strings.Repeat("<p>bitcosar kripto haber</p>", 200)
	ust := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(govde))
	})
	s := &Server{}

	for _, lvl := range []int{1, 4, 9} {
		SetGzipLevel(lvl)
		if GzipLevel() != lvl {
			t.Fatalf("seviye %d ayarlanmadı (okunan %d)", lvl, GzipLevel())
		}
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("Accept-Encoding", "gzip")
		s.withGzip(ust).ServeHTTP(w, r)

		if enc := w.Header().Get("Content-Encoding"); enc != "gzip" {
			t.Fatalf("seviye %d: Content-Encoding = %q", lvl, enc)
		}
		if got := gzipCoz(t, w.Body.Bytes()); got != govde {
			t.Fatalf("seviye %d: gövde bozuldu (%d bayt)", lvl, len(got))
		}
	}

	// Aralık dışı yok sayılır.
	SetGzipLevel(4)
	SetGzipLevel(0)
	SetGzipLevel(10)
	if GzipLevel() != 4 {
		t.Fatalf("aralık dışı değer kabul edildi: %d", GzipLevel())
	}
}

// TestGzipSeviyeHavuzuKirletmez: seviye değişince havuzdan yanlış seviyeli
// writer alınmamalı. İki seviyeyi art arda kullanıp her yanıtı çözüyoruz.
func TestGzipSeviyeHavuzuKirletmez(t *testing.T) {
	eski := GzipLevel()
	defer SetGzipLevel(eski)

	govde := testGovdeCesitli(500)
	ust := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(govde))
	})
	s := &Server{}

	boyut := map[int]int{}
	for _, lvl := range []int{1, 9, 1, 9, 4} {
		SetGzipLevel(lvl)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("Accept-Encoding", "gzip")
		s.withGzip(ust).ServeHTTP(w, r)
		if got := gzipCoz(t, w.Body.Bytes()); got != govde {
			t.Fatalf("seviye %d: gövde bozuldu", lvl)
		}
		boyut[lvl] = w.Body.Len()
	}
	// Seviye 9, seviye 1'den küçük çıktı üretmeli (yanlış writer alınsaydı
	// boyutlar eşitlenirdi). Gövde YETERİNCE ÇEŞİTLİ olmalı: aşırı tekrarlı
	// metinde her iki seviye de aynı tavana vurur ve test hiçbir şey sınamaz.
	if boyut[9] >= boyut[1] {
		t.Fatalf("seviye 9 (%d bayt) seviye 1'den (%d) küçük değil — havuz karışmış",
			boyut[9], boyut[1])
	}
}
