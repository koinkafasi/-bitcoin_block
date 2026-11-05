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
// YUVA GENİŞLİKLERİ UYDURULMADI — cd.css'ten hesaplandı:
// --cd-max 1280, .cd-wrap padding 0 20px → içerik en çok 1240px.
//
//	.cd-hero   1.55fr 1fr, gap 32 → lead (1240-32)*1.55/2.55 = 734
//	                                 yan  (1240-32)*1.00/2.55 = 474
//	.cd-block  2fr 1fr,    gap 28 → lead (1240-28)*2/3        = 808
//	.cd-most   1fr 1fr,    gap 24 →      (1240-24)/2          = 608
//	.cd-article max-width: 720px  →                             720
//	.cd-grid--3                   →      (1240-48)/3          = 397
//	.cd-opinion__avatar           →                              40 (sabit)
//
// Bu değerler tarayıcıda da ölçüldü ve birebir çıktı (bkz.
// docs/DENETIM-2026-11-05.md): lead 734x413, orta 474x266, küçük 292x164.
//
// Kırılma noktaları: 1024px'te .cd-hero ve .cd-block TEK sütuna düşer,
// 720px'te .cd-grid--* de düşer ve .cd-wrap padding'i 14px olur. sizes
// dalları bunu izler.

// CDYuva kartın hangi yuvada durduğunu söyler.
//
// Neden bir yuva tipi, `lead bool` değil: ölçüm gösterdi ki .cd-block
// içindeki "lead olmayan" kart 808px — yani hero'nun lead'inden (734)
// DAHA GENİŞ. Boolean ile doğru sizes üretilemiyor.
type CDYuva uint8

const (
	// YuvaKart genel ızgara kartı (.cd-grid--3 / --4). En dar yuva.
	YuvaKart CDYuva = iota
	// YuvaHeroLead ana sayfa hero'sunun büyük kartı (734px).
	YuvaHeroLead
	// YuvaHeroYan hero'nun yan sütunu (474px).
	YuvaHeroYan
	// YuvaBlokLead bölüm bloğunun büyük kartı (808px) — en geniş kart.
	YuvaBlokLead
	// YuvaEnCokOkunan "En Çok Okunan" iki sütunu (608px).
	YuvaEnCokOkunan
	// YuvaMakaleKapak makale kapağı (720px, LCP görseli).
	YuvaMakaleKapak
	// YuvaAvatar görüş yazarı avatarı (40px, sabit).
	YuvaAvatar
)

// Mobil/tablet dalları. Ortak olduğu için tek yerde:
//
//	≤720px  : tek sütun, .cd-wrap padding 14px → 100vw - 28px
//	≤1024px : hero ve blok tek sütun, padding 20px → 100vw - 40px
const (
	dalMobil  = "(max-width: 720px) calc(100vw - 28px)"
	dalTablet = "(max-width: 1024px) calc(100vw - 40px)"
)

// yuvaBilgi her yuvanın sizes'ı, srcset basamakları ve srcset'siz
// tarayıcılar için varsayılan genişliği.
//
// Basamak seçimi kuralı: en geniş basamak, yuvanın DPR 2'deki ihtiyacını
// KARŞILAMALI (yuva x 2). Aksi hâlde retina ekranda bulanıklaşır — bu hata
// theblock'ta bir kez yapıldı, srcset 480'de kesilince DPR 2-3'te görüntü
// bozuldu. Basamaklar httpapi'deki kapalı listeden (320/480/768/960/1200/1600).
var yuvaBilgi = map[CDYuva]struct {
	Sizes      string
	Basamaklar []int
	Varsayilan int
}{
	// .cd-grid--3 → 397; 1024'te --4 iki sütuna düşer → ~calc(50vw - 27px).
	YuvaKart: {
		Sizes:      dalMobil + ", (max-width: 1024px) calc(50vw - 27px), 397px",
		Basamaklar: []int{320, 480, 768, 960},
		Varsayilan: 480,
	},
	YuvaHeroLead: {
		Sizes:      dalMobil + ", " + dalTablet + ", 734px",
		Basamaklar: []int{480, 768, 960, 1200, 1600},
		Varsayilan: 768,
	},
	YuvaHeroYan: {
		Sizes:      dalMobil + ", " + dalTablet + ", 474px",
		Basamaklar: []int{320, 480, 768, 960, 1200},
		Varsayilan: 480,
	},
	YuvaBlokLead: {
		Sizes:      dalMobil + ", " + dalTablet + ", 808px",
		Basamaklar: []int{480, 768, 960, 1200, 1600},
		Varsayilan: 768,
	},
	YuvaEnCokOkunan: {
		Sizes:      dalMobil + ", " + dalTablet + ", 608px",
		Basamaklar: []int{320, 480, 768, 960, 1200},
		Varsayilan: 480,
	},
	// .cd-article max-width 720 — 1024 dalı gereksiz, zaten 720'de sabitlenir.
	YuvaMakaleKapak: {
		Sizes:      dalMobil + ", 720px",
		Basamaklar: []int{480, 768, 960, 1200, 1600},
		Varsayilan: 768,
	},
	// 40px sabit. Kapalı listenin en küçüğü 320: DPR 3'te bile fazlasıyla
	// yeter. Bilinçli küçük bir fazlalık — iki görsel sistemi (mediaSrcset
	// ve /kg/) aynı genişlik kümesinde kalsın diye liste bölünmedi.
	YuvaAvatar: {
		Sizes:      "40px",
		Basamaklar: []int{320},
		Varsayilan: 320,
	},
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
