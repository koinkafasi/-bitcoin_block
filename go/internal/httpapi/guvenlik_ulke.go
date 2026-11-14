package httpapi

import "strings"

// ÜLKE → KITA eşlemesi ve saat dilimi uyuşmazlığı.
//
// SINIRI DÜRÜSTÇE: bu bir coğrafi doğruluk aracı değil, kaba bir tutarlılık
// kontrolü. Amaç: VPN düğümünü başka kıtada tutarken tarayıcısı Türkiye
// saatinde olan ziyaretçiyi yakalamak. Aynı kıta içindeki uyuşmazlık
// (Almanya IP + Fransa saati) YAKALANMAZ ve yakalanmamalı — olağan.
//
// Yalnız KITALAR ARASI fark puanlanır (+40). Tek başına bot yapmaz ama
// hosting/VPN sinyaliyle birleşince kararı sağlamlaştırır.
//
// NEDEN ELLE TABLO: hazır bir tz→ülke kütüphanesi (time/tzdata ~450 KB)
// tek bir kontrol için ağır. Bu tablo yalnız "kıta" bilgisi tutar.

type kita int

const (
	kitaYok kita = iota
	kitaAvrupa
	kitaAmerika
	kitaAsya
	kitaAfrika
	kitaOkyanusya
)

// ulkeKita: sık gelen ülkeler + yaygın VPN çıkış ülkeleri.
var ulkeKita = map[string]kita{
	// Avrupa
	"TR": kitaAvrupa, "DE": kitaAvrupa, "GB": kitaAvrupa, "FR": kitaAvrupa,
	"NL": kitaAvrupa, "IT": kitaAvrupa, "ES": kitaAvrupa, "PT": kitaAvrupa,
	"IE": kitaAvrupa, "BE": kitaAvrupa, "AT": kitaAvrupa, "CH": kitaAvrupa,
	"PL": kitaAvrupa, "CZ": kitaAvrupa, "HU": kitaAvrupa, "RO": kitaAvrupa,
	"BG": kitaAvrupa, "GR": kitaAvrupa, "SE": kitaAvrupa, "NO": kitaAvrupa,
	"DK": kitaAvrupa, "FI": kitaAvrupa, "IS": kitaAvrupa, "EE": kitaAvrupa,
	"LV": kitaAvrupa, "LT": kitaAvrupa, "SK": kitaAvrupa, "SI": kitaAvrupa,
	"HR": kitaAvrupa, "RS": kitaAvrupa, "BA": kitaAvrupa, "MK": kitaAvrupa,
	"AL": kitaAvrupa, "XK": kitaAvrupa, "MD": kitaAvrupa, "UA": kitaAvrupa,
	"BY": kitaAvrupa, "RU": kitaAvrupa, "CY": kitaAvrupa, "MT": kitaAvrupa,
	"LU": kitaAvrupa, "LI": kitaAvrupa, "AD": kitaAvrupa, "MC": kitaAvrupa,
	"SM": kitaAvrupa, "VA": kitaAvrupa, "GI": kitaAvrupa, "JE": kitaAvrupa,
	"GG": kitaAvrupa, "IM": kitaAvrupa, "FO": kitaAvrupa, "AX": kitaAvrupa,
	"GE": kitaAvrupa, "AM": kitaAvrupa, "AZ": kitaAvrupa, "KZ": kitaAsya,
	// Amerika
	"US": kitaAmerika, "CA": kitaAmerika, "MX": kitaAmerika, "BR": kitaAmerika,
	"AR": kitaAmerika, "CL": kitaAmerika, "CO": kitaAmerika, "PE": kitaAmerika,
	"VE": kitaAmerika, "UY": kitaAmerika, "PY": kitaAmerika, "BO": kitaAmerika,
	"EC": kitaAmerika, "CR": kitaAmerika, "PA": kitaAmerika, "GT": kitaAmerika,
	"DO": kitaAmerika, "PR": kitaAmerika, "CU": kitaAmerika, "JM": kitaAmerika,
	"TT": kitaAmerika, "BZ": kitaAmerika, "HN": kitaAmerika, "SV": kitaAmerika,
	"NI": kitaAmerika, "BS": kitaAmerika, "BB": kitaAmerika, "HT": kitaAmerika,
	// Asya
	"CN": kitaAsya, "JP": kitaAsya, "KR": kitaAsya, "TW": kitaAsya,
	"HK": kitaAsya, "MO": kitaAsya, "SG": kitaAsya, "MY": kitaAsya,
	"TH": kitaAsya, "VN": kitaAsya, "ID": kitaAsya, "PH": kitaAsya,
	"IN": kitaAsya, "PK": kitaAsya, "BD": kitaAsya, "LK": kitaAsya,
	"NP": kitaAsya, "MM": kitaAsya, "KH": kitaAsya, "LA": kitaAsya,
	"MN": kitaAsya, "UZ": kitaAsya, "KG": kitaAsya, "TJ": kitaAsya,
	"TM": kitaAsya, "AF": kitaAsya, "IR": kitaAsya, "IQ": kitaAsya,
	"SA": kitaAsya, "AE": kitaAsya, "QA": kitaAsya, "KW": kitaAsya,
	"BH": kitaAsya, "OM": kitaAsya, "YE": kitaAsya, "JO": kitaAsya,
	"LB": kitaAsya, "SY": kitaAsya, "IL": kitaAsya, "PS": kitaAsya,
	// Afrika
	"EG": kitaAfrika, "MA": kitaAfrika, "DZ": kitaAfrika, "TN": kitaAfrika,
	"LY": kitaAfrika, "ZA": kitaAfrika, "NG": kitaAfrika, "KE": kitaAfrika,
	"ET": kitaAfrika, "GH": kitaAfrika, "CI": kitaAfrika, "SN": kitaAfrika,
	"CM": kitaAfrika, "UG": kitaAfrika, "TZ": kitaAfrika, "ZW": kitaAfrika,
	"ZM": kitaAfrika, "MZ": kitaAfrika, "AO": kitaAfrika, "BW": kitaAfrika,
	"NA": kitaAfrika, "MU": kitaAfrika, "SC": kitaAfrika, "MG": kitaAfrika,
	"SD": kitaAfrika, "SO": kitaAfrika, "RW": kitaAfrika, "BI": kitaAfrika,
	"CD": kitaAfrika, "CG": kitaAfrika, "GA": kitaAfrika, "ML": kitaAfrika,
	"BF": kitaAfrika, "NE": kitaAfrika, "TD": kitaAfrika, "GN": kitaAfrika,
	// Okyanusya
	"AU": kitaOkyanusya, "NZ": kitaOkyanusya, "FJ": kitaOkyanusya,
	"PG": kitaOkyanusya, "NC": kitaOkyanusya, "PF": kitaOkyanusya,
	"GU": kitaOkyanusya, "WS": kitaOkyanusya, "TO": kitaOkyanusya,
	"VU": kitaOkyanusya, "SB": kitaOkyanusya, "KI": kitaOkyanusya,
}

// dilKita: birincil dil → kıta. Yalnız saat dilimi yoksa yedek sinyal.
// Burada da amaç dilbilim değil, kaba tutarlılık.
var dilKita = map[string]kita{
	"tr": kitaAvrupa, "de": kitaAvrupa, "en": kitaYok, // en: dünya dili, kullanılmaz
	"fr": kitaAvrupa, "it": kitaAvrupa, "es": kitaYok, // es: İspanya + Latin Amerika
	"pt": kitaYok, "nl": kitaAvrupa, "pl": kitaAvrupa, "cs": kitaAvrupa,
	"hu": kitaAvrupa, "ro": kitaAvrupa, "bg": kitaAvrupa, "el": kitaAvrupa,
	"sv": kitaAvrupa, "no": kitaAvrupa, "da": kitaAvrupa, "fi": kitaAvrupa,
	"ru": kitaAvrupa, "uk": kitaAvrupa, "sr": kitaAvrupa, "hr": kitaAvrupa,
	"sk": kitaAvrupa, "sl": kitaAvrupa, "lt": kitaAvrupa, "lv": kitaAvrupa,
	"et": kitaAvrupa, "sq": kitaAvrupa, "mk": kitaAvrupa, "bs": kitaAvrupa,
	"ar": kitaYok, "fa": kitaAsya, "he": kitaAsya, "hi": kitaAsya,
	"ur": kitaAsya, "bn": kitaAsya, "id": kitaAsya, "ms": kitaAsya,
	"th": kitaAsya, "vi": kitaAsya, "ja": kitaAsya, "ko": kitaAsya,
	"zh": kitaAsya, "az": kitaAsya, "kk": kitaAsya, "uz": kitaAsya,
	"ky": kitaAsya, "tg": kitaAsya, "tk": kitaAsya, "ka": kitaAsya,
	"hy": kitaAsya, "sw": kitaAfrika, "am": kitaAfrika, "ha": kitaAfrika,
	"yo": kitaAfrika, "ig": kitaAfrika, "zu": kitaAfrika, "af": kitaAfrika,
}

// saatDilimiKita IANA saat dilimi adından kıta çıkarır.
//
// Atlantik/Indian gibi belirsiz bölgeler kitaYok döner: yanlış pozitif
// üretmektense sinyali kaçırmak yeğdir (yanlış engel gerçek okur kaybı).
func saatDilimiKita(tz string) kita {
	tz = strings.TrimSpace(tz)
	if tz == "" {
		return kitaYok
	}
	bolge := tz
	if i := strings.IndexByte(tz, '/'); i > 0 {
		bolge = tz[:i]
	}
	switch bolge {
	case "Europe":
		return kitaAvrupa
	case "America":
		return kitaAmerika
	case "Asia":
		return kitaAsya
	case "Africa":
		return kitaAfrika
	case "Australia", "Pacific":
		return kitaOkyanusya
	}
	// Atlantic/, Indian/, Arctic/, UTC → belirsiz.
	return kitaYok
}

// kitaUyusmazlik: IP ülkesi ile saat dilimi kıtası çelişiyor mu.
//
// YALNIZ iki taraf da biliniyorsa ve FARKLI ise true. Eşleme tablosunda
// olmayan ülke → bilinmiyor → çelişki yok (fail-open).
func kitaUyusmazlik(ulke, tz string) bool {
	uk, ok := ulkeKita[strings.ToUpper(strings.TrimSpace(ulke))]
	if !ok {
		return false
	}
	tk := saatDilimiKita(tz)
	if tk == kitaYok || uk == kitaYok {
		return false
	}
	return uk != tk
}

// dilUyusmazlik: IP ülkesi ile birincil tarayıcı dili çelişiyor mu.
//
// `en` ve `es`/`pt`/`ar` gibi çok kıtalı diller tabloda kitaYok → hiç
// puanlanmaz. Bu sinyal zaten düşük ağırlıklı (+15).
func dilUyusmazlik(ulke, dil string) bool {
	uk, ok := ulkeKita[strings.ToUpper(strings.TrimSpace(ulke))]
	if !ok {
		return false
	}
	d := strings.ToLower(strings.TrimSpace(dil))
	if i := strings.IndexAny(d, "-_"); i > 0 {
		d = d[:i]
	}
	dk, ok := dilKita[d]
	if !ok || dk == kitaYok || uk == kitaYok {
		return false
	}
	return uk != dk
}

// kitaAdi panelde gösterim (şu an panelde kullanılmıyor; ileride ülke
// kırılımında kıta sütunu için tutuluyor).
func kitaAdi(k kita) string {
	switch k {
	case kitaAvrupa:
		return "Avrupa"
	case kitaAmerika:
		return "Amerika"
	case kitaAsya:
		return "Asya"
	case kitaAfrika:
		return "Afrika"
	case kitaOkyanusya:
		return "Okyanusya"
	}
	return "Bilinmiyor"
}
