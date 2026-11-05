package httpapi

import (
	"context"
	"strings"
)

// footerConfigLang alt bilgiyi verilen dilde etiketlerle döndürür (Faz 189).
func (s *Server) footerConfigLang(ctx context.Context, locale string) *FooterConfig {
	return s.footerYerellestir(s.footerConfig(ctx), locale)
}

// footerYerellestir var olan bir alt bilgi yapılandırmasını yerelleştirir.
//
// Varsayılan alt bilgi bağlantıları Türkçe sabittir ("Künyemiz", "Bize
// Ulaşın"). Çeviri `translations` tablosunda `footer.<url-slug>`
// anahtarıyla tutulur; yoksa Türkçe etiket korunur.
//
// Neden ayrı fonksiyon: alt bilgi onlarca handler'da `s.footerConfig(ctx)`
// ile kuruluyor. Hepsini tek tek değiştirmek yerine renderPublic içinde tek
// noktada yerelleştiriliyor; aksi hâlde /en/arama gibi sayfalarda alt bilgi
// Türkçe kalıyordu (ölçüldü).
func (s *Server) footerYerellestir(fc *FooterConfig, locale string) *FooterConfig {
	if fc == nil || locale == "" || locale == "tr" || s.i18n == nil {
		return fc
	}
	cevir := func(link FootLink) FootLink {
		slug := strings.Trim(strings.TrimSpace(link.URL), "/")
		if i := strings.LastIndex(slug, "/"); i >= 0 {
			slug = slug[i+1:]
		}
		if slug == "" {
			return link
		}
		anahtar := "footer." + slug
		if v := s.i18n.T(locale, anahtar); v != anahtar && strings.TrimSpace(v) != "" {
			link.Label = v
		}
		return link
	}
	for i := range fc.Strip {
		fc.Strip[i] = cevir(fc.Strip[i])
	}
	for i := range fc.Cols {
		if t := s.localeText(locale, "footer.col."+strings.ToLower(fc.Cols[i].Title), ""); t != "" {
			fc.Cols[i].Title = t
		}
		for j := range fc.Cols[i].Links {
			fc.Cols[i].Links[j] = cevir(fc.Cols[i].Links[j])
		}
	}
	return fc
}
