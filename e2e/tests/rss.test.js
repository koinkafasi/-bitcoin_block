'use strict';
/**
 * RSS listesi (/rss-listesi), site ve kategori RSS akışları, footer linki.
 */
const test = require('node:test');
const assert = require('node:assert/strict');
const { BASE_URL, goto, setViewport, withAdminPage } = require('../lib/harness');

const SID = process.env.TB_SESSION;

test('RSS listesi ve kategori akışları', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdminPage(async (page, errors) => {
    await setViewport(page, 1440, 1000);
    await goto(page, '/rss-listesi');

    const pageInfo = await page.evaluate(() => ({
      h1: (document.querySelector('h1') || {}).textContent || '',
      canonical: (document.querySelector('link[rel="canonical"]') || {}).href || '',
      cards: [...document.querySelectorAll('.rss-card')].map((a) => ({
        href: a.getAttribute('href'), text: a.textContent.trim(),
      })),
      footer: (() => {
        const cols = [...document.querySelectorAll('.foot-col')];
        const corporate = cols.find((c) => (c.querySelector('.fh') || {}).textContent?.trim() === 'Kurumsal');
        const a = corporate && [...corporate.querySelectorAll('a')].find((x) => x.getAttribute('href') === '/rss-listesi');
        return a ? { href: a.getAttribute('href'), text: a.textContent.trim() } : null;
      })(),
      socialRSS: !!document.querySelector('.hp-social .hp-s-rss, .hp-social .foot-rss-text'),
    }));
    assert.ok(pageInfo.h1.includes('RSS'), 'RSS başlığı olmalı');
    assert.ok(pageInfo.canonical.endsWith('/rss-listesi'), 'canonical doğru olmalı');
    assert.ok(pageInfo.cards.length >= 5, 'RSS akış kartları listelenmeli: ' + pageInfo.cards.length);
    const urls = pageInfo.cards.map((c) => c.href);
    assert.ok(urls.some((u) => u.endsWith('/rss')), 'tüm haberler akışı olmalı');
    assert.ok(urls.some((u) => u.endsWith('/rss?kategori=gundem')), 'Gündem akışı olmalı');
    assert.ok(urls.some((u) => u.endsWith('/rss?kategori=ekonomi')), 'Ekonomi akışı olmalı');
    assert.ok(pageInfo.footer && pageInfo.footer.href === '/rss-listesi',
      'RSS Akışları bağlantısı Kurumsal sütununda olmalı');
    assert.equal(pageInfo.socialRSS, false, 'RSS simgesi sosyal medya satırında olmamalı');

    // Site akışı
    const feed = await page.goto(BASE_URL + '/rss', { waitUntil: 'domcontentloaded' });
    assert.equal(feed.status(), 200);
    assert.ok((feed.headers()['content-type'] || '').includes('application/rss+xml'));
    const siteXML = await feed.text();
    assert.ok(siteXML.includes('<rss') && siteXML.includes('<channel>'), 'geçerli RSS olmalı');
    assert.ok(siteXML.includes('<link>'), 'akış bağlantısı olmalı');

    // Kategori akışı
    const catFeed = await page.goto(BASE_URL + '/rss?kategori=ekonomi', { waitUntil: 'domcontentloaded' });
    assert.equal(catFeed.status(), 200);
    const catXML = await catFeed.text();
    assert.ok(catXML.includes('Ekonomi —'), 'kategori başlığı akışta olmalı');
    assert.ok(catXML.includes('/kategori/ekonomi'), 'kategori bağlantısı akışta olmalı');

    // Eski adres korunuyor
    const old = await page.goto(BASE_URL + '/rss.xml', { waitUntil: 'domcontentloaded' });
    assert.equal(old.status(), 200);
    assert.ok((await old.text()).includes('<rss'));

    // Bilinmeyen kategori 404
    const missing = await page.goto(BASE_URL + '/rss?kategori=yok-boyle-bir-sey', { waitUntil: 'domcontentloaded' });
    assert.equal(missing.status(), 404);

    assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
  });
});
