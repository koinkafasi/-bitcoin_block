'use strict';
/**
 * Faz 51 — yazar/editör dizini, kişi profili, yazı künyesi ve yönetim paneli.
 */
const test = require('node:test');
const assert = require('node:assert/strict');
const { goto, setViewport, withAdminPage } = require('../lib/harness');

const SID = process.env.TB_SESSION;

test('yazarlar dizini, kişi profili, künye ve yönetim', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdminPage(async (page, errors) => {
    await setViewport(page, 1440, 1000);

    await goto(page, '/yazarlar');
    const people = await page.$$eval('.person-card', (els) =>
      els.map((el) => ({
        name: el.querySelector('.person-name').textContent.trim(),
        href: el.getAttribute('href'),
        role: el.querySelector('.person-role').textContent.trim(),
      }))
    );
    assert.ok(people.length >= 1, 'en az bir yazar kartı olmalı');
    assert.ok(people[0].href.startsWith('/yazar/'), 'kart profil adresine gitmeli');
    assert.ok(people[0].role.length > 0, 'kartta rol/unvan görünmeli');

    await goto(page, people[0].href);
    assert.ok(await page.$('.person-head'), 'profil başlığı olmalı');
    const head = await page.evaluate(() => ({
      name: (document.querySelector('.person-name-lg') || {}).textContent || '',
      canonical: (document.querySelector('head link[rel="canonical"]') || {}).href || '',
      cardCount: document.querySelectorAll('.grid-2 .card').length,
    }));
    assert.ok(head.name.trim().length > 0, 'profil adı görünmeli');
    assert.ok(head.canonical.endsWith(people[0].href), 'kişi sayfası self-canonical taşımalı');

    // Yazı künyesi: yazar adı profile bağlanmalı.
    await goto(page, '/');
    const articleHref = await page.evaluate(() => {
      const a = [...document.querySelectorAll('a[href]')].find((x) =>
        /\/[^/]+-[0-9a-f]{4,8}-\d+$/.test(x.getAttribute('href') || '')
      );
      return a ? a.getAttribute('href') : null;
    });
    assert.ok(articleHref, 'makale bağlantısı bulunamadı');
    await goto(page, articleHref);
    const byline = await page.$eval('.art-byline-name', (a) => ({
      text: a.textContent.trim(), href: a.getAttribute('href'),
    }));
    assert.ok(byline.text.length > 0, 'künyede yazar adı olmalı');
    assert.ok(byline.href.startsWith('/yazar/'), 'künye yazar profiline bağlanmalı');

    // Yönetim: yazar listesi + içerik formunda atama alanları.
    await goto(page, '/admin/yazarlar');
    assert.ok(await page.$('#people-panel table'), 'yönetim listesi olmalı');
    await goto(page, '/admin/icerikler/yeni');
    assert.ok(await page.$('select[name="author_id"]'), 'içerik formunda yazar seçimi olmalı');
    assert.ok(await page.$('select[name="editor_id"]'), 'içerik formunda editör seçimi olmalı');

    assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
  });
});
