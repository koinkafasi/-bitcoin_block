package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"theblock/site/internal/repo"
)

// Toplu AI ile yeniden yazım paneli (Faz 204).
//
// Ekranın işi: yayındaki ince içerikleri göstermek, pilot başlatmak ve
// üretilen öneriyi eski metinle YAN YANA karşılaştırıp onaylamak/reddetmek.
//
// Kural: bu ekran yayına hiçbir şey yazmaz. "Onayla" düğmesi
// `repo.YenidenYazimOnayla` üzerinden sürüm korumalı tek bir yazma yapar ve
// eski hâli `content_revisions`a arşivler.

// yenidenView panelin tek satırıdır (diff için hazır metin).
type yenidenView struct {
	ID            int64
	ContentID     int64
	Baslik        string
	Kategori      string
	Locale        string
	Mod           string
	ModEtiket     string
	Kaynaklar     []repo.YenidenKaynak
	Durum         string
	DurumEtiket   string
	DurumSinif    string
	Surum         int
	EskiSpot      string
	EskiParagraf  []string
	YeniSpot      string
	YeniParagraf  []string
	EskiSeoBaslik string
	YeniSeoBaslik string
	EskiSeoAcik   string
	YeniSeoAcik   string
	Uyarilar      []string
	Hata          string
	Provider      string
	Model         string
	TokenGirdi    int
	TokenCikti    int
	Deneme        int
	EskiKarakter  int
	YeniKarakter  int
	EskiParagrafN int
	YeniParagrafN int
	Olusturma     string
}

// yenidenData panelin tamamı.
type yenidenData struct {
	Flash       *flashMsg
	Ozet        map[string]int
	AdaySayisi  int
	Toplam      int
	PilotAdet   string
	Mod         string
	FiltreDurum string
	FiltreDil   string
	Satirlar    []yenidenView
	// Detay tek öneri (GET /admin/yeniden-yazim/{id}).
	Detay *yenidenView
}

// yenidenModEtiketi modun kullanıcıya görünen adı.
func yenidenModEtiketi(mod string) string {
	if mod == repo.YenidenModYeniden {
		return "Yalnız mevcut metin"
	}
	return "Araştırmalı"
}

// durumEtiketi kullanıcıya görünen Türkçe durum metni.
func yenidenDurumEtiketi(d string) (string, string) {
	switch d {
	case repo.YenidenBekliyor:
		return "Sırada", "bekliyor"
	case repo.YenidenKuyrukta:
		return "Yazılıyor", "kuyrukta"
	case repo.YenidenHazir:
		return "Onay bekliyor", "hazir"
	case repo.YenidenHata:
		return "Hata", "hata"
	case repo.YenidenUygulandi:
		return "Uygulandı", "tamam"
	case repo.YenidenReddedildi:
		return "Reddedildi", "red"
	default:
		return d, ""
	}
}

// yenidenViewFrom repo satırını ekran satırına çevirir.
func yenidenViewFrom(y repo.YenidenYazim) yenidenView {
	eskiP := repo.GovdeParagraflari(y.OldDetail)
	yeniP := repo.GovdeParagraflari(y.NewDetail)
	etiket, sinif := yenidenDurumEtiketi(y.Status)
	v := yenidenView{
		ID:            y.ID,
		ContentID:     y.ContentID,
		Baslik:        y.GuncelBaslik,
		Kategori:      y.Kategori,
		Locale:        y.Locale,
		Mod:           y.Mod,
		ModEtiket:     yenidenModEtiketi(y.Mod),
		Kaynaklar:     y.Kaynaklar,
		Durum:         y.Status,
		DurumEtiket:   etiket,
		DurumSinif:    sinif,
		Surum:         y.ContentVersion,
		EskiSpot:      y.OldSpot,
		EskiParagraf:  eskiP,
		YeniSpot:      y.NewSpot,
		YeniParagraf:  yeniP,
		EskiSeoBaslik: y.OldSeoTitle,
		YeniSeoBaslik: y.NewSeoTitle,
		EskiSeoAcik:   y.OldSeoDesc,
		YeniSeoAcik:   y.NewSeoDesc,
		Uyarilar:      y.Warnings,
		Hata:          y.Error,
		Provider:      y.Provider,
		Model:         y.Model,
		TokenGirdi:    y.PromptTokens,
		TokenCikti:    y.CompletionTokens,
		Deneme:        y.Attempts,
		EskiKarakter:  karakterSayisi(eskiP),
		YeniKarakter:  karakterSayisi(yeniP),
		EskiParagrafN: len(eskiP),
		YeniParagrafN: len(yeniP),
		Olusturma:     y.CreatedAt.Format("02.01.2006 15:04"),
	}
	if v.Baslik == "" {
		v.Baslik = y.OldTitle
	}
	return v
}

func karakterSayisi(p []string) int {
	n := 0
	for _, s := range p {
		n += len([]rune(s))
	}
	return n
}

// handleYenidenYazim GET /admin/yeniden-yazim — liste.
func (s *Server) handleYenidenYazim(w http.ResponseWriter, r *http.Request) {
	s.renderYenidenYazim(w, r, nil, "")
}

// handleYenidenYazimDetay GET /admin/yeniden-yazim/{id} — eski/yeni diff.
func (s *Server) handleYenidenYazimDetay(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	if id <= 0 {
		s.flashRedirect(w, r, "/admin/yeniden-yazim", "error", "Öneri bulunamadı.")
		return
	}
	y, err := repo.YenidenYazimTek(r.Context(), s.pool, id)
	if err != nil {
		s.flashRedirect(w, r, "/admin/yeniden-yazim", "error", "Öneri bulunamadı.")
		return
	}
	v := yenidenViewFrom(y)
	s.renderYenidenYazim(w, r, &v, "")
}

// handleYenidenYazimPilot POST /admin/yeniden-yazim/pilot — pilotu başlat.
func (s *Server) handleYenidenYazimPilot(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	kok, _ := strconv.Atoi(strings.TrimSpace(r.FormValue("kok")))
	if kok <= 0 {
		kok = 10
	}
	if kok > 100 {
		kok = 100
	}
	mod := strings.TrimSpace(r.FormValue("mod"))
	n, err := repo.YenidenYazimPilot(r.Context(), s.pool, s.curSiteID(r), kok, mod)
	if err != nil {
		s.flashRedirect(w, r, "/admin/yeniden-yazim", "error", "Pilot başlatılamadı: "+err.Error())
		return
	}
	if n == 0 {
		s.flashRedirect(w, r, "/admin/yeniden-yazim", "info",
			"Uygun içerik bulunamadı (hepsinin zaten açık bir önerisi olabilir).")
		return
	}
	ek := "Araştırmalı mod: her içerik için web kaynağı toplanır, haber o kaynaklardan yazılır."
	if mod == repo.YenidenModYeniden {
		ek = "Yalnız mevcut metin modu: kaynak eklenmez, metin düzenlenir."
	}
	s.flashRedirect(w, r, "/admin/yeniden-yazim", "info",
		strconv.Itoa(n)+" öneri kuyruğa alındı. "+ek+" Sonuçlar bu sayfada belirir.")
}

// handleYenidenYazimOnayla POST /admin/yeniden-yazim/{id}/onayla.
func (s *Server) handleYenidenYazimOnayla(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	u := userFromContext(r.Context())
	err := repo.YenidenYazimOnayla(r.Context(), s.pool, id, &u.ID)
	switch {
	case errors.Is(err, repo.ErrConflict):
		s.flashRedirect(w, r, "/admin/yeniden-yazim/"+strconv.FormatInt(id, 10), "error",
			"İçerik bu arada değişti (başka bir editör kaydetti). Öneri uygulanmadı; yeniden üretin.")
		return
	case err != nil:
		s.flashRedirect(w, r, "/admin/yeniden-yazim/"+strconv.FormatInt(id, 10), "error",
			"Onaylanamadı: "+err.Error())
		return
	}
	s.audit(r, "yeniden_yazim_onay", strconv.FormatInt(id, 10))
	s.flashRedirect(w, r, "/admin/yeniden-yazim", "info",
		"Yeni metin yayındaki içeriğe yazıldı. Eski hâl sürüm geçmişinde duruyor.")
}

// handleYenidenYazimReddet POST /admin/yeniden-yazim/{id}/reddet.
func (s *Server) handleYenidenYazimReddet(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	if err := repo.YenidenYazimReddet(r.Context(), s.pool, id, "editör reddetti"); err != nil {
		s.flashRedirect(w, r, "/admin/yeniden-yazim", "error", "Reddedilemedi: "+err.Error())
		return
	}
	s.flashRedirect(w, r, "/admin/yeniden-yazim", "info", "Öneri reddedildi; içerik değişmedi.")
}

// handleYenidenYazimTekrar POST /admin/yeniden-yazim/{id}/tekrar — yeniden üret.
func (s *Server) handleYenidenYazimTekrar(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	if err := repo.YenidenYazimYenidenDenen(r.Context(), s.pool, id); err != nil {
		s.flashRedirect(w, r, "/admin/yeniden-yazim", "error", "Yeniden denenemedi: "+err.Error())
		return
	}
	s.flashRedirect(w, r, "/admin/yeniden-yazim", "info", "Öneri yeniden kuyruğa alındı.")
}

// renderYenidenYazim paneli çizer (liste veya detay).
func (s *Server) renderYenidenYazim(w http.ResponseWriter, r *http.Request, detay *yenidenView, _ string) {
	ctx := r.Context()
	d := s.baseView("tr", "/admin", "Yeniden Yazım")
	d.User = ptr(userFromContext(ctx))
	d.Flash = s.consumeFlash(r)

	veri := &yenidenData{Flash: d.Flash, PilotAdet: "10", Mod: repo.YenidenModArastirmali}
	veri.FiltreDurum = r.URL.Query().Get("durum")
	veri.FiltreDil = r.URL.Query().Get("dil")

	ozet, err := repo.YenidenYazimOzet(ctx, s.pool, s.curSiteID(r))
	if err == nil {
		veri.Ozet = ozet
	} else {
		veri.Ozet = map[string]int{}
	}
	veri.AdaySayisi = repo.YenidenYazimAdaySayisi(ctx, s.pool, s.curSiteID(r))

	if detay != nil {
		veri.Detay = detay
	} else {
		satirlar, err := repo.YenidenYazimListele(ctx, s.pool, s.curSiteID(r),
			veri.FiltreDurum, veri.FiltreDil, 200)
		if err != nil {
			s.logger.Warn("yeniden yazım listesi okunamadı", "error", err)
		}
		for _, y := range satirlar {
			veri.Satirlar = append(veri.Satirlar, yenidenViewFrom(y))
		}
		veri.Toplam = len(veri.Satirlar)
	}
	d.Yeniden = veri
	s.render(w, r, "yeniden_yazim", d)
}
