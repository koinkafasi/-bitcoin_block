//go:build integration

// Public sayfaların üçüncü taraf bağlantı yüzeyini kilitler.
//
// Neden test gerekiyor: fontlar bir süre fonts.googleapis.com ve
// fonts.gstatic.com üzerinden yükleniyordu. Bu, sayfayı açan her ziyaretçinin
// IP adresinin ve User-Agent bilgisinin, kullanıcı hiçbir şey yapmadan ve
// rızası alınmadan Google'a (ABD) gitmesi demekti — 6698 sayılı Kanun'un
// 9. maddesi anlamında yurt dışına aktarım. Fontlar yerele alındı; bu test
// geri gelmesini engeller.
//
// Bir <link href="https://fonts.googleapis.com/..."> satırını şablona geri
// koymak bu testi kırar.
package httpapi_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/config"
	"theblock/site/internal/httpapi"
	"theblock/site/internal/i18n"
)

// yasakliKaynaklar: public HTML'de görünmemesi gereken üçüncü taraf adresler.
// Sayfa açılışında otomatik istek üretenler listelenir.
var yasakliKaynaklar = []string{
	"fonts.googleapis.com",
	"fonts.gstatic.com",
	"ajax.googleapis.com",
	"cdn.jsdelivr.net",
	"cdnjs.cloudflare.com",
	"unpkg.com",
	"code.jquery.com",
	"connect.facebook.net",
	"platform.twitter.com",
}

func TestPublicSayfalardaUcuncuTarafKaynagiYok(t *testing.T) {
	skipIfNoDB(t)

	yollar := []string{"/", "/kategori/gundem", "/arama?q=test"}
	for _, yol := range yollar {
		resp, body := get(t, yol)
		if resp.StatusCode != 200 {
			// Sayfa test verisine bağlı olabilir; yoksa atla (içerik iddiası değil).
			t.Logf("%s → HTTP %d, atlandı", yol, resp.StatusCode)
			continue
		}
		for _, host := range yasakliKaynaklar {
			assert.NotContains(t, body, host,
				"%s sayfasında üçüncü taraf kaynak var: %s — yurt dışına aktarım yaratır, "+
					"varlığı yerele alın", yol, host)
		}
	}
}

// Fontların gerçekten kendi sunucumuzdan geldiğini doğrular: yalnız
// "googleapis yok" demek yeterli değil, yerel karşılığı da çalışmalı.
func TestYerelFontlarSunuluyor(t *testing.T) {
	skipIfNoDB(t)

	resp, css := get(t, "/static/fonts.css")
	assert.Equal(t, 200, resp.StatusCode, "fonts.css sunulmalı")
	assert.Contains(t, css, "@font-face", "fonts.css @font-face içermeli")
	assert.Contains(t, css, "Poppins")
	// Dosyanın yorum bloğu, kaldırılan adresleri açıklamak için adıyla anıyor;
	// bu yüzden metnin tamamında arama yapılmaz. Asıl değişmez şudur: hiçbir
	// src bildirimi uzak bir adrese işaret etmemeli.
	for _, uzak := range []string{"url(https://", "url('https://", "url(\"https://"} {
		assert.NotContains(t, css, uzak, "fonts.css uzak kaynağa işaret etmemeli")
	}
	// unicode-range korunmalı: yoksa tarayıcı gereksiz alt kümeyi de indirir.
	assert.Contains(t, css, "unicode-range", "alt küme aralıkları korunmalı")

	// Türkçe için iki alt küme de şart: "ı" latin içinde, "ğ ş İ" latin-ext.
	for _, ad := range []string{
		"/static/fonts/poppins-400-latin.woff2",
		"/static/fonts/poppins-400-latin-ext.woff2",
		"/static/fonts/poppins-600-latin.woff2",
		"/static/fonts/poppins-700-latin.woff2",
	} {
		r, gövde := get(t, ad)
		assert.Equal(t, 200, r.StatusCode, "%s sunulmalı", ad)
		assert.True(t, strings.HasPrefix(gövde, "wOF2"),
			"%s geçerli bir woff2 olmalı (sihirli bayt)", ad)
	}
}

// GA4 yapılandırılmış OLSA BİLE sayfa, gtag.js'i kendisi yüklememeli:
// ölçüm kimliği yalnız cerez.js'e data-ga4 ile geçer ve betik, ziyaretçi
// açıkça kabul ederse scripti istemci tarafında ekler.
//
// Kurul'un 08.08.2024 tarihli 2024/1361 sayılı kararı, kesinlikle gerekli
// olmayan analitik çerez için sitenin ilk girişinde açık rıza alacak bir
// mekanizma bulunmamasını ihlal saydı. Şablona <script src="...gtag/js...">
// geri konursa bu test kırılır.
func TestAnalitikRizaOncesiYuklenmez(t *testing.T) {
	skipIfNoDB(t)

	ctx := t.Context()
	_, err := itPool.Exec(ctx,
		`INSERT INTO settings (section,key,value) VALUES ('trend','ga4_olcum','"G-TESTONLY1"'::jsonb)
		  ON CONFLICT (section,key) DO UPDATE SET value=EXCLUDED.value`)
	if err != nil {
		t.Skipf("ayar yazılamadı: %v", err)
	}
	t.Cleanup(func() {
		_, _ = itPool.Exec(context.Background(),
			`DELETE FROM settings WHERE section='trend' AND key='ga4_olcum'`)
	})

	// Kendi sunucusunu kurar: ga4Measurement, cachedSetting üzerinden TTL
	// önbellekli okunur (ga4.go). Paketin ortak sunucusu önceki testlerde
	// ayarı BOŞ olarak önbelleğe aldığı için buradaki INSERT ona yansımaz;
	// yeni sunucunun önbelleği boştur ve değeri taze okur.
	bundle, err := i18n.Load(ctx, itPool)
	require.NoError(t, err)
	cfg := config.Config{
		Env:         "test",
		Debug:       false,
		ListenAddr:  "127.0.0.1:0",
		BaseURL:     "http://127.0.0.1",
		Port:        "0",
		DatabaseURL: testDB,
	}
	srv := httpapi.NewServer(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), itPool, bundle, nil)
	ts := httptest.NewServer(srv.Handler)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/")
	require.NoError(t, err)
	defer resp.Body.Close()
	ham, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	body := string(ham)
	if resp.StatusCode != 200 {
		t.Skipf("ana sayfa HTTP %d", resp.StatusCode)
	}

	// Ön koşul: ölçüm kimliği gerçekten sayfaya ulaşmış olmalı. Ulaşmadıysa
	// "googletagmanager yok" iddiası boş yere geçer ve test değersizleşir.
	require.Contains(t, body, `data-ga4="G-TESTONLY1"`,
		"ölçüm kimliği betiğe veri özniteliğiyle geçmeli (ön koşul)")

	assert.NotContains(t, body, "googletagmanager.com",
		"rıza alınmadan analitik script sunucudan basılmamalı")
	assert.NotContains(t, body, "gtag(", "gtag çağrısı sayfaya gömülmemeli")
	assert.Contains(t, body, "cerez.js", "çerez onayı betiği yüklenmeli")
	assert.Contains(t, body, "data-cerez-ayar",
		"tercihi sonradan değiştirme bağlantısı bulunmalı (rızanın geri alınması)")
}
