package i18n

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Faz 195 — boş çeviri değeri etiketi siliyordu.
//
// ÖLÇÜM (üretim): `translations` tablosunda `footer.akis` gibi 9 anahtarın
// satırı VAR ama `tr` değeri BOŞ. `T()` yalnız anahtar YOKSA geri düşüyordu;
// boş değeri döndürüp etiketi tamamen siliyordu. Çekmecede yalnız emoji
// kalan bağlantılar görüldü:
//
//	[2] "📰"   [1] "✉"   [2] "💳"   [2] "☀"  …
//
// Boş değer artık "çeviri yok" sayılır.

func yeniBundle(msgs map[string]string) *Bundle {
	b := &Bundle{msgs: msgs}
	return b
}

func TestTBosDegerYokSayilir(t *testing.T) {
	b := yeniBundle(map[string]string{
		"en\x00footer.akis": "Live Feed",
		"tr\x00footer.akis": "", // satır var, değer boş (üretimdeki durum)
	})
	// tr isteği: tr boş → tr'ye geri düşer, o da boş → anahtar.
	assert.Equal(t, "footer.akis", b.T("tr", "footer.akis"),
		"boş tr değeri etiketi silmemeli; anahtar görünmeli")
	// en isteği:en dolu → onu kullanır.
	assert.Equal(t, "Live Feed", b.T("en", "footer.akis"))
}

func TestTDoluDegerKorunur(t *testing.T) {
	b := yeniBundle(map[string]string{
		"tr\x00footer.akis": "Akış",
		"en\x00footer.akis": "Live Feed",
	})
	assert.Equal(t, "Akış", b.T("tr", "footer.akis"))
	assert.Equal(t, "Live Feed", b.T("en", "footer.akis"))
}

func TestTEksikAnahtarGeriDuser(t *testing.T) {
	b := yeniBundle(map[string]string{
		"tr\x00footer.akis": "Akış",
	})
	// de'de yok → tr'ye düşer (mevcut davranış korunur).
	assert.Equal(t, "Akış", b.T("de", "footer.akis"))
	// hiç yok → anahtar.
	assert.Equal(t, "yok.boyle", b.T("tr", "yok.boyle"))
}

func TestTBosEnDegerTrdenGelir(t *testing.T) {
	b := yeniBundle(map[string]string{
		"tr\x00footer.borsa": "Borsa",
		"en\x00footer.borsa": "", // en satırı var ama boş
	})
	assert.Equal(t, "Borsa", b.T("en", "footer.borsa"),
		"boş en değeri tr karşılığını gizlememeli")
}
