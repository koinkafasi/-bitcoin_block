package repo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMirrorSourceURLEspnCDNYenidenYazilir(t *testing.T) {
	cases := map[string]string{
		"https://a.espncdn.com/i/teamlogos/soccer/500/432.png":     "https://cdn.espn.com/i/teamlogos/soccer/500/432.png",
		"https://a1.espncdn.com/i/teamlogos/soccer/500/432.png":    "https://cdn.espn.com/i/teamlogos/soccer/500/432.png",
		"https://a4.espncdn.com/i/teamlogos/soccer/500/132335.png": "https://cdn.espn.com/i/teamlogos/soccer/500/132335.png",
	}
	for in, want := range cases {
		assert.Equal(t, want, mirrorSourceURL(in), "girdi: %s", in)
	}
}

func TestMirrorSourceURLDigerKaynaklarKorunur(t *testing.T) {
	for _, in := range []string{
		"https://r2.thesportsdb.com/images/media/team/badge/2giize1534005340.png",
		"https://cdn.espn.com/i/teamlogos/soccer/500/432.png",
		"https://ornek.com/logo.png",
		"bozuk url",
	} {
		assert.Equal(t, in, mirrorSourceURL(in))
	}
}
