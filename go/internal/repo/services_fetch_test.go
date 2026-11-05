package repo

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- sezon hesabı (Tem–Haz sınırı) ----

func TestSeasonOf(t *testing.T) {
	cases := []struct {
		name string
		in   time.Time
		want string
	}{
		{"ekim: yeni sezon", time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC), "2026-2027"},
		{"temil: sezon başı", time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), "2026-2027"},
		{"haziran sonu: eski sezon", time.Date(2026, 6, 30, 23, 59, 0, 0, time.UTC), "2025-2026"},
		{"ocak: ara", time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC), "2025-2026"},
		{"aralik sonu: devam", time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC), "2025-2026"},
		{"sifir zaman (yil 1, ocak)", time.Time{}, "0-1"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, seasonOf(c.in))
		})
	}
}

func TestPrevSeason(t *testing.T) {
	cases := []struct {
		name string
		in   time.Time
		want string
	}{
		{"ekim onceki", time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC), "2025-2026"},
		{"haziran onceki", time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), "2024-2025"},
		{"sifir zaman (yil 1)", time.Time{}, "-1-0"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, prevSeason(c.in))
		})
	}
}

// ---- atoiSafe: hatalı girdide 0 dönmeli, patlamamalı ----

func TestAtoiSafe(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want int
	}{
		{"normal", "18", 18},
		{"sifir", "0", 0},
		{"bos", "", 0},
		{"sayi degil", "abc", 0},
		{"karisik", "12x", 0},
		{"negatif", "-3", -3},
		{"bosluk kirpilir", " 7 ", 7},
		{"ondalik", "3.5", 0},
		{"tasma -> 0 (MaxInt64 degil)", "99999999999999999999", 0},
		{"yazi arasi", "1 2", 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, atoiSafe(c.in))
		})
	}
}

// ---- badgeURL ----
//
// Not: bu kod yolu ESPN puan durumu devreye girdikten sonra çağrılmıyor
// (legacy TheSportsDB lookuptable). Test, belgelenen GERÇEK davranışı
// sabitler: yalnızca sondaki "/tiny" soneki temizlenir.

func TestBadgeURL(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"sondaki /tiny kirpilir", "https://x/logo/tiny", "https://x/logo"},
		{"tiny yoksa dokunma", "https://x/logo.png", "https://x/logo.png"},
		{"bosluk kirpilir", "  https://x/logo/tiny  ", "https://x/logo"},
		{"bos", "", ""},
		{"sadece /tiny", "/tiny", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, badgeURL(c.in))
		})
	}
}

// ---- trTeamName: ASCII -> Türkçe ----

func TestTrTeamName(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"eslesme", "Besiktas", "Beşiktaş"},
		{"fenerbahce", "Fenerbahce", "Fenerbahçe"},
		{"basaksehir", "Istanbul Basaksehir", "İstanbul Başakşehir"},
		{"bilinmeyen oldugu gibi kalir", "Adana Demirspor", "Adana Demirspor"},
		{"bos", "", ""},
		{"buyuk-kucuk duyarli", "besiktas", "besiktas"},
		{"kisaltma eslesmez", "GS", "GS"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, trTeamName(c.in))
		})
	}
}

// ---- lig anahtari ----

func TestLeagueByKey(t *testing.T) {
	cases := []struct {
		name       string
		in         string
		wantKey    string
		wantRounds int
	}{
		{"superlig", "superlig", "superlig", 34},
		{"premierlig", "premierlig", "premierlig", 38},
		{"bilinmeyen -> superlig", "xyz", "superlig", 34},
		{"bos -> superlig", "", "superlig", 34},
		{"buyuk harf -> superlig", "BUNDESLIGA", "superlig", 34},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := LeagueByKey(c.in)
			assert.Equal(t, c.wantKey, got.Key)
			assert.Equal(t, c.wantRounds, got.Rounds)
			assert.NotEmpty(t, got.ESPN)
			assert.NotEmpty(t, got.TSDB)
		})
	}
}

func TestLeaguesHaveUniqueKeys(t *testing.T) {
	seen := map[string]bool{}
	for _, l := range Leagues {
		require.False(t, seen[l.Key], "lig anahtarı yineleniyor: %s", l.Key)
		seen[l.Key] = true
		assert.Positive(t, l.Rounds)
	}
}

// ---- puan durumu anahtarlari (cache izolasyonu) ----

func TestCacheKeysAreolatedPerLeague(t *testing.T) {
	seen := map[string]bool{}
	for _, l := range Leagues {
		k := standingsKey(l)
		require.False(t, seen[k], "puan onbellegi anahtari cakisiyor: %s", k)
		seen[k] = true
		assert.Equal(t, "L:"+l.Key, k)
	}
}

func TestFixturesRoundKey(t *testing.T) {
	lg := LeagueByKey("premierlig")
	assert.Equal(t, "premierlig:2026-2027:12", fixturesRoundKey(lg, "2026-2027", 12))
	// farkli lig ayni sezon/hafta ile cakismamali
	assert.NotEqual(t,
		fixturesRoundKey(LeagueByKey("superlig"), "2026-2027", 12),
		fixturesRoundKey(LeagueByKey("premierlig"), "2026-2027", 12))
}

// ---- manuel puan durumu ayristirma ----

func TestParseManualStandings(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		wantLen int
		check   func(t *testing.T, rows []StandingRow)
	}{
		{
			name:    "tam satir",
			in:      "Galatasaray | https://x/gs.png | 7 | 5 | 0 | 9 | 15",
			wantLen: 1,
			check: func(t *testing.T, rows []StandingRow) {
				r := rows[0]
				assert.Equal(t, "Galatasaray", r.Team)
				assert.Equal(t, "https://x/gs.png", r.Badge)
				assert.Equal(t, 7, r.Played)
				assert.Equal(t, 5, r.Win)
				assert.Equal(t, 0, r.Loss)
				assert.Equal(t, 9, r.GD)
				assert.Equal(t, 15, r.Points)
				assert.Equal(t, 1, r.Rank)
			},
		},
		{
			name:    "bos girdi",
			in:      "",
			wantLen: 0,
		},
		{
			name:    "yalnizca bosluk",
			in:      "   \n\t\n  ",
			wantLen: 0,
		},
		{
			name:    "yorum satirlari atlanir",
			in:      "# 2026-2027 sezonu\n# elle girildi\n\nGalatasaray | | 7 | 5 | 0 | 2 | 8 | 15",
			wantLen: 1,
			check: func(t *testing.T, rows []StandingRow) {
				assert.Equal(t, "Galatasaray", rows[0].Team)
			},
		},
		{
			name:    "eksik alanlar sifir",
			in:      "Fenerbahçe",
			wantLen: 1,
			check: func(t *testing.T, rows []StandingRow) {
				r := rows[0]
				assert.Equal(t, "Fenerbahçe", r.Team)
				assert.Equal(t, 0, r.Played)
				assert.Equal(t, 0, r.Points)
				assert.Equal(t, "", r.Badge)
			},
		},
		{
			name:    "sayi olmayan alanlar sifir (hata degil)",
			in:      "Beşiktaş | logo.png | oyun | galibiyet | maglubiyet | averaj | puan",
			wantLen: 1,
			check: func(t *testing.T, rows []StandingRow) {
				assert.Equal(t, 0, rows[0].Played)
				assert.Equal(t, 0, rows[0].Points)
				assert.Equal(t, "logo.png", rows[0].Badge)
			},
		},
		{
			name:    "sira otomatik artar",
			in:      "A | | 1 | 1 | 0 | 0 | 1 | 3\nB | | 1 | 0 | 1 | 0 | 1 | 1\nC | | 1 | 0 | 1 | 0 | 2 | 1",
			wantLen: 3,
			check: func(t *testing.T, rows []StandingRow) {
				assert.Equal(t, 1, rows[0].Rank)
				assert.Equal(t, 2, rows[1].Rank)
				assert.Equal(t, 3, rows[2].Rank)
			},
		},
		{
			name:    "takim adi bos olan satir atlanir",
			in:      "  | logo | 1 | 1 | 0 | 0 | 1 | 3\nGalatasaray | | 1 | 1 | 0 | 0 | 1 | 3",
			wantLen: 1,
			check: func(t *testing.T, rows []StandingRow) {
				assert.Equal(t, "Galatasaray", rows[0].Team)
			},
		},
		{
			name:    "CRLF satir sonlari",
			in:      "A | | 1 | 1 | 0 | 0 | 1 | 3\r\nB | | 1 | 0 | 1 | 0 | 0 | 0",
			wantLen: 2,
			check: func(t *testing.T, rows []StandingRow) {
				assert.Equal(t, "A", rows[0].Team)
				assert.Equal(t, "B", rows[1].Team)
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rows := parseManualStandings(c.in)
			require.Len(t, rows, c.wantLen)
			if c.check != nil {
				c.check(t, rows)
			}
		})
	}
}

// ---- TheSportsDB satirindan fiksture ----

func TestToFixtures(t *testing.T) {
	played := "2"
	evs := []tsRoundEvent{
		{
			Timestamp: "2026-10-16 17:00:00",
			Home:      "Galatasaray", Away: "Rizespor",
			HomeBadge: "https://x/gs.png/tiny", AwayBadge: "https://x/riz.png/tiny",
			HomeScore: played, AwayScore: "1",
		},
		{
			Timestamp: "2026-10-17", // saat yok
			Home:      "Fenerbahçe", Away: "Beşiktaş",
		},
	}
	fr := toFixtures(evs, 8, "2026-2027")
	require.Len(t, fr.Matches, 2)
	assert.Equal(t, 8, fr.Round)
	assert.Equal(t, "2026-2027", fr.Season)

	ilk := fr.Matches[0]
	assert.Equal(t, "2026-10-16", ilk.Date)
	assert.Equal(t, "17:00", ilk.Time)
	require.NotNil(t, ilk.HomeScore)
	assert.Equal(t, 2, *ilk.HomeScore)
	assert.Equal(t, 1, *ilk.AwayScore)
	assert.True(t, ilk.Played)

	ikinci := fr.Matches[1]
	assert.Equal(t, "2026-10-17", ikinci.Date)
	assert.Equal(t, "", ikinci.Time)
	assert.Nil(t, ikinci.HomeScore, "oynanmamis macda skor olmamali")
	assert.False(t, ikinci.Played)
}

func TestToFixturesEmptyAndBadScores(t *testing.T) {
	cases := []struct {
		name string
		in   []tsRoundEvent
	}{
		{"bos liste", nil},
		{"bos skor (oynanmamis)", []tsRoundEvent{{Timestamp: "2026-10-16 17:00", Home: "A", Away: "B"}}},
		{"sayi olmayan skor", []tsRoundEvent{{Timestamp: "2026-10-16 17:00", Home: "A", Away: "B", HomeScore: "x", AwayScore: "y"}}},
		{"bos takim adi", []tsRoundEvent{{Timestamp: "2026-10-16 17:00"}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fr := toFixtures(c.in, 1, "2026-2027")
			assert.Equal(t, 1, fr.Round)
			assert.Equal(t, "2026-2027", fr.Season)
			for _, m := range fr.Matches {
				assert.False(t, m.Played, "skor yoksa mac oynanmis sayilmamali")
				assert.Nil(t, m.HomeScore)
				assert.Nil(t, m.AwayScore)
			}
		})
	}
}

// ---- toStandingRow ----

func TestToStandingRow(t *testing.T) {
	r := toStandingRow(tsStanding{
		Rank: "1", Team: "  Galatasaray ", Badge: "https://x/gs.png/tiny",
		Played: "7", Win: "5", Draw: "2", Loss: "0",
		GF: "15", GA: "6", GD: "9", Points: "17", Form: " G G",
	})
	assert.Equal(t, 1, r.Rank)
	assert.Equal(t, "Galatasaray", r.Team)
	assert.Equal(t, "https://x/gs.png", r.Badge)
	assert.Equal(t, 7, r.Played)
	assert.Equal(t, 5, r.Win)
	assert.Equal(t, 2, r.Draw)
	assert.Equal(t, 17, r.Points)
	assert.Equal(t, "G G", r.Form)
}

func TestToStandingRowBadInput(t *testing.T) {
	r := toStandingRow(tsStanding{})
	assert.Equal(t, 0, r.Rank)
	assert.Equal(t, "", r.Team)
	assert.Equal(t, 0, r.Points)
}

// ---- demo puan durumu tutarli mi ----

func TestDemoStandingsShape(t *testing.T) {
	rows := demoStandings()
	require.NotEmpty(t, rows)
	for i, r := range rows {
		assert.Equal(t, i+1, r.Rank, "sira ardisik olmali")
		assert.NotEmpty(t, r.Team)
	}
}
