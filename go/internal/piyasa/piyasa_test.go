package piyasa

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Biçimlendirme (referans ekranla aynı) ---

func TestFormatTRBinlikVeOndalik(t *testing.T) {
	cases := map[string]struct {
		v       float64
		ondalik int
		want    string
	}{
		"bist":    {11151.03, 2, "11.151,03"},
		"dolar":   {49.8335, 4, "49,8335"},
		"ceyrek":  {10322.5, 2, "10.322,50"},
		"kucuk":   {0.003151, 4, "0,0032"},
		"sifir":   {0, 2, "0,00"},
		"negatif": {-1234.5, 2, "-1.234,50"},
		"milyon":  {4037867, 2, "4.037.867,00"},
	}
	for ad, c := range cases {
		assert.Equal(t, c.want, FormatTR(c.v, c.ondalik), ad)
	}
}

func TestFormatDegisimIsaret(t *testing.T) {
	assert.Equal(t, "+0,05", FormatDegisim(0.05))
	assert.Equal(t, "-0,15", FormatDegisim(-0.15))
	// 0,005 altı yatay: işaretsiz
	assert.Equal(t, "0,00", FormatDegisim(0.001))
	assert.Equal(t, "0,00", FormatDegisim(0))
	assert.Equal(t, "-2,03", FormatDegisim(-2.033))
}

func TestYonEsikDegeri(t *testing.T) {
	assert.Equal(t, 1, yon(0.006))
	assert.Equal(t, -1, yon(-0.006))
	assert.Equal(t, 0, yon(0))
	assert.Equal(t, 0, yon(0.004))
}

// --- Sahte kaynaklarla uçtan uca ---

const tgGovde = `{
  "Update_Date": "2026-10-28 07:28:02",
  "USD": {"Buying": 49.8335, "Selling": 49.841, "Change": 0.08, "Type": "Currency"},
  "EUR": {"Buying": 55.42, "Selling": 55.4296, "Change": -0.13, "Type": "Currency"},
  "GBP": {"Buying": 66.2532, "Selling": 66.2656, "Change": -0.15, "Type": "Currency"},
  "GRA": {"Buying": 6476.69, "Selling": 6477.59, "Change": -0.35, "Type": "Gold", "Name": "GRAMALTIN"},
  "GUMUS": {"Buying": 86.44, "Selling": 86.54, "Change": -0.73, "Type": "Gold"},
  "CEYREKALTIN": {"Buying": 10322.5, "Selling": 10558.61, "Change": 0, "Type": "Gold"},
  "XU100": {"Buying": null, "Selling": 11151.03, "Change": -0.01, "Type": "Gold"},
  "ONS": {"Buying": 0, "Selling": 0, "Change": -0.43, "Type": "Gold"}
}`

func yahooGovde(fiyat, onceki float64) string {
	return `{"chart":{"result":[{"meta":{"regularMarketPrice":` +
		f2(fiyat) + `,"chartPreviousClose":` + f2(onceki) + `}}],"error":null}}`
}

func f2(v float64) string {
	// JSON ondalık ayırıcısı nokta olmalı; FormatTR virgül üretir.
	return strconv.FormatFloat(v, 'f', 6, 64)
}

func TestGetTumKaynaklardanBirlestirir(t *testing.T) {
	// Yahoo yolu sembolü taşır (örn. /BTC-USD); eşleme sembol adına göre.
	fiyatlar := map[string][2]float64{
		"BTC-USD":  {80821.36, 80800.42},
		"EURUSD=X": {1.1121, 1.1125},
		"BZ=F":     {110.92, 109.41},
		"XU100.IS": {11151.03, 11152.0},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/today.json" {
			_, _ = w.Write([]byte(tgGovde))
			return
		}
		sym := strings.TrimPrefix(r.URL.Path, "/")
		if p, ok := fiyatlar[sym]; ok {
			_, _ = w.Write([]byte(yahooGovde(p[0], p[1])))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	eskiTG, eskiY := truncgilBase, yahooBase
	truncgilBase = srv.URL + "/today.json"
	yahooBase = srv.URL + "/%s"
	defer func() { truncgilBase, yahooBase = eskiTG, eskiY }()

	s, err := Get(context.Background())
	require.NoError(t, err)

	byKey := map[string]Kalem{}
	for _, k := range s.Kalemler {
		byKey[k.Anahtar] = k
	}
	// truncgil kaynaklı kalemler: satış fiyatı kullanılır (halkın gördüğü).
	assert.InDelta(t, 49.841, byKey["usd"].Deger, 0.0001, "dolar satış")
	assert.InDelta(t, 55.4296, byKey["eur"].Deger, 0.0001)
	assert.InDelta(t, 66.2656, byKey["gbp"].Deger, 0.0001)
	assert.InDelta(t, 6477.59, byKey["gram-altin"].Deger, 0.01)
	assert.InDelta(t, 86.54, byKey["gumus"].Deger, 0.01)
	assert.InDelta(t, 10558.61, byKey["ceyrek-altin"].Deger, 0.01)
	assert.InDelta(t, 11151.03, byKey["bist100"].Deger, 0.01)

	// Yahoo kaynaklı kalemler
	assert.InDelta(t, 80821.36, byKey["btc"].Deger, 0.01)
	assert.InDelta(t, 110.92, byKey["brent"].Deger, 0.01)
	assert.InDelta(t, 1.1121, byKey["eurusd"].Deger, 0.0001)

	// Değişim işaretleri
	assert.Equal(t, 1, byKey["usd"].Yon)
	assert.Equal(t, -1, byKey["eur"].Yon)
	assert.Equal(t, 0, byKey["ceyrek-altin"].Yon)
	assert.Contains(t, s.Kaynak, "truncgil")
	assert.Contains(t, s.Kaynak, "yahoo")
}

func TestGetSifirDegerliKalemAtlanir(t *testing.T) {
	// ONS 0/0 geliyor: ekranda "0,00" göstermek yanlış bilgi olurdu.
	// (ONS tanımı listede yok ama 0 değerinin atlanması kuralı korunmalı.)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/today.json" {
			_, _ = w.Write([]byte(`{"ONS":{"Buying":0,"Selling":0,"Change":-0.4}}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	eskiTG, eskiY := truncgilBase, yahooBase
	truncgilBase = srv.URL + "/today.json"
	yahooBase = srv.URL + "/%s"
	defer func() { truncgilBase, yahooBase = eskiTG, eskiY }()

	_, err := Get(context.Background())
	// Hiç geçerli kalem yok → hata (çağıran önbelleği ezmez).
	assert.Error(t, err)
}

func TestGetTruncgilKapaliysaYahooYeterli(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/today.json" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(yahooGovde(80821.36, 80800.42)))
	}))
	defer srv.Close()

	eskiTG, eskiY := truncgilBase, yahooBase
	truncgilBase = srv.URL + "/today.json"
	yahooBase = srv.URL + "/%s"
	defer func() { truncgilBase, yahooBase = eskiTG, eskiY }()

	s, err := Get(context.Background())
	require.NoError(t, err, "bir kaynak düşse de çalışmalı")
	assert.NotEmpty(t, s.Kalemler)
	assert.NotContains(t, s.Kaynak, "truncgil")
}

func TestFetchTruncgilSkalerAlanlariAtlar(t *testing.T) {
	// Update_Date bir string; çözümleme patlamamalı.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(tgGovde))
	}))
	defer srv.Close()
	eski := truncgilBase
	truncgilBase = srv.URL
	defer func() { truncgilBase = eski }()

	m, err := fetchTruncgil(context.Background())
	require.NoError(t, err)
	assert.Contains(t, m, "USD")
	assert.NotContains(t, m, "Update_Date")
}

func TestFetchYahooDegisimHesaplar(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// regularMarketChangePercent yok → önceki kapanıştan hesaplanmalı.
		_, _ = w.Write([]byte(yahooGovde(110, 100)))
	}))
	defer srv.Close()
	eski := yahooBase
	yahooBase = srv.URL + "/%s"
	defer func() { yahooBase = eski }()

	d := fetchYahoo(context.Background(), "X")
	require.True(t, d.ok)
	assert.InDelta(t, 110, d.fiyat, 0.001)
	assert.InDelta(t, 10, d.degisim, 0.001)
}

func TestGetZamanAsimiSessiz(t *testing.T) {
	// Yavaş kaynak Get'i kilitlememeli (client timeout).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
	}))
	defer srv.Close()
	eskiTG, eskiY, eskiClient := truncgilBase, yahooBase, httpClient
	truncgilBase = srv.URL + "/today.json"
	yahooBase = srv.URL + "/%s"
	httpClient = &http.Client{Timeout: 30 * time.Millisecond}
	defer func() { truncgilBase, yahooBase, httpClient = eskiTG, eskiY, eskiClient }()

	_, err := Get(context.Background())
	assert.Error(t, err)
}

func TestAnahtarlarSirasi(t *testing.T) {
	a := Anahtarlar()
	assert.Equal(t, "bist100", a[0], "BIST referansta ilk sırada")
	assert.Contains(t, a, "usd")
	assert.Contains(t, a, "ceyrek-altin")
	assert.Len(t, a, len(Tanimlar))
}
