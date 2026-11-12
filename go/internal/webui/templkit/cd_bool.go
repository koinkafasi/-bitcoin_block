package templkit

// cdBoolAttr bool'u HTML özniteliğine çevirir: "1" / "".
//
// `data-aktif={true}` templ'de derlenmez (string beklenir); boş string
// özniteliği tamamen kaldırır, bu yüzden "0" gibi açık bir değer basıyoruz
// ki JS tarafı "öznitelik var mı" değil "değer ne" diye baksın.
func cdBoolAttr(b bool) string {
	if b {
		return "1"
	}
	return "0"
}
