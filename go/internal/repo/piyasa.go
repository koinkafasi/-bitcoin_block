package repo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"theblock/site/internal/piyasa"
)

// Piyasa göstergeleri (döviz, altın, borsa, emtia) önbellek köprüsü.
//
// Neden repo: sayfa üretimi veritabanına bağlı kalmalı. Piyasa uçları 10
// saniyeye kadar yavaş olabilir; istek yolunda çağırmak sayfayı bekletir.
// Değerler scheduler tarafından `services_cache` kind=piyasa'ya yazılır,
// sayfa yalnız okur.

const (
	piyasaCacheKey = "ana"
	// piyasaMergeSuresi eski önbellekten kalem taşıma penceresi.
	//
	// Neden gerekli: iki kaynak var (truncgil + Yahoo). truncgil bir turda
	// yanıt vermezse sonuç 11 kalemden 4'üne düşüyor ve önbelleği EZİYORDU
	// (28 Ekim 2026'da tam bu oldu: döviz/altın kalemleri sayfadan kayboldu).
	// Taze önbellekte olup yeni sonuçta olmayan kalem korunur.
	piyasaMergeSuresi = 45 * time.Minute
)

// PiyasaHatasi birincil kaynağın neden atlandığını çağırana taşır.
//
// Ayrı tip: kısmi veri yine de yazılır, ama scheduler günlüğe nedeni yazar.
// Sessiz küçülme bir daha yaşanmasın.
type PiyasaHatasi struct {
	Kaynak string
	Kalem  int
	Neden  error
}

func (e *PiyasaHatasi) Error() string {
	return fmt.Sprintf("piyasa: %s kaynağı atlandı (%d kalem alındı): %v", e.Kaynak, e.Kalem, e.Neden)
}

// RefreshPiyasa göstergeleri kaynaklardan çeker ve önbelleğe yazar.
//
// Davranış:
//   - Tüm kaynaklar çalışırsa: yeni sonuç yazılır.
//   - Bir kaynak düşerse: yeni sonuç, taze eski önbellekle BİRLEŞTİRİLİR
//     (eksik kalemler eskisinden taşınır) ve hata döner — böylece scheduler
//     günlüğe nedeni yazar ama sayfa eksik kalmaz.
//   - Hiçbir kaynak çalışmazsa: önbellek korunur, hata döner.
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
	// Kısmi sonuç: yalnız kaynağı TANIMLI göstergeler beklenir. Kaynağı
	// olmayan kalem (ör. 2 yıllık tahvil — ücretsiz güvenilir uç yok)
	// "eksik" sayılmamalı, yoksa her turda yanlış alarm üretir.
	if beklenen := piyasa.KaynakliTanimSayisi(); len(birlesik.Kalemler) < beklenen {
		return &PiyasaHatasi{
			Kaynak: yeni.Kaynak,
			Kalem:  len(yeni.Kalemler),
			Neden:  errors.New("bazı göstergeler alınamadı"),
		}
	}
	return nil
}

// mergePiyasa yeni sonuçta olmayan ama eski önbellekte bulunan kalemleri
// taşır. Eski önbellek `maxYas`tan daha eski ise taşıma yapılmaz: bayat
// fiyat göstermek, kalemi hiç göstermemekten daha yanıltıcı olabilir.
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

// PiyasaOf önbellekteki göstergeleri döndürür.
//
// Önbellek boşsa boş Sonuc döner (hata değil): sayfa modülü o zaman hiç
// basılmaz, kırık bir "0,00" satırı göstermekten iyidir.
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
