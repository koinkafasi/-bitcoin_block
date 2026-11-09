package repo

import (
	"context"
	"os"
	"testing"
)

// TestHisseAralikEsleme CoinDesk sekmelerinin (1D·5D·1M·YTD·1Y·All) Yahoo
// parametrelerine doğru eşlendiğini kilitler. Yanlış interval/range uçtan
// boş seri döndürür ve grafik sessizce kaybolurdu.
func TestHisseAralikEsleme(t *testing.T) {
	beklenen := map[string][2]string{
		"1d":  {"5m", "1d"},
		"5d":  {"30m", "5d"},
		"1m":  {"1d", "1mo"},
		"ytd": {"1d", "ytd"},
		"1y":  {"1d", "1y"},
		"all": {"1mo", "max"},
	}
	if len(HisseAraliklari) != len(beklenen) {
		t.Fatalf("sekme sayısı %d, beklenen %d", len(HisseAraliklari), len(beklenen))
	}
	for _, a := range HisseAraliklari {
		b, ok := beklenen[a.Kod]
		if !ok {
			t.Fatalf("beklenmeyen sekme kodu: %q", a.Kod)
		}
		if a.Interval != b[0] || a.Range != b[1] {
			t.Errorf("%s: interval/range = %s/%s, beklenen %s/%s", a.Kod, a.Interval, a.Range, b[0], b[1])
		}
		if a.Ad == "" {
			t.Errorf("%s: sekme etiketi boş", a.Kod)
		}
	}
	// Sekme sırası CoinDesk ile aynı olmalı: 1D, 5D, 1M, YTD, 1Y, All.
	sira := []string{"1D", "5D", "1M", "YTD", "1Y", "All"}
	for i, ad := range sira {
		if HisseAraliklari[i].Ad != ad {
			t.Errorf("%d. sekme %q, beklenen %q", i, HisseAraliklari[i].Ad, ad)
		}
	}
}

// TestHisseAralikBulBilinmeyen bozuk parametre sayfayı düşürmemeli.
func TestHisseAralikBulBilinmeyen(t *testing.T) {
	for _, g := range []string{"", "2y", "  ", "1D ", "abc"} {
		if got := HisseAralikBul(g); got.Kod != "1d" {
			t.Errorf("%q → %q, beklenen 1d", g, got.Kod)
		}
	}
	if got := HisseAralikBul(" YTD "); got.Kod != "ytd" {
		t.Errorf("boşluklu/büyük harf eşleşmedi: %q", got.Kod)
	}
}

// TestHisseSembolGecerli adres parçası doğrudan uca girdiği için katı olmalı.
func TestHisseSembolGecerli(t *testing.T) {
	gecerli := []string{"META", "AAPL", "BRK-B", "GC=F", "7203.T"}
	for _, s := range gecerli {
		if !HisseSembolGecerli(s) {
			t.Errorf("%q geçerli olmalıydı", s)
		}
	}
	gecersiz := []string{"", "meta", "../etc", "META/../X", "A B", "ÇOK-UZUN-SEMBOL", "1234567890123"}
	for _, s := range gecersiz {
		if HisseSembolGecerli(s) {
			t.Errorf("%q geçersiz olmalıydı", s)
		}
	}
}

// TestHisseSeriKisaltma 240 nokta sınırını ve son noktanın korunmasını kilitler.
func TestHisseSeriKisaltma(t *testing.T) {
	const n = 500
	ts := make([]int64, n)
	q := yahooQuote{}
	for i := 0; i < n; i++ {
		ts[i] = int64(i)
		v := float64(i)
		q.Close = append(q.Close, &v)
	}
	out := hisseSeri(ts, q)
	if len(out) != 240 {
		t.Fatalf("nokta sayısı %d, beklenen 240", len(out))
	}
	if out[len(out)-1].T != int64(n-1) {
		t.Errorf("son nokta %d, beklenen %d (grafik güncel fiyatta bitmeli)", out[len(out)-1].T, n-1)
	}
	if out[0].T != 0 {
		t.Errorf("ilk nokta %d, beklenen 0", out[0].T)
	}
}

// TestHisseSeriNilAtla boş kapanışlar (borsa kapalı saatler) atlanmalı.
func TestHisseSeriNilAtla(t *testing.T) {
	ts := []int64{1, 2, 3}
	a, c := 1.0, 3.0
	q := yahooQuote{Close: []*float64{&a, nil, &c}}
	out := hisseSeri(ts, q)
	if len(out) != 2 || out[0].C != 1 || out[1].C != 3 {
		t.Fatalf("beklenmeyen seri: %+v", out)
	}
}

// TestHisseCekCanli gerçek uca bağlanır — yalnız BC_CANLI_AG=1 iken koşar.
// CI'da ağ yok; elle doğrulama için var.
func TestHisseCekCanli(t *testing.T) {
	if os.Getenv("BC_CANLI_AG") != "1" {
		t.Skip("BC_CANLI_AG=1 değil: ağ testi atlandı")
	}
	d, err := HisseCek(context.Background(), "META", "1y")
	if err != nil {
		t.Fatalf("HisseCek: %v", err)
	}
	if d.Ad == "" || d.Para == "" || d.Fiyat <= 0 {
		t.Fatalf("eksik kotasyon: %+v", d)
	}
	if d.H52Maks <= 0 || d.H52Min <= 0 {
		t.Errorf("52 hafta aralığı boş: %v–%v", d.H52Min, d.H52Maks)
	}
	if len(d.Noktalar) < 50 {
		t.Errorf("1y serisi %d nokta, en az 50 beklenirdi", len(d.Noktalar))
	}
	if d.Hacim <= 0 {
		t.Errorf("hacim boş")
	}
	t.Logf("%s (%s) %s %.2f %+.2f%% 52h %.2f–%.2f nokta=%d",
		d.Ad, d.Sembol, d.Para, d.Fiyat, d.Degisim, d.H52Min, d.H52Maks, len(d.Noktalar))
}
