package httpapi

import "theblock/site/internal/piyasa"

const (
	// Varsayılanlar referans düzendir: fiyat renkli, noktalı ok, arka plan yok.
	kbStilVarsayilan = "referans"
	kbOkVarsayilan   = "noktali"
	kbBgVarsayilan   = "yok"
)

// gecerliKbStil bilinen stil anahtarları.
var gecerliKbStil = map[string]bool{
	"referans": true, // fiyat yön renginde, çerçevesiz, alt çizgili (referans)
	"sade":     true, // nötr fiyat, yalnız değişim renkli
	"kart":     true, // her kalem ayrı kart içinde
	"dar":      true, // tek satır, kompakt (mobil/yan sütun)
}

var gecerliKbOk = map[string]bool{
	"noktali": true,
	"ucgen":   true,
	"ok":      true,
	"uzun":    true,
	"yok":     true,
}

var gecerliKbBg = map[string]bool{
	"kart":  true, // çerçeveli yüzey (isteğe bağlı)
	"yok":   true, // saydam (referans)
	"zemin": true, // yüzey rengi
	"ozel":  true, // stBg ile seçilen renk
}

// kbStilOf blok ayarlarından şablonu okur; bilinmeyen değer varsayılana döner.
func kbStilOf(opts map[string]any) string {
	if s, ok := opts["kb_style"].(string); ok && gecerliKbStil[s] {
		return s
	}
	return kbStilVarsayilan
}

// kbOkOf blok ayarlarından ok tasarımını okur.
func kbOkOf(opts map[string]any) string {
	if s, ok := opts["kb_arrow"].(string); ok && gecerliKbOk[s] {
		return s
	}
	return kbOkVarsayilan
}

// kbBgOf blok ayarlarından arka plan seçimini okur.
func kbBgOf(opts map[string]any) string {
	if s, ok := opts["kb_bg"].(string); ok && gecerliKbBg[s] {
		return s
	}
	return kbBgVarsayilan
}

func kbArkaPlanCSS(bg, ozelRenk string) string {
	switch bg {
	case "kart":
		return "background:var(--hp-surface,#fff);border:1px solid var(--hp-border,rgba(0,0,0,.08));border-radius:var(--hp-radius,8px);padding:10px 12px;"
	case "zemin":
		return "background:var(--hp-surface-2,#f4f4f5);border-radius:var(--hp-radius,8px);padding:10px 12px;"
	case "ozel":
		if c := sanitizeHexColor(ozelRenk); c != "" {
			return "background:" + c + ";border-radius:var(--hp-radius,8px);padding:10px 12px;"
		}
	}
	return ""
}

type kbArrow struct {
	Stil string
	Yon  int
	Cls  string
}

// kbArrowFor şablon yardımcısı: {{template "kb_ok" (kbArrow $.PiyasaOk .Yon)}}.
func kbArrowFor(stil string, yon int) kbArrow {
	if !gecerliKbOk[stil] {
		stil = kbOkVarsayilan
	}
	return kbArrow{Stil: stil, Yon: yon}
}

// hssOkFor Hisseler modülünün yön göstergesi (kb_ok, hss- sınıf önekiyle).
func hssOkFor(stil string, yon int) kbArrow {
	if !gecerliKbOk[stil] {
		stil = hisseOkVarsayilan
	}
	return kbArrow{Stil: stil, Yon: yon, Cls: "hss"}
}

// piyasaNum fiyatı Türkçe biçimde yazar: 11151.03, 2 → "11.151,03".
func piyasaNum(v float64, ondalik int) string {
	return piyasa.FormatTR(v, ondalik)
}

// piyasaChg yüzde değişimi işaretli yazar: 0.05 → "+0,05".
func piyasaChg(d float64) string {
	return piyasa.FormatDegisim(d)
}
