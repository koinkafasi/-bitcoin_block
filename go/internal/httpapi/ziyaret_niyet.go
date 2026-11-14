package httpapi

import (
	"net/url"
	"strings"
)

// ZİYARET NİYETİ — "ne yapmaya çalışıyor" sorusunun sunucu tarafı cevabı.
//
// TASARIM: iki ayrı katman, ikisi de SAF fonksiyon (test edilebilir,
// DNS/DB yok):
//
//	1) SINIFLANDIRMA (olayTuru / olayKaynagi) — her istek için tek etiket.
//	   Yazarken hesaplanır, DB'de durur. Panel listesi 120 ziyaretçinin
//	   niyetini tek GROUP BY ile okur.
//	2) ÇIKARIM (niyetCikar) — bir ziyaretçinin olay dizisinden cümle.
//	   Okurken hesaplanır; yalnız profil sayfasında, 200 olay üzerinde.
//
// NEDEN İKİSİ AYRI: sınıflandırma her istekte koşar (maliyeti O(1) olmalı),
// çıkarım yalnız panel açılınca koşar. Tek fonksiyonda birleştirmek, ya
// istek yoluna cümle kurmayı ya da panelde 120 ziyaretçi için 120 kez
// tüm yolu yeniden sınıflandırmayı getirirdi.
//
// SINIRI DÜRÜSTÇE: bu bir niyet OKUMA değil, davranış TAHMİNİ. "Zafiyet
// taraması" etiketi /wp-login.php istendiği anlamına gelir; isteyenin
// niyeti hakkında kesin bir iddia değildir. Panelde de böyle yazıyor.

// Olay türleri — migration 0133'teki `tur` sütunuyla aynı sözlük.
const (
	turIcerik  = "icerik"
	turBesleme = "besleme"
	turArama   = "arama"
	turAPI     = "api"
	turGiris   = "giris"
	turYonetim = "yonetim"
	turZafiyet = "zafiyet"
	turTuzak   = "tuzak"
	turDiger   = "diger"
)

// Kaynak türleri — `kaynak` sütunu.
const (
	kaynakYok    = "yok"
	kaynakIc     = "ic"
	kaynakArama  = "arama"
	kaynakSosyal = "sosyal"
	kaynakDis    = "dis"
)

// zafiyetYollari bilinen tarama/exploit hedefleri.
//
// NEDEN LİSTE, NEDEN SEZGİSEL DEĞİL: bu yolların hepsi meşru bir okurun
// ASLA istemeyeceği adresler. Sezgisel bir kural ("bilinmeyen yol")
// yeni açılan her sayfayı zafiyet sayardı; sabit liste yanlış pozitif
// üretmez, yalnız yeni bir tarama dalgasını kaçırır — asimetri doğru yönde.
var zafiyetYollari = []string{
	"wp-login", "wp-admin", "wp-content", "wp-includes", "wp-json",
	"xmlrpc", "phpmyadmin", "pma", "myadmin", "adminer",
	".env", ".git", ".aws", ".ssh", ".svn", ".ds_store", ".htaccess",
	"config.php", "config.json", "configuration.php", "settings.php",
	"phpunit", "eval-stdin", "shell.php", "cmd.php", "webshell",
	"cgi-bin", "vendor/phpunit", "actuator", "solr/admin", "console",
	"owa/auth", "autodiscover", "ecp/", "remote/login", "hnap1",
	"boaform", "setup.cgi", "shell", "passwd", "id_rsa",
	"api/v1/keys", "graphql", "swagger", "openapi.json", "server-status",
	"telescope", "_ignition", "laravel", "druid", "jenkins", "manager/html",
}

// beslemeYollari crawler'ın toplu çektiği uçlar.
var beslemeYollari = []string{
	"/rss", "/feed", "/atom", "/sitemap", "/robots.txt", "/news-sitemap",
	"/amp", "/index.xml", "/feed.xml", "/rss.xml", "/sitemap.xml",
}

// aramaYollari site içi arama.
var aramaYollari = []string{"/ara", "/arama", "/search", "/bul"}

// girisYollari kimlik işlemleri (zafiyet listesinden SONRA bakılır).
var girisYollari = []string{
	"/giris", "/kayit", "/uyelik", "/sifre", "/oturum", "/hesabim/giris",
	"/login", "/signin", "/register", "/password", "/reset",
}

// yonetimYollari panel.
var yonetimYollari = []string{"/admin", "/yonetim", "/panel"}

// olayTuru istek yolunu tek türe indirger. Sıra ÖNEMLİ: en dar ve en
// kesin kural önce. `/admin/giris` yönetimdir (zafiyet değil, giriş değil):
// panelin kendi giriş sayfasını tarama saymak gürültü olurdu.
func olayTuru(yol string) string {
	y := strings.ToLower(strings.TrimSpace(yol))
	if y == "" {
		return turDiger
	}
	// Sorgu dizgesi gelirse at: sınıflandırma YOL üzerinden yapılır.
	// (Middleware temiz yolu geçiyor; bu, fonksiyonu çağıran her yeni
	// yerin aynı hataya düşmemesi için savunma katmanı.)
	if i := strings.IndexByte(y, '?'); i >= 0 {
		y = y[:i]
	}
	// Tuzak yolu en önce: honeypot'a düşen istek başka hiçbir şey değildir.
	if isTrapPath(y) {
		return turTuzak
	}
	// Yönetim paneli: kendi girişimiz dahil.
	for _, p := range yonetimYollari {
		if y == p || strings.HasPrefix(y, p+"/") || strings.HasPrefix(y, p+"?") {
			return turYonetim
		}
	}
	// Zafiyet taraması: yolun HERHANGİ bir yerinde imza aranır.
	// (`/blog/wp-login.php`, `/?x=/.env` gibi sarmalamalar yaygın.)
	for _, imza := range zafiyetYollari {
		if strings.Contains(y, imza) {
			return turZafiyet
		}
	}
	for _, p := range beslemeYollari {
		if y == p || strings.HasPrefix(y, p+".") || strings.HasPrefix(y, p+"/") {
			return turBesleme
		}
	}
	for _, p := range aramaYollari {
		if y == p || strings.HasPrefix(y, p+"/") {
			return turArama
		}
	}
	if strings.HasPrefix(y, "/api/") || y == "/api" {
		return turAPI
	}
	for _, p := range girisYollari {
		if y == p || strings.HasPrefix(y, p+"/") || strings.HasPrefix(y, p+"-") {
			return turGiris
		}
	}
	return turIcerik
}

// aramaMotorlari Referer alanından arama motoru tanıyan imzalar.
var aramaMotorlari = []string{
	"google.", "bing.com", "yandex.", "duckduckgo.com", "baidu.com",
	"search.yahoo.", "brave.com", "ecosia.org", "qwant.com", "startpage.",
	"seznam.cz", "naver.com", "daum.net", "sogou.com",
}

// sosyalSiteler paylaşım/iletişim kaynakları.
var sosyalSiteler = []string{
	"facebook.com", "fb.me", "instagram.com", "twitter.com", "x.com",
	"t.co", "t.me", "telegram.", "reddit.com", "linkedin.com", "lnkd.in",
	"youtube.com", "youtu.be", "whatsapp.com", "wa.me", "pinterest.",
	"discord.", "medium.com", "tumblr.com", "vk.com", "threads.net",
	"news.google", "flipboard.com",
}

// olayKaynagi Referer'ı kaynak sınıfına indirger. `kendiHost` sitenin
// kendi alan adı (www'suz); site içi gezinme en sık durumdur ve bot
// olmadığının kanıtıdır.
func olayKaynagi(referer, kendiHost string) string {
	ref := strings.TrimSpace(referer)
	if ref == "" {
		return kaynakYok
	}
	u, err := url.Parse(ref)
	if err != nil || u.Host == "" {
		return kaynakDis
	}
	h := strings.ToLower(u.Hostname())
	kendi := strings.ToLower(strings.TrimPrefix(kendiHost, "www."))
	if kendi != "" && (h == kendi || strings.HasSuffix(h, "."+kendi)) {
		return kaynakIc
	}
	for _, m := range aramaMotorlari {
		if strings.Contains(h, m) {
			return kaynakArama
		}
	}
	for _, s := range sosyalSiteler {
		if strings.Contains(h, s) {
			return kaynakSosyal
		}
	}
	return kaynakDis
}

// hassasParametreler değeri ASLA saklanmayacak sorgu anahtarları.
//
// Parça eşleşmesi kullanılıyor (`token`, `access_token`, `csrf_token`…):
// tam eşleşme listesi her yeni adlandırmada sızdırırdı.
var hassasParametreler = []string{
	"token", "jeton", "sifre", "şifre", "password", "passwd", "pwd",
	"kod", "code", "key", "secret", "auth", "session", "oturum", "sid",
	"imza", "signature", "sig", "email", "eposta", "e-posta", "mail",
	"telefon", "phone", "tckn", "iban", "kart", "card", "cvv", "captcha",
}

// sorguTemizle sorgu dizgesini saklanabilir hâle getirir: hassas
// parametrelerin DEĞERİ maskelenir, toplam uzunluk kırpılır.
//
// NEDEN GEREKLİ: parola sıfırlama bağlantısındaki jeton ya da
// `?email=...` bu tabloya düşerse, "davranış kaydı" bir kişisel veri
// deposuna dönüşür — hem KVKK hem güvenlik sorunu. Değer maskelenince
// satır hâlâ ANLAM taşır ("hangi uç, hangi parametre"), kimseyi teşhis
// etmez.
func sorguTemizle(ham string) string {
	ham = strings.TrimSpace(strings.TrimPrefix(ham, "?"))
	if ham == "" {
		return ""
	}
	parcalar := strings.Split(ham, "&")
	out := make([]string, 0, len(parcalar))
	for _, p := range parcalar {
		if p == "" {
			continue
		}
		ad, deger, varMi := strings.Cut(p, "=")
		kucuk := strings.ToLower(ad)
		hassas := false
		for _, h := range hassasParametreler {
			if strings.Contains(kucuk, h) {
				hassas = true
				break
			}
		}
		switch {
		case !varMi:
			out = append(out, ad)
		case hassas && deger != "":
			out = append(out, ad+"=***")
		case len(deger) > 40:
			// Uzun serbest değer (base64/jeton görünümlü) kırpılır.
			out = append(out, ad+"="+deger[:40]+"…")
		default:
			out = append(out, ad+"="+deger)
		}
	}
	s := strings.Join(out, "&")
	const sinir = 200
	if len(s) > sinir {
		s = s[:sinir] + "…"
	}
	return s
}

// yolTemizle dinamik yol parçalarını (parola sıfırlama jetonu, uzun
// kimlikler) maskeler. `yol` sütunu indeksli olduğu için kimlik
// taşımamalı: aksi hâlde her jeton ayrı bir "yol" olur ve pivot
// sorguları işe yaramaz.
func yolTemizle(yol string) string {
	if yol == "" {
		return "/"
	}
	parcalar := strings.Split(yol, "/")
	for i, p := range parcalar {
		if len(p) < 24 {
			continue
		}
		// Jeton görünümlü: uzun ve yalnız harf/rakam/-/_ içeriyor.
		jeton := true
		for _, c := range p {
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
				(c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.') {
				jeton = false
				break
			}
		}
		if jeton && strings.ContainsAny(p, "0123456789") {
			parcalar[i] = "***"
		}
	}
	y := strings.Join(parcalar, "/")
	const sinir = 300
	if len(y) > sinir {
		y = y[:sinir] + "…"
	}
	return y
}

// Niyet — bir ziyaretçinin olay dizisinden çıkarılan davranış özeti.
type Niyet struct {
	// Baslik kısa etiket: "Zafiyet taraması", "İçerik kazıma"…
	Baslik string
	// Aciklama tek cümlelik gerekçe (hangi kanıt).
	Aciklama string
	// Seviye: "bilgi" | "dikkat" | "tehdit" — panelde renk.
	Seviye string
	// Karsilastirma: ölçülen oranlar (panelde çubuk).
	BaskinTur string
	BaskinPay int
}

// NiyetTuru tek türün sayımı (profil sayfasındaki dağılım).
type NiyetTuru struct {
	Tur  string
	Ad   string
	Adet int
	Pay  int
}

// olayTurAdlari OLAY tur kodunun panel karşılığı.
//
// AD BENZERLİĞİ TUZAK: guvenlik_view.go`daki `turAdlari` ASN sağlayıcı
// türünü (hosting→"veri merkezi") çeviriyor. İki sözlük karışırsa panelde
// "zafiyet taraması" yerine "veri merkezi" yazardı; ayrı ad bunu
// derleme zamanında imkânsız kılıyor.
var olayTurAdlari = map[string]string{
	turIcerik:  "içerik okuma",
	turBesleme: "besleme / harita",
	turArama:   "site içi arama",
	turAPI:     "API",
	turGiris:   "giriş / üyelik",
	turYonetim: "yönetim paneli",
	turZafiyet: "zafiyet taraması",
	turTuzak:   "bot tuzağı",
	turDiger:   "diğer",
}

// kaynakAdlari kaynak kodunun panel karşılığı.
var kaynakAdlari = map[string]string{
	kaynakYok:    "doğrudan",
	kaynakIc:     "site içi",
	kaynakArama:  "arama motoru",
	kaynakSosyal: "sosyal medya",
	kaynakDis:    "dış bağlantı",
}

// NiyetCikar olay sayımlarından davranış yorumu üretir. Saf fonksiyon.
//
// `sayim`: tur → adet. `toplam`: olay sayısı. `engellenen`: 403 alan istek.
//
// SIRALAMA ÖNEMLİ — en kesin kanıt önce:
//
//	zafiyet > tuzak > giriş yoğunluğu > besleme toplama > arama taraması >
//	içerik kazıma > yönetim > API > normal gezinme
//
// "Zafiyet" her zaman tehdit: meşru okur /wp-login.php istemez.
// "İçerik kazıma" yalnız HACİMLE iddia edilir (5+ içerik, site içi
// kaynak yok): 5 makale okuyan gerçek okur da vardır, o yüzden seviye
// "dikkat", "tehdit" değil.
func NiyetCikar(sayim map[string]int, toplam, engellenen int) Niyet {
	if toplam <= 0 {
		return Niyet{Baslik: "Kayıt yok", Aciklama: "Bu ziyaretçi için henüz istek kaydı yok.", Seviye: "bilgi"}
	}
	yuzde := func(n int) int { return (n*100 + toplam/2) / toplam }
	// Baskın tür: en çok istek alan.
	baskin, enCok := turDiger, -1
	for t, n := range sayim {
		if n > enCok {
			baskin, enCok = t, n
		}
	}
	switch {
	case sayim[turZafiyet] > 0:
		return Niyet{
			Baslik: "Zafiyet taraması",
			Aciklama: itoa(sayim[turZafiyet]) + " istek bilinen açık tarama yoluna gitti " +
				"(wp-login, .env, phpmyadmin gibi). Meşru okur bu adresleri istemez.",
			Seviye: "tehdit", BaskinTur: turZafiyet, BaskinPay: yuzde(sayim[turZafiyet]),
		}
	case sayim[turTuzak] > 0:
		return Niyet{
			Baslik: "Bot tuzağına düştü",
			Aciklama: "Yalnız otomatik araçların izlediği gizli bağlantıya " +
				itoa(sayim[turTuzak]) + " kez istedi. İnsan oraya gitmez.",
			Seviye: "tehdit", BaskinTur: turTuzak, BaskinPay: yuzde(sayim[turTuzak]),
		}
	}

	giris := sayim[turGiris]
	if giris >= 3 && giris*100/toplam >= 20 {
		return Niyet{
			Baslik: "Kimlik denemesi",
			Aciklama: itoa(giris) + " giriş/üyelik isteği (toplamın %" +
				itoa(yuzde(giris)) + "'i). Tek okur bu kadar giriş denemez; " +
				"parola deneme (credential stuffing) olabilir.",
			Seviye: "tehdit", BaskinTur: turGiris, BaskinPay: yuzde(giris),
		}
	}
	besleme := sayim[turBesleme]
	if besleme > 0 && besleme*100/toplam >= 10 {
		return Niyet{
			Baslik: "Besleme / harita toplama",
			Aciklama: "RSS, sitemap ya da robots.txt gibi toplu uçlara " +
				itoa(besleme) + " istek. Arşiv toplayan bot ya da okuyucu beslemesi.",
			Seviye: "dikkat", BaskinTur: turBesleme, BaskinPay: yuzde(besleme),
		}
	}
	arama := sayim[turArama]
	if arama >= 3 {
		return Niyet{
			Baslik: "Site içi arama taraması",
			Aciklama: itoa(arama) + " arama isteği. Sözlük denemesi ya da " +
				"belirli bir konuyu toplu arama.",
			Seviye: "dikkat", BaskinTur: turArama, BaskinPay: yuzde(arama),
		}
	}
	if icerik := sayim[turIcerik]; icerik >= 5 && yuzde(icerik) >= 60 {
		return Niyet{
			Baslik: "İçerik kazıma (olası)",
			Aciklama: itoa(icerik) + " içerik isteği (toplamın %" + itoa(yuzde(icerik)) +
				"'i). Hızlı ve ardışıksa bot; gerçek okur da çok okur — " +
				"istek hızına ve tarayıcı kanıtlarına bakın.",
			Seviye: "dikkat", BaskinTur: turIcerik, BaskinPay: yuzde(icerik),
		}
	}
	if yonetim := sayim[turYonetim]; yonetim > 0 && yuzde(yonetim) >= 50 {
		return Niyet{
			Baslik: "Panel yoklaması",
			Aciklama: "İsteklerin çoğu yönetim paneline. Oturumsuz panel " +
				"ziyareti olağan değildir.",
			Seviye: "dikkat", BaskinTur: turYonetim, BaskinPay: yuzde(yonetim),
		}
	}
	if api := sayim[turAPI]; api > 0 && yuzde(api) >= 50 {
		return Niyet{
			Baslik:   "API istemcisi",
			Aciklama: "İsteklerin çoğu API ucuna; tarayıcı yerine program olabilir.",
			Seviye:   "bilgi", BaskinTur: turAPI, BaskinPay: yuzde(api),
		}
	}
	aciklama := "İstekler ağırlıklı olarak içerik sayfalarında (%" +
		itoa(yuzde(sayim[turIcerik])) + ")."
	if engellenen > 0 {
		aciklama += " " + itoa(engellenen) + " istek engellendi."
	}
	// Baskın tür burada kullanılıyor: yukarıdaki dalların hiçbiri eşiği
	// aşmadıysa ziyaretçi "olağan" sayılır ve en çok isteği hangi türe
	// gittiyse o gösterilir (her zaman içerik olmayabilir).
	return Niyet{
		Baslik: "Olağan gezinme", Aciklama: aciklama, Seviye: "bilgi",
		BaskinTur: baskin, BaskinPay: yuzde(enCok),
	}
}
