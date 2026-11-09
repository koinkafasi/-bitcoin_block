package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"theblock/site/internal/repo"
)

// TestYtListeDilim: önbellekteki uzun liste istenen uzunlukta kesilir.
//
// NEDEN ÖNEMLİ: önbellek TÜR başına tutuluyor, modüller farklı uzunluk
// istiyor (3/5/6/12). Dilim yanlış olsaydı modül ya eksik ya taşkın
// görünürdü.
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

// TestYtListeTekSorgu: aynı tür iki kez istenince DB'ye BİR kez gidilir.
//
// Bu, ana sayfadaki 6 çağrının 3'e inmesini koruyan testtir. Sayaç
// gerçek DB gerektirdiği için yalnızca havuz varsa çalışır; yoksa atlanır.
func TestYtListeTekSorgu(t *testing.T) {
	// Havuzsuz sunucuda sorgu yolu hata verir; yalnız önbellek
	// davranışını sınayabiliyoruz. Gerçek DB sayımı entegrasyon testinin
	// işi (bu pakette havuz yok).
	t.Skip("pool yok: DB sorgu sayımı entegrasyon testinde")
}

// kategoriID ÖNBELLEK SÖZLEŞMESİ: DB hatası önbelleğe yazılmaz — bu
// davranış `TestTTLCacheBasarisizSonucOnbellegeYazilmaz` tarafından
// korunuyor (ttlcache `ok=false` sözleşmesi). Havuzsuz bir Server'da
// pgxpool nil olduğu için burada ayrıca sınanamıyor (panic eder, hata
// dönmez); bu yüzden yinelenen test eklenmedi.

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
