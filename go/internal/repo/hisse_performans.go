package repo

// Hisse performans satırı (CoinDesk §10.6 sağ kolon).
//
// Değerler SEÇİLEN aralığın serisinden değil, tek bir 1 yıllık günlük
// seriden hesaplanır: aynı seriden 24 saat/5 gün/1 ay/YTD/1 yıl yüzdeleri
// türetilir. Böylece sayfa aralık değiştikçe farklı yüzdeler göstermez
// (CoinDesk'te de satır sabittir).

import "time"

// HisseDonem tek performans dönemi.
type HisseDonem struct {
	Ad      string  // "24 hours"
	Yuzde   float64 // +1.24 gibi
	Gecerli bool    // seri dönemi KAPSAMIYORSA false
}

// hissePerformansTanim dönem tanımı + kapsama toleransı.
//
// TOLERANS NEDEN: seri istenen tarihten çok sonra başlıyorsa (yeni halka
// arz, veri kesintisi) o dönemin yüzdesi YANLIŞ olur — 2 günlük veriden
// "1 yıl" yüzdesi hesaplanmaz. Tolerans, normal hafta sonu/tatil
// boşluklarını hoş görür ama gerçek kapsama eksiğini yakalar.
type hissePerformansTanim struct {
	ad        string
	geriye    time.Duration // sabit geriye bakış (0 = takvim hesabı)
	takvimAy  int           // ay geri (1 month)
	takvimYil int           // yıl geri (1 year)
	ytd       bool          // yıl başından itibaren
	tolerans  time.Duration
}

// HissePerformans 1 yıllık seriden dönem yüzdelerini hesaplar.
//
// `onceki` önceki kapanış (24 saat için), `simdi` hesabın "şimdi"si.
func HissePerformans(seri []HisseNokta, onceki float64, simdi time.Time) []HisseDonem {
	if len(seri) == 0 {
		return nil
	}
	son := seri[len(seri)-1].C
	baslangicZaman := time.Unix(seri[0].T, 0).UTC()

	tanim := []hissePerformansTanim{
		{ad: "24 hours", geriye: 24 * time.Hour, tolerans: 24 * time.Hour},
		{ad: "5 days", geriye: 5 * 24 * time.Hour, tolerans: 48 * time.Hour},
		{ad: "1 month", takvimAy: 1, tolerans: 7 * 24 * time.Hour},
		{ad: "YTD", ytd: true, tolerans: 7 * 24 * time.Hour},
		{ad: "1 year", takvimYil: 1, tolerans: 14 * 24 * time.Hour},
	}

	out := make([]HisseDonem, 0, len(tanim))
	for _, t := range tanim {
		var esik time.Time
		switch {
		case t.ytd:
			esik = time.Date(simdi.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
		case t.takvimAy != 0:
			esik = simdi.AddDate(0, -t.takvimAy, 0)
		case t.takvimYil != 0:
			esik = simdi.AddDate(-t.takvimYil, 0, 0)
		default:
			esik = simdi.Add(-t.geriye)
		}

		d := HisseDonem{Ad: t.ad}
		// Seri dönem başlangıcını kapsamıyorsa yüzde hesaplanmaz.
		if baslangicZaman.After(esik.Add(t.tolerans)) {
			out = append(out, d)
			continue
		}

		bas := 0.0
		if t.ad == "24 hours" && onceki > 0 {
			bas = onceki
		} else {
			for _, n := range seri {
				if !time.Unix(n.T, 0).UTC().Before(esik) {
					bas = n.C
					break
				}
			}
		}
		if bas > 0 && son > 0 {
			d.Yuzde = (son - bas) / bas * 100
			d.Gecerli = true
		}
		out = append(out, d)
	}
	return out
}
