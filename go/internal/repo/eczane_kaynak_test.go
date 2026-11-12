package repo

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Gerçek sayfadan alınmış kayıt parçası (yapı birebir korundu).
const eczaneOrnekHTML = `<div class="rounded-2xl border">
<span class="...">Nöbetçi Eczane</span>
<div class="mb-2.5"><a class="block font-bold ..." title="Aksoy Eczanesi" href="/ankara-cankaya-aksoy-eczanesi">Aksoy Eczanesi</a></div>
<div class="flex flex-wrap mb-3"><span class="...">Ankara<!-- --> / <!-- -->Çankaya</span><span class="..."><svg
 class="w-3.5 h-3.5"><path d="M5.05 4.05"/></svg><span>Dikmen</span></span></div>
<div class="flex"><svg class="h-4 w-4"></svg><p class="text-sm sm:text-base line-clamp-2 leading-relaxed">Atatürk Mah. Osmanlı Cad. No:78/D Kahramankazan Ankara</p></div>
<div class="rounded-lg px-2.5"><span class="font-bold mr-1">Tarif:</span><span class="line-clamp-2">Onyx Ankara Avm Gratis Yanı</span></div>
<a href="tel:+905010086699" rel="noopener noreferrer nofollow" class="flex items-center">0 501 008 66 99</a>
</div>
<div class="rounded-2xl border">
<div class="mb-2.5"><a class="block font-bold ..." title="Canberk Eczanesi" href="/ankara-kecioren-canberk-eczanesi">Canberk Eczanesi</a></div>
<div class="flex flex-wrap mb-3"><span class="...">Ankara<!-- --> / <!-- -->Keçiören</span><span class="..."><svg
 class="w-3.5 h-3.5"><path d="M1"/></svg><span>Etlik</span></span></div>
<div class="flex"><svg class="h-4 w-4"></svg><p class="text-sm sm:text-base line-clamp-2 leading-relaxed">Özlem Cad. No:9/B Keçiören Ankara</p></div>
<a href="tel:+905332364334" rel="noopener noreferrer nofollow" class="flex items-center">0 533 236 43 34</a>
</div>`

func TestEczaneCozHTML(t *testing.T) {
	list := eczaneCozHTML("ankara", []byte(eczaneOrnekHTML))
	require.Len(t, list, 2, "iki kayıt beklenirdi")

	assert.Equal(t, "Aksoy Eczanesi", list[0].Ad)
	assert.Equal(t, "Çankaya", list[0].Ilce)
	assert.Equal(t, "0 501 008 66 99", list[0].Tel)
	// Adres + mahalle + tarif birleşir.
	assert.Contains(t, list[0].Adres, "Osmanlı Cad. No:78/D")
	assert.Contains(t, list[0].Adres, "Dikmen")
	assert.Contains(t, list[0].Adres, "Tarif: Onyx Ankara")
	// Varlık kaçışı çözülür, çift boşluk kalmaz.
	assert.NotContains(t, list[0].Adres, "  ")
	assert.NotContains(t, list[0].Adres, "&#")

	assert.Equal(t, "Canberk Eczanesi", list[1].Ad)
	assert.Equal(t, "Keçiören", list[1].Ilce)
	assert.Equal(t, "0 533 236 43 34", list[1].Tel)
}

func TestEczaneCozHTMLBosVeBozuk(t *testing.T) {
	// Telefon bağlantısı yoksa kayıt sayılmaz (her eczanede var; çapa odur).
	assert.Empty(t, eczaneCozHTML("ankara", []byte("<html><body>eczane yok</body></html>")))
	// Adı olmayan kayıt atlanır.
	eksik := `<a href="tel:+905010086699">tel</a>`
	assert.Empty(t, eczaneCozHTML("ankara", []byte(eksik)))
}

func TestEczaneCozHTMLCiftKayitTekSatir(t *testing.T) {
	// Aynı eczanenin iki tel bağlantısı (birebir kaynak yapısı: künye + Ara).
	kart := func(ad, slug, tel string) string {
		return `<div class="rounded-2xl border">
<div class="mb-2.5"><a class="block font-bold" title="` + ad + `" href="/ankara-cankaya-` + slug + `-eczanesi">` + ad + `</a></div>
<div class="flex flex-wrap mb-3"><span class="x">Ankara<!-- --> / <!-- -->Çankaya</span></div>
<div class="flex"><p class="text-sm line-clamp-2 leading-relaxed">Kızılay Mah. No:1</p></div>
<a href="tel:+90` + tel + `" rel="noopener noreferrer nofollow" class="flex items-center">tel</a>
</div>
<div class="p-4 pt-0 grid grid-cols-2 gap-2"><a href="/yol-tarifi">Yol Tarifi Al</a>
<a href="tel:+90` + tel + `" rel="noopener noreferrer nofollow" class="flex items-center">Ara</a></div>`
	}
	ham := kart("Aksoy Eczanesi", "aksoy", "5010086699") +
		kart("Canberk Eczanesi", "canberk", "5332364334")

	list := eczaneCozHTML("ankara", []byte(ham))
	require.Len(t, list, 2, "aynı eczanenin iki tel bağlantısı tek kayda inmeli")
	assert.Equal(t, "Aksoy Eczanesi", list[0].Ad)
	assert.Equal(t, "Canberk Eczanesi", list[1].Ad)
}

// Aynı ilçede aynı adlı iki AYRI eczane (farklı telefon) korunmalı.
func TestEczaneTekrarsizFarkliTelefonuKorur(t *testing.T) {
	girdi := []Eczane{
		{Ad: "Can Eczanesi", Ilce: "Bodrum", Tel: "0 252 387 98 23"},
		{Ad: "Can Eczanesi", Ilce: "Marmaris", Tel: "0 252 522 31 10"},
		{Ad: "Can Eczanesi", Ilce: "Bodrum", Tel: "0 252 387 98 23"}, // tekrar
	}
	out := eczaneTekrarsiz(girdi)
	require.Len(t, out, 2, "farklı telefonlu adaşlar korunur, aynı kayıt tekilleşir")
	assert.Equal(t, "Marmaris", out[1].Ilce)
}

func TestTurkceTelefon(t *testing.T) {
	assert.Equal(t, "0 501 008 66 99", turkceTelefon("5010086699"))
	// Beklenmeyen uzunluk bozulmadan döner.
	assert.Equal(t, "123", turkceTelefon("123"))
}

func TestEczaneKaynakURL(t *testing.T) {
	assert.Equal(t, "https://eczaneler.org/ankara-nobetci-eczaneleri", EczaneKaynakURL("ankara"))
	assert.Equal(t, "https://eczaneler.org/afyonkarahisar-nobetci-eczaneleri",
		EczaneKaynakURL("afyonkarahisar"),
		"cities.slug ASCII katlaması bu desenle uyuşmalı (ölçüldü: 200)")
}

// TestEczaneKaynakSozlesmesi — kaynak seçimi ve günlük yenileme kuralı.
func TestEczaneKaynakSozlesmesi(t *testing.T) {
	src := okuPaketDosya(t, "services_fetch.go")

	// Gerçek kaynak bağlı olmalı.
	assert.Contains(t, src, "eczaneEczanelerOrg(ctx, c.Slug)",
		"eczaneler.org kaynağı bağlı değil → 81 il demo kalır")
	// Günde bir: bugün gerçek veri varsa atla.
	assert.Contains(t, src, "mevcut.Tarih == bugun",
		"günlük yenileme kontrolü yok: her 30 dakikada 81 istek atılır")
	// Kaynak adı saklanır (sayfada gösterilebilir).
	assert.Contains(t, src, `Kaynak: kaynak`)
	assert.Contains(t, src, `kaynak = "ulasilamadi"`,
		"ulaşılamayan kaynak ayrı işaretlenmeli (sayfa dürüst boş durum gösterir)")
	// Kelimeye değil ÇAĞRIYA bakılır: yukarıdaki tarihsel yorum (ve bu
	// test) adı anmak zorunda, yoksa neden kaldırıldığı kaybolur.
	assert.NotContains(t, src, "list = demoEczaneler(",
		"uydurma eczane üreteci yeniden çağrılıyor — SAĞLIK verisinde temsilî liste basılmaz")
	assert.NotContains(t, src, "func demoEczaneler(",
		"uydurma eczane üreteci geri gelmiş — tanımı da kalmamalı")
}

func TestEczaneGercekSayfa(t *testing.T) {
	raw, err := os.ReadFile("/tmp/eorg-ankara.html")
	if err != nil || len(raw) < 10000 {
		t.Skip("gerçek sayfa yok (/tmp/eorg-ankara.html)")
	}
	list := eczaneCozHTML("ankara", raw)
	require.Greater(t, len(list), 20, "gerçek sayfada 20+ eczane beklenirdi")

	for _, e := range list {
		assert.True(t, strings.HasSuffix(e.Ad, "Eczanesi"), "ad biçimi bozuk: %q", e.Ad)
		assert.NotEmpty(t, e.Adres, "adres boş: %q", e.Ad)
		assert.True(t, strings.HasPrefix(e.Tel, "0"), "telefon bozuk: %q", e.Tel)
		assert.NotEmpty(t, e.Ilce, "ilçe boş: %q", e.Ad)
	}
	t.Logf("çözülen eczane: %d · ilk: %s / %s / %s",
		len(list), list[0].Ad, list[0].Ilce, list[0].Tel)
}

// okuPaketDosya test için paket dizinindeki dosyayı okur.
func okuPaketDosya(t *testing.T, ad string) string {
	t.Helper()
	b, err := os.ReadFile(ad)
	require.NoError(t, err, "dosya okunamadı: %s", ad)
	return string(b)
}

func TestEczaneVeriYokDurumu(t *testing.T) {
	src := okuPaketDosya(t, "services_fetch.go")

	// (b) kaynak açıldı/veri yok ile (c) ulaşılamadı ayrılmalı.
	assert.Contains(t, src, "bos := false")
	assert.Contains(t, src, `bos = true`,
		"kaynak erişildi ama liste yoksa 'veri yok' işaretlenmeli (demo değil)")
	assert.Contains(t, src, "Bos  bool") // gofmt hizalamasi: iki bosluk

	// "Veri yok" iken demo rozeti de görünmemeli: kaynak var, liste yok.
	// İki sinyal (rozet + "yayınlanmadı" kartı) çelişirdi.
	assert.Contains(t, src, "bos = true\n\t\t\t\tdemo = false",
		"bos durumunda demo=false olmalı, yoksa sayfa 'demo (kaynak bekliyor)' rozetini gösterir")

	// Boş il 2 saatte bir yeniden denenir; her turda vurulmaz.
	assert.Contains(t, src, "saatIkiSaatIcinde(mevcut.Saat)")
	assert.Contains(t, src, "func saatIkiSaatIcinde(")
}

func TestSaatIkiSaatIcinde(t *testing.T) {
	// Geçersiz damga: yeniden dene (güvenli taraf).
	assert.False(t, saatIkiSaatIcinde(""))
	assert.False(t, saatIkiSaatIcinde("abc"))
}
