'use strict';
/**
 * Native medya yükleme: ses/video/dosya yüklemeleri.
 *
 * KAPSAM:
 *  1) ses (wav), belge (pdf), metin (txt) yüklemeleri kabul edilir,
 *  2) kayıtlar doğru türle (audio/document) oluşur,
 *  3) görsel olmayanlar için thumb tür ikonu (SVG) döner,
 *  4) kütüphane kartlarında tür rozeti görünür.
 *
 * Temizlik: test kendi kayıtlarını siler (scratch veri bırakmaz).
 */

const test = require('node:test');
const assert = require('node:assert/strict');
const { goto, setViewport, withAdminPage } = require('../lib/harness');

const SID = process.env.TB_SESSION;

test('native yükleme: ses, belge ve metin kabul edilir; tür ikonu basılır', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdminPage(async (page, errors) => {
    await setViewport(page, 1400, 950);
    await goto(page, '/admin/dosyalar');
    assert.ok(!page.url().includes('/admin/giris'), 'medya sayfası açık olmalı');

    // Sayfa içinde küçük sahte dosyalar üret ve yükle (fixture dosyası gerekmez).
    const yuklenen = await page.evaluate(async () => {
      const dosyalar = [
        { ad: 'e2e-native.wav', icerik: 'RIFF....WAVEfmt ', tur: 'audio/wav' },
        { ad: 'e2e-native.pdf', icerik: '%PDF-1.4 e2e', tur: 'application/pdf' },
        { ad: 'e2e-native.txt', icerik: 'e2e metin dosyasi', tur: 'text/plain' },
      ];
      const cikan = [];
      for (const d of dosyalar) {
        const fd = new FormData();
        fd.append('dosya', new File([d.icerik], d.ad, { type: d.tur }));
        const r = await fetch('/admin/dosyalar?json=1', { method: 'POST', credentials: 'same-origin', body: fd });
        const j = await r.json();
        cikan.push({ ad: d.ad, status: r.status, ok: !!j.ok, id: j.id, thumb: j.thumb, url: j.url });
      }
      return cikan;
    });

    try {
      assert.equal(yuklenen.length, 3, 'üç dosya denenmeli');
      for (const y of yuklenen) {
        assert.equal(y.status, 200, y.ad + ' yüklenmeli');
        assert.ok(y.ok, y.ad + ' başarılı dönmeli');
        assert.ok(Number(y.id) > 0, y.ad + ' kimlik almalı');
      }

      // Türler DB'de doğru mu? (kütüphane rozetleri üzerinden)
      await page.reload({ waitUntil: 'domcontentloaded' });
      const rozetler = await page.evaluate(() => {
        const kartlar = [...document.querySelectorAll('.mi-card, [data-media-open]')];
        const metin = document.body.innerHTML;
        return {
          ses: (metin.match(/mi-kind">Ses</g) || []).length,
          dosya: (metin.match(/mi-kind">Dosya</g) || []).length,
          dosyaAdlari: ['e2e-native.wav', 'e2e-native.pdf', 'e2e-native.txt']
            .filter((ad) => metin.includes(ad)).length,
        };
      });
      assert.ok(rozetler.ses >= 1, 'ses dosyası "Ses" rozetiyle görünmeli');
      assert.ok(rozetler.dosya >= 2, 'pdf/txt "Dosya" rozetiyle görünmeli');
      assert.equal(rozetler.dosyaAdlari, 3, 'üç dosya da kütüphanede listelenmeli');

      // Görsel olmayan medya thumb'u tür ikonu (SVG) dönmeli.
      for (const y of yuklenen) {
        const t = await page.evaluate(async (thumb) => {
          const r = await fetch(thumb, { credentials: 'same-origin' });
          return { status: r.status, ct: r.headers.get('content-type') || '' };
        }, y.thumb);
        assert.equal(t.status, 200, y.ad + ' thumb 200 dönmeli (kırık görsel değil)');
        assert.match(t.ct, /svg/, y.ad + ' thumb tür ikonu (SVG) olmalı');
      }
    } finally {
      // Temizlik: oluşturulan kayıtları sil.
      for (const y of yuklenen) {
        if (Number(y.id) > 0) {
          await page.evaluate(async (id) => {
            await fetch('/admin/dosyalar/' + id + '/sil', { method: 'POST', credentials: 'same-origin' });
          }, y.id);
        }
      }
    }
    assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
  });
});

test('native yükleme: desteklenmeyen uzantı reddedilir', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdminPage(async (page) => {
    await setViewport(page, 1400, 950);
    await goto(page, '/admin/dosyalar');
    const sonuc = await page.evaluate(async () => {
      const fd = new FormData();
      fd.append('dosya', new File(['MZ...'], 'e2e-native.exe', { type: 'application/octet-stream' }));
      const r = await fetch('/admin/dosyalar?json=1', { method: 'POST', credentials: 'same-origin', body: fd });
      return { status: r.status, j: await r.json() };
    });
    assert.equal(sonuc.status, 400, 'bilinmeyen uzantı 400 dönmeli');
    assert.equal(sonuc.j.ok, false, 'yüklemeler reddedilmeli');
  });
});