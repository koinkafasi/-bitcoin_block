package httpapi

import (
	"encoding/json"
	"html/template"
	"net/http/httptest"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/webui"
)

func TestWithHTMXCurrentQuery(t *testing.T) {
	r := httptest.NewRequest("POST", "/admin/icerikler/1/sil", nil)
	r.Header.Set("HX-Current-URL", "http://127.0.0.1:8070/admin/icerikler?durum=taslak&tip=video&sayfa=2")
	r2 := withHTMXCurrentQuery(r)
	assert.Equal(t, "durum=taslak&tip=video&sayfa=2", r2.URL.RawQuery)
	// Orijinal istek değişmemeli.
	assert.Empty(t, r.URL.RawQuery)
}

func TestIsHTMX(t *testing.T) {
	r := httptest.NewRequest("GET", "/admin/icerikler", nil)
	assert.False(t, IsHTMX(r))

	r.Header.Set("HX-Request", "true")
	assert.True(t, IsHTMX(r))

	r.Header.Set("HX-Request", "TRUE")
	assert.True(t, IsHTMX(r))

	r.Header.Set("HX-Request", "false")
	assert.False(t, IsHTMX(r))
}

// Partial'lar dahil tüm şablon ağacının parse edilebildiğini doğrular.
// templateFuncs, NewServer'da eklenen mediaURL/asset/urlAbs/catIcon
// fonksiyonlarını içermez; parse için yalnızca isimlerinin var olması yeterli.
func TestAdminTemplatesParse(t *testing.T) {
	funcs := templateFuncs()
	for _, name := range []string{"mediaURL", "mediaThumbURL", "mediaVariantURL", "mediaSrcset", "asset", "urlAbs", "catIcon"} {
		funcs[name] = func(...any) string { return "" }
	}
	_, err := template.New("_").Funcs(funcs).ParseFS(webui.FS, "templates/*.html")
	require.NoError(t, err)
}

// TestWriteHTMXTriggerASCIISafe, HX-Trigger başlığının Türkçe karakter ve
// emoji içeren JSON'u bozmadan (ASCII kaçışlarla) taşıdığını doğrular.
// Başlıklar bayt düzeyinde ISO-8859-1 gibi çözüldüğünden ham UTF-8
// gönderilirse tarayıcıda "Türkçe" → "TÃ¼rkÃ§e" olur.
func TestWriteHTMXTriggerASCIISafe(t *testing.T) {
	const want = "Türkçe şğüöçİı — \"alıntı\"\nyeni satır 🚀"

	s := &Server{}
	rec := httptest.NewRecorder()
	s.writeHTMXTrigger(rec, "aiResult", map[string]any{"result": want, "target": "#f_title"})

	h := rec.Header().Get("HX-Trigger")
	require.NotEmpty(t, h)
	for i := 0; i < len(h); i++ {
		require.Less(t, h[i], byte(utf8.RuneSelf),
			"başlık ASCII dışı bayt içermemeli (konum %d)", i)
	}

	var parsed map[string]struct {
		Result string `json:"result"`
		Target string `json:"target"`
	}
	require.NoError(t, json.Unmarshal([]byte(h), &parsed))
	assert.Equal(t, want, parsed["aiResult"].Result)
	assert.Equal(t, "#f_title", parsed["aiResult"].Target)
}

// marshalJSONASCII yalnızca başlık güvenli çıktı üretmeli; astral
// karakterler UTF-16 vekil çiftine çevrilmeli.
func TestMarshalJSONASCII(t *testing.T) {
	out, err := marshalJSONASCII(map[string]string{"s": "ğ🚀"})
	require.NoError(t, err)
	for i := 0; i < len(out); i++ {
		require.Less(t, out[i], byte(utf8.RuneSelf))
	}
	assert.Contains(t, out, `\u011f`)
	assert.Contains(t, out, `\ud83d\ude80`)

	var back map[string]string
	require.NoError(t, json.Unmarshal([]byte(out), &back))
	assert.Equal(t, "ğ🚀", back["s"])

	_, err = marshalJSONASCII(make(chan int))
	require.Error(t, err)
}
