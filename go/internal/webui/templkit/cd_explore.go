package templkit

// CDExplorePanel şeritteki tek panel.
type CDExplorePanel struct {
	// Tur: "fiyat" | "video" | "klip" | "haber"
	Tur    string
	Baslik string
	Href   string
	// Aciklama panelin üst başlığının altındaki tek satır (14/24).
	Aciklama string

	// fiyat paneli
	Fiyatlar []CDPriceRow

	// video / klip paneli
	Videolar []CDVideo
	Klipler  []CDKlip

	// haber paneli (bölüm)
	Haberler []CDStory

	// Alt satır: "Daha fazla →" bağlantısı.
	AltMetin  string
	AltHref   string
	AltBaglam string
}

// CDExplore ana sayfa şeridi.
type CDExplore struct {
	Baslik   string
	Paneller []CDExplorePanel
}

// cdExploreVar en az bir panel dolu mu (boşsa modül basılmaz).
func cdExploreVar(e CDExplore) bool {
	for _, p := range e.Paneller {
		if len(p.Fiyatlar) > 0 || len(p.Videolar) > 0 ||
			len(p.Klipler) > 0 || len(p.Haberler) > 0 {
			return true
		}
	}
	return false
}
