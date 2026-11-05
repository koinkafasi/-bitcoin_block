package repo

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- HTTP mock yardımcıları ---------------------------------------
//
// ESPN/TheSportsDB uçları paket düzeyinde "var" olarak tanımlı; testlerde
// httptest sunucusuna yönlendirilir. Böylece hiçbir dış ağ çağrısı yapılmaz.

// withStubBase, paket değişkenlerini geçici olarak test sunucusuna bağlar.
func withStubBase(t *testing.T, h http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(h)

	oldESPN, oldTS := espnStandingsBase, tsAPI
	espnStandingsBase = srv.URL + "/%s/standings"
	tsAPI = srv.URL + "/%s/%s"
	t.Cleanup(func() {
		espnStandingsBase, tsAPI = oldESPN, oldTS
		srv.Close()
	})
	return srv
}

// espnEntryFixture, ESPN'in puan durumu girdisini üretir.
func espnEntryFixture(display, short string, logos []string, stats map[string]float64) map[string]any {
	st := make([]map[string]any, 0, len(stats))
	for k, v := range stats {
		st = append(st, map[string]any{"name": k, "value": v})
	}
	lg := make([]map[string]any, 0, len(logos))
	for _, h := range logos {
		lg = append(lg, map[string]any{"href": h})
	}
	return map[string]any{
		"team": map[string]any{
			"displayName":      display,
			"shortDisplayName": short,
			"logos":            lg,
		},
		"stats": st,
	}
}

// ---- ESPN puan durumu ---------------------------------------------

func TestFetchESPNStandingsSuccess(t *testing.T) {
	withStubBase(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/tur.1/standings", r.URL.Path, "lig kodu URL'e gömülmeli")
		assert.NotEmpty(t, r.Header.Get("User-Agent"), "bazı uçlar User-Agent zorunlu tutuyor")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"standings": map[string]any{"entries": []map[string]any{
				espnEntryFixture("Besiktas", "BJK",
					[]string{"https://a.espncdn.com/i/teamlogos/soccer/500/606.png"},
					map[string]float64{
						"rank": 1, "gamesPlayed": 7, "wins": 5, "ties": 2, "losses": 0,
						"pointsFor": 15, "pointsAgainst": 6, "pointDifferential": 9, "points": 17,
					}),
				espnEntryFixture("Fenerbahce", "FB",
					[]string{"https://a.espncdn.com/i/teamlogos/soccer/500/548.png"},
					map[string]float64{"rank": 2, "gamesPlayed": 7, "wins": 4, "points": 13}),
			}},
		})
	})

	rows := fetchESPNStandings("tur.1", 2*time.Second)
	require.Len(t, rows, 2)

	assert.Equal(t, "Beşiktaş", rows[0].Team, "ASCII takım adı Türkçeleştirilmeli")
	assert.Equal(t, "https://a.espncdn.com/i/teamlogos/soccer/500/606.png", rows[0].Badge)
	assert.Equal(t, 1, rows[0].Rank)
	assert.Equal(t, 7, rows[0].Played)
	assert.Equal(t, 5, rows[0].Win)
	assert.Equal(t, 2, rows[0].Draw)
	assert.Equal(t, 0, rows[0].Loss)
	assert.Equal(t, 15, rows[0].GF)
	assert.Equal(t, 6, rows[0].GA)
	assert.Equal(t, 9, rows[0].GD)
	assert.Equal(t, 17, rows[0].Points)

	assert.Equal(t, "Fenerbahçe", rows[1].Team)
	assert.Equal(t, 2, rows[1].Rank)
	assert.Equal(t, 13, rows[1].Points)
	// eksik istatistikler 0 olmalı, panik yok
	assert.Equal(t, 0, rows[1].Draw)
	assert.Equal(t, 0, rows[1].GA)
}

func TestFetchESPNStandingsChildrenFallback(t *testing.T) {
	// Bazı liglerde girdiler children[].standings altında gelir.
	withStubBase(t, func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"children": []map[string]any{
				{"standings": map[string]any{"entries": []map[string]any{
					espnEntryFixture("Kasimpasa", "KSM", nil, map[string]float64{"rank": 5, "points": 10}),
				}}},
			},
		})
	})
	rows := fetchESPNStandings("tur.1", 2*time.Second)
	require.Len(t, rows, 1, "children altındaki girdiler kullanılmalı")
	assert.Equal(t, "Kasımpaşa", rows[0].Team)
	assert.Equal(t, "", rows[0].Badge, "logo yoksa boş string, hata değil")
}

func TestFetchESPNStandingsShortDisplayNameFallback(t *testing.T) {
	withStubBase(t, func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"standings": map[string]any{"entries": []map[string]any{
				// displayName boş -> shortDisplayName kullanılmalı
				espnEntryFixture("", "GS", nil, map[string]float64{"rank": 1}),
			}},
		})
	})
	rows := fetchESPNStandings("tur.1", 2*time.Second)
	require.Len(t, rows, 1)
	assert.Equal(t, "GS", rows[0].Team)
}

func TestFetchESPNStandingsFailureModes(t *testing.T) {
	cases := []struct {
		name    string
		handler http.HandlerFunc
		timeout time.Duration
	}{
		{
			name: "bozuk JSON",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte("{{{not json"))
			},
			timeout: 2 * time.Second,
		},
		{
			name: "bos govde",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(""))
			},
			timeout: 2 * time.Second,
		},
		{
			name: "HTTP 500",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			timeout: 2 * time.Second,
		},
		{
			name: "HTTP 404",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			timeout: 2 * time.Second,
		},
		{
			name: "HTTP 429 (rate limit)",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusTooManyRequests)
			},
			timeout: 2 * time.Second,
		},
		{
			name: "timeout",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				time.Sleep(400 * time.Millisecond)
				_, _ = w.Write([]byte(`{}`))
			},
			timeout: 80 * time.Millisecond,
		},
		{
			name: "bos tablo (entries yok)",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(`{"standings":{"entries":[]}}`))
			},
			timeout: 2 * time.Second,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			withStubBase(t, c.handler)
			start := time.Now()
			rows := fetchESPNStandings("tur.1", c.timeout)
			elapsed := time.Since(start)

			// Sözleşme: satır dönmemeli (nil ya da boş). Çağıran taraf
			// (RefreshStandings) len(rows)==0 görünce demo veriye düşer.
			assert.Empty(t, rows, "hata durumunda satır dönmemeli")
			if c.name == "timeout" {
				assert.Less(t, elapsed, 350*time.Millisecond, "timeout'a saygı gösterilmeli")
			}
		})
	}
}

func TestFetchESPNStandingsNilStatValue(t *testing.T) {
	withStubBase(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"standings":{"entries":[{"team":{"displayName":"Galatasaray","logos":[{"href":"x.png"}]},"stats":[{"name":"points","value":null},{"name":"rank"}]}]}}`))
	})
	rows := fetchESPNStandings("tur.1", 2*time.Second)
	require.Len(t, rows, 1, "null değerli istatistik satırı düşürülmemeli")
	assert.Equal(t, 0, rows[0].Points)
	assert.Equal(t, 0, rows[0].Rank)
	assert.Equal(t, "Galatasaray", rows[0].Team)
}

func TestFetchESPNStandingsUnreachableHost(t *testing.T) {
	old := espnStandingsBase
	// kapalı/kullanilmayan adres -> bağlantı hatası
	espnStandingsBase = "http://127.0.0.1:1/%s/standings"
	t.Cleanup(func() { espnStandingsBase = old })

	assert.Nil(t, fetchESPNStandings("tur.1", 500*time.Millisecond), "erişilemeyen uç nil dönmeli, panic yok")
}

// ---- TheSportsDB hafta fikstürü ----------------------------------

func TestTSRoundSuccess(t *testing.T) {
	withStubBase(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/123/eventsround.php", r.URL.Path)
		assert.Equal(t, "4339", r.URL.Query().Get("id"))
		assert.Equal(t, "8", r.URL.Query().Get("r"))
		assert.Equal(t, "2026-2027", r.URL.Query().Get("s"))
		_, _ = w.Write([]byte(`{"events":[
			{"strTimestamp":"2026-10-16 17:00:00","strHomeTeam":"Galatasaray","strAwayTeam":"Rizespor",
			 "strHomeTeamBadge":"https://x/gs.png","strAwayTeamBadge":"https://x/riz.png",
			 "intHomeScore":"2","intAwayScore":"1","intRound":"8","strSeason":"2026-2027"},
			{"strTimestamp":"2026-10-17 19:00:00","strHomeTeam":"Fenerbahçe","strAwayTeam":"Beşiktaş",
			 "intHomeScore":null,"intAwayScore":null,"intRound":"8","strSeason":"2026-2027"}
		]}`))
	})
	evs := tsRound("123", "4339", 8, "2026-2027", 2*time.Second)
	require.Len(t, evs, 2)
	assert.Equal(t, "Galatasaray", evs[0].Home)
	assert.Equal(t, "2", evs[0].HomeScore)
	assert.Equal(t, "", evs[1].HomeScore, "null skor boş string olmalı")
}

func TestTSRoundFailureModes(t *testing.T) {
	cases := []struct {
		name    string
		handler http.HandlerFunc
		timeout time.Duration
	}{
		{"bozuk JSON", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("not json")) }, 2 * time.Second},
		{"bos events", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{"events":null}`)) }, 2 * time.Second},
		{"bos govde", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("")) }, 2 * time.Second},
		{"HTTP 500", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(500) }, 2 * time.Second},
		{"HTTP 401 (anahtar gecersiz)", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(401) }, 2 * time.Second},
		{"timeout", func(w http.ResponseWriter, _ *http.Request) { time.Sleep(400 * time.Millisecond) }, 80 * time.Millisecond},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			withStubBase(t, c.handler)
			assert.Nil(t, tsRound("123", "4339", 8, "2026-2027", c.timeout))
		})
	}
}

// ---- HTTP istemci yardımcısı -------------------------------------

func TestHTTPGetJSONUA(t *testing.T) {
	cases := []struct {
		name    string
		handler http.HandlerFunc
		want    string
		wantErr bool
	}{
		{"basarili", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{"ok":true}`)) }, `{"ok":true}`, false},
		{"404", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(404) }, "", true},
		{"500", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(500) }, "", true},
		{"bos govde", func(w http.ResponseWriter, _ *http.Request) {}, "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			srv := httptest.NewServer(c.handler)
			defer srv.Close()
			got, err := httpGetJSONUA(srv.URL, time.Second)
			if c.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, c.want, string(got))
			}
		})
	}
}

func TestHTTPGetJSONUAInvalidURL(t *testing.T) {
	_, err := httpGetJSONUA("://bozurl", time.Second)
	assert.Error(t, err, "geçersiz URL hata vermeli")
}

func TestHTTPGetJSONUALargeBodyTruncated(t *testing.T) {
	// 2MB sınırı: devasa gövde sınırsız okunmamalı
	big := make([]byte, 4<<20)
	for i := range big {
		big[i] = 'a'
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(big)
	}))
	defer srv.Close()
	got, err := httpGetJSONUA(srv.URL, 5*time.Second)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(got), 2<<20, "gövde 2MB ile sınırlanmalı")
}
