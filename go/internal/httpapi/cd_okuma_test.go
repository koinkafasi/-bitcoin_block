package httpapi

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func kelimeUret(n int) string {
	return strings.TrimSpace(strings.Repeat("kelime ", n))
}

func TestCdOkumaDakikaYukariYuvarlar(t *testing.T) {
	// Sınırlar: 200 kelime tam 1 dk, 201 ikinci dakikaya girer.
	assert.Equal(t, 1, cdOkumaDakika("", kelimeUret(200)))
	assert.Equal(t, 2, cdOkumaDakika("", kelimeUret(201)))
	assert.Equal(t, 2, cdOkumaDakika("", kelimeUret(400)))
	assert.Equal(t, 3, cdOkumaDakika("", kelimeUret(401)))

	// Asıl hata: 394 kelimelik yazı "1 dk" görünüyordu.
	assert.Equal(t, 2, cdOkumaDakika("", kelimeUret(394)),
		"399'a kadar aşağı yuvarlanıp 1 dk gösteriliyordu")
}

func TestCdOkumaDakikaSpotuSayar(t *testing.T) {
	// Spot da okunan metnin parçası.
	assert.Equal(t, 2, cdOkumaDakika(kelimeUret(30), kelimeUret(180)))
}

func TestCdOkumaDakikaEnAzBir(t *testing.T) {
	assert.Equal(t, 1, cdOkumaDakika("", ""))
	assert.Equal(t, 1, cdOkumaDakika("", "tek kelime"))
}
