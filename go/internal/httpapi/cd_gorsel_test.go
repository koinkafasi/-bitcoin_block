package httpapi

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"theblock/site/internal/repo"
)

// Kaynak görsel ucu (/kg/{hash}) güvenlik kapısı.
//
// NEDEN TEST: bu uç dışarıdan gelen bir adresi sunucuya indirtiyor.
// Yanlış yazılmış bir hash doğrulaması yol aşımına, SVG kabulü ise
// görselin içine gömülü script'e kapı açar.

// TestGecerliKaynakGorselHash: uzunluk VE karakter birlikte doğrulanır.
//
// Saldırı denemeleri: "../" (yol aşımı), kodlanmış "..", büyük harf
// (dosya sistemi büyük/küçük harf duyarsız olabilir → iki hash aynı
// dosyaya düşer), boşluk, uzunluk ihlali.
func TestGecerliKaynakGorselHash(t *testing.T) {
	kabul := []string{
		"80fe8a843afddb33661ff0f019b8ac11",
		"0123456789abcdef0123456789abcdef",
	}
	for _, s := range kabul {
		assert.True(t, gecerliKaynakGorselHash(s), "kabul edilmeli: %q", s)
	}
	ret := []string{
		"",
		"80FE8A843AFDDB33661FF0F019B8AC11", // büyük harf
		"../../etc/passwd",
		"80fe8a843afddb33661ff0f019b8ac1",   // 31 karakter
		"80fe8a843afddb33661ff0f019b8ac111", // 33 karakter
		"80fe8a843afddb33661ff0f019b8ac1z",  // hex dışı
		"80fe8a84/3afddb33661ff0f019b8ac1",
		"80fe8a843afddb33661ff0f019b8ac1 ",
	}
	for _, s := range ret {
		assert.False(t, gecerliKaynakGorselHash(s), "reddedilmeli: %q", s)
	}
}

// TestKaynakGorselYol: web yolu hash'ten üretilir, uzantı taşımaz.
func TestKaynakGorselYol(t *testing.T) {
	assert.Equal(t, "/kg/abc123", repo.KaynakGorselYol("abc123"))
	assert.Equal(t, "", repo.KaynakGorselYol(""), "boş hash yol üretmez")
}

// TestKaynakGorselDiskYoluDagilimi: ilk 2 karakter alt dizin olur.
//
// NEDEN: tek dizinde on binlerce dosya bazı dosya sistemlerinde yavaşlar;
// ayrıca yedekleme/rsync dağıtımı tek dizinde kilitlenir.
func TestKaynakGorselDiskYoluDagilimi(t *testing.T) {
	yol := kaynakGorselDiskYolu("80fe8a843afddb33661ff0f019b8ac11")
	assert.Contains(t, yol, "/kaynak/80/")
	assert.Contains(t, yol, "80fe8a843afddb33661ff0f019b8ac11")
}
