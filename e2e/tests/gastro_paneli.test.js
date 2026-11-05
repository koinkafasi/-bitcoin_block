'use strict';
/**
 * Kategori Panelleri — "Gastro vitrini (Habertürk birebir)" düzeni.
 *
 * Habertürk GASTRO bölümünün birebir karşılığı:
 * - Çerçevesiz panel; siyah font-black başlık + ince çizgi + TÜMÜ
 * - Yatay kaydırmalı şerit (swiper karşılığı)
 * - 1:2 dikey (portre) görselli kartlar
 * - Görünür kart: mobil 2, tablet 3, masaüstü 4
 * - Başlıklar dönüşümlü: çift sırada altta, tek sırada üstte
 * - Mobil/masaüstünde sayfa düzeyinde yatay taşma yok
 *
 * Düzen canlı ayarda seçili değilse test atlanır (veri değiştirmez).
 */

const test = require('node:test');
const assert = require('node:assert/strict');
const { BASE_URL, goto, setViewport, withPage, withAdminPage, withTempBlock } = require('../lib/harness');

const PANEL = '.kat-panel-blok.kp-gs';

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
    const scroll = sec.querySelector('.gs-scroll');
    const cards = [...sec.querySelectorAll('.gs-card')];
    const srect = scroll.getBoundingClientRect();
    const visible = cards.filter((c) => {
      const r = c.getBoundingClientRect();
      return r.left >= srect.left - 1 && r.right <= srect.right + 1;
    }).length;
    const first = cards[0];
    const second = cards[1];
    const img = first.querySelector('img');
    const t0 = first.querySelector('.gs-t');
    const t1 = second.querySelector('.gs-t');
    const tcs = getComputedStyle(t0);
    return {
      cls: sec.className,
      bg: getComputedStyle(sec).backgroundColor,
      border: getComputedStyle(sec).borderTopWidth,
      h2: {
        fs: getComputedStyle(h2).fontSize,
        weight: getComputedStyle(h2).fontWeight,
        w: rect(h2).w,
        bg: getComputedStyle(h2).backgroundColor,
      },
      lineW: rect(line).w,
      all: all.textContent.trim(),
      scroll: { w: rect(scroll).w, overflow: scroll.scrollWidth - scroll.clientWidth },
      count: cards.length,
      visible,
      cardW: rect(first).w,
      gap: getComputedStyle(cards[0]).paddingLeft,
      imgRatio: img ? +(img.getBoundingClientRect().width / img.getBoundingClientRect().height).toFixed(2) : null,
      revCount: sec.querySelectorAll('.gs-card.gs-rev').length,
      revFlex: getComputedStyle(second.querySelector('.gs-fig')).flexDirection,
      normalFlex: getComputedStyle(first.querySelector('.gs-fig')).flexDirection,
      title: { fs: tcs.fontSize, lh: tcs.lineHeight, weight: tcs.fontWeight, pad: tcs.padding },
      firstTitleAboveImg: t0.getBoundingClientRect().top < img.getBoundingClientRect().top,
      secondTitleAboveImg: t1.getBoundingClientRect().top < second.querySelector('img').getBoundingClientRect().top,
      overflow: document.documentElement.scrollWidth - document.documentElement.clientWidth,
    };
  }, PANEL);
}

test('kategori paneli: Gastro vitrini Habertürk düzeni (masaüstü/tablet/mobil)', async (t) => {
  let desktop = null;
  await withPage(async (page) => {
    await setViewport(page, 1440, 1200);
    await goto(page, BASE_URL + '/');
    desktop = await measure(page);
    if (!desktop) return t.skip('Gastro vitrini düzeni seçili değil');

    // Panel Habertürk görünümü: çerçevesiz + şeffaf zemin.
    assert.equal(desktop.bg, 'rgba(0, 0, 0, 0)', 'panel zemini şeffaf olmalı');
    assert.equal(desktop.border, '0px', 'panel çerçevesi olmamalı');

    // Başlık: siyah kalın 20px, yalnız metin genişliğinde; çizgi kalanı doldurur.
    assert.equal(desktop.h2.fs, '20px', 'başlık 1.25rem olmalı');
    assert.equal(desktop.h2.weight, '900', 'başlık font-black olmalı');
    assert.match(desktop.h2.bg, /rgba\(0, 0, 0, 0\)/, 'başlık chip zemini olmamalı');
    assert.ok(desktop.h2.w < 200, 'başlık yalnız metin kadar geniş olmalı: ' + desktop.h2.w);
    assert.ok(desktop.lineW > 300, 'ince çizgi kalan genişliği doldurmalı');
    assert.equal(desktop.all, 'TÜMÜ', 'bağlantı TÜMÜ olmalı');

    // Şerit yatay kaydırılabilir; sayfa taşmaz.
    assert.ok(desktop.scroll.overflow > 0, 'şerit yatay kaydırılabilir olmalı');
    assert.equal(desktop.overflow, 0, 'masaüstünde sayfa taşmamalı');
    assert.equal(desktop.visible, 4, 'masaüstünde 4 kart görünmeli');
    assert.ok(desktop.count >= 8, 'şeritte en az 8 kart olmalı: ' + desktop.count);

    // Kart: 1:2 portre görsel, 16px aralık.
    assert.equal(desktop.imgRatio, 0.5, 'görsel 1:2 portre olmalı');
    assert.equal(desktop.gap, '8px', 'kart içi yatay dolgu 8px olmalı (16px aralık)');

    // Dönüşümlü başlık konumu (Habertürk flex-col-reverse).
    assert.equal(desktop.revFlex, 'column-reverse', 'tek sıra kart ters akışta olmalı');
    assert.equal(desktop.normalFlex, 'column', 'çift sıra kart normal akışta olmalı');
    assert.ok(desktop.revCount > 0, 'ters (gs-rev) kartlar bulunmalı');
    assert.equal(desktop.firstTitleAboveImg, false, '1. kartta başlık görselin altında olmalı');
    assert.equal(desktop.secondTitleAboveImg, true, '2. kartta başlık görselin üstünde olmalı');

    // Tipografi Habertürk kırılımıyla aynı (lg: text-xl / leading-8).
    assert.equal(desktop.title.fs, '20px', 'başlık 20px olmalı');
    assert.equal(desktop.title.lh, '32px', 'satır yüksekliği 32px olmalı');
    assert.equal(desktop.title.weight, '700', 'başlık kalın olmalı');
    assert.equal(desktop.title.pad, '12px 8px', 'başlık dolgusu py-3 px-2 olmalı');
  });
  if (!desktop) return;

  await withPage(async (page) => {
    await setViewport(page, 900, 1100);
    await goto(page, BASE_URL + '/');
    const tab = await measure(page);
    assert.ok(tab, 'tablet görünümünde panel bulunmalı');
    assert.equal(tab.visible, 3, 'tablette 3 kart görünmeli (md:w-1/3)');
    assert.equal(tab.title.fs, '16px', 'tablet başlığı 16px olmalı');
    assert.equal(tab.title.lh, '28px', 'tablet satır yüksekliği 28px olmalı');
    assert.equal(tab.overflow, 0, 'tablette sayfa taşmamalı');

    await setViewport(page, 390, 900);
    await goto(page, BASE_URL + '/');
    const mob = await measure(page);
    assert.ok(mob, 'mobil görünümde panel bulunmalı');
    assert.equal(mob.visible, 2, 'mobilde 2 kart görünmeli (w-1/2)');
    assert.equal(mob.imgRatio, 0.5, 'mobilde de 1:2 portre olmalı');
    assert.equal(mob.title.fs, '14px', 'mobil başlık .875rem olmalı');
    assert.equal(mob.title.lh, '24px', 'mobil satır yüksekliği 24px olmalı');
    assert.equal(mob.overflow, 0, 'mobilde sayfa taşmamalı');
  });
});

test('tasarımcı: Gastro vitrini düzeni seçilebilir', async (t) => {
  if (!process.env.TB_SESSION) return t.skip('TB_SESSION yok (admin testi)');
  await withAdminPage(async (page, errors) => {
    // SAHNEYİ TEST KURAR: `duzen` alanı yalnız "Kategori Paneli"
    // modülünde var; taze E2E veritabanında ana sayfa düzeninde o modül
    // yoktu. withTempBlock geçici ekler, sonunda özgün düzeni geri yükler.
    await withTempBlock(page, { pageKey: 'ana', type: 'kategori_panel', opts: { kategori: 'gastronomi', count: 9 } },
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
        const gs = opts.find((o) => o.value === 'gastro');
        assert.ok(gs, '"gastro" düzeni seçeneklerde olmalı');
        assert.match(gs.label, /Habertürk/, 'seçenek adı Habertürk modelini belirtmeli');
        assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
      });
  });
});
