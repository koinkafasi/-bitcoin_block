package templkit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Tipografik kapak: kategori eşlemesi ve baş harf. İkisi de sessizce
// bozulup kapağı renksiz/harfsiz bırakabilecek yerler.

func TestCDKapakKategoriEslemesi(t *testing.T) {
	olc := func(kicker string) string { return cdKapakKategori(CDStory{Kicker: kicker}) }

	assert.Equal(t, "piyasa", olc("Piyasalar"))
	assert.Equal(t, "piyasa", olc("FİYATLAR"))
	assert.Equal(t, "politika", olc("Politika"))
	assert.Equal(t, "politika", olc("Düzenleme"))
	assert.Equal(t, "teknoloji", olc("Teknoloji"))
	assert.Equal(t, "analiz", olc("Haber Analizi"))
	assert.Equal(t, "analiz", olc("Araştırma"))
	assert.Equal(t, "finans", olc("Finans"))

	// Tanınmayan kategori renksiz kalmamalı, varsayılana düşmeli.
	assert.Equal(t, "genel", olc("Sponsorlu"))
	assert.Equal(t, "genel", olc(""))
}

func TestCDKapakHarf(t *testing.T) {
	assert.Equal(t, "B", cdKapakHarf("Bitcoin nedir?"))
	assert.Equal(t, "S", cdKapakHarf("SEC kararı"))
	// Noktalama ve boşlukla başlayan başlıkta ilk HARF alınır.
	assert.Equal(t, "E", cdKapakHarf("  “Ethereum” yükseldi"))
	// Rakamla başlayan başlık da geçerli.
	assert.Equal(t, "2", cdKapakHarf("2026 verileri"))
	// Türkçe büyük harf dönüşümü.
	assert.Equal(t, "Ç", cdKapakHarf("çöküş sonrası"))
	// Boş başlıkta marka harfi.
	assert.Equal(t, "B", cdKapakHarf(""))
	assert.Equal(t, "B", cdKapakHarf("—  "))
}
