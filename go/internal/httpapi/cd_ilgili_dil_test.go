package httpapi

import "testing"

// Dil karşılaştırması "İlgili Haberler" bölümünün tek süzgeci. Yanlış
// eşleşme Türkçe makalenin altına İngilizce haber koyar (yerelde ölçüldü:
// 6 bağlantının 6'sı /en/ idi); fazla katı olması da bölümü boşaltır.
func TestCDAyniDil(t *testing.T) {
	testler := []struct {
		a, b  string
		istek bool
		neden string
	}{
		{"tr", "tr", true, "aynı kod"},
		{"en", "en", true, "aynı kod"},
		{"tr", "en", false, "farklı dil — asıl yakalanması gereken durum"},
		{"en", "tr", false, "ters yön de farklı"},

		// Boş locale eski kayıtlarda görülüyor ve o içerikler Türkçe.
		// Boşu "farklı" saymak onları ilgili listesinden tümüyle düşürürdü.
		{"", "tr", true, "boş locale tr sayılır"},
		{"tr", "", true, "ters yönde de tr sayılır"},
		{"", "", true, "ikisi de boş"},
		{"", "en", false, "boş tr demek, en ile eşleşmez"},

		// Bölgesel varyantlar aynı dil: hreflang tr-TR yazabiliyor.
		{"tr-TR", "tr", true, "bölge soneki yoksayılır"},
		{"en-US", "en-GB", true, "iki İngilizce varyantı aynı dil"},
		{"tr_TR", "tr", true, "alt çizgili biçim de ayrıştırılır"},

		// Büyük/küçük harf ve boşluk veri girişinden gelebilir.
		{"TR", "tr", true, "büyük harf"},
		{" tr ", "tr", true, "kenar boşluğu"},
		{"TR-tr", "Tr", true, "karışık harf + bölge"},
	}
	for _, tt := range testler {
		if got := cdAyniDil(tt.a, tt.b); got != tt.istek {
			t.Errorf("cdAyniDil(%q, %q) = %v, beklenen %v (%s)",
				tt.a, tt.b, got, tt.istek, tt.neden)
		}
	}
}

// Limit, alaka motorunun döndürdüğü aday sayısının altında kalmamalı:
// kalırsa kademeler boşuna sorgulanır.
func TestCDIlgiliHaberLimitiMakul(t *testing.T) {
	if cdIlgiliHaberLimiti < 3 {
		t.Errorf("limit %d: iç bağlantı kazancı kaybolur", cdIlgiliHaberLimiti)
	}
	if cdIlgiliHaberLimiti > 12 {
		t.Errorf("limit %d: makale altı bağlantı yığınına döner", cdIlgiliHaberLimiti)
	}
}
