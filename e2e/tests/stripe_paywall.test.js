'use strict';
/**
 * Stripe abonelik → paywall (üyelik) tam döngüsü.
 *
 * KAPSAM:
 *  1) Stripe ayarları admin panelinden kaydedilir,
 *  2) imzasız webhook reddedilir (400),
 *  3) imzalı checkout.session.completed premium üyelik açar (kullanıcı
 *     oluşturulur, Stripe kimlikleri bağlanır),
 *  4) aynı olay tekrar işlenmez (idempotens),
 *  5) subscription.deleted üyeliği kapatır.
 *
 * Temizlik: test kendi kullanıcı/olay kayıtlarını siler ve ayarları sıfırlar.
 */

const test = require('node:test');
const assert = require('node:assert/strict');
const crypto = require('node:crypto');
const { goto, setViewport, withAdminPage } = require('../lib/harness');

const SID = process.env.TB_SESSION;
const SECRET = 'whsec_e2e_test';
const EMAIL = 'e2e-stripe@test.com';

function sign(body, ts) {
  const mac = crypto.createHmac('sha256', SECRET).update(ts + '.' + body).digest('hex');
  return 't=' + ts + ',v1=' + mac;
}

test('stripe: ayarlar, webhook imzası, premium üyelik ve idempotens', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdminPage(async (page, errors) => {
    await setViewport(page, 1400, 950);
    await goto(page, '/admin/members');
    assert.ok(!page.url().includes('/admin/giris'), 'üyelik paneli açık olmalı');

    try {
      // 1) Ayarlar
      const kayit = await page.evaluate(async (secret) => {
        const fd = new URLSearchParams({
          aktif: '1', secret_key: 'sk_test_e2e', webhook_secret: secret, price_id: 'price_e2e',
        });
        const r = await fetch('/admin/members/stripe', { method: 'POST', credentials: 'same-origin', body: fd });
        return r.status;
      }, SECRET);
      // fetch yönlendirmeleri takip eder: 302 (redirect) veya takip sonrası 200.
      assert.ok(kayit === 302 || kayit === 200, 'ayarlar kaydedilmeli: ' + kayit);
      await page.reload({ waitUntil: 'domcontentloaded' });
      const panel = await page.evaluate(() => document.body.innerHTML);
      assert.ok(panel.includes('Stripe Ödeme (abonelik)'), 'Stripe kartı görünmeli');
      assert.ok(panel.includes('(kayıtlı)'), 'sır kayıtlı görünmeli');

      // 2) İmzasız webhook → 400
      const imzasiz = await page.evaluate(async () => {
        const r = await fetch('/webhooks/stripe', {
          method: 'POST', credentials: 'same-origin',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ id: 'evt_t0_' + Date.now(), type: 'ping' }),
        });
        return r.status;
      });
      assert.equal(imzasiz, 400, 'imzasız webhook reddedilmeli');

      // 3) İmzalı checkout tamamlanması → premium
      const benzersiz = Date.now();
      const evt = JSON.stringify({
        id: 'evt_t1_' + benzersiz, type: 'checkout.session.completed',
        data: { object: { customer: 'cus_t1', subscription: 'sub_t1', customer_email: EMAIL, metadata: {} } },
      });
      const ts = Math.floor(Date.now() / 1000);
      const ok1 = await page.evaluate(async (payload, sig) => {
        const r = await fetch('/webhooks/stripe', {
          method: 'POST', credentials: 'same-origin',
          headers: { 'Content-Type': 'application/json', 'Stripe-Signature': sig },
          body: payload,
        });
        return r.status;
      }, evt, sign(evt, ts));
      assert.equal(ok1, 200, 'imzalı webhook işlenmeli');

      // Kullanıcı premium olmalı (admin listesinde görünür).
      await page.reload({ waitUntil: 'domcontentloaded' });
      const uye = await page.evaluate((email) => {
        const html = document.body.innerHTML;
        const i = html.indexOf(email);
        if (i < 0) return '';
        const bas = html.lastIndexOf('<tr', i);
        const son = html.indexOf('</tr>', i);
        return html.slice(bas, son);
      }, EMAIL);
      assert.ok(uye.includes(EMAIL), 'üye listede görünmeli');
      // Abonelik modelinde premium = atanmış katman rozeti.
      assert.ok(/data-premium="1"/.test(uye), 'üyelik premium/katman rozeti olmalı');

      // 4) Aynı olay tekrar → 200, durum değişmez (idempotens)
      const ok2 = await page.evaluate(async (payload, sig) => {
        const r = await fetch('/webhooks/stripe', {
          method: 'POST', credentials: 'same-origin',
          headers: { 'Content-Type': 'application/json', 'Stripe-Signature': sig },
          body: payload,
        });
        return r.status;
      }, evt, sign(evt, ts));
      assert.equal(ok2, 200, 'tekrar eden olay yine 200 dönmeli (idempotent)');

      // 5) Abonelik iptali → üyelik kapanır
      const iptal = JSON.stringify({
        id: 'evt_t2_' + benzersiz, type: 'customer.subscription.deleted',
        data: { object: { id: 'sub_t1', customer: 'cus_t1', status: 'canceled', metadata: {} } },
      });
      const ts2 = Math.floor(Date.now() / 1000);
      const ok3 = await page.evaluate(async (payload, sig) => {
        const r = await fetch('/webhooks/stripe', {
          method: 'POST', credentials: 'same-origin',
          headers: { 'Content-Type': 'application/json', 'Stripe-Signature': sig },
          body: payload,
        });
        return r.status;
      }, iptal, sign(iptal, ts2));
      assert.equal(ok3, 200, 'iptal olayı işlenmeli');
      await page.reload({ waitUntil: 'domcontentloaded' });
      const son = await page.evaluate((email) => {
        const html = document.body.innerHTML;
        const i = html.indexOf(email);
        if (i < 0) return '';
        const bas = html.lastIndexOf('<tr', i);
        const bitis = html.indexOf('</tr>', i);
        return html.slice(bas, bitis);
      }, EMAIL);
      assert.ok(son.includes(EMAIL), 'üye satırı durmalı');
      assert.ok(!/data-premium="1"/.test(son), 'üyelik kapanmış olmalı: ' + son.slice(0, 200));
    } finally {
      // Temizlik: oluşturulan üyeyi sil + Stripe ayarlarını sıfırla.
      await page.evaluate(async (email) => {
        const html = await (await fetch('/admin/members', { credentials: 'same-origin' })).text();
        const i = html.indexOf(email);
        if (i >= 0) {
          // Yalnızca kendi satırımızın silme bağlantısını kullan.
          const bas = html.lastIndexOf('<tr', i);
          const bitis = html.indexOf('</tr>', i);
          const satir = html.slice(bas, bitis);
          const m = satir.match(/\/admin\/members\/(\d+)\/sil/);
          if (m) {
            await fetch('/admin/members/' + m[1] + '/sil', { method: 'POST', credentials: 'same-origin' });
          }
        }
        await fetch('/admin/members/stripe', {
          method: 'POST', credentials: 'same-origin',
          body: new URLSearchParams({ aktif: '', price_id: '' }),
        });
      }, EMAIL);
    }
    assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
  });
});