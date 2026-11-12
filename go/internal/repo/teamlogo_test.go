package repo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"theblock/site/internal/platform"
)

func TestTeamLogoKeyESPNKimligi(t *testing.T) {
	// ESPN URL'lerinde sayısal kimlik var; anahtar ondan üretilir. Böylece
	// takım adı Türkçeye çevrilse de anahtar sabit kalır.
	cases := map[string]string{
		"https://a.espncdn.com/i/teamlogos/soccer/500/432.png":    "e-432",
		"https://a.espncdn.com/i/teamlogos/soccer/500/132335.png": "e-132335",
		"https://cdn.espn.com/i/teamlogos/soccer/500/20070.png":   "e-20070",
		"https://a.espncdn.com/i/teamlogos/cricket/500/997.png":   "e-997",
	}
	for url, want := range cases {
		assert.Equal(t, want, TeamLogoKey(url, "alakasız ad"), "url: %s", url)
	}
}

func TestTeamLogoKeyAdaGoreSlug(t *testing.T) {
	// TheSportsDB URL'i opak bir hash içeriyor; anahtar takım adından üretilir.
	url := "https://r2.thesportsdb.com/images/media/team/badge/2giize1534005340.png"
	assert.Equal(t, "t-galatasaray", TeamLogoKey(url, "Galatasaray"))
	assert.Equal(t, "t-fenerbahce", TeamLogoKey(url, "Fenerbahçe"))
	assert.Equal(t, "t-besiktas", TeamLogoKey(url, "Beşiktaş"))
	assert.Equal(t, "t-caykur-rizespor", TeamLogoKey(url, "Çaykur Rizespor"))
	assert.Equal(t, "t-istanbul-basaksehir", TeamLogoKey(url, "İstanbul Başakşehir"))
}

func TestTeamLogoKeyBosAda(t *testing.T) {
	assert.Equal(t, "", TeamLogoKey("https://ornek/logo.png", ""))
	assert.Equal(t, "", TeamLogoKey("", ""))
}

func TestSlugKeySinirUzunluk(t *testing.T) {
	// Aşırı uzun adlar dosya sistemi sınırına takılmasın.
	uzun := "Çok Uzun Bir Takım Adı Buraya Kadar Devam Ediyor Ve Daha Da Uzuyor"
	got := slugKey(uzun)
	assert.LessOrEqual(t, len(got), 40)
	assert.NotContains(t, got, "--")
}

func TestSlugKeyYalnizSembol(t *testing.T) {
	// Ad yalnızca sembolden oluşuyorsa anahtar üretilmemeli (boş dosya adı).
	assert.Equal(t, "", slugKey("***"))
	assert.Equal(t, "", slugKey("   "))
}

func TestTeamLogoURL(t *testing.T) {
	assert.Equal(t, "/takim-logo/e-432.png", TeamLogoURL("e-432"))
	assert.Equal(t, "", TeamLogoURL(""))
}

func testPNG() []byte {
	return append([]byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}, make([]byte, 600)...)
}

func withTempStorage(t *testing.T) {
	t.Helper()
	old := platform.StorageDir()
	platform.SetStorageDir(t.TempDir())
	t.Cleanup(func() { platform.SetStorageDir(old) })
}

func TestMirrorTeamLogoIndirirVeYerelDoner(t *testing.T) {
	withTempStorage(t)
	png := testPNG()
	var istek int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		istek++
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(png)
	}))
	defer srv.Close()

	old := teamLogoHTTP
	teamLogoHTTP = srv.Client()
	defer func() { teamLogoHTTP = old }()

	got := mirrorTeamLogo(context.Background(), "e-999", srv.URL+"/432.png")
	assert.Equal(t, "/takim-logo/e-999.png", got)
	assert.Equal(t, 1, istek)
	assert.True(t, platform.HasTeamLogo("e-999"))

	// İkinci çağrı ağa çıkmamalı: sayfa yenilemesi indirme tetiklemez.
	got = mirrorTeamLogo(context.Background(), "e-999", srv.URL+"/432.png")
	assert.Equal(t, "/takim-logo/e-999.png", got)
	assert.Equal(t, 1, istek, "diskte varken indirme yapılmamalı")
}

func TestMirrorTeamLogoHataDurumundaBosDoner(t *testing.T) {
	withTempStorage(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// CDN hata sayfası dönerse logo gibi kaydedilmemeli.
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("<html>403</html>"))
	}))
	defer srv.Close()
	old := teamLogoHTTP
	teamLogoHTTP = srv.Client()
	defer func() { teamLogoHTTP = old }()

	got := mirrorTeamLogo(context.Background(), "e-404", srv.URL+"/x.png")
	assert.Equal(t, "", got, "başarısız indirmede çağıran dış URL'i korur")
	assert.False(t, platform.HasTeamLogo("e-404"))
}

func TestMirrorTeamLogoHTMLGovdeyiKaydetmez(t *testing.T) {
	withTempStorage(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("<!DOCTYPE html><html>engellendi</html>"))
	}))
	defer srv.Close()
	old := teamLogoHTTP
	teamLogoHTTP = srv.Client()
	defer func() { teamLogoHTTP = old }()

	assert.Equal(t, "", mirrorTeamLogo(context.Background(), "e-html", srv.URL+"/x.png"))
	assert.False(t, platform.HasTeamLogo("e-html"))
}

func TestMirrorStandingBadgesKarismaz(t *testing.T) {
	withTempStorage(t)
	png := testPNG()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(png)
	}))
	defer srv.Close()
	old := teamLogoHTTP
	teamLogoHTTP = srv.Client()
	defer func() { teamLogoHTTP = old }()

	rows := []StandingRow{
		{Team: "Galatasaray", Badge: srv.URL + "/teamlogos/soccer/500/432.png"},
		{Team: "Logosuz", Badge: ""},
	}
	mirrorStandingBadges(context.Background(), rows)
	assert.Equal(t, "/takim-logo/e-432.png", rows[0].Badge)
	assert.Equal(t, "", rows[1].Badge)
}

func TestMirrorFixtureBadgesEvSaha(t *testing.T) {
	withTempStorage(t)
	png := testPNG()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(png)
	}))
	defer srv.Close()
	old := teamLogoHTTP
	teamLogoHTTP = srv.Client()
	defer func() { teamLogoHTTP = old }()

	fr := FixtureRound{Matches: []Fixture{{
		Home: "Galatasaray", Away: "Fenerbahçe",
		HomeBadge: srv.URL + "/badge/aaa.png",
		AwayBadge: srv.URL + "/badge/bbb.png",
	}}}
	mirrorFixtureBadges(context.Background(), &fr)
	assert.Equal(t, "/takim-logo/t-galatasaray.png", fr.Matches[0].HomeBadge)
	assert.Equal(t, "/takim-logo/t-fenerbahce.png", fr.Matches[0].AwayBadge)
}

func TestMirrorTeamLogoZamanAsimi(t *testing.T) {
	if testing.Short() {
		t.Skip("ağ/zaman testi")
	}
	withTempStorage(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
	}))
	defer srv.Close()
	old := teamLogoHTTP
	teamLogoHTTP = &http.Client{Timeout: 20 * time.Millisecond}
	defer func() { teamLogoHTTP = old }()

	assert.Equal(t, "", mirrorTeamLogo(context.Background(), "e-yavas", srv.URL+"/x.png"))
}
