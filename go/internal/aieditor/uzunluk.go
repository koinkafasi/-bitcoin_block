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
// ÖLÇÜM DÜZELTMESİ (13 Kasım 2026, aynı gün): ilk bantlar İKİ KAT
// YÜKSEK sayılara dayanıyordu. Kullanılan sorgu
// `jsonb_path_query_array(detail, '$.**.text')` idi; jsonpath LAX
// modda diziyi otomatik açtığı için her metin düğümü İKİ KEZ
// dönüyordu. `strict $.**.text` ile doğrulandı: Ghost'tan aktarılan
// bir yazıda lax 5478, strict 2739 — strict değer kaynağın kendi
// kelime sayısına birebir eşit.
//
// SİTENİN KENDİ MEDYANLARI (AI öncesi, yayında, özgün — DOĞRU sayım):
//
//	haber          2592 içerik, medyan 243   (yanlış ölçümde 486)
//	analiz          470 içerik, medyan 251
//	basin-bulteni    23 içerik, medyan 320
//	arastirma        74 içerik, medyan 378
//	makale           54 içerik, medyan 428
//	sponsorlu        11 içerik, medyan 641
//	kategori learn   59 içerik, medyan 440
//
// AI sonrası haber medyanı: 78 kelime (yanlış ölçümde 156).
//
// BANTLAR NEDEN KENDİ MEDYANIMIZA EŞİT DEĞİL: hedef kendi geçmişimizi
// tekrarlamak değil, arama sonuçlarında öne çıkmak. 243 kelime sektör
// ölçüsünde de kısa. Ama 78'den 450-700'e atlamak 6-9 KAT demekti ve
// kaynak RSS parçası bunu taşımıyor: üretimde içerik 10764 tam bu
// baskı altında 4 paragrafı 12 kez tekrarladı. Bantlar bu yüzden
// indirildi — ulaşılabilir ve kaynağın taşıyabileceği kadar.
//
// HABER İÇİN 1500 KELİME HEDEFİ YİNE YANLIŞ OLURDU. Google'ın haber
// için uzunluk ödülü yok; "yararlı içerik" değerlendirmesi dolguyu
// cezalandırıyor. 1500+ rehber ve sütun (pillar) sayfalarının bandı.
const (
	// UzunlukSapmaPayi doğrulayıcının uyarı vermeden önce tanıdığı pay.
	// Hedef bir bant, çit değil: 700 kelime hedefinde 690 kelimelik
	// metin için editörü uyarmak gürültü üretir.
	UzunlukSapmaPayi = 0.15

	// AltBaslikEsigi bu kelime sayısının üzerinde alt başlık beklenir.
	// Ölçüm: son 30 günün 359 içeriğinden YALNIZ 4'ünde alt başlık var.
	// 78 kelimede alt başlık gereksiz, ama 650+ kelimede metin duvarı
	// hem okunmuyor hem öne çıkan snippet'e aday olmuyor.
	//
	// Eşik 800'den 650'ye indi: bantlar aşağı çekilince analiz ailesi
	// (650-1000) eşiğin altında kalıyor ve başlıksız kalıyordu —
	// oysa alt başlığa en çok o uzunluk ihtiyaç duyuyor.
	AltBaslikEsigi = 650
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
	"haber":         {Ad: "haber", MinKelime: 350, MaxKelime: 550, MinParagraf: 6, MaxParagraf: 14},
	"analiz":        {Ad: "analiz", MinKelime: 650, MaxKelime: 1000, MinParagraf: 10, MaxParagraf: 24},
	"arastirma":     {Ad: "araştırma", MinKelime: 900, MaxKelime: 1400, MinParagraf: 13, MaxParagraf: 32},
	"makale":        {Ad: "makale", MinKelime: 900, MaxKelime: 1500, MinParagraf: 13, MaxParagraf: 34},
	"basin-bulteni": {Ad: "basın bülteni", MinKelime: 300, MaxKelime: 500, MinParagraf: 5, MaxParagraf: 12},
	"sponsorlu":     {Ad: "sponsorlu içerik", MinKelime: 700, MaxKelime: 1200, MinParagraf: 11, MaxParagraf: 26},
}

// kategoriBantlari KATEGORİ bazlı geçersiz kılma.
//
// NEDEN GEREKLİ: yazım hattı her taslağı type_key="haber" olarak açıyor
// (ölçüm: son 30 günün 325 AI içeriğinin 325'i "haber"). Bir taslak
// "Kripto 101" kategorisine düşmüşse bu bir rehberdir; type_key bunu
// söylemez, kategori söyler. Bu yüzden kategori tür anahtarını EZER.
var kategoriBantlari = map[string]UzunlukHedefi{
	// Rehber ve ansiklopedi: uzun bandın gerçek yeri. Kategori ölçümü
	// (DOĞRU sayım): learn medyanı 440 kelime / 59 içerik — sitenin en
	// uzun yazı grubu. Hedef bunun üstünde: rehber sayfası tek
	// oturumda soruyu kapatmalı.
	"learn":      {Ad: "rehber", MinKelime: 1200, MaxKelime: 2000, MinParagraf: 18, MaxParagraf: 40},
	"kriptopedi": {Ad: "ansiklopedi maddesi", MinKelime: 1000, MaxKelime: 1800, MinParagraf: 16, MaxParagraf: 34},
	"inceleme":   {Ad: "inceleme", MinKelime: 900, MaxKelime: 1500, MinParagraf: 13, MaxParagraf: 34},

	// Analiz ailesi.
	"analiz":        {Ad: "analiz", MinKelime: 650, MaxKelime: 1000, MinParagraf: 10, MaxParagraf: 24},
	"news-analysis": {Ad: "haber analizi", MinKelime: 650, MaxKelime: 1000, MinParagraf: 10, MaxParagraf: 24},
	"fiyat-tahmini": {Ad: "fiyat analizi", MinKelime: 650, MaxKelime: 1100, MinParagraf: 10, MaxParagraf: 26},

	// Araştırma ailesi.
	"arastirma": {Ad: "araştırma", MinKelime: 900, MaxKelime: 1400, MinParagraf: 13, MaxParagraf: 32},
	"research":  {Ad: "araştırma", MinKelime: 900, MaxKelime: 1400, MinParagraf: 13, MaxParagraf: 32},

	// Görüş yazısı haberden uzun, analizden kısa. Kategori medyanı 231
	// (14 içerik); köşe yazısı doğal olarak uzadığı için hedef üstte.
	"opinion": {Ad: "görüş yazısı", MinKelime: 500, MaxKelime: 850, MinParagraf: 8, MaxParagraf: 20},

	// Bülten aileleri. Kategori medyanı 320 — hedef ona yakın tutuldu:
	// bülten kaynağı zaten kısa, uzatmak dolgu demek.
	"basin-bulteni": {Ad: "basın bülteni", MinKelime: 300, MaxKelime: 500, MinParagraf: 5, MaxParagraf: 12},
	"press-release": {Ad: "basın bülteni", MinKelime: 300, MaxKelime: 500, MinParagraf: 5, MaxParagraf: 12},

	// Kategori medyanı 641 — sitenin en uzun grubu, hedef hafif üstte.
	"sponsorlu":         {Ad: "sponsorlu içerik", MinKelime: 700, MaxKelime: 1200, MinParagraf: 11, MaxParagraf: 26},
	"sponsored-content": {Ad: "sponsorlu içerik", MinKelime: 700, MaxKelime: 1200, MinParagraf: 11, MaxParagraf: 26},
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
