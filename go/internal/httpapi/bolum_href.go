package httpapi

// bolumHref bölüm sayfasının GERÇEKTEN açılan adresini döndürür.
//
// NEDEN: makale sayfasının kırıntı yolu ve JSON-LD breadcrumb'ı
// `/{bolum}` yazıyordu. Kök bölümler (markets, tech…) kayıtlı olduğu için
// açılıyor; ama WordPress'ten gelen kategoriler (haber, analiz, borsa…)
// kökte kayıtlı değil — /haber 404 dönüyordu ve kırıntı yolundaki bağlantı
// ölüydü. Google için de yapısal veri 404'e işaret ediyordu.
//
// Bu yüzden liste adresi `kategori` yolu altına düşer; makale adresi
// (/{bolum}/{YYYY}/{MM}/{DD}/{slug}) değişmez.

func (s *Server) bolumHref(slug string) string {
	if slug == "" {
		return "/"
	}
	if _, ok := cdSectionMeta[slug]; ok {
		return "/" + slug
	}
	return "/kategori/" + slug
}
