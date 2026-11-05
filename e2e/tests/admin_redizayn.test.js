'use strict';
/**
 * Admin redizayn (WiseCP tarzı) — Faz A–E:
 *
 * - Pano: KPI kartları + sparkline, aktivite, yayın şeridi.
 * - Görünüm: renk/font/navbar ayarı kaydedilir ve CSS'e enjekte edilir (sonra sıfırlanır).
 * - Navigasyon: breadcrumb + katlanabilir menü grupları + sabitleme.
 * - Tablo araçları: arama filtresi.
 * - Komut paleti: eylem arama + kısayol yardımı.
 *
 * TARİHÇE (Faz 210 triyajı): Bu test WiseCP redizaynında (5522b55) yazıldı.
 * O günkü pano `html/template` katmanındaydı ve şu sınıfları üretiyordu:
 * `.kpi`, `.qtiles .qtile`, `.auto-list .auto-row`, `.onboard .ob-item`,
 * `.spark`. Sonra pano templ'e taşındı (22fe4ac), tek kabuğa geçti (a151e90)
 * ve Pano 2.0 ile yeniden yazıldı (15f4963); ölü html/template katmanı
 * silindi (e934837). Bugünkü karşılıklar:
 *
 *   .kpi       → .pn-kpi        (KPI döşemesi, aynı işlev)
 *   .spark     → .pn-spark      (sparkline, aynı işlev)
 *   .qtiles    → /admin/ayarlar  (hızlı eylem kutucukları oraya taşındı)
 *   .auto-list → kaldırıldı     (otomasyon paneli panodan çıktı; işlevler
 *                                /admin/paylasim ve /admin/gazete#otomasyon'da)
 *   .onboard   → kaldırıldı     (başlangıç listesi panodan çıktı)
 *
 * Kaldırılan iki panel için assertion SİLİNDİ; yerine panonun gerçekten
 * çizdiği yapı (yayın şeridi + aktivite) kontrol ediliyor. Böylece "pano
 * boş geldi" regresyonu hâlâ yakalanıyor.
 */

const test = require('node:test');
const assert = require('node:assert/strict');
const { BASE_URL, goto, withAdminPage } = require('../lib/harness');

test('admin redizayn: pano, görünüm, navigasyon, tablolar, cmdk', async () => {
  await withAdminPage(async (page) => {
    // ---------- Pano ----------
    await goto(page, '/admin');
    assert.ok(await page.$('.pn-kpi'), 'KPI kartları yok');
    // Pano gerçekten içerik çiziyor mu: aktivite akışı + grafik kutuları.
    // (Eski testin .auto-list/.onboard kontrolleri kaldırıldı; bu ikisi
    // panonun boş gelmesini yakalayan asıl sinyaller.)
    assert.ok(await page.$('.pn-akt'), 'pano aktivite akışı çizilmemiş');
    assert.ok(await page.$('.pn-chartbox'), 'pano grafik kutuları çizilmemiş');
    const spark = await page.$$('.pn-spark');
    assert.ok(spark.length > 0, 'sparkline grafik yok');

    // Breadcrumb dolu olmalı
    const crumb = await page.evaluate(() => document.getElementById('adminCrumb').textContent.trim());
    assert.ok(crumb.includes('Pano'), 'breadcrumb: ' + crumb);

    // Sabitleme (📌): ilk alanı sabitle → "Sabitlenmişler" grubu oluşmalı
    const pinned = await page.evaluate(() => {
      const pin = document.querySelector('.ws-menu .pin-btn');
      if (!pin) return { ok: false };
      pin.click();
      return {
        ok: !!document.querySelector('.pinned-menu'),
        count: document.querySelectorAll('.ws-menu .pin-btn').length,
      };
    });
    assert.ok(pinned.ok, 'sabitleyen öğe "Sabitlenmişler" grubuna girmeli');
    assert.ok(pinned.count >= 11, 'tüm alanlarda pin düğmesi olmalı');
    // geri bırak
    await page.evaluate(() => {
      const p = document.querySelector('.pinned-menu .pin-btn') || document.querySelector('.ws-menu .pin-btn.pinned');
      if (p) p.click();
    });

    // ---------- Hızlı eylem kutucukları (artık Ayarlar'da) ----------
    await goto(page, '/admin/ayarlar');
    assert.ok(await page.$('.qtiles .qtile'), 'hızlı eylem kutucukları yok');

    // ---------- Görünüm (marka kiti) ----------
    await goto(page, '/admin/gorunum');
    assert.ok(await page.$('#gorunumForm'), 'görünüm formu yok');
    assert.ok(await page.$('#gorunumPreview'), 'canlı önizleme yok');
    await page.evaluate(() => {
      const f = document.getElementById('gorunumForm');
      f.font.value = 'Poppins';
      f.primary.value = '#2244aa';
    });
    const saved = await page.evaluate(async () => {
      const f = document.getElementById('gorunumForm');
      const fd = new FormData(f);
      fd.set('font', 'Poppins');
      fd.set('primary', '#2244aa');
      const r = await fetch('/admin/gorunum', { method: 'POST', body: fd, credentials: 'same-origin' });
      return r.status;
    });
    assert.equal(saved, 200);
    await goto(page, '/admin');
    const css = await page.evaluate(() => {
      const st = [...document.querySelectorAll('style')].map((s) => s.textContent).join('\n');
      return { accent: st.includes('#2244aa'), font: st.includes('Poppins') };
    });
    assert.ok(css.accent, 'ana renk CSS enjekte edilmeli');
    assert.ok(css.font, 'font CSS enjekte edilmeli');

    // Yatay navbar seçeneği kaydedilebilmeli (test amaçlı dikeyde bırak)
    const navSaved = await page.evaluate(async () => {
      const fd = new FormData();
      fd.set('navbar', 'yatay');
      fd.set('font', 'Poppins');
      fd.set('primary', '#2244aa');
      const r = await fetch('/admin/gorunum', { method: 'POST', body: fd, credentials: 'same-origin' });
      return r.status;
    });
    assert.equal(navSaved, 200);
    await goto(page, '/admin');
    assert.ok(await page.evaluate(() => document.body.classList.contains('admin-hnav')), 'yatay navbar sınıfı gelmeli');

    // Varsayılana dön (test kalıntısı bırakma)
    await page.evaluate(async () => {
      const fd = new FormData();
      fd.set('sifirla', '1');
      await fetch('/admin/gorunum', { method: 'POST', body: fd, credentials: 'same-origin' });
    });
    await goto(page, '/admin');
    const reset = await page.evaluate(() => {
      const st = [...document.querySelectorAll('style')].map((s) => s.textContent).join('\n');
      return !st.includes('#2244aa');
    });
    assert.ok(reset, 'sıfırlama sonrası özel renk kalmamalı');

    // ---------- Tablo araçları ----------
    await goto(page, '/admin/icerikler');
    const tools = await page.evaluate(() => {
      const t = document.querySelector('.tbl-tools');
      return !!t;
    });
    assert.ok(tools, 'tablo araç çubuğu yok');
    const filtered = await page.evaluate(() => {
      const search = document.querySelector('.tbl-tools input');
      if (!search) return -1;
      search.value = 'zzz-olmayan-arama';
      search.dispatchEvent(new Event('input'));
      const rows = [...document.querySelectorAll('.admin-body table.table tbody tr')]
        .filter((r) => r.style.display !== 'none');
      return rows.length;
    });
    assert.equal(filtered, 0, 'arama filtresi satırları gizlemeli');

    // ---------- Cmdk: eylem arama ----------
    await page.keyboard.down('Control');
    await page.keyboard.press('KeyK');
    await page.keyboard.up('Control');
    const cmdkOpen = await page.evaluate(() => document.getElementById('cmdk').classList.contains('open'));
    assert.ok(cmdkOpen, 'Ctrl+K komut paleti açılmadı');
    await page.type('#cmdkInput', 'gazete');
    const items = await page.evaluate(() =>
      [...document.querySelectorAll('#cmdkList li')].map((li) => li.textContent));
    assert.ok(items.some((t) => t.includes('Gazete')), 'cmdk gazete sonucu yok: ' + items.join('|'));
    await page.keyboard.press('Escape');
    await page.evaluate(() => document.activeElement && document.activeElement.blur());

    // ---------- Kısayol yardımı (?) ----------
    await page.keyboard.press('?');
    const helpOpen = await page.evaluate(() =>
      [...document.querySelectorAll('.ui-modal')].some((m) => m.classList.contains('open')));
    assert.ok(helpOpen, '? kısayol penceresi açılmadı');
    await page.keyboard.press('Escape');
  });
});