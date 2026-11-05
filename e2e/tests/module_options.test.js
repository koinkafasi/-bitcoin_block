'use strict';
/**
 * Modül başlık/etiket seçenekleri:
 *
 * - Başlık işaret stili görsel seçiciden (çizgi, çubuk, alt çizgi, etiket,
 *   nokta, köşeli, sekme, çerçeve, gradyan, işaretsiz) seçilir.
 * - "Modül başlığını gizle" başlığı görsel olarak gizler, DOM'da bırakır.
 * - "Kategori etiketini gizle" etiketi görsel olarak gizler, DOM'da bırakır.
 */

const test = require('node:test');
const assert = require('node:assert/strict');
const { BASE_URL, goto, setViewport, withAdminPage, ensureLegacyCategory } = require('../lib/harness');

const SID = process.env.TB_SESSION;
const PAGE_KEY = 'kategori:ekonomi';
const BUILDER = '/admin/tasarim?sayfa=kategori&kategori=ekonomi';
const SAVE_URL = BASE_URL + '/admin/tasarim/kaydet?sayfa=' + encodeURIComponent(PAGE_KEY);
const RESET_URL = BASE_URL + '/admin/tasarim/sifirla?sayfa=' + encodeURIComponent(PAGE_KEY);

async function resetLayout(page) {
  return page.evaluate(async (url) => {
    const r = await fetch(url, { method: 'POST', credentials: 'same-origin' });
    return r.status;
  }, RESET_URL);
}

async function saveBlocks(page, blocks) {
  return page.evaluate(async ({ url, raw }) => {
    const r = await fetch(url, {
      method: 'POST',
      credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ blocks: JSON.parse(raw) }),
    });
    return await r.json();
  }, { url: SAVE_URL, raw: blocks });
}

test('modül seçenekleri: başlık stili görsel seçici ve etiket/başlık kapatma', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdminPage(async (page, errors) => {
    await setViewport(page, 1400, 1000);
    let usingGlobal = false;
    let original = null;

    try {
      await goto(page, BUILDER);
      await page.waitForFunction(() => document.querySelector('.bp-chip[data-id]'), { timeout: 15000 });
      usingGlobal = await page.evaluate(() => !!window.__bldUsingGlobal);
      original = await page.evaluate(() => JSON.stringify(window.__bldInit || []));
      // Genel kategori düzeni modülere ayrılmış olabilir; eski kategori_liste
      // seçeneklerini sınamak için geçici tek bloklu düzen kurulur.
      await ensureLegacyCategory(page, {
        pageKey: PAGE_KEY, builderUrl: BUILDER,
        opts: { hero_style: 'portre', hero_count: 5, hero_width: 58, list_cols: '2' },
      });
      await page.evaluate(() => {
        const chip = [...document.querySelectorAll('.bp-chip[data-id]')]
          .find((c) => c.innerHTML.includes('kategori_liste'));
        chip.querySelector('.ops button[data-act="cfg"]').click();
      });
      await page.waitForFunction(
        () => document.querySelector('.bp-opts [data-k="head_style"]'),
        { timeout: 8000 }
      );

      const ui = await page.evaluate(() => {
        const radios = [...document.querySelectorAll('.bp-opts input[data-k="head_style"]')];
        const demos = [...document.querySelectorAll('.bp-opts .hs-demo')];
        const hideBadge = document.querySelector('.bp-opts input[data-k="hide_badge"]');
        const hideHeading = document.querySelector('.bp-opts input[data-k="hide_heading"]');
        const b = (window.__bldInit || []).find((x) => x.type === 'kategori_liste');
        return {
          values: radios.map((r) => r.value),
          visibleDemos: demos.filter((d) => d.getBoundingClientRect().width > 10).length,
          radioType: radios.length ? radios[0].type : '',
          hideBadge: !!hideBadge,
          hideBadgeDefault: hideBadge ? hideBadge.checked : null,
          hideHeading: !!hideHeading,
          hideHeadingDefault: hideHeading ? hideHeading.checked : null,
          savedHideBadge: !!(b && b.opts && b.opts.hide_badge),
          savedHideHeading: !!(b && b.opts && b.opts.hide_heading),
        };
      });
      assert.deepEqual(ui.values,
        ['line', 'bar', 'underline', 'pill', 'dot', 'corner', 'tab', 'boxed', 'gradient', 'none'],
        '10 başlık stili seçeneği olmalı');
      assert.equal(ui.radioType, 'radio', 'başlık stilleri radyo düğmeleriyle seçilmeli');
      assert.equal(ui.visibleDemos, 10, 'her stil için görsel mini önizleme görünmeli');
      assert.ok(ui.hideBadge, 'Kategori etiketini gizle seçeneği bulunmalı');
      assert.ok(ui.hideHeading, 'Modül başlığını gizle seçeneği bulunmalı');
      assert.equal(ui.hideBadgeDefault, ui.savedHideBadge,
        'etiket gizleme anahtarı bloğun kayıtlı ayarını yansıtmalı');
      assert.equal(ui.hideHeadingDefault, ui.savedHideHeading,
        'başlık gizleme anahtarı bloğun kayıtlı ayarını yansıtmalı');

      // Ekonomi'ye özel düzene bir trend modülü ekle ve kapatma seçeneklerini uygula.
      const raw = await page.evaluate(() => JSON.stringify(window.__bldInit || []));
      const changed = await page.evaluate((blocksRaw) => {
        const arr = JSON.parse(blocksRaw);
        const list = arr.find((b) => b.type === 'kategori_liste');
        if (!list) throw new Error('kategori_liste bloğu yok');
        list.opts = Object.assign({}, list.opts, { hide_badge: true });
        arr.push({
          id: 'e2e-trend', type: 'trend_numarali', visible: true,
          opts: { zone: 'govde', count: 5, head_style: 'corner', hide_heading: true, hide_badge: true },
        });
        return JSON.stringify(arr);
      }, raw);
      assert.equal((await saveBlocks(page, changed)).ok, true, 'modül ayarları kaydedilmeli');

      await goto(page, '/kategori/ekonomi');
      const publicState = await page.evaluate(() => {
        const kicker = document.querySelector('.cat-card-kicker');
        const heroCat = document.querySelector('.cat-hero .hero-cat');
        const heading = document.querySelector('.trend-num .side-h');
        const badge = document.querySelector('.trend-num .card-badge, .trend-num .one-cat');
        const hr = heading ? heading.getBoundingClientRect() : null;
        return {
          kickerHidden: !!(kicker && kicker.classList.contains('sr-only')),
          kickerText: kicker ? kicker.textContent.trim() : '',
          heroCatHidden: !!(heroCat && heroCat.classList.contains('sr-only')),
          headingCls: heading ? heading.className : '',
          headingText: heading ? heading.textContent.trim() : '',
          headingBox: hr ? { w: hr.width, h: hr.height } : null,
          badgeHidden: !!(badge && badge.classList.contains('sr-only')),
        };
      });
      assert.ok(publicState.kickerHidden, 'kategori kartı etiketi görsel olarak gizlenmeli');
      assert.ok(publicState.kickerText.length > 0, 'gizli etiket metni DOM’da kalmalı');
      assert.ok(publicState.heroCatHidden, 'manşet kategori etiketi de gizlenmeli');
      assert.ok(publicState.headingCls.includes('head-mark-corner'),
        'seçilen başlık stili public sınıfa yansımalı: ' + publicState.headingCls);
      assert.ok(publicState.headingCls.includes('sr-only'),
        'başlık kapatılınca sr-only olmalı: ' + publicState.headingCls);
      assert.ok(publicState.headingText.includes('Trend'), 'kapalı başlık metni DOM’da kalmalı');
      assert.ok(publicState.headingBox.w <= 1 && publicState.headingBox.h <= 1,
        'kapalı başlık görsel kutu oluşturmamalı');

      // Aynı modülde başlığı aç, farklı bir stile geç: gerçekten görünmeli.
      const visible = changed.replace('"hide_heading":true', '"hide_heading":false')
        .replace('"head_style":"corner"', '"head_style":"tab"');
      assert.equal((await saveBlocks(page, visible)).ok, true, 'görünür başlık ayarı kaydedilmeli');
      await goto(page, '/kategori/ekonomi');
      const shown = await page.evaluate(() => {
        const h = document.querySelector('.trend-num .side-h');
        const r = h ? h.getBoundingClientRect() : null;
        const cs = h ? getComputedStyle(h) : null;
        return {
          cls: h ? h.className : '',
          visible: !!(r && r.width > 30 && r.height > 10),
          bg: cs ? cs.backgroundColor : '',
          text: h ? h.textContent.trim() : '',
        };
      });
      assert.ok(shown.cls.includes('head-mark-tab'), 'sekme stili uygulanmalı: ' + shown.cls);
      assert.ok(!shown.cls.includes('sr-only'), 'başlık açık olmalı');
      assert.ok(shown.visible, 'başlık görünür kutuya sahip olmalı');
      assert.ok(shown.text.includes('Trend'), 'başlık metni görünmeli');
      assert.notEqual(shown.bg, 'rgba(0, 0, 0, 0)',
        'seçilen sekme stilinin zemini uygulanmalı (varsayılan şeffaf kalmamalı)');
      assert.equal(shown.bg, 'rgb(42, 46, 69)', 'sekme zemini #2a2e45 olmalı: ' + shown.bg);
    } finally {
      if (usingGlobal) {
        assert.equal(await resetLayout(page), 200, 'geçici Ekonomi özel düzeni temizlenmeli');
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
    }
    assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
  });
});
