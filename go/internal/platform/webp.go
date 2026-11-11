// WebP kodlama: Google cwebp binary'si varsa kullanılır; yoksa ErrNoWebP döner.
// (Go ekosisteminde saf-Go WebP kodlayıcı yok — x/image yalnız çözer.)
package platform

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

// ErrNoWebP cwebp bulunamadığında döner.
var ErrNoWebP = errors.New("cwebp yok; WebP üretilemedi")

// WebpFromJpeg JPEG baytlarını WebP'ye çevirir (geçici dosya + cwebp).
func WebpFromJpeg(jpeg []byte, quality int) ([]byte, error) {
	if quality <= 0 || quality > 100 {
		quality = 85
	}
	bin, err := exec.LookPath("cwebp")
	if err != nil {
		return nil, ErrNoWebP
	}
	dir, err := os.MkdirTemp("", "wb")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	in := filepath.Join(dir, "in.jpg")
	out := filepath.Join(dir, "out.webp")
	if err := os.WriteFile(in, jpeg, 0o644); err != nil {
		return nil, err
	}
	cmd := exec.Command(bin, "-quiet", "-q", strconv.Itoa(quality), in, "-o", out)
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return os.ReadFile(out)
}
