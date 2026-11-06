package templkit

// VarlikSurum statik varlık (CSS/JS) sürümüdür — `?v=` sorgu parametresine
// yazılır, tarayıcı/edge önbelleğini kırar.
//
// NEDEN GLOBAL: `cdHead` şablonu her sayfada çağrılıyor ve sürümü sunucudan
// alması gerekiyor; `CDHead`'e alan eklemek ~10 çağrı yerini değiştirmek
// demekti. httpapi.NewServer açılışta `assetVer`'i buraya yazıyor.
//
// NEDEN GEREKLİ (yaşanmış hata, 6 Kasım 2026): sürüm `v=1` olarak SABİT
// yazılmıştı. Tasarım sistemi baştan ölçülüp yeniden yazıldığında
// tarayıcılar eski `cd.css`'i önbellekten servis etmeye devam etti;
// ölçüm "hiçbir değişiklik uygulanmamış" gibi göründü ve yanlış teşhis
// riski doğdu. Sürüm artık süreç başına değişiyor.
var VarlikSurum = "1"

// cdVarlik sürüm parametresini üretir (boşsa "1").
func cdVarlik() string {
	if VarlikSurum == "" {
		return "1"
	}
	return VarlikSurum
}
