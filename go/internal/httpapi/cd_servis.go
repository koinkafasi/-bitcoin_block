package httpapi

import (
	"context"
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

// ---- Namaz vakitleri ----

// ---- İmsakiye ----

// ---- Astroloji ----

// ---- Fikstür / puan durumu ----

// bosIseTire boş metni "—" yapar (tablo hücresi boş kalmasın).
func bosIseTire(v string) string {
	if strings.TrimSpace(v) == "" {
		return "—"
	}
	return v
}

// ---- Sayfa sarmalayıcıları ----

// ---- Resmi ilanlar (BİK) ----

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
