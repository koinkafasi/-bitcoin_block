package httpapi

import "strings"

// Özet kutusu yardımcıları (spec §10.2.2).
//
// CoinDesk kutusu 3 madde gösteriyor; maddeler AI özetinin cümleleridir.
// NEDEN cümleye bölüyoruz: özet tek paragraf metin olarak saklanıyor
// (contents.ai_summary). Madde işaretli liste göstermek için cümlelere
// ayırmak gerekiyor — ayrı bir "maddeler" kolonu eklemek şemayı
// şişirirdi ve mevcut veriyi yeniden üretmek gerekirdi.

// ilkCumle özetin kapalı durumda görünen tek satırını üretir.
func ilkCumle(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if i := cumleSonu(s); i > 0 {
		return strings.TrimSpace(s[:i])
	}
	return s
}

// cumlelereBol özeti en çok `adet` cümleye böler.
//
// Kısa/kalıntısız özetlerde tek madde döner; boş metin boş dilim döner
// (çağıran zaten boşta kutu basmıyor).
func cumlelereBol(s string, adet int) []string {
	s = strings.TrimSpace(s)
	if s == "" || adet <= 0 {
		return nil
	}
	var out []string
	kalan := s
	for len(out) < adet {
		i := cumleSonu(kalan)
		if i <= 0 {
			if t := strings.TrimSpace(kalan); t != "" {
				out = append(out, t)
			}
			break
		}
		out = append(out, strings.TrimSpace(kalan[:i]))
		kalan = strings.TrimSpace(kalan[i:])
		if kalan == "" {
			break
		}
	}
	return out
}

// cumleSonu metindeki ilk cümle bitişinin BİTİŞ indeksini verir (ayırıcı
// dahil). Bulunamazsa -1.
//
// Yalnız ". " / "! " / "? " ve metin sonundaki noktalama sayılır:
// "3.5 milyar" gibi ondalık sayı cümle sonu DEĞİLDİR. Türkçe kısaltmalar
// (vb., Dr.) yanlış bölebilir; bu kutu bir ÖZET olduğu için en kötü
// durumda fazladan bir madde çıkar — bilgi kaybı olmaz.
func cumleSonu(s string) int {
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '.', '!', '?':
			// Sonraki karakter boşluk ya da metin sonu olmalı.
			if i+1 == len(s) {
				return i + 1
			}
			if s[i+1] == ' ' || s[i+1] == '\n' || s[i+1] == '\t' {
				// Ondalık/binlik ayırıcı: ". " değil ".5" → zaten boşluk
				// kontrolü bunu eler. Yine de iki nokta üst üste
				// ardışıklığını (… ) atlamak için tek karakter ilerle.
				return i + 1
			}
		}
	}
	return -1
}
