'use strict';
/**
 * Kategori Sekmeli modülü:
 * - blok ayarlarında kategori çoklu seçimi, sekme stili ve renk alanları
 * - blok bazında seçilen sekmelerin ve stillerin önizlemeye yansıması
 *
 * Test kalıcı düzen değiştirmez; yalnız /admin/tasarim/onizleme ucunu kullanır.
 */

const test = require('node:test');
const assert = require('node:assert/strict');
const { withAdminPage, withPage, goto, setViewport } = require('../lib/harness');

test('kategori sekmeli: kategori seçimi, stil ve renkler', async (t) => {
  if (!process.env.TB_SESSION) return t.skip('TB_SESSION yok (admin testi)');
  await withAdminPage(async (page, errors) => {
    await goto(page, '/admin/tasarim?sayfa=ana');
    await page.waitForSelector('.bp-chip[data-id]', { timeout: 15000 });

    const opened = await page.evaluate(() => {
      const chips = [...document.querySelectorAll('.bp-chip[data-id]')];
      for (const ch of chips) {
        const ty = ch.querySelector('.ty');
        if (ty && ty.textContent.trim() === 'kategori_sekmeli') {
          ch.querySelector('button[data-act="cfg"]').click();
          return true;
        }
      }
      return false;
    });
    assert.ok(opened, 'ana sayfada kategori_sekmeli blok çipi bulunmalı');

    await page.waitForSelector('.cats-multi', { timeout: 5000 });
    const ui = await page.evaluate(() => {
      const box = document.querySelector('.cats-multi');
      const style = box.parentElement.querySelector('select[data-k="tab_style"]');
      return {
        hasPicker: box.querySelectorAll('.cm-pick option').length > 1,
        hasChipsArea: !!box.querySelector('[data-cm-list]'),
        styles: style ? [...style.options].map((o) => o.value).filter(Boolean) : [],
        colors: [...box.parentElement.querySelectorAll('input[type=color]')].map((i) => i.dataset.k),
        radius: !!box.parentElement.querySelector('input[data-k="tab_radius"]'),
      };
    });
    assert.ok(ui.hasPicker, 'kategori seçim listesi olmalı');
    assert.ok(ui.hasChipsArea, 'seçili sekmeler alanı olmalı');
    assert.deepEqual(ui.styles, ['segmented', 'square', 'pill', 'underline', 'boxed', 'minimal', 'tab', 'tabflat']);
    assert.ok(ui.colors.includes('tab_bg'));
    assert.ok(ui.colors.includes('tab_color'));
    assert.ok(ui.colors.includes('tab_active_bg'));
    assert.ok(ui.colors.includes('tab_active_color'));
    assert.ok(ui.radius, 'köşe yarıçapı alanı olmalı');

    const prev = await page.evaluate(async () => {
      const body = {
        blocks: [{
          id: 'ks-e2e', type: 'kategori_sekmeli',
          opts: {
            sekmeler: 'ekonomi,siyaset', count: 4, tab_style: 'square',
            tab_bg: '#111111', tab_active_bg: '#ff0000',
            tab_active_color: '#ffffff', tab_radius: 0,
          },
        }],
      };
      const r = await fetch('/admin/tasarim/onizleme?sayfa=ana', {
        method: 'POST',
        credentials: 'same-origin',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });
      return await r.json();
    });
    assert.equal(prev.ok, true, 'önizleme üretilmeli');
    const html = (prev.items.find((x) => x.type === 'kategori_sekmeli') || {}).html || '';
    assert.match(html, /kat-sekmeli kt-square/, 'köşe düz stil sınıfı basılmalı');
    assert.match(html, /--kt-bg:#111111/);
    assert.match(html, /--kt-active-bg:#ff0000/);
    assert.match(html, /--kt-radius:0px/);
    assert.equal((html.match(/data-tab=/g) || []).length, 2, 'yalnız seçilen 2 sekme basılmalı');
    assert.match(html, /Ekonomi/);
    assert.match(html, /Siyaset/);
    assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
  });
});

test('kategori sekmeli: sağ/sol oklar 5\'erli sonraki gruba geçirir', async () => {
  await withPage(async (page) => {
    await setViewport(page, 1440, 1000);
    await goto(page, '/');

    const panel = '.kat-panel.on';
    const hasNav = await page.$eval(panel, (p) => !!p.querySelector('.kat-nav.kat-next'));
    assert.ok(hasNav, 'seçili sekmede sağ/sol oklar bulunmalı');

    const pageCount = await page.$$eval(panel + ' .kat-page', (els) => els.length);
    assert.ok(pageCount > 1, 'oklarla gezinmek için birden fazla sayfa olmalı');

    const info = () => page.evaluate((sel) => {
      const p = document.querySelector(sel);
      const pages = [...p.querySelectorAll('.kat-page')];
      const active = pages.findIndex((pg) => !pg.hidden);
      const pg = pages[active];
      const lead = pg ? pg.querySelector('.kat-buyuk > a') : null;
      return {
        active,
        items: pg ? pg.querySelectorAll('.kat-buyuk > a, .kat-mini').length : 0,
        dot: [...p.querySelectorAll('.kat-dot')].findIndex((d) => d.classList.contains('on')),
        href: lead ? lead.getAttribute('href') : null,
        count: pages.length,
      };
    }, panel);

    const first = await info();
    assert.equal(first.active, 0, 'ilk sayfa açık başlamalı');
    assert.equal(first.dot, 0, 'ilk nokta seçili olmalı');
    assert.equal(first.items, 5, 'bir sayfada 1 büyük + 4 küçük = 5 haber olmalı');

    await page.click(panel + ' .kat-next');
    await page.waitForFunction((sel) => {
      const pages = [...document.querySelectorAll(sel + ' .kat-page')];
      return pages[1] && !pages[1].hidden;
    }, { timeout: 3000 }, panel);

    const second = await info();
    assert.equal(second.active, 1, 'sağ ok ikinci sayfayı açmalı');
    assert.equal(second.dot, 1, 'ikinci nokta seçili olmalı');
    assert.equal(second.items, 5, 'ikinci sayfa da 5 haberlük olmalı');
    assert.notEqual(second.href, first.href, 'sonraki grup farklı haberler göstermeli');

    await page.click(panel + ' .kat-prev');
    await page.waitForFunction((sel) => {
      const pages = [...document.querySelectorAll(sel + ' .kat-page')];
      return pages[0] && !pages[0].hidden;
    }, { timeout: 3000 }, panel);
    assert.equal((await info()).href, first.href, 'sol ok önceki gruba dönmeli');

    // İlk sayfada sol ok son sayfaya döner (döngü).
    await page.click(panel + ' .kat-prev');
    await page.waitForFunction((sel, n) => {
      const pages = [...document.querySelectorAll(sel + ' .kat-page')];
      return pages[n - 1] && !pages[n - 1].hidden;
    }, { timeout: 3000 }, panel, first.count);
    assert.equal((await info()).active, first.count - 1, 'sol ok son sayfaya dönmeli');

    // Mobilde de oklar görünür alanda kalmalı ve çalışmalı.
    await setViewport(page, 390, 844);
    await goto(page, '/');
    const mobile = await page.$eval(panel, (p) => {
      const r = (sel) => {
        const el = p.querySelector(sel);
        if (!el) return null;
        const b = el.getBoundingClientRect();
        return { left: Math.round(b.left), right: Math.round(b.right), w: Math.round(b.width) };
      };
      return { prev: r('.kat-nav.kat-prev'), next: r('.kat-nav.kat-next'), vw: window.innerWidth };
    });
    assert.ok(mobile.prev && mobile.next, 'mobilde oklar bulunmalı');
    assert.ok(mobile.prev.left >= 0 && mobile.prev.right <= mobile.vw, 'sol ok görünür alanda olmalı');
    assert.ok(mobile.next.left >= 0 && mobile.next.right <= mobile.vw, 'sağ ok görünür alanda olmalı');
    await page.click(panel + ' .kat-next');
    await page.waitForFunction((sel) => {
      const pages = [...document.querySelectorAll(sel + ' .kat-page')];
      return pages[1] && !pages[1].hidden;
    }, { timeout: 3000 }, panel);
  });
});

