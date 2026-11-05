package httpapi

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Faz 190 — İçerik düzenlemede "Dil / Çeviriler" paneli.
//
// Sahadaki sorun: panel 24 dilin 23'ünü "+" bağlantısı olarak gösteriyordu ve
// hepsi GET ile açılıyordu; rota yalnız POST olduğu için her tıklama 404
// dönüyordu (canlı ölçüm: 23 "+", 0 düzenlenebilir bağlantı).

// TestCeviriRotaVeTiklama: şablon hx-post kullanmalı (GET 404 veriyordu).
func TestCeviriRotaVeTiklama(t *testing.T) {
	tpl := okuDosya(t, "../webui/templkit/icerikform.templ")

	// Eski <a href> yaklaşımı kalkmalı.
	assert.NotContains(t, tpl, `Label:  l.NativeName + " +"`)
	// Yeni satırlar hx-post ile çeviri üretiyor.
	assert.Contains(t, tpl, `hx-post={ "/admin/icerikler/" + v.KaynakID + "/cevir/" + l.Code }`)
	assert.Contains(t, tpl, `hx-vals={ `+"`"+`{"ai":"1"}`+"`"+` }`)
	// Hedef AÇIKÇA eylem hücresi olmalı: htmx varsayılanı düğmenin İÇİNE
	// yazar (innerHTML) ve bekleme paneli bir <button> içinde kalırdı.
	assert.Contains(t, tpl, `hx-target="closest .ceviri-satir__eylem"`)
	assert.Contains(t, tpl, `hx-swap="outerHTML"`)
	// Var olan çeviri düzenlemeye gider.
	assert.Contains(t, tpl, `"/admin/icerikler/" + itoaT64(l.ContentID)`)

	// Sunucu tarafı: rota POST ve AI kipi var.
	src := okuDosya(t, "server.go")
	assert.Contains(t, src, `{"POST", "/admin/icerikler/{id}/cevir/{lang}", s.handleCreateTranslation}`)

	h := okuDosya(t, "admin_handlers.go")
	assert.Contains(t, h, `if r.FormValue("ai") == "1"`)

	// Faz 190b: AI çağrısı artık İSTEK İÇİNDE YAPILMAZ.
	//
	// Ölçüm: 28 paragraflık haberin çevirisi 160 sn sürdü; istek HTTP 000
	// ile düştü (sunucu WriteTimeout 60 sn, Cloudflare ~100 sn). Düğme
	// yalnız kuyruğa yazar, işi arka plan işçisi (ceviri.istekIsci) yapar.
	assert.Contains(t, h, "ceviri.IstekEkle(")
	assert.NotContains(t, h, "ceviri.UretTekrar(",
		"AI çağrısı istek yolunda olmamalı (proxy zaman aşımı)")
	assert.NotContains(t, h, "ceviri.Kaydet(",
		"kayıt istek yolunda olmamalı (uzun süren iş)")

	// Durum yoklama rotası kayıtlı olmalı.
	assert.Contains(t, src, `{"GET", "/admin/icerikler/{id}/cevir/{lang}/durum", s.handleTranslationStatus}`)
}

// TestCeviriHTMXYonlendirme: HTMX isteğinde 302 izlenmez; HX-Redirect gerekir.
//
// Aksi hâlde AI çeviri bitse bile sayfa hiç değişmiyordu; kullanıcı
// "çalışmıyor" der.
func TestCeviriHTMXYonlendirme(t *testing.T) {
	h := okuDosya(t, "admin_handlers.go")

	i := strings.Index(h, "func (s *Server) ceviriYonlendir(")
	assert.Greater(t, i, 0, "ceviriYonlendir bulunamadı")
	son := strings.Index(h[i:], "\n}")
	govde := h[i : i+son]

	assert.Contains(t, govde, `w.Header().Set("HX-Redirect", hedef)`)
	assert.Contains(t, govde, "IsHTMX(r)")
	assert.Contains(t, govde, "http.Redirect(w, r, hedef, http.StatusFound)")

	// Durum yoklaması bu yardımcıyı kullanmalı (düz http.Redirect değil):
	// htmx 302'yi fetch ile TAKİP eder ve dönen SAYFAYI hedefe yazar.
	j := strings.Index(h, "func (s *Server) handleTranslationStatus(")
	assert.Greater(t, j, 0, "handleTranslationStatus bulunamadı")
	aiSon := strings.Index(h[j:], "\nfunc ")
	aiGovde := h[j : j+aiSon]
	assert.Contains(t, aiGovde, "s.ceviriYonlendir(")

	// Faz 190: "Boş taslak" düğmesi de HTMX ile gönderiyor. flashRedirect
	// 302 döndürür; htmx takip edip tam editör sayfasını düğmenin içine
	// yazardı. Çeviri oluşturma yolunda flash yönlendirmesi HTMX-güvenli
	// yardımcıdan geçmeli.
	i = strings.Index(h, "func (s *Server) handleCreateTranslation(")
	assert.Greater(t, i, 0, "handleCreateTranslation bulunamadı")
	son = strings.Index(h[i:], "\n// ceviriYonlendir")
	require.Greater(t, son, 0)
	olustur := h[i : i+son]
	assert.NotContains(t, olustur, "s.flashRedirect(",
		"çeviri oluşturmada düz 302 kullanılmamalı (htmx takip eder)")
	assert.Contains(t, olustur, "s.ceviriFlash(")

	// Yardımcının kendisi HX-Redirect yolundan geçer.
	k := strings.Index(h, "func (s *Server) ceviriFlash(")
	require.Greater(t, k, 0)
	kSon := strings.Index(h[k:], "\n}")
	kGovde := h[k : k+kSon]
	assert.Contains(t, kGovde, "s.flash(w, r, kind, text)")
	assert.Contains(t, kGovde, "s.ceviriYonlendir(w, r, url)")
}

// TestCeviriKaynakKimligi: kaynak içerikte KaynakID kendi kimliği olmalı.
//
// Bu alan boş kalınca panel 24 satır gösterdi ama AI düğmesi hiç basılmadı
// (canlı ölçüm: 24 satır, 0 düğme).
func TestCeviriKaynakKimligi(t *testing.T) {
	v := okuDosya(t, "icerikform_view.go")
	assert.Contains(t, v, "v.KaynakID = strconv.FormatInt(f.Content.ID, 10)",
		"çeviri olmayan içerikte kaynak kendisidir")
	assert.Contains(t, v, "f.Content.TranslationRootID(), 10",
		"çeviride kaynak köktür")
}

// TestCeviriDurumEtiketleri: satır durumu (yok/taslak/yayında) gösterilmeli.
func TestCeviriDurumEtiketleri(t *testing.T) {
	src := okuDosya(t, "icerik_dil.go")

	assert.Contains(t, src, "func dilDurumTR(")
	assert.Contains(t, src, `case "yayinda":`)
	assert.Contains(t, src, `case "taslak":`)
	assert.Contains(t, src, `return "çeviri yok"`)

	// Mevcut çeviriler listenin başında olmalı (editörün en sık işi).
	assert.Contains(t, src, "r.Active || r.Once")
}

// TestCeviriBaglamBandi: çeviri düzenlenirken kaynak metin gösterilmeli.
func TestCeviriBaglamBandi(t *testing.T) {
	tpl := okuDosya(t, "../webui/templkit/icerikform.templ")
	assert.Contains(t, tpl, `class="ceviri-bant"`)
	assert.Contains(t, tpl, `ceviri-kaynak`)
	assert.Contains(t, tpl, "v.IcerikLocaleAd")

	h := okuDosya(t, "admin_handlers.go")
	assert.Contains(t, h, "s.ceviriBaglam(ctx, *c)")

	// Boş kayıt açmak yerine hata gösterilmeli (kuyruğu kirletmesin).
	ai := okuDosya(t, "admin_handlers.go")
	assert.Contains(t, ai, "templkit.CeviriHata(")
}

// TestCeviriCSS: satır düzeni ve uzun süren çeviri durumu.
func TestCeviriCSS(t *testing.T) {
	css := okuDosya(t, "../webui/static/admin-only.css")
	assert.Contains(t, css, ".ceviri-liste")
	assert.Contains(t, css, ".ceviri-satir")
	// AI 20-90 sn sürebiliyor: "çevriliyor" metni görünmeli.
	assert.Contains(t, css, "çevriliyor…")
}

// TestCeviriKuyrugu — Faz 190b: uzun AI işi istek yolundan çıkarıldı.
//
// Kök neden: senkron istek. Ölçüm: 28 paragraflık haber 160 sn sürdü,
// istek HTTP 000 ile düştü. Sunucu WriteTimeout 60 sn, Cloudflare ~100 sn.
func TestCeviriKuyrugu(t *testing.T) {
	src := okuDosya(t, "../ceviri/istek.go")

	// Kuyruk yazımı ve sahiplenme.
	assert.Contains(t, src, "func IstekEkle(")
	assert.Contains(t, src, "ON CONFLICT (content_id, locale) DO UPDATE")
	assert.Contains(t, src, "FOR UPDATE SKIP LOCKED",
		"aynı istek iki kez işlenmemeli")

	// İşçi tarafı: üret + kaydet + isteği bitir.
	assert.Contains(t, src, "UretTekrar(ctx, aiclient, src, is.Locale, logger)")
	assert.Contains(t, src, "Kaydet(ctx, pool, src, is.Locale, cev, durum)")
	assert.Contains(t, src, "istekBitir(ctx, pool, is.ID, IstekBitti")

	// Zaten çeviri varsa AI çağrılmaz (maliyet).
	assert.Contains(t, src, "t.Locale == is.Locale && t.ID != src.ID")

	// Ham SQL hatası kullanıcıya gösterilmez.
	assert.Contains(t, src, "kayitHataMetni(err)")
	assert.Contains(t, src, "foreign key")

	// Editör istekleri AYRI döngüde işlenir.
	//
	// Ölçüm: aynı döngüde işlendiğinde otomatik tur bitene kadar istek
	// "bekliyor" kaldı (2 dk sonra hâlâ bekliyordu). Editörün işi otomatik
	// turun süresine bağlı olamaz.
	run := okuDosya(t, "../ceviri/ceviri.go")
	i := strings.Index(run, "func Run(")
	require.Greater(t, i, 0)
	son := strings.Index(run[i:], "\nfunc istekDongusu(")
	govde := run[i : i+son]
	assert.Contains(t, govde, "go istekDongusu(ctx, pool, aiclient, logger)",
		"istek döngüsü ayrı goroutine olmalı")
	assert.NotContains(t, govde, "istekIsle(",
		"otomatik tur döngüsü editör isteğini kendi içinde işlememeli")

	assert.Contains(t, run, "func istekDongusu(")
	// NOT: işi artık ayrı işçiler yapar (aşağıdaki eşzamanlılık testi);
	// istekDongusu yalnız işçileri başlatır.
	assert.Contains(t, run, "for i := 0; i < istekEsZamanli; i++ {")
	// Süreç çeviri ortasında öldüyse "calisiyor" satırı sonsuza kadar kalırdı.
	assert.Contains(t, run, "WHERE durum = 'calisiyor'")

	// Eşzamanlı işçiler: tek tüketici, uzun süren bir isteğin arkasında
	// yeni isteği 6 dakika bekletiyordu (ölçüm: 361 sn "bekliyor").
	assert.Contains(t, run, "const istekEsZamanli = 2")
	assert.Contains(t, run, "for i := 0; i < istekEsZamanli; i++ {")
	assert.Contains(t, run, "go istekIsci(ctx, pool, aiclient, logger)")
	assert.Contains(t, run, "func istekIsci(")
	// İşçi, sıradaki isteği `FOR UPDATE SKIP LOCKED` ile alır.
	worker := run[strings.Index(run, "func istekIsci("):]
	worker = worker[:strings.Index(worker, "\n}\n")]
	assert.Contains(t, worker, "istekAl(ctx, pool)")
	assert.Contains(t, worker, "istekIsle(ctx, pool, aiclient, logger, *is, \"taslak\")")

	// İşçi panikle ölürse satır "calisiyor" kalır; temizleyici olmalı.
	assert.Contains(t, run, "func bayatIstekTemizle(")
	assert.Contains(t, run, "now() - interval '20 minutes'")
	assert.Contains(t, run, "go bayatIstekTemizle(ctx, pool, logger)")

	// Otomatik kuyruk, editörün istediği içeriği tekrar çevirmemeli.
	assert.Contains(t, run, "AND i.durum IN ('bekliyor','calisiyor')")
	// Kök üzerinden bakılmalı: editör bir ÇEVİRİ kaydını düzenleyip başka
	// dil isterse istek satırı kökü değil o kaydı gösterir.
	assert.Contains(t, run, "JOIN contents ic ON ic.id = i.content_id")
	assert.Contains(t, run,
		"COALESCE(ic.translation_of, ic.id) = COALESCE(c.translation_of, c.id)")
}

// TestCeviriBeklemePaneli — panel yoklaması ve kullanıcıya görünen metin.
func TestCeviriBeklemePaneli(t *testing.T) {
	tpl := okuDosya(t, "../webui/templkit/ceviri_bekliyor.templ")
	assert.Contains(t, tpl, `hx-trigger="every 2s"`)
	assert.Contains(t, tpl, `hx-swap="outerHTML"`)
	assert.Contains(t, tpl, "çevirisi hazırlanıyor")
	assert.Contains(t, tpl, "1–2 dakika")

	css := okuDosya(t, "../webui/static/admin-only.css")
	assert.Contains(t, css, ".ceviri-bekliyor")
	assert.Contains(t, css, ".ceviri-bekliyor__halka")
	// Hareket azaltma tercihi korunur.
	i := strings.Index(css, ".ceviri-bekliyor__halka")
	assert.Greater(t, i, 0)
	assert.Contains(t, css, "prefers-reduced-motion")

	// Panel eylem hücresinin yerini alır (grid'de `auto` sütun). Genişlik
	// sınırı olmadan uzun açıklama satırı yatay olarak şişiriyordu.
	j := strings.Index(css, ".ceviri-bekliyor {")
	require.Greater(t, j, 0)
	kural := css[j:]
	kural = kural[:strings.Index(kural, "}")]
	assert.Contains(t, kural, "max-width: 340px")
	// Mobilde tam genişlik: tek sütuna düşen düzen.
	assert.Contains(t, css, ".ceviri-satir > .ceviri-bekliyor { grid-column: 1 / -1; max-width: none; }")
}

// TestCeviriIstekTablosu — kuyruk tablosu migration ile gelir.
func TestCeviriIstekTablosu(t *testing.T) {
	up := okuDosya(t, "../../migrations/0102_ceviri_istek.up.sql")
	assert.Contains(t, up, "CREATE TABLE IF NOT EXISTS ceviri_istek")
	assert.Contains(t, up, "REFERENCES contents(id) ON DELETE CASCADE")
	assert.Contains(t, up, "CREATE UNIQUE INDEX IF NOT EXISTS ceviri_istek_tek")
	assert.Contains(t, up, "CREATE INDEX IF NOT EXISTS ceviri_istek_bekleyen")

	down := okuDosya(t, "../../migrations/0102_ceviri_istek.down.sql")
	assert.Contains(t, down, "DROP TABLE IF EXISTS ceviri_istek")
	// Geri alma üretilen çevirileri SİLMEMELİ.
	assert.NotContains(t, down, "DELETE FROM contents")
}

// TestOtokayitKirliGuard — Faz 191.
//
// Belirti: izole ortamda hiçbir alana dokunulmadan "değişiklik var" yazıyor
// ve gezinmede beforeunload diyaloğu çıkıyordu. Ölçüm (canlı, gerçek veri):
// durum "Henüz otomatik kaydedilmedi" — yani üretimde parmak izi tutuyor.
// Hata, ortak çalışma odasının sayfaya basılandan FARKLI belge döndürdüğü
// durumda ortaya çıkıyor (izolede oda üretim verisini taşıyordu).
//
// Kural artık: KULLANICI dokunmadan kirli sayılmaz. Bu test kuralın
// koddan sessizce kaybolmasını engeller; kaybolursa hata geri gelir.
func TestOtokayitKirliGuard(t *testing.T) {
	tpl := okuDosya(t, "../webui/templkit/icerikform.templ")

	i := strings.Index(tpl, "var sonParmak = parmak();")
	require.Greater(t, i, 0, "otomatik kayıt bloğu bulunamadı")
	son := strings.Index(tpl[i:], "function yolla(){")
	require.Greater(t, son, 0)
	govde := tpl[i : i+son]

	// Kullanıcı dokunuşu izlenir.
	assert.Contains(t, govde, "var dokundu = false")
	assert.Contains(t, govde, "'pointerdown'", "araç çubuğu düğmeleri de sayılmalı")
	assert.Contains(t, govde, "if (!dokundu) return")
	// Editör hazır olduğunda taban çizgi tazelenir (oda senkronu sonradır).
	assert.Contains(t, govde, "be.dataset.ready === '1'")
	assert.Contains(t, govde, "if (!dokundu) sonParmak = parmak()")

	// Gerçek düzenleme hâlâ kirli sayılmalı — guard her şeyi yutuyor olamaz.
	assert.Contains(t, govde, "arm()")
	assert.Contains(t, govde, "if (p === sonParmak) return")
}

// TestGelismisPanelTureGore — Faz 192.
//
// Kullanıcı bildirimi: "içerik düzenle'de Gelişmiş alandaki türe özel alanlar
// açılmıyor, bir şey göremedim".
//
// Ölçüm (üretim, HTTP): `haber` (13.486 içeriğin 13.470'i) için panel gövdesi
// 424 bayt ve tek alan GİZLİ → görünen alan 0. `video` için 1 görünen alan.
// Yani panel boş bir kutu olarak açılıyordu.
func TestGelismisPanelTureGore(t *testing.T) {
	// Tür→alan kararı tek yerde.
	v := okuDosya(t, "icerikform_view.go")
	assert.Contains(t, v, "func gelismisAlanlar(typeKey string) (bool, string)")
	i := strings.Index(v, "func gelismisAlanlar(")
	require.Greater(t, i, 0)
	govde := v[i:]
	govde = govde[:strings.Index(govde, "\n}\n")]
	assert.Contains(t, govde, `case "video":`)
	assert.Contains(t, govde, `case "hesaplayici":`)
	// Bilinmeyen tür (haber dahil) → panel YOK.
	assert.Contains(t, govde, "return false, \"\"")
	assert.Contains(t, v, "v.GelismisVar, v.GelismisIpucu = gelismisAlanlar(f.TypeKey)")

	// Şablon paneli koşullu basmalı: boş kutu üretmemeli.
	tpl := okuDosya(t, "../webui/templkit/icerikform.templ")
	j := strings.Index(tpl, `<summary data-grup="gelismis">`)
	require.Greater(t, j, 0, "Gelişmiş paneli bulunamadı")
	once := tpl[maxInt(0, j-400):j]
	assert.Contains(t, once, "if v.GelismisVar {",
		"panel koşulsuz basılırsa haber türünde boş kutu görünür")
	// İpucu türe göre yazılmalı, sabit metin kalmamalı.
	assert.Contains(t, tpl, `{ v.GelismisIpucu }`)
	assert.NotContains(t, tpl, `Gelişmiş <span class="form-grup-ipucu">türe özel alanlar</span>`)

	// Video süresi alanı hâlâ var ve yalnız video'da görünür.
	assert.Contains(t, tpl, `id="vidSureField" hidden?={ v.TypeKey != "video" }`)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
