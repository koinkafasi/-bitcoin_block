package repo

import (
	"net/url"
	"strings"
)

// Reklam / tanıtım içeriği kapısı — gelen beslemeler için.
//
// ÖLÇÜM (6 Kasım 2026, dört canlı kaynak, 97 öğe):
//
//	kaynak           öğe  PR-yol  sponsor-ifşa  teşvik  affiliate
//	cryptobriefing    30       0             0       0          0
//	cryptoslate       10       0             0       0          0
//	decrypt           37       0             0       0          0
//	theblock          20       0             0       0          0
//
// Yani **mevcut kaynaklarda reklam/spam yok**; dördü de ciddi yayın ve
// beslemeleri temiz. Bu kapı bugün hiçbir şeyi düşürmüyor — bilinçli.
// Amacı daha gürültülü bir kaynak eklendiğinde (örn. basın bülteni
// toplayıcısı, borsa blogu, Türkçe kripto siteleri) hazır olmak.
//
// NİYE DAR TUTULDU — ikinci ölçüm:
// Aynı 97 öğe, projenin mevcut `kriptoGucluTerimler` listesiyle de
// süzüldü. Toplayıcılara uygulanırsa **%27 meşru haber düşüyor**:
// Decrypt %43, The Block %50. Düşenler arasında "Zcash'ın NU7 ağ
// yükseltmesi mainnet'te", "Ark Invest Coinbase hissesi aldı",
// "Hyperliquid CFTC'ye FCM başvurusu" var — hepsi gerçek kripto haberi,
// yalnız terim listesinde geçmiyor. Bu yüzden:
//
//	Bu kapı bir ALAKA filtresi DEĞİLDİR ve öyle genişletilmemelidir.
//	Alaka kapısı yalnız resmî kurum kaynaklarına uygulanır (alaka.go).
//
// Yanlış pozitif gerçek haberi kaybettirir; o yüzden yalnız **tartışmasız**
// işaretler kullanılıyor ve gövde metninde serbest sözcük aranmıyor.
// Ölçümde "North Korea reportedly rewards top hackers with Pyongyang
// apartments after Binance heist" başlığı geniş bir desene takılmıştı —
// gerçek haber. O desen kaldırıldı, test olarak sabitlendi.

// reklamYolIsaretleri adres yolunda geçerse içerik tanıtımdır.
//
// Yol işareti en güvenilir sinyal: yayıncı kendi URL şemasında ayırıyor.
// Gövde metninde "press release" aramak yanlış pozitif üretir ("şirket
// basın bülteninde açıkladı" gerçek haberde sık geçer) — bu yüzden bu
// ifadeler YALNIZ yolda aranır.
var reklamYolIsaretleri = []string{
	"/press-release", "/press-releases", "/pressrelease",
	"/sponsored", "/sponsored-content", "/sponsored-post",
	"/partner-content", "/partnercontent",
	"/advertorial", "/paid-post", "/paid-content", "/promoted",
	// Türkçe kaynaklar için
	"/basin-bulteni", "/basin-aciklamasi", "/sponsorlu", "/tanitim", "/reklam",
}

// reklamSorguParametreleri adresin sorgusunda geçerse yönlendirme bağlantısı.
//
// `utm_campaign` BİLİNÇLİ OLARAK YOK: meşru haber bağlantılarında da
// bulunuyor, ayırt etmiyor.
var reklamSorguParametreleri = []string{
	"ref", "aff", "affiliate", "referral", "invite", "promo",
}

// sponsorIfsaIfadeleri açık sponsor beyanı.
//
// Hepsi tam ifade; tek sözcük yok. "in partnership with" ÇIKARILDI:
// gerçek haberde geçiyor ("Samsung, Solana ile ortaklıkla…").
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

// tesvikIfadeleri başlıkta geçerse promosyon/dolandırıcılık kalıbı.
//
// DİKKAT — tek başına "airdrop", "presale", "token sale" YOK: bunlar
// gerçek kripto haberinin konusu ("Zcash airdrop tarihi açıklandı").
// Yalnız okuyucuyu harekete çağıran kalıplar var.
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

// ReklamIcerikMi öğe tanıtım/reklam içeriği mi? İkinci dönüş değeri
// gerekçedir (loga ve yönetim panelinde gösterilmek için).
//
// Sıra önemli: en güvenilir sinyal (yol) önce denenir, en kırılgan
// (başlıktaki teşvik) en sonda.
func ReklamIcerikMi(link, baslik, govde string) (bool, string) {
	if yol, ok := reklamYolundaMi(link); ok {
		return true, "adres yolu: " + yol
	}
	if p, ok := reklamSorgusundaMi(link); ok {
		return true, "yönlendirme parametresi: " + p
	}
	// Sponsor ifşası başlıkta ya da gövdenin BAŞINDA aranır. Yayıncılar
	// ifşayı yazının başına koyar; sonda arama uzun metinde yanlış
	// pozitif üretir (yorum, alıntı, ilgili haber bloğu).
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
