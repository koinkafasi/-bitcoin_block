package httpapi

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Yazı içi varlık rozeti: gövdeyi bozmadan, sınırlı sayıda ve doğru
// yerlerde basılmalı. Hatalar sessiz olur (geçersiz HTML, iç içe link),
// bu yüzden kurallar teste bağlı.

func ornekVarliklar() []cdVarlikEslesme {
	return []cdVarlikEslesme{
		{Ad: "Bitcoin", Sym: "BTC", Slug: "bitcoin", Fiyat: "$100", Artis: true, Yuzde: "+1,00%"},
		{Ad: "Ethereum", Sym: "ETH", Slug: "ethereum", Fiyat: "$50", Artis: false, Yuzde: "−2,00%"},
	}
}

func TestRozetMetindeBasilir(t *testing.T) {
	govde := "<p>Bugün Bitcoin yükseldi.</p>"
	out := cdVarlikRozetle(govde, ornekVarliklar())

	assert.Contains(t, out, `href="/price/bitcoin"`)
	assert.Contains(t, out, "BTC")
	// Rozet kelimenin ARDINA gelir, kelimeyi silmez.
	assert.Contains(t, out, "Bitcoin<a class=\"cd-varlik")
	assert.Contains(t, out, "yükseldi.")
}

func TestRozetHerVarlikIcinBirKez(t *testing.T) {
	govde := "<p>Bitcoin ve yine Bitcoin, sonra Bitcoin.</p>"
	out := cdVarlikRozetle(govde, ornekVarliklar())
	assert.Equal(t, 1, strings.Count(out, `href="/price/bitcoin"`),
		"aynı varlık tek kez rozetlenmeli")
}

func TestRozetBaglantiIcindeBasilmaz(t *testing.T) {
	// İç içe <a> geçersiz HTML'dir ve tarayıcı bağlantıyı bozar.
	govde := `<p><a href="/x">Bitcoin hakkında</a> yazı.</p>`
	out := cdVarlikRozetle(govde, ornekVarliklar())
	assert.NotContains(t, out, "cd-varlik")
}

func TestRozetBaslikVeTabloDaBasilmaz(t *testing.T) {
	baslik := "<h2>Bitcoin nedir?</h2><p>Metin.</p>"
	assert.NotContains(t, cdVarlikRozetle(baslik, ornekVarliklar()), "cd-varlik",
		"başlık tipografisi bozulmamalı")

	// Tablo hücresi dar: rozet alt alta dağılıyordu (ölçüldü).
	tablo := "<table><tbody><tr><td>Bitcoin</td></tr></tbody></table>"
	assert.NotContains(t, cdVarlikRozetle(tablo, ornekVarliklar()), "cd-varlik")
}

func TestRozetUstSinir(t *testing.T) {
	varliklar := []cdVarlikEslesme{
		{Ad: "Bitcoin", Sym: "BTC", Slug: "bitcoin", Fiyat: "$1"},
		{Ad: "Ethereum", Sym: "ETH", Slug: "ethereum", Fiyat: "$2"},
		{Ad: "Solana", Sym: "SOL", Slug: "solana", Fiyat: "$3"},
		{Ad: "Cardano", Sym: "ADA", Slug: "cardano", Fiyat: "$4"},
		{Ad: "Polkadot", Sym: "DOT", Slug: "polkadot", Fiyat: "$5"},
		{Ad: "Chainlink", Sym: "LINK", Slug: "chainlink", Fiyat: "$6"},
	}
	govde := "<p>Bitcoin Ethereum Solana Cardano Polkadot Chainlink</p>"
	out := cdVarlikRozetle(govde, varliklar)
	assert.Equal(t, rozetUstSinir, strings.Count(out, "cd-varlik__sym"),
		"rozet sayısı üst sınırı aşmamalı")
}

func TestRozetKelimeSinirinaSaygili(t *testing.T) {
	// "Bitcoincash" içinde "Bitcoin" geçiyor ama ayrı bir varlık.
	govde := "<p>Bitcoincash farklı bir ağdır.</p>"
	assert.NotContains(t, cdVarlikRozetle(govde, ornekVarliklar()), "cd-varlik")

	// Türkçe ekler eşleşmeyi bozmamalı.
	ekli := "<p>Bitcoin'in fiyatı arttı.</p>"
	assert.Contains(t, cdVarlikRozetle(ekli, ornekVarliklar()), "cd-varlik")
}

func TestRozetBosGirdi(t *testing.T) {
	assert.Equal(t, "", cdVarlikRozetle("", ornekVarliklar()))
	assert.Equal(t, "<p>Metin</p>", cdVarlikRozetle("<p>Metin</p>", nil))
}

func TestVarlikSlug(t *testing.T) {
	assert.Equal(t, "bitcoin", varlikSlug("Bitcoin"))
	assert.Equal(t, "binance-coin", varlikSlug("Binance Coin"))
	assert.Equal(t, "bitcoin-cash", varlikSlug(" Bitcoin Cash "))
}

// Türkçe büyük harfler bayt uzunluğunu değiştirebiliyor; rozet kelimenin
// ortasına girmemeli (ölçülen hata: "Ethereu<rozet>m").
func TestRozetTurkceKarakterKaydirmaz(t *testing.T) {
	govde := "<p>İş kanıtı güvenliği enerjiye dayanır; Ethereum 2022'de geçti.</p>"
	out := cdVarlikRozetle(govde, ornekVarliklar())

	assert.Contains(t, out, "Ethereum<a class=\"cd-varlik",
		"rozet kelimenin hemen ardına gelmeli")
	assert.NotContains(t, out, "Ethereu<a", "kelime ortadan bölünmemeli")
	assert.Contains(t, out, "İş kanıtı", "metnin başı bozulmamalı")
}
