package httpapi

import (
	"bytes"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/repo"
)

// Resmî ilan adresleri BİK site haritası maddesinin dayanağı: adres kalıcı,
// okunabilir ve kimlik sonda olmalı ki slug değişse bile ilan bulunabilsin.
func TestNoticeSlug(t *testing.T) {
	cases := map[string]string{
		"İstanbul 4. Sulh Hukuk Mahkemesi": "istanbul-4-sulh-hukuk-mahkemesi",
		"Türkiye'nin İlanı":                "turkiyenin-ilani",
		"  ":                               "ilan",
		"!!!":                              "ilan",
	}
	for in, want := range cases {
		assert.Equal(t, want, noticeSlug(in), "slug: %q", in)
	}
}

func TestNoticeSlugUzunBaslikKirpilir(t *testing.T) {
	s := noticeSlug(strings.Repeat("çok uzun bir ilan başlığı ", 20))
	require.LessOrEqual(t, len(s), 80)
	assert.NotEmpty(t, s)
	// Kırpma sonrası baştaki/sondaki ayraç kalmamalı; yoksa adreste "--"
	// gibi çirkin parçalar oluşur.
	assert.NotEqual(t, "-", s[:1])
	assert.NotEqual(t, "-", s[len(s)-1:])
}

func TestPublicNoticeURL(t *testing.T) {
	n := repo.OfficialNotice{ID: 42, Title: "Satılık Arsa İlanı"}
	assert.Equal(t, "/resmi-ilanlar/satilik-arsa-ilani-42", publicNoticeURL("", n))
	assert.Equal(t, "/en/resmi-ilanlar/satilik-arsa-ilani-42", publicNoticeURL("/en", n))
}

// Adres <slug>-<id> biçiminde; kimlik çözülemezse 404 verilir. Bu test
// adres ile ayrıştırmanın aynı kuralı paylaştığını doğrular.
func TestNoticeURLSegmentRoundTrip(t *testing.T) {
	n := repo.OfficialNotice{ID: 9137, Title: "İhale İlanı — Karayolları"}
	seg := strings.TrimPrefix(publicNoticeURL("", n), "/resmi-ilanlar/")
	assert.Equal(t, int64(9137), articleIDFromSegment(seg))
	assert.Equal(t, int64(0), articleIDFromSegment("ilan"))
	assert.Equal(t, int64(0), articleIDFromSegment("ilan-abc"))
}

// Kurum/şehir/arama daraltması yalnız en son güne hapsolmamalı; aksi hâlde
// "Ankara" seçen kullanıcı boş liste görüyordu.
func TestNoticeFilterFromQuery(t *testing.T) {
	f := noticeFilterFromQuery(url.Values{})
	assert.Empty(t, f.Date)
	assert.False(t, strings.EqualFold(f.Date, repo.NoticeAllDates))

	f = noticeFilterFromQuery(url.Values{"kurum": {"Ankara Büyükşehir"}})
	assert.Equal(t, "Ankara Büyükşehir", f.Institution)
	assert.True(t, strings.EqualFold(f.Date, repo.NoticeAllDates))

	f = noticeFilterFromQuery(url.Values{"sehir": {"İzmir"}})
	assert.True(t, strings.EqualFold(f.Date, repo.NoticeAllDates))

	f = noticeFilterFromQuery(url.Values{"q": {"ihale"}})
	assert.True(t, strings.EqualFold(f.Date, repo.NoticeAllDates))

	// Tarih açıkça seçilmişse daraltma o güne uygulanır.
	f = noticeFilterFromQuery(url.Values{"tarih": {"2026-10-28"}, "kurum": {"Belediye"}})
	assert.Equal(t, "2026-10-28", f.Date)
	assert.False(t, strings.EqualFold(f.Date, repo.NoticeAllDates))

	// "tumu" seçimi korunur.
	f = noticeFilterFromQuery(url.Values{"tarih": {"tumu"}})
	assert.Equal(t, repo.NoticeAllDates, f.Date)

	// Boşluklar kırpılır: panelden gelen tek boşluk filtre sayılmamalı.
	f = noticeFilterFromQuery(url.Values{"kurum": {"  "}})
	assert.Empty(t, f.Institution)
	assert.Empty(t, f.Date)
}

// Şablonların gerçekten derlenip çizilebildiğini doğrular: noticeSlug
// fonksiyonu FuncMap'e eklenmeyi unutulursa üretimde sayfa 500 verirdi.
func TestResmiIlanListeSablonu(t *testing.T) {
	tmpl := parseTestTemplates(t)
	d := &tmplData{
		Lang: "tr", LangBase: "", SiteName: "TheBlock",
		Header: &HeaderConfig{},
		Ilanlar: []repo.OfficialNotice{
			{ID: 7, Title: "İhale İlanı", Institution: "Belediye", City: "Ankara",
				NoticeDate: "2026-10-28", Body: "Metin", URL: "https://example.com/belge.pdf"},
			{ID: 8, Title: "Duyuru", NoticeDate: "2026-10-28"},
		},
		IlanTarih:    "2026-10-28",
		IlanTarihler: []string{"2026-10-28"},
		IlanKurumlar: []string{"Belediye"},
		IlanSehirler: []string{"Ankara"},
		IlanFiltre:   noticeFilterView{Tarih: "2026-10-28"},
	}
	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "resmi-ilanlar", d))
	out := buf.String()
	assert.Contains(t, out, "/resmi-ilanlar/ihale-ilani-7")
	assert.Contains(t, out, "/resmi-ilanlar/duyuru-8")
	assert.Contains(t, out, "Belediye")
	assert.Contains(t, out, "https://example.com/belge.pdf")
}

func TestResmiIlanDetaySablonu(t *testing.T) {
	tmpl := parseTestTemplates(t)
	d := &tmplData{
		Lang: "tr", LangBase: "", SiteName: "TheBlock",
		Header:    &HeaderConfig{},
		Canonical: "https://ornek.test/resmi-ilanlar/ihale-ilani-7",
		Ilan: &repo.OfficialNotice{
			ID: 7, Title: "İhale İlanı", Institution: "Belediye", City: "Ankara",
			NoticeDate: "2026-10-28", Body: "İlan metni", URL: "https://example.com/belge.pdf",
		},
	}
	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "resmi-ilan-detay", d))
	out := buf.String()
	assert.Contains(t, out, "İhale İlanı")
	assert.Contains(t, out, "İlan metni")
	assert.Contains(t, out, `href="https://ornek.test/resmi-ilanlar/ihale-ilani-7"`)
	assert.Contains(t, out, "/resmi-ilanlar?tarih=2026-10-28")
	assert.Contains(t, out, "/resmi-ilanlar?kurum=Belediye")
}
