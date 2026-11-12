package repo

import (
	"testing"

	"theblock/site/internal/quality"
)

func TestAyniBaslikFarkliSpotMukerrer(t *testing.T) {
	// Gerçek üretim verisinden (55583 / 55587 / 55591).
	baslik := "Borsa İstanbul Genel Müdürü Korkmaz Enes Ergun gözaltına alındı"
	adaylar := []quality.Candidate{
		{
			Ref:   "55583",
			Title: baslik,
			Text:  baslik + " " + "Yeni Şafak'ın haberine göre Borsa İstanbul'da gözaltı işlemi yapıldı ve soruşturma başlatıldı.",
		},
		{
			Ref:   "55587",
			Title: baslik,
			Text:  baslik + " " + "Ekonomim: Korkmaz Enes Ergun hakkında başlatılan soruşturmanın ayrıntıları netleşiyor.",
		},
	}
	bulgu, dup := DuplicateOfCandidates(baslik,
		"Gazete Pencere: gözaltı kararı sonrası açıklama yapıldı, süreç devam ediyor.", adaylar)
	if !dup {
		t.Fatal("aynı başlık mükerrer sayılmadı (spot farkı eşiğin altına düşürdü)")
	}
	if bulgu.Ref == "" {
		t.Error("eşleşen kayıt kimliği boş")
	}
}

func TestBaslikAyniDegilseGecir(t *testing.T) {
	adaylar := []quality.Candidate{
		{Ref: "1", Title: "Merkez Bankası faiz kararını açıkladı", Text: "Merkez Bankası faiz kararını açıkladı piyasa tepkisi"},
	}
	// Farklı olay, farklı başlık → geçmeli.
	if _, dup := DuplicateOfCandidates("Mardin'de asansör kabini çöktü",
		"Kızıltepe ilçesinde bir binada asansör kabini çöktü, iki kişi yaralandı.", adaylar); dup {
		t.Error("farklı haber mükerrer sayıldı")
	}
}

func TestNormalizeBaslikEslesmesi(t *testing.T) {
	// Noktalama/büyük-küçük farkı eşleşmeli (Türkçe İ/I dahil).
	adaylar := []quality.Candidate{
		{Ref: "9", Title: "Bakan Göktaş Duyurdu: Sosyal Medya Düzenlemesi", Text: "x"},
	}
	if _, ok := ayniBaslikBulgu("bakan göktaş duyurdu sosyal medya düzenlemesi", adaylar); !ok {
		t.Error("noktalama/büyük-küçük farkı eşleşmedi")
	}
	if _, ok := ayniBaslikBulgu("bambaşka bir başlık", adaylar); ok {
		t.Error("farklı başlık eşleşti")
	}
	if _, ok := ayniBaslikBulgu("", adaylar); ok {
		t.Error("boş başlık eşleşti (dedupe karar vermemeli)")
	}
}

func TestDedupeHierarchyStrongRule(t *testing.T) {
	// Agregatör kopyası varken orijinal gelirse: orijinal kazanır.
	if got := DedupeDecision(false, true); got != DedupeReplace {
		t.Errorf("orijinal, agregatörü yenmeli; alınan %q", got)
	}
	// Orijinal varken agregatör gelirse: agregatör atılır.
	if got := DedupeDecision(true, false); got != DedupeSkip {
		t.Errorf("agregatör, orijinali yenmemeli; alınan %q", got)
	}
}
