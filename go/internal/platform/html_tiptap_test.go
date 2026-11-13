package platform

import (
	"encoding/json"
	"strings"
	"testing"
)

// coz dönüşümü çalıştırıp belgeyi geri okur.
func coz(t *testing.T, in string) TTNode {
	t.Helper()
	ham, err := HTMLToTipTap(in)
	if err != nil {
		t.Fatalf("dönüşüm hatası: %v", err)
	}
	var d TTNode
	if err := json.Unmarshal(ham, &d); err != nil {
		t.Fatalf("JSON okunamadı: %v\n%s", err, ham)
	}
	if d.Type != "doc" {
		t.Fatalf("kök 'doc' olmalı, %q geldi", d.Type)
	}
	return d
}

// tipler belgedeki blok tiplerini sırayla verir.
func tipler(d TTNode) []string {
	var out []string
	for _, n := range d.Content {
		out = append(out, n.Type)
	}
	return out
}

// ASIL KAZANÇ ALT BAŞLIKLAR. Ölçüm (13 Kasım 2026): taşınacak 50
// yazıda 824 adet h2 var, sitedeki 359 içerikten yalnız 4'ünde alt
// başlık var. Düz metne indirmek bu yapıyı öldürürdü.
func TestHTMLToTipTapBasliklariKorur(t *testing.T) {
	d := coz(t, `<p>Giriş</p><h2>Birinci Bölüm</h2><p>Metin</p><h3>Alt</h3><p>Son</p>`)
	istek := []string{"paragraph", "heading", "paragraph", "heading", "paragraph"}
	got := tipler(d)
	if strings.Join(got, ",") != strings.Join(istek, ",") {
		t.Fatalf("blok sırası = %v, beklenen %v", got, istek)
	}
	if lvl := d.Content[1].Attrs["level"]; lvl != float64(2) {
		t.Errorf("h2 seviyesi = %v", lvl)
	}
	if lvl := d.Content[3].Attrs["level"]; lvl != float64(3) {
		t.Errorf("h3 seviyesi = %v", lvl)
	}
	if ttDuzMetin(d.Content[1].Content) != "Birinci Bölüm" {
		t.Errorf("başlık metni = %q", ttDuzMetin(d.Content[1].Content))
	}
}

// h5/h6 kırpılmalı: çizici yalnız h2-h4 basıyor, fazlası sessizce
// h4'e düşürülüyor — dönüşüm de aynı sınırı uygulamalı.
func TestHTMLToTipTapBaslikSeviyesiKirpilir(t *testing.T) {
	d := coz(t, `<h5>Beş</h5><h6>Altı</h6>`)
	for i := range d.Content {
		if lvl := d.Content[i].Attrs["level"]; lvl != float64(4) {
			t.Errorf("%d. başlık seviyesi = %v, 4 beklenir", i, lvl)
		}
	}
}

func TestHTMLToTipTapListeler(t *testing.T) {
	d := coz(t, `<ul><li>Bir</li><li>İki</li></ul><ol><li>Tek</li></ol>`)
	if got := tipler(d); strings.Join(got, ",") != "bulletList,orderedList" {
		t.Fatalf("blok tipleri = %v", got)
	}
	ul := d.Content[0]
	if len(ul.Content) != 2 {
		t.Fatalf("ul 2 öğe bekliyordu, %d geldi", len(ul.Content))
	}
	for _, oge := range ul.Content {
		if oge.Type != "listItem" {
			t.Errorf("ul içinde %q var; çizici yalnız listItem bekliyor", oge.Type)
		}
	}
	if s := ttDuzMetin(ul.Content[1].Content); s != "İki" {
		t.Errorf("ikinci öğe = %q", s)
	}
}

func TestHTMLToTipTapAlintiVeCizgi(t *testing.T) {
	d := coz(t, `<blockquote><p>Alıntı</p></blockquote><hr><p>Son</p>`)
	if got := tipler(d); strings.Join(got, ",") != "blockquote,horizontalRule,paragraph" {
		t.Fatalf("blok tipleri = %v", got)
	}
	if s := ttDuzMetin(d.Content[0].Content); s != "Alıntı" {
		t.Errorf("alıntı metni = %q", s)
	}
}

// Biçim işaretleri metni kaybetmeden taşınmalı.
func TestHTMLToTipTapIsaretler(t *testing.T) {
	d := coz(t, `<p><strong>kalın</strong> ve <em>eğik</em> ve <a href="https://ornek.com/x">bağlantı</a></p>`)
	p := d.Content[0]
	var bulunan []string
	for _, n := range p.Content {
		for _, m := range n.Marks {
			bulunan = append(bulunan, m.Type)
			if m.Type == "link" && m.Attrs["href"] != "https://ornek.com/x" {
				t.Errorf("bağlantı adresi = %v", m.Attrs["href"])
			}
		}
	}
	for _, istek := range []string{"bold", "italic", "link"} {
		var var_ bool
		for _, b := range bulunan {
			if b == istek {
				var_ = true
			}
		}
		if !var_ {
			t.Errorf("%q işareti yok (bulunan: %v)", istek, bulunan)
		}
	}
	if s := ttDuzMetin(p.Content); !strings.Contains(s, "kalın") || !strings.Contains(s, "bağlantı") {
		t.Errorf("metin kaybolmuş: %q", s)
	}
}

// GÜVENLİK: javascript: şeması bağlantı olmamalı ama METİN KALMALI.
func TestHTMLToTipTapGuvensizBaglanti(t *testing.T) {
	d := coz(t, `<p><a href="javascript:alert(1)">tıkla</a></p>`)
	s := ttDuzMetin(d.Content[0].Content)
	if !strings.Contains(s, "tıkla") {
		t.Errorf("metin düşürülmüş: %q", s)
	}
	for _, n := range d.Content[0].Content {
		for _, m := range n.Marks {
			if m.Type == "link" {
				t.Errorf("güvensiz şema bağlantı olarak kalmış: %v", m.Attrs)
			}
		}
	}
}

// Ölçümde gövdelerde 24 adet <script> bulundu. İçeriğiyle birlikte
// tümüyle atılmalı.
func TestHTMLToTipTapScriptAtilir(t *testing.T) {
	d := coz(t, `<p>Önce</p><script>var kotu = 1; document.cookie;</script><p>Sonra</p>`)
	ham, _ := HTMLToTipTap(`<p>Önce</p><script>var kotu = 1;</script><p>Sonra</p>`)
	if strings.Contains(string(ham), "kotu") || strings.Contains(string(ham), "cookie") {
		t.Errorf("script gövdesi belgeye sızmış: %s", ham)
	}
	if got := tipler(d); strings.Join(got, ",") != "paragraph,paragraph" {
		t.Errorf("blok tipleri = %v", got)
	}
}

// Ghost kart yapısı bol div/span üretiyor (ölçüm: 1348 div, 1216
// span). Bunlar blok üretmemeli ama içlerindeki metin kaybolmamalı.
func TestHTMLToTipTapSarmalayicilarSeffaf(t *testing.T) {
	d := coz(t, `<div class="kart"><div><p>İçerik</p></div></div>`)
	if got := tipler(d); strings.Join(got, ",") != "paragraph" {
		t.Fatalf("sarmalayıcı blok üretmiş: %v", got)
	}
	if s := ttDuzMetin(d.Content[0].Content); s != "İçerik" {
		t.Errorf("metin = %q", s)
	}
}

// Etiketsiz serbest metin de kaybolmamalı: Ghost gövdelerinde
// sarmalayıcı dışına düşen parçalar oluyor.
func TestHTMLToTipTapSerbestMetin(t *testing.T) {
	d := coz(t, `Başlıksız metin<p>Paragraf</p>`)
	if len(d.Content) < 2 {
		t.Fatalf("serbest metin düşmüş: %v", tipler(d))
	}
	if s := ttDuzMetin(d.Content[0].Content); !strings.Contains(s, "Başlıksız") {
		t.Errorf("ilk blok = %q", s)
	}
}

func TestHTMLToTipTapGorselVeAltYazi(t *testing.T) {
	d := coz(t, `<figure><img src="https://cdn.ornek.com/a.png" alt="Alt metin"><figcaption>Açıklama</figcaption></figure>`)
	var gorsel *TTNode
	for i := range d.Content {
		if d.Content[i].Type == "image" {
			gorsel = &d.Content[i]
		}
	}
	if gorsel == nil {
		t.Fatalf("görsel düğümü yok: %v", tipler(d))
	}
	if gorsel.Attrs["src"] != "https://cdn.ornek.com/a.png" {
		t.Errorf("src = %v", gorsel.Attrs["src"])
	}
	if gorsel.Attrs["alt"] != "Alt metin" {
		t.Errorf("alt = %v", gorsel.Attrs["alt"])
	}
	if gorsel.Attrs["caption"] != "Açıklama" {
		t.Errorf("caption = %v", gorsel.Attrs["caption"])
	}
}

// HTML varlıkları çözülmeli; yoksa okuyucu "&nbsp;" görür (sitede bu
// tam olarak 3.628 içerikte yaşandı, bkz. yapi.go).
func TestHTMLToTipTapVarliklar(t *testing.T) {
	d := coz(t, `<p>Bir&nbsp;iki &amp; üç &#8217;dört&#8217; &hellip;</p>`)
	s := ttDuzMetin(d.Content[0].Content)
	for _, kotu := range []string{"&nbsp;", "&amp;", "&#8217;", "&hellip;"} {
		if strings.Contains(s, kotu) {
			t.Errorf("%q çözülmemiş: %q", kotu, s)
		}
	}
	if !strings.Contains(s, "&") || !strings.Contains(s, "’") {
		t.Errorf("çözülmüş metin beklenenden farklı: %q", s)
	}
}

// Boş girdi geçerli bir belge üretmeli: çizici nil içerikte patlamasın.
func TestHTMLToTipTapBosGirdi(t *testing.T) {
	for _, in := range []string{"", "   ", "<div></div>"} {
		d := coz(t, in)
		if len(d.Content) == 0 {
			t.Errorf("%q için boş belge üretildi", in)
		}
	}
}

// Kapanmamış etiket sonsuz döngüye veya veri kaybına yol açmamalı.
func TestHTMLToTipTapBozukHTML(t *testing.T) {
	d := coz(t, `<p>Açık<div><strong>yarım<h2>Başlık`)
	s := ttDuzMetin(TTNode{Content: d.Content}.Content)
	for _, k := range []string{"Açık", "yarım", "Başlık"} {
		if !strings.Contains(sTumu(d), k) {
			t.Errorf("%q kaybolmuş (metin: %q)", k, s)
		}
	}
}

// sTumu belgedeki tüm metni toplar (iç içe bloklar dahil).
func sTumu(d TTNode) string {
	var sb strings.Builder
	var gez func(ns []TTNode)
	gez = func(ns []TTNode) {
		for _, n := range ns {
			if n.Text != "" {
				sb.WriteString(n.Text + " ")
			}
			gez(n.Content)
		}
	}
	gez(d.Content)
	return sb.String()
}
