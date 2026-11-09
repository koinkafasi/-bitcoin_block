package repo

// Kısa büyük-sayı biçimi: piyasa değeri gibi 13 haneli sayılar.
//
// NEDEN: "1,635,401,465,856" 326 px'lik sol raya sığmıyor ve okunmuyor
// (ölçüldü: taşıyor). CoinDesk de kısaltma kullanıyor ("$1.64T").
//
// BİÇİM: para simgesi + iki ondalık + kısaltma harfi (T/B/M/K). ABD'de
// T/B/M (trilyon/milyar/milyon) yazımı uluslararası finans yazımıyla ortak;
// TRY için aynı harfler kullanılır (₺423.35B) — sayfa zaten İngiliz
// biçimli binlik/ondalık ayırıcı kullanıyor (repo.SayiBicim).

// SayiBicimKisa 1_635_401_465_856 → "$1.64T" · 423_349_420_032 → "₺423.35B".
func SayiBicimKisa(f float64, para string) string {
	if f <= 0 {
		return "—"
	}
	simge := "$"
	switch para {
	case "TRY":
		simge = "₺"
	case "EUR":
		simge = "€"
	case "GBP":
		simge = "£"
	}
	switch {
	case f >= 1e12:
		return simge + binlikAyrac(f/1e12, 2) + "T"
	case f >= 1e9:
		return simge + binlikAyrac(f/1e9, 2) + "B"
	case f >= 1e6:
		return simge + binlikAyrac(f/1e6, 2) + "M"
	case f >= 1e3:
		return simge + binlikAyrac(f/1e3, 2) + "K"
	default:
		return simge + binlikAyrac(f, 2)
	}
}
