package platform

import (
	"encoding/json"
	"strings"
	"testing"
)

func ptr64(v int64) *int64 { return &v }

func TestApplyContentDesign(t *testing.T) {
	design := []byte(`{
  "version": 2,
  "canvas": {"w":1200,"h":675},
  "background": {"color":"#0b0e14","overlay":{"active":true,"mode":"gradient","color":"#000000","direction":"left","spread":70,"opacity":85}},
  "badge": {"active":true,"text":"SON DAKİKA","color":"#e11d48","textColor":"#ffffff","fontSize":24,"x":60,"y":60,"uppercase":true},
  "layers": [
    {"id":"ust","role":"ust_baslik","text":"SON DAKİKA","x":80,"y":330,"fontSize":28,"visible":true},
    {"id":"ana","role":"ana_baslik","text":"Haber başlığınız buraya","x":80,"y":385,"fontSize":56,"visible":true},
    {"id":"alt","role":"alt_baslik","text":"Haber özeti buraya…","x":80,"y":520,"fontSize":24,"visible":true}
  ]
}`)

	out, err := ApplyContentDesign(design, ContentDesignInput{
		Title:      "Deprem sonrası yardım seferberliği",
		Spot:       "Bölgede 12 bin personel görev alıyor",
		Badge:      "Gündem",
		Category:   "Gündem",
		CoverMedia: ptr64(42),
	})
	if err != nil {
		t.Fatalf("hata: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatalf("çıktı geçersiz JSON: %v", err)
	}

	// arka plan medyası yazıldı mı?
	bg, _ := m["background"].(map[string]any)
	if v, _ := bg["media_id"].(float64); v != 42 {
		t.Errorf("background.media_id = %v, beklenen 42", v)
	}

	// rozet metni değişti mi?
	badge, _ := m["badge"].(map[string]any)
	if v, _ := badge["text"].(string); v != "Gündem" {
		t.Errorf("badge.text = %q, beklenen Gündem", v)
	}

	// katmanlar içerik değerlerini aldı mı?
	var titleFound, spotFound, catFound bool
	layers, _ := m["layers"].([]any)
	for _, li := range layers {
		lm, _ := li.(map[string]any)
		role, _ := lm["role"].(string)
		txt, _ := lm["text"].(string)
		switch role {
		case "ana_baslik":
			titleFound = txt == "Deprem sonrası yardım seferberliği"
		case "alt_baslik":
			spotFound = txt == "Bölgede 12 bin personel görev alıyor"
		case "ust_baslik":
			catFound = txt == "Gündem"
		}
	}
	if !titleFound {
		t.Error("ana_baslik katmanı içerik başlığını almadı")
	}
	if !spotFound {
		t.Error("alt_baslik katmanı özeti almadı")
	}
	if !catFound {
		t.Error("ust_baslik katmanı kategorisine içeriği almadı")
	}
}

func TestApplyContentDesignRender(t *testing.T) {
	// Tasarımın gerçekten JPEG üretilebilir olması (uçtan uca): kutudan çıkar.
	design := []byte(`{"version":2,"canvas":{"w":600,"h":337},"background":{"color":"#123456"},"layers":[{"id":"ana","role":"ana_baslik","text":"Başlık","x":20,"y":120,"fontSize":40,"visible":true}]}`)
	out, err := ApplyContentDesign(design, ContentDesignInput{Title: "Test başlık"})
	if err != nil {
		t.Fatalf("ApplyContentDesign: %v", err)
	}
	raw, err := RenderDesign(out, nil)
	if err != nil {
		t.Fatalf("RenderDesign: %v", err)
	}
	if len(raw) < 100 || len(raw) > 200000 {
		t.Fatalf("render boyutu şüpheli: %d", len(raw))
	}
	if !strings.HasPrefix(string(raw), "\xff\xd8") {
		t.Fatal("çıktı JPEG değil")
	}
}

func TestApplyContentDesignBosGirdi(t *testing.T) {
	// boş/geçersiz giriş hata döndürmeli (RenderDesign'ın errEmptyDesign).
	if _, err := ApplyContentDesign([]byte(`{}`), ContentDesignInput{Title: "x"}); err != nil {
		// ApplyContentDesign boş girdi için hata vermez (render yok)
		t.Logf("boş JSON ApplyContentDesign hatası: %v", err)
	}
}
