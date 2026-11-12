package httpapi

import (
	"compress/gzip"
	"net/http"
	"strings"
	"sync"
)

const varsayilanGzipSeviyesi = 4

var (
	gzipSeviyeMu sync.RWMutex
	gzipSeviye   = varsayilanGzipSeviyesi

	gzipPools [gzip.BestCompression + 1]sync.Pool
)

func init() {
	for lvl := gzip.BestSpeed; lvl <= gzip.BestCompression; lvl++ {
		lvl := lvl
		gzipPools[lvl] = sync.Pool{New: func() any {
			w, err := gzip.NewWriterLevel(nil, lvl)
			if err != nil {
				// BestSpeed..BestCompression aralığında hata beklenmez;
				// yine de panik yerine güvenli varsayılana düşülür.
				return gzip.NewWriter(nil)
			}
			return w
		}}
	}
}

func SetGzipLevel(n int) {
	if n < gzip.BestSpeed || n > gzip.BestCompression {
		return
	}
	gzipSeviyeMu.Lock()
	gzipSeviye = n
	gzipSeviyeMu.Unlock()
}

// GzipLevel yapılandırılmış seviyeyi döndürür (tanı/log için).
func GzipLevel() int {
	gzipSeviyeMu.RLock()
	defer gzipSeviyeMu.RUnlock()
	return gzipSeviye
}

// gzipAl seviyeye uygun havuzdan writer alır.
func gzipAl(seviye int) *gzip.Writer {
	if seviye < gzip.BestSpeed || seviye > gzip.BestCompression {
		seviye = varsayilanGzipSeviyesi
	}
	return gzipPools[seviye].Get().(*gzip.Writer)
}

// gzipBirak writer'ı kendi seviyesindeki havuza koyar.
func gzipBirak(w *gzip.Writer, seviye int) {
	if w == nil || seviye < gzip.BestSpeed || seviye > gzip.BestCompression {
		return
	}
	gzipPools[seviye].Put(w)
}

// withGzip metin yanıtlarını (HTML/CSS/JS/JSON/XML) sıkıştırır. Görseller ve
// SSE akışları sıkıştırılmaz; SSE'de Flush davranışı korunur.
func (s *Server) withGzip(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !acceptsGzip(r) || skipGzipPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		gw := &gzipResponseWriter{ResponseWriter: w}
		defer gw.Close()
		next.ServeHTTP(gw, r)
	})
}

func acceptsGzip(r *http.Request) bool {
	for _, part := range strings.Split(r.Header.Get("Accept-Encoding"), ",") {
		if strings.EqualFold(strings.TrimSpace(strings.SplitN(part, ";", 2)[0]), "gzip") {
			return true
		}
	}
	return false
}

func skipGzipPath(p string) bool {
	return strings.HasPrefix(p, "/media/") || strings.HasPrefix(p, "/canli/")
}

func compressibleContentType(ct string) bool {
	ct = strings.ToLower(strings.TrimSpace(strings.SplitN(ct, ";", 2)[0]))
	switch ct {
	case "text/html", "text/css", "text/plain", "text/xml", "text/event-stream",
		"text/javascript", "application/javascript", "application/x-javascript",
		"application/json", "application/xml",
		"application/manifest+json", "image/svg+xml":
		return ct != "text/event-stream"
	}
	return false
}

func addVary(h http.Header, value string) {
	for _, v := range h.Values("Vary") {
		for _, part := range strings.Split(v, ",") {
			if strings.EqualFold(strings.TrimSpace(part), value) {
				return
			}
		}
	}
	h.Add("Vary", value)
}

type gzipResponseWriter struct {
	http.ResponseWriter
	gz          *gzip.Writer
	gzSeviye    int // writer'ın havuza dönerken taşıdığı seviye
	wroteHeader bool
	compress    bool
}

// YazildiMi yanıt başlığının yazılıp yazılmadığını bildirir.
// Panik kurtarma bunu kullanır: başlık yazıldıysa durum kodu değiştirilemez.
func (w *gzipResponseWriter) YazildiMi() bool { return w.wroteHeader }

func (w *gzipResponseWriter) WriteHeader(code int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true
	// 1xx/204/304/206 gibi gövdesiz veya parçalı yanıtlar sıkıştırılmaz.
	if code >= 200 && code != http.StatusNoContent && code != http.StatusNotModified &&
		code != http.StatusPartialContent && w.Header().Get("Content-Encoding") == "" &&
		compressibleContentType(w.Header().Get("Content-Type")) {
		w.compress = true
		w.Header().Set("Content-Encoding", "gzip")
		addVary(w.Header(), "Accept-Encoding")
		w.Header().Del("Content-Length")
		// Seviye writer'a GİRİŞTE sabitlenir; havuza aynı seviyeye döner.
		seviye := GzipLevel()
		gz := gzipAl(seviye)
		gz.Reset(w.ResponseWriter)
		w.gz = gz
		w.gzSeviye = seviye
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *gzipResponseWriter) Write(p []byte) (int, error) {
	if !w.wroteHeader {
		// html/template ve bazı handler'lar Content-Type'ı yazmadan bırakır;
		// ilk 512 bayttan türü belirleyip sıkıştırma kararını ver.
		if w.Header().Get("Content-Type") == "" && len(p) > 0 {
			w.Header().Set("Content-Type", http.DetectContentType(p))
		}
		w.WriteHeader(http.StatusOK)
	}
	if w.gz != nil {
		return w.gz.Write(p)
	}
	return w.ResponseWriter.Write(p)
}

func (w *gzipResponseWriter) Flush() {
	if w.gz != nil {
		_ = w.gz.Flush()
	}
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (w *gzipResponseWriter) Close() {
	if w.gz != nil {
		_ = w.gz.Close()
		gzipBirak(w.gz, w.gzSeviye)
		w.gz = nil
	}
}

// Unwrap ResponseController'ın alt writer'a erişebilmesini sağlar.
func (w *gzipResponseWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
