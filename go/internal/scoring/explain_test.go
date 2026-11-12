package scoring

import (
	"strings"
	"testing"
)

func TestExplainKalemToplamiPuaniVerir(t *testing.T) {
	cases := []Input{
		{},
		{Title: "Kısa"},
		{Title: strings.Repeat("a", 50), Spot: strings.Repeat("b", 130), TagCount: 5, HasCover: true, HasSource: true},
		{Title: strings.Repeat("a", 30), Spot: strings.Repeat("b", 70), TagCount: 1},
		{Title: strings.Repeat("a", 200), Spot: strings.Repeat("b", 30), TagCount: 0},
		{Title: "deprem deprem bölgesi bölgesi haberi haberi geldi", Spot: strings.Repeat("c", 200), TagCount: 3, HasCover: true, HasSource: true},
	}
	for _, in := range cases {
		puan, sig := Explain(in)
		ham := 0
		for _, s := range sig {
			ham += s.Puan
		}
		if clamp(ham) != puan {
			t.Errorf("kalem toplamı %d (kırpılmış %d) ≠ puan %d — girdi %+v",
				ham, clamp(ham), puan, in)
		}
		rel, note := Relevance(in)
		if rel != puan {
			t.Errorf("Relevance %d ≠ Explain %d — girdi %+v", rel, puan, in)
		}
		if note != Note {
			t.Errorf("not değişmiş: %q", note)
		}
	}
}

// Aritmetik DEĞİŞMEMELİ: bu değerler Explain eklenmeden önceki formülden
// elle hesaplandı. Kırılırsa kayıtlı puanlar yenileriyle kıyaslanamaz.
func TestPuanAritmetigiSabit(t *testing.T) {
	cases := []struct {
		ad   string
		in   Input
		want int
	}{
		{"bos", Input{}, 0}, // 20 -10(başlık<10) -10(spot boş) = 0
		{"ideal", Input{Title: strings.Repeat("a", 50), Spot: strings.Repeat("b", 130), TagCount: 3, HasCover: true, HasSource: true}, 100}, // 20+20+25+15+10+10 = 100
		{"orta", Input{Title: strings.Repeat("a", 30), Spot: strings.Repeat("b", 70), TagCount: 1}, 55},                                     // 20+12+15+8 = 55
		{"uzun-baslik-kisa-spot", Input{Title: strings.Repeat("a", 200), Spot: strings.Repeat("b", 30)}, 20},                                // 20+0+0 = 20
		{"yalniz-kaynak", Input{Title: strings.Repeat("a", 50), Spot: strings.Repeat("b", 130), HasSource: true}, 75},                       // 20+20+25+10 = 75
	}
	for _, c := range cases {
		if got, _ := Relevance(c.in); got != c.want {
			t.Errorf("%s: %d, beklenen %d", c.ad, got, c.want)
		}
	}
}

// Eksik sinyaller editöre ne yapacağını söylemeli; ipucusuz "eksik" kalem
// gerekçe satırını işe yaramaz hâle getirir.
func TestEksikKalemlerIpucuTasir(t *testing.T) {
	_, sig := Explain(Input{Title: "Çok kısa"})
	eksik := 0
	for _, s := range sig {
		if s.Durum == DurumEksik {
			eksik++
			if strings.TrimSpace(s.Ipucu) == "" {
				t.Errorf("%q eksik ama ipucusuz", s.Ad)
			}
		}
	}
	if eksik == 0 {
		t.Fatal("eksik sinyal bekleniyordu")
	}
}

// "bilgi" kalemleri editörün eyleme çeviremeyeceği kalemlerdir; onlara ipucu
// konmamalı — özellikle "başlıkta sözcük tekrarla" demek istemiyoruz.
func TestBilgiKalemleriIpucusuz(t *testing.T) {
	_, sig := Explain(Input{Title: strings.Repeat("a", 50)})
	for _, s := range sig {
		if s.Durum == DurumBilgi && s.Ipucu != "" {
			t.Errorf("%q bilgi kalemi ama ipucu taşıyor: %q", s.Ad, s.Ipucu)
		}
	}
}
