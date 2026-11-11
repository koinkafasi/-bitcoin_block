package templkit

// Ana sayfa giriş hero'su (cd_giris) testleri.
//
// NEDEN AYRI DOSYA: hero'nun üç kırılgan noktası var ve üçü de sessizce
// bozulabilir:
//   1) Veri yokken modül BASILMAMALI (uydurma sayı yasak).
//   2) Yay geometrisi: dasharray ve nokta konumu aynı yarıçaptan (84)
//      türemeli; biri değişip diğeri unutulursa gösterge yanlış yeri
//      gösterir ama hata vermez.
//   3) Hero yüksekliği 532 px'e çakılı — kullanıcı isteği ("bu
//      yükseklikte"). CSS'te min-height düşerse görünüm kayar.
//
// Ve bir bütçe kuralı: hero stili cd.css'e DEĞİL ayrı dosyaya yazılmalı
// (cd.css kendi testinde 200 KB bütçeye tabi).

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/webui"
)

// cdGirisRender hero'yu HTML'e çevirir (testler için).
func cdGirisRender(t *testing.T, g CDGiris) string {
	t.Helper()
	var b bytes.Buffer
	require.NoError(t, cdGirisHero(g).Render(context.Background(), &b))
	return b.String()
}

func TestCdGirisHeroVeriYoksaBasilmaz(t *testing.T) {
	// Hazir=false: veri gelmedi. Modül HİÇ basılmamalı — boş iskelet
	// göstermek "veri var ama boş" izlenimi verirdi.
	html := cdGirisRender(t, CDGiris{})
	assert.NotContains(t, html, "cd-giris")
	assert.Empty(t, strings.TrimSpace(html))
}

func TestCdGirisHeroGerçekDegerleriBasar(t *testing.T) {
	html := cdGirisRender(t, CDGiris{
		Hazir:       true,
		BaslikOnce:  "Türkiye'nin En Güvenilir ",
		BaslikVurgu: "Dijital Varlık Bilgi",
		BaslikSonra: " Platformu",
		BTCFiyat:    "$73,422.00",
		BTCDegisim:  "-2.34%",
		BTCUp:       false,
		BTCYol:      "M8.00 60.00L312.00 72.00",
		BTCAlan:     "M8.00 60.00L312.00 72.00 L312.00 112.00 L8.00 112.00 Z",
		Hacim:       "$79.5 Mr",
		Dominans:    "59.7%",
		FNGVar:      true,
		FNGDeger:    29,
		FNGEtiket:   "Korku",
		FNGDun:      32,
		FNGHafta:    34,
		RSIVar:      true,
		RSI:         29,
		RSIEtiket:   "Aşırı Satım",
	})
	assert.Contains(t, html, "$73,422.00", "BTC fiyatı basılmalı")
	assert.Contains(t, html, "-2.34%")
	assert.Contains(t, html, "$79.5 Mr")
	assert.Contains(t, html, "59.7%")
	assert.Contains(t, html, ">29<", "FNG değeri")
	assert.Contains(t, html, "Korku")
	assert.Contains(t, html, "Aşırı Satım")
	assert.Contains(t, html, "Dün: <b>32</b>")
	assert.Contains(t, html, "Geçen Hafta: <b>34</b>")
	// Sparkline yolu SVG'ye girmeli (sunucu tarafı; JS yok).
	assert.Contains(t, html, "M8.00 60.00L312.00 72.00")
}

func TestCdGirisHeroEksikParcaBasilmaz(t *testing.T) {
	// RSI/FNG yok: o kartlar hiç çizilmemeli (0 yazmak uydurma olurdu).
	html := cdGirisRender(t, CDGiris{
		Hazir:    true,
		BTCFiyat: "$73,422.00",
		BTCYol:   "M8.00 60.00L312.00 72.00",
	})
	assert.Contains(t, html, "$73,422.00")
	assert.NotContains(t, html, "Korku &amp; Açgözlülük", "FNG kartı basılmamalı")
	assert.NotContains(t, html, "RSI", "RSI kartı basılmamalı")
	// Yüzen kartlar da veri yoksa basılmaz.
	assert.NotContains(t, html, "cd-giris__yuzen")
}

func TestCdGirisYayGeometrisi(t *testing.T) {
	// Yarıçap 84 → tam yay uzunluğu π·84 ≈ 263.9.
	// %0: 0, %50: yarım, %100: tam yay.
	assert.Equal(t, "0.0 264", cdGirisYay(0))
	// π·84·0.5 = 131.946… → tek ondalık 131.9
	assert.Equal(t, "131.9 264", cdGirisYay(50))
	assert.Equal(t, "263.9 264", cdGirisYay(100))

	// Sınır dışı değerler kırpılır (bozuk veri göstergesi kaydırmasın).
	assert.Equal(t, "0.0 264", cdGirisYay(-20))
	assert.Equal(t, "263.9 264", cdGirisYay(180))

	// Nokta konumu: %0 → sol uç (16,100), %100 → sağ uç (184,100),
	// %50 → tepe (100,16).
	assert.Equal(t, "16.0", cdGirisNoktaX(0))
	assert.Equal(t, "100.0", cdGirisNoktaY(0))
	assert.Equal(t, "184.0", cdGirisNoktaX(100))
	assert.Equal(t, "100.0", cdGirisNoktaY(100))
	assert.Equal(t, "100.0", cdGirisNoktaX(50))
	assert.Equal(t, "16.0", cdGirisNoktaY(50))
}

func TestCdGirisCSSAyriDosyada(t *testing.T) {
	// cd.css bütçesi 200 KB; hero stili ORAYA yazılırsa bütçe aşılır.
	// Bu test, birinin "kolayına" gelip stili cd.css'e taşımasını engeller.
	cdCSS, err := webui.FS.ReadFile("static/cd.css")
	require.NoError(t, err)
	assert.NotContains(t, string(cdCSS), ".cd-giris",
		"hero stili cd.css'e taşınmış — ayrı dosyada kalmalı (bütçe)")

	girisCSS, err := webui.FS.ReadFile("static/cd-giris.css")
	require.NoError(t, err, "static/cd-giris.css yok")
	css := string(girisCSS)
	// Ölçülen yerleşim değerleri — sessizce değişmesin.
	assert.Contains(t, css, "min-height: 532px", "hero yüksekliği 532 px'e çakılı olmalı")
	assert.Contains(t, css, "1.05fr 1fr 300px", "ızgara ghost ölçümüyle aynı olmalı")
	assert.Contains(t, css, "gap: 30px")
	// Koyu tema ve mobil kırılımları var olmalı.
	assert.Contains(t, css, "html.dark .cd-giris__panel")
	assert.Contains(t, css, "@media (max-width: 1240px)")
	assert.Contains(t, css, "@media (max-width: 900px)")
}

func TestCdGirisHeroGorunumDerlenir(t *testing.T) {
	// templ bileşeni boş görünümle de hata vermemeli.
	require.NotPanics(t, func() {
		_ = cdGirisRender(t, CDGiris{Hazir: true})
	})
	_ = templ.Component(nil)
}
