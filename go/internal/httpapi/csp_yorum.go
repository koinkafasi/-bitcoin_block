package httpapi

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
