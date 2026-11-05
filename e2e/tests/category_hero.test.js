'use strict';
/**
 * Kategori gövdesi portre manşet (GÖVDE):
 *
 * - solda 5 slaytlı dikey slider, sağda iki sabit manşet
 * - sol manşet yüksekliği sağdaki iki manşetin toplamına eşit
 * - alt kartlar seçilen sütun sayısıyla (varsayılan 2) dizilir
 * - mobil/tablet/masaüstünde yatay taşma olmaz
 */

const test = require('node:test');
const assert = require('node:assert/strict');
const { BASE_URL, goto, setViewport, withAdminPage, measureOverflow } = require('../lib/harness');

const CATEGORY = '/kategori/ekonomi';
const SID = process.env.TB_SESSION;

/** Testin kullanacağı bilinen portre düzeni (global düzenden bağımsız). */
const PORTRE_BLOCKS = JSON.stringify([{
  id: 'k-liste', type: 'kategori_liste', visible: true,
  opts: {
    zone: 'govde', hero_style: 'portre', hero_count: 5, hero_width: 58,
    hero_ratio: '4/5', list_cols: '2', count: 24,
  },
}]);

/**
 * Kategori sayfasını bilinen bir düzende test ettirir; bitişte özgün düzeni
 * geri yükler (yoksa geçici kaydı siler). Böylece testler canlı/genel
 * kategori düzeninden bağımsız çalışır.
 */
async function withPortreLayout(t, fn) {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdminPage(async (page, errors) => {
    const PAGE_KEY = 'kategori:ekonomi';
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

    await goto(page, '/admin/tasarim?sayfa=kategori&kategori=ekonomi');
    await page.waitForFunction(() => window.__bldInit, { timeout: 15000 });
    const original = await page.evaluate(() => JSON.stringify(window.__bldInit));
    const usingGlobal = await page.evaluate(() => !!window.__bldUsingGlobal);

    try {
      assert.equal((await save(PORTRE_BLOCKS)).ok, true, 'portre test düzeni kaydedilmeli');
      await fn(page);
    } finally {
      if (usingGlobal) {
        assert.equal(await reset(), 200, 'geçici Ekonomi özel düzeni temizlenmeli');
      } else {
        assert.equal((await save(original)).ok, true, 'özgün Ekonomi düzeni geri yüklenmeli');
      }
    }
    assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
  });
}

test('kategori portre manşet: 5 slayt + 2 yan manşet, eşit yükseklik', async (t) => {
  await withPortreLayout(t, async (page) => {
    await setViewport(page, 1440, 1000);
    await goto(page, CATEGORY);

    const hero = await page.evaluate(() => {
      const rect = (el) => {
        if (!el) return null;
        const r = el.getBoundingClientRect();
        return { x: Math.round(r.x), y: Math.round(r.y), w: Math.round(r.width), h: Math.round(r.height) };
      };
      const root = document.querySelector('.cat-hero');
      const slider = document.querySelector('.cat-hero-slider');
      const rail = document.querySelector('.cat-hero-rail');
      const slides = [...document.querySelectorAll('.cat-hero-slider .hero-slide')];
      const dots = [...document.querySelectorAll('.cat-hero-slider .hero-dot')];
      const railItems = [...document.querySelectorAll('.cat-hero-rail > a')];
      const cards = document.querySelector('.cat-cards');
      return {
        root: rect(root),
        slider: rect(slider),
        rail: rect(rail),
        slides: slides.length,
        dots: dots.length,
        railItems: railItems.map(rect),
        cardCols: cards ? getComputedStyle(cards).gridTemplateColumns.split(' ').filter(Boolean).length : 0,
        cardCount: document.querySelectorAll('.cat-cards .cat-card').length,
        ratioClass: root ? root.className : '',
      };
    });

    assert.ok(hero.root, 'kategori gövdesinde .cat-hero bulunmalı');
    assert.equal(hero.slides, 5, 'solda 5 portre slayt olmalı');
    assert.equal(hero.dots, 5, '5 slayt göstergesi olmalı');
    assert.equal(hero.railItems.length, 2, 'sağ tarafta 2 manşet olmalı');
    assert.match(hero.ratioClass, /cat-ratio-4-5/, 'varsayılan oran 4:5 olmalı');
    assert.ok(Math.abs(hero.slider.h - hero.rail.h) <= 1,
      `sol manşet (${hero.slider.h}px) sağ ray (${hero.rail.h}px) yüksekliğine eşit olmalı`);
    assert.ok(hero.slider.h > hero.slider.w, 'portre manşet dikey olmalı');
    assert.ok(hero.railItems.every((r) => r.h > 80), 'sağ manşetler görünür yükseklikte olmalı');
    assert.equal(hero.cardCols, 2, 'alt kartlar varsayılan 2 sütun olmalı');
    assert.ok(hero.cardCount > 0, 'alt kartlar basılmalı');

    const overflow = await measureOverflow(page);
    assert.equal(overflow.overflowing.length, 0, 'masaüstünde yatay taşma olmamalı: ' +
      JSON.stringify(overflow.overflowing));
  });
});

test('kategori portre manşet: slayt göstergesi ve oklar çalışır', async (t) => {
  await withPortreLayout(t, async (page) => {
    await setViewport(page, 1440, 1000);
    await goto(page, CATEGORY);

    const activeHref = () => page.evaluate(() => {
      const slide = document.querySelector('.cat-hero-slider .hero-slide.on');
      return slide ? slide.getAttribute('href') : null;
    });
    const hrefs = await page.$$eval('.cat-hero-slider .hero-slide', (els) => els.map((e) => e.getAttribute('href')));
    assert.ok(hrefs.length >= 3);

    // Sayı göstergesine tıkla → ilgili slayt aktif olur.
    await page.click('.cat-hero-slider .hero-dot[data-idx="2"]');
    await page.waitForFunction(() => {
      const dot = document.querySelector('.cat-hero-slider .hero-dot[data-idx="2"]');
      return dot && dot.classList.contains('on');
    }, { timeout: 3000 });
    assert.equal(await activeHref(), hrefs[2], '3. gösterge 3. slaytı açmalı');

    // Sonraki oku → bir sonraki slayt.
    await page.click('.cat-hero-slider .hero-next');
    await page.waitForFunction((expected) => {
      const slide = document.querySelector('.cat-hero-slider .hero-slide.on');
      return slide && slide.getAttribute('href') !== expected;
    }, { timeout: 3000 }, hrefs[2]);
    const after = await activeHref();
    assert.notEqual(after, hrefs[2], 'sonraki oku slaytı değiştirmeli');
  });
});

test('kategori portre manşet: tablet ve mobilde bozulmaz', async (t) => {
  await withPortreLayout(t, async (page) => {
    for (const [w, h] of [[900, 1200], [768, 1024], [390, 844], [320, 640]]) {
      await setViewport(page, w, h);
      await goto(page, CATEGORY);

      const m = await measureOverflow(page);
      assert.equal(m.overflowing.length, 0, `${w}px yatay taşma: ` + JSON.stringify(m.overflowing));

      const layout = await page.evaluate(() => {
        const rect = (el) => {
          const r = el.getBoundingClientRect();
          return { w: Math.round(r.width), h: Math.round(r.height), y: Math.round(r.y) };
        };
        const slider = document.querySelector('.cat-hero-slider');
        const rail = document.querySelector('.cat-hero-rail');
        const cards = document.querySelector('.cat-cards');
        return {
          slider: slider ? rect(slider) : null,
          rail: rail ? rect(rail) : null,
          cols: cards ? getComputedStyle(cards).gridTemplateColumns.split(' ').filter(Boolean).length : 0,
          viewH: window.innerHeight,
        };
      });
      assert.ok(layout.slider, `${w}px portre slider görünmeli`);
      assert.ok(layout.slider.w <= w, `${w}px slider viewport'tan geniş olmamalı`);
      assert.equal(layout.cols, 2, `${w}px alt kartlar 2 sütun olmalı`);
      if (w < 768) {
        assert.ok(layout.slider.h <= layout.viewH * 0.7 + 2,
          `${w}px portre görsel viewport yüksekliğinin %70'ini aşmamalı: ${layout.slider.h}`);
      } else {
        assert.ok(layout.rail && layout.rail.y >= layout.slider.y, `${w}px sağ ray yerleşmeli`);
        assert.ok(Math.abs(layout.slider.h - layout.rail.h) <= 1,
          `${w}px sol/sağ yükseklik eşit olmalı (${layout.slider.h}/${layout.rail.h})`);
      }
    }
  });
});

test('kategori üçlü manşet: 3 + slider + 2, altında 4 sütunlu kartlar', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdminPage(async (page, errors) => {
    await setViewport(page, 1440, 1000);

    // Test yalnız Ekonomi'ye özel düzende çalışır; genel kategori düzenine
    // dokunmaz. Böylece paralel/ardışık testler portre düzenini bozamaz.
    const PAGE_KEY = 'kategori:ekonomi';
    const SAVE_URL = '/admin/tasarim/kaydet?sayfa=' + encodeURIComponent(PAGE_KEY);
    const RESET_URL = '/admin/tasarim/sifirla?sayfa=' + encodeURIComponent(PAGE_KEY);
    const reset = () => page.evaluate(async (url) => {
      const r = await fetch(url, { method: 'POST', credentials: 'same-origin' });
      return r.status;
    }, BASE_URL + RESET_URL);

    let usingGlobal = false;
    let original = null;
    let originalHero = '';
    try {
      await goto(page, '/admin/tasarim?sayfa=kategori&kategori=ekonomi');
      await page.waitForFunction(() => window.__bldInit, { timeout: 15000 });
      original = await page.evaluate(() => JSON.stringify(window.__bldInit));
      usingGlobal = await page.evaluate(() => !!window.__bldUsingGlobal);
      originalHero = await page.evaluate(() => {
        const b = (window.__bldInit || []).find((x) => x.type === 'kategori_liste');
        return b && b.opts ? b.opts.hero_style || '' : '';
      });
      const save = (blocks) => page.evaluate(async ({ url, raw }) => {
        const r = await fetch(url, {
          method: 'POST',
          credentials: 'same-origin',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ blocks: JSON.parse(raw) }),
        });
        return await r.json();
      }, { url: SAVE_URL, raw: blocks });

      const changed = await page.evaluate((raw) => {
        const arr = JSON.parse(raw);
        const b = arr.find((x) => x.type === 'kategori_liste');
        if (b) {
          b.opts = Object.assign({}, b.opts, {
            hero_style: 'uclu', hero_count: 5, hero_width: 44, list_cols: 4,
          });
          return JSON.stringify(arr);
        }
        // Genel kategori düzeni bağımsız modüllere ayrılmış olabilir. Bu test
        // eski tek bloklu üçlü şablonu doğruladığı için Ekonomi'ye geçici bir
        // kategori_liste düzeni yazılır; bitişte özgün düzen geri gelir.
        return JSON.stringify([{
          id: 'e2e-trio-legacy', type: 'kategori_liste', visible: true,
          opts: {
            zone: 'govde', view: 'klasik', hero_style: 'uclu', hero_count: 5,
            hero_width: 44, hero_ratio: '4/5', list_cols: '4',
            overlay_gradient: 'bottom', overlay_color: '#0b0a11',
            overlay_opacity: '62', overlay_height: 55,
            pager_shape: 'outline', pager_pos: 'center', pager_size: 30,
            akis_limit: 60, akis_per: 12, akis_auto: true,
          },
        }]);
      }, original);
      assert.equal((await save(changed)).ok, true, 'üçlü manşet düzeni kaydedilmeli');

      for (const [w, h] of [[1440, 1200], [900, 1200], [390, 900], [320, 800]]) {
        await setViewport(page, w, h);
        await goto(page, CATEGORY);
        const trio = await page.evaluate(() => {
          const rect = (el) => {
            if (!el) return null;
            const r = el.getBoundingClientRect();
            return { x: Math.round(r.x), y: Math.round(r.y), w: Math.round(r.width), h: Math.round(r.height) };
          };
          const root = document.querySelector('.cat-hero');
          const slider = document.querySelector('.cat-hero-slider');
          const left = document.querySelector('.cat-trio-left');
          const right = document.querySelector('.cat-trio-right');
          const cards = document.querySelector('.cat-cards');
          const lead = left ? left.querySelector(':scope > a') : null;
          const leadTitle = lead ? lead.querySelector('.chr-t') : null;
          const leadMedia = lead ? lead.querySelector('.chr-media') : null;
          const leadRect = rect(lead);
          const titleRect = rect(leadTitle);
          const mediaRect = rect(leadMedia);
          return {
            cls: root ? root.className : '',
            root: rect(root),
            slider: rect(slider),
            left: rect(left),
            right: rect(right),
            leftCount: left ? left.querySelectorAll(':scope > a').length : 0,
            rightCount: right ? right.querySelectorAll(':scope > a').length : 0,
            slides: document.querySelectorAll('.cat-hero-slider .hero-slide').length,
            dots: document.querySelectorAll('.cat-hero-slider .hero-dot').length,
            cols: cards ? getComputedStyle(cards).gridTemplateColumns.split(' ').filter(Boolean).length : 0,
            cardCount: document.querySelectorAll('.cat-cards .cat-card').length,
            leadRect,
            titleRect,
            mediaRect,
            titleDisplay: leadTitle ? getComputedStyle(leadTitle).display : 'yok',
            titleVisible: !!(leadTitle && titleRect && titleRect.w > 20 && titleRect.h > 8),
            overflowX: document.documentElement.scrollWidth - document.documentElement.clientWidth,
          };
        });
        assert.ok(trio.cls.includes('cat-hero-trio'), `${w}px üçlü manşet sınıfı olmalı`);
        assert.equal(trio.leftCount, 3, `${w}px solda 3 manşet olmalı`);
        assert.equal(trio.rightCount, 2, `${w}px sağda 2 manşet olmalı`);
        assert.equal(trio.slides, 5, `${w}px ortada 5 slayt olmalı`);
        assert.equal(trio.dots, 5, `${w}px 5 slayt göstergesi olmalı`);
        assert.equal(trio.overflowX, 0, `${w}px yatay taşma olmamalı`);
        assert.ok(trio.cardCount > 0, `${w}px alt kartlar devam etmeli`);
        assert.ok(trio.titleVisible, `${w}px yan manşet başlığı görünür olmalı`);
        assert.ok(trio.titleRect.y + trio.titleRect.h <= trio.leadRect.y + trio.leadRect.h + 1,
          `${w}px yan manşet başlığı kart içinde kalmalı`);
        assert.ok(Math.abs(trio.mediaRect.w - trio.leadRect.w) <= 1,
          `${w}px yan manşet görseli kart genişliğini doldurmalı (${trio.mediaRect.w}/${trio.leadRect.w})`);
        assert.ok(Math.abs(trio.mediaRect.h - trio.leadRect.h) <= 1,
          `${w}px yan manşet görseli kart yüksekliğini doldurmalı (${trio.mediaRect.h}/${trio.leadRect.h})`);
        if (w >= 1024) {
          assert.equal(trio.cols, 4, `${w}px alt kartlar 4 sütun olmalı`);
          assert.ok(Math.abs(trio.slider.h - trio.left.h) <= 1,
            `${w}px sol ray slider yüksekliğine eşit olmalı (${trio.left.h}/${trio.slider.h})`);
          assert.ok(Math.abs(trio.slider.h - trio.right.h) <= 1,
            `${w}px sağ ray slider yüksekliğine eşit olmalı (${trio.right.h}/${trio.slider.h})`);
        } else if (w >= 768) {
          assert.equal(trio.cols, 3, `${w}px alt kartlar 3 sütun olmalı`);
        } else {
          assert.equal(trio.cols, 2, `${w}px mobilde kartlar 2 sütun olmalı`);
        }
      }
    } finally {
      if (usingGlobal) {
        assert.equal(await reset(), 200, 'geçici Ekonomi özel düzeni temizlenmeli');
      } else if (original) {
        const restored = await page.evaluate(async ({ url, raw }) => {
          const r = await fetch(url, {
            method: 'POST', credentials: 'same-origin',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ blocks: JSON.parse(raw) }),
          });
          return await r.json();
        }, { url: SAVE_URL, raw: original });
        assert.equal(restored.ok, true, 'özgün Ekonomi düzeni geri yüklenmeli');
      }
      await goto(page, CATEGORY);
      const back = await page.evaluate(() => {
        const hero = document.querySelector('.cat-hero');
        return hero ? hero.className : '';
      });
      if (originalHero === 'uclu') {
        assert.ok(back.includes('cat-hero-trio'), 'özgün üçlü düzen geri gelmeli');
      } else {
        assert.ok(back && !back.includes('cat-hero-trio'), 'genel portre düzeni geri gelmeli');
      }
    }
    assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
  });
});
