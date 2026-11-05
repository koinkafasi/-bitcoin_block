package templkit

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/webui"
)

// CoinDesk tasarım sistemi kalıcı kapısı.
//
// NEDEN: tasarım kararları görsel doğrulama gerektirir ve sessizce
// bozulabilir (birisi gölge ekler, ritmi değiştirir, token'ı siler).
// docs/COINDESK-SPEC.md §2'deki kurallar burada KAPIDIR.
func cdCSS(t *testing.T) string {
	t.Helper()
	b, err := webui.FS.ReadFile("static/cd.css")
	require.NoError(t, err, "static/cd.css okunamadı")
	return string(b)
}

// TestCDTasarimTokenlari: ölçülen renk ve ritim değerleri yerinde.
func TestCDTasarimTokenlari(t *testing.T) {
	css := cdCSS(t)
	for _, token := range []string{
		"--cd-bg: #ffffff",
		"--cd-fg: #000000",
		"--cd-link: #4d90fe",
		"--cd-up: #68b631",
		"--cd-down: #ff0062",
		"--cd-warn: #ffd200",
		"--cd-border: #e9e9e9",
		"--cd-section-gap: 40px",
		"proxima-nova",
	} {
		assert.Contains(t, css, token, "ölçülen token eksik: %s", token)
	}
}

// TestCDTasarimKurallari: CoinDesk görsel dili — GÖLGE YOK, ayırma çizgiyle.
func TestCDTasarimKurallari(t *testing.T) {
	css := cdCSS(t)
	// box-shadow yalnız "yok" olabilir; gerçek gölge CoinDesk'te yok.
	for _, satir := range strings.Split(css, "\n") {
		s := strings.TrimSpace(satir)
		if strings.HasPrefix(s, "box-shadow:") && !strings.HasPrefix(s, "box-shadow: none") {
			t.Errorf("CoinDesk dilinde gölge YOK: %s", s)
		}
	}
	// Köşe yarıçapı 2 px (measured).
	assert.Contains(t, css, "--cd-radius: 2px")
	// Dikey ritim: modüller arası 40 px.
	assert.Contains(t, css, "--cd-section-gap: 40px")
	// Erişilebilirlik: hareket azaltma bloğu zorunlu.
	assert.Contains(t, css, "prefers-reduced-motion")
	// Karanlık tema medya sorgusuyla gelir (CoinDesk açık tema varsayılan).
	assert.Contains(t, css, "prefers-color-scheme: dark")
}

// TestCDButce: CSS bütçesi. Tasarım sistemi kalınlaştıkça dosya büyür;
// bütçe aşıldığında bilinçli bir güncelleme gerekir (sessiz kabul yok).
func TestCDButce(t *testing.T) {
	css := cdCSS(t)
	// 16 KB hedef; 20 KB uyarı eşiği.
	assert.Less(t, len(css), 20*1024, "cd.css bütçe aşıldı: %d bayt", len(css))
}

// TestCDShellSiniflari: şablon ile CSS aynı sınıflarda buluşuyor.
func TestCDShellSiniflari(t *testing.T) {
	css := cdCSS(t)
	for _, sinif := range []string{
		"cd-ticker", "cd-topbar", "cd-footer", "cd-logo",
		"cd-card", "cd-list", "cd-article", "cd-source", "cd-ad",
		"cd-up", "cd-down", "cd-section-head",
	} {
		assert.Contains(t, css, "."+sinif, "CSS sınıfı eksik: %s", sinif)
	}
}
