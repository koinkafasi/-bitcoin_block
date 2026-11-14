package templkit

import (
	"strings"
	"testing"

	"github.com/a-h/templ"
)

// ZİYARETÇİ PROFİLİ ŞABLON TESTİ.
//
// NEDEN: templ derleme hatasını yakalar ama VERİ hatasını yakalamaz.
// Bu sayfa 14 alanlı bir tanım listesi ve üç tablo çiziyor; bir alanı
// yanlış yere koymak (ör. tam IP yerine maskeli IP) sessizce yanlış bilgi
// gösterirdi — panelin tek işi doğru bilgi göstermek.
func TestTrafikZiyaretciPanelBasilir(t *testing.T) {
	v := TrafikZiyaretciView{
		Anahtar:       "ip:5.9.1.1",
		Var:           true,
		IP:            "5.9.1.1",
		MaskeliIP:     "5.9.1.***",
		IPAgg:         "5.9.0.0/16",
		Ulke:          "Almanya",
		UlkeKod:       "DE",
		ASN:           24940,
		Saglayici:     "HETZNER-AS",
		ASNAd:         "HETZNER-AS - Hetzner Online GmbH, DE",
		Tur:           "hosting",
		TurAd:         "veri merkezi",
		Sinif:         "bot",
		SinifAd:       "Bot",
		Puan:          92,
		IstekSayisi:   1841,
		IPDegisim:     3,
		UlkeDegisim:   1,
		IlkGorulme:    "13.11.2026 21:04",
		SonGorulme:    "14.11.2026 03:12",
		UV:            "python-requests",
		UserAgent:     "python-requests/2.31.0",
		WebRTCIP:      "",
		Hosting:       true,
		Bot:           true,
		Sinyaller:     []string{"Veri merkezi ağı", "Otomasyon aracı (UA)"},
		NiyetBaslik:   "Zafiyet taraması",
		NiyetAciklama: "39 istek bilinen açık tarama yoluna gitti.",
		NiyetSeviye:   "tehdit",
		OlaySayisi:    212,
		Engellenen:    198,
		Turler: []ZiyaretciTurSatir{
			{Kod: "zafiyet", Ad: "zafiyet taraması", Adet: 39, Pay: 18},
			{Kod: "icerik", Ad: "içerik okuma", Adet: 170, Pay: 80},
		},
		Yollar: []ZiyaretciYolSatir{
			{Yol: "/wp-login.php", Adet: 12, Durum: "403"},
		},
		Akrabalar: []ZiyaretciAkrabaSatir{
			{Anahtar: "ip:5.9.1.2", Adet: 44, Son: "14.11.2026 03:10"},
		},
		Olaylar: []ZiyaretciOlaySatir{
			{Zaman: "14.11 03:12:04", Yontem: "GET", Yol: "/wp-login.php",
				Sorgu: "token=***", Durum: 403, Tur: "zafiyet",
				TurAd: "zafiyet taraması", Kaynak: "yok", KaynakAd: "doğrudan",
				Engellendi: true},
		},
	}

	var sb strings.Builder
	if err := TrafikZiyaretciPanel(v).Render(t.Context(), &sb); err != nil {
		t.Fatalf("şablon çizilemedi: %v", err)
	}
	html := sb.String()

	// Tam IP görünmeli — sayfanın varlık sebebi bu (liste maskeli kalır).
	if !strings.Contains(html, "5.9.1.1") {
		t.Error("tam IP basılmadı")
	}
	if !strings.Contains(html, "Zafiyet taraması") {
		t.Error("niyet başlığı basılmadı")
	}
	// Niyet seviyesi CSS sınıfına yansımalı (renk kodlaması).
	if !strings.Contains(html, "tz-niyet--tehdit") {
		t.Error("niyet seviyesi sınıfı yok")
	}
	if !strings.Contains(html, "HETZNER-AS") {
		t.Error("sağlayıcı basılmadı")
	}
	if !strings.Contains(html, "/wp-login.php") {
		t.Error("yol listesi basılmadı")
	}
	// Pivot: aynı IP'den gelen diğer kimlik tıklanabilir olmalı.
	if !strings.Contains(html, "/admin/trafik-guvenligi/ziyaretci/ip:5.9.1.2") {
		t.Error("akraba bağlantısı yok")
	}
	// Maskeleme notu: yöneticiye jetonun görünmediği söylenmeli.
	if !strings.Contains(html, "maskelenmiştir") {
		t.Error("sorgu maskeleme notu yok")
	}
}

// Boş durum: risk satırı yokken de sayfa çizilmeli (olaylar var, özet yok).
func TestTrafikZiyaretciPanelBosDurum(t *testing.T) {
	v := TrafikZiyaretciView{Anahtar: "ip:1.2.3.4", IP: "1.2.3.4", NiyetBaslik: "Kayıt yok"}
	var sb strings.Builder
	if err := TrafikZiyaretciPanel(v).Render(t.Context(), &sb); err != nil {
		t.Fatalf("boş durum çizilemedi: %v", err)
	}
	if !strings.Contains(sb.String(), "Risk satırı henüz yazılmadı") {
		t.Error("risk satırı yok uyarısı basılmadı")
	}
}

// templ.Component arayüzü korunmalı (imza değişirse derleme kırılır,
// bu satır da niyeti belgeler).
var _ templ.Component = TrafikZiyaretciPanel(TrafikZiyaretciView{})
