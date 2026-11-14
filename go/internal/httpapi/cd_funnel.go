package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"theblock/site/internal/webui/templkit"
)

// Funnel satış sayfası — GET /funnel.
//
// NEDEN AYRI SAYFA: Milyonerler Kulübü'ne Trader Bitcoşar panelinin sinyal
// aboneliğini satmak için tek işli bir sayfa gerekiyordu; haber ana
// sayfasının şeridi/menüsü/reklam yuvaları dönüşümü bölüyor.
//
// VERİ KAYNAĞI: planlar ve fiyatlar traders.bitcosar.com panelindeki plan
// kayıtlarından alındı (14 Kasım 2026: Bronz $60, Altın $80, Premium $99).
// Panel fiyatı değişirse burası da güncellenmeli — sayfa panelin API'sine
// bağlı değil (ayrı uygulama, ayrı veritabanı).
const funnelPanelKok = "https://traders.bitcosar.com"

// funnelPlanlar paneldeki üç plan. Açıklamalar paneldeki plan notlarından
// sadeleştirildi (yazım hataları düzeltildi, anlam korundu).
func funnelPlanlar() []templkit.CDFunnelPlan {
	return []templkit.CDFunnelPlan{
		{
			Ad:       "Bronz",
			Aciklama: "Günlük ve haftalık BTC spot ve vadeli sinyaller, temel analiz notları.",
			Tutar:    "$60",
			Donem:    "/ ay",
			Ozellikler: []string{
				"Haftada 3–5 spot sinyal",
				"Giriş ve hedef seviyeleri",
				"E-posta bildirimi",
			},
			CtaHref: funnelPanelKok + "/signup?plan=bronz",
		},
		{
			Ad:       "Altın",
			Aciklama: "Spot ve vadeli sinyaller, kaldıraç önerisi, Telegram anlık bildirimi.",
			Tutar:    "$80",
			Donem:    "/ ay",
			Ozellikler: []string{
				"Günde 5–10 sinyal",
				"Vadeli işlem sinyalleri",
				"Kaldıraç ve risk önerisi",
				"Telegram anlık bildirim",
			},
			CtaHref:  funnelPanelKok + "/signup?plan=altin",
			OneCikan: true,
			Rozet:    "Önerilen",
		},
		{
			Ad:       "Premium",
			Aciklama: "Tüm sinyaller, portföy takibi, özel analiz ve öncelikli destek.",
			Tutar:    "$99",
			Donem:    "/ ay",
			Ozellikler: []string{
				"Sınırsız sinyal",
				"Portföy takibi",
				"Haftalık özel analiz",
				"Öncelikli destek",
			},
			CtaHref: funnelPanelKok + "/signup?plan=premium",
		},
	}
}

// funnelSSS cevapları panelin kendi yardım metinlerinden alındı
// (traders.bitcosar.com/faq, 14 Kasım 2026). Uydurma vaat yok: her cevap
// panelin gerçek davranışını anlatıyor. Aynı dilim JSON-LD'yi de besler.
func funnelSSS() []templkit.CDFunnelSSS {
	return []templkit.CDFunnelSSS{
		{
			Soru: "Sinyalleri nasıl alırım?",
			Cevap: "Aktif aboneliğin olduğunda sinyaller panelde görünür. Telegram " +
				"hesabını bağlarsan yayınlandığı anda telefonuna da düşer. Bağlamak " +
				"için Sinyaller sayfasındaki üç adımı izle: kod oluştur, botu " +
				"başlat, kodu gönder, doğrula.",
		},
		{
			Soru: "Sinyalde ne yazıyor?",
			Cevap: "Yön (AL/SAT), parite, zaman aralığı ve borsa üstte. Altında " +
				"açılış fiyatı, zarar durdurma (stop) ve üç kâr al hedefi. Stop, " +
				"işlemin kabul edilebilir en kötü noktası; hedefler kademeli kâr " +
				"alma seviyeleri.",
		},
		{
			Soru: "Başarı oranı nasıl hesaplanıyor?",
			Cevap: "Yalnızca kapanmış sinyaller üzerinden: hedefe ulaşan / (hedefe " +
				"ulaşan + stop olan). Açık pozisyonlar hesaba katılmaz; katılsaydı " +
				"oran yapay olarak yüksek görünürdü.",
		},
		{
			Soru: "Sinyal geldi ama fiyat geçmiş, ne yapmalıyım?",
			Cevap: "Açılış fiyatı geçtiyse işleme girmemek genellikle daha doğrudur. " +
				"Geciken girişte risk/kazanç oranı bozulur: stop mesafesi aynı " +
				"kalırken hedefe uzaklık azalır.",
		},
		{
			Soru: "Hangi ödeme yöntemleri kabul ediliyor?",
			Cevap: "Kredi kartı (uluslararası ve Türkiye), kripto para (USDT, BTC) ve " +
				"banka havalesi. Açık olan yöntemler ödeme sayfasında görünür.",
		},
		{
			Soru: "Aboneliğim bitince ne olur?",
			Cevap: "Sinyaller durur. Panelde yeni sinyal görünmez, Telegram'a mesaj " +
				"gelmez. Geçmiş sinyaller de listeden kalkar. Yenilemek için bizimle " +
				"iletişime geçebilirsin.",
		},
		{
			Soru: "Aboneliğimi durdurabilir miyim?",
			Cevap: "Evet. Aboneliği istediğin an durdurabilirsin; kalan dönem için " +
				"yeni sinyal yayınlanmaz. Kripto ve havale ödemelerinde otomatik " +
				"yenileme yoktur, süre dolmadan hatırlatma e-postası gönderilir.",
		},
		{
			Soru: "Telegram bağlantımı kaldırabilir miyim?",
			Cevap: "Evet, Sinyaller sayfasındaki Telegram kartından. Kaldırdığında " +
				"sinyaller panelde görünmeye devam eder, sadece Telegram'a gitmez.",
		},
	}
}

// funnelJSONLD Product + FAQPage bloğu üretir.
//
// FAQ metni funnelSSS ile AYNI kaynaktan gelir: sayfada görünen cevap ile
// yapısal veri ayrışırsa Google zengin sonuç cezası veriyor.
func funnelJSONLD(site string, planlar []templkit.CDFunnelPlan, sss []templkit.CDFunnelSSS) string {
	type teklif struct {
		Type          string `json:"@type"`
		Name          string `json:"name"`
		Price         string `json:"price"`
		PriceCurrency string `json:"priceCurrency"`
	}
	type urun struct {
		Type        string   `json:"@type"`
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Brand       string   `json:"brand"`
		Offers      []teklif `json:"offers"`
	}
	type cevap struct {
		Type string `json:"@type"`
		Text string `json:"text"`
	}
	type soru struct {
		Type  string `json:"@type"`
		Name  string `json:"name"`
		Cevap cevap  `json:"acceptedAnswer"`
	}
	type sssBlok struct {
		Type  string `json:"@type"`
		Items []soru `json:"mainEntity"`
	}

	teklifler := make([]teklif, 0, len(planlar))
	for _, p := range planlar {
		teklifler = append(teklifler, teklif{
			Type:          "Offer",
			Name:          p.Ad + " planı",
			Price:         strings.TrimPrefix(p.Tutar, "$"),
			PriceCurrency: "USD",
		})
	}
	sorular := make([]soru, 0, len(sss))
	for _, s := range sss {
		sorular = append(sorular, soru{
			Type:  "Question",
			Name:  s.Soru,
			Cevap: cevap{Type: "Answer", Text: s.Cevap},
		})
	}

	blok := map[string]any{
		"@context": "https://schema.org",
		"@graph": []any{
			urun{
				Type:        "Product",
				Name:        "Trader Bitcoşar sinyal aboneliği",
				Description: "Kripto sinyal paneli: giriş, stop ve üç kâr al hedefiyle yayınlanan sinyaller; panelde ve Telegram'da anlık teslim.",
				Brand:       site,
				Offers:      teklifler,
			},
			sssBlok{Type: "FAQPage", Items: sorular},
		},
	}
	b, err := json.Marshal(blok)
	if err != nil {
		return ""
	}
	// cdHead JSONLD'yi templ.Raw ile bastığı için script etiketi BURADA
	// ekleniyor (bkz. cd_jsonld.go aynı kalıp).
	return `<script type="application/ld+json">` + string(b) + `</script>`
}

// funnelKabuk sayfanın ihtiyaç duyduğu kabuk alanlarını toplar.
//
// cdShellCtx BİLEREK kullanılmadı: o yol fiyat şeridi, mega menü ve
// etiket listesi için DB'ye gidiyor; satış sayfasında bunların hiçbiri
// görünmüyor. Yalnız marka alanları (ad, logo, favicon) okunuyor.
func (s *Server) funnelKabuk(ctx context.Context, r *http.Request) templkit.CDShell {
	sh := templkit.CDShell{
		SiteName: s.siteAdi(),
		Locale:   "tr",
		BaseURL:  s.kanonikKok(r),
	}
	bc := s.brandConfig(ctx)
	if bc == nil {
		return sh
	}
	gorsel := func(id *int64) string {
		if id == nil || *id <= 0 {
			return ""
		}
		v := s.mediaURL(id)
		if v == "" || v == "/media/none.svg" {
			return ""
		}
		return v
	}
	sh.LogoURL = gorsel(bc.ImageID)
	sh.FaviconURL = gorsel(bc.FaviconID)
	return sh
}

func (s *Server) handleFunnel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	planlar := funnelPlanlar()
	sss := funnelSSS()
	site := s.siteAdi()

	v := templkit.CDFunnel{
		Shell: s.funnelKabuk(ctx, r),
		Title: "Trader Bitcoşar — sinyal paneli ve planlar",
		Desc: "Milyonerler Kulübü için Trader Bitcoşar paneli: sinyaller yayınlandığı anda panelde ve Telegram'da. " +
			"Bronz $60, Altın $80, Premium $99 / ay. Başarı oranı yalnızca kapanmış sinyallerden hesaplanır.",
		Canonical: strings.TrimRight(s.kanonikKok(r), "/") + "/funnel",
		JSONLD:    funnelJSONLD(site, planlar, sss),
		Plans:     planlar,
		SSS:       sss,
		KayitHref: funnelPanelKok + "/signup",
		GirisHref: funnelPanelKok + "/login",
		PanelHref: funnelPanelKok + "/",
		SiteHref:  "/",
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", cacheControlPublic)
	if err := templkit.CDFunnelSayfa(v).Render(r.Context(), w); err != nil {
		s.logger.Error("funnel render", "err", err)
	}
}
