package templkit

import (
	"strconv"
	"strings"
	"testing"
)

// Ölçekleme YALNIZ /kg/<hash> biçimine uygulanmalı.
func TestKgOlcekliSadeceKaynakGorseli(t *testing.T) {
	const h = "80fe8a843afddb33661ff0f019b8ac11"
	if !cdKgOlcekli("/kg/" + h) {
		t.Error("geçerli /kg/ adresi ölçeklenebilir sayılmalı")
	}
	for _, url := range []string{
		"",
		"/media/thumb/12",
		"/media/variant/12/480",
		"https://cdn.decrypt.co/resim.png", // dış adres
		"/kg/",
		"/kg/" + h + "/480", // zaten genişlik taşıyor
		"/kg/kisa",
		"/kg/80FE8A843AFDDB33661FF0F019B8AC11", // büyük harf
		"/kg/../../etc/passwd",
		"/static/logo.svg",
	} {
		if cdKgOlcekli(url) {
			t.Errorf("ölçeklenebilir sayılmamalı: %q", url)
		}
	}
}

// Ölçeklenemeyen adres OLDUĞU GİBİ kalmalı; srcset üretilmemeli.
func TestKgOlceklenemeyenAdresDegismez(t *testing.T) {
	for _, url := range []string{"", "/media/thumb/12", "https://dis/resim.png"} {
		if got := cdKgSrc(url, YuvaHeroLead); got != url {
			t.Errorf("adres değişmiş: %q → %q", url, got)
		}
		if got := cdKgSrcset(url, YuvaHeroLead); got != "" {
			t.Errorf("srcset üretilmemeli: %q → %q", url, got)
		}
	}
}

func TestYuvaBasamaklariEtkinYogunlukYetiyor(t *testing.T) {
	// Yuva genişlikleri cd.css'ten hesaplandı (bkz. cd_gorsel_yuva.go).
	yuvaGenislik := map[CDYuva]int{
		YuvaKart:        472,
		YuvaHeroLead:    1094,
		YuvaHeroYan:     290,
		YuvaBlokLead:    334,
		YuvaEnCokOkunan: 720,
		YuvaMakaleKapak: 680,
	}
	// Çarpanlı sizes'ın hedeflediği etkin yoğunluk.
	const carpan = 1.3
	// Ucun en büyük basamağı 1600 (httpapi.kaynakGorselGenislikGecerli).
	const tavan = 1600
	for yuva, genislik := range yuvaGenislik {
		b := yuvaBilgi[yuva].Basamaklar
		if len(b) == 0 {
			t.Fatalf("yuva %d basamaksız", yuva)
		}
		enBuyuk := b[len(b)-1]
		gerek := int(float64(genislik) * carpan)
		if enBuyuk >= tavan {
			continue // tavana dayanmış; daha genisi uc zaten vermiyor
		}
		if enBuyuk < gerek {
			t.Errorf("yuva %d (%dpx): en büyük basamak %d, etkin yoğunluk ~%d gerekiyor",
				yuva, genislik, enBuyuk, gerek)
		}
	}
}

func TestYuvaSizesCarpanli(t *testing.T) {
	for yuva, bilgi := range yuvaBilgi {
		if yuva == YuvaAvatar {
			continue // 48px sabit; çarpan anlamsız
		}
		if !strings.Contains(bilgi.Sizes, "* 0.65") {
			t.Errorf("yuva %d: 2dppx çarpanı (0.65) yok: %q", yuva, bilgi.Sizes)
		}
		if !strings.Contains(bilgi.Sizes, "* 0.433") {
			t.Errorf("yuva %d: 3dppx çarpanı (0.433) yok: %q", yuva, bilgi.Sizes)
		}
	}
}

func TestYuvaBasamaklariUcunListesinde(t *testing.T) {
	izinli := map[int]bool{320: true, 480: true, 768: true, 960: true, 1200: true, 1600: true}
	for yuva, bilgi := range yuvaBilgi {
		var once int
		for _, g := range bilgi.Basamaklar {
			if !izinli[g] {
				t.Errorf("yuva %d: %d genişliği /kg/ ucunun listesinde yok", yuva, g)
			}
			if g <= once {
				t.Errorf("yuva %d: basamaklar artan olmalı (%d sonra %d)", yuva, once, g)
			}
			once = g
		}
		if !izinli[bilgi.Varsayilan] {
			t.Errorf("yuva %d: varsayılan %d listede yok", yuva, bilgi.Varsayilan)
		}
	}
}

// srcset biçimi: "<adres>/<g> <g>w", virgülle ayrık.
func TestKgSrcsetBicimi(t *testing.T) {
	const h = "80fe8a843afddb33661ff0f019b8ac11"
	url := "/kg/" + h
	ss := cdKgSrcset(url, YuvaHeroYan)

	parcalar := strings.Split(ss, ", ")
	basamaklar := yuvaBilgi[YuvaHeroYan].Basamaklar
	if len(parcalar) != len(basamaklar) {
		t.Fatalf("srcset %d parça, %d bekleniyordu: %q", len(parcalar), len(basamaklar), ss)
	}
	for i, g := range basamaklar {
		s := strconv.Itoa(g)
		beklenen := url + "/" + s + " " + s + "w"
		if parcalar[i] != beklenen {
			t.Errorf("srcset parçası %d: %q, beklenen %q", i, parcalar[i], beklenen)
		}
	}
}

func TestKgSrcVarsayilanBasamaklarIcinde(t *testing.T) {
	const url = "/kg/80fe8a843afddb33661ff0f019b8ac11"
	for yuva, bilgi := range yuvaBilgi {
		src := cdKgSrc(url, yuva)
		bulundu := false
		for _, g := range bilgi.Basamaklar {
			if src == url+"/"+strconv.Itoa(g) {
				bulundu = true
				break
			}
		}
		if !bulundu {
			t.Errorf("yuva %d: src %q basamaklar arasında değil", yuva, src)
		}
	}
}

// Her yuvanın sizes'ı olmalı ve mobil dalı taşımalı (avatar hariç: sabit).
func TestYuvaSizesDoldurulmus(t *testing.T) {
	for yuva, bilgi := range yuvaBilgi {
		if bilgi.Sizes == "" {
			t.Errorf("yuva %d: sizes boş", yuva)
			continue
		}
		if yuva == YuvaAvatar {
			continue // 40px sabit; vw dalı anlamsız
		}
		if !strings.Contains(bilgi.Sizes, "max-width: 767px") {
			t.Errorf("yuva %d: mobil dalı yok (%q)", yuva, bilgi.Sizes)
		}
	}
}

func TestBlokKartiHeroLeadtenDar(t *testing.T) {
	// .cd-blok__ic 724'lük hücrede (724 - 2*20 dolgu - 16 gap) / 2 = 334
	// .cd-hero hero lead = 1464-80-290 = 1094 (hero en geniş yuva)
	blok := yuvaBilgi[YuvaBlokLead].Sizes
	hero := yuvaBilgi[YuvaHeroLead].Sizes
	if !strings.Contains(blok, "334px") {
		t.Errorf("blok kartı sizes'ında 334px yok: %q", blok)
	}
	if !strings.Contains(hero, "1094px") {
		t.Errorf("hero lead sizes'ında 1094px yok: %q", hero)
	}
	// Yan ray 290 px — ölçülen CoinDesk sağ rayıyla aynı.
	if yan := yuvaBilgi[YuvaHeroYan].Sizes; !strings.Contains(yan, "290px") {
		t.Errorf("hero yan sizes'ında 290px yok: %q", yan)
	}
}
