// KUYRUK DAVRANIŞININ UÇTAN UCA KANITI: gerçek bir veritabanı olmadan
// olayların yazma yolunu (kanal -> toplu dilim) doğrular.
//
// NEDEN AYRI TEST: TestOlayKuyrukDusurme kanalı doldurup taşmayı ölçüyor;
// burada amaç işçinin KAPANIŞTA elde kalanı boşaltmaya çalıştığını
// görmek — süreç kapanırken son 2 saniyenin kaydı kaybolmasın.
// Havuz yoksa OlayTopluKaydet hiçbir şey yapmaz; test yalnız kanal
// davranışını sınar, DB gerektirmez.
package httpapi

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"theblock/site/internal/repo"
)

func TestOlayIsciKapanistaBosaltir(t *testing.T) {
	k := olaylariBaslat()
	for len(k) > 0 {
		<-k
	}

	// Havuzsuz sunucu: yazma yolu no-op, ama döngü çalışır.
	s := &Server{logger: slog.New(slog.NewTextHandler(io.Discard, nil))}

	ctx, iptal := context.WithCancel(context.Background())
	s.OlayIsci(ctx)

	for i := 0; i < 5; i++ {
		olayYaz(repo.ZiyaretOlay{Anahtar: "a", Yol: "/", Tur: turIcerik})
	}
	if len(k) != 5 {
		t.Fatalf("kuyruğa 5 olay girmeliydi, %d", len(k))
	}

	iptal() // kapanış: işçi elde kalanı yazmayı DENER (havuz yok, no-op)

	// Kısa bekleme: işçi ctx.Done'ı görüp yaz() çağırsın.
	bitis := time.Now().Add(2 * time.Second)
	for time.Now().Before(bitis) {
		if len(k) == 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("kapanışta kuyruk boşaltılmadı: %d olay kaldı", len(k))
}
