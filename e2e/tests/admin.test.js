'use strict';
/**
 * Yetkilendirme (permission) testleri.
 *
 * Kritik güvenlik davranışı: oturumsuz kullanıcı /admin altına girmeye
 * çalışınca giriş sayfasına yönlendirilmelidir — panel içeriği sızdırılmaz.
 */

const test = require('node:test');
const assert = require('node:assert/strict');
const { BASE_URL, goto, withPage } = require('../lib/harness');

const ADMIN_PATHS = [
  '/admin',
  '/admin/icerikler',
  '/admin/kategoriler',
  '/admin/ayarlar',
  '/admin/tasarim',
  '/admin/servisler',
  '/admin/merkez',
  '/admin/reklamlar',
];

test('oturumsuz /admin erişimi girişe yönlendirir', async () => {
  await withPage(async (page) => {
    for (const path of ADMIN_PATHS) {
      const resp = await goto(page, path);
      const finalUrl = page.url();
      assert.ok(
        finalUrl.includes('/admin/giris'),
        `${path} girişe yönlendirilmeli, gidilen: ${finalUrl}`
      );
      assert.ok(resp.status() < 400 || resp.status() === 302, `${path} beklenmeyen durum: ${resp.status()}`);

      // panel içeriği sızmamalı
      const text = await page.evaluate(() => document.body.innerText);
      assert.doesNotMatch(
        text,
        /Site Yönetim Merkezi|Genel Ayarlar/,
        `${path} panel içeriği sızdırmamalı`
      );
    }
  });
});

test('giriş sayfası form içerir', async () => {
  await withPage(async (page) => {
    await goto(page, '/admin/giris');
    assert.ok(await page.$('input[type="password"]'), 'parola alanı olmalı');
    assert.ok(await page.$('form'), 'giriş formu olmalı');
  });
});

test('yetkisiz POST admin endpoint\'lerine ızgaraya alınmaz', async () => {
  // Node fetch kullanılır (tarayıcı bağlamından değil): yönlendirme
  // davranışını net görebilmek ve CORS etkisi olmaması için.
  const paths = [
    ['/admin/tasarim/kaydet', { sayfa: 'home', blocks: '[]' }],
    ['/admin/kategori-panelleri/kaydet', { section: 'ana' }],
    ['/admin/olcu/kaydet', { section: 'ana' }],
    ['/admin/arka-plan/kaydet', { section: 'ana' }],
    ['/admin/servisler/ayar', { fikstur_key: 'kötü' }],
    ['/admin/menu/kaydet', { section: 'ana' }],
  ];

  for (const [path, body] of paths) {
    const res = await fetch(BASE_URL + path, {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
      body: new URLSearchParams(body).toString(),
      redirect: 'manual',
    });
    assert.notEqual(res.status, 200, `${path} oturumsuz POST'u kabul etti!`);
    assert.ok(
      res.status === 0 || res.status >= 300,
      `${path} oturumsuz POST ${res.status} döndü — yazma yapılmamalıydı`
    );
    if (res.status === 0) {
      // opaqueredirect: yönlendirme hedefi giriş olmalı
      const followed = await fetch(BASE_URL + path, {
        method: 'POST',
        headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
        body: new URLSearchParams(body).toString(),
        redirect: 'follow',
      });
      const finalUrl = followed.url;
      assert.ok(
        finalUrl.includes('/admin/giris'),
        `${path} girişe yönlendirmeli, gidilen: ${finalUrl}`
      );
    } else {
      const loc = res.headers.get('location') || '';
      assert.ok(
        loc.includes('/admin/giris') || res.status === 401 || res.status === 403,
        `${path} -> ${res.status} Location=${loc}`
      );
    }
  }
});

test('sahte oturum çerezi kabul edilmez', async () => {
  await withPage(async (page) => {
    await page.setCookie({
      name: 'tb_session',
      value: 'sahte-oturum-123',
      domain: '127.0.0.1',
      path: '/',
    });
    await goto(page, '/admin');
    assert.ok(
      page.url().includes('/admin/giris'),
      'geçersiz oturumla panele girilmemeli'
    );
  });
});
