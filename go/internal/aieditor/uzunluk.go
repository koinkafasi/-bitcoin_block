package aieditor

import (
	"fmt"
	"strings"
)

// Konu bazlı uzunluk hedefleri.
//
// NEDEN: 13 Kasım 2026 ölçümünde AI hattının ürettiği haberin medyanı
// 156 kelimeydi; AYNI sitenin AI öncesi haber medyanı 486. Yani hat,
// kendi editoryal ölçümüzün üçte birini üretiyordu. Sebep prompt'taydı:
// "3 ile 8 arasında paragraf, her paragraf 2-4 cümle" diyordu ve model
// her seferinde alt sınıra yapışıyordu (3 × 2 × ~20 = ~120 kelime).
//
// Bantlar uydurma değil, SİTENİN KENDİ tarihsel medyanlarından
// türetildi (AI öncesi, yayında, özgün içerik):
//
//	haber          2592 içerik, medyan 486
//	analiz          470 içerik, medyan 502
//	basin-bulteni    23 içerik, medyan 640
//	arastirma        74 içerik, medyan 756
//	makale           54 içerik, medyan 856
//	sponsorlu        11 içerik, medyan 1282
//
// HABER İÇİN 1500 KELİME HEDEFİ YANLIŞ OLURDU. Google'ın haber için
// uzunluk ödülü yok; "yararlı içerik" değerlendirmesi dolgu metni
// cezalandırıyor. 1500+ rehber ve sütun (pillar) sayfalarının bandı.
// Haberde doğru hedef kendi ölçümüz: 450-700.
const (
	// UzunlukSapmaPayi doğrulayıcının uyarı vermeden önce tanıdığı pay.
	// Hedef bir bant, çit değil: 700 kelime hedefinde 690 kelimelik
	// metin için editörü uyarmak gürültü üretir.
	UzunlukSapmaPayi = 0.15

	// AltBaslikEsigi bu kelime sayısının üzerinde alt başlık beklenir.
	// Ölçüm: son 30 günün 359 içeriğinden YALNIZ 4'ünde alt başlık var.
	// 156 kelimede alt başlık gereksiz, ama 800+ kelimede metin duvarı
	// hem okunmuyor hem öne çıkan snippet'e aday olmuyor.
	AltBaslikEsigi = 800
)

// UzunlukHedefi bir içerik türünün gövde hedefidir.
type UzunlukHedefi struct {
	// Ad hedefin okunur adı; prompt'a ve uyarı metnine girer.
	Ad string
	// MinKelime / MaxKelime gövde hedefi (kelime).
	MinKelime int
	MaxKelime int
	// MinParagraf / MaxParagraf paragraf bandı.
	MinParagraf int
	MaxParagraf int
}

// AltBaslikGerekli hedef uzunlukta alt başlık beklenip beklenmediği.
func (h UzunlukHedefi) AltBaslikGerekli() bool {
	return h.MinKelime >= AltBaslikEsigi
}

// AltSinir / UstSinir doğrulayıcının kullandığı paylı sınırlar.
func (h UzunlukHedefi) AltSinir() int {
	return int(float64(h.MinKelime) * (1 - UzunlukSapmaPayi))
}

func (h UzunlukHedefi) UstSinir() int {
	return int(float64(h.MaxKelime) * (1 + UzunlukSapmaPayi))
}

// Paragraf bantları kelime bandından TÜRETİLDİ, ayrı seçilmedi.
//
// Prompt her paragraftan "2-4 cümle" istiyor; Türkçe'de bu ~35-70
// kelime. Bantlar bu orana göre hesaplandı, yoksa modele çelişkili iki
// hedef verilirdi (ör. "700 kelime yaz" + "5 paragrafı geçme" =
// paragraf başına 140 kelime, yani metin duvarı).
//
// uzunlukBantlari tür anahtarına göre hedefler.
var uzunlukBantlari = map[string]UzunlukHedefi{
	"haber":         {Ad: "haber", MinKelime: 450, MaxKelime: 700, MinParagraf: 7, MaxParagraf: 16},
	"analiz":        {Ad: "analiz", MinKelime: 800, MaxKelime: 1200, MinParagraf: 12, MaxParagraf: 24},
	"arastirma":     {Ad: "araştırma", MinKelime: 1200, MaxKelime: 1800, MinParagraf: 17, MaxParagraf: 34},
	"makale":        {Ad: "makale", MinKelime: 1200, MaxKelime: 2000, MinParagraf: 17, MaxParagraf: 36},
	"basin-bulteni": {Ad: "basın bülteni", MinKelime: 350, MaxKelime: 600, MinParagraf: 5, MaxParagraf: 14},
	"sponsorlu":     {Ad: "sponsorlu içerik", MinKelime: 800, MaxKelime: 1500, MinParagraf: 13, MaxParagraf: 28},
}

// kategoriBantlari KATEGORİ bazlı geçersiz kılma.
//
// NEDEN GEREKLİ: yazım hattı her taslağı type_key="haber" olarak açıyor
// (ölçüm: son 30 günün 325 AI içeriğinin 325'i "haber"). Bir taslak
// "Kripto 101" kategorisine düşmüşse bu bir rehberdir; type_key bunu
// söylemez, kategori söyler. Bu yüzden kategori tür anahtarını EZER.
var kategoriBantlari = map[string]UzunlukHedefi{
	// Rehber ve ansiklopedi: 1500+ bandının gerçek yeri. Ölçümde
	// "Kripto 101" içeriklerinin medyanı 1440 kelime ve bunlar elle
	// yazılmış, doğru uzunluktaki tek grup.
	"learn":      {Ad: "rehber", MinKelime: 1500, MaxKelime: 2500, MinParagraf: 22, MaxParagraf: 45},
	"kriptopedi": {Ad: "ansiklopedi maddesi", MinKelime: 1200, MaxKelime: 2200, MinParagraf: 19, MaxParagraf: 40},
	"inceleme":   {Ad: "inceleme", MinKelime: 1200, MaxKelime: 2000, MinParagraf: 17, MaxParagraf: 36},

	// Analiz ailesi.
	"analiz":        {Ad: "analiz", MinKelime: 800, MaxKelime: 1200, MinParagraf: 12, MaxParagraf: 24},
	"news-analysis": {Ad: "haber analizi", MinKelime: 800, MaxKelime: 1200, MinParagraf: 12, MaxParagraf: 24},
	"fiyat-tahmini": {Ad: "fiyat analizi", MinKelime: 800, MaxKelime: 1400, MinParagraf: 12, MaxParagraf: 28},

	// Araştırma ailesi.
	"arastirma": {Ad: "araştırma", MinKelime: 1200, MaxKelime: 1800, MinParagraf: 17, MaxParagraf: 34},
	"research":  {Ad: "araştırma", MinKelime: 1200, MaxKelime: 1800, MinParagraf: 17, MaxParagraf: 34},

	// Görüş yazısı haberden uzun, analizden kısa. Ölçümde son 30 günün
	// "opinion" içeriklerinin medyanı 462 — tek doğru uzunlukta çıkan
	// AI grubu, çünkü köşe yazısı doğal olarak uzuyor.
	"opinion": {Ad: "görüş yazısı", MinKelime: 600, MaxKelime: 1000, MinParagraf: 9, MaxParagraf: 20},

	// Bülten aileleri.
	"basin-bulteni": {Ad: "basın bülteni", MinKelime: 350, MaxKelime: 600, MinParagraf: 5, MaxParagraf: 14},
	"press-release": {Ad: "basın bülteni", MinKelime: 350, MaxKelime: 600, MinParagraf: 5, MaxParagraf: 14},

	"sponsorlu":         {Ad: "sponsorlu içerik", MinKelime: 800, MaxKelime: 1500, MinParagraf: 13, MaxParagraf: 28},
	"sponsored-content": {Ad: "sponsorlu içerik", MinKelime: 800, MaxKelime: 1500, MinParagraf: 13, MaxParagraf: 28},
}

// uzunlukVarsayilan tanınmayan tür/kategori için haber bandı.
var uzunlukVarsayilan = uzunlukBantlari["haber"]

// UzunlukHedefiIcin tür anahtarı ve kategori slug'ından hedefi verir.
// Kategori tür anahtarını ezer (bkz. kategoriBantlari).
func UzunlukHedefiIcin(typeKey, kategoriSlug string) UzunlukHedefi {
	if h, ok := kategoriBantlari[normalizeAnahtar(kategoriSlug)]; ok {
		return h
	}
	if h, ok := uzunlukBantlari[normalizeAnahtar(typeKey)]; ok {
		return h
	}
	return uzunlukVarsayilan
}

func normalizeAnahtar(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// PromptSatiri modele verilecek hedef metnidir.
//
// Hedef, bağlam JSON'una gömülmekle YETMEZ: ölçümde model JSON içindeki
// sayıyı yok sayıp alt sınıra yapışıyordu. Bu yüzden ayrı ve buyurgan
// bir blok olarak basılıyor.
func (h UzunlukHedefi) PromptSatiri() string {
	var b strings.Builder
	fmt.Fprintf(&b, "HEDEF UZUNLUK (%s): gövde %d-%d KELİME, %d-%d paragraf.\n",
		h.Ad, h.MinKelime, h.MaxKelime, h.MinParagraf, h.MaxParagraf)
	fmt.Fprintf(&b, "- %d kelimenin altında kalma: eksik metin arama motorunda \"ince içerik\" sayılır.\n",
		h.MinKelime)
	b.WriteString("- Uzunluğu DOLGU ile tutturma. Kaynakta olmayan olgu ekleyerek, " +
		"aynı bilgiyi başka cümleyle yineleyerek veya genel geçer kripto " +
		"anlatımı yazarak uzatma. Kaynak hedefi doldurmaya yetmiyorsa " +
		"kısa kalsın — uydurmak, kısa kalmaktan kötüdür.\n")
	b.WriteString("- Uzunluğu şunlarla kazan: olayın arka planı, rakamın ne anlama " +
		"geldiği, taraflarca söylenenler, piyasaya/kullanıcıya etkisi, " +
		"bundan sonra ne olacağı.\n")
	if h.AltBaslikGerekli() {
		b.WriteString("- Bu uzunlukta ALT BAŞLIK zorunlu: metni 3-6 alt başlıkla böl. " +
			"Başlıksız metin duvarı okunmuyor ve öne çıkan snippet'e aday olmuyor.\n")
	}
	return b.String()
}

// KelimeSay metindeki kelime sayısını verir (boşlukla ayrılmış öbekler).
func KelimeSay(metin string) int {
	return len(strings.Fields(metin))
}

// uzunlukUyarilari gövdeyi hedefe göre denetler.
//
// Ayrı bir birim olarak duruyor çünkü yapiUyarilari türden bağımsız
// taban denetimleri yapıyor (3 paragraf, 700 karakter, HTML kalıntısı);
// uzunluk hedefi türe göre değişiyor ve kendi başına test edilebilmesi
// gerekiyor.
//
// Döndürülenler UYARI, engel değil: editör kısa haberi bilerek
// yayımlayabilir. Önemli olan hattın kısa ürettiğini FARK ETMEK —
// eski eşik (700 karakter ≈ 100 kelime) 156 kelimelik metni "yeterli"
// buluyordu, yani hiç tetiklenmiyordu.
func uzunlukUyarilari(govde string, hedef UzunlukHedefi, altBaslikVar bool) []string {
	govde = strings.TrimSpace(govde)
	if govde == "" {
		return nil
	}
	var out []string
	n := KelimeSay(govde)

	switch {
	case n < hedef.AltSinir():
		out = append(out, fmt.Sprintf(
			"Gövde %d kelime; %s için %d-%d kelime hedefleniyor. Eksik metin "+
				"arama motorunda \"ince içerik\" sayılır — olayın arka planı, "+
				"rakamın anlamı ve etkisi eklenebilir.",
			n, hedef.Ad, hedef.MinKelime, hedef.MaxKelime))
	case n > hedef.UstSinir():
		out = append(out, fmt.Sprintf(
			"Gövde %d kelime; %s için üst hedef %d. Dolgu veya tekrar olup "+
				"olmadığını kontrol edin.",
			n, hedef.Ad, hedef.MaxKelime))
	}

	// Alt başlık yalnız hedef uzunsa ve metin gerçekten uzunsa istenir:
	// hedefi tutturamamış 200 kelimelik metne "alt başlık ekle" demek
	// yanlış sırayla uyarmak olurdu.
	if hedef.AltBaslikGerekli() && !altBaslikVar && n >= AltBaslikEsigi {
		out = append(out, fmt.Sprintf(
			"Gövde %d kelime ve alt başlık yok. Bu uzunlukta metin duvarı "+
				"okunmuyor ve öne çıkan snippet'e aday olmuyor; 3-6 alt başlıkla bölün.", n))
	}
	return out
}

// altBaslikVar sonuçta alt başlık üretilmiş mi.
//
// normalize.go alt başlıkları "Alt başlıklar" adlı bölüme yazıyor
// (news, research_news ve audio_draft dallarının üçü de). Gövdenin
// içinde markdown başlığı aramıyoruz: prompt düz metin istiyor ve
// "## " arayan bir denetim yanlış pozitif üretirdi.
func altBaslikVar(r *Result) bool {
	for _, s := range r.Sections {
		if s.Title == "Alt başlıklar" && strings.TrimSpace(s.Body) != "" {
			return true
		}
	}
	return false
}

// uzunlukHedefiUygulanir yalnız GÖVDE yazan eylemler için true.
//
// Başlık önerisi, SEO alanı, özet ve doğrulama eylemlerine uzunluk
// hedefi basmak anlamsız olurdu: onların çıktısı gövde değil ve
// prompt'u gereksiz uzatıp modeli şaşırtırdı.
func uzunlukHedefiUygulanir(actID string) bool {
	switch actID {
	case "news", "research_news", "rewrite_angle", "audio_draft":
		return true
	default:
		return false
	}
}
