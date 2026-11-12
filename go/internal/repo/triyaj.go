package repo

import "theblock/site/internal/scoring"

var contentOrderOptions = map[string]string{
	"guncel":  "c.updated_at DESC",
	"oncelik": oncelikSiralamasi(),
	// Tamlık sıralaması. Adı "skor" kalıyor çünkü kayıtlı görünümlerde
	// (saved views) bu anahtar kullanılıyor; kırmamak için korundu.
	"skor":   "c.ai_score DESC NULLS LAST, c.updated_at DESC",
	"eski":   "c.updated_at ASC",
	"baslik": "c.title ASC",
	"yayin":  "c.published_at DESC NULLS LAST, c.id DESC",
}

func oncelikSiralamasi() string {
	return "(" + scoring.OncelikSQL("c") + ") DESC, c.created_at DESC"
}

// contentOrderDefault beyaz listede olmayan her değerin karşılığı.
const contentOrderDefault = "c.updated_at DESC"

// contentOrderBy sıralama anahtarını güvenli SQL ifadesine çevirir.
func contentOrderBy(key string) string {
	if expr, ok := contentOrderOptions[key]; ok {
		return expr
	}
	return contentOrderDefault
}

// normalizeMinScore skor eşiğini 0..100 aralığına sıkıştırır. 0 ve altı
// "eşik yok" demektir (editör filtreyi kapatmıştır).
func normalizeMinScore(v int) int {
	if v <= 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}
