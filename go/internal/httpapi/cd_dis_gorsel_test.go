package httpapi

import (
	"encoding/json"
	"strings"
	"testing"
)

// Gövde görseli medyaya alınınca düğüm `src` yerine `mediaId`
// taşımalı: çizici ancak o zaman görseli KENDİ sunucumuzdan basar
// (bkz. renderTipTapNode, "image" dalı).
func TestTipTapGorselDegistir(t *testing.T) {
	belge := `{"type":"doc","content":[
		{"type":"paragraph","content":[{"type":"text","text":"Metin"}]},
		{"type":"image","attrs":{"src":"https://ghost.bitcosar.com/a.png","alt":"Grafik"}},
		{"type":"image","attrs":{"src":"https://baska.com/b.png"}}
	]}`

	yeni, degisti := tipTapGorselDegistir([]byte(belge), "https://ghost.bitcosar.com/a.png", 42)
	if !degisti {
		t.Fatal("eşleşen görsel değiştirilmedi")
	}

	var d map[string]any
	if err := json.Unmarshal(yeni, &d); err != nil {
		t.Fatalf("çıktı geçerli JSON değil: %v", err)
	}
	icerik := d["content"].([]any)

	deg := icerik[1].(map[string]any)["attrs"].(map[string]any)
	if _, varMi := deg["src"]; varMi {
		t.Error("src silinmedi; çizici hâlâ dış adrese bakar")
	}
	if deg["mediaId"] != float64(42) {
		t.Errorf("mediaId = %v, 42 beklenir", deg["mediaId"])
	}
	if deg["alt"] != "Grafik" {
		t.Errorf("alt metni kaybolmuş: %v", deg["alt"])
	}

	// EŞLEŞMEYEN görsele dokunulmamalı.
	digeri := icerik[2].(map[string]any)["attrs"].(map[string]any)
	if digeri["src"] != "https://baska.com/b.png" {
		t.Errorf("eşleşmeyen görsel değişmiş: %v", digeri)
	}
	if _, varMi := digeri["mediaId"]; varMi {
		t.Error("eşleşmeyen görsele mediaId yazılmış")
	}
}

// Aynı adres birden çok kez geçebilir (aynı grafik iki bölümde);
// hepsi değişmeli, yoksa kalanlar ölü adrese bakmaya devam eder.
func TestTipTapGorselDegistirTumKopyalar(t *testing.T) {
	belge := `{"type":"doc","content":[
		{"type":"image","attrs":{"src":"https://x.com/a.png"}},
		{"type":"blockquote","content":[{"type":"image","attrs":{"src":"https://x.com/a.png"}}]}
	]}`
	yeni, degisti := tipTapGorselDegistir([]byte(belge), "https://x.com/a.png", 7)
	if !degisti {
		t.Fatal("değişiklik yok")
	}
	if n := strings.Count(string(yeni), "mediaId"); n != 2 {
		t.Errorf("mediaId sayısı = %d, 2 beklenir (iç içe düğüm atlanmış olabilir)", n)
	}
	if strings.Contains(string(yeni), "https://x.com/a.png") {
		t.Error("ölü adres belgede kalmış")
	}
}

// Eşleşme yoksa belge AYNEN dönmeli: gereksiz yazma, sürüm artışı ve
// gürültü üretmesin.
func TestTipTapGorselDegistirEslesmezse(t *testing.T) {
	belge := []byte(`{"type":"doc","content":[{"type":"image","attrs":{"src":"https://a.com/1.png"}}]}`)
	yeni, degisti := tipTapGorselDegistir(belge, "https://b.com/2.png", 5)
	if degisti {
		t.Error("eşleşme yokken değişiklik bildirildi")
	}
	if string(yeni) != string(belge) {
		t.Error("belge gereksiz yere yeniden yazılmış")
	}
}

// Bozuk JSON patlatmamalı: gövde her zaman geçerli olmayabilir.
func TestTipTapGorselDegistirBozukJSON(t *testing.T) {
	ham := []byte(`{"type":"doc",`)
	yeni, degisti := tipTapGorselDegistir(ham, "https://a.com/1.png", 1)
	if degisti || string(yeni) != string(ham) {
		t.Error("bozuk JSON'da değişiklik yapılmış")
	}
}

// Dosya adı uzantısı İÇERİKTEN gelmeli, adresten değil: Ghost ve
// CDN'ler uzantısız veya yanlış uzantılı adres verebiliyor.
func TestDisGorselDosyaAdi(t *testing.T) {
	png := []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a, 0, 0, 0, 0}
	if ad := disGorselDosyaAdi("/content/images/grafik.jpg", png); !strings.HasSuffix(ad, ".png") {
		t.Errorf("uzantı içerikten alınmadı: %q", ad)
	}
	// Sondaki eğik çizgi dosya adını DÜŞÜRMEZ: Base("/a/images/") ==
	// "images" ve onu kullanmak doğru. Yedek ad yalnız gerçekten boş
	// yolda devreye girer (ör. "https://cdn.example.com/").
	if ad := disGorselDosyaAdi("/content/images/", png); ad != "images.png" {
		t.Errorf("son eğik çizgili yolda ad = %q, \"images.png\" beklenir", ad)
	}
	for _, bos := range []string{"", "/"} {
		if ad := disGorselDosyaAdi(bos, png); !strings.HasPrefix(ad, "dis-gorsel") {
			t.Errorf("boş yol (%q) için yedek ad üretilmedi: %q", bos, ad)
		}
	}
	// Türkçe ve tehlikeli karakterler dosya adına girmemeli.
	ad := disGorselDosyaAdi("/a/çöğüşi ürün%00.png", png)
	if strings.ContainsAny(ad, " %\x00/") {
		t.Errorf("dosya adı temizlenmemiş: %q", ad)
	}
}
