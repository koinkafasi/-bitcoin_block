package aieditor

import (
	"strings"
	"testing"
)

// ÜRETİMDE GÖRÜLEN ARIZA (içerik 10764, 13 Kasım 2026): model 4
// paragraf üretip 12 kez tekrarlamış — 48 blok, 3.000 kelime, hepsi
// aynı dört cümle. yapi.go bunu yakalıyordu ama yalnız UYARI olarak,
// yani bozuk metin taslağa yazılıyordu.
func TestYazimSonucuDonguyuReddeder(t *testing.T) {
	dort := []string{
		"ENGIE ve Google şirketleri, Fransa'nın Châteauroux kentinde yerel ekonomi üzerinde etkili bir yatırım planlıyor.",
		"Planlanan yatırımın bölgedeki enerji tüketim modellerini kökten dönüştürme potansiyeli bulunuyor ve tartışma yaratıyor.",
		"Ancak projeye yönelik beklentilerin yanı sıra çevresel etkiler konusunda ciddi sorular da gündeme geliyor bu süreçte.",
		"Bu durum topluluk içerisinde projenin faydaları ve riskleri üzerine derin görüş ayrılıklarına yol açmış durumda şu an.",
	}
	var govde []string
	for i := 0; i < 12; i++ {
		govde = append(govde, dort...)
	}

	res := Result{Fields: map[string]string{
		"title": "ENGIE ve Google'dan Veri Merkezi Planı",
		"body":  strings.Join(govde, "\n\n"),
	}}
	if _, ok := yazimSonucu(res); ok {
		t.Fatal("12 kez tekrarlanan gövde kabul edildi; içeriğe yazılırdı")
	}
}

// Sağlıklı metin elenmemeli: kapı yalnız döngüyü kesmeli.
func TestYazimSonucuSaglikliMetniGecirir(t *testing.T) {
	govde := strings.Join([]string{
		"Birinci paragraf olayı anlatıyor ve 5N1K sorularına cevap veriyor burada okuyucu için.",
		"İkinci paragraf arka planı veriyor, rakamın ne anlama geldiğini açıklıyor ayrıntılı biçimde.",
		"Üçüncü paragraf tarafların söylediklerini aktarıyor ve piyasaya etkisini tartışıyor burada.",
		"Dördüncü paragraf bundan sonra ne olacağını ele alıyor ve beklentileri sıralıyor okuyucuya.",
	}, "\n\n")

	y, ok := yazimSonucu(Result{Fields: map[string]string{
		"title": "Başlık", "body": govde,
	}})
	if !ok {
		t.Fatal("sağlıklı metin reddedildi")
	}
	if len(y.Paragraf) != 4 {
		t.Errorf("paragraf sayısı = %d, 4 beklenir", len(y.Paragraf))
	}
}

// Az sayıda tekrar metni öldürmemeli: yinelenen atılır, gerisi kalır.
func TestYazimSonucuTekTekrariTemizler(t *testing.T) {
	a := "Birinci paragraf olayı anlatıyor ve 5N1K sorularına cevap veriyor burada okuyucu için."
	b := "İkinci paragraf arka planı veriyor ve rakamın anlamını açıklıyor ayrıntılı biçimde burada."
	c := "Üçüncü paragraf tarafların sözlerini aktarıyor, piyasaya etkisini tartışıyor ve kapatıyor."
	d := "Dördüncü paragraf bundan sonrasını ele alıyor, beklentileri okuyucuya sıralayarak bitiriyor."
	// 5 paragraf, 1'i tekrar -> yarıdan az, kurtarilmali
	govde := strings.Join([]string{a, b, c, d, a}, "\n\n")

	y, ok := yazimSonucu(Result{Fields: map[string]string{"title": "T", "body": govde}})
	if !ok {
		t.Fatal("tek tekrar yüzünden metin tümüyle reddedildi")
	}
	if len(y.Paragraf) != 4 {
		t.Errorf("paragraf = %d, tekrar atılıp 4 kalmalıydı", len(y.Paragraf))
	}
}

func TestParagraflariTekille(t *testing.T) {
	uzun1 := "Bu paragraf kırk karakterden uzun olduğu için tekilleştirme kapsamına giriyor burada."
	uzun2 := "Bu ikinci paragraf da kırk karakterden uzun ve içerik olarak tamamen farklı bir şey söylüyor."

	t.Run("birebir tekrar atilir", func(t *testing.T) {
		out, yin := paragraflariTekille([]string{uzun1, uzun2, uzun1})
		if len(out) != 2 || yin != 1 {
			t.Errorf("out=%d yinelenen=%d, 2/1 beklenir", len(out), yin)
		}
	})

	// Boşluk farkı tekrar sayılmalı: model aynı cümleyi fazladan
	// girinti veya çift boşlukla yineleyebiliyor.
	//
	// NOT — harf büyüklüğü için Türkçe'de ToUpper/ToLower gidiş-dönüşte
	// AYNI harfi vermiyor ("ı"/"i" çifti), o yüzden tam metni büyük
	// harfe çevirip tekrar beklemek yanlış bir sınav olurdu. Normalize
	// yalnız baş harf gibi gerçekçi farkları yakalar.
	t.Run("bosluk farki tekrar sayilir", func(t *testing.T) {
		out, yin := paragraflariTekille([]string{uzun1, "   " + strings.ReplaceAll(uzun1, " ", "  ") + "  "})
		if len(out) != 1 || yin != 1 {
			t.Errorf("out=%d yinelenen=%d; boşluk farkı tekrar sayılmalı", len(out), yin)
		}
	})

	t.Run("bas harf farki tekrar sayilir", func(t *testing.T) {
		out, yin := paragraflariTekille([]string{uzun1, "bu" + uzun1[2:]})
		if len(out) != 1 || yin != 1 {
			t.Errorf("out=%d yinelenen=%d; baş harf farkı tekrar sayılmalı", len(out), yin)
		}
	})

	t.Run("kisa satirlar korunur", func(t *testing.T) {
		// Ara başlık gibi kısa satırlar meşru olarak tekrarlanabilir.
		out, yin := paragraflariTekille([]string{"Özet", "Özet", "Özet"})
		if len(out) != 3 || yin != 0 {
			t.Errorf("out=%d yinelenen=%d; kısa satır elenmemeli", len(out), yin)
		}
	})

	t.Run("bos giris", func(t *testing.T) {
		out, yin := paragraflariTekille(nil)
		if len(out) != 0 || yin != 0 {
			t.Errorf("out=%d yinelenen=%d", len(out), yin)
		}
	})
}
