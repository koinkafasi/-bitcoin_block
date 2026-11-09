package httpapi

import (
	"net/url"
	"testing"
)

func TestTarihArsivMi(t *testing.T) {
	arsiv := []string{"/2019", "/2019/12", "/2019/12/11", "/2019/12/11/"}
	for _, yol := range arsiv {
		if !tarihArsivMi(yol) {
			t.Errorf("%q arşiv sayılmalıydı", yol)
		}
	}
	yazi := []string{
		"/2019/12/11/bitcoin-nedir",       // yazı adresi
		"/haber/2019/12/11/bitcoin-nedir", // bölüm önekli yazı
		"/kategori/haber",                 // kategori listesi
		"/price",                          // sayfa
		"/2019/12/11/bitcoin-nedir/",      // sondaki eğik çizgi
	}
	for _, yol := range yazi {
		if tarihArsivMi(yol) {
			t.Errorf("%q arşiv sayılmamalıydı", yol)
		}
	}
}

func TestWPKimlikSorgusu(t *testing.T) {
	ornekler := []struct {
		sorgu string
		bekle int64
	}{
		{"", 0},
		{"p=44739", 44739},
		{"p=abc", 0},
		{"p=0", 0},
		{"p=-3", 0},
		{"sayfa=2", 0},
	}
	for _, o := range ornekler {
		q, _ := url.ParseQuery(o.sorgu)
		if got := wpKimlikSorgusu(q); got != o.bekle {
			t.Errorf("wpKimlikSorgusu(%q) = %d, beklenen %d", o.sorgu, got, o.bekle)
		}
	}
}
