package httpapi

import (
	"context"
	"regexp"
	"strings"

	"theblock/site/internal/repo"
	"theblock/site/internal/webui/templkit"
)

// Üçüncü taraf entegrasyon ayarları — GÜVENLİK TASARIMI.
//
// Panelden serbest HTML/JS yapıştırma YOK. Her entegrasyon için tek bir
// KİMLİK saklanır, biçimi regex ile doğrulanır, `<script>` etiketini site
// üretir. Nedenleri:
//
//  1) XSS: ham kod alanı, panel erişimi olan (ya da oturumu çalınan) birine
//     her sayfada çalışan kod yazma imkânı verir; CSP `script-src 'self'`
//     koruması da işe yaramaz çünkü kod bizim origin'imizden çıkar.
//  2) CSP: sayfa başındaki satır içi blokların sha256'sı hesaplanıyor;
//     panelden değişen bir kimlik hash'i bozar ve GA sessizce ölür.
//  3) Doğrulama: yanlış yapıştırılmış bir izleme kimliği ya sessizce
//     çalışmaz ya da başka birinin mülküne veri gönderir. Burada reddedilir.

var (
	// gtmIDRe GTM kapsayıcı kimliği: GTM-XXXXXXX (harf/rakam, 4-20).
	gtmIDRe = regexp.MustCompile(`^GTM-[A-Z0-9]{4,20}$`)

	// reklamZoneRe Coinzilla bölge kimliği (C- öneki atılmış hâli).
	reklamZoneRe = regexp.MustCompile(`^[a-zA-Z0-9]{8,64}$`)

	// cointrafficRe Cointraffic wkey.
	cointrafficRe = regexp.MustCompile(`^[A-Za-z0-9]{6,64}$`)
)

// websubIzinliHub bilinen WebSub hub alan adları.
//
// BEYAZ LİSTE ZORUNLU: panelden girilen adres, her yayında sunucumuzun
// istek ATACAĞI hedef olur. Serbest bırakılsaydı iç ağ adreslerine
// (169.254.169.254 gibi) istek attırılabilir, yani SSRF olurdu.
var websubIzinliHub = map[string]bool{
	"pubsubhubbub.appspot.com":    true,
	"pubsubhubbub.superfeedr.com": true,
	"websubhub.com":               true,
	"hub.superfeedr.com":          true,
	"pubsubhubbub.appspot.com/":   true,
}

// validGTM kimliği doğrular; geçersizse boş döner ("kapalı").
func validGTM(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	if !gtmIDRe.MatchString(s) {
		return ""
	}
	return s
}

// validReklamZone Coinzilla bölge kimliğini doğrular.
// Panelde `C-` önekiyle ya da tırnaklarla yapıştırılan değerler temizlenir.
func validReklamZone(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, `"'`)
	s = strings.TrimPrefix(s, "C-")
	s = strings.TrimPrefix(s, "c-")
	if !reklamZoneRe.MatchString(s) {
		return ""
	}
	return s
}

// validCointraffic anahtarı doğrular.
func validCointraffic(s string) string {
	s = strings.TrimSpace(s)
	if !cointrafficRe.MatchString(s) {
		return ""
	}
	return s
}

// validWebSubHubs virgül/satır ayrılmış listeyi beyaz listeden süzer.
//
// Geçersiz girdiler SESSİZCE atılmaz: çağıran uyarı basabilsin diye
// ikinci değer reddedilenlerin listesidir.
func validWebSubHubs(s string) ([]string, []string) {
	var gecerli, red []string
	for _, parca := range strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r'
	}) {
		u := strings.TrimSpace(parca)
		if u == "" {
			continue
		}
		if !strings.HasPrefix(u, "https://") {
			red = append(red, u)
			continue
		}
		host := strings.TrimPrefix(u, "https://")
		if i := strings.IndexByte(host, '/'); i >= 0 {
			host = host[:i]
		}
		host = strings.ToLower(host)
		if !websubIzinliHub[host] {
			red = append(red, u)
			continue
		}
		gecerli = append(gecerli, u)
	}
	return gecerli, red
}

// cdEntegrasyonCoz ayarlardan doğrulanmış entegrasyon kimliklerini üretir.
func (s *Server) cdEntegrasyonCoz(ctx context.Context) *templkit.CDEntegrasyon {
	bag, _ := repo.GetSettings(ctx, s.pool, "baglantilar")
	e := &templkit.CDEntegrasyon{
		GA4ID:           s.ga4Measurement(ctx),
		GSCVerification: s.gVerification(ctx),
	}
	if meta := obj(bag["meta"]); meta != nil {
		e.GTMID = validGTM(jsonStrAny(meta["gtm_id"]))
		e.GTMNoScript = jsonStrAny(meta["gtm_noscript"]) == "1"
	}
	e.ReklamZone = validReklamZone(jsonStr(bag["reklam_zone"]))
	e.ReklamAktif = jsonStr(bag["reklam_aktif"]) == "1"
	e.CointrafficKey = validCointraffic(jsonStr(bag["cointraffic_key"]))
	hubs, _ := validWebSubHubs(jsonStr(bag["websub_hubs"]))
	e.WebSubHubs = hubs
	return e
}

// CDEntegrasyonYenile ayarları okuyup templkit'e yayınlar.
//
// Sunucu açılışında ve HER ayar kaydından sonra çağrılır; sayfa istekleri
// atomik işaretçiyi okur, veritabanına gitmez.
func (s *Server) CDEntegrasyonYenile(ctx context.Context) {
	templkit.SetCDEntegrasyon(s.cdEntegrasyonCoz(ctx))
}

// jsonStrAny harita değerini stringe çevirir (tip karışık olabilir).
func jsonStrAny(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case bool:
		if t {
			return "1"
		}
		return "0"
	}
	return ""
}
