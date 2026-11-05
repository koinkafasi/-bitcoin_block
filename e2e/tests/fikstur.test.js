'use strict';
/** Fikstür / puan durumu sayfası davranış testleri. */

const test = require('node:test');
const assert = require('node:assert/strict');
const { BASE_URL, goto, measureBrokenImages, withPage } = require('../lib/harness');

test('fikstür sayfası yükleniyor', async () => {
  await withPage(async (page) => {
    const resp = await goto(page, '/fikstur');
    assert.equal(resp.status(), 200);
    const text = await page.evaluate(() => document.body.innerText);
    assert.match(text, /Puan Durumu/i, 'puan durumu başlığı olmalı');
    assert.match(text, /Fikstür/i, 'fikstür başlığı olmalı');
  });
});

test('lig seçici mevcut ve en az 6 lig sunuyor', async () => {
  await withPage(async (page) => {
    await goto(page, '/fikstur');
    const options = await page.$$eval('select option', (os) =>
      os.map((o) => ({ value: o.value, text: o.textContent.trim() }))
    );
    assert.ok(options.length >= 6, `en az 6 lig beklenir, bulunan: ${options.length}`);
    assert.ok(
      options.some((o) => /süper lig/i.test(o.text)),
      'Süper Lig seçeneği olmalı'
    );
  });
});

test('PUAN DURUMU / FİKSTÜR sekmeleri çalışıyor', async () => {
  await withPage(async (page) => {
    await goto(page, '/fikstur');

    const tabs = await page.$$eval('[role="tab"], .fx-tab, .tabs button, .tab', (els) =>
      els.map((e) => e.textContent.trim())
    );
    assert.ok(tabs.length > 0, 'sekme bulunamadı');

    const joined = tabs.join(' ').toUpperCase();
    assert.match(joined, /PUAN/, 'PUAN DURUMU sekmesi olmalı');
    assert.match(joined, /FİKSTÜR|FIKSTUR/, 'FİKSTÜR sekmesi olmalı');
  });
});

test('hafta seçici mevcut', async () => {
  await withPage(async (page) => {
    await goto(page, '/fikstur');
    const selects = await page.$$('select');
    let hasWeek = false;
    for (const s of selects) {
      const opts = await s.$$eval('option', (os) => os.length);
      const val = await s.evaluate((e) => e.value);
      if (opts > 5 || /hafta/i.test(val)) {
        hasWeek = true;
        break;
      }
    }
    assert.ok(hasWeek, 'hafta seçici bulunamadı');
  });
});

test('maç listesi ya da boş durum mesajı var', async () => {
  await withPage(async (page) => {
    await goto(page, '/fikstur');
    const text = await page.evaluate(() => document.body.innerText);
    const hasMatches = /Galatasaray|Fenerbahçe|Besiktas|Beşiktaş|maç|Maç/i.test(text);
    const hasEmpty = /maç yok|Maç bulunamadı|veri yok|—/i.test(text);
    assert.ok(hasMatches || hasEmpty, 'maç listesi veya boş durum mesajı olmalı');
  });
});

test('geçersiz lig parametresi sayfayı bozmaz (200 + varsayılan)', async () => {
  await withPage(async (page) => {
    const resp = await goto(page, '/fikstur?lig=olmayan-boyle-bir-lig');
    assert.equal(resp.status(), 200, 'geçersiz lig 404 veya 500 vermemeli');
    const text = await page.evaluate(() => document.body.innerText);
    assert.match(text, /Puan Durumu/i);
  });
});

test('geçersiz hafta parametresi sayfayı bozmaz', async () => {
  for (const q of ['?hafta=abc', '?hafta=-5', '?hafta=99999', '?hafta=']) {
    await withPage(async (page) => {
      const resp = await goto(page, '/fikstur' + q);
      assert.equal(resp.status(), 200, `${q} 200 dönmeli`);
    });
  }
});

test('lig değiştirilebiliyor', async () => {
  await withPage(async (page) => {
    await goto(page, '/fikstur?lig=bundesliga');
    const text = await page.evaluate(() => document.body.innerText);
    assert.match(text, /Bundesliga/i, 'Bundesliga seçiliyken sayfa Bundesliga göstermeli');
  });
});

test('kendi kaynaklarımızda kırık görsel yok', async () => {
  await withPage(async (page) => {
    await goto(page, '/fikstur');
    // Yalnızca kendi sunucumuzdan gelen görseller denetlenir; dış kaynaklar
    // (ESPN logoları) test sırasında bilerek engellenir.
    const broken = await measureBrokenImages(page, BASE_URL);
    assert.deepEqual(broken, [], `kırık görseller: ${JSON.stringify(broken)}`);
  });
});
