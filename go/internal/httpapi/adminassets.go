package httpapi

import "strings"

var adminPageScripts = []struct {
	// prefix yol öneki; tam eşleşme ya da ardından "/" gelmesi gerekir
	// ("/admin/icerikler-arsiv" içerik formu değildir).
	prefix  string
	exact   bool
	scripts []string
}{
	{"/admin/icerikler/yeni", false, contentFormScripts},
	{"/admin/icerikler/", false, contentFormScripts}, // düzenleme: /admin/icerikler/{id}
	{"/admin/manset-editor", false, []string{"manset.js"}},
	{"/admin/ses-ayarlari", false, []string{"ses_player.js"}},
	{"/admin", true, []string{"pano.js"}},
	{"/admin/pano", false, []string{"pano.js"}},
}

// contentFormScripts içerik formunun ağır istemci yükü.
var contentFormScripts = []string{
	"editor.js",      // TipTap + Yjs paketi (770 KB)
	"ai_editor.js",   // alan bazlı AI eylemleri
	"audio_panel.js", // röportaj sesi / transkripsiyon paneli
	"ses_player.js",  // dalga formu oynatıcı
}

// adminScriptsFor yola göre ek betikleri döndürür. Bilinmeyen yolda boş
// döner: güvenli taraf, ağır paketi yüklememektir.
func adminScriptsFor(path string) []string {
	if path == "" {
		return nil
	}
	for _, m := range adminPageScripts {
		if m.exact {
			if path == m.prefix {
				return m.scripts
			}
			continue
		}
		if matchAdminPrefix(path, m.prefix) {
			return m.scripts
		}
	}
	return nil
}

// matchAdminPrefix önek kaçağını engeller: önek ya yolun tamamına eşittir
// ya da hemen ardından "/" gelir.
func matchAdminPrefix(path, prefix string) bool {
	if strings.HasSuffix(prefix, "/") {
		// "/admin/icerikler/" → ardından en az bir karakter gelmeli
		return strings.HasPrefix(path, prefix) && len(path) > len(prefix)
	}
	if path == prefix {
		return true
	}
	return strings.HasPrefix(path, prefix+"/")
}
