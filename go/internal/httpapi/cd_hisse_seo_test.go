package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCDHisseDizinYonlendirme(t *testing.T) {
	s := &Server{}

	for giris, beklenen := range map[string]string{
		"/price/stock":           "/price",
		"/price/stock?aralik=1y": "/price?aralik=1y",
	} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, giris, nil)
		s.handleCDHisseDizin(w, r)

		if w.Code != http.StatusMovedPermanently {
			t.Errorf("%s: kod %d, beklenen 301", giris, w.Code)
		}
		if got := w.Header().Get("Location"); got != beklenen {
			t.Errorf("%s: Location %q, beklenen %q", giris, got, beklenen)
		}
	}
}

func TestCdFiyatSayi(t *testing.T) {
	for giris, beklenen := range map[string]float64{
		"$610.15":   610.15,
		"₺306.25":   306.25,
		"₺1,234.56": 1234.56,
		"$1,610.80": 1610.80,
		"€99.9":     99.9,
	} {
		got, ok := cdFiyatSayi(giris)
		if !ok {
			t.Errorf("%q çözülemedi", giris)
			continue
		}
		if fark := got - beklenen; fark > 0.005 || fark < -0.005 {
			t.Errorf("%q → %v, beklenen %v", giris, got, beklenen)
		}
	}
	// Anlamsız girdiler sayı üretmemeli (yapısal veriye 0 yazılmasın).
	for _, g := range []string{"", "—", "abc", "$0.00"} {
		if v, ok := cdFiyatSayi(g); ok {
			t.Errorf("%q sayı üretti: %v", g, v)
		}
	}
}

func TestCdHisseJSONLD(t *testing.T) {
	s := string(cdHisseJSONLD("bitcosar", "https://ornek.test/price/stock/meta",
		"Meta Platforms, Inc.", "META", "NasdaqGS", "USD", "$610.15", "+0.84%"))

	for _, istenen := range []string{
		`"FinancialProduct"`,
		`"tickerSymbol":"META"`,
		`"name":"Exchange","value":"NasdaqGS"`,
		`"name":"Price","unitText":"USD","value":610.15`,
		`"name":"Price Change (24 hours)","unitText":"%","value":0.84`,
	} {
		if !strings.Contains(s, istenen) {
			t.Errorf("eksik: %s\n%s", istenen, s)
		}
	}
	// Yuvarlanmamış float sızmasın (0.8400000000000001 gibi).
	if strings.Contains(s, "0.8400000") {
		t.Errorf("yuvarlanmamış yüzde yazılmış: %s", s)
	}
	// Market Cap UYDURULMAZ (uç vermiyor).
	if strings.Contains(strings.ToLower(s), "marketcap") {
		t.Errorf("uydurma market cap alanı yazılmış")
	}
}
