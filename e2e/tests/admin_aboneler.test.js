'use strict';
/**
 * Aboneler merkezi (denetim açık maddesinin kapatılması):
 *
 * - Bülten + gazete abone listeleri tek sayfada (çoğalma toparlandı).
 * - Ekleme, durum değiştirme (aktif/iptal) ve KVKK silme her iki liste için.
 * - Durum filtresi çalışır.
 * - Test kendi kayıtlarını oluşturur ve sonunda siler.
 */

const test = require('node:test');
const assert = require('node:assert/strict');
const { goto, withAdminPage } = require('../lib/harness');

async function post(page, path, data) {
  return page.evaluate(async ({ path, data }) => {
    const fd = new FormData();
    Object.entries(data).forEach(([k, v]) => fd.append(k, v));
    const r = await fetch(path, { method: 'POST', body: fd, credentials: 'same-origin' });
    const html = await r.text();
    // HTMX outerHTML taklidi: dönen paneli DOM'a uygula (aksi halde DOM bayatlar)
    const tmp = document.createElement('div');
    tmp.innerHTML = html;
    const fresh = tmp.querySelector('#aboneler-panel');
    if (fresh) document.getElementById('aboneler-panel').replaceWith(fresh);
    return { status: r.status, body: html };
  }, { path, data });
}

// Tabloda e-posta satırını bulur ve verilen işlemi (durum|sil) fetch ile yapar.
async function rowAction(page, tableIndex, email, actionPart) {
  return page.evaluate(async ({ tableIndex, email, actionPart }) => {
    const tables = document.querySelectorAll('#aboneler-panel table');
    const rows = [...(tables[tableIndex] ? tables[tableIndex].querySelectorAll('tbody tr') : [])];
    const row = rows.find((r) => r.textContent.includes(email));
    if (!row) return null;
    const f = [...row.querySelectorAll('form')].find((x) => x.action.includes(actionPart));
    if (!f) return null;
    const fd = new FormData(f);
    const r = await fetch(f.action, { method: 'POST', body: fd, credentials: 'same-origin' });
    const html = await r.text();
    const tmp = document.createElement('div');
    tmp.innerHTML = html;
    const fresh = tmp.querySelector('#aboneler-panel');
    if (fresh) document.getElementById('aboneler-panel').replaceWith(fresh);
    return r.status;
  }, { tableIndex, email, actionPart });
}

test('aboneler merkezi: bülten + gazete birleşik yönetim', async () => {
  await withAdminPage(async (page) => {
    const BUL = 'e2e-abone-bul@example.com';
    const GAZ = 'e2e-abone-gaz@example.com';

    // ---------- Sayfa yapısı ----------
    await goto(page, '/admin/aboneler');
    const yapi = await page.evaluate(() => ({
      panel: !!document.getElementById('aboneler-panel'),
      kpi: document.querySelectorAll('#aboneler-panel .kpi').length,
      tables: document.querySelectorAll('#aboneler-panel table').length,
    }));
    assert.ok(yapi.panel, 'aboneler paneli yok');
    assert.ok(yapi.kpi >= 4, 'özet kartları olmalı: ' + yapi.kpi);
    assert.equal(yapi.tables, 2, 'iki liste tablosu olmalı (bülten + gazete)');

    // ---------- Bülten abonesi ekle → durum değiştir → sil ----------
    let r = await post(page, '/admin/aboneler/bulten', { email: BUL });
    assert.match(r.body, /Bülten abonesi eklendi/);
    let satir = await page.evaluate((email) => {
      const t = document.querySelectorAll('#aboneler-panel table')[0];
      return t && t.textContent.includes(email);
    }, BUL);
    assert.ok(satir, 'bülten abonesi listede görünmeli');

    // Yeni abone "onay bekliyor" başlar: ilk dokunuş → aktif, ikincisi → iptal
    await rowAction(page, 0, BUL, '/durum');
    satir = await page.evaluate((email) => {
      const t = document.querySelectorAll('#aboneler-panel table')[0];
      const row = [...t.querySelectorAll('tbody tr')].find((r) => r.textContent.includes(email));
      return row ? row.textContent.includes('aktif') : null;
    }, BUL);
    assert.ok(satir, 'ilk dokunuşta durum "aktif" olmalı');

    await rowAction(page, 0, BUL, '/durum');
    satir = await page.evaluate((email) => {
      const t = document.querySelectorAll('#aboneler-panel table')[0];
      const row = [...t.querySelectorAll('tbody tr')].find((r) => r.textContent.includes(email));
      return row ? row.textContent.includes('iptal') : null;
    }, BUL);
    assert.ok(satir, 'ikinci dokunuşta durum "iptal" olmalı');

    // ---------- Gazete abonesi ekle → sil ----------
    r = await post(page, '/admin/aboneler/gazete', {
      email: GAZ, kanal: 'email', siklik: 'haftalik',
    });
    assert.match(r.body, /Gazete abonesi eklendi/);
    satir = await page.evaluate((email) => {
      const t = document.querySelectorAll('#aboneler-panel table')[1];
      return t && t.textContent.includes(email);
    }, GAZ);
    assert.ok(satir, 'gazete abonesi listede görünmeli');

    // ---------- Durum filtresi ----------
    await goto(page, '/admin/aboneler?durum=aktif');
    const filtre = await page.evaluate(() => {
      const t = document.querySelectorAll('#aboneler-panel table')[0];
      return t ? t.textContent.includes('e2e-abone-bul@example.com') : null;
    });
    assert.ok(!filtre, 'aktif filtresi iptal kaydı göstermemeli');

    // ---------- Temizlik (KVKK silme) ----------
    await goto(page, '/admin/aboneler');
    await rowAction(page, 0, BUL, '/sil');
    await rowAction(page, 1, GAZ, '/sil');
    const kalan = await page.evaluate((e1, e2) => {
      const t = document.getElementById('aboneler-panel');
      return t ? (t.textContent.includes(e1) || t.textContent.includes(e2)) : true;
    }, BUL, GAZ);
    assert.ok(!kalan, 'test kayıtları silinmeli');
  });
});