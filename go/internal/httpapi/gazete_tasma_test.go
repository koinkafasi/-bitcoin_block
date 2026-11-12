package httpapi

import (
	"strings"
	"testing"
)

// TestGazeteOlcumBetigiYalnizDogrulamada ölçüm betiğinin baskı çıktısına
// sızmadığını doğrular. PDF'e giren dokümanda script olmamalı.
func TestGazeteOlcumBetigiYalnizDogrulamada(t *testing.T) {
	if got := olcumBetigi(false); got != "" {
		t.Errorf("baskı dokümanına ölçüm betiği eklenmiş: %q", got)
	}
	got := olcumBetigi(true)
	if !strings.Contains(got, "gz-tasma-raporu") {
		t.Errorf("doğrulama betiği rapor kabını kurmuyor: %q", got)
	}
	if !strings.Contains(got, "scrollHeight") {
		t.Error("doğrulama betiği taşmayı ölçmüyor")
	}
}

// TestGazeteTasmaAyikla dump-dom çıktısından raporun doğru çözüldüğünü
// doğrular; sıralama ve öneri üretimi dahil.
func TestGazeteTasmaAyikla(t *testing.T) {
	dom := `<html><body><div class="gazete-sayfa"></div>` +
		`<div id="gz-tasma-raporu">[` +
		`{"sayfa":3,"blok":"p3l","tip":"liste","dikey":40,"yatay":0,"kutuY":200},` +
		`{"sayfa":1,"blok":"p1m","tip":"manset","dikey":10,"yatay":0,"kutuY":400},` +
		`{"sayfa":1,"blok":"p1l","tip":"liste","dikey":90,"yatay":0,"kutuY":300}` +
		`]</div></body></html>`

	got, err := gazeteTasmaAyikla(dom)
	if err != nil {
		t.Fatalf("ayıklanamadı: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("3 taşma beklenirken %d döndü", len(got))
	}
	// Sayfa artan, aynı sayfada taşma azalan sırada olmalı: kullanıcı önce
	// en kötü sorunu görsün.
	bekle := []string{"p1l", "p1m", "p3l"}
	for i, id := range bekle {
		if got[i].BlokID != id {
			t.Errorf("sıra %d: %q beklenirken %q", i, id, got[i].BlokID)
		}
	}
	for _, x := range got {
		if x.Oneri == "" {
			t.Errorf("%s için öneri üretilmedi", x.BlokID)
		}
	}
}

// TestGazeteTasmaAyiklaBozukGirdi ölçüm yapılamadığında sessiz "sorun yok"
// dönülmediğini doğrular. Sessiz geçmek, taşmayı gizlemekle aynı şey olurdu.
func TestGazeteTasmaAyiklaBozukGirdi(t *testing.T) {
	for ad, dom := range map[string]string{
		"rapor yok":  `<html><body>sayfa yüklenemedi</body></html>`,
		"kapanmamış": `<div id="gz-tasma-raporu">[{"sayfa":1}`,
		"bozuk json": `<div id="gz-tasma-raporu">bu json değil</div>`,
	} {
		if _, err := gazeteTasmaAyikla(dom); err == nil {
			t.Errorf("%s: hata beklenirken sorunsuz döndü", ad)
		}
	}
	// Boş rapor geçerli: taşma yok demek.
	got, err := gazeteTasmaAyikla(`<div id="gz-tasma-raporu">[]</div>`)
	if err != nil {
		t.Fatalf("boş rapor hata verdi: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("boş rapor için %d kayıt döndü", len(got))
	}
}

// TestGazeteTasmaOneri önerinin aritmetiğini sabitler. Öneri yanlışsa
// kullanıcı bloğu yeterince büyütmez ve taşma sürer.
func TestGazeteTasmaOneri(t *testing.T) {
	// 200px kutu, 50px taşma → %25 büyütme, ölçek 0.80.
	o := gazeteTasmaOneri(gazeteTasma{KutuY: 200, Dikey: 50})
	if !strings.Contains(o, "%25") {
		t.Errorf("yükseklik önerisi yanlış: %q", o)
	}
	if !strings.Contains(o, "0.80") {
		t.Errorf("ölçek önerisi yanlış: %q", o)
	}
	// Yalnız yatay taşmada yükseklik önerilmemeli.
	y := gazeteTasmaOneri(gazeteTasma{KutuY: 100, Dikey: 0, Yatay: 12})
	if strings.Contains(y, "yuksekligi") {
		t.Errorf("yatay taşmada yükseklik önerildi: %q", y)
	}
	// Kutu yüksekliği bilinmiyorsa çökmemeli.
	if s := gazeteTasmaOneri(gazeteTasma{KutuY: 0, Dikey: 10}); s == "" {
		t.Error("ölçüsüz blok için öneri boş")
	}
}

// TestGazeteTasmaOzeti kullanıcıya dönen metnin sayıyı, örnekleri ve çıkış
// yolunu içerdiğini doğrular.
func TestGazeteTasmaOzeti(t *testing.T) {
	if got := gazeteTasmaOzeti(nil); got != "" {
		t.Errorf("taşma yokken özet üretildi: %q", got)
	}
	var çok []gazeteTasma
	for i := 0; i < 5; i++ {
		çok = append(çok, gazeteTasma{Sayfa: i + 1, BlokID: "b", Tip: "liste", KutuY: 100, Dikey: 10, Oneri: "x"})
	}
	got := gazeteTasmaOzeti(çok)
	if !strings.Contains(got, "5 blok") {
		t.Errorf("toplam sayı yok: %q", got)
	}
	if !strings.Contains(got, "2 blok daha") {
		t.Errorf("kalan sayısı bildirilmedi: %q", got)
	}
	// Kullanıcı sıkışmamalı: yine de üretmenin yolu söylenmeli.
	if !strings.Contains(got, "Zorla") {
		t.Errorf("zorlama yolu bildirilmedi: %q", got)
	}
}

// TestGazeteBlokKimligiBaskida ölçümün blokları adlandırabilmesi için baskı
// HTML'inin data-* niteliklerini taşıdığını doğrular.
func TestGazeteBlokKimligiBaskida(t *testing.T) {
	h := gazeteBlockHTML(gazeteBlock{ID: "p1m", Tip: "manset", X: 1, Y: 2, W: 50, H: 40})
	for _, gerek := range []string{`data-id="p1m"`, `data-tip="manset"`} {
		if !strings.Contains(h, gerek) {
			t.Errorf("baskı bloğunda %s yok: %s", gerek, h)
		}
	}
	// Kimlik kaçırılmalı (blok kimliği kullanıcıdan gelebilir).
	k := gazeteBlockHTML(gazeteBlock{ID: `a"><script>`, Tip: "metin"})
	if strings.Contains(k, `<script>`) {
		t.Errorf("blok kimliği kaçırılmadı: %s", k)
	}
}

func TestGazeteSarmalayiciSinifCarpismaz(t *testing.T) {
	tipler := []string{"manset", "liste", "kart", "baslik", "metin",
		"gorsel", "cizgi", "reklam", "kunye"}

	// Baski CSS'indeki ic govde sinif adlari.
	icSiniflar := map[string]bool{}
	for sec := range cssKurallar(t, gazeteBaskiCSS()) {
		for _, p := range strings.Split(sec, ",") {
			p = strings.TrimSpace(p)
			// "\.gb-metin" gibi ilk sinifi al.
			if !strings.HasPrefix(p, ".gb-") {
				continue
			}
			ad := p[1:]
			if i := strings.IndexAny(ad, " .:>["); i > 0 {
				ad = ad[:i]
			}
			if strings.HasPrefix(ad, "gb-t-") {
				continue
			}
			icSiniflar[ad] = true
		}
	}
	if len(icSiniflar) < 5 {
		t.Fatalf("baski CSS'inden ic sinif cikarilamadi: %v", icSiniflar)
	}

	for _, tip := range tipler {
		sarmal := "gb-t-" + tip
		if icSiniflar[sarmal] {
			t.Errorf("sarmalayici sinifi %q ic govde sinifiyla carpisiyor", sarmal)
		}
		// Uretilen HTML gercekten bu sinifi tasimali.
		h := gazeteBlockHTML(gazeteBlock{ID: "x", Tip: tip, W: 10, H: 10})
		if !strings.Contains(h, `class="gb `+sarmal+`"`) {
			t.Errorf("%s: sarmalayici sinifi beklenen gibi degil: %s", tip, h)
		}
	}
}
