'use strict';
/**
 * HTMX panel sözleşmesi — GET + HX-Request yalnız fragment döndürmeli.
 *
 * Veri yazmaz; her panel için:
 *  - HTTP 200
 *  - <!doctype yok
 *  - panel kökü var
 *  - en az bir hx-* özniteliği var
 */

const test = require('node:test');
const assert = require('node:assert/strict');
const { goto, withAdminPage } = require('../lib/harness');

const SID = process.env.TB_SESSION;

const PANELS = [
  ['/admin/menu', 'menu-panel'],
  ['/admin/footer', 'footer-panel'],
  ['/admin/tab-kategorileri', 'tabcats-panel'],
  ['/admin/marka', 'marka-panel'],
  ['/admin/manset-editor', 'manset-panel'],
  ['/admin/bot', 'bot-panel'],
  ['/admin/servisler', 'servisler-panel'],
  ['/admin/raporlar', 'reports-panel'],
  ['/admin/ai-sohbet', 'ai-sohbet-panel'],
  ['/admin/siteler', 'siteler-panel'],
  ['/admin/kumeler', 'kumeler-panel'],
  ['/admin/members', 'members-panel'],
  ['/admin/2fa', '2fa-panel'],
  ['/admin/icerikler/yeni', 'content-form-panel'],
  // NOT: /admin (Pano) BİLİNÇLİ olarak bu listede DEĞİL. Pano HTMX boost'a
  // katılmıyor: yan menüdeki bağlantısı `hx-boost="false"` taşıyor ve tam
  // sayfa dönüyor. Sebep: pano kendi betiklerini/grafiklerini kuruyor ve
  // .admin-main takasında bunlar yeniden çalışmıyordu. Bu, testin altındaki
  // ayrı assertion ile kilitleniyor.
];

test('HTMX panelleri HX-Request ile yalnız fragment döner', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdminPage(async (page) => {
    await goto(page, '/admin');
    assert.ok(!page.url().includes('/admin/giris'), 'admin oturumu açık: ' + page.url());

    for (const [path, panel] of PANELS) {
      const r = await page.evaluate(async (p) => {
        const res = await fetch(p.path, { headers: { 'HX-Request': 'true' } });
        const text = await res.text();
        return {
          status: res.status,
          doctype: /<!doctype/i.test(text),
          panel: text.includes('id="' + p.panel + '"'),
          hx: /hx-(get|post|target|swap|trigger|confirm|indicator)=/.test(text),
          vary: res.headers.get('vary') || '',
          cc: res.headers.get('cache-control') || '',
        };
      }, { path, panel });
      assert.equal(r.status, 200, path + ' HTTP 200 dönmeli');
      assert.equal(r.doctype, false, path + ' tam sayfa döndürmemeli');
      assert.equal(r.panel, true, path + ' panel kökü içermeli');
      assert.equal(r.hx, true, path + ' en az bir hx-* özniteliği içermeli');
      assert.match(r.vary, /HX-Request/i, path + ' Vary: HX-Request başlığı taşımalı');
      assert.match(r.cc, /no-store/i, path + ' Cache-Control: no-store başlığı taşımalı');
    }

    // Pano HTMX boost'a KATILMAZ (kendi betikleri .admin-main takasında
    // yeniden kurulmuyordu): bağlantısı hx-boost="false" olmalı ve tam
    // sayfa dönmeli. Bu, yukarıdaki listeden çıkarılmasının gerekçesidir.
    const pano = await page.evaluate(() => {
      const a = document.querySelector('.ws-menu a[href="/admin"], a[data-ws="pano"]');
      return a ? a.getAttribute('hx-boost') : null;
    });
    assert.equal(pano, 'false', 'Pano bağlantısı hx-boost="false" taşımalı (tam sayfa)');
    const panoYanit = await page.evaluate(async () => {
      const res = await fetch('/admin', { headers: { 'HX-Request': 'true' } });
      const text = await res.text();
      return { doctype: /<!doctype/i.test(text) };
    });
    assert.equal(panoYanit.doctype, true, 'Pano HX-Request ile de tam sayfa dönmeli');
  });
});
