'use strict';
/**
 * Kategori Haber Listesi — "Akış (Enerji)" şablonu (coinhit.net/enerji modeli):
 *
 * - ilk 4 haber 4 sütunlu kart ızgarasında (mobilde 2 sütun)
 * - sonrası 3 sütun × 4 kartlık "enerji" bloklarıyla akar
 *   (tip5 kare görsel + görselin üstüne taşan başlık, tip2/4 yan görselli,
 *    tip1 yalnız metin)
 * - "Daha fazla göster" düğmesi + kaydırınca otomatik yükleme
 * - akış haber sınırı manuel ayarlanabilir; sınır dolunca akış biter
 *
 * Test yalnız Ekonomi'ye özel düzende çalışır; genel kategori düzenine
 * dokunmaz ve bitişte özgün düzeni geri yükler (category_hero ile aynı kalıp).
 */

const test = require('node:test');
const assert = require('node:assert/strict');
const { BASE_URL, goto, setViewport, withPage, withAdminPage, measureOverflow, ensureLegacyCategory } = require('../lib/harness');

const CATEGORY = '/kategori/ekonomi';
const SID = process.env.TB_SESSION;
const LIMIT = 40; // akış haber sınırı (ilk 4 + 36 akış kartı)

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

async function flowInfo(page) {
  return page.evaluate(() => {
    const flow = document.querySelector('[data-ak-flow]');
    const first = document.querySelector('.ak-first');
    const end = document.querySelector('[data-ak-end]');
    return {
      firstCards: first ? first.querySelectorAll('.cat-card').length : 0,
      firstCols: first ? getComputedStyle(first).gridTemplateColumns.split(' ').filter(Boolean).length : 0,
      grids: document.querySelectorAll('.ak-flow .ak-grid').length,
      items: document.querySelectorAll('.ak-flow .ak-item').length,
      type5: document.querySelectorAll('.ak-flow .ak-type5').length,
      type2: document.querySelectorAll('.ak-flow .ak-type2').length,
      type4: document.querySelectorAll('.ak-flow .ak-type4').length,
      type1: document.querySelectorAll('.ak-flow .ak-type1').length,
      bas: flow ? flow.getAttribute('data-ak-bas') : null,
      sinir: flow ? flow.getAttribute('data-ak-sinir') : null,
      oto: flow ? flow.getAttribute('data-ak-oto') : null,
      moreVisible: !!document.querySelector('[data-ak-more]:not(.is-done)'),
      endVisible: !!(end && !end.hidden),
      pager: !!document.querySelector('nav.pager'),
      overflowX: document.documentElement.scrollWidth - document.documentElement.clientWidth,
    };
  });
}

test('kategori akış şablonu: 4 kart + enerji blokları, sınırlı akış', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdminPage(async (page, errors) => {
    await setViewport(page, 1440, 1200);

    const PAGE_KEY = 'kategori:ekonomi';
    const SAVE_URL = '/admin/tasarim/kaydet?sayfa=' + encodeURIComponent(PAGE_KEY);
    const RESET_URL = '/admin/tasarim/sifirla?sayfa=' + encodeURIComponent(PAGE_KEY);
    const reset = () => page.evaluate(async (url) => {
      const r = await fetch(url, { method: 'POST', credentials: 'same-origin' });
      return r.status;
    }, BASE_URL + RESET_URL);

    let usingGlobal = false;
    let original = null;

    const save = (raw) => page.evaluate(async ({ url, raw }) => {
      const r = await fetch(url, {
        method: 'POST',
        credentials: 'same-origin',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ blocks: JSON.parse(raw) }),
      });
      return await r.json();
    }, { url: SAVE_URL, raw });

    try {
      await goto(page, '/admin/tasarim?sayfa=kategori&kategori=ekonomi');
      await page.waitForFunction(() => window.__bldInit, { timeout: 15000 });
      original = await page.evaluate(() => JSON.stringify(window.__bldInit));
      usingGlobal = await page.evaluate(() => !!window.__bldUsingGlobal);

      const base = await ensureLegacyCategory(page, {
        pageKey: PAGE_KEY,
        opts: {
          view: 'akis', hero_style: 'uclu', hero_count: 5, hero_width: 44,
          list_cols: '4', akis_limit: String(LIMIT), akis_per: '12',
          akis_auto: true, akis_hero: true,
        },
      });
      const raw = await page.evaluate(({ src, limit }) => {
        const arr = JSON.parse(src);
        const b = arr.find((x) => x.type === 'kategori_liste');
        if (!b) throw new Error('kategori_liste bloğu bulunamadı');
        b.opts = Object.assign({}, b.opts, {
          view: 'akis', hero_style: 'uclu', hero_count: 5, hero_width: 44,
          akis_limit: String(limit), akis_per: '12', akis_auto: true, akis_hero: true,
        });
        return JSON.stringify(arr);
      }, { src: base.blocks, limit: LIMIT });
      assert.equal((await save(raw)).ok, true, 'akış şablonu kaydedilmeli');

      await setViewport(page, 1440, 1200);
      await goto(page, CATEGORY);
      const first = await flowInfo(page);

      /* Akışın üstünde üçlü manşet alanı (solda 3, ortada slider, sağda 2). */
      const hero = await page.evaluate(() => {
        const root = document.querySelector('.cat-hero');
        const left = document.querySelector('.cat-trio-left');
        const right = document.querySelector('.cat-trio-right');
        return {
          cls: root ? root.className : '',
          leftCount: left ? left.querySelectorAll(':scope > a').length : 0,
          rightCount: right ? right.querySelectorAll(':scope > a').length : 0,
          slides: document.querySelectorAll('.cat-hero-slider .hero-slide').length,
        };
      });
      assert.ok(hero.cls.includes('cat-hero-trio'), 'akışın üstünde üçlü manşet alanı olmalı');
      assert.equal(hero.leftCount, 3, 'solda 3 manşet olmalı');
      assert.equal(hero.rightCount, 2, 'sağda 2 manşet olmalı');
      assert.equal(hero.slides, 5, 'ortada 5 slaytlı slider olmalı');

      assert.equal(first.firstCards, 4, 'manşetten sonra 4 kartlık satır olmalı');
      assert.equal(first.firstCols, 4, 'masaüstünde ilk 4 kart 4 sütun olmalı');
      assert.ok(first.grids >= 1, 'enerji bloğu basılmalı');
      // Akış imleci (`data-ak-bas`) sayfada GÖSTERİLEN haber sayısı olmalı:
      // manşet (3 sol + 5 slider + 2 sağ) + 4 kart + enerji kartları.
      // Sabit '24' yazmak içerik sayısına bağlıydı ve kategoriye bir haber
      // eklendiğinde testi kırıyordu (Faz 210: ekonomi 27 içerikle 'bas=27').
      // Burada beklenen değer DOM'dan türetiliyor; bozulma yine yakalanır
      // (imleç gösterilenden farklı olursa düşer).
      const gosterilen = (hero.leftCount + hero.slides + hero.rightCount) + first.firstCards + first.items;
      assert.equal(
        first.bas,
        String(Math.min(gosterilen, LIMIT)),
        'akış imleci gösterilen haber sayısından devam etmeli (gösterilen=' + gosterilen + ')',
      );
      assert.equal(first.sinir, String(LIMIT), 'haber sınırı DOM\'a yazılmalı');
      assert.equal(first.oto, '1', 'otomatik yükleme açık olmalı');
      assert.equal(first.pager, false, 'akış şablonunda sayfalama olmamalı');
      assert.equal(first.overflowX, 0, 'masaüstünde yatay taşma olmamalı');

      // Enerji bloğu: 3 sütunlu ızgara, 12'lik kart grupları.
      const grid = await page.evaluate(() => {
        const g = document.querySelector('.ak-flow .ak-grid');
        const t5 = g.querySelector('.ak-type5');
        const img5 = t5 ? t5.querySelector('.ak-img5') : null;
        const cap5 = t5 ? t5.querySelector('.ak-cap5') : null;
        const t2 = g.querySelector('.ak-type2');
        return {
          cols: getComputedStyle(g).gridTemplateColumns.split(' ').filter(Boolean).length,
          cards: g.querySelectorAll('.ak-item').length,
          img5Ratio: img5 ? +(img5.getBoundingClientRect().width / img5.getBoundingClientRect().height).toFixed(2) : null,
          cap5Overlaps: !!(img5 && cap5 && cap5.getBoundingClientRect().top < img5.getBoundingClientRect().bottom - 4),
          type2Flex: t2 ? getComputedStyle(t2.querySelector('.ak-fig')).flexDirection : null,
          titleFs: t5 ? getComputedStyle(t5.querySelector('.ak-t5')).fontSize : null,
        };
      });
      assert.equal(grid.cols, 3, 'enerji vitrini 3 sütun olmalı');
      // Bir enerji bloğu en fazla 12 kart (3 sütun × 4) içerir.
      //
      // NOT (Faz 210): burada eskiden `grid.cards === min(12, first.items)`
      // vardı. O eşitlik sayfa dilimine bağlıydı: kategori sayfası ilk turda
      // 24 içerik getirdiği için akış kartları 10+3 gibi bölünebiliyor ve
      // sabit beklenti kırılıyordu. Blok sözleşmesi (≤12 kart, 3 sütun,
      // blok sayısı = ceil(items/12)) veri sayısından bağımsızdır.
      assert.ok(grid.cards > 0 && grid.cards <= 12,
        'ilk enerji bloğu 1..12 kart içermeli, gelen: ' + grid.cards);
      assert.equal(grid.img5Ratio, 1, 'tip5 görseli kare olmalı');
      assert.ok(grid.cap5Overlaps, 'tip5 başlık kutusu görselin üstüne taşmalı');
      assert.equal(grid.type2Flex, 'row', 'tip2 kartında görsel solda olmalı');
      assert.equal(grid.titleFs, '20px', 'tip5 başlığı 1.25rem serif olmalı');

      // Akış tamamlanana kadar yükle (önce otomatik, sonra düğme).
      let info = first;
      for (let i = 0; i < 6 && !info.endVisible; i++) {
        await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight));
        await sleep(400);
        const btn = await page.$('[data-ak-more]:not(.is-done)');
        if (btn) {
          await btn.click();
          await sleep(500);
        } else {
          await sleep(400);
        }
        info = await flowInfo(page);
      }
      assert.ok(info.endVisible, 'sınır dolunca "Tüm haberler gösterildi" görünmeli');
      assert.equal(info.moreVisible, false, 'akış bitince yükleme düğmesi kalkmalı');
      // Üçlü manşet 10 haber (5 slayt + 3 + 2), kart satırı 4 haber tüketir.
      assert.equal(info.items, LIMIT - 14, `akışta ${LIMIT - 14} kart olmalı`);
      assert.equal(10 + info.firstCards + info.items, LIMIT,
        'manşet + kart satırı + akış toplamı haber sınırına eşit olmalı');
      assert.equal(info.bas, String(LIMIT), 'imleç sınırın sonuna gelmeli');
      assert.equal(info.overflowX, 0, 'akış sonunda yatay taşma olmamalı');

      // Mobil: ilk ızgara 2 sütun, enerji bloğu tek sütun, taşma yok.
      await setViewport(page, 390, 900);
      await goto(page, CATEGORY);
      const mob = await flowInfo(page);
      assert.equal(mob.firstCols, 2, 'mobilde ilk kartlar 2 sütun olmalı');
      assert.ok(mob.items > 0, 'mobilde akış kartları bulunmalı');
      const mobGrid = await page.evaluate(() => {
        const g = document.querySelector('.ak-flow .ak-grid');
        return g ? getComputedStyle(g).gridTemplateColumns.split(' ').filter(Boolean).length : 0;
      });
      assert.equal(mobGrid, 1, 'mobilde enerji bloğu tek sütun olmalı');
      assert.equal(mob.overflowX, 0, 'mobilde yatay taşma olmamalı');

      /* Manşet kapatılınca akış ilk haberle başlar. */
      const rawNoHero = await page.evaluate(({ src, limit }) => {
        const arr = JSON.parse(src);
        const b = arr.find((x) => x.type === 'kategori_liste');
        b.opts = Object.assign({}, b.opts, {
          view: 'akis', hero_style: 'uclu', hero_count: 5,
          akis_limit: String(limit), akis_per: '12', akis_auto: true, akis_hero: false,
        });
        return JSON.stringify(arr);
      }, { src: base.blocks, limit: LIMIT });
      assert.equal((await save(rawNoHero)).ok, true, 'manşetsiz akış kaydedilmeli');
      await setViewport(page, 1440, 1200);
      await goto(page, CATEGORY);
      const noHero = await flowInfo(page);
      assert.equal(await page.evaluate(() => document.querySelectorAll('.cat-hero').length), 0,
        'manşet kapatıldığında üstte manşet alanı olmamalı');
      assert.equal(noHero.firstCards, 4, 'ilk 4 haber kart ızgarasında kalmalı');
      assert.equal(noHero.items, 20, 'akış 20 haberle başlamalı (24 - ilk 4 kart)');
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

test('tasarımcı: Akış (Enerji) şablonu ve akış ayarları seçilebilir', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdminPage(async (page, errors) => {
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

    let original = null;
    let usingGlobal = false;
    try {
      await goto(page, BUILDER);
      await page.waitForSelector('.bp-chip[data-id]', { timeout: 15000 });
      original = await page.evaluate(() => JSON.stringify(window.__bldInit || []));
      usingGlobal = await page.evaluate(() => !!window.__bldUsingGlobal);
      // Genel kategori düzeni modülere ayrılmış olabilir; eski Akış şablonu
      // ayarlarını sınamak için geçici bir kategori_liste düzeni kurulur.
      await ensureLegacyCategory(page, {
        pageKey: PAGE_KEY, builderUrl: BUILDER,
        opts: { view: 'akis', akis_limit: 60, akis_per: 12, akis_auto: true, akis_hero: true },
      });
      await page.waitForSelector('.bp-chip[data-id]', { timeout: 15000 });

      const opened = await page.evaluate(() => {
        const chips = [...document.querySelectorAll('.bp-chip[data-id]')];
        for (const ch of chips) {
          const ty = ch.querySelector('.ty');
          if (ty && ty.textContent.trim() === 'kategori_liste') {
            ch.querySelector('button[data-act="cfg"]').click();
            return true;
          }
        }
        return false;
      });
      assert.ok(opened, 'kategori_liste blok çipi bulunmalı');

      await page.waitForSelector('select[data-k="view"]', { timeout: 5000 });
      const ui = await page.evaluate(() => {
        const tpl = document.querySelector('.catview-btn[data-view="akis"]');
        return {
          tplName: tpl ? tpl.textContent.replace(/\s+/g, ' ').trim() : '',
          views: [...document.querySelectorAll('select[data-k="view"] option')].map((o) => o.value).filter(Boolean),
          limit: !!document.querySelector('input[data-k="akis_limit"]'),
          per: document.querySelector('select[data-k="akis_per"]') ?
            [...document.querySelectorAll('select[data-k="akis_per"] option')].map((o) => o.value).filter(Boolean) : [],
          auto: !!document.querySelector('input[data-k="akis_auto"]'),
          hero: !!document.querySelector('input[data-k="akis_hero"]'),
          fonts: document.querySelector('select[data-k="akis_font"]') ?
            [...document.querySelectorAll('select[data-k="akis_font"] option')].map((o) => o.value) : [],
        };
      });
      assert.ok(ui.tplName, '"Akış (Enerji)" şablon düğmesi bulunmalı');
      assert.match(ui.tplName, /Akış/, 'düğme adı Akış şablonunu işaret etmeli');
      assert.ok(ui.views.includes('akis'), 'liste görünümü seçeneğinde akış bulunmalı');
      assert.ok(ui.limit, 'akış haber sınırı alanı bulunmalı');
      assert.deepEqual(ui.per, ['12', '24', '36'], 'yüklenen haber sayısı seçimi bulunmalı');
      assert.ok(ui.auto, 'otomatik yükleme anahtarı bulunmalı');
      assert.ok(ui.hero, 'üst manşet alanı anahtarı bulunmalı');
      assert.ok(ui.fonts.includes('') && ui.fonts.includes('serif') &&
        ui.fonts.includes('mono') && ui.fonts.includes('system'),
        'akış yazı fontu seçenekleri olmalı: ' + JSON.stringify(ui.fonts));
      assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
    } finally {
      if (usingGlobal) {
        assert.equal(await reset(), 200, 'geçici Ekonomi özel düzeni temizlenmeli');
      } else if (original) {
        assert.equal((await save(original)).ok, true, 'özgün Ekonomi düzeni geri yüklenmeli');
      }
    }
  });
});

test('akış yazı fontu: tek ayar tüm akışa uygulanır, coinhit serif\'e geri dönülür', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdminPage(async (page, errors) => {
    await setViewport(page, 1440, 1200);
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

    let usingGlobal = false;
    let original = null;
    try {
      await goto(page, '/admin/tasarim?sayfa=kategori&kategori=ekonomi');
      await page.waitForFunction(() => window.__bldInit, { timeout: 15000 });
      original = await page.evaluate(() => JSON.stringify(window.__bldInit));
      usingGlobal = await page.evaluate(() => !!window.__bldUsingGlobal);
      const base = await ensureLegacyCategory(page, {
        pageKey: PAGE_KEY,
        opts: {
          view: 'akis', akis_hero: false, akis_limit: '60',
          akis_per: '12', akis_auto: true,
        },
      });

      const withFont = (font) => page.evaluate(({ src, font }) => {
        const arr = JSON.parse(src);
        const b = arr.find((x) => x.type === 'kategori_liste');
        if (!b) throw new Error('kategori_liste bloğu bulunamadı');
        b.opts = Object.assign({}, b.opts, {
          view: 'akis', akis_hero: false, akis_limit: '60', akis_per: '12', akis_auto: true,
        });
        if (font) {
          b.opts.akis_font = font;
        } else {
          delete b.opts.akis_font;
        }
        return JSON.stringify(arr);
      }, { src: base.blocks, font });

      const fonts = () => page.evaluate(() => {
        const ff = (sel) => {
          const el = document.querySelector(sel);
          return el ? getComputedStyle(el).fontFamily : null;
        };
        const flow = document.querySelector('.ak-flow');
        return {
          stream: ff('.ak-flow .ak-t'),
          big: ff('.ak-flow .ak-t5'),
          first: ff('.ak-first .card-pad h3'),
          flowVar: flow ? (flow.getAttribute('style') || '') : '',
        };
      });

      /* Seçilen font tüm akışa tek ayarla uygulanır. */
      assert.equal((await save(await withFont('mono'))).ok, true, 'mono font kaydedilmeli');
      await goto(page, CATEGORY);
      const mono = await fonts();
      assert.match(mono.flowVar, /--ak-font:/, 'akış konteynerinde font değişkeni olmalı');
      assert.match(mono.stream, /monospace/, 'enerji kartı başlığı seçilen fontta olmalı');
      assert.match(mono.big, /monospace/, 'tip5 başlığı da aynı fontta olmalı');
      assert.equal(mono.first, mono.stream, 'ilk 4 kart ile akış kartları aynı fontu kullanmalı');
      assert.match(mono.first, /monospace/, 'ilk 4 kartın başlığı da aynı fontta olmalı');

      /* Boş seçim: sayfa başlıkları (slider) ile aynı font — akış tek font. */
      assert.equal((await save(await withFont(''))).ok, true, 'varsayılana dönüş kaydedilmeli');
      await goto(page, CATEGORY);
      const theme = await fonts();
      assert.doesNotMatch(theme.flowVar, /--ak-font:/, 'boş seçimde ek font değişkeni olmamalı');
      assert.equal(theme.first, theme.stream, 'varsayılanda da tüm akış tek font kullanmalı');
      assert.equal(theme.big, theme.stream, 'tip5 başlığı da aynı fontta olmalı');
      assert.doesNotMatch(theme.stream, /Georgia/, 'varsayılan artık tema fontu olmalı');

      /* Coinhit serif'ine istenen an geri dönülebilir. */
      assert.equal((await save(await withFont('serif'))).ok, true, 'coinhit serif kaydedilmeli');
      await goto(page, CATEGORY);
      const serif = await fonts();
      assert.match(serif.stream, /Georgia/, 'coinhit serif fontuna geri dönülebilmeli');
      assert.match(serif.big, /Georgia/, 'tip5 başlığı da serif olmalı');
      assert.equal(serif.first, serif.stream, 'serif seçiminde de tüm akış tek font kullanmalı');
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