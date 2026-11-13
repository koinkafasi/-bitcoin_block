package httpapi

import (
	"encoding/json"
	"html"
	"net/http"
	"sort"
	"strings"

	"theblock/site/internal/webui/templkit"
)

// /sozluk — Kripto ve Blokzincir Sözlüğü.
//
// NEDEN TEK SAYFA, 55 AYRI SAYFA DEĞİL: her terime ayrı sayfa açmak
// 55 uzun kuyruk adresi üretirdi ama her biri ~40 kelimelik İNCE
// içerik olurdu. Google'ın "yararlı içerik" değerlendirmesi tam bunu
// cezalandırıyor ve site geneli kaliteyi aşağı çekiyor. Ölçtüğümüz
// sorunun (haber medyanı 78 kelime) üzerine 55 tane daha koymak
// anlamsız olurdu.
//
// Bunun yerine: tek derli toplu sayfa + her terime ÇIPA
// (/sozluk#blockchain) + DefinedTermSet yapısal verisi. Google
// sözlüğü bir bütün olarak anlıyor, çıpalar paylaşılabiliyor ve
// sayfa gerçek bir referans kaynağı oluyor.
//
// VERİ: cd_sozluk_veri.go (üretilmiş, Ghost temasından taşındı).

// sozlukSeviyeSirasi öğrenme sırası: önce giriş seviyesi.
var sozlukSeviyeSirasi = map[string]int{"Başlangıç": 0, "Orta": 1, "İleri": 2}

func (s *Server) handleSozlukPage(w http.ResponseWriter, r *http.Request) {
	kok := s.kanonikKok(r)
	v := s.cdSayfaKabuk(r, "sozluk",
		"Kripto Para ve Blokzincir Sözlüğü — "+s.siteName,
		"Kripto para, DeFi, NFT ve blokzincir terimlerinin Türkçe karşılıkları ve açıklamaları.",
		"Kripto Sözlüğü",
		"Kripto para, DeFi, Web3, NFT ve blokzincir teknolojisine dair temel kavramlar.",
		kok+"/sozluk")
	// Govde SUNUCUDA üretilmiş güvenli HTML: tüm değişken alanlar
	// html.EscapeString'den geçiyor, kullanıcı girdisi yok.
	v.Govde = templkit.SafeHTML(sozlukGovde())
	v.Serit = true
	// cdSayfaKabuk JSONLD'yi boş bırakıyor; sözlük kendi yapısal
	// verisini buraya koyuyor.
	v.JSONLD = sozlukJSONLD(kok)
	s.cdBasitRender(w, r, v)
}

// sozlukSirali terimleri sayfa düzeni için sıralar.
//
// ÖNCE SEVİYE, SONRA ALFABE: sözlüğü baştan okuyan biri "Blockchain"
// ile başlamalı, "Atomic Swap" ile değil. Harf süzgeci isteyene
// istemci tarafında zaten var.
func sozlukSirali() []SozlukTerimi {
	out := append([]SozlukTerimi(nil), sozlukTerimleri...)
	sort.SliceStable(out, func(i, j int) bool {
		si, sj := sozlukSeviyeSirasi[out[i].Seviye], sozlukSeviyeSirasi[out[j].Seviye]
		if si != sj {
			return si < sj
		}
		return strings.ToLower(out[i].Terim) < strings.ToLower(out[j].Terim)
	})
	return out
}

// sozlukKategoriler sayfadaki süzgeç düğmeleri için.
func sozlukKategoriler() []string {
	gorulen := map[string]bool{}
	var out []string
	for _, t := range sozlukTerimleri {
		if t.Kategori != "" && !gorulen[t.Kategori] {
			gorulen[t.Kategori] = true
			out = append(out, t.Kategori)
		}
	}
	sort.Strings(out)
	return out
}

func sozlukGovde() string {
	terimler := sozlukSirali()

	var b strings.Builder
	b.WriteString(`<p class="cd-card__spot">` +
		`Kripto para, DeFi, NFT ve blokzincir dünyasının ` +
		itoa(len(terimler)) + ` temel terimi; İngilizce karşılıkları, ` +
		`Türkçe çevirileri ve kısa açıklamalarıyla.</p>`)

	// SÜZGEÇLER JS GEREKTİRMEDEN ÇALIŞIR: CSS :target yerine basit
	// bağlantılar kullanılmıyor; süzgeç istemci tarafında, JS kapalıysa
	// TÜM terimler görünür kalıyor. Arama motoru da tümünü görür.
	b.WriteString(`<div class="cd-sozluk__suzgec" data-sozluk-suzgec>`)
	b.WriteString(`<input class="cd-sozluk__ara" type="search" placeholder="Terim ara (ör. Smart Contract, Gas)" ` +
		`aria-label="Sözlükte ara" data-sozluk-ara>`)
	b.WriteString(`<span class="cd-sozluk__etiketler">`)
	for _, k := range sozlukKategoriler() {
		b.WriteString(`<button type="button" class="cd-sozluk__etiket" data-sozluk-kategori="` +
			html.EscapeString(k) + `">` + html.EscapeString(k) + `</button>`)
	}
	b.WriteString(`</span></div>`)

	b.WriteString(`<dl class="cd-sozluk" data-sozluk-liste>`)
	for _, t := range terimler {
		b.WriteString(`<div class="cd-sozluk__madde" id="` + html.EscapeString(t.Slug) + `"` +
			` data-kategori="` + html.EscapeString(t.Kategori) + `"` +
			` data-seviye="` + html.EscapeString(t.Seviye) + `"` +
			` data-arama="` + html.EscapeString(strings.ToLower(t.Terim+" "+t.Turkce)) + `">`)

		b.WriteString(`<dt class="cd-sozluk__terim">`)
		// ÇIPA BAĞLANTISI: /sozluk#blockchain paylaşılabilir olmalı.
		b.WriteString(`<a class="cd-sozluk__cipa" href="#` + html.EscapeString(t.Slug) +
			`" aria-label="` + html.EscapeString(t.Terim) + ` bağlantısı">#</a> `)
		b.WriteString(html.EscapeString(t.Terim))
		if t.Turkce != "" {
			b.WriteString(` <span class="cd-sozluk__tr">(` + html.EscapeString(t.Turkce) + `)</span>`)
		}
		if t.Kategori != "" {
			b.WriteString(` <span class="cd-sozluk__rozet">` + html.EscapeString(t.Kategori) + `</span>`)
		}
		if t.Seviye != "" {
			b.WriteString(` <span class="cd-sozluk__rozet cd-sozluk__rozet--seviye">` +
				html.EscapeString(t.Seviye) + `</span>`)
		}
		b.WriteString(`</dt>`)

		b.WriteString(`<dd class="cd-sozluk__aciklama">` + html.EscapeString(t.Aciklama) + `</dd>`)
		b.WriteString(`</div>`)
	}
	b.WriteString(`</dl>`)
	return b.String()
}

// sozlukJSONLD DefinedTermSet yapısal verisi.
//
// Google sözlükleri bu tiple anlıyor: terimler tek tek `DefinedTerm`
// olarak veriliyor ve her birinin kendi çıpası `url` alanında.
// Tek tek sayfa açmadan terim düzeyinde görünürlük sağlayan yol bu.
func sozlukJSONLD(kok string) string {
	terimler := sozlukSirali()
	liste := make([]any, 0, len(terimler))
	for _, t := range terimler {
		ad := t.Terim
		if t.Turkce != "" {
			ad = t.Terim + " (" + t.Turkce + ")"
		}
		liste = append(liste, map[string]any{
			"@type":            "DefinedTerm",
			"name":             ad,
			"description":      t.Aciklama,
			"url":              kok + "/sozluk#" + t.Slug,
			"inDefinedTermSet": kok + "/sozluk",
		})
	}
	ham, err := json.Marshal(map[string]any{
		"@context":       "https://schema.org",
		"@type":          "DefinedTermSet",
		"name":           "Kripto Para ve Blokzincir Sözlüğü",
		"description":    "Kripto para, DeFi, Web3, NFT ve blokzincir terimlerinin Türkçe açıklamaları.",
		"url":            kok + "/sozluk",
		"inLanguage":     "tr",
		"hasDefinedTerm": liste,
	})
	if err != nil {
		return ""
	}
	return `<script type="application/ld+json">` + string(ham) + `</script>`
}
