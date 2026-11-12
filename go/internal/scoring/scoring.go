package scoring

import (
	"strconv"
	"strings"
)

// Input puanlamaya giren sinyallerdir. Etiket, kapak ve kaynak sinyalleri
// çağıran tarafta veritabanından okunur; bu paket veritabanı bilmez.
type Input struct {
	Title     string
	Spot      string
	TagCount  int
	HasCover  bool
	HasSource bool
}

// Sonuç notu tek yerde tutulur ki panel ile bot aynı açıklamayı göstersin.
const Note = "yerel sezgisel (başlık/spot/etiket/görsel/kaynak sinyalleri)"

// Signal puana giren tek bir kalemdir. Panelde gerekçe satırı olarak
// gösterilir: editör sayının nereden geldiğini görmeden eşiği yorumlayamaz.
type Signal struct {
	Ad    string `json:"ad"`
	Puan  int    `json:"puan"`  // bu kalemin katkısı (negatif olabilir)
	Enb   int    `json:"enb"`   // bu kalemden alınabilecek en yüksek katkı
	Durum string `json:"durum"` // iyi | orta | eksik | bilgi
	Ipucu string `json:"ipucu"` // ne yapılırsa artar; boşsa yapılacak bir şey yok
}

// Durum değerleri.
const (
	DurumIyi   = "iyi"
	DurumOrta  = "orta"
	DurumEksik = "eksik"
	DurumBilgi = "bilgi" // editörün eyleme çeviremeyeceği kalem
)

// Taban puan: hiçbir sinyal yokken bile içerik sıfırdan başlamaz.
const taban = 20

func Explain(in Input) (int, []Signal) {
	sig := make([]Signal, 0, 7)
	// ham sıfırdan başlar; taban puanı da bir kalem olarak eklenir, böylece
	// kalemlerin toplamı her zaman ham puana eşit kalır (testle sabitlendi).
	ham := 0
	ekle := func(s Signal) {
		ham += s.Puan
		sig = append(sig, s)
	}

	ekle(Signal{Ad: "Taban", Puan: taban, Enb: taban, Durum: DurumBilgi,
		Ipucu: ""})

	// Başlık uzunluğu: ideal 40–90 karakter (manşet ölçüsü).
	tl := len([]rune(strings.TrimSpace(in.Title)))
	switch {
	case tl >= 40 && tl <= 90:
		ekle(Signal{Ad: "Başlık uzunluğu", Puan: 20, Enb: 20, Durum: DurumIyi})
	case tl >= 25 && tl <= 120:
		ekle(Signal{Ad: "Başlık uzunluğu", Puan: 12, Enb: 20, Durum: DurumOrta,
			Ipucu: "Başlığı 40–90 karaktere getir (şu an " + strconv.Itoa(tl) + ")."})
	case tl < 10:
		ekle(Signal{Ad: "Başlık uzunluğu", Puan: -10, Enb: 20, Durum: DurumEksik,
			Ipucu: "Başlık çok kısa (" + strconv.Itoa(tl) + " karakter); 40–90 arası olmalı."})
	default:
		ekle(Signal{Ad: "Başlık uzunluğu", Puan: 0, Enb: 20, Durum: DurumEksik,
			Ipucu: "Başlık 40–90 karakter aralığında değil (şu an " + strconv.Itoa(tl) + ")."})
	}

	// Spot doluluğu: arama motoru ve manşet özeti için gerekli.
	sl := len([]rune(strings.TrimSpace(in.Spot)))
	switch {
	case sl >= 120:
		ekle(Signal{Ad: "Spot uzunluğu", Puan: 25, Enb: 25, Durum: DurumIyi})
	case sl >= 60:
		ekle(Signal{Ad: "Spot uzunluğu", Puan: 15, Enb: 25, Durum: DurumOrta,
			Ipucu: "Spotu 120 karaktere çıkar (şu an " + strconv.Itoa(sl) + ")."})
	case sl == 0:
		ekle(Signal{Ad: "Spot uzunluğu", Puan: -10, Enb: 25, Durum: DurumEksik,
			Ipucu: "Spot boş; en az 120 karakterlik özet yaz."})
	default:
		ekle(Signal{Ad: "Spot uzunluğu", Puan: 0, Enb: 25, Durum: DurumEksik,
			Ipucu: "Spot çok kısa (" + strconv.Itoa(sl) + "); 120 karaktere çıkar."})
	}

	// Etiket sinyali: konu sınıflandırmasının göstergesi.
	switch {
	case in.TagCount >= 3:
		ekle(Signal{Ad: "Etiketler", Puan: 15, Enb: 15, Durum: DurumIyi})
	case in.TagCount >= 1:
		ekle(Signal{Ad: "Etiketler", Puan: 8, Enb: 15, Durum: DurumOrta,
			Ipucu: "En az 3 etiket ekle (şu an " + strconv.Itoa(in.TagCount) + ")."})
	default:
		ekle(Signal{Ad: "Etiketler", Puan: 0, Enb: 15, Durum: DurumEksik,
			Ipucu: "Hiç etiket yok; en az 3 ekle."})
	}

	if in.HasCover {
		ekle(Signal{Ad: "Kapak görseli", Puan: 10, Enb: 10, Durum: DurumIyi})
	} else {
		ekle(Signal{Ad: "Kapak görseli", Puan: 0, Enb: 10, Durum: DurumEksik,
			Ipucu: "Kapak görseli seç."})
	}

	if in.HasSource {
		ekle(Signal{Ad: "Kaynak bağlantısı", Puan: 10, Enb: 10, Durum: DurumIyi})
	} else {
		ekle(Signal{Ad: "Kaynak bağlantısı", Puan: 0, Enb: 10, Durum: DurumEksik,
			Ipucu: "Doğrulanabilirlik için kaynak bağlantısı ekle."})
	}

	kd := KeywordDensity(in.Title)
	ekle(Signal{Ad: "Başlıkta tekrar eden sözcük", Puan: kd, Enb: 15,
		Durum: DurumBilgi})

	return clamp(ham), sig
}

func Relevance(in Input) (int, string) {
	score, _ := Explain(in)
	return score, Note
}

func clamp(v int) int {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}

// KeywordDensity başlıktaki tekrar eden anlamlı sözcükleri ödüllendirir (0–15).
// Tekrar, konunun başlıkta gerçekten merkezde olduğunun kaba göstergesidir.
func KeywordDensity(title string) int {
	counts := map[string]int{}
	for _, w := range strings.Fields(strings.ToLower(title)) {
		w = strings.Trim(w, ".,:;!?'\"()[]")
		if len([]rune(w)) < 4 {
			continue
		}
		counts[w]++
	}
	dup := 0
	for _, n := range counts {
		if n > 1 {
			dup++
		}
	}
	return dup * 5
}
