package templkit

// CDModul ana sayfanın tek bölümü.
type CDModul struct {
	// Tip: cd_hero · cd_analiz · cd_nabiz · cd_explore · cd_ikili · cd_bloklar
	Tip string
	// Goster false ise modül hiç basılmaz (tasarımcıdaki görünürlük anahtarı).
	Goster bool
	// Adet: bazı modüllerde içerik sınırı (0 = modülün kendi varsayılanı).
	Adet int
}

var CDModulTipleri = []string{
	"cd_hero", "cd_analiz", "cd_nabiz", "cd_explore", "cd_ikili", "cd_bloklar",
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
