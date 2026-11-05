package httpapi

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/repo"
)

func TestContentRequiredWeight(t *testing.T) {
	assert.Equal(t, 0, contentRequiredWeight(false, 0), "açık içerik")
	assert.Equal(t, 1, contentRequiredWeight(true, 0), "özel içerik varsayılan 1")
	assert.Equal(t, 3, contentRequiredWeight(true, 3))
	assert.Equal(t, 4, contentRequiredWeight(false, 4), "ağırlık verilmişse uygulanır")
}

func TestTierAllows(t *testing.T) {
	assert.True(t, tierAllows(0, 0))
	assert.False(t, tierAllows(0, 1))
	assert.True(t, tierAllows(1, 1))
	assert.False(t, tierAllows(2, 3))
	assert.True(t, tierAllows(5, 3))
	assert.True(t, tierAllows(999, 5), "editoryal roller sınırsız")
}

func TestNormalizeCycleVeTr(t *testing.T) {
	assert.Equal(t, "monthly", normalizeCycle(""))
	assert.Equal(t, "monthly", normalizeCycle("aylik"))
	assert.Equal(t, "annual", normalizeCycle("annual"))
	assert.Equal(t, "annual", normalizeCycle("yillik"))
	assert.Equal(t, "yıllık", cycleTR("annual"))
	assert.Equal(t, "aylık", cycleTR("monthly"))
}

func TestParsePrice(t *testing.T) {
	assert.Equal(t, 150.0, parsePrice("150"))
	assert.Equal(t, 150.5, parsePrice("150,50"))
	assert.Equal(t, 1200.0, parsePrice(" 1200.00 "))
	assert.Equal(t, 0.0, parsePrice("abc"))
	assert.Equal(t, 0.0, parsePrice("-5"))
}

func TestTrimMoney(t *testing.T) {
	assert.Equal(t, "150", trimMoney(150))
	assert.Equal(t, "150,50", trimMoney(150.5))
	assert.Equal(t, "0", trimMoney(0))
	assert.Equal(t, "1234,99", trimMoney(1234.99))
}

func TestSplitName(t *testing.T) {
	n, s := splitName("Ayşe Yılmaz")
	assert.Equal(t, "Ayşe", n)
	assert.Equal(t, "Yılmaz", s)
	n, s = splitName("  ")
	assert.Equal(t, "Abone", n)
	assert.Equal(t, "Uye", s)
	n, s = splitName("Madonna")
	assert.Equal(t, "Madonna", n)
	assert.Equal(t, "Uye", s)
	n, s = splitName("Ayşe Nur Yılmaz")
	assert.Equal(t, "Ayşe", n)
	assert.Equal(t, "Nur Yılmaz", s)
}

func TestAbonelikSablonuKademeler(t *testing.T) {
	tmpl := parseTestTemplates(t)
	d := headerData()
	d.MetaTitle = "Abonelik"
	d.Abonelik = &abonelikData{
		Cycle: "monthly",
		Tiers: []repo.Tier{
			{ID: 1, Slug: "filiz", Name: "Filiz", ColorHex: "#22c55e", Weight: 1,
				MonthlyPrice: 150, AnnualPrice: 1500, Description: "Giriş seviyesi",
				Perks: []string{"Ayrıcalıklı özetler"}},
			{ID: 3, Slug: "gumus", Name: "Gümüş", ColorHex: "#94a3b8", Weight: 3,
				MonthlyPrice: 750, AnnualPrice: 7500, Description: "Tam erişim",
				Perks: []string{"Tüm analizler"}},
		},
		CurrentID:   1,
		CurrentName: "Filiz",
		IyzicoOn:    true,
		IsLoggedIn:  true,
	}
	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "abonelik", d))
	out := buf.String()
	assert.Contains(t, out, "Abonelik Paketleri")
	assert.Contains(t, out, "Filiz")
	assert.Contains(t, out, "Gümüş")
	assert.Contains(t, out, "150 ₺")
	assert.Contains(t, out, "1500 ₺")
	assert.Contains(t, out, "Mevcut planınız")
	assert.Contains(t, out, "2 ay hediye")
	assert.Contains(t, out, `action="/abonelik/odeme"`)
	assert.Contains(t, out, `name="tckn"`)
	assert.Contains(t, out, "data-tier=\"gumus\"")
}

func TestAbonelikMisafirGirisCagrisi(t *testing.T) {
	tmpl := parseTestTemplates(t)
	d := headerData()
	d.Abonelik = &abonelikData{
		Tiers: []repo.Tier{{ID: 1, Slug: "filiz", Name: "Filiz", MonthlyPrice: 150, AnnualPrice: 1500}},
	}
	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "abonelik", d))
	out := buf.String()
	assert.Contains(t, out, "/giris?redirect=abonelik")
	assert.NotContains(t, out, `action="/abonelik/odeme"`)
}

func TestHesapAbonelikKademeKarti(t *testing.T) {
	d := hesapTestData("abonelik")
	d.Hesap.TierName = "Gümüş"
	d.Hesap.TierColor = "#94a3b8"
	d.Hesap.TierUntil = "31.12.2026"
	d.Hesap.TierCycle = "annual"
	out := renderHesap(t, "hesap-abonelik", d)
	assert.Contains(t, out, "Gümüş")
	assert.Contains(t, out, "31.12.2026")
	assert.Contains(t, out, "Yıllık")
	assert.Contains(t, out, "/abonelik")
}
