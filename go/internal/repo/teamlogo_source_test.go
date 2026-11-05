package repo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ESPN rozet adresi indirme anında erişilebilir ana bilgisayara çevrilir.
// Ölçüm (28 Ekim 2026): a.espncdn.com sunucudan 18/18 "connection reset"
// döndü; aynı yol cdn.espn.com üzerinden 200 ve birebir aynı dosya.

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
