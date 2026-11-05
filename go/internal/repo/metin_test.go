package repo

import (
	"strings"
	"testing"
)

// Faz 193a — metin hijyeni.
//
// ÖLÇÜM (üretim): 1094 içerikte gövdede "&nbsp;", 2405'inde sayısal varlık,
// 1088'inde agregatör başlık yığını; yayındaki 982/986 gövde TEK paragraf;
// 94 spot cümle ortasında bitiyordu.

func TestVarlikCoz(t *testing.T) {
	cases := []struct{ in, out string }{
		{"Mardin&#039;in Kızıltepe ilçesinde", "Mardin'in Kızıltepe ilçesinde"},
		{"a&nbsp;b", "a b"},
		{"fiyat &amp; kur", "fiyat & kur"},
		{`"tırnak&quot; içinde"`, `"tırnak" içinde"`},
		{"çift&amp;nbsp;kaçırma", "çift kaçırma"}, // iki geçiş
		{"&ldquo;alıntı&rdquo;", "“alıntı”"},
		{"boşluk   çoklu\tsekme", "boşluk çoklu sekme"},
		{"kırılmasız\u00a0boşluk", "kırılmasız boşluk"},
	}
	for _, c := range cases {
		if got := varlikCoz(c.in); got != c.out {
			t.Errorf("varlikCoz(%q) = %q, beklenen %q", c.in, got, c.out)
		}
	}
}

func TestYapistirilmisBasliklar(t *testing.T) {
	// Gerçek üretim örneği (içerik 45493'ten kısaltıldı).
	glue := "Diaz-Canel, BM'de 130 oyla kabul edilen ambargo kararını halkının " +
		"zaferi olarak gördü&nbsp;&nbsp;Son DakikaBM'de ABD'nin Küba ambargosunun " +
		"kaldırılmasına destek geriledi&nbsp;&nbsp;Euronews.comABD'li ve Kübalı"
	ilk, kes := yapistirilmisBasliklar(glue)
	if !kes {
		t.Fatal("agregatör yapıştırması kesilmedi")
	}
	if strings.Contains(ilk, "Son Dakika") || strings.Contains(ilk, "&nbsp;") {
		t.Errorf("ilk cümleden sonrası kalmış: %q", ilk)
	}
	if !strings.Contains(ilk, "Diaz-Canel") {
		t.Errorf("ilk haber cümlesi kaybolmuş: %q", ilk)
	}

	// Meşru metin: tek nbsp geçiyor, site adı yok → DOKUNULMAZ.
	mesru := "Bakan açıklama yaptı ve rakamları paylaştı&nbsp;bugün."
	if _, kes := yapistirilmisBasliklar(mesru); kes {
		t.Error("meşru metin kesildi")
	}

	// Kısa ilk parça: kesmek bilgi kaybettirir → dokunulmaz.
	kisa := "Kısa&nbsp;&nbsp;Son Dakika bir şeyler oldu .com"
	if _, kes := yapistirilmisBasliklar(kisa); kes {
		t.Error("kısa ilk parça kesildi (bilgi kaybı)")
	}
}

func TestCumleGuvenliKirp(t *testing.T) {
	uzun := strings.Repeat("Bu bir cümledir. ", 20) + "SON"
	got := cumleGuvenliKirp(uzun, 100)
	if len([]rune(got)) > 101 {
		t.Errorf("sınır aşıldı: %d", len([]rune(got)))
	}
	if !strings.HasSuffix(got, ".") {
		t.Errorf("cümle sonunda bitmedi: %q", got)
	}

	// Cümle sonu yoksa kelime sınırı + üç nokta.
	uzun2 := "kelime " + strings.Repeat("dolgu ", 40)
	got2 := cumleGuvenliKirp(uzun2, 40)
	if !strings.HasSuffix(got2, "…") {
		t.Errorf("üç nokta eklenmedi: %q", got2)
	}
	if strings.HasSuffix(strings.TrimSuffix(got2, "…"), " ") {
		t.Errorf("kelime ortasında boşlukla bitti: %q", got2)
	}

	// Sığan metin aynen döner.
	if got := cumleGuvenliKirp("kısa metin.", 100); got != "kısa metin." {
		t.Errorf("kısa metin değişti: %q", got)
	}
}

func TestParagraflaraBol(t *testing.T) {
	// Tek paragraf duvarı → birkaç okunur paragraf.
	duvar := strings.Repeat("Bu ilk cümledir ve yeterince uzundur. ", 3) +
		strings.Repeat("İkinci küme cümlesi de uzundur. ", 3) +
		strings.Repeat("Üçüncü küme cümlesi de uzundur. ", 3)
	p := paragraflaraBol(duvar)
	if len(p) < 2 {
		t.Fatalf("tek paragraf kaldı: %d", len(p))
	}
	// Hiçbir cümle kaybolmamalı.
	birlesik := strings.Join(p, " ")
	for _, cumle := range []string{"Bu ilk cümledir", "İkinci küme", "Üçüncü küme"} {
		if !strings.Contains(birlesik, cumle) {
			t.Errorf("%q kayboldu", cumle)
		}
	}
	// Varlık çözülmüş olmalı.
	if got := paragraflaraBol("a&nbsp;b c."); strings.Contains(strings.Join(got, " "), "&nbsp;") {
		t.Error("varlık çözülmedi")
	}
	// Boş girdi.
	if got := paragraflaraBol("   "); len(got) != 0 {
		t.Errorf("boş girdi paragraf üretti: %v", got)
	}
}

func TestStripHTMLVarlikVeYapistirma(t *testing.T) {
	// Bitişik gerçek akış: HTML etiketi + varlık + agregatör yapıştırması.
	in := `<p>Mahkeme kararı <b>açıklandı</b>&nbsp;&nbsp;Son DakikaYeni karar ` +
		`bugün yayımlandı&nbsp;&nbsp;Euronews.comAyrıntılar geliyor</p>`
	got := stripHTML(in)
	if strings.Contains(got, "&nbsp;") {
		t.Errorf("varlık kaldı: %q", got)
	}
	if strings.Contains(got, "<") {
		t.Errorf("etiket kaldı: %q", got)
	}
	if strings.Contains(got, "Euronews") {
		t.Errorf("agregatör yapıştırması kesilmedi: %q", got)
	}
	if !strings.Contains(got, "Mahkeme kararı açıklandı") {
		t.Errorf("ilk cümle kayboldu: %q", got)
	}
}
