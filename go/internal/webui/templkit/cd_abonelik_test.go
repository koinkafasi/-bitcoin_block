package templkit

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// abonelikRender /abonelik görünümünü metne çevirir.
func abonelikRender(t *testing.T, v CDAbonelikSayfa) string {
	t.Helper()
	var b bytes.Buffer
	require.NoError(t, CDAbonelikSayfaView(v).Render(context.Background(), &b))
	return b.String()
}

// abonelikOrnek iki katmanlı örnek veri.
func abonelikOrnek() CDAbonelikSayfa {
	return CDAbonelikSayfa{
		Shell:        CDShell{SiteName: "Bitcoşar"},
		Title:        "Abonelik Paketleri — Bitcoşar",
		Canonical:    "https://bitcosar.com/abonelik",
		GirisYapildi: true,
		GirisHref:    "/giris?redirect=abonelik",
		Planlar: []CDAbonelikPlan{
			{Slug: "filiz", Ad: "Filiz", Aciklama: "Giriş seviyesi",
				Renk: "#22c55e", Aylik: "150", Yillik: "1500",
				Perks: []string{"Ayrıcalıklı özetler"}, Mevcut: true},
			{Slug: "gumus", Ad: "Gümüş", Aciklama: "Tam erişim",
				Renk: "#94a3b8", Aylik: "750", Yillik: "7500",
				Perks: []string{"Tüm analizler"}},
		},
	}
}

// Kademeler, fiyatlar ve ödeme formu basılmalı; mevcut plan rozetlenmeli.
func TestCDAbonelikKademelerBasilir(t *testing.T) {
	out := abonelikRender(t, abonelikOrnek())

	assert.Contains(t, out, "Abonelik Paketleri")
	assert.Contains(t, out, "Filiz")
	assert.Contains(t, out, "Gümüş")
	// Fiyat panelde girilen tutarın AYNISI olmalı (biçimlenmiş).
	assert.Contains(t, out, "150")
	assert.Contains(t, out, "1500")
	assert.Contains(t, out, "Mevcut planınız", "mevcut plan rozetlenmeli")
	assert.Contains(t, out, "Ayrıcalıklı özetler")
	assert.Contains(t, out, `action="/abonelik/odeme"`)
	assert.Contains(t, out, `name="tckn"`)
}

// Mevcut planın düğmesi devre dışı olmalı: aynı plana yeniden ödeme
// başlatmak kullanıcıyı boşuna iyzico'ya gönderirdi.
func TestCDAbonelikMevcutPlanDevreDisi(t *testing.T) {
	out := abonelikRender(t, abonelikOrnek())
	assert.Contains(t, out, "Aktif plan")
	assert.Contains(t, out, "disabled")
}

// Oturum yoksa: giriş çağrısı olmalı, ödeme formu HİÇ basılmamalı.
func TestCDAbonelikMisafirGirisCagrisi(t *testing.T) {
	v := abonelikOrnek()
	v.GirisYapildi = false
	out := abonelikRender(t, v)

	assert.Contains(t, out, "/giris?redirect=abonelik")
	assert.NotContains(t, out, `action="/abonelik/odeme"`,
		"oturum yokken ödeme formu basılmamalı")
	assert.NotContains(t, out, `name="tckn"`)
}

// Ödeme hata metni görünür olmalı (kullanıcı neden ödeyemediğini bilmeli).
func TestCDAbonelikHataGorunur(t *testing.T) {
	v := abonelikOrnek()
	v.Hata = "Ödeme başlatılamadı: bağlantı hatası"
	out := abonelikRender(t, v)
	assert.Contains(t, out, "Ödeme başlatılamadı")
	assert.Contains(t, out, `role="alert"`)
}

// Plan yoksa dürüst boş durum; uydurma kart basılmaz.
func TestCDAbonelikPlansizBosDurum(t *testing.T) {
	v := abonelikOrnek()
	v.Planlar = nil
	out := abonelikRender(t, v)
	assert.Contains(t, out, "Henüz tanımlı abonelik paketi yok")
	assert.NotContains(t, out, "cd-abn-kart")
}

// Fatura dönemi anahtarı ve SSS her durumda basılır.
func TestCDAbonelikDonemVeSSS(t *testing.T) {
	out := abonelikRender(t, abonelikOrnek())
	assert.Contains(t, out, `data-cycle="monthly"`)
	assert.Contains(t, out, `data-cycle="annual"`)
	assert.Contains(t, out, "Sık sorulan sorular")
	assert.Contains(t, out, "Cayma hakkım var mı?")
	// Sözleşme bağlantıları hukuki zorunluluk.
	assert.Contains(t, out, "/mesafeli-satis-sozlesmesi")
	assert.Contains(t, out, "/on-bilgilendirme-formu")
}

// Ödeme adımı plan kartları yerine iyzico yuvasını basmalı.
func TestCDAbonelikOdemeAdimi(t *testing.T) {
	v := abonelikOrnek()
	v.OdemeAdimi = true
	v.CheckoutHTML = `<form id="iyzico-inline"></form>`
	out := abonelikRender(t, v)

	assert.Contains(t, out, `id="iyzico-checkout"`)
	assert.Contains(t, out, "iyzico-inline")
	assert.NotContains(t, out, "cd-abn-kart", "ödeme adımında plan kartı olmamalı")
}

// Sayfa kendi CSS ve JS dosyalarını istemeli (CSP: satır içi betik yok).
func TestCDAbonelikVarlikBaglari(t *testing.T) {
	out := abonelikRender(t, abonelikOrnek())
	assert.Contains(t, out, "/static/cd-abonelik.css")
	assert.Contains(t, out, "/static/cd-abonelik.js")
	// Sayfaya özel davranış HARİCİ dosyada: CSP `script-src 'self'`
	// satır içi betiği çalıştırmaz. Formdan sonra satır içi betik yok.
	i := strings.Index(out, `id="abonForm"`)
	require.Greater(t, i, 0)
	assert.NotContains(t, out[i:], "<script",
		"ödeme formu satır içi betik taşımamalı")
}
