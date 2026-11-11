package platform

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// defaultStorageDir env STORAGE_DIR yoksa "storage" (çalışma dizinine göre).
var storageDir = func() string {
	if v := os.Getenv("STORAGE_DIR"); v != "" {
		return v
	}
	return "storage"
}()

// StorageDir medya depolama kök dizinini döndürür.
func StorageDir() string { return storageDir }

// SetStorageDir depolama kökünü değiştirir.
//
// Üretimde kullanılmaz: kök `STORAGE_DIR` ortam değişkeninden bir kez okunur.
// Testler ve bakım araçları (cmd/teamlogos) geçici/özel bir dizinle
// çalışabilsin diye açıktır.
func SetStorageDir(dir string) {
	if dir != "" {
		storageDir = dir
	}
}

// privateStorageDir röportaj sesleri gibi gizli dosyaların köküdür. Bilerek
// public storageDir'in DIŞINDA tutulur; /media/ handler'ı bu ağaca erişemez.
var privateStorageDir = func() string {
	if v := os.Getenv("PRIVATE_STORAGE_DIR"); v != "" {
		return v
	}
	base := storageDir
	if abs, err := filepath.Abs(base); err == nil {
		base = abs
	}
	return filepath.Join(filepath.Dir(base), "storage-private")
}()

// PrivateStorageDir private depolama kök dizinini döndürür.
func PrivateStorageDir() string { return privateStorageDir }

// SavePrivateAudio kaynağı private depolamaya akıtır; belleğe tamamını almaz.
// Dönen yol private köke göre "audio/yyyymm/{ad}.{uzantı}" biçimindedir.
func SavePrivateAudio(src io.Reader, filename string) (string, int64, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" || len(ext) > 12 {
		ext = ".bin"
	}
	dir := filepath.Join(privateStorageDir, "audio", time.Now().Format("200601"))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", 0, err
	}
	tmp, err := os.CreateTemp(dir, "*.part")
	if err != nil {
		return "", 0, err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	size, err := io.Copy(tmp, src)
	if err != nil {
		_ = tmp.Close()
		return "", 0, err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return "", 0, err
	}
	if err := tmp.Close(); err != nil {
		return "", 0, err
	}
	name := randHex(8) + ext
	full := filepath.Join(dir, name)
	if err := os.Rename(tmpName, full); err != nil {
		return "", 0, err
	}
	return filepath.ToSlash(filepath.Join("audio", time.Now().Format("200601"), name)), size, nil
}

// PrivateFilePath private depolama yolunu güvenli biçimde tam yola çevirir.
func PrivateFilePath(storedPath string) (string, error) {
	safe := filepath.Clean(filepath.FromSlash(storedPath))
	if safe == "." || strings.HasPrefix(safe, "..") || filepath.IsAbs(safe) {
		return "", os.ErrPermission
	}
	return filepath.Join(privateStorageDir, safe), nil
}

// OpenPrivate private dosyayı okumak için açar.
func OpenPrivate(storedPath string) (*os.File, error) {
	full, err := PrivateFilePath(storedPath)
	if err != nil {
		return nil, err
	}
	return os.Open(full)
}

// DeletePrivate private dosyayı siler.
func DeletePrivate(storedPath string) error {
	full, err := PrivateFilePath(storedPath)
	if err != nil {
		return err
	}
	return os.Remove(full)
}

// SaveMedia veriyi {storage}/media/{yyyymm}/{rnd}{ext} altına yazar ve depolama yolunu döndürür.
func SaveMedia(data []byte, filename string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		ext = ".bin"
	}
	dir := filepath.Join(storageDir, "media", time.Now().Format("200601"))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	name := randHex(8) + ext
	p := filepath.Join(dir, name)
	// #nosec G703 -- Yol bileşenleri kod tarafından üretilir: `dir` sabit
	// depolama kökü + YYYYMM, `name` rastgele hex + filepath.Ext (ayırıcı
	// içeremez). Kullanıcı dosya adı yola girmiyor.
	if err := os.WriteFile(p, data, 0o644); err != nil {
		return "", err
	}
	// depolamada yayın yolunu normalize et (web: /media/...)
	return filepath.ToSlash(filepath.Join("media", time.Now().Format("200601"), name)), nil
}

// SaveMediaFile depolama içindeki mevcut dosyayı medya ağacına taşır (ör. TTS çıktısı).
func SaveMediaFile(fullSrcPath, filename string) (string, error) {
	data, err := os.ReadFile(fullSrcPath)
	if err != nil {
		return "", err
	}
	_ = os.Remove(fullSrcPath)
	return SaveMedia(data, filename)
}

// MediaVariantPath deterministik responsive görsel yolunu döndürür.
func MediaVariantPath(id int64, width int) string {
	return fmt.Sprintf("media/variants/%d-%d.jpg", id, width)
}

// SaveMediaVariant üretilen responsive görseli atomik olarak kaydeder.
func SaveMediaVariant(id int64, width int, data []byte) (string, error) {
	rel := MediaVariantPath(id, width)
	full := filepath.Join(storageDir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return "", err
	}
	tmp := full + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return "", err
	}
	if err := os.Rename(tmp, full); err != nil {
		return "", err
	}
	return rel, nil
}

// DeleteMediaVariants bir medya için üretilmiş tüm responsive varyantları siler.
func DeleteMediaVariants(id int64) {
	pattern := filepath.Join(storageDir, "media", "variants", fmt.Sprintf("%d-*.jpg", id))
	paths, _ := filepath.Glob(pattern)
	for _, p := range paths {
		_ = os.Remove(p)
	}
}

// ReadMedia depolama yolundan (slashed relative) dosyayı okur.
func ReadMedia(storedPath string) ([]byte, error) {
	safe := filepath.Clean(filepath.FromSlash(storedPath))
	if strings.HasPrefix(safe, "..") || filepath.IsAbs(safe) {
		return nil, os.ErrPermission
	}
	full := filepath.Join(storageDir, safe)
	_ = io.EOF
	return os.ReadFile(full)
}

// WriteMedia mevcut depolama yolunu atomik olmayan şekilde yeniden yazar.
func WriteMedia(storedPath string, data []byte) error {
	safe := filepath.Clean(filepath.FromSlash(storedPath))
	if strings.HasPrefix(safe, "..") || filepath.IsAbs(safe) {
		return os.ErrPermission
	}
	full := filepath.Join(storageDir, safe)
	tmp := full + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, full)
}

// DeleteMedia depolamadaki dosyayı siler.
func DeleteMedia(storedPath string) error {
	safe := filepath.Clean(filepath.FromSlash(storedPath))
	if strings.HasPrefix(safe, "..") || filepath.IsAbs(safe) {
		return os.ErrPermission
	}
	return os.Remove(filepath.Join(storageDir, safe))
}

func randHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return strings.Repeat("0", n*2)
	}
	return hex.EncodeToString(b)
}
