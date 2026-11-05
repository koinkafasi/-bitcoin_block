package httpapi

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/webui/templkit"
)

func TestUyelerPanelKademeVeOdemeRender(t *testing.T) {
	v := templkit.UyelerView{
		Items: []templkit.UyeRow{
			{ID: 1, Name: "Ayşe", Email: "a@b.c", Role: "Üye", Membership: "premium",
				Premium: true, TierName: "Gümüş", TierColor: "#94a3b8", Until: "2026-12-31", Cycle: "annual"},
		},
		Tiers: []templkit.UyeTier{
			{ID: 1, Slug: "filiz", Name: "Filiz", ColorHex: "#22c55e", Weight: 1,
				Monthly: "150", Annual: "1500", Perks: "Özetler\nBülten", Active: true},
			{ID: 3, Slug: "gumus", Name: "Gümüş", ColorHex: "#94a3b8", Weight: 3,
				Monthly: "750", Annual: "7500", Active: true},
		},
		Payments: []templkit.UyePayment{
			{ID: 1, UserName: "Ayşe", TierName: "Gümüş", Provider: "iyzico",
				Status: "success", Amount: "750", Cycle: "aylık", Created: "01.10.2026 12:00"},
		},
		StatActive: 1, StatRev30: "750", StatRevY: "9.000", StatFailed: 0,
		TierCounts:  []templkit.UyeTierCount{{Name: "Gümüş", Count: 1}},
		IyzicoKey:   "sandbox-key",
		IyzicoMode:  "sandbox",
		IyzicoAktif: true,
		KayitAcik:   true,
	}
	var buf bytes.Buffer
	require.NoError(t, templkit.UyelerPanel(v).Render(context.Background(), &buf))
	out := buf.String()
	assert.Contains(t, out, "Abonelik Kademeleri")
	assert.Contains(t, out, "Filiz")
	assert.Contains(t, out, "Gümüş")
	assert.Contains(t, out, "iyzico")
	assert.Contains(t, out, "Aktif abone: 1")
	assert.Contains(t, out, "/admin/members/kademe")
	assert.Contains(t, out, "Son ödemeler")
}
