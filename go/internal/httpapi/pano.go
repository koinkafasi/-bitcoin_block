package httpapi

import (
	"encoding/json"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"theblock/site/internal/repo"
	"theblock/site/internal/webui/templkit"
)

// ---- Pano 2.0 (Faz 149) ----
//
// Tek sayfa: /admin/pano. Eski /admin panosuna dokunulmadı — yeni düzen
// ayrı adreste büyüyor, beğenilince /admin'e taşınacak. Böylece iki düzen
// yan yana karşılaştırılabiliyor ve geri dönüş için bayrak gerekmiyor.

const panoAraliklar = 3 // 7 / 14 / 30 gün seçenekleri

// Yayın şeridi kart ölçüleri. Genişlik yüzde olarak sabit: kart içeriği
// başlık + saat, süre bilgisi yok — kartın gerçek süresi bilinmediği için
// genişliği süreye bağlamak yanıltıcı bir kesinlik üretirdi.
const (
	planKartGenislik = 15 // yüzde (36 saatlik eksende ~5,4 saat)
	panoPlanLaneMax  = 3  // dikey sıra sınırı (daha fazlası okunaksız)
)

// ---- İkon galerisi (Faz 153) ----

// ikonKart galerideki tek ikon.
type ikonKart struct{ Ad string }

// ikonAnim animasyon yardımcı sınıfı tanıtımı.
type ikonAnim struct {
	Sinif string
	Ad    string
	Not   string
	Ikon  string
}

// ikonSpinnerDemo satır içi basılmış spinner.
type ikonSpinnerDemo struct {
	Ad  string
	SVG template.HTML
}

// ikonlarData galeri verisi.
type ikonlarData struct {
	Toplam    int
	Kartlar   []ikonKart
	Animasyon []ikonAnim
	Spinner   []ikonSpinnerDemo
}

// handleIkonlar GET /admin/ikonlar — ikon kütüphanesi galerisi.
func (s *Server) handleIkonlar(w http.ResponseWriter, r *http.Request) {
	d := s.baseView("tr", "/admin", "İkon Kütüphanesi")
	d.User = ptr(userFromContext(r.Context()))
	d.Flash = s.consumeFlash(r)
	d.Ikonlar = s.buildIkonlarData()
	s.render(w, r, "ikonlar", d)
}

func (s *Server) buildIkonlarData() *ikonlarData {
	adlar := ikonAdlari()
	out := &ikonlarData{Toplam: len(adlar)}
	for _, a := range adlar {
		out.Kartlar = append(out.Kartlar, ikonKart{Ad: a})
	}
	out.Animasyon = []ikonAnim{
		{Sinif: "ico-spin", Ad: "Döndür", Not: "yenileme, derleme, yükleniyor", Ikon: "refresh"},
		{Sinif: "ico-spin-yavas", Ad: "Yavaş döndür", Not: "arka plan tazeleme", Ikon: "loader-2"},
		{Sinif: "ico-pulse", Ad: "Nabız", Not: "canlı yayın, bağlı, senkron", Ikon: "circle-dot"},
		{Sinif: "ico-ring", Ad: "Çıngırak", Not: "bildirim zili, dikkat çağrısı", Ikon: "bell"},
		{Sinif: "ico-bounce", Ad: "Zıpla", Not: "durum geçişi vurgusu", Ikon: "arrow-down-circle"},
		{Sinif: "ico-draw", Ad: "Çizilerek belir", Not: "ilk yüklemede zarif giriş", Ikon: "check"},
		{Sinif: "ico-shake", Ad: "Salla", Not: "hata geri bildirimi (tek sefer)", Ikon: "alert-triangle"},
		{Sinif: "ico-fade-in", Ad: "Belir", Not: "liste/kart ikonları sırayla", Ikon: "star"},
	}
	for _, ad := range ikonSpinnerAdlari() {
		out.Spinner = append(out.Spinner, ikonSpinnerDemo{Ad: ad, SVG: ikonSpinnerSVG(ad)})
	}
	return out
}

// handlePano ana ekranı çizer.
func (s *Server) handlePano(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sid := s.curSiteID(r)
	site := s.siteByDomain(ctx, r.Host)

	counts, _ := repo.CountByStatus(ctx, s.pool, sid)
	yayinda := counts["yayinda"]
	taslak := counts["taslak"]
	planli := counts["planlanmis"]

	hits := repo.SiteHits(ctx, s.pool, sid)

	// --- Okunma akışı: 7 / 14 / 30 gün. Üç grafik de sunucuda çizilir,
	// istemci yalnız görünürlüğü değiştirir (yeniden çizim = kayan nokta
	// farkı ve yeniden istek; ikisi de gereksiz).
	araliklar := []struct {
		key   string
		label string
		gun   int
	}{{"7", "7 gün", 7}, {"14", "14 gün", 14}, {"30", "30 gün", 30}}
	// Varsayılan 14: bugünün verisi + taban satırı görünür olsun.
	var seriler []templkit.PanoSeri
	var son14 []int
	for i, a := range araliklar {
		gunler := repo.HitsPerDay(ctx, s.pool, sid, a.gun)
		vals := make([]int, len(gunler))
		labels := make([]string, len(gunler))
		for j, d := range gunler {
			vals[j] = d.Count
			labels[j] = kisaGun(d.Day)
		}
		if a.gun == 14 {
			son14 = vals
		}
		seriler = append(seriler, templkit.PanoSeri{
			Key:   a.key,
			Label: a.label,
			Chart: panoAlanChart(panoSeri{Values: vals, Labels: labels}, 720, 220, a.key),
			On:    i == 1,
		})
	}

	// --- KPI döşemeleri ---
	buAy, gecenAy := repo.PublishTotals(ctx, s.pool, sid)
	son7, onceki7 := repo.HitTotals(ctx, s.pool, sid, 7)

	// Sparkline serisi: son 14 günün okunması (KPI'larda küçük grafik).
	spark := panoSpark(son14, 92, 30)

	kpiler := []templkit.PanoKPI{
		{Label: "Yayında", Value: itoa(yayinda), Note: "bu ay " + itoa64(buAy) + " yayın",
			Delta: panoDelta(int(buAy), int(gecenAy)), DeltaT: panoDeltaTone(int(buAy), int(gecenAy)),
			Icon: "file-text", Href: "/admin/icerikler?durum=yayinda", Tone: 1, Spark: spark},
		{Label: "Taslak", Value: itoa(taslak), Note: "yayına hazır değil",
			Icon: "pencil", Href: "/admin/icerikler?durum=taslak", Tone: 2},
		{Label: "Planlanmış", Value: itoa(planli), Note: "ileri tarihli",
			Icon: "clock", Href: "/admin/icerikler?durum=planlanmis", Tone: 3},
		{Label: "Okunma (7 gün)", Value: itoa(int(son7)), Note: "toplam " + itoa64(hits),
			Delta: panoDelta(int(son7), int(onceki7)), DeltaT: panoDeltaTone(int(son7), int(onceki7)),
			Icon: "eye", Href: "/admin/raporlar", Tone: 4},
	}

	// --- Kategori halkası ---
	var refs []CatSliceRef
	var legend []templkit.PanoDilimLegend
	for _, c := range repo.CategoryShare(ctx, s.pool, sid, 6) {
		refs = append(refs, CatSliceRef{Name: c.Name, Count: c.Count})
	}
	dilimler := panoDilimler(refs)
	for i, d := range dilimler {
		legend = append(legend, templkit.PanoDilimLegend{
			Name: d.Name, Count: d.Count, Pct: d.Pct, Tone: (i % 5) + 1,
		})
	}

	// --- Yayın ritmi (6 ay) ---
	aylar := repo.PublishPerMonth(ctx, s.pool, sid, 6)
	ayVals := make([]int, len(aylar))
	ayLbl := make([]string, len(aylar))
	for i, a := range aylar {
		ayVals[i] = a.Count
		ayLbl[i] = kisaAy(a.Day)
	}

	// --- En çok okunanlar ---
	var yatayRows []PanoCubuk
	if top, err := repo.TopRead(ctx, s.pool, sid, 5); err == nil && len(top) > 0 {
		ids := make([]int64, len(top))
		for i, c := range top {
			ids[i] = c.ID
		}
		h := repo.HitsOf(ctx, s.pool, ids)
		max := int64(1)
		for _, v := range h {
			if v > max {
				max = v
			}
		}
		for _, c := range top {
			yatayRows = append(yatayRows, PanoCubuk{
				Label: c.Title,
				Value: int(h[c.ID]),
				Pct:   int(h[c.ID] * 100 / max),
				Href:  "/admin/icerikler/" + itoa64(c.ID),
			})
		}
	}

	// --- Yayın şeridi ---
	//
	// İki mod var:
	//   1. "36 saat"  : canlı pencere (-6s … +30s). Planlı/yeni yayın varsa.
	//   2. "son yayınlar": pencere boşsa son yayınlanan içerikler, gerçek
	//      zaman aralığına yayılır. Boş kutu göstermek yerine gerçek veri
	//      gösterilir; eksen etiketi de moda göre değişir.
	now := time.Now()
	pencereBas := now.Add(-6 * time.Hour)
	pencereSon := now.Add(30 * time.Hour) // toplam 36 saat
	plan := repo.PlannedUpcoming(ctx, s.pool, sid, 24)
	planEtiket := "36 saat"

	if len(plan) == 0 {
		plan = repo.LastPublishedForPlan(ctx, s.pool, sid, 10)
		if len(plan) > 0 {
			planEtiket = "son yayınlar"
			// Pencereyi gerçek veriye oturt: en eskiden en yeniye, iki uca
			// yarım saat pay. Tek kayıt varsa pencere ±3 saat açılır, yoksa
			// kart sıfır genişlikli eksende ortada kalırdı.
			pencereBas, pencereSon = plan[0].When, plan[len(plan)-1].When
			if pencereSon.Sub(pencereBas) < time.Hour {
				pencereBas = pencereBas.Add(-3 * time.Hour)
				pencereSon = pencereSon.Add(3 * time.Hour)
			} else {
				pay := pencereSon.Sub(pencereBas) / 12
				pencereBas = pencereBas.Add(-pay)
				pencereSon = pencereSon.Add(pay)
			}
		}
	}

	toplamDk := pencereSon.Sub(pencereBas).Minutes()
	if toplamDk <= 0 {
		toplamDk = 1
	}
	simdiPct := int(now.Sub(pencereBas).Minutes() * 100 / toplamDk)
	simdiGorunur := simdiPct >= 0 && simdiPct <= 100
	var planRows []templkit.PanoPlanRow
	toneByStatus := map[string]string{
		"yayinda":    "ok",
		"planlanmis": "plan",
		"taslak":     "taslak",
	}
	for _, p := range plan {
		if p.When.Before(pencereBas) || p.When.After(pencereSon) {
			continue
		}
		sol := int(p.When.Sub(pencereBas).Minutes() * 100 / toplamDk)
		if sol < 0 {
			sol = 0
		}
		if sol > 96 {
			sol = 96
		}
		tone := toneByStatus[p.Status]
		if tone == "" {
			tone = "plan"
		}
		planRows = append(planRows, templkit.PanoPlanRow{
			ID: p.ID, Title: p.Title,
			Time:   p.When.Format("02.01 15:04"),
			Status: durumTR(p.Status), Tone: tone,
			Left: sol, Width: planKartGenislik,
		})
	}
	// Şeritleri (lane) ata: aynı saatte birden çok kart varsa üst üste
	// binmesinler. Kart genişliği sabit olduğu için çakışma kontrolü
	// yüzde aralığıyla yapılır.
	laneSonlari := make([]int, 0, 4)
	for i := range planRows {
		yer := -1
		for l := range laneSonlari {
			if planRows[i].Left >= laneSonlari[l] {
				yer = l
				break
			}
		}
		if yer < 0 {
			if len(laneSonlari) < panoPlanLaneMax {
				yer = len(laneSonlari)
				laneSonlari = append(laneSonlari, 0)
			} else {
				// Sıra doldu: en erken boşalan şeride koy (üst üste binsin,
				// ama tümüyle görünmez olmasın).
				yer = 0
				for l := range laneSonlari {
					if laneSonlari[l] < laneSonlari[yer] {
						yer = l
					}
				}
			}
		}
		planRows[i].Lane = yer
		laneSonlari[yer] = planRows[i].Left + planRows[i].Width + 1
	}
	planLanes := len(laneSonlari)
	if planLanes < 1 {
		planLanes = 1
	}

	// Eksen etiketleri: aynı gün içindeyse saat, günler geçiyorsa tarih.
	// Sabit "00/09/18/03/12" yazmak pencere kayınca yalan söylüyordu.
	planTicks := panoPlanTicks(pencereBas, pencereSon, 5)

	// --- Aktivite ---
	var akt []templkit.PanoActRow
	if logs, err := repo.ListLogs(ctx, s.pool, 7); err == nil {
		for _, l := range logs {
			meta := l.Action
			if l.Detail != nil && *l.Detail != "" {
				meta = *l.Detail
			}
			akt = append(akt, templkit.PanoActRow{
				Icon:  "activity",
				Title: l.UserName,
				Meta:  meta,
				When:  l.CreatedAt.Local().Format("15:04"),
			})
		}
	}

	// --- Yan menü: gerçek sayaçlarla ---
	nav := []templkit.PanoNavGrup{
		{Title: "Bugün", Items: []templkit.PanoNav{
			{Label: "Pano", Href: "/admin", Icon: "layout-dashboard", On: true},
			{Label: "İçerikler", Href: "/admin/icerikler", Icon: "file-text", Badge: itoa(lenAll(counts))},
			{Label: "Yeni içerik", Href: "/admin/icerikler/yeni", Icon: "plus"},
		}},
		{Title: "Yayın", Items: []templkit.PanoNav{
			{Label: "Resmî ilanlar", Href: "/admin/resmi-ilanlar", Icon: "speakerphone"},
			{Label: "Kategoriler", Href: "/admin/kategoriler", Icon: "category"},
			{Label: "Medya", Href: "/admin/dosyalar", Icon: "photo"},
			{Label: "Yorumlar", Href: "/admin/yorumlar", Icon: "message-2"},
			{Label: "Raporlar", Href: "/admin/raporlar", Icon: "chart-line"},
		}},
		{Title: "Sistem", Items: []templkit.PanoNav{
			{Label: "Ayarlar", Href: "/admin/ayarlar", Icon: "settings"},
			{Label: "Klasik pano", Href: "/admin", Icon: "layout-grid"},
		}},
	}

	// Yayın hedefi: son 30 günün günlük ortalaması — sabit bir sayı uydurmak
	// yerine gerçek ritimden türetilir. 7 günlük ortalama yayın olmayan
	// dönemde 0 çıkıp "hedef 1" gibi anlamsız bir gösterge üretiyordu.
	hedef := 1
	var ayTop, ayGun int
	if pd := repo.PublishPerDay(ctx, s.pool, sid, 30); len(pd) > 0 {
		for _, d := range pd {
			ayTop += d.Count
		}
		ayGun = len(pd)
	}
	if ayGun > 0 && ayTop/ayGun > 0 {
		hedef = ayTop / ayGun
	}
	bugun := 0
	if pd := repo.PublishPerDay(ctx, s.pool, sid, 1); len(pd) > 0 {
		bugun = pd[0].Count
	}

	v := templkit.PanoView{
		Selam:        selamSozu(now),
		Tarih:        turkceTarih(now),
		UserName:     kullaniciAdi(r),
		UserRole:     "Yönetici",
		SiteName:     site.Name,
		Nav:          nav,
		KPIs:         kpiler,
		Seriler:      seriler,
		OkunmaToplam: itoa64(hits),
		Halka:        panoHalka(dilimler, 190),
		HalkaLegend:  legend,
		Sutun:        panoSutun(panoSeri{Values: ayVals, Labels: ayLbl}, 340, 190),
		Yatay:        panoYatay(yatayRows),
		PlanRows:     planRows,
		PlanBos:      len(planRows) == 0,
		PlanSimdi:    now.Format("15:04"),
		PlanNowPct:   simdiPct,
		PlanNowVar:   simdiGorunur,
		PlanLanes:    planLanes,
		PlanEtiket:   planEtiket,
		PlanTicks:    planTicks,
		Aktivite:     akt,
		BosAkt:       len(akt) == 0,
		HedefBugun:   bugun,
		HedefGun:     hedef,
	}
	if fl := s.consumeFlash(r); fl != nil {
		v.FlashKind, v.FlashText = fl.Kind, fl.Text
	}

	d := s.baseView("tr", "/admin", "Pano")
	d.User = ptr(userFromContext(ctx))
	d.Flash = nil // pano kendi balonunu çizer
	d.Scripts = append(d.Scripts, "pano.js")
	// Yalnız pano'ya özgü iç stiller (KPI döşemeleri, grafikler, şerit).
	// Kabuk stilleri admin-pano.css'te ve panelin tamamında ortak.
	d.PageHead = template.HTML(`<link rel="stylesheet" href="` + s.assetURL("pano.css") + `">`)

	// Pano artık ORTAK kabuğu kullanıyor (admin_side/admin_top). Önceden
	// kendi aside/main/header'ını çiziyordu ve iki kabuk iki farklı görünüm
	// demekti: yan menü zemini, arama hapının biçimi, gövde boşlukları
	// tutmuyordu. Tek kabuk tek kaynak.
	s.renderTemplPage(w, r, d, templkit.PanoPanel(v))
}

// handlePanoTasi yayın şeridinde sürüklenen kartın saatini kaydeder.
//
// Girdi: id + saat (RFC3339 ya da "2006-01-02 15:04"). Yanıt JSON: istemci
// optimistic güncellemeyi geri alabilsin diye yeni zamanı döndürür.
func (s *Server) handlePanoTasi(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		panoJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "hata": "form okunamadı"})
		return
	}
	id := parseID(r.FormValue("id"))
	if id <= 0 {
		panoJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "hata": "geçersiz içerik"})
		return
	}
	ham := strings.TrimSpace(r.FormValue("saat"))
	when, err := panoSaatCoz(ham)
	if err != nil {
		panoJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "hata": "saat okunamadı"})
		return
	}
	n, err := repo.RescheduleContent(r.Context(), s.pool, s.curSiteID(r), id, when)
	if err != nil {
		s.logger.Error("pano taşıma", "error", err, "id", id)
		panoJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "hata": "kaydedilemedi"})
		return
	}
	if n == 0 {
		panoJSON(w, http.StatusNotFound, map[string]any{"ok": false, "hata": "içerik bulunamadı"})
		return
	}
	s.audit(r, "pano_tasi", strconv.FormatInt(id, 10)+" → "+when.Format("02.01 15:04"))
	panoJSON(w, http.StatusOK, map[string]any{
		"ok": true, "id": id, "saat": when.Format(time.RFC3339), "etiket": when.Format("02.01 15:04"),
	})
}

// panoSaatCoz sürükle-bırak girdisindeki saati çözer.
//
// Neden ayrı: form gövdesinde "+03:00" ofsetinin artı işareti boşluğa
// dönüşebilir ("2026-10-30T09:30:00 03:00"). İstemci URLSearchParams
// kullanıyor ve doğru kodluyor, ama elle/curl çağrısında da çalışması
// gerekiyor. Önce bilinen biçimler denenir; hepsi düşerse son boşluk artıya
// çevrilip RFC3339 bir kez daha denenir.
func panoSaatCoz(ham string) (time.Time, error) {
	ham = strings.TrimSpace(ham)
	layouts := []string{time.RFC3339, "2006-01-02T15:04", "2006-01-02 15:04", "2006-01-02T15:04:05", "2006-01-02"}
	var sonHata error
	for _, l := range layouts {
		if t, err := time.ParseInLocation(l, ham, time.Local); err == nil {
			return t, nil
		} else {
			sonHata = err
		}
	}
	if i := strings.LastIndex(ham, " "); i > 10 {
		duzeltilmis := ham[:i] + "+" + ham[i+1:]
		if t, err := time.Parse(time.RFC3339, duzeltilmis); err == nil {
			return t, nil
		}
	}
	return time.Time{}, sonHata
}

// panoJSON küçük JSON yanıtı.
func panoJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// panoPlanTicks şerit ekseninin etiketlerini üretir.
//
// Pencere aynı gün içindeyse saat ("18"), gün aşıyorsa tarih ("26.10")
// yazılır. Saat etiketi tek başına, günler geçen bir eksende hangi gün
// olduğunu söylemez — panelde "03" görüp üç gün sonrası mı üç saat sonrası mı
// olduğunu anlamamak gerçek bir karışıklıktı.
func panoPlanTicks(bas, son time.Time, n int) []templkit.PanoPlanTick {
	if n < 2 {
		n = 2
	}
	ayniGun := bas.Year() == son.Year() && bas.YearDay() == son.YearDay()
	span := son.Sub(bas)
	out := make([]templkit.PanoPlanTick, 0, n)
	for i := 0; i < n; i++ {
		pct := i * 100 / (n - 1)
		t := bas.Add(time.Duration(float64(span) * float64(i) / float64(n-1)))
		etiket := t.Format("15")
		if !ayniGun {
			etiket = t.Format("02.01")
		}
		out = append(out, templkit.PanoPlanTick{Left: pct, Label: etiket})
	}
	return out
}

// kisaGun "2026-10-29" → "29.10" (grafik ekseni).
func kisaGun(day string) string {
	if len(day) < 10 {
		return day
	}
	return day[8:10] + "." + day[5:7]
}

// kisaAy "2026-10" → "Eki" (grafik ekseni).
func kisaAy(ay string) string {
	if len(ay) < 7 {
		return ay
	}
	adlar := map[string]string{
		"01": "Oca", "02": "Şub", "03": "Mar", "04": "Nis", "05": "May", "06": "Haz",
		"07": "Tem", "08": "Ağu", "09": "Eyl", "10": "Eki", "11": "Kas", "12": "Ara",
	}
	if a, ok := adlar[ay[5:7]]; ok {
		return a
	}
	return ay[5:7]
}

// panoDelta iki sayı arasındaki değişimi rozet metnine çevirir.
//
// Önceki değer 0 ise oran tanımsızdır: "yeni" yazılır, "%∞" yanıltıcı olurdu.
// Oran 10 katı geçerse yüzde yerine KAT yazılır: "%4200" hem okunaksız hem
// de anlamsız (10 yayından 430'a çıkmak bir oran değil, ölçek değişimi).
func panoDelta(yeni, eski int) string {
	if eski <= 0 {
		if yeni > 0 {
			return "↑ yeni"
		}
		return ""
	}
	if yeni >= eski*10 {
		return "↑ " + strconv.Itoa(yeni/eski) + "×"
	}
	if eski >= yeni*10 && yeni > 0 {
		return "↓ " + strconv.Itoa(eski/yeni) + "×"
	}
	fark := float64(yeni-eski) * 100 / float64(eski)
	if fark >= 0 {
		return "↑ " + strconv.FormatFloat(fark, 'f', 1, 64) + "%"
	}
	return "↓ " + strconv.FormatFloat(-fark, 'f', 1, 64) + "%"
}

// panoDeltaTone rozetin rengini belirler (up | down | flat).
func panoDeltaTone(yeni, eski int) string {
	switch {
	case yeni > eski:
		return "up"
	case yeni < eski:
		return "down"
	}
	return "flat"
}

// selamSozu saate göre selamlama.
func selamSozu(t time.Time) string {
	switch h := t.Hour(); {
	case h < 6:
		return "İyi geceler"
	case h < 12:
		return "Günaydın"
	case h < 18:
		return "İyi günler"
	}
	return "İyi akşamlar"
}

// turkceTarih "29 Ekim 2026, Perşembe" biçiminde tarih.
func turkceTarih(t time.Time) string {
	aylar := []string{"Ocak", "Şubat", "Mart", "Nisan", "Mayıs", "Haziran",
		"Temmuz", "Ağustos", "Eylül", "Ekim", "Kasım", "Aralık"}
	gunler := []string{"Pazar", "Pazartesi", "Salı", "Çarşamba", "Perşembe", "Cuma", "Cumartesi"}
	return itoa(t.Day()) + " " + aylar[int(t.Month())-1] + " " + itoa(t.Year()) +
		", " + gunler[int(t.Weekday())]
}

// durumTR içerik durumunu okunur Türkçeye çevirir.
func durumTR(st string) string {
	switch st {
	case "yayinda":
		return "Yayında"
	case "planlanmis":
		return "Planlanmış"
	case "taslak":
		return "Taslak"
	case "arsiv":
		return "Arşiv"
	}
	return st
}

// kullaniciAdi oturumdaki kullanıcının adını verir.
func kullaniciAdi(r *http.Request) string {
	u := userFromContext(r.Context())
	if strings.TrimSpace(u.Name) != "" {
		return u.Name
	}
	if strings.TrimSpace(u.Email) != "" {
		return u.Email
	}
	return "editör"
}
