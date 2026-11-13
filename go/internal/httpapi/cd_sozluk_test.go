package httpapi

import (
	"encoding/json"
	"html"
	"strings"
	"testing"
)

// Sözlük gövdesi SUNUCUDA üretiliyor; 55 terimin de sayfaya girdiğini
// ve çıpaların tekil olduğunu doğrular.
func TestSozlukGovdeTumTerimleriBasar(t *testing.T) {
	h := sozlukGovde()
	if len(sozlukTerimleri) < 50 {
		t.Fatalf("terim verisi eksik: %d", len(sozlukTerimleri))
	}
	for _, terim := range sozlukTerimleri {
		if !strings.Contains(h, `id="`+terim.Slug+`"`) {
			t.Errorf("%q terimi için çıpa basılmadı", terim.Terim)
		}
		// KAÇIRILMIŞ hâliyle aranır: açıklamalarda tırnak ve kesme
		// işareti var ("All-Time High" gibi) ve bunlar &#34; olarak
		// basılıyor — doğru davranış. Ham metni aramak kodu değil
		// testi yanıltırdı.
		if terim.Aciklama != "" && !strings.Contains(h, html.EscapeString(terim.Aciklama)) {
			t.Errorf("%q açıklaması basılmadı", terim.Terim)
		}
	}
}

// ÇIPALAR TEKİL OLMALI: iki terim aynı id'yi alırsa /sozluk#slug
// yanlış yere götürür ve DefinedTerm url'leri çakışır.
func TestSozlukCipalariTekil(t *testing.T) {
	gorulen := map[string]string{}
	for _, terim := range sozlukTerimleri {
		if terim.Slug == "" {
			t.Errorf("%q teriminin slug'ı boş", terim.Terim)
			continue
		}
		if onceki, varMi := gorulen[terim.Slug]; varMi {
			t.Errorf("çıpa çakışması %q: %q ve %q", terim.Slug, onceki, terim.Terim)
		}
		gorulen[terim.Slug] = terim.Terim
	}
}

// Sıralama ÖNCE SEVİYE sonra alfabe: sözlüğü baştan okuyan biri
// "Blockchain" ile başlamalı, "Atomic Swap" ile değil.
func TestSozlukSiralamaSeviyeOnce(t *testing.T) {
	s := sozlukSirali()
	if len(s) != len(sozlukTerimleri) {
		t.Fatalf("sıralama terim düşürmüş: %d -> %d", len(sozlukTerimleri), len(s))
	}
	sonSeviye := -1
	for _, terim := range s {
		sv, varMi := sozlukSeviyeSirasi[terim.Seviye]
		if !varMi {
			t.Errorf("%q bilinmeyen seviye: %q", terim.Terim, terim.Seviye)
			continue
		}
		if sv < sonSeviye {
			t.Errorf("seviye sırası bozuk: %q (%s) geriye gitti", terim.Terim, terim.Seviye)
		}
		sonSeviye = sv
	}
}

// Yapısal veri geçerli JSON olmalı ve her terimi taşımalı: Google
// sözlüğü DefinedTermSet ile anlıyor, tek tek sayfa açmadan terim
// düzeyinde görünürlük sağlayan yol bu.
func TestSozlukJSONLD(t *testing.T) {
	ham := sozlukJSONLD("https://bitcosar.com")
	i, j := strings.Index(ham, ">"), strings.LastIndex(ham, "<")
	if i < 0 || j <= i {
		t.Fatalf("script sarmalı bozuk: %q", ham[:60])
	}
	var d map[string]any
	if err := json.Unmarshal([]byte(ham[i+1:j]), &d); err != nil {
		t.Fatalf("JSON-LD geçersiz: %v", err)
	}
	if d["@type"] != "DefinedTermSet" {
		t.Errorf("@type = %v", d["@type"])
	}
	liste, _ := d["hasDefinedTerm"].([]any)
	if len(liste) != len(sozlukTerimleri) {
		t.Fatalf("terim sayısı = %d, %d beklenir", len(liste), len(sozlukTerimleri))
	}
	ilk, _ := liste[0].(map[string]any)
	if ilk["@type"] != "DefinedTerm" {
		t.Errorf("terim @type = %v", ilk["@type"])
	}
	if u, _ := ilk["url"].(string); !strings.Contains(u, "/sozluk#") {
		t.Errorf("terim url'si çıpa taşımıyor: %v", ilk["url"])
	}
}

// HTML KAÇIŞI: terim ve açıklamalar şablona ham girmemeli. Veri şu an
// güvenli ama üretilmiş dosya yarın elle genişletilebilir.
func TestSozlukHTMLKacisi(t *testing.T) {
	h := sozlukGovde()
	if strings.Contains(h, "<script") {
		t.Error("gövdede script etiketi var")
	}
	// Açılmamış tırnak öznitelikleri bozardı.
	if strings.Count(h, `id="`) != len(sozlukTerimleri) {
		t.Errorf("id öznitelik sayısı %d, %d beklenir",
			strings.Count(h, `id="`), len(sozlukTerimleri))
	}
}
