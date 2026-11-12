package httpapi

import (
	"encoding/json"
	"html/template"
	"strings"
)

// faqSoru tek bir soru-cevap çifti.
type faqSoru struct {
	Soru  string
	Cevap string
}

// faqBaslikKaliplari SSS bölümünü tanıyan başlıklar. Küçük harfe çevrilmiş
// başlıkta bunlardan biri geçiyorsa bölüm başlamış sayılır.
var faqBaslikKaliplari = []string{
	"sık sorulan soru",
	"sıkça sorulan soru",
	"merak edilenler",
}

// faqBolumMu başlık metni SSS bölümünü açıyor mu?
func faqBolumMu(baslik string) bool {
	b := strings.ToLower(strings.TrimSpace(baslik))
	for _, k := range faqBaslikKaliplari {
		if strings.Contains(b, k) {
			return true
		}
	}
	return false
}

func faqCikar(detail []byte) []faqSoru {
	var doc tipTapNode
	if json.Unmarshal(detail, &doc) != nil || doc.Type != "doc" {
		return nil
	}

	var out []faqSoru
	icerde := false   // SSS bölümünün içinde miyiz
	var acik *faqSoru // cevabı toplanan soru

	kapat := func() {
		if acik != nil && strings.TrimSpace(acik.Cevap) != "" {
			out = append(out, *acik)
		}
		acik = nil
	}

	for _, n := range doc.Content {
		switch n.Type {
		case "heading":
			seviye := intAttr(n.Attrs, "level")
			metin := strings.TrimSpace(tipTapNodeText(n))
			if seviye <= 2 {
				// Yeni üst başlık: açık soruyu kapat, bölümü yeniden belirle.
				kapat()
				icerde = faqBolumMu(metin)
				continue
			}
			if !icerde {
				continue
			}
			// H3 = yeni soru.
			kapat()
			if metin != "" {
				acik = &faqSoru{Soru: metin}
			}
		case "paragraph", "bulletList", "orderedList", "blockquote":
			if !icerde || acik == nil {
				continue
			}
			metin := strings.TrimSpace(tipTapNodeText(n))
			if metin == "" {
				continue
			}
			if acik.Cevap != "" {
				acik.Cevap += " "
			}
			acik.Cevap += metin
		}
	}
	kapat()

	// Tek soruluk bölüm için şema basmak anlamsız; Google da en az iki
	// soru bekler.
	if len(out) < 2 {
		return nil
	}
	return out
}

// faqJSONLD FAQPage bloğunu <script> ile sarar; soru yoksa boş döner.
func faqJSONLD(sorular []faqSoru) template.HTML {
	if len(sorular) == 0 {
		return ""
	}
	ogeler := make([]any, 0, len(sorular))
	for _, s := range sorular {
		ogeler = append(ogeler, map[string]any{
			"@type": "Question",
			"name":  s.Soru,
			"acceptedAnswer": map[string]any{
				"@type": "Answer",
				"text":  s.Cevap,
			},
		})
	}
	return jsonldScript(map[string]any{
		"@context":   "https://schema.org",
		"@type":      "FAQPage",
		"mainEntity": ogeler,
	})
}
