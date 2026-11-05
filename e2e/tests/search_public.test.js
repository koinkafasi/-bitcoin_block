'use strict';
/**
 * Public arama — sonuç listesi ve ALTYAPI SIZINTISI regresyonu.
 *
 *  - /arama makale sonuçları listeler
 *  - sayfa HANGİ arama motorunun/veritabanının çalıştığını GÖSTERMEZ
 *
 * NEDEN: üretimde (5 Kasım 2026) `/arama?q=…` "Motor: PostgreSQL (pg_trgm) ·
 * 0 sonuç" basıyordu. Bu altyapı bilgisidir: ziyaretçiye faydası yok,
 * saldırgana sürüm/CVE hedeflemesi için ipucu verir. Eski test tam tersini
 * (etiketin GÖRÜNMESİNİ) bekliyordu; bu yüzden sızıntıyı koruyordu.
 */

const test = require('node:test');
const assert = require('node:assert/strict');
const { goto, setViewport, withPage } = require('../lib/harness');

test('arama: sonuçlar listelenir, motor/veritabanı adı sızdırılmaz', async () => {
  await withPage(async (page) => {
    await setViewport(page, 1400);
    await goto(page, '/arama?q=borsa');
    const r = await page.evaluate(() => {
      const metin = document.body.textContent || '';
      return {
        title: document.title,
        hasForm: !!document.querySelector('form[action="/arama"], form[action*="arama"]'),
        results: [...document.querySelectorAll('a[href]')].filter((a) => /\-\d+$/.test(a.getAttribute('href'))).length,
        sizinti: ['Motor:', 'PostgreSQL', 'pg_trgm', 'Tantivy', 'search-svc'].filter((s) =>
          metin.includes(s),
        ),
      };
    });
    assert.match(r.title, /Arama/i, 'arama sayfası başlığı');
    assert.ok(r.hasForm, 'arama formu bulunmalı');
    assert.deepEqual(r.sizinti, [], 'sayfa altyapı bilgisi sızdırmamalı');
  });
});
