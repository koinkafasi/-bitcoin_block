package httpapi

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Gazete şablon belgesi — specs/13-gazete-sayfa-duzenleyici.md §5.3
//
// Şablon, sayfanın "peyzajını" taşır: hangi blok nerede, hangi içeriğe bağlı,
// hangi sayfa boyunda ve hangi ızgarada. Sayı oluşturulurken uygulanır.
//
// Geriye dönük uyum: eski biçim düz bir sayfa dizisiydi (`[{no,ad,bloklar}]`).
// Çözücü ikisini de kabul eder; düz dizi gelirse A4 varsayılanlarıyla sarılır.

// gazeteSayfaAyar şablonun sayfa ölçüsü ve tipografi bildirimi (K2).
type gazeteSayfaAyar struct {
	Boyut     string  `json:"boyut"`     // A4 | A3
	Yon       string  `json:"yon"`       // dikey
	TipOlcek  float64 `json:"tipOlcek"`  // tipografi çarpanı
	TasmaPayi float64 `json:"tasmaPayi"` // mm, bleed
	KesimPayi float64 `json:"kesimPayi"` // mm, trim
	YaziTipi  string  `json:"yaziTipi"`  // boş ise baskı CSS varsayılanı
}

// gazeteIzgara sütun ızgarası (K4). Ölçüler mm.
type gazeteIzgara struct {
	Sutun int     `json:"sutun"`
	Oluk  float64 `json:"oluk"`
	Kenar float64 `json:"kenar"`
}

// gazeteSablonDoc şablon belgesinin tamamı.
type gazeteSablonDoc struct {
	Surum    int             `json:"surum"`
	Sayfa    gazeteSayfaAyar `json:"sayfa"`
	Izgara   gazeteIzgara    `json:"izgara"`
	Sayfalar []gazetePageDoc `json:"sayfalar"`
}

// gazeteSablonVarsayilanSayfa bildirilmemiş alanlar için A4 varsayılanı.
func gazeteSablonVarsayilanSayfa() gazeteSayfaAyar {
	return gazeteSayfaAyar{Boyut: "A4", Yon: "dikey", TipOlcek: 1, TasmaPayi: 3, KesimPayi: 5}
}

// gazeteSablonVarsayilanIzgara makul bir gazete ızgarası.
func gazeteSablonVarsayilanIzgara() gazeteIzgara {
	return gazeteIzgara{Sutun: 12, Oluk: 4, Kenar: 12}
}

// gazeteSablonCoz ham JSONB'yi şablon belgesine çevirir.
// Hem yeni nesne biçimini hem eski düz sayfa dizisini kabul eder.
func gazeteSablonCoz(raw []byte) (gazeteSablonDoc, error) {
	d := gazeteSablonDoc{
		Surum:  1,
		Sayfa:  gazeteSablonVarsayilanSayfa(),
		Izgara: gazeteSablonVarsayilanIzgara(),
	}
	ham := strings.TrimSpace(string(raw))
	if ham == "" || ham == "null" {
		return d, fmt.Errorf("şablon belgesi boş")
	}
	if strings.HasPrefix(ham, "[") {
		// Eski biçim: yalnız sayfa dizisi.
		var sayfalar []gazetePageDoc
		if err := json.Unmarshal(raw, &sayfalar); err != nil {
			return d, fmt.Errorf("sayfa dizisi çözülemedi: %w", err)
		}
		d.Sayfalar = sayfalar
		return d, nil
	}
	var gelen gazeteSablonDoc
	if err := json.Unmarshal(raw, &gelen); err != nil {
		return d, fmt.Errorf("şablon belgesi çözülemedi: %w", err)
	}
	if gelen.Surum > 0 {
		d.Surum = gelen.Surum
	}
	if gelen.Sayfa.Boyut != "" {
		d.Sayfa.Boyut = gelen.Sayfa.Boyut
	}
	if gelen.Sayfa.Yon != "" {
		d.Sayfa.Yon = gelen.Sayfa.Yon
	}
	if gelen.Sayfa.TipOlcek > 0 {
		d.Sayfa.TipOlcek = gelen.Sayfa.TipOlcek
	}
	if gelen.Sayfa.TasmaPayi > 0 {
		d.Sayfa.TasmaPayi = gelen.Sayfa.TasmaPayi
	}
	if gelen.Sayfa.KesimPayi > 0 {
		d.Sayfa.KesimPayi = gelen.Sayfa.KesimPayi
	}
	d.Sayfa.YaziTipi = gelen.Sayfa.YaziTipi
	if gelen.Izgara.Sutun > 0 {
		d.Izgara = gelen.Izgara
	}
	d.Sayfalar = gelen.Sayfalar
	return d, nil
}

// gazeteBlokTipleri tanınan blok tipleri. Go üreticisi, editör paleti ve
// şablon doğrulayıcı aynı listeyi kullanır.
var gazeteBlokTipleri = []string{
	"manset", "liste", "kart", "baslik", "metin",
	"gorsel", "cizgi", "reklam", "kunye",
}

func gazeteBlokTipiGecerli(t string) bool {
	for _, x := range gazeteBlokTipleri {
		if x == t {
			return true
		}
	}
	return false
}

// gazeteSablonDogrula şablonu denetler ve sorunları insan diliyle döner.
// Boş dilim = şablon geçerli.
//
// Sessiz kabul etmiyoruz: sayfa dışına taşan ya da tipi bilinmeyen bir blok,
// baskıda görünmeyen veya yanlış yerde duran içerik demektir.
func gazeteSablonDogrula(d gazeteSablonDoc) []string {
	var sorun []string
	ekle := func(f string, a ...any) { sorun = append(sorun, fmt.Sprintf(f, a...)) }

	if d.Surum != 1 {
		ekle("bilinmeyen şablon sürümü: %d", d.Surum)
	}
	if _, ok := gazeteBoyutlar[strings.ToUpper(d.Sayfa.Boyut)]; !ok {
		ekle("bilinmeyen sayfa boyu: %q", d.Sayfa.Boyut)
	}
	if d.Sayfa.TipOlcek < 0.5 || d.Sayfa.TipOlcek > 2 {
		ekle("tipografi ölçeği 0.5–2 arasında olmalı: %.2f", d.Sayfa.TipOlcek)
	}
	if d.Izgara.Sutun < 1 || d.Izgara.Sutun > 24 {
		ekle("ızgara sütunu 1–24 arasında olmalı: %d", d.Izgara.Sutun)
	}
	if len(d.Sayfalar) == 0 {
		ekle("şablonda hiç sayfa yok")
		return sorun
	}

	gorulen := map[int]bool{}
	for _, p := range d.Sayfalar {
		if p.No < 1 {
			ekle("geçersiz sayfa numarası: %d", p.No)
			continue
		}
		if gorulen[p.No] {
			ekle("sayfa %d iki kez tanımlı", p.No)
		}
		gorulen[p.No] = true

		kimlik := map[string]bool{}
		for i, b := range p.Blocks {
			yer := fmt.Sprintf("sayfa %d blok %d", p.No, i+1)
			if b.ID == "" {
				ekle("%s: kimliksiz", yer)
			} else if kimlik[b.ID] {
				ekle("%s: kimlik %q tekrar ediyor", yer, b.ID)
			}
			kimlik[b.ID] = true

			if !gazeteBlokTipiGecerli(b.Tip) {
				ekle("%s: bilinmeyen tip %q", yer, b.Tip)
			}
			if b.W <= 0 || b.H <= 0 {
				ekle("%s: ölçüsü sıfır veya negatif (%.1f×%.1f)", yer, b.W, b.H)
			}
			if b.X < 0 || b.Y < 0 {
				ekle("%s: sayfanın dışında başlıyor (%.1f,%.1f)", yer, b.X, b.Y)
			}
			// Yüzdelik kutu sayfayı aşmamalı; aşarsa baskıda kırpılır.
			if b.X+b.W > 100.01 {
				ekle("%s: sağdan taşıyor (%.1f+%.1f > 100)", yer, b.X, b.W)
			}
			if b.Y+b.H > 100.01 {
				ekle("%s: alttan taşıyor (%.1f+%.1f > 100)", yer, b.Y, b.H)
			}
			if s := gazeteStilSorunlari(b.Stil); len(s) > 0 {
				ekle("%s: %s", yer, strings.Join(s, ", "))
			}
		}
	}
	sort.Strings(sorun)
	return sorun
}

// ---- Stil beyaz listesi (SPEC §5.2) ----
//
// `stil` serbest bir sözlük; doğrudan CSS'e geçirmek enjeksiyon yüzeyi açar.
// Bu yüzden yalnız bilinen anahtarlar ve bilinen DEĞERLER kabul edilir;
// değerler jeton, serbest renk değil. Liste dışı anahtar sessizce atılır.

var gazeteStilDegerleri = map[string][]string{
	"zemin":     {"yok", "lacivert", "turuncu", "koyu", "acik", "beyaz"},
	"metinRenk": {"koyu", "acik"},
	"hiza":      {"sol", "orta", "sag"},
	"bindirme":  {"yok", "alt", "ust", "tam"},
	"kenarlik":  {"yok", "ince", "kalin"},
}

// gazeteStilSayisal sayısal stil anahtarlarının sınırları.
var gazeteStilSayisal = map[string]struct{ min, max float64 }{
	"sutun": {1, 4},
	"olcek": {0.6, 2.0},
}

func gazeteStilDegerGecerli(anahtar, deger string) bool {
	for _, v := range gazeteStilDegerleri[anahtar] {
		if v == deger {
			return true
		}
	}
	return false
}

// gazeteStilSorunlari bilinmeyen anahtar/değerleri bildirir (doğrulama için).
func gazeteStilSorunlari(stil map[string]any) []string {
	var out []string
	for k, v := range stil {
		if _, sayisal := gazeteStilSayisal[k]; sayisal {
			f, ok := gazeteSayiyaCevir(v)
			sinir := gazeteStilSayisal[k]
			if !ok || f < sinir.min || f > sinir.max {
				out = append(out, fmt.Sprintf("stil %q değeri aralık dışı (%v)", k, v))
			}
			continue
		}
		if _, bilinen := gazeteStilDegerleri[k]; !bilinen {
			out = append(out, fmt.Sprintf("bilinmeyen stil anahtarı %q", k))
			continue
		}
		s, _ := v.(string)
		if !gazeteStilDegerGecerli(k, s) {
			out = append(out, fmt.Sprintf("stil %q için geçersiz değer %v", k, v))
		}
	}
	sort.Strings(out)
	return out
}

func gazeteSayiyaCevir(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case int:
		return float64(t), true
	case json.Number:
		f, err := t.Float64()
		return f, err == nil
	}
	return 0, false
}

// gazeteStilSinif stil sözlüğünü güvenli CSS sınıflarına çevirir.
// Liste dışı her şey atılır; çıktı yalnız sabit önekli sınıf adlarıdır.
func gazeteStilSinif(stil map[string]any) string {
	if len(stil) == 0 {
		return ""
	}
	onek := map[string]string{
		"zemin": "gb-z-", "metinRenk": "gb-m-", "hiza": "gb-h-",
		"bindirme": "gb-b-", "kenarlik": "gb-k-",
	}
	var siniflar []string
	for anahtar, pre := range onek {
		s, _ := stil[anahtar].(string)
		if s == "" || s == "yok" || !gazeteStilDegerGecerli(anahtar, s) {
			continue
		}
		siniflar = append(siniflar, pre+s)
	}
	if f, ok := gazeteSayiyaCevir(stil["sutun"]); ok && f >= 1 && f <= 4 {
		siniflar = append(siniflar, fmt.Sprintf("gb-s-%d", int(f)))
	}
	sort.Strings(siniflar)
	if len(siniflar) == 0 {
		return ""
	}
	return " " + strings.Join(siniflar, " ")
}

// gazeteStilDeger ölçek gibi sayısal stilleri satır içi özel değişkene çevirir.
// Değer burada biçimlendirildiği için CSS'e ham metin geçmez.
func gazeteStilDeger(stil map[string]any) string {
	f, ok := gazeteSayiyaCevir(stil["olcek"])
	if !ok || f < 0.6 || f > 2.0 {
		return ""
	}
	return fmt.Sprintf("--gb-olcek:%.2f;", f)
}
