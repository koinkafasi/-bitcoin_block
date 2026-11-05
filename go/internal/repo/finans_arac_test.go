package repo

import (
	"context"
	"strings"
	"testing"

	"theblock/site/internal/cryptoicon"
)

func TestKurTablosuCevir(t *testing.T) {
	kt := KurTablosu{
		Kurlar: map[string]float64{"TRY": 1, "USD": 50, "EUR": 55, "JPY": 0.0032},
		Adlar:  map[string]string{"USD": "Amerikan Doları"},
	}

	// USD → TRY
	if v, ok := kt.Cevir(100, "usd", "try"); !ok || v != 5000 {
		t.Errorf("100 USD = 5000 TRY beklenirdi, %v %v", v, ok)
	}
	// TRY → USD
	if v, ok := kt.Cevir(1000, "TRY", "USD"); !ok || v != 20 {
		t.Errorf("1000 TRY = 20 USD beklenirdi, %v %v", v, ok)
	}
	// Çapraz: EUR → USD
	if v, ok := kt.Cevir(1, "EUR", "USD"); !ok || v != 1.1 {
		t.Errorf("1 EUR = 1.1 USD beklenirdi, %v %v", v, ok)
	}
	// Bilinmeyen para birimi.
	if _, ok := kt.Cevir(1, "XXX", "TRY"); ok {
		t.Error("bilinmeyen para birimi çevrilmemeli")
	}
	// Sıfır/negatif kur güvenli davranış.
	kt.Kurlar["BAD"] = 0
	if _, ok := kt.Cevir(1, "BAD", "TRY"); ok {
		t.Error("sıfır kurla çeviri yapılmamalı")
	}
}

func TestKurTablosuKodlarVeAd(t *testing.T) {
	kt := KurTablosu{Kurlar: map[string]float64{
		"TRY": 1, "USD": 50, "EUR": 55, "GBP": 66, "JPY": 0.0032, "SAR": 13,
		"XXX": 2, "AAA": 3,
	}}
	kodlar := kt.Kodlar()
	// Referans sırası önce gelir.
	want := []string{"USD", "EUR", "TRY", "GBP", "JPY", "SAR"}
	for i, c := range want {
		if kodlar[i] != c {
			t.Fatalf("%d. kod %s beklenirdi, %s geldi (tümü: %v)", i, c, kodlar[i], kodlar)
		}
	}
	// Referans dışı kodlar sona, alfabetik.
	if len(kodlar) != 8 || kodlar[6] != "AAA" || kodlar[7] != "XXX" {
		t.Errorf("referans dışı kodlar sonda alfabetik olmalı: %v", kodlar)
	}
	// Ad: özel ad → genel ad → kod.
	if got := kt.Ad("USD"); got != "Amerikan Doları" {
		t.Errorf("USD adı yanlış: %q", got)
	}
	if got := kt.Ad("XXX"); got != "XXX" {
		t.Errorf("bilinmeyen kod kendini döndürmeli: %q", got)
	}
}

func TestKriptoOfIkonlariYerellestirir(t *testing.T) {
	// Eski önbellek: CoinGecko'nun dış `image` URL'si kalmış olabilir.
	// KriptoOf okurken her satırın ikonunu yerel kütüphaneden üretmeli.
	payload := `{"rows":[
		{"symbol":"BTC","name":"Bitcoin","image":"https://coin-images.coingecko.com/coins/images/1/large/bitcoin.png"},
		{"symbol":"IOTA","name":"IOTA","image":"https://coin-images.coingecko.com/coins/images/692/large/IOTA_.png"},
		{"symbol":"YOKBOYLE","name":"Bilinmeyen","image":"https://x/y.png"}
	]}`
	db := &fakeDB{rowFn: bytesRow(payload)}

	kd := KriptoOf(context.Background(), db)
	if len(kd.Rows) != 3 {
		t.Fatalf("3 satır beklenirdi, %d geldi", len(kd.Rows))
	}
	if got, want := kd.Rows[0].Image, cryptoicon.URL("BTC"); got != want {
		t.Errorf("BTC ikonu yerelleşmeli: %q != %q", got, want)
	}
	if got, want := kd.Rows[1].Image, cryptoicon.URL("IOTA"); got != want {
		t.Errorf("IOTA takma adı yerelleşmeli: %q != %q", got, want)
	}
	if kd.Rows[2].Image != "" {
		t.Errorf("ikonu olmayan sembol boş kalmalı, %q geldi", kd.Rows[2].Image)
	}
	for _, r := range kd.Rows {
		if strings.HasPrefix(r.Image, "http://") || strings.HasPrefix(r.Image, "https://") {
			t.Errorf("%s için dış URL kaldı: %q", r.Symbol, r.Image)
		}
	}
}
