package httpapi

import (
	"strings"
	"testing"

	"theblock/site/internal/repo"
)

// TEŞHİS CÜMLELERİ panelin asıl değeri: bu oturumdaki üç arızanın
// hiçbiri tek bir sayıya bakarak görülmedi, hepsi İKİ sayının birlikte
// anlamından çıktı. Paneli okuyan kişi o bağlantıyı her seferinde
// kendi kurmak zorunda kalmasın.
func TestUretimTeshisKuyrukBosAmaBekleyenVar(t *testing.T) {
	var s Server
	// Gerçek durum (13 Kasım 2026): kuyruk 0, bekleyen 1972.
	h := s.uretimTeshis(nil, repo.UretimOzet{Kuyrukta: 0, BekleyenTaslak: 1972})
	if !strings.Contains(h, "besleme kesik") {
		t.Errorf("kuyruk boş + bekleyen çok durumu teşhis edilmedi:\n%s", h)
	}
	if !strings.Contains(h, "1972") {
		t.Error("teşhiste ölçülen sayı geçmiyor")
	}
}

// Kuyruk boş ama bekleyen de azsa UYARI OLMAMALI: hat gerçekten boşta
// ve bu normal. Yanlış alarm paneli gürültüye çevirir.
func TestUretimTeshisKuyrukBosBekleyenAz(t *testing.T) {
	var s Server
	h := s.uretimTeshis(nil, repo.UretimOzet{Kuyrukta: 0, BekleyenTaslak: 3})
	if strings.Contains(h, "besleme kesik") {
		t.Errorf("az bekleyen için yanlış alarm:\n%s", h)
	}
}

// Yüksek hata oranı uyarmalı VE eşzamanlılık tuzağını söylemeli:
// sağlayıcı kaynaklı kayıpta işçi sayısını artırmak kaybı büyütür.
func TestUretimTeshisYuksekHataOrani(t *testing.T) {
	var s Server
	// Gerçek durum: 318 tamam / 143 hata = %31.
	h := s.uretimTeshis(nil, repo.UretimOzet{Son24Tamam: 318, Son24Hata: 143})
	if !strings.Contains(h, "%31") {
		t.Errorf("hata oranı hesaplanmadı:\n%s", h)
	}
	if !strings.Contains(h, "eşzamanlılığı artırmak") {
		t.Error("eşzamanlılık tuzağı uyarısı yok")
	}
}

func TestUretimTeshisDusukHataOraniSessiz(t *testing.T) {
	var s Server
	h := s.uretimTeshis(nil, repo.UretimOzet{Son24Tamam: 100, Son24Hata: 2})
	if strings.Contains(h, "başarısız") {
		t.Errorf("%%2 hata için uyarı verildi:\n%s", h)
	}
}

// Medyan uzunluk hedefin altındaysa uyarmalı: çıplak sayı "iyi mi
// kötü mü" sorusunu yanıtlamıyor.
func TestUretimTeshisKisaIcerik(t *testing.T) {
	var s Server
	// Gerçek durum: medyan 85 kelime, haber hedefi 350-550.
	h := s.uretimTeshis(nil, repo.UretimOzet{Son24MedyanKelime: 85})
	if !strings.Contains(h, "85") || !strings.Contains(h, "ince içerik") {
		t.Errorf("kısa içerik teşhis edilmedi:\n%s", h)
	}
}

// Hedef bandın içindeki uzunluk uyarı üretmemeli.
func TestUretimTeshisYeterliUzunluk(t *testing.T) {
	var s Server
	h := s.uretimTeshis(nil, repo.UretimOzet{Son24MedyanKelime: 420})
	if strings.Contains(h, "ince içerik") {
		t.Errorf("hedef içindeki uzunluk için uyarı verildi:\n%s", h)
	}
}

// Her şey yolundaysa panel bunu AÇIKÇA söylemeli; boş bölüm "ölçüm
// yapılmadı" ile karışır.
func TestUretimTeshisTemizDurum(t *testing.T) {
	var s Server
	h := s.uretimTeshis(nil, repo.UretimOzet{
		Son24Tamam: 100, Son24Hata: 1, Kuyrukta: 5, BekleyenTaslak: 10, Son24MedyanKelime: 400,
	})
	if !strings.Contains(h, "Dikkat gerektiren bir durum görünmüyor") {
		t.Errorf("temiz durum bildirilmedi:\n%s", h)
	}
}

// Sıfıra bölme: iş yokken oran hesaplanamaz, panel çökmemeli.
func TestUretimKutularSifirIs(t *testing.T) {
	h := uretimKutular(repo.UretimOzet{})
	if !strings.Contains(h, "Tamamlanan") {
		t.Error("boş özet kutuları basmadı")
	}
	if strings.Contains(h, "NaN") || strings.Contains(h, "+Inf") {
		t.Errorf("sıfıra bölme sızmış:\n%s", h)
	}
}

// Medyan uzunluk kutusu HEDEFİ de göstermeli.
func TestUretimKutularUzunlukHedefi(t *testing.T) {
	h := uretimKutular(repo.UretimOzet{Son24MedyanKelime: 85})
	if !strings.Contains(h, "hedef 350-550") {
		t.Errorf("uzunluk kutusunda hedef yok:\n%s", h)
	}
}

// Trafik bölümü boş olmamalı: NEDEN boş olduğu yazmalı, yoksa
// "veri yok" ile "bağlantı yok" karışır.
func TestUretimTrafikYuvasiSebepYazar(t *testing.T) {
	h := uretimTrafikYuvasi()
	if !strings.Contains(h, "gsc_property") {
		t.Error("eksik bağlantının adı yazılmamış")
	}
	if !strings.Contains(h, "Marjinal fayda") {
		t.Error("marjinal fayda analizinin bekleyen iş olduğu belirtilmemiş")
	}
}

func TestKisaltMetin(t *testing.T) {
	if got := kisaltMetin("kısa", 10); got != "kısa" {
		t.Errorf("kısa metin değişmiş: %q", got)
	}
	// Türkçe karakterler RUNE sayılmalı, bayt değil.
	uzun := "çok uzun bir başlık ğüşiöç"
	got := kisaltMetin(uzun, 10)
	if len([]rune(got)) != 10 {
		t.Errorf("kırpma rune saymıyor: %q (%d rune)", got, len([]rune(got)))
	}
	if !strings.HasSuffix(got, "…") {
		t.Errorf("kırpma işareti yok: %q", got)
	}
}
