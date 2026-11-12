package templkit

import (
	"strconv"
	"strings"
)

type CDYuva uint8

const (
	// YuvaKart genel ızgara kartı (.cd-grid--3 / --4). En dar yuva.
	YuvaKart CDYuva = iota
	// YuvaHeroLead ana sayfa hero'sunun büyük kartı (1094px).
	YuvaHeroLead
	// YuvaHeroYan hero'nun yan sütunu (290px) — dar ray kartı.
	YuvaHeroYan
	YuvaBlokLead
	// YuvaEnCokOkunan "En Çok Okunan" iki sütunu (720px).
	YuvaEnCokOkunan
	// YuvaMakaleKapak makale kapağı (680px, LCP görseli).
	YuvaMakaleKapak
	// YuvaAvatar görüş yazarı avatarı (48px, sabit).
	YuvaAvatar
	// YuvaKatLead kategori sayfasının öne çıkan kartı (689px).
	// 12 kolonlu ızgaranın 6'sı: 6×100.656 + 5×16 = 689 (spec §10.3).
	YuvaKatLead
	// YuvaKatKart kategori ızgarasının normal kartı (327px).
	// 12 kolonun 3'ü: 3×100.656 + 2×16 = 327 (spec §10.3).
	YuvaKatKart
	// YuvaAkis hero orta kolonundaki 8'li haber akışının küçük görseli
	// (152×86; 16/9). Ölçüldü 6 Kasım 2026.
	YuvaAkis
	YuvaKatLeadYatay
	// YuvaKatAlt alt sıradaki 4'lü ızgaranın kartı (239×135).
	YuvaKatAlt
	// YuvaKatSatir "More From" listesinin satır görseli (327×185).
	YuvaKatSatir
	// YuvaTagBanner etiket sayfası banner görseli (692×389).
	YuvaTagBanner
)

const (
	ifadeMobil = "100vw - 32px"
	ifadeTam   = "100vw - 48px"
	// Izgara 768–1024 arası HÂLÂ 3 sütun: (100vw - 2*24 yan - 2*24 gap) / 3
	ifadeKartTablet = "(100vw - 96px) / 3"
	// Kategori ızgarası 1279 altında 4 kolona düşer ve kart 2 kolon kaplar,
	// yani 2-up: (100vw - 2*24 yan - 16 gap) / 2
	ifadeKatTablet = "(100vw - 64px) / 2"
)

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
		Sizes:      cdSizes(ifadeMobil, ifadeTam, 334),
		Basamaklar: []int{320, 480, 768},
		Varsayilan: 480,
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
	YuvaAvatar: {
		Sizes:      "48px",
		Basamaklar: []int{320},
		Varsayilan: 320,
	},
	YuvaKatLead: {
		Sizes:      cdSizes(ifadeMobil, ifadeTam, 689),
		Basamaklar: []int{480, 768, 960},
		Varsayilan: 768,
	},
	YuvaKatKart: {
		Sizes:      cdSizes(ifadeMobil, ifadeKatTablet, 327),
		Basamaklar: []int{320, 480, 768},
		Varsayilan: 480,
	},
	// 152 px küçük görsel; etkin ihtiyaç 152 × 1,3 ≈ 198 → 320 yeter.
	YuvaAkis: {
		Sizes:      cdSizes(ifadeMobil, ifadeTam, 152),
		Basamaklar: []int{320, 480},
		Varsayilan: 320,
	},
	YuvaKatLeadYatay: {
		Sizes:      cdSizes(ifadeMobil, ifadeTam, 686),
		Basamaklar: []int{480, 768, 960},
		Varsayilan: 768,
	},
	YuvaKatAlt: {
		Sizes:      cdSizes(ifadeMobil, ifadeKatTablet, 239),
		Basamaklar: []int{320, 480},
		Varsayilan: 320,
	},
	YuvaKatSatir: {
		Sizes:      cdSizes(ifadeMobil, ifadeKatTablet, 327),
		Basamaklar: []int{320, 480},
		Varsayilan: 480,
	},
	YuvaTagBanner: {
		Sizes:      cdSizes(ifadeMobil, ifadeTam, 692),
		Basamaklar: []int{480, 768, 960},
		Varsayilan: 768,
	},
}

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

func cdLCPOnYukle(s CDStory) *CDPreload {
	if s.Image == "" {
		return nil
	}
	if !cdKgOlcekli(s.Image) {
		return &CDPreload{Src: s.Image}
	}
	return &CDPreload{
		Src:    cdKgSrc(s.Image, YuvaHeroLead),
		Srcset: cdKgSrcset(s.Image, YuvaHeroLead),
		Sizes:  cdKgSizes(YuvaHeroLead),
	}
}
