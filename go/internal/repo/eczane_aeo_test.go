package repo

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const aeoFikstur = "testdata/aeo-nobet.html"

func TestEczaneAEOCoz(t *testing.T) {
	ham, err := os.ReadFile(aeoFikstur)
	require.NoError(t, err, "fikstür okunamadı")

	list := eczaneAEOCoz(string(ham))
	require.Len(t, list, 3, "fikstürde 3 kayıt var")

	// Ölçülen gerçek değerler.
	assert.Equal(t, "SUN ECZANESİ", list[0].Ad)
	assert.Equal(t, "Çankaya", list[0].Ilce)
	assert.Equal(t, "0 312 213 16 78", list[0].Tel,
		"kaynak 03122131678 veriyor; baştaki 0 atılıp biçimlendirilmeli")
	// Adres ve TARİF <br> ile ayrılıyor; ikisi birleşmeli.
	assert.Contains(t, list[0].Adres, "12.Cadde No:18/B Emek")
	assert.Contains(t, list[0].Adres, "Emek Cami Karşısı")
	// Telefon adrese KARIŞMAMALI (<span> kesiliyor).
	assert.NotContains(t, list[0].Adres, "03122131678")
	// Etiket kalıntısı olmamalı.
	assert.NotContains(t, list[0].Adres, "<")

	assert.Equal(t, "GAZİ ECZANESİ", list[1].Ad)
	assert.Equal(t, "Yenimahalle", list[1].Ilce)
	assert.Equal(t, "0 312 223 43 86", list[1].Tel)

	assert.Equal(t, "ERTÜRK ECZANESİ", list[2].Ad)
	assert.Equal(t, "Yenimahalle", list[2].Ilce)

	// Hiçbiri eksik kalmamalı: ad/adres/telefon üçü de dolu olmalı.
	for i, e := range list {
		assert.NotEmpty(t, e.Ad, "kayıt %d: ad", i)
		assert.NotEmpty(t, e.Adres, "kayıt %d: adres", i)
		assert.NotEmpty(t, e.Tel, "kayıt %d: telefon", i)
		assert.NotEmpty(t, e.Ilce, "kayıt %d: ilçe", i)
	}
}

// Bozuk/boş girdide ÇÖKMEMELİ ve uydurma kayıt üretmemeli.
func TestEczaneAEOCozBosGirdi(t *testing.T) {
	assert.Empty(t, eczaneAEOCoz(""))
	assert.Empty(t, eczaneAEOCoz("<html><body>hiç kayıt yok</body></html>"))
	// Kayıt sınırı var ama ad yok: atlanmalı (adsız eczane gösterilmez).
	assert.Empty(t, eczaneAEOCoz(`<div class="inline-box" data-district="Çankaya"><p>adres</p></div>`))
}

// Ankara tabloda bağlı olmalı; yoksa il sessizce boş kalır.
func TestEczaneOdaKaynaklariAnkara(t *testing.T) {
	_, ok := eczaneOdaKaynaklari[6]
	assert.True(t, ok, "Ankara (plaka 6) resmi oda adaptörü bağlı değil")
}

// Uç adresi: `www.` zorunlu — çıplak alan adı 301 veriyor (ölçüldü).
func TestAEOURL(t *testing.T) {
	u := aeoURL("2026-11-05")
	assert.Equal(t, "https://www.aeo.org.tr/getPharmacies/2026-11-05", u)
	assert.True(t, strings.HasPrefix(u, "https://www."),
		"çıplak aeo.org.tr 301 veriyor; www zorunlu")
}

// RefreshEczane sözleşmesi: resmi oda, toplayıcıdan ÖNCE denenmeli.
func TestEczaneOdaOnceDenenir(t *testing.T) {
	src := okuPaketDosya(t, "services_fetch.go")
	i := strings.Index(src, "eczaneOdaKaynaklari[c.Plaka]")
	j := strings.Index(src, "eczaneEczanelerOrg(ctx, c.Slug)")
	require.NotEqual(t, -1, i, "resmi oda kaynağı RefreshEczane'e bağlı değil")
	require.NotEqual(t, -1, j, "eczaneler.org kaynağı kaybolmuş")
	assert.Less(t, i, j,
		"resmi eczacı odası toplayıcıdan ÖNCE denenmeli: liste asıl kaynak odadır")
}
