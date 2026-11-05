package httpapi

import (
	"os/exec"
	"strings"
	"sync"
)

// Baskı yazı tipi — specs/13-gazete-sayfa-duzenleyici.md Faz 6
//
// Baskı CSS'i `Georgia, 'Times New Roman', serif` istiyordu. Georgia bu
// sunucuda kurulu değil; çıktı sessizce DejaVu Serif'e düşüyordu. Yani gazete
// amaçlanan yazı tipiyle basılmıyordu ve bunu ancak PDF'in font tablosuna
// bakınca görüyorduk.
//
// Çözüm: yedeğe düşmeyi tesadüfe bırakmak yerine, sunucuda GERÇEKTEN kurulu
// olanlar arasından seçip tek bir aile bildirmek. Hangi ailenin kullanıldığı
// böylece belirli olur ve değiştiğinde test yakalar.

// gazeteSerifTercihi baskı için tercih sırası.
//
// Sıralama gazete gövde metnine göre: küçük puntoda okunaklı ve EKONOMİK
// olan önce gelir, çünkü sütun genişliği sabit ve satıra giren kelime sayısı
// doğrudan sayfaya sığan haber sayısını belirler. Times ölçülü aileler
// (Liberation Serif, Tinos) bu yüzden Noto Serif'in önünde: Noto daha geniş
// set eder, aynı blokta daha az metin alır.
//
// Sonuncular güvenli liman: hemen her Linux'ta bulunur ama gazete dizgisi
// için ideal değildir.
var gazeteSerifTercihi = []string{
	"Source Serif 4",   // ideal ama varsayılan depolarda yok
	"Liberation Serif", // Times ölçülü, ekonomik, Türkçe tam
	"Tinos",            // Liberation Serif ile aynı ölçüler
	"Times New Roman",
	"Noto Serif", // okunaklı ama geniş set eder
	"Georgia",
	"DejaVu Serif", // son çare
}

var gazeteYaziTipiBir struct {
	bir sync.Once
	ad  string
}

// gazeteKuruluAileler sistemde kurulu yazı tipi ailelerini döner.
// fc-list yoksa boş küme döner; o durumda tercih listesinin sonuncusu seçilir.
func gazeteKuruluAileler() map[string]bool {
	out := map[string]bool{}
	cmd := exec.Command("fc-list", ":", "family")
	b, err := cmd.Output()
	if err != nil {
		return out
	}
	for _, satir := range strings.Split(string(b), "\n") {
		// Bir satır virgülle ayrılmış takma adlar taşıyabilir.
		for _, ad := range strings.Split(satir, ",") {
			if ad = strings.TrimSpace(ad); ad != "" {
				out[ad] = true
			}
		}
	}
	return out
}

// gazeteBaskiYaziTipi baskıda kullanılacak aileyi seçer (bir kez).
func gazeteBaskiYaziTipi() string {
	gazeteYaziTipiBir.bir.Do(func() {
		kurulu := gazeteKuruluAileler()
		for _, ad := range gazeteSerifTercihi {
			if kurulu[ad] {
				gazeteYaziTipiBir.ad = ad
				return
			}
		}
		// Hiçbiri yoksa son çare: listenin sonuncusu. Yoksa da tarayıcının
		// genel serif'i devreye girer.
		gazeteYaziTipiBir.ad = gazeteSerifTercihi[len(gazeteSerifTercihi)-1]
	})
	return gazeteYaziTipiBir.ad
}

// gazeteYaziTipiCSS CSS font-family değeri üretir.
// Şablon kendi ailesini bildirmişse o öne alınır; arkasına seçilen aile ve
// genel serif konur, böylece bildirilen aile yoksa da okunur bir çıktı kalır.
func gazeteYaziTipiCSS(sablonAilesi string) string {
	var parcalar []string
	if ad := strings.TrimSpace(sablonAilesi); ad != "" && gazeteAileGuvenli(ad) {
		parcalar = append(parcalar, `"`+ad+`"`)
	}
	parcalar = append(parcalar, `"`+gazeteBaskiYaziTipi()+`"`, "serif")
	return strings.Join(parcalar, ", ")
}

// gazeteAileGuvenli şablondan gelen aile adının CSS'e güvenle yazılabildiğini
// denetler. Aile adı veri; tırnak veya noktalı virgül taşıyorsa kural
// gövdesinden çıkıp başka bildirim enjekte edebilirdi.
func gazeteAileGuvenli(ad string) bool {
	if len(ad) > 64 {
		return false
	}
	for _, r := range ad {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == ' ', r == '-', r == '_':
		default:
			return false
		}
	}
	return true
}
