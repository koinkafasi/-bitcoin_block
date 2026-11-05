package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/repo"
)

// ---- Faz 143: kurumsal sayfa düzeni ------------------------------

func TestBuilderPageSayfaAnahtari(t *testing.T) {
	cases := []struct {
		url  string
		want string
	}{
		{"/admin/tasarim?sayfa=sayfa", "sayfa"},
		{"/admin/tasarim?sayfa=sayfa&sayfa_slug=kullanim-kosullari", "sayfa:kullanim-kosullari"},
		{"/admin/tasarim?sayfa=sayfa%3Akunye", "sayfa:kunye"},
		// Geçersiz slug genel anahtara düşer (yol enjeksiyonu yok).
		{"/admin/tasarim?sayfa=sayfa%3A../etc", "sayfa"},
		{"/admin/tasarim?sayfa=sayfa&sayfa_slug=k%C3%B6t%C3%BC", "sayfa"},
	}
	for _, c := range cases {
		r := httptest.NewRequest("GET", c.url, nil)
		if got := builderPage(r); got != c.want {
			t.Errorf("%s → %q, beklenen %q", c.url, got, c.want)
		}
	}
}

func TestBuilderZonesSayfa(t *testing.T) {
	got := zoneKeys("sayfa:kullanim-kosullari")
	assert.Equal(t, []string{"ust", "ana", "genis"}, got)
	assert.NotContains(t, got, "yan", "kurumsal sayfada yan sütun yok")
}

func TestBuilderTypesSayfa(t *testing.T) {
	var govde, metin bool
	for _, typ := range builderTypesFor("sayfa:kunye") {
		switch typ.Type {
		case "sayfa_govde":
			govde = true
		case "metin":
			metin = true
		}
	}
	assert.True(t, govde, "sayfa paletinde sayfa_govde olmalı")
	assert.True(t, metin, "sayfa paletinde metin olmalı")
}

func TestNormalizeSayfaGovdeTekilVeAna(t *testing.T) {
	s := &Server{}
	in := json.RawMessage(`[
		{"id":"g1","type":"sayfa_govde","opts":{"zone":"genis"}},
		{"id":"g2","type":"sayfa_govde","opts":{"zone":"ust"}},
		{"id":"m","type":"metin","opts":{"zone":"ana","html":"<p>x</p>"}}]`)
	_, arr, err := s.normalizeBlocksForPage("sayfa:kunye", in)
	require.NoError(t, err)
	require.Len(t, arr, 2, "ikinci sayfa_govde düşürülmeli")
	opts0, _ := arr[0]["opts"].(map[string]any)
	assert.Equal(t, "ana", opts0["zone"], "gövde bloğu her zaman ana bölgesindedir")
}

func TestNormalizeGovdeBloklariCaprazDuser(t *testing.T) {
	s := &Server{}
	// sayfa_govde yazı sayfasında, yazi_govde kurumsal sayfada tutulmaz.
	in := json.RawMessage(`[
		{"id":"a","type":"sayfa_govde","opts":{"zone":"ana"}},
		{"id":"b","type":"yazi_govde","opts":{"zone":"ana"}},
		{"id":"c","type":"baslik","opts":{"zone":"ana"}}]`)
	_, arrA, err := s.normalizeBlocksForPage("article", in)
	require.NoError(t, err)
	require.Len(t, arrA, 2)
	assert.Equal(t, "yazi_govde", arrA[0]["type"])

	_, arrB, err := s.normalizeBlocksForPage("sayfa:kunye", in)
	require.NoError(t, err)
	require.Len(t, arrB, 2)
	assert.Equal(t, "sayfa_govde", arrB[0]["type"])
	assert.Equal(t, "baslik", arrB[1]["type"])
}

func TestResolveSayfaGovdeYerTutucusuz(t *testing.T) {
	// Gövde HTML'i {…} içerebilir; yer tutucu değişimi uygulanmamalı.
	defs := pageTextDefsFor("sayfa:kunye")
	for _, d := range defs {
		if d.Key == "sayfa.govde" {
			assert.True(t, d.Raw, "sayfa.govde Raw olmalı")
		}
	}
	rows := []repo.PageText{
		{PageKey: "sayfa:kunye", TextKey: "sayfa.govde", Locale: "tr", Value: "<p>{baslik} metni</p>"},
	}
	got := resolvePageTexts("sayfa:kunye", "sayfa:kunye", rows, "tr", "tr", defs,
		map[string]string{"baslik": "Künye"})
	assert.Equal(t, "<p>{baslik} metni</p>", got["sayfa.govde"])
}

func TestSayfaLegacyRedirect(t *testing.T) {
	s := &Server{}
	// Türkçe (dil öneki yok).
	r := httptest.NewRequest("GET", "/sayfa/kullanim-kosullari?x=1", nil)
	w := httptest.NewRecorder()
	s.handlePageLegacyRedirect(w, r, "tr", "kullanim-kosullari")
	assert.Equal(t, http.StatusMovedPermanently, w.Code)
	assert.Equal(t, "/kullanim-kosullari?x=1", w.Header().Get("Location"))

	// İngilizce dil öneki korunur.
	r2 := httptest.NewRequest("GET", "/en/sayfa/kunye", nil)
	w2 := httptest.NewRecorder()
	s.handlePageLegacyRedirect(w2, r2, "en", "kunye")
	assert.Equal(t, "/en/kunye", w2.Header().Get("Location"))
}
