package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestCSPWriterDurumKodu CSP tamponlayıcısı handler'ın bildirdiği durum
// kodunu KAYBETMEMELİ. Regresyon: ilk uygulamada WriteHeader yalnız tampona
// yazılıyor, alttaki yazıcıya hiç geçmiyordu; sonuç: 204/302/404 yanıtları
// 200 olarak dönüyordu.
func TestCSPWriterDurumKodu(t *testing.T) {
	cases := []struct {
		ad      string
		handler http.HandlerFunc
		want    int
		govde   string
	}{
		{
			ad: "204 gövdesiz",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			},
			want: http.StatusNoContent,
		},
		{
			ad: "404 metinli",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte("yok"))
			},
			want:  http.StatusNotFound,
			govde: "yok",
		},
		{
			ad: "302 yönlendirme",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Location", "/admin/giris")
				w.WriteHeader(http.StatusFound)
			},
			want: http.StatusFound,
		},
		{
			ad: "200 HTML",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				_, _ = w.Write([]byte("<html><script>x()</script></html>"))
			},
			want:  http.StatusOK,
			govde: "<script>x()</script>",
		},
		{
			ad: "500 hata",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte("<html>hata</html>"))
			},
			want:  http.StatusInternalServerError,
			govde: "hata",
		},
	}

	for _, c := range cases {
		t.Run(c.ad, func(t *testing.T) {
			rec := httptest.NewRecorder()
			s := &Server{}
			h := s.withHTMLCSP(c.handler)
			h.ServeHTTP(rec, httptest.NewRequest("GET", "/test", nil))

			if rec.Code != c.want {
				t.Fatalf("durum kodu = %d, beklenen %d (CSP tamponu kodu kaybetti)", rec.Code, c.want)
			}
			if c.govde != "" && !strings.Contains(rec.Body.String(), c.govde) {
				t.Errorf("gövde beklenen parçayı içermiyor: %q", rec.Body.String())
			}
		})
	}
}

// TestCSPWriterHTMLHash CSP başlığı HTML yanıtta gövdeyle TUTARLI olmalı.
func TestCSPWriterHTMLHash(t *testing.T) {
	rec := httptest.NewRecorder()
	s := &Server{}
	html := `<html><script>var a=1;</script><script><\/script></html>`
	h := s.withHTMLCSP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(html))
	}))
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))

	csp := rec.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Fatal("CSP başlığı yok")
	}
	want := hashOf("var a=1;")
	if !strings.Contains(csp, want) {
		t.Errorf("gövdedeki script'in hash'i başlıkta yok\n hash: %s\n csp: %s", want, csp)
	}
	if strings.Contains(csp, "nonce-") {
		t.Error("önbellek güvenliği için nonce kullanılmamalı")
	}
}

// TestCSPWriterHTMLFlushSonrasiHash — REGRESYON (ölçüldü, 6 Kasım 2026).
//
// `templ` bileşeni bitirirken kendi tamponunu boşaltıp yazıcı
// `http.Flusher` ise `Flush()` çağırıyor (`runtime.ReleaseBuffer`).
// Eski `Flush`, HTML yanıtında bile "akışa geç" diyordu; `finish()` de
// akışta CSP üretmediği için yanıt HASH'SİZ politika alıyordu ve
// sayfadaki satır içi `<script>` sessizce engelleniyordu.
//
// Belirti görünmüyordu: sayfa 200 dönüyor, yalnız betik çalışmıyordu.
// Bu test, HTML tamponunun Flush'tan sonra da korunduğunu doğrular.
func TestCSPWriterHTMLFlushSonrasiHash(t *testing.T) {
	rec := httptest.NewRecorder()
	s := &Server{}
	h := s.withHTMLCSP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte("<html><body>"))
		// templ'in ReleaseBuffer davranışının aynısı:
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		_, _ = w.Write([]byte("<script>var tema=1;</script></body></html>"))
	}))
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))

	csp := rec.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, hashOf("var tema=1;")) {
		t.Fatalf("Flush sonrası satır içi script'in hash'i başlıkta yok\n csp: %s", csp)
	}
	if !strings.Contains(rec.Body.String(), "<script>var tema=1;</script>") {
		t.Fatal("gövde bozuldu: Flush'tan sonraki yazım kayboldu")
	}
}

// TestCSPWriterAkisFlush — SSE/HTML dışı akışlar eskisi gibi akmalı.
//
// HTML tamponu korunuyor diye bu yollar da tamponlanırsa zip/csv dışa
// aktarma ve SSE bozulur (bellek şişer, veri geç gelir).
func TestCSPWriterAkisFlush(t *testing.T) {
	rec := httptest.NewRecorder()
	ss := &Server{}
	h := ss.withHTMLCSP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		_, _ = w.Write([]byte("parca1"))
		if f, ok := w.(http.Flusher); ok {
			f.Flush() // hemen iletilmeli
		}
		_, _ = w.Write([]byte("parca2"))
	}))
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/disa-aktar", nil))
	if got := rec.Body.String(); got != "parca1parca2" {
		t.Errorf("akış gövdesi = %q", got)
	}
	if strings.Contains(rec.Header().Get("Content-Security-Policy"), "sha256-") {
		t.Error("zip yanıtına CSP hash'i eklenmemeli")
	}
}

// TestCSPYorumIcindekiEtiketSayilmaz — REGRESYON (ölçüldü, 6 Kasım 2026).
//
// HTML yorumu içinde "<script" kelimesi geçebilir (açıklama metni).
// Tarayıcı bunu etiket saymaz; sunucu sayarsa hash YANLIŞ olur ve sayfadaki
// satır içi betik sessizce engellenir. Ölçülen belirti: sunucu 478 baytı,
// tarayıcı 400 baytı hash'ledi → "Executing inline script violates…".
func TestCSPYorumIcindekiEtiketSayilmaz(t *testing.T) {
	html := `<!-- bu yorumda <script> kelimesi geçiyor -->` +
		`<html><script>var a=1;</script></html>`
	hashes := cspScriptHashes(html)
	if len(hashes) != 1 {
		t.Fatalf("1 hash beklenirdi, %d bulundu: %v", len(hashes), hashes)
	}
	if want := hashOf("var a=1;"); hashes[0] != want {
		t.Errorf("hash = %s, beklenen %s", hashes[0], want)
	}
}

// TestCSPYorumIcindekiStilSayilmaz — aynı sınıf, <style> için.
func TestCSPYorumIcindekiStilSayilmaz(t *testing.T) {
	html := `<!-- <style> ornegi --><html><style>.a{color:red}</style></html>`
	hashes := cspStyleHashes(html)
	if len(hashes) != 1 {
		t.Fatalf("1 hash beklenirdi, %d bulundu: %v", len(hashes), hashes)
	}
	if want := hashOf(".a{color:red}"); hashes[0] != want {
		t.Errorf("hash = %s, beklenen %s", hashes[0], want)
	}
}

// TestCSPYorumIcindekiScriptTamamenAtlanir — script yorumun İÇİNDEyse
// hiç hash üretilmemeli (tarayıcı da çalıştırmaz).
func TestCSPYorumIcindekiScriptTamamenAtlanir(t *testing.T) {
	html := `<html><!-- <script>var gizli=1;</script> --><p>metin</p></html>`
	if h := cspScriptHashes(html); len(h) != 0 {
		t.Errorf("yorum içindeki script hash'lenmemeli: %v", h)
	}
}

// TestCSPWriterHTMLOlmeyan HTML olmayan yanıtlara CSP hash'i eklenmemeli.
func TestCSPWriterHTMLOlmeyan(t *testing.T) {
	rec := httptest.NewRecorder()
	s := &Server{}
	h := s.withHTMLCSP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/api", nil))
	if strings.Contains(rec.Header().Get("Content-Security-Policy"), "sha256-") {
		t.Errorf("JSON yanıtına script hash'i eklenmemeli: %s", rec.Header().Get("Content-Security-Policy"))
	}
}
