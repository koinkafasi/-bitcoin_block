package repo

import "testing"

// Varsayılanın KAPALI olması bir güvenlik garantisidir: migration
// uygulanıp kod dağıtıldığı anda içerik kendiliğinden yayına çıkmamalı.
func TestOtomatikYayinVarsayilanKapali(t *testing.T) {
	a := OtomatikYayinVarsayilan()
	if a.Aktif {
		t.Fatal("otomatik yayın varsayılan olarak AÇIK — içerik insan onayı olmadan çıkar")
	}
	if !a.SeoZorunlu {
		t.Error("SEO zorunluluğu varsayılan kapalı; boş meta ile yayın amacın tersi")
	}
	if a.GunlukLimit <= 0 {
		t.Error("günlük limit sıfır/negatif: 429 aday bir anda yayına çıkabilir")
	}
}

// Eşik, yayındaki içeriğin güven ortalamasının (51,4) belirgin üstünde
// olmalı. Aksi hâlde "kaliteli içeriği öne çıkarma" amacı anlamını yitirir.
func TestOtomatikYayinEsigiYayinOrtalamasindanYuksek(t *testing.T) {
	const yayindakiOrtalama = 51.4 // 13 Kasım 2026 ölçümü
	a := OtomatikYayinVarsayilan()
	if a.GuvenEsigi <= yayindakiOrtalama {
		t.Fatalf("güven eşiği %.1f, yayındaki ortalama %.1f — eşik ayırt etmiyor",
			a.GuvenEsigi, yayindakiOrtalama)
	}
	// 60'a inmek 1281 taslaklık "60-69 orta" dilimini içine alırdı;
	// ölçümde kütle orada. Eşiğin o dilimin üstünde kalması gerekiyor.
	if a.GuvenEsigi < 70 {
		t.Errorf("güven eşiği %.1f < 70: 60-69 dilimi (1281 taslak) içeri girer",
			a.GuvenEsigi)
	}
}

func TestOtomatikYayinAyarSinirlari(t *testing.T) {
	// Varsayılanların kendisi geçerli aralıkta olmalı.
	a := OtomatikYayinVarsayilan()
	if a.GuvenEsigi <= 0 || a.GuvenEsigi > 100 {
		t.Errorf("güven eşiği aralık dışı: %v", a.GuvenEsigi)
	}
	if a.GunlukLimit > 200 {
		t.Errorf("günlük limit üst sınırın üstünde: %d", a.GunlukLimit)
	}
}
