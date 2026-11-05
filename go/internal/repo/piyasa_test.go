package repo

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"theblock/site/internal/piyasa"
)

// Kısmi sonuç önbelleği küçültmemeli: 28 Ekim 2026'da truncgil bir turda
// yanıt vermedi, sonuç 11 kalemden 4'üne düştü ve döviz/altın kalemleri
// sayfadan kayboldu.

func kalem(anahtar string, deger float64) piyasa.Kalem {
	return piyasa.Kalem{Anahtar: anahtar, Ad: anahtar, Deger: deger, Ondalik: 2}
}

func TestMergePiyasaEksikKalemleriTasir(t *testing.T) {
	simdi := time.Now()
	eski := piyasa.Sonuc{
		Guncel: simdi.Add(-2 * time.Minute),
		Kaynak: "truncgil+yahoo",
		Kalemler: []piyasa.Kalem{
			kalem("bist100", 11151), kalem("usd", 49.8), kalem("eur", 55.4),
			kalem("gram-altin", 6476), kalem("btc", 80900),
		},
	}
	// Yeni turda yalnız Yahoo kalemleri geldi.
	yeni := piyasa.Sonuc{
		Guncel:   simdi,
		Kaynak:   "yahoo",
		Kalemler: []piyasa.Kalem{kalem("bist100", 11160), kalem("btc", 81000)},
	}
	out := mergePiyasa(eski, yeni, simdi)

	assert.Len(t, out.Kalemler, 5, "eksik kalemler eskisinden taşınmalı")
	// Yeni değerler kazanır
	for _, k := range out.Kalemler {
		if k.Anahtar == "bist100" {
			assert.InDelta(t, 11160, k.Deger, 0.001, "yeni fiyat korunmalı")
		}
	}
	// Taşınan kalem korunur
	var usd float64
	for _, k := range out.Kalemler {
		if k.Anahtar == "usd" {
			usd = k.Deger
		}
	}
	assert.InDelta(t, 49.8, usd, 0.001)
	assert.Contains(t, out.Kaynak, "onbellek")
}

func TestMergePiyasaBayatOnbellegiTasimaz(t *testing.T) {
	simdi := time.Now()
	eski := piyasa.Sonuc{
		Guncel:   simdi.Add(-2 * time.Hour), // merge penceresinden eski
		Kalemler: []piyasa.Kalem{kalem("usd", 49.8)},
	}
	yeni := piyasa.Sonuc{Guncel: simdi, Kalemler: []piyasa.Kalem{kalem("btc", 81000)}}
	out := mergePiyasa(eski, yeni, simdi)
	assert.Len(t, out.Kalemler, 1, "bayat fiyat taşınmamalı")
	assert.Equal(t, "btc", out.Kalemler[0].Anahtar)
}

func TestMergePiyasaTamSonucDegismez(t *testing.T) {
	simdi := time.Now()
	var tam []piyasa.Kalem
	for _, anahtar := range piyasa.Anahtarlar() {
		tam = append(tam, kalem(anahtar, 1))
	}
	eski := piyasa.Sonuc{Guncel: simdi.Add(-time.Minute), Kalemler: tam}
	yeni := piyasa.Sonuc{Guncel: simdi, Kaynak: "truncgil+yahoo", Kalemler: tam}
	out := mergePiyasa(eski, yeni, simdi)
	assert.Len(t, out.Kalemler, len(tam))
	assert.Equal(t, "truncgil+yahoo", out.Kaynak, "tam sonuçta kaynak etiketi değişmez")
}

func TestMergePiyasaBosOnbellek(t *testing.T) {
	simdi := time.Now()
	yeni := piyasa.Sonuc{Guncel: simdi, Kalemler: []piyasa.Kalem{kalem("usd", 49.8)}}
	out := mergePiyasa(piyasa.Sonuc{}, yeni, simdi)
	assert.Equal(t, yeni, out)
}

func TestMergePiyasaSiraKorunur(t *testing.T) {
	simdi := time.Now()
	// Tanım sırası: bist100, usd, eur, ... — çıktı bu sıraya uymalı.
	eski := piyasa.Sonuc{
		Guncel:   simdi.Add(-time.Minute),
		Kalemler: []piyasa.Kalem{kalem("eur", 55), kalem("usd", 49), kalem("bist100", 11151)},
	}
	yeni := piyasa.Sonuc{Guncel: simdi, Kalemler: []piyasa.Kalem{kalem("eur", 56)}}
	out := mergePiyasa(eski, yeni, simdi)
	assert.Equal(t, []string{"bist100", "usd", "eur"}, []string{
		out.Kalemler[0].Anahtar, out.Kalemler[1].Anahtar, out.Kalemler[2].Anahtar,
	})
}
