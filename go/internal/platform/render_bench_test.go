package platform

import (
	"encoding/json"
	"os"
	"testing"
)

// BenchmarkRenderDesign manşet raster'ını ölçer.
// Gerçek tasarım belgesi yoksa küçük bir belge kurulur.
func BenchmarkRenderDesign(b *testing.B) {
	raw := os.Getenv("DESIGN_JSON")
	var doc []byte
	if raw != "" {
		b, err := os.ReadFile(raw)
		if err != nil {
			b2 := err
			_ = b2
			panic(err)
		}
		doc = b
	} else {
		doc = []byte(`{"version":1,"canvas":{"w":1200,"h":675},
		"background":{"color":"#101820"},
		"badge":{"active":true,"text":"SON DAKİKA","color":"#e11d48","textColor":"#fff","fontSize":22,"x":60,"y":60,"radius":6,"padding":14},
		"layers":[
		 {"id":"t","role":"title","text":"Bitcoin 100 bin doları aştı: kurumsal talep rekor kırdı","x":60,"y":380,"fontSize":64,"weight":700,"color":"#ffffff","align":"left","lineHeight":1.15,"visible":true},
		 {"id":"s","role":"spot","text":"Analistler yıl sonu hedefini yukarı çekti.","x":60,"y":560,"fontSize":26,"weight":400,"color":"#c9d1d9","align":"left","lineHeight":1.3,"visible":true}
		]}`)
	}
	// Geçerlilik: NormalizeDesign paniklemesin.
	_ = NormalizeDesign(doc)
	var js any
	if err := json.Unmarshal(doc, &js); err != nil {
		b.Fatalf("design json: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, err := RenderDesign(doc, nil)
		if err != nil {
			b.Fatal(err)
		}
		if len(out) == 0 {
			b.Fatal("boş çıktı")
		}
	}
}
