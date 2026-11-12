package cryptoicon

import (
	"embed"
	"io/fs"
	"sort"
	"strings"
)

//go:embed icons/*.svg
var embedded embed.FS

const (
	URLBase = "/media/crypto/"
	// ContentType, SVG ikonların MIME türüdür.
	ContentType = "image/svg+xml"
	// SourceVersion, ikon setinin alındığı kaynak commit'in kısa kimliğidir.
	// URL'ye ?v=... olarak eklenir; ikon değişince tarayıcı önbelleği kırılır.
	SourceVersion = "1a63530"
)

var aliases = map[string]string{
	"iota": "miota",
	"pol":  "matic",
}

// icons, gömülü SVG dosyalarını dosya adı (uzantısız, küçük harf) ile eşler.
var icons = func() map[string][]byte {
	m := make(map[string][]byte)
	entries, err := fs.ReadDir(embedded, "icons")
	if err != nil {
		return m
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".svg") {
			continue
		}
		b, err := embedded.ReadFile("icons/" + name)
		if err != nil {
			continue
		}
		m[strings.TrimSuffix(name, ".svg")] = b
	}
	return m
}()

// normalize sembolü dosya adı biçimine indirger. Yalnız a-z, 0-9 ve $ korunur;
// böylece "../" gibi yol parçaları veya boşluklar aramaya giremez.
func normalize(symbol string) string {
	s := strings.ToLower(strings.TrimSpace(symbol))
	if s == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '$' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// key, sembolün ikon anahtarını döndürür; ikon yoksa boş dize.
func key(symbol string) string {
	k := normalize(symbol)
	if k == "" {
		return ""
	}
	if _, ok := icons[k]; ok {
		return k
	}
	if a, ok := aliases[k]; ok {
		if _, ok := icons[a]; ok {
			return a
		}
	}
	return ""
}

// URL, sembol için yerel ikon URL'sini döndürür. İkon yoksa boş dize döner;
// şablonlar bu durumda harf avatarına düşer.
func URL(symbol string) string {
	k := key(symbol)
	if k == "" {
		return ""
	}
	return URLBase + k + ".svg?v=" + SourceVersion
}

// Bytes, sembolün SVG içeriğini döndürür. İkon yoksa ok=false.
func Bytes(symbol string) ([]byte, bool) {
	k := key(symbol)
	if k == "" {
		return nil, false
	}
	b, ok := icons[k]
	return b, ok
}

// Count, kullanılabilir ikon sayısını döndürür.
func Count() int { return len(icons) }

// Symbols, ikon anahtarlarını sıralı olarak döndürür (test ve teşhis için).
func Symbols() []string {
	out := make([]string, 0, len(icons))
	for k := range icons {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
