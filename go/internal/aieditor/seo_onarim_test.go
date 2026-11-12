package aieditor

import "testing"

// Onarım kuyruğu SeoBaslikSertMax'ı eşik alır. O sabit değişirse kuyruk
// tanımı da değişmeli; bu test ikisinin ayrışmasını yakalar.
func TestSeoOnarimEsigiSertMaxIleAyniKalmali(t *testing.T) {
	if SeoBaslikSertMax <= SeoBaslikMax {
		t.Fatalf("sert max (%d) ideal max (%d) ile aynı veya küçük: onarım kuyruğu "+
			"ideal aralıktaki başlıkları da içine alır", SeoBaslikSertMax, SeoBaslikMax)
	}
	// Ölçüm (13 Kasım 2026): 70+ karakterli 788 başlık var, 61-70 arası 703.
	// Eşik 70 iken kuyruk 788 içerikle başlar — bu bilinçli: 61-70 arası
	// kısmen görünür, 70+ ise belirgin kesiliyor.
	if SeoBaslikSertMax != 70 {
		t.Errorf("sert max %d; ölçüm 70 varsayımıyla yapıldı (788 içerik), "+
			"değiştiyse kuyruk boyutu yeniden ölçülmeli", SeoBaslikSertMax)
	}
}

// Günlük limit, yayındaki meta değişikliğinin etkisini ölçülebilir
// tutmak için var. Sıfır veya aşırı yüksek olmamalı.
func TestSeoOnarimGunlukLimitMakul(t *testing.T) {
	if SeoOnarimGunlukLimit <= 0 {
		t.Fatal("günlük limit sıfır: onarım hiç çalışmaz")
	}
	if SeoOnarimGunlukLimit > 100 {
		t.Errorf("günlük limit %d: 1491 içerik iki haftada değişir, "+
			"Search Console'da etki ayrıştırılamaz", SeoOnarimGunlukLimit)
	}
	if SeoOnarimTurLimiti > SeoOnarimGunlukLimit {
		t.Errorf("tur limiti (%d) günlük limitten (%d) büyük: tek tur günü bitirir",
			SeoOnarimTurLimiti, SeoOnarimGunlukLimit)
	}
}
