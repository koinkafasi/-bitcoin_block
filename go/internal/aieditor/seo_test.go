package aieditor

import (
	"os"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// SEO alanlarının otomatik doldurulması (Faz 188).
//
// Bu değerler <title> ve <meta content> içine gidiyor; uzunluk ve satır sonu
// hataları doğrudan SERP'te görünüyor. Testler bu yüzden üretim kurallarını
// (temizle, kırp, doğrula) ayrı ayrı kilitliyor.

func TestSeoTemizleTekSatir(t *testing.T) {
	// Model satır sonu ve tırnak döndürüyor; <title> içinde bunlar
	// boşluk/kaçış olarak sızıyordu.
	assert.Equal(t, "Deprem bölgesinde yeni konutlar teslim edildi",
		seoTemizle("  \"Deprem bölgesinde\n yeni   konutlar\tteslim edildi\"  "))

	// Sarma tırnakları (kıvrık olanlar dahil).
	assert.Equal(t, "Kısa başlık", seoTemizle("\u201cKısa başlık\u201d"))
	assert.Equal(t, "Kısa başlık", seoTemizle("'Kısa başlık'"))

	// Kontrol karakterleri (NUL, BEL) atılır.
	assert.Equal(t, "Temiz", seoTemizle("Te\x00miz\x07"))
}

func TestSeoKirpRuneSiniri(t *testing.T) {
	// Türkçe karakterler iki bayt; bayt ile kesmek metni bozuyordu.
	s := strings.Repeat("ş", 100)
	out := seoKirp(s, 10)
	assert.Equal(t, 10, utf8.RuneCountInString(out))
	assert.True(t, utf8.ValidString(out))

	// Kelime ortasında kesilmez.
	uzun := "Deprem bölgesinde yeni konutlar hak sahiplerine teslim edildi ve süreç devam ediyor"
	k := seoKirp(uzun, 40)
	assert.LessOrEqual(t, utf8.RuneCountInString(k), 40)
	assert.NotContains(t, k, "  ")
	// Son kelime bütün olmalı: kesilen metin, orijinalin bir ön eki olmalı.
	assert.True(t, strings.HasPrefix(uzun, k), "kırpma kelimeyi bölmemeli: %q", k)

	// Zaten kısa olan metne dokunulmaz.
	assert.Equal(t, "kısa", seoKirp("kısa", 40))
}

func TestSeoDogrulaSertSinir(t *testing.T) {
	// Aşırı uzun alanlar sert sınıra kırpılır ve uyarı üretir.
	o := SeoDogrula(SeoOneri{
		Baslik:   strings.Repeat("başlık ", 40),
		Aciklama: strings.Repeat("açıklama ", 60),
	})
	assert.LessOrEqual(t, utf8.RuneCountInString(o.Baslik), SeoBaslikSertMax)
	assert.LessOrEqual(t, utf8.RuneCountInString(o.Aciklama), SeoAciklamaSertMax)
	require.NotEmpty(t, o.Uyarilar)
	assert.Contains(t, strings.Join(o.Uyarilar, " "), "uzun")
}

func TestSeoDogrulaKisaUyarisi(t *testing.T) {
	o := SeoDogrula(SeoOneri{Baslik: "Kısa", Aciklama: "Çok kısa."})
	uyar := strings.Join(o.Uyarilar, " ")
	assert.Contains(t, uyar, "kısa")
	// Kısa olsa da alan korunur: tamamen silmek editöre boş alan bırakırdı.
	assert.Equal(t, "Kısa", o.Baslik)
}

func TestSeoDogrulaBosAlanUyarisi(t *testing.T) {
	o := SeoDogrula(SeoOneri{})
	assert.Contains(t, strings.Join(o.Uyarilar, " "), "üretilemedi")
}

func TestSeoJSONBlokSarmasi(t *testing.T) {
	// Modeller JSON'u ```json bloğuna sarıyor veya başına cümle ekliyor.
	ham := "İşte öneri:\n```json\n{\"seo_title\":\"A\",\"meta_description\":\"B\",\"keywords\":[\"x\"]}\n```"
	m, err := seoJSON(ham)
	require.NoError(t, err)
	assert.Equal(t, "A", m["seo_title"])
	assert.Equal(t, "B", m["meta_description"])

	// Bozuk yanıt hata vermeli (sessizce boş yazmamalı).
	_, err = seoJSON("JSON yok")
	assert.Error(t, err)
}

func TestSeoAnahtarlarSinirli(t *testing.T) {
	// En çok 6 anahtar; boşlar ve satır sonları atılır.
	a := seoAnahtarlar([]any{"bir", "iki\n", "", "üç", "dört", "beş", "altı", "yedi"})
	assert.Len(t, a, 6)
	assert.Equal(t, "iki", a[1])

	// Virgülle ayrılmış metin de kabul edilir.
	assert.Equal(t, []string{"a", "b"}, seoAnahtarlar("a, b"))
}

// TestSeoYazDoluAlanKorunur: force=false iken dolu alan ezilmemeli.
//
// SQL'i doğrudan test etmek yerine kuralı belgeleyen sabitleri kontrol
// ediyoruz; gerçek UPDATE entegrasyon testinde (DB gerekir) doğrulanır.
func TestSeoYazForceDavranisi(t *testing.T) {
	// force=false → yalnız boş alanlar dolar. Bu davranış SeoYaz'ın SQL
	// CASE ifadesinde; burada imzanın varsayılanını kilitliyoruz.
	_, err := SeoYaz(nil, nil, 0, SeoOneri{Baslik: "a"}, false)
	assert.Error(t, err, "geçersiz içerik hata vermeli")

	_, err = SeoYaz(nil, nil, 5, SeoOneri{}, false)
	assert.Error(t, err, "boş öneri hata vermeli")
}

// TestSeoPromptUzunlukKurallari: modele verilen talimat ölçülen bandı
// taşımalı; aksi hâlde üretim sürekli bandın dışına çıkıyordu.
func TestSeoPromptUzunlukKurallari(t *testing.T) {
	assert.Contains(t, SeoInstruksiyon, "30-60")
	assert.Contains(t, SeoInstruksiyon, "110-155")
	assert.Contains(t, SeoInstruksiyon, "UYDURMA")
	assert.Contains(t, SeoInstruksiyon, "seo_title")
	assert.Contains(t, SeoInstruksiyon, "meta_description")
	assert.Contains(t, SeoInstruksiyon, "keywords")

	// Eylem kayıt defterinde gizli olarak durmalı: gövdenin tamamına bakar,
	// seçili metin menüsünde görünmemeli.
	act, ok := ActionByID("seo_fill")
	require.True(t, ok, "seo_fill eylemi kayıtlı olmalı")
	assert.True(t, act.Hidden)
	assert.Equal(t, SeoInstruksiyon, act.Instruction)
}

// TestSeoAlanlariVar: içerik modelinde SEO alanları duruyor (iş bunlara
// yazıyor).
func TestSeoAlanlariVar(t *testing.T) {
	// SeoYaz kolon adları ile repo modeli aynı olmalı.
	assert.Contains(t, seoYazSQL, "seo_title")
	assert.Contains(t, seoYazSQL, "seo_description")
	// Dolu alan korunmalı: CASE + force parametresi sorguda olmalı.
	assert.Contains(t, seoYazSQL, "CASE")
	assert.Contains(t, seoYazSQL, "$3")
}

// TestSeoAlanAdiTemizle — canlıda yakalanan gerçek hata.
//
// Gövdesinde "Cumhuriyet" GEÇMEYEN bir spor haberinin meta açıklaması
// "… Cumhuriyet.com.tr'da" diye bitiyordu: model, kaynak yayının adını
// uydurdu. Bu yanlış atıf yayına girmemeli.
func TestSeoAlanAdiTemizle(t *testing.T) {
	kaynak := "Fenerbahçe Göztepe maçı canlı anlatım. Süper Lig 10. hafta."

	// Uydurma atıf: o cümle atılır.
	temiz, degisti := seoAlanAdiTemizle(
		"Maçın canlı anlatımı başladı. Detaylar Cumhuriyet.com.tr'da.", kaynak)
	assert.True(t, degisti)
	assert.NotContains(t, temiz, "Cumhuriyet")
	assert.Contains(t, temiz, "canlı anlatımı")

	// Kaynakta geçen alan adı korunur (haberin konusu olabilir).
	ozel := "Resmi site theblock.com.tr üzerinden duyuruldu."
	temiz, degisti = seoAlanAdiTemizle(ozel, "Açıklama theblock.com.tr adresinde.")
	assert.False(t, degisti, "kaynakta geçen alan adı korunmalı")
	assert.Equal(t, ozel, temiz)

	// Hiç alan adı yoksa metin aynen kalır.
	temiz, degisti = seoAlanAdiTemizle("Sade bir cümle.", kaynak)
	assert.False(t, degisti)
	assert.Equal(t, "Sade bir cümle.", temiz)

	// Boş metin güvenli.
	temiz, degisti = seoAlanAdiTemizle("", kaynak)
	assert.False(t, degisti)
	assert.Equal(t, "", temiz)

	// Tüm cümleler uydurma atıf içeriyorsa alan boşalır (uydurma tutulmaz).
	temiz, degisti = seoAlanAdiTemizle("Detaylar hurriyet.com.tr'de.", kaynak)
	assert.True(t, degisti)
	assert.Equal(t, "", temiz)
}

// TestSeoPromptSiteAdiYasagi: modele açıkça "site adı yazma" denmeli.
func TestSeoPromptSiteAdiYasagi(t *testing.T) {
	assert.Contains(t, SeoInstruksiyon, "alan adını")
	assert.Contains(t, SeoInstruksiyon, "YAZMA")
	assert.Contains(t, SeoInstruksiyon, "Site adı")
}

// TestSeoCumlelere: cümle bölme sonlandırıcıyı cümlede bırakır ve kırıntı
// üretmez.
func TestSeoCumlelere(t *testing.T) {
	c := seoCumlelere("Bir. İki! Üç?")
	assert.Equal(t, []string{"Bir.", "İki!", "Üç?"}, c)
	// Sonlandırıcısız kalan parça da cümle sayılır.
	assert.Equal(t, []string{"Bir.", "İki"}, seoCumlelere("Bir. İki"))

	// ALAN ADINDAKİ NOKTA CÜMLE BİTİRMEZ.
	//
	// Gerçek vaka: "Ayrıntılar Cumhuriyet.com.tr'da." cümlesi üçe
	// bölünüyordu ("Ayrıntılar Cumhuriyet." / "com." / "tr'da."). Bu hem
	// metni bozuyor hem alan adı korumasını etkisiz kılıyordu: parçalara
	// ayrılmış alan adı desene uymuyor, uydurma atıf temizlenmiyordu.
	assert.Equal(t,
		[]string{"Ayrıntılar Cumhuriyet.com.tr'da.", "Maç başladı."},
		seoCumlelere("Ayrıntılar Cumhuriyet.com.tr'da. Maç başladı."))

	// Kısaltma benzeri bitişler de bölünmemeli (nokta + harf).
	assert.Equal(t, []string{"Saat 10.30'da başladı."},
		seoCumlelere("Saat 10.30'da başladı."))
}

// TestSeoAlanAdiKorumasiUygulanir — korumanın GERÇEKTEN uygulandığını
// doğrular.
//
// İlk turda temizleyici ayrı test ediliyordu ama SeoUret içindeki çağrı
// silinse bile testler yeşil kalıyordu (mutasyonla yakalandı). Bu test
// SeoUret'in kullandığı tam yolu çağırır.
func TestSeoAlanAdiKorumasiUygulanir(t *testing.T) {
	kaynak := "Fenerbahçe Göztepe maçı canlı anlatım. Süper Lig 10. hafta."

	o := SeoOneri{
		Baslik:   "Maç canlı anlatım: detaylar Cumhuriyet.com.tr'da",
		Aciklama: "Maçın canlı anlatımı başladı. Ayrıntılar Cumhuriyet.com.tr'da.",
	}
	o = SeoAlanAdiKorumasi(o, kaynak)

	assert.NotContains(t, o.Baslik, "Cumhuriyet", "başlıktaki uydurma atıf çıkmalı")
	assert.NotContains(t, o.Aciklama, "Cumhuriyet", "açıklamadaki uydurma atıf çıkmalı")
	assert.Contains(t, o.Aciklama, "canlı anlatımı",
		"uydurma olmayan cümle korunmalı")
	require.NotEmpty(t, o.Uyarilar, "kullanıcı bilgilendirilmeli")
	assert.Contains(t, strings.Join(o.Uyarilar, " "), "site adı")

	// Uydurma yoksa metin AYNEN kalmalı ve uyarı eklenmemeli.
	temiz := SeoOneri{Baslik: "Maç canlı anlatım", Aciklama: "Maçın canlı anlatımı başladı."}
	ayni := SeoAlanAdiKorumasi(temiz, kaynak)
	assert.Equal(t, temiz.Baslik, ayni.Baslik)
	assert.Equal(t, temiz.Aciklama, ayni.Aciklama)
	assert.Empty(t, ayni.Uyarilar)

	// SeoUret bu fonksiyonu çağırmalı (satır içi kopya değil).
	src := mustReadAieditor(t, "seo.go")
	assert.Contains(t, src, "o = SeoAlanAdiKorumasi(o, kaynak)",
		"SeoUret korumayı uygulamalı")
}

// mustReadAieditor paket dizinindeki dosyayı okur.
func mustReadAieditor(t *testing.T, yol string) string {
	t.Helper()
	b, err := os.ReadFile(yol)
	require.NoError(t, err)
	return string(b)
}

// TestSeoJSONKurtar — canlıda yakalanan kesik yanıt.
//
// Ölçüm: max_tokens 400/700 iken bazı yanıtlar token sınırında ortasından
// kesiliyor ("unexpected end of JSON input"). Başlık ve açıklama kesik
// metnin başında olduğu için kurtarılabilir.
func TestSeoJSONKurtar(t *testing.T) {
	// Klasik kesik yanıt: ikinci alanın ortasında bitiyor.
	kesik := `{"seo_title":"Enflasyon Verileri Açıklandı", "meta_description":"Merkez Bankası`
	m, err := seoJSON("```json\n" + kesik)
	require.NoError(t, err, "kesik yanıt kurtarılmalı")
	assert.Equal(t, "Enflasyon Verileri Açıklandı", m["seo_title"])
	assert.Equal(t, "Merkez Bankası", m["meta_description"])

	// Yalnız başlık varsa o kurtarılır.
	m, err = seoJSON(`{"seo_title":"Başlık Burada"`)
	require.NoError(t, err)
	assert.Equal(t, "Başlık Burada", m["seo_title"])

	// Kaçış dizileri çözülür (tırnak içeren başlık).
	m, err = seoJSON(`{"seo_title":"\"Alıntı\" başlık"`)
	require.NoError(t, err)
	assert.Contains(t, m["seo_title"], "Alıntı")

	// Kurtarılacak alan yoksa hata: uydurma kabul edilmez.
	_, err = seoJSON(`{"keywords":["a","b"`)
	assert.Error(t, err)

	// Tamamen bozuk metin de hata.
	_, err = seoJSON("JSON yok")
	assert.Error(t, err)
}
