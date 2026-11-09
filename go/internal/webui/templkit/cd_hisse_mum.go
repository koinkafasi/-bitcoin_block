package templkit

// Mum (candlestick) grafiği — hisse sayfasının "Advanced" kipi.
//
// NEDEN: CoinDesk hisse sayfasında Basic/Advanced düğmesi var (spec §10.6).
// Basic alan+çizgi, Advanced mum grafiği. Düğme aynı grafiği gösteriyorsa
// ölü kontroldür — bu yüzden mum geometrisi gerçekten üretilir.
//
// Koordinatlar CDGrafikYolu ile AYNI viewBox'ta (640×180, 6 px nefes payı):
// iki kip arasında geçerken eksen ölçeği değişmez.

// CDHisseMum tek mumun SVG geometrisi (viewBox birimi).
type CDHisseMum struct {
	X         float64 // mum merkezi
	YUst      float64 // fitil üstü (yüksek)
	YAlt      float64 // fitil altı (düşük)
	YGovdeUst float64 // gövde üstü
	YGovdeAlt float64 // gövde altı
	Genislik  float64
	Up        bool
}

// CDMumlar OHLC serisinden mum geometrisi üretir.
//
// Diziler aynı uzunlukta olmalı; eksik/sıfır değerli bar ATLANIR.
// Yüksek/düşük yoksa kapanış kullanılır (fitil sıfır uzunluk olur).
func CDMumlar(acilis, yuksek, dusuk, kapanis []float64) []CDHisseMum {
	n := len(kapanis)
	if n < 2 {
		return nil
	}
	min, maks := 0.0, 0.0
	var ok bool
	for i := 0; i < n; i++ {
		if kapanis[i] <= 0 {
			continue
		}
		lo, hi := kapanis[i], kapanis[i]
		if i < len(dusuk) && dusuk[i] > 0 {
			lo = dusuk[i]
		}
		if i < len(yuksek) && yuksek[i] > 0 {
			hi = yuksek[i]
		}
		if !ok || lo < min {
			min = lo
		}
		if !ok || hi > maks {
			maks = hi
		}
		ok = true
	}
	if !ok {
		return nil
	}
	const pad = 6.0
	aralik := maks - min
	if aralik <= 0 {
		aralik = 1
	}
	y := func(v float64) float64 { return pad + (1-(v-min)/aralik)*(GrafikH-2*pad) }

	adim := GrafikW / float64(n)
	gen := adim * 0.6
	if gen < 1 {
		gen = 1
	}
	out := make([]CDHisseMum, 0, n)
	for i := 0; i < n; i++ {
		if kapanis[i] <= 0 {
			continue
		}
		ac := kapanis[i]
		if i < len(acilis) && acilis[i] > 0 {
			ac = acilis[i]
		}
		hi, lo := kapanis[i], kapanis[i]
		if i < len(yuksek) && yuksek[i] > 0 {
			hi = yuksek[i]
		}
		if i < len(dusuk) && dusuk[i] > 0 {
			lo = dusuk[i]
		}
		gu, ga := y(ac), y(kapanis[i])
		if gu > ga {
			gu, ga = ga, gu
		}
		out = append(out, CDHisseMum{
			X:         float64(i)*adim + adim/2,
			YUst:      y(hi),
			YAlt:      y(lo),
			YGovdeUst: gu,
			YGovdeAlt: ga,
			Genislik:  gen,
			Up:        kapanis[i] >= ac,
		})
	}
	return out
}
