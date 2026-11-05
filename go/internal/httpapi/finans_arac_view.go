package httpapi

import (
	"sort"
	"strings"

	"theblock/site/internal/repo"
)

// Kripto Para ve Kur Çevirici modülleri.
//
// Referans: haberturk.com/ekonomi
//   KRİPTO PARA : başlık + TÜMÜ + ikon/ad/fiyat/%24s ızgarası
//   KUR ÇEVİRİCİ: iki para birimi seçici + tutar + DÖNÜŞTÜR + sonuç
//
// İkisi de önbellekten beslenir (istek yolunda ağa çıkılmaz) ve görünüm
// sözlüğü diğer finans modülleriyle aynıdır.

const (
	kriptoSayiVarsayilan        = 12
	kriptoSayiAzami             = 100
	kriptoKolonVarsayilan       = 4
	kriptoKolonEnAz             = 2
	kriptoKolonEnCok            = 6
	kriptoSiralaVarsayilan      = "deger"
	kurCeviriciBaslikVarsayilan = "KUR ÇEVİRİCİ"
	kurCeviriciMiktarVarsayilan = 100

	// Kripto modülü yerleşim/görünüm/geçiş — Hisseler ile aynı sözlük.
	kriptoYerlesimVarsayilan = "izgara"
	kriptoStilVarsayilan     = "otomatik"
	kriptoGecisVarsayilan    = "surekli"
	kriptoHizVarsayilan      = 60 // px/sn
)

// gecerliKriptoYerlesim kripto modülü yerleşimleri.
//
//	izgara — referans: sabit ızgara (Habertürk ekonomi vitrini)
//	serit  — kayan şerit (Hisseler ile aynı hareket motoru)
var gecerliKriptoYerlesim = map[string]bool{
	"izgara": true,
	"serit":  true,
}

// gecerliKriptoStil görünüm şablonları — Kur & Borsa / Hisseler ile aynı.
var gecerliKriptoStil = map[string]bool{
	"otomatik": true, // bölgeye göre: geniş → referans, yan sütun → rail
	"referans": true, // çerçeveli kart, fiyat yön renginde
	"sade":     true, // çerçevesiz, nötr fiyat
	"kart":     true, // belirgin kart (gölge/çerçeve)
	"dar":      true, // tek satır kompakt
	"rail":     true, // yan sütun listesi: ikon + kod + fiyat + %24s (dikey)
	"mini":     true, // mini şerit: tek satır, çok sıkı
}

// kriptoYerlesimOf yerleşimi okur (eski kayıtlarda yok → izgara).
func kriptoYerlesimOf(s string) string {
	if gecerliKriptoYerlesim[s] {
		return s
	}
	return kriptoYerlesimVarsayilan
}

// kriptoStilOf görünüm şablonunu okur.
//
// "otomatik" bölgeye göre çözülür (bkz. kriptoEtkinStil).
func kriptoStilOf(s string) string {
	if gecerliKriptoStil[s] {
		return s
	}
	return kriptoStilVarsayilan
}

// kriptoEtkinStil "otomatik" seçimini bölgeye göre çözer.
//
//	geniş bölge (ust/ana/genis/govde) → referans (vitrin ızgarası)
//	dar bölge  (yan)                  → rail (yan sütun listesi)
//
// Yan sütun ~300-380 px: 4 sütunlu kart ızgarası orada okunmaz, bu yüzden
// otomatik seçimde modül kendiliğinden rail görünümüne geçer.
func kriptoEtkinStil(secili string, dar bool) string {
	if secili == "otomatik" {
		if dar {
			return "rail"
		}
		// Geniş bölgede vitrin şablonu (varsayılanın kendisi \"otomatik\"
		// olduğu için burada sabit ad kullanılır; yoksa kendine dönerdi).
		return "referans"
	}
	return secili
}

// kriptoListeStili rail/mini gibi liste tabanlı şablonlar mı?
func kriptoListeStili(stil string) bool {
	return stil == "rail" || stil == "mini"
}

// kriptoGecisOf akış modelini okur (sürekli | adım).
func kriptoGecisOf(s string) string {
	if gecerliHisseGecis[s] {
		return s
	}
	return kriptoGecisVarsayilan
}

// kriptoBgOf kripto modülünün arka plan seçimini okur.
// Sözlük: kart (modül yüzeyi) | yok (saydam) | zemin | ozel.
func kriptoBgOf(s string) string {
	if gecerliKbBg[s] {
		return s
	}
	return "kart"
}

// kurCeviriciBgOf çevirici modülünün arka plan seçimini okur.
func kurCeviriciBgOf(s string) string {
	if gecerliKbBg[s] {
		return s
	}
	return "kart"
}

// servisBgOf servis bandının arka plan seçimini okur.
func servisBgOf(s string) string {
	if gecerliKbBg[s] {
		return s
	}
	return "kart"
}

// gecerliKriptoSirala kripto sıralamaları.
var gecerliKriptoSirala = map[string]bool{
	"deger":    true, // piyasa değeri sırası (CoinGecko sırası, referans)
	"yukselen": true, // en çok yükselen
	"dusen":    true, // en çok düşen
	"kod":      true, // alfabetik
}

// kriptoSiralaOf sıralamayı okur.
func kriptoSiralaOf(s string) string {
	if gecerliKriptoSirala[s] {
		return s
	}
	return kriptoSiralaVarsayilan
}

// kriptoSatirlar önbellek satırlarını sıralayıp kırpar (girdi değişmez).
func kriptoSatirlar(rows []repo.KriptoRow, sirala string, count int) []repo.KriptoRow {
	if len(rows) == 0 {
		return nil
	}
	out := make([]repo.KriptoRow, len(rows))
	copy(out, rows)
	switch kriptoSiralaOf(sirala) {
	case "yukselen":
		sort.SliceStable(out, func(i, j int) bool { return out[i].Delta > out[j].Delta })
	case "dusen":
		sort.SliceStable(out, func(i, j int) bool { return out[i].Delta < out[j].Delta })
	case "kod":
		sort.SliceStable(out, func(i, j int) bool { return out[i].Symbol < out[j].Symbol })
	}
	if count > 0 && len(out) > count {
		out = out[:count]
	}
	return out
}

// ---- Kur Çevirici ----

// kurPara çevirici listesindeki tek para birimi.
type kurPara struct {
	Code  string
	Label string
	Rate  float64 // 1 birim = X TRY
}

// kurCeviriciView modülün şablon verisi.
type kurCeviriciView struct {
	Baslik string
	Para   []kurPara
	From   string
	To     string
	Miktar float64
	// Sonuc sunucuda da hesaplanır: JS yoksa/gecikirse boş kutu görünmesin.
	Sonuc float64
	Var   bool
}

// kurCeviriciOlustur tablodan modül görünümünü üretir.
//
// Seçili para birimleri tabloda yoksa (kur güncellenmediyse) ilk geçerli
// çift kullanılır; modül asla boş seçimle basılmaz.
func kurCeviriciOlustur(kt repo.KurTablosu, baslik, from, to string, miktar float64) *kurCeviriciView {
	codes := kt.Kodlar()
	if len(codes) < 2 {
		return nil
	}
	if strings.TrimSpace(baslik) == "" {
		baslik = kurCeviriciBaslikVarsayilan
	}
	v := &kurCeviriciView{Baslik: baslik, Miktar: miktar}
	if v.Miktar <= 0 {
		v.Miktar = kurCeviriciMiktarVarsayilan
	}
	for _, c := range codes {
		v.Para = append(v.Para, kurPara{Code: c, Label: kt.Ad(c), Rate: kt.Kurlar[c]})
	}
	has := func(c string) bool {
		_, ok := kt.Kurlar[strings.ToUpper(c)]
		return ok
	}
	from = strings.ToUpper(strings.TrimSpace(from))
	to = strings.ToUpper(strings.TrimSpace(to))
	if !has(from) {
		from = codes[0]
	}
	if !has(to) || to == from {
		// TRY varsa hedef olarak onu seç (referans davranışı); yoksa ilk farklı kod.
		if has("TRY") && from != "TRY" {
			to = "TRY"
		} else {
			to = codes[0]
			if to == from {
				to = codes[1]
			}
		}
	}
	v.From, v.To = from, to
	if sonuc, ok := kt.Cevir(v.Miktar, from, to); ok {
		v.Sonuc, v.Var = sonuc, true
	}
	return v
}

// kriptoSayfaStilleri kripto sayfasının görünümleri.
//
// Sayfada üç görünüm seçilebilir (Habertürk tabloyu kullanır):
//
//	tablo   — KRIPTO · 24s · 7g · FİYAT · 24s HACİM · PİYASA DEĞERİ (referans)
//	kart    — ikon + ad + fiyat kartları
//	kompakt — tek satır, yalnız kod + fiyat + 24s
var kriptoSayfaStilleri = []struct{ Key, Label string }{
	{"tablo", "Tablo"},
	{"kart", "Kart"},
	{"kompakt", "Kompakt"},
}

// gecerliKriptoSayfaStil sayfa görünümü doğrulaması.
func gecerliKriptoSayfaStil(s string) bool {
	for _, v := range kriptoSayfaStilleri {
		if v.Key == s {
			return true
		}
	}
	return false
}

// kriptoSayfa /borsa/kripto sayfası görünümü.
type kriptoSayfa struct {
	Rows       []repo.KriptoRow
	Guncel     string
	Toplam     int
	Adet       int
	Q          string // arama metni
	Stil       string // tablo | kart | kompakt
	Stiller    []struct{ Key, Label string }
	SeciliStil string // seçili görünümün etiketi
}

// kriptoSayfaOlustur önbellek satırlarını arama ile süzer.
//
// Arama kodu, adı ve slug'ı birlikte tarar; Türkçe duyarsızdır
// (hisseAramaNorm aynı sadeleştirmeyi yapar). Eşleşme yoksa boş liste
// döner; sayfa "sonuç yok" durumunu basar.
func kriptoSayfaOlustur(kd repo.KriptoData, q, stil string) *kriptoSayfa {
	if !gecerliKriptoSayfaStil(stil) {
		stil = "tablo"
	}
	v := &kriptoSayfa{
		Q: strings.TrimSpace(q), Stil: stil, Stiller: kriptoSayfaStilleri,
		Toplam: len(kd.Rows),
	}
	if !kd.Updated.IsZero() {
		v.Guncel = kd.Updated.Format("02.01.2006 15:04")
	}
	for _, s := range kriptoSayfaStilleri {
		if s.Key == stil {
			v.SeciliStil = s.Label
		}
	}
	qn := hisseAramaNorm(q)
	for _, r := range kd.Rows {
		if qn != "" {
			hay := hisseAramaNorm(r.Symbol + " " + r.Name + " " + r.Slug)
			if !strings.Contains(hay, qn) {
				continue
			}
		}
		v.Rows = append(v.Rows, r)
	}
	v.Adet = len(v.Rows)
	return v
}
