package repo

import (
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFiyatBirlestirCanliDegerleriUygular(t *testing.T) {
	meta := []FiyatSatiri{
		{Rank: 1, Sym: "BTC", Name: "Bitcoin", Price: 70000, Change: 1, HasData: true},
		{Rank: 2, Sym: "ETH", Name: "Ethereum", Price: 2000, Change: 2, HasData: true},
		{Rank: 3, Sym: "USDT", Name: "Tether", Price: 1, Change: 0, HasData: true},
	}
	canli := map[string]binanceFiyat{
		"BTC": {Fiyat: 74000, Degisim: -1.5},
		"ETH": {Fiyat: 2066, Degisim: -2.4},
		// USDT yok: Binance'te USDTUSDT paritesi yok.
	}
	out := fiyatBirlestir(meta, canli)
	require.Len(t, out, 3)

	assert.Equal(t, 74000.0, out[0].Price, "BTC canlı fiyatı almalı")
	assert.Equal(t, -1.5, out[0].Change)
	assert.Equal(t, 2066.0, out[1].Price)

	// USDT meta değerini KORUMALI (sıfır yazmamalı).
	assert.Equal(t, 1.0, out[2].Price, "paritesi olmayan sembol 0'a düşmemeli")
	assert.Equal(t, "Tether", out[2].Name)

	// Sıra korunur (piyasa değeri sırası CoinGecko'dan).
	assert.Equal(t, "BTC", out[0].Sym)
	assert.Equal(t, 1, out[0].Rank)
}

func TestFiyatBirlestirSifirFiyatYazmaz(t *testing.T) {
	meta := []FiyatSatiri{{Rank: 1, Sym: "BTC", Price: 70000, HasData: true}}
	// Binance 0 döndürdü (bozuk yanıt): meta değeri kalmalı.
	out := fiyatBirlestir(meta, map[string]binanceFiyat{"BTC": {Fiyat: 0, Degisim: 5}})
	assert.Equal(t, 70000.0, out[0].Price)
}

func TestGirisGunlukKapanisGunSonunuAlir(t *testing.T) {
	// 3 gün × 24 saat = 72 nokta. Her günün SON değeri kapanıştır.
	seri := make([]float64, 72)
	for g := 0; g < 3; g++ {
		for s := 0; s < 24; s++ {
			// g. günün saatleri: g*100 + saat
			seri[g*24+s] = float64(g*100 + s)
		}
	}
	out := girisGunlukKapanis(seri, 24)
	require.Len(t, out, 3)
	// g0 son saat 23 → 23; g1 son saat 23 → 123; g2 → 223
	assert.Equal(t, []float64{23, 123, 223}, out)

	// Yetersiz veri: nil (uydurma yok).
	assert.Nil(t, girisGunlukKapanis(seri[:10], 24))
	assert.Nil(t, girisGunlukKapanis(nil, 24))
	assert.Nil(t, girisGunlukKapanis(seri, 0))
}

func TestCdRSIBilinenDegerler(t *testing.T) {
	// Sürekli artan seri → RSI 100 (hiç kayıp yok).
	artan := make([]float64, 20)
	for i := range artan {
		artan[i] = float64(100 + i)
	}
	r, ok := cdRSI(artan, 14)
	require.True(t, ok)
	assert.Equal(t, 100.0, r)

	// Sürekli azalan → RSI 0'a çok yakın (kazanç yok).
	azalan := make([]float64, 20)
	for i := range azalan {
		azalan[i] = float64(100 - i)
	}
	r, ok = cdRSI(azalan, 14)
	require.True(t, ok)
	assert.Equal(t, 0.0, r)

	// Düz seri → 50 (sıfıra bölme yok).
	duz := make([]float64, 20)
	for i := range duz {
		duz[i] = 50
	}
	r, ok = cdRSI(duz, 14)
	require.True(t, ok)
	assert.Equal(t, 50.0, r)

	// Yetersiz veri → ok=false.
	_, ok = cdRSI([]float64{1, 2, 3}, 14)
	assert.False(t, ok)
}

func TestCdSparklineGecerliYolUretir(t *testing.T) {
	yol, alan := cdSparkline([]float64{10, 12, 11, 15, 14, 18}, 320, 120, 8)
	require.NotEmpty(t, yol)
	require.NotEmpty(t, alan)
	assert.True(t, strings.HasPrefix(yol, "M8.00"), "sol dolgu 8 px: %s", yol)
	assert.NotContains(t, yol, "NaN")
	assert.NotContains(t, yol, "+Inf")
	assert.True(t, strings.HasSuffix(alan, "Z"), "dolgu yolu kapalı olmalı")

	// Baklava son nokta sağ üstte olmalı (en yüksek değer → küçük y).
	assert.Contains(t, yol, "312.00 8.00")

	// Tek değer / boş → boş yol (şablon grafiği hiç basmaz).
	_, _ = cdSparkline([]float64{5}, 320, 120, 8)
	yol, alan = cdSparkline([]float64{5}, 320, 120, 8)
	assert.Empty(t, yol)
	assert.Empty(t, alan)

	// Eşit seri: düz çizgi, NaN yok.
	yol, _ = cdSparkline([]float64{7, 7, 7, 7}, 320, 120, 8)
	assert.NotEmpty(t, yol)
	assert.NotContains(t, yol, "NaN")
	// Hepsi ortada (oran 0.5) → y = 8 + 0.5*104 = 60.
	assert.True(t, strings.Contains(yol, "60.00"), "%s", yol)
	_ = math.Pi
}
