// GERÇEK POSTGRES'E KARŞI toplu yazma + niyet sorguları.
//
// NEDEN ENTEGRASYON: toplu INSERT `UNNEST` ile 13 dizi geçiyor; sütun
// sayısı ya da tipi bir yerde kayarsa birim testler (havuzsuz) bunu
// göremez. Bu test yalnız OLAY_DB_URL verildiğinde koşar; normal kapı
// veritabanı istemez.
package repo_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"theblock/site/internal/repo"
)

func olayHavuzu(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("OLAY_DB_URL")
	if url == "" {
		t.Skip("OLAY_DB_URL yok")
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("havuz: %v", err)
	}
	t.Cleanup(pool.Close)
	if _, err := pool.Exec(context.Background(),
		`DELETE FROM visitor_events WHERE anahtar LIKE 'test-%'`); err != nil {
		t.Fatalf("temizlik: %v", err)
	}
	return pool
}

func TestOlayTopluKaydetGercekDB(t *testing.T) {
	pool := olayHavuzu(t)
	ctx := context.Background()
	anahtar := "test-ziyaretci-1"
	simdi := time.Now()

	olaylar := []repo.ZiyaretOlay{
		{Anahtar: anahtar, IP: "5.9.1.17", Zaman: simdi.Add(-3 * time.Minute),
			Yontem: "GET", Yol: "/wp-login.php", Durum: 403, Tur: "zafiyet",
			Kaynak: "yok", UA: "python-requests/2.31.0", Sinif: "bot", Engellendi: true},
		{Anahtar: anahtar, IP: "5.9.1.17", Zaman: simdi.Add(-2 * time.Minute),
			Yontem: "GET", Yol: "/.env", Durum: 403, Tur: "zafiyet",
			Kaynak: "yok", UA: "python-requests/2.31.0", Sinif: "bot", Engellendi: true},
		{Anahtar: anahtar, IP: "5.9.1.17", Zaman: simdi.Add(-1 * time.Minute),
			Yontem: "GET", Yol: "/haber/bitcoin", Sorgu: "q=bitcoin&token=***",
			Durum: 200, Tur: "icerik", Kaynak: "arama", Referer: "https://www.google.com/",
			UA: "Mozilla/5.0 Chrome/120", Sinif: "supheli"},
		// Boş IP ve boş sorgu NULL'a düşmeli (NULLIF).
		{Anahtar: "test-ziyaretci-2", Zaman: simdi, Yontem: "HEAD", Yol: "/",
			Durum: 200, Tur: "icerik", Kaynak: "yok"},
	}
	if err := repo.OlayTopluKaydet(ctx, pool, olaylar); err != nil {
		t.Fatalf("toplu kaydet: %v", err)
	}

	okunan, err := repo.Olaylar(ctx, pool, anahtar, 10)
	if err != nil {
		t.Fatalf("olaylar: %v", err)
	}
	if len(okunan) != 3 {
		t.Fatalf("3 olay beklenirdi, %d geldi", len(okunan))
	}
	// Yeniden eskiye sıralı olmalı.
	if okunan[0].Yol != "/haber/bitcoin" {
		t.Errorf("sıralama yanlış: ilk olay %q", okunan[0].Yol)
	}
	if okunan[0].Sorgu != "q=bitcoin&token=***" {
		t.Errorf("sorgu bozuldu: %q", okunan[0].Sorgu)
	}
	if !okunan[2].Engellendi || okunan[2].Durum != 403 {
		t.Errorf("403/engel bilgisi kayboldu: %+v", okunan[2])
	}

	// Niyet sayımı: 2 zafiyet + 1 içerik.
	sayim, toplam, engellenen, err := repo.OlaySayim(ctx, pool, anahtar)
	if err != nil {
		t.Fatalf("sayım: %v", err)
	}
	if toplam != 3 || sayim["zafiyet"] != 2 || sayim["icerik"] != 1 {
		t.Errorf("sayım yanlış: toplam=%d %v", toplam, sayim)
	}
	if engellenen != 2 {
		t.Errorf("engellenen = %d, beklenen 2", engellenen)
	}

	// Yol sayımı: en sık yollar.
	yollar, err := repo.OlayYolSayim(ctx, pool, anahtar, 5)
	if err != nil {
		t.Fatalf("yol sayımı: %v", err)
	}
	if len(yollar) != 3 {
		t.Fatalf("3 farklı yol beklenirdi, %d", len(yollar))
	}

	// IP pivotu: çerezsiz kimlik de aynı IP'den görünmeli.
	akraba, err := repo.OlayIPAkrabalari(ctx, pool, "5.9.1.17", anahtar, 5)
	if err != nil {
		t.Fatalf("akraba: %v", err)
	}
	if len(akraba) != 0 {
		t.Errorf("tek kimlik varken akraba olmamalı: %+v", akraba)
	}
	if err := repo.OlayTopluKaydet(ctx, pool, []repo.ZiyaretOlay{
		{Anahtar: "test-ziyaretci-3", IP: "5.9.1.17", Zaman: simdi,
			Yontem: "GET", Yol: "/", Durum: 200, Tur: "icerik", Kaynak: "yok"},
	}); err != nil {
		t.Fatalf("ikinci kimlik: %v", err)
	}
	akraba, err = repo.OlayIPAkrabalari(ctx, pool, "5.9.1.17", anahtar, 5)
	if err != nil {
		t.Fatalf("akraba 2: %v", err)
	}
	if len(akraba) != 1 || akraba[0].Anahtar != "test-ziyaretci-3" {
		t.Errorf("pivot çalışmadı: %+v", akraba)
	}

	// Toplu niyet: liste sayfasının tek sorgusu.
	niyetler, err := repo.OlayNiyetleriToptan(ctx, pool,
		[]string{anahtar, "test-ziyaretci-2", "test-ziyaretci-yok"})
	if err != nil {
		t.Fatalf("toplu niyet: %v", err)
	}
	if niyetler[anahtar]["zafiyet"] != 2 {
		t.Errorf("toplu niyet yanlış: %v", niyetler)
	}
	if _, var_ := niyetler["test-ziyaretci-yok"]; var_ {
		t.Error("olmayan anahtar için kayıt dönmemeli")
	}

	// Temizlik: süresi dolmuş olaylar silinir.
	if _, err := pool.Exec(ctx, `UPDATE visitor_events SET zaman = now() - interval '20 days'
		WHERE anahtar = 'test-ziyaretci-2'`); err != nil {
		t.Fatalf("eskiltme: %v", err)
	}
	n, err := repo.OlayTemizle(ctx, pool, 14)
	if err != nil {
		t.Fatalf("temizlik: %v", err)
	}
	if n != 1 {
		t.Errorf("1 satır silinmeliydi, %d silindi", n)
	}
}

// RiskTek: profil sayfasının tek satır okuması (ASN birleştirmesi dahil).
func TestRiskTekGercekDB(t *testing.T) {
	pool := olayHavuzu(t)
	ctx := context.Background()

	if _, err := pool.Exec(ctx, `DELETE FROM visitor_risk WHERE anahtar = 'test-risk-1'`); err != nil {
		t.Fatalf("temizlik: %v", err)
	}
	if err := repo.ASNKaydet(ctx, pool, repo.ASNSatir{
		Ag: "5.9.0.0/16", ASN: 24940, Ad: "HETZNER-AS - Hetzner Online GmbH, DE",
		Ulke: "DE", Tur: "hosting", Kaynak: "test",
	}); err != nil {
		t.Fatalf("asn: %v", err)
	}
	if err := repo.RiskKaydet(ctx, pool, repo.RiskSatir{
		Anahtar: "test-risk-1", VisitorID: "", IP: "5.9.1.17",
		Puan: 92, Sinif: "bot", Bot: true, Hosting: true,
		Sinyaller:   `[{"k":"asn_hosting","p":45,"n":"veri merkezi"}]`,
		IstekSayisi: 7, UserAgent: "python-requests/2.31.0",
	}); err != nil {
		t.Fatalf("risk: %v", err)
	}

	s, var_, err := repo.RiskTek(ctx, pool, "test-risk-1")
	if err != nil {
		t.Fatalf("RiskTek: %v", err)
	}
	if !var_ {
		t.Fatal("satır bulunamadı")
	}
	if s.IP != "5.9.1.17" || s.Puan != 92 {
		t.Errorf("temel alanlar yanlış: %+v", s)
	}
	// ASN önbellekten TAMAMLANMALI: risk satırında asn boş, ip_asn'de var.
	if s.ASN != 24940 {
		t.Errorf("ASN önbellekten tamamlanmadı: %d", s.ASN)
	}
	if s.ASNTur != "hosting" {
		t.Errorf("tür önbellekten gelmedi: %q", s.ASNTur)
	}
	if s.UserAgent != "python-requests/2.31.0" {
		t.Errorf("UA saklanmadı: %q", s.UserAgent)
	}

	// İkinci yazma UA'yı korur (COALESCE) ve istek sayısını artırır.
	if err := repo.RiskKaydet(ctx, pool, repo.RiskSatir{
		Anahtar: "test-risk-1", IP: "5.9.1.17", Puan: 0, Sinif: "temiz",
		IstekSayisi: 1,
	}); err != nil {
		t.Fatalf("risk 2: %v", err)
	}
	s, _, err = repo.RiskTek(ctx, pool, "test-risk-1")
	if err != nil {
		t.Fatalf("RiskTek 2: %v", err)
	}
	if s.UserAgent == "" {
		t.Error("UA boş yazımla silindi (COALESCE çalışmadı)")
	}
	if s.IstekSayisi != 8 {
		t.Errorf("istek sayısı birikmedi: %d", s.IstekSayisi)
	}
	// Bot bayrağı OR ile korunur: temiz yazım onu silmemeli.
	if !s.Bot || !s.Hosting {
		t.Errorf("bayrak korunmadı: bot=%v hosting=%v", s.Bot, s.Hosting)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM visitor_risk WHERE anahtar = 'test-risk-1'`); err != nil {
		t.Logf("temizlik: %v", err)
	}
}
