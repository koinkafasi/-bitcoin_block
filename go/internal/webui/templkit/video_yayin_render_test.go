package templkit

import (
	"context"
	"strings"
	"testing"
)

// TestVideoYayinPanelRender: panelin boş ve dolu hâli hatasız çizilmeli.
//
// NEDEN: panel veritabanından gelen listeye göre tablo/sekmeler basıyor;
// nil dilim + boş ayar kombinasyonunda templ çalışma zamanı hatası vermemeli
// (panel açılışında 500 görmek, ayar sayfasını tamamen erişilmez yapar).
func TestVideoYayinPanelRender(t *testing.T) {
	// Boş hâl.
	var b strings.Builder
	if err := VideoYayinPanel(VideoYayinView{}).Render(context.Background(), &b); err != nil {
		t.Fatalf("boş panel çizilemedi: %v", err)
	}
	if !strings.Contains(b.String(), "Kanal Kimliği") {
		t.Error("ayar formu basılmadı")
	}

	// Dolu hâl.
	v := VideoYayinView{
		FlashKind: "info", FlashText: "Ayarlar kaydedildi.",
		KanalID: "UCPtM806ijpKatyIHH5w4kmg", Handle: "bitcosar", Dakika: "15",
		Aktif: true, VideoGoster: true, ShortGoster: true, CanliGoster: true,
		VideoAdet: 2, ShortAdet: 3, YayinAdet: 1, IcerikToplam: 6,
		Sekmeler: []VideoYayinSekme{{Kod: "", Ad: "Tümü", Adet: 6, Aktif: true}},
		Satirlar: []VideoYayinSatir{{
			VideoID: "KfLVTp9jlyU", Tur: "short", TurAd: "Shorts",
			Baslik: "Örnek başlık", Gorsel: "https://i.ytimg.com/vi/KfLVTp9jlyU/oar2.jpg",
			Sure: "1:02", Zaman: "2 gün önce", Canli: true, OneCikan: true,
		}},
	}
	b.Reset()
	if err := VideoYayinPanel(v).Render(context.Background(), &b); err != nil {
		t.Fatalf("dolu panel çizilemedi: %v", err)
	}
	out := b.String()
	for _, beklenen := range []string{
		"Ayarlar kaydedildi.", "Örnek başlık", "Shorts", "/video/KfLVTp9jlyU",
		"/admin/video-yayin/cek", "/admin/video-yayin/KfLVTp9jlyU/one_cikan",
	} {
		if !strings.Contains(out, beklenen) {
			t.Errorf("panelde eksik: %q", beklenen)
		}
	}
}
