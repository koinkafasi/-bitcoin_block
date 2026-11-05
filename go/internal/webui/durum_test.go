package webui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Boş durum, iskelet ve yapışkan tablo başlığı (Faz 163–164).
//
// Önce: 31 ayrı yerde elle yazılmış boş-durum cümlesi, 16 yerde düz
// "Yükleniyor…" metni, tablo başlığı kaydırmada kayboluyordu.

func templMetni(t *testing.T, ad string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("templkit", ad))
	require.NoError(t, err)
	return string(b)
}

// Bileşenler tanımlı olmalı.
func TestDurumBilesenleriTanimli(t *testing.T) {
	s := templMetni(t, "durum.templ")
	for _, ad := range []string{"templ BosDurum(", "templ IskeletTablo(", "templ IskeletKart(", "templ Yukleniyor("} {
		assert.Containsf(t, s, ad, "bileşen tanımlı olmalı: %s", ad)
	}
	// Erişilebilirlik: boş durum ve iskelet ekran okuyucuya duyurulmalı.
	assert.Contains(t, s, `role="status"`)
	assert.Contains(t, s, `aria-busy="true"`)
}

// Boş durum en az 8 ekranda kullanılmalı (elle yazılmış cümle yerine).
//
// Faz 174: tablo içi boş durumlar @TabloBos / @TabloBosMetin üzerinden
// gidiyor; onlar da BosDurum'u çağırıyor. Sayım ikisini birlikte yapmalı,
// yoksa "bileşen yaygınlaştı" ölçütü yanlış kırmızı verir.
func TestBosDurumYaygin(t *testing.T) {
	dizin, err := filepath.Glob(filepath.Join("templkit", "*.templ"))
	require.NoError(t, err)
	toplam := 0
	for _, f := range dizin {
		b, _ := os.ReadFile(f)
		s := string(b)
		toplam += strings.Count(s, "@BosDurum(")
		toplam += strings.Count(s, "@TabloBos(")
		toplam += strings.Count(s, "@TabloBosMetin(")
	}
	assert.GreaterOrEqual(t, toplam, 8, "en az 8 ekran BosDurum/TabloBos kullanmalı")
}

// Düz "Yükleniyor…" göstergesi kalmamalı.
func TestYukleniyorGostergesiBilesen(t *testing.T) {
	dizin, err := filepath.Glob(filepath.Join("templkit", "*.templ"))
	require.NoError(t, err)
	for _, f := range dizin {
		b, _ := os.ReadFile(f)
		s := string(b)
		if strings.Contains(s, `class="htmx-indicator small muted"`) {
			t.Errorf("%s: düz metin göstergesi kaldı; @Yukleniyor kullanılmalı", filepath.Base(f))
		}
	}
}

// İskelet en az 3 yerde kullanılmalı.
func TestIskeletKullanilir(t *testing.T) {
	dizin, err := filepath.Glob(filepath.Join("templkit", "*.templ"))
	require.NoError(t, err)
	toplam := 0
	for _, f := range dizin {
		b, _ := os.ReadFile(f)
		toplam += strings.Count(string(b), "@Iskelet")
	}
	assert.GreaterOrEqual(t, toplam, 3, "en az 3 yerde iskelet kullanılmalı")
}

// CSS sınıfları tanımlı ve hareket hassasiyetine saygılı olmalı.
func TestDurumCssSiniflari(t *testing.T) {
	css := cssMetni(t)
	for _, s := range []string{".bos-durum", ".iskelet-tablo", ".iskelet-cizgi", ".yukleniyor"} {
		assert.Containsf(t, css, s, "CSS sınıfı tanımlı olmalı: %s", s)
	}
	// İskelet animasyonu reduced-motion altında durmalı.
	i := strings.Index(css, ".iskelet-cizgi")
	require.Greater(t, i, 0)
	assert.Contains(t, css, "prefers-reduced-motion")
}

// Yapışkan başlık: zemin + sticky + z-index birlikte gerekli.
func TestTabloBasligiYapiskan(t *testing.T) {
	css := cssMetni(t)
	i := strings.Index(css, ".table thead th {")
	require.Greater(t, i, 0)
	son := strings.Index(css[i:], "}")
	blok := css[i : i+son]
	assert.Contains(t, blok, "position: sticky")
	assert.Contains(t, blok, "top: 0")
	assert.Contains(t, blok, "background: var(--card)",
		"saydam yapışkan başlıkta alttaki satırlar görünür")
	assert.Contains(t, blok, "z-index: 2")
}

// Uzun liste kendi içinde kaydırılmalı.
func TestIcerikListesiKendiIcindeKaydirilir(t *testing.T) {
	s := templMetni(t, "icerikler.templ")
	assert.Contains(t, s, `class="table-scroll"`)
	css := cssMetni(t)
	assert.Contains(t, css, ".table-scroll {")
	assert.Contains(t, css, "overflow: auto")
}

// Boş durum ve iskelet emoji taşımamalı.
func TestDurumBilesenleriEmojisiz(t *testing.T) {
	s := templMetni(t, "durum.templ")
	for _, e := range []string{"🎉", "✨", "🔎", "⚡"} {
		assert.NotContainsf(t, s, e, "durum bileşeninde emoji olmamalı: %s", e)
	}
}
