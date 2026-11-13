package platform

import (
	"encoding/json"
	"strings"
)

// HTML -> TipTap doğrulama belgesi dönüşümü.
//
// NEDEN VAR: içerik gövdesi veritabanında TipTap "doc" JSON'u olarak
// duruyor (contents.detail). Dışarıdan HTML olarak gelen yazıyı düz
// metne indirmek YAPIYI ÖLDÜRÜR: alt başlıklar, listeler ve alıntılar
// kaybolur. Ölçüm (13 Kasım 2026): Ghost'tan taşınacak 50 yazıda 824
// adet h2 var — yazı başına ~16 alt başlık. Sitedeki 359 içerikten
// yalnız 4'ünde alt başlık olduğu düşünülürse taşınan asıl değer bu.
//
// NEDEN YENİ BAĞIMLILIK YOK: depo golang.org/x/net/html kullanmıyor ve
// her bağımlılık CVE taramasına giriyor. `richtext.go` zaten elle
// yazılmış bir HTML süzgeci taşıyor; aynı üslup sürdürüldü. Girdi
// keyfi kullanıcı HTML'i değil, bir CMS'in ürettiği düzenli biçim.

// TTNode TipTap belge düğümü.
type TTNode struct {
	Type    string         `json:"type"`
	Attrs   map[string]any `json:"attrs,omitempty"`
	Content []TTNode       `json:"content,omitempty"`
	Text    string         `json:"text,omitempty"`
	Marks   []TTMark       `json:"marks,omitempty"`
}

// TTMark metin üzerindeki biçim işareti (kalın, bağlantı...).
type TTMark struct {
	Type  string         `json:"type"`
	Attrs map[string]any `json:"attrs,omitempty"`
}

// ttBlokEtiket HTML etiketinden TipTap blok tipine eşleme.
var ttBlokEtiket = map[string]string{
	"p": "paragraph", "h2": "heading", "h3": "heading", "h4": "heading",
	"h5": "heading", "h6": "heading",
	"ul": "bulletList", "ol": "orderedList", "li": "listItem",
	"blockquote": "blockquote", "pre": "codeBlock", "hr": "horizontalRule",
}

// ttAtla içeriği TÜMÜYLE atılan etiketler. `script` ve `style` dış
// kaynaklı gövdede en tehlikeli ikisi; ölçümde 24 adet script bulundu.
var ttAtla = map[string]bool{"script": true, "style": true, "noscript": true, "iframe": true}

// ttSeffaf kendisi bir blok ÜRETMEYEN sarmalayıcılar: içindekiler
// doğrudan akışa katılır. Ghost kart yapısı bunlardan bol üretiyor
// (ölçümde 1348 div, 1216 span).
var ttSeffaf = map[string]bool{
	"div": true, "span": true, "section": true, "article": true,
	"header": true, "footer": true, "main": true, "figure": true,
}

// HTMLToTipTap HTML gövdesini TipTap "doc" JSON'una çevirir.
//
// Tanınmayan etiketler şeffaf sayılır: metni kaybetmemek, biçimi
// korumaktan önemlidir.
func HTMLToTipTap(in string) ([]byte, error) {
	bloklar := htmlBloklari(in)
	if len(bloklar) == 0 {
		bloklar = []TTNode{{Type: "paragraph"}}
	}
	return json.Marshal(TTNode{Type: "doc", Content: bloklar})
}

// htmlBloklari üst düzey blok listesini üretir.
func htmlBloklari(in string) []TTNode {
	p := &ttAyristirici{giris: in}
	return p.bloklar(nil)
}

type ttAyristirici struct {
	giris string
	i     int
}

// ttEtiket tek bir HTML etiketi.
type ttEtiket struct {
	ad     string
	kapan  bool
	tekil  bool
	nitele map[string]string
}

// sonraki bir sonraki metin parçasını veya etiketi döndürür.
// metin boşsa ve etiket yoksa ok=false.
func (p *ttAyristirici) sonraki() (metin string, et *ttEtiket, ok bool) {
	if p.i >= len(p.giris) {
		return "", nil, false
	}
	if p.giris[p.i] != '<' {
		j := strings.IndexByte(p.giris[p.i:], '<')
		if j < 0 {
			metin = p.giris[p.i:]
			p.i = len(p.giris)
		} else {
			metin = p.giris[p.i : p.i+j]
			p.i += j
		}
		return metin, nil, true
	}
	j := strings.IndexByte(p.giris[p.i:], '>')
	if j < 0 {
		p.i = len(p.giris)
		return "", nil, false
	}
	ham := p.giris[p.i : p.i+j+1]
	p.i += j + 1
	return "", ttEtiketCoz(ham), true
}

// ttEtiketCoz "<a href="x">" biçimini ayrıştırır.
func ttEtiketCoz(ham string) *ttEtiket {
	iç := strings.TrimSuffix(strings.TrimPrefix(ham, "<"), ">")
	e := &ttEtiket{nitele: map[string]string{}}
	if strings.HasSuffix(iç, "/") {
		e.tekil = true
		iç = strings.TrimSuffix(iç, "/")
	}
	if strings.HasPrefix(iç, "/") {
		e.kapan = true
		iç = iç[1:]
	}
	iç = strings.TrimSpace(iç)
	if iç == "" {
		return e
	}
	// Ad: ilk boşluğa kadar.
	ad := iç
	if k := strings.IndexAny(iç, " \t\n\r"); k >= 0 {
		ad = iç[:k]
		iç = iç[k:]
	} else {
		iç = ""
	}
	e.ad = strings.ToLower(ad)
	if e.ad == "br" || e.ad == "img" || e.ad == "hr" {
		e.tekil = true
	}
	ttNitelikCoz(iç, e.nitele)
	return e
}

// ttNitelikCoz ad="değer" çiftlerini toplar.
func ttNitelikCoz(s string, out map[string]string) {
	for {
		s = strings.TrimLeft(s, " \t\n\r")
		if s == "" {
			return
		}
		e := strings.IndexByte(s, '=')
		if e < 0 {
			return
		}
		ad := strings.ToLower(strings.TrimSpace(s[:e]))
		s = strings.TrimLeft(s[e+1:], " \t\n\r")
		if s == "" {
			return
		}
		var deger string
		if s[0] == '"' || s[0] == '\'' {
			tirnak := s[0]
			k := strings.IndexByte(s[1:], tirnak)
			if k < 0 {
				return
			}
			deger = s[1 : 1+k]
			s = s[k+2:]
		} else {
			k := strings.IndexAny(s, " \t\n\r")
			if k < 0 {
				deger, s = s, ""
			} else {
				deger, s = s[:k], s[k:]
			}
		}
		if ad != "" {
			out[ad] = ttVarlikCoz(deger)
		}
	}
}

// bloklar akışı blok düğümlerine böler. `dur` verilirse o kapanış
// etiketinde durur (liste ve alıntı gövdesi için).
func (p *ttAyristirici) bloklar(dur map[string]bool) []TTNode {
	var out []TTNode
	var akan []TTNode // henüz bir bloğa girmemiş satır içi metin
	var marklar []TTMark

	bosalt := func() {
		if len(akan) > 0 {
			out = append(out, TTNode{Type: "paragraph", Content: akan})
			akan = nil
		}
	}

	for {
		metin, et, ok := p.sonraki()
		if !ok {
			break
		}
		if et == nil {
			if t := ttVarlikCoz(metin); strings.TrimSpace(t) != "" {
				akan = append(akan, ttMetin(t, marklar))
			}
			continue
		}
		if dur != nil && et.kapan && dur[et.ad] {
			break
		}
		if ttAtla[et.ad] {
			if !et.kapan && !et.tekil {
				p.atla(et.ad)
			}
			continue
		}
		if et.kapan {
			if m := ttMarkAdi(et.ad); m != "" {
				marklar = ttMarkCikar(marklar, m)
			}
			continue
		}
		if m := ttMarkAdi(et.ad); m != "" && !et.tekil {
			marklar = append(marklar, ttMarkYap(m, et.nitele))
			continue
		}

		switch {
		case et.ad == "br":
			akan = append(akan, TTNode{Type: "hardBreak"})
		case et.ad == "hr":
			bosalt()
			out = append(out, TTNode{Type: "horizontalRule"})
		case et.ad == "img":
			bosalt()
			if n := ttGorsel(et.nitele); n != nil {
				out = append(out, *n)
			}
		case et.ad == "figcaption":
			// Resim altı yazısı: önceki görsele caption olarak bağlanır.
			ic := p.satirIci("figcaption")
			if s := strings.TrimSpace(ttDuzMetin(ic)); s != "" && len(out) > 0 {
				if out[len(out)-1].Type == "image" {
					if out[len(out)-1].Attrs == nil {
						out[len(out)-1].Attrs = map[string]any{}
					}
					out[len(out)-1].Attrs["caption"] = s
				}
			}
		case ttSeffaf[et.ad]:
			// sarmalayıcı: hiçbir şey yapma, içi akışa karışsın
		case ttBlokEtiket[et.ad] != "":
			bosalt()
			out = append(out, p.blok(et))
		default:
			// tanınmayan etiket: şeffaf say, metni koru
		}
	}
	bosalt()
	return out
}

// blok tek bir blok düğümünü (ve içeriğini) üretir.
func (p *ttAyristirici) blok(et *ttEtiket) TTNode {
	tip := ttBlokEtiket[et.ad]
	n := TTNode{Type: tip}

	switch tip {
	case "heading":
		// h5/h6 -> h4: çizici 2-4 arasını basıyor, fazlası kırpılıyor.
		seviye := int(et.ad[1] - '0')
		if seviye > 4 {
			seviye = 4
		}
		n.Attrs = map[string]any{"level": seviye}
		n.Content = p.satirIci(et.ad)
	case "paragraph":
		n.Content = p.satirIci(et.ad)
	case "codeBlock":
		n.Content = []TTNode{ttMetin(ttDuzMetin(p.satirIci(et.ad)), nil)}
	case "bulletList", "orderedList":
		n.Content = p.bloklar(map[string]bool{et.ad: true})
		n.Content = ttYalnizOge(n.Content)
	case "listItem":
		ic := p.bloklar(map[string]bool{"li": true, "ul": true, "ol": true})
		if len(ic) == 0 {
			ic = []TTNode{{Type: "paragraph"}}
		}
		n.Content = ic
	case "blockquote":
		ic := p.bloklar(map[string]bool{"blockquote": true})
		if len(ic) == 0 {
			ic = []TTNode{{Type: "paragraph"}}
		}
		n.Content = ic
	case "horizontalRule":
		// içerik yok
	}
	return n
}

// satirIci kapanış etiketine kadarki satır içi düğümleri toplar.
func (p *ttAyristirici) satirIci(kapanis string) []TTNode {
	var out []TTNode
	var marklar []TTMark
	for {
		metin, et, ok := p.sonraki()
		if !ok {
			break
		}
		if et == nil {
			if t := ttVarlikCoz(metin); t != "" {
				out = append(out, ttMetin(t, marklar))
			}
			continue
		}
		if et.kapan && et.ad == kapanis {
			break
		}
		if ttAtla[et.ad] {
			if !et.kapan && !et.tekil {
				p.atla(et.ad)
			}
			continue
		}
		if et.ad == "br" {
			out = append(out, TTNode{Type: "hardBreak"})
			continue
		}
		if m := ttMarkAdi(et.ad); m != "" {
			if et.kapan {
				marklar = ttMarkCikar(marklar, m)
			} else if !et.tekil {
				marklar = append(marklar, ttMarkYap(m, et.nitele))
			}
			continue
		}
		// Blok etiketi satır içinde kapanmamış demektir: burada bitir.
		if !et.kapan && ttBlokEtiket[et.ad] != "" {
			p.i -= 0 // geri sarmıyoruz: bozuk HTML'de sonsuz döngüyü önler
			break
		}
	}
	return ttTemizle(out)
}

// atla açılmış bir etiketin kapanışına kadar her şeyi yutar.
func (p *ttAyristirici) atla(ad string) {
	derinlik := 1
	for derinlik > 0 {
		_, et, ok := p.sonraki()
		if !ok {
			return
		}
		if et == nil {
			continue
		}
		if et.ad == ad {
			if et.kapan {
				derinlik--
			} else if !et.tekil {
				derinlik++
			}
		}
	}
}

// ---- yardımcılar ----

func ttMarkAdi(etiket string) string {
	switch etiket {
	case "strong", "b":
		return "bold"
	case "em", "i":
		return "italic"
	case "u":
		return "underline"
	case "s", "strike", "del":
		return "strike"
	case "code":
		return "code"
	case "a":
		return "link"
	}
	return ""
}

func ttMarkYap(tip string, nitele map[string]string) TTMark {
	if tip == "link" {
		href := safeHref(nitele["href"])
		if href == "" {
			// Güvensiz şema: bağlantıyı düşür ama metni koru.
			return TTMark{Type: "italic"}
		}
		return TTMark{Type: "link", Attrs: map[string]any{"href": href}}
	}
	return TTMark{Type: tip}
}

func ttMarkCikar(ms []TTMark, tip string) []TTMark {
	for i := len(ms) - 1; i >= 0; i-- {
		if ms[i].Type == tip || (tip == "link" && ms[i].Type == "italic") {
			return append(ms[:i:i], ms[i+1:]...)
		}
	}
	return ms
}

func ttMetin(s string, marklar []TTMark) TTNode {
	n := TTNode{Type: "text", Text: s}
	if len(marklar) > 0 {
		n.Marks = append([]TTMark(nil), marklar...)
	}
	return n
}

func ttGorsel(nitele map[string]string) *TTNode {
	src := strings.TrimSpace(nitele["src"])
	if src == "" {
		return nil
	}
	attrs := map[string]any{"src": src}
	if alt := strings.TrimSpace(nitele["alt"]); alt != "" {
		attrs["alt"] = alt
	}
	return &TTNode{Type: "image", Attrs: attrs}
}

// ttYalnizOge liste gövdesinde yalnız listItem bırakır: çizici
// <ul> içinde serbest paragraf beklemiyor.
func ttYalnizOge(in []TTNode) []TTNode {
	var out []TTNode
	for _, n := range in {
		if n.Type == "listItem" {
			out = append(out, n)
		}
	}
	return out
}

// ttTemizle baştaki/sondaki boş metin düğümlerini atar.
func ttTemizle(in []TTNode) []TTNode {
	var out []TTNode
	for _, n := range in {
		if n.Type == "text" && n.Text == "" {
			continue
		}
		out = append(out, n)
	}
	return out
}

// ttDuzMetin düğüm ağacını düz metne indirir.
func ttDuzMetin(ns []TTNode) string {
	var sb strings.Builder
	for _, n := range ns {
		if n.Type == "text" {
			sb.WriteString(n.Text)
		}
		if len(n.Content) > 0 {
			sb.WriteString(ttDuzMetin(n.Content))
		}
	}
	return sb.String()
}

// ttVarlik yaygın HTML varlıkları. Ölçümde gövdelerde &nbsp; ve
// &#8217; bol geçiyor; çözülmezse okuyucu ham kodu görür.
var ttVarlik = strings.NewReplacer(
	"&nbsp;", " ", "&amp;", "&", "&lt;", "<", "&gt;", ">",
	"&quot;", "\"", "&#39;", "'", "&apos;", "'",
	"&#8217;", "’", "&rsquo;", "’", "&#8216;", "‘",
	"&lsquo;", "‘", "&#8220;", "“", "&ldquo;", "“",
	"&#8221;", "”", "&rdquo;", "”", "&mdash;", "—",
	"&ndash;", "–", "&hellip;", "…", "&#8230;", "…",
)

func ttVarlikCoz(s string) string { return ttVarlik.Replace(s) }
