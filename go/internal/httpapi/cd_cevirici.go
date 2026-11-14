package httpapi

import (
	"context"
	"encoding/json"
	"html"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"theblock/site/internal/repo"
	"theblock/site/internal/webui/templkit"
)

// /cevirici — Kripto ve Döviz Çevirici.
//
// NEDEN YENİ API UÇ NOKTASI YOK: Ghost teması bu sayfayı
// `/api/simple-price` çağrısıyla besliyordu. Bizde iki veri kaynağı
// ZATEN var ve ikisi de arka planda tazeleniyor:
//   repo.Fiyatlar  — kripto fiyatları (USD)
//   repo.KurlarOf  — döviz kurları (TRY tabanlı)
// Kurları sayfaya GÖMMEK hem bir ağ gidiş-dönüşünü hem de yeni bir
// genel API'yi ortadan kaldırıyor.
//
// JS KAPALIYKEN DE ÇALIŞIR: form GET gönderiyor, sonuç sunucuda
// hesaplanıyor ve adres paylaşılabilir oluyor
// (/cevirici?kaynak=BTC&hedef=TRY&miktar=2). Gömülü kurlar yalnız
// istemci tarafı anında yeniden hesaplasın diye.

const (
	ceviriciVarsayilanKaynak = "BTC"
	ceviriciVarsayilanHedef  = "TRY"
	ceviriciVarsayilanMiktar = 1.0
	// ceviriciKriptoLimiti açılır listedeki en fazla kripto. Sıralama
	// piyasa değerine göre; kuyrukta kimsenin aramadığı yüzlerce jeton
	// listeyi kullanılmaz yapardı.
	ceviriciKriptoLimiti = 40
	// ceviriciMaksMiktar üstü anlamsız; girdi sessizce varsayılana düşer.
	ceviriciMaksMiktar = 1e12
)

// CeviriciBirim açılır listedeki tek seçenek.
type CeviriciBirim struct {
	Kod string  `json:"kod"`
	Ad  string  `json:"ad"`
	USD float64 `json:"usd"` // 1 birim kaç USD eder
	// Kripto true ise listede ayrı grupta gösterilir.
	Kripto bool `json:"kripto"`
}

func (s *Server) handleCeviriciPage(w http.ResponseWriter, r *http.Request) {
	birimler := s.ceviriciBirimleri(r.Context())

	q := r.URL.Query()
	kaynak := ceviriciSec(q.Get("kaynak"), ceviriciVarsayilanKaynak, birimler)
	hedef := ceviriciSec(q.Get("hedef"), ceviriciVarsayilanHedef, birimler)
	miktar := ceviriciMiktar(q.Get("miktar"))

	kok := s.kanonikKok(r)
	v := s.cdSayfaKabuk(r, "cevirici",
		"Kripto Para ve Döviz Çevirici — "+s.siteAdi(),
		"Bitcoin, Ethereum ve diğer kripto paraları TL, dolar ve euroya anlık piyasa kuruyla çevirin.",
		"Kripto ve Döviz Çevirici",
		"Kripto paraları ve dövizleri birbirine çevirin; kurlar piyasa verisinden gelir.",
		kok+"/cevirici")
	v.Govde = templkit.SafeHTML(ceviriciGovde(birimler, kaynak, hedef, miktar))
	v.Serit = true
	s.cdBasitRender(w, r, v)
}

// ceviriciBirimleri kripto ve dövizi TEK TABANDA (USD) birleştirir.
//
// Kripto fiyatları zaten USD; döviz tablosu TRY tabanlı ve "1 USD kaç
// TRY" değeri üzerinden çevriliyor. İkisini tek tabanda buluşturmak
// çapraz kurda yuvarlama hatası biriktirmiyor.
func (s *Server) ceviriciBirimleri(ctx context.Context) []CeviriciBirim {
	var kripto []CeviriciBirim
	for _, f := range repo.Fiyatlar(ctx, s.pool) {
		if !f.HasData || f.Price <= 0 || f.Sym == "" {
			continue
		}
		kripto = append(kripto, CeviriciBirim{
			Kod: strings.ToUpper(f.Sym), Ad: f.Name, USD: f.Price, Kripto: true,
		})
		if len(kripto) >= ceviriciKriptoLimiti {
			break
		}
	}

	var doviz []CeviriciBirim
	kt := repo.KurlarOf(ctx, s.pool)
	// Tablo TRY tabanlı: kt.Kurlar["USD"] = 1 USD kaç TRY.
	usdTry := kt.Kurlar["USD"]
	if usdTry > 0 {
		for kod, tryDegeri := range kt.Kurlar {
			if tryDegeri <= 0 {
				continue
			}
			doviz = append(doviz, CeviriciBirim{
				Kod: strings.ToUpper(kod),
				Ad:  kt.Adlar[kod],
				// 1 <kod> = tryDegeri TRY; 1 USD = usdTry TRY.
				USD: tryDegeri / usdTry,
			})
		}
	}
	return ceviriciSirala(kripto, doviz)
}

// ceviriciSirala kripto piyasa değeri sırasını korur, dövizi
// alfabetik dizer.
func ceviriciSirala(kripto, doviz []CeviriciBirim) []CeviriciBirim {
	sort.SliceStable(doviz, func(i, j int) bool { return doviz[i].Kod < doviz[j].Kod })
	out := make([]CeviriciBirim, 0, len(kripto)+len(doviz))
	out = append(out, kripto...)
	return append(out, doviz...)
}

// ceviriciSec geçerli bir birim kodu seçer; tanınmayan istek sessizce
// varsayılana düşer.
func ceviriciSec(istek, varsayilan string, birimler []CeviriciBirim) string {
	istek = strings.ToUpper(strings.TrimSpace(istek))
	for _, b := range birimler {
		if b.Kod == istek {
			return istek
		}
	}
	return varsayilan
}

// ceviriciMiktar girilen miktarı güvenli aralığa çeker.
//
// Virgül de kabul ediliyor: Türkçe klavyede ondalık ayırıcı virgül ve
// "0,5" yazan kullanıcıya hata göstermek gereksiz sürtünme.
func ceviriciMiktar(ham string) float64 {
	ham = strings.TrimSpace(strings.ReplaceAll(ham, ",", "."))
	n, err := strconv.ParseFloat(ham, 64)
	if err != nil || n <= 0 || n > ceviriciMaksMiktar {
		return ceviriciVarsayilanMiktar
	}
	return n
}

// ceviriciCevir kaynak biriminden hedefe çevirir (ikisi de USD üzerinden).
func ceviriciCevir(miktar float64, kaynak, hedef CeviriciBirim) float64 {
	if hedef.USD <= 0 {
		return 0
	}
	return miktar * kaynak.USD / hedef.USD
}

func ceviriciBul(kod string, birimler []CeviriciBirim) (CeviriciBirim, bool) {
	for _, b := range birimler {
		if b.Kod == kod {
			return b, true
		}
	}
	return CeviriciBirim{}, false
}

// ceviriciSayi sonucu okunur biçimde yazar.
//
// Ondalık basamak sayısı BÜYÜKLÜĞE göre değişiyor: 0,00001234 BTC'yi
// iki basamakla göstermek "0,00" eder ve aracı işe yaramaz kılar.
func ceviriciSayi(n float64) string {
	switch {
	case n == 0:
		return "0"
	case n < 0.0001:
		return strconv.FormatFloat(n, 'f', 8, 64)
	case n < 1:
		return strconv.FormatFloat(n, 'f', 6, 64)
	case n < 1000:
		return strconv.FormatFloat(n, 'f', 4, 64)
	default:
		return strconv.FormatFloat(n, 'f', 2, 64)
	}
}

func ceviriciGovde(birimler []CeviriciBirim, kaynak, hedef string, miktar float64) string {
	var b strings.Builder
	b.WriteString(`<p class="cd-card__spot">Kripto paraları ve dövizleri birbirine çevirin. ` +
		`Kurlar piyasa verisinden gelir ve gecikmeli olabilir; bilgilendirme amaçlıdır, ` +
		`yatırım tavsiyesi değildir.</p>`)

	if len(birimler) == 0 {
		b.WriteString(`<p class="cd-cevirici__sonuc">Kur verisi şu an alınamadı. ` +
			`Birkaç dakika sonra tekrar deneyin.</p>`)
		return b.String()
	}

	b.WriteString(`<form class="cd-cevirici" method="get" action="/cevirici" data-cevirici>`)
	b.WriteString(`<input class="cd-cevirici__miktar" type="text" inputmode="decimal" name="miktar" ` +
		`value="` + html.EscapeString(ceviriciSayi(miktar)) + `" aria-label="Miktar" data-cevirici-miktar>`)
	b.WriteString(ceviriciSecim("kaynak", "Kaynak birim", kaynak, birimler))
	b.WriteString(`<span class="cd-cevirici__ok" aria-hidden="true">→</span>`)
	b.WriteString(ceviriciSecim("hedef", "Hedef birim", hedef, birimler))
	b.WriteString(`<button class="cd-cevirici__dugme" type="submit">Çevir</button>`)
	b.WriteString(`</form>`)

	kb, kVar := ceviriciBul(kaynak, birimler)
	hb, hVar := ceviriciBul(hedef, birimler)
	if kVar && hVar {
		sonuc := ceviriciCevir(miktar, kb, hb)
		b.WriteString(`<p class="cd-cevirici__sonuc" data-cevirici-sonuc>` +
			`<strong>` + html.EscapeString(ceviriciSayi(miktar)) + ` ` + html.EscapeString(kb.Kod) +
			`</strong> = <strong>` + html.EscapeString(ceviriciSayi(sonuc)) + ` ` +
			html.EscapeString(hb.Kod) + `</strong></p>`)
		// Birim kur ayrıca yazılıyor: okuyucu girdiği miktardan
		// bağımsız olarak kuru görmek istiyor.
		b.WriteString(`<p class="cd-cevirici__kur">1 ` + html.EscapeString(kb.Kod) + ` = ` +
			html.EscapeString(ceviriciSayi(ceviriciCevir(1, kb, hb))) + ` ` +
			html.EscapeString(hb.Kod) + `</p>`)
	}

	// Kurlar sayfaya gömülüyor: istemci tarafı yazarken anında
	// hesaplasın. JS kapalıysa yukarıdaki sunucu sonucu geçerli kalır.
	//
	// `<` kaçırılıyor: veri `<script>` içinde ve kaçırılmazsa bir birim
	// adı etiketi erken kapatabilirdi.
	if ham, err := json.Marshal(birimler); err == nil {
		b.WriteString(`<script type="application/json" data-cevirici-kurlar>` +
			strings.ReplaceAll(string(ham), "<", `<`) + `</script>`)
	}
	return b.String()
}

func ceviriciSecim(ad, etiket, secili string, birimler []CeviriciBirim) string {
	var b strings.Builder
	b.WriteString(`<select class="cd-cevirici__secim" name="` + html.EscapeString(ad) +
		`" aria-label="` + html.EscapeString(etiket) +
		`" data-cevirici-` + html.EscapeString(ad) + `>`)

	grup := func(baslik string, kripto bool) {
		var yazildi bool
		for _, x := range birimler {
			if x.Kripto != kripto {
				continue
			}
			if !yazildi {
				b.WriteString(`<optgroup label="` + baslik + `">`)
				yazildi = true
			}
			sec := ""
			if x.Kod == secili {
				sec = ` selected`
			}
			b.WriteString(`<option value="` + html.EscapeString(x.Kod) + `"` + sec + `>` +
				html.EscapeString(x.Kod))
			if x.Ad != "" {
				b.WriteString(` — ` + html.EscapeString(x.Ad))
			}
			b.WriteString(`</option>`)
		}
		if yazildi {
			b.WriteString(`</optgroup>`)
		}
	}
	grup("Kripto", true)
	grup("Döviz", false)
	b.WriteString(`</select>`)
	return b.String()
}
