'use strict';
/**
 * Sayfa Tasarımcısı > Yerleşim & Ölçü > Site içerik genişliği.
 * Üst menü, ana gövde ve footer aynı genişliği kullanmalı; footer'daki
 * eski 95rem taşması geri gelmemeli.
 */

const test = require('node:test');
const assert = require('node:assert/strict');
const { BASE_URL, goto, setViewport, withAdminPage, withPage } = require('../lib/harness');

const SID = process.env.TB_SESSION;

test('site genişliği ayarı header, gövde ve footer için ortak', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  let configured = 0;
  await withAdminPage(async (page, errors) => {
    await setViewport(page, 1536, 900);
    await goto(page, '/admin/tasarim');
    assert.ok(!page.url().includes('/admin/giris'), 'admin oturumu açık olmalı');
    const field = await page.$eval('[data-tok="siteWidth"]', (el) => ({
      value: parseInt(el.value, 10),
      min: parseInt(el.getAttribute('min'), 10),
      max: parseInt(el.getAttribute('max'), 10),
    }));
    assert.ok(field.value >= field.min && field.value <= field.max, 'genişlik aralıkta olmalı');
    configured = field.value;
    assert.deepEqual(errors, [], 'admin sayfasında JS hatası olmamalı');
  });

  await withPage(async (page, errors) => {
    await setViewport(page, 1536, 900);
    await goto(page, BASE_URL + '/');
    const m = await page.evaluate(() => {
      const w = (sel) => {
        const el = document.querySelector(sel);
        return el ? Math.round(el.getBoundingClientRect().width) : null;
      };
      return {
        viewport: window.innerWidth,
        css: getComputedStyle(document.documentElement).getPropertyValue('--site-width').trim(),
        header: w('.site-header .container'),
        main: w('main .container'),
        footer: w('.site-footer .cols'),
      };
    });
    const expected = parseInt(m.css || '1280', 10);
    assert.equal(m.viewport, 1536);
    assert.equal(expected, configured, 'CSS değişkeni admin ayarıyla aynı olmalı');
    assert.equal(m.header, configured, 'üst menü ayarlanan genişlikte olmalı');
    assert.equal(m.main, configured, 'ana gövde ayarlanan genişlikte olmalı');
    assert.equal(m.footer, configured, 'footer ayarlanan genişlikte olmalı');
    assert.equal(m.header, expected, 'CSS değişkeni ile üst menü uyuşmalı');
    assert.notEqual(m.footer, 1520, 'footer eski 95rem genişliğinde kalmamalı');
    assert.deepEqual(errors, [], 'public sayfada JS hatası olmamalı');
  }, { width: 1536 });
});
