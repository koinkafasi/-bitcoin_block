package migrations_test

import (
	"io/fs"
	"regexp"
	"sort"
	"strings"
	"testing"

	"theblock/site/migrations"
)

// GÖÇ NUMARALARI TEKİL OLMALI.
//
// NEDEN BU TEST: 14 Kasım 2026'da iki ayrı iş kolu aynı numarayı (0132)
// üretti — biri yasal metinler, öbürü ziyaret olayları. İki dosya da
// "geçerli SQL" olduğu için derleme ve test kırılmadı; kırılan şey
// SIRAYDI: hangi göç önce koşacak belirsizleşti ve biri diğerinin
// tablosuna bağımlıysa üretimde sessizce patlardı.
//
// Kural: her numarada en çok bir .up.sql ve bir .down.sql; her up'ın
// down'ı olmalı (geri alınabilirlik iddiası boşta kalmasın).
func TestGocNumaralariTekil(t *testing.T) {
	girdiler, err := fs.ReadDir(migrations.FS, ".")
	if err != nil {
		t.Fatalf("göç dizini okunamadı: %v", err)
	}
	desen := regexp.MustCompile(`^(\d{4})_([a-z0-9_]+)\.(up|down)\.sql$`)
	up := map[string]string{}   // numara -> ad
	down := map[string]string{} // numara -> ad
	var adsiz []string
	for _, g := range girdiler {
		ad := g.Name()
		if !strings.HasSuffix(ad, ".sql") {
			continue
		}
		m := desen.FindStringSubmatch(ad)
		if m == nil {
			adsiz = append(adsiz, ad)
			continue
		}
		num, isim, yon := m[1], m[2], m[3]
		if yon == "up" {
			if onceki, var_ := up[num]; var_ {
				t.Errorf("numara %s iki kez kullanılmış: %s ve %s (biri yeniden numaralanmalı)",
					num, onceki, isim)
				continue
			}
			up[num] = isim
		} else {
			if onceki, var_ := down[num]; var_ {
				t.Errorf("numara %s iki kez kullanılmış (down): %s ve %s", num, onceki, isim)
				continue
			}
			down[num] = isim
		}
	}
	if len(adsiz) > 0 {
		t.Errorf("adlandırma kuralına uymayan dosya: %v", adsiz)
	}

	var numaralar []string
	for n := range up {
		numaralar = append(numaralar, n)
	}
	sort.Strings(numaralar)
	for _, n := range numaralar {
		if down[n] == "" {
			t.Errorf("%s_%s.up.sql için .down.sql yok", n, up[n])
		} else if down[n] != up[n] {
			// Aynı numarada farklı ad = kopyala-yapıştır artığı.
			t.Errorf("%s: up adı %q, down adı %q (aynı olmalı)", n, up[n], down[n])
		}
	}
	for n := range down {
		if up[n] == "" {
			t.Errorf("%s_%s.down.sql için .up.sql yok", n, down[n])
		}
	}
}
