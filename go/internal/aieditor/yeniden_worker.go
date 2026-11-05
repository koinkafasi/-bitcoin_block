package aieditor

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgxpool"

	"theblock/site/internal/ai"
	"theblock/site/internal/aieditor/research"
	"theblock/site/internal/platform"
	"theblock/site/internal/repo"
)

// Toplu AI ile yeniden yazım işçisi (Faz 204).
//
// ÖLÇÜM (üretim, 3 Kasım 2026): yayındaki 986 içeriğin 982'si tek paragraf,
// 602'si 200 karakterin altında. Faz 194 hattı yalnız YENİ taslaklara
// uygulanıyor; yayındaki metinlere hiç dokunulmadı.
//
// Bu işçi öneri ÜRETİR, yayına YAZMAZ:
//
//	bekleyen öneri → AI (dil = içeriğin dili) → kalite kapısı
//	  → geçti:  `yeniden_yazim.status = hazir`  (panelde eski/yeni diff)
//	  → geçmedi: `status = hata` + gerekçe      (içerik değişmez)
//
// Yazma yalnız editör onayında olur (`repo.YenidenYazimOnayla`, sürüm
// korumalı). EN satırları da kendi İngilizce metninden yeniden yazılır:
// çeviri yolunda güncelleme yolu yok, ayrıca İngilizce metnin kendi
// akıcılığı korunmalı.

const (
	// yenidenMinParagraf: haber gövdesi en az bu kadar paragraf olmalı.
	yenidenMinParagraf = 3
	// yenidenMinKarakter: altındaki metin "ince içerik" sayılır.
	yenidenMinKarakter = 700
	// yenidenTabanKarakter: kaynağı çok kısa metinler için alt sınır.
	// (Sabit 700 eşiği 20/20 pilot önerisini reddetti; gerekçe kapıda.)
	yenidenTabanKarakter = 300
	// yenidenKisaKaynak: bu uzunluğun altındaki kaynak "tek cümlelik özet"
	// sayılır. O kaynaktan 3+ paragraf istemek dolgu cümle üretir.
	yenidenKisaKaynak = 250
	// yenidenKisaTaban: kısa kaynakta gövde alt sınırı (dolgu istemeden).
	yenidenKisaTaban = 250
	// yenidenKesinTaban: asla altına inilmeyecek gövde uzunluğu.
	//
	// Neden 180: ölçüm (pilot, 3 Kasım 2026) — kaynak 94-228 karakterlik tek
	// cümlelik özet. "Yeni olgu yok + dolgu yok" kuralıyla model 180-250
	// karakterlik 2-3 paragraf üretiyor. Bunun altı gerçekten "içerik yok";
	// üstü ise bugünkü 1 paragraflık metinden yapısal olarak iyi. Uzunluk
	// kararı (700 haber standardı) UYARI olarak editöre bırakılır — yayına
	// yazma zaten onayla oluyor.
	yenidenKesinTaban = 180
	// yenidenMaxParagraf: fazlası tek haber için bölünmemiş yığın olur.
	yenidenMaxParagraf = 9
	// yenidenEszamanli aynı anda uçuşta olacak AI çağrısı sayısı.
	yenidenEszamanli = 2
)

// sayiRe metindeki sayıları yakalar (uydurma sayı denetimi için).
var sayiRe = regexp.MustCompile(`\d+(?:[.,]\d+)*`)

// RunYenidenYazim öneri üretim döngüsünü başlatır.
//
// Ayar yok: öneri üretimi yalnız editörün "pilotu başlat" isteğiyle kuyruğa
// girer. Döngü kuyruk boşken hiçbir AI çağrısı yapmaz.
func RunYenidenYazim(ctx context.Context, pool *pgxpool.Pool, aiclient *ai.Client, logger *slog.Logger, every time.Duration) {
	if every <= 0 {
		every = 10 * time.Second
	}
	logger.Info("yeniden yazım işçisi başladı", "her", every.String())
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			logger.Info("yeniden yazım işçisi durdu")
			return
		case <-ticker.C:
			if aiclient == nil || !aiclient.Enabled() {
				continue
			}
			if n := yenidenTur(ctx, pool, aiclient, logger); n > 0 {
				logger.Info("yeniden yazım turu", "islenen", n)
			}
		}
	}
}

// YenidenBirTur bekleyen önerilerden bir partiyi işler ve işlenen sayısını
// döndürür. Döngü (`RunYenidenYazim`) bunu kendi zamanlayıcısıyla çağırır;
// komut satırı aracı (pilot ölçümü) doğrudan çağırıp kuyruk boşalana kadar
// yineleyebilir.
func YenidenBirTur(ctx context.Context, pool *pgxpool.Pool, aiclient *ai.Client, logger *slog.Logger) int {
	return yenidenTur(ctx, pool, aiclient, logger)
}

// yenidenTur bekleyen önerileri sahiplenip eşzamanlı işler.
func yenidenTur(ctx context.Context, pool *pgxpool.Pool, aiclient *ai.Client, logger *slog.Logger) int {
	isler, err := repo.YenidenYazimSahiplen(ctx, pool, yenidenEszamanli*3)
	if err != nil {
		logger.Warn("yeniden yazım: kuyruk okunamadı", "error", err)
		return 0
	}
	if len(isler) == 0 {
		return 0
	}
	// Arama sağlayıcısı tur başına bir kez kurulur (env okuması ucuz).
	arastirma := research.FromEnv(logger)
	sinir := make(chan struct{}, yenidenEszamanli)
	var wg sync.WaitGroup
	for _, is := range isler {
		if ctx.Err() != nil {
			break
		}
		wg.Add(1)
		sinir <- struct{}{}
		go func(is repo.YenidenYazimIs) {
			defer wg.Done()
			defer func() { <-sinir }()
			yenidenIsle(ctx, pool, aiclient, logger, arastirma, is)
		}(is)
	}
	wg.Wait()
	return len(isler)
}

// yenidenIsle tek öneri: (gerekirse) araştırma, AI çağrısı, kalite kapısı, kayıt.
func yenidenIsle(ctx context.Context, pool *pgxpool.Pool, aiclient *ai.Client, logger *slog.Logger, arastirma research.Provider, is repo.YenidenYazimIs) {
	bas := time.Now()
	eskiParagraflar := repo.GovdeParagraflari(is.Detail)
	kaynakMetin := strings.Join(eskiParagraflar, "\n\n")

	// Araştırmalı mod: gerçek web kaynakları toplanır.
	//
	// Neden zorunlu: yayındaki ince içeriğin kaynağı tek cümlelik RSS özeti.
	// Model o cümleden 700 karakterlik haber üretemez — üretirse uydurur.
	// Kaynak bulunamazsa öneri HATA olur; uydurulmuş bir "haber" yazmaktansa
	// hiç yazmamak doğrudur.
	var kaynaklar []repo.YenidenKaynak
	if is.Mod == repo.YenidenModArastirmali {
		toplanan, metin, err := yenidenArastir(ctx, arastirma, is)
		if err != nil {
			logger.Info("yeniden yazım: araştırma yapılamadı",
				"öneri", is.ID, "içerik", is.ContentID, "error", err)
			_ = repo.YenidenYazimHata(ctx, pool, is.ID, "araştırma: "+err.Error())
			return
		}
		kaynaklar = toplanan
		kaynakMetin = kaynakMetin + "\n\n" + metin
	}

	sonuc, resp, err := yenidenUret(ctx, aiclient, is.Locale, is.Title, is.Spot, kaynakMetin)
	if err != nil {
		// Sağlayıcı zaman aşımı/5xx geçici olabilir: deneme sınırına kadar
		// geri kuyruğa konur, sınır aşılırsa kalıcı hata olur.
		tekrar, herr := repo.YenidenYazimGeciciHata(ctx, pool, is.ID, "AI: "+err.Error())
		if herr != nil {
			logger.Warn("yeniden yazım: hata durumu yazılamadı", "öneri", is.ID, "error", herr)
		}
		logger.Warn("yeniden yazım: AI hatası",
			"öneri", is.ID, "içerik", is.ContentID, "yeniden_denenecek", tekrar, "error", err)
		return
	}

	uyarilar, engel := yenidenKapisi(sonuc, kaynakMetin, is.SourceURL)
	if engel != "" {
		// Teşhis alanları: ham yanıt uzunluğu ve paragraf sayısı. Kapı
		// reddettiğinde "model kısa yazdı" ile "ayrıştırma metni düşürdü"
		// birbirinden ayrılabilsin.
		logger.Info("yeniden yazım: kalite kapısı reddetti",
			"öneri", is.ID, "içerik", is.ContentID, "engel", engel,
			"ham_karakter", utf8.RuneCountInString(resp.Text),
			"paragraf", len(sonuc.Paragraflar),
			"kaynak_karakter", utf8.RuneCountInString(kaynakMetin),
			"mod", is.Mod, "kaynak_sayisi", len(kaynaklar))
		_ = repo.YenidenYazimHata(ctx, pool, is.ID, "kalite: "+engel)
		return
	}

	if err := repo.YenidenYazimKaydet(ctx, pool, is.ID, repo.YenidenYazimSonucu{
		Spot:     sonuc.Spot,
		Paragraf: sonuc.Paragraflar,
		SeoTitle: platform.SeoKirp(sonuc.SeoBaslik, 70),
		SeoDesc:  platform.SeoKirp(sonuc.SeoAciklama, 165),
	}, uyarilar, kaynaklar, resp.Provider, resp.Model, resp.PromptTokens, resp.CompletionTokens); err != nil {
		logger.Warn("yeniden yazım: sonuç yazılamadı", "öneri", is.ID, "error", err)
		_ = repo.YenidenYazimKuyrukBirak(ctx, pool, is.ID)
		return
	}

	// Maliyet/denetim kaydı: hangi içerik, hangi model, kaç token.
	_, _ = repo.LogAIUsage(ctx, pool, &repo.AIUsage{
		UserID:           nil,
		ContentID:        &is.ContentID,
		Provider:         resp.Provider,
		Action:           "yeniden_yazim",
		Model:            &resp.Model,
		PromptVersion:    ptrStr(PromptVersion + "-yeniden"),
		PromptTokens:     resp.PromptTokens,
		CompletionTokens: resp.CompletionTokens,
		CostEst:          yenidenMaliyet(resp.PromptTokens, resp.CompletionTokens),
		LatencyMS:        int(time.Since(bas).Milliseconds()),
	})
	logger.Info("yeniden yazım: öneri hazır",
		"öneri", is.ID, "içerik", is.ContentID, "dil", is.Locale,
		"paragraf", len(sonuc.Paragraflar), "uyarı", len(uyarilar),
		"sure_ms", time.Since(bas).Milliseconds())
}

// yenidenArastir öneri için gerçek web kaynakları toplar.
//
// Sorgular başlıktan türetilir: başlık zaten haberin öznesini taşıyor.
// İkinci sorgu spot'u ekler (başlık kısaysa isabet artar). Sayfa metinleri
// çekilir (`fetchPages=true`): modelin kanıtı özet değil, kaynağın kendisi
// olmalı.
func yenidenArastir(ctx context.Context, p research.Provider, is repo.YenidenYazimIs) ([]repo.YenidenKaynak, string, error) {
	if p == nil || p.Name() == "disabled" {
		return nil, "", fmt.Errorf("arama sağlayıcısı yapılandırılmadı (BRAVE_SEARCH_API_KEY)")
	}
	sorgular := []string{temizSorgu(is.Title)}
	if ek := temizSorgu(is.Spot); ek != "" && !strings.Contains(sorgular[0], ek) {
		sorgular = append(sorgular, sorgular[0]+" "+ek)
	}
	items, err := research.Collect(ctx, p, sorgular, 5, true)
	if err != nil {
		return nil, "", err
	}
	if len(items) == 0 {
		return nil, "", fmt.Errorf("kaynak bulunamadı")
	}

	kaynaklar := make([]repo.YenidenKaynak, 0, len(items))
	var sb strings.Builder
	for i, it := range items {
		ref := fmt.Sprintf("S%d", i+1)
		metin := strings.TrimSpace(it.Content)
		if metin == "" {
			metin = strings.TrimSpace(it.Description)
		}
		tarih := ""
		if it.PublishedAt != nil {
			tarih = it.PublishedAt.Format("2006-01-02")
		}
		kaynaklar = append(kaynaklar, repo.YenidenKaynak{
			Ref: ref, Baslik: it.Title, URL: it.URL, Yayinci: it.Publisher,
			Ozet: kisalt(it.Description, 300), YayinTari: tarih,
		})
		sb.WriteString("[" + ref + "] " + it.Publisher + " — " + it.Title + "\n")
		sb.WriteString(it.URL + "\n")
		sb.WriteString(kisalt(metin, 2200) + "\n\n")
	}
	return kaynaklar, sb.String(), nil
}

// temizSorgu arama sorgusunu sadeleştirir: tırnak, site adı ve fazla
// noktalama arama motorunu şaşırtıyor.
func temizSorgu(s string) string {
	s = strings.ReplaceAll(s, "\"", " ")
	s = strings.ReplaceAll(s, "’", "'")
	s = strings.Join(strings.Fields(s), " ")
	r := []rune(s)
	if len(r) > 140 {
		s = string(r[:140])
	}
	return strings.TrimSpace(s)
}

// yenidenYanit AI'nın döndürdüğü yeni metindir.
type yenidenYanit struct {
	Spot        string
	Paragraflar []string
	SeoBaslik   string
	SeoAciklama string
}

// yenidenUret sağlayıcıdan yeniden yazılmış gövdeyi ister.
//
// `CompleteStrict`: mock yanıtı kabul edilmez — pilot ölçümü gerçek model
// çıktısıyla yapılmalı, aksi hâlde "kalite" ölçümü anlamsız olur.
func yenidenUret(ctx context.Context, aiclient *ai.Client, locale, baslik, spot, govde string) (yenidenYanit, ai.Response, error) {
	// Kaynak listesi varsa istem "araştırmalı" olur: model yalnız aşağıdaki
	// kaynak metinlerine dayanır ve uzunluk hedefi gerçekten yüksektir.
	arastirmali := strings.Contains(govde, "[S1]")
	sistem, kullanici := yenidenIstem(locale, baslik, spot, govde, arastirmali)
	resp, err := aiclient.CompleteStrict(ctx, ai.Request{
		Messages: []ai.Msg{
			{Role: ai.RoleSystem, Content: sistem},
			{Role: ai.RoleUser, Content: kullanici},
		},
		JSONMode:    true,
		MaxTokens:   3000,
		Temperature: 0.4,
	})
	if err != nil {
		return yenidenYanit{}, resp, err
	}
	y, err := yenidenJSON(resp.Text)
	if err != nil {
		return yenidenYanit{}, resp, err
	}
	return y, resp, nil
}

// yenidenIstem dil başına istem üretir.
//
// Kural: YENİ OLGU YOK. Model yalnız var olan bilgiyi yeniden düzenler ve
// genişletir (bağlam, geçiş, açıklama). Kaynak adı, "devamını oku" kalıntısı
// ve HTML yasak.
func yenidenIstem(locale, baslik, spot, govde string, arastirmali bool) (string, string) {
	if locale == "en" {
		sistem := `You are an English news editor at an English-language news site.
Rules:
- Output only JSON: {"spot": "...", "paragraphs": ["...", ...], "seo_title": "...", "meta_description": "..."}
- Rewrite the article into 2-4 paragraphs of plain prose. Use ONLY facts in the source text.
- Every sentence must carry information from the source. No filler, no generic commentary,
  no scene-setting that the source does not contain. A short source gives a short article.
- Never invent names, numbers, dates, quotes, causes or outcomes. Do not add background that is not in the source.
- Do not repeat any sentence. Do not add headings, lists, HTML tags or entities, or "read more" lines.
- Do not mention the source website or agency name.
- spot: 120-160 characters. seo_title: 40-70 characters. meta_description: 120-160 characters.
- Keep proper nouns and existing figures exactly as given.`
		kullanici := fmt.Sprintf("Headline: %s\n\nSummary: %s\n\nArticle text:\n%s", baslik, spot, govde)
		if arastirmali {
			sistem = `You are an English news editor at an English-language news site.
Rules:
- Output only JSON: {"spot": "...", "paragraphs": ["...", ...], "seo_title": "...", "meta_description": "..."}
- Below are numbered REAL SOURCES. Rewrite the story into 4-6 paragraphs, 900-1600 characters.
- Use ONLY information stated in these sources. Never invent names, figures, dates, quotes, causes or outcomes.
- If sources disagree, follow the majority; if unsure, omit the detail.
- Do not name the source outlets or agencies ("according to ..."). Official institutions may be named.
- Do not repeat any sentence. No headings, lists, HTML tags/entities, or "read more" lines.
- spot: 120-160 characters. seo_title: 40-70 characters. meta_description: 120-160 characters.
- Keep proper nouns and figures exactly as given in the sources.`
			kullanici = fmt.Sprintf("Headline: %s\n\nCurrent summary: %s\n\nSOURCES:\n%s", baslik, spot, govde)
		}
		return sistem, kullanici
	}
	sistem := `Sen Türkçe yayın yapan bir haber sitesinin editörüsün.
Kurallar:
- Yalnızca JSON döndür: {"spot": "...", "paragraphs": ["...", ...], "seo_title": "...", "meta_description": "..."}
- Haberi 2-4 paragraf düz metne dönüştür. YALNIZCA kaynak metindeki bilgileri kullan.
- Her cümle kaynaktaki bir bilgiyi taşımalı. Dolgu cümle, genel yorum, kaynakta olmayan
  sahne kurulumu YASAK. Kaynak kısaysa haber de kısa olur.
- Yeni isim, sayı, tarih, alıntı, neden veya sonuç UYDURMA. Kaynakta olmayan arka plan ekleme.
- Aynı cümleyi tekrarlama. Alt başlık, madde işareti, HTML etiketi/varlığı, "devamını oku" satırı ekleme.
- Kaynak site veya ajans adını yazma.
- spot: 120-160 karakter. seo_title: 40-70 karakter. meta_description: 120-160 karakter.
- Özel adları ve mevcut sayıları aynen koru; Türkçe karakterleri doğru yaz.`
	kullanici := fmt.Sprintf("Başlık: %s\n\nSpot: %s\n\nHaber metni:\n%s", baslik, spot, govde)
	if arastirmali {
		sistem = `Sen Türkçe yayın yapan bir haber sitesinin editörüsün.
Kurallar:
- Yalnızca JSON döndür: {"spot": "...", "paragraphs": ["...", ...], "seo_title": "...", "meta_description": "..."}
- Aşağıda numaralı GERÇEK KAYNAKLAR var. Haberi 4-6 paragraf, 900-1600 karakter olarak YENİDEN YAZ.
- Yalnızca bu kaynaklarda yazan bilgiyi kullan. Kaynaklarda olmayan isim, sayı, tarih, alıntı, neden, sonuç UYDURMA.
- Bilgi çelişirse en çok kaynağın söylediğini yaz; emin değilsen o ayrıntıyı atla.
- Kaynak/ajans adını ("...'a göre", "...com.tr") yazma. Kurum adları (TÜİK, AA gibi resmî kurum) yazılabilir.
- Aynı cümleyi tekrarlama. Alt başlık, madde işareti, HTML etiketi/varlığı, "devamını oku" satırı ekleme.
- spot: 120-160 karakter. seo_title: 40-70 karakter. meta_description: 120-160 karakter.
- Türkçe karakterleri doğru yaz.`
		kullanici = fmt.Sprintf("Başlık: %s\n\nMevcut spot: %s\n\nKAYNAKLAR:\n%s", baslik, spot, govde)
	}
	return sistem, kullanici
}

// yenidenJSON model yanıtını ayrıştırır; `paragraphs` yoksa `body`
// metnini satırlara böler (model zaman zaman düz metin döndürüyor).
func yenidenJSON(ham string) (yenidenYanit, error) {
	m, err := parseJSONObject(ham)
	if err != nil {
		return yenidenYanit{}, err
	}
	y := yenidenYanit{
		Spot:        metinDegeri(m["spot"]),
		SeoBaslik:   metinDegeri(m["seo_title"]),
		SeoAciklama: metinDegeri(m["meta_description"]),
	}
	for _, p := range metinDizisi(m["paragraphs"]) {
		if p = strings.TrimSpace(p); p != "" {
			y.Paragraflar = append(y.Paragraflar, p)
		}
	}
	if len(y.Paragraflar) == 0 {
		for _, p := range strings.Split(metinDegeri(m["body"]), "\n") {
			if p = strings.TrimSpace(p); p != "" {
				y.Paragraflar = append(y.Paragraflar, p)
			}
		}
	}
	if len(y.Paragraflar) == 0 {
		return yenidenYanit{}, fmt.Errorf("yanıtta paragraf yok")
	}
	return y, nil
}

func metinDegeri(v any) string {
	if s, ok := v.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}

func metinDizisi(v any) []string {
	liste, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(liste))
	for _, e := range liste {
		if s, ok := e.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// yenidenKapisi AI çıktısının yapı/kalite denetimi.
//
// Dönüş: (uyarılar, engel). `engel` doluysa öneri yazılmaz (editöre
// yalnız gerekçe görünür) — uyarı ise öneriyle birlikte gösterilir.
func yenidenKapisi(y yenidenYanit, kaynakMetin, sourceURL string) ([]string, string) {
	var uyarilar []string
	govde := strings.Join(y.Paragraflar, "\n\n")

	// Paragraf alt sınırı da KAYNAĞA göre ölçeklenir: 250 karakterin
	// altındaki bir kaynaktan 4 paragraf çıkarmak dolgu cümle üretmek demek.
	// Ölçüm (pilot, 3 Kasım 2026): EN metinlerde 4 paragraf zorlanınca
	// "She emphasized how their dedication impacts the nation." gibi
	// bilgi taşımayan cümleler çıktı.
	kaynakK := utf8.RuneCountInString(strings.TrimSpace(kaynakMetin))
	minParagraf := yenidenMinParagraf
	if kaynakK < yenidenKisaKaynak {
		minParagraf = 2
	}
	if n := len(y.Paragraflar); n < minParagraf {
		return nil, fmt.Sprintf("gövde %d paragraf; en az %d gerekir", n, minParagraf)
	} else if n > yenidenMaxParagraf {
		uyarilar = append(uyarilar, fmt.Sprintf("gövde %d paragraf; %d üzeri bölünmeli", n, yenidenMaxParagraf))
	}

	// Uzunluk eşiği KAYNAĞA göre ölçeklenir.
	//
	// SORUN (pilot ölçümü, 3 Kasım 2026): yayındaki ince içeriklerin kaynağı
	// 94-226 karakterlik RSS özeti. Sabit 700 karakter eşiği 20/20 öneriyi
	// reddetti — model 300-540 karakter yazdı ve 700'e çıkmak için olgu
	// uydurmak gerekirdi. "Yeni olgu yok" kuralı ile sabit 700 çelişiyor.
	//
	// ÇÖZÜM: en az 300 karakter (kaynak 250'nin altındaysa 250) ve kaynağın
	// 1,5 katı, tavan 700. Böylece metin gerçekten genişler ama dolgu ya da
	// uydurmaya zorlanmaz; 700'ün altında kalan öneri UYARI olarak görünür.
	k := utf8.RuneCountInString(strings.TrimSpace(govde))
	hedef := yenidenTabanKarakter
	if kaynakK < yenidenKisaKaynak {
		hedef = yenidenKisaTaban
	}
	if n := int(float64(kaynakK) * 1.5); n > hedef {
		hedef = n
	}
	if hedef > yenidenMinKarakter {
		hedef = yenidenMinKarakter
	}
	// Alt sınır: hiçbir koşulda "tek cümle + iki dolgu" kabul edilmez.
	if hedef < yenidenKesinTaban {
		hedef = yenidenKesinTaban
	}
	if k < hedef {
		return nil, fmt.Sprintf("gövde %d karakter; bu kaynak için en az %d gerekir", k, hedef)
	}
	if k < yenidenMinKarakter {
		uyarilar = append(uyarilar, fmt.Sprintf(
			"gövde %d karakter; haber standardı %d. Kaynak kısa olduğu için daha fazlası uydurma gerektirir.",
			k, yenidenMinKarakter))
	}

	tum := strings.Join([]string{y.Spot, govde}, " ")
	if m := varlikRe.FindString(tum); m != "" {
		return nil, "metinde HTML varlığı var (" + m + ")"
	}
	if m := htmlEtiket.FindString(tum); m != "" {
		return nil, "metinde HTML etiketi var (" + m + ")"
	}
	if m := kalintiRe.FindString(tum); m != "" {
		return nil, "metinde yönlendirme kalıntısı var (\"" + m + "\")"
	}
	if t := tekrarCumle(y.Paragraflar); t != "" {
		return nil, "aynı cümle birden çok kez geçiyor: \"" + kisalt(t, 60) + "\""
	}

	// Kaynak alan adı sızıntısı: yalnız gerçek kaynak adresi varsa aranır.
	if sourceURL != "" {
		kaynaklar := []Source{{URL: sourceURL}}
		if s := kaynakSizintisi(tum, kaynaklar); s != "" {
			return nil, "metinde kaynak site adı geçiyor (\"" + s + "\")"
		}
	}

	// Uydurma sayı denetimi: kaynakta olmayan bir sayı varsa ENGELLEME,
	// uyarı ver. Yeniden yazımda sayıların yazıyla açıklanması meşru
	// olabilir; editör diff'te görüp karar verir.
	if yeni := bilinmeyenSayilar(govde, kaynakMetin); len(yeni) > 0 {
		uyarilar = append(uyarilar, "kaynak metinde geçmeyen sayı(lar): "+strings.Join(yeni, ", "))
	}
	if k := utf8.RuneCountInString(y.Spot); k > 0 && (k < 90 || k > 200) {
		uyarilar = append(uyarilar, fmt.Sprintf("spot %d karakter; 120-160 önerilir", k))
	}
	if y.SeoBaslik == "" {
		uyarilar = append(uyarilar, "SEO başlığı üretilmedi")
	}
	if y.SeoAciklama == "" {
		uyarilar = append(uyarilar, "meta açıklama üretilmedi")
	}
	return uyarilar, ""
}

// bilinmeyenSayilar kaynakta geçmeyen sayıları döndürür (en çok 5).
func bilinmeyenSayilar(yeni, kaynak string) []string {
	var out []string
	gorulen := map[string]bool{}
	for _, s := range sayiRe.FindAllString(yeni, -1) {
		if gorulen[s] || strings.Contains(kaynak, s) {
			continue
		}
		gorulen[s] = true
		out = append(out, s)
		if len(out) >= 5 {
			break
		}
	}
	return out
}

// yenidenMaliyet kaba maliyet tahmini (USD). Panelde büyüklük sırası verir;
// sağlayıcının gerçek faturası değildir.
func yenidenMaliyet(prompt, completion int) float64 {
	const (
		girdiMilyon = 0.15
		ciktiMilyon = 0.60
	)
	return float64(prompt)/1e6*girdiMilyon + float64(completion)/1e6*ciktiMilyon
}

func ptrStr(s string) *string { return &s }
