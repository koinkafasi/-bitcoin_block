'use strict';
/**
 * Google ile giriş (okuyucu) — ayarlar, yönlendirme, state koruması.
 *
 * KAPSAM:
 *  1) Google ayarları panelinden kaydedilir,
 *  2) /giris/google Google yetkilendirmesine doğru parametrelerle yollar,
 *  3) callback'te hatalı state reddedilir (400, CSRF koruması),
 *  4) doğru state ile devam edilir (sahte kod → Google değişimi başarısız
 *     olur; 400/502 ile biter, asıl olan state'in KABUL edilmesi).
 *
 * Not: Gerçek Google alışverişi dış ağ gerektirir; test yalnızca
 * tarafa kadar olan akışı doğrular. Temizlik: ayarlar sıfırlanır.
 */

const test = require('node:test');
const assert = require('node:assert/strict');
const { goto, setViewport, withAdminPage } = require('../lib/harness');

const SID = process.env.TB_SESSION;

test('google giriş: ayarlar, yönlendirme ve state koruması', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdminPage(async (page, errors) => {
    await setViewport(page, 1400, 950);
    await goto(page, '/admin/members');
    assert.ok(!page.url().includes('/admin/giris'), 'üyelik paneli açık olmalı');

    try {
      // 1) Ayarlar
      const kayit = await page.evaluate(async () => {
        const fd = new URLSearchParams({ client_id: 'e2e.apps.googleusercontent.com', client_secret: 'e2e-secret' });
        const r = await fetch('/admin/members/google', { method: 'POST', credentials: 'same-origin', body: fd });
        return r.status;
      });
      assert.ok(kayit === 200 || kayit === 302, 'ayarlar kaydedilmeli: ' + kayit);
      await page.reload({ waitUntil: 'domcontentloaded' });
      const html = await page.evaluate(() => document.body.innerHTML);
      assert.ok(html.includes('Google ile Giriş (okuyucu)'), 'Google kartı görünmeli');
      assert.ok(html.includes('(kayıtlı)'), 'secret kayıtlı görünmeli');

      // 2) Yönlendirme: hedefi ağ isteklerinden yakalarız (dış ağ harness
      // tarafından engellenir; isteğin kendisi yeterli kanıttır).
      //
      // ÜST DÜZEY YÖNLENDİRME kullanılır: gerçek kullanıcı bağlantıya tıklar
      // ve tarayıcı sayfayı değiştirir. fetch() ile ölçmek yanıltıcıydı —
      // fetch CSP connect-src kapsamındadır, üst düzey yönlendirme değildir.
      const istekler = [];
      const dinle = (req) => istekler.push(req.url());
      page.on('request', dinle);
      try {
        await page.goto(new URL('/giris/google', page.url()).href,
                        { waitUntil: 'domcontentloaded', timeout: 15000 });
      } catch (_) {
        /* dış ağ engelli — istek yine de kaydedildi */
      }
      await new Promise((r) => setTimeout(r, 300));
      page.off('request', dinle);
      // Yönlendirme bizi Google'a götürdü; sonraki adımlar kendi
      // kaynağımızdan çalışmalı.
      await goto(page, '/admin/members');
      const loc = istekler.find((u) => u.includes('accounts.google.com')) || '';
      assert.ok(loc, 'Google yetkilendirme adresine gitmeli: ' + istekler.slice(0, 3).join(', '));
      assert.ok(loc.includes('client_id=e2e.apps.googleusercontent.com'), 'client_id taşınmalı');
      assert.ok(/state=[0-9a-f]{16,}/.test(loc), 'state üretilmeli');
      const hedef = { loc };

      // 3) Hatalı state → 400
      const hatali = await page.evaluate(async () => {
        const r = await fetch('/giris/google/callback?state=yanlis-kod&code=fake', { credentials: 'same-origin' });
        return r.status;
      });
      assert.equal(hatali, 400, 'hatalı state reddedilmeli');

      // 4) Doğru state (redirect URL'inden alınan) → akış devam eder;
      //    sahte kodla Google değişimi başarısız olur ama state KABUL edilir.
      const state = (hedef.loc.match(/state=([0-9a-f]+)/) || [])[1];
      const dogru = await page.evaluate(async (st) => {
        const r = await fetch('/giris/google/callback?state=' + st + '&code=sahte-kod', { credentials: 'same-origin' });
        return { status: r.status, text: (await r.text()).slice(0, 120) };
      }, state);
      assert.notEqual(dogru.status, 400, 'doğru state 400 dönmemeli: ' + dogru.text);
      assert.ok(!dogru.text.includes('Geçersiz oturum durumu'), 'state doğrulanmış olmalı');
    } finally {
      // Temizlik: Google ayarlarını sıfırla.
      await page.evaluate(async () => {
        await fetch('/admin/members/google', {
          method: 'POST', credentials: 'same-origin',
          body: new URLSearchParams({ client_id: '', client_secret: '' }),
        });
      });
    }
    assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
  });
});