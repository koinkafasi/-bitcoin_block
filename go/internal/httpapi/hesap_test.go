package httpapi

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/repo"
)

func hesapTestData(tab string) *tmplData {
	d := headerData()
	d.MemberLoggedIn = true
	d.MemberName = "Ayşe Yılmaz"
	d.MemberInitial = "A"
	d.MemberURL = "/hesabim"
	d.MetaTitle = "Hesabım"
	d.Hesap = &hesapData{
		Tab: tab,
		Member: repo.Member{
			ID: 7, Name: "Ayşe Yılmaz", Email: "ayse@example.com",
			Membership: "premium",
		},
		Stats:   repo.MemberStats{Bookmarks: 3, Comments: 2, Follows: 1, Unread: 4, Points: 40, Streak: 2},
		Socials: map[string]string{"x": "https://x.com/ayse"},
		Avatars: []hesapAvatar{{ID: 11, URL: "/media/11.jpg", Title: "portre"}},
	}
	return d
}

func renderHesap(t *testing.T, name string, d *tmplData) string {
	t.Helper()
	tmpl := parseTestTemplates(t)
	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, name, d))
	return buf.String()
}

func TestHesapHubRenders(t *testing.T) {
	out := renderHesap(t, "hesap-hub", hesapTestData("hub"))
	assert.Contains(t, out, "Merhaba, Ayşe Yılmaz")
	assert.Contains(t, out, "Premium Üye")
	assert.Contains(t, out, `href="/hesabim/profil"`)
	assert.Contains(t, out, `href="/hesabim/abonelik"`)
	assert.Contains(t, out, `href="/hesabim/kaydedilenler"`)
	assert.Contains(t, out, `aria-current="page"`)
	assert.Contains(t, out, "hesap-nav")
}

func TestHesapProfilFormRenders(t *testing.T) {
	out := renderHesap(t, "hesap-profil", hesapTestData("profil"))
	assert.Contains(t, out, `action="/hesabim/profil"`)
	assert.Contains(t, out, `name="s_x"`)
	assert.Contains(t, out, `value="https://x.com/ayse"`)
	assert.Contains(t, out, `action="/hesabim/sifre"`)
	assert.Contains(t, out, `action="/hesabim/sil"`)
	assert.Contains(t, out, `name="onay"`)
	assert.Contains(t, out, "medya")
}

func TestHesapAbonelikVeYorumlar(t *testing.T) {
	d := hesapTestData("yorumlar")
	d.Hesap.Comments = []repo.MemberComment{{
		ID: 1, Body: "Harika haber", Status: "onayli",
		CreatedAt: time.Date(2026, 3, 4, 10, 0, 0, 0, time.UTC),
		ContentID: 9, ContentSlug: "ornek-haber", ContentTitle: "Örnek Haber",
	}}
	out := renderHesap(t, "hesap-yorumlar", d)
	assert.Contains(t, out, "Harika haber")
	assert.Contains(t, out, "Örnek Haber")
	assert.Contains(t, out, "04.03.2026")

	ab := renderHesap(t, "hesap-abonelik", hesapTestData("abonelik"))
	assert.Contains(t, ab, "Ücretsiz üyelik")
	assert.Contains(t, ab, "Plan ayrıntıları")
	assert.Contains(t, ab, "/abonelik")
}

func TestHesapTakipBirakma(t *testing.T) {
	d := hesapTestData("takip")
	d.Hesap.Follows = []repo.Follow{{ID: 1, Type: "kategori", Slug: "ekonomi"}}
	out := renderHesap(t, "hesap-takip", d)
	assert.Contains(t, out, "data-takip-birak=\"kategori\"")
	assert.Contains(t, out, "/kategori/ekonomi")
}

func TestSlugifyHesap(t *testing.T) {
	cases := map[string]string{
		"Ayşe Yılmaz":      "ayse-yilmaz",
		"  Çağrı  ÖZTÜRK ": "cagri-ozturk",
		"a__b":             "a-b",
		"--x--":            "x",
		"":                 "",
	}
	for in, want := range cases {
		assert.Equal(t, want, slugifyHesap(in), "girdi: %q", in)
	}
}

func TestHesapSosyalURL(t *testing.T) {
	assert.Equal(t, "https://x.com/ayse", hesapSosyalURL("x", "ayse"))
	assert.Equal(t, "https://t.me/ayse", hesapSosyalURL("telegram", "ayse"))
	assert.Equal(t, "https://youtube.com/@ayse", hesapSosyalURL("youtube", "ayse"))
	assert.Equal(t, "https://ayse.dev", hesapSosyalURL("website", "ayse.dev"))
}

func TestHesapMesaj(t *testing.T) {
	assert.Equal(t, "Profiliniz kaydedildi.", hesapMesaj("profil"))
	assert.Equal(t, "Şifreniz güncellendi.", hesapMesaj("sifre"))
	assert.Equal(t, "", hesapMesaj("bilinmeyen"))
}
