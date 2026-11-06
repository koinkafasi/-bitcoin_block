package httpapi

import "strings"

// htmlYorumAtla, `rest` içinde `<etiket` aramasından ÖNCE bir HTML yorumu
// (`<!-- … -->`) varsa yorumu atlayıp kalan metni döndürür.
//
// NEDEN GEREKLİ (yaşanmış hata, 6 Kasım 2026): CSP hash'leyicisi metinde
// `<script` arıyordu ve HTML YORUMU içindeki `<script>` kelimesini gerçek
// etiket sandı. Yorum + gerçek script tek parça olarak hash'lendi:
//
//	sunucu : <script>…</script> → 478 bayt hash (yanlış)
//	tarayıcı: yalnız gerçek script  → 400 bayt hash (doğru)
//
// Sonuç: başlıktaki hash hiçbir zaman tutmadı, sayfadaki satır içi betik
// sessizce engellendi (konsolda "Executing inline script violates…").
// Belirti sinsidir: sayfa 200 döner, yalnız betik çalışmaz.
//
// Kural: HTML yorumu içindeki etiket metni DEĞİLDİR. Yorumlar (HTML'de
// iç içe geçemez) atlanır; `<script>` yorumun İÇİNDEyse tamamen yok sayılır.
//
// found, aranan etiketin yeni metindeki konumudur (-1: yok).
func htmlYorumAtla(rest, etiket string) (string, int) {
	for {
		si := indexFold(rest, etiket)
		if si < 0 {
			return rest, -1
		}
		ci := indexFold(rest, "<!--")
		if ci < 0 || ci > si {
			// Yorum yok ya da etiketten SONRA başlıyor: gerçek etiket.
			return rest, si
		}
		ce := indexFold(rest[ci:], "-->")
		if ce < 0 {
			// Kapanmamış yorum: geri kalanı yok say (tarayıcı da öyle yapar).
			return "", -1
		}
		// Yorumu tamamen atla; içindeki etiket metni sayılmaz.
		rest = rest[ci+ce+3:]
	}
}

// htmlYorumTemizle, testlerde/teşhiste yorumları kaldırmak için yardımcı.
func htmlYorumTemizle(s string) string {
	var b strings.Builder
	for {
		i := indexFold(s, "<!--")
		if i < 0 {
			b.WriteString(s)
			return b.String()
		}
		b.WriteString(s[:i])
		j := indexFold(s[i:], "-->")
		if j < 0 {
			return b.String()
		}
		s = s[i+j+3:]
	}
}
