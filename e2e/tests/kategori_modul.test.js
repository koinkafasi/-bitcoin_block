'use strict';
/**
 * Kategori gövdesi — bağımsız modüller.
 *
 * Doğrulananlar:
 * - slider / seçimli manşet rayları / 4'lü ızgara / akış ayrı modüller olarak
 *   tek manşet satırında birleşir; haberler tekrar etmez.
 * - uzun başlıkta manşet bandı (overlay) büyür, başlık kırpılmaz.
 * - tek başına slider tam genişlikte 16:9 akar (boş yükseklik bırakmaz).
 * - tüm hero satır yerleşimleri (lcr/lc/cr/lr/c/l/r) doğru ızgarayı kurar.
 * - tasarımcı paletinde modüller görünür, "Modüllere ayır" çalışır,
 *   kaydet/yeniden yükle aralıkları korur ve önizleme dolu HTML üretir.
 *
 * Testler yalnız Ekonomi'ye özel düzende çalışır; bitişte özgün düzeni
 * geri yükler (yoksa geçici kaydı siler).
 */

const test = require('node:test');
const assert = require('node:assert/strict');
const { BASE_URL, goto, setViewport, withAdminPage } = require('../lib/harness');

const CATEGORY = '/kategori/ekonomi';
const PAGE_KEY = 'kategori:ekonomi';
const SID = process.env.TB_SESSION;

const SAVE_URL = '/admin/tasarim/kaydet?sayfa=' + encodeURIComponent(PAGE_KEY);
const RESET_URL = '/admin/tasarim/sifirla?sayfa=' + encodeURIComponent(PAGE_KEY);
// Sayfa şablonu (sütun düzeni) kategori genelinde saklanır. Tek başına
// slider tam genişlik ölçümü sağ sütun açıkken anlamsız olur; bu yüzden test
// kendi sütun ayarını kurar ve bitişte özgün değerine döndürür.
const LAYOUT_URL = '/admin/tasarim/yerlesim?sayfa=' + encodeURIComponent('kategori');

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

/** Ekonomi düzenini geçici olarak kurar; bitişte özgün hâline döndürür. */
async function withCategoryLayout(page, fn) {
  let usingGlobal = false;
  let original = null;
  const save = (blocks) => page.evaluate(async ({ url, blocks }) => {
    const r = await fetch(url, {
      method: 'POST',
      credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ blocks }),
    });
    return await r.json();
  }, { url: SAVE_URL, blocks });
  const reset = () => page.evaluate(async (url) => {
    const r = await fetch(url, { method: 'POST', credentials: 'same-origin' });
    return r.status;
  }, BASE_URL + RESET_URL);

  /* Sütun şablonunu belirle: sol kolon tam genişlik, sağ sütun yok. */
  const applyLayout = (cfg) => page.evaluate(async ({ url, cfg }) => {
    const r = await fetch(url, {
      method: 'POST', credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(cfg),
    });
    return await r.json();
  }, { url: BASE_URL + LAYOUT_URL, cfg });
  const readLayout = () => page.evaluate(async (url) => {
    const r = await fetch(url, { credentials: 'same-origin' });
    return await r.json();
  }, BASE_URL + LAYOUT_URL);
  const resetLayout = () => page.evaluate(async (url) => {
    await fetch(url, {
      method: 'POST', credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ reset: true }),
    });
  }, BASE_URL + LAYOUT_URL);

  let layoutBackup = null;
  try {
    await goto(page, '/admin/tasarim?sayfa=kategori&kategori=ekonomi');
    await page.waitForFunction(() => window.__bldInit, { timeout: 15000 });
    layoutBackup = await readLayout();
    await applyLayout({ sidebar: false, contentWidth: 1280 });
    original = await page.evaluate(() => JSON.stringify(window.__bldInit));
    usingGlobal = await page.evaluate(() => !!window.__bldUsingGlobal);
    return await fn({ save, reset });
  } finally {
    if (usingGlobal) await reset();
    else if (original) await save(JSON.parse(original));
    /* Sütun şablonunu özgün hâline döndür: önce sıfırla (yama birikmesin),
       sonra kayıtlı özgün config varsa yaz. */
    await resetLayout();
    const cfg = layoutBackup && layoutBackup.config ? layoutBackup.config : {};
    if (layoutBackup && layoutBackup.hasCustom && Object.keys(cfg).length) {
      await applyLayout(cfg);
    }
  }
}

function sliderBlock(extra = {}) {
  return {
    id: 'e2e-slider', type: 'kategori_slider', visible: true,
    opts: Object.assign({
      zone: 'govde', count: 10, row_layout: 'hero-center', item_offset: 0,
      hero_width: 44, hero_ratio: '1/1', overlay_color: '#0c1020',
      overlay_opacity_preset: '80', overlay_height: 27, overlay_blur: 30,
      pager_shape: 'round', pager_pos: 'center', pager_size: 30,
    }, extra),
  };
}

const RAIL_LEFT = {
  id: 'e2e-left', type: 'kategori_secimli', visible: true,
  opts: { zone: 'govde', count: 3, row_layout: 'hero-left', item_offset: 10, rail_style: 'overlay' },
};
const RAIL_RIGHT = {
  id: 'e2e-right', type: 'kategori_secimli', visible: true,
  opts: { zone: 'govde', count: 2, row_layout: 'hero-right', item_offset: 13, rail_style: 'overlay' },
};
const GRID = {
  id: 'e2e-grid', type: 'kategori_4lu', visible: true,
  opts: { zone: 'govde', count: 4, item_offset: 15, list_cols: '4' },
};
const FLOW = {
  id: 'e2e-flow', type: 'kategori_akis', visible: true,
  opts: { zone: 'govde', item_offset: 19, akis_limit: '40', akis_per: '12', akis_auto: true },
};

const MODULES = [sliderBlock(), RAIL_LEFT, RAIL_RIGHT, GRID, FLOW];

/** Manşet satırındaki geometri ve kart sayıları. */
async function heroInfo(page) {
  return page.evaluate(() => {
    const rect = (el) => {
      if (!el) return null;
      const r = el.getBoundingClientRect();
      return { x: Math.round(r.x), y: Math.round(r.y), w: Math.round(r.width), h: Math.round(r.height) };
    };
    const hero = document.querySelector('.cat-hero');
    const slider = document.querySelector('.cat-hero-slider');
    const left = document.querySelector('.cat-slot-left');
    const right = document.querySelector('.cat-slot-right');
    const grid = document.querySelector('.cat-cards');
    const flow = document.querySelector('[data-ak-flow]');
    return {
      hero: rect(hero),
      heroCls: hero ? hero.className : '',
      heroCols: hero ? getComputedStyle(hero).gridTemplateColumns.split(' ').filter(Boolean).length : 0,
      slider: rect(slider),
      left: rect(left),
      right: rect(right),
      leftCount: left ? left.querySelectorAll(':scope > a').length : 0,
      rightCount: right ? right.querySelectorAll(':scope > a').length : 0,
      grid: rect(grid),
      gridCols: grid ? getComputedStyle(grid).gridTemplateColumns.split(' ').filter(Boolean).length : 0,
      gridCount: grid ? grid.querySelectorAll('.cat-card').length : 0,
      flowBas: flow ? flow.getAttribute('data-ak-bas') : null,
      flowSinir: flow ? flow.getAttribute('data-ak-sinir') : null,
      flowCount: document.querySelectorAll('.ak-flow .ak-item').length,
      hrefs: [...document.querySelectorAll(
        '.cat-hero-slider .hero-slide, .cat-hero-rail > a, .cat-cards .cat-card, .ak-flow .ak-item a'
      )].map((a) => a.getAttribute('href')),
      overflowX: document.documentElement.scrollWidth - document.documentElement.clientWidth,
    };
  });
}

/** Slayt başına bant yüksekliği, başlık konumu ve satır içi stil. */
async function overlayInfo(page) {
  return page.evaluate(() => {
    const slider = document.querySelector('.cat-hero-slider');
    if (!slider) return null;
    const sr = slider.getBoundingClientRect();
    const slides = [...slider.querySelectorAll('.hero-slide')];
    return slides.map((s, i) => {
      const ov = s.querySelector('.hero-overlay');
      const t = s.querySelector('.hero-title');
      const spot = s.querySelector('.hero-spot');
      const or = ov.getBoundingClientRect();
      const tr = t.getBoundingClientRect();
      const spr = spot ? spot.getBoundingClientRect() : null;
      return {
        i,
        len: t.textContent.trim().length,
        h: +or.height.toFixed(1),
        minH: +(sr.height * 0.27).toFixed(1),
        style: ov.getAttribute('style') || '',
        titleTopGap: +(tr.top - or.top).toFixed(1),
        titleBottomGap: +(or.bottom - tr.bottom).toFixed(1),
        spotBottomGap: spr ? +(or.bottom - spr.bottom).toFixed(1) : null,
      };
    });
  });
}

test('kategori modülleri: 3+2 ray, 4\'lü ızgara ve akış tekrar etmeden', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdminPage(async (page) => {
    await setViewport(page, 1440, 1200);
    await withCategoryLayout(page, async ({ save }) => {
      const saved = await save(MODULES);
      assert.equal(saved.ok, true, 'modül düzeni kaydedilmeli');

      await goto(page, CATEGORY);
      const h = await heroInfo(page);

      assert.ok(h.heroCls.includes('cat-hero'), 'manşet satırı .cat-hero olmalı');
      assert.ok(h.heroCls.includes('cat-row-lcr'), 'slider + iki ray tek satırda birleşmeli');
      assert.ok(h.heroCls.includes('cat-ratio-1-1'), '1:1 oranı satıra taşınmalı');
      assert.equal(h.heroCols, 3, 'masaüstünde üç sütun olmalı');
      assert.equal(h.leftCount, 3, 'solda 3 seçimli manşet olmalı');
      assert.equal(h.rightCount, 2, 'sağda 2 seçimli manşet olmalı');
      assert.equal(h.gridCols, 4, '4\'lü ızgara 4 sütun olmalı');
      assert.equal(h.gridCount, 4, '4\'lü ızgarada 4 kart olmalı');
      assert.equal(h.flowBas, '24', 'akış imleci ilk sayfanın sonundan sürmeli');
      assert.equal(h.flowSinir, '40', 'akış sınırı modülden gelmeli');
      assert.equal(h.flowCount, 5, 'ilk sayfada 19..23 arası 5 akış kartı olmalı');

      assert.equal(h.hrefs.length, 24, 'ilk sayfada 24 haber bağlantısı olmalı');
      assert.equal(new Set(h.hrefs).size, h.hrefs.length,
        'slider + raylar + ızgara + akış aynı haberi tekrar etmemeli');
      assert.equal(h.overflowX, 0, 'masaüstünde yatay taşma olmamalı');

      // Raylar slider ile aynı satırda başlamalı (grid-row:1).
      assert.equal(h.left.y, h.slider.y, 'sol ray slider ile aynı hizada olmalı');
      assert.equal(h.right.y, h.slider.y, 'sağ ray slider ile aynı hizada olmalı');

      /* Uzun başlık bandı büyütür; başlık bandın dışına taşmaz. */
      const ov = await overlayInfo(page);
      assert.ok(Array.isArray(ov) && ov.length === 10, 'slider 10 slayt basmalı');
      const heights = ov.map((o) => o.h);
      const minH = ov[0].minH;
      for (const o of ov) {
        assert.ok(o.style.includes('height:auto'), `slayt ${o.i + 1}: bant yüksekliği auto olmalı`);
        assert.ok(o.style.includes('min-height:27%'), `slayt ${o.i + 1}: %27 taban korunmalı`);
        assert.ok(!o.style.includes('overflow:hidden'), `slayt ${o.i + 1}: bant kırpmamalı`);
        assert.ok(o.h >= minH - 1, `slayt ${o.i + 1}: bant en az %27 yüksekliğinde olmalı (${o.h} < ${minH})`);
        assert.ok(o.titleTopGap >= -1, `slayt ${o.i + 1}: başlık bandın üstünden taşmamalı (${o.titleTopGap})`);
        assert.ok(o.titleBottomGap >= -1, `slayt ${o.i + 1}: başlık bandın altından taşmamalı (${o.titleBottomGap})`);
        if (o.spotBottomGap !== null) {
          assert.ok(o.spotBottomGap >= -1, `slayt ${o.i + 1}: spot bandın altından taşmamalı`);
        }
      }
      assert.ok(Math.max(...heights) - Math.min(...heights) >= 5,
        `uzun başlıkta bant büyümeli (yükseklikler: ${heights.join(', ')})`);

      /* Mobil: tek sütun, 2 sütunlu ızgara, taşma yok. */
      await setViewport(page, 390, 900);
      await goto(page, CATEGORY);
      const m = await heroInfo(page);
      assert.equal(m.heroCols, 1, 'mobilde manşet tek sütun olmalı');
      assert.equal(m.gridCols, 2, 'mobilde 4\'lü ızgara 2 sütun olmalı');
      assert.equal(m.leftCount, 3, 'mobilde de 3 sol manşet olmalı');
      assert.equal(m.overflowX, 0, 'mobilde yatay taşma olmamalı');
    });
  });
});

test('tek başına kategori slider: tam genişlik 16:9, boş yükseklik yok', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdminPage(async (page) => {
    await setViewport(page, 1440, 1200);
    await withCategoryLayout(page, async ({ save }) => {
      const solo = {
        id: 'e2e-solo', type: 'kategori_slider', visible: true,
        opts: {
          zone: 'govde', count: 5, hero_width: 44, overlay_height: 27,
          overlay_color: '#0c1020', overlay_opacity_preset: '80',
        },
      };
      assert.equal((await save([solo])).ok, true, 'tek slider kaydedilmeli');

      await goto(page, CATEGORY);
      const info = await page.evaluate(() => {
        const rect = (el) => {
          if (!el) return null;
          const r = el.getBoundingClientRect();
          return { x: Math.round(r.x), y: Math.round(r.y), w: Math.round(r.width), h: Math.round(r.height) };
        };
        const slider = document.querySelector('.cat-hero-slider');
        const slide = document.querySelector('.cat-hero-slider .hero-slide.on');
        const img = slide ? slide.querySelector('img') : null;
        const ov = slide ? slide.querySelector('.hero-overlay') : null;
        return {
          hero: !!document.querySelector('.cat-hero'),
          cls: slider ? slider.className : '',
          slider: rect(slider),
          slide: rect(slide),
          img: rect(img),
          overlay: rect(ov),
          slidePos: slide ? getComputedStyle(slide).position : null,
          ratio: slider ? +(slider.getBoundingClientRect().width / slider.getBoundingClientRect().height).toFixed(2) : null,
          overflowX: document.documentElement.scrollWidth - document.documentElement.clientWidth,
        };
      });

      assert.equal(info.hero, false, 'satır yokken .cat-hero kurulmamalı');
      assert.ok(info.cls.includes('cat-slider-solo'), 'tek slider solo sınıfını almalı');
      assert.ok(info.slider && info.slider.w > 1000, 'slider tam genişlikte olmalı');
      assert.ok(Math.abs(info.ratio - 16 / 9) < 0.05, `varsayılan oran 16:9 olmalı (${info.ratio})`);
      assert.ok(info.slider.h < 900, `slider gereksiz yere uzamamalı (${info.slider.h}px)`);
      assert.equal(info.slidePos, 'absolute', 'solo slaytlar bölümü doldurmalı');
      assert.ok(info.img && info.img.h >= info.slider.h - 2, 'görsel slider yüksekliğini doldurmalı');
      assert.ok(info.overlay && Math.abs(info.overlay.y + info.overlay.h - (info.slider.y + info.slider.h)) <= 2,
        'bant slider tabanına oturmalı');
      assert.equal(info.overflowX, 0, 'yatay taşma olmamalı');
    });
  });
});

test('hero satır yerleşimleri: lcr/lc/cr/lr/c/l/r', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  const variants = [
    { name: 'lcr', blocks: [sliderBlock({ hero_ratio: '16/9' }), RAIL_LEFT, RAIL_RIGHT], cls: 'cat-row-lcr', cols: 3 },
    { name: 'lc', blocks: [sliderBlock({ hero_ratio: '16/9' }), RAIL_LEFT], cls: 'cat-row-lc', cols: 2 },
    { name: 'cr', blocks: [sliderBlock({ hero_ratio: '16/9' }), RAIL_RIGHT], cls: 'cat-row-cr', cols: 2 },
    { name: 'lr', blocks: [RAIL_LEFT, RAIL_RIGHT], cls: 'cat-row-lr', cols: 2 },
    { name: 'c', blocks: [sliderBlock({ hero_ratio: '16/9' })], cls: 'cat-row-c', cols: 1 },
    { name: 'l', blocks: [RAIL_LEFT], cls: 'cat-row-l', cols: 1 },
    { name: 'r', blocks: [RAIL_RIGHT], cls: 'cat-row-r', cols: 1 },
  ];
  await withAdminPage(async (page) => {
    await setViewport(page, 1440, 1200);
    await withCategoryLayout(page, async ({ save }) => {
      for (const v of variants) {
        assert.equal((await save(v.blocks)).ok, true, `${v.name} düzeni kaydedilmeli`);
        await goto(page, CATEGORY);
        const info = await page.evaluate(() => {
          const hero = document.querySelector('.cat-hero');
          if (!hero) return { hero: false };
          const slots = [...hero.children].map((el) => {
            const r = el.getBoundingClientRect();
            return { cls: el.className, y: Math.round(r.y), h: Math.round(r.height) };
          });
          return {
            hero: true,
            cls: hero.className,
            cols: getComputedStyle(hero).gridTemplateColumns.split(' ').filter(Boolean).length,
            slots,
            overflowX: document.documentElement.scrollWidth - document.documentElement.clientWidth,
          };
        });
        assert.equal(info.hero, true, `${v.name}: manşet satırı kurulmalı`);
        assert.ok(info.cls.includes(v.cls), `${v.name}: ${v.cls} sınıfı beklenir (${info.cls})`);
        assert.equal(info.cols, v.cols, `${v.name}: ${v.cols} sütun beklenir`);
        for (const s of info.slots) {
          assert.equal(s.y, info.slots[0].y, `${v.name}: tüm yuvalar aynı satırda olmalı`);
        }
        assert.equal(info.overflowX, 0, `${v.name}: yatay taşma olmamalı`);
      }
    });
  });
});

test('tasarımcı: modül paleti, ayırma, aralık kaydı ve dolu önizleme', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdminPage(async (page) => {
    await setViewport(page, 1440, 1200);
    await withCategoryLayout(page, async ({ save }) => {
      const legacy = [{
        id: 'e2e-legacy', type: 'kategori_liste', visible: true,
        opts: {
          zone: 'govde', view: 'akis', hero_style: 'uclu', hero_count: 5,
          hero_width: 44, hero_ratio: '4/5', list_cols: '4',
          overlay_gradient: 'bottom', overlay_color: '#112233',
          overlay_opacity_preset: '45', overlay_height: 55, overlay_blur: 6,
          pager_shape: 'outline', pager_pos: 'center', pager_size: 32,
          pager_color: '#00ff00', hero_shadow: true, title_color: '#ffcc00',
          card_style: 'bloomberg-cizgi',
          card_box_bg: '#112233', card_box_dark_bg: '#445566',
          card_shadow: false,
          akis_limit: '40', akis_per: '12', akis_auto: true,
        },
      }];
      assert.equal((await save(legacy)).ok, true, 'eski düzen kaydedilmeli');

      await goto(page, '/admin/tasarim?sayfa=kategori&kategori=ekonomi');
      await page.waitForFunction(() => window.__bldInit, { timeout: 15000 });

      const ui = await page.evaluate(() => ({
        palette: [...document.querySelectorAll('.bp-pal .bld-add')].map((b) => b.dataset.type),
        chips: [...document.querySelectorAll('.bp-chip[data-id] .ty')].map((e) => e.textContent.trim()),
        split: !!document.getElementById('catSplitBtn'),
        tag: (document.getElementById('catViewTag') || {}).textContent,
      }));
      for (const ty of ['kategori_slider', 'kategori_secimli', 'kategori_4lu', 'kategori_akis']) {
        assert.ok(ui.palette.includes(ty), `palette ${ty} olmalı`);
      }
      assert.deepEqual(ui.chips, ['kategori_liste'], 'başlangıçta tek eski liste olmalı');
      assert.equal(ui.split, true, '"Modüllere ayır" düğmesi görünmeli');

      await page.click('#catSplitBtn');
      await page.waitForFunction(
        () => document.querySelectorAll('.bp-chip[data-id] .ty').length >= 5,
        { timeout: 5000 }
      );
      const afterSplit = await page.evaluate(() => ({
        types: [...document.querySelectorAll('.bp-chip[data-id] .ty')].map((e) => e.textContent.trim()),
        tag: (document.getElementById('catViewTag') || {}).textContent,
        split: !!document.getElementById('catSplitBtn'),
      }));
      assert.deepEqual(afterSplit.types, [
        'kategori_slider', 'kategori_secimli', 'kategori_secimli', 'kategori_4lu', 'kategori_akis',
      ], 'eski blok beş bağımsız modüle ayrılmalı');
      assert.equal(afterSplit.split, false, 'ayırma sonrası düğme kalkmalı');
      assert.ok(String(afterSplit.tag).includes('Bağımsız'), 'şablon etiketi bağımsız modüllere dönmeli');

      // Otomatik kayıt tamamlanınca yeniden yükle.
      await sleep(2200);
      await goto(page, '/admin/tasarim?sayfa=kategori&kategori=ekonomi');
      await page.waitForFunction(() => window.__bldInit, { timeout: 15000 });
      const saved = await page.evaluate(() => window.__bldInit);
      assert.equal(saved.length, 5, 'kaydedilen düzen 5 modül içermeli');
      assert.deepEqual(saved.map((b) => b.opts && b.opts.item_offset), [0, 5, 8, 10, 14],
        'üçlü şablonda aralıklar slider 0, sol 5, sağ 8, 4\'lü 10, akış 14 olmalı');
      // Görünüm anahtarları yalnız ilgili modüle taşınmalı.
      assert.equal(saved[0].opts.akis_limit, undefined, 'slider akış ayarı taşımamalı');
      assert.equal(saved[0].opts.card_style, undefined, 'slider kart stili taşımamalı');
      assert.equal(saved[1].opts.overlay_color, undefined, 'seçimli manşet slider bandı ayarı taşımamalı');
      assert.equal(saved[3].opts.card_style, 'bloomberg-cizgi', '4\'lü ızgara kart stilini almalı');
      assert.equal(saved[3].opts.card_box_bg, '#112233', '4\'lü ızgara açık tema kart gövdesini almalı');
      assert.equal(saved[3].opts.card_box_dark_bg, '#445566', '4\'lü ızgara koyu tema kart gövdesini almalı');
      assert.equal(saved[3].opts.card_shadow, false, '4\'lü ızgara gölge kapatmayı almalı');
      assert.equal(saved[4].opts.card_style, 'bloomberg-cizgi', 'akış kart stilini almalı');
      assert.equal(saved[4].opts.card_box_bg, undefined, 'akış kart gövdesi ayarı taşımamalı');
      assert.equal(saved[4].opts.card_box_dark_bg, undefined, 'akış kart gövdesi koyu ayarı taşımamalı');
      assert.equal(saved[4].opts.card_shadow, undefined, 'akış gölge ayarı taşımamalı');
      assert.equal(saved[4].opts.akis_limit, '40', 'akış sınırı korunmalı');

      // Slider ayarları: satır yerleşimi, aralık, genişlik, oran, bant ve pager.
      const opened = await page.evaluate(() => {
        const ch = [...document.querySelectorAll('.bp-chip[data-id]')]
          .find((c) => (c.querySelector('.ty') || {}).textContent.trim() === 'kategori_slider');
        if (!ch) return false;
        ch.querySelector('button[data-act="cfg"]').click();
        return true;
      });
      assert.equal(opened, true, 'slider modülünün ayarları açılabilmeli');
      await page.waitForSelector('select[data-k="row_layout"]', { timeout: 5000 });
      const fields = await page.evaluate(() => {
        const opts = (k) => {
          const el = document.querySelector('select[data-k="' + k + '"]');
          return el ? [...el.options].map((o) => o.value) : null;
        };
        return {
          rowLayout: opts('row_layout'),
          ratio: opts('hero_ratio'),
          pager: opts('pager_shape'),
          hasOffset: !!document.querySelector('input[data-k="item_offset"]'),
          hasWidth: !!document.querySelector('input[data-k="hero_width"]'),
          hasOverlay: !!document.querySelector('select[data-k="overlay_gradient"]'),
        };
      });
      assert.ok(fields.rowLayout && fields.rowLayout.includes('hero-center'), 'satır yerleşimi seçenekleri olmalı');
      assert.ok(fields.rowLayout.includes('hero-left') && fields.rowLayout.includes('hero-right'));
      assert.equal(fields.hasOffset, true, 'haber aralığı alanı olmalı');
      assert.equal(fields.hasWidth, true, 'manşet genişliği alanı olmalı');
      assert.ok(fields.ratio && fields.ratio.includes('16/9'), 'oran seçenekleri olmalı');
      assert.ok(fields.hasOverlay, 'bant tasarımı alanı olmalı');
      assert.ok(fields.pager && fields.pager.length >= 3, 'sayı çerçevesi seçenekleri olmalı');

      // Önizleme: her modül dolu HTML üretmeli; kategori verisiyle beslenmeli.
      const prev = await page.evaluate(async ({ pageKey, blocks }) => {
        const r = await fetch('/admin/tasarim/onizleme?sayfa=' + encodeURIComponent(pageKey), {
          method: 'POST',
          credentials: 'same-origin',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ blocks }),
        });
        return await r.json();
      }, { pageKey: PAGE_KEY, blocks: saved });
      assert.equal(prev.ok, true, 'önizleme ucuna ulaşılmalı');
      assert.equal(prev.items.length, 5, 'her modül için önizleme dönmeli');
      const marker = {
        kategori_slider: 'hero-slide',
        kategori_secimli: 'cat-hero-rail',
        kategori_4lu: 'cat-cards',
        kategori_akis: 'ak-flow',
      };
      for (const it of prev.items) {
        const m = marker[it.type];
        assert.ok(m, `bilinmeyen modül tipi: ${it.type}`);
        assert.ok(String(it.html).includes(m), `${it.type} önizlemesi ${m} içermeli`);
        assert.ok(!String(it.html).includes('bp-prev-empty'), `${it.type} önizlemesi yer tutucu olmamalı`);
      }
      const sliderPrev = prev.items.find((i) => i.type === 'kategori_slider');
      assert.ok(sliderPrev.html.includes('cat-hero'), 'slider önizlemesi manşet satırına sarılmalı');
      assert.ok(sliderPrev.html.includes('cat-slot-center'), 'slider önizlemesi orta yuvada olmalı');
      assert.ok(/Enerji haberi|Akaryakıt|ekonomi/i.test(sliderPrev.html), 'önizleme gerçek haber başlığı içermeli');
    });
  });
});

test('4\'lü manşet: kart gövdesi rengi ve gölge kapatma', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdminPage(async (page) => {
    await setViewport(page, 1440, 1200);
    await withCategoryLayout(page, async ({ save }) => {
      const blocks = [{
        id: 'e2e-grid-style', type: 'kategori_4lu', visible: true,
        opts: {
          zone: 'govde', count: 4, item_offset: 0, list_cols: '4',
          card_style: 'bloomberg-cizgi',
          card_title_bg: '#ff0000', card_title_fg: '#ffffff',
          card_dark_bg: '#00ff00', card_dark_fg: '#000000',
          card_box_bg: '#112233', card_box_dark_bg: '#445566',
          card_shadow: false,
        },
      }];
      assert.equal((await save(blocks)).ok, true, 'kart gövdesi/gölge ayarları kaydedilmeli');

      await goto(page, '/admin/tasarim?sayfa=kategori&kategori=ekonomi');
      await page.waitForFunction(() => window.__bldInit, { timeout: 15000 });
      const opened = await page.evaluate(() => {
        const ch = [...document.querySelectorAll('.bp-chip[data-id]')]
          .find((c) => (c.querySelector('.ty') || {}).textContent.trim() === 'kategori_4lu');
        if (!ch) return false;
        ch.querySelector('button[data-act="cfg"]').click();
        return true;
      });
      assert.ok(opened, '4\'lü manşet ayarları açılmalı');
      await page.waitForSelector('input[data-k="card_box_bg"]', { timeout: 5000 });
      const ui = await page.evaluate(() => {
        const val = (k) => {
          const el = document.querySelector('input[data-k="' + k + '"]');
          return el ? el.value : null;
        };
        const shadow = document.querySelector('input[data-k="card_shadow"]');
        return {
          boxBg: val('card_box_bg'),
          boxDarkBg: val('card_box_dark_bg'),
          hasShadow: !!shadow,
          shadowChecked: shadow ? shadow.checked : null,
        };
      });
      assert.equal(ui.boxBg, '#112233', 'açık tema kart gövdesi alanı dolu olmalı');
      assert.equal(ui.boxDarkBg, '#445566', 'koyu tema kart gövdesi alanı dolu olmalı');
      assert.equal(ui.hasShadow, true, 'kart gölgesi anahtarı olmalı');
      assert.equal(ui.shadowChecked, false, 'kapatılan gölge anahtarı işaretsiz gelmeli');

      await goto(page, CATEGORY);
      const read = () => page.evaluate(() => {
        const card = document.querySelector('.cat-cards .cat-card');
        const pad = card ? card.querySelector('.card-pad') : null;
        const cs = card ? getComputedStyle(card) : null;
        const ps = pad ? getComputedStyle(pad) : null;
        return {
          cardBg: cs ? cs.backgroundColor : null,
          cardShadow: cs ? cs.boxShadow : null,
          padBg: ps ? ps.backgroundColor : null,
        };
      });
      const light = await read();
      assert.equal(light.cardBg, 'rgb(17, 34, 51)', 'açık temada kart gövdesi rengi uygulanmalı');
      assert.equal(light.cardShadow, 'none', 'kart gölgesi kapatılmalı');
      assert.equal(light.padBg, 'rgb(255, 0, 0)', 'başlık bandı kendi renginde kalmalı');

      await page.evaluate(() => document.documentElement.setAttribute('data-theme', 'dark'));
      const dark = await read();
      assert.equal(dark.cardBg, 'rgb(68, 85, 102)', 'koyu temada kart gövdesi rengi uygulanmalı');
      assert.equal(dark.cardShadow, 'none', 'koyu temada da gölge kapalı olmalı');
      assert.equal(dark.padBg, 'rgb(0, 255, 0)', 'koyu temada başlık bandı kendi renginde kalmalı');
    });
  });
});

test('kategori slider: gölge yok seçiliyken başlık arkası yüksekliği düzenlenebilir', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdminPage(async (page) => {
    await setViewport(page, 1440, 1200);
    await withCategoryLayout(page, async ({ save }) => {
      const blocks = [{
        id: 'e2e-slider-none', type: 'kategori_slider', visible: true,
        opts: {
          zone: 'govde', count: 3, overlay_gradient: 'none',
          overlay_height: 70, overlay_color: '#123456',
          overlay_opacity_preset: '80',
        },
      }];
      assert.equal((await save(blocks)).ok, true, 'gölgesiz slider kaydedilmeli');

      await goto(page, '/admin/tasarim?sayfa=kategori&kategori=ekonomi');
      await page.waitForFunction(() => window.__bldInit, { timeout: 15000 });
      const opened = await page.evaluate(() => {
        const ch = [...document.querySelectorAll('.bp-chip[data-id]')]
          .find((c) => (c.querySelector('.ty') || {}).textContent.trim() === 'kategori_slider');
        if (!ch) return false;
        ch.querySelector('button[data-act="cfg"]').click();
        return true;
      });
      assert.ok(opened, 'slider ayarları açılmalı');
      await page.waitForSelector('input[data-k="overlay_height"]', { timeout: 5000 });
      const ui = await page.evaluate(() => {
        const wrap = document.querySelector('[data-ov-field="overlay_height"]');
        const inp = document.querySelector('input[data-k="overlay_height"]');
        const g = document.querySelector('select[data-k="overlay_gradient"]');
        return {
          gradient: g ? g.value : null,
          value: inp ? inp.value : null,
          visible: wrap ? getComputedStyle(wrap).display !== 'none' : false,
          note: (document.querySelector('[data-ov-note]') || {}).textContent || '',
        };
      });
      assert.equal(ui.gradient, 'none', 'gölge yok seçili olmalı');
      assert.equal(ui.value, '70', 'yükseklik değeri korunmalı');
      assert.equal(ui.visible, true, 'gölge yok seçiliyken yükseklik alanı görünür olmalı');
      assert.match(ui.note, /Yükseklik/i, 'not yüksekliğin etkisini açıklamalı');

      await goto(page, CATEGORY);
      const rendered = await page.evaluate(() => {
        const ov = document.querySelector('.cat-hero-slider .hero-overlay');
        return { style: ov ? ov.getAttribute('style') || '' : '' };
      });
      assert.match(rendered.style, /min-height:\s*70%/, 'yükseklik katman kapalıyken de uygulanmalı');
      assert.match(rendered.style, /background:\s*transparent/, 'renk/geçiş katmanı çizilmemeli');
    });
  });
});
