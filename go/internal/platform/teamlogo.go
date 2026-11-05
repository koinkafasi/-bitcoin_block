package platform

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

// Takım logoları — kendi altyapımızda barındırma.
//
// Sorun: puan durumu ve fikstür logoları dış CDN'lerden hotlink ediliyordu
// (a.espncdn.com, r2.thesportsdb.com). Dış CDN'e bağımlılık üç şekilde
// kırılıyor: CDN bizim IP'mizi engelleyebiliyor (ölçüm 28 Ekim 2026: 18/18
// logo bağlantı sıfırlandı), takım kimliği değişince URL 404 oluyor, ve
// tarayıcıda referer/CORS politikası değişebiliyor. Sonuç sayfada kırık
// resim simgesi.
//
// Çözüm: logo bir kez indirilir, `storage/logos/teams/` altına yazılır ve
// sonsuza kadar kendi alan adımızdan servis edilir. Dış CDN yalnızca
// "tedarikçi" olur, çalışma zamanı bağımlılığı olmaktan çıkar.

// TeamLogoDir takım logolarının disk köküdür.
func TeamLogoDir() string {
	return filepath.Join(storageDir, "logos", "teams")
}

// teamLogoRe güvenli anahtar biçimi: küçük harf, rakam, tire. Bu doğrulama
// yol geçişi (../) ve gizli dosya erişimini baştan engeller.
var teamLogoRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)

// ValidTeamLogoKey anahtarın güvenli olup olmadığını söyler.
func ValidTeamLogoKey(key string) bool { return teamLogoRe.MatchString(key) }

// ErrTeamLogoNotFound logo diskte yok.
var ErrTeamLogoNotFound = errors.New("takım logosu yok")

// TeamLogoPath anahtarın disk yolunu döndürür. Anahtar geçersizse hata verir.
func TeamLogoPath(key string) (string, error) {
	if !ValidTeamLogoKey(key) {
		return "", fmt.Errorf("geçersiz logo anahtarı: %q", key)
	}
	return filepath.Join(TeamLogoDir(), key+".png"), nil
}

// OpenTeamLogo logoyu okumak için açar. Dosya yoksa ErrTeamLogoNotFound döner.
func OpenTeamLogo(key string) (*os.File, os.FileInfo, error) {
	p, err := TeamLogoPath(key)
	if err != nil {
		return nil, nil, err
	}
	f, err := os.Open(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, ErrTeamLogoNotFound
		}
		return nil, nil, err
	}
	st, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, nil, err
	}
	return f, st, nil
}

// HasTeamLogo logosu diskte var mı?
func HasTeamLogo(key string) bool {
	p, err := TeamLogoPath(key)
	if err != nil {
		return false
	}
	st, err := os.Stat(p)
	return err == nil && st.Mode().IsRegular() && st.Size() > 0
}

// pngMagic indirilen gövdenin PNG olduğunu doğrular. İçerik türü başlığına
// güvenilmez: CDN'ler hata sayfasını da 200 ile dönebiliyor.
var pngMagic = []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}

// SaveTeamLogo logoyu diske atomik yazar (önce .tmp, sonra rename) — yarım
// kalan indirme geçerli bir logo gibi görünmesin.
func SaveTeamLogo(key string, data []byte) error {
	if !ValidTeamLogoKey(key) {
		return fmt.Errorf("geçersiz logo anahtarı: %q", key)
	}
	if len(data) < len(pngMagic)+64 || !bytes.HasPrefix(data, pngMagic) {
		return errors.New("gövde PNG değil")
	}
	p, err := TeamLogoPath(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(p), key+"-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // rename başarılıysa bu no-op
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, p)
}

// TeamLogoModTime logonun yazılma zamanı (ETag için); yoksa sıfır zaman.
func TeamLogoModTime(key string) time.Time {
	p, err := TeamLogoPath(key)
	if err != nil {
		return time.Time{}
	}
	st, err := os.Stat(p)
	if err != nil {
		return time.Time{}
	}
	return st.ModTime()
}
