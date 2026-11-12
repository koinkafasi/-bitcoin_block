package httpapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
