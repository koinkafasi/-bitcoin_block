package templkit

import (
	"context"
	"strings"
	"testing"
)

// TestCDVPlayerKontrolleri: kendi oynatıcının kontrol çubuğu eksiksiz
// çizilmeli. NEDEN: oynatıcı JS'e bağlı; templ tarafında bir düğüm
// (ilerleme çubuğu, ses, hız, tam ekran) düşerse JS sessizce çalışmaz hâle
// gelir ve kullanıcı "play'e basınca bir şey olmuyor" der.
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
	// Kapak ve büyük düğme AYNI ANDA basılır ve iframe'in üstünde durur:
	// oynatma engellenirse YouTube'un poster'ı değil bizim düğmemiz görünür
	// (bkz. cd-player.js başlığı — "ortada play simgesi kalıyor" hatası).
	if !strings.Contains(out, "data-cd-kapak") || !strings.Contains(out, "data-cd-buyuk") {
		t.Error("kapak ve büyük oynat düğmesi birlikte basılmalı")
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
