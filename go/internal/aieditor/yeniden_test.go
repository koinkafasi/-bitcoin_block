package aieditor

import (
	"strings"
	"testing"
)

func yenidenIyiMetin() []string {
	// Her paragraf farklı: tekrar cümle kapısı özdeş metni yakalar.
	return []string{
		"Türkiye Yurt Dışından Öğrenci Kabul Sınavı bu yıl iki ayrı oturumda yapılacak. " +
			"Ölçme, Seçme ve Yerleştirme Merkezi takvimi internet sitesinden duyurdu.",
		"Başvurular e-devlet üzerinden alınacak ve adaylar sınav merkezini kendileri seçecek. " +
			"Merkez, başvuru sırasında girilen bilgilerin doğruluğundan adayın sorumlu olduğunu bildirdi.",
		"Sınav sonuçları açıklandıktan sonra yerleştirme işlemleri puan üstünlüğüne göre yapılacak. " +
			"Adaylar sonuç belgelerini aynı sistemden görüntüleyebilecek.",
		"Kılavuzda yer alan kurallar ve takvim, sınavla ilgili tüm ayrıntıları kapsıyor. " +
			"Adayların başvuru öncesi kılavuzu dikkatle okumaları istendi.",
		"Ölçme kurulu, sınav güvenliği için merkezlerde ek önlem alınacağını açıkladı. " +
			"Kimlik kontrolü ve elektronik cihaz denetimi bu önlemler arasında yer alıyor.",
	}
}

func yenidenIyiSpot() string {
	return "Yurt dışından öğrenci kabul sınavının takvimi açıklandı; başvurular e-devlet üzerinden alınacak ve adaylar sınav merkezini kendileri seçecek."
}

func TestYenidenKapisiGecerliMetniKabulEder(t *testing.T) {
	y := yenidenYanit{
		Spot:        yenidenIyiSpot(),
		Paragraflar: yenidenIyiMetin(),
		SeoBaslik:   "Yurt dışından öğrenci kabul sınavı takvimi",
		SeoAciklama: "Yurt dışından öğrenci kabul sınavı başvuruları e-devlet üzerinden alınacak. Sınav takvimi ve merkez seçimiyle ilgili ayrıntılar açıklandı.",
	}
	uyarilar, engel := yenidenKapisi(y, strings.Join(y.Paragraflar, "\n\n"), "")
	if engel != "" {
		t.Fatalf("geçerli metin engellendi: %s (uyarılar %v)", engel, uyarilar)
	}
	if len(uyarilar) != 0 {
		t.Errorf("beklenmeyen uyarı: %v", uyarilar)
	}
}

func TestYenidenKapisiTekParagrafiEngeller(t *testing.T) {
	y := yenidenYanit{Paragraflar: []string{"Kısa bir haber metni."}}
	_, engel := yenidenKapisi(y, "", "")
	if !strings.Contains(engel, "paragraf") {
		t.Errorf("tek paragraf engellenmedi: %q", engel)
	}
}

func TestYenidenKapisiKisaGovdeyiEngeller(t *testing.T) {
	// Üç paragraf ama toplam uzunluk eşiğin altında: hâlâ ince içerik.
	y := yenidenYanit{Paragraflar: []string{"Birinci kısa paragraf.", "İkinci kısa paragraf.", "Üçüncü kısa paragraf."}}
	_, engel := yenidenKapisi(y, "", "")
	if !strings.Contains(engel, "karakter") {
		t.Errorf("kısa gövde engellenmedi: %q", engel)
	}
}

func TestYenidenKapisiKisaKaynaktaEsigiOlecekler(t *testing.T) {
	paragraflar := []string{
		"Hizmet üretim endeksi ağustosta yıllık yüzde 2,3 artış gösterdi. " +
			"TÜİK verisine göre aylık artış yüzde 0,2 oldu ve seri yukarı yönlü seyretti.",
		"Endeksin alt kalemlerinde ulaştırma ve depolama ile konaklama öne çıktı. " +
			"Bilgi ve iletişim hizmetlerinde de sınırlı bir artış kaydedildi.",
		"Veriler, hizmet sektörünün yılın üçüncü çeyreğinde büyümeye katkı verdiğini gösteriyor. " +
			"Ekonomistler önümüzdeki aylarda iç talebin belirleyici olacağını belirtiyor.",
	}
	y := yenidenYanit{Spot: "Hizmet üretim endeksi ağustosta yıllık yüzde 2,3 arttı; aylık artış yüzde 0,2 oldu ve alt kalemlerde ulaştırma öne çıktı.", Paragraflar: paragraflar}
	kaynak := "Hizmet üretim endeksi, ağustosta yıllık bazda yüzde 2,3, aylık bazda yüzde 0,2 artış gösterdi."
	uyarilar, engel := yenidenKapisi(y, kaynak, "")
	if engel != "" {
		t.Fatalf("kısa kaynakta eşik ölçeklenmedi, öneri engellendi: %s", engel)
	}
	if !uyariVar(uyarilar, "haber standardı") {
		t.Errorf("700 altı gövde uyarısı yok: %v", uyarilar)
	}
}

func TestYenidenKapisiKaynagaGoreKisaGovdeyiEngeller(t *testing.T) {
	// Kaynak 400 karakter, gövde 420: 1,6 katına ulaşmıyor.
	kaynak := strings.Repeat("Kaynak cümlesi burada yer alıyor. ", 12) // ~420
	y := yenidenYanit{Paragraflar: []string{
		strings.Repeat("Kısa paragraf bir. ", 8),
		strings.Repeat("Kısa paragraf iki. ", 8),
		strings.Repeat("Kısa paragraf üç. ", 8),
	}}
	_, engel := yenidenKapisi(y, kaynak, "")
	if !strings.Contains(engel, "karakter") {
		t.Errorf("kaynağa göre kısa gövde engellenmedi: %q", engel)
	}
}

// Kısa kaynakta 2 paragraf yeterli: 3+ paragraf istemek dolgu cümle üretir.
func TestYenidenKapisiKisaKaynaktaIkiParagrafYeter(t *testing.T) {
	y := yenidenYanit{
		Spot: "Hizmet üretim endeksi ağustosta yıllık yüzde 2,3 arttı; aylık artış yüzde 0,2 oldu.",
		Paragraflar: []string{
			"Hizmet üretim endeksi ağustosta yıllık bazda yüzde 2,3, aylık bazda yüzde 0,2 artış gösterdi. " +
				"TÜİK tarafından açıklanan veri, hizmet sektöründeki hareketin sınırlı da olsa sürdüğünü gösteriyor.",
			"Endeksin alt kalemlerinde ulaştırma ve depolama ile konaklama hizmetleri öne çıktı. " +
				"Bilgi ve iletişim hizmetlerinde de sınırlı bir artış kaydedildi.",
		},
	}
	kaynak := "Hizmet üretim endeksi, ağustosta yıllık bazda yüzde 2,3, aylık bazda yüzde 0,2 artış gösterdi."
	uyarilar, engel := yenidenKapisi(y, kaynak, "")
	if engel != "" {
		t.Fatalf("kısa kaynakta 2 paragraf engellendi: %s", engel)
	}
	if !uyariVar(uyarilar, "haber standardı") {
		t.Errorf("700 altı gövde uyarısı yok: %v", uyarilar)
	}
}

// Tek paragraf, kaynak kısa olsa da kabul edilmez: hedef paragraflı metin.
func TestYenidenKapisiKisaKaynaktaTekParagrafiEngeller(t *testing.T) {
	y := yenidenYanit{Paragraflar: []string{
		"Hizmet üretim endeksi ağustosta yıllık yüzde 2,3 artış gösterdi. " +
			"TÜİK verisine göre aylık artış yüzde 0,2 oldu ve seri yukarı yönlü seyretti. " +
			"Alt kalemlerde ulaştırma ve konaklama öne çıktı, bilgi ve iletişimde sınırlı artış görüldü.",
	}}
	_, engel := yenidenKapisi(y, "Hizmet üretim endeksi ağustosta yıllık yüzde 2,3 arttı.", "")
	if !strings.Contains(engel, "paragraf") {
		t.Errorf("tek paragraf engellenmedi: %q", engel)
	}
}

// Araştırmalı mod: istem GERÇEK KAYNAKLAR bloğunu ve daha uzun hedefi
// içermeli; kaynak/ajans adı yazma yasağı korunmalı.
func TestYenidenIstemArastirmaliMod(t *testing.T) {
	govde := "Başlık metni\n\n[S1] TÜİK — Hizmet üretim endeksi\nhttps://ornek.com\nVeri metni."
	sistemTR, kullaniciTR := yenidenIstem("tr", "Başlık", "Spot", govde, true)
	if !strings.Contains(sistemTR, "GERÇEK KAYNAKLAR") {
		t.Error("araştırmalı TR isteminde kaynak bloğu yok")
	}
	if !strings.Contains(sistemTR, "900-1600") {
		t.Error("araştırmalı TR isteminde uzunluk hedefi yok")
	}
	if !strings.Contains(kullaniciTR, "[S1]") {
		t.Error("kaynaklar kullanıcı istemine girmemiş")
	}
	sistemEN, _ := yenidenIstem("en", "Headline", "Summary", govde, true)
	if !strings.Contains(sistemEN, "REAL SOURCES") {
		t.Error("araştırmalı EN isteminde kaynak bloğu yok")
	}
}

// Kaynak metni geldiğinde eşik 700'e çıkar: artık uydurmadan uzun yazılabilir.
func TestYenidenKapisiKaynakliMetindeTamEsikIster(t *testing.T) {
	kaynak := strings.Repeat("Kaynak kurumu veriyi açıkladı ve ayrıntıları paylaştı. ", 40)
	y := yenidenYanit{Paragraflar: []string{
		strings.Repeat("Kısa paragraf bir. ", 8),
		strings.Repeat("Kısa paragraf iki. ", 8),
		strings.Repeat("Kısa paragraf üç. ", 8),
	}}
	_, engel := yenidenKapisi(y, kaynak, "")
	if !strings.Contains(engel, "700") {
		t.Errorf("kaynaklı metinde 700 eşiği uygulanmadı: %q", engel)
	}
}

// temizSorgu arama sorgusunu bozan karakterleri atmalı.
func TestTemizSorgu(t *testing.T) {
	got := temizSorgu(`  "Ukrayna:  Rusya, Poltava"  `)
	if strings.Contains(got, `"`) {
		t.Errorf("tırnak kalmış: %q", got)
	}
	if strings.Contains(got, "  ") {
		t.Errorf("çift boşluk kalmış: %q", got)
	}
	if got == "" {
		t.Error("boş sorgu")
	}
	long := temizSorgu(strings.Repeat("kelime ", 100))
	if len([]rune(long)) > 140 {
		t.Errorf("sorgu kırpılmadı: %d", len([]rune(long)))
	}
}

// Kaynak kısaltma sınırı: istem şişmesin diye her kaynak 2200 karaktere
// indirilir, ama kaynak tamamen kaybolmaz.
func TestKisaltSiniri(t *testing.T) {
	uzun := strings.Repeat("a", 5000)
	got := kisalt(uzun, 2200)
	if len([]rune(got)) > 2201 {
		t.Errorf("kısaltma sınırı aştı: %d", len([]rune(got)))
	}
	if len(got) < 100 {
		t.Errorf("kısaltma metni yok etti: %d", len(got))
	}
	if k := kisalt("kısa", 2200); k != "kısa" {
		t.Errorf("kısa metin değişmemeli: %q", k)
	}
}

func TestYenidenKapisiHTMLEntityEngeller(t *testing.T) {
	y := yenidenYanit{Spot: yenidenIyiSpot(), Paragraflar: yenidenIyiMetin()}
	y.Paragraflar[0] += " Metinde &nbsp; varlığı var."
	_, engel := yenidenKapisi(y, "", "")
	if !strings.Contains(engel, "HTML varlığı") {
		t.Errorf("HTML varlığı engellenmedi: %q", engel)
	}
}

func TestYenidenKapisiTekrarCumleyiEngeller(t *testing.T) {
	p := yenidenIyiMetin()
	p[4] = p[0] // aynı cümle iki kez
	y := yenidenYanit{Spot: yenidenIyiSpot(), Paragraflar: p}
	_, engel := yenidenKapisi(y, "", "")
	if !strings.Contains(engel, "aynı cümle") {
		t.Errorf("tekrar cümle engellenmedi: %q", engel)
	}
}

func TestYenidenKapisiKaynakSizintisiniEngeller(t *testing.T) {
	y := yenidenYanit{Spot: yenidenIyiSpot(), Paragraflar: yenidenIyiMetin()}
	y.Paragraflar[1] += " Ayrıntılar anadolugazetesi.com adresinde."
	_, engel := yenidenKapisi(y, "", "https://www.anadolugazetesi.com/haber/1")
	if !strings.Contains(engel, "kaynak site") {
		t.Errorf("kaynak sızıntısı engellenmedi: %q", engel)
	}
}

func TestYenidenKapisiBilinmeyenSayiyiUyarir(t *testing.T) {
	y := yenidenYanit{Spot: yenidenIyiSpot(), Paragraflar: yenidenIyiMetin()}
	y.Paragraflar[2] += " Sınav 2029 yılında da yapılacak."
	uyarilar, engel := yenidenKapisi(y, strings.Join(yenidenIyiMetin(), "\n\n"), "")
	if engel != "" {
		t.Fatalf("sayı uyarısı engellememeli: %s", engel)
	}
	if !uyariVar(uyarilar, "2029") {
		t.Errorf("uydurma sayı uyarısı yok: %v", uyarilar)
	}
}

func TestYenidenJSONBodyMetniniBolumlereBöler(t *testing.T) {
	ham := `{"spot":"Spot metni","body":"Birinci paragraf.\nİkinci paragraf.","seo_title":"Başlık","meta_description":"Açıklama"}`
	y, err := yenidenJSON(ham)
	if err != nil {
		t.Fatalf("ayrıştırılamadı: %v", err)
	}
	if len(y.Paragraflar) != 2 {
		t.Fatalf("paragraf sayısı 2 bekleniyordu: %v", y.Paragraflar)
	}
	if y.SeoBaslik != "Başlık" {
		t.Errorf("SEO başlığı okunmadı: %q", y.SeoBaslik)
	}
}

func TestYenidenJSONKodBlogunuSoyar(t *testing.T) {
	ham := "```json\n{\"paragraphs\":[\"Bir paragraf.\"]}\n```"
	y, err := yenidenJSON(ham)
	if err != nil {
		t.Fatalf("kod bloğu ayrıştırılamadı: %v", err)
	}
	if len(y.Paragraflar) != 1 {
		t.Errorf("paragraf okunmadı: %v", y.Paragraflar)
	}
}

func TestYenidenIstemDileUyar(t *testing.T) {
	trSistem, trKullanici := yenidenIstem("tr", "Başlık", "Spot", "Gövde", false)
	if !strings.Contains(trSistem, "Türkçe") {
		t.Error("Türkçe istem beklenirdi")
	}
	if !strings.Contains(trKullanici, "Başlık") {
		t.Error("kullanıcı isteminde başlık yok")
	}
	enSistem, _ := yenidenIstem("en", "Headline", "Summary", "Body", false)
	if !strings.Contains(enSistem, "English") {
		t.Error("İngilizce istem beklenirdi")
	}
	if strings.Contains(enSistem, "Türkçe") {
		t.Error("İngilizce istem Türkçe metin içeriyor")
	}
}
