'use strict';
/**
 * Üye sistemi — public yüzey (giriş, kayıt, fiyat sayfası, header simgesi).
 *
 * KAPSAM (salt-okunur; DB'ye veri yazmaz):
 *  1) Header'da arama yanında üye simgesi var ve /giris'e gidiyor,
 *  2) /giris formu (e-posta, şifre, şifremi unuttum, KVKK'lı kayıt bağlantısı),
 *  3) /kayit formu (ad, e-posta, şifre, KVKK onayı),
 *  4) /abonelik sayfası 5 kademeyi listeler, aylık/yıllık geçişi çalışır,
 *  5) misafir "Giriş yapıp abone ol" ile /giris?redirect=abonelik'e gider,
 *  6) /hesabim oturumsuz kullanıcıyı /giris'e yönlendirir.
 */

const test = require('node:test');
const assert = require('node:assert/strict');
const { goto, setViewport, withPage, BASE_URL } = require('../lib/harness');

test('üye sistemi: header simgesi + giriş/kayıt formları', async (t) => {
  await withPage(async (page, errors) => {
    await setViewport(page, 1400, 950);
    await goto(page, '/');

    const header = await page.evaluate(() => {
      const btn = document.querySelector('.header-tools .member-btn');
      const search = document.querySelector('.header-tools .search-btn');
      return { var: !!btn, href: btn ? btn.getAttribute('href') : '', searchVar: !!search };
    });
    assert.ok(header.searchVar, 'arama düğmesi görünmeli');
    assert.ok(header.var, 'üye simgesi arama yanında görünmeli');
    assert.equal(header.href, '/giris', 'misafir üye simgesi /giris açmalı');

    await goto(page, '/giris');
    const giris = await page.evaluate(() => ({
      form: !!document.querySelector('form[action$="/giris"]'),
      login: !!document.querySelector('input[name="login"]'),
      pass: !!document.querySelector('input[name="password"]'),
      forgot: !!document.querySelector('a[href$="/sifremi-unuttum"]'),
      kayit: !!document.querySelector('a[href*="/kayit"]'),
      honeypot: !!document.querySelector('input[name="website"]'),
      // Faz 198 "huddle" yeniden tasarımı: başlık artık .panel__title ve
      // metin "Tekrar hoş geldiniz". Eski .uye-title / "Giriş Yap" kaldırıldı.
      baslik: (document.querySelector('.panel__title, .uye-title') || {}).textContent || '',
    }));
    assert.ok(giris.form && giris.login && giris.pass, 'giriş formu alanları');
    assert.ok(giris.forgot, 'şifremi unuttum bağlantısı');
    assert.ok(giris.kayit, 'kayıt bağlantısı');
    assert.ok(giris.honeypot, 'bot tuzağı alanı bulunmalı');
    assert.match(giris.baslik, /(giriş|hoş geldiniz)/i, 'giriş başlığı: ' + giris.baslik);

    await goto(page, '/kayit');
    const kayit = await page.evaluate(() => ({
      form: !!document.querySelector('form[action$="/kayit"]'),
      name: !!document.querySelector('input[name="name"]'),
      email: !!document.querySelector('input[name="email"]'),
      p1: !!document.querySelector('input[name="password"]'),
      p2: !!document.querySelector('input[name="password2"]'),
      terms: !!document.querySelector('input[name="terms"]'),
      kvkk: document.body.textContent.includes('KVKK'),
    }));
    assert.ok(kayit.form && kayit.name && kayit.email && kayit.p1 && kayit.p2, 'kayıt alanları');
    assert.ok(kayit.terms, 'KVKK/kullanım koşulları onayı zorunlu');
    assert.ok(kayit.kvkk, 'KVKK metni bağlantısı');

    assert.deepEqual(errors, [], 'sayfa hatası olmamalı');
  });
});

test('üye sistemi: fiyat sayfası kademeler ve dönem geçişi', async (t) => {
  await withPage(async (page, errors) => {
    await setViewport(page, 1400, 950);
    const resp = await goto(page, '/abonelik');
    assert.equal(resp.status(), 200);

    const sayfa = await page.evaluate(() => {
      const kartlar = [...document.querySelectorAll('.abon-card')];
      return {
        adet: kartlar.length,
        adlar: kartlar.map((k) => k.querySelector('h2')?.textContent.trim()),
        aylik: kartlar[0]?.querySelector('[data-monthly]')?.textContent || '',
        yillik: kartlar[0]?.querySelector('[data-annual]')?.textContent || '',
        hediye: document.body.textContent.includes('2 ay hediye'),
        misafirCta: document.querySelector('a.abon-cta')?.getAttribute('href') || '',
      };
    });
    assert.ok(sayfa.adet >= 5, 'en az 5 kademe satılmalı, gelen: ' + sayfa.adet);
    for (const ad of ['Filiz', 'Bronz', 'Gümüş', 'Altın', 'Elmas']) {
      assert.ok(sayfa.adlar.includes(ad), ad + ' kademesi listelenmeli');
    }
    assert.match(sayfa.aylik, /150/);
    assert.ok(sayfa.hediye, 'yıllık "2 ay hediye" rozeti');
    assert.ok(sayfa.misafirCta.endsWith('/giris?redirect=abonelik'), 'misafir CTA: ' + sayfa.misafirCta);

    // Yıllık döneme geç: fiyat ve gizli alan güncellenmeli.
    await page.click('[data-cycle="annual"]');
    const sonra = await page.evaluate(() => {
      const kart = document.querySelector('.abon-card');
      const m = kart.querySelector('[data-monthly]');
      const y = kart.querySelector('[data-annual]');
      return { mGizli: m.hasAttribute('hidden'), yGorunur: !y.hasAttribute('hidden'), y: y.textContent };
    });
    assert.ok(sonra.mGizli, 'aylık fiyat gizlenmeli');
    assert.ok(sonra.yGorunur, 'yıllık fiyat görünmeli');
    assert.match(sonra.y, /1500/);

    assert.deepEqual(errors, [], 'sayfa hatası olmamalı');
  });
});

test('üye sistemi: hesabım oturumsuz girişe yönlendirir', async (t) => {
  await withPage(async (page) => {
    const resp = await goto(page, '/hesabim');
    assert.equal(resp.status(), 200);
    assert.ok(page.url().includes('/giris'), 'oturumsuz kullanıcı /giris?redirect=hesabim görmeli');
    assert.ok(page.url().includes('redirect=hesabim'));
  });
});

test('üye sistemi: abonelik ödeme oturumsuz girişe yönlendirir', async (t) => {
  await withPage(async (page) => {
    await goto(page, '/');
    const sonuc = await page.evaluate(async (base) => {
      const r = await fetch(base + '/abonelik/odeme', {
        method: 'POST', credentials: 'same-origin',
        headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
        body: 'tier=filiz', redirect: 'manual',
      });
      return { status: r.status, type: r.type };
    }, BASE_URL);
    assert.ok([0, 302].includes(sonuc.status) || sonuc.type === 'opaqueredirect',
      'ödemeye oturumsuz erişim yönlendirmeli: ' + JSON.stringify(sonuc));
  });
});
