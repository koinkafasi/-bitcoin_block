package httpapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCdSiteSonekiAt(t *testing.T) {
	// Asıl iş: sondaki site adı eki atılır.
	assert.Equal(t, "Bitcoin 100 bin doları aştı",
		cdSiteSonekiAt("Bitcoin 100 bin doları aştı | BitCoŞaR"))
	// Büyük/küçük harf ve şapkasız yazım da yakalanır.
	assert.Equal(t, "Ethereum güncellemesi",
		cdSiteSonekiAt("Ethereum güncellemesi | bitcosar"))

	// KORUNMASI GEREKENLER — site adı başlığın gerçek parçasıysa
	// dokunulmaz. Sitede 39 bülten içeriği böyle.
	assert.Equal(t, "Haftanın özeti (BİTCOŞAR Kripto Bülteni)",
		cdSiteSonekiAt("Haftanın özeti (BİTCOŞAR Kripto Bülteni)"))
	// Boru var ama kuyruk site adı değil: başlık olduğu gibi kalır.
	assert.Equal(t, "SEC | yeni düzenleme",
		cdSiteSonekiAt("SEC | yeni düzenleme"))
	// Boru yok.
	assert.Equal(t, "Düz başlık", cdSiteSonekiAt("Düz başlık"))
	// Baştaki boru silinmez (i <= 0 koruması).
	assert.Equal(t, "| BitCoŞaR", cdSiteSonekiAt("| BitCoŞaR"))
	// Boşluklar temizlenir.
	assert.Equal(t, "Başlık", cdSiteSonekiAt("  Başlık   |   BitCoŞaR  "))
}
