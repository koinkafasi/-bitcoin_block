package webui

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"unicode"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func emojiMi(r rune) bool {
	if r == 0x2318 { // ⌘
		return false
	}
	switch {
	case r >= 0x1F000 && r <= 0x1FAFF: // emoji blokları
		return true
	case r >= 0x2600 && r <= 0x27BF: // çeşitli simgeler + dingbats
		return true
	case r >= 0x2B00 && r <= 0x2BFF:
		return true
	case r >= 0x2300 && r <= 0x23FF: // teknik
		return true
	case r >= 0x25A0 && r <= 0x25FF: // geometrik şekiller
		return true
	case r >= 0x2800 && r <= 0x28FF: // braille: sürükleme tutamacı olarak kullanılıyordu
		return true
	}
	return false
}

func metinDugumleri(s string) []string {
	re := regexp.MustCompile(`(?s)<(style|script)[^>]*>.*?</(style|script)>`)
	s = re.ReplaceAllString(s, " ")

	var temiz []string
	for _, satir := range strings.Split(s, "\n") {
		if strings.HasPrefix(strings.TrimSpace(satir), "//") {
			continue
		}
		temiz = append(temiz, satir)
	}
	s = strings.Join(temiz, "\n")
	var out []string
	depth := 0
	var sb strings.Builder
	for _, r := range s {
		switch r {
		case '<':
			depth++
			if sb.Len() > 0 {
				out = append(out, sb.String())
				sb.Reset()
			}
		case '>':
			if depth > 0 {
				depth--
			}
		default:
			if depth == 0 {
				sb.WriteRune(r)
			}
		}
	}
	if sb.Len() > 0 {
		out = append(out, sb.String())
	}
	return out
}

func emojiBul(yol string) []string {
	b, err := os.ReadFile(yol)
	if err != nil {
		return nil
	}
	return emojiMetin(string(b))
}

// emojiMetin ham şablon metninden görünen metin düğümlerindeki emojileri
// toplar.
func emojiMetin(ham string) []string {
	var kotu []string
	for _, t := range metinDugumleri(ham) {
		for _, r := range t {
			if emojiMi(r) && !unicode.IsSpace(r) {
				kotu = append(kotu, string(r)+" → "+strings.TrimSpace(t))
			}
		}
	}
	return kotu
}

// TestIcerikFormEmojiKullanmaz: içerik formundaki 33 glif temizlendi.
func TestIcerikFormEmojiKullanmaz(t *testing.T) {
	kotu := emojiBul(filepath.Join("templkit", "icerikform.templ"))
	assert.Emptyf(t, kotu, "içerik formunda emoji kaldı: %v", kotu)
}

// TestGazeteEditorEmojiKullanmaz: editör araç çubuğu + blok paleti.
func TestGazeteEditorEmojiKullanmaz(t *testing.T) {
	kotu := emojiBul(filepath.Join("templkit", "gazeteeditor.templ"))
	assert.Emptyf(t, kotu, "gazete editöründe emoji kaldı: %v", kotu)
}

func TestEditorAracCubuguSimgeli(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "..", "..", "editor-src", "src", "editor.js"))
	require.NoError(t, err)
	s := string(b)

	for _, ad := range []string{"@arrow-back-up", "@microphone", "@table", "@photo",
		"@link", "@message-circle", "@history", "@keyboard"} {
		assert.Containsf(t, s, "'"+ad+"'", "araç çubuğunda %s olmalı", ad)
	}

	// Araç çubuğu bölümünde emoji kalmamalı.
	i := strings.Index(s, "function buildToolbar()")
	require.Greater(t, i, 0)
	son := strings.Index(s[i:], "const more = iconButton")
	require.Greater(t, son, 0)
	blok := s[i : i+son]
	for _, e := range []string{"🎤", "🌐", "📄", "🔗", "💬", "🕘", "▦", "☰", "▸"} {
		assert.NotContainsf(t, blok, e, "araç çubuğunda emoji kaldı: %s", e)
	}
}

func TestIkonUreticiEditorKaynaginiTara(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "..", "..", "deploy", "scripts", "ikon-uret.py"))
	require.NoError(t, err)
	s := string(b)

	assert.Contains(t, s, "editor-src/src", "ikon üretici editör kaynağını taramalı")
	assert.Contains(t, s, `"'@([a-z0-9-]+)'"`, "'@ad' etiket deseni taranmalı")
}

// TestGazetePaletiSimgeli: 11 palet düğmesinin hepsi SVG taşımalı.
func TestGazetePaletiSimgeli(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("templkit", "gazeteeditor.templ"))
	require.NoError(t, err)
	s := string(b)

	assert.Equal(t, 11, strings.Count(s, `class="gz-add"`),
		"blok paletinde 11 düğme olmalı")

	dugme := regexp.MustCompile(`(?s)<button\b[^>]*>(.*?)</button>`)
	eslesmeler := dugme.FindAllStringSubmatch(s, -1)
	require.NotEmpty(t, eslesmeler, "düzenleyicide hiç düğme bulunamadı")
	for _, m := range eslesmeler {
		assert.Contains(t, m[1], `class="ti"`,
			"simgesiz düğme: %s", strings.TrimSpace(m[1]))
	}
}

func TestTumTemplDosyalariEmojisiz(t *testing.T) {
	dizin, err := filepath.Glob(filepath.Join("templkit", "*.templ"))
	require.NoError(t, err)
	require.Greater(t, len(dizin), 20, "taranacak dosya sayısı beklenenden az")

	for _, f := range dizin {
		kotu := emojiBul(f)
		assert.Emptyf(t, kotu, "%s: emoji kaldı: %v", filepath.Base(f), kotu)
	}
}

func TestAdminSablonlariEmojisiz(t *testing.T) {
	dizin, err := filepath.Glob(filepath.Join("templates", "*.html"))
	require.NoError(t, err)

	disarida := map[string]bool{
		"site_public.html": true, // okuyucu sayfaları
		"site_uye.html":    true,
	}

	taranan := 0
	for _, f := range dizin {
		ad := filepath.Base(f)
		if disarida[ad] {
			continue
		}
		b, err := os.ReadFile(f)
		require.NoError(t, err)
		s := string(b)
		if !strings.Contains(s, "admin_head") && ad != "partials.html" {
			continue
		}
		taranan++

		// Public menü satırlarını çıkar.
		var temiz []string
		for _, satir := range strings.Split(s, "\n") {
			if strings.Contains(satir, "LangBase") {
				continue
			}
			temiz = append(temiz, satir)
		}
		kotu := emojiMetin(strings.Join(temiz, "\n"))
		assert.Emptyf(t, kotu, "%s: emoji kaldı: %v", ad, kotu)
	}
	assert.GreaterOrEqual(t, taranan, 15, "panel şablonu sayısı beklenenden az")
}

func TestAdminJSTutamaciSpritedir(t *testing.T) {
	for _, ad := range []string{"menu.js", "tabcats.js"} {
		b, err := os.ReadFile(filepath.Join("static", ad))
		require.NoError(t, err)
		s := string(b)
		assert.NotContainsf(t, s, "⠿", "%s: braille tutamaç kaldı", ad)
		assert.Containsf(t, s, "ti-grip-vertical", "%s: tutamaç sprite'tan gelmeli", ad)
	}
}
