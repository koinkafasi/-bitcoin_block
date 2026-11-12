package ai

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMockStructuredTitleUsesValidJSON(t *testing.T) {
	p := &mockProvider{}
	input := `Türkçe "alıntı" \ başlık şğüöçİı`
	out := p.structured("JSON:title " + input)

	var list []string
	if err := json.Unmarshal([]byte(out), &list); err != nil {
		t.Fatalf("geçersiz JSON: %v: %s", err, out)
	}
	if len(list) == 0 {
		t.Fatalf("boş başlık listesi: %s", out)
	}
	want := input + " — Mock başlık A"
	if list[0] != want {
		t.Fatalf("başlık = %q; want %q", list[0], want)
	}
}

// TestMockStructuredPlainFields, spot/SEO alanlarının düz metin (tırnaksız)
// döndüğünü ve Türkçe karakterleri koruduğunu doğrular.
func TestMockStructuredPlainFields(t *testing.T) {
	p := &mockProvider{}
	input := `Türkçe "alıntı" başlık`
	for _, kind := range []string{"spot", "seo_title", "seo_description"} {
		out := p.structured("JSON:" + kind + " " + input)
		if strings.HasPrefix(out, `"`) || strings.HasSuffix(out, `"`) {
			t.Errorf("%s düz metin olmalı, tırnaklı: %s", kind, out)
		}
		if !strings.Contains(out, input) {
			t.Errorf("%s girdiyi korumalı: %s", kind, out)
		}
	}
}
