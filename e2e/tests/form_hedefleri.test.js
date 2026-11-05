'use strict';
/**
 * Kamuya açık form hedefleri denetimi.
 *
 * NEDEN: Şablonlar form action'larını `{{.LangBase}}/...` ile kuruyor;
 * LangBase "/tr" olabildiği için hedef /tr/... olabilir. Hedef kayıtlı
 * değilse form 404/405 döner ve kullanıcı işlemi sessizce kaybolur.
 *
 * YÖNTEM: Herkese açık sayfaları gezer, method="post" formların
 * action'larını toplar ve her birine BOŞ gövdeyle POST atar. Boş gövde
 * doğrulama hatası üretir; veri yazılmaz. Yalnız "hedef yok / metot yok"
 * (404/405) hata sayılır.
 */

const test = require('node:test');
const assert = require('node:assert/strict');
const { goto, setViewport, withPage, BASE_URL } = require('../lib/harness');

const SAYFALAR = ['/', '/gazete', '/bulten', '/giris', '/kayit', '/abonelik', '/iletisim'];

test('public form hedefleri 404/405 dönmemeli', async (t) => {
  await withPage(async (page, errors) => {
    await setViewport(page, 1400, 950);

    const hedefler = new Set();
    for (const p of SAYFALAR) {
      const resp = await page.goto(BASE_URL + p, { waitUntil: 'domcontentloaded' });
      if (!resp || resp.status() !== 200) continue; // sayfa yoksa atla (kullanıcı verisi)
      const list = await page.evaluate(() => {
        const out = [];
        document.querySelectorAll('form').forEach((f) => {
          if ((f.getAttribute('method') || 'get').toLowerCase() !== 'post') return;
          const a = f.getAttribute('action');
          if (!a) return;
          // Üyelik/abonelik dışı formlar da denetlenir.
          out.push(a);
        });
        return out;
      });
      list.forEach((a) => hedefler.add(a));
    }

    assert.ok(hedefler.size > 0, 'en az bir POST formu bulunmalı');

    const hatalar = [];
    for (const action of hedefler) {
      const url = action.startsWith('http') ? action : BASE_URL + action;
      const sonuc = await page.evaluate(async (u) => {
        const r = await fetch(u, {
          method: 'POST', credentials: 'same-origin', redirect: 'follow',
          headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
          body: '',
        });
        return r.status;
      }, url);
      if (sonuc === 404 || sonuc === 405) {
        hatalar.push(action + ' -> ' + sonuc);
      }
    }
    assert.deepEqual(hatalar, [], 'form hedefleri çözümlenmeli: ' + hatalar.join(', '));

    assert.deepEqual(errors, [], 'sayfa hatası olmamalı');
  });
});
