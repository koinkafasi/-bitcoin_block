package youtube

// Kapak görseli seçimi testleri (11 Kasım 2026).
//
// NEDEN: Kullanıcı "shorts küçük resimleri çok kötü kalitede" dedi. Kök
// neden ölçümle bulundu: short kayıtlarında `hqdefault.jpg` (480x360 YATAY)
// kullanılıyordu; 9:16 dikey kartta bunun yalnız ~200x360'lık ortası
// görünüp kalan alan büyütülüyordu. Bu testler doğru varyantı sabitler.
//
// Ağ GEREKMEZ: yalnız URL kuralları.

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

func TestGorselYedekHerZamanVarOlan(t *testing.T) {
	// maxresdefault 404 verirse hqdefault'a düşülür (her videoda var).
	assert.Equal(t, "https://i.ytimg.com/vi/8oVvSLjkMv8/hqdefault.jpg",
		GorselYedek("8oVvSLjkMv8", "video"))
	assert.Equal(t, "https://i.ytimg.com/vi/iPG7pIB-P1c/hqdefault.jpg",
		GorselYedek("iPG7pIB-P1c", "short"))
	assert.Empty(t, GorselYedek("", "video"))
}

func TestBuyukGorselURL(t *testing.T) {
	assert.Equal(t, "https://i.ytimg.com/vi/8oVvSLjkMv8/maxresdefault.jpg",
		BuyukGorselURL("8oVvSLjkMv8"))
	assert.Empty(t, BuyukGorselURL(""))
}
