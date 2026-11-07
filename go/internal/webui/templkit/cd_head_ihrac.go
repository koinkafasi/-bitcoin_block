package templkit

import "github.com/a-h/templ"

// CDHeadParca cdHead bileşenini testlerden çağrılabilir hâle getirir.
//
// NEDEN: cdHead templ dosyasında tanımlı ve paket dışına kapalı; SEO meta
// setini HTTP katmanından test etmek için (cd_seo_test) küçük bir kapı.
func CDHeadParca(h CDHead) templ.Component { return cdHead(h) }
