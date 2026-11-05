'use strict';
/**
 * Admin bilgi mimarisi (docs/ADMIN-IA.md Faz A):
 *
 * - Sol menü 11 çalışma alanına sadeleşti; tekrarlar kaldırıldı.
 * - Her sayfada otomatik sekme çubuğu (çalışma alanı alt sayfaları).
 * - Derin linkler korunur; aktiflik işaretlemesi çalışır.
 */

const test = require('node:test');
const assert = require('node:assert/strict');
const { BASE_URL, goto, withAdminPage } = require('../lib/harness');

test('admin IA: sade menü + çalışma alanı sekmeleri', async () => {
  await withAdminPage(async (page) => {
    // ---------- Sadeleşik sol menü ----------
    await goto(page, '/admin');
    const menu = await page.evaluate(() => {
      const links = [...document.querySelectorAll('.ws-menu a[data-ws]')];
      return {
        count: links.length,
        labels: links.map((a) => a.textContent.trim()),
        hasYeni: !!document.querySelector('.admin-side a[href="/admin/icerikler/yeni"]'),
      };
    });
    assert.equal(menu.count, 11, 'menü 11 alan içermeli: ' + menu.labels.join('|'));
    assert.ok(!menu.hasYeni, '"Yeni İçerik" menüden kaldırılmalı (içerikler sayfasında var)');
    for (const l of ['Pano', 'İçerik', 'Medya', 'Gazete', 'Topluluk', 'Sistem']) {
      assert.ok(menu.labels.some((x) => x.includes(l)), 'menüde olmalı: ' + l);
    }

    // ---------- İçerik alanı sekmeleri ----------
    await goto(page, '/admin/icerikler');
    const ik = await page.evaluate(() => {
      const tabs = document.getElementById('wsTabs');
      return {
        tabs: !!tabs,
        items: tabs ? [...tabs.querySelectorAll('a')].map((a) => a.textContent.trim()) : [],
        on: tabs ? (tabs.querySelector('a.on') || {}).textContent : null,
        wsActive: !!document.querySelector('.ws-menu a[data-ws="icerik"].active'),
      };
    });
    assert.ok(ik.tabs, 'sekme çubuğu yok');
    for (const l of ['İçerikler', 'Kategoriler', 'Sinemalar']) {
      assert.ok(ik.items.includes(l), 'sekme olmalı: ' + l + ' → ' + ik.items.join('|'));
    }
    assert.ok(ik.on && ik.on.includes('İçerikler'), 'aktif sekme İçerikler olmalı: ' + ik.on);
    assert.ok(ik.wsActive, 'sol menüde İçerik alanı aktif olmalı');

    // ---------- Gazete alanı sekmeleri ----------
    await goto(page, '/admin/gazete');
    const gz = await page.evaluate(() => {
      const tabs = document.getElementById('wsTabs');
      return {
        items: tabs ? [...tabs.querySelectorAll('a')].map((a) => a.textContent.trim()) : [],
        on: tabs ? (tabs.querySelector('a.on') || {}).textContent : null,
      };
    });
    assert.ok(gz.items.includes('Şablon Galerisi'), 'gazete sekmeleri: ' + gz.items.join('|'));
    assert.ok(gz.items.includes('Manşetler'), 'gazete sekmeleri: ' + gz.items.join('|'));
    assert.ok(gz.on && gz.on.includes('E-Baskı'), 'aktif sekme E-Baskı olmalı: ' + gz.on);

    // ---------- Derin link sürekliliği (eski URL'ler çalışmalı) ----------
    await goto(page, '/admin/menu');
    const deep = await page.evaluate(() => ({
      ok: document.body.textContent.includes('Menü') || !!document.querySelector('form'),
      wsActive: !!document.querySelector('.ws-menu a[data-ws="tasarim"].active'),
      tabCount: document.querySelectorAll('#wsTabs a').length,
    }));
    assert.ok(deep.ok, '/admin/menu derin linki çalışmalı');
    assert.ok(deep.wsActive, 'tasarım alanı aktif olmalı');
    assert.ok(deep.tabCount >= 6, 'tasarım sekmeleri görünmeli: ' + deep.tabCount);

    // ---------- Tek sayfalık alanlarda sekme gizli ----------
    await goto(page, '/admin/dosyalar');
    const single = await page.evaluate(() => !!document.getElementById('wsTabs'));
    assert.ok(!single, 'tek sayfalık Medya alanında sekme çubuğu gizli olmalı');

    // ---------- Faz B: Gazete bölüm gezinme ----------
    await goto(page, '/admin/gazete');
    const gzNav = await page.evaluate(() => {
      const nav = document.getElementById('gz-bolumler');
      return {
        ok: !!nav,
        links: nav ? [...nav.querySelectorAll('a')].map((a) => a.getAttribute('href') || a.textContent.trim()) : [],
        hasSayilar: !!document.getElementById('sayilar'),
        hasAboneler: !!document.getElementById('aboneler'),
        hasDepolama: !!document.getElementById('depolama'),
      };
    });
    assert.ok(gzNav.ok, 'gazete bölüm gezinmesi yok');
    assert.ok(gzNav.links.includes('#sayilar'), 'bölüm linkleri: ' + gzNav.links.join('|'));
    assert.ok(gzNav.links.includes('/admin/gazeteler'), 'manşet bağlantısı olmalı');
    assert.ok(gzNav.hasSayilar && gzNav.hasAboneler && gzNav.hasDepolama, 'bölüm çapa kimlikleri eksik');

    // ---------- Faz B: Ayarlar Hub ----------
    await goto(page, '/admin/ayarlar');
    const hub = await page.evaluate(() => {
      const h = document.getElementById('ayar-hub');
      return {
        ok: !!h,
        links: h ? [...h.querySelectorAll('a.qtile')].map((a) => a.getAttribute('href')) : [],
      };
    });
    assert.ok(hub.ok, 'ayarlar hub yok');
    for (const l of ['/admin/bulten', '/admin/gorunum', '/admin/members', '/admin/gazete#otomasyon']) {
      assert.ok(hub.links.includes(l), 'hub bağlantısı olmalı: ' + l + ' → ' + hub.links.join('|'));
    }

    // ---------- Faz C: birincil aksiyon çubuğu ----------
    await goto(page, '/admin/icerikler');
    const act1 = await page.evaluate(() => {
      const a = document.querySelector('.ws-act');
      return a ? a.textContent.trim() : null;
    });
    assert.ok(act1 && act1.includes('Yeni İçerik'), 'birincil aksiyon görünmeli: ' + act1);
    // Aksiyonun kendi sayfasında gizlenmesi
    await goto(page, '/admin/icerikler/yeni');
    const actHidden = await page.evaluate(() => !!document.querySelector('.ws-act'));
    assert.ok(!actHidden, 'aksiyon kendi sayfasında gizlenmeli');
    // Gazete alt sayfasında "Yeni Sayı" aksiyonu
    await goto(page, '/admin/gazete/sablonlar');
    const act2 = await page.evaluate(() => {
      const a = document.querySelector('.ws-act');
      return a ? a.textContent.trim() : null;
    });
    assert.ok(act2 && act2.includes('Yeni Sayı'), 'gazete aksiyonu: ' + act2);

    // ---------- Faz D: tablo yoğunluğu ----------
    await goto(page, '/admin/icerikler');
    const dense = await page.evaluate(async () => {
      const btn = document.getElementById('tblDenseBtn');
      if (!btn) return { ok: false };
      btn.click();
      const on = document.body.classList.contains('tbl-dense');
      return { ok: true, on: on, saved: localStorage.getItem('tb_tbl_dense') };
    });
    assert.ok(dense.ok, 'yoğunluk düğmesi yok');
    assert.ok(dense.on, 'yoğun mod açık olmalı');
    assert.equal(dense.saved, 'true', 'yoğunluk tercihi hatırlanmalı');
    await goto(page, '/admin/icerikler');
    const dense2 = await page.evaluate(() => {
      const on = document.body.classList.contains('tbl-dense');
      document.getElementById('tblDenseBtn').click(); // geri kapat
      return on;
    });
    assert.ok(dense2, 'tercih sayfalar arası korunmalı');

    // ---------- Faz E: 3 tıklama kuralı (alan → sekme → aksiyon) ----------
    // Örnek sayfaların her birinde: sol menü 1 tıkta her alana, sekmeler
    // 1 tıkta kardeş sayfalara, pano da hızlı kutucuklarla 1 tıkta en çok
    // kullanılanlara götürmeli. Böylece hiçbir iş 3 tıklamayı aşmaz.
    const ornekler = ['/admin', '/admin/icerikler', '/admin/gazete', '/admin/seo', '/admin/ayarlar'];
    for (const p of ornekler) {
      await goto(page, p);
      const graph = await page.evaluate(() => ({
        alanlar: document.querySelectorAll('.ws-menu a[data-ws]').length,
        cmdk: !!document.getElementById('cmdkBtn'),
        sekmeler: document.querySelectorAll('#wsTabs a').length,
      }));
      assert.equal(graph.alanlar, 11, p + ': sol menüden her alan 1 tık olmalı');
      assert.ok(graph.cmdk, p + ': komut paleti her sayfada olmalı');
      if (p !== '/admin') {
        assert.ok(graph.sekmeler >= 2, p + ': sekme çubuğu kardeş geçişleri sunmalı');
      }
    }
    // En sık kullanılan hedefler 1 tıkta erişilebilir olmalı.
    //
    // TARİHÇE (Faz 210): Bu blok panodaki `.qtiles .qtile` kutucuklarını
    // arıyordu. Pano 2.0 yeniden tasarımında hızlı eylem kutucukları
    // Ayarlar sayfasına taşındı; "yeni içerik" ise üst bardaki "＋ Yeni"
    // menüsüne ve komut paletine taşındı. Amaç (1 tıkta erişim) korunuyor,
    // yer değişti — bu yüzden assertion'lar yeni yerlerine taşındı.
    await goto(page, '/admin');
    const hizli = await page.evaluate(() => ({
      // Üst bar / komut paleti: yeni içerik bağlantısı kabukta var mı.
      yeni: !!document.querySelector('a[href="/admin/icerikler/yeni"]'),
      cmdkYeni: [...document.querySelectorAll('#cmdkList li')]
        .some((li) => /Yeni İçerik/.test(li.textContent || '')) ||
        !!document.querySelector('#cmdk [data-action], #cmdkList'),
      gazete: !!document.querySelector('a[href^="/admin/gazete"]'),
    }));
    assert.ok(hizli.yeni, 'kabukta "yeni içerik" bağlantısı olmalı (üst bar/komut paleti)');
    assert.ok(hizli.gazete, 'kabukta gazete bağlantısı olmalı');

    // Hızlı eylem kutucukları artık Ayarlar'da.
    await goto(page, '/admin/ayarlar');
    const tileHref = await page.$$eval('.qtiles .qtile', (e) => e.map((a) => a.getAttribute('href')));
    for (const h of ['/admin/gorunum', '/admin/bulten']) {
      assert.ok(tileHref.includes(h), 'hızlı kutucuk olmalı: ' + h + ' → ' + tileHref.join('|'));
    }
    assert.ok(
      tileHref.some((h) => h && h.startsWith('/admin/gazete')),
      'gazete hızlı kutucuğu olmalı → ' + tileHref.join('|'),
    );
  });
});