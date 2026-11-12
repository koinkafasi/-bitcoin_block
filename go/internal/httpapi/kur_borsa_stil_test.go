package httpapi

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"theblock/site/internal/piyasa"
)

func kbTestKalemler() []piyasa.Kalem {
	return []piyasa.Kalem{
		{Anahtar: "usd", Kisa: "DOLAR", Ad: "Dolar", Deger: 49.8335, Degisim: 0.05, Ondalik: 4, Yon: 1},
		{Anahtar: "eur", Kisa: "EURO", Ad: "Euro", Deger: 55.4420, Degisim: -0.12, Ondalik: 4, Yon: -1},
		{Anahtar: "ceyrek-altin", Kisa: "ÇEYREK ALTIN", Ad: "Çeyrek Altın", Deger: 10558.61, Degisim: 0, Ondalik: 2, Yon: 0},
	}
}

func TestKbStilSecenekleriSinifBasilir(t *testing.T) {
	for _, stil := range []string{"referans", "sade", "kart", "dar"} {
		h := renderKbStil(t, kbTestKalemler(), stil, "noktali", "yok")
		assert.Contains(t, h, "pys-style-"+stil, "şablon sınıfı basılmalı: %s", stil)
	}
}

func TestKbOkTasarimlari(t *testing.T) {
	cases := map[string]struct {
		yukari string // yükseliş yol parçası
		asagi  string
	}{
		// Referans: ok başı + gövde, altında kesikli kuyruk (rect'ler).
		"noktali": {"M17.529,5.623l6.636,6.636", "M17.529,5.623l6.636,6.636"},
		"ucgen":   {"M12 5l7 12H5z", "M12 19l7-12H5z"},
		"ok":      {"M18 15l-6-6-6 6", "M6 9l6 6 6-6"},
		"uzun":    {"M6 18L18 6M8 6h10v10", "M6 6l12 12M18 8v10H8"},
	}
	for ad, c := range cases {
		h := renderKbStil(t, kbTestKalemler(), "referans", ad, "yok")
		assert.Contains(t, h, "pys-arrow-"+ad, "gösterge sınıfı: %s", ad)
		assert.Contains(t, h, c.yukari, "%s tasarımında yükseliş çizimi", ad)
	}
	// Noktalı varyantta düşüş 180° döndürülür (referansta da rotate(180)).
	h := renderKbStil(t, kbTestKalemler(), "referans", "noktali", "yok")
	assert.Contains(t, h, "pys-ico-down")
}

func TestKbOkYokSecenegi(t *testing.T) {
	h := renderKbStil(t, kbTestKalemler(), "sade", "yok", "yok")
	assert.Contains(t, h, "pys-arrow-yok")
	assert.NotContains(t, h, `class="pys-ico`, "gösterge yok seçildiğinde ikon basılmamalı")
	// Yüzde yine görünür
	assert.Contains(t, h, "%")
}

func TestKbYatayKalemdeGostergeYonuYok(t *testing.T) {
	// Değişim 0 → yön yok: kesikli dikey çizgi (referans icon-graphic-fixed).
	h := renderKbStil(t, kbTestKalemler(), "referans", "noktali", "yok")
	assert.Contains(t, h, "pys-ico-flat")
	assert.Contains(t, h, `<rect x="8.13" y="25.032" width="2.115" height="3.172"/>`)

	// Diğer tasarımlarda da düz çizgi kullanılır
	h = renderKbStil(t, kbTestKalemler(), "referans", "ucgen", "yok")
	assert.Contains(t, h, "M5 11h14v2H5z")
	h = renderKbStil(t, kbTestKalemler(), "referans", "ok", "yok")
	assert.Contains(t, h, "M5 12h14")
}

func TestKbStilAyarOkuma(t *testing.T) {
	assert.Equal(t, "referans", kbStilOf(nil), "ayar yoksa referans")
	assert.Equal(t, "kart", kbStilOf(map[string]any{"kb_style": "kart"}))
	assert.Equal(t, "referans", kbStilOf(map[string]any{"kb_style": "bilinmeyen"}))

	assert.Equal(t, "noktali", kbOkOf(nil), "varsayılan noktalı ok")
	assert.Equal(t, "uzun", kbOkOf(map[string]any{"kb_arrow": "uzun"}))
	assert.Equal(t, "noktali", kbOkOf(map[string]any{"kb_arrow": "yoksa"}))
}

func TestKbArrowForBilinmeyenStiliDuzeltir(t *testing.T) {
	a := kbArrowFor("bilinmeyen", 1)
	assert.Equal(t, "noktali", a.Stil)
	assert.Equal(t, 1, a.Yon)
}

func TestKbOnizlemeIleCanliAyniSablon(t *testing.T) {
	h := renderKbStil(t, kbTestKalemler(), "kart", "uzun", "zemin")
	// Önizleme bu HTML'i olduğu gibi panoya basar; sınıf ve yapı korunmalı.
	assert.Contains(t, h, `class="pys-strip pys-style-kart pys-arrow-uzun pys-bg-zemin"`)
	assert.Contains(t, h, `class="pys-item pys-up"`)
	assert.Contains(t, h, `class="pys-col"`)
}
