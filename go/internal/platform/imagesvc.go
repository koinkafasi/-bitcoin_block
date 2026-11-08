// image-svc istemcisi — görsel kırpma/ölçekleme işini Rust mikroservisine
// devreder, servis yoksa/başarısızsa yerel Go yoluna düşer.
//
// NEDEN: aynı iş (1600×900 JPEG → 1200 px, 16:9 kırpma, JPEG) ölçüldü
// (7 Kasım 2026, aynı makine):
//
//	Rust image-svc (libjpeg-turbo): p50 45 ms
//	Go (x/image CatmullRom + image/jpeg): p50 109 ms
//
// Fark dil değil kodlayıcı: Go'nun image/jpeg'i saf Go, SIMD/libjpeg-turbo
// kullanmıyor. Servis yalnız JPEG girdide devreye alınır (PNG saydamlığı
// Rust tarafında beyaz zemine çevrilmiyor) ve HER hata durumunda yerel yol
// kullanılır — servis çökmesi görsel üretimini durdurmaz.
package platform

import (
	"bytes"
	"context"
	"fmt"
	"image/jpeg"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// imageSvcYenidenDene: servis bir kez hata verince bu süre boyunca hiç
// çağrılmaz. Aksi hâlde ölü servis her görselde 5 sn zaman aşımı eklerdi —
// yani kazanç sağlamak yerine sayfayı yavaşlatırdı.
const imageSvcYenidenDene = 30 * time.Second

type imageSvcDurum struct {
	mu    sync.RWMutex
	url   string
	hc    *http.Client
	until time.Time // bu ana kadar devre dışı
}

var imageSvc = imageSvcDurum{
	hc: &http.Client{Timeout: 5 * time.Second},
}

// SetImageSvcURL image-svc adresini ayarlar (ör. http://127.0.0.1:8170).
// Boş değer servisi kapatır ve davranışı tamamen yerel yola döndürür.
func SetImageSvcURL(u string) {
	imageSvc.mu.Lock()
	defer imageSvc.mu.Unlock()
	imageSvc.url = strings.TrimRight(strings.TrimSpace(u), "/")
	imageSvc.until = time.Time{}
}

// ImageSvcURL yapılandırılmış adresi döndürür (boşsa servis kapalı).
func ImageSvcURL() string {
	imageSvc.mu.RLock()
	defer imageSvc.mu.RUnlock()
	return imageSvc.url
}

// imageSvcKapali servisin geçici olarak devre dışı olup olmadığını söyler.
func imageSvcKapali() bool {
	imageSvc.mu.RLock()
	defer imageSvc.mu.RUnlock()
	return imageSvc.url == "" || time.Now().Before(imageSvc.until)
}

func imageSvcHata(err error) {
	imageSvc.mu.Lock()
	imageSvc.until = time.Now().Add(imageSvcYenidenDene)
	imageSvc.mu.Unlock()
	_ = err
}

// imageSvcIsle servise ham görsel gönderir; 16:9 kırpma + ölçekleme + JPEG
// kodlamayı Rust tarafı yapar.
func imageSvcIsle(ctx context.Context, veri []byte, ratio float64, width, quality int, format string) ([]byte, error) {
	if imageSvcKapali() {
		return nil, fmt.Errorf("image-svc kapalı")
	}
	imageSvc.mu.RLock()
	base, hc := imageSvc.url, imageSvc.hc
	imageSvc.mu.RUnlock()

	q := url.Values{}
	q.Set("ratio", strconv.FormatFloat(ratio, 'f', -1, 64))
	q.Set("width", strconv.Itoa(width))
	if quality > 0 {
		q.Set("quality", strconv.Itoa(quality))
	}
	if format != "" {
		q.Set("format", format)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/v1/process?"+q.Encode(), bytes.NewReader(veri))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := hc.Do(req)
	if err != nil {
		imageSvcHata(err)
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	// Servis hata gövdesi JSON döndürür ve çıktı görselden küçüktür; 8 MB
	// sınırı bozuk/yanlış yanıtta belleği korur.
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		imageSvcHata(err)
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		imageSvcHata(fmt.Errorf("image-svc %d: %s", resp.StatusCode, strings.TrimSpace(string(body))))
		return nil, fmt.Errorf("image-svc %d", resp.StatusCode)
	}
	if len(body) == 0 {
		imageSvcHata(fmt.Errorf("image-svc boş gövde"))
		return nil, fmt.Errorf("image-svc boş gövde")
	}
	return body, nil
}

// jpegMi girdi gerçekten JPEG mi (SOI işareti). PNG/WebP yerel yolda kalır:
// Rust tarafı saydamlığı beyaz zemine çevirmiyor, renk değişirdi.
func jpegMi(b []byte) bool {
	return len(b) > 3 && b[0] == 0xFF && b[1] == 0xD8
}

// resizeViaImageSvc JPEG girdiyi image-svc ile hedef genişliğe ölçekler.
// İkinci dönüş servis kullanıldı mı sorusunun cevabıdır.
func resizeViaImageSvc(ctx context.Context, data []byte, width int) ([]byte, bool) {
	if width <= 0 || !jpegMi(data) || imageSvcKapali() {
		return nil, false
	}
	cfg, err := jpeg.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 {
		return nil, false
	}
	// Küçük görseli büyütmeme kuralı yerel yolda da aynı: hedef genişlik
	// kaynaktan büyükse dokunma.
	if width > cfg.Width {
		return nil, false
	}
	ratio := float64(cfg.Width) / float64(cfg.Height)
	// q=82 yerel yoldaki encodeJPEG(dst, 82) ile aynı: servise geçmek çıktı
	// boyutunu ya da kaliteyi değiştirmesin, yalnız süreyi düşürsün.
	out, err := imageSvcIsle(ctx, data, ratio, width, 82, "jpeg")
	if err != nil {
		return nil, false
	}
	return out, true
}
