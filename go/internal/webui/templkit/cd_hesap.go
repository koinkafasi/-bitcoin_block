package templkit

// Hesap merkezi — YENİ (CoinDesk) tasarım.
//
// NEDEN: /hesabim ve sekmeleri hâlâ theblock şablonunu (kırmızı üst çubuk,
// Poppins, kırmızı alt bilgi, 12 px köşe) kullanıyordu. Sitenin geri kalanı
// CoinDesk tasarımına geçtiği için bu sayfalar "başka site" gibi duruyordu
// (ölçüldü, 7 Kasım 2026: gövde yazı tipi Poppins, 179 elemanda).
//
// ÖLÇÜ KAYNAĞI: CoinDesk hesap alanı Auth0 oturumu istiyor
// (auth.coindesk.com/u/login — girişsiz erişilemiyor). Bu yüzden geometri
// uydurulmadı; sitenin ÖLÇÜLMÜŞ tasarım jetonları kullanıldı:
//   - kapsayıcı 1512/1432, kart köşesi 4 px, çizgi #d9d9d9
//   - H1 serif 36/44 w700, gövde 16/24, etiket 11/16 büyük harf #676767
//   - birincil düğme siyah zemin/beyaz metin; marka sarısı #ffd200 vurgu
//   - koyu temada zemin #161616, kart zemini saydam + çizgi

// CDHesapNav hesap kenar menüsü satırı.
type CDHesapNav struct {
	Etiket string
	Href   string
	Aktif  bool
	Sayi   int
	// Yeni sayı (bildirim) — sarı nokta.
	Yeni bool
}

// CDHesapKutu özet sayı kutusu (Kaydedilen · Yorum · Puan · Seri).
type CDHesapKutu struct {
	Sayi   string
	Etiket string
	Href   string
}

// CDHesapKisayol hub sayfasındaki hızlı erişim kartı.
type CDHesapKisayol struct {
	Baslik   string
	Aciklama string
	Href     string
}

// CDSosyalAlan profil formundaki sosyal medya satırı.
type CDSosyalAlan struct {
	Anahtar   string
	Etiket    string
	YerTutucu string
	Deger     string
}

// CDAvatarSec profil fotoğrafı seçeneği.
type CDAvatarSec struct {
	ID     string
	URL    string
	Baslik string
	Secili bool
}

// CDHesapYorum "Yorumlarım" satırı.
type CDHesapYorum struct {
	Baslik string
	Href   string
	Tarih  string
	Metin  string
	Durum  string
	Begeni int
}

// CDHesapKayit "Kaydedilenler" kartı.
type CDHesapKayit struct {
	Baslik string
	Spot   string
	Href   string
	Gorsel string
	Zaman  string
}

// CDHesapTakip takip edilen kategori/etiket/yazar.
type CDHesapTakip struct {
	Tur    string // kategori | etiket | yazar
	Etiket string
	Href   string
	Slug   string
}

// CDHesapBildirim bildirim satırı.
type CDHesapBildirim struct {
	Tur      string
	Baslik   string
	Metin    string
	Href     string
	Zaman    string
	Okunmadi bool
}

// CDHesapRozet başarım rozeti.
type CDHesapRozet struct {
	Ad        string
	Aciklama  string
	Kazanildi bool
}

// CDHesapLider liderlik tablosu satırı.
type CDHesapLider struct {
	Sira int
	Ad   string
	Puan int
	Seri int
	Ben  bool
}

// CDHesapSayfa /hesabim ailesinin tüm sekmeleri.
type CDHesapSayfa struct {
	Shell     CDShell
	Title     string
	Desc      string
	Canonical string
	JSONLD    string
	// Sekme: hub | profil | abonelik | yorumlar | takip | kaydedilenler |
	// bildirimler | rozetler
	Sekme string

	Nav []CDHesapNav

	// Üye kimliği
	Ad               string
	Eposta           string
	BasHarfler       string
	AvatarURL        string
	AvatarID         string
	Rol              string
	EpostaDogrulandi bool

	// Plan
	PlanAd       string // "Premium Üye" | "Ücretsiz Üye"
	PlanAciklama string
	PlanBitis    string
	Premium      bool

	// Hub
	Kutular    []CDHesapKutu
	Kisayollar []CDHesapKisayol
	UyariMetin string

	// Profil
	Slug      string
	Unvan     string
	Biyografi string
	Sosyal    []CDSosyalAlan
	Avatarlar []CDAvatarSec

	// Listeler
	Yorumlar      []CDHesapYorum
	Kaydedilenler []CDHesapKayit
	Takip         []CDHesapTakip
	Bildirimler   []CDHesapBildirim
	Rozetler      []CDHesapRozet
	Liderlik      []CDHesapLider
	Puan          int
	Seri          int
	BosMetin      string

	// Abonelik
	AbonelikVar   bool
	AbonelikAd    string
	AbonelikRenk  string
	AbonelikBitis string
	AbonelikDongu string
	IptalTarih    string
	OdemeAcik     bool
	OdemeEtiket   string

	// Geri bildirim
	FlashTur   string // ok | hata
	FlashMetin string
}

// cdHesapOnayRozet e-posta doğrulama etiketi metni.
func cdHesapOnayRozet(dogrulandi bool) string {
	if dogrulandi {
		return "Doğrulandı"
	}
	return "Doğrulanmadı"
}
