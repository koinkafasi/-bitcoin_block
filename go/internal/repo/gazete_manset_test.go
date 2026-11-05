package repo

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Faz 197 — gazete manşetleri: kaynak ayrıştırma.
//
// ÖLÇÜM (2 Kasım 2026, üretim sunucusu):
//
//	GET https://www.gazeteoku.com/  →  200 · 213 KB · 0.44 sn
//	benzersiz kapak                 →  30 gazete
//	900x1310 boy isteği             →  200 · 600 KB · image/jpeg
//	300x437  boy isteği             →  200 ·  87 KB · image/jpeg

// Gerçek sayfadan alınmış iki kapak (yapı birebir).
const gazeteOrnekHTML = `<div><a href="/hurriyet"><img
  src="https://i.gazeteoku.com/3/255/365/storage/files/images/2026/11/02/hurriyet-2026-11-02-0bsu.jpg"
  alt="Hürriyet gazetesi"></a></div>
<div><a href="/sozcu"><img
  src="https://i.gazeteoku.com/3/255/365/storage/files/images/2026/11/02/sozcu-2026-11-02-aaaa.jpg"
  alt="Sözcü gazetesi"></a></div>
<div><a href="/hurriyet"><img
  src="https://i.gazeteoku.com/3/600/873/storage/files/images/2026/11/02/hurriyet-2026-11-02-zzzz.jpg"
  alt="Hürriyet (öne çıkan)"></a></div>`

func TestGazeteKapakCoz(t *testing.T) {
	kapak := gazeteKapakCoz([]byte(gazeteOrnekHTML))
	require.Len(t, kapak, 2, "aynı gazete iki kez basılsa da tek kayıt olmalı")

	// Sıra listesi Hürriyet'i Sözcü'den önce tutar.
	assert.Equal(t, "hurriyet", kapak[0].Slug)
	assert.Equal(t, "Hürriyet", kapak[0].Ad)
	assert.Equal(t, "2026-11-02", kapak[0].Tarih, "tarih görselin yolundan okunur")
	assert.Equal(t, "sozcu", kapak[1].Slug)
	assert.Equal(t, "Sözcü", kapak[1].Ad)
}

func TestGazeteKapakBoyutlari(t *testing.T) {
	kapak := gazeteKapakCoz([]byte(gazeteOrnekHTML))
	require.NotEmpty(t, kapak)

	// CDN bir boyut vekilidir: yol parçası değişir, dosya adı korunur.
	assert.Equal(t,
		"https://i.gazeteoku.com/3/900/1310/storage/files/images/2026/11/02/hurriyet-2026-11-02-0bsu.jpg",
		kapak[0].Gorsel)
	assert.Equal(t,
		"https://i.gazeteoku.com/3/300/437/storage/files/images/2026/11/02/hurriyet-2026-11-02-0bsu.jpg",
		kapak[0].Kucuk)
	assert.NotEqual(t, kapak[0].Gorsel, kapak[0].Kucuk, "küçük boy ayrı istek olmalı")
}

func TestGazeteKapakBosVeBozuk(t *testing.T) {
	assert.Empty(t, gazeteKapakCoz(nil))
	assert.Empty(t, gazeteKapakCoz([]byte("<html><body>kapak yok</body></html>")))
	// Farklı alan adından gelen görseller kabul edilmez.
	assert.Empty(t, gazeteKapakCoz([]byte(`<img src="https://baska.com/hurriyet-2026-11-02-aaaa.jpg">`)))
}

func TestGazeteAdiOf(t *testing.T) {
	assert.Equal(t, "Hürriyet", gazeteAdiOf("hurriyet"))
	assert.Equal(t, "Hürriyet Daily News", gazeteAdiOf("hurriyet-daily-news"))
	assert.Equal(t, "Millî Gazete", gazeteAdiOf("milli-gazete"))
	// Bilinmeyen slug başlık biçimine çevrilir (modül gazete kaybetmez).
	assert.Equal(t, "Yeni Gazete", gazeteAdiOf("yeni-gazete"))
}

func TestGazeteSirasi(t *testing.T) {
	// Bilinen gazeteler sıra listesine göre; bilinmeyen sona.
	assert.Less(t, gazeteSiraOf("hurriyet"), gazeteSiraOf("sozcu"))
	assert.Equal(t, len(gazeteSiraListesi), gazeteSiraOf("bilinmeyen-gazete"))

	// Sıra listesi ile ad sözlüğü tutarlı olmalı: listedeki her slug'ın
	// görünen adı var, sözlükteki her slug listede.
	for _, s := range gazeteSiraListesi {
		assert.NotEmpty(t, gazeteAdlari[s], "sıra listesindeki %q için görünen ad yok", s)
	}
	for s := range gazeteAdlari {
		assert.NotEqual(t, len(gazeteSiraListesi), gazeteSiraOf(s),
			"ad sözlüğündeki %q sıra listesinde yok", s)
	}
}

// TestGazeteGercekSayfa gerçek (ağdan alınmış) sayfayla doğrulama.
//
// Elle: curl -s https://www.gazeteoku.com/ -o /tmp/gazeteoku.html
func TestGazeteGercekSayfa(t *testing.T) {
	ham, err := os.ReadFile("/tmp/gazeteoku.html")
	if err != nil || len(ham) < 50000 {
		t.Skip("gerçek sayfa yok (/tmp/gazeteoku.html)")
	}
	kapak := gazeteKapakCoz(ham)
	require.Greater(t, len(kapak), 15, "gerçek sayfada 15+ gazete beklenirdi")

	for _, k := range kapak {
		assert.NotEmpty(t, k.Ad)
		assert.Regexp(t, `^\d{4}-\d{2}-\d{2}$`, k.Tarih)
		assert.Contains(t, k.Gorsel, "/900/1310/")
		assert.Contains(t, k.Kucuk, "/300/437/")
		assert.True(t, strings.HasSuffix(k.Gorsel, ".jpg"))
	}
	// Sıra monoton olmalı (bilinmeyenler sona).
	for i := 1; i < len(kapak); i++ {
		assert.LessOrEqual(t, kapak[i-1].Sira, kapak[i].Sira)
	}
	t.Logf("çözülen gazete: %d · ilk: %s (%s)", len(kapak), kapak[0].Ad, kapak[0].Tarih)
}
