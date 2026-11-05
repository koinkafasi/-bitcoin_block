package scoring

import (
	"strings"
	"testing"
	"time"
)

func tamIcerik() Input {
	return Input{
		Title:     strings.Repeat("a", 50),
		Spot:      strings.Repeat("b", 130),
		TagCount:  3,
		HasCover:  true,
		HasSource: true,
	}
}

func TestTazelikSonumu(t *testing.T) {
	cases := []struct {
		yas  time.Duration
		want int
	}{
		{0, 100},
		{12 * time.Hour, 85},
		{24 * time.Hour, 70},
		{48 * time.Hour, 58},
		{72 * time.Hour, 47},
		{168 * time.Hour, 0},
		{30 * 24 * time.Hour, 0},
	}
	for _, c := range cases {
		if got := Tazelik(c.yas); got != c.want {
			t.Errorf("Tazelik(%v) = %d, beklenen %d", c.yas, got, c.want)
		}
	}
	// Negatif yaş (saat kayması) 100 vermeli, panik değil.
	if got := Tazelik(-time.Hour); got != 100 {
		t.Errorf("negatif yaş %d", got)
	}
}

func TestTazelikMonotonAzalir(t *testing.T) {
	onceki := 101
	for s := 0; s <= 200; s += 4 {
		v := Tazelik(time.Duration(s) * time.Hour)
		if v > onceki {
			t.Fatalf("%d saatte tazelik arttı: %d > %d", s, v, onceki)
		}
		onceki = v
	}
}

func TestTrendUyumu(t *testing.T) {
	simdi := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	// Eşleşme yoksa 0.
	if got := TrendUyumu(0, simdi, simdi, 0); got != 0 {
		t.Errorf("eşleşmesiz %d", got)
	}
	// Taze eşleşme tam puan.
	if got := TrendUyumu(90, simdi, simdi, 0); got != 90 {
		t.Errorf("taze eşleşme %d, beklenen 90", got)
	}
	// Terim 48 saatten eskiyse sinyal sayılmaz.
	if got := TrendUyumu(100, simdi.Add(-49*time.Hour), simdi, 0); got != 0 {
		t.Errorf("eski terim %d, beklenen 0", got)
	}
	// 24 saatlik terim yarı ağırlık.
	if got := TrendUyumu(100, simdi.Add(-24*time.Hour), simdi, 0); got != 50 {
		t.Errorf("24 saatlik terim %d, beklenen 50", got)
	}
	// Trafik bonusu eklenir ama 100'ü aşmaz.
	if got := TrendUyumu(100, simdi, simdi, 100000); got != 100 {
		t.Errorf("trafik bonusu kırpılmadı: %d", got)
	}
	if got := TrendUyumu(60, simdi, simdi, 100000); got != 70 {
		t.Errorf("trafik bonusu %d, beklenen 70", got)
	}
	// Zaman bilinmiyorsa (sıfır zaman) eşleşme tam sayılır.
	if got := TrendUyumu(80, time.Time{}, simdi, 0); got != 80 {
		t.Errorf("zamansız eşleşme %d", got)
	}
}

func TestKaynakGuveni(t *testing.T) {
	if got := KaynakGuveni(85, false); got != 85 {
		t.Errorf("%d", got)
	}
	if got := KaynakGuveni(0, false); got != 50 {
		t.Errorf("bilinmeyen kaynak %d, beklenen 50", got)
	}
	if got := KaynakGuveni(70, true); got != 55 {
		t.Errorf("agregatör %d, beklenen 55", got)
	}
	if got := KaynakGuveni(10, true); got != 0 {
		t.Errorf("alt sınır %d", got)
	}
}

// Eksenlerin asıl işi AYIRT ETMEK. Eski ai_score taslakların %88'ini tek
// banda koyuyordu; bu test yeni önceliğin gerçekten yayıldığını sabitler.
func TestOncelikAyirtEdiyor(t *testing.T) {
	simdi := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	// Ölçülen korpusu taklit et: yaş 0-182 saate yayılmış, kaynak güveni
	// 60-90, çok azı trendle eşleşiyor.
	var puanlar []int
	guvenler := []int{60, 65, 70, 72, 75, 78, 80, 85, 90}
	for saat := 0; saat <= 182; saat += 2 {
		g := OncelikGirdi{
			Tamlik:          tamIcerik(),
			KaynakGuven:     guvenler[(saat/2)%len(guvenler)],
			OlusturmaZamani: simdi.Add(-time.Duration(saat) * time.Hour),
			Simdi:           simdi,
		}
		if saat%40 == 0 { // ~%5'i trendle eşleşsin
			g.TrendEslesme = 90
			g.TrendSonGorulme = simdi
		}
		puanlar = append(puanlar, Hesapla(g).Oncelik)
	}

	bant := map[int]int{}
	for _, p := range puanlar {
		bant[p/10]++
	}
	enKalabalik := 0
	for _, n := range bant {
		if n > enKalabalik {
			enKalabalik = n
		}
	}
	pay := float64(enKalabalik) / float64(len(puanlar))
	if pay > 0.45 {
		t.Errorf("öncelik hâlâ tek bantta toplanıyor: en kalabalık 10'luk bant %%%.0f (eski ai_score %%88 idi); dağılım %v",
			pay*100, bant)
	}
	if len(bant) < 5 {
		t.Errorf("yalnız %d farklı bant var, yayılım yetersiz: %v", len(bant), bant)
	}
}

// Trendle eşleşen içerik, her şeyi eşit olan eşleşmeyenden ÖNDE olmalı.
func TestTrendEslesmesiOneCikarir(t *testing.T) {
	simdi := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	temel := OncelikGirdi{
		Tamlik:          tamIcerik(),
		KaynakGuven:     70,
		OlusturmaZamani: simdi.Add(-72 * time.Hour),
		Simdi:           simdi,
	}
	trendli := temel
	trendli.TrendEslesme = 90
	trendli.TrendSonGorulme = simdi

	a := Hesapla(temel).Oncelik
	b := Hesapla(trendli).Oncelik
	if b <= a {
		t.Errorf("trendli %d, trendsiz %d — trend öne çıkarmıyor", b, a)
	}
	// Üç günlük ama trend olan, bir günlük ama trendsizden önde olmalı:
	// güncellik tek başına trend talebini yenmemeli.
	birGunluk := temel
	birGunluk.OlusturmaZamani = simdi.Add(-24 * time.Hour)
	if Hesapla(trendli).Oncelik <= Hesapla(birGunluk).Oncelik {
		t.Errorf("trendli(72s)=%d, trendsiz(24s)=%d — trend ağırlığı yetersiz",
			Hesapla(trendli).Oncelik, Hesapla(birGunluk).Oncelik)
	}
}

func TestEksenlerSinirlariAsmaz(t *testing.T) {
	simdi := time.Now()
	girdiler := []OncelikGirdi{
		{},
		{Tamlik: tamIcerik(), TrendEslesme: 999, KaynakGuven: 999, Simdi: simdi},
		{TrendEslesme: -5, KaynakGuven: -5, OlusturmaZamani: simdi.Add(time.Hour), Simdi: simdi},
	}
	for i, g := range girdiler {
		e := Hesapla(g)
		for ad, v := range map[string]int{
			"tamlik": e.Tamlik, "trend": e.Trend, "guven": e.Guven,
			"tazelik": e.Tazelik, "oncelik": e.Oncelik,
		} {
			if v < 0 || v > 100 {
				t.Errorf("girdi %d: %s = %d, 0-100 dışında", i, ad, v)
			}
		}
	}
}

// Simdi verilmezse time.Now() kullanılmalı; sıfır zaman 1 yılı 1'e
// çevirip tazeliği sıfırlamamalı.
func TestSimdiVarsayilani(t *testing.T) {
	e := Hesapla(OncelikGirdi{
		Tamlik:          tamIcerik(),
		OlusturmaZamani: time.Now().Add(-time.Hour),
	})
	if e.Tazelik < 90 {
		t.Errorf("bir saatlik içeriğin tazeliği %d, ~97 bekleniyordu", e.Tazelik)
	}
}
