package httpapi

import (
	"encoding/json"
	"strings"
	"testing"
)

// Faz 3 koruyucu testleri — specs/13-gazete-sayfa-duzenleyici.md

// TestGazeteSablonSemasi projeyle gelen her hazır şablonun şemayı geçtiğini
// doğrular. Bozuk şablon, bozuk gazete demek: sayfa dışına taşan blok baskıda
// kırpılır, tipi bilinmeyen blok hiç basılmaz.
func TestGazeteSablonSemasi(t *testing.T) {
	tohumlar := gazeteTohumSablonlari()
	if len(tohumlar) < 5 {
		t.Fatalf("en az 5 hazır şablon beklenir, %d var", len(tohumlar))
	}
	adlar := map[string]bool{}
	for _, s := range tohumlar {
		if adlar[s.Ad] {
			t.Errorf("şablon adı tekrar ediyor: %q", s.Ad)
		}
		adlar[s.Ad] = true
		if s.Aciklama == "" {
			t.Errorf("%s: açıklama yok (galeride ne olduğu anlaşılmaz)", s.Ad)
		}
		if sorun := gazeteSablonDogrula(s.Doc); len(sorun) > 0 {
			t.Errorf("%s geçersiz:\n  - %s", s.Ad, strings.Join(sorun, "\n  - "))
		}
		// Her şablon sayfa boyunu ve tipografi ölçeğini bildirmeli (K2).
		if s.Doc.Sayfa.Boyut == "" || s.Doc.Sayfa.TipOlcek <= 0 {
			t.Errorf("%s: sayfa boyu/tipografi ölçeği bildirilmemiş: %+v", s.Ad, s.Doc.Sayfa)
		}
	}
	// Referans peyzajı mutlaka bulunmalı.
	if !adlar["A4 — Klasik Manşet"] {
		t.Error("referans peyzajı şablonu yok")
	}
	// Eski A3 görünümünü koruyan şablon da bulunmalı (K2 sözü).
	if !adlar["A3 — Mevcut Düzen"] {
		t.Error("A3 şablonu yok; eski görünüm yeniden üretilemez")
	}
}

// TestGazeteKlasikMansetAnatomisi referans peyzajının yedi bölgesinin
// şablonda karşılığı olduğunu doğrular (specs/13 §3).
func TestGazeteKlasikMansetAnatomisi(t *testing.T) {
	kapak := gazeteKlasikMansetKapak()
	byID := map[string]gazeteBlock{}
	for _, b := range kapak.Blocks {
		byID[b.ID] = b
	}
	for _, beklenen := range []struct {
		id, tip, neden string
	}{
		{"k-ust", "kunye", "sayı no / tarih şeridi"},
		{"k-bant", "baslik", "lacivert künye bandı"},
		{"k-kat", "baslik", "kategori şeridi"},
		{"k-hero", "manset", "kahraman görsel + bindirmeli metin"},
		{"k-yan", "kart", "renkli yan kutu"},
		{"k-o2", "gorsel", "orta şerit görseli"},
		{"k-a1", "manset", "alt büyük manşet"},
		{"k-alt", "kunye", "alt bilgi"},
	} {
		b, ok := byID[beklenen.id]
		if !ok {
			t.Errorf("%s (%s) eksik", beklenen.id, beklenen.neden)
			continue
		}
		if b.Tip != beklenen.tip {
			t.Errorf("%s: %q beklenirken %q", beklenen.id, beklenen.tip, b.Tip)
		}
	}
	// Lacivert bant ve bindirme gerçekten stil taşımalı.
	if s := byID["k-bant"].Stil["zemin"]; s != "lacivert" {
		t.Errorf("künye bandı lacivert değil: %v", s)
	}
	if s := byID["k-hero"].Stil["bindirme"]; s != "alt" {
		t.Errorf("kahraman görselde bindirme yok: %v", s)
	}
}

// TestGazeteSablonCozGeriyeUyumlu çözücünün hem yeni belge biçimini hem eski
// düz sayfa dizisini kabul ettiğini doğrular.
func TestGazeteSablonCozGeriyeUyumlu(t *testing.T) {
	// Eski biçim: yalnız dizi → A4 varsayılanlarıyla sarılmalı.
	eski := []byte(`[{"no":1,"ad":"Kapak","bloklar":[{"id":"a","tip":"metin","x":1,"y":1,"w":10,"h":10}]}]`)
	d, err := gazeteSablonCoz(eski)
	if err != nil {
		t.Fatalf("eski biçim çözülemedi: %v", err)
	}
	if len(d.Sayfalar) != 1 || d.Sayfa.Boyut != "A4" || d.Surum != 1 {
		t.Errorf("eski biçim varsayılanlarla sarılmadı: %+v", d)
	}

	// Yeni biçim: bildirilen değerler korunmalı.
	yeni := []byte(`{"surum":1,"sayfa":{"boyut":"A3","tipOlcek":1.2},
	                 "izgara":{"sutun":6,"oluk":5,"kenar":10},
	                 "sayfalar":[{"no":1,"ad":"K","bloklar":[]}]}`)
	d, err = gazeteSablonCoz(yeni)
	if err != nil {
		t.Fatalf("yeni biçim çözülemedi: %v", err)
	}
	if d.Sayfa.Boyut != "A3" || d.Sayfa.TipOlcek != 1.2 || d.Izgara.Sutun != 6 {
		t.Errorf("bildirilen değerler korunmadı: %+v", d)
	}

	// Boş/bozuk girdi sessizce geçerli sayılmamalı.
	for _, bozuk := range [][]byte{nil, []byte(""), []byte("null"), []byte("{bozuk")} {
		if _, err := gazeteSablonCoz(bozuk); err == nil {
			t.Errorf("bozuk girdi %q hatasız geçti", bozuk)
		}
	}
}

// TestGazeteSablonDogrulaYakalar doğrulayıcının gerçek hataları gördüğünü
// doğrular. Yakalamazsa bozuk şablon sessizce yüklenir.
func TestGazeteSablonDogrulaYakalar(t *testing.T) {
	temel := func(b gazeteBlock) gazeteSablonDoc {
		return gazeteSablonDoc{
			Surum: 1, Sayfa: gazeteSablonVarsayilanSayfa(),
			Izgara:   gazeteSablonVarsayilanIzgara(),
			Sayfalar: []gazetePageDoc{{No: 1, Ad: "K", Blocks: []gazeteBlock{b}}},
		}
	}
	durumlar := map[string]struct {
		doc    gazeteSablonDoc
		icerir string
	}{
		"sağdan taşma":    {temel(gzBlok("a", "metin", 60, 2, 50, 10)), "sağdan taşıyor"},
		"alttan taşma":    {temel(gzBlok("a", "metin", 2, 80, 10, 50)), "alttan taşıyor"},
		"bilinmeyen tip":  {temel(gzBlok("a", "uçan-blok", 2, 2, 10, 10)), "bilinmeyen tip"},
		"sıfır ölçü":      {temel(gzBlok("a", "metin", 2, 2, 0, 10)), "ölçüsü sıfır"},
		"kimliksiz":       {temel(gzBlok("", "metin", 2, 2, 10, 10)), "kimliksiz"},
		"negatif konum":   {temel(gzBlok("a", "metin", -5, 2, 10, 10)), "dışında başlıyor"},
		"geçersiz stil":   {temel(gzBlok("a", "metin", 2, 2, 10, 10).stil("zemin", "pembe")), "geçersiz değer"},
		"bilinmeyen stil": {temel(gzBlok("a", "metin", 2, 2, 10, 10).stil("gölge", "var")), "bilinmeyen stil anahtarı"},
	}
	for ad, d := range durumlar {
		sorun := gazeteSablonDogrula(d.doc)
		if len(sorun) == 0 {
			t.Errorf("%s: yakalanmadı", ad)
			continue
		}
		if !strings.Contains(strings.Join(sorun, " "), d.icerir) {
			t.Errorf("%s: %q beklenirken %v", ad, d.icerir, sorun)
		}
	}
	// Geçerli şablon temiz geçmeli.
	if sorun := gazeteSablonDogrula(temel(gzBlok("a", "metin", 2, 2, 10, 10))); len(sorun) > 0 {
		t.Errorf("geçerli şablon reddedildi: %v", sorun)
	}
}

// TestGazeteStilBeyazListe stil sözlüğünün CSS'e sızmadığını doğrular.
// `stil` serbest bir map; doğrudan CSS'e geçirilse enjeksiyon yüzeyi olurdu.
func TestGazeteStilBeyazListe(t *testing.T) {
	kotu := map[string]any{
		"zemin":     `lacivert"><script>alert(1)</script>`,
		"hiza":      "orta; background:url(x)",
		"gölge":     "0 0 10px red",
		"olcek":     99.0,
		"sutun":     7,
		"metinRenk": "acik", // tek geçerli olan
	}
	sinif := gazeteStilSinif(kotu)
	if strings.ContainsAny(sinif, `<>"();:`) {
		t.Errorf("stil sınıfına ham değer sızdı: %q", sinif)
	}
	if !strings.Contains(sinif, "gb-m-acik") {
		t.Errorf("geçerli jeton atıldı: %q", sinif)
	}
	for _, yasak := range []string{"script", "gölge", "gb-s-7", "url"} {
		if strings.Contains(sinif, yasak) {
			t.Errorf("beyaz liste dışı %q sınıfa girdi: %q", yasak, sinif)
		}
	}
	// Aralık dışı ölçek satır içi değişkene yazılmamalı.
	if d := gazeteStilDeger(kotu); d != "" {
		t.Errorf("aralık dışı ölçek yazıldı: %q", d)
	}
	if d := gazeteStilDeger(map[string]any{"olcek": 1.15}); !strings.Contains(d, "1.15") {
		t.Errorf("geçerli ölçek yazılmadı: %q", d)
	}
	// Üretilen HTML'de de sızıntı olmamalı.
	h := gazeteBlockHTML(gazeteBlock{ID: "x", Tip: "metin", W: 10, H: 10, Stil: kotu})
	if strings.Contains(h, "<script>") || strings.Contains(h, "url(") {
		t.Errorf("baskı HTML'ine stil sızdı: %s", h)
	}
}

// TestGazeteSablonJSONTurlari her şablonun JSON'a çevrilip geri okunabildiğini
// doğrular; veritabanına JSONB olarak böyle yazılıyor.
func TestGazeteSablonJSONTurlari(t *testing.T) {
	for _, s := range gazeteTohumSablonlari() {
		raw, err := json.Marshal(s.Doc)
		if err != nil {
			t.Fatalf("%s: marshal: %v", s.Ad, err)
		}
		geri, err := gazeteSablonCoz(raw)
		if err != nil {
			t.Fatalf("%s: tur sonrası çözülemedi: %v", s.Ad, err)
		}
		if len(geri.Sayfalar) != len(s.Doc.Sayfalar) {
			t.Errorf("%s: sayfa sayısı değişti %d → %d",
				s.Ad, len(s.Doc.Sayfalar), len(geri.Sayfalar))
		}
		if sorun := gazeteSablonDogrula(geri); len(sorun) > 0 {
			t.Errorf("%s: tur sonrası geçersiz: %v", s.Ad, sorun)
		}
	}
}
