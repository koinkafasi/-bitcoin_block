package templkit

import "strconv"

// Funnel şablonunun küçük yardımcıları. Şablon içinde mantık bırakmamak
// için (bkz. cd_shell.templ başındaki not) Go tarafında duruyorlar.

// fnlSayi boyutları özniteliğe çevirir.
func fnlSayi(n int) string {
	if n <= 0 {
		return ""
	}
	return strconv.Itoa(n)
}

// fnlYukleme hero görselini hemen, alt bölümdekileri tembel yükler.
//
// NEDEN: hero ekran görüntüsü LCP adayı; lazy verilirse tarayıcı onu
// görünür alan hesaplaması bitene kadar bekleterek LCP'yi geciktiriyor.
func fnlYukleme(oncelik bool) string {
	if oncelik {
		return "eager"
	}
	return "lazy"
}
