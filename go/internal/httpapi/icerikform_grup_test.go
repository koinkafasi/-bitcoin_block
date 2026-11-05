package httpapi

import (
	"bytes"
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"theblock/site/internal/webui/templkit"
)

// İçerik formu U2'de katlandı (spec 11 U2): ilk görünümde yalnız yazıyı
// yazmak için gerekenler kalıyor, gerisi <details> gruplarında.
//
// Katlamanın tek şartı vardı: KAYDETME DAVRANIŞI DEĞİŞMEYECEK. <details>
// bunu sağlar çünkü içeriği DOM'da kalır ve forma normal gönderilir — ama
// bu bir varsayım olarak bırakılamaz. Alanlar JS ile gizlenseydi ya da
// koşullu render'a alınsaydı sessizce kaybolurlardı ve editör sebebini
// anlamadan veri kaybederdi.

var (
	reAlanAdi = regexp.MustCompile(`<(?:input|select|textarea)\b[^>]*\bname="([^"]+)"`)
	reDetay   = regexp.MustCompile(`<details class="[^"]*form-grup`)
)

// ornekForm testlerde kullanılan dolu bir görünüm üretir.
//
// HasContent ve IsNew bilerek set edilir: yan sütun kartlarının çoğu
// `if v.HasContent` arkasında, bazıları `if !v.IsNew` arkasında. Kaydedilmiş
// bir içeriğin formunu temsil etmezsek test, alanların katlandığını değil
// hiç render edilmediğini ölçerdi.
func ornekForm() templkit.IcerikFormView {
	return templkit.IcerikFormView{
		Title:      "Test başlığı",
		Spot:       "Test spotu",
		TypeKey:    "haber",
		AiScore:    "72",
		HasContent: true,
		IsNew:      false,
		Rozets:     []templkit.IcerikRozet{{Code: "AM"}, {Code: "SD"}},
		Langs:      []templkit.IcerikLang{{Label: "Türkçe", URL: "/admin", Active: true}},
		Mahrets:    []templkit.IOpt{{Value: "1", Label: "AA"}},
		Authors:    []templkit.IOpt{{Value: "1", Label: "Yazar"}},
		Editors:    []templkit.IOpt{{Value: "1", Label: "Editör"}},
		Clusters:   []templkit.IOpt{{Value: "1", Label: "Küme"}},
	}
}

func TestIcerikFormKatlananAlanlarHTMLdeKaliyor(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, templkit.IcerikFormPanel(ornekForm()).Render(context.Background(), &buf))
	html := buf.String()

	// Katlanmış olsun olmasın, formda bulunması gereken alanlar.
	// Bunlar kaydetme sırasında sunucunun okuduğu adlardır; biri kaybolursa
	// o alan sessizce boşalır.
	zorunlu := []string{
		"baslik", "spot", "kategori", "etiketler", "tip",
		"mahrec", "author_id", "editor_id", "kume",
		"yayin_tarihi", "yorum_durumu", "rozet",
		"is_exclusive", "min_tier_weight",
		"kaynak_url", "kaynak_ad",
		"seo_baslik", "seo_aciklama",
		"gorsel_kredi", "sosyal_ozet", "ai_ozet",
		"is_ymyl", "ymyl_durumu", "ymyl_uzman",
		"ai_skor",
	}
	var bulunan = map[string]bool{}
	for _, m := range reAlanAdi.FindAllStringSubmatch(html, -1) {
		bulunan[m[1]] = true
	}
	var eksik []string
	for _, ad := range zorunlu {
		if !bulunan[ad] {
			eksik = append(eksik, ad)
		}
	}
	require.Emptyf(t, eksik,
		"şu alanlar formdan kaybolmuş: %s — katlama alanları DOM'dan "+
			"çıkarmamalı, yoksa kaydetme sessizce veri kaybeder",
		strings.Join(eksik, ", "))
}

func TestIcerikFormGruplariVarsayilanKapali(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, templkit.IcerikFormPanel(ornekForm()).Render(context.Background(), &buf))
	html := buf.String()

	gruplar := reDetay.FindAllString(html, -1)
	require.GreaterOrEqualf(t, len(gruplar), 4,
		"en az 4 katlanabilir grup bekleniyordu, %d bulundu", len(gruplar))

	// open niteliği taşıyan grup olmamalı: varsayılan kapalı olmazsa
	// sadeleştirmenin anlamı kalmaz. (Kullanıcının açtığı durum
	// localStorage'dan geri yüklenir, HTML'den değil.)
	require.NotContains(t, html, `<details class="form-grup" open`,
		"gruplar varsayılan olarak kapalı başlamalı")
}

// İlk görünümde (hiçbir grup açılmadan) kalan alan sayısı sınırlı olmalı.
// Sayı artarsa sadeleştirme sessizce geri alınmış demektir.
func TestIcerikFormIlkGorunumSade(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, templkit.IcerikFormPanel(ornekForm()).Render(context.Background(), &buf))
	html := buf.String()

	// <details> bloklarını çıkar, kalanı say.
	var disarida strings.Builder
	kalan := html
	for {
		i := strings.Index(kalan, "<details")
		if i < 0 {
			disarida.WriteString(kalan)
			break
		}
		disarida.WriteString(kalan[:i])
		j := strings.Index(kalan[i:], "</details>")
		if j < 0 {
			break
		}
		kalan = kalan[i+j+len("</details>"):]
	}

	var adlar []string
	for _, m := range reAlanAdi.FindAllStringSubmatch(disarida.String(), -1) {
		ad := m[1]
		if ad == "version" || ad == "kaydet" || ad == "_csrf" {
			continue // gizli alan ve gönder düğmeleri
		}
		adlar = append(adlar, ad)
	}
	require.LessOrEqualf(t, len(adlar), 10,
		"ilk görünümde %d alan var (%s) — katlama geri alınmış olabilir",
		len(adlar), strings.Join(adlar, ", "))
}
