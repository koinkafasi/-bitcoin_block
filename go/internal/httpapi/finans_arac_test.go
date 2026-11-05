package httpapi

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/repo"
)

func kriptoOrnek() []repo.KriptoRow {
	return []repo.KriptoRow{
		{Symbol: "BTC", Name: "Bitcoin", Image: "https://x/btc.png", Value: 4000000, Delta: 0.56},
		{Symbol: "ETH", Name: "Ethereum", Value: 117000, Delta: -1.2},
		{Symbol: "USDT", Name: "Tether", Value: 49.79, Delta: 0.04},
	}
}

func TestKriptoSatirlar(t *testing.T) {
	rows := kriptoOrnek()

	// "deger" = kaynak sırası (CoinGecko piyasa değeri).
	val := kriptoSatirlar(rows, "deger", 0)
	assert.Equal(t, []string{"BTC", "ETH", "USDT"}, kriptoSembol(val))

	yuk := kriptoSatirlar(rows, "yukselen", 2)
	assert.Equal(t, []string{"BTC", "USDT"}, kriptoSembol(yuk))

	dus := kriptoSatirlar(rows, "dusen", 1)
	assert.Equal(t, []string{"ETH"}, kriptoSembol(dus))

	kod := kriptoSatirlar(rows, "kod", 0)
	assert.Equal(t, []string{"BTC", "ETH", "USDT"}, kriptoSembol(kod))

	// Girdi değişmemeli.
	assert.Equal(t, "BTC", rows[0].Symbol)
	assert.Empty(t, kriptoSatirlar(nil, "deger", 5))
}

func kriptoSembol(rows []repo.KriptoRow) []string {
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.Symbol)
	}
	return out
}

func kurTablosuOrnek() repo.KurTablosu {
	return repo.KurTablosu{
		Kurlar: map[string]float64{
			"TRY": 1, "USD": 49.84, "EUR": 55.69, "GBP": 66.52, "JPY": 0.003168, "SAR": 13.28,
		},
		Adlar: map[string]string{"USD": "Amerikan Doları"},
	}
}

func TestKurCeviriciOlustur(t *testing.T) {
	kt := kurTablosuOrnek()

	v := kurCeviriciOlustur(kt, "", "USD", "TRY", 100)
	require.NotNil(t, v)
	assert.Equal(t, kurCeviriciBaslikVarsayilan, v.Baslik)
	assert.Equal(t, "USD", v.From)
	assert.Equal(t, "TRY", v.To)
	assert.True(t, v.Var)
	assert.InDelta(t, 4984, v.Sonuc, 1)
	assert.NotEmpty(t, v.Para)
	// Referans sırası: USD, EUR, TRY, GBP …
	assert.Equal(t, "USD", v.Para[0].Code)
	assert.Equal(t, "EUR", v.Para[1].Code)
	assert.Equal(t, "TRY", v.Para[2].Code)

	// Geçersiz hedef → TRY'ye düşer (kaynak TRY değilse).
	v2 := kurCeviriciOlustur(kt, "ÇEVİR", "EUR", "YOKBOYLE", 0)
	assert.Equal(t, "EUR", v2.From)
	assert.Equal(t, "TRY", v2.To, "geçersiz hedef TRY'ye düşer")
	assert.Equal(t, float64(kurCeviriciMiktarVarsayilan), v2.Miktar, "0 tutar varsayılana döner")

	// Kaynak geçersiz → ilk geçerli kod.
	v3 := kurCeviriciOlustur(kt, "", "", "", 10)
	assert.NotEmpty(t, v3.From)
	assert.NotEqual(t, v3.From, v3.To)

	// Aynı para birimi seçilemez.
	v4 := kurCeviriciOlustur(kt, "", "USD", "USD", 10)
	assert.NotEqual(t, v4.From, v4.To)

	// Tablo boşsa modül basılmaz.
	assert.Nil(t, kurCeviriciOlustur(repo.KurTablosu{}, "", "USD", "TRY", 10))
}

func kriptoBlogu(rows []repo.KriptoRow) *homeBlock {
	return &homeBlock{
		Type: "kripto_para", Kripto: rows, KriptoKolon: 4,
		KriptoDegisim: true, KriptoIkon: true, KriptoSirala: "deger",
		KriptoGuncel: time.Date(2026, 10, 29, 9, 0, 0, 0, time.UTC),
		D:            &tmplData{LangBase: ""},
	}
}

func TestKriptoModulRender(t *testing.T) {
	tmpl := parseTestTemplates(t)
	b := kriptoBlogu(kriptoOrnek())
	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "render_block", b))
	out := buf.String()

	assert.Contains(t, out, "krp-block")
	assert.Contains(t, out, "krp-grid krp-cols-4")
	assert.Equal(t, 3, bytes.Count([]byte(out), []byte(`class="krp-card`)))
	assert.Contains(t, out, "Bitcoin")
	assert.Contains(t, out, "4.000.000,00", "fiyat TR biçiminde")
	assert.Contains(t, out, "%&#43;0,56")
	assert.Contains(t, out, "krp-up")
	assert.Contains(t, out, "krp-down")
	assert.Contains(t, out, `href="/borsa/kripto"`, "TÜMÜ kripto sayfasına")
	assert.Contains(t, out, "krp-ico-yok", "ikon yedeği her kartta var")
	// Görsel yüklenmezse gizlenir: onerror artık CSP nedeniyle öznitelik
	// olarak yazılmaz, data-hata-gizle + csp-delegated.js ile yapılır.
	assert.Contains(t, out, "data-hata-gizle", "görsel yüklenmezse gizlenir")
	assert.Contains(t, out, "yatırım tavsiyesi değildir")

	// İkon ve yüzde kapatılabilir.
	b.KriptoIkon = false
	b.KriptoDegisim = false
	buf.Reset()
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "render_block", b))
	out = buf.String()
	assert.NotContains(t, out, `class="krp-ico"`)
	assert.NotContains(t, out, "krp-chg")

	// Veri yoksa modül basılmaz.
	buf.Reset()
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "render_block", kriptoBlogu(nil)))
	assert.Empty(t, bytes.TrimSpace(buf.Bytes()))
}

func TestKurCeviriciModulRender(t *testing.T) {
	tmpl := parseTestTemplates(t)
	v := kurCeviriciOlustur(kurTablosuOrnek(), "KUR ÇEVİRİCİ", "USD", "TRY", 100)
	require.NotNil(t, v)
	b := &homeBlock{Type: "kur_cevirici", KurCevirici: v, D: &tmplData{LangBase: ""}}
	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "render_block", b))
	out := buf.String()

	assert.Contains(t, out, "kcr-block")
	assert.Contains(t, out, "KUR ÇEVİRİCİ")
	assert.Contains(t, out, "data-kcr")
	assert.Contains(t, out, `data-kcr-from`)
	assert.Contains(t, out, `data-kcr-to`)
	// 6 para birimi × 2 seçici.
	assert.Equal(t, 12, bytes.Count([]byte(out), []byte("<option")))
	assert.Contains(t, out, `data-rate="49.84"`)
	assert.Contains(t, out, "4.984,0000 TRY", "sunucu başlangıç sonucu (JS'siz de doğru)")
	assert.Contains(t, out, "Güncel döviz kurlarını kullanarak")
	assert.Contains(t, out, "USD — Amerikan Doları", "seçenek etiketi kod + ad")
}

func TestKriptoVeCeviriciPalette(t *testing.T) {
	for _, page := range []string{"home", "kategori", "article"} {
		keys := typeKeys(builderTypesFor(page))
		assert.Contains(t, keys, "kripto_para", "%s paletinde kripto olmalı", page)
		assert.Contains(t, keys, "kur_cevirici", "%s paletinde çevirici olmalı", page)
	}
}

func TestKriptoYerlesimStilGecis(t *testing.T) {
	// Varsayılanlar: ızgara + otomatik + sürekli + 60 px/sn.
	b := parseHomeBlocks([]byte(`[{"id":"k1","type":"kripto_para","opts":{}}]`))[0]
	assert.Equal(t, "izgara", b.KriptoYerlesim)
	assert.Equal(t, "otomatik", b.KriptoStil, "yeni varsayılan: bölgeye göre otomatik")
	assert.Equal(t, "referans", b.KriptoEtkin, "geniş bölgede referans")
	assert.Equal(t, "surekli", b.KriptoGecis)
	assert.Equal(t, 60, b.KriptoHiz)
	assert.Equal(t, 4, b.KriptoKolon)

	// Şerit + kart + adım modeli + hız ön ayarı.
	s2 := parseHomeBlocks([]byte(`[{"id":"k2","type":"kripto_para","opts":{
		"kp_layout":"serit","kp_style":"kart","kp_gecis":"adim",
		"kp_hiz_secim":"120","kp_bekle":6,"kp_kayma":900}}]`))[0]
	assert.Equal(t, "serit", s2.KriptoYerlesim)
	assert.Equal(t, "kart", s2.KriptoStil)
	assert.Equal(t, "adim", s2.KriptoGecis)
	assert.Equal(t, 120, s2.KriptoHiz, "hız ön ayarı px/sn olarak okunur")
	assert.Equal(t, 6, s2.KriptoBekle)
	assert.Equal(t, 900, s2.KriptoKayma)

	// "ozel" → serbest px/sn alanı kullanılır.
	s3 := parseHomeBlocks([]byte(`[{"id":"k3","type":"kripto_para","opts":{
		"kp_hiz_secim":"ozel","kp_speed":175}}]`))[0]
	assert.Equal(t, 175, s3.KriptoHiz)

	// Geçersiz değerler güvenli varsayılana düşer.
	s4 := parseHomeBlocks([]byte(`[{"id":"k4","type":"kripto_para","opts":{
		"kp_layout":"gecersiz","kp_style":"gecersiz","kp_gecis":"gecersiz",
		"kp_bekle":99,"kp_kayma":10}}]`))[0]
	assert.Equal(t, "izgara", s4.KriptoYerlesim)
	assert.Equal(t, "otomatik", s4.KriptoStil, "geçersiz stil → otomatik")
	assert.Equal(t, "referans", s4.KriptoEtkin)
	assert.Equal(t, "surekli", s4.KriptoGecis)
	assert.Equal(t, 30, s4.KriptoBekle, "bekleme üst sınırı 30 sn")
	assert.Equal(t, 200, s4.KriptoKayma, "geçiş alt sınırı 200 ms")

	// Sütun sınırı ve kripto sayısı üst sınırı (100 kayıt evreni).
	s5 := parseHomeBlocks([]byte(`[{"id":"k5","type":"kripto_para","opts":{"count":500,"kp_kolon":"9"}}]`))[0]
	assert.Equal(t, kriptoSayiAzami, s5.Count)
	assert.Equal(t, kriptoKolonEnCok, s5.KriptoKolon)
}

func TestKriptoSeritRender(t *testing.T) {
	tmpl := parseTestTemplates(t)
	b := kriptoBlogu(kriptoOrnek())
	b.KriptoYerlesim = "serit"
	b.KriptoStil = "kart"
	b.KriptoGecis = "adim"
	b.KriptoHiz = 120
	b.KriptoBekle = 5
	b.KriptoKayma = 800
	b.KriptoSure = 30
	b.KriptoCSS = "--krp-dur:30s"

	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "render_block", b))
	out := buf.String()

	assert.Contains(t, out, "krp-block krp-serit krp-style-kart")
	assert.Contains(t, out, `data-marquee`)
	assert.Contains(t, out, `data-model="adim"`)
	assert.Contains(t, out, `data-speed="120"`)
	assert.Contains(t, out, `data-hold="5"`)
	assert.Contains(t, out, `data-glide="800"`)
	assert.Contains(t, out, "--krp-dur:30s")
	// Kayan şerit: liste iki kez basılır, kopya ekran okuyucudan gizlenir.
	assert.Equal(t, 2, bytes.Count([]byte(out), []byte(`class="krp-group"`)))
	assert.Equal(t, 1, bytes.Count([]byte(out), []byte(`class="krp-group" aria-hidden="true"`)))
	assert.Contains(t, out, "krp-frame")
	// Izgarada şerit işaretleri olmamalı.
	b.KriptoYerlesim = "izgara"
	buf.Reset()
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "render_block", b))
	out = buf.String()
	assert.NotContains(t, out, "data-marquee")
	assert.NotContains(t, out, `class="krp-group"`)
	assert.Contains(t, out, "krp-grid krp-cols-4")
}

func TestKriptoSayfaAramaVeStil(t *testing.T) {
	kd := repo.KriptoData{Rows: kriptoOrnek(), Updated: time.Date(2026, 10, 29, 9, 0, 0, 0, time.UTC)}

	hepsi := kriptoSayfaOlustur(kd, "", "")
	assert.Equal(t, "tablo", hepsi.Stil, "varsayılan görünüm tablo (referans)")
	assert.Equal(t, 3, hepsi.Adet)
	assert.Equal(t, "29.10.2026 09:00", hepsi.Guncel)
	assert.Len(t, hepsi.Stiller, 3)

	// Koda göre, Türkçe/küçük harf duyarsız.
	btc := kriptoSayfaOlustur(kd, "BTc", "kart")
	require.Len(t, btc.Rows, 1)
	assert.Equal(t, "BTC", btc.Rows[0].Symbol)
	assert.Equal(t, "kart", btc.Stil)
	assert.Equal(t, "Kart", btc.SeciliStil)

	// Ada göre.
	ad := kriptoSayfaOlustur(kd, "ethereum", "kompakt")
	require.Len(t, ad.Rows, 1)
	assert.Equal(t, "ETH", ad.Rows[0].Symbol)
	// Alt dizgi eşleşmesi: "ether" Tether'i de bulur (istenen davranış).
	assert.Len(t, kriptoSayfaOlustur(kd, "ether", "").Rows, 2)

	// Geçersiz görünüm tabloya döner; sonuç yoksa boş liste.
	v := kriptoSayfaOlustur(kd, "", "gecersiz")
	assert.Equal(t, "tablo", v.Stil)
	assert.Empty(t, kriptoSayfaOlustur(kd, "zzzz", "").Rows)
	assert.Equal(t, 3, kriptoSayfaOlustur(kd, "zzzz", "").Toplam, "toplam filtresiz sayıyı korur")
}

func TestParaKisa(t *testing.T) {
	for _, c := range []struct {
		v    float64
		want string
	}{
		{0, "—"},
		{845.5, "845,50"},
		{12345, "12,3 b"},
		{45_600_000, "45,6 mn"},
		{1_234_000_000, "1,23 mr"},
		{80_210_000_000_000, "80,21 tn"},
	} {
		assert.Equal(t, c.want, paraKisa(c.v), "değer: %v", c.v)
	}
}

func TestHisseHizOnAyar(t *testing.T) {
	// Ön ayar px/sn döndürür.
	if px, ok := hisseHizSecimPx("90"); !ok || px != 90 {
		t.Errorf("90 → 90 px/sn beklenirdi, %d %v", px, ok)
	}
	// "ozel" ve boş ön ayar sayılmaz (serbest alan kullanılır).
	if _, ok := hisseHizSecimPx("ozel"); ok {
		t.Error("ozel ön ayar sayılmamalı")
	}
	if _, ok := hisseHizSecimPx(""); ok {
		t.Error("boş ön ayar sayılmamalı")
	}
	// Çözümleme: ön ayar serbest alanı ezer.
	assert.Equal(t, 30, hisseHizCoz("30", 150, 0, 20))
	assert.Equal(t, 150, hisseHizCoz("ozel", 150, 0, 20))
	// Ön ayar da serbest alan da yoksa varsayılan.
	assert.Equal(t, hisseHizVarsayilan, hisseHizCoz("", 0, 0, 20))
}

func TestKriptoOtomatikBolgeStili(t *testing.T) {
	// Sağ ray + otomatik → rail listesi (kart ızgarası yerine).
	y := parseHomeBlocks([]byte(`[{"id":"k1","type":"kripto_para","opts":{"zone":"yan","kp_style":"otomatik"}}]`))[0]
	assert.True(t, y.KriptoDar, "yan bölge dar sayılır")
	assert.Equal(t, "rail", y.KriptoEtkin)
	assert.True(t, y.KriptoListe)

	// Geniş bölge + otomatik → referans ızgara.
	g := parseHomeBlocks([]byte(`[{"id":"k2","type":"kripto_para","opts":{"zone":"govde","kp_style":"otomatik"}}]`))[0]
	assert.False(t, g.KriptoDar)
	assert.Equal(t, "referans", g.KriptoEtkin)
	assert.False(t, g.KriptoListe)

	// Açık seçim korunur: yan + kart → kart (otomatik değil).
	m := parseHomeBlocks([]byte(`[{"id":"k3","type":"kripto_para","opts":{"zone":"yan","kp_style":"kart"}}]`))[0]
	assert.True(t, m.KriptoDar)
	assert.Equal(t, "kart", m.KriptoEtkin)
	assert.False(t, m.KriptoListe)

	// Eski kayıtta stil yok → varsayılan otomatik.
	d := parseHomeBlocks([]byte(`[{"id":"k4","type":"kripto_para","opts":{}}]`))[0]
	assert.Equal(t, "otomatik", d.KriptoStil)
	assert.Equal(t, "referans", d.KriptoEtkin)
}

func TestKriptoRailRender(t *testing.T) {
	tmpl := parseTestTemplates(t)
	b := kriptoBlogu(kriptoOrnek())
	b.KriptoEtkin = "rail"
	b.KriptoDar = true
	b.KriptoListe = true

	var buf bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&buf, "render_block", b))
	out := buf.String()

	assert.Contains(t, out, "krp-style-rail")
	assert.Contains(t, out, "krp-dar")
	assert.Contains(t, out, "krp-list krp-list-rail")
	assert.Equal(t, 3, bytes.Count([]byte(out), []byte(`class="krp-row `)))
	assert.Contains(t, out, "krp-row-code")
	assert.NotContains(t, out, "data-marquee", "rail listesi kaymaz")
	assert.NotContains(t, out, "krp-grid", "rail kart ızgarası basmaz")
}

func TestHisseOtomatikBolgeStili(t *testing.T) {
	// Sağ ray + otomatik → dar (tek satır kompakt).
	y := parseHomeBlocks([]byte(`[{"id":"h1","type":"hisseler","opts":{"zone":"yan","hs_style":"otomatik"}}]`))[0]
	assert.True(t, y.HisseDar)
	assert.Equal(t, "dar", y.HisseEtkin)

	// Geniş bölge + otomatik → referans.
	g := parseHomeBlocks([]byte(`[{"id":"h2","type":"hisseler","opts":{"zone":"ana","hs_style":"otomatik"}}]`))[0]
	assert.False(t, g.HisseDar)
	assert.Equal(t, "referans", g.HisseEtkin)
}
