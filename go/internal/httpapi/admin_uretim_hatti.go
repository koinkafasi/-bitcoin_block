package httpapi

import (
	"context"
	"html"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"theblock/site/internal/aieditor"
	"theblock/site/internal/repo"
)

// /admin/uretim-hatti — AI üretim hattının canlı göstergesi.
//
// NEDEN VAR: hattın çalışıp çalışmadığı bugüne kadar ancak veritabanına
// elle sorgu atarak anlaşılabiliyordu. 13 Kasım 2026'da art arda ÜÇ
// sessiz arıza çıktı ve üçü de yalnız elle ölçümle görüldü:
//
//	medya kaydı yabancı anahtar hatası  — iki işçi aylarca hiçbir şey
//	                                      kaydedememiş; hata WARN olarak
//	                                      loglanıp tur devam ediyordu
//	ölü adresler kuyruğu tıkıyordu      — 49 kayıt 2012'lik kuyruğu
//	                                      durdurdu
//	sağlayıcı zaman aşımları            — günde ~103 iş kaybı, yalnız
//	                                      akşam saatlerinde
//
// Üçü de "her şey yeşil görünüyor ama hiçbir şey ilerlemiyor" türünden.
// Bu sayfa İLERLEME ile KAYBI yan yana koyuyor; biri olmadan diğeri
// yanıltıcı.
//
// TRAFİK VE SIRALAMA BURADA YOK: Search Console bağlı değil
// (settings trend.gsc_property boş). Yuvası ayrılmış ve neden boş
// olduğu sayfada yazıyor — sessizce atlamak "veri yok" ile "bağlantı
// yok" arasındaki farkı gizlerdi.

const (
	uretimHataEsigi  = 10 // yüzde: üstünde kırmızı
	uretimSaatGenis  = 14 // çubuk genişliği (px)
	uretimSaatLimiti = 24
)

func (s *Server) handleUretimHatti(w http.ResponseWriter, r *http.Request) {
	d := s.baseView("tr", "/admin", "Üretim Hattı")
	d.User = ptr(userFromContext(r.Context()))
	d.Flash = s.consumeFlash(r)
	d.SlotHTML = template.HTML(s.uretimHattiGovde(r.Context()))
	s.render(w, r, "admin_yardim", d)
}

func (s *Server) uretimHattiGovde(ctx context.Context) string {
	var b strings.Builder
	b.WriteString(`<div class="card card-pad"><h1 style="margin:0 0 4px">Üretim Hattı</h1>`)
	b.WriteString(`<p class="muted small" style="margin:0 0 14px">` +
		`Son 24 saat. İlerleme ve kayıp yan yana: biri olmadan diğeri yanıltıcı.</p>`)

	ozet, err := repo.UretimOzetOku(ctx, s.pool)
	if err != nil {
		b.WriteString(`<p class="hata">Özet okunamadı: ` + html.EscapeString(err.Error()) + `</p>`)
		return b.String() + `</div>`
	}

	b.WriteString(uretimKutular(ozet))
	b.WriteString(s.uretimTeshis(ctx, ozet))
	b.WriteString(s.uretimSaatlik(ctx))
	b.WriteString(s.uretimHatalar(ctx))
	b.WriteString(s.uretimCalisanlar(ctx))
	b.WriteString(s.uretimGorsel(ctx))
	b.WriteString(uretimTrafikYuvasi())
	b.WriteString(`</div>`)
	return b.String()
}

// uretimKutular üst sıradaki sayı kutuları.
func uretimKutular(o repo.UretimOzet) string {
	hedef := aieditor.UzunlukHedefiIcin("haber", "")
	toplam := o.Son24Tamam + o.Son24Hata
	hataYuzde := 0
	if toplam > 0 {
		hataYuzde = o.Son24Hata * 100 / toplam
	}

	kutu := func(baslik, deger, alt, sinif string) string {
		return `<div class="kutu ` + sinif + `" style="flex:1;min-width:150px;padding:12px;border:1px solid var(--cizgi,#333);border-radius:8px">` +
			`<div class="muted small">` + html.EscapeString(baslik) + `</div>` +
			`<div style="font-size:1.6rem;font-weight:600">` + html.EscapeString(deger) + `</div>` +
			`<div class="muted small">` + alt + `</div></div>`
	}

	var b strings.Builder
	b.WriteString(`<div style="display:flex;gap:10px;flex-wrap:wrap;margin-bottom:16px">`)
	b.WriteString(kutu("Şu an işleniyor", itoa(o.Calisan),
		"kuyrukta "+itoa(o.Kuyrukta), ""))
	b.WriteString(kutu("Tamamlanan (24s)", itoa(o.Son24Tamam), "başarılı iş", ""))

	hataSinif := ""
	if hataYuzde >= uretimHataEsigi {
		hataSinif = "uyari"
	}
	b.WriteString(kutu("Başarısız (24s)", itoa(o.Son24Hata),
		"<b>%"+itoa(hataYuzde)+"</b> kayıp", hataSinif))

	// MEDYAN UZUNLUK HEDEFLE BİRLİKTE: çıplak sayı "iyi mi kötü mü"
	// sorusunu yanıtlamıyor.
	uzunlukSinif := ""
	if o.Son24MedyanKelime > 0 && o.Son24MedyanKelime < hedef.AltSinir() {
		uzunlukSinif = "uyari"
	}
	b.WriteString(kutu("Medyan uzunluk", itoa(o.Son24MedyanKelime)+" kelime",
		"hedef "+itoa(hedef.MinKelime)+"-"+itoa(hedef.MaxKelime), uzunlukSinif))

	b.WriteString(kutu("Bekleyen taslak", itoa(o.BekleyenTaslak),
		itoa(o.Son24Uretilen)+" içerik üretildi (24s)", ""))
	b.WriteString(`</div>`)
	return b.String()
}

// uretimTeshis sayıları CÜMLEYE çevirir.
//
// NEDEN: bu oturumdaki üç arızanın hiçbiri tek bir sayıya bakarak
// görülmedi; hepsi iki sayının BİRLİKTE anlamından çıktı (kuyruk boş
// + bekleyen çok = besleme kesik). Paneli okuyan kişinin o bağlantıyı
// her seferinde kendi kurması gerekmesin.
func (s *Server) uretimTeshis(ctx context.Context, o repo.UretimOzet) string {
	var notlar []string

	toplam := o.Son24Tamam + o.Son24Hata
	if toplam > 0 {
		hataYuzde := o.Son24Hata * 100 / toplam
		if hataYuzde >= uretimHataEsigi {
			notlar = append(notlar,
				"İşlerin <b>%"+itoa(hataYuzde)+"</b>'i başarısız. Aşağıdaki hata "+
					"nedenlerine bakın; sağlayıcı kaynaklıysa eşzamanlılığı artırmak "+
					"kaybı büyütür.")
		}
	}
	if o.Kuyrukta == 0 && o.BekleyenTaslak > 50 {
		notlar = append(notlar,
			"Kuyruk <b>boş</b> ama <b>"+itoa(o.BekleyenTaslak)+"</b> taslak bekliyor: "+
				"hat işsiz değil, <b>besleme kesik</b>. Yazım otomatiği kapalı olabilir "+
				"ya da aday sorgusu taslakları süzüyor olabilir.")
	}
	hedef := aieditor.UzunlukHedefiIcin("haber", "")
	if o.Son24MedyanKelime > 0 && o.Son24MedyanKelime < hedef.AltSinir() {
		notlar = append(notlar,
			"Medyan uzunluk <b>"+itoa(o.Son24MedyanKelime)+"</b> kelime, hedefin "+
				"(<b>"+itoa(hedef.MinKelime)+"</b>) altında. Üretilen metin "+
				"\"ince içerik\" sınırına yakın.")
	}
	if len(notlar) == 0 {
		return `<p class="muted small" style="margin-bottom:14px">Dikkat gerektiren bir durum görünmüyor.</p>`
	}
	var b strings.Builder
	b.WriteString(`<div class="card card-pad" style="margin-bottom:16px;border-left:3px solid #c80"><b>Dikkat</b><ul style="margin:8px 0 0;padding-left:18px">`)
	for _, n := range notlar {
		b.WriteString(`<li style="margin-bottom:4px">` + n + `</li>`)
	}
	b.WriteString(`</ul></div>`)
	return b.String()
}

// uretimSaatlik saatlik çubuk dökümü.
//
// SAATLİK KIRILIM ŞART: sağlayıcı zaman aşımlarının yalnız akşam
// saatlerinde toplandığı ancak bu dağılıma bakınca görüldü; günlük
// toplam bunu gizliyordu.
func (s *Server) uretimSaatlik(ctx context.Context) string {
	satirlar, err := repo.UretimSaatlikOku(ctx, s.pool)
	if err != nil || len(satirlar) == 0 {
		return ""
	}
	enYuksek := 1
	for _, r := range satirlar {
		if n := r.Tamam + r.Hata; n > enYuksek {
			enYuksek = n
		}
	}
	var b strings.Builder
	b.WriteString(`<h2 style="font-size:1rem;margin:0 0 6px">Saatlik üretim</h2>`)
	b.WriteString(`<div style="display:flex;align-items:flex-end;gap:3px;height:90px;margin-bottom:6px">`)
	for _, r := range satirlar {
		tamamY := r.Tamam * 80 / enYuksek
		hataY := r.Hata * 80 / enYuksek
		baslik := r.Saat.Local().Format("15:04") + " — " +
			itoa(r.Tamam) + " tamam, " + itoa(r.Hata) + " hata"
		b.WriteString(`<div title="` + html.EscapeString(baslik) +
			`" style="width:` + itoa(uretimSaatGenis) + `px;display:flex;flex-direction:column;justify-content:flex-end">`)
		if hataY > 0 {
			b.WriteString(`<div style="height:` + itoa(hataY) + `px;background:#c33"></div>`)
		}
		if tamamY > 0 {
			b.WriteString(`<div style="height:` + itoa(tamamY) + `px;background:#3a7"></div>`)
		}
		b.WriteString(`</div>`)
	}
	b.WriteString(`</div>`)
	b.WriteString(`<p class="muted small" style="margin:0 0 16px">` +
		`<span style="color:#3a7">■</span> tamamlanan &nbsp; ` +
		`<span style="color:#c33">■</span> başarısız &nbsp;—&nbsp; ` +
		html.EscapeString(satirlar[0].Saat.Local().Format("02.01 15:04")) + ` → ` +
		html.EscapeString(satirlar[len(satirlar)-1].Saat.Local().Format("15:04")) + `</p>`)
	return b.String()
}

func (s *Server) uretimHatalar(ctx context.Context) string {
	hatalar, err := repo.UretimHatalariOku(ctx, s.pool, 6)
	if err != nil || len(hatalar) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(`<h2 style="font-size:1rem;margin:0 0 6px">Başarısızlık nedenleri (24s)</h2>`)
	b.WriteString(`<table class="tablo" style="width:100%;margin-bottom:16px"><thead><tr>` +
		`<th style="text-align:left">Neden</th><th style="width:70px">Adet</th>` +
		`<th style="width:110px">Son</th></tr></thead><tbody>`)
	for _, h := range hatalar {
		b.WriteString(`<tr><td style="font-family:monospace;font-size:11px">` +
			html.EscapeString(h.Neden) + `</td><td>` + itoa(h.Adet) + `</td><td class="muted small">` +
			html.EscapeString(h.Son.Local().Format("02.01 15:04")) + `</td></tr>`)
	}
	b.WriteString(`</tbody></table>`)
	return b.String()
}

func (s *Server) uretimCalisanlar(ctx context.Context) string {
	liste, err := repo.UretimCalisanlarOku(ctx, s.pool, 10)
	if err != nil || len(liste) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(`<h2 style="font-size:1rem;margin:0 0 6px">Şu an işlenen içerikler</h2>`)
	b.WriteString(`<table class="tablo" style="width:100%;margin-bottom:16px"><thead><tr>` +
		`<th style="text-align:left">İçerik</th><th style="width:110px">Eylem</th>` +
		`<th style="width:90px">Süre</th></tr></thead><tbody>`)
	for _, c := range liste {
		sure := "—"
		if !c.Baslangic.IsZero() {
			sure = uretimSure(time.Since(c.Baslangic))
		}
		baslik := html.EscapeString(kisaltMetin(c.Baslik, 70))
		if c.ContentID > 0 {
			baslik = `<a href="/admin/icerikler/` + itoa64(c.ContentID) + `">` + baslik + `</a>`
		}
		b.WriteString(`<tr><td>` + baslik + `</td><td class="muted small">` +
			html.EscapeString(c.Eylem) + `</td><td class="muted small">` + sure + `</td></tr>`)
	}
	b.WriteString(`</tbody></table>`)
	return b.String()
}

func (s *Server) uretimGorsel(ctx context.Context) string {
	aktarilan, toplam, err := repo.UretimGorselDurumu(ctx, s.pool)
	if err != nil || toplam == 0 {
		return ""
	}
	yuzde := aktarilan * 100 / toplam
	return `<h2 style="font-size:1rem;margin:0 0 6px">Görsel aktarımı</h2>` +
		`<div style="background:var(--cizgi,#333);height:10px;border-radius:5px;overflow:hidden;margin-bottom:4px">` +
		`<div style="width:` + itoa(yuzde) + `%;height:100%;background:#3a7"></div></div>` +
		`<p class="muted small" style="margin:0 0 16px">` + itoa(aktarilan) + ` / ` + itoa(toplam) +
		` kaynak görseli medya kütüphanesine alındı (%` + itoa(yuzde) + `).</p>`
}

// uretimTrafikYuvasi trafik/sıralama bölümünün NEDEN boş olduğunu yazar.
//
// Sessizce atlamak "veri yok" ile "bağlantı yok" arasındaki farkı
// gizlerdi; panelin amacı tam tersi.
func uretimTrafikYuvasi() string {
	return `<h2 style="font-size:1rem;margin:0 0 6px">Trafik ve sıralama</h2>` +
		`<p class="muted small" style="margin:0">Search Console bağlı değil ` +
		`(<code>trend.gsc_property</code> boş), bu yüzden üretim sonrası organik ` +
		`trafik ve sıralama değişimi ölçülemiyor. Marjinal fayda analizi — ` +
		`"bu içerik ne kazandırdı" — bu bağlantı kurulduğunda açılacak.</p>`
}

func uretimSure(d time.Duration) string {
	switch {
	case d < time.Minute:
		return itoa(int(d.Seconds())) + " sn"
	case d < time.Hour:
		return itoa(int(d.Minutes())) + " dk"
	default:
		return strconv.FormatFloat(d.Hours(), 'f', 1, 64) + " sa"
	}
}

func kisaltMetin(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

var _ = http.StatusOK
