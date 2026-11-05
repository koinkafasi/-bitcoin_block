package httpapi

import (
	"compress/gzip"
	"net/http"
	"strings"
	"sync"
)

// gzipPool büyük sayfalarda sürekli yeni writer üretmemek için kullanılır.
var gzipPool = sync.Pool{New: func() any { return gzip.NewWriter(nil) }}

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
		// JavaScript'in İKİ kayıtlı türü var. Go 1.20'den beri `.js`
		// dosyaları `text/javascript` olarak sunuluyor (WHATWG).
		// Faz 172'ye kadar yalnız `application/javascript` listede olduğu
		// için panelin HİÇBİR JS dosyası sıkıştırılmıyordu: editor.js
		// 771 KB ham gidiyordu, gzip'li hâli 240 KB.
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
		gz := gzipPool.Get().(*gzip.Writer)
		gz.Reset(w.ResponseWriter)
		w.gz = gz
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
		gzipPool.Put(w.gz)
		w.gz = nil
	}
}

// Unwrap ResponseController'ın alt writer'a erişebilmesini sağlar.
func (w *gzipResponseWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
