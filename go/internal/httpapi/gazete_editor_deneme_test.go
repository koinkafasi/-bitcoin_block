package httpapi

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"theblock/site/internal/webui"
)

// Geliştirici aracı: editörü sunucusuz denemek için tek dosyalık sayfa üretir.
//
//	GAZETE_EDITOR_DENEME=/tmp/gzed/index.html go test ./internal/httpapi/ \
//	    -run TestGazeteEditorDeneme
//
// Gerçek editör betiğini, gerçek baskı CSS'ini ve gerçek bir şablonu kullanır;
// blok gövdeleri sunucudaki üreticiden önceden hesaplanıp boot verisine konur,
// böylece ağ isteği gerekmez. Editörün sürükleme, ızgara, katman, stil ve
// yakınlaştırma davranışı tarayıcıda denenebilir.
func TestGazeteEditorDeneme(t *testing.T) {
	hedef := os.Getenv("GAZETE_EDITOR_DENEME")
	if hedef == "" {
		t.Skip("GAZETE_EDITOR_DENEME ayarlı değil")
	}

	hab := func(b string) gazeteArticle {
		return gazeteArticle{Baslik: b,
			Spot:     "Haberin spot metni burada yer alır ve birkaç satır sürer.",
			Kategori: "Gündem", Kaynak: "theblock", Tarih: "30.10.2026"}
	}
	basliklar := []string{
		"Ekonomide yeni dönem: enflasyon beklentilerin altında kaldı",
		"Meclis bütçe görüşmelerine başladı",
		"İstanbul'da ulaşıma yeni hat açılıyor",
		"Derbide puanlar paylaşıldı",
	}

	kapak := gazeteKlasikMansetKapak()
	bolum := gazeteBolumSayfasi(2, "Gündem", "GÜNDEM")

	type edSayfa struct {
		No       int                        `json:"no"`
		Ad       string                     `json:"ad"`
		Bloklar  []gazeteBlock              `json:"bloklar"`
		BlokHTML map[string]gazeteBlokGovde `json:"blokHtml"`
	}
	hazirla := func(p gazetePageDoc) edSayfa {
		hm := map[string]gazeteBlokGovde{}
		bl := make([]gazeteBlock, 0, len(p.Blocks))
		for i, b := range p.Blocks {
			b.Z = i
			if b.Kaynak != nil {
				n := b.Kaynak.Limit
				if n <= 0 {
					n = 1
				}
				var hs []gazeteArticle
				for j := 0; j < n; j++ {
					hs = append(hs, hab(basliklar[j%len(basliklar)]))
				}
				b.Icerik = &gazeteIcerik{Haberler: hs}
			}
			if b.Tip == "metin" && b.Metin == "" {
				b.Metin = strings.Repeat("Değerlendirme metni. ", 8)
			}
			hm[b.ID] = gazeteBlokGovdesi(b)
			bl = append(bl, b)
		}
		return edSayfa{No: p.No, Ad: p.Ad, Bloklar: bl, BlokHTML: hm}
	}

	a4 := gazeteBoyut("A4")
	boot := map[string]any{
		"issueId": 0, "title": "Deneme Sayısı", "issueDate": "2026-10-30",
		"status": "taslak", "surum": "deneme",
		"sayfalar":  []edSayfa{hazirla(kapak), hazirla(bolum)},
		"sablonlar": []map[string]any{},
		"sayfa":     gazeteSablonVarsayilanSayfa(),
		"izgara":    gazeteSablonVarsayilanIzgara(),
		"sayfaW":    a4.W, "sayfaH": a4.H, "kenar": a4.Kenar, "tipTab": a4.TipTab,
	}
	bootJSON, err := json.Marshal(boot)
	if err != nil {
		t.Fatal(err)
	}
	// `<` kaçırılır: script etiketi kapanış sızıntısına karşı (sunucudaki
	// davranışın aynısı).
	bootStr := strings.ReplaceAll(string(bootJSON), "<", `<`)

	oku := func(ad string) string {
		b, err := webui.FS.ReadFile("static/" + ad)
		if err != nil {
			t.Fatalf("%s okunamadı: %v", ad, err)
		}
		return string(b)
	}

	// Araç çubuğu ve panel iskeleti: gerçek editörün beklediği kimlikler.
	dugme := func(id, metin string) string {
		return `<button class="btn small" id="` + id + `">` + metin + `</button>`
	}
	palet := ""
	for _, tip := range gazeteBlokTipleri {
		palet += `<button class="gz-add" data-tip="` + tip + `">` + tip + `</button>`
	}

	doc := `<!doctype html><html lang="tr"><head><meta charset="utf-8">
<title>Gazete editörü — deneme</title>
<style>` + oku("gazete-baski.css") + `</style>
<style>` + oku("gazete.css") + `</style>
<style>
body{margin:0;padding:14px;background:#f3f2ef;font-family:system-ui,sans-serif}
.card{background:#fff;border:1px solid #e5e7eb;border-radius:10px}
.card-pad{padding:10px}
.btn{padding:5px 9px;border:1px solid #e5e7eb;border-radius:7px;background:#fff;cursor:pointer;font-size:.82rem}
.input{width:100%;padding:4px 6px;border:1px solid #e5e7eb;border-radius:6px;font-size:.8rem}
.small{font-size:.8rem}.hint{color:#6b7280}
#gz-canvas{--gz-sayfa-w:` + a4.W + `;--gz-sayfa-h:` + a4.H + `;--gz-kenar:` + a4.Kenar + `;
  --gz-tip-tab:` + a4.TipTab + `;--gz-tip-olcek:1;
  --gz-tip:calc(var(--gz-tip-tab) * var(--gz-tip-olcek));}
</style></head><body>
<div class="gz-toolbar card card-pad">
  <strong>Deneme Sayısı</strong><span style="flex:1"></span>
  ` + dugme("gz-undo", "Geri") + dugme("gz-redo", "Yinele") + `
  <span class="gz-zoom">` + dugme("gz-zoom-out", "−") +
		`<span class="gz-zoom-etiket" id="gz-zoom-etiket">100%</span>` +
		dugme("gz-zoom-in", "+") + `</span>
  ` + dugme("gz-izgara-ac", "Izgara") + `
  <span class="gz-tasma-sayac" id="gz-tasma-sayac" hidden></span>
  ` + dugme("gz-kaydet", "Kaydet") + `
</div>
<div class="gz-bildirim info" id="gz-bildirim" hidden></div>
<div class="gz-layout">
  <aside class="gz-palette card"><h3 class="gz-h">Bloklar</h3>` + palet + `
    <h3 class="gz-h" style="margin-top:14px">Sayfalar ` + dugme("gz-sayfa-ekle", "+") + `</h3>
    <div id="gz-pages" class="gz-pages"></div>
    <select class="input" id="gz-sablonlar"><option value="">Şablon…</option></select>
  </aside>
  <section class="gz-canvas-wrap card">
    <div id="gz-canvas-olcek" class="gz-canvas-olcek">
      <div id="gz-canvas" class="gazete-sayfa"></div>
    </div>
  </section>
  <aside class="gz-props card">
    <h3 class="gz-h">Katmanlar</h3><div id="gz-layers" class="gz-layers"></div>
    <h3 class="gz-h" style="margin-top:14px">Özellikler</h3>
    <div id="gz-props-form" class="gz-props-form"></div>
  </aside>
</div>
<div id="gz-boot" hidden>` + bootStr + `</div>
<script>
// Sunucusuz deneme: ağ istekleri boş yanıtla karşılanır.
window.fetch = function(){ return Promise.resolve({
  ok:true, status:200, json:function(){return Promise.resolve({});},
  text:function(){return Promise.resolve("");}}); };
</script>
<script>` + oku("gazete-editor.js") + `</script>
</body></html>`

	if err := os.WriteFile(hedef, []byte(doc), 0o644); err != nil {
		t.Fatalf("yazılamadı: %v", err)
	}
	t.Logf("yazıldı: %s", hedef)
}
