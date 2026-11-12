package templkit

// CDVarsayilanOG görseli olmayan sayfalarda kullanılan markalı OG kartı.
// httpapi tarafında `/og-varsayilan.png` olarak üretilir ve sunulur
// (bkz. cd_og_gorsel.go). Sosyal botlar 1200×630 bekler.
const CDVarsayilanOG = "/og-varsayilan.png"

// cdOGGorsel og:image için kullanılacak MUTLAK adresi seçer.
//
// Open Graph mutlak URL ister: göreli `/og-varsayilan.png` yazan kartlar
// Facebook/X tarafından yok sayılır. Bu yüzden taban adresle birleştirilir.
//
// Görseli olmayan sayfa BOŞ bırakılmaz: `twitter:card=summary_large_image`
// görselsiz kalınca paylaşım boş kart olarak çıkıyordu. Marka kartına
// düşmek "görsel yok" demekten iyidir.
func cdOGGorsel(h CDHead) string {
	if h.Image != "" {
		return cdMutlak(h.Image, h.BaseURL)
	}
	return cdMutlak(CDVarsayilanOG, h.BaseURL)
}

// cdOGGorselVar görsel var mı (alt metin/ölçü etiketleri buna göre basılır).
//
// NOT: OG görsel işi sürüyor (opencode); şablona henüz bağlanmadı.
func cdOGGorselVar(h CDHead) bool { return h.Image != "" }
