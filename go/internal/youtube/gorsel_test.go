package youtube

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGorselURLShortDikeyVaryant(t *testing.T) {
	// Short → oar2.jpg (1080x1920, 9:16). Ölçüldü: gerçek bir short
	// kimliğinde 1080x1920 döndü, hqdefault ise 480x360.
	u := GorselURL("iPG7pIB-P1c", "short")
	assert.Equal(t, "https://i.ytimg.com/vi/iPG7pIB-P1c/oar2.jpg", u)
	assert.True(t, strings.HasSuffix(u, "/oar2.jpg"))
}

func TestGorselURLVideoYuksekCozunurluk(t *testing.T) {
	// Yatay video → maxresdefault (1280x720), hqdefault (480x360) DEĞİL.
	u := GorselURL("8oVvSLjkMv8", "video")
	assert.Equal(t, "https://i.ytimg.com/vi/8oVvSLjkMv8/maxresdefault.jpg", u)
	assert.NotContains(t, u, "hqdefault")
}

func TestGorselURLBosKimlik(t *testing.T) {
	assert.Empty(t, GorselURL("", "video"))
	assert.Empty(t, GorselURL("", "short"))
}

func TestGorselSrcsetYalnizYatayda(t *testing.T) {
	// Yatayda üç aday (480/640/1280) — yan kolon 142 px'lik satır için
	// 1280'lik dosyayı indirmek israftı.
	ss := GorselSrcset("8oVvSLjkMv8", "video")
	assert.Contains(t, ss, "hqdefault.jpg 480w")
	assert.Contains(t, ss, "sddefault.jpg 640w")
	assert.Contains(t, ss, "maxresdefault.jpg 1280w")

	// Shorts'ta ölçekli sürüm YOK: srcset boş döner, tek dosya kullanılır.
	assert.Empty(t, GorselSrcset("iPG7pIB-P1c", "short"))
	assert.Empty(t, GorselSrcset("", "video"))
}

func TestGorselYedekZinciriShortMaxresIceriyor(t *testing.T) {
	ad := strings.Split(GorselYedekZincir("iPG7pIB-P1c", "short"), ",")
	if len(ad) != 2 {
		t.Fatalf("short zinciri 2 aday olmalı, %d geldi", len(ad))
	}
	// KRİTİK: ilk yedek maxresdefault (1280×720) olmalı, hqdefault DEĞİL.
	assert.Contains(t, ad[0], "maxresdefault.jpg",
		"oar2 yoksa 1280×720'ye düşülmeli; hqdefault 480×360 → bulanık")
	// Son çare: her videoda var olan hqdefault (kart asla kırık kalmaz).
	assert.Contains(t, ad[1], "hqdefault.jpg")
}

func TestGorselYedekZinciriYatay(t *testing.T) {
	ad := strings.Split(GorselYedekZincir("8oVvSLjkMv8", "video"), ",")
	if len(ad) != 2 {
		t.Fatalf("2 aday bekleniyordu, %d geldi", len(ad))
	}
	assert.Contains(t, ad[0], "sddefault.jpg", "maxres yoksa 640×480")
	assert.Contains(t, ad[1], "hqdefault.jpg")
	assert.NotContains(t, ad[0], "maxresdefault", "zaten maxres src'de")
}

func TestGorselYedekZinciriBosKimlik(t *testing.T) {
	// Eski hata: "…/vi//hqdefault.jpg" gibi bozuk adres basılıyordu.
	assert.Empty(t, GorselYedekZincir("", "video"))
	assert.Empty(t, GorselYedekZincir("", "short"))
}

func TestBuyukGorselURL(t *testing.T) {
	assert.Equal(t, "https://i.ytimg.com/vi/8oVvSLjkMv8/maxresdefault.jpg",
		BuyukGorselURL("8oVvSLjkMv8"))
	assert.Empty(t, BuyukGorselURL(""))
}
