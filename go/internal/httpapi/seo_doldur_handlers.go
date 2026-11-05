package httpapi

import (
	"net/http"
	"strings"
	"unicode/utf8"

	"theblock/site/internal/aieditor"
	"theblock/site/internal/repo"
	"theblock/site/internal/webui/templkit"
)

// handleSeoDoldur AI ile SEO başlığı ve meta açıklamasını üretir ve öneri
// panelini döndürür (Faz 188).
//
// Akış bilinçli olarak iki adımlı: üret → göster → uygula. Panel doğrudan
// forma yazmaz; kullanıcı metni görüp tek düğmeyle doldurur, kaydetmeyi
// kendisi yapar. Otomatik haberlerde ise editör olmadığı için aynı üretim
// arka planda (aieditor.RunSeoDoldurma) DB'ye yazılır.
func (s *Server) handleSeoDoldur(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := parseID(r.PathValue("id"))
	if _, ok := s.editableContent(w, r, id); !ok {
		return
	}
	c, err := repo.GetContent(ctx, s.pool, id)
	if err != nil {
		s.renderSeoOneri(w, r, templkit.SeoOneriView{
			ContentID: id,
			Hata:      "İçerik bulunamadı.",
		})
		return
	}
	v := templkit.SeoOneriView{ContentID: id}
	if c.SeoTitle != nil {
		v.MevcutBaslik = strings.TrimSpace(*c.SeoTitle)
	}
	if c.SeoDesc != nil {
		v.MevcutAciklama = strings.TrimSpace(*c.SeoDesc)
	}

	oneri, err := aieditor.SeoUret(ctx, s.ai, c)
	if err != nil {
		s.renderSeoOneri(w, r, templkit.SeoOneriView{
			ContentID:      id,
			MevcutBaslik:   v.MevcutBaslik,
			MevcutAciklama: v.MevcutAciklama,
			Hata:           seoHataMetni(err),
		})
		return
	}

	// ÖNEMLİ: bu uç nokta DB'ye YAZMAZ.
	//
	// İlk sürümde "Yeniden üret" dolu alanların üzerine yazıyordu; oysa
	// panel bir ÖNİZLEME. Kullanıcı metni görüp "İki alanı doldur" diyor,
	// kaydetmeyi formun kendi düğmesiyle yapıyor. Önizleme sırasında
	// yazmak, kullanıcının görmediği bir metni kaydetmek demekti.
	//
	// DB'ye yalnızca arka plan işi yazar (aieditor.RunSeoDoldurma) ve
	// orada da force=false'tur: dolu alan korunur.

	v.Baslik = oneri.Baslik
	v.Aciklama = oneri.Aciklama
	v.Anahtar = oneri.Anahtar
	v.Uyarilar = oneri.Uyarilar
	v.Provider = oneri.Provider
	v.Model = oneri.Model
	v.BaslikLen = utf8.RuneCountInString(oneri.Baslik)
	v.AciklamaLen = utf8.RuneCountInString(oneri.Aciklama)
	s.renderSeoOneri(w, r, v)
}

// seoHataMetni kullanıcıya gösterilecek hatayı sadeleştirir. Sağlayıcı
// yokluğu en sık durum: teknik ayrıntı yerine ne yapılacağı yazılır.
func seoHataMetni(err error) string {
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "yapılandırılmadı"):
		return "AI sağlayıcısı yapılandırılmadı. Ayarlar → AI bölümünden sağlayıcı ekleyin."
	case strings.Contains(msg, "başlık veya gövde"):
		return "SEO üretmek için haberin başlığı veya gövdesi olmalı."
	case strings.Contains(msg, "json"):
		return "AI yanıtı çözümlenemedi; yeniden deneyin."
	}
	return "SEO alanları üretilemedi: " + err.Error()
}

// renderSeoOneri öneri panelini HTMX parçası olarak yazar.
func (s *Server) renderSeoOneri(w http.ResponseWriter, r *http.Request, v templkit.SeoOneriView) {
	var sb strings.Builder
	if err := templkit.SeoOneriPanel(v).Render(r.Context(), &sb); err != nil {
		s.logger.Error("seo öneri render", "error", err)
		http.Error(w, "iç sunucu hatası", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(sb.String()))
}
