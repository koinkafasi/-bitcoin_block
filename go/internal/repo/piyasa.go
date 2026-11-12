package repo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"theblock/site/internal/piyasa"
)

const (
	piyasaCacheKey    = "ana"
	piyasaMergeSuresi = 45 * time.Minute
)

type PiyasaHatasi struct {
	Kaynak string
	Kalem  int
	Neden  error
}

func (e *PiyasaHatasi) Error() string {
	return fmt.Sprintf("piyasa: %s kaynağı atlandı (%d kalem alındı): %v", e.Kaynak, e.Kalem, e.Neden)
}

func RefreshPiyasa(ctx context.Context, pool DB) error {
	yeni, err := piyasa.Get(ctx)
	if err != nil {
		// Hiç veri yok: mevcut önbelleğe dokunma.
		return err
	}
	eski := PiyasaOf(ctx, pool)
	birlesik := mergePiyasa(eski, yeni, time.Now())
	b, merr := json.Marshal(birlesik)
	if merr != nil {
		return merr
	}
	if err := CacheSet(ctx, pool, "piyasa", piyasaCacheKey, b); err != nil {
		return err
	}
	if beklenen := piyasa.KaynakliTanimSayisi(); len(birlesik.Kalemler) < beklenen {
		return &PiyasaHatasi{
			Kaynak: yeni.Kaynak,
			Kalem:  len(yeni.Kalemler),
			Neden:  errors.New("bazı göstergeler alınamadı"),
		}
	}
	return nil
}

func mergePiyasa(eski, yeni piyasa.Sonuc, simdi time.Time) piyasa.Sonuc {
	if len(eski.Kalemler) == 0 || eski.Guncel.IsZero() {
		return yeni
	}
	if simdi.Sub(eski.Guncel) > piyasaMergeSuresi {
		return yeni
	}
	varolan := make(map[string]bool, len(yeni.Kalemler))
	for _, k := range yeni.Kalemler {
		varolan[k.Anahtar] = true
	}
	if len(varolan) == len(piyasa.Tanimlar) {
		return yeni
	}
	// Sıra korunur: tanım sırasına göre yürü, yeni sonuçta varsa onu kullan,
	// yoksa eski önbellekten al.
	eskiMap := make(map[string]piyasa.Kalem, len(eski.Kalemler))
	for _, k := range eski.Kalemler {
		eskiMap[k.Anahtar] = k
	}
	out := make([]piyasa.Kalem, 0, len(piyasa.Tanimlar))
	for _, t := range piyasa.Tanimlar {
		if varolan[t.Anahtar] {
			for _, k := range yeni.Kalemler {
				if k.Anahtar == t.Anahtar {
					out = append(out, k)
					break
				}
			}
			continue
		}
		if k, ok := eskiMap[t.Anahtar]; ok {
			out = append(out, k)
		}
	}
	yeni.Kalemler = out
	if yeni.Kaynak == "" {
		yeni.Kaynak = eski.Kaynak
	} else if eski.Kaynak != "" {
		yeni.Kaynak += "+onbellek"
	}
	return yeni
}

func PiyasaOf(ctx context.Context, pool DB) piyasa.Sonuc {
	b, err := CacheGet(ctx, pool, "piyasa", piyasaCacheKey)
	if err != nil || len(b) == 0 {
		return piyasa.Sonuc{}
	}
	var s piyasa.Sonuc
	if json.Unmarshal(b, &s) != nil {
		return piyasa.Sonuc{}
	}
	return s
}

// PiyasaBayat önbelleğin yaşını döndürür (teşhis/panel için).
func PiyasaBayat(ctx context.Context, pool DB) time.Duration {
	s := PiyasaOf(ctx, pool)
	if s.Guncel.IsZero() {
		return -1
	}
	return time.Since(s.Guncel)
}
