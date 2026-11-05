'use strict';
/**
 * SEO — sayfa bazlı meta, canonical, Open Graph, BreadcrumbList, noindex ve sitemap.
 */
const test = require('node:test');
const assert = require('node:assert/strict');
const { BASE_URL, goto, setViewport, withAdminPage } = require('../lib/harness');

const SID = process.env.TB_SESSION;

test('SEO meta, canonical, OG, breadcrumb ve sitemap', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdminPage(async (page, errors) => {
    await setViewport(page, 1440, 1000);

    await goto(page, '/');
    const home = await page.evaluate(() => ({
      title: document.title,
      desc: (document.querySelector('meta[name="description"]') || {}).content || '',
      canonical: (document.querySelector('link[rel="canonical"]') || {}).href || '',
      ogType: (document.querySelector('meta[property="og:type"]') || {}).content || '',
      twitter: (document.querySelector('meta[name="twitter:card"]') || {}).content || '',
    }));
    assert.ok(home.title.length > 0 && home.desc.length > 0, 'ana sayfada başlık/açıklama olmalı');
    assert.ok(!/^(.+) — \1$/.test(home.title), 'ana sayfa başlığı tekrarlanmamalı');
    assert.ok(home.canonical.endsWith('/'), 'ana sayfa self-canonical taşımalı');
    assert.equal(home.ogType, 'website');
    assert.ok(home.twitter.length > 0, 'twitter card olmalı');

    await goto(page, '/');
    const href = await page.evaluate(() => {
      const a = [...document.querySelectorAll('a[href]')].find((x) =>
        /\/[^/]+-[0-9a-f]{4,8}-\d+$/.test(x.getAttribute('href') || '')
      );
      return a ? a.getAttribute('href') : null;
    });
    assert.ok(href, 'makale bağlantısı yok');
    await goto(page, href);
    const art = await page.evaluate(() => {
      const meta = (n, attr = 'name') => {
        const el = document.querySelector(`meta[${attr}="${n}"]`);
        return el ? el.getAttribute('content') : '';
      };
      const ld = [...document.querySelectorAll('script[type="application/ld+json"]')].map((s) => s.textContent);
      return {
        title: document.title,
        desc: meta('description'),
        robots: meta('robots'),
        canonical: (document.querySelector('link[rel="canonical"]') || {}).href || '',
        ogTitle: meta('og:title', 'property'),
        ogImage: meta('og:image', 'property'),
        ogType: meta('og:type', 'property'),
        breadcrumb: ld.filter((x) => x.includes('BreadcrumbList')).length,
        newsArticle: ld.filter((x) => x.includes('NewsArticle')).length,
        path: location.pathname,
      };
    });
    assert.ok(art.desc.length >= 50, 'makale açıklaması içerikten gelmeli');
    assert.ok(art.robots.includes('index'), 'makale indekslenebilir olmalı');
    assert.equal(new URL(art.canonical).pathname, art.path, 'canonical kendi adresi olmalı');
    assert.equal(art.ogType, 'article');
    assert.ok(art.ogTitle.includes('—'), 'OG başlığı marka ekiyle gelmeli');
    assert.ok(art.ogImage.startsWith('http'), 'OG görseli mutlak olmalı');
    assert.equal(art.breadcrumb, 1, 'BreadcrumbList JSON-LD olmalı');
    assert.ok(art.newsArticle >= 1, 'NewsArticle JSON-LD olmalı');

    // Kategori: canonical sayfa parametresini korumalı
    await goto(page, '/kategori/dunya?s=2');
    const cat = await page.evaluate(() => ({
      canonical: (document.querySelector('link[rel="canonical"]') || {}).href || '',
      desc: (document.querySelector('meta[name="description"]') || {}).content || '',
      breadcrumb: [...document.querySelectorAll('script[type="application/ld+json"]')]
        .filter((s) => s.textContent.includes('BreadcrumbList')).length,
    }));
    assert.ok(cat.canonical.endsWith('?s=2'), 'kategori canonical sayfa parametresini korumalı');
    assert.ok(cat.desc.toLowerCase().includes('dünya'), 'kategori açıklaması kategori adını içermeli');
    assert.equal(cat.breadcrumb, 1, 'kategori BreadcrumbList taşımalı');

    // Arama: noindex
    await goto(page, '/arama?q=test');
    const robots = await page.$eval('meta[name="robots"]', (m) => m.content);
    assert.ok(robots.includes('noindex'), 'arama sonuçları noindex olmalı');

    // Sitemap: arama yok, yazarlar var
    await page.goto(BASE_URL + '/sitemap.xml', { waitUntil: 'domcontentloaded' });
    const xml = await page.content();
    assert.ok(!xml.includes('/arama'), 'sitemap arama sayfasını içermemeli');
    assert.ok(xml.includes('/yazarlar'), 'sitemap yazar dizinini içermeli');
    assert.ok(xml.includes('/yazar/'), 'sitemap yazar profillerini içermeli');

    assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
  });
});
