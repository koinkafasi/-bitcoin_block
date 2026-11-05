package webui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Faz 195 — çekmece (hamburger menü) renkleri.
//
// KULLANICI BİLDİRİMİ: "hamburger menüde yanda açılan bazı linkler renkleri
// düzgün değil."
//
// ÖLÇÜM (tarayıcı, canlı): `.hp-drawer-group` rengi `--hp-footer-baslik`
// idi — ALT BİLGİ token'ı. Çekmecede bu token tanımsız kalıp kök değere
// düşüyordu (#a1a1aa):
//
//	açık tema  panel #ffffff → kontrast 2.56:1   ← okunmuyor (AA: 4.5)
//	koyu tema  panel #0c1021 → kontrast 7.37:1   (kaza eseri iyi)
//
// Yani "Kategoriler / İçerikler / Hizmetler / Tercihler" başlıkları açık
// temada neredeyse görünmüyordu. Düzeltme: çekmece kendi token'larını taşır.

// blokToken verilen kural bloğundaki bir değişkenin değerini okur.
func blokToken(t *testing.T, blok, ad string) string {
	t.Helper()
	for _, satir := range strings.Split(blok, "\n") {
		satir = strings.TrimSpace(satir)
		if !strings.HasPrefix(satir, ad+":") {
			continue
		}
		deger := strings.TrimPrefix(satir, ad+":")
		if i := strings.IndexAny(deger, ";}"); i >= 0 {
			deger = deger[:i]
		}
		return strings.TrimSpace(deger)
	}
	return ""
}

// publicCSS test için public.css okur (cssMetni admin-pano.css okuyor).
func publicCSS(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("static", "public.css"))
	require.NoError(t, err, "public.css okunamadı")
	return string(b)
}

func TestCekmeceGrupEtiketiFooterTokeniKullanmaz(t *testing.T) {
	css := publicCSS(t)

	i := strings.Index(css, ".hp-drawer-group {")
	require.Greater(t, i, 0, ".hp-drawer-group kuralı yok")
	kural := css[i:]
	kural = kural[:strings.Index(kural, "}")]

	assert.NotContains(t, kural, "--hp-footer-baslik",
		"alt bilgi token'ı çekmecede tanımsız kalıp okunmaz gri üretiyor (2.56:1)")
	assert.Contains(t, kural, "var(--hp-drawer-etiket)")
}

func TestCekmeceTokenlariKontrastGecer(t *testing.T) {
	css := publicCSS(t)
	// Açık tema: kök tanımlar.
	acikZemin := blokToken(t, publicBlok(t, css, ":root"), "--hp-surface")
	acikEtiket := blokToken(t, publicBlok(t, css, ":root"), "--hp-drawer-etiket")
	acikAlt := blokToken(t, publicBlok(t, css, ":root"), "--hp-drawer-alt")
	require.NotEmpty(t, acikEtiket, "--hp-drawer-etiket yok")
	require.NotEmpty(t, acikAlt, "--hp-drawer-alt yok")

	k := oran(renkCoz(t, acikEtiket), renkCoz(t, acikZemin))
	assert.GreaterOrEqual(t, k, 4.5,
		"açık temada grup etiketi kontrastı %.2f (AA 4.5) — eski ölçüm 2.56", k)

	k2 := oran(renkCoz(t, acikAlt), renkCoz(t, acikZemin))
	assert.GreaterOrEqual(t, k2, 4.5, "açık temada alt kategori kontrastı %.2f", k2)

	// Koyu tema: kendi bloğundaki tanımlar. Panel zemini sitenin koyu
	// paletinden geliyor (tasarım CSS'i) — ölçülen değer kullanılır.
	koyu := publicBlok(t, css, `html[data-theme="dark"]`)
	koyuEtiket := blokToken(t, koyu, "--hp-drawer-etiket")
	koyuAlt := blokToken(t, koyu, "--hp-drawer-alt")
	require.NotEmpty(t, koyuEtiket, "koyu temada --hp-drawer-etiket yok")
	require.NotEmpty(t, koyuAlt, "koyu temada --hp-drawer-alt yok")

	koyuZemin := "#0c1021"
	k3 := oran(renkCoz(t, koyuEtiket), renkCoz(t, koyuZemin))
	assert.GreaterOrEqual(t, k3, 4.5, "koyu temada grup etiketi kontrastı %.2f", k3)

	k4 := oran(renkCoz(t, koyuAlt), renkCoz(t, koyuZemin))
	assert.GreaterOrEqual(t, k4, 4.5, "koyu temada alt kategori kontrastı %.2f", k4)
}

func TestCekmeceOdakVeDokunmaHedefi(t *testing.T) {
	css := publicCSS(t)

	// Klavye odağı görünür olmalı (önce hiç odak stili yoktu).
	assert.Contains(t, css, ".hp-drawer a:focus-visible")
	assert.Contains(t, css, "outline: 2px solid var(--hp-tema)")

	// Mobilde 34px altı hedefler yanlış tıklanıyor.
	assert.Contains(t, css, ".hp-drawer-grid > a { min-height: 34px; }")
	assert.Contains(t, css, ".hp-drawer-list li a { min-height: 34px; }")

	// Geçerli sayfa vurgusu.
	assert.Contains(t, css, `.hp-drawer a[aria-current="page"]`)
}

// publicBlok verilen CSS metninde `seçici {` ile başlayan bloğun gövdesini
// döndürür. token_test.go'daki blokBul admin-pano.css'e sabit; çekmece
// token'ları public.css'te olduğu için ayrı bir yardımcı gerekiyor.
func publicBlok(t *testing.T, css, secici string) string {
	t.Helper()
	i := strings.Index(css, "\n"+secici+" {")
	require.Greaterf(t, i, 0, "kural bulunamadı: %s", secici)
	bas := i + len(secici) + 2
	son := strings.Index(css[bas:], "}")
	require.Greater(t, son, 0)
	return css[bas : bas+son]
}
