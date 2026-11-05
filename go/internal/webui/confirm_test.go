package webui

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// `data-confirm` projede iki ayrı anlamda kullanılıyor ve ayrımı DEĞERİ
// belirliyor:
//
//	"/..." ile başlar → POST edilecek URL   (form dışındaki öğeler, app.js)
//	aksi halde        → onay METNİ          (form içindeki öğeler, admin-ui.js)
//
// app.js'teki belge düzeyi tık dinleyicisi bu ayrımı yapmıyordu: metin
// taşıyan formları da yakalıyor, gerçek gönderimi `preventDefault` ile iptal
// edip onay metnini URL sanarak POST ediyordu. Sonuç: gazete panelindeki
// "Sil" ve "Gönder" düğmeleri hiçbir şey yapmıyordu — istek
// `/admin/Sayı silinsin mi?` gibi anlamsız bir adrese gidiyordu.
//
// İkinci hata: onay metni düğmede, formda değilse (`<button type=submit
// data-confirm="...">`) hiç onay sorulmuyordu; "Varsayılana Dön" ve
// "Tümünde değiştir" doğrudan gönderiliyordu.
//
// Aşağıdaki testler bu iki hatanın geri gelmesini engeller.

func webuiOkuStatic(t *testing.T, ad string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("static", ad))
	if err != nil {
		t.Fatalf("%s okunamadı: %v", ad, err)
	}
	return string(b)
}

// TestDataConfirmURLKuraliAyrishik app.js'in yalnız URL taşıyan öğeleri
// üstlendiğini doğrular.
func TestDataConfirmURLKuraliAyrishik(t *testing.T) {
	js := webuiOkuStatic(t, "app.js")

	if !strings.Contains(js, `startsWith("/")`) {
		t.Error("app.js data-confirm değerinin URL olup olmadığına bakmıyor; " +
			"onay metni taşıyan formları da yakalar ve gönderimi bozar")
	}
	// "Form içinde mi" yanlış ölçüt: içerik formundaki silme BAĞLANTILARI
	// form göndermez, bu dinleyicinin işidir.
	if strings.Contains(js, `t.closest("form")`) {
		t.Error("app.js 'form içinde mi' diye bakıyor; form içindeki silme " +
			"bağlantıları (Kalıcı Sil, Kaynağı sil) sessizce çalışmaz olur")
	}
	if !strings.Contains(js, `t.form`) {
		t.Error("app.js formu gönderen kontrolü ayırmıyor")
	}
	if strings.Contains(js, `f.action = t.getAttribute("data-confirm")`) {
		t.Error("onay metni doğrudan form action'ına yazılıyor")
	}
}

// TestDataConfirmFormKuraliDevirli admin-ui.js'in olay devri kullandığını ve
// düğme düzeyindeki onayı da okuduğunu doğrular.
func TestDataConfirmFormKuraliDevirli(t *testing.T) {
	js := webuiOkuStatic(t, "admin-ui.js")

	if strings.Contains(js, `querySelectorAll('form[data-confirm]').forEach`) {
		t.Error("admin-ui.js onay formlarına tek tek bağlanıyor; HTMX takasından " +
			"sonra gelen formlar onay sorulmadan gönderilir")
	}
	if !strings.Contains(js, `document.addEventListener('submit'`) {
		t.Error("admin-ui.js onayı belge düzeyinde dinlemiyor")
	}
	// HTMX formunda native submit, htmx'i atlatır.
	if !strings.Contains(js, "htmx.trigger") {
		t.Error("HTMX ile yönetilen onay formu native submit ile gönderiliyor; " +
			"htmx devreye giremez")
	}
	// Düğme düzeyindeki onay metni de okunmalı.
	if !strings.Contains(js, "submitter") {
		t.Error("onay metni yalnız formda aranıyor; <button data-confirm> " +
			"biçimindeki yıkıcı düğmeler onay sormadan gönderilir")
	}
}

// TestDataConfirmKullanimlariTutarli şablonlardaki her kullanımın değerine
// ve öğe türüne göre doğru kurala oturduğunu doğrular.
//
//	<form data-confirm>            → METİN (formun tamamı için onay)
//	<button type=submit data-confirm> → METİN (yalnız o düğme için onay)
//	<a>/<span>… data-confirm       → URL   (app.js POST eder)
//
// Bağlantının form içinde olması sorun değil: tıklama form göndermez.
// Karışırsa iki hata sınıfından biri geri gelir.
func TestDataConfirmKullanimlariTutarli(t *testing.T) {
	oznitelik := regexp.MustCompile(`data-confirm="([^"]*)"`)
	// data-confirm'in bulunduğu öğenin açılış etiketi (geriye doğru).
	etiket := regexp.MustCompile(`(?s)<([a-zA-Z][a-zA-Z0-9]*)\b([^<>]*)$`)

	var kontrol int
	for _, dizin := range []string{"templkit", "templates"} {
		girisler, err := os.ReadDir(dizin)
		if err != nil {
			t.Fatalf("%s okunamadı: %v", dizin, err)
		}
		for _, g := range girisler {
			ad := g.Name()
			if !strings.HasSuffix(ad, ".templ") && !strings.HasSuffix(ad, ".html") {
				continue
			}
			b, err := os.ReadFile(filepath.Join(dizin, ad))
			if err != nil {
				t.Fatal(err)
			}
			icerik := string(b)

			for _, m := range oznitelik.FindAllStringSubmatchIndex(icerik, -1) {
				deger := icerik[m[2]:m[3]]
				urlMu := strings.HasPrefix(deger, "/")

				// Öğenin adı: özniteliğin solundaki açılış etiketinden.
				onceki := icerik[:m[0]]
				e := etiket.FindAllStringSubmatch(onceki, -1)
				if len(e) == 0 {
					continue
				}
				son := e[len(e)-1]
				ad2 := strings.ToLower(son[1])
				nitelikler := son[2]

				// Metin bekleyenler: form gövdeleri ve submit düğmeleri.
				metinBekler := ad2 == "form"
				if (ad2 == "button" || ad2 == "input") && strings.Contains(nitelikler, "submit") {
					metinBekler = true
				}

				kontrol++
				if metinBekler {
					if urlMu {
						t.Errorf("%s/%s: <%s> üzerinde URL var (%q); burada METİN beklenir",
							dizin, ad, ad2, deger)
					}
				} else if !urlMu {
					t.Errorf("%s/%s: <%s> üzerinde metin var (%q); burada URL beklenir — "+
						"tıklama hiçbir şey yapmaz", dizin, ad, ad2, deger)
				}
			}
		}
	}
	if kontrol == 0 {
		t.Skip("şablonlarda data-confirm kullanımı bulunamadı")
	}
	t.Logf("denetlenen data-confirm kullanımı: %d", kontrol)
}
