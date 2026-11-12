package httpapi

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"net/http"
	"sync"
)

// Varsayılan Open Graph görseli — 1200×630 (Facebook/X önerilen ölçü).
//
// NEDEN GEREKLİ: görseli olmayan makalelerde `og:image` hiç basılmıyordu ve
// Twitter kartı `summary_large_image` olduğu hâlde görselsiz kalıyordu;
// paylaşımlar boş/kırık kart olarak çıkıyordu. Kaynak görseli indirilemeyen
// (hotlink 403) içeriklerde de `/kg/...` 404 veriyordu — sosyal botlar
// başarısız sonucu önbelleğe alır ve bir daha denemez.
//
// NEDEN ÇALIŞMA ANINDA ÜRETİLİYOR: depoya ikili (binary) PNG koymak yerine
// marka renkleriyle çiziyoruz. Renk değişirse görsel de değişir; ayrı bir
// tasarım dosyası bakımı kalmaz. Yazı YOK — font gömmek gerekmiyor, marka
// işareti (üç blok) zaten dil-bağımsız.
//
// Bir kez üretilip bellekte tutulur: 1200×630 PNG ~10 ms, ama her istekte
// yeniden çizmek gereksiz CPU olurdu.
var (
	varsayilanOGBir  sync.Once
	varsayilanOGBayt []byte
)

// varsayilanOGPng markalı varsayılan OG görselini döndürür (üretir + saklar).
func varsayilanOGPng() []byte {
	varsayilanOGBir.Do(func() {
		varsayilanOGBayt = varsayilanOGCiz()
	})
	return varsayilanOGBayt
}

// varsayilanOGCiz 1200×630 markalı kartı çizer.
func varsayilanOGCiz() []byte {
	const G, Y = 1200, 630
	img := image.NewRGBA(image.Rect(0, 0, G, Y))

	krem := color.RGBA{243, 241, 233, 255} // --cd-bg (açık tema)
	koyu := color.RGBA{10, 11, 14, 255}    // --cd-dark
	sari := color.RGBA{252, 195, 52, 255}  // --cd-brand

	draw.Draw(img, img.Bounds(), &image.Uniform{krem}, image.Point{}, draw.Src)

	// Alt bant: markanın sarı vurgusu.
	draw.Draw(img, image.Rect(0, Y-96, G, Y), &image.Uniform{sari}, image.Point{}, draw.Src)

	// Marka işareti (cdBasSimge'nin üç bloğu) — 24'lük ızgaradan büyütülmüş.
	// Ölçek 9: 24*9 = 216 px'lik işaret, sol üstte dengeli duruyor.
	const olcek, sol, ust = 9, 96, 120
	blok := func(x, y, w, h float64, op uint8) {
		kutu := image.Rect(
			sol+int(x*olcek), ust+int(y*olcek),
			sol+int((x+w)*olcek), ust+int((y+h)*olcek),
		)
		draw.Draw(img, kutu, &image.Uniform{color.RGBA{koyu.R, koyu.G, koyu.B, op}}, image.Point{}, draw.Over)
	}
	blok(0, 6, 6, 7, 128)
	blok(7.5, 1, 9.5, 12, 199)
	blok(13, 14.5, 11, 8.5, 255)

	// Metin yerine geometrik denge: sağ tarafta ince sarı çizgiler.
	for i := 0; i < 5; i++ {
		y := 150 + i*52
		w := 460 - i*56
		if w <= 80 {
			break
		}
		draw.Draw(img, image.Rect(640, y, 640+w, y+18),
			&image.Uniform{color.RGBA{koyu.R, koyu.G, koyu.B, uint8(38 - i*6)}}, image.Point{}, draw.Over)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil
	}
	return buf.Bytes()
}

// handleVarsayilanOG /og-varsayilan.png sunar.
//
// Uzun önbellek: içerik derleme zamanında sabit; marka rengi değişirse
// `?v=` sorgusu kırıcı olarak kullanılır.
func (s *Server) handleVarsayilanOG(w http.ResponseWriter, r *http.Request) {
	b := varsayilanOGPng()
	if len(b) == 0 {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeContent(w, r, "og-varsayilan.png", ogSabitTarih, bytes.NewReader(b))
}

// varsayilanOGYol sitedeki varsayılan OG görselinin göreli yoludur.
//
// NOT: OG görsel işi sürüyor (opencode). Henüz çağrılmıyor; iş bitince
// paylaşım etiketlerinde kullanılacak.
func varsayilanOGYol() string { return "/og-varsayilan.png" }
