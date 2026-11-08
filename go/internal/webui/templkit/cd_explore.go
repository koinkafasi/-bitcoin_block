package templkit

// "Explore more from CoinDesk" — yatay kaydırmalı panel şeridi.
//
// ÖLÇÜM (8 Kasım 2026, gerçek Chrome 1920 px, coindesk.com ana sayfa):
//
//	kapsayıcı 1464×672 (`overflow: hidden`)
//	başlık     "Explore more from CoinDesk" 220×24 · 16/24 w500
//	şerit      1464×632 · `flex gap:16px overflow-x:auto snap-x mandatory`
//	           dolgu `0 16px`, kaydırma çubuğu GİZLİ, scrollWidth 2512
//	panel      400×632 · `flex-shrink:0 snap-start`
//	panel kartı 400×632 · radius 8 · zemin beyaz · üst çizgi 1 px #d9d9d9
//	iç dolgu   16 px, `flex column`, yükseklik %100
//	başlık satırı 366×32 · gap 8 · 24×24 ikon + (16/32 w500 başlık + 16×16 ok)
//	açıklama   366×24 · 14/24 w400 · margin-top 16
//	satırlar   382×472 · `-mx-2` (8 px taşma) · gap 8 · satır 382×36 (pad 6/8)
//	alt satır  366×22 · 14/22 · margin-top 16
//
// CoinDesk'te 6 panel var: Prices · Stocks · Exchange Benchmark · Videos ·
// CoinDesk Indices · Podcast. Bizde hisse/endeks/benchmark VERİSİ YOK ve
// bülten içeriği yayında değil; uydurma panel basmıyoruz. Onun yerine
// gerçek veriyle doldurulabilen paneller üretiliyor:
//   fiyat (canlı şeritten) · video (YouTube) · klipler (shorts) ·
//   bölüm panelleri (Piyasalar · Politika · Teknoloji · Finans)
// Böylece "bölüm alanları" isteği de karşılanıyor: her bölüm kendi
// panelinde son haberlerini gösteriyor.

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
