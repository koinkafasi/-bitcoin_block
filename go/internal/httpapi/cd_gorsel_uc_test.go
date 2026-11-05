package httpapi

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/config"
	"theblock/site/internal/platform"
)

// Ölçekli kaynak görsel ucu — HTTP seviyesinde.
//
// NEDEN BU TESTLER: /kg/{hash}/{genislik} iki şeyi birlikte yapıyor —
// güvenlik kapısı (hash + kapalı genişlik listesi) ve üret-yaz-sun
// önbelleği. Birim testleri yardımcıları ayrı ayrı sınıyor; burada ROTA'nın
// kayıtlı olduğu ve akışın uçtan uca çalıştığı doğrulanıyor. Rota kaydı
// unutulduğunda yardımcılar yeşil kalır, sayfa görselsiz çıkar.

const ucTestHash = "80fe8a843afddb33661ff0f019b8ac11"

func gorselUcSunucu(t *testing.T) *http.ServeMux {
	t.Helper()
	s := &Server{
		mux:    http.NewServeMux(),
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:    config.Config{Env: "test", BaseURL: "https://bitcosar.com"},
	}
	s.routes()
	return s.mux
}

// testJPEG belirtilen boyutta, sıkışmayacak kadar desenli bir JPEG üretir.
func testJPEG(t *testing.T, g, y int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, g, y))
	for j := 0; j < y; j++ {
		for i := 0; i < g; i++ {
			img.Set(i, j, color.RGBA{uint8(i * j % 251), uint8(i % 253), uint8(j % 247), 255})
		}
	}
	var buf bytes.Buffer
	require.NoError(t, jpeg.Encode(&buf, img, nil))
	return buf.Bytes()
}

// depoHazirla geçici depolama kökü kurar ve eskisini geri yükler.
//
// platform.StorageDir paket düzeyinde global; testten sonra geri alınmazsa
// aynı paketteki diğer testler geçici dizine bakar.
func depoHazirla(t *testing.T) string {
	t.Helper()
	eski := platform.StorageDir()
	kok := t.TempDir()
	platform.SetStorageDir(kok)
	t.Cleanup(func() { platform.SetStorageDir(eski) })
	return kok
}

// Rota KAYITLI olmalı: diskteki varyant doğrudan sunulur.
func TestKgOlcekliRotaVarDiskteniSunar(t *testing.T) {
	kok := depoHazirla(t)
	gorsel := testJPEG(t, 480, 270)

	// Varyantı önceden diske koy: "diskte varsa sun" dalı.
	dizin := filepath.Join(kok, "kaynak", ucTestHash[:2])
	require.NoError(t, os.MkdirAll(dizin, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dizin, ucTestHash+"-480"), gorsel, 0o644))

	rec := httptest.NewRecorder()
	gorselUcSunucu(t).ServeHTTP(rec,
		httptest.NewRequest(http.MethodGet, "/kg/"+ucTestHash+"/480", nil))

	assert.Equal(t, http.StatusOK, rec.Code, "rota kayıtlı değil ya da dosya bulunamadı")
	assert.Equal(t, "image/jpeg", rec.Header().Get("Content-Type"))
	assert.Contains(t, rec.Header().Get("Cache-Control"), "immutable",
		"içerik adresli URL uzun önbellek almalı")
	assert.Equal(t, len(gorsel), rec.Body.Len())
}

// Varyant yoksa ORİJİNALDEN üretilir, diske YAZILIR ve ölçekli sunulur.
//
// Asıl kazanç bu dalda: 1600px'lik kaynak 480px'e inince gövde küçülür.
func TestKgOlcekliUretirYazarSunar(t *testing.T) {
	kok := depoHazirla(t)
	asil := testJPEG(t, 1600, 900)

	dizin := filepath.Join(kok, "kaynak", ucTestHash[:2])
	require.NoError(t, os.MkdirAll(dizin, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dizin, ucTestHash), asil, 0o644))

	rec := httptest.NewRecorder()
	gorselUcSunucu(t).ServeHTTP(rec,
		httptest.NewRequest(http.MethodGet, "/kg/"+ucTestHash+"/480", nil))

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "image/jpeg", rec.Header().Get("Content-Type"))
	assert.Less(t, rec.Body.Len(), len(asil),
		"ölçekli gövde orijinalden küçük olmalı (%d >= %d)", rec.Body.Len(), len(asil))

	// Gerçekten 480px mi?
	ol, _, err := image.Decode(bytes.NewReader(rec.Body.Bytes()))
	require.NoError(t, err)
	assert.Equal(t, 480, ol.Bounds().Dx())

	// Önbelleğe yazıldı mı? Yazılmazsa her istek yeniden ölçekler (CPU).
	varyant := filepath.Join(dizin, ucTestHash+"-480")
	st, err := os.Stat(varyant)
	require.NoError(t, err, "varyant diske yazılmalı")
	assert.Greater(t, st.Size(), int64(0))

	// Orijinal YERİNDE kalmalı — üzerine yazılırsa bir daha elde edilemez.
	st2, err := os.Stat(filepath.Join(dizin, ucTestHash))
	require.NoError(t, err)
	assert.Equal(t, int64(len(asil)), st2.Size(), "orijinal değişmemeli")
}

// Liste dışı genişlik 404 — disk doldurma/CPU tüketme kapısı.
func TestKgOlcekliListeDisiGenislik404(t *testing.T) {
	kok := depoHazirla(t)
	dizin := filepath.Join(kok, "kaynak", ucTestHash[:2])
	require.NoError(t, os.MkdirAll(dizin, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dizin, ucTestHash),
		testJPEG(t, 800, 450), 0o644))

	mux := gorselUcSunucu(t)
	for _, g := range []string{"1", "2", "481", "1601", "99999", "0", "-480", "abc"} {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/kg/"+ucTestHash+"/"+g, nil))
		assert.Equal(t, http.StatusNotFound, rec.Code,
			"liste dışı genişlik %q kabul edilmemeli", g)
	}
}

// Geçersiz hash 404 — yol aşımı kapısı.
func TestKgOlcekliGecersizHash404(t *testing.T) {
	depoHazirla(t)
	mux := gorselUcSunucu(t)
	for _, h := range []string{
		"kisa",
		"80FE8A843AFDDB33661FF0F019B8AC11",  // büyük harf
		"80fe8a843afddb33661ff0f019b8ac1z",  // hex dışı
		"80fe8a843afddb33661ff0f019b8ac111", // 33 karakter
	} {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/kg/"+h+"/480", nil))
		assert.Equal(t, http.StatusNotFound, rec.Code, "geçersiz hash kabul edilmemeli: %q", h)
	}
}

// Her genişlik AYRI dosyaya yazılmalı; biri diğerini ezmemeli.
func TestKgOlcekliGenisliklerAyriDosya(t *testing.T) {
	kok := depoHazirla(t)
	dizin := filepath.Join(kok, "kaynak", ucTestHash[:2])
	require.NoError(t, os.MkdirAll(dizin, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dizin, ucTestHash),
		testJPEG(t, 1600, 900), 0o644))

	mux := gorselUcSunucu(t)
	boyut := map[int]int{}
	for _, g := range []int{320, 480, 768} {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet,
			"/kg/"+ucTestHash+"/"+itoa(g), nil))
		require.Equal(t, http.StatusOK, rec.Code, "genişlik %d", g)
		boyut[g] = rec.Body.Len()
	}
	assert.Less(t, boyut[320], boyut[480], "320 varyantı 480'den küçük olmalı")
	assert.Less(t, boyut[480], boyut[768], "480 varyantı 768'den küçük olmalı")

	for _, g := range []int{320, 480, 768} {
		_, err := os.Stat(filepath.Join(dizin, ucTestHash+"-"+itoa(g)))
		assert.NoError(t, err, "genişlik %d için ayrı dosya olmalı", g)
	}
}
