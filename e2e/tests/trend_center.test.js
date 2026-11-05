'use strict';
/**
 * Trend Merkezi — panel + public çıktılar (Faz 101-106).
 *
 * Admin testi TB_SESSION gerektirir. Dış ağa çıkılmaz; yalnızca panelin
 * render edildiği, kaynak sağlığının göründüğü ve filtre/HTMX akışının
 * çalıştığı doğrulanır. Public testler RSS + öneri uçlarını kontrol eder.
 */

const test = require('node:test');
const assert = require('node:assert/strict');
const { BASE_URL, goto, withAdminPage, withPage } = require('../lib/harness');

const SID = process.env.TB_SESSION;

test('Trend Merkezi paneli render edilir ve filtrelenir', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdminPage(async (page, errors) => {
    await goto(page, '/admin/bot');
    assert.ok(!page.url().includes('/admin/giris'), 'admin oturumu açık olmalı');

    const text = await page.evaluate(() => document.body.innerText);
    assert.match(text, /Trend Merkezi/, 'Trend Merkezi kartı görünmeli');
    assert.match(text, /Trend Kaynakları ve Sağlık/, 'kaynak sağlık tablosu görünmeli');
    assert.match(text, /Google Trends — Türkiye/, 'varsayılan Google Trends kaynağı listelenmeli');
    assert.match(text, /Search Console \(GSC\)/, 'GSC kartı görünmeli');
    assert.match(text, /Site İçi Arama Talebi/, 'site içi arama kartı görünmeli');
    // Tablo başlıkları CSS text-transform: uppercase ile büyük harf basılır;
    // innerText "SIRA (7G)" döner. Bu yüzden karşılaştırma büyük/küçük harf duyarsız.
    assert.match(text, /S[ıi]ra \(7g\)/i, 'sparkline sütunu görünmeli');

    // Filtre formu HTMX ile aynı paneli tazeler.
    await page.type('input[name="tq"]', 'murat');
    await page.select('select[name="tf"]', 'bosluk');
    await page.click('form[hx-get="/admin/bot"] button[type="submit"]');
    await page.waitForFunction(() => {
      const el = document.querySelector('input[name="tq"]');
      return el && el.value === 'murat';
    }, { timeout: 8000 });
    const after = await page.evaluate(() => document.body.innerText);
    assert.match(after, /Trend Merkezi/, 'filtre sonrası panel korunmalı');
    assert.deepEqual(errors, [], 'sayfa hatası olmamalı');
  });
});

test('Gündem RSS ve trend öneri ucu çalışır', async () => {
  const rss = await fetch(BASE_URL + '/gundem.xml');
  assert.equal(rss.status, 200, 'gundem.xml HTTP 200');
  assert.match(rss.headers.get('content-type') || '', /application\/rss\+xml/);
  const rssText = await rss.text();
  assert.match(rssText, /<rss[^>]*version="2\.0"/);
  assert.match(rssText, /<channel>/);

  const sug = await fetch(BASE_URL + '/api/trend-suggest?q=murat');
  assert.equal(sug.status, 200);
  const j = await sug.json();
  assert.equal(j.success, true);
  assert.ok(Array.isArray(j.data), 'data dizi olmalı');
});

test('Ana sayfa ve trend API v2 public yanıtta', async () => {
  await withPage(async (page, errors) => {
    const resp = await goto(page, '/');
    assert.ok(resp && resp.status() === 200, 'ana sayfa HTTP 200');
    assert.deepEqual(errors, [], 'public sayfada JS hatası olmamalı');
  });
  const api = await fetch(BASE_URL + '/api/trends?kaynak=google');
  assert.equal(api.status, 200);
  const j = await api.json();
  assert.equal(j.success, true);
  assert.equal(j.version, 2);
  assert.ok(Array.isArray(j.data));
});
