package aieditor

import (
	"strings"
	"testing"
)

// Faz 193c — AI haber çıktısının yapı/SEO kapısı.
//
// ÖLÇÜM (üretim): yayındaki 986 içeriğin 982'si tek paragraf; 94 spot cümle
// ortasında; 3.628 içerikte &nbsp;/&#039;; 206 SEO başlığı başlığın birebir
// kopyası ve 60 karakterden uzun.

func newsAction() Action { return Action{ID: "news"} }

func goodResult() *Result {
	// Her paragraf FARKLI olmalı: tekrar kapısı aynı cümleyi yakalıyor
	// (ilk sürümde dört özdeş paragraf kullanmıştım ve test kendini yakaladı).
	p1 := "Borsa İstanbul'da gözaltı kararı sonrası soruşturma derinleşti ve " +
		"kurum yöneticileri açıklama yaptı. Sürecin ekonomik yansımaları izleniyor."
	p2 := "Soruşturma kapsamında kurumun üst yönetiminden iki kişinin ifadesine " +
		"başvuruldu. Yetkililer sürecin şeffaf yürütüleceğini bildirdi."
	p3 := "Piyasa uzmanları gelişmenin Borsa İstanbul üzerindeki etkisini değerlendirdi. " +
		"Uzmanlar kısa vadede dalgalanma beklediklerini söyledi."
	p4 := "Kurumdan yapılan yazılı açıklamada operasyonel faaliyetlerin kesintisiz " +
		"sürdüğü belirtildi. Açıklamada müşteri hesaplarının ve saklama hizmetlerinin " +
		"etkilenmediği vurgulandı. Ayrıntılı bilgi için adli süreç izleniyor ve " +
		"gelişmeler gün içinde kamuoyuyla paylaşılacak. Editör notu: metin uzunluk " +
		"eşiğini geçmesi için bu paragraf bilinçli olarak geniş tutuldu."
	r := &Result{
		Action: "news",
		Title:  "Borsa İstanbul yöneticisine gözaltı: soruşturma derinleşiyor",
		Fields: map[string]string{
			"spot":             "Borsa İstanbul Genel Müdürü hakkında başlatılan soruşturma kapsamında gözaltı kararı verildi; kurum süreci yakından izliyor ve piyasa etkisi tartışılıyor.",
			"body":             strings.Join([]string{p1, p2, p3, p4}, "\n\n"),
			"seo_title":        "Borsa İstanbul'da gözaltı: soruşturma büyüyor",
			"meta_description": "Borsa İstanbul Genel Müdürü hakkındaki soruşturmada gözaltı kararı verildi. Kurumdan açıklama geldi, piyasalardaki etkisi izleniyor.",
		},
	}
	return r
}

func uyariVar(w []string, parca string) bool {
	for _, s := range w {
		if strings.Contains(s, parca) {
			return true
		}
	}
	return false
}

func TestYapiKapisiTekParagrafiYakalar(t *testing.T) {
	r := goodResult()
	// Sahadaki gerçek durum: tek blok, kısa gövde.
	r.Fields["body"] = "Borsa İstanbul'da gözaltı kararı verildi."
	w := yapiUyarilari(r, newsAction(), "")
	if !uyariVar(w, "en az 3 paragraf") {
		t.Errorf("tek paragraf yakalanmadı: %v", w)
	}
	if !uyariVar(w, "karakter") {
		t.Errorf("kısa gövde yakalanmadı: %v", w)
	}
}

func TestYapiKapisiTemizSonucGecer(t *testing.T) {
	if w := yapiUyarilari(goodResult(), newsAction(), ""); len(w) != 0 {
		t.Errorf("temiz sonuç uyarı üretti: %v", w)
	}
}

func TestYapiKapisiVarlikVeEtiket(t *testing.T) {
	r := goodResult()
	r.Fields["body"] = strings.Replace(r.Fields["body"], "Borsa İstanbul'da", "Borsa&nbsp;İstanbul'da", 1)
	if w := yapiUyarilari(r, newsAction(), ""); !uyariVar(w, "HTML varlığı") {
		t.Errorf("&nbsp; yakalanmadı: %v", w)
	}

	r2 := goodResult()
	r2.Fields["body"] = "<p>" + r2.Fields["body"] + "</p>"
	if w := yapiUyarilari(r2, newsAction(), ""); !uyariVar(w, "HTML etiketi") {
		t.Errorf("<p> yakalanmadı: %v", w)
	}
}

func TestYapiKapisiKalintiVeKaynakSizintisi(t *testing.T) {
	r := goodResult()
	r.Fields["body"] += "\n\nHaberin devamını oku"
	if w := yapiUyarilari(r, newsAction(), ""); !uyariVar(w, "kalıntısı") {
		t.Errorf("\"devamını oku\" yakalanmadı: %v", w)
	}

	r2 := goodResult()
	r2.Sources = []Source{{Ref: "S1", URL: "https://www.hurriyet.com.tr/gundem/haber"}}
	r2.Fields["body"] += "\n\nHurriyet'e göre süreç devam ediyor."
	if w := yapiUyarilari(r2, newsAction(), ""); !uyariVar(w, "kaynak site adı") {
		t.Errorf("kaynak alan adı sızıntısı yakalanmadı: %v", w)
	}
}

func TestYapiKapisiTekrarCumle(t *testing.T) {
	r := goodResult()
	cumle := "Soruşturmanın ekonomiye etkisi piyasa oyuncuları tarafından yakından izleniyor."
	r.Fields["body"] = strings.Join([]string{cumle, "İkinci paragraf farklı bir bilgi veriyor ve ayrıntı ekliyor burada.", cumle}, "\n\n")
	if w := yapiUyarilari(r, newsAction(), ""); !uyariVar(w, "Aynı cümle") {
		t.Errorf("tekrar eden cümle yakalanmadı: %v", w)
	}
}

func TestYapiKapisiUzunlukBantlari(t *testing.T) {
	r := goodResult()
	r.Fields["seo_title"] = strings.Repeat("uzun ", 20)
	r.Title = strings.Repeat("başlık ", 20)
	r.Fields["spot"] = "kısa spot"
	w := yapiUyarilari(r, newsAction(), "")
	if !uyariVar(w, "SEO başlığı uzun") {
		t.Errorf("uzun SEO başlığı yakalanmadı: %v", w)
	}
	if !uyariVar(w, "Başlık uzun") {
		t.Errorf("uzun başlık yakalanmadı: %v", w)
	}
	if !uyariVar(w, "Spot kısa") {
		t.Errorf("kısa spot yakalanmadı: %v", w)
	}
}

func TestYapiKapisiEksikSeoAlanlari(t *testing.T) {
	r := goodResult()
	delete(r.Fields, "seo_title")
	delete(r.Fields, "meta_description")
	w := yapiUyarilari(r, newsAction(), "")
	if !uyariVar(w, "SEO başlığı üretilmedi") {
		t.Errorf("eksik SEO başlığı yakalanmadı: %v", w)
	}
	if !uyariVar(w, "Meta açıklama üretilmedi") {
		t.Errorf("eksik meta yakalanmadı: %v", w)
	}
}

func TestYapiKapisiDigerEylemleriEtkilemez(t *testing.T) {
	r := goodResult()
	r.Fields["body"] = "kısa"
	if w := yapiUyarilari(r, Action{ID: "summary"}, ""); len(w) != 0 {
		t.Errorf("özet eyleminde haber kapısı çalıştı: %v", w)
	}
}

func TestNormalizeParagraphsGovdeyeYazilir(t *testing.T) {
	m := map[string]any{
		"title":            "Başlık",
		"spot":             "Spot metni",
		"paragraphs":       []any{"Birinci paragraf.", "İkinci paragraf.", "Üçüncü paragraf."},
		"seo_title":        "SEO başlığı",
		"meta_description": "Meta açıklama",
	}
	r, err := normalizeResult(newsAction(), m, nil)
	if err != nil {
		t.Fatalf("normalize hatası: %v", err)
	}
	body := alanDegeri(&r, "body")
	if strings.Count(body, "\n\n") != 2 {
		t.Errorf("paragraflar birleştirilmedi: %q", body)
	}
	if !strings.Contains(body, "Üçüncü paragraf.") {
		t.Errorf("son paragraf kayboldu: %q", body)
	}
	if alanDegeri(&r, "seo_title") == "" || alanDegeri(&r, "meta_description") == "" {
		t.Errorf("SEO alanları alınmadı: %+v", r.Fields)
	}
}

func TestNormalizeParagraphsYoksaBodyKullanilir(t *testing.T) {
	m := map[string]any{"title": "Başlık", "body": "Tek blok gövde"}
	r, err := normalizeResult(newsAction(), m, nil)
	if err != nil {
		t.Fatalf("normalize hatası: %v", err)
	}
	if alanDegeri(&r, "body") != "Tek blok gövde" {
		t.Errorf("gövde bozuldu: %q", alanDegeri(&r, "body"))
	}
}
