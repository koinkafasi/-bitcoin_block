package httpapi

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"theblock/site/internal/platform"
	"theblock/site/internal/repo"
)

// ---- Resmi ilanlar / BİK (spec 02 §217) ----

type noticesData struct {
	Items []repo.OfficialNotice
	Date  string
	Dates []string
	Total int64
}

// handleNoticesAdmin panel: resmi ilan yönetimi.
func (s *Server) handleNoticesAdmin(w http.ResponseWriter, r *http.Request) {
	d := s.buildNoticesData(r)
	if IsHTMX(r) {
		s.renderIlanlarPanel(w, r, d)
		return
	}
	s.renderIlanlarPage(w, r, d)
}

func (s *Server) buildNoticesData(r *http.Request) *tmplData {
	ctx := r.Context()
	sid := s.curSiteID(r)
	date := strings.TrimSpace(r.URL.Query().Get("tarih"))
	items, last, _ := repo.ListNotices(ctx, s.pool, sid, date, 300)
	dates, _ := repo.NoticeDates(ctx, s.pool, sid, 30)
	total, _ := repo.CountNotices(ctx, s.pool, sid)
	d := s.baseView("tr", "/admin", "Resmi İlanlar (BİK)")
	d.User = ptr(userFromContext(ctx))
	d.Flash = s.consumeFlash(r)
	d.IlanlarPanel = &noticesData{Items: items, Date: last, Dates: dates, Total: total}
	return d
}

func (s *Server) renderNoticesPanel(w http.ResponseWriter, r *http.Request, kind, text string) {
	d := s.buildNoticesData(withHTMXCurrentQuery(r))
	if kind != "" {
		d.Flash = &flashMsg{Kind: kind, Text: text}
	}
	s.renderIlanlarPanel(w, r, d)
}

func (s *Server) noticeResult(w http.ResponseWriter, r *http.Request, kind, text string) {
	if IsHTMX(r) {
		s.renderNoticesPanel(w, r, kind, text)
		return
	}
	s.flashRedirect(w, r, "/admin/resmi-ilanlar", kind, text)
}

// handleNoticeSave ilan ekler/günceller.
func (s *Server) handleNoticeSave(w http.ResponseWriter, r *http.Request) {
	title := strings.TrimSpace(r.FormValue("baslik"))
	if title == "" {
		s.noticeResult(w, r, "error", "İlan başlığı gerekli.")
		return
	}
	date := strings.TrimSpace(r.FormValue("tarih"))
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	n := &repo.OfficialNotice{
		SiteID:      s.curSiteID(r),
		Title:       title,
		Institution: strings.TrimSpace(r.FormValue("kurum")),
		City:        strings.TrimSpace(r.FormValue("sehir")),
		NoticeDate:  date,
		URL:         strings.TrimSpace(r.FormValue("url")),
		Body:        strings.TrimSpace(r.FormValue("metin")),
		Sort:        parseIntOr(r.FormValue("sira")),
	}
	if err := repo.UpsertNotice(r.Context(), s.pool, n); err != nil {
		s.logger.Error("ilan kaydı", "error", err)
		s.noticeResult(w, r, "error", "Kaydedilemedi: "+err.Error())
		return
	}
	s.audit(r, "ilan_kaydet", title)
	if IsHTMX(r) {
		r2 := withHTMXCurrentQuery(r)
		q := r2.URL.Query()
		q.Set("tarih", date)
		r2.URL.RawQuery = q.Encode()
		d := s.buildNoticesData(r2)
		d.Flash = &flashMsg{Kind: "info", Text: "İlan kaydedildi: " + title}
		s.renderIlanlarPanel(w, r, d)
		return
	}
	s.flashRedirect(w, r, "/admin/resmi-ilanlar?tarih="+date, "info", "İlan kaydedildi: "+title)
}

// handleNoticeDelete ilanı siler.
func (s *Server) handleNoticeDelete(w http.ResponseWriter, r *http.Request) {
	if err := repo.DeleteNotice(r.Context(), s.pool, parseID(r.PathValue("id"))); err != nil {
		s.noticeResult(w, r, "error", "Silinemedi.")
		return
	}
	s.audit(r, "ilan_sil", r.PathValue("id"))
	s.noticeResult(w, r, "info", "İlan silindi.")
}

// handleNoticesPublic genel resmi ilanlar sayfası.
func (s *Server) handleNoticesPublic(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	locale := langOf(r, s)
	site := s.siteByDomain(ctx, r.Host)
	f := noticeFilterFromQuery(r.URL.Query())
	items, last, _ := repo.FilterNotices(ctx, s.pool, site.ID, f)
	dates, _ := repo.NoticeDates(ctx, s.pool, site.ID, 30)
	kurumlar, sehirler, _ := repo.NoticeFacets(ctx, s.pool, site.ID)
	d := s.baseView(locale, s.langBase(locale), "Resmi İlanlar")
	d.IlanKurumlar = kurumlar
	d.IlanSehirler = sehirler
	d.IlanFiltre = noticeFilterView{
		Tarih: f.Date, Kurum: f.Institution, Sehir: f.City, Arama: f.Query,
		Aktif: f.Institution != "" || f.City != "" || f.Query != "",
		Tumu:  strings.EqualFold(f.Date, repo.NoticeAllDates),
	}
	d.LangBase = s.langBase(locale)
	d.HomeURL = homeURL(locale)
	d.NavCategories = s.navCategories(ctx)
	d.Menu = s.publicMenu(ctx)
	d.Footer = s.footerConfig(ctx)
	d.Theme = themeFromReq(r)
	// Kanonik adres sorgudan bağımsızdır: filtre kombinasyonları aynı
	// listenin varyantıdır, ayrı sayfa değil.
	d.Canonical = strings.TrimRight(s.absBase(r), "/") + d.LangBase + "/resmi-ilanlar"
	d.Ilanlar = items
	d.IlanTarih = last
	d.IlanTarihler = dates
	s.renderPublic(w, r, "resmi-ilanlar", d, cacheControlPublic)
}

// noticeFilterFromQuery public liste sorgusunu NoticeFilter'a çevirir.
//
// Kural: kurum/şehir/arama verildiğinde ve tarih seçilmediğinde varsayılan
// "tüm tarihler" olur. Aksi hâlde daraltma yalnız en son güne uygulanır ve
// kullanıcı aradığını bulamaz ("Ankara" seçip boş liste görmek gibi).
func noticeFilterFromQuery(q url.Values) repo.NoticeFilter {
	f := repo.NoticeFilter{
		Date:        strings.TrimSpace(q.Get("tarih")),
		Institution: strings.TrimSpace(q.Get("kurum")),
		City:        strings.TrimSpace(q.Get("sehir")),
		Query:       strings.TrimSpace(q.Get("q")),
		Limit:       300,
	}
	if f.Date == "" && (f.Institution != "" || f.City != "" || f.Query != "") {
		f.Date = repo.NoticeAllDates
	}
	return f
}

// noticeSlug ilan başlığından adres parçası üretir. Kimlik adresin sonunda
// ayrıca taşındığı için slug yalnızca okunabilirlik ve SEO içindir; başlık
// değişse bile eski adres çalışmaya devam eder.
func noticeSlug(title string) string {
	s := platform.Slugify(title)
	// Slugify boş/anlamsız başlıkta "icerik" yer tutucusunu döndürür;
	// ilan adresinde "ilan" daha doğru.
	if s == "" || s == "icerik" {
		return "ilan"
	}
	// Resmî ilan başlıkları çok uzun olabiliyor (mahkeme/kurum adları);
	// adresi kullanılabilir tutmak için kırpılır.
	if len(s) > 80 {
		s = s[:80]
		if i := strings.LastIndex(s, "-"); i > 20 {
			s = s[:i]
		}
	}
	return s
}

// publicNoticeURL ilanın kalıcı adresidir: /resmi-ilanlar/<slug>-<id>
func publicNoticeURL(langBase string, n repo.OfficialNotice) string {
	return langBase + "/resmi-ilanlar/" + noticeSlug(n.Title) + "-" + itoa64(n.ID)
}

// handleNoticeDetail tek bir resmî ilanın kalıcı sayfasıdır.
//
// Neden kalıcı adres: BİK tebliği içeriklere ait TAM URL adreslerinin site
// haritasında bulunmasını istiyor. Ayrıca ilan metninin kendi adresinde
// yayımlanması arşiv ve atıf açısından da doğru; liste sayfasındaki kart
// kaybolduğunda ilan erişilemez olmamalı.
func (s *Server) handleNoticeDetail(w http.ResponseWriter, r *http.Request, seg string) {
	ctx := r.Context()
	locale := langOf(r, s)
	site := s.siteByDomain(ctx, r.Host)

	// Adres biçimi: <slug>-<id>. Kimlik yoksa veya çözülemezse 404.
	id := articleIDFromSegment(seg)
	if id <= 0 {
		s.handleNotFound(w, r)
		return
	}
	n, err := repo.NoticeByID(ctx, s.pool, site.ID, id)
	if err != nil {
		s.handleNotFound(w, r)
		return
	}

	langBase := s.langBase(locale)
	dogruYol := publicNoticeURL(langBase, n)
	// Başlık değiştiyse eski slug'la gelen istek kanonik adrese taşınır:
	// aynı ilan iki adreste indekslenmesin.
	if r.URL.Path != dogruYol {
		http.Redirect(w, r, dogruYol, http.StatusMovedPermanently)
		return
	}

	d := s.baseView(locale, langBase, n.Title)
	d.LangBase = langBase
	d.HomeURL = homeURL(locale)
	d.NavCategories = s.navCategories(ctx)
	d.Menu = s.publicMenu(ctx)
	d.Footer = s.footerConfig(ctx)
	d.Theme = themeFromReq(r)
	d.Canonical = strings.TrimRight(s.absBase(r), "/") + dogruYol
	d.Ilan = &n
	s.renderPublic(w, r, "resmi-ilan-detay", d, cacheControlPublic)
}
