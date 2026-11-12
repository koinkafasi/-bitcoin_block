package scoring

import "fmt"

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
