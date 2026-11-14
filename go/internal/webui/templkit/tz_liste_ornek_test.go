package templkit

import (
	"os"
	"testing"
)

// TestYazListeOrnekHTML trafik güvenliği LİSTESİNİN örnek HTML'ini üretir
// (tarayıcı ölçümü için). Yalnız TZ_LISTE_HTML verildiğinde yazar.
func TestYazListeOrnekHTML(t *testing.T) {
	yol := os.Getenv("TZ_LISTE_HTML")
	if yol == "" {
		t.Skip("TZ_LISTE_HTML yok")
	}
	v := TrafikGuvenligiView{
		Gun: 30, Aktif: true, HostingEngel: true, BotEngel: true, WebRTC: true,
		Toplam: 1284, Gercek: 806, Supheli: 211, Yuksek: 39, Bot: 208, Arama: 20,
		VPN: 48, Hosting: 193, Tor: 6, GercekYuzde: 63,
		EsikYuksek: 60, EsikBot: 85,
		ASN: []GuvenlikASNSatir{
			{Etiket: "HETZNER-AS", Tur: "hosting", ASN: 24940, Adet: 193, Pay: 15},
			{Etiket: "SUPERONLINE", Tur: "mobil", ASN: 34984, Adet: 402, Pay: 31},
			{Etiket: "M247", Tur: "vpn", ASN: 9009, Adet: 48, Pay: 4},
			{Etiket: "TTNET", Tur: "konut", ASN: 9121, Adet: 318, Pay: 25},
		},
		Sinyaller: []GuvenlikSinyalSatir{
			{Kod: "asn_hosting", Ad: "Veri merkezi ağı", Adet: 193, Pay: 100},
			{Kod: "ua_bot", Ad: "Otomasyon aracı (UA)", Adet: 156, Pay: 81},
			{Kod: "hiz_asimi", Ad: "İstek hızı aşımı", Adet: 74, Pay: 38},
			{Kod: "asn_vpn", Ad: "Ticari VPN ağı", Adet: 48, Pay: 25},
		},
		Satirlar: []GuvenlikRiskSatir{
			{Anahtar: "ip:5.9.1.17", Sinif: "bot", SinifAd: "Bot", Puan: 92,
				Niyet: "Zafiyet taraması", NiyetSeviye: "tehdit",
				Saglayici: "HETZNER-AS", Tur: "hosting", TurAd: "veri merkezi",
				Ulke: "DE", IP: "5.9.1.***", IstekSayisi: 1841,
				SonGorulme: "14.11 03:12",
				Sinyaller:  []string{"Veri merkezi ağı", "Otomasyon aracı (UA)"}},
			{Anahtar: "ip:45.83.89.4", Sinif: "yuksek", SinifAd: "Yüksek", Puan: 66,
				Niyet: "Besleme / harita toplama", NiyetSeviye: "dikkat",
				Saglayici: "M247", Tur: "vpn", TurAd: "VPN",
				Ulke: "RO", IP: "45.83.89.***", IstekSayisi: 212,
				SonGorulme: "14.11 02:55",
				Sinyaller:  []string{"Ticari VPN ağı"}},
			{Anahtar: "ip:88.230.11.9", Sinif: "temiz", SinifAd: "Gerçek", Puan: 12,
				Niyet: "Olağan gezinme", NiyetSeviye: "bilgi",
				Saglayici: "TTNET", Tur: "konut", TurAd: "konut",
				Ulke: "TR", IP: "88.230.11.***", IstekSayisi: 14,
				SonGorulme: "14.11 03:09", Sinyaller: []string{"Dil uyuşmazlığı"}},
			// Olay kaydı olmayan satır: niyet sütunu "—" kalmalı.
			{Anahtar: "ip:203.0.113.5", Sinif: "supheli", SinifAd: "Şüpheli", Puan: 35,
				Saglayici: "", Tur: "", Ulke: "FR", IP: "203.0.113.***",
				IstekSayisi: 3, SonGorulme: "13.11 22:40"},
		},
	}
	buf := &yazici{}
	if err := TrafikGuvenligiPanel(v).Render(t.Context(), buf); err != nil {
		t.Fatalf("çizilemedi: %v", err)
	}
	if err := os.WriteFile(yol, buf.b, 0o644); err != nil {
		t.Fatalf("yazılamadı: %v", err)
	}
	t.Logf("%d bayt: %s", len(buf.b), yol)
}
