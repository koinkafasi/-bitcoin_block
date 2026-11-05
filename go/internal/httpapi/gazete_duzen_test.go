package httpapi

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"theblock/site/internal/webui"
)

// Faz 4/5 koruyucu testleri — specs/13-gazete-sayfa-duzenleyici.md

// TestGazeteZSirasiKorunur boyama sırasının z'ye göre olduğunu ve z
// bildirilmemiş eski kayıtlarda dizi sırasının korunduğunu doğrular.
//
// Katman paneli ile tuvalin aynı sırayı göstermesi şart; ayrışırsa kullanıcı
// panelde üstte gördüğü bloğu tuvalde altta bulur.
func TestGazeteZSirasiKorunur(t *testing.T) {
	// z bildirilmiş: küçükten büyüğe.
	bloklar := []gazeteBlock{
		{ID: "c", Tip: "metin", Z: 5},
		{ID: "a", Tip: "metin", Z: 1},
		{ID: "b", Tip: "metin", Z: 3},
	}
	got := gazeteBloklariSirala(bloklar)
	if sira(got) != "a,b,c" {
		t.Errorf("z sırası yanlış: %s", sira(got))
	}
	// Kaynak dilim değişmemeli (çağıran yerlerde yan etki olmasın).
	if bloklar[0].ID != "c" {
		t.Error("sıralama kaynak dilimi değiştirdi")
	}

	// z yok (hepsi 0): dizi sırası korunmalı — eski kayıtların görünümü değişmez.
	eski := []gazeteBlock{{ID: "x"}, {ID: "y"}, {ID: "z"}}
	if s := sira(gazeteBloklariSirala(eski)); s != "x,y,z" {
		t.Errorf("z yokken dizi sırası bozuldu: %s", s)
	}

	// Eşit z: kararlı sıralama, girişteki sıra korunur.
	esit := []gazeteBlock{{ID: "p", Z: 2}, {ID: "q", Z: 2}, {ID: "r", Z: 1}}
	if s := sira(gazeteBloklariSirala(esit)); s != "r,p,q" {
		t.Errorf("eşit z kararlı sıralanmadı: %s", s)
	}

	// JSON turunda z korunmalı.
	raw, err := json.Marshal(bloklar)
	if err != nil {
		t.Fatal(err)
	}
	var geri []gazeteBlock
	if err := json.Unmarshal(raw, &geri); err != nil {
		t.Fatal(err)
	}
	for i := range bloklar {
		if geri[i].Z != bloklar[i].Z {
			t.Errorf("%s: z turda değişti %d → %d", bloklar[i].ID, bloklar[i].Z, geri[i].Z)
		}
	}
}

func sira(b []gazeteBlock) string {
	ids := make([]string, len(b))
	for i, x := range b {
		ids[i] = x.ID
	}
	return strings.Join(ids, ",")
}

// TestGazeteKayitDogrula tek işlemli kaydın girdiyi sessizce kabul etmediğini
// doğrular. Tekrar eden sayfa numarası, "gönderilmeyen sayfayı sil" kuralıyla
// birleşince yanlış sayfanın silinmesine yol açabilir.
func TestGazeteKayitDogrula(t *testing.T) {
	ok := []gazeteKayitSayfa{
		{No: 1, Ad: "Kapak", Bloklar: json.RawMessage(`[]`)},
		{No: 2, Ad: "Gündem", Bloklar: json.RawMessage(`[{"id":"a"}]`)},
	}
	if err := gazeteKayitDogrula(ok); err != nil {
		t.Errorf("geçerli yük reddedildi: %v", err)
	}

	durumlar := map[string]struct {
		sayfalar []gazeteKayitSayfa
		icerir   string
	}{
		"boş":          {nil, "hiç sayfa"},
		"sıfır numara": {[]gazeteKayitSayfa{{No: 0}}, "geçersiz sayfa numarası"},
		"aşırı numara": {[]gazeteKayitSayfa{{No: 99}}, "geçersiz sayfa numarası"},
		"tekrar eden":  {[]gazeteKayitSayfa{{No: 1}, {No: 1}}, "iki kez"},
		"bozuk blok": {[]gazeteKayitSayfa{
			{No: 1, Bloklar: json.RawMessage(`{bozuk`)}}, "blok JSON'u geçersiz"},
	}
	for ad, d := range durumlar {
		err := gazeteKayitDogrula(d.sayfalar)
		if err == nil {
			t.Errorf("%s: yakalanmadı", ad)
			continue
		}
		if !strings.Contains(err.Error(), d.icerir) {
			t.Errorf("%s: %q beklenirken %q", ad, d.icerir, err.Error())
		}
	}

	// 50 sayfa sınırı: tam sınır geçmeli, bir fazlası geçmemeli.
	var elli []gazeteKayitSayfa
	for i := 1; i <= 50; i++ {
		elli = append(elli, gazeteKayitSayfa{No: i})
	}
	if err := gazeteKayitDogrula(elli); err != nil {
		t.Errorf("50 sayfa reddedildi: %v", err)
	}
}

// TestGazeteSurumDamgasi iyimser kilit damgasının kararlı ve ayırt edici
// olduğunu doğrular. Damga kararsız olsaydı her kayıt çakışma üretirdi.
func TestGazeteSurumDamgasi(t *testing.T) {
	t1 := time.Date(2026, 10, 30, 12, 0, 0, 123456789, time.UTC)
	if gazeteSurum(t1) != gazeteSurum(t1) {
		t.Error("aynı zaman farklı damga üretti")
	}
	// Farklı saat dilimi, aynı an → aynı damga (UTC'ye çevriliyor).
	t2 := t1.In(time.FixedZone("TRT", 3*3600))
	if gazeteSurum(t1) != gazeteSurum(t2) {
		t.Errorf("saat dilimi damgayı değiştirdi: %s vs %s", gazeteSurum(t1), gazeteSurum(t2))
	}
	// Nanosaniye farkı ayırt edilmeli: aksi halde hızlı ardışık kayıtlarda
	// çakışma denetimi kör kalır.
	t3 := t1.Add(time.Nanosecond)
	if gazeteSurum(t1) == gazeteSurum(t3) {
		t.Error("nanosaniye farkı damgaya yansımadı")
	}
}

// TestGazeteEditorStiliSunucudan editörün blok görünümünü sunucudan aldığını
// doğrular: stil sınıfı ve ölçek tek yerde hesaplanmalı.
//
// Ayrıştıklarında gerçekten ayrışmıştı: zemin renkleri baskıda vardı ama
// tuvalde yoktu, çünkü editör sarmalayıcıya yalnız tip sınıfını yazıyordu.
func TestGazeteEditorStiliSunucudan(t *testing.T) {
	b := gazeteBlock{ID: "k", Tip: "baslik", W: 96, H: 8, Baslik: "THEBLOCK",
		Stil: map[string]any{"zemin": "lacivert", "metinRenk": "acik", "olcek": 1.2}}
	g := gazeteBlokGovdesi(b)

	if !strings.Contains(g.Sinif, "gb-z-lacivert") || !strings.Contains(g.Sinif, "gb-m-acik") {
		t.Errorf("stil sınıfı eksik: %q", g.Sinif)
	}
	if strings.HasPrefix(g.Sinif, " ") {
		t.Errorf("sınıf dizesi boşlukla başlıyor: %q", g.Sinif)
	}
	if !strings.Contains(g.Stil, "1.20") {
		t.Errorf("ölçek değeri yok: %q", g.Stil)
	}
	if !strings.Contains(g.HTML, "THEBLOCK") {
		t.Errorf("gövde içeriği yok: %q", g.HTML)
	}
	// Kapsayıcı iç gövdede olmamalı; editör kendi sarmalayıcısını kuruyor.
	if strings.Contains(g.HTML, `class="gb `) {
		t.Errorf("iç gövde kapsayıcı taşıyor: %q", g.HTML)
	}

	// Baskı yolu aynı sınıfı üretmeli: editör ile baskı ayrışmamalı.
	baski := gazeteBlockHTML(b)
	for _, sinif := range strings.Fields(g.Sinif) {
		if !strings.Contains(baski, sinif) {
			t.Errorf("baskı %q sınıfını taşımıyor", sinif)
		}
	}
}

// TestGazeteEditorBetigiSunucuSinifiKullanir editörün stil hesabını JS'te
// tekrarlamadığını doğrular. Tekrarlanırsa iki uygulama olur ve ayrışırlar.
func TestGazeteEditorBetigiSunucuSinifiKullanir(t *testing.T) {
	b, err := webui.FS.ReadFile("static/gazete-editor.js")
	if err != nil {
		t.Fatal(err)
	}
	js := string(b)
	if !strings.Contains(js, "g.sinif") {
		t.Error("editör sunucudan gelen stil sınıfını uygulamıyor")
	}
	// Jeton adları JS'te hesaplanmamalı.
	for _, yasak := range []string{"gb-z-", "gb-m-", "gb-b-", "gb-k-ince"} {
		if strings.Contains(js, yasak) {
			t.Errorf("stil jetonu %q JS'te üretiliyor; hesap sunucuda kalmalı", yasak)
		}
	}
}
