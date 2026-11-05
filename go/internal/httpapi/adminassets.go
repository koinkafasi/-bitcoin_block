package httpapi

import "strings"

// Admin varlık yükleme kapsamı.
//
// Sorun: `admin_head` 3 CSS + 16 JS dosyasını koşulsuz yüklüyordu — her
// admin sayfasında 1.113.576 bayt. Bunun 770.939 baytı `editor.js`
// (TipTap + Yjs), 32.592 baytı `manset.js`. Kontrol paneli, kategori listesi
// ve ayarlar sayfaları içerik editörünü ve manşet motorunu boşuna indiriyordu.
//
// Çözüm: ağır betikler yalnız ihtiyaç duyan sayfalara. Eşleme tek bir saf
// fonksiyonda tutulur — 40 handler'a dokunmadan, tablo ile test edilebilir.
//
// Kapsam **muhafazakâr** seçildi: yalnızca bağlanma noktası tek bir sayfada
// olduğu doğrulanmış betikler daraltıldı. Küçük betikler (media.js, menu.js,
// header.js, footer.js, marka.js, kumeler.js, raporlar.js, ai-sohbet.js,
// tabcats.js — toplam ~41 KB) koşulsuz kalmaya devam eder; daraltmanın
// kazancı, yanlış daraltmanın riskini karşılamıyor.

// adminPageScripts sayfa yolu → o sayfaya özel betikler.
//
// Doğrulama (28 Ekim 2026): her betiğin `getElementById` hedefleri taranıp
// hangi `.templ` dosyasında tanımlı oldukları bulundu.
//   - editor.js, ai_editor.js, audio_panel.js → yalnız icerikform.templ
//   - manset.js                               → yalnız manset.templ
var adminPageScripts = []struct {
	// prefix yol öneki; tam eşleşme ya da ardından "/" gelmesi gerekir
	// ("/admin/icerikler-arsiv" içerik formu değildir).
	prefix string
	// exact true ise yalnız tam eşleşme sayılır. "/admin" için zorunlu:
	// önek olarak değerlendirilseydi /admin/dosyalar dahil bütün panel
	// sayfaları bu betikleri yüklerdi.
	exact   bool
	scripts []string
}{
	{"/admin/icerikler/yeni", false, contentFormScripts},
	{"/admin/icerikler/", false, contentFormScripts}, // düzenleme: /admin/icerikler/{id}
	{"/admin/manset-editor", false, []string{"manset.js"}},
	{"/admin/ses-ayarlari", false, []string{"ses_player.js"}},
	// Pano 2.0: kendi etkileşim betiği (ray katlama, segment kontrolü,
	// grafik balonu, sürüklenebilir yayın şeridi). Panonun giriş sayfası
	// olması nedeniyle İKİ yolda da gerekir; "/admin" tam eşleşmedir.
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
