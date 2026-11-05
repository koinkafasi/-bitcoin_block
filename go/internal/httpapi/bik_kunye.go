package httpapi

import (
	"context"
	"encoding/json"
	"html"
	"html/template"
	"strings"

	"theblock/site/internal/repo"
)

// BİK künye standardı — Tebliğ, künyede on zorunlu alanın bulunmasını ve her
// birinin kendi kimliğiyle etiketlenmiş bir div içinde yer almasını istiyor.
// Kurum kontrolü bu kimliklerle otomatik yapılıyor.
//
// Neden şablondan üretiliyor, sayfa metninden değil: künye bir `pages`
// kaydıdır ve içeriği editör tarafından serbestçe yazılır. Kimlikleri
// metne gömmek, her düzenlemede kaybolma riski taşır — bir editörün
// yanlışlıkla sildiği div, uyumu sessizce bozar. Blok burada üretilince
// kimliklerin varlığı koda bağlanır; editör yalnız DEĞERLERİ değiştirir.
//
// Değerler settings tablosunun "kunye" bölümünden okunur. Üçü zaten
// yönetimde var olan iletişim ayarlarına düşer (telefon, e-posta, adres);
// böylece aynı bilgi iki yerde tutulmaz.

// bikKunyeAlan tek bir zorunlu künye alanıdır.
type bikKunyeAlan struct {
	ID    string // BİK'in beklediği div kimliği
	Key   string // settings "kunye" bölümündeki anahtar
	Label string
	Value string
}

// bikKunyeAlanlari tebliğdeki sırayla on zorunlu alanı döndürür.
// Sıra tebliğdeki listeyle aynı tutulur ki karşılaştırma kolay olsun.
func bikKunyeAlanlari() []bikKunyeAlan {
	return []bikKunyeAlan{
		{ID: "bik-kunye-ticaret-unvani", Key: "ticaret_unvani", Label: "Yayın Sahibi Ticaret Unvanı"},
		{ID: "bik-kunye-tuzel-kisi-temsilcisi", Key: "tuzel_kisi_temsilcisi", Label: "Tüzel Kişi Temsilcisi"},
		{ID: "bik-kunye-yayinci", Key: "yayinci", Label: "Yayıncı"},
		{ID: "bik-kunye-sorumlu-yim", Key: "sorumlu_yim", Label: "Sorumlu Müdür / Yazı İşleri Müdürü"},
		{ID: "bik-kunye-yonetim-yeri", Key: "yonetim_yeri", Label: "Yönetim Yeri"},
		{ID: "bik-kunye-telefon", Key: "telefon", Label: "İletişim Telefonu"},
		{ID: "bik-kunye-eposta", Key: "eposta", Label: "Kurumsal E-Posta"},
		{ID: "bik-kunye-uets", Key: "uets", Label: "Ulusal Elektronik Tebligat Sistemi Adresi"},
		{ID: "bik-kunye-yer-saglayici-unvan", Key: "yer_saglayici_unvan", Label: "Yer Sağlayıcı Ticaret Unvanı"},
		{ID: "bik-kunye-yer-saglayici-adres", Key: "yer_saglayici_adres", Label: "Yer Sağlayıcı Adresi"},
	}
}

// bikKunyeDoldur alanları ayarlardan doldurur. Boş kalanlar boş döner;
// çağıran taraf eksikleri raporlayabilir.
func (s *Server) bikKunyeDoldur(ctx context.Context) []bikKunyeAlan {
	alanlar := bikKunyeAlanlari()
	gs, _ := repo.GetSettings(ctx, s.pool, "kunye")

	// Üç alan için yönetimdeki iletişim ayarları yedek kaynaktır: aynı
	// bilgiyi iki yerde tutup birini güncellemeyi unutmak, künyeyi sessizce
	// yanlışlar.
	c := s.contactConfig(ctx)
	yedek := map[string]string{
		"telefon":      strings.TrimSpace(c.Telefon),
		"eposta":       strings.TrimSpace(c.Info),
		"yonetim_yeri": strings.TrimSpace(c.Adres),
	}

	for i := range alanlar {
		v := strings.TrimSpace(bikSettingStr(gs, alanlar[i].Key))
		if v == "" {
			v = yedek[alanlar[i].Key]
		}
		alanlar[i].Value = v
	}
	return alanlar
}

// bikKunyeEksikler doldurulmamış zorunlu alanların etiketlerini döndürür.
// Yönetim panelinde uyarı göstermek ve denetim raporu üretmek için.
func bikKunyeEksikler(alanlar []bikKunyeAlan) []string {
	var eksik []string
	for _, a := range alanlar {
		if strings.TrimSpace(a.Value) == "" {
			eksik = append(eksik, a.Label)
		}
	}
	return eksik
}

// bikKunyeHTML künye bloğunu üretir.
//
// Değer boş olsa bile div BASILIR: Kurum kontrolü kimliğin varlığına bakıyor,
// eksik kimlik doğrudan başarısızlık demek. Boş değer yerine "—" yazılır ki
// sayfa okunabilir kalsın ve eksiklik gözle de görülsün.
func bikKunyeHTML(alanlar []bikKunyeAlan) template.HTML {
	var b strings.Builder
	b.WriteString(`<section class="bik-kunye" aria-label="Künye bilgileri">`)
	for _, a := range alanlar {
		deger := strings.TrimSpace(a.Value)
		if deger == "" {
			deger = "—"
		}
		b.WriteString(`<div id="` + a.ID + `" class="bik-kunye-satir">`)
		b.WriteString(`<span class="bik-kunye-etiket">` + html.EscapeString(a.Label) + `:</span> `)
		b.WriteString(`<span class="bik-kunye-deger">` + html.EscapeString(deger) + `</span>`)
		b.WriteString(`</div>`)
	}
	b.WriteString(`</section>`)
	return template.HTML(b.String())
}

// bikSettingStr JSONB ayar değerini düz metne çevirir. settings tablosunda
// değerler JSON olarak tutuluyor; çoğu basit dize ("abc").
func bikSettingStr(gs map[string][]byte, key string) string {
	b, ok := gs[key]
	if !ok {
		return ""
	}
	var v string
	if json.Unmarshal(b, &v) == nil {
		return v
	}
	return strings.Trim(string(b), `"`)
}
