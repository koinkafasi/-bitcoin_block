package httpapi

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPanoNoktaEkseniDogru(t *testing.T) {
	// En büyük değer kutunun TEPESİNE, en küçük değer TABANA gelmeli.
	// (SVG'de y aşağı büyür; ters çevirmeyi unutmak klasik hata.)
	pts := panoNokta([]int{0, 10}, 100, 100, 10)
	require.Len(t, pts, 2)
	assert.Greater(t, pts[0][1], pts[1][1], "büyük değer daha yukarıda (küçük y) olmalı")
	assert.InDelta(t, 90, pts[0][1], 0.2, "sıfır değer tabanda")
	assert.InDelta(t, 10, pts[1][1], 0.2, "maksimum değer tepede")
	assert.InDelta(t, 10, pts[0][0], 0.2)
	assert.InDelta(t, 90, pts[1][0], 0.2)
}

func TestPanoNoktaSifirSeri(t *testing.T) {
	// Hepsi sıfırsa bölme olmamalı ve çizgi tabanda kalmalı.
	pts := panoNokta([]int{0, 0, 0}, 100, 100, 10)
	require.Len(t, pts, 3)
	for _, p := range pts {
		assert.InDelta(t, 90, p[1], 0.2)
	}
}

func TestPanoEgriGecerliYol(t *testing.T) {
	pts := panoNokta([]int{3, 9, 4, 12}, 200, 100, 6)
	d := panoEgri(pts)
	require.True(t, strings.HasPrefix(d, "M"), "yol M ile başlamalı")
	assert.Equal(t, 3, strings.Count(d, "C"), "her parça için bir C (n-1)")
	assert.NotContains(t, d, "NaN")
	assert.NotContains(t, d, "Inf")
}

func TestPanoAlanKapanir(t *testing.T) {
	pts := panoNokta([]int{1, 2, 3}, 100, 50, 4)
	d := panoAlan(pts, 50, 4)
	assert.True(t, strings.HasSuffix(d, "Z"), "alan yolu kapanmalı")
	assert.Contains(t, d, "L", "tabana inen çizgi olmalı")
}

func TestPanoDilimlerYuzdeYuz(t *testing.T) {
	d := panoDilimler([]CatSliceRef{
		{Name: "A", Count: 3}, {Name: "B", Count: 3}, {Name: "C", Count: 3},
	})
	toplam := 0
	for _, s := range d {
		toplam += s.Pct
	}
	// 33+33+33 = 99; kalan en büyük dilime eklenir ki halka kapansın.
	assert.Equal(t, 100, toplam, "yuvarlama kalanı bir dilime eklenmeli")
}

func TestPanoDilimlerBos(t *testing.T) {
	assert.Empty(t, panoDilimler(nil))
	assert.Empty(t, panoDilimler([]CatSliceRef{{Name: "A", Count: 0}}))
}

func TestPanoHalkaTumArgumanlarYerinde(t *testing.T) {
	h := string(panoHalka([]PanoDilim{
		{Name: "Teknoloji", Count: 92, Pct: 46},
		{Name: "Gündem", Count: 31, Pct: 16},
	}, 190))
	assert.NotContains(t, h, "%!", "biçim argümanı eksik olmamalı")
	assert.Equal(t, 2, strings.Count(h, `class="pn-dilim"`))
	assert.Contains(t, h, "Teknoloji: 92 (46%)")
	assert.Contains(t, h, "rotate(-90 95.0 95.0)")
}

func TestPanoSutunSifirCubukGorunur(t *testing.T) {
	// Sıfır değerli ay tamamen kaybolmamalı: taban çizgisi görünmeli ki
	// "veri yok" ile "sıfır yayın" ayırt edilebilsin.
	s := string(panoSutun(panoSeri{Values: []int{0, 5}, Labels: []string{"Eyl", "Eki"}}, 200, 100))
	assert.Equal(t, 2, strings.Count(s, `class="pn-cubuk"`))
	assert.Contains(t, s, `height="2.0"`)
	assert.Contains(t, s, "Eyl")
}

func TestPanoSparkKisaSeri(t *testing.T) {
	assert.Empty(t, panoSpark([]int{1}, 90, 30), "tek noktadan çizgi olmaz")
	assert.NotEmpty(t, panoSpark([]int{1, 4, 2}, 90, 30))
}

func TestPanoYatayCubuklar(t *testing.T) {
	h := string(panoYatay([]PanoCubuk{
		{Label: "Haber <b>", Value: 120, Pct: 100, Href: "/admin/icerikler/1"},
		{Label: "Diğer", Value: 60, Pct: 50},
	}))
	assert.Contains(t, h, "Haber &lt;b&gt;", "başlık kaçırılmalı")
	assert.Contains(t, h, "width:50%")
	assert.Equal(t, 2, strings.Count(h, "pn-yatay__satir"))
}

// Sürükle-bırak saati: form gövdesinde "+03:00" ofseti boşluğa dönüşebilir.
func TestPanoSaatCoz(t *testing.T) {
	kontrol := func(ham string, yil int, ay time.Month, gun, saat, dk int) {
		t.Helper()
		ts, err := panoSaatCoz(ham)
		require.NoErrorf(t, err, "girdi: %q", ham)
		assert.Equalf(t, yil, ts.Year(), "yıl (%q)", ham)
		assert.Equalf(t, ay, ts.Month(), "ay (%q)", ham)
		assert.Equalf(t, gun, ts.Day(), "gün (%q)", ham)
		assert.Equalf(t, saat, ts.Hour(), "saat (%q)", ham)
		assert.Equalf(t, dk, ts.Minute(), "dakika (%q)", ham)
	}

	// Tam RFC3339 (istemcinin gönderdiği biçim).
	kontrol("2026-10-30T09:30:00+03:00", 2026, time.October, 30, 9, 30)
	// Ofset boşluğa dönmüş (kodlanmamış form gövdesi).
	kontrol("2026-10-30T09:30:00 03:00", 2026, time.October, 30, 9, 30)
	// datetime-local ve düz tarih.
	kontrol("2026-10-30T09:30", 2026, time.October, 30, 9, 30)
	kontrol("2026-10-30 09:30", 2026, time.October, 30, 9, 30)
	kontrol("2026-10-30", 2026, time.October, 30, 0, 0)

	_, err := panoSaatCoz("yarın sabah")
	assert.Error(t, err)
	_, err = panoSaatCoz("")
	assert.Error(t, err)
}

func TestPanoDelta(t *testing.T) {
	assert.Equal(t, "↑ 7.3%", panoDelta(118, 110), "artış bir ondalıkla")
	assert.Equal(t, "↓ 12.5%", panoDelta(35, 40))
	assert.Equal(t, "", panoDelta(0, 0), "iki sıfırda rozet yok")
	assert.Equal(t, "↑ yeni", panoDelta(5, 0), "önceki sıfırsa yüzde değil 'yeni'")
	assert.Equal(t, "up", panoDeltaTone(5, 1))
	assert.Equal(t, "down", panoDeltaTone(1, 5))
	assert.Equal(t, "flat", panoDeltaTone(5, 5))
}

func TestKisaGunVeAy(t *testing.T) {
	assert.Equal(t, "29.10", kisaGun("2026-10-29"))
	assert.Equal(t, "x", kisaGun("x"), "kısa girdi olduğu gibi döner")
	assert.Equal(t, "Eki", kisaAy("2026-10"))
	assert.Equal(t, "Oca", kisaAy("2026-01"))
	assert.Equal(t, "13", kisaAy("2026-13"), "bilinmeyen ay numarası ham döner")
}

func TestSelamVeTarih(t *testing.T) {
	yap := func(h int) time.Time { return time.Date(2026, 10, 29, h, 0, 0, 0, time.Local) }
	assert.Equal(t, "İyi geceler", selamSozu(yap(3)))
	assert.Equal(t, "Günaydın", selamSozu(yap(9)))
	assert.Equal(t, "İyi günler", selamSozu(yap(15)))
	assert.Equal(t, "İyi akşamlar", selamSozu(yap(21)))
	assert.Equal(t, "29 Ekim 2026, Perşembe", turkceTarih(yap(9)))
}

func TestDurumTR(t *testing.T) {
	assert.Equal(t, "Yayında", durumTR("yayinda"))
	assert.Equal(t, "Planlanmış", durumTR("planlanmis"))
	assert.Equal(t, "Taslak", durumTR("taslak"))
	assert.Equal(t, "bilinmeyen", durumTR("bilinmeyen"))
}
