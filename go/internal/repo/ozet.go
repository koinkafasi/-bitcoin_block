package repo

import (
	"context"
	"strings"
)

// AiOzet içeriğin AI özetini (contents.ai_summary) döndürür.
//
// NEDEN AYRI SORGU: özet YALNIZ makale sayfasında gerekiyor; içerik
// listelerinde (ana sayfa, bölüm, etiket) taşınması 126 satırda boşuna
// veri çekmek olurdu. Tek kolonluk indeksli okuma, liste sorgularının
// kolon listesini şişirmekten daha ucuz.
//
// Boş dönerse çağıran özet kutusunu HİÇ basmaz: uydurma özet yanlış bilgi
// olurdu (spec §10.2.2 kutusu gerçek veri bekliyor).
func AiOzet(ctx context.Context, pool DB, contentID int64) string {
	var s *string
	if err := pool.QueryRow(ctx,
		`SELECT ai_summary FROM contents WHERE id = $1`, contentID).Scan(&s); err != nil || s == nil {
		return ""
	}
	return strings.TrimSpace(*s)
}
