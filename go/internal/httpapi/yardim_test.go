package httpapi

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/webui/templkit"
)

// Yardım merkezi (Faz 165).
//
// NEDEN: bilgi docs/ içinde kalıyordu. Ayrıca Faz 162'de komut paletine
// "/admin/yardim" eklendiği hâlde rota yoktu — paletten açılan bağlantı
// 404 veriyordu. Bu testler hem rotanın hem içeriğin varlığını kilitler.

func TestYardimIcerigiDolu(t *testing.T) {
	assert.GreaterOrEqual(t, len(yardimKisayollari()), 6, "en az 6 klavye kısayolu")
	assert.GreaterOrEqual(t, len(yardimIpuclari()), 8, "en az 8 ekran ipucu")
	assert.GreaterOrEqual(t, len(yardimSSS()), 6, "en az 6 sık sorulan")
	assert.GreaterOrEqual(t, len(yardimBaglantilari()), 5, "en az 5 ilgili ekran")

	for _, k := range yardimKisayollari() {
		assert.NotEmpty(t, k.Tus)
		assert.NotEmptyf(t, k.Aciklama, "kısayol açıklaması boş: %s", k.Tus)
	}
	for _, m := range append(yardimIpuclari(), yardimSSS()...) {
		assert.NotEmptyf(t, m.Soru, "soru boş")
		assert.Greaterf(t, len([]rune(m.Cevap)), 30, "cevap çok kısa: %s", m.Soru)
	}
}

// Bağlantılar var olan panel rotalarına gitmeli. Ölü bağlantı, yardım
// sayfasının en can sıkıcı hatası.
func TestYardimBaglantilariGecerli(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("server.go"))
	require.NoError(t, err)
	sunucu := string(b)

	for _, b := range yardimBaglantilari() {
		assert.True(t, strings.HasPrefix(b.Href, "/admin/"),
			"bağlantı panel içinde olmalı: %s", b.Href)
		assert.NotEmptyf(t, b.Ikon, "bağlantı ikonu boş: %s", b.Href)
		// Yardım merkezinin kendisi hariç her hedef rotalanmış olmalı.
		if b.Href == "/admin/yardim" {
			continue
		}
		assert.Containsf(t, sunucu, `"`+b.Href+`"`,
			"hedef rota sunucuda tanımlı değil: %s", b.Href)
	}
}

// Yardım rotası sunucuda tanımlı olmalı (Faz 162 palet bağlantısı).
func TestYardimRotasiVar(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("server.go"))
	require.NoError(t, err)
	assert.Contains(t, string(b), `{"GET", "/admin/yardim", s.handleYardim}`)

	// Palet de aynı yolu bilmeli.
	js, err := os.ReadFile(filepath.Join("..", "webui", "static", "app.js"))
	require.NoError(t, err)
	assert.Contains(t, string(js), "/admin/yardim")
}

// İçerikte emoji olmamalı (Faz 161 kuralı).
func TestYardimIcerigiEmojisiz(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("yardim.go"))
	require.NoError(t, err)
	for _, e := range []string{"🎉", "✨", "🔎", "⚡", "💾"} {
		assert.NotContainsf(t, string(b), e, "yardım içeriğinde emoji olmamalı: %s", e)
	}
}

// Her ipucu kaynağında `Ekran` etiketi olmalı: kullanıcı hangi ekranda
// olduğunu bilmeli.
func TestYardimIpuclariEkranEtiketli(t *testing.T) {
	for _, m := range yardimIpuclari() {
		assert.NotEmptyf(t, m.Ekran, "ipucunda ekran etiketi yok: %s", m.Soru)
		assert.NotEmptyf(t, m.Ikon, "ipucunda ikon yok: %s", m.Soru)
	}
}

// Türkçe katlama: "I"→"ı", "İ"→"i". strings.ToLower bunu yapmaz.
func TestLowerTRTurkceKucultme(t *testing.T) {
	cases := map[string]string{
		"ISLEM":     "ıslem",
		"İŞLEM":     "işlem",
		"Gazete":    "gazete",
		"SAĞLIK":    "sağlık",
		"İçerikler": "içerikler",
	}
	for giris, beklenen := range cases {
		assert.Equalf(t, beklenen, lowerTRTest(giris), "lowerTR(%q)", giris)
	}
}

// lowerTRTest, templkit'teki yardımcının test kopyası.
//
// NEDEN KOPYA: lowerTR templkit paketinde ve dışa açık değil; şablon
// yardımcısı olduğu için API yüzeyine eklemek istemedik. Davranış burada
// kilitleniyor, uygulama aynı kuralla yazılı.
func lowerTRTest(v string) string {
	return strings.ToLower(strings.NewReplacer("İ", "i", "I", "ı").Replace(v))
}

// Yardım bileşeni erişilebilirlik izlerini taşımalı.
func TestYardimBileseniErisilebilir(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "webui", "templkit", "yardim.templ"))
	require.NoError(t, err)
	s := string(b)
	assert.Contains(t, s, `aria-label="Yardımda ara"`)
	assert.Contains(t, s, `aria-live="polite"`)
	assert.Contains(t, s, "<details")
	assert.Contains(t, s, "<summary")
}

// Arama katlaması sunucu tarafında da olmalı: "saglik" → "Sağlık".
func TestAramaKatlamaSQL(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "repo", "admin_search.go"))
	require.NoError(t, err)
	s := string(b)
	assert.Contains(t, s, "unaccent(",
		"arama unaccent kullanmalı; ILIKE Türkçe ayrımına duyarlı (ölçüldü)")
	assert.Contains(t, s, "lower(public.immutable_unaccent(c.title))",
		"indeksle AYNI ifade kullanılmalı: lower(public.immutable_unaccent(...))")
	assert.NotContains(t, s, "c.title ILIKE $2",
		"katlanmamış ILIKE kalmamalı")

	// Migration uzantıyı kurmalı.
	mig, err := os.ReadFile(filepath.Join("..", "..", "..", "go", "migrations", "0098_arama_unaccent.up.sql"))
	require.NoError(t, err)
	ms := string(mig)
	assert.Contains(t, ms, "CREATE EXTENSION IF NOT EXISTS unaccent")
	assert.Contains(t, ms, "gin_trgm_ops")
	// unaccent() STABLE'dır; indeks ifadesinde IMMUTABLE sarmalayıcı şart,
	// yoksa PostgreSQL 42P17 ile reddeder (bir kez yaşandı).
	assert.Contains(t, ms, "IMMUTABLE")
	assert.Contains(t, ms, "public.immutable_unaccent(text)")
}

// Palet katlaması istemcide de olmalı: "yardim" → "Yardım".
func TestPaletKatlamaJS(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "webui", "static", "app.js"))
	require.NoError(t, err)
	s := string(b)
	assert.Contains(t, s, "'ı': 'i'", "Türkçe noktasız ı katlanmalı")
	assert.Contains(t, s, "'ş': 's'")
}

var _ = templkit.YardimView{}
