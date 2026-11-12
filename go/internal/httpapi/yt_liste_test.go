package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"theblock/site/internal/repo"
)

func TestYtListeDilim(t *testing.T) {
	kayitlar := make([]repo.YoutubeKayit, 12)
	for i := range kayitlar {
		kayitlar[i].VideoID = itoa(i)
	}
	for _, adet := range []int{3, 5, 6, 12, 20} {
		got := ytListeDilim(kayitlar, adet)
		beklenen := adet
		if beklenen > len(kayitlar) {
			beklenen = len(kayitlar)
		}
		if len(got) != beklenen {
			t.Errorf("adet=%d -> %d satır, %d beklenirdi", adet, len(got), beklenen)
		}
	}
	if ytListeDilim(kayitlar, 0) != nil {
		t.Error("adet=0 nil dönmeli")
	}
	if ytListeDilim(nil, 5) != nil {
		t.Error("boş liste nil dönmeli")
	}
}

func TestYtListeTekSorgu(t *testing.T) {
	t.Skip("pool yok: DB sorgu sayımı entegrasyon testinde")
}

// TestSayfaOnbellekIleUyum: bu değişiklikler önbellek kapsamını değiştirmedi.
func TestSayfaOnbellekIleUyum(t *testing.T) {
	for _, yol := range []string{"/", "/markets", "/tag/bitcoin",
		"/policy/2026/11/05/ornek-makale"} {
		r := httptest.NewRequest(http.MethodGet, yol, nil)
		if !sayfaOnbellegeUygun(r) {
			t.Errorf("%q hâlâ önbelleğe uygun olmalıydı", yol)
		}
	}
}
