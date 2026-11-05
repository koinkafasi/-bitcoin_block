'use strict';
/**
 * Uzun başlık kırpma regresyonu.
 *
 * Sorun: `-webkit-line-clamp` ile `padding` aynı elemanda olduğunda, taşan
 * satırın birkaç pikseli alt dolgu bandında görünüyordu (bitiş kesiminden
 * sonra silik bir satır). Çözüm: dolgu dış elemanda, kırpma iç elemanda.
 *
 * Bu test, dış elemanda dolgu + iç elemanda kırpma yapısını ve kırpılan
 * yüksekliğin tam satır sayısı olduğunu doğrular (dolgu bandına sızma yok).
 */

const test = require('node:test');
const assert = require('node:assert/strict');
const { BASE_URL, goto, setViewport, withPage } = require('../lib/harness');

const LONG = 'Bu çok uzun bir manşet başlığıdır ve kırpıldıktan sonra üçüncü satırın hafifçe görünmemesi gerekir diye test amaçlı yazılmıştır.';

// dış seçici → iç (kırpılan) seçici
const PAIRS = [
  ['.kat-panel-blok.kp-ht .ek-t', '.ek-tx', 2],
  ['.kat-panel-blok.kp-gs .gs-t', '.gs-tx', 2],
  ['.ust-manset .um-t', '.um-tx', null],
  ['.gc-t', '.gc-tx', 2],
  ['.ec-t', '.ec-tx', 2],
  ['.mo-t', '.mo-tx', 2],
  ['.sb-t', '.sb-tx', 2],
  ['.u3-t', '.u3-tx', 2],
  ['.cat-hero-rail .chr-t', '.chr-tx', null],
];

test('uzun başlıklar kırpılırken dolgu bandına sızmaz', async () => {
  await withPage(async (page) => {
    await setViewport(page, 1440, 1200);
    await goto(page, BASE_URL + '/');

    const checked = await page.evaluate(({ pairs, long }) => {
      const out = [];
      for (const [outerSel, innerSel, expected] of pairs) {
        const outer = document.querySelector(outerSel);
        if (!outer) { out.push({ outerSel, skip: 'yok' }); continue; }
        let inner = outer.querySelector(innerSel);
        if (!inner) { out.push({ outerSel, fail: 'iç eleman yok' }); continue; }
        inner.textContent = long;

        const o = outer.getBoundingClientRect();
        const i = inner.getBoundingClientRect();
        const cs = getComputedStyle(outer);
        const ics = getComputedStyle(inner);
        const lh = parseFloat(ics.lineHeight);
        const pt = parseFloat(cs.paddingTop);
        const pb = parseFloat(cs.paddingBottom);
        const lines = i.height / lh;
        out.push({
          outerSel,
          padT: pt,
          padB: pb,
          lh,
          lines: +lines.toFixed(2),
          // içerik kutusu tam satır sayısı kadar mı?
          lineExact: Math.abs(lines - Math.round(lines)) < 0.02,
          // iç eleman, dış elemanın alt dolgusunun üstünde bitiyor mu?
          gap: +(o.bottom - i.bottom).toFixed(2),
          innerDisplay: ics.display,
          clamp: ics.webkitLineClamp,
          overflow: ics.overflowY,
          // taşan içerik var mı (kırpma gerçekten çalışıyor mu)?
          clipped: inner.scrollHeight > inner.clientHeight + 1,
          expected,
        });
      }
      return out;
    }, { pairs: PAIRS, long: LONG });

    for (const r of checked) {
      if (r.skip) continue; // bu sayfada o modül yok
      assert.ok(!r.fail, r.outerSel + ': ' + r.fail);
      assert.ok(Math.abs(r.padB - r.gap) <= 0.5,
        `${r.outerSel}: kırpma alt dolgu bandına sızıyor (gap=${r.gap}, padB=${r.padB})`);
      assert.ok(r.lineExact, `${r.outerSel}: kırpılan yükseklik tam satır değil (${r.lines} satır)`);
      assert.ok(r.clipped, `${r.outerSel}: uzun metin kırpılmıyor`);
      assert.equal(r.overflow, 'hidden', `${r.outerSel}: iç elemanda overflow hidden olmalı`);
      if (r.expected) {
        assert.equal(String(r.clamp), String(r.expected),
          `${r.outerSel}: satır sınırı ${r.expected} olmalı`);
      }
    }
  });
});
