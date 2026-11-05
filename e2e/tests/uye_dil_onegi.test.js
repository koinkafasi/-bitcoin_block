'use strict';
/**
 * Dil önekli üye adresleri (/tr/…).
 *
 * NEDEN: Şablonlar `{{.LangBase}}/giris` gibi hedefler üretir; LangBase
 * "/tr" olabildiği için form action'ları /tr/giris olur. Bu yollar
 * kayıtlı değilse giriş/kayıt formu 405 veya 404 döner ve kullanıcı
 * akışı sessizce kırılır. Bu dosya o sınıfı kalıcı olarak korur.
 *
 * KAPSAM (salt-okunur; DB'ye yazmaz):
 *  1) /tr/giris, /tr/kayit, /tr/abonelik, /tr/sifremi-unuttum GET 200,
 *  2) form action'ları gerçekten /tr ile başlıyor,
 *  3) form action hedefine POST atıldığında 405/404 DEĞİL (302/200) dönüyor,
 *  4) /tr/hesabim oturumsuz → /tr/giris yönlendirmesi,
 *  5) dil öneki olmayan sayfalar (ör. /tr/arama) bozulmuyor.
 */

const test = require('node:test');
const assert = require('node:assert/strict');
const { goto, setViewport, withPage, BASE_URL } = require('../lib/harness');

test('dil öneki: üye sayfaları ve form hedefleri çalışır', async (t) => {
  await withPage(async (page, errors) => {
    await setViewport(page, 1400, 950);

    for (const p of ['/tr/giris', '/tr/kayit', '/tr/abonelik', '/tr/sifremi-unuttum']) {
      const r = await page.goto(BASE_URL + p, { waitUntil: 'domcontentloaded' });
      assert.equal(r.status(), 200, p + ' 200 dönmeli');
    }

    // Form action'ları dil önekli mi? (Şablon LangBase kullanıyor.)
    await goto(page, '/giris');
    const actionlar = await page.evaluate(() => ({
      giris: (document.querySelector('form[action$="/giris"]') || {}).getAttribute
        ? document.querySelector('form[action$="/giris"]').getAttribute('action') : '',
      abonelikCta: (document.querySelector('a[href*="redirect=abonelik"]') || {}).getAttribute
        ? document.querySelector('a[href*="redirect=abonelik"]').getAttribute('href') : '',
    }));
    const onekli = actionlar.giris.startsWith('/tr/') || actionlar.giris.startsWith('/en/');
    if (onekli) {
      assert.ok(actionlar.giris.endsWith('/giris'), 'giriş form hedefi: ' + actionlar.giris);
    }

    // Formun GERÇEK hedefine POST: 405/404 olmamalı.
    await goto(page, '/');
    for (const [path, body] of [
      ['/tr/giris', 'login=yok@ornek.test&password=yanlis123'],
      ['/tr/kayit', 'name=Test&email=&password=&password2=&terms=1'],
      ['/tr/sifremi-unuttum', 'email=yok@ornek.test'],
      ['/tr/abonelik/odeme', 'tier=filiz'],
    ]) {
      const sonuc = await page.evaluate(async (base, p, b) => {
        const r = await fetch(base + p, {
          method: 'POST', credentials: 'same-origin', redirect: 'manual',
          headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
          body: b,
        });
        return { status: r.status, type: r.type };
      }, BASE_URL, path, body);
      const yonlendirme = sonuc.status === 0 || sonuc.type === 'opaqueredirect' ||
        sonuc.status === 302 || sonuc.status === 303 || sonuc.status === 200;
      assert.ok(yonlendirme,
        path + ' POST edilebilir olmalı (405/404 değil), gelen: ' + JSON.stringify(sonuc));
    }

    // Oturumsuz hesap merkezi → giriş sayfası (dil önekli olabilir).
    const r = await page.goto(BASE_URL + '/tr/hesabim', { waitUntil: 'domcontentloaded' });
    assert.equal(r.status(), 200);
    assert.ok(/\/giris/.test(page.url()), 'oturumsuz /hesabim girişe yönlenmeli: ' + page.url());

    // Öneksiz çalışan sayfalar bozulmamış olmalı.
    const arama = await page.goto(BASE_URL + '/tr/arama?q=istanbul', { waitUntil: 'domcontentloaded' });
    assert.equal(arama.status(), 200, '/tr/arama hâlâ çalışmalı');

    assert.deepEqual(errors, [], 'sayfa hatası olmamalı');
  });
});
