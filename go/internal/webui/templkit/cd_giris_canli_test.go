package templkit

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCdGirisCanliKancelari(t *testing.T) {
	html := cdGirisRender(t, CDGiris{
		Hazir: true, BTCFiyat: "$74,000.00", BTCDegisim: "-1.17%",
		BTCYol: "M8.00 60.00L312.00 72.00", BTCAlan: "M8.00 60.00L312.00 72.00 Z",
		Hacim: "$83.1 Mr", Dominans: "59.9%", Endeks: "29",
		ETHFiyat: "$2,066.00", ETHDegisim: "-2.40%",
		SOLFiyat: "$83.18", SOLDegisim: "-3.44%",
		FNGVar: true, FNGDeger: 29, FNGEtiket: "Korku", FNGDun: 32, FNGHafta: 34,
		RSIVar: true, RSI: 36, RSIEtiket: "Nötr",
		Guncelleme: "AZ ÖNCE",
	})

	// JS'in aradığı TÜM kancalar.
	for _, kanca := range []string{
		`data-cg="yas"`, `data-cg="btc-fiyat"`, `data-cg="btc-degisim"`,
		`data-cg="alan"`, `data-cg="cizgi"`,
		`data-cg="hacim"`, `data-cg="dominans"`, `data-cg="endeks"`,
		`data-cg="eth-fiyat"`, `data-cg="eth-degisim"`,
		`data-cg="sol-fiyat"`, `data-cg="sol-degisim"`,
		`data-cg="fng-deger"`, `data-cg="fng-etiket"`, `data-cg="fng-yay"`,
		`data-cg="fng-nokta"`, `data-cg="fng-dun"`, `data-cg="fng-hafta"`,
		`data-cg="rsi-deger"`, `data-cg="rsi-etiket"`, `data-cg="rsi-yay"`,
		`data-cg="rsi-nokta"`,
	} {
		assert.Contains(t, html, kanca, "JS kancası eksik: %s", kanca)
	}

	// Sunucu tarafı ilk değerler basılmış olmalı (JS kapalıysa da içerik var).
	assert.Contains(t, html, "$74,000.00")
	assert.Contains(t, html, "AZ ÖNCE")
}

func TestCdHeadEkJSYukler(t *testing.T) {
	var b bytes.Buffer
	require.NoError(t, CDHeadParca(CDHead{
		Title: "t", SiteName: "s", Locale: "tr",
		EkJS: []string{"/static/cd-giris.js"},
	}).Render(context.Background(), &b))
	html := b.String()
	assert.Contains(t, html, "/static/cd-giris.js", "EkJS betiği basılmalı")
	assert.Contains(t, html, `defer`, "betik defer ile yüklenmeli")
}
