package ai

import (
	"errors"
	"fmt"
	"strings"
)

// ---- Kullanılabilir metin çıkarma ----
//
// NEDEN (ölçüm, 14 Kasım 2026 — Evren'deki deepseek-v4.1-flash, gerçek
// hat isteğiyle):
//
//	finish_reason     : "length"
//	message.content   : null
//	message.reasoning : "We need answer Turkish. User asks..."
//	reasoning_tokens  : 200 / max_tokens 200
//
// "Düşünen" modeller çıktıyı message.reasoning'e yazar ve content'i boş
// bırakır. Eskiden bu hata DEĞİL, boş BAŞARI sayılıyordu: Response.Text
// boş dönüyor, hiçbir uyarı günlüğe düşmüyor ve iş birkaç adım sonra
// "AI çıktısı geçerli JSON değil" diye ölüyordu. Yani sebep sağlayıcıda,
// hata mesajı ayrıştırıcıdaydı — tanıyı yanlış yere baktıran tam bu.
//
// Aynı modelde düşünme açıkken ölçülen süre 204 sn: bizim HTTP sınırımız
// 180 sn olduğu için o yol sabitlense bile her çağrı zaman aşımına
// düşerdi. Yani bu hatanın çaresi bütçeyi büyütmek değil, düşünmeyi
// kapatmaktır (OMNIROUTE_REASONING_EFFORT=none → ölçümde 0 düşünme
// token'ı, 40-82 sn, geçerli JSON).

// ErrBosIcerik sağlayıcı HTTP 200 döndürdü ama kullanılabilir metin yok.
var ErrBosIcerik = errors.New("AI boş içerik döndürdü")

// ErrKesikIcerik çıktı max_tokens sınırında kesildi.
//
// ErrBosIcerik'ten ayrı tutulur çünkü çaresi farklı: kesik çıktıda bütçe
// artırılır, boş çıktıda düşünme kapatılır. Tek hata tipi bu ikisini
// karıştırır ve okuyanı yanlış düzeltmeye götürür.
var ErrKesikIcerik = errors.New("AI çıktısı token sınırında kesildi")

// IcerikHatasi hatanın "sağlayıcıya ulaşıldı ama çıktı kullanılamaz"
// türünden olup olmadığını söyler.
//
// Çağıran taraf bunu YENİDEN DENEMEMELİ: aynı istek aynı biçimi
// döndürür, yalnızca gecikme ikiye katlanır. (aieditor'ün JSON modu
// yedeği eskiden her hatada tekrar deniyordu ve başarısız işler tam
// 2 × sınır kadar sürüyordu.)
func IcerikHatasi(err error) bool {
	return errors.Is(err, ErrBosIcerik) || errors.Is(err, ErrKesikIcerik)
}

// metinCikar bir yanıt seçeneğinden kullanılabilir metni çıkarır; metin
// yoksa SEBEBİNİ söyleyen bir hata döndürür.
//
// kesikHataSay, kesilmiş ama dolu çıktının hata sayılıp sayılmayacağını
// belirler. Complete yolunda çıktı katı JSON'dur; kesik JSON zaten
// ayrıştırılamaz, o yüzden sebebi burada söylenir. Chat yolu etkileşimli
// düz metindir; orada kısmi yanıt hatadan iyidir, uyarı yeter.
func metinCikar(bitisSebebi, icerik, dusunme string, tamamlamaToken int, kesikHataSay bool) (string, error) {
	if strings.TrimSpace(icerik) != "" {
		if kesikHataSay && bitisSebebi == bitisUzunluk {
			return "", fmt.Errorf("%w (%d token üretildi); max_tokens'ı artırın",
				ErrKesikIcerik, tamamlamaToken)
		}
		return icerik, nil
	}
	if strings.TrimSpace(dusunme) != "" {
		return "", fmt.Errorf("%w: yanıt yalnız düşünme adımında kaldı "+
			"(reasoning %d bayt, content boş, bitiş %q). Bu bir düşünen "+
			"modeldir: OMNIROUTE_REASONING_EFFORT=none verin, ya da "+
			"max_tokens'ı düşünme payıyla birlikte artırın",
			ErrBosIcerik, len(dusunme), bitisSebebi)
	}
	return "", fmt.Errorf("%w (bitiş %q, %d token)", ErrBosIcerik, bitisSebebi, tamamlamaToken)
}

// bitisUzunluk sağlayıcının "token sınırına dayandım" bitiş sebebidir.
const bitisUzunluk = "length"

// Kesik sayılmayan ama kesilmiş çıktıyı çağıranın görebilmesi için.
func kesikMi(bitisSebebi string) bool { return bitisSebebi == bitisUzunluk }
