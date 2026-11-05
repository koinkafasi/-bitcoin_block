'use strict';
/**
 * Canlı içerik: admin kaydı sonrası yayındaki makale sayfası yenilenmeden
 * başlığın SSE + JSON snapshot ile güncellenmesini doğrular.
 */
const test = require('node:test');
const assert = require('node:assert/strict');
const { goto, setViewport, withAdminPage, BASE_URL } = require('../lib/harness');

const SID = process.env.TB_SESSION;

test('makale admin kaydında sayfa yenilenmeden güncellenir', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');

  await withAdminPage(async (page, errors) => {
    await setViewport(page, 1440, 950);
    await goto(page, '/');

    const links = await page.evaluate(() =>
      [...document.querySelectorAll('a[href]')]
        .map((a) => a.getAttribute('href'))
        .filter((h) => h && /-\d+$/.test(h) && !h.startsWith('/admin')));
    assert.ok(links.length > 0, 'ana sayfada makale bağlantısı olmalı');

    let contentID = '';
    for (const href of links.slice(0, 20)) {
      const url = href.startsWith('http') ? href : BASE_URL + href;
      await page.goto(url, { waitUntil: 'domcontentloaded', timeout: 45000 });
      contentID = await page.evaluate(() => {
        const m = document.querySelector('main[data-article-id]');
        return m ? m.getAttribute('data-article-id') : '';
      });
      if (contentID) break;
    }
    assert.ok(contentID, 'yayında bir makale bulunmalı');

    const originalTitle = (await page.$eval('#article-title', (el) => el.textContent.trim()));
    assert.ok(originalTitle.length > 0, 'makale başlığı okunmalı');
    await new Promise((resolve) => setTimeout(resolve, 700)); // EventSource bağlantısı kurulsun

    const admin = await page.browser().newPage();
    const adminErrors = [];
    admin.on('pageerror', (e) => adminErrors.push(String(e.message)));
    // Formun beforeunload uyarısı tarayıcı kapanışını kilitlemesin.
    admin.on('dialog', (d) => { d.accept().catch(() => {}); });
    await admin.setCacheEnabled(false);
    await admin.goto(`${BASE_URL}/admin/icerikler/${contentID}`, { waitUntil: 'networkidle2' });
    assert.ok(await admin.$('input[name="baslik"]'), 'içerik formu açılmalı');

    const newTitle = 'Canlı güncelleme testi ' + Date.now();
    const save = async (title) => {
      await admin.evaluate((value) => {
        const input = document.querySelector('input[name="baslik"]');
        input.value = value;
        input.dispatchEvent(new Event('input', { bubbles: true }));
        const form = input.closest('form');
        const btn = form.querySelector('button[name="kaydet"][value="yayinla"]');
        btn.click();
      }, title);
      // HTMX yanıtı formu yeniden basar; kısa bekleme yeterli.
      await new Promise((resolve) => setTimeout(resolve, 1200));
    };

    try {
      await save(newTitle);
      await page.bringToFront();
      await page.waitForFunction(
        (expected) => {
          const el = document.querySelector('#article-title');
          return !!el && el.textContent.trim() === expected;
        },
        { timeout: 15000 },
        newTitle,
      );
      assert.equal(await page.$eval('#article-title', (el) => el.textContent.trim()), newTitle);
    } finally {
      await save(originalTitle).catch(() => {});
      await page.bringToFront();
      await page.waitForFunction(
        (expected) => {
          const el = document.querySelector('#article-title');
          return !!el && el.textContent.trim() === expected;
        },
        { timeout: 15000 },
        originalTitle,
      ).catch(() => {});
      await admin.close();
      // SSE bağlantısını kapat; tarayıcı kapanışını beklemesin.
      await page.close().catch(() => {});
    }

    assert.deepEqual(errors, [], 'public sayfada JS hatası olmamalı');
    assert.deepEqual(adminErrors, [], 'admin sayfasında JS hatası olmamalı');
  });
});
