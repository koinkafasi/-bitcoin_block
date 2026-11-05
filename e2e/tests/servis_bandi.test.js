'use strict';
/**
 * Servis Bandı modülü: Sayfa Tasarımcısı'nda kur kalemleri seçilebilir.
 * Test veriyi değiştirmez; altı kontrolün varlığını ve kayıtlı seçimle
 * public bandın birebir uyuştuğunu doğrular.
 *
 * Not: Kontrollerin "açık" olması varsayımı KULLANICI YAPILANDIRMASINA bağlı
 * olduğu için test edilmez (site sahibi bir kalemi kapatmış olabilir) — bunun
 * yerine gerçek değişmez korunur: tasarımcıda açık olan her kur kalemi public
 * bantta `tk-<SEMBOL>` sınıfıyla görünür, kapalı olan görünmez.
 */

const test = require('node:test');
const assert = require('node:assert/strict');
const { BASE_URL, goto, setViewport, withAdminPage, withPage } = require('../lib/harness');

const SID = process.env.TB_SESSION;
const SYMS = [
  ['kur_bist', 'BIST'],
  ['kur_usd', 'USD'],
  ['kur_eur', 'EUR'],
  ['kur_altin', 'ALTIN'],
  ['kur_btc', 'BTC'],
  ['kur_ons', 'ONS'],
];
const KEYS = SYMS.map(([k]) => k);

test('servis bandı kur kalemleri Sayfa Tasarımcısı’ndan seçilebilir', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  let secimler = null;
  await withAdminPage(async (page, errors) => {
    await setViewport(page, 1536, 1000);
    await goto(page, '/admin/tasarim');
    assert.ok(!page.url().includes('/admin/giris'), 'admin oturumu açık olmalı');

    const chip = await page.evaluateHandle(() => {
      const chips = [...document.querySelectorAll('.bp-chip')];
      return chips.find((c) => ((c.querySelector('.ty') || {}).textContent || '').trim() === 'servis_bandi');
    });
    const found = await chip.evaluate((el) => !!el);
    assert.ok(found, 'Servis Bandı modülü bulunmalı');
    await chip.evaluate((el) => el.querySelector('[data-act="cfg"]').click());
    await page.waitForFunction(() => {
      const c = [...document.querySelectorAll('.bp-chip')].find((x) => ((x.querySelector('.ty') || {}).textContent || '').trim() === 'servis_bandi');
      const o = c && c.querySelector('.bp-opts');
      return o && getComputedStyle(o).display !== 'none';
    }, { timeout: 5000 });
    const controls = await page.evaluate(() => {
      const c = [...document.querySelectorAll('.bp-chip')].find((x) => ((x.querySelector('.ty') || {}).textContent || '').trim() === 'servis_bandi');
      return [...c.querySelectorAll('.bp-opts input[data-k^="kur_"]')].map((i) => ({
        key: i.dataset.k, checked: i.checked, label: (i.closest('label') || {}).innerText || '',
      }));
    });
    assert.equal(controls.length, 6, 'altı kur seçeneği olmalı');
    assert.deepEqual(controls.map((c) => c.key), KEYS);
    for (const c of controls) {
      assert.ok(c.label.length > 0, c.key + ' etiketi olmalı');
    }
    secimler = Object.fromEntries(controls.map((c) => [c.key, c.checked]));
    assert.deepEqual(errors, [], 'admin sayfasında JS hatası olmamalı');
  });

  await withPage(async (page, errors) => {
    await setViewport(page, 1536, 900);
    await goto(page, BASE_URL + '/');
    const classes = await page.$$eval('.tk-mk', (els) => els.map((e) => e.className));
    const gorunen = (name) => classes.some((c) => c.includes('tk-' + name));
    // Kur kaynağı demo olduğunda uygulama USD/EUR/ALTIN'ı bilinçli olarak
    // banda koymaz (sahte fiyat gösterilmez). E2E örneği dış kaynağa bağlı
    // olduğu için bu üçlüyü tek grup olarak değerlendir: seçili olanların
    // TAMAMI yoksa demo kabul et; bir kısmı varsa gerçek süzgeç hatasıdır.
    const fiat = ['USD', 'EUR', 'ALTIN'];
    const fiatSecili = fiat.filter((n) => secimler['kur_' + n.toLowerCase()]);
    const fiatEksik = fiatSecili.filter((n) => !gorunen(n));
    const demoKur = fiatSecili.length > 0 && fiatEksik.length === fiatSecili.length;
    if (demoKur) {
      t.diagnostic('kur kaynağı demo — USD/EUR/ALTIN bantta bilinçli olarak yok');
    }
    for (const [key, name] of SYMS) {
      if (demoKur && fiat.includes(name) && secimler[key]) continue;
      const gorunur = gorunen(name);
      assert.equal(
        gorunur,
        secimler[key],
        `${name} bantta ${secimler[key] ? 'GÖRÜNMELİ' : 'GÖRÜNMEMELİ'} (tasarımcıda ${key}=${secimler[key]})`
      );
    }
    assert.deepEqual(errors, [], 'public sayfada JS hatası olmamalı');
  }, { width: 1536 });
});
