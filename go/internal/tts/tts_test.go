package tts

import (
	"strings"
	"testing"
)

func TestSplitTextLimitAndContent(t *testing.T) {
	text := strings.Repeat("Bu bir deneme cümlesidir. ", 600)
	parts := SplitText(text, 3500)
	if len(parts) < 2 {
		t.Fatalf("uzun metin tek parçaya sığmamalı, parça=%d", len(parts))
	}
	for i, p := range parts {
		if n := len([]rune(p)); n > 3500 {
			t.Fatalf("parça %d çok uzun: %d rune", i, n)
		}
		if strings.TrimSpace(p) == "" {
			t.Fatalf("parça %d boş", i)
		}
	}
	// Cümle sayısı korunmalı (kelime kaybı olmamalı).
	if got, want := strings.Count(strings.Join(parts, " "), "deneme"), 600; got != want {
		t.Fatalf("kelime kaybı: got=%d want=%d", got, want)
	}
}

func TestSplitTextShort(t *testing.T) {
	parts := SplitText("Kısa bir haber metni.", 3500)
	if len(parts) != 1 || parts[0] != "Kısa bir haber metni." {
		t.Fatalf("kısa metin tek parça olmalı: %#v", parts)
	}
	if SplitText("   ", 3500) != nil {
		t.Fatal("boş metin nil dönmeli")
	}
}

func TestHTMLToText(t *testing.T) {
	h := `<p>Birinci <strong>paragraf</strong>.</p><script>alert(1)</script><p>İkinci &amp; son.</p>`
	got := HTMLToText(h)
	if strings.Contains(got, "alert") {
		t.Fatal("script içeriği metne girmemeli")
	}
	if !strings.Contains(got, "Birinci paragraf.") || !strings.Contains(got, "İkinci & son.") {
		t.Fatalf("beklenmeyen metin: %q", got)
	}
}

func TestEstimateSeconds(t *testing.T) {
	if got := EstimateSeconds(""); got != 0 {
		t.Fatalf("boş metin 0 olmalı, got=%v", got)
	}
	if got := EstimateSeconds(strings.Repeat("a", 140)); got < 9.9 || got > 10.1 {
		t.Fatalf("140 karakter ~10 sn olmalı, got=%v", got)
	}
}
