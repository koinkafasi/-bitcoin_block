package httpapi

// Görünüm eşleyicilerinin ortak yardımcıları.
//
// strVal daha önce panel_view.go içinde duruyordu — eski panonun eşleyicisi
// ile aynı dosyada. O pano kaldırılırken bu yardımcı da gitti ve üç görünüm
// dosyası derlenemez oldu (icerikler_view, kategoriler_view, medya_view).
// Ortak yardımcı, onu kullanan tek bir özelliğin dosyasında durmamalı.

// strVal nil olabilen metin işaretçisini boş dizeye indirger.
// Şablonlar nil işaretçi alamaz; veritabanından gelen NULL alanlar burada
// düzleştirilir.
func strVal(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
