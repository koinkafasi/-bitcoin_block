package httpapi

import "strings"

// categoryIntro üst kategoriler için kısa, benzersiz SEO giriş metni.
// Alt kategorilerde boş döner; başlık ve kırıntı yolu yeterlidir.
var categoryIntros = map[string]string{
	"gundem":        "Türkiye ve dünyadan güncel gelişmeler; asayiş, yerel haberler, analizler ve toplum gündemi.",
	"siyaset":       "Siyaset, meclis, seçim, yerel yönetim ve politika haberleri; güncel gelişmeler ve analizler.",
	"ekonomi":       "Ekonomi, piyasalar, borsa, döviz, altın, enflasyon, bankacılık ve iş dünyasından son dakika haberleri.",
	"dunya":         "Afrika, Amerika, Asya, Avrupa ve Orta Doğu’dan dünya haberleri, analizler ve bölgesel gelişmeler.",
	"spor":          "Futbol, basketbol ve diğer spor dallarından maç sonuçları, transfer haberleri ve güncel gelişmeler.",
	"yasam":         "Sağlık, yaşam, seyahat, kadın, gençlik ve kişisel gelişim üzerine güncel haberler ve öneriler.",
	"teknoloji":     "Teknoloji, yapay zekâ, yazılım, donanım, mobil, uzay ve oyun dünyasından son gelişmeler.",
	"egitim-kultur": "Eğitim, kültür, sanat, edebiyat, kitap, sinema, tarih ve mimari üzerine haberler ve analizler.",
	"guncel":        "Türkiye ve dünyadan güncel gelişmeler; asayiş, yerel haberler, analizler ve toplum gündemi.",
}

func categoryIntro(slug string) string {
	return strings.TrimSpace(categoryIntros[slug])
}
