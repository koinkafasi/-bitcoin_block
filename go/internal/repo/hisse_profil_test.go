package repo

import (
	"context"
	"os"
	"testing"
)

// TestOzetKirp cümle sınırında kısaltmayı kilitler.
func TestOzetKirp(t *testing.T) {
	uzun := "Birinci cümle burada. İkinci cümle de burada ve daha uzun. " +
		"Üçüncü cümle ise çok daha uzun olacak şekilde yazıldı ki kırpma devreye girsin."

	// Sınır cümle sonundan SONRA geliyorsa cümle sonunda kesilir.
	got := OzetKirp(uzun, 70)
	if len(got) > 70 {
		t.Errorf("kırpma sınırı aştı: %d > 70 (%q)", len(got), got)
	}
	if len(got) > 0 && got[len(got)-1] != '.' {
		t.Errorf("cümle sonunda kesilmedi: %q", got)
	}

	// Sınır çok küçükse cümle sonu aranmaz, "…" ile kesilir ve sınır YİNE
	// aşılmaz (… 3 bayt yer kaplar).
	kucuk := OzetKirp(uzun, 40)
	if len(kucuk) > 40 {
		t.Errorf("küçük sınırda aşım: %d > 40 (%q)", len(kucuk), kucuk)
	}

	// Kısa metin aynen döner.
	kisa := "Kısa özet."
	if got := OzetKirp(kisa, 500); got != kisa {
		t.Errorf("kısa metin değişti: %q", got)
	}

	// Boşluk temizlenir (satır sonları tek boşluğa iner).
	if got := OzetKirp("a\n\nb\tc", 500); got != "a b c" {
		t.Errorf("boşluk normalizasyonu yanlış: %q", got)
	}
}

// TestYahooSayiCoz hem {"raw":…} sarmalını hem düz sayıyı çözmeli.
func TestYahooSayiCoz(t *testing.T) {
	kaynak := []byte(`{
		"quoteSummary":{"result":[{
			"assetProfile":{"longBusinessSummary":"Şirket tanımı.","sector":"Technology",
				"industry":"Consumer Electronics","fullTimeEmployees":{"raw":167000,"fmt":"167k"},
				"website":"https://www.apple.com"},
			"summaryDetail":{"marketCap":{"raw":5065410084864},"beta":1.089,
				"averageVolume":{"raw":42880493}},
			"price":{"marketCap":{"raw":5065410084864}}
		}]}}`)

	p, err := hisseProfilCoz(kaynak)
	if err != nil {
		t.Fatalf("çözülemedi: %v", err)
	}
	if p.Sektor != "Technology" || p.Endustri != "Consumer Electronics" {
		t.Errorf("sektör yanlış: %q / %q", p.Sektor, p.Endustri)
	}
	if p.Calisan != 167000 {
		t.Errorf("çalışan %d, beklenen 167000", p.Calisan)
	}
	if p.PiyasaDeger != 5065410084864 {
		t.Errorf("piyasa değeri %v", p.PiyasaDeger)
	}
	if p.Beta < 1.08 || p.Beta > 1.10 {
		t.Errorf("beta %v", p.Beta)
	}
	if p.Web != "https://www.apple.com" {
		t.Errorf("web %q", p.Web)
	}
	if p.Bos() {
		t.Errorf("dolu profil Bos() dedi")
	}
}

// TestHisseProfilCozBos boş yanıt hata döndürmeli (şablon bölümü basmasın).
func TestHisseProfilCozBos(t *testing.T) {
	for _, ham := range []string{
		`{"quoteSummary":{"result":[]}}`,
		`{"quoteSummary":{"result":[{"assetProfile":{}}]}}`,
		`{}`,
	} {
		if _, err := hisseProfilCoz([]byte(ham)); err == nil {
			t.Errorf("boş yanıt hata vermedi: %s", ham)
		}
	}
}

// TestHisseProfilCekCanli gerçek uca bağlanır (BC_CANLI_AG=1).
//
// Ölçülen (9 Kasım 2026): crumb akışı çerez + getcrumb ile çalışıyor;
// hem ABD hem BIST sembolünde longBusinessSummary ve marketCap geliyor.
func TestHisseProfilCekCanli(t *testing.T) {
	if os.Getenv("BC_CANLI_AG") != "1" {
		t.Skip("BC_CANLI_AG=1 değil: ağ testi atlandı")
	}
	for _, sym := range []string{"META", "THYAO.IS"} {
		p, err := HisseProfilCek(context.Background(), sym)
		if err != nil {
			t.Fatalf("%s: %v", sym, err)
		}
		if len(p.Ozet) < 100 {
			t.Errorf("%s: iş tanımı kısa (%d)", sym, len(p.Ozet))
		}
		if p.PiyasaDeger <= 0 {
			t.Errorf("%s: piyasa değeri yok", sym)
		}
		if p.Sektor == "" {
			t.Errorf("%s: sektör yok", sym)
		}
		t.Logf("%s: %s / %s · çalışan=%d · piyasa=%.0f · özet=%d krkt",
			sym, p.Sektor, p.Endustri, p.Calisan, p.PiyasaDeger, len(p.Ozet))
	}
}
