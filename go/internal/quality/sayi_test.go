package quality

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeNumber(t *testing.T) {
	cases := map[string]string{
		"480":       "480",
		"1.500":     "1500",
		"1.250.000": "1250000",
		"12,5":      "12.5",
		"12.50":     "12.5",
		"1,500":     "1500",
		"3.2":       "3.2",
		"007":       "7",
		"2027":      "2027",
		"%42,3":     "42.3",
	}
	for in, want := range cases {
		assert.Equalf(t, want, normalizeNumber(in), "girdi %q", in)
	}
}

func TestVerifyNumbersVerbatim(t *testing.T) {
	src := "Yatırım planının bütçesi 480 milyon lira olarak açıklandı. Çalışmalar 2027 yılında başlayacak."
	r := VerifyNumbers("Bütçe 480 milyon lira. Plan 2027 yılında başlayacak.", src)
	assert.True(t, r.OK(), "kaynaktaki sayılar doğrulanmalı: %+v", r.Unverified)
	assert.Equal(t, 2, r.Checked)
	assert.Equal(t, 100, r.Score)
}

func TestVerifyNumbersDetectsFabricatedNumber(t *testing.T) {
	src := "Bütçe 480 milyon lira olarak açıklandı."
	r := VerifyNumbers("Bütçe 840 milyon lira olarak açıklandı.", src)
	assert.False(t, r.OK(), "uydurma sayı yakalanmalı")
	assert.Len(t, r.Unverified, 1)
	assert.Equal(t, "840", r.Unverified[0].Value)
	assert.Equal(t, 0, r.Score)
}

func TestVerifyNumbersFormatDifferences(t *testing.T) {
	// Kaynakta 1.500, metinde 1500 → aynı sayı; uyarı olmamalı.
	src := "Katılımcı sayısı 1.500 olarak açıklandı."
	r := VerifyNumbers("Katılımcı sayısı 1500 kişiydi.", src)
	assert.True(t, r.OK(), "%+v", r.Unverified)
}

func TestVerifyNumbersTurkishDecimal(t *testing.T) {
	src := "Oran yüzde 42,3 olarak ölçüldü."
	r := VerifyNumbers("Oran %42,3 arttı.", src)
	assert.True(t, r.OK(), "%+v", r.Unverified)
}

func TestVerifyNumbersSmallNumbersAlsoChecked(t *testing.T) {
	// Küçük sayılar da denetlenir: "18 ay" yerine "24 ay" yazmak haber
	// hatasıdır. Kaynakta yazıyla geçen sayı ("üç") eşleşmez; bu bilinçli.
	r := VerifyNumbers("Süre 24 ay olarak açıklandı.", "Süre 18 ay olarak açıklandı.")
	assert.Equal(t, 1, r.Checked)
	assert.False(t, r.OK())
	assert.Equal(t, "24", r.Unverified[0].Value)
}

func TestVerifyNumbersYearIsChecked(t *testing.T) {
	r := VerifyNumbers("Plan 2027 yılında başlayacak.", "Plan 2028 yılında başlayacak.")
	assert.False(t, r.OK())
	assert.Equal(t, "2027", r.Unverified[0].Value)
}

func TestVerifyNumbersEmptySource(t *testing.T) {
	r := VerifyNumbers("Bütçe 480 milyon lira.", "")
	assert.False(t, r.OK())
	assert.Equal(t, 0, r.Score)
}

func TestVerifyNumbersDeduplicates(t *testing.T) {
	src := "Bütçe 480 milyon lira."
	r := VerifyNumbers("480 milyon. 480 milyon tekrar. 480 milyon yine.", src)
	assert.Equal(t, 1, r.Checked, "aynı sayı bir kez sayılmalı")
}

func TestVerifyNumbersNoNumbers(t *testing.T) {
	r := VerifyNumbers("Hiç sayı yok burada.", "kaynak")
	assert.Equal(t, 0, r.Checked)
	assert.Equal(t, 100, r.Score)
	assert.True(t, r.OK())
}

func TestVerifyNumbersMixedPartiallyVerified(t *testing.T) {
	src := "Bütçe 480 milyon lira, süre 18 ay."
	r := VerifyNumbers("Bütçe 480 milyon lira, süre 24 ay.", src)
	assert.Equal(t, 2, r.Checked)
	assert.Equal(t, 1, r.Verified)
	assert.Equal(t, 50, r.Score)
	assert.Equal(t, "24", r.Unverified[0].Value)
}
