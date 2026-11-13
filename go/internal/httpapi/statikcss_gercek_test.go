package httpapi

import (
	"io/fs"
	"strings"
	"testing"

	"theblock/site/internal/webui"
)

// TestCSSYorumlariSilGercekDosyalar: depodaki GERÇEK CSS'lerde dönüşüm
// kuralları bozmamalı.
//
// Kontrol: yorumlar silindikten sonra
//   - çıktı, girdinin BİREBİR ALT DİZİSİ olmalı (yalnız bayt atılır, hiçbir
//     bayt değiştirilmez/eklenmez — bu, tüm yapıyı tek hamlede kanıtlar),
//   - çıktıda `/*` kalmamalı,
//   - dosya küçülmeli.
//
// NOT: eski sürümde `{`/`;` sayıyordum, YANLIŞTI: yorum METNİNİN içindeki
// süslü parantez ve noktalı virgüller de sayılıyor ("ölçüldü: {x} → ..."),
// yani sayaç yorum silinince doğal olarak düşüyordu. Alt dizi kontrolü hem
// daha güçlü hem yanıltıcı değil.
func TestCSSYorumlariSilGercekDosyalar(t *testing.T) {
	staticFS, err := fs.Sub(webui.FS, "static")
	if err != nil {
		t.Fatalf("static FS: %v", err)
	}
	sayac := 0
	err = fs.WalkDir(staticFS, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(strings.ToLower(p), ".css") {
			return err
		}
		ham, err := fs.ReadFile(staticFS, p)
		if err != nil {
			return err
		}
		temiz := cssYorumlariSil(ham)
		if !altDiziMi(ham, temiz) {
			t.Errorf("%s: çıktı girdinin alt dizisi DEĞİL (bayt değişmiş/eklenmiş)", p)
		}
		if strings.Contains(string(temiz), "/*") {
			t.Errorf("%s: yorum kaldı", p)
		}
		if len(temiz) > len(ham) {
			t.Errorf("%s: çıktı girdiden BÜYÜK (%d > %d)", p, len(temiz), len(ham))
		}
		sayac++
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	if sayac < 10 {
		t.Fatalf("yalnız %d CSS dosyası tarandı (beklenen daha fazla)", sayac)
	}
	t.Logf("%d CSS dosyası doğrulandı", sayac)
}

// altDiziMi alt, buy'un elemanlarını SIRAYLA içeriyor mu.
func altDiziMi(buy, alt []byte) bool {
	if len(alt) > len(buy) {
		return false
	}
	i := 0
	for _, b := range buy {
		if i < len(alt) && b == alt[i] {
			i++
		}
	}
	return i == len(alt)
}

// TestCSSYorumlariSilIdempotent: iki kez uygulamak aynı sonucu vermeli
// (önbelleğe yazılan çıktı yeniden işlenirse bozulmasın).
func TestCSSYorumlariSilIdempotent(t *testing.T) {
	ham := []byte(".a{color:red}/* x */.b{content:\"/*\";}")
	bir := cssYorumlariSil(ham)
	iki := cssYorumlariSil(bir)
	if string(bir) != string(iki) {
		t.Fatalf("idempotent değil:\n%q\n%q", bir, iki)
	}
}
