package httpapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatTL(t *testing.T) {
	assert.Equal(t, "150,00", formatTL(150))
	assert.Equal(t, "1.234,56", formatTL(1234.56))
	assert.Equal(t, "30.000,00", formatTL(30000))
	assert.Equal(t, "0,50", formatTL(0.5))
}

func TestHesapAbonelikIptalAkisiSablonu(t *testing.T) {
	d := hesapTestData("abonelik")
	d.Hesap.TierName = "Altın"
	d.Hesap.TierColor = "#eab308"
	d.Hesap.TierUntil = "31.01.2027"
	d.Hesap.CancelAt = "15.10.2026"
	out := renderHesap(t, "hesap-abonelik", d)
	assert.Contains(t, out, "İptali geri al")
	assert.Contains(t, out, "/hesabim/abonelik/iptal")
	assert.Contains(t, out, "İptal talebi:")
	assert.Contains(t, out, "31.01.2027 tarihine kadar erişiminiz sürecek")
}

func TestHesapAbonelikIptalButonu(t *testing.T) {
	d := hesapTestData("abonelik")
	d.Hesap.TierName = "Bronz"
	d.Hesap.TierColor = "#b45309"
	out := renderHesap(t, "hesap-abonelik", d)
	assert.Contains(t, out, "Aboneliği iptal et")
	assert.NotContains(t, out, `name="geri_al"`)
}
