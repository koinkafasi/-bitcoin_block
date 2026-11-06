package templkit

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/webui"
)

// CoinDesk tasarım sistemi kalıcı kapısı.
//
// NEDEN: tasarım kararları görsel doğrulama gerektirir ve sessizce
// bozulabilir (birisi gölge ekler, ritmi değiştirir, token'ı siler).
//
// KAYNAK: `docs/COINDESK-TASARIM-SPEC.md` — 6 Kasım 2026'da gerçek
// Chrome'da `getComputedStyle` ile ÖLÇÜLDÜ. Eski kapı `COINDESK-SPEC.md`
// §2'yi tutuyordu; o liste eski bir build'dendi ve yeni ölçümde yanlış
// çıktı (mavi bağlantı, 2 px köşe, proxima-nova). Kapı artık ÖLÇÜLEN
// değerleri tutar.
func cdCSS(t *testing.T) string {
	t.Helper()
	b, err := webui.FS.ReadFile("static/cd.css")
	require.NoError(t, err, "static/cd.css okunamadı")
	return string(b)
}

// TestCDTasarimTokenlari: ölçülen renk ve ritim değerleri yerinde.
//
// Her satır spec §5'teki semantik ad → değer tablosundan.
func TestCDTasarimTokenlari(t *testing.T) {
	css := cdCSS(t)
	for _, token := range []string{
		// zemin (surface-*)
		"--cd-bg: #ffffff",
		"--cd-surface-soft: #f4f4f4",
		"--cd-surface-action: #d9d9d9",
		"--cd-surface-disable: #e9e9e9",
		// metin (text-*)
		"--cd-fg: #262626",
		"--cd-fg-strong: #000000",
		"--cd-muted: #676767",
		// ayırıcı (border-*)
		"--cd-border: #d9d9d9",
		"--cd-border-strong: #a6a6a6",
		// finans
		"--cd-up: #218410",
		"--cd-down: #bb020a",
		// marka — temayla dönmez
		"--cd-brand: #ffd200",
		// kapsayıcı ve köşe (ölçüldü: 1512 / 4 px)
		"--cd-max: 1512px",
		"--cd-max-section: 1432px",
		"--cd-max-footer: 1280px",
		"--cd-radius: 4px",
		"--cd-section-gap: 24px",
		// semantik font ölçeği (spec §4.1)
		"--cd-font-headline: 36px/44px",
		"--cd-font-body-lg: 22px/30px",
		"--cd-font-metadata-lg: 14px/22px",
		// sticky iki katman toplamı
		"--cd-sticky-h: 140px",
	} {
		assert.Contains(t, css, token, "ölçülen token eksik: %s", token)
	}
}

// TestCDEskiYanlisDegerlerYok: spec §0'da "yanlış" işaretlenen değerler
// geri dönmemeli. Bu değerler bir kez ölçümsüz kopyalanmıştı; geri
// gelirlerse görünüm sessizce CoinDesk'ten uzaklaşır.
//
// Yorumlar AYIKLANIR: dosya bu değerlerin neden kaldırıldığını yorumda
// anlatıyor ("eski mavi #4d90fe" gibi). Kural bildirimler için geçerli —
// yoksa kapı, tarihçeyi anlatan doğru bir yorumu da kırmızıya çevirirdi.
func TestCDEskiYanlisDegerlerYok(t *testing.T) {
	css := cdYorumsuz(cdCSS(t))
	for _, eski := range []string{
		"#4d90fe",          // mavi bağlantı: sitede hiç yok
		"#68b631",          // eski yükseliş rengi
		"#ff0062",          // eski düşüş rengi
		"proxima-nova",     // font değişti (Neue Haas)
		"--cd-warn",        // adı yanlıştı: uyarı değil, MARKA
		"--cd-radius: 2px", // 2 px kullanılmıyor
	} {
		assert.NotContains(t, css, eski, "eski/yanlış değer geri gelmiş: %s", eski)
	}
}

// cdYorumsuz CSS'ten /* … */ yorumlarını çıkarır.
func cdYorumsuz(css string) string {
	var b strings.Builder
	for {
		i := strings.Index(css, "/*")
		if i < 0 {
			b.WriteString(css)
			return b.String()
		}
		b.WriteString(css[:i])
		j := strings.Index(css[i:], "*/")
		if j < 0 {
			return b.String()
		}
		css = css[i+j+2:]
	}
}

// TestCDTasarimKurallari: CoinDesk görsel dili.
func TestCDTasarimKurallari(t *testing.T) {
	css := cdCSS(t)
	// Gölge KARTLARDA yok; yalnız bindirme katmanında. Bunu iki kuralla
	// tutuyoruz: (1) gölge yalnız jeton üzerinden gelir, (2) box-shadow
	// kullanan tek yer o jeton ve onu tüketen açılır panel.
	for _, satir := range strings.Split(css, "\n") {
		s := strings.TrimSpace(satir)
		if strings.HasPrefix(s, "box-shadow:") && !strings.Contains(s, "var(--cd-shadow-overlay)") {
			t.Errorf("gölge yalnız --cd-shadow-overlay jetonuyla kullanılmalı: %s", s)
		}
	}
	assert.Contains(t, css, "--cd-shadow-overlay: 0 0 10px rgba(0, 0, 0, 0.4)")
	// Erişilebilirlik: hareket azaltma bloğu zorunlu.
	assert.Contains(t, css, "prefers-reduced-motion")
	// Koyu tema: kullanıcı anahtarı (html.dark) + medya sorgusu ilk değer.
	assert.Contains(t, css, "html.dark")
	assert.Contains(t, css, "prefers-color-scheme: dark")
	// Medya sorgusu kullanıcı seçimini EZMEMELİ.
	assert.Contains(t, css, ":root:not(.light):not(.dark)")
}

// TestCDMakaleTipografi: makale ölçüleri — spec §11'in "en görünür fark"
// dediği kalemler.
func TestCDMakaleTipografi(t *testing.T) {
	css := cdCSS(t)
	for _, kural := range []struct{ ad, deger string }{
		{"makale sütunu 680px", ".cd-article { max-width: 680px"},
		{"gövde 22px", "font-size: 22px;"},
		{"gövde paragraf arası 32px", ".cd-article__body p { margin: 0 0 32px; }"},
		{"ara başlık 28px", "font-size: 28px;"},
		{"makale H1 36/44", "line-height: 44px;"},
	} {
		assert.Contains(t, css, kural.deger, "%s eksik", kural.ad)
	}
	// Bölüm H1'i makale H1'inden AĞIR (700 vs 500) — ölçülen fark.
	assert.Contains(t, css, ".cd-section-title h1 { margin: 0 0 6px; font-size: 36px; line-height: 44px; font-weight: 700; }")
}

// TestCDModulBasligi: modül başlığı **14/21 w500**, altında çizgi YOK.
//
// NEDEN TEST: iki tur boyunca 16/24 w500 + `border-bottom` + 10 px dolgu
// kullanıldı. 6 Kasım 2026'da coindesk.com/tr'de dört modül başlığının
// dördü de `font-sans text-sm font-medium` = 14/21 w500 ölçüldü ve
// hiçbirinde alt çizgi yoktu. Uppercase ya da çizgi geri gelirse
// hiyerarşi CoinDesk'ten kopar.
func TestCDModulBasligi(t *testing.T) {
	css := cdCSS(t)
	baslikBlok := css[strings.Index(css, ".cd-section-head h2"):]
	baslikBlok = baslikBlok[:strings.Index(baslikBlok, "}")]
	assert.Contains(t, baslikBlok, "font-size: 14px")
	assert.Contains(t, baslikBlok, "line-height: 21px")
	assert.Contains(t, baslikBlok, "font-weight: 500")
	assert.NotContains(t, baslikBlok, "uppercase", "CoinDesk modül başlığı büyük harf değil")

	// Alt çizgi olmamalı: ölçülen dört başlıkta da `border-bottom: 0`.
	kapsayici := css[strings.Index(css, ".cd-section-head {"):]
	kapsayici = kapsayici[:strings.Index(kapsayici, "}")]
	assert.NotContains(t, kapsayici, "border-bottom", "modül başlığının altında çizgi yok (ölçüldü)")
}

// TestCDEtiketlerDuz: CoinDesk'te etiketler PILL değil, düz bağlantı.
func TestCDEtiketlerDuz(t *testing.T) {
	css := cdCSS(t)
	i := strings.Index(css, ".cd-tag {")
	require.GreaterOrEqual(t, i, 0, ".cd-tag kuralı yok")
	blok := css[i:]
	blok = blok[:strings.Index(blok, "}")]
	assert.Contains(t, blok, "border: 0", "etiket çerçevesi olmamalı")
	assert.Contains(t, blok, "padding: 0", "etiket dolgusu olmamalı")
}

// TestCDButce: CSS bütçesi. Tasarım sistemi kalınlaştıkça dosya büyür;
// bütçe aşıldığında bilinçli bir güncelleme gerekir (sessiz kabul yok).
//
// Eşik 52 KB (6 Kasım 2026, beşinci tur): hero üç kolonu (ray modülleri)
// ve bölüm bloklarının 2×2 ızgarası eklendi; eski 923/461 blok kuralları
// kaldırıldı. Değerler px olarak yazılı ve her jetonun gerekçesi yorumda
// duruyor. Gerçek maliyet sıkıştırılmış boyut: bu dosya gzip'te ~9 KB —
// tek bloklayıcı CSS için kabul edilebilir.
func TestCDButce(t *testing.T) {
	css := cdCSS(t)
	assert.Less(t, len(css), 52*1024, "cd.css bütçe aşıldı: %d bayt", len(css))
}

// TestCDModulRitmi: modüller MARGIN YAZMAZ, ritim ebeveynin gap'inden
// gelir ve ayırıcı İKİ KADEMELİ (spec §6.1, §6.4).
//
// NEDEN TEST: margin ile ritim kurmak çalışır ama modül yer değiştirdiğinde
// (order-* / tek HTML iki kompozisyon) çift boşluk üretir. Bu kural
// sessizce bozulabilir — kimse "margin-top: 40px" eklendiğinde hata görmez.
func TestCDModulRitmi(t *testing.T) {
	css := cdCSS(t)
	i := strings.Index(css, ".cd-mod {")
	require.GreaterOrEqual(t, i, 0, ".cd-mod kuralı yok")
	blok := css[i:]
	blok = blok[:strings.Index(blok, "}")]
	assert.NotContains(t, blok, "margin", "modül dışına margin yazmamalı")
	assert.Contains(t, blok, "border-top: 1px solid var(--cd-border-strong)",
		"modül ayırıcısı GÜÇLÜ kademe olmalı")
	assert.Contains(t, blok, "padding-top: var(--cd-mod-pt)")

	// Ana kolon modülleri gap ile dizer.
	i = strings.Index(css, ".cd-main {")
	require.GreaterOrEqual(t, i, 0, ".cd-main kuralı yok")
	main := css[i:]
	main = main[:strings.Index(main, "}")]
	assert.Contains(t, main, "gap: var(--cd-section-gap)")
}

// TestCDUcGenislikKademesi: gövde 1512 · bölüm kapsayıcısı 1432 (içerik
// 1384 = 1432 − 2×24 dolgu) · footer 1280 (spec §2.2 — üç ayrı kademe).
func TestCDUcGenislikKademesi(t *testing.T) {
	css := cdCSS(t)
	assert.Contains(t, css, "--cd-max: 1512px")
	assert.Contains(t, css, "--cd-max-section: 1432px")
	assert.Contains(t, css, "--cd-max-footer: 1280px")
	assert.Contains(t, css, ".cd-wrap--bolum { max-width: var(--cd-max-section); }")
	assert.Contains(t, css, ".cd-footer__ic {")
	i := strings.Index(css, ".cd-footer__ic {")
	blok := css[i:]
	blok = blok[:strings.Index(blok, "}")]
	assert.Contains(t, blok, "max-width: var(--cd-max-footer)")
}

// TestCDFooterTipografi: sütun başlığı 14/22 w500 capitalize, bağlantı
// 12/16 w400 (spec §7.4). Eski sürüm 12 px UPPERCASE idi.
func TestCDFooterTipografi(t *testing.T) {
	css := cdCSS(t)
	i := strings.Index(css, ".cd-footer h3 {")
	require.GreaterOrEqual(t, i, 0)
	blok := css[i:]
	blok = blok[:strings.Index(blok, "}")]
	assert.Contains(t, blok, "font-size: 14px")
	assert.Contains(t, blok, "font-weight: 500")
	assert.Contains(t, blok, "text-transform: capitalize")
	// Yorumlar ayıklanır: blok, eski sürümün "uppercase" olduğunu ANLATIYOR.
	assert.NotContains(t, cdYorumsuz(blok), "uppercase",
		"CoinDesk footer başlığı büyük harf değil")

	i = strings.Index(css, ".cd-footer a {")
	require.GreaterOrEqual(t, i, 0)
	blok = css[i:]
	blok = blok[:strings.Index(blok, "}")]
	assert.Contains(t, blok, "font-size: 12px")
	assert.Contains(t, blok, "line-height: 16px")
}

// TestCDHeroKirilma: hero 1284'te tek kolona düşer (spec §7A — 1280 değil).
func TestCDHeroKirilma(t *testing.T) {
	css := cdCSS(t)
	assert.Contains(t, css, "@media (max-width: 1283px)")
	i := strings.Index(css, ".cd-hero {")
	require.GreaterOrEqual(t, i, 0)
	blok := css[i:]
	blok = blok[:strings.Index(blok, "}")]
	assert.Contains(t, blok, "column-gap: 80px", "sütun aralığı 80 px (dikey çizginin ortası)")
	assert.Contains(t, blok, "padding: 32px 0 16px")
}

// TestCDOzetKutusu: iki iç içe yarıçap + 5 px nokta maddesi (spec §10.2.2).
func TestCDOzetKutusu(t *testing.T) {
	css := cdCSS(t)
	assert.Contains(t, css, "--cd-radius-ozet: 7px")
	assert.Contains(t, css, "--cd-radius-ozet-ic: 5px")
	i := strings.Index(css, ".cd-ozet__govde li::before {")
	require.GreaterOrEqual(t, i, 0, "madde noktası kuralı yok")
	blok := css[i:]
	blok = blok[:strings.Index(blok, "}")]
	assert.Contains(t, blok, "width: 5px")
	assert.Contains(t, blok, "border-radius: 9999px")
	// JS'siz: <details> kullanılmalı, buton+state değil.
	assert.Contains(t, css, ".cd-ozet[open]")
}

// TestCDEylemPilleri: 32 px yüksek, 9999 px yarıçap (spec §10.2.3).
func TestCDEylemPilleri(t *testing.T) {
	css := cdCSS(t)
	i := strings.Index(css, ".cd-pill {")
	require.GreaterOrEqual(t, i, 0)
	blok := css[i:]
	blok = blok[:strings.Index(blok, "}")]
	assert.Contains(t, blok, "height: 32px")
	assert.Contains(t, blok, "border-radius: 9999px")
	assert.Contains(t, blok, "font-size: 14px")
}

// TestCDTamGenislikBant: reklam yuvası kenardan kenara (spec §2.1).
//
// NEDEN `overflow-x: clip`: 100vw bant yatay kaydırma üretir; `hidden`
// ile kapatmak sticky'yi bozar (kaydırma kabı yaratır). Bu ayrımı test
// tutuyor, çünkü ikisi de "taşmayı kes" gibi görünüyor.
func TestCDTamGenislikBant(t *testing.T) {
	css := cdCSS(t)
	assert.Contains(t, css, "overflow-x: clip")
	assert.NotContains(t, css, "overflow-x: hidden")
	i := strings.Index(css, ".cd-tam {")
	require.GreaterOrEqual(t, i, 0)
	blok := css[i:]
	blok = blok[:strings.Index(blok, "}")]
	assert.Contains(t, blok, "width: 100vw")
	assert.Contains(t, blok, "padding: 32px 0")
}

// TestCDShellSiniflari: şablon ile CSS aynı sınıflarda buluşuyor.
func TestCDShellSiniflari(t *testing.T) {
	css := cdCSS(t)
	for _, sinif := range []string{
		"cd-ticker", "cd-topbar", "cd-footer", "cd-logo",
		"cd-card", "cd-list", "cd-article", "cd-source", "cd-ad",
		"cd-up", "cd-down", "cd-section-head", "cd-tema", "cd-daha",
		"cd-main", "cd-mod", "cd-hero", "cd-wrap--bolum", "cd-tam",
		"cd-ozet", "cd-pill", "cd-aksiyon", "cd-footer__ic", "cd-tuy",
	} {
		assert.Contains(t, css, "."+sinif, "CSS sınıfı eksik: %s", sinif)
	}
}
