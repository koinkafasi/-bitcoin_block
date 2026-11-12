package templkit

import (
	"context"
	"strings"
	"testing"
)

func TestCDVPlayerKontrolleri(t *testing.T) {
	v := CDVideoHero{
		VideoID: "8oVvSLjkMv8", Baslik: "Örnek video",
		Image: "https://i.ytimg.com/vi/8oVvSLjkMv8/hqdefault.jpg", Sure: "22:31",
		SonrakiID: "MctQ1WmwSaA", SonrakiBaslik: "Sıradaki örnek",
	}
	var b strings.Builder
	if err := cdVPlayer(v, false).Render(context.Background(), &b); err != nil {
		t.Fatalf("oynatıcı çizilemedi: %v", err)
	}
	out := b.String()
	for _, gerekli := range []string{
		`data-cd-player`, `data-video="8oVvSLjkMv8"`,
		`data-sonraki="MctQ1WmwSaA"`,
		`data-cd-buyuk`, `data-cd-kapak`, `data-cd-kontrol`,
		`data-cd-cizgi`, `data-cd-dolu`, `data-cd-tampon`,
		`data-cd-oynat`, `data-cd-ses`, `data-cd-ses-aralik`,
		`data-cd-hiz`, `data-cd-hiz-sec="1.5"`, `data-cd-tam`,
		`role="slider"`, `aria-label="Video ilerlemesi"`,
		`data-cd-katman`,
		`data-cd-duyuru`, `aria-live="polite"`,
		"www.youtube.com/watch?v=8oVvSLjkMv8",
	} {
		if !strings.Contains(out, gerekli) {
			t.Errorf("oynatıcıda eksik: %s", gerekli)
		}
	}
	// Oynatıcı iframe'i BAŞTAN basmamalı (tıklamayla kurulur).
	if strings.Contains(out, "<iframe") {
		t.Error("iframe baştan basılmamalı — kapak + oynat düğmesi beklenir")
	}
	if !strings.Contains(out, "data-cd-kapak") || !strings.Contains(out, "data-cd-buyuk") {
		t.Error("kapak ve büyük oynat düğmesi birlikte basılmalı")
	}
	if !strings.Contains(out, "data-cd-katman") {
		t.Error("etkileşim katmanı (data-cd-katman) basılmalı")
	}
}

// TestCDVPlayerDikey: short videosunda dikey sınıf uygulanır (9:16).
func TestCDVPlayerDikey(t *testing.T) {
	v := CDVideoHero{VideoID: "KfLVTp9jlyU", Baslik: "Short"}
	var b strings.Builder
	if err := cdVPlayer(v, true).Render(context.Background(), &b); err != nil {
		t.Fatalf("dikey oynatıcı çizilemedi: %v", err)
	}
	if !strings.Contains(b.String(), "cd-vplayer--dikey") {
		t.Error("dikey oynatıcı sınıfı basılmadı")
	}
}
