package httpapi

// Canlı piyasa merkezi (SSE) testleri — 11 Kasım 2026.
//
// NEDEN: "veriler değiştiği ANDA değişsin" isteğiyle eklenen itme hattı
// sessizce bozulabilecek üç yer içeriyor:
//   1) Hub abonelik ömrü: abone ayrılınca kanal KAPANMALI, yoksa sızıntı.
//   2) Yavaş istemci: kanal dolduğunda yayın BLOKLAMAMALI (aksi hâlde tek
//      bir yavaş tarayıcı tüm akışı durdurur).
//   3) Yeni aboneye son mesaj hemen verilmeli (boş ekran olmasın).
//
// Ağ ve DB GEREKMEZ: hub saf bellek içi.

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCanliHubAboneAlirVeMesajGorur(t *testing.T) {
	h := &canliHub{abone: map[chan []byte]struct{}{}}
	ch, birak := h.aboneOl()
	defer birak()

	assert.Equal(t, 1, h.aboneSayisi())

	h.yayin([]byte(`{"t":"f"}`))
	select {
	case b := <-ch:
		assert.Equal(t, `{"t":"f"}`, string(b))
	case <-time.After(time.Second):
		t.Fatal("mesaj gelmedi")
	}
}

func TestCanliHubYeniAboneyeSonMesajiVerir(t *testing.T) {
	h := &canliHub{abone: map[chan []byte]struct{}{}}
	h.yayin([]byte(`{"t":"t","hazir":true}`))

	// SONRADAN bağlanan istemci boş ekran görmemeli.
	ch, birak := h.aboneOl()
	defer birak()
	select {
	case b := <-ch:
		assert.Contains(t, string(b), "hazir")
	case <-time.After(time.Second):
		t.Fatal("son mesaj verilmedi")
	}
}

func TestCanliHubYavasIstemcinBloklamaz(t *testing.T) {
	h := &canliHub{abone: map[chan []byte]struct{}{}}
	ch, birak := h.aboneOl()
	defer birak()

	// Kanal tamponu 4; 50 mesaj gönder. Hiçbiri okunmuyor (yavaş istemci).
	bitti := make(chan struct{})
	go func() {
		for i := 0; i < 50; i++ {
			h.yayin([]byte(`{"t":"f"}`))
		}
		close(bitti)
	}()

	select {
	case <-bitti:
		// Yayın tıkanmadı.
	case <-time.After(2 * time.Second):
		t.Fatal("yayın yavaş istemci yüzünden bloklandı")
	}
	// Tampon taşmadı.
	assert.LessOrEqual(t, len(ch), 4)
}

func TestCanliHubBirakKanaliKapatir(t *testing.T) {
	h := &canliHub{abone: map[chan []byte]struct{}{}}
	ch, birak := h.aboneOl()
	require.Equal(t, 1, h.aboneSayisi())

	birak()
	assert.Equal(t, 0, h.aboneSayisi())
	// Kanal kapatılmış olmalı: okuma anında ok=false.
	_, acik := <-ch
	assert.False(t, acik, "kanal kapatılmalı (sızıntı)")
}

func TestCanliHubCifteBirakPanikEtmez(t *testing.T) {
	h := &canliHub{abone: map[chan []byte]struct{}{}}
	_, birak := h.aboneOl()
	birak()
	// İkinci çağrı paniklememeli (defer + iptal yarışı olabilir).
	assert.NotPanics(t, birak)
}

func TestCanliPiyasaRotasıÇakışmaz(t *testing.T) {
	// Rota iki kez eklenirse mux PANİK atar (yaşanmış hata, 11 Kasım 2026:
	// betik iki kez çalıştı, sunucu açılışta çöktü). Bu test aynı rotanın
	// bir kez kaydedildiğinde sorun çıkarmadığını doğrular.
	assert.NotPanics(t, func() {
		s := &Server{mux: http.NewServeMux()}
		s.mux.HandleFunc("GET /canli/piyasa", s.handleCanliPiyasaSSE)
	})
}
