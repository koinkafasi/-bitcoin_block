package httpapi

// Yeni kabukta basılan HİZMET sayfaları.
//
// NEDEN: /hava-durumu, /namaz-vakitleri, /imsakiye, /astroloji, /fikstur,
// /resmi-ilanlar ve /yazarlar theblock kabuğunu (site_header/site_footer)
// kullanıyordu — menüden tıklanan kullanıcı eski tasarıma düşüyordu
// (kullanıcı bildirimi, 9 Kasım 2026).
//
// YAKLAŞIM: her sayfa için yeni bir templ dosyası açmak yerine ORTAK
// `CDBasitSayfa` iskeleti kullanılır; sayfa yalnız GÖVDESİNİ üretir.
// Veri kaynakları DEĞİŞMEDİ (aynı repo fonksiyonları, aynı önbellekler):
// bu bir görünüm geçişi, veri değişikliği değil.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"theblock/site/internal/repo"
	"theblock/site/internal/webui/templkit"
)

// cdBasitRender ortak iskeleti basar.
func (s *Server) cdBasitRender(w http.ResponseWriter, r *http.Request, v templkit.CDBasitSayfa) {
	if v.Shell.SiteName == "" {
		v.Shell = s.cdShellCtx(r.Context(), r, "")
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", cacheControlPublic)
	if err := templkit.CDBasitSayfaView(v).Render(r.Context(), w); err != nil {
		s.logger.Error("cd basit sayfa render", "path", r.URL.Path, "error", err)
	}
}

// cdSayfaKabuk ortak alanları doldurur.
func (s *Server) cdSayfaKabuk(r *http.Request, aktif, title, desc, baslik, aciklama, kanonik string) templkit.CDBasitSayfa {
	return templkit.CDBasitSayfa{
		Shell:           s.cdShellCtx(r.Context(), r, aktif),
		Title:           title,
		Desc:            desc,
		Canonical:       kanonik,
		Baslik:          baslik,
		Aciklama:        aciklama,
		BaseURL:         strings.TrimRight(s.cfg.BaseURL, "/"),
		Locale:          "tr",
		PageCategory:    "service_page",
		ContentCategory: aktif,
	}
}

// ---- Hava durumu ----

// cdHavaGovde 81 ilin sıcaklık/açıklama tablosu.
//
// Veri: `hava` önbelleği (repo.CacheList). Önbellek boşsa DÜRÜST boş
// durum basılır — uydurma sıcaklık yazılmaz.
func (s *Server) cdHavaGovde(ctx context.Context) templkit.SafeHTML {
	cities, _ := repo.ListCities(ctx, s.pool)
	rows, _ := repo.CacheList(ctx, s.pool, "hava")
	byKey := map[string]repo.CacheRow{}
	for _, row := range rows {
		byKey[row.Key] = row
	}
	if len(cities) == 0 {
		return templkit.SafeHTML(`<p class="cd-card__spot">Şehir listesi alınamadı.</p>`)
	}

	var b strings.Builder
	b.WriteString(`<div class="cd-grid cd-grid--3 cd-hava">`)
	dolu := 0
	for _, c := range cities {
		val, ok := "—", false
		if rw, var_ := byKey[strconv.Itoa(c.Plaka)]; var_ {
			var w repo.WeatherInfo
			if json.Unmarshal(rw.Data, &w) == nil {
				val, ok = fmt.Sprintf("%.0f° %s", w.Sicaklik, w.Aciklama), true
			}
		}
		if ok {
			dolu++
		}
		b.WriteString(`<div class="cd-basit__satir"><strong>` + htmlEscape(c.Name) +
			`</strong><span>` + htmlEscape(val) + `</span></div>`)
	}
	b.WriteString(`</div>`)
	if dolu == 0 {
		b.WriteString(`<p class="cd-card__spot">Hava verisi şu an alınamıyor; birkaç dakika sonra yeniden deneyin.</p>`)
	}
	return templkit.SafeHTML(b.String())
}

// ---- Namaz vakitleri ----

// cdNamazGovde illerin bugünkü vakitleri (öğle/ikindi/akşam/yatsı).
func (s *Server) cdNamazGovde(ctx context.Context) templkit.SafeHTML {
	cities, _ := repo.ListCities(ctx, s.pool)
	rows, _ := repo.CacheList(ctx, s.pool, "namaz")
	byKey := map[string]repo.CacheRow{}
	for _, row := range rows {
		byKey[row.Key] = row
	}
	if len(cities) == 0 {
		return templkit.SafeHTML(`<p class="cd-card__spot">Şehir listesi alınamadı.</p>`)
	}

	var b strings.Builder
	b.WriteString(`<div class="cd-tablo-sarmal"><table class="cd-table"><thead><tr>` +
		`<th>Şehir</th><th>İmsak</th><th>Güneş</th><th class="num">Öğle</th>` +
		`<th class="num">İkindi</th><th class="num">Akşam</th><th class="num">Yatsı</th>` +
		`</tr></thead><tbody>`)
	for _, c := range cities {
		v := repo.NamazVakit{}
		if rw, ok := byKey[strconv.Itoa(c.Plaka)]; ok {
			_ = json.Unmarshal(rw.Data, &v)
		}
		b.WriteString(`<tr><td><strong>` + htmlEscape(c.Name) + `</strong></td>`)
		for _, s := range []string{v.Imsak, v.Gunes, v.Ogle, v.Ikindi, v.Aksam, v.Yatsi} {
			b.WriteString(`<td class="num">` + htmlEscape(bosIseTire(s)) + `</td>`)
		}
		b.WriteString(`</tr>`)
	}
	b.WriteString(`</tbody></table></div>`)
	return templkit.SafeHTML(b.String())
}

// ---- İmsakiye ----

// cdImsakiyeGovde ramazan imsakiyesi (30 gün).
func (s *Server) cdImsakiyeGovde(ctx context.Context) templkit.SafeHTML {
	var list []repo.NamazVakit
	if b, err := repo.CacheGet(ctx, s.pool, "imsakiye", "6"); err == nil {
		_ = json.Unmarshal(b, &list)
	}
	if len(list) == 0 {
		return templkit.SafeHTML(`<p class="cd-card__spot">İmsakiye verisi şu an alınamıyor.</p>`)
	}
	var b strings.Builder
	b.WriteString(`<div class="cd-tablo-sarmal"><table class="cd-table"><thead><tr>` +
		`<th>Tarih</th><th>İmsak</th><th>Güneş</th><th class="num">Öğle</th>` +
		`<th class="num">İkindi</th><th class="num">Akşam</th><th class="num">Yatsı</th>` +
		`</tr></thead><tbody>`)
	for _, v := range list {
		b.WriteString(`<tr><td>` + strconv.Itoa(v.Gun) + ". gün" + `</td>`)
		for _, x := range []string{v.Imsak, v.Gunes, v.Ogle, v.Ikindi, v.Aksam, v.Yatsi} {
			b.WriteString(`<td class="num">` + htmlEscape(bosIseTire(x)) + `</td>`)
		}
		b.WriteString(`</tr>`)
	}
	b.WriteString(`</tbody></table></div>`)
	return templkit.SafeHTML(b.String())
}

// ---- Astroloji ----

// cdAstrolojiGovde 12 burcun günlük yorumu.
func (s *Server) cdAstrolojiGovde(ctx context.Context) templkit.SafeHTML {
	ad := repo.AstrolojiOf(ctx, s.pool)
	if len(ad.Burclar) == 0 {
		return templkit.SafeHTML(`<p class="cd-card__spot">Burç yorumları şu an alınamıyor.</p>`)
	}
	var b strings.Builder
	b.WriteString(`<div class="cd-grid cd-grid--3">`)
	for _, burc := range ad.Burclar {
		b.WriteString(`<article class="cd-basit__kart"><h2 class="cd-basit__kart-ad">` +
			htmlEscape(burc.Ad) + `</h2><p class="cd-card__spot">` + htmlEscape(burc.Tarih) +
			`</p><p>` + htmlEscape(burc.Yorum) + `</p></article>`)
	}
	b.WriteString(`</div>`)
	return templkit.SafeHTML(b.String())
}

// ---- Fikstür / puan durumu ----

// cdFiksturGovde lig puan tablosu + haftanın maçları.
func (s *Server) cdFiksturGovde(ctx context.Context, r *http.Request) templkit.SafeHTML {
	lg := repo.LeagueByKey(strings.TrimSpace(r.URL.Query().Get("lig")))
	standings := repo.StandingsOfLeague(ctx, s.pool, lg).Rows
	meta := repo.LeagueMetaOf(ctx, s.pool, lg)
	round := parseFiksturRound(r.URL.Query().Get("hafta"), meta)
	fx := repo.FiksturRoundOf(ctx, s.pool, lg, meta.Season, round)

	var b strings.Builder
	// Lig seçici.
	b.WriteString(`<nav class="cd-basit__cipler" aria-label="Lig seçimi">`)
	for _, l := range repo.Leagues {
		sinif := "cd-basit__cip"
		if l.Key == lg.Key {
			sinif += " cd-basit__cip--aktif"
		}
		b.WriteString(`<a class="` + sinif + `" href="?lig=` + htmlEscape(l.Key) + `">` +
			htmlEscape(l.Name) + `</a>`)
	}
	b.WriteString(`</nav>`)

	if len(standings) > 0 {
		b.WriteString(`<h2 class="cd-hisse__baslik">Puan Durumu</h2>`)
		b.WriteString(`<div class="cd-tablo-sarmal"><table class="cd-table"><thead><tr>` +
			`<th>#</th><th>Takım</th><th class="num">O</th><th class="num">G</th>` +
			`<th class="num">B</th><th class="num">M</th><th class="num">P</th></tr></thead><tbody>`)
		for i, t := range standings {
			b.WriteString(`<tr><td>` + strconv.Itoa(i+1) + `</td><td><strong>` +
				htmlEscape(t.Team) + `</strong></td>` +
				`<td class="num">` + strconv.Itoa(t.Played) + `</td>` +
				`<td class="num">` + strconv.Itoa(t.Win) + `</td>` +
				`<td class="num">` + strconv.Itoa(t.Draw) + `</td>` +
				`<td class="num">` + strconv.Itoa(t.Loss) + `</td>` +
				`<td class="num"><strong>` + strconv.Itoa(t.Points) + `</strong></td></tr>`)
		}
		b.WriteString(`</tbody></table></div>`)
	}

	if len(fx.Matches) > 0 {
		b.WriteString(`<h2 class="cd-hisse__baslik">` + strconv.Itoa(round) + `. Hafta</h2>`)
		b.WriteString(`<div class="cd-grid cd-grid--2">`)
		for _, m := range fx.Matches {
			skor := "—"
			if m.HomeScore != nil && m.AwayScore != nil {
				skor = strconv.Itoa(*m.HomeScore) + " - " + strconv.Itoa(*m.AwayScore)
			}
			b.WriteString(`<div class="cd-basit__satir"><span>` + htmlEscape(m.Home) +
				`</span><strong>` + htmlEscape(skor) + `</strong><span>` +
				htmlEscape(m.Away) + `</span></div>`)
		}
		b.WriteString(`</div>`)
	}
	if b.Len() == 0 || (len(standings) == 0 && len(fx.Matches) == 0) {
		b.WriteString(`<p class="cd-card__spot">Puan durumu ve fikstür verisi şu an alınamıyor.</p>`)
	}
	return templkit.SafeHTML(b.String())
}

// bosIseTire boş metni "—" yapar (tablo hücresi boş kalmasın).
func bosIseTire(v string) string {
	if strings.TrimSpace(v) == "" {
		return "—"
	}
	return v
}

// ---- Sayfa sarmalayıcıları ----

// cdHavaSayfasi /hava-durumu
func (s *Server) cdHavaSayfasi(w http.ResponseWriter, r *http.Request) {
	v := s.cdSayfaKabuk(r, "hava-durumu",
		"Hava Durumu — "+s.siteName,
		"81 ilin güncel hava durumu ve sıcaklık bilgisi.",
		"Hava Durumu", "81 ilin güncel sıcaklığı ve hava durumu.",
		s.kanonikKok(r)+"/hava-durumu")
	v.Govde = s.cdHavaGovde(r.Context())
	v.Serit = true
	s.cdBasitRender(w, r, v)
}

// cdNamazSayfasi /namaz-vakitleri
func (s *Server) cdNamazSayfasi(w http.ResponseWriter, r *http.Request) {
	v := s.cdSayfaKabuk(r, "namaz-vakitleri",
		"Namaz Vakitleri — "+s.siteName,
		"81 il için günlük namaz vakitleri: imsak, güneş, öğle, ikindi, akşam, yatsı.",
		"Namaz Vakitleri", "81 il için bugünün namaz vakitleri.",
		s.kanonikKok(r)+"/namaz-vakitleri")
	v.Govde = s.cdNamazGovde(r.Context())
	v.Serit = true
	s.cdBasitRender(w, r, v)
}

// cdImsakiyeSayfasi /imsakiye
func (s *Server) cdImsakiyeSayfasi(w http.ResponseWriter, r *http.Request) {
	v := s.cdSayfaKabuk(r, "imsakiye",
		"İmsakiye — "+s.siteName,
		"Ramazan imsakiyesi: 30 günlük imsak, güneş, öğle, ikindi, akşam ve yatsı vakitleri.",
		"İmsakiye", "Ramazan ayı boyunca günlük vakitler.",
		s.kanonikKok(r)+"/imsakiye")
	v.Govde = s.cdImsakiyeGovde(r.Context())
	v.Serit = true
	s.cdBasitRender(w, r, v)
}

// cdAstrolojiSayfasi /astroloji
func (s *Server) cdAstrolojiSayfasi(w http.ResponseWriter, r *http.Request) {
	v := s.cdSayfaKabuk(r, "astroloji",
		"Astroloji — Günlük Burç Yorumları — "+s.siteName,
		"12 burcun günlük yorumu, elementi, uğurlu sayısı ve rengi.",
		"Astroloji", "12 burcun günlük yorumu.",
		s.kanonikKok(r)+"/astroloji")
	v.Govde = s.cdAstrolojiGovde(r.Context())
	v.Serit = true
	s.cdBasitRender(w, r, v)
}

// cdFiksturSayfasi /fikstur
func (s *Server) cdFiksturSayfasi(w http.ResponseWriter, r *http.Request) {
	v := s.cdSayfaKabuk(r, "fikstur",
		"Puan Durumu ve Fikstür — "+s.siteName,
		"Süper Lig puan durumu, haftanın maç programı ve sonuçları.",
		"Puan Durumu ve Fikstür", "Lig puan durumu ve haftanın maçları.",
		s.kanonikKok(r)+"/fikstur")
	v.Govde = s.cdFiksturGovde(r.Context(), r)
	v.Serit = true
	s.cdBasitRender(w, r, v)
}

// ---- Resmi ilanlar (BİK) ----

// cdResmiIlanGovde resmi ilan listesi (filtre + tablo).
//
// Kaynak: repo.FilterNotices — aynı veri, yalnız görünüm değişti.
func (s *Server) cdResmiIlanGovde(ctx context.Context, r *http.Request, siteID int64) templkit.SafeHTML {
	f := noticeFilterFromQuery(r.URL.Query())
	items, last, _ := repo.FilterNotices(ctx, s.pool, siteID, f)

	var b strings.Builder
	// Filtre formu: kurum/şehir/arama.
	b.WriteString(`<form class="cd-basit__filtre" method="get" action="/resmi-ilanlar" role="search">` +
		`<input class="cd-basit__girdi" type="search" name="q" placeholder="Kurum, şehir veya başlık ara" ` +
		`value="` + htmlEscape(f.Query) + `" aria-label="Resmi ilan ara">` +
		`<button class="cd-basit__dugme" type="submit">Ara</button>`)
	if f.Query != "" || f.Institution != "" || f.City != "" {
		b.WriteString(`<a class="cd-basit__temizle" href="/resmi-ilanlar">Temizle</a>`)
	}
	b.WriteString(`</form>`)

	if last != "" {
		b.WriteString(`<p class="cd-card__spot">Son ilan tarihi: ` + htmlEscape(last) + `</p>`)
	}
	if len(items) == 0 {
		b.WriteString(`<p class="cd-card__spot">Bu filtreye uygun resmi ilan yok.</p>`)
		return templkit.SafeHTML(b.String())
	}

	b.WriteString(`<div class="cd-tablo-sarmal"><table class="cd-table"><thead><tr>` +
		`<th>Tarih</th><th>Kurum</th><th>Şehir</th><th>İlan</th></tr></thead><tbody>`)
	for _, it := range items {
		b.WriteString(`<tr><td>` + htmlEscape(bosIseTire(it.NoticeDate)) + `</td>` +
			`<td>` + htmlEscape(bosIseTire(it.Institution)) + `</td>` +
			`<td>` + htmlEscape(bosIseTire(it.City)) + `</td><td>`)
		if it.URL != "" {
			b.WriteString(`<a href="` + htmlEscape(it.URL) + `" rel="noopener" target="_blank">` +
				htmlEscape(it.Title) + `</a>`)
		} else {
			b.WriteString(htmlEscape(it.Title))
		}
		b.WriteString(`</td></tr>`)
	}
	b.WriteString(`</tbody></table></div>`)
	return templkit.SafeHTML(b.String())
}

// ---- Yazarlar ----

// cdYazarlarGovde yazar/editör kart ızgarası.
func (s *Server) cdYazarlarGovde(ctx context.Context) templkit.SafeHTML {
	people, _ := repo.ListPeople(ctx, s.pool, []string{"Yönetici", "Editör", "Muhabir", "Yazar"})
	if len(people) == 0 {
		return templkit.SafeHTML(`<p class="cd-card__spot">Yazar listesi şu an alınamıyor.</p>`)
	}
	var b strings.Builder
	b.WriteString(`<div class="cd-grid cd-grid--3 cd-yazarlar">`)
	for _, p := range people {
		b.WriteString(`<a class="cd-basit__kart cd-basit__kart--bag" href="/yazar/` +
			htmlEscape(p.Slug) + `">`)
		if p.AvatarMediaID != nil {
			b.WriteString(`<img class="cd-basit__avatar" src="/medya/` +
				strconv.FormatInt(*p.AvatarMediaID, 10) + `" alt="` + htmlEscape(p.Name) +
				`" loading="lazy" decoding="async">`)
		} else {
			b.WriteString(`<span class="cd-basit__avatar cd-basit__avatar--harf" aria-hidden="true">` +
				htmlEscape(ilkHarf(p.Name)) + `</span>`)
		}
		b.WriteString(`<h2 class="cd-basit__kart-ad">` + htmlEscape(p.Name) + `</h2>`)
		if p.Title != "" {
			b.WriteString(`<p class="cd-card__spot">` + htmlEscape(p.Title) + `</p>`)
		}
		if p.ArticleCount > 0 {
			b.WriteString(`<p class="cd-card__meta">` + strconv.Itoa(p.ArticleCount) + ` yazı</p>`)
		}
		b.WriteString(`</a>`)
	}
	b.WriteString(`</div>`)
	return templkit.SafeHTML(b.String())
}

// cdResmiIlanSayfasi /resmi-ilanlar
func (s *Server) cdResmiIlanSayfasi(w http.ResponseWriter, r *http.Request, siteID int64) {
	v := s.cdSayfaKabuk(r, "resmi-ilanlar",
		"Resmi İlanlar (BİK) — "+s.siteName,
		"Basın İlan Kurumu resmi ilanları: ihale, duyuru ve tebliğler.",
		"Resmi İlanlar", "Basın İlan Kurumu resmi ilanları.",
		s.kanonikKok(r)+"/resmi-ilanlar")
	v.Govde = s.cdResmiIlanGovde(r.Context(), r, siteID)
	v.Serit = true
	s.cdBasitRender(w, r, v)
}

// cdYazarlarSayfasi /yazarlar
func (s *Server) cdYazarlarSayfasi(w http.ResponseWriter, r *http.Request) {
	v := s.cdSayfaKabuk(r, "yazarlar",
		"Yazarlar ve Editörler — "+s.siteName,
		"Haberleri kaleme alan yazarlar, muhabirler ve editörler.",
		"Yazarlar ve Editörler", "Haberleri kaleme alan yazarlar, muhabirler ve editörler.",
		s.kanonikKok(r)+"/yazarlar")
	v.Govde = s.cdYazarlarGovde(r.Context())
	v.Serit = true
	s.cdBasitRender(w, r, v)
}
