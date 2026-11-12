package repo

import (
	"net/url"
	"strings"
)

var reklamYolIsaretleri = []string{
	"/press-release", "/press-releases", "/pressrelease",
	"/sponsored", "/sponsored-content", "/sponsored-post",
	"/partner-content", "/partnercontent",
	"/advertorial", "/paid-post", "/paid-content", "/promoted",
	// Türkçe kaynaklar için
	"/basin-bulteni", "/basin-aciklamasi", "/sponsorlu", "/tanitim", "/reklam",
}

var reklamSorguParametreleri = []string{
	"ref", "aff", "affiliate", "referral", "invite", "promo",
}

var sponsorIfsaIfadeleri = []string{
	"this article is sponsored",
	"this is a sponsored",
	"this post is sponsored",
	"sponsored article",
	"sponsored post",
	"sponsored content",
	"paid post",
	"advertorial",
	"promoted post",
	// Türkçe
	"bu icerik sponsorludur",
	"sponsorlu icerik",
	"tanitim yazisi",
	"reklam icerigi",
	"isbirligi ile hazirlanmistir",
}

var tesvikIfadeleri = []string{
	"guaranteed return", "guaranteed profit", "risk-free profit",
	"promo code", "use code", "bonus code", "deposit bonus",
	"claim your free", "claim free", "free giveaway", "giveaway now",
	"limited time offer", "act now", "don't miss out on this",
	"100x gem", "1000x gem", "next 100x", "guaranteed 10x",
	// Türkçe
	"garantili kazanc", "garantili getiri", "kacirmayin",
	"hemen kazan", "bedava kazanin", "promosyon kodu",
}

func ReklamIcerikMi(link, baslik, govde string) (bool, string) {
	if yol, ok := reklamYolundaMi(link); ok {
		return true, "adres yolu: " + yol
	}
	if p, ok := reklamSorgusundaMi(link); ok {
		return true, "yönlendirme parametresi: " + p
	}
	if i := sponsorIfsasiVarMi(baslik); i != "" {
		return true, "başlıkta sponsor ifşası: " + i
	}
	if i := sponsorIfsasiVarMi(ilkBolum(govde, 600)); i != "" {
		return true, "gövde başında sponsor ifşası: " + i
	}
	if t := tesvikVarMi(baslik); t != "" {
		return true, "başlıkta teşvik kalıbı: " + t
	}
	return false, ""
}

func reklamYolundaMi(link string) (string, bool) {
	u, err := url.Parse(strings.TrimSpace(link))
	if err != nil {
		return "", false
	}
	yol := strings.ToLower(u.EscapedPath())
	for _, m := range reklamYolIsaretleri {
		if strings.Contains(yol, m) {
			return m, true
		}
	}
	return "", false
}

func reklamSorgusundaMi(link string) (string, bool) {
	u, err := url.Parse(strings.TrimSpace(link))
	if err != nil {
		return "", false
	}
	q := u.Query()
	for _, p := range reklamSorguParametreleri {
		if v := q.Get(p); v != "" {
			return p, true
		}
	}
	return "", false
}

func sponsorIfsasiVarMi(metin string) string {
	m := kriptoNormalize(metin)
	for _, i := range sponsorIfsaIfadeleri {
		if strings.Contains(m, kriptoNormalize(i)) {
			return i
		}
	}
	return ""
}

func tesvikVarMi(metin string) string {
	m := kriptoNormalize(metin)
	for _, t := range tesvikIfadeleri {
		if strings.Contains(m, kriptoNormalize(t)) {
			return t
		}
	}
	return ""
}

// ilkBolum metnin ilk n baytını döndürür (çok baytlı karakteri bölmeden).
func ilkBolum(s string, n int) string {
	if len(s) <= n {
		return s
	}
	k := s[:n]
	// UTF-8 karakterini ortadan bölmemek için son tam karaktere kırp.
	for len(k) > 0 && !utf8BaslangicMi(k[len(k)-1]) && k[len(k)-1] >= 0x80 {
		k = k[:len(k)-1]
	}
	return k
}

// utf8BaslangicMi bayt bir UTF-8 karakterinin ilk baytı mı.
func utf8BaslangicMi(b byte) bool { return b&0xC0 != 0x80 }
