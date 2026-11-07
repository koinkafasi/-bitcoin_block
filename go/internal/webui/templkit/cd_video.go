package templkit

import "strconv"

// YouTube video sayfalarının veri tipleri (/videos, /videos/shorts,
// /canli-yayin, /video/<id>).
//
// Tasarım referansı: coindesk.com/videos ve coindesk.com/videos/shorts
// (ölçüldü 7 Kasım 2026). Ölçüler cd.css'te `.cd-vband*`, `.cd-vkart*`,
// `.cd-vplayer*` bloklarında; burada yalnız yapı ve sınıf adları var.

// CDVideoHero sayfa üstündeki büyük oynatıcı.
type CDVideoHero struct {
	VideoID string
	Baslik  string
	Spot    string
	Image   string
	Zaman   string
	Sure    string
	Etiket  string
	Href    string
	Canli   bool
	// SonrakiID/SonrakiBaslik oynatıcının bitiş ekranında "sıradaki video"
	// olarak gösterilir (5 sn geri sayımlı geçiş). Boşsa yalnız "tekrar
	// oynat" çıkar.
	SonrakiID     string
	SonrakiBaslik string
}

// CDVideoSatir "Sıradaki" listesinin tek satırı (küçük kapak + başlık).
type CDVideoSatir struct {
	VideoID string
	Baslik  string
	Image   string
	Sure    string
	Zaman   string
	Href    string
	Canli   bool
}

// CDVideoKutu ızgara kartı.
type CDVideoKutu struct {
	VideoID string
	Baslik  string
	Image   string
	Sure    string
	Zaman   string
	Href    string
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
	Dikey bool
	// CanliSekme canlı sayfasında oynatıcı YouTube'un canlı akışına bağlanır
	// (video kimliği yerine kanal kimliği) — böylece yayın değişse de doğru
	// akış açılır.
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

// cdVSure saniyeyi "1:02:03" biçimine çevirir (boş süre "" döner).
func cdVSure(sn int) string {
	if sn <= 0 {
		return ""
	}
	s := sn % 60
	d := (sn / 60) % 60
	sa := sn / 3600
	if sa > 0 {
		return strconv.Itoa(sa) + ":" + ikiHane(d) + ":" + ikiHane(s)
	}
	return strconv.Itoa(d) + ":" + ikiHane(s)
}

func ikiHane(n int) string {
	if n < 10 {
		return "0" + strconv.Itoa(n)
	}
	return strconv.Itoa(n)
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
