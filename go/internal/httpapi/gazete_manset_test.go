package httpapi

import (
	"bytes"
	"html/template"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/repo"
	"theblock/site/internal/webui"
)

// Faz 197 — Gazete Manşetleri modülü.
//
// Kullanıcı isteği: "manşetler yalnız sağ modüle geçince orası için tek
// slide gibi stil de ekle, yer değişince sorun olmaması için otomatik
// manşetler sağda geçiş yapar."

func gzmKapaklar() []repo.NewspaperPage {
	return []repo.NewspaperPage{
		{Name: "Hürriyet", Slug: "hurriyet", PageDate: "2026-11-02",
			ImageURL: "https://i.gazeteoku.com/3/900/1310/a.jpg",
			ThumbURL: "https://i.gazeteoku.com/3/300/437/a.jpg", Sort: 0},
		{Name: "Sözcü", Slug: "sozcu", PageDate: "2026-11-02",
			ImageURL: "https://i.gazeteoku.com/3/900/1310/b.jpg",
			ThumbURL: "https://i.gazeteoku.com/3/300/437/b.jpg", Sort: 3},
	}
}

func renderGazeteManset(t *testing.T, b *homeBlock) string {
	t.Helper()
	funcs := templateFuncs()
	for _, name := range []string{"mediaURL", "mediaThumbURL", "mediaVariantURL", "mediaSrcset", "asset", "urlAbs", "catIcon"} {
		funcs[name] = func(...any) string { return "" }
	}
	tmpl, err := template.New("_").Funcs(funcs).ParseFS(webui.FS, "templates/*.html")
	require.NoError(t, err)

	b.Type = "gazete_manset"
	if b.D == nil {
		b.D = &tmplData{}
	}
	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "blk_gazete_manset", b))
	return buf.String()
}

func TestGazeteTarihBicimi(t *testing.T) {
	assert.Equal(t, "2 Kasım 2026", gazeteTarih("2026-11-02"))
	assert.Equal(t, "1 Ocak 2026", gazeteTarih("2026-01-01"))
	// Boş/geçersiz → boş (şablon tarih satırını hiç basmaz).
	assert.Equal(t, "", gazeteTarih(""))
	assert.Equal(t, "", gazeteTarih("02.11.2026"))
}

func TestGazeteMansetYapisi(t *testing.T) {
	h := renderGazeteManset(t, &homeBlock{
		GazeteKapak:  gzmKapaklar(),
		GazeteGun:    "2026-11-02",
		GazeteAralik: 6,
	})

	assert.Contains(t, h, `data-gm`, "modül JS çengeli taşımalı")
	assert.Contains(t, h, `data-gm-aralik="6"`)
	assert.Contains(t, h, `class="gzm-title">Gazete Manşetleri</h2>`)
	assert.Contains(t, h, `href="/gazete-mansetleri"`, "başlıkta 'Tümü' bağlantısı")
	assert.Equal(t, 2, bytes.Count([]byte(h), []byte(`data-gm-slide`)))
	// Küçük görsel şeritte, büyük görsel büyütücüde kullanılır.
	assert.Contains(t, h, `src="https://i.gazeteoku.com/3/300/437/a.jpg"`)
	assert.Contains(t, h, `data-gm-buyuk="https://i.gazeteoku.com/3/900/1310/a.jpg"`)
	// Erişilebilirlik: her kapakta okunur alt metin ve gazete adı.
	assert.Contains(t, h, `alt="Hürriyet — 2 Kasım 2026 manşeti"`)
	assert.Contains(t, h, `<span class="gzm-ad">Sözcü</span>`)
	// Tarih Türkçe yazılır, ISO biçim okuyucuya gösterilmez.
	assert.Contains(t, h, "2 Kasım 2026")
	assert.NotContains(t, h, "2026-11-02 · ", "ISO tarih doğrudan basılmamalı")
}

// Kapağa tıklama KAYNAĞA GİTMEMELİ, sayfa içinde büyütmeli.
//
// Ölçüm (Faz 197, canlı): kapağın `<a href="https://i.gazeteoku.com/…"
// target="_blank">` olması yüzünden tıklama yeni sekmede CDN'e gidiyordu.
// Artık kapak bir <button>; href yok, hedef yok, yalnız data-gm-zoom var.
// Buton seçilmesi bilinçli: JS yüklenmezse tıklama hiçbir yere gitmez
// (link olsaydı kullanıcı siteyi kaybederdi).
func TestGazeteMansetKaynagaGitmez(t *testing.T) {
	h := renderGazeteManset(t, &homeBlock{GazeteKapak: gzmKapaklar()})

	assert.NotContains(t, h, `target="_blank"`, "yeni sekme açılmamalı")
	assert.NotContains(t, h, `href="https://i.gazeteoku.com`, "CDN'e bağlantı olmamalı")
	assert.NotContains(t, h, `rel="noopener"`, "dış bağlantı kalmamalı")

	// Kapak bir düğme: hem şerit hem büyütücü çengelini taşır.
	assert.Contains(t, h, `<button type="button" class="gzm-slide" data-gm-slide data-gm-zoom`)
	assert.Equal(t, 2, bytes.Count([]byte(h), []byte(`data-gm-zoom`)))
	assert.Contains(t, h, `data-gm-grup`, "gezinme grubu (önceki/sonraki) için gerekli")
	assert.Contains(t, h, `aria-label="Hürriyet — 2 Kasım 2026 manşetini büyüt"`)
}

// Sağ sütuna (zone=yan) taşınan modül TEK SLAYT olur. Geniş bölgede aynı
// şablon şerit olarak kalır — "yer değişince sorun olmaması" kuralı.
func TestGazeteMansetTekSlaytKurali(t *testing.T) {
	dar := renderGazeteManset(t, &homeBlock{
		GazeteKapak: gzmKapaklar(), GazeteGun: "2026-11-02", GazeteDar: true,
	})
	assert.Contains(t, dar, `class="gzm gzm-tek"`,
		"sağ sütunda tek slayt sınıfı basılmalı")

	genis := renderGazeteManset(t, &homeBlock{
		GazeteKapak: gzmKapaklar(), GazeteGun: "2026-11-02", GazeteDar: false,
	})
	assert.Contains(t, genis, `class="gzm"`)
	assert.NotContains(t, genis, "gzm-tek", "geniş bölgede tek slayt zorlanmaz")
	// İki yerleşim de aynı veriyi ve aynı çengelleri taşır.
	assert.Equal(t, bytes.Count([]byte(dar), []byte(`data-gm-slide`)),
		bytes.Count([]byte(genis), []byte(`data-gm-slide`)))
}

func TestGazeteMansetVeriYoksaBasilmaz(t *testing.T) {
	assert.Empty(t, renderGazeteManset(t, &homeBlock{}),
		"kapak yoksa modül hiç basılmamalı (boş kutu gösterme)")
}

func TestGazeteMansetBaslikOzel(t *testing.T) {
	h := renderGazeteManset(t, &homeBlock{
		GazeteKapak: gzmKapaklar(), GazeteBaslik: "GÜNÜN GAZETELERİ",
	})
	assert.Contains(t, h, "GÜNÜN GAZETELERİ")
}

func TestGazeteMansetBaslikGizlenebilir(t *testing.T) {
	h := renderGazeteManset(t, &homeBlock{
		GazeteKapak: gzmKapaklar(), HideHeading: true,
	})
	assert.NotContains(t, h, `class="gzm-title"`, "görünen başlık basılmaz")
	assert.Contains(t, h, `class="sr-only">Gazete Manşetleri</h2>`, "okuyucu için başlık kalır")
	// Başlık gizlenince de "Tümü" bağlantısı erişilebilir kalır.
	assert.Contains(t, h, `class="gzm-all"`)
	assert.Contains(t, h, `data-gm-track`)
}

func TestGazeteMansetPaletteVar(t *testing.T) {
	home := typeKeys(builderTypesFor("home"))
	assert.Contains(t, home, "gazete_manset", "ana sayfa paletinde olmalı")

	// Yazı sayfası paletinde tam sayfa modülleri sunulmaz; gazete modülü
	// de ana sayfa/kategori modülüdür.
	art := typeKeys(builderTypesFor("article"))
	assert.NotContains(t, art, "gazete_manset")
}

// Ayar okuma: aralık 0 = otomatik geçiş kapalı; sınırlar kırpılır.
func TestGazeteMansetAyarOkuma(t *testing.T) {
	bloklar := parseHomeBlocks([]byte(`[
		{"id":"a","type":"gazete_manset","opts":{"zone":"yan","gazete_aralik":0,"gazete_baslik":"Manşetler"}},
		{"id":"b","type":"gazete_manset","opts":{"zone":"ana","gazete_aralik":99}},
		{"id":"c","type":"gazete_manset","opts":{"zone":"yan"}}
	]`))
	require.Len(t, bloklar, 3)

	assert.Equal(t, 0, bloklar[0].GazeteAralik, "0 = otomatik geçiş kapalı")
	assert.Equal(t, "Manşetler", bloklar[0].GazeteBaslik)
	assert.True(t, darBolge(bloklar[0].Zone))

	assert.Equal(t, 30, bloklar[1].GazeteAralik, "üst sınır 30 sn")
	assert.Equal(t, 6, bloklar[2].GazeteAralik, "varsayılan 6 sn")
	assert.Equal(t, 12, bloklar[2].Count, "varsayılan 12 kapak")
}

// CSS/JS sözleşmesi: tek slayt kuralı CSS'te (container query + .gzm-tek)
// ve otomatik geçiş JS'te olmalı. Şablon sınıfı ile stil dosyası
// eşleşmezse modül sessizce yanlış görünür.
func TestGazeteMansetStilSozlesmesi(t *testing.T) {
	css, err := webui.FS.ReadFile("static/gazete-manset.css")
	require.NoError(t, err)
	s := string(css)
	assert.Contains(t, s, ".gzm.gzm-tek .gzm-track { --gzm-adet: 1; }",
		"sağ sütunda tek slayt kuralı")
	// Tek slayt yalnız masaüstünde: 64rem altında sağ sütun tam genişliğe
	// iner, tek kapak orada gereksiz büyük olurdu.
	assert.Contains(t, s, "@media (min-width: 64rem)", "tek slayt masaüstüne sınırlı")
	assert.Contains(t, s, "@container (min-width: 520px)", "geniş bölgede 3 kapak")
	assert.Contains(t, s, "container-type: inline-size", "kapsayıcı sorgusu için gerekli")
	assert.Contains(t, s, "prefers-reduced-motion")

	js, err := webui.FS.ReadFile("static/gazete-manset.js")
	require.NoError(t, err)
	j := string(js)
	assert.Contains(t, j, "setInterval", "otomatik geçiş")
	assert.Contains(t, j, "--gzm-adet", "görünümdeki kapak sayısı CSS'ten okunur")
	assert.Contains(t, j, "prefers-reduced-motion")
	assert.Contains(t, j, "showModal", "kapak büyütme")
	// Sağ sütunda kaydırma adımı ölçülür, sabit piksel yok.
	assert.True(t, strings.Contains(j, "getBoundingClientRect"), "adım ölçülerek bulunur")
}

// Büyütücünün yakınlaştırma özellikleri — kullanıcı isteği: "açılan slide
// zoom özelliğini geliştir".
func TestGazeteZoomOzellikleri(t *testing.T) {
	js, err := webui.FS.ReadFile("static/gazete-manset.js")
	require.NoError(t, err)
	j := string(js)

	// Tekerlek: imlecin altındaki nokta sabit kalır (cx/cy hesaplanır).
	assert.Contains(t, j, "'wheel'")
	assert.Contains(t, j, "Math.exp(", "tekerlek ölçeği üstel (doğal his)")
	assert.Contains(t, j, "ix = (cx - g.x) / eski", "imleç noktası sabitleme")

	// Sürükleme + iki parmak (pinch).
	assert.Contains(t, j, "'pointerdown'")
	assert.Contains(t, j, "'pointermove'")
	assert.Contains(t, j, "kivilcim.length >= 2", "iki parmakla yakınlaştırma")
	assert.Contains(t, j, "Math.hypot(", "parmak mesafesi")

	// Çift tıklama: sığdır ↔ yakınlaştır.
	assert.Contains(t, j, "'dblclick'")

	// Kademeli yakınlaştırma düğmeleri ve yüzde göstergesi.
	assert.Contains(t, j, "data-gzz-arti")
	assert.Contains(t, j, "data-gzz-eksi")
	assert.Contains(t, j, "data-gzz-sigdir")
	assert.Contains(t, j, "'%' + Math.round(g.olcek * 100)")

	// Klavye: gezinme + yakınlaştırma + kapatma.
	assert.Contains(t, j, "case 'ArrowLeft'")
	assert.Contains(t, j, "case 'ArrowRight'")
	assert.Contains(t, j, "case '0'", "0 = ekrana sığdır")
	assert.Contains(t, j, "case '+'")

	// Sınırlar: görüntü sahne dışına taşamaz, aşırı yakınlaşma yok.
	assert.Contains(t, j, "sinirlaKonum()")
	assert.Contains(t, j, "AZAMI_OLCEK = 6")

	// Galeri + modül aynı motoru kullanır: tek delege tıklama.
	assert.Contains(t, j, "[data-gm-zoom]")
	assert.Contains(t, j, "[data-gm-grup]")
}

// Galeri sayfası (/gazete-mansetleri) kaynağa göndermemeli; aynı büyütücüyü
// kullanmalı ve tarihleri Türkçe yazmalı.
func TestGazeteGaleriSayfasi(t *testing.T) {
	ham, err := webui.FS.ReadFile("templates/site_newspapers.html")
	require.NoError(t, err)
	s := string(ham)

	assert.Contains(t, s, `data-gm-zoom`, "galeri de büyütücüyü kullanır")
	assert.Contains(t, s, `data-gm-grup`, "galeride önceki/sonraki gezinmesi")
	assert.NotContains(t, s, `target="_blank"`, "yeni sekme yok")
	assert.NotContains(t, s, `href="{{.ImageURL}}"`, "kaynağa bağlantı yok")
	assert.NotContains(t, s, `class="gzp-kart" data-gm-zoom data-gm-buyuk="{{.ImageURL}}"`+"\n      href",
		"kart bağlantı değil düğme")

	// Türkçe tarih: h1 ve kartlar gazeteTarih kullanır.
	assert.Contains(t, s, `{{gazeteTarih .GazeteTarih}}`)
	assert.Contains(t, s, `{{gazeteTarih .PageDate}}`)
	assert.Contains(t, s, `{{gazeteTarih .}}`, "tarih seçici de Türkçe")

	// Yönetim şablonu korunur (aynı dosyada tanımlı).
	assert.Contains(t, s, `{{define "admin_gazeteler"}}`)
}
