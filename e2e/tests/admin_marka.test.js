'use strict';
/**
 * Marka Merkezi (ADMIN-IA konsolidasyonu + hata düzeltmeleri):
 *
 * - Görsel logo yüklenince mod otomatik "görsel"e geçmeli (eski hata:
 *   radyo işaretli değilse logo hiç değişmezdi).
 * - Yüklenemeyen dosya sessizce yutulmaz; flash'ta görünür.
 * - Site Başlığı + Slogan artık bu sayfadan yönetilir.
 * - webp gibi çözülemeyen ama bilinen uzantılı görseller yine saklanır.
 *
 * Test marka ayarlarını YEDEKLER ve sonunda GERİ YÜKLER.
 */

const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { goto, withAdminPage } = require('../lib/harness');

// 1x1 PNG
const PNG_B64 = 'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJgggg==';

async function backupBrand(page) {
  await goto(page, '/admin/marka');
  const b = await page.evaluate(() => {
    const f = document.querySelector('#marka-panel form');
    const q = (n) => { const el = f.querySelector('[name="' + n + '"]'); return el ? el.value : ''; };
    const img = f.querySelector('img[alt="logo"]');
    return {
      mode: (f.querySelector('input[name="mode"]:checked') || {}).value || 'text',
      text: q('text'),
      tagline: q('tagline'),
      site_baslik: q('site_baslik'),
      slogan: q('slogan'),
      logoSrc: img ? img.src : null,
    };
  });
  // Mevcut logo görselini indir (geri yüklemede yeniden yüklemek için)
  if (b.logoSrc) {
    b.logoB64 = await page.evaluate(async (src) => {
      const r = await fetch(src, { credentials: 'same-origin' });
      const buf = new Uint8Array(await r.arrayBuffer());
      let s = '';
      for (let i = 0; i < buf.length; i++) s += String.fromCharCode(buf[i]);
      return btoa(s);
    }, b.logoSrc);
  }
  return b;
}

async function saveBrand(page, fields, files = {}) {
  await goto(page, '/admin/marka');
  // Dosyaları yükle
  for (const [name, file] of Object.entries(files)) {
    const input = await page.$('#marka-panel form input[name="' + name + '"]');
    assert.ok(input, 'dosya alanı yok: ' + name);
    await input.uploadFile(file);
  }
  // Alanları doldur
  await page.evaluate((f) => {
    const form = document.querySelector('#marka-panel form');
    for (const [k, v] of Object.entries(f)) {
      if (k === 'mode') {
        const r = form.querySelector('input[name="mode"][value="' + v + '"]');
        if (r) r.checked = true;
        continue;
      }
      const el = form.querySelector('[name="' + k + '"]');
      if (el) {
        if (el.type === 'checkbox') el.checked = !!v;
        else el.value = v;
      }
    }
  }, fields);
  // Kaydet ve panelin değişmesini bekle
  await page.evaluate(() => {
    const form = document.querySelector('#marka-panel form');
    form.querySelector('button[type="submit"]').click();
  });
  await page.waitForFunction(
    () => document.querySelector('#marka-panel .flash'),
    { timeout: 45000 },
  );
  return page.evaluate(() => document.querySelector('#marka-panel .flash').textContent.trim());
}

test('marka merkezi: otomatik mod geçişi, görünür hata, başlık/slogan', async () => {
  await withAdminPage(async (page) => {
    const backup = await backupBrand(page);

    const png = path.join(os.tmpdir(), 'tb-marka-e2e.png');
    const txt = path.join(os.tmpdir(), 'tb-marka-e2e.txt');
    fs.writeFileSync(png, Buffer.from(PNG_B64, 'base64'));
    fs.writeFileSync(txt, 'bu bir gorsel degil');

    try {
      // ---------- 1) Logo yükle, radyoyu İŞARETLEME (eski hata senaryosu) ----------
      let flash = await saveBrand(page, {
        mode: 'text', // bilerek metin kalsın
        text: backup.text || 'theblock',
        tagline: backup.tagline,
        site_baslik: backup.site_baslik,
        slogan: backup.slogan,
      }, { logo: png });
      assert.match(flash, /otomatik açıldı/, 'mod otomatik geçmeli: ' + flash);
      const after = await page.evaluate(() => {
        const f = document.querySelector('#marka-panel form');
        return {
          mode: (f.querySelector('input[name="mode"]:checked') || {}).value,
          hasLogo: !!f.querySelector('img[alt="logo"]'),
        };
      });
      assert.equal(after.mode, 'image', 'mod "görsel" olmalı');
      assert.ok(after.hasLogo, 'logo önizlemesi görünmeli');

      // ---------- 2) Başlık/slogan bu sayfadan yönetilir ----------
      flash = await saveBrand(page, {
        mode: 'image',
        site_baslik: 'E2E Başlık ' + Date.now(),
        slogan: 'E2E Slogan',
      });
      assert.match(flash, /kaydedildi/);
      const kimlik = await page.evaluate(() => {
        const f = document.querySelector('#marka-panel form');
        return {
          baslik: f.querySelector('[name="site_baslik"]').value,
          slogan: f.querySelector('[name="slogan"]').value,
        };
      });
      assert.match(kimlik.baslik, /E2E Başlık/, 'başlık korunmalı');
      assert.equal(kimlik.slogan, 'E2E Slogan');

      // ---------- 3) Geçersiz dosya sessizce yutulmaz ----------
      flash = await saveBrand(page, { mode: 'image' }, { favicon: txt });
      assert.match(flash, /Kısmen kaydedildi|yüklenemeyen/i, 'hata görünür olmalı: ' + flash);
    } finally {
      // ---------- Geri yükleme ----------
      const fields = {
        mode: backup.mode,
        text: backup.text,
        tagline: backup.tagline,
        site_baslik: backup.site_baslik,
        slogan: backup.slogan,
      };
      const files = {};
      if (backup.logoB64) {
        const orig = path.join(os.tmpdir(), 'tb-marka-restore.png');
        fs.writeFileSync(orig, Buffer.from(backup.logoB64, 'base64'));
        files.logo = orig;
      } else {
        fields.remove_logo = true;
      }
      await saveBrand(page, fields, files);
      fs.rmSync(png, { force: true });
      fs.rmSync(txt, { force: true });
    }

    // Geri yükleme sonrası orijinal değerlere dönülmüş olmalı
    const restored = await backupBrand(page);
    assert.equal(restored.mode, backup.mode, 'mod geri yüklenmeli');
    assert.equal(restored.text, backup.text, 'metin logo geri yüklenmeli');
    assert.equal(restored.site_baslik, backup.site_baslik, 'başlık geri yüklenmeli');
    assert.equal(restored.slogan, backup.slogan, 'slogan geri yüklenmeli');
  });
});