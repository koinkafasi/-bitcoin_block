'use strict';
/**
 * AI Kaynak Merkezi: kaynak listesi, kategori planı ve küme aksiyonları.
 */
const test = require('node:test');
const assert = require('node:assert/strict');
const { goto, setViewport, withAdminPage } = require('../lib/harness');

const SID = process.env.TB_SESSION;

test('AI kaynak merkezi sayfası', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdminPage(async (page, errors) => {
    await setViewport(page, 1440, 1000);
    await goto(page, '/admin/ai-kaynak');

    const info = await page.evaluate(() => ({
      title: (document.querySelector('h2') || {}).textContent || '',
      scan: !!document.querySelector('form[action="/admin/ai-kaynak/tara"]'),
      enrich: !!document.querySelector('form[action="/admin/ai-kaynak/zenginlestir"]'),
      feeds: [...document.querySelectorAll('a[href*="gzt.com/rss"]')].map((a) => a.textContent.trim()),
      clusters: document.querySelectorAll('form[action*="/ai-kaynak/"][action$="/taslak"]').length,
      plan: document.body.textContent.includes('Kategori planı'),
      production: document.body.textContent.includes('Kategori üretim planı'),
      planButtons: document.querySelectorAll('button[formaction*="/admin/ai-kaynak/plan/"]').length,
      trend: document.body.textContent.includes('Trend eşleşmeleri'),
      bank: document.body.textContent.includes('Kaynak bankası'),
      sources: ['AA Gündem', 'TRT Haber', 'NTV Son Dakika', 'CNN Türk', 'Hürriyet', 'BBC Türkçe', 'MSN Türkiye', 'DHA', 'İHA', 'Technopat']
        .filter((name) => document.body.textContent.includes(name)),
      dhaRow: [...document.querySelectorAll('table tbody tr')].map((r) => r.textContent).find((t) => t.includes('DHA') && t.includes('dha.com.tr')) || '',
      msnRow: [...document.querySelectorAll('table tbody tr')].map((r) => r.textContent).find((t) => t.includes('MSN Türkiye') && t.includes('msn.com')) || '',
      sitemapRows: [...document.querySelectorAll('table tbody tr')].filter((r) => r.textContent.includes('Sitemap')).length,
    }));
    assert.ok(info.title.includes('AI Kaynak Merkezi'), 'başlık doğru olmalı');
    assert.ok(info.scan, 'tara düğmesi olmalı');
    assert.ok(info.enrich, 'özet zenginleştirme düğmesi olmalı');
    assert.ok(info.feeds.length >= 5, 'GZT RSS kaynakları listelenmeli: ' + info.feeds.length);
    assert.ok(info.clusters >= 1, 'en az bir benzer haber kümesi olmalı');
    assert.ok(info.plan, 'kategori planı görünmeli');
    assert.ok(info.production, 'kategori üretim planı görünmeli');
    assert.ok(info.planButtons >= 5, 'kategori planı çalıştırma düğmeleri olmalı: ' + info.planButtons);
    assert.ok(info.trend, 'trend eşleşmeleri görünmeli');
    assert.ok(info.bank, 'kaynak bankası görünmeli');
    assert.equal(info.sources.length, 10, 'ulusal kaynaklar bankada olmalı: ' + info.sources.join(', '));
    assert.ok(info.dhaRow.includes('HTML'), 'DHA HTML kaynağı olarak görünmeli');
    assert.ok(info.msnRow.includes('Bing RSS'), 'MSN Bing RSS kaynağı olarak görünmeli');
    assert.ok(info.sitemapRows >= 2, 'BirGün ve DonanımHaber sitemap kaynağı olarak görünmeli: ' + info.sitemapRows);

    // Küme detayındaki kaynak adları görünür.
    const meta = await page.evaluate(() =>
      [...document.querySelectorAll('.card-pad .small.muted')].map((x) => x.textContent).join(' '));
    assert.ok(/kaynak/.test(meta), 'kümelerde kaynak sayısı olmalı');

    assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
  });
});
