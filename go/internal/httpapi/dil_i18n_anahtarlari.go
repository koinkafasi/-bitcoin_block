package httpapi

// jsI18nAnahtarlari istemci tarafında (JS ile üretilen metinlerde) kullanılan
// çeviri anahtarlarıdır (Faz 189).
//
// Şablon metinleri {{.T "..."}} ile doğrudan çevrilebiliyor; ama JS'in
// oluşturduğu düğme etiketleri şablon değişkeni alamaz. Bu anahtarlar
// sayfaya bir kez JSON olarak basılır (bkz. i18n.JSSozluk).
var jsI18nAnahtarlari = []string{
	"related_more",
	"related_less",
	"copied",
	"copy_link",
	"load_more",
	"no_results",
	"news_count",
}
