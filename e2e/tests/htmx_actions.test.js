'use strict';
/**
 * HTMX panel aksiyonları — reload yok + editör yeniden bağlanma.
 *
 * Veri yazmadan/idempotent save'lerle çalışır:
 *  - Menü/Front footer/Sekme kaydet (mevcut değer aynen yazılır)
 *  - Raporlar "Yenile" (hx-get)
 * Hepsi admin gerektirir; TB_SESSION yoksa atlanır.
 */

const test = require('node:test');
const assert = require('node:assert/strict');
const { goto, setViewport, withAdminPage } = require('../lib/harness');

const SID = process.env.TB_SESSION;

async function saveTwice(page, saveSel, flashNeedle, panelSel, childSel) {
  await page.evaluate(() => { window.__tbNoReload = 1; });
  await page.click(saveSel);
  await page.waitForFunction((txt) => {
    const f = document.querySelector('.flash');
    return f && f.textContent.includes(txt);
  }, { timeout: 15000 }, flashNeedle);
  // İkinci kayıt: swap sonrası inline betiğin yeniden bağlandığını doğrular.
  await page.click(saveSel);
  await new Promise((res) => setTimeout(res, 800));
  return page.evaluate((sel, ps) => ({
    noReload: window.__tbNoReload === 1,
    panel: !!document.querySelector(ps),
    alive: !!document.querySelector(sel),
    children: document.querySelector(sel) ? document.querySelector(sel).children.length : -1,
  }), childSel, panelSel);
}

test('menü editörü HTMX ile iki kez kaydeder, reload yok', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok');
  await withAdminPage(async (page) => {
    await setViewport(page, 1400);
    await goto(page, '/admin/menu');
    if (page.url().includes('/admin/giris')) return t.skip('giriş gerekli');
    const r = await saveTwice(page, '#menuSave', 'Menü kaydedildi', '#menu-panel', '#menuItems');
    assert.equal(r.noReload, true, 'reload olmamalı');
    assert.equal(r.panel, true, 'panel korunmalı');
    assert.ok(r.children > 0, 'menü öğeleri korunmalı');
  });
});

test('footer editörü HTMX ile kaydeder, reload yok', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok');
  await withAdminPage(async (page) => {
    await setViewport(page, 1400);
    await goto(page, '/admin/footer');
    if (page.url().includes('/admin/giris')) return t.skip('giriş gerekli');
    const r = await saveTwice(page, '#fSave', 'Footer kaydedildi', '#footer-panel', '#footCols');
    assert.equal(r.noReload, true, 'reload olmamalı');
    assert.equal(r.panel, true, 'panel korunmalı');
    assert.ok(r.children > 0, 'footer sütunları korunmalı');
  });
});

test('sekme kategorileri HTMX ile kaydeder, reload yok', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok');
  await withAdminPage(async (page) => {
    await setViewport(page, 1400);
    await goto(page, '/admin/tab-kategorileri');
    if (page.url().includes('/admin/giris')) return t.skip('giriş gerekli');
    const r = await saveTwice(page, '#tcSave', 'Sekme kategorileri kaydedildi', '#tabcats-panel', '#tcItems');
    assert.equal(r.noReload, true, 'reload olmamalı');
    assert.equal(r.panel, true, 'panel korunmalı');
    assert.ok(r.children >= 0, 'sekme listesi korunmalı');
  });
});

test('raporlar "Yenile" hx-get ile paneli tazeler, reload yok', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok');
  await withAdminPage(async (page) => {
    await setViewport(page, 1400);
    await goto(page, '/admin/raporlar');
    if (page.url().includes('/admin/giris')) return t.skip('giriş gerekli');
    await page.evaluate(() => { window.__tbNoReload = 1; });
    await page.evaluate(() => {
      window.__swapped = 0;
      window.addEventListener('htmx:afterSwap', function () { window.__swapped++; });
    });
    await page.click('a[hx-get="/admin/raporlar"]');
    await page.waitForFunction(() => window.__swapped > 0, { timeout: 15000 });
    const r = await page.evaluate(() => ({
      noReload: window.__tbNoReload === 1,
      panel: !!document.querySelector('#reports-panel'),
      cards: document.querySelectorAll('#reports-panel .card').length,
    }));
    assert.equal(r.noReload, true, 'reload olmamalı');
    assert.equal(r.panel, true, 'panel korunmalı');
    assert.ok(r.cards >= 2, 'kartlar yenilenmeli');
  });
});

test('bot: kaynak eklenir sonra HTMX ile silinir, reload yok', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok');
  await withAdminPage(async (page) => {
    await setViewport(page, 1400);
    await goto(page, '/admin/bot');
    if (page.url().includes('/admin/giris')) return t.skip('giriş gerekli');
    const ad = 'HTMX Kaynak ' + Date.now();
    await page.evaluate(() => { window.__tbNoReload = 1; });
    await page.type('form[action="/admin/bot/ekle"] input[name="name"]', ad);
    await page.type('form[action="/admin/bot/ekle"] input[name="url"]', 'https://example.com/rss.xml');
    await page.click('form[action="/admin/bot/ekle"] button[type="submit"]');
    await page.waitForFunction((txt) => {
      const f = document.querySelector('.flash');
      return f && f.textContent.includes('eklendi');
    }, { timeout: 15000 }, ad);
    // Silme URL'sini HTMX ile çağır (native confirm yerine htmx.ajax; CI'da kararlı).
    const post = await page.evaluate((txt) => {
      const els = [...document.querySelectorAll('#bot-panel a[hx-post^="/admin/bot/"][hx-post$="/sil"]')];
      const a = els.find((x) => x.closest('tr') && x.closest('tr').textContent.includes(txt));
      return a ? a.getAttribute('hx-post') : null;
    }, ad);
    assert.ok(post, 'sil düğmesi bulunmalı');
    await page.evaluate((u) => window.htmx.ajax('POST', u, { target: '#bot-panel', swap: 'outerHTML' }), post);
    await page.waitForFunction(() => {
      const f = document.querySelector('.flash');
      return f && f.textContent.includes('Silindi');
    }, { timeout: 15000 });
    const r = await page.evaluate((txt) => ({
      noReload: window.__tbNoReload === 1,
      panel: !!document.querySelector('#bot-panel'),
      kalan: [...document.querySelectorAll('#bot-panel tr')].some((tr) => tr.textContent.includes(txt)),
    }), ad);
    assert.equal(r.noReload, true, 'reload olmamalı');
    assert.equal(r.panel, true, 'panel korunmalı');
    assert.equal(r.kalan, false, 'kaynak listede kalmamalı');
  });
});

test('siteler: site eklenir sonra HTMX ile silinir, reload yok', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok');
  await withAdminPage(async (page) => {
    await setViewport(page, 1400);
    await goto(page, '/admin/siteler');
    if (page.url().includes('/admin/giris')) return t.skip('giriş gerekli');
    const ad = 'HTMX Site ' + Date.now();
    await page.evaluate(() => { window.__tbNoReload = 1; });
    await page.type('form[action="/admin/siteler/ekle"] input[name="name"]', ad);
    await page.type('form[action="/admin/siteler/ekle"] input[name="domain"]', 'htmx-' + Date.now() + '.test');
    await page.click('form[action="/admin/siteler/ekle"] button[type="submit"]');
    await page.waitForFunction((txt) => {
      const f = document.querySelector('.flash');
      return f && f.textContent.includes('eklendi');
    }, { timeout: 15000 }, ad);
    const post = await page.evaluate((txt) => {
      const els = [...document.querySelectorAll('#siteler-panel a[hx-post*="/sil"]')];
      const a = els.find((x) => x.closest('tr') && x.closest('tr').textContent.includes(txt));
      return a ? a.getAttribute('hx-post') : null;
    }, ad);
    assert.ok(post, 'sil düğmesi bulunmalı');
    await page.evaluate((u) => window.htmx.ajax('POST', u, { target: '#siteler-panel', swap: 'outerHTML' }), post);
    await page.waitForFunction(() => {
      const f = document.querySelector('.flash');
      return f && f.textContent.includes('Silindi');
    }, { timeout: 15000 });
    const r = await page.evaluate((txt) => ({
      noReload: window.__tbNoReload === 1,
      panel: !!document.querySelector('#siteler-panel'),
      kalan: [...document.querySelectorAll('#siteler-panel tr')].some((tr) => tr.textContent.includes(txt)),
    }), ad);
    assert.equal(r.noReload, true, 'reload olmamalı');
    assert.equal(r.panel, true, 'panel korunmalı');
    assert.equal(r.kalan, false, 'site listede kalmamalı');
  });
});

test('manset: şablon eklenir sonra HTMX ile silinir, reload yok', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok');
  await withAdminPage(async (page) => {
    await setViewport(page, 1500, 1000);
    await goto(page, '/admin/manset-editor');
    if (page.url().includes('/admin/giris')) return t.skip('giriş gerekli');
    const ad = 'HTMX Manşet ' + Date.now();
    await page.evaluate(() => { window.__tbNoReload = 1; });
    await page.click('.me-tabs button[data-tab="tpl"]');
    const presetCount = await page.$$eval('#me-presets button', (els) => els.length);
    assert.ok(presetCount >= 6, 'en az 6 numaralı hazır şablon olmalı: ' + presetCount);
    await page.click('#me-presets button:nth-child(5)');
    const presetColor = await page.$eval('#me-overlay-color', (el) => el.value);
    assert.equal(presetColor, '#0b1e3a', 'hazır şablon karartma rengini uygulamalı');
    await page.type('#me-name', ad);
    await page.click('#me-save');
    await page.waitForFunction((txt) => {
      const f = document.querySelector('.flash');
      return f && f.textContent.includes('kaydedildi');
    }, { timeout: 20000 }, ad);
    const id = await page.evaluate((txt) => {
      const b = [...document.querySelectorAll('button[data-tplid-del]')].find((x) =>
        x.closest('span').textContent.includes(txt));
      return b ? b.getAttribute('data-tplid-del') : null;
    }, ad);
    assert.ok(id, 'şablon sil düğmesi bulunmalı');
    await page.evaluate((i) => window.htmx.ajax('POST', '/admin/manset-editor/sil',
      { target: '#manset-panel', swap: 'outerHTML', values: { id: i } }), id);
    await page.waitForFunction(() => {
      const f = document.querySelector('.flash');
      return f && f.textContent.includes('Şablon silindi');
    }, { timeout: 15000 });
    const r = await page.evaluate((txt) => ({
      noReload: window.__tbNoReload === 1,
      panel: !!document.querySelector('#manset-panel'),
      kalan: [...document.querySelectorAll('button[data-tplid-del]')].some((b) =>
        b.closest('span').textContent.includes(txt)),
    }), ad);
    assert.equal(r.noReload, true, 'reload olmamalı');
    assert.equal(r.panel, true, 'panel korunmalı');
    assert.equal(r.kalan, false, 'şablon listede kalmamalı');
  });
});

test('sayfa tasarımcısı: manuel Kaydet HTMX, tuval korunur, reload yok', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok');
  await withAdminPage(async (page) => {
    await setViewport(page, 1500, 1000);
    await goto(page, '/admin/tasarim?sayfa=home');
    if (page.url().includes('/admin/giris')) return t.skip('giriş gerekli');
    await page.evaluate(() => {
      window.__tbNoReload = 1;
      try { localStorage.setItem('tb_builder_sess', 'abc'); } catch (e) {}
    });
    await page.click('#bldSave');
    await page.waitForFunction(() => {
      const s = document.querySelector('#bld-status .flash');
      return s && /Kaydedildi/.test(s.textContent);
    }, { timeout: 15000 });
    const r = await page.evaluate(() => ({
      noReload: window.__tbNoReload === 1,
      panel: !!document.querySelector('#bld-status .flash'),
      chips: document.querySelectorAll('.bp-chip[data-id], .mod-chip').length,
      sess: (() => { try { return localStorage.getItem('tb_builder_sess'); } catch (e) { return null; } })(),
    }));
    assert.equal(r.noReload, true, 'reload olmamalı');
    assert.ok(r.chips > 0, 'modül çipleri korunmalı');
    assert.equal(r.sess, 'abc', 'editör oturumu korunmalı (script yeniden çalışmamalı)');
  });
});

test('AI sohbet: HTMX mesaj gönderimi panel günceller, sonra silinir', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok');
  await withAdminPage(async (page) => {
    await setViewport(page, 1400, 1000);
    await goto(page, '/admin/ai-sohbet');
    if (page.url().includes('/admin/giris')) return t.skip('giriş gerekli');
    await page.evaluate(() => { window.__tbNoReload = 1; });
    // Yeni sohbet (HTMX)
    await page.click('a[href="/admin/ai-sohbet/yeni"]');
    await page.waitForFunction(() => {
      const f = document.querySelector('.flash');
      return f && /Yeni sohbet açıldı/.test(f.textContent);
    }, { timeout: 15000 });
    const conv = await page.evaluate(() => document.querySelector('#ai-form').getAttribute('data-conv') || '');
    assert.ok(conv && conv !== '', 'yeni sohbet olmalı');
    await page.type('#ai-input', 'HTMX sohbet test mesajı');
    await page.click('#ai-send');
    // NOT (Faz 210): Asistan balonundaki 🤖 emoji KALDIRILDI (Faz 161 emoji
    // temizliği); yerine "AI" etiketi geliyor. Bu yüzden kontrol emojiye
    // değil, iki taraflı iletinin varlığına bakıyor.
    await page.waitForFunction((txt) => {
      const m = document.querySelector('#ai-msgs');
      return m && m.children.length >= 2 && m.textContent.includes(txt) &&
        /AI/.test(m.textContent);
    }, { timeout: 30000 }, 'HTMX sohbet test mesajı');
    const r = await page.evaluate(() => ({
      noReload: window.__tbNoReload === 1,
      msgs: document.querySelector('#ai-msgs').children.length,
      conv: document.querySelector('#ai-form').getAttribute('data-conv') || '',
    }));
    assert.equal(r.noReload, true, 'reload olmamalı');
    assert.ok(r.msgs >= 2, 'kullanıcı + asistan iletisi görünmeli');
    // Temizlik — oluşan konuşmayı HTMX ile sil
    await page.evaluate((id) => window.htmx.ajax('POST', '/admin/ai-sohbet/' + id + '/sil',
      { target: '#ai-sohbet-panel', swap: 'outerHTML' }), conv);
    await page.waitForFunction(() => {
      const f = document.querySelector('.flash');
      return f && f.textContent.includes('Sohbet silindi');
    }, { timeout: 15000 });
    const kalan = await page.evaluate((id) =>
      [...document.querySelectorAll('#ai-sohbet-panel a[href*="konu="]')].some((a) =>
        a.getAttribute('href').includes(id)), conv);
    assert.equal(kalan, false, 'silinen sohbet listede kalmamalı');
  });
});
