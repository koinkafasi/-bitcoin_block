package templkit

import (
	"strconv"
	"strings"
	"testing"
)

// Kaynak görsel ölçekleme — srcset/sizes sözleşmesi.
//
// NEDEN TEST: canlı ölçüm (5 Kasım 2026) ana sayfadaki beş /kg/ görselinin
// 1,39 MB olduğunu gösterdi (en büyüğü 708 KB PNG), en geniş yuva ise 808px.
// Bu testler iki hatayı önler:
//   1. srcset'i fazla kısa kesmek — theblock'ta yapıldı; 480'de kesilince
//      DPR 2-3 ekranlarda görüntü bozuldu (ölçüldü).
//   2. /kg/ olmayan adreslere genişlik eklemek — ölçekleme ucu yalnız
//      kaynak görselleri tanır, başkasına eklemek 404 üretir.

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

// En geniş basamak, yuvanın DPR 2 ihtiyacını KARŞILAMALI.
//
// Bu testin tamamı theblock'ta yapılan hatanın tekrarını önlemek için:
// srcset 480'de kesilmişti, DPR 2-3'te görüntü bozuldu.
func TestYuvaBasamaklariDPR2Yetiyor(t *testing.T) {
	// Yuva genişlikleri cd.css'ten hesaplandı (bkz. cd_gorsel_yuva.go).
	yuvaGenislik := map[CDYuva]int{
		YuvaKart:        397,
		YuvaHeroLead:    734,
		YuvaHeroYan:     474,
		YuvaBlokLead:    808,
		YuvaEnCokOkunan: 608,
		YuvaMakaleKapak: 720,
	}
	// Ucun en buyuk basamagi 1600 (httpapi.kaynakGorselGenislikGecerli);
	// bunun ustu istenemez. Ayrica basamaklar kaba: 808px yuva DPR 2'de
	// 1616 ister, en yakin basamak 1600 — %1 fark, gozle ayirt edilemez.
	// Kural: en buyuk basamak ya DPR 2 ihtiyacinin %95'ini karsilar ya da
	// zaten tavandir.
	const tavan = 1600
	for yuva, genislik := range yuvaGenislik {
		b := yuvaBilgi[yuva].Basamaklar
		if len(b) == 0 {
			t.Fatalf("yuva %d basamaksız", yuva)
		}
		enBuyuk := b[len(b)-1]
		gerek := genislik * 2
		if enBuyuk >= tavan {
			continue // tavana dayanmis; daha genisi uc zaten vermiyor
		}
		if enBuyuk*100 < gerek*95 {
			t.Errorf("yuva %d: en büyük basamak %d, DPR 2 için ~%d gerekiyor",
				yuva, enBuyuk, gerek)
		}
	}
}

// Basamaklar artan sırada ve httpapi'deki KAPALI LİSTEDEN olmalı.
//
// NEDEN: liste dışı bir genişlik 404 döner — sayfada boş kare çıkar.
// Uç tarafı: httpapi.kaynakGorselGenislikGecerli.
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

// Varsayılan src, yuvanın basamakları ARASINDA olmalı.
//
// NEDEN: srcset'i olmayan tarayıcı src'yi indirir. Basamak dışı bir değer
// ucun listesinde olmayabilir → 404 → görselsiz sayfa.
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
		if !strings.Contains(bilgi.Sizes, "max-width: 720px") {
			t.Errorf("yuva %d: mobil dalı yok (%q)", yuva, bilgi.Sizes)
		}
	}
}

// Blok lead'i hero lead'inden GENİŞ olmalı — bu ölçümün ta kendisi.
//
// NEDEN TEST: `lead bool` ile genişlik türetme denemesi tam burada kırılıyor.
// Biri yuva tipini boolean'a geri çevirmeye kalkarsa burada görünür.
func TestBlokLeadHeroLeadtenGenis(t *testing.T) {
	// .cd-block 2fr/1fr gap 28 → (1240-28)*2/3 = 808
	// .cd-hero 1.55fr/1fr gap 32 → (1240-32)*1.55/2.55 = 734
	blok := yuvaBilgi[YuvaBlokLead].Sizes
	hero := yuvaBilgi[YuvaHeroLead].Sizes
	if !strings.Contains(blok, "808px") {
		t.Errorf("blok lead sizes'ında 808px yok: %q", blok)
	}
	if !strings.Contains(hero, "734px") {
		t.Errorf("hero lead sizes'ında 734px yok: %q", hero)
	}
}
