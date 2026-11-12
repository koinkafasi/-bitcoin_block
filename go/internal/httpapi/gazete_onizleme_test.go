package httpapi

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestGazeteOnizlemeUret(t *testing.T) {
	hedef := os.Getenv("GAZETE_ONIZLEME")
	if hedef == "" {
		t.Skip("GAZETE_ONIZLEME ayarlı değil")
	}

	hab := func(b, s, kat string) gazeteArticle {
		return gazeteArticle{Baslik: b, Spot: s, Kategori: kat,
			Kaynak: "theblock", Tarih: "30.10.2026"}
	}
	basliklar := []string{
		"Ekonomide yeni dönem: enflasyon beklentilerin altında kaldı",
		"Meclis bütçe görüşmelerine başladı, komisyon hafta boyu çalışacak",
		"İstanbul'da ulaşıma yeni hat: ilk etap yılbaşında açılıyor",
		"Derbide puanlar paylaşıldı, lider değişmedi",
		"Modern sanat koleksiyonu ziyarete açıldı",
		"Tarımsal üretimde beklenenin üzerinde artış kaydedildi",
	}

	// Şablon bloklarına örnek içerik bağla: anatominin doğru durduğu
	// ancak gerçek içerikle görülür.
	doldur := func(b gazeteBlock) gazeteBlock {
		if b.Kaynak != nil {
			n := b.Kaynak.Limit
			if n <= 0 {
				n = 1
			}
			var hs []gazeteArticle
			for i := 0; i < n; i++ {
				hs = append(hs, hab(basliklar[i%len(basliklar)],
					"Haberin spot metni burada yer alır ve birkaç satır sürer.",
					"Gündem"))
			}
			b.Icerik = &gazeteIcerik{Haberler: hs}
		}
		if b.Tip == "metin" && b.Metin == "" {
			b.Metin = strings.Repeat("Değerlendirme metni bu blokta yer alır. ", 10)
		}
		return b
	}

	sayfaHTML := func(p gazetePageDoc) string {
		var sb strings.Builder
		sb.WriteString(`<div class="gazete-sayfa"><div class="gb-kunye-ust">Sayfa ` +
			fmt.Sprint(p.No) + `</div>`)
		for _, b := range p.Blocks {
			sb.WriteString(gazeteBlockHTML(doldur(b)))
		}
		sb.WriteString(`</div>`)
		return sb.String()
	}

	n := 0
	kutu := func(baslik string, b gazeteSayfaBoyut, olcek float64, icerik string) string {
		n++
		sinif := fmt.Sprintf("kap%d", n)
		return fmt.Sprintf(`
<h2>%s — %s × %s · tipografi tabanı %s · ölçek %.2f</h2>
<style>
.%s{--gz-sayfa-w:%s;--gz-sayfa-h:%s;--gz-kenar:%s;--gz-tip-tab:%s;--gz-tip-olcek:1;
  --gz-tip:calc(var(--gz-tip-tab) * var(--gz-tip-olcek));}
.%s .gazete-sayfa{transform:scale(%.4f);transform-origin:top left;
  box-shadow:0 2px 18px rgba(0,0,0,.25);}
.%s{height:calc(%s * %.4f);margin-bottom:28px;}
</style>
<div class="%s">%s</div>`,
			baslik, b.W, b.H, b.TipTab, olcek,
			sinif, b.W, b.H, b.Kenar, b.TipTab,
			sinif, olcek,
			sinif, b.H, olcek,
			sinif, icerik)
	}

	a4 := gazeteBoyut("A4")
	a3 := gazeteBoyut("A3")

	kapak := sayfaHTML(gazeteKlasikMansetKapak())
	bolum := sayfaHTML(gazeteBolumSayfasi(2, "Gündem", "GÜNDEM"))

	doc := `<!doctype html><html lang="tr"><head><meta charset="utf-8">
<title>Gazete baskı önizleme</title><style>
body{margin:0;padding:20px;background:#555;font-family:system-ui,sans-serif}
h2{color:#fff;font-size:14px;font-weight:600;margin:0 0 10px}
` + gazeteBaskiCSS() + `
</style></head><body>` +
		kutu("A4 — Klasik Manşet kapağı (referans peyzaj), gerçek ölçek", a4, 1, kapak) +
		kutu("A4 — Klasik Manşet kapağı, editör tuvali ölçeğinde", a4, 0.55, kapak) +
		kutu("A4 — bölüm sayfası", a4, 0.55, bolum) +
		kutu("A3 — tipografi tabanı 9.5pt (eski görünüm)", a3, 0.55, kapak) +
		`</body></html>`

	if err := os.WriteFile(hedef, []byte(doc), 0o644); err != nil {
		t.Fatalf("yazılamadı: %v", err)
	}
	t.Logf("yazıldı: %s", hedef)
}
