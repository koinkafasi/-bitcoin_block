'use strict';
/**
 * Kategori Panelleri — "Ekonomi vitrini (Habertürk birebir)" düzeni.
 *
 * Habertürk Ekonomi bölümünün birebir karşılığı:
 * - Bölüm çerçevesiz/beyaz; başlık siyah kalın (1.25rem/900) + ince çizgi + TÜMÜ
 * - 4 sütunlu ızgara; solda 2 sütun × 2 satır büyük KARE kart
 * - Sağda iki sütun, her sütunda 2 kart (kare + 16:9 dönüşümlü)
 * - Başlıklar görselin ALTINDA, 2 satırla sınırlı
 * - Mobilde büyük kart tam genişlik, sağda 2 sütun; yatay taşma yok
 *
 * Düzen canlı ayarda seçili değilse test atlanır (veri değiştirmez).
 */

const test = require('node:test');
const assert = require('node:assert/strict');
const { BASE_URL, goto, setViewport, withPage, withAdminPage, withTempBlock } = require('../lib/harness');

const PANEL = '.kat-panel-blok.kp-ht';

async function measure(page) {
  return page.evaluate((sel) => {
    const sec = document.querySelector(sel);
    if (!sec) return null;
    const rect = (el) => {
      if (!el) return null;
      const r = el.getBoundingClientRect();
      return { x: Math.round(r.x), y: Math.round(r.y), w: Math.round(r.width), h: Math.round(r.height) };
    };
    const h2 = sec.querySelector('.galeri-etiket');
    const line = sec.querySelector('.kp-line');
    const all = sec.querySelector('.galeri-head > a');
    const grid = sec.querySelector('.ek-grid');
    const big = sec.querySelector('.ek-big');
    const cols = [...sec.querySelectorAll('.ek-col')];
    const cards = [...sec.querySelectorAll('.ek-card')];
    const cs = getComputedStyle(sec);
    const h2cs = getComputedStyle(h2);
    const ratio = (card) => {
      const img = card.querySelector('img');
      if (!img) return null;
      const r = img.getBoundingClientRect();
      return r.height ? +(r.width / r.height).toFixed(2) : null;
    };
    return {
      cls: sec.className,
      bg: cs.backgroundColor,
      border: cs.borderTopWidth,
      pad: cs.padding,
      h2: { ...rect(h2), fs: h2cs.fontSize, weight: h2cs.fontWeight, color: h2cs.color, bg: h2cs.backgroundColor },
      line: rect(line),
      all: { text: all.textContent.trim(), fs: getComputedStyle(all).fontSize },
      grid: { ...rect(grid), cols: getComputedStyle(grid).gridTemplateColumns.split(' ').filter(Boolean).length, gap: getComputedStyle(grid).gap },
      big: rect(big),
      colRects: cols.map(rect),
      count: cards.length,
      cards: cards.map((c) => {
        const t = c.querySelector('.ek-t');
        const img = c.querySelector('img');
        const tcs = getComputedStyle(t);
        return {
          ratio: ratio(c),
          titleBelow: img ? !!(t.compareDocumentPosition(img) & Node.DOCUMENT_POSITION_PRECEDING) : false,
          fs: tcs.fontSize,
          lh: tcs.lineHeight,
          weight: tcs.fontWeight,
          lines: Math.max(1, Math.round(
            (t.getBoundingClientRect().height - parseFloat(tcs.paddingTop) - parseFloat(tcs.paddingBottom))
            / parseFloat(tcs.lineHeight))),
        };
      }),
      overflow: document.documentElement.scrollWidth - document.documentElement.clientWidth,
    };
  }, PANEL);
}

test('kategori paneli: Ekonomi vitrini Habertürk düzeni (masaüstü + mobil)', async (t) => {
  let desktop = null;
  await withPage(async (page) => {
    await setViewport(page, 1440, 1200);
    await goto(page, BASE_URL + '/');
    desktop = await measure(page);
    if (!desktop) return t.skip('Ekonomi vitrini düzeni seçili değil');

    // Bölüm: çerçevesiz, beyaz/şeffaf.
    assert.equal(desktop.bg, 'rgba(0, 0, 0, 0)', 'panel zemini şeffaf olmalı');
    assert.equal(desktop.border, '0px', 'panel çerçevesi olmamalı');
    assert.equal(desktop.pad, '0px', 'panel dolgusu olmamalı');

    // Başlık: siyah, kalın, 20px; ince çizgi kalan genişliği doldurur.
    assert.equal(desktop.h2.fs, '20px', 'başlık 1.25rem olmalı');
    assert.equal(desktop.h2.weight, '900', 'başlık font-black olmalı');
    assert.match(desktop.h2.bg, /rgba\(0, 0, 0, 0\)/, 'başlık chip zemini olmamalı');
    assert.ok(desktop.h2.w < 200, 'başlık yalnız metin kadar geniş olmalı: ' + desktop.h2.w);
    assert.ok(desktop.line.w > desktop.h2.w * 2, 'çizgi kalan genişliği doldurmalı');
    assert.equal(desktop.all.text, 'TÜMÜ', 'bağlantı TÜMÜ olmalı');
    assert.equal(desktop.all.fs, '12px', 'TÜMÜ .75rem olmalı');

    // Izgara: 4 sütun, 16px boşluk, 5 kart.
    assert.equal(desktop.grid.cols, 4, '4 sütunlu ızgara olmalı');
    assert.equal(desktop.grid.gap, '16px', 'ızgara boşluğu 16px olmalı');
    assert.equal(desktop.count, 5, '1 büyük + 4 küçük kart olmalı');
    assert.equal(desktop.colRects.length, 2, 'sağda iki sütun olmalı');

    // Büyük kart 2 sütun genişliğinde ve kare görselli.
    const bigShare = desktop.big.w / desktop.grid.w;
    assert.ok(bigShare > 0.45 && bigShare < 0.55, 'büyük kart ızgaranın yarısını kaplamalı: ' + bigShare.toFixed(2));
    assert.equal(desktop.cards[0].ratio, 1, 'büyük görsel kare (1:1) olmalı');
    assert.equal(desktop.cards[1].ratio, 1, 'ilk küçük görsel kare olmalı');
    assert.equal(desktop.cards[2].ratio, 1.78, 'ikinci küçük görsel 16:9 olmalı');
    assert.equal(desktop.cards[3].ratio, 1.78, 'üçüncü küçük görsel 16:9 olmalı');
    assert.equal(desktop.cards[4].ratio, 1, 'dördüncü küçük görsel kare olmalı');

    // Başlıklar görselin altında, kalın, en fazla 2 satır.
    for (const c of desktop.cards) {
      assert.ok(c.titleBelow, 'başlık görselin altında olmalı');
      assert.equal(c.weight, '700', 'başlık kalın olmalı');
      assert.ok(c.lines <= 2, 'başlık en fazla 2 satır olmalı: ' + c.lines);
    }
    assert.equal(desktop.cards[0].fs, '20px', 'başlık 20px olmalı (lg:text-xl)');
    assert.equal(desktop.cards[0].lh, '32px', 'satır yüksekliği 32px olmalı (lg:leading-8)');
    assert.equal(desktop.overflow, 0, 'masaüstünde yatay taşma olmamalı');
  });
  if (!desktop) return;

  await withPage(async (page) => {
    await setViewport(page, 390, 900);
    await goto(page, BASE_URL + '/');
    const m = await measure(page);
    assert.ok(m, 'mobilde panel bulunmalı');
    assert.equal(m.grid.cols, 2, 'mobilde 2 sütun olmalı');
    assert.ok(Math.abs(m.big.w - m.grid.w) <= 1, 'mobilde büyük kart tam genişlik olmalı');
    assert.equal(m.colRects.length, 2, 'mobilde sağ taraf 2 sütun olmalı');
    assert.equal(m.count, 5, 'mobilde de 5 kart olmalı');
    assert.equal(m.overflow, 0, 'mobilde yatay taşma olmamalı');
    assert.equal(m.cards[0].fs, '14px', 'mobil başlık .875rem olmalı');
  });
});

test('tasarımcı: Ekonomi vitrini düzeni seçilebilir', async (t) => {
  if (!process.env.TB_SESSION) return t.skip('TB_SESSION yok (admin testi)');
  await withAdminPage(async (page, errors) => {
    // SAHNEYİ TEST KURAR: `duzen` alanı yalnız "Kategori Paneli"
    // modülünde var. Taze E2E veritabanında ana sayfa düzeninde o modül
    // yoktu, dolayısıyla test hiçbir şey ölçemiyordu (dolu veritabanında
    // geçmesi yanıltıcıydı). withTempBlock geçici ekler ve sonunda
    // özgün düzeni geri yükler — kalıcı değişiklik bırakmaz.
    await withTempBlock(page, { pageKey: 'ana', type: 'kategori_panel', opts: { kategori: 'ekonomi', count: 9 } },
      async () => {
        // Tasarımcı varsayılan olarak "plan" (bp) görünümünde açılır; modül
        // ayarları LİSTE görünümünde çizilir. Eski test `#kpList .kp-row`
        // arıyordu — o DOM Faz 162'de kaldırıldı (kategori panelleri artık
        // modülün kendi ayarı). Bugünkü karşılık: `.bld-block` + `select[data-k=duzen]`.
        await page.waitForSelector('.bld-views button[data-view="list"]', { timeout: 15000 });
        await page.click('.bld-views button[data-view="list"]');
        await page.waitForSelector('#bldCanvas .bld-block', { timeout: 15000 });
        const opts = await page.$$eval('select[data-k="duzen"]', (els) =>
          els.length ? [...els[0].options].map((o) => ({ value: o.value, label: o.textContent.trim() })) : []);
        assert.ok(opts.length > 0, 'vitrin düzeni seçeneği (duzen) bulunamadı');
        const eco = opts.find((o) => o.value === 'ekonomi');
        assert.ok(eco, '"ekonomi" düzeni seçeneklerde olmalı');
        assert.match(eco.label, /Habertürk/, 'seçenek adı Habertürk modelini belirtmeli');
        // Seçenek kümesi tasarım kararıdır; daralması gerçek regresyondur.
        for (const v of ['iki', 'ust3alt2', 'mozaik', 'solbuyuk', 'buyuk']) {
          assert.ok(opts.some((o) => o.value === v), v + ' düzeni kaybolmamalı');
        }
        assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
      });
  });
});
