package httpapi

import (
	"context"
	"html"
	"strings"
	"time"

	"theblock/site/internal/mailer"
)

// İLETİŞİM MESAJI E-POSTA BİLDİRİMİ.
//
// NEDEN GEREKLİ: form mesajı yalnız `contact_messages` tablosuna yazıyordu.
// Kimse panele bakmazsa okuyucu mesajı HİÇ okunmuyordu — kullanıcının
// "form çalışmıyor" demesinin bir nedeni de bu: gönderiyor, hiçbir yanıt
// gelmiyor. Artık konuya göre doğru adrese e-posta gider.
//
// SMTP KURULU DEĞİLSE sessizce atlanır: mesaj zaten veritabanında, panelde
// görünür. Bildirim hatası kullanıcıya ASLA yansımaz (aksi hâlde mesajı
// kaydedilmiş olsa bile "gönderilemedi" derdik).
func (s *Server) contactBildir(ctx context.Context, id int64, c ContactConfig,
	ad, eposta, telefon, konu, mesaj string) {
	hedef := strings.TrimSpace(c.Info)
	switch konu {
	case "haber":
		if c.Haber != "" {
			hedef = c.Haber
		}
	case "duzeltme":
		if c.Duzeltme != "" {
			hedef = c.Duzeltme
		}
	case "reklam":
		if c.Reklam != "" {
			hedef = c.Reklam
		}
	case "kvkk":
		if c.KVKK != "" {
			hedef = c.KVKK
		}
	case "teknik":
		if c.Destek != "" {
			hedef = c.Destek
		}
	case "kariyer":
		if c.Kariyer != "" {
			hedef = c.Kariyer
		}
	}
	if hedef == "" || !validEmail(hedef) {
		return
	}
	// Gövde HTML'dir: kullanıcı girdisi KAÇIRILIR (e-posta istemcisinde
	// HTML enjeksiyonu olmasın).
	esc := html.EscapeString
	konuAd := konu
	for _, t := range contactTopics() {
		if t.Key == konu {
			konuAd = t.Name
			break
		}
	}
	var b strings.Builder
	b.WriteString("<h2 style=\"margin:0 0 12px\">Yeni iletişim mesajı</h2>")
	b.WriteString("<p style=\"margin:0 0 4px\"><strong>Konu:</strong> " + esc(konuAd) + "</p>")
	b.WriteString("<p style=\"margin:0 0 4px\"><strong>Ad:</strong> " + esc(ad) + "</p>")
	b.WriteString("<p style=\"margin:0 0 4px\"><strong>E-posta:</strong> " + esc(eposta) + "</p>")
	if telefon != "" {
		b.WriteString("<p style=\"margin:0 0 4px\"><strong>Telefon:</strong> " + esc(telefon) + "</p>")
	}
	b.WriteString("<hr style=\"border:0;border-top:1px solid #ddd;margin:14px 0\">")
	b.WriteString("<div style=\"white-space:pre-wrap\">" + esc(mesaj) + "</div>")
	b.WriteString("<hr style=\"border:0;border-top:1px solid #ddd;margin:14px 0\">")
	b.WriteString("<p style=\"color:#666;font-size:12px\">Kayıt #" + itoa(int(id)) +
		" · Panelden yanıtlayın: /admin/iletisim</p>")

	// İstek bağlamından BAĞIMSIZ: kullanıcı sayfayı kapatsa da bildirim
	// gitmeli. SMTP yoksa mailer.Send hemen döner (maliyet yok).
	//nolint:gosec // G118: istek bağlamı kullanıcı sayfayı kapatınca iptal
	// olur ve bildirim yarıda kesilir; bu iş bilinçli olarak bağımsızdır.
	go func() {
		bctx, iptal := context.WithTimeout(context.Background(), 15*time.Second)
		defer iptal()
		if err := mailer.Send(bctx, s.pool, hedef,
			"[İletişim] "+konuAd+" — "+ad, b.String()); err != nil {
			s.logger.Warn("iletişim bildirimi gönderilemedi", "err", err, "id", id)
		}
	}()
}
