package repo

import "theblock/site/internal/scoring"

// Triyaj: taslak havuzunu yönetilebilir kılan sıralama ve eşik.
//
// Sorun (28 Ekim 2026 ölçümü): 39.698 içeriğin 39.206'sı taslak; bot günde
// ~5.600 yeni taslak üretiyor. Admin içerik listesi `ORDER BY updated_at
// DESC` ile sabitti, yani editör havuzun en üstünde her zaman "en son bot
// kaydı"nı görüyordu. Hangi taslağın yayına değdiğini anlamanın yolu yoktu.
//
// Çözüm bilinçli olarak **ayrı bir ekran değil**: mevcut içerik listesi
// sıralama ve skor eşiğiyle genişletildi. İkinci bir içerik listesi, zaten
// şikâyet konusu olan "aynı işi yapan iki alan" sorununu büyütürdü
// (bkz. specs/10-admin-mimari-ve-arayuz.md §3.1).

// contentOrderOptions sıralama beyaz listesi.
//
// Bu ifadeler SQL'e **doğrudan** girer (parametre olamazlar). Bu yüzden
// anahtar dışında hiçbir kullanıcı girdisi buraya ulaşmaz; bilinmeyen değer
// varsayılana düşer. İfadelerin tamamı sabittir.
var contentOrderOptions = map[string]string{
	"guncel": "c.updated_at DESC",
	// ÖNCELİK — triyajın asıl sıralaması. Dört eksenin ağırlıklı birleşimi:
	// trend uyumu, tazelik, kaynak güveni, tamlık (internal/scoring).
	// İfade Go sabitlerinden üretilir ki panelde görünen puanla sıra ayrışmasın.
	//
	// Neden ai_score değil: ai_score yalnız ÜSTVERİ EKSİKSİZLİĞİNİ ölçüyor ve
	// 34.884 taslağın %87,9'unu tek bir 60-79 bandına koyuyordu — ona göre
	// sıralamak kuyruğu rastgele dizmekti. Önceliğe geçince en kalabalık bant
	// %26,2'ye indi ve 10 bandın tamamı doldu (üretim verisinde ölçüldü).
	"oncelik": oncelikSiralamasi(),
	// Tamlık sıralaması. Adı "skor" kalıyor çünkü kayıtlı görünümlerde
	// (saved views) bu anahtar kullanılıyor; kırmamak için korundu.
	"skor":   "c.ai_score DESC NULLS LAST, c.updated_at DESC",
	"eski":   "c.updated_at ASC",
	"baslik": "c.title ASC",
}

// oncelikSiralamasi öncelik ifadesini üretir. Eşitlik bozucu olarak
// created_at kullanılır: aynı öncelikteki iki taslaktan yeni olan önce
// gelir (tazelik zaten ifadenin içinde, bu yalnız kararlılık sağlar).
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
