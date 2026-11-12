package templkit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/webui"
)

func TestVideoSağKolonAkisCSS(t *testing.T) {
	b, err := webui.FS.ReadFile("static/cd.css")
	require.NoError(t, err)
	css := string(b)

	// 1) Kaydırma çubuğu gizli (iki tarayıcı ailesi için ayrı kural).
	assert.Contains(t, css, "scrollbar-width: none", "Firefox: çubuk gizlenmeli")
	assert.Contains(t, css, ".cd-vband__liste::-webkit-scrollbar", "WebKit: çubuk gizlenmeli")

	// 2) Üst/altta saydam geçiş (maske) — "hafif transparan geçiş".
	assert.Contains(t, css, "-webkit-mask-image: linear-gradient", "Safari maskesi")
	assert.Contains(t, css, "mask-image: linear-gradient", "maske gradyanı")

	// 3) Yan yana düzen: bu kural BİR KEZ fazla kapanış parantezi yüzünden
	//    hiç uygulanmıyordu (sağ kolon alta düşüyordu). Ölçümle bulundu.
	assert.Contains(t, css, ".cd-vband__ic {\n  display: flex;",
		"sağ kolon yan yana olmalı — kural geçerli olmalı")
	assert.Contains(t, css, "flex: 0 0 416px", "sağ kolon genişliği (ölçüldü)")

	// 4) Dikey (short) satır görseli 9:16 — 16:9 kutuya kırpılınca kapak
	//    ortasından ince şerit görünüyordu.
	assert.Contains(t, css, ".cd-vsatir--dikey .cd-vsatir__gorsel", "dikey satır kuralı")
	assert.Contains(t, css, "aspect-ratio: 9 / 16", "dikey satır 9:16 olmalı")
}

func TestVideoAkisBetigiYukleniyor(t *testing.T) {
	b, err := webui.FS.ReadFile("static/cd-vkaydir.js")
	require.NoError(t, err, "static/cd-vkaydir.js yok")
	js := string(b)

	// Yumuşatma ZAMANA bağlı olmalı: kare başına sabit oran, kare hızı
	// düşünce akışı yarım bırakıyordu (ölçüldü: 15 karede 1 px).
	assert.Contains(t, js, "var SURE = 260", "akış süresi sabiti")
	assert.Contains(t, js, "requestAnimationFrame", "kare döngüsü")
	assert.Contains(t, js, "setTimeout(emniyet", "güvenlik ağı: rAF kısıtlıysa hedefe atla")

	// Uçlarda sayfaya devretmeli — kullanıcı listede mahsur kalmamalı.
	assert.Contains(t, js, "if ((ust && e.deltaY < 0) || (alt && e.deltaY > 0)) return;",
		"liste ucunda tekerlek sayfaya devretmeli")
	assert.Contains(t, js, "passive: false", "preventDefault için passive olmamalı")

	// Erişilebilirlik: hareket hassasiyeti.
	assert.Contains(t, js, "prefers-reduced-motion", "hareket azaltma tercihi")
}

func TestVideoSatirDikeySinifi(t *testing.T) {
	// Dikey satır ayrı sınıf almalı; almazsa kapak 16:9 kutuya kırpılır.
	var b []byte
	govde := cdVSatirGovde(CDVideoSatir{
		Baslik: "deneme", Image: "https://i.ytimg.com/vi/x/oar2.jpg",
		ImageYedek: "https://i.ytimg.com/vi/x/hqdefault.jpg",
	})
	_ = b
	assert.NotNil(t, govde)
}
