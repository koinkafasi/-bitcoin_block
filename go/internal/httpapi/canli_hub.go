package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"theblock/site/internal/repo"
)

// canliHub bağlı istemcilere mesaj dağıtır.
type canliHub struct {
	mu       sync.RWMutex
	abone    map[chan []byte]struct{}
	sonMesaj []byte // yeni bağlanana hemen verilir (boş ekran olmasın)
}

var canliMerkez = &canliHub{abone: map[chan []byte]struct{}{}}

// canliSonFiyat: son YAYINLANAN fiyat metinleri (delta tabanı).
// Yalnız hub döngüsü yazar; kilit altında.
var (
	canliSonKilit sync.Mutex
	canliSonFiyat = map[string]string{}
)

// Abone ol / ayrıl.
func (h *canliHub) aboneOl() (chan []byte, func()) {
	kanal := make(chan []byte, 4)
	h.mu.Lock()
	h.abone[kanal] = struct{}{}
	son := h.sonMesaj
	h.mu.Unlock()
	if len(son) > 0 {
		select {
		case kanal <- son:
		default:
		}
	}
	return kanal, func() {
		h.mu.Lock()
		if _, ok := h.abone[kanal]; ok {
			delete(h.abone, kanal)
			close(kanal)
		}
		h.mu.Unlock()
	}
}

func (h *canliHub) yayin(b []byte) {
	h.mu.Lock()
	h.sonMesaj = b
	for kanal := range h.abone {
		select {
		case kanal <- b:
		default:
		}
	}
	h.mu.Unlock()
}

func (h *canliHub) aboneSayisi() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.abone)
}

// --- mesaj gövdeleri ---

// canliFiyatMesaj yalnız fiyat katmanı (küçük, sık).
type canliFiyatMesaj struct {
	Tip       string                     `json:"t"` // "f"
	Sa        int64                      `json:"sa"`
	Cekildi   int64                      `json:"cekildi"`
	BTC       girisCanliFiyat            `json:"btc"`
	ETH       girisCanliFiyat            `json:"eth"`
	SOL       girisCanliFiyat            `json:"sol"`
	Serit     []girisCanliSerit          `json:"serit"`
	SeritBist []girisCanliSerit          `json:"serit_bist"`
	Coin      map[string]girisCanliFiyat `json:"coin"`
}

// canliTamMesaj tam gövde (sparkline, göstergeler, kur, piyasa).
type canliTamMesaj struct {
	girisCanliYanit
	Tip string                      `json:"t"` // "t"
	Kur map[string]float64          `json:"kur"`
	Piy map[string]girisCanliPiyasa `json:"piyasa"`
}

// girisCanliPiyasa kur/BIST kalemi (PiyasaOf).
type girisCanliPiyasa struct {
	Deger   float64 `json:"f"`
	Degisim float64 `json:"d"`
	Yon     int     `json:"y"`
	Ondalik int     `json:"o"`
	Birim   string  `json:"b"`
}

func (s *Server) fiyatMesaji(ctx context.Context) (string, []byte, bool) {
	g := s.cdGirisOlustur(ctx)
	if !g.Hazir {
		return "", nil, false
	}

	// canliSonHub: son yayınlanan fiyat metinleri (delta tabanı).
	canliSonKilit.Lock()
	defer canliSonKilit.Unlock()

	m := canliFiyatMesaj{Tip: "f", Coin: map[string]girisCanliFiyat{}}
	degisti := false

	fiyatEkle := func(anahtar, fiyat, degisim string, up bool, hedef *girisCanliFiyat) bool {
		if canliSonFiyat[anahtar] == fiyat && canliSonFiyat[anahtar+"|d"] == degisim {
			return false
		}
		canliSonFiyat[anahtar] = fiyat
		canliSonFiyat[anahtar+"|d"] = degisim
		*hedef = girisCanliFiyat{Var: true, Fiyat: fiyat, Degisim: degisim, Up: up}
		return true
	}

	if fiyatEkle("btc", g.BTCFiyat, g.BTCDegisim, g.BTCUp, &m.BTC) {
		degisti = true
	}
	if g.ETHFiyat != "" && fiyatEkle("eth", g.ETHFiyat, g.ETHDegisim, g.ETHUp, &m.ETH) {
		degisti = true
	}
	if g.SOLFiyat != "" && fiyatEkle("sol", g.SOLFiyat, g.SOLDegisim, g.SOLUp, &m.SOL) {
		degisti = true
	}

	for _, f := range repo.Fiyatlar(ctx, s.pool) {
		if f.Price <= 0 || f.Sym == "" {
			continue
		}
		fiyat, degisim := repo.FiyatBicim(f.Price), repo.DegisimBicim(f.Change)
		if !fiyatEkle("c:"+f.Sym, fiyat, degisim, f.Change >= 0, &girisCanliFiyat{}) {
			continue
		}
		degisti = true
		m.Coin[f.Sym] = girisCanliFiyat{Var: true, Fiyat: fiyat, Degisim: degisim, Up: f.Change >= 0}
		// Şeritte görünen ilk 7 kalem ayrıca listelenir.
		if len(m.Serit) < 7 {
			m.Serit = append(m.Serit, girisCanliSerit{Sym: f.Sym, Fiyat: fiyat, Degisim: degisim, Up: f.Change >= 0})
		}
	}

	if bd := repo.BorsaOf(ctx, s.pool); !bd.Demo && len(bd.Rows) > 0 {
		for _, r := range bd.Rows {
			if r.Kind == "kripto" || r.Value <= 0 {
				continue
			}
			sym := strings.TrimSuffix(r.Code, ".IS")
			fiyat := repo.FiyatBicimP(r.Value, r.Para)
			degisim := repo.DegisimBicim(r.Delta)
			if !fiyatEkle("b:"+sym, fiyat, degisim, r.Delta >= 0, &girisCanliFiyat{}) {
				continue
			}
			degisti = true
			m.SeritBist = append(m.SeritBist, girisCanliSerit{
				Sym: sym, Fiyat: fiyat, Degisim: degisim, Up: r.Delta >= 0,
			})
			if len(m.SeritBist) >= 7 {
				break
			}
		}
	}

	if !degisti {
		return "", nil, false // hiçbir şey değişmedi: MESAJ YOK
	}

	m.Sa = time.Now().Unix()
	m.Cekildi = g.Cekildi
	govdeB, err := json.Marshal(m)
	if err != nil {
		return "", nil, false
	}
	return string(govdeB), govdeB, true
}

// tamMesaji hero + kur + piyasa katmanını üretir (imza + gövde).
func (s *Server) tamMesaji(ctx context.Context) (string, []byte, bool) {
	g := s.cdGirisOlustur(ctx)
	if !g.Hazir {
		return "", nil, false
	}
	y := girisCanliYanit{
		Hazir:    true,
		BTC:      girisCanliFiyat{Var: true, Fiyat: g.BTCFiyat, Degisim: g.BTCDegisim, Up: g.BTCUp},
		ETH:      girisCanliFiyat{Var: g.ETHFiyat != "", Fiyat: g.ETHFiyat, Degisim: g.ETHDegisim, Up: g.ETHUp},
		SOL:      girisCanliFiyat{Var: g.SOLFiyat != "", Fiyat: g.SOLFiyat, Degisim: g.SOLDegisim, Up: g.SOLUp},
		Yol:      g.BTCYol,
		Alan:     g.BTCAlan,
		Hacim:    g.Hacim,
		Dominans: g.Dominans,
		Endeks:   g.Endeks,
		EndeksUp: g.EndeksUp,
	}
	if g.FNGVar {
		y.FNG = &girisCanliGosterge{
			Var: true, Deger: g.FNGDeger, Etiket: g.FNGEtiket,
			Up: g.FNGUp, Dun: g.FNGDun, Hafta: g.FNGHafta,
		}
	}
	if g.RSIVar {
		y.RSI = &girisCanliGosterge{Var: true, Deger: g.RSI, Etiket: g.RSIEtiket, Up: g.RSIUp}
	}
	for _, f := range repo.Fiyatlar(ctx, s.pool) {
		if len(y.Serit) >= 7 {
			break
		}
		if f.Price <= 0 {
			continue
		}
		y.Serit = append(y.Serit, girisCanliSerit{
			Sym: f.Sym, Fiyat: repo.FiyatBicim(f.Price),
			Degisim: repo.DegisimBicim(f.Change), Up: f.Change >= 0,
		})
	}

	// `t` TAM listedir: yeni bağlanan istemci delta değil bütün durumu alır.
	// Delta tabanı da burada tazelenir ki bir sonraki `f` doğru farkı bulsun.
	canliSonKilit.Lock()
	for _, f := range repo.Fiyatlar(ctx, s.pool) {
		if f.Price <= 0 || f.Sym == "" {
			continue
		}
		canliSonFiyat["c:"+f.Sym] = repo.FiyatBicim(f.Price)
		canliSonFiyat["c:"+f.Sym+"|d"] = repo.DegisimBicim(f.Change)
	}
	canliSonFiyat["btc"], canliSonFiyat["btc|d"] = g.BTCFiyat, g.BTCDegisim
	canliSonFiyat["eth"], canliSonFiyat["eth|d"] = g.ETHFiyat, g.ETHDegisim
	canliSonFiyat["sol"], canliSonFiyat["sol|d"] = g.SOLFiyat, g.SOLDegisim
	canliSonKilit.Unlock()

	t := canliTamMesaj{girisCanliYanit: y, Tip: "t"}
	if kt := repo.KurlarOf(ctx, s.pool); len(kt.Kurlar) > 0 {
		t.Kur = make(map[string]float64, len(kt.Kurlar))
		for kod, deger := range kt.Kurlar {
			if deger > 0 {
				t.Kur[kod] = deger
			}
		}
	}
	if ps := repo.PiyasaOf(ctx, s.pool); len(ps.Kalemler) > 0 {
		t.Piy = make(map[string]girisCanliPiyasa, len(ps.Kalemler))
		for _, k := range ps.Kalemler {
			if k.Anahtar == "" || k.Demo || k.Deger <= 0 {
				continue
			}
			t.Piy[k.Anahtar] = girisCanliPiyasa{
				Deger: k.Deger, Degisim: k.Degisim,
				Yon: k.Yon, Ondalik: k.Ondalik, Birim: k.Birim,
			}
		}
	}

	imzaB, err := json.Marshal(t) // sa/cekildi sıfır
	if err != nil {
		return "", nil, false
	}
	t.Sa = time.Now().Unix()
	t.Cekildi = g.Cekildi
	govdeB, err := json.Marshal(t)
	if err != nil {
		return "", nil, false
	}
	return string(imzaB), govdeB, true
}

func (s *Server) CanliPiyasaBaslat(ctx context.Context) {
	s.startCanliPiyasa(ctx)
}

func (s *Server) startCanliPiyasa(ctx context.Context) {
	go func() {
		fiyatTicker := time.NewTicker(5 * time.Second)
		tamTicker := time.NewTicker(30 * time.Second)
		defer fiyatTicker.Stop()
		defer tamTicker.Stop()

		var sonFiyatImza, sonTamImza string

		yayinlaFiyat := func() {
			if imza, govde, ok := s.fiyatMesaji(ctx); ok {
				if imza != sonFiyatImza { // fiyat GERÇEKTEN değiştiyse
					sonFiyatImza = imza
					canliMerkez.yayin(govde)
				}
			}
		}
		yayinlaTam := func() {
			if imza, govde, ok := s.tamMesaji(ctx); ok {
				if imza != sonTamImza {
					sonTamImza = imza
					canliMerkez.yayin(govde)
				}
			}
		}

		// Açılışta bir kez: önbellek sıcak olsun, ilk ziyaretçi beklemesin.
		yayinlaTam()
		yayinlaFiyat()

		for {
			select {
			case <-ctx.Done():
				return
			case <-fiyatTicker.C:
				yayinlaFiyat()
			case <-tamTicker.C:
				yayinlaTam()
			}
		}
	}()
}

// handleCanliPiyasaSSE GET /canli/piyasa — sunucudan itme akışı.
func (s *Server) handleCanliPiyasaSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming desteklenmiyor", http.StatusInternalServerError)
		return
	}
	// Yazma zaman aşımı akışı kesmesin.
	_ = http.NewResponseController(w).SetWriteDeadline(time.Time{})

	h := w.Header()
	h.Set("Content-Type", "text/event-stream; charset=utf-8")
	h.Set("Cache-Control", "no-cache, no-transform")
	h.Set("Connection", "keep-alive")
	h.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	kanal, birak := canliMerkez.aboneOl()
	defer birak()

	// İlk mesaj: tam gövde (bağlanan istemci boş ekran görmesin).
	if _, govde, ok := s.tamMesaji(r.Context()); ok {
		if err := writeSSEVeri(w, govde); err != nil {
			return
		}
	}

	// Heartbeat: 20 sn. Ters vekil/Cloudflare boşta kalan akışı kesmesin.
	kalp := time.NewTicker(20 * time.Second)
	defer kalp.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case b, acik := <-kanal:
			if !acik {
				return
			}
			if err := writeSSEVeri(w, b); err != nil {
				return
			}
		case <-kalp.C:
			if _, err := w.Write([]byte(": ping\n\n")); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func writeSSEVeri(w http.ResponseWriter, b []byte) error {
	if _, err := w.Write([]byte("data: ")); err != nil {
		return err
	}
	if _, err := w.Write(b); err != nil {
		return err
	}
	if _, err := w.Write([]byte("\n\n")); err != nil {
		return err
	}
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
	return nil
}

// canliAboneSayisi tanı/testler için.
func canliAboneSayisi() int { return canliMerkez.aboneSayisi() }
