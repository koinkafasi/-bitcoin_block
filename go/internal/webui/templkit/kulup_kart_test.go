package templkit

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// renderKulupKart kartı geçerli ayarla render eder.
//
// Kart templkit içinde (cdKulupKart) tanımlı, bu yüzden test de bu pakette:
// dışa açık bir sarmalayıcı eklemek yalnız test için üretim yüzeyi
// büyütmek olurdu.
func renderKulupKart(t *testing.T) string {
	t.Helper()
	var b bytes.Buffer
	require.NoError(t, cdKulupKart("tr").Render(context.Background(), &b))
	return b.String()
}

// TestKulupKartVarsayilanHedef: ayar boşken kart ESKİ davranışını korumalı.
//
// Panelden adres girilmediği sürece kart üye paneline gitmeye devam eder;
// bu iş öncesinde sabit olan hedef geriye dönük bozulmamalı.
func TestKulupKartVarsayilanHedef(t *testing.T) {
	SetCDEntegrasyon(&CDEntegrasyon{})
	t.Cleanup(func() { SetCDEntegrasyon(nil) })

	html := renderKulupKart(t)
	assert.Contains(t, html, `href="https://traders.bitcosar.com/"`)
	assert.Contains(t, html, `target="_blank"`, "dış adres yeni sekmede açılmalı")
	assert.Contains(t, html, "Milyonerler Kulübü")
	assert.Contains(t, html, "Kripto sinyalleri · Üye girişi")
}

// TestKulupKartSiteIciYol: /funnel gibi site içi hedef AYNI sekmede
// açılmalı. Yeni sekme ziyaretçiyi ana sayfadan koparır.
func TestKulupKartSiteIciYol(t *testing.T) {
	SetCDEntegrasyon(&CDEntegrasyon{KulupURL: "/funnel"})
	t.Cleanup(func() { SetCDEntegrasyon(nil) })

	html := renderKulupKart(t)
	assert.Contains(t, html, `href="/funnel"`)
	assert.NotContains(t, html, `target="_blank"`,
		"site içi hedef yeni sekmede açılmamalı")
}

// TestKulupKartDisAdresYeniSekme: dış adres yeni sekmede + rel=noopener.
func TestKulupKartDisAdresYeniSekme(t *testing.T) {
	SetCDEntegrasyon(&CDEntegrasyon{KulupURL: "https://example.com/satis"})
	t.Cleanup(func() { SetCDEntegrasyon(nil) })

	html := renderKulupKart(t)
	assert.Contains(t, html, `href="https://example.com/satis"`)
	assert.Contains(t, html, `target="_blank"`)
	assert.Contains(t, html, `rel="noopener"`,
		"yeni sekmede window.opener sızdırılmamalı")
}

// TestKulupKartTehlikeliSemaVarsayilanaDuser: doğrulamayı geçmeyen bir
// değer bir şekilde templkit'e ulaşırsa kart onu BASMAMALI.
//
// httpapi tarafı adresi kayıt anında doğruluyor; bu ikinci kapı, ileride
// eklenecek bir çağıran doğrulamayı atlarsa XSS'e dönüşmesin diye.
func TestKulupKartTehlikeliSemaVarsayilanaDuser(t *testing.T) {
	for _, kotu := range []string{
		"javascript:alert(1)",
		"data:text/html,<script>x</script>",
		"//evil.example/x",
		"/\\evil.example",
	} {
		SetCDEntegrasyon(&CDEntegrasyon{KulupURL: kotu})
		html := renderKulupKart(t)
		assert.NotContains(t, strings.ToLower(html), "javascript:", kotu)
		assert.NotContains(t, html, "evil.example", kotu)
		assert.Contains(t, html, `href="https://traders.bitcosar.com/"`,
			"%s: varsayılana düşmeli", kotu)
	}
	SetCDEntegrasyon(nil)
}
