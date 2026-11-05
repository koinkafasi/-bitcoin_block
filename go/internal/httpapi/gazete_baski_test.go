package httpapi

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"theblock/site/internal/platform"
	"theblock/site/internal/webui"
)

// Faz 1 koruyucu testleri — docs/SPEC-GAZETE-DUZENLEYICI.md
//
// Korunan davranış:
//   K1 — baskı tasarımı tek kaynaktan gelir (gömülü statik dosya)
//   K2 — sayfa ölçüsü ve tipografi tabanı CSS'te sabit değil, bildirilir

// cssYorumsuz CSS'ten /* ... */ yorumlarını atar. Yorum metinleri kural
// aramasını yanıltıyor (yorumda geçen "@page" gerçek kural sanılıyordu).
func cssYorumsuz(css string) string {
	return regexp.MustCompile(`(?s)/\*.*?\*/`).ReplaceAllString(css, "")
}

// cssKurallar CSS'i {seçici: gövde} olarak ayrıştırır. Basit bir ayrıştırıcı:
// baskı CSS'inde iç içe kural (@media vb.) yok ve olmaması da isteniyor.
func cssKurallar(t *testing.T, css string) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, parca := range strings.Split(cssYorumsuz(css), "}") {
		i := strings.Index(parca, "{")
		if i < 0 {
			continue
		}
		sec := strings.TrimSpace(parca[:i])
		if sec == "" {
			continue
		}
		if strings.Contains(sec, "{") {
			t.Fatalf("iç içe kural bulundu, ayrıştırıcı yetersiz: %q", sec)
		}
		out[sec] = parca[i+1:]
	}
	return out
}

// TestGazeteBaskiCSSParametrik baskı tasarımının sayfa ölçüsünü kendi içinde
// sabitlemediğini doğrular. Ölçü sabitlenirse şablon başına sayfa boyu
// seçilemez ve A4/A3 ayrımı çöker.
func TestGazeteBaskiCSSParametrik(t *testing.T) {
	ham := gazeteBaskiCSS()
	if strings.Contains(ham, "okunamadi") {
		t.Fatalf("gömülü baskı CSS'i okunamadı: %s", ham)
	}
	css := cssYorumsuz(ham)
	// @page kuralı Go tarafında yazılır; CSS dosyasında olmamalı.
	if strings.Contains(css, "@page") {
		t.Error("gazete-baski.css içinde @page var; sayfa kutusu Go tarafında yazılmalı")
	}
	// Sayfa ölçüsü yalnız :root varsayılanında geçebilir; kural gövdelerinde
	// sabit A3 ölçüsü kalmamalı.
	for _, yasak := range []string{"297mm 420mm", "size: A3", "size:A3"} {
		if strings.Contains(css, yasak) {
			t.Errorf("baskı CSS'i sabit sayfa ölçüsü taşıyor: %q", yasak)
		}
	}
	// Tipografi tabandan türetilmeli: doğrudan punto yazılmış kural olmamalı.
	// (:root içindeki --gz-tip-tab tanımı hariç.)
	satirlar := strings.Split(css, "\n")
	pt := regexp.MustCompile(`font-size:\s*[0-9.]+pt`)
	for i, satir := range satirlar {
		if pt.MatchString(satir) {
			t.Errorf("satır %d sabit punto kullanıyor, --gz-tip çarpanı olmalı: %s",
				i+1, strings.TrimSpace(satir))
		}
	}
}

// TestGazeteSayfaDegiskenleriKutu bildirilen sayfa kutusunun @page kuralına
// ve değişkenlere birebir yazıldığını doğrular.
func TestGazeteSayfaDegiskenleriKutu(t *testing.T) {
	for _, tc := range []struct{ ad, w, h string }{
		{"A4", "210mm", "297mm"},
		{"A3", "297mm", "420mm"},
		{"", "210mm", "297mm"},         // bilinmeyen → A4
		{"bilinmez", "210mm", "297mm"}, // bilinmeyen → A4
	} {
		b := gazeteBoyut(tc.ad)
		d := gazeteSayfaDegiskenleri(b, 1)
		beklenen := fmt.Sprintf("@page { size: %s %s; margin: 0; }", tc.w, tc.h)
		if !strings.Contains(d, beklenen) {
			t.Errorf("boyut %q: @page kuralı beklenen %q değil:\n%s", tc.ad, beklenen, d)
		}
		if !strings.Contains(d, "--gz-sayfa-w:"+tc.w) || !strings.Contains(d, "--gz-sayfa-h:"+tc.h) {
			t.Errorf("boyut %q: sayfa değişkenleri eksik:\n%s", tc.ad, d)
		}
	}
	// Geçersiz ölçek 1'e düşmeli; 0 olursa tüm tipografi kaybolur.
	if d := gazeteSayfaDegiskenleri(gazeteBoyut("A4"), 0); !strings.Contains(d, "--gz-tip-olcek:1") {
		t.Errorf("sıfır ölçek 1'e düşmeli:\n%s", d)
	}
}

// TestGazeteA3TipografiKorunur A3 tabanının (9.5pt) baskı CSS'indeki
// çarpanlarla eski sabit puntolara denk geldiğini doğrular. Bu test, sayfa
// ölçüsünü parametrik yaparken A3 çıktısının görünümünü koruduğumuzun kanıtı:
// biri çarpanı elle değiştirirse burada yakalanır.
func TestGazeteA3TipografiKorunur(t *testing.T) {
	const taban = 9.5 // A3 tipografi tabanı, gazeteBoyutlar["A3"].TipTab

	// Eski sabit baskı CSS'indeki punto değerleri (A3 dikey).
	beklenen := map[string]float64{
		".gb-baslik":       16,
		".gb-baslik-k":     11,
		".gb-manset h1":    26,
		".gb-manset p":     11,
		".gb-kucuk":        7.5,
		".gb-liste strong": 11,
		".gb-liste span":   8.5,
		".gb-kart h3":      10.5,
		".gb-kart p":       8,
		".gb-kunye":        7,
		".gb-kunye-ust":    8,
	}

	if ta := gazeteBoyutlar["A3"].TipTab; ta != "9.5pt" {
		t.Fatalf("A3 tipografi tabanı değişmiş (%s); bu testin varsayımı 9.5pt", ta)
	}

	// Her kuralın gövdesindeki --gz-tip çarpanını oku ve A3 tabanıyla çarp.
	re := regexp.MustCompile(`font-size:\s*calc\(var\(--gz-tip\)\s*\*\s*([0-9.]+)\)`)
	bulunan := map[string]float64{}
	for sec, govde := range cssKurallar(t, gazeteBaskiCSS()) {
		m := re.FindStringSubmatch(govde)
		if m == nil {
			continue
		}
		carpan, err := strconv.ParseFloat(m[1], 64)
		if err != nil {
			t.Fatalf("%s: çarpan okunamadı %q: %v", sec, m[1], err)
		}
		bulunan[sec] = carpan * taban
	}

	for sec, punto := range beklenen {
		got, ok := bulunan[sec]
		if !ok {
			t.Errorf("%s için --gz-tip çarpanı bulunamadı", sec)
			continue
		}
		if diff := got - punto; diff > 0.01 || diff < -0.01 {
			t.Errorf("%s: A3'te %.4fpt üretiyor, eski değer %.2fpt (sapma %.4f)",
				sec, got, punto, diff)
		}
	}
}

// TestGazeteBaskiCSSGomulu baskı tasarımının gerçekten gömülü statik dosyadan
// geldiğini doğrular. Editör aynı dosyayı HTTP ile yükleyeceği için tek
// render yolunun ön koşulu bu (SPEC K1).
func TestGazeteBaskiCSSGomulu(t *testing.T) {
	b, err := webui.FS.ReadFile("static/gazete-baski.css")
	if err != nil {
		t.Fatalf("static/gazete-baski.css gömülü değil: %v", err)
	}
	if string(b) != gazeteBaskiCSS() {
		t.Error("gazeteBaskiCSS() gömülü dosyadan farklı içerik dönüyor")
	}
}

// TestGazeteTekRenderYolu editörün kendi blok gövdesini üretmediğini
// doğrular (SPEC K1). Editör yer tutucu çizerse "editörde sığdı ama baskıda
// taştı" sınıfı geri gelir; bu yüzden gövde yalnız sunucudan gelmeli.
func TestGazeteTekRenderYolu(t *testing.T) {
	b, err := webui.FS.ReadFile("static/gazete-editor.js")
	if err != nil {
		t.Fatalf("gazete-editor.js okunamadı: %v", err)
	}
	js := string(b)

	// Editörün eski taklit üreticisi ve onun sınıfları geri gelmemeli.
	for _, yasak := range []string{
		"blokIcerik", // yer tutucu gövde üreticisi
		"gz-blok",    // taklit blok kapsayıcısı; artık .gb kullanılıyor
		"— haber —",  // liste yer tutucusu
		"gz-manset",  // tipe özel taklit gövdeler
		"gz-kart",
		"gz-reklam",
	} {
		if strings.Contains(js, yasak) {
			t.Errorf("gazete-editor.js blok gövdesini kendisi üretiyor (%q bulundu); "+
				"gövde sunucudan gelmeli — bkz. specs/13 K1", yasak)
		}
	}
	// Sunucu gövdesini gerçekten kullanıyor olmalı.
	for _, gerek := range []string{"blokHtml", "blok-html"} {
		if !strings.Contains(js, gerek) {
			t.Errorf("gazete-editor.js sunucu gövdesini kullanmıyor: %q yok", gerek)
		}
	}
	// Tuval baskı sınıflarını kullanmalı.
	if !strings.Contains(js, `"gb gb-t-"`) {
		t.Error("tuval baskı sınıflarını (.gb gb-<tip>) kullanmıyor")
	}
}

// TestGazeteBlokIcDisTutarli kapsayıcılı üretimin, iç gövdeyi olduğu gibi
// sardığını doğrular. İkisi ayrışırsa baskı ile editör yine farklı HTML
// gösterir.
func TestGazeteBlokIcDisTutarli(t *testing.T) {
	bloklar := []gazeteBlock{
		{ID: "a", Tip: "baslik", X: 2, Y: 3, W: 96, H: 6, Baslik: "GÜNDEM"},
		{ID: "b", Tip: "metin", X: 10, Y: 20, W: 40, H: 30, Metin: "deneme şşğü"},
		{ID: "c", Tip: "reklam", X: 0, Y: 0, W: 30, H: 15},
		{ID: "d", Tip: "manset", X: 1, Y: 1, W: 50, H: 40, Icerik: &gazeteIcerik{
			Haberler: []gazeteArticle{{Baslik: "Başlık", Spot: "Spot", Kategori: "Gündem"}},
		}},
	}
	for _, b := range bloklar {
		dis := gazeteBlockHTML(b)
		ic := gazeteBlockInnerHTML(b)
		if !strings.Contains(dis, ic) {
			t.Errorf("%s/%s: kapsayıcılı çıktı iç gövdeyi içermiyor", b.ID, b.Tip)
		}
		// Kapsayıcı baskı sınıfını ve yüzdelik konumu taşımalı.
		if !strings.Contains(dis, `class="gb gb-t-`+b.Tip+`"`) {
			t.Errorf("%s: kapsayıcı sınıfı eksik:\n%s", b.ID, dis)
		}
		if !strings.Contains(dis, "left:") || !strings.Contains(dis, "height:") {
			t.Errorf("%s: konum stili eksik:\n%s", b.ID, dis)
		}
		// İç gövde kapsayıcı taşımamalı, yoksa editörde iç içe kutu olur.
		if strings.HasPrefix(strings.TrimSpace(ic), `<div class="gb `) ||
			strings.HasPrefix(strings.TrimSpace(ic), `<div class="gb gb-t-`) {
			t.Errorf("%s: iç gövde kendi kapsayıcısını taşıyor", b.ID)
		}
	}
}

// TestGazeteBlokHTMLHaritasi editör boot verisinin blok kimliğine göre gövde
// döndürdüğünü doğrular.
func TestGazeteBlokHTMLHaritasi(t *testing.T) {
	raw := []byte(`[{"id":"p1b","tip":"baslik","baslik":"EKONOMİ","x":2,"y":2,"w":96,"h":6},
	                {"id":"p1r","tip":"reklam","x":2,"y":90,"w":40,"h":8}]`)
	h := gazeteBlokHTMLHaritasi(raw)
	if len(h) != 2 {
		t.Fatalf("2 blok beklenirken %d döndü: %#v", len(h), h)
	}
	if !strings.Contains(h["p1b"].HTML, "EKONOMİ") {
		t.Errorf("başlık gövdesi içeriği taşımıyor: %q", h["p1b"].HTML)
	}
	if !strings.Contains(h["p1r"].HTML, "REKLAM") {
		t.Errorf("reklam gövdesi beklenen içeriği taşımıyor: %q", h["p1r"].HTML)
	}
	// Bozuk JSON çökertmemeli.
	if got := gazeteBlokHTMLHaritasi([]byte("bozuk")); got != nil {
		t.Errorf("bozuk JSON için nil beklenirken %#v döndü", got)
	}
}

// TestGazeteBaskiCSSPanelGuvenli paylaşılan baskı CSS'inin genel seçici
// kullanmadığını doğrular. Dosya admin panelinde de yükleniyor; `body` veya
// `*` kuralı panelin tamamını Georgia yapar.
func TestGazeteBaskiCSSPanelGuvenli(t *testing.T) {
	for sec := range cssKurallar(t, gazeteBaskiCSS()) {
		for _, parca := range strings.Split(sec, ",") {
			parca = strings.TrimSpace(parca)
			if parca == "body" || parca == "*" || parca == "html" {
				t.Errorf("baskı CSS'inde genel seçici var: %q — panele sızar", parca)
			}
			if !strings.HasPrefix(parca, ".gazete-sayfa") && !strings.HasPrefix(parca, ".gb") &&
				parca != ":root" {
				t.Errorf("baskı CSS'inde kapsam dışı seçici: %q", parca)
			}
		}
	}
}

// TestGazeteBaskiMedyaMutlak baskıdaki görsel yollarının file:// altında
// çözülebilir hale geldiğini doğrular.
//
// Baski dokümanı Chromium'a file:// ile veriliyor; "/media/x" orada dosya
// sistemi köküne çözülüyordu ve depolama {STORAGE_DIR}/media altında olduğu
// için üretilen gazete PDF'lerinde hiçbir görsel görünmüyordu.
func TestGazeteBaskiMedyaMutlak(t *testing.T) {
	// Gerçek bir dosya kur; dönüşen yol bu dosyayı göstermeli.
	// Depolama kökü paket düzeyinde genel bir değer; diğer testleri
	// etkilememesi için geri alınıyor.
	onceki := platform.StorageDir()
	t.Cleanup(func() { platform.SetStorageDir(onceki) })
	kok := t.TempDir()
	platform.SetStorageDir(kok)
	rel := filepath.Join("media", "202610", "ornek.png")
	tam := filepath.Join(kok, rel)
	if err := os.MkdirAll(filepath.Dir(tam), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tam, []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}

	doc := `<img src="/media/202610/ornek.png" alt=""><img src="/media/yok.png" alt="">`
	got := gazeteBaskiMedyaMutlak(doc)

	if strings.Contains(got, `src="/media/`) {
		t.Errorf("göreli /media yolu kaldı:\n%s", got)
	}
	// file:// önekini söküp dosyanın gerçekten orada olduğunu doğrula.
	re := regexp.MustCompile(`src="file://([^"]+)"`)
	eslesmeler := re.FindAllStringSubmatch(got, -1)
	if len(eslesmeler) != 2 {
		t.Fatalf("2 mutlak yol beklenirken %d bulundu:\n%s", len(eslesmeler), got)
	}
	if _, err := os.Stat(eslesmeler[0][1]); err != nil {
		t.Errorf("dönüşen yol gerçek dosyayı göstermiyor: %s (%v)", eslesmeler[0][1], err)
	}
	// Yalnız src rewrite edilmeli; metin içindeki /media/ dokunulmamalı.
	if g := gazeteBaskiMedyaMutlak(`<p>/media/ hakkında</p>`); !strings.Contains(g, "<p>/media/ hakkında</p>") {
		t.Errorf("src dışındaki metin değişti: %s", g)
	}
}

// TestGazeteFirstLineRuneGuvenli kısaltmanın çok baytlı harfi ortadan
// bölmediğini doğrular. Bayt bazlı kesme baskıya geçersiz UTF-8 yazıyordu.
func TestGazeteFirstLineRuneGuvenli(t *testing.T) {
	// Kesme noktası tam bir Türkçe harfin üzerine gelecek şekilde kurulu.
	s := strings.Repeat("ş", 200)
	for _, max := range []int{1, 5, 99, 100, 120, 199} {
		got := firstLine(s, max)
		if !utf8.ValidString(got) {
			t.Errorf("max=%d: geçersiz UTF-8 üretildi", max)
		}
		gövde := strings.TrimSuffix(got, "…")
		if n := len([]rune(gövde)); n > max {
			t.Errorf("max=%d: %d karakter döndü", max, n)
		}
	}
	// Kısa metin olduğu gibi dönmeli, üç nokta eklenmemeli.
	if got := firstLine("kısa", 100); got != "kısa" {
		t.Errorf("kısa metin değişti: %q", got)
	}
	// Satır sonları tek satıra inmeli.
	if got := firstLine("bir\niki", 100); got != "bir iki" {
		t.Errorf("satır sonu birleştirilmedi: %q", got)
	}
}
