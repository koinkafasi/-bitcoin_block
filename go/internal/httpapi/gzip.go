package httpapi

import (
	"compress/gzip"
	"net/http"
	"strings"
	"sync"
)

// gzipPool büyük sayfalarda sürekli yeni writer üretmemek için kullanılır.
//
// SEVİYE ölçülerek seçildi (9 Kasım 2026, 77 KB gerçek ana sayfa gövdesi,
// 60 tur, aynı makine):
//
//	seviye  çıktı    oran     cpu/istek
//	1       18 KB    23,8%    0,37 ms   (BestSpeed)
//	3       17 KB    22,7%    0,43 ms
//	4       17 KB    22,2%    0,45 ms
//	6       16 KB    21,2%    0,63 ms   (Go varsayılanı — ESKİ davranış)
//	9       15 KB    20,6%    1,16 ms
//
// Varsayılan 6'dan 4'e inmek sıkıştırma CPU'sunu **%29** düşürüyor,
// karşılığında gövde 16 → 17 KB (+%6) oluyor. Profil (yük altında, 10 sn):
// flate CPU'nun %31'i — istek yolundaki EN PAHALI tek kalem. 1 KB'lik
// aktarım farkı, Cloudflare'ın önbelleği ve kenar sıkıştırması düşünülünce
// önemsiz; buna karşılık haber patlamalarında (ör. borsa çöküşü, bot
// taraması) önbellek ıskalayan her istek bu CPU'yu ödüyor.
//
// `GZIP_LEVEL` ile değiştirilebilir (1–9). Geçersiz değer varsayılana döner.
const varsayilanGzipSeviyesi = 4

var (
	gzipSeviyeMu sync.RWMutex
	gzipSeviye   = varsayilanGzipSeviyesi

	// SEVİYE BAŞINA AYRI HAVUZ.
	//
	// `gzip.Writer.Reset()` seviyeyi DEĞİŞTİRMEZ: seviye, writer
	// oluşturulurken sabitlenir. Tek havuz kullanılsaydı seviye
	// değiştiğinde havuzdan eski seviyeli bir writer alınır ve istenen
	// seviye sessizce yok sayılırdı. Bunu bir test yakaladı
	// (gzip_seviye_test.go: seviye 9 ile 1 aynı boyutu üretiyordu).
	//
	// Havuzlar 1..9 indekslidir (0 ve negatifler kullanılmaz), böylece
	// indeks = seviye olur ve ayrı bir eşleme gerekmez.
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

// SetGzipLevel sıkıştırma seviyesini ayarlar (1–9). Aralık dışı yok sayılır.
// -1 (DefaultCompression) ve -2 (HuffmanOnly) bilinçli olarak kabul edilmez:
// yapılandırma değeri olarak 1–9 beklenir.
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
