package httpapi

func (s *Server) bolumHref(slug string) string {
	if slug == "" {
		return "/"
	}
	if _, ok := cdSectionMeta[slug]; ok {
		return "/" + slug
	}
	return "/kategori/" + slug
}
