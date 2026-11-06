package ceviri

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"theblock/site/internal/repo"
)

// Otomatik çeviri (Faz 189).
//
// Sahadaki sorun: 24 dil açık, /en 200 dönüyor ama ana sayfada manşet YOK
// (TR'de 150 hero öğesi, EN'de 0) ve başlık Türkçe kalıyordu. Sebep içerik
// sorgularının `locale = $1` ile sıkı filtrelenmesi ve hiç İngilizce
// içerik olmaması. Bu paket o boşluğu doldurur.

func TestParagraflarBlokDizisi(t *testing.T) {
	detail := []byte(`[{"type":"p","text":"Birinci"},{"type":"p","text":"İkinci"}]`)
	p := Paragraflar(detail)
	assert.Equal(t, []string{"Birinci", "İkinci"}, p)
}

func TestParagraflarTipTap(t *testing.T) {
	detail := []byte(`{"type":"doc","content":[
		{"type":"paragraph","content":[{"type":"text","text":"Merhaba "},{"type":"text","text":"dünya"}]},
		{"type":"heading","content":[{"type":"text","text":"Başlık"}]}
	]}`)
	p := Paragraflar(detail)
	require.Len(t, p, 2)
	assert.Equal(t, "Merhaba dünya", p[0])
	assert.Equal(t, "Başlık", p[1])
}

func TestParagraflarBos(t *testing.T) {
	assert.Nil(t, Paragraflar(nil))
	assert.Nil(t, Paragraflar([]byte{}))
	assert.Nil(t, Paragraflar([]byte("çöp")))
}

func TestBloklarGidisDonus(t *testing.T) {
	giris := []string{"Birinci paragraf", "", "İkinci paragraf"}
	b := Bloklar(giris)
	var bloklar []map[string]any
	require.NoError(t, json.Unmarshal(b, &bloklar))
	// Boş paragraf atlanır.
	assert.Len(t, bloklar, 2)
	assert.Equal(t, "p", bloklar[0]["type"])
	assert.Equal(t, "Birinci paragraf", bloklar[0]["text"])

	// Geri okuma aynı metni verir.
	assert.Equal(t, []string{"Birinci paragraf", "İkinci paragraf"}, Paragraflar(b))
}

func TestCeviriJSONSarmasi(t *testing.T) {
	// Model JSON'u ```json bloğuna sarabiliyor veya başına cümle ekliyor.
	ham := "İşte çeviri:\n```json\n{\"title\":\"T\",\"spot\":\"S\",\"paragraphs\":[\"a\"]}\n```"
	m, err := ceviriJSON(ham)
	require.NoError(t, err)
	assert.Equal(t, "T", m["title"])
	assert.Equal(t, []any{"a"}, m["paragraphs"])

	_, err = ceviriJSON("JSON yok")
	assert.Error(t, err)
}

func TestTemizTekSatir(t *testing.T) {
	// Başlık alanına satır sonu/kontrol karakteri girmemeli.
	assert.Equal(t, "Başlık burada", temiz("  Başlık\n\t burada\x00  "))
	// Boşluk karakterleri silinirse kelimeler yapışır; boşluğa çevrilir.
	assert.Equal(t, "iki kelime", temiz("iki\tkelime"))
}

func TestSinirDakika(t *testing.T) {
	assert.Equal(t, VarsayilanDakika, sinirDakika(0))
	assert.Equal(t, VarsayilanDakika, sinirDakika(-3))
	assert.Equal(t, MaksDakika, sinirDakika(9999))
	assert.Equal(t, 12, sinirDakika(12))
}

func TestBeklemeTempo(t *testing.T) {
	// dakika=8 → öğe başına 7.5 sn; 4 öğe = 30 sn hedef.
	assert.Equal(t, 20*time.Second, Bekleme(8, 4, 10*time.Second))
	// Hedef aşıldıysa bekleme yok.
	assert.Equal(t, time.Duration(0), Bekleme(8, 4, 40*time.Second))
	// İş yoksa hesap yapılmaz.
	assert.Equal(t, time.Duration(0), Bekleme(8, 0, 0))
}

// TestKaydetParagrafSayisiKontrolu: model eksik/fazla paragraf döndürürse
// kayıt reddedilmeli — sıra kayması gövdeyi anlamsız karıştırır.
func TestKaydetParagrafSayisiKontrolu(t *testing.T) {
	kaynak := repo.Content{
		ID:     42,
		Locale: "tr",
		Detail: []byte(`[{"type":"p","text":"a"},{"type":"p","text":"b"}]`),
	}
	// Paragraf sayısı uyuşmuyor → hata (DB'ye gitmeden).
	_, err := Kaydet(t.Context(), nil, kaynak, "en", Ceviri{
		Baslik:      "Title",
		Paragraflar: []string{"only one"},
	}, "yayinda")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "paragraf")

	// Kaynak id'siz → hata.
	_, err = Kaydet(t.Context(), nil, repo.Content{}, "en", Ceviri{Baslik: "T"}, "yayinda")
	assert.Error(t, err)

	// Başlık boş → hata.
	_, err = Kaydet(t.Context(), nil, kaynak, "en", Ceviri{}, "yayinda")
	assert.Error(t, err)
}

func TestCeviriInstruksiyonKurallari(t *testing.T) {
	// Faz 190: istem ikiye ayrıldı (başlık/spot ve gövde). Kurallar artık
	// GERÇEK istem üreten fonksiyonlarda doğrulanır; tek bir sabit metin
	// kullanılmadığı için onu test etmek yanıltıcıydı.
	govde := ceviriInstruksiyonGovde("English")
	assert.Contains(t, govde, "AYNI uzunlukta")
	assert.Contains(t, govde, "DEĞİŞTİRME")
	assert.Contains(t, govde, "EKLEME")
	assert.Contains(t, govde, `"paragraphs"`)
	assert.Contains(t, govde, "Paragraf BİRLEŞTİRME veya BÖLME")

	baslik := ceviriInstruksiyonBaslik("English")
	assert.Contains(t, baslik, "30-60")
	assert.Contains(t, baslik, "110-155")
	assert.NotContains(t, baslik, "paragraphs", "başlık istemi gövde taşımamalı")

	// JSON zorlaması sistem isteminde.
	assert.Contains(t, ceviriSystem("English"), "SADECE geçerli JSON")
}

func TestVarsayilanAyarlar(t *testing.T) {
	// İngilizce öncelikli; Türkçe hedef olamaz (kaynak dil).
	assert.Equal(t, []string{"en"}, VarsayilanDiller)
	for _, d := range VarsayilanDiller {
		assert.NotEqual(t, "tr", d)
	}
	// Yayında varsayılan: kullanıcı İngilizce içeriği temada görmek istedi.
	a := Ayarlar{Aktif: true, Diller: VarsayilanDiller, Durum: "yayinda", Dakika: VarsayilanDakika}
	assert.Equal(t, "yayinda", a.Durum)
}

func TestDilAdi(t *testing.T) {
	assert.Equal(t, "İngilizce", dilAdi("en"))
	assert.Equal(t, "Almanca", dilAdi("de"))
	// Bilinmeyen kod olduğu gibi döner (prompt yine anlaşılır).
	assert.Equal(t, "xx", dilAdi("xx"))
}

// TestParagrafSayisiToleransi: kaynakta paragraf yoksa (yalnız başlık)
// çeviri de paragraf istemez; kayıt engellenmemeli.
func TestParagrafSayisiToleransi(t *testing.T) {
	kaynak := repo.Content{ID: 7, Locale: "tr", Detail: nil, Title: "Başlık"}
	// Paragraf yok → kontrol atlanır, DB'ye gidilir (nil pool panikler;
	// burada yalnız kontrolün atlandığını doğruluyoruz: hata "paragraf"
	// içermemeli).
	defer func() { _ = recover() }()
	_, err := Kaydet(t.Context(), nil, kaynak, "en", Ceviri{Baslik: "Title"}, "yayinda")
	if err != nil {
		assert.NotContains(t, strings.ToLower(err.Error()), "paragraf")
	}
}

// TestHataSayaci: kalıcı başarısız içerik 3 denemeden sonra atlanmalı.
//
// Neden: kayıt FK/veri hatasıyla sürekli düşerse kuyruğun başında kalır ve
// her turda yeniden denenir (her deneme AI maliyeti). İzole ortamda
// `contents_category_id_fkey` hatası tam olarak bunu gösterdi.
func TestHataSayaci(t *testing.T) {
	h := &hataSayaci{sayi: map[int64]int{}}
	assert.False(t, h.asildi(1))
	h.basarisiz(1)
	h.basarisiz(1)
	assert.False(t, h.asildi(1), "iki denemeden sonra hâlâ denenmeli")
	h.basarisiz(1)
	assert.True(t, h.asildi(1), "üç denemeden sonra atlanmalı")

	// Başarı sayacı sıfırlar.
	h.basarili(1)
	assert.False(t, h.asildi(1))
}

// TestTurTekrarveSayac: tur, tekrar denemeli üreticiyi ve sayacı kullanmalı.
func TestTurTekrarveSayac(t *testing.T) {
	src := mustOku(t, "ceviri.go")
	assert.Contains(t, src, "func UretTekrar(")
	assert.Contains(t, src, "cev, err := UretTekrar(ctx, aiclient, c, dil, logger)",
		"tur tekrar denemeli üreticiyi çağırmalı")
	assert.Contains(t, src, "hata.basarisiz(e.ID)")
	assert.Contains(t, src, "hata.basarili(e.ID)")
	assert.Contains(t, src, "hata.asildi(e.ID)")

	// Run kalıcı sayaç taşımalı (tur başına yeni sayaç = her turda yeniden
	// deneme demek).
	assert.Contains(t, src, "hata := &hataSayaci{sayi: map[int64]int{}}")
}

// mustOku paket dizinindeki dosyayı okur.
func mustOku(t *testing.T, yol string) string {
	t.Helper()
	b, err := os.ReadFile(yol)
	require.NoError(t, err)
	return string(b)
}

// TestGecerliReferanslarSifirGirdi — canlıda yakalanan FK hatasının kökü.
//
// Sahadan gelen gerçek vaka: `contents_category_id_fkey` ihlali. Silinmiş
// kategoriye işaret eden kaynak, çeviri kaydını patlatıyordu ve kullanıcıya
// ham SQL hatası gösteriliyordu. Aynı hata arka plan kuyruğunda kaydı her
// turda düşürüyordu.
//
// Bu testler nil/0 girdide nil dönmeyi (panik yok) doğrular; asıl FK
// doğrulaması entegrasyon testinde DB ile çalışır.
func TestGecerliReferanslarSifirGirdi(t *testing.T) {
	// nil havuz + nil kimlik: panik olmadan nil.
	assert.Nil(t, gecerliKategori(t.Context(), nil, nil))
	assert.Nil(t, gecerliMedya(t.Context(), nil, nil))

	// Sıfır/negatif kimlik de nil.
	zero := int64(0)
	neg := int64(-5)
	assert.Nil(t, gecerliKategori(t.Context(), nil, &zero))
	assert.Nil(t, gecerliKategori(t.Context(), nil, &neg))

	// Yazar nil dönemez (AuthorID NOT NULL): yedek 1.
	assert.Equal(t, int64(1), gecerliYazar(t.Context(), nil, 42))
}

// TestKaydetGecerliReferansKullanir: Kaydet artık doğrulanmış kimlik
// kullanmalı (ham kaynak alanı değil).
func TestKaydetGecerliReferansKullanir(t *testing.T) {
	src := mustOku(t, "ceviri.go")

	assert.Contains(t, src, "catID := gecerliKategori(ctx, pool, kaynak.CategoryID)")
	assert.Contains(t, src, "mediaID := gecerliMedya(ctx, pool, kaynak.CoverMediaID)")
	assert.Contains(t, src, "authorID := gecerliYazar(ctx, pool, kaynak.AuthorID)")

	// Kayıt yapısında doğrulanmış değişkenler kullanılmalı.
	i := strings.Index(src, "nc := &repo.Content{")
	require.Greater(t, i, 0)
	son := strings.Index(src[i:], "\n\t}")
	govde := src[i : i+son]
	assert.Contains(t, govde, "CategoryID:    catID")
	assert.Contains(t, govde, "AuthorID:      authorID")
	assert.NotContains(t, govde, "CategoryID:    kaynak.CategoryID",
		"ham kategori kimliği kullanılmamalı (FK patlıyordu)")
}

// TestParcaliUretim — canlıda yakalanan kesik yanıt hatası.
//
// Ölçüm 1: 28 paragraflık haberin çeviri yanıtı 1200 token sınırını aştı ve
// ORTASINDAN kesildi ("unexpected end of JSON input"); iki deneme de
// başarısız oldu, içerik çevirisiz kaldı.
//
// Ölçüm 2: yalnız PARAGRAF sayısına bakmak yetmedi — 6 uzun paragraf da
// 1200 tokenı doldurdu (completion_tokens=1200, iki deneme). Sınır artık
// karakter: 2200.
func TestParcaliUretim(t *testing.T) {
	src := mustOku(t, "ceviri.go")

	assert.Contains(t, src, "parcaKarakter = 2200")
	// 1600 → 2600: muhakeme tabanlı model bütçenin bir kısmını düşünmeye
	// harcıyor; 6 Kasım 2026'da 1600'lük bütçe kesiliyordu.
	assert.Contains(t, src, "MaxTokens:   2600",
		"muhakeme tabanlı modelde 1600 kesiliyor; 2600 pay bırakır")
	assert.Contains(t, src, "for _, grup := range parcalaraBol(p)")

	// Başlık ve gövde AYRI istekler: başlık isteminde paragraf yok.
	i := strings.Index(src, "func ceviriBaslik(")
	require.Greater(t, i, 0)
	son := strings.Index(src[i:], "\n}")
	govde := src[i : i+son]
	assert.NotContains(t, govde, `"paragraphs"`,
		"başlık isteği paragraf taşımamalı (token israfı)")
	// 500 → 1200: 500'de başlık+spot+SEO JSON'u yarıda kesiliyordu
	// (canlı ölçüm 6 Kasım 2026: "unexpected end of JSON input").
	assert.Contains(t, govde, "MaxTokens:   1200")

	// Parça sayısı doğrulanır: eksik/fazla paragraf kabul edilmez.
	assert.Contains(t, src, "parça paragraf sayısı uyuşmuyor")

	// Prompt paragraf sayısının korunmasını açıkça ister.
	assert.Contains(t, src, "AYNI uzunlukta")
	assert.Contains(t, src, "Paragraf BİRLEŞTİRME veya BÖLME")
}

// TestParcalaraBol: bölme sırayı ve bütünlüğü korur.
//
// Kural: her parça en fazla parcaBoyu paragraf ve ~parcaKarakter karakter;
// tek paragraf sınırı aşsa bile ilerler (aksi hâlde döngü kilitlenir).
func TestParcalaraBol(t *testing.T) {
	// 28 paragraf × 300 karakter = 8400 → 4 parça (2200'lük sınır).
	var p []string
	for i := 0; i < 28; i++ {
		p = append(p, strings.Repeat("a", 300))
	}
	gruplar := parcalaraBol(p)
	require.NotEmpty(t, gruplar)

	toplam := 0
	for _, g := range gruplar {
		assert.LessOrEqual(t, len(g), parcaBoyu)
		assert.NotEmpty(t, g, "boş parça gönderilmemeli")
		kar := 0
		for _, x := range g {
			kar += len(x)
		}
		// Tek paragraf sınırı aşabilir; çoklu grupta sınır korunur.
		if len(g) > 1 {
			assert.LessOrEqual(t, kar, parcaKarakter)
		}
		toplam += len(g)
	}
	assert.Equal(t, 28, toplam, "hiçbir paragraf kaybolmamalı")

	// Çok kısa paragraflar: 100 paragraf, parcaBoyu ile sınırlanır.
	var kisa []string
	for i := 0; i < 100; i++ {
		kisa = append(kisa, "kısa")
	}
	g2 := parcalaraBol(kisa)
	assert.Len(t, g2, 17, "100 paragraf / 6 = 17 parça")
	assert.Len(t, g2[16], 4)

	// Tek uzun paragraf: bölünmez, tek başına gider (sonsuz döngü yok).
	uzun := parcalaraBol([]string{strings.Repeat("b", 5000), "kısa"})
	require.Len(t, uzun, 2)
	assert.Len(t, uzun[0], 1)
	assert.Len(t, uzun[1], 1)

	// Boş gövde.
	assert.Empty(t, parcalaraBol(nil))
}

// TestKesilenParcaBolunur — canlıda yakalanan ikinci kesilme.
//
// Ölçüm: 2200 karakterlik parça bile 1600 tokenı doldurdu
// (completion_tokens=1600 → "unexpected end of JSON input"). O durumda tüm
// makaleyi baştan çevirmek 3,5 dakika kaybettiriyordu. Parça ikiye bölünür,
// hata o parçayla sınırlı kalır.
func TestKesilenParcaBolunur(t *testing.T) {
	src := mustOku(t, "ceviri.go")

	assert.Contains(t, src, "func ceviriParcaGuvenli(")
	// Bölme: yarıdan ikiye, sıra korunarak birleştirme.
	assert.Contains(t, src, "orta := len(grup) / 2")
	assert.Contains(t, src, "return append(sol, sağ...), nil")
	// Tek paragrafta bölme yok (sonsuz özyineleme olurdu).
	assert.Contains(t, src, "if len(grup) <= 1 {")
	// Gövde döngüsü güvenli sürümü kullanır.
	assert.Contains(t, src, "ceviriParcaGuvenli(ctx, aiclient, grup, dilAdiMetin)")
}

// TestKopyaCeviriEngeli — aynı dile ikinci kayıt oluşmamalı.
//
// Ölçüm: editör isteği 3,5 dk sürerken otomatik tur aynı içeriği çevirdi.
// Kayıt öncesi son kontrol olmasa iki EN çevirisi oluşuyordu (contents
// tablosunda (translation_of, locale) benzersiz kısıtı YOK).
func TestKopyaCeviriEngeli(t *testing.T) {
	src := mustOku(t, "istek.go")

	i := strings.Index(src, "nid, err := Kaydet(")
	require.Greater(t, i, 0)
	once := src[:i]
	j := strings.LastIndex(once, "// Üretim sürerken otomatik tur")
	require.Greater(t, j, 0, "Kaydet öncesi kopya kontrolü yok")
	kontrol := once[j:]
	assert.Contains(t, kontrol, "repo.TranslationsOf(")
	assert.Contains(t, kontrol, "istekBitir(ctx, pool, is.ID, IstekBitti, t.ID, \"\")")
}
