'use strict';
/**
 * Bülten çift tıkla onay akışı (denetim düzeltmesi):
 *
 * - Kayıt sonrası SMTP varsa onay e-postası gönderilir; yoksa onay
 *   bağlantısı yanıtta döner (geliştirme kolaylığı).
 * - Onay bağlantısı aboneliği "aktif" yapar.
 * - Eski hata: onay e-postası HİÇ gönderilmiyordu, bağlantı yalnızca JSON
 *   yanıtında dönüyordu.
 */

const test = require('node:test');
const assert = require('node:assert/strict');
const { goto, withPage, withAdminPage } = require('../lib/harness');

// Admin oturumu varsa temizlik için admin bağlamı kullanılır (abone silme).
const calistir = process.env.TB_SESSION ? withAdminPage : withPage;

test('bülten: onay bağlantısı akışı çalışır', async () => {
  await calistir(async (page) => {
    await goto(page, '/bulten');

    const kayit = await page.evaluate(async () => {
      const fd = new FormData();
      fd.append('email', 'e2e-onay@example.com');
      const r = await fetch('/bulten', { method: 'POST', body: fd, credentials: 'same-origin' });
      return { status: r.status, body: await r.json() };
    });
    assert.equal(kayit.status, 200);
    assert.ok(kayit.body.ok, 'kayıt ok olmalı');
    // NOT (Faz 210): metin "onay e-postası" (küçük o) diyor; eski /Onay/
    // büyük harf beklediği için hiç eşleşmiyordu. Büyük/küçük harf duyarsız.
    assert.match(kayit.body.mesaj, /onay/i, 'onay e-postası ya da onay bağlantısı bildirilmeli: ' + kayit.body.mesaj);

    // Onay bağlantısı yalnız İKİ durumda yanıtta döner:
    //   1) SMTP yok + APP_DEBUG=true (geliştirme kolaylığı),
    //   2) SMTP var + e-posta gönderildi (bağlantı e-postada, yanıtta yok).
    // Üretimde token'ın yanıtta dönmesi sızıntıdır; bu yüzden `Debug`
    // kapalıyken ürün bilinçli olarak bağlantı vermiyor.
    // Test ortamı APP_DEBUG olmadan koşuyorsa akışın yalnız "kayıt alındı"
    // bölümü doğrulanabilir; bu bir eksik değil, doğru güvenlik davranışı.
    const link = (kayit.body.mesaj.match(/\/bulten\/onay\/[a-f0-9]+/) || [])[0];
    if (link) {
      await goto(page, link);
      const text = await page.evaluate(() => document.body.textContent);
      assert.match(text, /aktif edildi|Aboneliğiniz/i, 'onay sayfası onaylandığını söylemeli: ' + text.slice(0, 200));
    } else {
      // Bağlantı dönmedi: ya SMTP gönderdi ya da SMTP yok + Debug kapalı.
      assert.match(
        kayit.body.mesaj,
        /gönderildi|gönderilemiyor/,
        'yanıt, onay e-postasının gönderildiğini ya da gönderilemediğini açıkça söylemeli: ' + kayit.body.mesaj,
      );
      // "Gönderilemiyor" diyorsa abonelik ONAYSIZ kalmalı (bülten gitmemeli).
      if (/gönderilemiyor/.test(kayit.body.mesaj)) {
        assert.match(
          kayit.body.mesaj,
          /onaylanmadan size bülten gönderilmez/i,
          'onaysız aboneye bülten gitmeyeceği belirtilmeli: ' + kayit.body.mesaj,
        );
      }
    }

    // Test kendi kaydını temizler (kullanıcı verisine dokunmaz).
    // Admin oturumu yoksa temizlik yapılamaz; bu durumda e-posta sabit
    // olduğu için ikinci çalıştırma yeni satır üretmez (upsert).
    if (process.env.TB_SESSION) {
      await goto(page, '/admin/aboneler');
      const silindi = await page.evaluate(async (email) => {
        const html = await (await fetch('/admin/aboneler', { credentials: 'same-origin' })).text();
        const i = html.indexOf(email);
        if (i < 0) return 'yok';
        const bas = html.lastIndexOf('<tr', i);
        const son = html.indexOf('</tr>', i);
        const satir = html.slice(bas, son);
        const m = satir.match(/\/admin\/aboneler\/bulten\/(\d+)\/sil/);
        if (!m) return 'id-yok';
        const r = await fetch('/admin/aboneler/bulten/' + m[1] + '/sil', {
          method: 'POST', credentials: 'same-origin', redirect: 'manual',
        });
        return String(r.status);
      }, 'e2e-onay@example.com');
      assert.ok(silindi !== 'id-yok', 'abone satırı silinebilmeli: ' + silindi);
    }
  });
});