package webui

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMutedMetinRengiDegil: hiçbir dosyada `color: var(--muted)` kalmamalı.
func TestMutedMetinRengiDegil(t *testing.T) {
	dizinler := []string{"static", "templkit", "templates"}
	uzantilar := map[string]bool{".css": true, ".templ": true, ".html": true, ".js": true}

	var suclu []string
	for _, d := range dizinler {
		yol, err := filepath.Glob(filepath.Join(d, "*"))
		require.NoError(t, err)
		for _, f := range yol {
			if !uzantilar[filepath.Ext(f)] || strings.HasSuffix(f, "_templ.go") {
				continue
			}
			b, err := os.ReadFile(f)
			require.NoError(t, err)
			s := string(b)
			// Yorum satırlarındaki açıklama metnini sayma.
			for i, satir := range strings.Split(s, "\n") {
				if strings.Contains(satir, "var(--muted)") &&
					!strings.Contains(satir, "var(--muted-fg)") {
					kirli := strings.TrimSpace(satir)
					// Yalnızca kod satırları kural dışı; yorumlar serbest.
					if strings.HasPrefix(kirli, "//") || strings.HasPrefix(kirli, "*") ||
						strings.HasPrefix(kirli, "/*") || strings.Contains(kirli, "gecersiz") {
						continue
					}
					suclu = append(suclu, filepath.Base(f)+":"+strconv.Itoa(i+1))
				}
			}
		}
	}
	assert.Emptyf(t, suclu,
		"`--muted` metin rengi olarak kullanılmış (shadcn'de zemin rengidir); "+
			"`--muted-fg` kullanın: %v", suclu)
}

// TestMutedFgHerTemada: --muted-fg tanımlı olmalı, yoksa public tarafta
// olduğu gibi bildirim geçersiz olur ve renk sessizce miras alınır.
func TestMutedFgHerTemada(t *testing.T) {
	for _, dosya := range []string{"app.css", "public.css", "admin-shadcn.css"} {
		b, err := os.ReadFile(filepath.Join("static", dosya))
		require.NoError(t, err)
		s := string(b)
		assert.Containsf(t, s, "--muted-fg:",
			"%s içinde --muted-fg tanımlı olmalı", dosya)
	}

	// app.css ve public.css iki tema bloğu taşır (açık + koyu): ikisinde de
	// tanımlı olmalı, yoksa koyu temada renk miras alınır.
	for _, dosya := range []string{"app.css", "public.css"} {
		b, err := os.ReadFile(filepath.Join("static", dosya))
		require.NoError(t, err)
		n := strings.Count(string(b), "--muted-fg:")
		assert.GreaterOrEqualf(t, n, 2,
			"%s: --muted-fg iki temada da tanımlı olmalı (bulunan: %d)", dosya, n)
	}
}

func TestMutedArkaPlanTokeniDuruyor(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("static", "admin-shadcn.css"))
	require.NoError(t, err)
	assert.Contains(t, string(b), "--muted: var(--sh-muted);",
		"--muted zemin token'ı korunmalı")
}
