package repo

import (
	"testing"

	"theblock/site/internal/quality"
)

// Bu testler B2 hatasını kalıcı olarak korur: bot yalnızca url_hash ile
// mükerrer denetimi yapıyordu. Aynı haberi MSN, Cumhuriyet ve Habertürk
// farklı URL'lerle verdiği için aynı metin 122 kez taslağa düşebiliyordu
// (ölçüm: 1.923 başlık grubunda 10.442 fazladan kayıt). Yakın-kopya
// denetimi (quality.CheckDuplicates) projede zaten vardı ama bot hattına
// bağlanmamıştı.

func TestDuplicateOfCandidatesAyniHaberiYakalar(t *testing.T) {
	cands := []quality.Candidate{{
		Ref:   "1201",
		Title: "Kara para aklama soruşturmasında üç kişiye tutuklama",
		Text:  "Kara para aklama soruşturmasında üç kişiye tutuklama İstanbul Cumhuriyet Başsavcılığı tarafından yürütülen soruşturmada üç şüpheli tutuklandı",
	}}
	bulgu, dup := DuplicateOfCandidates(
		"Kara para aklama soruşturmasında üç kişiye tutuklama",
		"İstanbul Cumhuriyet Başsavcılığı tarafından yürütülen soruşturmada üç şüpheli tutuklandı",
		cands)
	if !dup {
		t.Fatal("aynı haber mükerrer sayılmalıydı")
	}
	if bulgu.Ref != "1201" {
		t.Errorf("bulgu.Ref = %q, beklenen \"1201\"", bulgu.Ref)
	}
	if bulgu.Score < 0.6 {
		t.Errorf("benzerlik skoru = %v, eşiğin (0.60) üstünde olmalı", bulgu.Score)
	}
}

func TestDuplicateOfCandidatesFarkliHaberiGecirir(t *testing.T) {
	cands := []quality.Candidate{{
		Ref:   "1201",
		Title: "Kara para aklama soruşturmasında üç kişiye tutuklama",
		Text:  "Kara para aklama soruşturmasında üç kişiye tutuklama İstanbul Cumhuriyet Başsavcılığı tarafından yürütülen soruşturmada üç şüpheli tutuklandı",
	}}
	_, dup := DuplicateOfCandidates(
		"Merkez Bankası faiz kararını açıkladı",
		"Para Politikası Kurulu politika faizini yüzde kırk beş seviyesinde sabit tuttu",
		cands)
	if dup {
		t.Fatal("alakasız haber mükerrer sayılmamalıydı")
	}
}

func TestDuplicateOfCandidatesAdaySizKabulEder(t *testing.T) {
	if _, dup := DuplicateOfCandidates("Bir başlık", "Bir spot", nil); dup {
		t.Fatal("aday yoksa mükerrer kararı verilemez")
	}
}

// Boş başlık geldiğinde dedupe engel olmamalı: FetchRSS boş başlıklı öğeyi
// zaten atlıyor, buraya düşerse karar vermek yerine geçirmek güvenlidir.
func TestDuplicateOfCandidatesBosBaslikEngellemez(t *testing.T) {
	cands := []quality.Candidate{{Ref: "1", Title: "X", Text: "bir iki üç dört beş altı"}}
	if _, dup := DuplicateOfCandidates("", "", cands); dup {
		t.Fatal("boş başlık/spot için mükerrer kararı verilmemeli")
	}
}

// BİLİNEN BOŞLUK (silinmez — B2 ikinci aşaması).
//
// Küçük düzenleme (bot'un "kelime değiştirici" çıktısı) henüz yakalanmıyor:
// quality.ShingleSize = 5 olduğu için, başlık VE spot birlikte yeniden
// yazıldığında 5'li n-gram kümeleri örtüşmüyor ve benzerlik 0.60 eşiğinin
// altında kalıyor. Mevcut hâl aynı/çok yakın metni yakalar (asıl sorun olan
// 10.442 fazladan kaydın tamamı bu sınıftaydı), yeniden yazılmış tekrarı
// yakalamaz.
//
// Çözüm adayları: kısa metinler için ShingleSize'ı 3'e indirmek (yanlış
// pozitif riski ölçülmeli) veya başlık ve gövdeyi ayrı eşiklerle
// karşılaştırmak. Karar verilip uygulanınca bu Skip kaldırılacak.
func TestDuplicateOfCandidatesKucukDuzenlemeyiYakalar(t *testing.T) {
	t.Skip("bilinen boşluk: yeniden yazılmış başlık+spot 5'li shingle ile yakalanmıyor")
	cands := []quality.Candidate{{
		Ref:   "55",
		Title: "Fenerbahçe'nin toplam borcu belli oldu",
		Text:  "Fenerbahçe'nin toplam borcu belli oldu kulübün açıkladığı bilançoya göre toplam borç on milyar lirayı aştı",
	}}
	_, dup := DuplicateOfCandidates(
		"Fenerbahçe'nin toplam borcu açıklandı",
		"Kulübün açıkladığı bilançoya göre toplam borç on milyar lirayı aştı",
		cands)
	if !dup {
		t.Fatal("küçük düzenlemeli tekrar mükerrer sayılmalıydı")
	}
}
