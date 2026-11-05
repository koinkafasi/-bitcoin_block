'use strict';
/**
 * HTMX polisaj davranışları (Faz 95):
 *  1) non-2xx htmx yanıtı → sağ alt flash.error toast görünür
 *  2) panel flash'ları otomatik söner (tbFlashHideMs ile hızlandırılır)
 */

const test = require('node:test');
const assert = require('node:assert/strict');
const { goto, setViewport, withAdminPage } = require('../lib/harness');

const SID = process.env.TB_SESSION;

test('HTMX hata yanıtı global toast gösterir', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok');
  await withAdminPage(async (page) => {
    await setViewport(page, 1400);
    await goto(page, '/admin/bot');
    if (page.url().includes('/admin/giris')) return t.skip('giriş gerekli');

    // TARİHÇE (Faz 210): Test eskiden olmayan bir uca POST atıp gerçek ağ
    // hatası bekliyordu. Bu uygulamada bilinmeyen yol catch-all'a düşüyor
    // (/admin/... GET → 404 HTML sayfası, POST → 405 düz metin) ve HTMX
    // akışı sayfayı yönlendirebiliyor; sonuç testin konusuyla (global hata
    // işleyicisi) ilgisiz bir kırılma oluyordu.
    // Konu global işleyici olduğu için olay doğrudan, gerçek htmx olay
    // biçimiyle tetikleniyor: dinleyici + toast + metin doğrulanır.
    const otobusVar = await page.evaluate(() => !!window.tbToast);
    assert.equal(otobusVar, true, 'global toast fonksiyonu (tbToast) yüklü olmalı');

    await page.evaluate(() => {
      document.body.dispatchEvent(new CustomEvent('htmx:responseError', {
        bubbles: true,
        detail: { xhr: { status: 500 } },
      }));
    });
    await page.waitForFunction(() => {
      return [...document.querySelectorAll('.flash.error')].some((el) => /Sunucu hatası/.test(el.textContent));
    }, { timeout: 10000 });
    const toast = await page.evaluate(() => {
      const el = [...document.querySelectorAll('.flash.error')].find((x) => /Sunucu hatası/.test(x.textContent));
      return el ? el.textContent.trim() : '';
    });
    assert.match(toast, /Sunucu hatası/, "hata toast'u görünmeli");
    assert.match(toast, /500/, 'hata kodu toast metnine yansımalı: ' + toast);

    // Ağ hatası ayrı mesaj üretmeli (sendError → "Ağ hatası").
    await page.evaluate(() => {
      document.body.dispatchEvent(new CustomEvent('htmx:sendError', { bubbles: true, detail: {} }));
    });
    await page.waitForFunction(() => {
      return [...document.querySelectorAll('.flash.error')].some((el) => /Ağ hatası/.test(el.textContent));
    }, { timeout: 5000 });
  });
});

test('panel flash otomatik söner', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok');
  await withAdminPage(async (page) => {
    await setViewport(page, 1400);
    await goto(page, '/admin/servisler');
    if (page.url().includes('/admin/giris')) return t.skip('giriş gerekli');
    await page.evaluate(() => { window.tbFlashHideMs = 400; });
    await page.click('form[action="/admin/servisler/yenile"] button');
    await page.waitForFunction(() => {
      const f = document.querySelector('.admin-body .flash');
      return f && /Yenile/i.test(f.textContent);
    }, { timeout: 15000 });
    await page.waitForFunction(() => {
      return !document.querySelector('.admin-body .flash');
    }, { timeout: 8000 });
    const kalan = await page.evaluate(() => !!document.querySelector('.admin-body .flash'));
    assert.equal(kalan, false, 'flash sönmüş olmalı');
  });
});

test('HTMX swap sonrası odak tetikleyiciye geri döner', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok');
  await withAdminPage(async (page) => {
    await setViewport(page, 1400);
    await goto(page, '/admin/menu');
    if (page.url().includes('/admin/giris')) return t.skip('giriş gerekli');
    await page.evaluate(() => {
      const b = document.getElementById('menuSave');
      if (b) b.focus();
    });
    const before = await page.evaluate(() => (document.activeElement || {}).id || '');
    assert.equal(before, 'menuSave', 'odak öncesi kaydet düğmesinde');
    await page.click('#menuSave');
    await page.waitForFunction(() => {
      const f = document.querySelector('.flash');
      return f && /Menü kaydedildi/.test(f.textContent);
    }, { timeout: 15000 });
    await new Promise((r) => setTimeout(r, 700));
    const after = await page.evaluate(() => (document.activeElement || {}).id || '');
    assert.equal(after, 'menuSave', 'swap sonrası odak kaydet düğmesine dönmeli');
  });
});
