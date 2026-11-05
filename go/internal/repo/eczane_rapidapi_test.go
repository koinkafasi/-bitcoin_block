package repo

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Gerçek `pharmacies-on-duty?city=ankara` yanıtından alınmış örnek (alan
// adları ve biçimler birebir korundu; 5 Kasım 2026'da ölçüldü).
const eczaneRapidAPIOrnek = `{
  "status": "success",
  "rowCount": 4,
  "creditUsed": 0,
  "data": [
    {"pharmacyID":1,"pharmacyName":"Aksoy Eczanesi","address":"Atatürk Mah. Osmanlı Cad. No:78/D","city":"Ankara","cityId":6,"district":"Çankaya","town":"Dikmen","directions":"Onyx Ankara AVM Gratis yanı","phone":"0 (312) 512-26-12","phone2":null},
    {"pharmacyID":1,"pharmacyName":"Aksoy Eczanesi","address":"Atatürk Mah. Osmanlı Cad. No:78/D","city":"Ankara","cityId":6,"district":"Çankaya","town":"Dikmen","directions":"Onyx Ankara AVM Gratis yanı","phone":"0 (312) 512-26-12","phone2":null},
    {"pharmacyID":2,"pharmacyName":"Doğa Eczanesi","address":"Tuzcuzade Mah. Vali Nihat Üçyıldız Cad. No:32/A","city":"Ankara","cityId":6,"district":"Merkez","town":"","directions":null,"phone":"0 (543) 261-99-85","phone2":null},
    {"pharmacyID":4,"pharmacyName":"","address":"Adsız","city":"Ankara","cityId":6,"district":"Çankaya","phone":"0 (312) 000-00-00"}
  ]
}`

func TestEczaneRapidAPICoz(t *testing.T) {
	liste, err := eczaneRapidAPICoz([]byte(eczaneRapidAPIOrnek))
	require.NoError(t, err)

	// Aynı kayıt iki kez gelirse tek satıra iner; adsız kayıt atılır.
	require.Len(t, liste, 2, "tekilleşmiş 2 kayıt beklenirdi")

	aksoy := liste[0]
	assert.Equal(t, "Aksoy Eczanesi", aksoy.Ad)
	assert.Equal(t, "Çankaya", aksoy.Ilce)
	assert.Contains(t, aksoy.Adres, "Osmanlı Cad. No:78/D")
	assert.Contains(t, aksoy.Adres, "Tarif: Onyx Ankara AVM Gratis yanı")
	assert.Equal(t, "0 312 512 26 12", aksoy.Tel)

	doga := liste[1]
	assert.Equal(t, "Doğa Eczanesi", doga.Ad)
	assert.Equal(t, "Merkez", doga.Ilce)
	assert.Equal(t, "0 543 261 99 85", doga.Tel)
}

func TestEczaneRapidAPIBosListe(t *testing.T) {
	// İlde bugün nöbetçi eczane yoksa API boş dizi döner; bu hata değildir.
	liste, err := eczaneRapidAPICoz([]byte(`{"status":"success","rowCount":0,"creditUsed":0,"data":[]}`))
	require.NoError(t, err)
	assert.Empty(t, liste)
}

func TestEczaneRapidAPIBozukYanit(t *testing.T) {
	_, err := eczaneRapidAPICoz([]byte(`{"status":"error","message":"kota bitti"}`))
	require.Error(t, err, "status=error hata olmalı")

	_, err = eczaneRapidAPICoz([]byte(`bu JSON değil`))
	require.Error(t, err)
}

func TestEczaneTelefonDuzelt(t *testing.T) {
	cases := []struct{ in, want string }{
		{"0 (312) 512-26-12", "0 312 512 26 12"},
		{"0(553)925-98-78", "0 553 925 98 78"},
		{"+90 (216) 111-22-33", "0 216 111 22 33"},
		{"0212 444 33 22", "0 212 444 33 22"},
		{"", ""},
		// Çözülemeyen değer olduğu gibi kalır; uydurma numara üretilmez.
		{"123", "123"},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, eczaneTelefonDuzelt(c.in), "girdi: %q", c.in)
	}
}

// TestEczaneRapidAPIGercek, gerçek API'ye karşı sözleşmeyi doğrular.
// Varsayılan olarak ATLANIR: ağ gerektirir ve kota harcar (2 istek).
//
//	NOSYAPI_KEY=… go test ./internal/repo -run TestEczaneRapidAPIGercek -v
func TestEczaneRapidAPIGercek(t *testing.T) {
	anahtar := os.Getenv("NOSYAPI_KEY")
	if anahtar == "" {
		t.Skip("NOSYAPI_KEY yok — gerçek API testi atlandı")
	}

	// Ankara: her gün nöbetçi eczane yayımlar.
	ankara, err := eczaneRapidAPISehir(context.Background(), anahtar, "ankara")
	require.NoError(t, err)
	require.NotEmpty(t, ankara, "Ankara listesi boş")
	for _, e := range ankara {
		assert.NotEmpty(t, e.Ad, "adsız kayıt")
		assert.NotEmpty(t, e.Tel, "telefonsuz kayıt: %s", e.Ad)
	}

	// Küçük il de çalışmalı (slug/plaka eşleşmesi yalnız büyükşehirde
	// doğru olmasın).
	bayburt, err := eczaneRapidAPISehir(context.Background(), anahtar, "bayburt")
	require.NoError(t, err)
	require.NotEmpty(t, bayburt, "Bayburt listesi boş")

	t.Logf("ankara=%d bayburt=%d", len(ankara), len(bayburt))
}
