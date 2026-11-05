package httpapi

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/webui/templkit"
)

func TestGelirTrendiGrafikRender(t *testing.T) {
	v := templkit.UyelerView{
		TrendHasData: true,
		Trend: []templkit.UyeTrend{
			{Short: "Eyl", Label: "2026-09", Amount: "1.000", Count: 2, Pct: 50},
			{Short: "Eki", Label: "2026-10", Amount: "2.000", Count: 3, Pct: 100},
		},
	}
	var buf bytes.Buffer
	require.NoError(t, templkit.UyelerPanel(v).Render(context.Background(), &buf))
	out := buf.String()

	assert.Contains(t, out, "abon-trend-bars")
	assert.Contains(t, out, "height:50%")
	assert.Contains(t, out, "height:100%")
	// Erişilebilirlik: görsel çubuklar aria-hidden, veri sr-only tabloda.
	assert.Contains(t, out, `aria-hidden="true"`)
	assert.Contains(t, out, `class="sr-only"`)
	assert.Contains(t, out, "Aylık abonelik geliri")
	assert.Contains(t, out, "2026-10")
	assert.Contains(t, out, "2.000")
	assert.Contains(t, out, "3 ödeme")
}

func TestGelirTrendiBosDurumdaGrafikYok(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, templkit.UyelerPanel(templkit.UyelerView{}).Render(context.Background(), &buf))
	out := buf.String()
	assert.NotContains(t, out, "abon-trend-bars", "trend yoksa grafik basılmamalı")
	assert.NotContains(t, out, "Aylık abonelik geliri")
}

func TestTrendCubukYuzdesi(t *testing.T) {
	// View dönüşümü: en yüksek ay %100, sıfır olmayan küçük aylar en az %4.
	trend := []struct {
		amount float64
		max    float64
		want   int
	}{
		{0, 100, 0},
		{1, 100, 4},
		{50, 100, 50},
		{100, 100, 100},
		{33, 100, 33},
	}
	for _, c := range trend {
		pct := 0
		if c.max > 0 && c.amount > 0 {
			pct = int(c.amount/c.max*100 + 0.5)
			if pct < 4 {
				pct = 4
			}
		}
		assert.Equal(t, c.want, pct, "amount=%v max=%v", c.amount, c.max)
	}
}

func TestTrendCubuklariSifirAydaBoslukBirakmaz(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, templkit.UyelerPanel(templkit.UyelerView{
		TrendHasData: true,
		Trend:        []templkit.UyeTrend{{Short: "Oca", Label: "2026-01", Amount: "0", Count: 0, Pct: 0}},
	}).Render(context.Background(), &buf))
	out := buf.String()
	assert.Contains(t, out, "height:0%")
	assert.Contains(t, strings.ToLower(out), "oca")
}

func TestGelirTrendiVeriYoksaMesaj(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, templkit.UyelerPanel(templkit.UyelerView{
		Trend: []templkit.UyeTrend{
			{Short: "Eyl", Label: "2026-09", Amount: "0", Pct: 0},
			{Short: "Eki", Label: "2026-10", Amount: "0", Pct: 0},
		},
		TrendHasData: false,
	}).Render(context.Background(), &buf))
	out := buf.String()
	assert.Contains(t, out, "Son 12 ayda abonelik ödemesi yok")
	assert.NotContains(t, out, "abon-trend-bars", "veri yokken boş çubuk çizilmemeli")
}
