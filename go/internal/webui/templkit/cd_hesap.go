package templkit

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
