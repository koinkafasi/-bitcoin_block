'use strict';
/**
 * Yazı sayfası üst araçları: dil / kaydet, kırıntı yolundaki kategori takip (+),
 * font kontrolleri, görüntülenme ve hover ile açılan Google Material ikonlu paylaşım menüsü.
 */

const test = require('node:test');
const assert = require('node:assert/strict');
const { goto, setViewport, withAdminPage } = require('../lib/harness');

const SID = process.env.TB_SESSION;

test('yazı sayfası üst araçları ve hover paylaşım menüsü', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdminPage(async (page, errors) => {
    await setViewport(page, 1536, 1000);
    await goto(page, '/');
    const href = await page.evaluate(() => {
      const a = [...document.querySelectorAll('a[href]')].find((x) =>
        /\/[^/]+-[0-9a-f]{4,8}-\d+$/.test(x.getAttribute('href') || '')
      );
      return a ? a.getAttribute('href') : null;
    });
    assert.ok(href, 'makale bağlantısı bulunamadı');
    await goto(page, href);

    const before = await page.evaluate(() => {
      const box = (sel) => {
        const el = document.querySelector(sel);
        if (!el) return null;
        const r = el.getBoundingClientRect();
        return { x: Math.round(r.x), y: Math.round(r.y), w: Math.round(r.width), h: Math.round(r.height) };
      };
      const menu = document.querySelector('.art-share-menu');
      return {
        path: location.pathname,
        canonical: (document.querySelector('head link[rel="canonical"]') || {}).href || '',
        langSwitcher: document.querySelectorAll('.art-topbar .lang-switch').length,
        langLinks: document.querySelectorAll('.art-topbar .lang-switch a').length,
        headHreflangs: [...document.querySelectorAll('head link[rel="alternate"][hreflang]')]
          .map((l) => l.getAttribute('hreflang')),
        save: box('.article > #bmBtn'), hero: box('.article .hero'),
        article: box('.article'),
        saveFrame: (() => {
          const el = document.querySelector('.article > #bmBtn');
          if (!el) return null;
          const cs = getComputedStyle(el);
          return { border: cs.borderTopWidth + ' ' + cs.borderTopStyle, bg: cs.backgroundColor };
        })(),
        topbarSaves: document.querySelectorAll('.art-user-actions #bmBtn').length,
        savePressed: document.querySelector('#bmBtn') ? document.querySelector('#bmBtn').getAttribute('aria-pressed') : null,
        cat: box('.breadcrumb .bc-cat a'), follow: box('.breadcrumb .bc-cat #flBtn'),
        tools: box('.art-tools'), share: box('.art-share'),
        fontBtns: document.querySelectorAll('.art-tools [data-font]').length,
        topbarFollows: document.querySelectorAll('.art-user-actions #flBtn').length,
        followPressed: document.querySelector('#flBtn') ? document.querySelector('#flBtn').getAttribute('aria-pressed') : null,
        followAuth: document.querySelector('#flBtn') ? document.querySelector('#flBtn').dataset.auth : null,
        toolbarViews: document.querySelectorAll('.art-views').length,
        audioPlayer: document.querySelectorAll('#artAudio').length,
        audioNote: (document.getElementById('ttsNote') || {}).textContent || '',
        metaViews: box('.art-meta-views'),
        metaEye: !!document.querySelector('.art-meta-views .art-eye svg'),
        shareItems: menu ? menu.querySelectorAll('.art-share-item').length : 0,
        shareIcons: menu ? menu.querySelectorAll('.art-share-ico svg').length : 0,
        menuWidth: menu ? Math.round(menu.getBoundingClientRect().width) : -1,
      };
    });

    assert.ok(before.cat && before.follow, 'kırıntı yolu öğeleri bulunmalı');
    assert.ok(before.save, 'Kaydet düğmesi bulunmalı');
    assert.ok(before.article, 'yazı kartı bulunmalı');
    assert.equal(before.topbarSaves, 0, 'üst çubukta ikinci kaydet düğmesi olmamalı');
    assert.ok(['true', 'false'].includes(before.savePressed), 'Kaydet aria-pressed taşımalı');
    const saveRightGap = before.article.x + before.article.w - (before.save.x + before.save.w);
    assert.ok(saveRightGap >= 0 && saveRightGap <= 28, 'Kaydet yazı kartının sağ üst köşesinde olmalı');
    assert.ok(before.save.y - before.article.y <= 28, 'Kaydet kartın üst kısmında olmalı');
    assert.equal(before.saveFrame.border.split(' ')[1], 'none', 'Kaydet çerçevesiz olmalı');
    assert.equal(before.saveFrame.bg, 'rgba(0, 0, 0, 0)', 'Kaydet arka plansız olmalı');
    assert.equal(before.topbarFollows, 0, 'resmin üstünde kategori takip düğmesi kalmamalı');
    assert.ok(before.follow.x > before.cat.x, 'takip (+) kategori bağlantısının sağında olmalı');
    assert.ok(before.follow.x < before.save.x, 'takip (+) sağ üstteki kaydet ikonunun solunda olmalı');
    assert.equal(before.followAuth, '1', 'oturumlu kullanıcıda takip düğmesi aktif olmalı');
    assert.ok(['true', 'false'].includes(before.followPressed), 'takip düğmesi aria-pressed taşımalı');
    assert.equal(before.audioPlayer, 1, 'sesli haber oynatıcısı olmalı');
    assert.ok(before.audioNote.length > 0, 'oynatıcı durum notu olmalı');
    assert.equal(before.fontBtns, 3, 'üç yazı boyutu düğmesi olmalı');
    assert.equal(before.toolbarViews, 0, 'görsel altında ikinci görüntülenme olmamalı');
    assert.ok(before.metaViews, 'üst bilgi satırında görüntülenme olmalı');
    assert.equal(before.metaEye, true, 'üst görüntülenmede göz ikonu olmalı');
    assert.ok(before.tools.x < before.share.x, 'font kontrolleri paylaşımın solunda olmalı');
    assert.equal(before.shareItems, 8, 'altı sosyal + kopyala + cihazda paylaş olmalı');
    assert.equal(before.shareIcons, 8, 'paylaşım menüsü marka SVG ikonları taşımalı');
    assert.equal(before.menuWidth, 0, 'sürgü başlangıçta kapalı olmalı');

    await page.click('.art-share-btn');
    await new Promise((r) => setTimeout(r, 400));
    const after = await page.evaluate(() => {
      const m = document.querySelector('.art-share-menu');
      const cs = getComputedStyle(m);
      return {
        visibility: cs.visibility, opacity: cs.opacity,
        position: cs.position, background: cs.backgroundColor, border: cs.borderTopWidth,
        height: Math.round(m.getBoundingClientRect().height),
        width: Math.round(m.getBoundingClientRect().width),
        btn: (() => {
          const b = document.querySelector('.art-share-btn').getBoundingClientRect();
          return { x: Math.round(b.x), y: Math.round(b.y), w: Math.round(b.width) };
        })(),
        items: [...m.querySelectorAll('.art-share-item')].map((el) => {
          const r = el.getBoundingClientRect();
          return { x: Math.round(r.x), y: Math.round(r.y) };
        }),
      };
    });
    assert.equal(after.visibility, 'visible', 'hover ile paylaşım menüsü açılmalı');
    assert.equal(after.opacity, '1');
    assert.ok(after.height > 30 && after.height < 100, 'menü tek sıra yüksekliğinde olmalı');
    assert.equal(after.position, 'static', 'sürgü akış içinde açılmalı (kaplama değil)');
    assert.equal(after.background, 'rgba(0, 0, 0, 0)', 'sürgünün arka planı olmamalı');
    assert.equal(after.border, '0px', 'sürgünün çerçevesi olmamalı');
    assert.equal(after.items.length, 8, 'sekiz paylaşım öğesi olmalı');
    assert.equal(new Set(after.items.map((i) => i.y)).size, 1, 'menü tek sıra olmalı');
    const xs = after.items.map((i) => i.x);
    assert.deepEqual(xs, [...xs].sort((a, b) => a - b), 'öğeler soldan sağa sıralı olmalı');
    assert.ok(after.items[after.items.length - 1].x < after.btn.x,
      'şerit butonun solunda açılmalı');
    assert.ok(after.width > 250, 'şerit yatay tek sıra genişliğinde olmalı');

    // Kırıntı yolundaki + düğmesi takibi açar/kapatır.
    await page.click('.breadcrumb .bc-follow');
    await page.waitForFunction(
      (v) => document.querySelector('.breadcrumb .bc-follow').getAttribute('aria-pressed') !== v,
      {}, before.followPressed
    );
    const toggled = await page.$eval('.breadcrumb .bc-follow', (el) => el.getAttribute('aria-pressed'));
    assert.notEqual(toggled, before.followPressed, 'tıklama takip durumunu değiştirmeli');
    assert.equal(await page.$eval('.breadcrumb .bc-follow', (el) => el.classList.contains('on')), toggled === 'true');
    // Test sonrası durumu geri al.
    await page.click('.breadcrumb .bc-follow');
    await page.waitForFunction(
      (v) => document.querySelector('.breadcrumb .bc-follow').getAttribute('aria-pressed') === v,
      {}, before.followPressed
    );

    // Self-canonical: mutlak ve sayfanın kendi (dil-farkında) adresi olmalı.
    assert.ok(before.canonical.startsWith('http'), 'canonical mutlak adres olmalı');
    assert.equal(new URL(before.canonical).pathname, before.path, 'canonical kendi adresini göstermeli');

    // Dil blokları: tek dilde gizli ve hreflang yok; çok dilde ikisi birlikte.
    if (before.langSwitcher === 0) {
      assert.equal(before.headHreflangs.length, 0, 'tek dilde dil bloğu ve hreflang basılmamalı');
    } else {
      assert.equal(before.headHreflangs.length, before.langLinks + 2, 'her dil + x-default için hreflang olmalı');
      assert.ok(before.headHreflangs.includes('x-default'), 'x-default hreflang bulunmalı');
      const codes = await page.$$eval('.art-topbar .lang-switch [lang]', (els) => els.map((e) => e.getAttribute('lang')));
      assert.ok(codes.length >= 2, 'dil bağlantıları lang niteliği taşımalı');
    }

    // Kartın sağ üstündeki Kaydet düğmesi kaydı açar/kapatır.
    const bm = await page.$eval('#bmBtn', (el) => el.getAttribute('aria-pressed'));
    await page.click('.article > #bmBtn');
    await page.waitForFunction(
      (v) => document.querySelector('#bmBtn').getAttribute('aria-pressed') !== v,
      {}, bm
    );
    const bmAfter = await page.$eval('#bmBtn', (el) => el.getAttribute('aria-pressed'));
    assert.notEqual(bmAfter, bm, 'tıklama kayıt durumunu değiştirmeli');
    assert.equal(await page.$eval('#bmBtn', (el) => el.classList.contains('on')), bmAfter === 'true');
    // Test sonrası durumu geri al.
    await page.click('.article > #bmBtn');
    await page.waitForFunction(
      (v) => document.querySelector('#bmBtn').getAttribute('aria-pressed') === v,
      {}, bm
    );

    assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
  });
});
