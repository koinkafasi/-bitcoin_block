package httpapi

import (
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"
)

// ---- panic kurtarma ----
//
// NEDEN GEREKLİ: Go'nun net/http sunucusu bir handler'da panic olursa
// bağlantıyı sıfırlar (`readLoop`/`conn.serve` içindeki recover). Yani:
//   - istemci 500 SAYFASI değil, KOPUK BAĞLANTI görür (curl'de "empty reply",
//     tarayıcıda "bağlantı sıfırlandı"),
//   - yanıt gövdesine hiçbir açıklama yazılmaz,
//   - HTMX istemcisi yanıtı takas edemez; hata sessizce kaybolur,
//   - cspWriter tamponu flush edilmediği için yazılmış kısım da kaybolur.
//
// withRecover bunu düzeltir: panic'i yakalar, günlüğe yığın iziyle yazar ve
// istemciye düzgün bir 500 yanıtı döner (HTML istekte site kabuğu, API'de
// JSON, HTMX'te kısa fragment).
//
// GÜVENLİK: yığın izi ve panic metni YALNIZ sunucu günlüğüne gider.
// İstemciye sabit bir mesaj döner — iç yol, SQL, değişken değeri sızmaz.
func (s *Server) withRecover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}
			// http.ErrAbortHandler bilinçli bir iptaldir (SSE'de yaygın);
			// yeniden fırlatılır ki net/http normal davransın.
			if err, ok := rec.(error); ok && errors.Is(err, http.ErrAbortHandler) {
				panic(rec)
			}

			s.logger.Error("handler panic",
				"method", r.Method,
				"path", r.URL.Path,
				"panic", fmt.Sprint(rec),
				"stack", string(debug.Stack()),
			)

			s.panicYanit(w, r)
		}()
		next.ServeHTTP(w, r)
	})
}

// panicYanit panic sonrası istemciye döndürülecek yanıtı seçer.
//
// ÖNEMLİ: handler yanıtı YAZMAYA BAŞLAMIŞ olabilir (kısmi gövde). O durumda
// HTTP durum kodu artık değiştirilemez ve tam bir HTML belgesi eklemek
// yanıtı bozar. Böyle bir durumda yalnız kısa bir işaret yazılır.
//
// Yazılıp yazılmadığını bilmek için ara katmanlar `YazildiMi()` sunar
// (bkz. gzipResponseWriter, cspWriter). Sunmuyorsa "yazılmamış" varsayılır;
// bu, cspWriter tamponladığı için normal akışta doğrudur.
func (s *Server) panicYanit(w http.ResponseWriter, r *http.Request) {
	if yazildi(w) {
		s.logger.Warn("panic: yanıt zaten başlamıştı, durum kodu değiştirilemedi",
			"method", r.Method, "path", r.URL.Path)
		// Yalnız işaret; tam belge eklemek kısmi gövdeyi bozar.
		_, _ = w.Write([]byte("\n<!-- sunucu hatası -->"))
		return
	}

	hx := strings.EqualFold(r.Header.Get("HX-Request"), "true")
	kabul := r.Header.Get("Accept")

	switch {
	case hx:
		// HTMX: 500 + kısa, güvenli fragment. Global toast HX-Trigger ile
		// tetiklenir (bkz. static/app.js hata dinleyicisi).
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("HX-Trigger", `{"tbToast":{"kind":"error","text":"Beklenmeyen bir hata oluştu, lütfen tekrar deneyin."}}`)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`<div class="flash error" role="alert">Beklenmeyen bir hata oluştu.</div>`))

	case istekJSON(r, kabul):
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"ok":false,"error":"beklenmeyen hata"}`))

	case strings.Contains(kabul, "text/html"):
		// Tam sayfa: marka uyumlu hata sayfası (CSP'li; middleware üretir).
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(hataSayfasiHTML))

	default:
		// curl, bot, dosya indirme vb.: kısa düz metin.
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("iç sunucu hatası\n"))
	}
}

// yazildiMi yanıt başlığının/gövdesinin yazılmaya başlandığını bildiren
// ara katmanların arayüzü.
type yazildiMi interface{ YazildiMi() bool }

// yazildi ara katman zincirinde yanıtın başlayıp başlamadığını sorar.
func yazildi(w http.ResponseWriter) bool {
	for {
		if y, ok := w.(yazildiMi); ok {
			return y.YazildiMi()
		}
		u, ok := w.(interface{ Unwrap() http.ResponseWriter })
		if !ok {
			return false
		}
		w = u.Unwrap()
	}
}

// istekJSON yanıtın JSON beklenip beklemediğini kaba ama güvenli biçimde
// tahmin eder.
func istekJSON(r *http.Request, kabul string) bool {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		return true
	}
	if strings.Contains(kabul, "application/json") && !strings.Contains(kabul, "text/html") {
		return true
	}
	return false
}

// hataSayfasiHTML sunucu hatası için sabit, bağımsız bir sayfa.
//
// NEDEN ŞABLON DEĞİL: panic anında şablon motoru veya tmplData hazır
// olmayabilir; ikinci bir panic üretmek en kötü senaryodur. Bu yüzden
// sayfa hiçbir dış veriye bağlı değil.
const hataSayfasiHTML = `<!doctype html><html lang="tr"><head><meta charset="utf-8">` +
	`<meta name="viewport" content="width=device-width,initial-scale=1">` +
	`<meta name="robots" content="noindex">` +
	`<title>Sunucu hatası</title>` +
	`<style>body{font:16px/1.6 system-ui,-apple-system,"Segoe UI",sans-serif;margin:0;` +
	`display:grid;place-items:center;min-height:100vh;background:#f7f8fa;color:#1f2937}` +
	`main{max-width:34rem;padding:2rem;text-align:center}` +
	`h1{font-size:1.5rem;margin:0 0 .5rem}p{color:#4b5563;margin:0 0 1.25rem}` +
	`a{display:inline-block;padding:.6rem 1.2rem;background:#2563eb;color:#fff;` +
	`border-radius:.5rem;text-decoration:none}a:focus-visible{outline:3px solid #93c5fd}` +
	`</style></head><body><main><h1>Beklenmeyen bir hata oluştu</h1>` +
	`<p>Sayfa yüklenemedi. Sorun devam ederse lütfen daha sonra tekrar deneyin.</p>` +
	`<a href="/">Ana sayfaya dön</a></main></body></html>`
