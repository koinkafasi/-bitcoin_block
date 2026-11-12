package httpapi

import (
	"html"
	"math"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"theblock/site/internal/repo"
)

// ---- Yazı içinde otomatik varlık rozeti ----
//
// REFERANS DAVRANIŞ (coindesk.com makale sayfası, ölçüldü 12 Kasım 2026):
// metinde adı geçen varlıklar, gövdenin içinde küçük bir hap olarak
// gösteriliyor — sembol + güncel fiyat + varlık sayfasına bağlantı
// (`COIN $136.23 · /price/stock/coin`). Okuyucu yazıdan çıkmadan fiyatı
// görüyor, site de her yazıdan varlık sayfalarına iç link dağıtıyor.
//
// BİZDEKİ KARŞILIĞI: fiyat verisi zaten var (CoinGecko meta + Binance
// canlı, `repo.Fiyatlar`), varlık sayfaları da var (`/price/{coin}`).
// Eksik olan tek şey bağdı.
//
// NEDEN RENDER ANINDA, EDİTÖRDE DEĞİL: 3.360 yayınlanmış yazı var.
// Editöre düğme koymak yalnız yeni yazıları kazandırırdı; render anında
// eşleme, arşivin tamamını aynı anda kazandırıyor. Fiyat da her zaman
// güncel kalıyor — gövdeye sabit sayı yazılmıyor.
//
// KURALLAR (spam olmaması için):
//   · Bir varlık yazıda YALNIZ BİR KEZ rozetlenir (ilk geçtiği yerde).
//   · En çok `rozetUstSinir` rozet basılır.
//   · Bağlantı içindeki metne dokunulmaz (iç içe <a> geçersiz HTML'dir).
//   · Başlıklarda rozet basılmaz; başlık tipografisini bozuyor.

const rozetUstSinir = 4

// cdVarlikEslesme metinde aranacak varlık.
type cdVarlikEslesme struct {
	Ad    string // "Bitcoin"
	Sym   string // "BTC"
	Slug  string // /price/bitcoin
	Fiyat string // "$74.512,30"
	Artis bool
	Yuzde string // "+1,24%"
}

// cdVarlikRozetle gövde HTML'inde geçen varlıkları rozetler.
//
// Basit bir tarayıcı kullanıyor: etiketlerin DIŞINDAKİ metin parçalarında
// arama yapıyor, etiketin içine hiç girmiyor. Tam HTML ağacı kurmak bu iş
// için gereksiz; gövde zaten bizim ürettiğimiz dar bir etiket kümesi
// (<p>, <h2>, <ul>, <a>, <strong>…).
func cdVarlikRozetle(govde string, varliklar []cdVarlikEslesme) string {
	if govde == "" || len(varliklar) == 0 {
		return govde
	}
	var sb strings.Builder
	sb.Grow(len(govde) + 512)

	kullanildi := map[string]bool{}
	basilan := 0
	linkDerinligi := 0
	baslikDerinligi := 0
	// Tablo hücresi dar: rozet oraya sığmayıp alt alta dağılıyordu
	// (ölçüldü). Tablolarda hiç basmıyoruz.
	tabloDerinligi := 0
	i := 0

	for i < len(govde) {
		if govde[i] == '<' {
			kapanis := strings.IndexByte(govde[i:], '>')
			if kapanis < 0 {
				sb.WriteString(govde[i:])
				break
			}
			etiket := govde[i : i+kapanis+1]
			ad := strings.ToLower(etiketAdi(etiket))
			switch ad {
			case "a":
				linkDerinligi++
			case "/a":
				if linkDerinligi > 0 {
					linkDerinligi--
				}
			case "h1", "h2", "h3", "h4":
				baslikDerinligi++
			case "/h1", "/h2", "/h3", "/h4":
				if baslikDerinligi > 0 {
					baslikDerinligi--
				}
			case "table":
				tabloDerinligi++
			case "/table":
				if tabloDerinligi > 0 {
					tabloDerinligi--
				}
			}
			sb.WriteString(etiket)
			i += kapanis + 1
			continue
		}

		sonraki := strings.IndexByte(govde[i:], '<')
		if sonraki < 0 {
			sonraki = len(govde) - i
		}
		parca := govde[i : i+sonraki]
		if linkDerinligi == 0 && baslikDerinligi == 0 && tabloDerinligi == 0 && basilan < rozetUstSinir {
			yeni, eklendi := rozetUygula(parca, varliklar, kullanildi, rozetUstSinir-basilan)
			basilan += eklendi
			sb.WriteString(yeni)
		} else {
			sb.WriteString(parca)
		}
		i += sonraki
	}
	return sb.String()
}

// etiketAdi "<p class=...>" → "p", "</p>" → "/p".
func etiketAdi(etiket string) string {
	s := strings.TrimPrefix(etiket, "<")
	s = strings.TrimSuffix(s, ">")
	s = strings.TrimSuffix(s, "/")
	if k := strings.IndexAny(s, " \t\n"); k >= 0 {
		s = s[:k]
	}
	return s
}

// rozetUygula tek bir metin parçasında eşleme yapar.
func rozetUygula(parca string, varliklar []cdVarlikEslesme, kullanildi map[string]bool, kalan int) (string, int) {
	if kalan <= 0 {
		return parca, 0
	}
	eklendi := 0
	for _, v := range varliklar {
		if eklendi >= kalan || kullanildi[v.Sym] {
			continue
		}
		yer := kelimeAra(parca, v.Ad)
		if yer < 0 {
			continue
		}
		kullanildi[v.Sym] = true
		eklendi++
		parca = parca[:yer+len(v.Ad)] + rozetHTML(v) + parca[yer+len(v.Ad):]
	}
	return parca, eklendi
}

// kelimeAra büyük/küçük harf duyarsız, KELİME SINIRINA saygılı arama.
// "Bitcoin" ararken "Bitcoin'in" eşleşir ama "Bitcoincash" eşleşmez.
//
// NEDEN ToLower KULLANILMIYOR: Türkçede küçültme BAYT UZUNLUĞUNU
// değiştirebiliyor — "İ" (2 bayt) küçülünce "i̇" (3 bayt) oluyor. Küçük
// harfli kopyada bulunan indeks, özgün metinde başka bir yere denk
// geliyordu ve rozet kelimenin ortasına giriyordu ("Ethereu<rozet>m",
// ölçüldü). Burada özgün metin üzerinde, aynı uzunlukta dilimler
// karşılaştırılıyor; indeks hiç kaymıyor.
func kelimeAra(metin, aranan string) int {
	if aranan == "" || len(aranan) > len(metin) {
		return -1
	}
	for i := 0; i+len(aranan) <= len(metin); i++ {
		if !strings.EqualFold(metin[i:i+len(aranan)], aranan) {
			continue
		}
		if oncekiRuneHarf(metin[:i]) {
			continue
		}
		if sonrakiRuneHarf(metin[i+len(aranan):]) {
			continue
		}
		return i
	}
	return -1
}

func oncekiRuneHarf(s string) bool {
	if s == "" {
		return false
	}
	r, _ := utf8.DecodeLastRuneInString(s)
	return harfMi(r)
}

func sonrakiRuneHarf(s string) bool {
	if s == "" {
		return false
	}
	r, _ := utf8.DecodeRuneInString(s)
	return harfMi(r)
}

func harfMi(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

// rozetHTML tek rozetin işaretlemesi. Fiyat GÖVDEYE YAZILMAZ, her
// render'da yeniden üretilir; eski yazıda eski fiyat kalmaz.
func rozetHTML(v cdVarlikEslesme) string {
	yon := "cd-varlik--dusus"
	if v.Artis {
		yon = "cd-varlik--yukselis"
	}
	return `<a class="cd-varlik ` + yon + `" href="/price/` + html.EscapeString(v.Slug) +
		`" title="` + html.EscapeString(v.Ad) + ` fiyat sayfası">` +
		`<span class="cd-varlik__sym">` + html.EscapeString(v.Sym) + `</span>` +
		`<span class="cd-varlik__fiyat">` + html.EscapeString(v.Fiyat) + `</span>` +
		`<span class="cd-varlik__degisim">` + html.EscapeString(v.Yuzde) + `</span></a>`
}

// cdVarlikListesi fiyat şeridinden eşleme listesi kurar.
//
// Yalnız ilk `sinir` varlık alınır: listenin kuyruğundaki düşük hacimli
// coinlerin adları günlük dilde geçen kelimelerle çakışabiliyor.
func (s *Server) cdVarlikListesi(satirlar []repo.FiyatSatiri, sinir int) []cdVarlikEslesme {
	out := make([]cdVarlikEslesme, 0, sinir)
	for i, sat := range satirlar {
		if i >= sinir {
			break
		}
		if !sat.HasData || sat.Price <= 0 || strings.TrimSpace(sat.Name) == "" {
			continue
		}
		isaret := "+"
		if sat.Change < 0 {
			isaret = "−"
		}
		out = append(out, cdVarlikEslesme{
			Ad:    sat.Name,
			Sym:   strings.ToUpper(sat.Sym),
			Slug:  varlikSlug(sat.Name),
			Fiyat: repo.FiyatBicim(sat.Price),
			Artis: sat.Change >= 0,
			Yuzde: isaret + strconv.FormatFloat(math.Abs(sat.Change), 'f', 2, 64) + "%",
		})
	}
	return out
}

// varlikSlug varlık adını URL parçasına çevirir ("Bitcoin" → "bitcoin").
func varlikSlug(ad string) string {
	s := strings.ToLower(strings.TrimSpace(ad))
	s = strings.NewReplacer(" ", "-", ".", "", "'", "", "’", "").Replace(s)
	return s
}
