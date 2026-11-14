package templkit

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Konum listesi haritayla bağlı olmalı: çerçeve ilk ziyaretçiyle açılır,
// her satırın bağlantısı aynı çerçeveyi hedefler (betik yok).
func TestZiyaretcilerHarita(t *testing.T) {
	const gom1 = "https://www.google.com/maps?q=41.0082,28.9784&z=15&output=embed"
	const gom2 = "https://www.google.com/maps?q=39.9334,32.8597&z=15&output=embed"
	v := ZiyaretcilerView{
		Gun: 30,
		Konumlular: []ZiyaretciSatir{
			{Ulke: "Türkiye", Sehir: "İstanbul", Enlem: "41.0082", Boylam: "28.9784", HaritaGomURL: gom1},
			{Ulke: "Türkiye", Sehir: "Ankara", Enlem: "39.9334", Boylam: "32.8597", HaritaGomURL: gom2},
		},
	}
	var b strings.Builder
	require.NoError(t, ZiyaretcilerPanel(v).Render(context.Background(), &b))
	html := strings.ReplaceAll(b.String(), "&amp;", "&")

	require.Equal(t, 1, strings.Count(html, `name="zv-harita"`), "tek harita çerçevesi")
	require.Contains(t, html, `src="`+gom1+`"`, "çerçeve ilk ziyaretçiyle açılmalı")
	require.Equal(t, 2, strings.Count(html, `target="zv-harita"`), "her satır çerçeveyi hedeflemeli")
	require.Contains(t, html, `href="`+gom2+`"`)
}

// Konum veren yoksa boş çerçeve basılmamalı.
func TestZiyaretcilerHaritaBos(t *testing.T) {
	var b strings.Builder
	require.NoError(t, ZiyaretcilerPanel(ZiyaretcilerView{Gun: 7}).Render(context.Background(), &b))
	require.NotContains(t, b.String(), "zv-harita")
}
