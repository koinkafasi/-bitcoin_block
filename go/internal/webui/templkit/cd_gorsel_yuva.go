package templkit

import (
	"strconv"
	"strings"
)

// Kaynak görsellerinin yuvaya göre ölçeklenmesi — srcset/sizes üretimi.
//
// NEDEN (canlı ölçüm, 5 Kasım 2026): /kg/{hash} ucu görseli önbellekliyor
// ama boyutlandırmıyordu. Ana sayfadaki beş görsel 1,39 MB ölçüldü (en
// büyüğü 708 KB PNG) ve hepsi en fazla 808px'lik yuvalara giriyordu.
// httpapi'ye /kg/{hash}/{genislik} eklendi; burası şablonların doğru
// genişliği istemesini sağlıyor.
//
// YUVA GENİŞLİKLERİ UYDURULMADI — cd.css'ten hesaplandı (6 Kasım 2026,
// üçüncü ölçüm turu: kapsayıcı 1512, yan boşluk 24 → içerik 1464):
//
//	.cd-hero   minmax(0,1fr) 290px, sütun aralığı 80
//	           → lead 1464-80-290 = 1094 · yan 290
//	.cd-block  2fr 1fr,    sütun aralığı 80 → lead (1464-80)*2/3 = 922
//	.cd-most   1fr 1fr,    gap 24           →      (1464-24)/2   = 720
//	.cd-article max-width: 680px            →                     680
//	.cd-grid--3 gap 24                      →      (1464-48)/3   = 472
//	bölüm sayfası (.cd-wrap--bolum, 1384)   →      (1384-48)/3   = 445
//	.cd-opinion__avatar                     →                      48 (sabit)
//
// Kırılma noktaları: 1284 (hero tek kolon — CoinDesk 1284 kullanıyor,
// 1280 değil), 1024 (blok tek kolon), 768 (ızgara tek kolon, yan boşluk 16).

// CDYuva kartın hangi yuvada durduğunu söyler.
//
// Neden bir yuva tipi, `lead bool` değil: ölçüm gösterdi ki .cd-block
// içindeki "lead olmayan" kart hero lead'inden DAHA GENİŞ. Boolean ile
// doğru sizes üretilemiyor.
type CDYuva uint8

const (
	// YuvaKart genel ızgara kartı (.cd-grid--3 / --4). En dar yuva.
	YuvaKart CDYuva = iota
	// YuvaHeroLead ana sayfa hero'sunun büyük kartı (1094px).
	YuvaHeroLead
	// YuvaHeroYan hero'nun yan sütunu (290px) — dar ray kartı.
	YuvaHeroYan
	// YuvaBlokLead bölüm bloğunun büyük kartı (922px) — en geniş kart.
	YuvaBlokLead
	// YuvaEnCokOkunan "En Çok Okunan" iki sütunu (720px).
	YuvaEnCokOkunan
	// YuvaMakaleKapak makale kapağı (680px, LCP görseli).
	YuvaMakaleKapak
	// YuvaAvatar görüş yazarı avatarı (48px, sabit).
	YuvaAvatar
)

// Mobil/tablet genişlik İFADELERİ (calc() sarmalayıcısı olmadan).
//
//	≤767px  : tek sütun, .cd-wrap yan boşluğu 16px → 100vw - 32px
//	≤1024px : hero ve blok tek sütun, yan boşluk 24px → 100vw - 48px
const (
	ifadeMobil = "100vw - 32px"
	ifadeTam   = "100vw - 48px"
	// Izgara 768–1024 arası HÂLÂ 3 sütun: (100vw - 2*24 yan - 2*24 gap) / 3
	ifadeKartTablet = "(100vw - 96px) / 3"
)

// yuvaBilgi her yuvanın sizes'ı, srcset basamakları ve srcset'siz
// tarayıcılar için varsayılan genişliği.
//
// Basamak seçimi kuralı: en geniş basamak, yuvanın **etkin yoğunluk**
// ihtiyacını karşılamalı. Çarpanlı sizes kullanıldığı için bu ihtiyaç
// yuva × 1,3 (aşağıdaki cdSizes açıklamasına bakın), yuva × 2 değil.
var yuvaBilgi = map[CDYuva]struct {
	Sizes      string
	Basamaklar []int
	Varsayilan int
}{
	YuvaKart: {
		Sizes:      cdSizes(ifadeMobil, ifadeKartTablet, 472),
		Basamaklar: []int{320, 480, 768, 960},
		Varsayilan: 480,
	},
	YuvaHeroLead: {
		Sizes:      cdSizes(ifadeMobil, ifadeTam, 1094),
		Basamaklar: []int{480, 768, 960, 1200, 1600},
		Varsayilan: 960,
	},
	YuvaHeroYan: {
		Sizes:      cdSizes(ifadeMobil, ifadeTam, 290),
		Basamaklar: []int{320, 480, 768},
		Varsayilan: 480,
	},
	YuvaBlokLead: {
		Sizes:      cdSizes(ifadeMobil, ifadeTam, 922),
		Basamaklar: []int{480, 768, 960, 1200, 1600},
		Varsayilan: 960,
	},
	YuvaEnCokOkunan: {
		Sizes:      cdSizes(ifadeMobil, ifadeTam, 720),
		Basamaklar: []int{320, 480, 768, 960, 1200},
		Varsayilan: 768,
	},
	// .cd-article max-width 680 — tablette de 680'de sabitlenir.
	YuvaMakaleKapak: {
		Sizes:      cdSizes(ifadeMobil, "680px", 680),
		Basamaklar: []int{480, 768, 960, 1200},
		Varsayilan: 768,
	},
	// 48px sabit. Kapalı listenin en küçüğü 320: DPR 3'te bile fazlasıyla
	// yeter. Bilinçli küçük bir fazlalık — iki görsel sistemi (mediaSrcset
	// ve /kg/) aynı genişlik kümesinde kalsın diye liste bölünmedi.
	YuvaAvatar: {
		Sizes:      "48px",
		Basamaklar: []int{320},
		Varsayilan: 320,
	},
}

// cdSizes CoinDesk'in "çarpanlı" sizes tekniğini üretir (spec §9).
//
// NEDEN (bu, spec'in bizim için en değerli bulgusu): tarayıcı `w`
// tanımlayıcısında "yoğunluk = aday_piksel / sizes_değeri" hesaplar ve
// DPR'ye eşit olanı hedefler. Yani sizes'ı küçük beyan edersek, DPR 2'de
// seçilen aday = sizes × 2 olur. CoinDesk 2dppx dalında sizes'ı 0.65 ile
// çarpıyor → aday = yuva × 0.65 × 2 = yuva × 1,3; 3dppx'te 0.433 ile
// çarpıyor → yuva × 0.433 × 3 = yuva × 1,3. İkisi de **1,3×**'te buluşuyor.
//
// Etki: retina ekranda 2×/3× yerine 1,3× indiriliyor. Gözle fark yok,
// bayt ~%35 azalıyor ve srcset basamağı bir-iki kademe küçüğe iniyor.
//
// Biçim (spec §9'daki ölçülen dizenin aynısı):
//
//	(mobil, 3dppx) → calc(<mobil> * 0.583)   // 3 × 0,583 ≈ 1,75×
//	(mobil, 2dppx) → calc(<mobil> * 0.875)   // 2 × 0,875 = 1,75×
//	(mobil)        → calc(<mobil>)
//	… tablet katmanı …
//	(3dppx)        → calc(<px> * 0.433)
//	(2dppx)        → calc(<px> * 0.65)
//	               → <px>
//
// Mobilde 1,75× (masaüstünde 1,3×) — CoinDesk de böyle yapıyor: küçük
// ekranda piksel yoğunluğu daha çok önemli, ağ maliyeti daha az.
func cdSizes(mobil, tablet string, masaustu int) string {
	c := make([]string, 0, 9)
	if mobil != "" {
		c = append(c,
			"(max-width: 767px) and (min-resolution: 3dppx) calc(("+mobil+") * 0.583)",
			"(max-width: 767px) and (min-resolution: 2dppx) calc(("+mobil+") * 0.875)",
			"(max-width: 767px) calc("+mobil+")")
	}
	if tablet != "" {
		c = append(c,
			"(max-width: 1024px) and (min-resolution: 3dppx) calc(("+tablet+") * 0.583)",
			"(max-width: 1024px) and (min-resolution: 2dppx) calc(("+tablet+") * 0.875)",
			"(max-width: 1024px) calc("+tablet+")")
	}
	px := strconv.Itoa(masaustu) + "px"
	c = append(c,
		"(min-resolution: 3dppx) calc("+px+" * 0.433)",
		"(min-resolution: 2dppx) calc("+px+" * 0.65)",
		px)
	return strings.Join(c, ", ")
}

// cdKgOlcekli adres ölçeklenebilir bir kaynak görseli mi.
//
// YALNIZ /kg/<hash> biçimine dokunulur. Dış adresler, /media/... yolları ve
// boş değer olduğu gibi bırakılır: ölçekleme ucu yalnız kaynak görselleri
// tanır, başkasına genişlik eklemek 404 üretir.
func cdKgOlcekli(url string) bool {
	if !strings.HasPrefix(url, "/kg/") {
		return false
	}
	h := url[len("/kg/"):]
	// Zaten genişlik taşıyorsa (ya da başka bir şey varsa) karışmayalım.
	if len(h) != 32 || strings.Contains(h, "/") {
		return false
	}
	for _, r := range h {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			return false
		}
	}
	return true
}

// cdKgSrc yuvanın varsayılan genişliğindeki adres (srcset'i olmayan
// tarayıcılar ve srcset'in başarısız olduğu durumlar için).
func cdKgSrc(url string, yuva CDYuva) string {
	if !cdKgOlcekli(url) {
		return url
	}
	return url + "/" + strconv.Itoa(yuvaBilgi[yuva].Varsayilan)
}

// cdKgSrcset yuvanın basamaklarını "<adres>/<g> <g>w" biçiminde üretir.
func cdKgSrcset(url string, yuva CDYuva) string {
	if !cdKgOlcekli(url) {
		return ""
	}
	b := yuvaBilgi[yuva].Basamaklar
	var sb strings.Builder
	for i, g := range b {
		if i > 0 {
			sb.WriteString(", ")
		}
		s := strconv.Itoa(g)
		sb.WriteString(url)
		sb.WriteString("/")
		sb.WriteString(s)
		sb.WriteString(" ")
		sb.WriteString(s)
		sb.WriteString("w")
	}
	return sb.String()
}

// cdKgSizes yuvanın sizes değeri.
func cdKgSizes(yuva CDYuva) string { return yuvaBilgi[yuva].Sizes }
