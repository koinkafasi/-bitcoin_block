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
		"--cd-radius: 4px",
		"--cd-section-gap: 32px",
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

// TestCDModulBasligi: modül başlığı 16/24 w500 ve BÜYÜK HARF DEĞİL.
//
// NEDEN TEST: eski sürüm 1.05rem + UPPERCASE + 2 px alt çizgi kullanıyordu;
// spec §4 "modül başlıklarını küçük tutuyor" diyor. Uppercase geri gelirse
// hiyerarşi CoinDesk'ten kopar.
func TestCDModulBasligi(t *testing.T) {
	css := cdCSS(t)
	baslikBlok := css[strings.Index(css, ".cd-section-head h2"):]
	baslikBlok = baslikBlok[:strings.Index(baslikBlok, "}")]
	assert.Contains(t, baslikBlok, "font-size: 16px")
	assert.Contains(t, baslikBlok, "font-weight: 500")
	assert.NotContains(t, baslikBlok, "uppercase", "CoinDesk modül başlığı büyük harf değil")
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
// Eşik 32 KB (6 Kasım 2026): ölçülen değerler px olarak yazıldığı için
// (36px/44px gibi) rem kısaltmasından uzun; ayrıca her jetonun neden o
// değeri taşıdığı yorumda duruyor. Gerçek maliyet sıkıştırılmış boyut:
// bu dosya gzip'te ~6 KB — tek bloklayıcı CSS için kabul edilebilir.
func TestCDButce(t *testing.T) {
	css := cdCSS(t)
	assert.Less(t, len(css), 32*1024, "cd.css bütçe aşıldı: %d bayt", len(css))
}

// TestCDShellSiniflari: şablon ile CSS aynı sınıflarda buluşuyor.
func TestCDShellSiniflari(t *testing.T) {
	css := cdCSS(t)
	for _, sinif := range []string{
		"cd-ticker", "cd-topbar", "cd-footer", "cd-logo",
		"cd-card", "cd-list", "cd-article", "cd-source", "cd-ad",
		"cd-up", "cd-down", "cd-section-head", "cd-tema", "cd-daha",
	} {
		assert.Contains(t, css, "."+sinif, "CSS sınıfı eksik: %s", sinif)
	}
}
