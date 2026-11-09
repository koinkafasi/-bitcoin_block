package repo

// Türkçe tarih biçimleme — TEK KAYNAK.
//
// NEDEN VAR: Go'nun `time.Format` layout'unda ay belirteci "Jan"/"January"dır.
// `"2 Ocak 2006"` yazıldığında "Ocak" TANINMAZ ve **düz metin** olarak
// basılır: hangi ayda yayınlanırsa yayınlansın tarih "6 Ocak 2026" çıkıyordu.
// Ölçüldü (9 Kasım 2026): Kasım'da yayınlanan makale "6 Ocak 2026",
// günlük özet başlığı da "… Ocak 2026 Günlük Özeti" gösteriyordu.
//
// ÇÖZÜM: ay adları elle eşlenir. Tarih üreten her yer bu yardımcıyı
// kullanır; ikinci bir layout dizgisi yazılmaz.

import (
	"strconv"
	"time"
)

// AyAdlariTR tam ay adları (indeks 0 = Ocak).
var AyAdlariTR = [...]string{
	"Ocak", "Şubat", "Mart", "Nisan", "Mayıs", "Haziran",
	"Temmuz", "Ağustos", "Eylül", "Ekim", "Kasım", "Aralık",
}

// AyTR takvim ayının Türkçe adı (1-12; aralık dışı boş).
func AyTR(month time.Month) string {
	if month < time.January || month > time.December {
		return ""
	}
	return AyAdlariTR[int(month)-1]
}

// TarihTRUzun "6 Kasım 2026" biçiminde Türkçe tarih.
func TarihTRUzun(t time.Time) string {
	return strconv.Itoa(t.Day()) + " " + AyTR(t.Month()) + " " + strconv.Itoa(t.Year())
}

// TarihTRSaat "6 Kasım 2026 20:40 +03" biçiminde tarih + saat + dilim.
func TarihTRSaat(t time.Time) string {
	return TarihTRUzun(t) + " " + pad2Saat(t.Hour()) + ":" + pad2Saat(t.Minute()) + " " + t.Format("MST")
}

func pad2Saat(n int) string {
	if n < 10 {
		return "0" + strconv.Itoa(n)
	}
	return strconv.Itoa(n)
}
