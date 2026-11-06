package repo

import "strings"

// CoinSlug coin ADINI kanonik fiyat yolu parçasına çevirir.
//
// NEDEN TAM AD, SEMBOL DEĞİL (spec §10.5): CoinDesk `/price/bitcoin`
// kullanıyor. SEO'da "bitcoin price" aramasıyla birebir eşleşiyor;
// `/price/btc` kısaltması o sorguda zayıf. Sembol adresleri 301 ile
// kanonik adrese taşınır, yani eski bağlantılar kırılmaz.
//
// Örnekler:
//
//	"Bitcoin"       → "bitcoin"
//	"Bitcoin Cash"  → "bitcoin-cash"
//	"BNB"           → "bnb"
//	"XRP"           → "xrp"
//	"Staked Ether"  → "staked-ether"
//
// Yalnız a-z0-9 ve tek tire bırakır: adres güvenli, büyük/küçük harf
// duyarsız ve yol aşımı üretemez.
func CoinSlug(ad string) string {
	ad = strings.ToLower(strings.TrimSpace(ad))
	var b strings.Builder
	b.Grow(len(ad))
	for _, r := range ad {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ' || r == '-' || r == '_' || r == '.' || r == '/':
			b.WriteByte('-')
		// Türkçe harfler: ada düşerse harf SESSİZCE kaybolurdu
		// ("türk" → "trk"). Latin karşılığına çevir.
		case r == 'ç':
			b.WriteByte('c')
		case r == 'ğ':
			b.WriteByte('g')
		case r == 'ı':
			b.WriteByte('i')
		case r == 'ö':
			b.WriteByte('o')
		case r == 'ş':
			b.WriteByte('s')
		case r == 'ü':
			b.WriteByte('u')
		}
	}
	out := b.String()
	for strings.Contains(out, "--") {
		out = strings.ReplaceAll(out, "--", "-")
	}
	return strings.Trim(out, "-")
}
