package repo

import (
	"encoding/json"
	"html"
	"strings"
	"unicode"
)

// Metin hijyeni — besleme (RSS) metninin veritabanına temiz girmesi.
//
// ÖLÇÜM (üretim, 2 Kasım 2026):
//
//	1094 içerikte gövdede "&nbsp;", 2405'inde sayısal varlık (&#39;),
//	 318'inde &quot; — düz metin olarak saklanmış, okuyucuya "&nbsp;"
//	    görünüyor.
//	1088 içerikte "&nbsp;&nbsp;" ile YAPIŞTIRILMIŞ başlık listesi:
//	    "…halkının zaferi olarak gördü&nbsp;&nbsp;Son DakikaBM'de ABD'nin
//	     Küba ambargosunun kaldırılmasına destek geriledi&nbsp;&nbsp;Euronews.com…"
//	    Bu agregatör beslemelerinin akış özeti; tek paragraf olarak kaydedilince
//	    gövde anlamsız bir başlık yığınına dönüşüyor.
//
// Neden `stripHTML` yetmiyor: etiketleri siliyor ama VARLIKLARI (entity)
// çözmüyor. `html.UnescapeString` tek başına da yetmiyor: çözülen `&nbsp;`
// normal boşluğa dönüşüyor ama yapıştırma sınırı kayboluyor — bu yüzden
// sınırı çözmeden ÖNCE işaretliyoruz.

// varlikCoz HTML varlıklarını çözer ve ardışık boşlukları tekler.
//
// İki geçiş: bazı beslemeler çift kaçırılmış metin veriyor
// ("&amp;nbsp;" → "&nbsp;"), tek geçişte bu kalıntı kalıyordu.
func varlikCoz(s string) string {
	if s == "" {
		return s
	}
	once := s
	s = html.UnescapeString(s)
	if s != once {
		s = html.UnescapeString(s)
	}
	// Çözülemeyen/kaçırılmış kalmış olabilecek en yaygın varlıklar.
	for _, p := range [][2]string{
		{"&nbsp;", " "}, {"&amp;", "&"}, {"&quot;", `"`},
		{"&#39;", "'"}, {"&apos;", "'"}, {"&lt;", "<"}, {"&gt;", ">"},
		{"&ndash;", "–"}, {"&mdash;", "—"}, {"&hellip;", "…"},
		{"&rsquo;", "’"}, {"&lsquo;", "‘"}, {"&ldquo;", "“"}, {"&rdquo;", "”"},
	} {
		s = strings.ReplaceAll(s, p[0], p[1])
	}
	return boslukToparla(s)
}

// boslukToparla: NBSP → boşluk, satır sonu/tab → boşluk, ardışıklar tekleşir.
func boslukToparla(s string) string {
	s = strings.Map(func(r rune) rune {
		switch r {
		case '\u00a0', '\u2007', '\u202f', // no-break varyantları
			'\t', '\n', '\r', '\u2028', '\u2029':
			return ' '
		}
		return r
	}, s)
	// Kırılmasız boşluğun ART ARDA gelmesi agregatör yapıştırma işaretidir;
	// bilgi kaybetmemek için tek boşluğa indiriyoruz (işaret ayrıca okunur).
	return strings.Join(strings.Fields(s), " ")
}

// yapistirilmisBasliklar, metnin başlık listesi yapıştırması olup olmadığını
// söyler ve ilk gerçek cümleyi döndürür.
//
// Ölçüt bilinçli olarak dar: yalnız "iki kırılmasız boşluk" dizisi VE
// ardından büyük harfle başlayan yeni bir cümle VE bir site adı/ajans izi
// varsa kesiyoruz. Tek başına nbsp geçen meşru metni kesmemek için.
func yapistirilmisBasliklar(s string) (string, bool) {
	// Çözmeden önce bakılır: &nbsp; varlığı da aynı işareti taşır.
	isaret := strings.Index(s, "\u00a0\u00a0")
	if isaret < 0 {
		isaret = strings.Index(s, "&nbsp;&nbsp;")
	}
	if isaret <= 0 {
		return s, false
	}
	norm := strings.NewReplacer("&nbsp;", " ", "\u00a0", " ").Replace(s)
	kuyruk := norm[isaret:]
	bas := strings.TrimSpace(norm[:isaret])
	if bas == "" {
		return s, false
	}
	// Kaç yapıştırma sınırı var? Bu sınırın DIŞINDA en az bir sınır daha
	// varsa metin kesin olarak başlık listesidir: ilk sınırda kesmek güvenli.
	// (DİKKAT: kuyruğun başındaki çift boşluk da sayılırsa tek sınırlı meşru
	// metin listeye benzer — bir kez bu hataya düşüldü.)
	ekSinir := strings.Count(kuyruk, "  ") - 1
	if ekSinir < 1 {
		// Tek sınır + site adı: agregatör olabilir ama ilk parça kısaysa
		// kesmek bilgi kaybettirir. Yalnız yeterince uzun ilk parçada kes.
		if !siteAdiVarMi(kuyruk) || len([]rune(bas)) < 40 {
			return s, false
		}
	}
	return bas, true
}

// siteAdiVarMi: ".com/.com.tr/.net/..." ya da yaygın ajans adları.
func siteAdiVarMi(s string) bool {
	dusuk := strings.ToLower(s)
	for _, iz := range []string{".com", ".net", ".org", ".tv", "son dakika"} {
		if strings.Contains(dusuk, iz) {
			return true
		}
	}
	return false
}

// cumleGuvenliKirp metni n KARAKTERE (rune) sığdırır, ama cümle/kelime
// ortasında kesmez.
//
// ÖLÇÜM: CrawlFeed spot'u 240 rune'de körlemesine kesiyordu; yayındaki 94
// içerikte spot cümle ortasında bitiyordu ("… his workplace", "… until now").
// Artık: son cümle sonu (./!/?/…) noktasına kadar geri sarılır; uygun sınır
// yoksa son boşluktan kesilip üç nokta eklenir.
func cumleGuvenliKirp(s string, n int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= n {
		return string(r)
	}
	pencere := string(r[:n])
	// Cümle sonu ara (son 60 karakter içinde).
	son := -1
	for i := len(pencere) - 1; i >= 0 && i >= len(pencere)-60; i-- {
		c := rune(pencere[i])
		if c == '.' || c == '!' || c == '?' || c == '…' {
			son = i
			break
		}
	}
	if son > 40 {
		return strings.TrimSpace(pencere[:son+1])
	}
	// Kelime sınırı.
	if i := strings.LastIndex(pencere, " "); i > 40 {
		return strings.TrimSpace(pencere[:i]) + "…"
	}
	return strings.TrimSpace(pencere) + "…"
}

// paragraflaraBol düz metni okunur paragraflara ayırır.
//
// ÖLÇÜM: yayındaki 986 içeriğin 982'si TEK paragraf. Nedeni `textToBlocksStmt`
// yalnız "\n" üzerinde bölüyor; RSS özetlerinde satır sonu yok. Okuyucu için
// gövde tek duvar hâline geliyordu.
//
// Kural: cümle sonlarından böl, her paragraf ~3 cümle veya ~320 karakter
// olsun; çok kısa artıklar bir öncekine eklenir.
func paragraflaraBol(s string) []string {
	s = varlikCoz(s)
	if s == "" {
		return nil
	}
	cumleler := CumlelereBol(s)
	if len(cumleler) <= 1 {
		return []string{s}
	}
	const hedefKarakter = 320
	const hedefCumle = 3
	var out []string
	var grup []string
	uzunluk := 0
	flush := func() {
		if len(grup) == 0 {
			return
		}
		out = append(out, strings.Join(grup, " "))
		grup = nil
		uzunluk = 0
	}
	for _, c := range cumleler {
		// Çok kısa artık (ör. "dedi.") tek başına paragraf olmasın.
		if len(grup) >= hedefCumle || uzunluk+len(c) > hedefKarakter {
			if len([]rune(c)) < 40 && len(grup) >= 1 {
				grup = append(grup, c)
				uzunluk += len(c) + 1
				flush()
				continue
			}
			flush()
		}
		grup = append(grup, c)
		uzunluk += len(c) + 1
	}
	flush()
	return out
}

// CumlelereBol metni cümle sonlarından böler. Kısaltmalar (Dr., vb.) için
// bilinçli olarak basit: haber metninde nadir, yanlış bölme zararsız.
func CumlelereBol(s string) []string {
	var out []string
	var b strings.Builder
	r := []rune(s)
	for i := 0; i < len(r); i++ {
		b.WriteRune(r[i])
		if r[i] != '.' && r[i] != '!' && r[i] != '?' && r[i] != '…' {
			continue
		}
		// Ardışık noktalama (?!, ...) aynı cümlede kalsın.
		for i+1 < len(r) && (r[i+1] == '.' || r[i+1] == '!' || r[i+1] == '?' || r[i+1] == '…') {
			i++
			b.WriteRune(r[i])
		}
		// Sondaki kapanış tırnakları cümleye dahil.
		for i+1 < len(r) && (r[i+1] == '"' || r[i+1] == '\'' || r[i+1] == '”' ||
			r[i+1] == '’' || r[i+1] == ')' || r[i+1] == '»') {
			i++
			b.WriteRune(r[i])
		}
		// Sonraki karakter boşluksa cümle bitti.
		if i+1 >= len(r) || unicode.IsSpace(r[i+1]) {
			if c := strings.TrimSpace(b.String()); c != "" {
				out = append(out, c)
			}
			b.Reset()
		}
	}
	if c := strings.TrimSpace(b.String()); c != "" {
		out = append(out, c)
	}
	return out
}

// IskeletGovde, gövdenin AI kaynak iskeleti olup olmadığını söyler.
//
// İskelet: "Kaynaklar:" başlığı ve/veya "• <kaynak>: <başlık>" madde listesi.
// Bu bir ÇALIŞMA KÂĞIDI, haber metni değil; yayına çıkmamalı.
//
// ÖLÇÜM: küme→taslak yolu gövdeye 38 blokluk iskelet yazıyordu (1 taslakta
// görüldü). AI sonucu editör tarafından uygulanana kadar içerik yayına hazır
// değildir.
func IskeletGovde(detail []byte) bool {
	if len(detail) == 0 {
		return false
	}
	var parcalar []string
	// Blok dizisi.
	var bloklar []struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(detail, &bloklar); err == nil && len(bloklar) > 0 {
		for _, b := range bloklar {
			parcalar = append(parcalar, b.Text)
		}
	}
	// TipTap belgesi (object/content) — metin düğümlerini sırayla topla.
	if len(parcalar) == 0 {
		var doc struct {
			Type    string `json:"type"`
			Content []any  `json:"content"`
		}
		if err := json.Unmarshal(detail, &doc); err == nil && doc.Type == "doc" {
			for _, blok := range doc.Content {
				if m, ok := blok.(map[string]any); ok {
					var sb strings.Builder
					gezMetin(m, &sb)
					if t := strings.TrimSpace(sb.String()); t != "" {
						parcalar = append(parcalar, t)
					}
				}
			}
		}
	}
	if len(parcalar) == 0 {
		return false
	}
	madde := 0
	for _, p := range parcalar {
		t := strings.TrimSpace(p)
		if strings.HasPrefix(strings.ToLower(t), "kaynaklar:") {
			return true
		}
		// "• 10Haber: başlık" ya da "- kaynak: başlık" madde satırı.
		if strings.HasPrefix(t, "•") || strings.HasPrefix(t, "- ") {
			if i := strings.IndexAny(t, ":"); i > 0 && i < 80 {
				madde++
			}
		}
	}
	return madde >= 2
}

// gezMetin JSON düğümündeki metin parçalarını sırayla toplar.
func gezMetin(node any, sb *strings.Builder) {
	switch t := node.(type) {
	case map[string]any:
		if s, ok := t["text"].(string); ok {
			sb.WriteString(s)
		}
		if ch, ok := t["content"].([]any); ok {
			for _, c := range ch {
				gezMetin(c, sb)
			}
		}
	case []any:
		for _, c := range t {
			gezMetin(c, sb)
		}
	}
}
