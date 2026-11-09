package templkit

// YENİ TASARIMIN MODÜLER YÖNETİMİ (9 Kasım 2026)
//
// SORUN: ana sayfanın yeni (CoinDesk) tasarımı `cd_home.templ` içinde SABİT
// bir sırayla yazılıydı: hero → nabız → explore → görüş/en çok okunan →
// bölüm blokları. Sayfa tasarımcısı (builder) ise yalnız theblock'tan kalan
// `site_public.html` bloklarını yönetiyordu; yeni tasarımın hiçbir alanı
// panelden açılıp kapatılamıyor, sıralanamıyordu.
//
// ÇÖZÜM: yeni tasarımın her alanı bir `CDModul` kaydı. Şablon bu listeyi
// SIRAYLA basar; liste boşsa ESKİ SABİT SIRA aynen uygulanır (geriye dönük
// uyum: mevcut düzen kayıtları bozulmaz, düzen yazılmamış sitelerde görünüm
// birebir aynı kalır).
//
// Veri tarafı DEĞİŞMEDİ: her modül `CDHomeView` alanlarını kullanır, yani
// panelden bir modülü kaldırmak yalnız o alanın BASILMAMASI demek — yeni
// sorgu yolu yok, uydurma veri yok.

// CDModul ana sayfanın tek bölümü.
type CDModul struct {
	// Tip: cd_hero · cd_nabiz · cd_explore · cd_ikili · cd_bloklar
	Tip string
	// Goster false ise modül hiç basılmaz (tasarımcıdaki görünürlük anahtarı).
	Goster bool
	// Adet: bazı modüllerde içerik sınırı (0 = modülün kendi varsayılanı).
	Adet int
}

// CDModulTipleri tanınan modül kimlikleri — tasarımcı paleti ve testler
// aynı listeyi kullanır (yazım kayması sessizce modül düşürürdü).
//
// `cd_ticker` BİLEREK YOK: fiyat şeridi kabuğun parçası (üst çubuk gibi
// her sayfada) ve sayfa tasarımcısı sayfa gövdesini düzenler. Şeridi
// buraya eklemek, kaldırıldığında tüm sayfalarda kaybolan bir anahtar
// vaat edip yalnız ana sayfada etkili olmak olurdu.
var CDModulTipleri = []string{
	"cd_hero", "cd_nabiz", "cd_explore", "cd_ikili", "cd_bloklar",
}

// CDModulGecerli tip kimliği tanınıyor mu.
func CDModulGecerli(t string) bool {
	for _, x := range CDModulTipleri {
		if x == t {
			return true
		}
	}
	return false
}
