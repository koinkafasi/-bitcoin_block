package httpapi

import (
	"net/http"
	"strings"

	"theblock/site/internal/repo"
)

// ---- Yeni tasarımda içerik kilidi (abonelik kademesi) ----
//
// DURUM (12 Kasım 2026, denetim): veri modeli, katmanlar (5 kademe),
// admin alanı ve Stripe bağlantısı HAZIRDI; kilit kararı ise yalnız ESKİ
// şablonda uygulanıyordu. Yeni CoinDesk tasarımına taşınmadığı için
// `min_tier_weight` taşıyan içerikler tam metin görünüyordu (ölçüldü:
// kademe 1 işaretli yazı, üye olmayan ziyaretçiye 7 paragraf birden).
//
// Bu dosya kilidi makale akışına bağlar. Kural tek yerde:
//   gerekli kademe = min_tier_weight (yoksa is_exclusive → 1)
//   erişim         = üye ağırlığı >= gerekli kademe
//   anahtar        = settings[paywall.aktif] (kapalıysa hiçbir şey kilitlenmez)

// cdKilitDurumu içerik kilitli mi, kilitliyse hangi katman gerekiyor.
type cdKilitDurumu struct {
	Kilitli bool
	// Katman gereken abonelik katmanının görünen adı ("Gümüş").
	Katman string
	// Agirlik gereken kademe (1..5). Kilitli değilse 0.
	Agirlik int
}

// cdKilitHesapla kilidi değerlendirir. Paywall kapalıyken ya da ziyaretçi
// yeterli katmandayken her zaman açık döner.
func (s *Server) cdKilitHesapla(r *http.Request, c repo.Content) cdKilitDurumu {
	gerekli := contentRequiredWeight(c.IsExclusive, c.MinTierWeight)
	if gerekli <= 0 {
		return cdKilitDurumu{}
	}
	if !s.paywallAktif(r.Context()) {
		return cdKilitDurumu{}
	}
	if tierAllows(s.memberWeight(r), gerekli) {
		return cdKilitDurumu{}
	}
	return cdKilitDurumu{Kilitli: true, Agirlik: gerekli, Katman: s.cdKatmanAdi(r, gerekli)}
}

// cdKatmanAdi ağırlığa karşılık gelen katmanın adını verir; bulunamazsa
// boş döner ve şablon genel ifade kullanır ("abonelere özel").
func (s *Server) cdKatmanAdi(r *http.Request, agirlik int) string {
	tiers, err := repo.ListTiers(r.Context(), s.pool, true)
	if err != nil {
		return ""
	}
	// Gerekli ağırlığı karşılayan EN DÜŞÜK katman: kullanıcıya en ucuz
	// yolu göstermek doğru olan.
	ad, bulunan := "", 0
	for _, t := range tiers {
		if t.Weight < agirlik {
			continue
		}
		if bulunan == 0 || t.Weight < bulunan {
			ad, bulunan = t.Name, t.Weight
		}
	}
	return ad
}

// cdOnizlemeKes kilitli içeriğin ücretsiz önizlemesini üretir.
//
// NEDEN PARAGRAF SAYISI: Ghost ve CoinDesk gibi yayınlar içeriğin ilk
// bölümünü açık bırakıp gerisini kapatıyor. Karakter sayısıyla kesmek
// cümleyi ortadan bölüyordu; paragraf sınırı hem okunur hem de HTML'i
// bozmuyor.
//
// Sonuç HTML'i ilk `adet` paragrafla sınırlıdır; gövde daha kısaysa
// olduğu gibi döner (kilit ekranı yine basılır).
func cdOnizlemeKes(govde string, adet int) string {
	if adet <= 0 || govde == "" {
		return ""
	}
	var sb strings.Builder
	kalan := adet
	i := 0
	for kalan > 0 {
		bas := strings.Index(govde[i:], "<p>")
		if bas < 0 {
			break
		}
		bas += i
		son := strings.Index(govde[bas:], "</p>")
		if son < 0 {
			break
		}
		son += bas + len("</p>")
		sb.WriteString(govde[bas:son])
		i = son
		kalan--
	}
	return sb.String()
}
