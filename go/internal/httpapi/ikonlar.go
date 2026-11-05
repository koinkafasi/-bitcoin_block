package httpapi

import (
	"html/template"
	"path"
	"strings"
	"sync"

	"theblock/site/internal/webui"
)

// ---- İkon kütüphanesi (Faz 153) ----
//
// Sprite: static/vendor/tabler-icons.svg (Tabler, MIT) — panelde tam set,
// public'te yalnız çekirdek (static/vendor/ikon-cekirdek.svg).
//
// Animasyonlu "yükleniyor" SVG'leri: static/vendor/spinner/*.svg
// (SVG Spinners, MIT). Animasyon dosyanın içinde tanımlı; JS gerekmez.
// <img> ile kullanılamaz çünkü rengi temadan alması gerekiyor — bu yüzden
// satır içi basılıyor.

var (
	spinnerMu    sync.RWMutex
	spinnerCache = map[string]template.HTML{}
	spinnerIsim  = []string{
		"ring-resize", "90-ring", "180-ring", "270-ring", "3-dots-bounce",
		"3-dots-fade", "bars-scale", "pulse-ring", "pulse-2", "eclipse", "dot-revolve",
	}
)

// ikonSpinnerSVG spinner dosyasını satır içi basar (bir kez okunur).
//
// Bilinmeyen ad -> boş. Sessizce boş dönmek yerine "yok" demek daha dürüst
// olurdu ama şablonda gürültü çıkarır; bilinmeyen ad zaten testte yakalanıyor
// (TestIkonSpinnerAdlari).
func ikonSpinnerSVG(ad string) template.HTML {
	ad = strings.TrimSpace(ad)
	if ad == "" {
		ad = "ring-resize"
	}
	spinnerMu.RLock()
	if v, ok := spinnerCache[ad]; ok {
		spinnerMu.RUnlock()
		return v
	}
	spinnerMu.RUnlock()

	b, err := webui.FS.ReadFile(path.Join("static/vendor/spinner", ad+".svg"))
	if err != nil {
		return ""
	}
	// <style> blokları yalnız bu dosyaya ait sınıf adları kullanıyor
	// (spinner_XXXX); sayfada birden çok spinner olsa da çakışmaz.
	v := template.HTML(`<span class="ikon-spin-svg" role="status" aria-label="Yükleniyor">` + string(b) + `</span>`)

	spinnerMu.Lock()
	spinnerCache[ad] = v
	spinnerMu.Unlock()
	return v
}

// ikonSpinnerAdlari galeri ve testler için mevcut spinner listesi.
func ikonSpinnerAdlari() []string { return spinnerIsim }

// ikonAdlari sprite manifestinden ikon adlarını okur (galeri için).
//
// Manifest (static/vendor/ikonlar.txt) üretici tarafından yazılıyor;
// sprite'ı ayrıştırmaya gerek yok ve liste sıralı geliyor.
func ikonAdlari() []string {
	b, err := webui.FS.ReadFile("static/vendor/ikonlar.txt")
	if err != nil {
		return nil
	}
	var out []string
	for _, s := range strings.Split(string(b), "\n") {
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}
