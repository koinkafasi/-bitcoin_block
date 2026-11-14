package httpapi

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Funnel sayfası testleri — fiyat/plan ve yapısal veri tutarlılığı.
//
// NEDEN: sayfa paneldeki planları ELLE taşıyor (ayrı uygulama, ayrı
// veritabanı). Fiyat dizgesi ile JSON-LD'deki teklif birbirinden koparsa
// Google zengin sonuçta yanlış fiyat gösterir; iki kaynağı burada
// karşılaştırıyoruz.

func TestFunnelPlanFiyatlariJSONLDIleAyni(t *testing.T) {
	planlar := funnelPlanlar()
	require.Len(t, planlar, 3)

	js := funnelJSONLD("BİTCOŞAR", planlar, funnelSSS())
	govde := strings.TrimSuffix(strings.TrimPrefix(js, `<script type="application/ld+json">`), `</script>`)

	var v struct {
		Graph []map[string]any `json:"@graph"`
	}
	require.NoError(t, json.Unmarshal([]byte(govde), &v))
	require.Len(t, v.Graph, 2)

	var urun map[string]any
	for _, g := range v.Graph {
		if g["@type"] == "Product" {
			urun = g
		}
	}
	require.NotNil(t, urun, "Product bloğu yok")

	teklifler, ok := urun["offers"].([]any)
	require.True(t, ok, "offers dizisi yok")
	require.Len(t, teklifler, len(planlar))

	for i, p := range planlar {
		of, ok := teklifler[i].(map[string]any)
		require.True(t, ok)
		// Sayfadaki "$60" ile JSON-LD'deki "60" aynı olmalı.
		assert.Equal(t, strings.TrimPrefix(p.Tutar, "$"), of["price"], p.Ad)
		assert.Equal(t, "USD", of["priceCurrency"], p.Ad)
	}
}

func TestFunnelSSSCevaplariJSONLDIleAyni(t *testing.T) {
	sss := funnelSSS()
	require.NotEmpty(t, sss)

	js := funnelJSONLD("BİTCOŞAR", funnelPlanlar(), sss)
	govde := strings.TrimSuffix(strings.TrimPrefix(js, `<script type="application/ld+json">`), `</script>`)

	var v struct {
		Graph []struct {
			Type  string `json:"@type"`
			Items []struct {
				Name  string `json:"name"`
				Cevap struct {
					Text string `json:"text"`
				} `json:"acceptedAnswer"`
			} `json:"mainEntity"`
		} `json:"@graph"`
	}
	require.NoError(t, json.Unmarshal([]byte(govde), &v))

	var faq []struct {
		Name  string `json:"name"`
		Cevap struct {
			Text string `json:"text"`
		} `json:"acceptedAnswer"`
	}
	for _, g := range v.Graph {
		if g.Type == "FAQPage" {
			faq = g.Items
		}
	}
	require.Len(t, faq, len(sss), "FAQ sayısı sayfadakiyle aynı olmalı")

	for i, s := range sss {
		assert.Equal(t, s.Soru, faq[i].Name)
		assert.Equal(t, s.Cevap, faq[i].Cevap.Text)
	}
}

// TestFunnelPlanCtaPanelAdresi: plan düğmeleri traders.bitcosar.com'a
// gitmeli. Göreli bir yol ya da boş bağlantı, ziyaretçiyi haber sitesinde
// bırakıyordu.
func TestFunnelPlanCtaPanelAdresi(t *testing.T) {
	for _, p := range funnelPlanlar() {
		assert.True(t, strings.HasPrefix(p.CtaHref, funnelPanelKok+"/"),
			"%s planı panel dışına gidiyor: %s", p.Ad, p.CtaHref)
	}
}
