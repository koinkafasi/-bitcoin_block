package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/repo"
)

func TestFinansTurleri(t *testing.T) {
	r := httptest.NewRequest("GET", "/api/piyasa/canli?tur=hisse,kripto", nil)
	out := finansTurleri(r)
	assert.True(t, out["hisse"])
	assert.True(t, out["kripto"])
	assert.False(t, out["kur"], "istenmeyen tür okunmamalı")

	// Parametre yoksa tüm türler (eski istemciler/elle deneme için).
	all := finansTurleri(httptest.NewRequest("GET", "/api/piyasa/canli", nil))
	for _, tur := range []string{"kur", "hisse", "kripto", "piyasa", "servis"} {
		assert.True(t, all[tur], "%s varsayılanda olmalı", tur)
	}

	// Geçersiz tür adları boş yanıt üretmez; güvenli biçimde tümüne döner.
	bad := finansTurleri(httptest.NewRequest("GET", "/api/piyasa/canli?tur=bilinmeyen", nil))
	assert.True(t, bad["hisse"])
	assert.True(t, bad["kripto"])
}

func TestFinansETag(t *testing.T) {
	a := finansETag([]byte(`{"ok":true}`))
	b := finansETag([]byte(`{"ok":true}`))
	c := finansETag([]byte(`{"ok":false}`))
	assert.Equal(t, a, b, "aynı gövde aynı ETag")
	assert.NotEqual(t, a, c, "değişen gövde farklı ETag")

	assert.True(t, finansETagEsles(a, a))
	assert.True(t, finansETagEsles("W/"+a, a))
	assert.True(t, finansETagEsles(`"baska", `+a, a))
	assert.True(t, finansETagEsles("*", a))
	assert.False(t, finansETagEsles(`"baska"`, a))
	assert.False(t, finansETagEsles("", a))
}

func TestFinansCanliJSONSozlesmesi(t *testing.T) {
	veri := finansCanli{
		Ok: true,
		Guncel: map[string]finansZaman{
			"hisse": {HM: "15:04", DT: "29.10.2026 15:04"},
		},
		Hisse: map[string]finansFiyat{
			"THYAO": {F: 252.5, D: -0.198},
		},
		Kripto: map[string]finansKripto{
			"BTC": {F: 4000000, D: 0.56, D7: -1.2, H: 123, P: 456},
		},
		Piyasa: map[string]finansPiyasa{
			"usd": {F: 49.83, D: 0.05, Y: 1, O: 4},
		},
		Kur:    map[string]float64{"USD": 49.83},
		Servis: map[string]finansFiyat{"USD": {F: 49.83, D: 0.05}},
	}
	b, err := json.Marshal(veri)
	require.NoError(t, err)
	s := string(b)
	assert.Contains(t, s, `"hisse":{"THYAO":{"f":252.5,"d":-0.198}}`)
	assert.Contains(t, s, `"kripto":{"BTC":{"f":4000000,"d":0.56,"d7":-1.2,"h":123,"p":456}}`)
	assert.Contains(t, s, `"piyasa":{"usd":{"f":49.83,"d":0.05,"y":1,"o":4}}`)
	assert.Contains(t, s, `"guncel":{"hisse":{"hm":"15:04","dt":"29.10.2026 15:04"}}`)
}

func TestFinansSablonBaglantilari(t *testing.T) {
	tmpl := parseTestTemplates(t)

	// Hisseler modülü: sembol + fiyat + değişim alanları.
	var buf bytes.Buffer
	b := hisselerBlogu(hisseOrnek(), "serit")
	b.HisseDegisim = true
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "render_block", b))
	h := buf.String()
	assert.Contains(t, h, `data-fin="hisse" data-fin-sym="THYAO"`)
	assert.Contains(t, h, `data-fin-fld="f" data-fin-v="252.5"`)
	assert.Contains(t, h, `data-fin-fld="d" data-fin-v="-0.198"`)
	assert.Contains(t, h, `data-fin-live`)

	// Kripto modülü.
	buf.Reset()
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "render_block", kriptoBlogu(kriptoOrnek())))
	k := buf.String()
	assert.Contains(t, k, `data-fin="kripto" data-fin-sym="BTC"`)
	// html/template öznitelik bağlamında "+" işaretini &#43; olarak kaçırır;
	// tarayıcı getAttribute ile "4e+06" okur (parseFloat için geçerli).
	assert.Contains(t, k, `data-fin-fld="f" data-fin-v="4e&#43;06"`)
	assert.Contains(t, k, `data-fin-fld="d" data-fin-v="0.56"`)
	assert.Contains(t, k, `data-fin-time="kripto"`)
	assert.Contains(t, k, `data-fin-live`)

	// Kur & Borsa şeridi.
	kb := renderKurBorsa(t, ornekKalemler())
	assert.Contains(t, kb, `data-fin="piyasa" data-fin-sym="usd"`)
	assert.Contains(t, kb, `data-fin-fld="f" data-fin-v="49.8335"`)
	assert.Contains(t, kb, `data-fin-dec="4"`)
	assert.Contains(t, kb, `data-fin-time="piyasa"`)

	// Kur Çevirici: seçenekler canlı kur güncellemesine bağlı.
	buf.Reset()
	v := kurCeviriciOlustur(kurTablosuOrnek(), "", "USD", "TRY", 100)
	require.NotNil(t, v)
	cb := &homeBlock{Type: "kur_cevirici", KurCevirici: v, D: &tmplData{LangBase: ""}}
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "render_block", cb))
	c := buf.String()
	assert.Contains(t, c, `data-fin="kur"`)
	assert.Contains(t, c, `data-fin-kur="USD"`)
	assert.Equal(t, 12, bytes.Count([]byte(c), []byte("data-fin-kur=")), "iki seçicide tüm para birimleri")
}

func TestFinansZamanBicimi(t *testing.T) {
	z := finansZamanOf(time.Date(2026, 10, 29, 15, 4, 0, 0, time.UTC))
	assert.Equal(t, "15:04", z.HM)
	assert.Equal(t, "29.10.2026 15:04", z.DT)
	assert.Equal(t, finansZaman{}, finansZamanOf(time.Time{}), "sıfır zaman boş kalır")
}

// ---- servisKurlar: demo/eksik değer süzgeci ----

// finansFakeDB yalnız CacheGet sorgusunu yanıtlar; diğer çağrılar boş döner.
type finansFakeDB struct {
	kayit map[string][]byte // "kind/key" → JSON
}

func (f finansFakeDB) QueryRow(_ context.Context, _ string, args ...any) pgx.Row {
	if len(args) < 2 {
		return finansSatir{err: pgx.ErrNoRows}
	}
	b, ok := f.kayit[fmt.Sprintf("%v/%v", args[0], args[1])]
	if !ok {
		return finansSatir{err: pgx.ErrNoRows}
	}
	return finansSatir{b: b}
}

func (f finansFakeDB) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, nil
}

func (f finansFakeDB) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

type finansSatir struct {
	b   []byte
	err error
}

func (r finansSatir) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) != 1 {
		return fmt.Errorf("beklenmeyen hedef sayısı: %d", len(dest))
	}
	p, ok := dest[0].(*[]byte)
	if !ok {
		return fmt.Errorf("beklenmeyen hedef tipi: %T", dest[0])
	}
	*p = r.b
	return nil
}

func servisAdlari(items []repo.Kur) []string {
	out := make([]string, 0, len(items))
	for _, k := range items {
		out = append(out, k.Name)
	}
	return out
}

func TestServisKurlarDemoKurYok(t *testing.T) {
	// Demo bayrağı: USD/EUR/ALTIN banda hiç girmemeli; BIST/BTC/ONS kalmalı.
	db := finansFakeDB{kayit: map[string][]byte{
		"kur/ana":   []byte(`{"demo":true,"dolar":34.8,"euro":38.9,"altin":2980}`),
		"borsa/ana": []byte(`{"rows":[{"kind":"endeks","code":"XU100","value":11165.44,"delta":0.13},{"kind":"kripto","code":"BTC","value":3979074,"delta":-1.2},{"kind":"altin_ons","code":"ONS","value":4054.9,"delta":0.4}]}`),
	}}
	items, demo := servisKurlar(context.Background(), db)
	assert.True(t, demo)
	assert.Equal(t, []string{"BIST", "BTC", "ONS"}, servisAdlari(items))
	for _, k := range items {
		assert.Greater(t, k.Value, 0.0, "%s sıfır/negatif fiyatla basılmamalı", k.Name)
	}
}

func TestServisKurlarEksikDegerAtlanir(t *testing.T) {
	// demo=false ama EUR alınamamış (0): yalnız gerçek değerler banda girer.
	db := finansFakeDB{kayit: map[string][]byte{
		"kur/ana":   []byte(`{"demo":false,"dolar":49.83,"euro":0,"altin":5000}`),
		"borsa/ana": []byte(`{"rows":[{"kind":"endeks","code":"XU100","value":11165.44},{"kind":"kripto","code":"BTC","value":3979074},{"kind":"altin_ons","code":"ONS","value":4054.9}]}`),
	}}
	items, demo := servisKurlar(context.Background(), db)
	assert.False(t, demo)
	assert.Equal(t, []string{"BIST", "USD", "ALTIN", "BTC", "ONS"}, servisAdlari(items))
	for _, k := range items {
		assert.NotZero(t, k.Value, "%s eksik değerle basılmamalı", k.Name)
	}
}

// Piyasa önbelleği (truncgil + Yahoo) doluysa servis bandı USD/EUR/gram
// altın'ı oradan almalı; eski `kur/ana` (floatrates) kaynağı düşse bile bant
// eksik kalmamalı.
func TestServisKurlarPiyasaOnbelleginiKullanir(t *testing.T) {
	db := finansFakeDB{kayit: map[string][]byte{
		"piyasa/ana": []byte(`{"guncel":"2026-11-02T12:00:00Z","kalemler":[
			{"anahtar":"bist100","deger":10730.58,"degisim":-0.27},
			{"anahtar":"usd","deger":49.9718,"degisim":0.06},
			{"anahtar":"eur","deger":56.1705,"degisim":0.72},
			{"anahtar":"gram-altin","deger":6268.21,"degisim":1.1},
			{"anahtar":"btc","deger":3908110,"degisim":0.2},
			{"anahtar":"tahvil2y","deger":0,"degisim":0}]}`),
		"kur/ana":   []byte(`{"demo":true,"dolar":34.8,"euro":38.9,"altin":2980}`),
		"borsa/ana": []byte(`{"rows":[{"kind":"kripto","code":"BTC","value":3979074,"delta":-1.2},{"kind":"altin_ons","code":"ONS","value":4054.9,"delta":0.4}]}`),
	}}
	items, demo := servisKurlar(context.Background(), db)
	assert.False(t, demo, "gerçek veri varken demo bayrağı kalkmalı")
	assert.Equal(t, []string{"BIST", "USD", "EUR", "ALTIN", "BTC", "ONS"}, servisAdlari(items))
	assert.InDelta(t, 49.9718, items[1].Value, 1e-9, "USD canlı piyasa önbelleğinden gelmeli")
	assert.InDelta(t, 6268.21, items[3].Value, 1e-9, "gram altın canlı piyasa önbelleğinden gelmeli")
}

func TestServisKurlarOnbellekYok(t *testing.T) {
	items, demo := servisKurlar(context.Background(), finansFakeDB{})
	assert.False(t, demo)
	assert.Empty(t, items, "veri yoksa bant boş kalır (uydurma fiyat yok)")
}
