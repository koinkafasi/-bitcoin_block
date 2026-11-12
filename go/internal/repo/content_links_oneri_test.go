package repo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// İç link önerisinin kelime ayıklaması: sorgunun ne kadar daraltılacağını
// bu belirliyor. Fazla kelime sonuçsuz bırakır, bağlaçlar ise her başlığı
// eşleştirip sıralamayı anlamsızlaştırır.

func TestIcLinkKelimelerBaglaclariAtar(t *testing.T) {
	k := icLinkKelimeler("Bitcoin ve Ethereum için bir karar")
	assert.Equal(t, []string{"bitcoin", "ethereum", "karar"}, k)
}

func TestIcLinkKelimelerKisaParcalariAtar(t *testing.T) {
	// İki harfli parçalar ILIKE'ta her yere uyuyor.
	k := icLinkKelimeler("AB io SEC kararı")
	assert.Equal(t, []string{"sec", "kararı"}, k)
}

func TestIcLinkKelimelerNoktalamaTemizler(t *testing.T) {
	// Kesme işareti ve noktalama kelimeyi bölmemeli.
	k := icLinkKelimeler("Bitcoin'in fiyatı, SEC kararı!")
	assert.Equal(t, []string{"bitcoin", "fiyatı", "sec", "kararı"}, k)
}

func TestIcLinkKelimelerTekrarVeSinir(t *testing.T) {
	// Aynı kelime iki kez sayılmaz.
	assert.Equal(t, []string{"bitcoin", "fiyat"}, icLinkKelimeler("Bitcoin bitcoin fiyat BITCOIN"))

	// En çok altı kelime: fazlası sorguyu sonuçsuz bırakıyor.
	uzun := icLinkKelimeler("bir iki üç dört beş altı yedi sekiz dokuz onbir oniki")
	assert.LessOrEqual(t, len(uzun), 6)
}

func TestIcLinkKelimelerBosGirdi(t *testing.T) {
	assert.Empty(t, icLinkKelimeler(""))
	// Yalnız bağlaçtan oluşan seçim: çağıran eski aramaya düşer.
	assert.Empty(t, icLinkKelimeler("ve ile için bu şu"))
}
