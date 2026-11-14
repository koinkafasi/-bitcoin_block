package httpapi

import (
	"net/http"

	"theblock/site/internal/webui/templkit"
)

// KONUM KAPISI — üyelik paneli ve kayıt, konum izni olmadan açılmaz.
//
// KULLANICI İSTEĞİ (14 Kasım 2026): "lokasyon açmadan üyelik paneline
// kayıt olamıyordu; bu işlemi bizde de yapalım".
//
// NEDEN SUNUCUDA: kapıyı yalnız JS'e bırakmak, JS kapalı/engelli
// ziyaretçide (veya doğrudan POST'ta) kapıyı tamamen açık bırakırdı.
// Sunucu, çerezdeki `tb_konum=1` durumuna bakıp karar veriyor.
//
// ÇIKIŞ YOLU HER ZAMAN VAR: kullanıcı reddederse panel açılmaz ama
// site okunabilir kalır; ayrıca yönetici `baglantilar|konum_zorunlu=0`
// ile kapıyı tamamen kapatabilir (o zaman konum yine toplanır).

// konumKapiGerekli mi — kapı açık VE bu istekte izin yok mu?
func (s *Server) konumKapiGerekli(r *http.Request) bool {
	if !s.konumZorunlu(r.Context()) {
		return false
	}
	return konumDurumu(r) != "izin"
}

// konumKapiSayfa kapı sayfasını çizer (200; hata değil, yönlendirme).
//
// NEDEN 200: bu bir hata değil, akışın bir adımı. 403 dönmek tarayıcı
// hata sayfası/arama motoru için yanlış sinyal olurdu; sayfa ayrıca
// noindex taşıyor (hesap sayfaları aranmaz).
func (s *Server) konumKapiSayfa(w http.ResponseWriter, r *http.Request) {
	yerel := s.cdYerel(r)
	v := s.cdSayfaKabuk(r, "hesabim",
		templkit.CDT(yerel, "Konum izni gerekli")+" — "+s.siteName,
		templkit.CDT(yerel, "Üyelik panelini kullanmak için konum izni gerekiyor."),
		templkit.CDT(yerel, "Konum izni gerekli"),
		"",
		s.kanonikKok(r)+r.URL.Path)
	v.Shell.KonumKapi = true
	v.Govde = templkit.SafeHTML(konumKapiGovde(yerel))
	v.JSONLD = ""
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Robots-Tag", "noindex, nofollow")
	if err := templkit.CDBasitSayfaView(v).Render(r.Context(), w); err != nil {
		s.logger.Error("konum kapısı render", "error", err)
	}
}

// konumKapiGovde kapı kartının HTML'i.
//
// Metinler SUNUCUDA üretiliyor ve sabit (kullanıcı girdisi yok), bu
// yüzden SafeHTML güvenli. Çeviri sözlükten geçiyor.
func konumKapiGovde(dil string) string {
	T := func(s string) string { return templkit.CDT(dil, s) }
	return `<div class="cd-kapi">
  <div class="cd-kapi__kart">
    <div class="cd-konum__rozet" aria-hidden="true">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"
        stroke-linecap="round" stroke-linejoin="round" focusable="false">
        <path d="M12 21s7-6.1 7-11a7 7 0 10-14 0c0 4.9 7 11 7 11z"></path>
        <circle cx="12" cy="10" r="2.6"></circle>
      </svg>
    </div>
    <h2 class="cd-kapi__baslik">` + T("Üyelik paneli için konum izni") + `</h2>
    <p class="cd-kapi__metin">` + T("Hesabınızı güvende tutmak ve size bulunduğunuz bölgeye uygun içerik göstermek için konumunuza ihtiyacımız var. İzin vermeden panel ve kayıt işlemleri açılmıyor.") + `</p>
    <button class="cd-btn cd-btn--kayit" type="button" data-konum-ver>` + T("Konumu etkinleştir") + `</button>
    <p class="cd-kapi__not">` + T("Konum yaklaşık 11 metreye yuvarlanır, adresiniz kaydedilmez ve üçüncü taraflarla paylaşılmaz. İzni tarayıcı ayarlarından geri alabilirsiniz.") + `</p>
    <p class="cd-kapi__not"><a href="/">` + T("Ana sayfaya dön") + `</a></p>
  </div>
</div>`
}
