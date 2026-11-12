package aieditor

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"theblock/site/internal/repo"
)

const (
	// Haber gövdesi alt sınırları. Google'ın "thin content" eşiği değil,
	// editoryal ölçüt: 3 paragraf altındaki metin haber değil, duyurudur.
	haberMinParagraf = 3
	haberMinKarakter = 700
	haberMaxParagraf = 12

	// Başlık/spot bantları (rune).
	haberBaslikMin = 30
	haberBaslikMax = 110
	haberSpotMin   = 80
	haberSpotMax   = 240
)

var (
	varlikRe   = regexp.MustCompile(`&[a-zA-Z]{2,10};|&#[0-9]{2,5};`)
	htmlEtiket = regexp.MustCompile(`<(/?)(p|br|div|span|a|strong|em|ul|ol|li|h[1-6])\b[^>]*>`)
	// Satır içi kalıntı: "devamını oku", "haberin devamı", kaynak sitesi
	// yönlendirmesi. Okuyucuya kazandırmıyor, SEO'da kopya metin sayılıyor.
	kalintiRe = regexp.MustCompile(`(?i)(devam[ıi]n[ıi] oku|haberin devam[ıi]|` +
		`t[ıi]klay[ıi]n[ıi]z|okumaya devam|kayna[ğg]a git|buradan ula[şs]abilirsiniz)`)
)

func yapiUyarilari(r *Result, act Action, kaynak string) []string {
	if act.ID != "news" && act.ID != "research_news" && act.ID != "audio_draft" {
		return nil
	}
	var out []string
	govde := alanDegeri(r, "body")
	paragraflar := paragraflarOf(govde)

	if act.ID == "news" || act.ID == "research_news" {
		if n := len(paragraflar); n < haberMinParagraf {
			out = append(out, fmt.Sprintf(
				"Gövde %d paragraf; haber için en az %d paragraf gerekir. Tek blok hâlinde "+
					"yayımlanan metin arama motorunda \"ince içerik\" sayılır.", n, haberMinParagraf))
		} else if n > haberMaxParagraf {
			out = append(out, fmt.Sprintf(
				"Gövde %d paragraf; %d üzeri bölünmeyi (alt başlık) düşünün.", n, haberMaxParagraf))
		}
		if k := utf8.RuneCountInString(strings.TrimSpace(govde)); k < haberMinKarakter {
			out = append(out, fmt.Sprintf(
				"Gövde %d karakter; en az %d önerilir. Kaynak özeti kopyalanmış olabilir.",
				k, haberMinKarakter))
		}
	}

	// HTML varlığı / etiketi: veritabanına düz metin olarak girerse okuyucu
	// "&nbsp;" görür (ölçüm: 3.628 içerik).
	tum := strings.Join([]string{r.Title, alanDegeri(r, "spot"), govde}, " ")
	if m := varlikRe.FindString(tum); m != "" {
		out = append(out, "Metinde HTML varlığı var ("+m+"); düz metin olmalı.")
	}
	if m := htmlEtiket.FindString(tum); m != "" {
		out = append(out, "Metinde HTML etiketi var ("+m+"); düz metin olmalı.")
	}
	if m := kalintiRe.FindString(tum); m != "" {
		out = append(out, "Metinde yönlendirme kalıntısı var (\""+m+"\"); kaldırın.")
	}

	if sizinti := kaynakSizintisi(tum, r.Sources); sizinti != "" {
		out = append(out, "Metinde kaynak site adı geçiyor (\""+sizinti+
			"\"); kendi haberimiz gibi yazılmalı.")
	}

	// Aynı cümlenin iki kez geçmesi: model uzun metinde kendini tekrar ediyor.
	if tekrar := tekrarCumle(paragraflar); tekrar != "" {
		out = append(out, "Aynı cümle birden çok kez geçiyor: \""+kisalt(tekrar, 60)+"\"")
	}

	// Başlık / spot bantları her habere uygulanır.
	out = append(out, uzunlukUyarisi("Başlık", r.Title, haberBaslikMin, haberBaslikMax)...)
	out = append(out, uzunlukUyarisi("Spot", alanDegeri(r, "spot"), haberSpotMin, haberSpotMax)...)

	if act.ID == "news" || act.ID == "research_news" {
		st := alanDegeri(r, "seo_title")
		if st == "" {
			out = append(out, "SEO başlığı üretilmedi; SERP'te ham başlık görünür.")
		} else {
			out = append(out, uzunlukUyarisi("SEO başlığı", st, SeoBaslikMin, SeoBaslikMax)...)
		}
		ma := alanDegeri(r, "meta_description")
		if ma == "" {
			out = append(out, "Meta açıklama üretilmedi.")
		} else {
			out = append(out, uzunlukUyarisi("Meta açıklama", ma, SeoAciklamaMin, SeoAciklamaMax)...)
		}
	}

	return out
}

// alanDegeri önce Fields'a, yoksa bölümlere bakar.
func alanDegeri(r *Result, ad string) string {
	if v := strings.TrimSpace(r.Fields[ad]); v != "" {
		return v
	}
	basliklar := map[string]string{
		"spot": "Spot", "intro": "Giriş", "body": "Gövde",
	}
	if b, ok := basliklar[ad]; ok {
		for _, s := range r.Sections {
			if s.Title == b {
				return strings.TrimSpace(s.Body)
			}
		}
	}
	return ""
}

// paragraflarOf gövdeyi paragraflara böler (boş satır veya tek satır sonu).
func paragraflarOf(govde string) []string {
	govde = strings.ReplaceAll(govde, "\r\n", "\n")
	var out []string
	for _, p := range strings.Split(govde, "\n") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func uzunlukUyarisi(ad, metin string, min, max int) []string {
	metin = strings.TrimSpace(metin)
	if metin == "" {
		return nil
	}
	n := utf8.RuneCountInString(metin)
	switch {
	case n < min:
		return []string{fmt.Sprintf("%s kısa (%d karakter); %d-%d arası önerilir.", ad, n, min, max)}
	case n > max:
		return []string{fmt.Sprintf("%s uzun (%d karakter); %d üzeri kesiliyor.", ad, n, max)}
	}
	return nil
}

func kaynakSizintisi(metin string, kaynaklar []Source) string {
	dusuk := strings.ToLower(metin)
	for _, s := range kaynaklar {
		host := hostOf(s.URL)
		if host == "" {
			continue
		}
		kok := strings.TrimPrefix(host, "www.")
		if strings.Contains(dusuk, kok) {
			return kok
		}
		if i := strings.Index(kok, "."); i > 0 {
			marka := kok[:i]
			if len(marka) >= 5 && strings.Contains(dusuk, marka) {
				return marka
			}
		}
	}
	return ""
}

func hostOf(u string) string {
	u = strings.TrimSpace(u)
	if i := strings.Index(u, "//"); i >= 0 {
		u = u[i+2:]
	}
	if i := strings.IndexAny(u, "/?#"); i >= 0 {
		u = u[:i]
	}
	return strings.ToLower(u)
}

// tekrarCumle paragraflar arasında birebir tekrar eden cümleyi bulur.
func tekrarCumle(paragraflar []string) string {
	gorulen := map[string]bool{}
	for _, p := range paragraflar {
		for _, c := range repo.CumlelereBol(p) {
			anahtar := strings.ToLower(strings.Join(strings.Fields(c), " "))
			if utf8.RuneCountInString(anahtar) < 25 {
				continue
			}
			if gorulen[anahtar] {
				return c
			}
			gorulen[anahtar] = true
		}
	}
	return ""
}

func kisalt(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
