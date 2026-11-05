'use strict';
/**
 * Kategori Haber Listesi — manşet slider'ı görünüm ayarları (Üçlü/Portre).
 *
 * Slider modülündeki seçeneklerin kategori gövdesi manşetine taşındığını
 * doğrular:
 * - başlık arkası tasarım stili (geçiş yönü, renk, saydamlık)
 * - başlık arkası yüksekliği ve arka plan bulanıklığı
 * - sayı çerçeve stili (kare/yuvarlak/çerçeveli), konumu, boyutu, rengi
 * - başlık gölgesi ve başlık yazı rengi
 *
 * Test yalnız Ekonomi'ye özel düzende çalışır ve özgün düzeni geri yükler.
 */

const test = require('node:test');
const assert = require('node:assert/strict');
const { BASE_URL, goto, setViewport, withAdminPage, ensureLegacyCategory } = require('../lib/harness');

const CATEGORY = '/kategori/ekonomi';
const SID = process.env.TB_SESSION;
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

test('kategori manşet sliderı: başlık arkası, sayı çerçevesi ve gölge ayarları', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdminPage(async (page, errors) => {
    await setViewport(page, 1440, 1200);

    const PAGE_KEY = 'kategori:ekonomi';
    const BUILDER = '/admin/tasarim?sayfa=kategori&kategori=ekonomi';
    const SAVE_URL = '/admin/tasarim/kaydet?sayfa=' + encodeURIComponent(PAGE_KEY);
    const RESET_URL = '/admin/tasarim/sifirla?sayfa=' + encodeURIComponent(PAGE_KEY);
    const reset = () => page.evaluate(async (url) => {
      const r = await fetch(url, { method: 'POST', credentials: 'same-origin' });
      return r.status;
    }, BASE_URL + RESET_URL);
    const save = (raw) => page.evaluate(async ({ url, raw }) => {
      const r = await fetch(url, {
        method: 'POST',
        credentials: 'same-origin',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ blocks: JSON.parse(raw) }),
      });
      return await r.json();
    }, { url: SAVE_URL, raw });

    let usingGlobal = false;
    let original = null;
    try {
      await goto(page, '/admin/tasarim?sayfa=kategori&kategori=ekonomi');
      await page.waitForFunction(() => window.__bldInit, { timeout: 15000 });
      original = await page.evaluate(() => JSON.stringify(window.__bldInit));
      usingGlobal = await page.evaluate(() => !!window.__bldUsingGlobal);
      const base = await ensureLegacyCategory(page, {
        pageKey: PAGE_KEY, builderUrl: BUILDER,
        opts: { hero_style: 'portre', hero_count: 5, hero_width: 58, list_cols: '2' },
      });

      /* Tasarımcıda alanların varlığı ve seçenekleri. */
      const opened = await page.evaluate(() => {
        const ch = [...document.querySelectorAll('.bp-chip[data-id]')].find((c) => {
          const ty = c.querySelector('.ty');
          return ty && ty.textContent.trim() === 'kategori_liste';
        });
        if (!ch) return false;
        ch.querySelector('button[data-act="cfg"]').click();
        return true;
      });
      assert.ok(opened, 'kategori_liste blok çipi bulunmalı');
      await page.waitForSelector('select[data-k="overlay_gradient"]', { timeout: 5000 });

      const ui = await page.evaluate(() => {
        const q = (k) => document.querySelector('select[data-k="' + k + '"], input[data-k="' + k + '"]');
        const opts = (k) => {
          const el = document.querySelector('select[data-k="' + k + '"]');
          return el ? [...el.options].map((o) => o.value).filter(Boolean) : [];
        };
        return {
          gradient: opts('overlay_gradient'),
          opacityPreset: opts('overlay_opacity_preset'),
          pagerShape: opts('pager_shape'),
          pagerPos: opts('pager_pos'),
          hasColor: !!q('overlay_color'),
          hasHeight: !!q('overlay_height'),
          hasBlur: !!q('overlay_blur'),
          hasSize: !!q('pager_size'),
          hasPagerColor: !!q('pager_color'),
          hasShadow: !!q('hero_shadow'),
          hasTitleColor: !!q('title_color'),
          hasBoxBg: !!q('card_box_bg'),
          hasBoxDarkBg: !!q('card_box_dark_bg'),
          hasCardShadow: !!q('card_shadow'),
        };
      });
      assert.deepEqual(ui.gradient, ['none', 'solid', 'bottom', 'left', 'right', 'diagonal'],
        'başlık arkası tasarım stili seçenekleri olmalı');
      assert.ok(ui.opacityPreset.includes('custom'), 'saydamlık ön ayarları olmalı');
      assert.deepEqual(ui.pagerShape, ['square', 'round', 'outline'], 'sayı çerçeve stilleri olmalı');
      assert.deepEqual(ui.pagerPos, ['left', 'center', 'right'], 'sayı konumu seçenekleri olmalı');
      assert.ok(ui.hasColor && ui.hasHeight && ui.hasBlur, 'renk / yükseklik / bulanıklık alanları olmalı');
      assert.ok(ui.hasSize && ui.hasPagerColor, 'sayı boyutu ve rengi alanları olmalı');
      assert.ok(ui.hasShadow, 'başlık gölgesi anahtarı olmalı');
      assert.ok(ui.hasTitleColor, 'başlık yazı rengi alanı olmalı');
      assert.ok(ui.hasBoxBg && ui.hasBoxDarkBg, 'kart gövdesi açık/koyu tema alanları olmalı');
      assert.ok(ui.hasCardShadow, 'kart gölgesi anahtarı olmalı');

      /* Ayarları kaydet: üçlü manşet + tüm slider görünüm seçenekleri. */
      const raw = await page.evaluate((src) => {
        const arr = JSON.parse(src);
        const b = arr.find((x) => x.type === 'kategori_liste');
        if (!b) throw new Error('kategori_liste bloğu bulunamadı');
        b.opts = Object.assign({}, b.opts, {
          hero_style: 'uclu', hero_count: 5, hero_width: 44, list_cols: '4',
          overlay_gradient: 'bottom', overlay_color: '#112233',
          overlay_opacity_preset: '45', overlay_height: 55, overlay_blur: 6,
          pager_shape: 'outline', pager_pos: 'center', pager_size: 32,
          pager_color: '#00ff00', hero_shadow: true, title_color: '#ffcc00',
        });
        return JSON.stringify(arr);
      }, base.blocks);
      assert.equal((await save(raw)).ok, true, 'manşet slider görünümü kaydedilmeli');

      await goto(page, CATEGORY);
      const got = await page.evaluate(() => {
        const hero = document.querySelector('.cat-hero');
        const overlay = hero.querySelector('.hero-slide.on .hero-overlay') ||
          hero.querySelector('.hero-overlay');
        const title = overlay.querySelector('.hero-title');
        const dots = hero.querySelector('.hero-dots');
        const on = hero.querySelector('.hero-dot.on');
        const slide = hero.querySelector('.hero-slide');
        const ov = getComputedStyle(overlay);
        const ts = getComputedStyle(title);
        const ds = on ? getComputedStyle(on) : null;
        const drect = on ? on.getBoundingClientRect() : null;
        return {
          inline: overlay.getAttribute('style') || '',
          ovH: overlay.getBoundingClientRect().height,
          slideH: slide.getBoundingClientRect().height,
          backdrop: ov.backdropFilter || ov.webkitBackdropFilter || '',
          titleCls: title.className,
          textShadow: ts.textShadow,
          titleColor: ts.color,
          dotCls: dots ? dots.className : '',
          dotStyle: dots ? dots.getAttribute('style') || '' : '',
          dotW: drect ? Math.round(drect.width) : 0,
          dotRadius: ds ? ds.borderTopLeftRadius : '',
          dotBg: ds ? ds.backgroundColor : '',
          overflowX: document.documentElement.scrollWidth - document.documentElement.clientWidth,
        };
      });

      assert.match(got.inline, /linear-gradient\(to top/, 'aşağıdan yukarı geçiş uygulanmalı');
      assert.match(got.inline, /rgba\(17, 34, 51, 0\.45\)|rgba\(17,34,51,0\.45\)/, 'renk + saydamlık uygulanmalı');
      assert.match(got.inline, /min-height:\s*55%/, 'başlık arkası en az %55 yüksekliğinde olmalı');
      assert.match(got.inline, /height:\s*auto/, 'başlık arkası içerikle büyümeli');
      assert.match(got.inline, /blur\(6px\)/, 'arka plan bulanıklığı uygulanmalı');
      assert.ok(got.backdrop.includes('blur(6px)'), 'bulanıklık hesaplanmış stile yansımalı');
      assert.ok(got.ovH / got.slideH >= 0.54,
        `başlık arkası slider yüksekliğinin en az %55'i olmalı (${Math.round(got.ovH)}/${Math.round(got.slideH)})`);

      assert.match(got.titleCls, /has-shadow/, 'başlık gölgesi sınıfı eklenmeli');
      assert.notEqual(got.textShadow, 'none', 'başlıkta gerçek metin gölgesi olmalı');
      assert.equal(got.titleColor, 'rgb(255, 204, 0)', 'başlık yazı rengi uygulanmalı');

      assert.match(got.dotCls, /hero-dots--shape-outline/, 'çerçeveli sayı biçimi uygulanmalı');
      assert.match(got.dotCls, /hero-dots--pos-center/, 'sayı konumu orta olmalı');
      assert.match(got.dotStyle, /--hero-dot-size:\s*32px/, 'sayı boyutu göstergede olmalı');
      assert.match(got.dotStyle, /--pager-color:\s*#00ff00/, 'gösterge rengi göstergede olmalı');
      assert.equal(got.dotW, 32, 'sayı kutusu 32px olmalı');
      assert.equal(got.dotRadius, '6px', 'çerçeveli biçimde köşeler kare kalmalı');
      assert.equal(got.dotBg, 'rgb(0, 255, 0)', 'aktif sayı seçilen renkte olmalı');
      assert.equal(got.overflowX, 0, 'yatay taşma olmamalı');
      assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
    } finally {
      if (usingGlobal) {
        assert.equal(await reset(), 200, 'geçici Ekonomi özel düzeni temizlenmeli');
      } else if (original) {
        const restored = await save(original);
        assert.equal(restored.ok, true, 'özgün Ekonomi düzeni geri yüklenmeli');
      }
    }
  });
});

test('tasarımcı: manşet slider ayarları arayüzden kaydedilir', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdminPage(async (page, errors) => {
    await setViewport(page, 1440, 1200);
    const PAGE_KEY = 'kategori:ekonomi';
    const BUILDER = '/admin/tasarim?sayfa=kategori&kategori=ekonomi';
    const SAVE_URL = '/admin/tasarim/kaydet?sayfa=' + encodeURIComponent(PAGE_KEY);
    const RESET_URL = '/admin/tasarim/sifirla?sayfa=' + encodeURIComponent(PAGE_KEY);
    const reset = () => page.evaluate(async (url) => {
      const r = await fetch(url, { method: 'POST', credentials: 'same-origin' });
      return r.status;
    }, BASE_URL + RESET_URL);
    const save = (raw) => page.evaluate(async ({ url, raw }) => {
      const r = await fetch(url, {
        method: 'POST', credentials: 'same-origin',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ blocks: JSON.parse(raw) }),
      });
      return await r.json();
    }, { url: SAVE_URL, raw });

    let usingGlobal = false;
    let original = null;
    try {
      await goto(page, '/admin/tasarim?sayfa=kategori&kategori=ekonomi');
      await page.waitForFunction(() => window.__bldInit, { timeout: 15000 });
      original = await page.evaluate(() => JSON.stringify(window.__bldInit));
      usingGlobal = await page.evaluate(() => !!window.__bldUsingGlobal);
      await ensureLegacyCategory(page, {
        pageKey: PAGE_KEY, builderUrl: BUILDER,
        opts: { hero_style: 'portre', hero_count: 5, hero_width: 58, list_cols: '2' },
      });

      const opened = await page.evaluate(() => {
        const ch = [...document.querySelectorAll('.bp-chip[data-id]')].find((c) => {
          const ty = c.querySelector('.ty');
          return ty && ty.textContent.trim() === 'kategori_liste';
        });
        if (!ch) return false;
        ch.querySelector('button[data-act="cfg"]').click();
        return true;
      });
      assert.ok(opened, 'kategori_liste blok çipi bulunmalı');
      await page.waitForSelector('select[data-k="overlay_gradient"]', { timeout: 5000 });

      await page.select('select[data-k="overlay_gradient"]', 'solid');
      await page.select('select[data-k="overlay_opacity_preset"]', '40');
      await page.evaluate(() => {
        const cb = document.querySelector('input[data-k="overlay_color__on"]');
        if (cb && !cb.checked) cb.click();
        const el = document.querySelector('input[type=color][data-k="overlay_color"]');
        if (el) {
          el.value = '#223344';
          el.dispatchEvent(new Event('input', { bubbles: true }));
          el.dispatchEvent(new Event('change', { bubbles: true }));
        }
      });
      await page.select('select[data-k="pager_shape"]', 'outline');
      await page.select('select[data-k="pager_pos"]', 'center');
      await page.evaluate(() => {
        const el = document.querySelector('input[data-k="pager_size"]');
        if (el) {
          el.value = '28';
          el.dispatchEvent(new Event('input', { bubbles: true }));
          el.dispatchEvent(new Event('change', { bubbles: true }));
        }
        const cb = document.querySelector('input[data-k="hero_shadow"]');
        if (cb && !cb.checked) cb.click();
      });
      await sleep(1600);

      await goto(page, '/admin/tasarim?sayfa=kategori&kategori=ekonomi');
      await page.waitForFunction(() => window.__bldInit, { timeout: 15000 });
      const opts = await page.evaluate(() => {
        const b = (window.__bldInit || []).find((x) => x.type === 'kategori_liste');
        return b ? (b.opts || {}) : null;
      });
      assert.ok(opts, 'kategori_liste bloğu korunmalı');
      assert.equal(opts.overlay_gradient, 'solid', 'başlık arkası tasarımı kaydedilmeli');
      assert.equal(String(opts.overlay_opacity), '40', 'saydamlık kaydedilmeli');
      assert.equal(opts.overlay_color, '#223344', 'başlık arkası rengi kaydedilmeli');
      assert.equal(opts.pager_shape, 'outline', 'sayı çerçevesi kaydedilmeli');
      assert.equal(opts.pager_pos, 'center', 'sayı konumu kaydedilmeli');
      assert.equal(Number(opts.pager_size), 28, 'sayı boyutu kaydedilmeli');
      assert.equal(opts.hero_shadow, true, 'başlık gölgesi kaydedilmeli');
      assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
    } finally {
      if (usingGlobal) {
        assert.equal(await reset(), 200, 'geçici Ekonomi özel düzeni temizlenmeli');
      } else if (original) {
        const restored = await save(original);
        assert.equal(restored.ok, true, 'özgün Ekonomi düzeni geri yüklenmeli');
      }
    }
  });
});
