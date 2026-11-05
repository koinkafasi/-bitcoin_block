package scoring

import "fmt"

// Tazelik ve öncelik SQL tarafında hesaplanır; neden:
//
// Tazelik zamanın fonksiyonudur. Sütuna yazılsaydı her saat bayatlardı ve
// 34 bin satırı sürekli yeniden yazan bir iş gerekirdi — üstelik iki koşum
// arasında sıralama yanlış kalırdı. created_at zaten tabloda olduğu için
// tazeliği sorgu anında hesaplamak hem her zaman doğru hem bedava.
//
// Sütunda yalnız YAVAŞ DEĞİŞEN eksenler durur:
//
//	score_trend — trend eşleşmesi değiştiğinde yazılır
//	score_guven — kaynağın feeds.trust değeri değiştiğinde yazılır
//	ai_score    — tamlık; içerik düzenlendiğinde yazılır
//
// Ağırlıklar ve eğri Go sabitlerinden üretilir, elle yazılmaz: aksi hâlde
// Go ile SQL zamanla ayrışır ve panelde görünen puan ile sıralama birbirini
// tutmaz. TestSQLVeGoAyniOnceligiVerir bunu sabitler.

// TazelikSQL, yaşa göre 0–100 arası tazeliği veren SQL ifadesini üretir.
// alias, contents tablosunun sorgudaki takma adıdır (ör. "c").
func TazelikSQL(alias string) string {
	saat := fmt.Sprintf("(EXTRACT(EPOCH FROM (now() - %s.created_at)) / 3600.0)", alias)
	ilkGun := tazelikIlkGun.Hours()
	sinir := tazelikSinir.Hours()
	return fmt.Sprintf(`CASE
		WHEN %[1]s <= 0 THEN 100.0
		WHEN %[1]s >= %[3]g THEN 0.0
		WHEN %[1]s <= %[2]g THEN 100.0 - %[4]g * (%[1]s / %[2]g)
		ELSE %[5]g * (1.0 - ((%[1]s - %[2]g) / (%[3]g - %[2]g)))
	END`, saat, ilkGun, sinir, 100-tazelikIlkGunDe, tazelikIlkGunDe)
}

// OncelikSQL triyaj sıralaması için 0–100 arası önceliği üretir.
// Ağırlıklar Go sabitlerinden gelir.
func OncelikSQL(alias string) string {
	toplam := AgirlikTrend + AgirlikTazelik + AgirlikGuven + AgirlikTamlik
	return fmt.Sprintf(`((%d * COALESCE(%s.score_trend,0)
		+ %d * (%s)
		+ %d * COALESCE(%s.score_guven,0)
		+ %d * COALESCE(%s.ai_score,0)) / %d.0)`,
		AgirlikTrend, alias,
		AgirlikTazelik, TazelikSQL(alias),
		AgirlikGuven, alias,
		AgirlikTamlik, alias,
		toplam)
}

// GuvenSQL, feeds satırı join'lenmiş bir sorguda güven eksenini üretir.
// KaynakGuveni ile aynı kural: bilinmeyen kaynak 50, agregatör 15 düşük.
func GuvenSQL(feedAlias string) string {
	return fmt.Sprintf(`GREATEST(0, LEAST(100,
		(CASE WHEN COALESCE(%[1]s.trust,0) <= 0 THEN 50 ELSE %[1]s.trust END)
		- (CASE WHEN COALESCE(%[1]s.aggregator,FALSE) THEN 15 ELSE 0 END)))`, feedAlias)
}
