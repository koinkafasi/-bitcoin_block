package webui

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Sayfa Tasarımcısı yerleşimi ve teması (Faz 182).
//
// NEDEN: sayfa blok önizlemeleri için public.css yüklüyor. O dosya `.card`,
// `.card-pad`, `.btn`, `.muted`, `.input`, `.table` ve `--accent` tanımlıyor;
// admin sürümlerini eziyordu. Ölçülen sonuçlar:
//
//	.card-pad{display:flex;flex-direction:column}  -> her panel 171px
//	  (olması gereken ~82px) ve başlıkla düğmeler alt alta
//	.card{overflow:hidden}                          -> 1024px'te alanlar kırpılıyor
//	:root{--accent:var(--hp-tema)}                  -> marka rengi #fa0000 yerine
//	  #d10015; koyu temada marka yazısı 3.3 kontrast
//
// Ayrıca panelde 38 sabit renk (koyu temada okunmayan) token'a çevrildi.

func tasarimciKaynak(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("templates", "site_builder.html"))
	require.NoError(t, err)
	return string(b)
}

// stilBloku: sayfanın <style> ... </style> gövdesi.
func stilBloku(t *testing.T, s string) string {
	t.Helper()
	bas := strings.Index(s, "<style>")
	son := strings.Index(s, "</style>")
	require.Greater(t, bas, 0)
	require.Greater(t, son, bas)
	return s[bas:son]
}

// TestTasarimciKartCakismasi: public.css'in `.card`/`.card-pad` kuralları
// admin panelinde etkisizleştirilmeli.
func TestTasarimciKartCakismasi(t *testing.T) {
	stil := stilBloku(t, tasarimciKaynak(t))

	// Panel kartları tekrar blok akışına dönmeli ve kırpılmamalı.
	assert.Contains(t, stil, ".admin-body > .card-pad",
		"public.css .card-pad çakışması için geri alma kuralı olmalı")
	assert.Contains(t, stil, "flex: none",
		"public.css `flex:1` geri alınmalı")
	assert.Contains(t, stil, "overflow: visible",
		"public.css `.card{overflow:hidden}` alanları kırpıyordu")

	// Marka rengi geri yüklenmeli.
	assert.Contains(t, stil, "--accent: var(--tb-accent",
		"public.css --accent'i eziyor; marka rengi geri yüklenmeli")
}

// TestTasarimciPanelleriKatlanabilir: ikincil paneller <details> olmalı.
// Aksi hâlde tasarımcı listesi sayfada ~1155px aşağıda başlıyordu.
func TestTasarimciPanelleriKatlanabilir(t *testing.T) {
	s := tasarimciKaynak(t)

	for _, id := range []string{"plPanel", "bpTokens"} {
		assert.Regexpf(t, `<details[^>]*id="`+id+`"`, s,
			"%s <details> olmalı (katlanabilir)", id)
	}

	// Kapalı başlamalı: `open` özniteliği olmamalı.
	for _, id := range []string{"plPanel", "bpTokens"} {
		acilis := regexp.MustCompile(`<details[^>]*id="` + id + `"[^>]*>`).FindString(s)
		require.NotEmptyf(t, acilis, "%s açılış etiketi bulunamadı", id)
		assert.NotContainsf(t, acilis, " open", "%s varsayılan olarak kapalı olmalı", id)
	}

	// "varsayılana dön" düğmesi artık gövdede: <summary> içinde olsa
	// tıklama paneli de açıp kapatırdı.
	i := strings.Index(s, `id="plReset"`)
	require.Greater(t, i, 0)
	onceki := s[maxInt(0, i-600):i]
	assert.False(t, strings.Contains(onceki, "<summary"),
		"plReset <summary> içinde olmamalı (tıklama paneli açıp kapatır)")
}

// TestTasarimciSabitRenkYok: koyu temada okunmayan açık-tema sabitleri
// kalmamalı. Yalnız kasıtlı renk örnekleri (mini swatch'lar) serbest.
func TestTasarimciSabitRenkYok(t *testing.T) {
	stil := stilBloku(t, tasarimciKaynak(t))

	// Bu değerler metin/zemin olarak sabitlendiğinde koyu temada okunmuyordu.
	yasak := []struct{ deger, aciklama string }{
		{"#5b6180", "yardımcı metin (koyu temada koyu kalıyordu)"},
		{"#8b90a5", "yardımcı metin"},
		{"#4a5070", "yer tutucu metin"},
		{"#b3b8cc", "yer tutucu metin"},
		{"rgba(255,255,255,.35)", "yarı saydam beyaz zemin"},
		{"rgba(255,255,255,.55)", "yarı saydam beyaz zemin"},
	}
	for _, y := range yasak {
		assert.NotContainsf(t, stil, y.deger,
			"sabit renk kalmamalı (%s): %s", y.deger, y.aciklama)
	}

	// Bölge cipleri: beyaz yazı eşiği geçmeli.
	assert.NotContains(t, stil, "background:#22c55e",
		"z-ana cipi beyaz yazıyla 2.3 kontrast veriyordu")
	assert.NotContains(t, stil, "background:#0ea5e9",
		"z-ust cipi beyaz yazıyla 2.8 kontrast veriyordu")
	assert.NotContains(t, stil, "background:#a855f7",
		"z-genis cipi beyaz yazıyla 4.0 kontrast veriyordu")
}

// TestTasarimciAlanTasmasi: .ly-grid daralabilmeli, .bld-field sarabilmeli.
func TestTasarimciAlanTasmasi(t *testing.T) {
	stil := stilBloku(t, tasarimciKaynak(t))

	assert.Contains(t, stil, "minmax(300px, 1fr)",
		".ly-grid erken tek sütuna inmeli (1024px'te alanlar taşıyordu)")
	assert.Contains(t, stil, ".bld-field { flex-wrap: wrap; }",
		"etiket + denetim satırı gerektiğinde sarabilmeli")
	assert.Contains(t, stil, "min-width: 0",
		"ızgara/flex çocukları küçülebilmeli")
}
