'use strict';
/**
 * Medya paneli (templ geçişi + media.js ayrıştırması):
 *
 * - Panel yapısı, filtreler, yükleme kutusu yerinde.
 * - Yükleme → kart görünür → düzenleme modalı açılır/kapanır.
 * - Silme temizler; boş filtre durumu görünür.
 * - Test kendi dosyasını yükler ve sonunda siler.
 *
 * TARİHÇE (Faz 210 triyajı): Panel Murmura /library/media düzenine göre
 * yeniden yazıldı. Eski sınıflar:
 *   .media-upload (ayrı yükleme kartı + form submit)
 *   .media-filters (filtre kutusu)
 *   #medyaDosya (dosya girdisi)
 * Bugünkü karşılıklar:
 *   .mgal-drop    → galerinin İLK kutusu (sürükle-bırak + gözat)
 *   .media-cipler → kategori çipleri, .media-arac → araç çubuğu
 *   [data-mg-file] → dosya girdisi; değişince media.js OTOMATİK yükler
 *                    (form submit yok — başlık/alt metin modalda soruluyor)
 */

const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { goto, withAdminPage } = require('../lib/harness');

const PNG_B64 = 'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJgggg==';

test('medya: yükle, düzenle modalı, filtre, sil', async () => {
  await withAdminPage(async (page) => {
    const dosya = path.join(os.tmpdir(), 'tb-medya-e2e.png');
    fs.writeFileSync(dosya, Buffer.from(PNG_B64, 'base64'));

    try {
      // ---------- Yapı ----------
      await goto(page, '/admin/dosyalar');
      const yapi = await page.evaluate(() => ({
        panel: !!document.getElementById('medya-panel'),
        upload: !!document.querySelector('.mgal-drop'),
        filters: !!document.querySelector('.media-cipler'),
        toolbar: !!document.querySelector('.media-arac'),
        input: !!document.querySelector('[data-mg-file]'),
        modal: !!document.getElementById('mediaEditModal'),
        js: typeof window.__tbMediaInit === 'function',
      }));
      assert.ok(yapi.panel, 'medya paneli yok');
      assert.ok(yapi.upload, 'yükleme kutusu yok');
      assert.ok(yapi.filters, 'kategori filtreleri yok');
      assert.ok(yapi.toolbar, 'araç çubuğu yok');
      assert.ok(yapi.input, 'dosya girdisi yok');
      assert.ok(yapi.modal, 'düzenleme modalı yok');
      assert.ok(yapi.js, 'media.js bağlı olmalı');

      // ---------- Yükle ----------
      // Değişiklik olayı media.js'te otomatik yüklemeyi tetikler; form yok.
      const input = await page.$('[data-mg-file]');
      await input.uploadFile(dosya);
      await page.waitForFunction(
        () => document.querySelector('.media-item[data-filename="tb-medya-e2e.png"]'),
        { timeout: 30000 },
      );

      // ---------- Düzenleme modalı ----------
      const modalOpen = await page.evaluate(() => {
        const item = document.querySelector('.media-item[data-filename="tb-medya-e2e.png"]');
        item.querySelector('[data-media-open]').click();
        return !document.getElementById('mediaEditModal').hidden;
      });
      assert.ok(modalOpen, 'modal açılmalı');
      await page.keyboard.press('Escape');
      const modalClosed = await page.evaluate(() => document.getElementById('mediaEditModal').hidden);
      assert.ok(modalClosed, 'Esc ile kapanmalı');

      // ---------- Boş filtre durumu ----------
      await goto(page, '/admin/dosyalar?q=zzz-olmayan-arama');
      const bos = await page.evaluate(() => document.body.textContent.includes('Bu filtreye uygun medya yok'));
      assert.ok(bos, 'boş filtre durumu görünmeli');

      // ---------- Sil (kendi kaydımızı temizle) ----------
      await goto(page, '/admin/dosyalar?q=tb-medya-e2e');
      const silindi = await page.evaluate(async () => {
        const item = document.querySelector('.media-item[data-filename="tb-medya-e2e.png"]');
        if (!item) return 'zaten-yok';
        const del = item.querySelector('[data-confirm]');
        const url = del.getAttribute('data-confirm');
        const fd = new FormData();
        const r = await fetch(url, { method: 'POST', body: fd, credentials: 'same-origin' });
        return r.status === 200 ? 'silindi' : 'hata:' + r.status;
      });
      assert.match(silindi, /silindi|zaten-yok/, 'temizlik yapılabilmeli: ' + silindi);
    } finally {
      fs.rmSync(dosya, { force: true });
    }
  });
});