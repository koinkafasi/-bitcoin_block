package aieditor

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var sentenceRe = regexp.MustCompile(`[.!?]+`)

func parseJSONObject(raw string) (map[string]any, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return nil, errors.New("boş yanıt")
	}
	if strings.HasPrefix(s, "```") {
		if i := strings.IndexByte(s, '\n'); i >= 0 {
			s = s[i+1:]
		}
		s = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(s), "```"))
	}
	start := strings.IndexByte(s, '{')
	end := strings.LastIndexByte(s, '}')
	if start < 0 || end <= start {
		return nil, errors.New("JSON nesnesi bulunamadı")
	}
	s = s[start : end+1]
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return nil, err
	}
	return m, nil
}

func normalizeResult(act Action, m map[string]any, sources []Source) (Result, error) {
	r := Result{Action: act.ID, Sources: sources}
	add := func(title, body string) {
		body = strings.TrimSpace(body)
		if body != "" {
			r.Sections = append(r.Sections, Section{Title: title, Body: body})
		}
	}
	field := func(k, v string) {
		v = strings.TrimSpace(v)
		if v == "" {
			return
		}
		if r.Fields == nil {
			r.Fields = map[string]string{}
		}
		r.Fields[k] = v
	}
	warn := func(s string) {
		s = strings.TrimSpace(s)
		if s != "" {
			r.Warnings = append(r.Warnings, s)
		}
	}

	switch act.ID {
	case "news", "research_news":
		r.Title = str(m, "title")
		spot, intro, body := str(m, "spot"), str(m, "intro"), str(m, "body")
		// Faz 193c — model artık `paragraphs` dizisi döndürüyor. Gövdeye
		// boş satır bırakılarak birleştirilir: düzenleyici textarea'ya iki
		// satır sonuyla eklediği için bloklar gerçekten ayrı paragraf oluyor
		// (eskiden tek blok kalıyordu; ölçüm: %99,6 tek paragraf).
		if ps := strSlice(m, "paragraphs"); len(ps) > 0 {
			body = strings.Join(ps, "\n\n")
		}
		if r.Title == "" || body == "" {
			return r, errors.New("haber taslağı eksik: title ve paragraphs/body zorunlu")
		}
		add("Spot", spot)
		add("Giriş", intro)
		add("Gövde", body)
		if subs := strSlice(m, "subheadings"); len(subs) > 0 {
			add("Alt başlıklar", bullets(subs))
		}
		if claims := strSlice(m, "claims_to_verify"); len(claims) > 0 {
			add("Doğrulanması gereken iddialar", bullets(claims))
		}
		if act.ID == "research_news" && len(strSlice(m, "source_refs")) == 0 {
			warn("Model kaynak numarası kullanmadı; kaynakları editörün kontrol etmesi gerekir.")
		}
		field("title", r.Title)
		field("spot", spot)
		field("intro", intro)
		field("body", body)
		// SEO alanları da aynı yanıtta geliyor: editör ayrıca "SEO" işi
		// çalıştırmak zorunda kalmasın. `SeoDogrula` ile kırpılır.
		if st := str(m, "seo_title"); st != "" {
			field("seo_title", seoTemizle(st))
		}
		if ma := str(m, "meta_description"); ma != "" {
			field("meta_description", seoTemizle(ma))
		}

	case "audio_draft":
		r.Title = str(m, "title")
		spot, body := str(m, "spot"), str(m, "body")
		if r.Title == "" || body == "" {
			return r, errors.New("ses taslağı eksik: title ve body zorunlu")
		}
		add("Spot", spot)
		add("Gövde", body)
		if subs := strSlice(m, "subheadings"); len(subs) > 0 {
			add("Alt başlıklar", bullets(subs))
		}
		if quotes := strSlice(m, "quotes"); len(quotes) > 0 {
			r.Quotes = quotes
			add("Alıntılar", bullets(quotes))
		}
		if claims := strSlice(m, "claims_to_verify"); len(claims) > 0 {
			add("Doğrulanması gereken iddialar", bullets(claims))
		}
		field("title", r.Title)
		field("spot", spot)
		field("body", body)

	case "headlines":
		items := objSlice(m, "headlines")
		if len(items) != 5 {
			return r, fmt.Errorf("tam 5 başlık bekleniyordu, %d geldi", len(items))
		}
		for _, it := range items {
			text := str(it, "text")
			if text == "" {
				return r, errors.New("boş başlık üretildi")
			}
			note := strings.TrimSpace(str(it, "angle"))
			if fit := strings.TrimSpace(str(it, "source_fit")); fit != "" {
				if note != "" {
					note += " · "
				}
				note += fit
			}
			r.Alternatives = append(r.Alternatives, Alternative{Text: text, Note: note})
		}

	case "intro":
		spot, intro := str(m, "spot"), str(m, "intro")
		if spot == "" && intro == "" {
			return r, errors.New("spot veya giriş paragrafı üretilemedi")
		}
		add("Spot", spot)
		add("Giriş paragrafı", intro)
		field("spot", spot)
		field("intro", intro)

	case "rewrite_angle":
		angle, title, body := str(m, "angle"), str(m, "title"), str(m, "body")
		if body == "" {
			return r, errors.New("yeniden yazım gövdesi boş")
		}
		r.Title = title
		add("Açı", angle)
		add("Yeni metin", body)
		field("title", title)
		field("body", body)

	case "summary":
		summary := str(m, "summary")
		if summary == "" {
			return r, errors.New("özet boş")
		}
		n := len(sentenceRe.FindAllString(summary, -1))
		if n < 2 || n > 3 {
			warn(fmt.Sprintf("Özet %d cümle görünüyor; spec 2–3 cümle istiyor.", n))
		}
		add("Özet", summary)
		field("body", summary)

	case "social":
		platforms := []struct{ key, label string }{
			{"x", "X"}, {"instagram", "Instagram"}, {"facebook", "Facebook"},
			{"linkedin", "LinkedIn"}, {"video_caption", "Kısa video açıklaması"},
		}
		count := 0
		for _, p := range platforms {
			if v := str(m, p.key); v != "" {
				add(p.label, v)
				count++
			}
		}
		if count == 0 {
			return r, errors.New("sosyal medya metni üretilemedi")
		}
		warn("Platform karakter sınırları yayınlama entegrasyonunda ayrıca doğrulanmalıdır.")

	case "seo":
		primary := str(m, "primary")
		if primary == "" {
			return r, errors.New("ana anahtar kelime boş")
		}
		add("Ana anahtar kelime", primary)
		add("İkincil anahtar kelimeler", bullets(strSlice(m, "secondary")))
		add("İlgili varlık ve kavramlar", bullets(strSlice(m, "entities")))
		add("Arama niyeti", str(m, "intent"))
		seoTitle, meta, slug := str(m, "seo_title"), str(m, "meta_description"), str(m, "slug")
		add("SEO başlığı", seoTitle)
		add("Meta açıklama", meta)
		add("URL slug önerisi", slug)
		field("seo_title", seoTitle)
		field("meta_description", meta)
		field("slug", slug)
		warn("Arama hacmi/rekabet metriği üretilmedi; gerçek SEO sağlayıcısı olmadan bu değerler doğrulanmış sayılmaz.")

	case "questions":
		items := objSlice(m, "questions")
		if len(items) < 3 {
			return r, fmt.Errorf("en az 3 soru bekleniyordu, %d geldi", len(items))
		}
		for i, it := range items {
			q, a, miss := str(it, "question"), str(it, "answer"), str(it, "missing_info")
			if q == "" {
				continue
			}
			body := a
			if body == "" {
				body = "Cevap metinde yok."
			}
			if miss != "" {
				body += "\nEksik bilgi: " + miss
			}
			add(fmt.Sprintf("Soru %d: %s", i+1, q), body)
		}
		if len(r.Sections) == 0 {
			return r, errors.New("soru-cevap üretilemedi")
		}

	case "background":
		add("Kronoloji", bullets(strSlice(m, "timeline")))
		add("İlgili kişi ve kurumlar", bullets(strSlice(m, "actors")))
		add("Önceki gelişmeler", bullets(strSlice(m, "prior_events")))
		add("Önemli tarih ve rakamlar", bullets(strSlice(m, "figures")))
		add("Çelişen bilgiler", bullets(strSlice(m, "conflicts")))
		add("Ek araştırma soruları", bullets(strSlice(m, "open_questions")))
		if len(r.Sections) == 0 {
			return r, errors.New("arka plan dosyası boş")
		}
		if len(strSlice(m, "source_refs")) == 0 {
			warn("Model kaynak numarası kullanmadı; bulguları kaynaklarla eşleştirin.")
		}

	case "verify":
		items := objSlice(m, "claims")
		if len(items) == 0 {
			return r, errors.New("doğrulanabilir iddia bulunamadı")
		}
		validRefs := make(map[string]bool, len(sources))
		for _, src := range sources {
			validRefs[strings.ToUpper(src.Ref)] = true
		}
		for _, it := range items {
			claim := str(it, "claim")
			if claim == "" {
				continue
			}
			refs := normalizeRefs(strSlice(it, "source_refs"), validRefs)
			status := statusTR(str(it, "status"))
			if (status == "Destekleniyor" || status == "Kısmen destekleniyor" || status == "Çelişen kanıt var") && len(refs) == 0 {
				status = "Yeterli kaynak bulunamadı"
				warn("Bir iddia için geçerli kaynak numarası yoktu; durum doğrulanamadı olarak işaretlendi.")
			}
			r.Claims = append(r.Claims, Claim{
				Text:        claim,
				Excerpt:     str(it, "excerpt"),
				Status:      status,
				Evidence:    str(it, "evidence"),
				Explanation: str(it, "explanation"),
				Action:      str(it, "recommended_action"),
				Sources:     refs,
			})
		}
		if len(r.Claims) == 0 {
			return r, errors.New("iddia listesi boş")
		}
		warn("“Doğrulanamadı” sonucu iddianın yanlış olduğu anlamına gelmez.")

	default:
		return r, fmt.Errorf("desteklenmeyen eylem: %s", act.ID)
	}
	return r, nil
}

// normalizeRefs yalnızca gerçekten var olan [S1] kaynaklarına izin verir.
// Modelin uydurduğu veya aralık dışı kaynak numaraları sessizce kabul edilmez.
func normalizeRefs(refs []string, valid map[string]bool) []string {
	out := make([]string, 0, len(refs))
	seen := map[string]bool{}
	for _, r := range refs {
		r = strings.ToUpper(strings.Trim(strings.TrimSpace(r), "[]"))
		if r == "" || !valid[r] || seen[r] {
			continue
		}
		seen[r] = true
		out = append(out, r)
	}
	return out
}

func statusTR(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "supported", "destekleniyor":
		return "Destekleniyor"
	case "partial", "kismen", "kısmen", "kismen_destekleniyor":
		return "Kısmen destekleniyor"
	case "contradicted", "celisen", "çelişen":
		return "Çelişen kanıt var"
	case "unverified", "dogrulanamadi", "doğrulanamadı":
		return "Doğrulanamadı"
	default:
		return "Yeterli kaynak bulunamadı"
	}
}

func str(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case json.Number:
		return t.String()
	case float64:
		return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.4f", t), "0"), ".")
	default:
		return strings.TrimSpace(fmt.Sprint(t))
	}
}

func strSlice(m map[string]any, key string) []string {
	v, ok := m[key]
	if !ok || v == nil {
		return nil
	}
	switch t := v.(type) {
	case []any:
		out := make([]string, 0, len(t))
		for _, x := range t {
			if s, ok := x.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, strings.TrimSpace(s))
			}
		}
		return out
	case []string:
		return t
	case string:
		if strings.TrimSpace(t) == "" {
			return nil
		}
		return []string{strings.TrimSpace(t)}
	default:
		return nil
	}
}

func objSlice(m map[string]any, key string) []map[string]any {
	v, ok := m[key]
	if !ok {
		return nil
	}
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(arr))
	for _, x := range arr {
		if obj, ok := x.(map[string]any); ok {
			out = append(out, obj)
		}
	}
	return out
}

func bullets(items []string) string {
	if len(items) == 0 {
		return ""
	}
	var b strings.Builder
	for _, s := range items {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		b.WriteString("• ")
		b.WriteString(s)
		b.WriteByte('\n')
	}
	return strings.TrimSpace(b.String())
}
