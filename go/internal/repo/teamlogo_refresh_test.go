package repo

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/platform"
)

func TestRefreshStandingsTumLigleriYazar(t *testing.T) {
	withTempStorage(t)
	srv := espnTestServer(t)
	defer srv.Close()
	espnTestBase(t, srv)
	logoTestClient(t, srv)

	db := &fakeDB{}
	require.NoError(t, RefreshStandings(context.Background(), db))

	yazilan := map[string]bool{}
	for _, c := range db.ExecCalls() {
		if len(c.Args) >= 2 {
			if key, ok := c.Args[1].(string); ok {
				yazilan[key] = true
			}
		}
	}
	for _, lg := range Leagues {
		assert.True(t, yazilan[standingsKey(lg)], "%s ligi önbelleğe yazılmalıydı", lg.Key)
	}
}

// Yerelleştirme sonrası rozet yerel yola dönmeli; dış URL kalmamalı.
func TestRefreshStandingsRozetleriYerellestirir(t *testing.T) {
	withTempStorage(t)
	srv := espnTestServer(t)
	defer srv.Close()
	espnTestBase(t, srv)
	logoTestClient(t, srv)

	db := &fakeDB{}
	require.NoError(t, RefreshStandings(context.Background(), db))

	var bulundu bool
	for _, c := range db.ExecCalls() {
		if len(c.Args) < 3 {
			continue
		}
		b, ok := c.Args[2].([]byte)
		if !ok {
			continue
		}
		var sd StandingsData
		if json.Unmarshal(b, &sd) != nil || len(sd.Rows) == 0 {
			continue
		}
		bulundu = true
		assert.Equal(t, "/takim-logo/e-111.png", sd.Rows[0].Badge,
			"rozet yerel yola çevrilmeliydi")
	}
	assert.True(t, bulundu, "en az bir lig yazılmalıydı")
	assert.True(t, platform.HasTeamLogo("e-111"), "logo diske yazılmalıydı")
}

func espnTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".png") {
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(testPNG())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"standings":{"entries":[
			{"team":{"displayName":"Takim A","logos":[{"href":"` + "http://" + r.Host + `/i/teamlogos/soccer/500/111.png"}]},
			 "stats":[{"name":"rank","value":1},{"name":"points","value":9}]}
		]}}`))
	}))
}

func espnTestBase(t *testing.T, srv *httptest.Server) {
	t.Helper()
	eski := espnStandingsBase
	espnStandingsBase = srv.URL + "/%s"
	t.Cleanup(func() { espnStandingsBase = eski })
}

func logoTestClient(t *testing.T, srv *httptest.Server) {
	t.Helper()
	eski := teamLogoHTTP
	teamLogoHTTP = srv.Client()
	t.Cleanup(func() { teamLogoHTTP = eski })
}
