package httpapi

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"time"

	"theblock/site/internal/repo"
)

// İSTEMCİ SİNYAL BEACON'U — POST /api/guvenlik
//
// NEDEN GEREKLİ: bazı kanıtlar yalnız tarayıcıda var — WebRTC'nin sızdırdığı
// gerçek IP, navigator.webdriver, yazılım grafik birimi, saat dilimi.
// Sunucu bunları göremez.
//
// GÜVENLİK MODELİ (tek cümle): bu uç YALNIZ PUANI YÜKSELTEBİLİR.
// Aşağıdaki katmanlar bunu zorlar:
//   · Gövde 4 KB ile sınırlı, tek JSON nesnesi.
//   · Ziyaretçi çerezi yoksa istek yok sayılır (anonim beacon yok).
//   · Gönderilen IP'ler yalnız "sunucunun gördüğünden FARKLI mı" diye
//     karşılaştırılır; istemcinin verdiği IP KAYDEDİLMEZ, yalnız bayrak.
//   · RiskIstemciGuncelle puanı GREATEST ile yazar: bot "ben temizim"
//     diyerek puanını düşüremez.
//   · Saat dilimi/dil uyuşmazlığı SUNUCUDA, IP ülkesine göre hesaplanır;
//     istemcinin "uyuşuyor" iddiası kabul edilmez.

const (
	beaconGovdeSinir = 4096
	// beaconOturumAralik aynı ziyaretçiden en sık beacon kabulü.
	beaconOturumAralik = 30 * time.Second
)

type beaconGovde struct {
	Webdriver  bool     `json:"webdriver"`
	GPU        string   `json:"gpu"`
	SaatDilimi string   `json:"tz"`
	Dil        string   `json:"dil"`
	WebRTC     []string `json:"webrtc"`
	Etkilesim  *int     `json:"etkilesim"`
}

// guvenlikAPI istemci sinyallerini alır.
func (s *Server) guvenlikAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	pol := s.guvenlikPolitikasi(r.Context())
	if !pol.Aktif {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, beaconGovdeSinir)
	var g beaconGovde
	if err := json.NewDecoder(r.Body).Decode(&g); err != nil {
		http.Error(w, "geçersiz gövde", http.StatusBadRequest)
		return
	}

	anahtar := s.ziyaretciID(w, r, false)
	if anahtar == "" {
		// Anonim beacon kabul edilmez: çerezsiz istek zaten ASN/UA
		// katmanıyla sınıflanıyor.
		w.WriteHeader(http.StatusNoContent)
		return
	}
	ip := clientIP(r)
	ulke := cfUlke(r)

	d := guvenlikDurumlari.al(anahtar, ip, ulke)

	// ---- sinyal hesapları (SUNUCU tarafı) ----
	webdriver := g.Webdriver
	headless := yazilimGPU(g.GPU)
	tzUyusmaz := kitaUyusmazlik(ulke, g.SaatDilimi)
	dilUyusmaz := dilUyusmazlik(ulke, g.Dil)

	// WebRTC: yalnız sunucunun gördüğünden farklı GENEL IP anlamlı.
	// Yönetici WebRTC katmanını kapattıysa IP'ler hiç işlenmez.
	farkliIP := ""
	if pol.WebRTC {
		for _, aday := range g.WebRTC {
			aday = strings.TrimSpace(aday)
			if aday == "" || aday == ip {
				continue
			}
			p := net.ParseIP(aday)
			if p == nil || p.IsLoopback() || p.IsPrivate() || p.IsLinkLocalUnicast() || p.IsLinkLocalMulticast() || p.IsUnspecified() {
				continue
			}
			farkliIP = aday
			break
		}
	}
	etkilesimYok := g.Etkilesim != nil && *g.Etkilesim == 0

	// ---- duruma yaz ----
	guvenlikDurumlari.mu.Lock()
	d.Webdriver = d.Webdriver || webdriver
	d.HeadlessGPU = d.HeadlessGPU || headless
	d.TZUyusmazlik = d.TZUyusmazlik || tzUyusmaz
	d.DilUyusmazlik = d.DilUyusmazlik || dilUyusmaz
	d.EtkilesimYok = d.EtkilesimYok || etkilesimYok
	if farkliIP != "" {
		d.WebRTCFarkli = true
		d.WebRTCIP = farkliIP
	}
	webdriverV := d.Webdriver
	headlessV := d.HeadlessGPU
	tzV := d.TZUyusmazlik
	dilV := d.DilUyusmazlik
	etkilesimV := d.EtkilesimYok
	rtcV := d.WebRTCFarkli
	rtcIP := d.WebRTCIP
	guvenlikDurumlari.mu.Unlock()

	// ---- puanı yeniden hesapla ve DB'ye yaz ----
	girdi := guvenlikGirdi{
		Anahtar:       anahtar,
		IP:            ip,
		Ulke:          ulke,
		EsikYuksek:    s.ayarSayi(r.Context(), "guvenlik", "esik_yuksek", 60, 30, 100),
		EsikBot:       s.ayarSayi(r.Context(), "guvenlik", "esik_bot", 85, 50, 100),
		Politika:      pol,
		Webdriver:     webdriverV,
		HeadlessGPU:   headlessV,
		TZUyusmazlik:  tzV,
		DilUyusmazlik: dilV,
		WebRTCFarkli:  rtcV,
		WebRTCIP:      rtcIP,
		EtkilesimYok:  etkilesimV,
	}
	if ua := strings.TrimSpace(r.UserAgent()); ua == "" {
		girdi.UABos = true
	} else if oto, neden := otomatikMi(ua); oto {
		girdi.UAOtomasyon = true
		girdi.UANeden = neden
	}
	if b, ok := s.asnSorgula(r.Context(), ip); ok {
		bb := b
		girdi.ASN = &bb
	}
	girdi.IPRotasyon = guvenlikDurumlari.ozet(anahtar).IPRotasyon

	karar := guvenlikHesapla(girdi)

	// Beacon kararı okunmayı ETKİLEMEZ (sayfa zaten sunulmuş); yalnız
	// kaydı günceller. Bu yüzden engelleme burada uygulanmaz.

	if s.pool != nil {
		ctx, iptal := context.WithTimeout(context.Background(), 4*time.Second)
		go func() {
			defer iptal()
			if err := repo.RiskIstemciGuncelle(ctx, s.pool, anahtar, rtcIP,
				karar.Puan, karar.Sinif, karar.sinyallerJSON()); err != nil {
				s.logger.Debug("beacon risk güncellenemedi", "error", err)
			}
		}()
	}

	jsonCevap(w, map[string]any{"ok": true})
}

// yazilimGPU: yazılım tabanlı grafik birimi (headless/bulut).
//
// ÖLÇÜLEN İMZALAR: Chrome headless varsayılan olarak SwiftShader kullanır.
// llvmpipe/Mesa software rasterizer ise sanal sunucularda yaygın.
// GERÇEK kullanıcıda Intel/AMD/NVIDIA/Apple imzası görülür.
func yazilimGPU(gpu string) bool {
	u := strings.ToLower(strings.TrimSpace(gpu))
	if u == "" {
		return false
	}
	for _, imza := range []string{
		"swiftshader", "llvmpipe", "software rasterizer", "mesa offscreen",
		"google swiftshader", "microsoft basic render", "vmware svga",
		"virtualbox", "qxl", "cirrus logic",
	} {
		if strings.Contains(u, imza) {
			return true
		}
	}
	return false
}

func jsonCevap(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(v)
}
