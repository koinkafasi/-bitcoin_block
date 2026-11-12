package templkit

import "strconv"

// CDVideoHero sayfa üstündeki büyük oynatıcı.
type CDVideoHero struct {
	VideoID       string
	Baslik        string
	Spot          string
	Image         string
	Zaman         string
	Sure          string
	Etiket        string
	Href          string
	Canli         bool
	SonrakiID     string
	SonrakiBaslik string
	// ImageYedek poster yüklenemezse denenecek ZİNCİR (virgüllü).
	ImageYedek string
}

// CDVideoSatir "Sıradaki" listesinin tek satırı (küçük kapak + başlık).
type CDVideoSatir struct {
	VideoID string
	Baslik  string
	Image   string
	// Srcset 16:9 kapak adayları (genişlik tanımlayıcılı). Boşsa yalnız
	// Image kullanılır (shorts: tek dikey dosya).
	Srcset string
	// ImageYedek yüksek çözünürlüklü kapak 404 verirse düşülecek adres.
	ImageYedek string
	Dikey      bool
	Sure       string
	Zaman      string
	Href       string
	Canli      bool
}

// CDVideoKutu ızgara kartı.
type CDVideoKutu struct {
	VideoID string
	Baslik  string
	Image   string
	// Srcset 16:9 kapak adayları; boşsa yalnız Image (shorts).
	Srcset string
	// ImageYedek kapak 404 verirse düşülecek adres.
	ImageYedek string
	Sure       string
	Zaman      string
	Href       string
	// Dikey 9:16 kart (shorts).
	Dikey bool
	Canli bool
}

// CDVideoSekme üst gezinme sekmeleri (Videolar · Shorts · Canlı).
type CDVideoSekme struct {
	Label string
	Href  string
	Aktif bool
}

// CDVideoPage /videos, /videos/shorts ve /canli-yayin sayfası.
type CDVideoPage struct {
	Shell     CDShell
	Title     string
	Desc      string
	Canonical string
	JSONLD    string
	Baslik    string
	Aciklama  string
	Sekmeler  []CDVideoSekme
	Hero      CDVideoHero
	Siradaki  []CDVideoSatir
	Videolar  []CDVideoKutu
	Shorts    []CDVideoKutu
	Yayinlar  []CDVideoKutu
	BosMetin  string
	// Dikey hero (shorts): oynatıcı 9:16 açılır.
	Dikey   bool
	KanalID string
	// CanliYayinAktif şu an gerçekten canlı yayın var mı.
	CanliYayinAktif bool
}

// CDVideoWatch tek video sayfası (/video/<id>).
type CDVideoWatch struct {
	Shell     CDShell
	Title     string
	Desc      string
	Canonical string
	JSONLD    string
	Sekmeler  []CDVideoSekme
	Video     CDVideoHero
	Aciklama  string
	Siradaki  []CDVideoSatir
	Videolar  []CDVideoKutu
	Shorts    []CDVideoKutu
	// Dikey short videosu: oynatıcı 9:16 oranında açılır.
	Dikey bool
}

// CDCanli ana sayfa sol kolonundaki "Canlı Yayın" modülü (Son Haberler'in
// üstünde). CoinDesk'in sol ray canlı kartıyla aynı geometri.
type CDCanli struct {
	VideoID string
	Baslik  string
	Image   string
	Href    string
	Zaman   string
	Sure    string
	Canli   bool
	// ImageYedek kapak yüklenemezse denenecek ZİNCİR (virgüllü).
	ImageYedek string
}

// cdVPlayerSinif oynatıcı sınıfı (dikey varyant ayrı sınıf alır).
func cdVPlayerSinif(dikey bool) string {
	if dikey {
		return "cd-vplayer cd-vplayer--dikey"
	}
	return "cd-vplayer"
}

// cdVKartSinif ızgara kartı sınıfı.
func cdVKartSinif(dikey bool) string {
	if dikey {
		return "cd-vkart cd-vkart--dikey"
	}
	return "cd-vkart"
}

// cdCanliSinif canlı yayın varsa çubuğu kırmızı gösteren sınıfı ekler.
func cdCanliSinif(canli bool) string {
	if canli {
		return " cd-vplayer--canli"
	}
	return ""
}

// cdBool bool'u templ özniteliği için metne çevirir.
func cdBool(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

// sayiMetni panelde sayıyı metne çevirir (templ içinde fmt kullanmamak için).
func sayiMetni(n int) string { return strconv.Itoa(n) }
