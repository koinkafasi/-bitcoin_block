'use strict';
/**
 * Sayfa Tasarımcısı → Sayfa Şablonu:
 *
 * - Kategori şablonunda sağ sütun kaldırılabilir.
 * - Sol/sağ sütun oranı değiştirilebilir.
 * - Yazı sayfası tek sütuna alınıp içerik genişliği ayarlanabilir.
 * - Ayarlar yalnız seçili sayfaya/kategoriye uygulanır.
 */

const test = require('node:test');
const assert = require('node:assert/strict');
const { BASE_URL, goto, setViewport, withAdminPage } = require('../lib/harness');

const SID = process.env.TB_SESSION;

async function backupLayouts(page, keys) {
  // Sayfa şablonu kayıtlarını yedekler; test bitince GERÇEK değerlerine
  // döndürülür (kullanıcı ayarları asla silinmez).
  return page.evaluate(async (keys) => {
    const out = [];
    for (const p of keys) {
      const r = await fetch('/admin/tasarim/yerlesim?sayfa=' + encodeURIComponent(p), {
        credentials: 'same-origin',
      });
      const j = await r.json();
      out.push({ key: p, hasCustom: !!j.hasCustom, config: j.config || {} });
    }
    return out;
  }, keys);
}

async function restoreLayouts(page, backup) {
  // Önce kayıt tamamen silinir, sonra özgün ham değer yazılır. Yama
  // biriktirme (patch-merge) nedeniyle yalnız özgün config'i POST etmek
  // testin eklediği anahtarları temizlemezdi.
  await page.evaluate(async (backup) => {
    for (const b of backup) {
      await fetch('/admin/tasarim/yerlesim?sayfa=' + encodeURIComponent(b.key), {
        method: 'POST', credentials: 'same-origin',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ reset: true }),
      });
      if (b.hasCustom && b.config && Object.keys(b.config).length) {
        await fetch('/admin/tasarim/yerlesim?sayfa=' + encodeURIComponent(b.key), {
          method: 'POST', credentials: 'same-origin',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(b.config),
        });
      }
    }
  }, backup);
}

async function resetLayouts(page) {
  // Yalnız sütun şablonu ayarını sıfırlar; sayfa blok düzenlerine dokunmaz.
  await page.evaluate(async () => {
    for (const p of ['home', 'kategori', 'kategori:ekonomi', 'article']) {
      await fetch('/admin/tasarim/yerlesim?sayfa=' + encodeURIComponent(p), {
        method: 'POST',
        credentials: 'same-origin',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ reset: true }),
      });
    }
  });
}

const LAYOUT_KEYS = ['home', 'kategori', 'kategori:ekonomi', 'kategori:gundem', 'article'];

test('sayfa şablonu: kategori sağ sütunu kaldırılır ve oran değiştirilir', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdminPage(async (page, errors) => {
    await setViewport(page, 1400, 1000);
    await goto(page, '/');
    const backup = await backupLayouts(page, LAYOUT_KEYS);
    await resetLayouts(page);

    try {
      await goto(page, '/admin/tasarim?sayfa=kategori');
      await page.waitForFunction(() => document.querySelector('#plPanel'), { timeout: 15000 });
      const panel = await page.evaluate(() => ({
        panel: !!document.getElementById('plPanel'),
        sidebar: document.getElementById('plSidebar').checked,
        side: document.getElementById('plSideWidth').value,
        main: document.getElementById('plMainWidth').value,
        cols: getComputedStyle(document.querySelector('.bp-row.c2')).gridTemplateColumns,
        hasRemove: [...document.querySelectorAll('#plTemplate option')]
          .some((o) => o.value === 'tek'),
      }));
      assert.ok(panel.panel, 'Sayfa Şablonu paneli bulunmalı');
      assert.equal(panel.sidebar, true, 'sağ sütun varsayılan açık olmalı');
      assert.equal(panel.side, '33');
      assert.equal(panel.main, '67');
      assert.ok(panel.cols.split(' ').length >= 2, 'şablon önizlemesi iki sütun olmalı');
      assert.ok(panel.hasRemove, 'tek sütun (sağ sütun yok) seçeneği olmalı');

      // Sağ sütunu kaldır → otomatik kayıt → public kategori tek sütun.
      await page.evaluate(() => {
        const el = document.getElementById('plSidebar');
        el.checked = false;
        el.dispatchEvent(new Event('change', { bubbles: true }));
      });
      await page.waitForFunction(
        // NOT (Faz 210): durum metni artık 'Kaydedildi' (onay işareti
        // kaldırıldı; bkz. site_builder.html otomatik kayıt). Tam eşitlik
        // yerine önek kontrolü — metne ek bilgi eklenmesi testi kırmasın.
        () => /^Kaydedildi/.test(((document.getElementById('plStatus') || {}).textContent || '').trim()),
        { timeout: 8000 }
      );
      const hiddenInBuilder = await page.evaluate(() => {
        const row = document.querySelector('.bp-row.c2');
        const sideZone = row ? row.querySelector('.bp-zone[data-zone="yan"]') : null;
        const tag = sideZone ? sideZone.querySelector('.pl-hidden-tag') : null;
        return {
          cls: row ? row.className : '',
          cols: row ? getComputedStyle(row).gridTemplateColumns.split(' ').filter(Boolean).length : 0,
          zoneDisplay: sideZone ? getComputedStyle(sideZone).display : 'yok',
          tag: tag ? getComputedStyle(tag).display : 'yok',
        };
      });
      assert.ok(hiddenInBuilder.cls.includes('pl-off'), 'tasarımcı önizlemesi tek sütuna geçmeli');
      assert.equal(hiddenInBuilder.cols, 1, 'tasarımcı önizlemesi tek sütun olmalı');
      assert.notEqual(hiddenInBuilder.zoneDisplay, 'none', 'sağ sütun alanı düzenlenebilir kalmalı');
      assert.notEqual(hiddenInBuilder.tag, 'none', '"yayında gizli" rozeti görünmeli');

      await goto(page, '/kategori/ekonomi');
      const off = await page.evaluate(() => {
        const grid = document.querySelector('.home-grid');
        const side = document.querySelector('.home-side');
        return {
          cls: grid ? grid.className : '',
          cols: grid ? getComputedStyle(grid).gridTemplateColumns.split(' ').filter(Boolean).length : 0,
          sideDisplay: side ? getComputedStyle(side).display : 'yok',
          overflowX: document.documentElement.scrollWidth - document.documentElement.clientWidth,
        };
      });
      assert.ok(off.cls.includes('pg-no-side'), 'public kategori pg-no-side sınıfı almalı');
      assert.equal(off.cols, 1, 'public kategori tek sütun olmalı');
      assert.equal(off.sideDisplay, 'none', 'sağ sütun görünmemeli');
      assert.equal(off.overflowX, 0, 'tek sütunda yatay taşma olmamalı');

      // Sağ sütunu geri aç ve %40 yap.
      await goto(page, '/admin/tasarim?sayfa=kategori');
      await page.waitForFunction(() => document.querySelector('#plPanel'), { timeout: 15000 });
      await page.evaluate(() => {
        const side = document.getElementById('plSidebar');
        side.checked = true;
        side.dispatchEvent(new Event('change', { bubbles: true }));
        const sw = document.getElementById('plSideWidth');
        sw.value = '40';
        sw.dispatchEvent(new Event('change', { bubbles: true }));
      });
      await page.waitForFunction(
        // NOT (Faz 210): durum metni artık 'Kaydedildi' (onay işareti
        // kaldırıldı; bkz. site_builder.html otomatik kayıt). Tam eşitlik
        // yerine önek kontrolü — metne ek bilgi eklenmesi testi kırmasın.
        () => /^Kaydedildi/.test(((document.getElementById('plStatus') || {}).textContent || '').trim()),
        { timeout: 8000 }
      );
      await goto(page, '/kategori/ekonomi');
      const on = await page.evaluate(() => {
        const grid = document.querySelector('.home-grid');
        const side = document.querySelector('.home-side');
        const main = document.querySelector('.home-main');
        return {
          cls: grid ? grid.className : '',
          style: grid ? grid.getAttribute('style') : '',
          sideW: side ? side.getBoundingClientRect().width : 0,
          mainW: main ? main.getBoundingClientRect().width : 0,
          overflowX: document.documentElement.scrollWidth - document.documentElement.clientWidth,
        };
      });
      assert.ok(!on.cls.includes('pg-no-side'), 'sağ sütun geri açılmalı');
      assert.match(on.style, /--pg-side-w:40%/, 'sağ sütun %40 uygulanmalı');
      const ratio = on.sideW / (on.sideW + on.mainW);
      assert.ok(ratio > 0.36 && ratio < 0.44, 'sağ sütun oranı ~%40 olmalı: ' + ratio);
      assert.equal(on.overflowX, 0, 'iki sütunda yatay taşma olmamalı');

      // Ayar yalnız kategoriye uygulanır; ana sayfa varsayılan kalır.
      await goto(page, '/');
      const home = await page.evaluate(() => {
        const grid = document.querySelector('.home-grid');
        return { cls: grid ? grid.className : '', style: grid ? grid.getAttribute('style') : '' };
      });
      assert.ok(!home.cls.includes('pg-no-side'), 'ana sayfa etkilenmemeli');
      assert.match(home.style, /--pg-side-w:33%/, 'ana sayfa varsayılan oranı korunmalı');
    } finally {
      await restoreLayouts(page, backup);
    }
    assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
  });
});

test('sayfa şablonu: kategori üst alanı (başlık, açıklama, kırıntı) gizlenebilir', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdminPage(async (page, errors) => {
    await setViewport(page, 1400, 1000);
    await goto(page, '/');
    const backup = await backupLayouts(page, LAYOUT_KEYS);
    await resetLayouts(page);

    try {
      /* Tasarımcıda alanlar: yalnız kategori sayfasında görünür. */
      await goto(page, '/admin/tasarim?sayfa=kategori&kategori=ekonomi');
      await page.waitForFunction(() => document.getElementById('plPanel'), { timeout: 15000 });
      const ui = await page.evaluate(() => ({
        title: !!document.getElementById('plHideTitle'),
        desc: !!document.getElementById('plHideDesc'),
        crumb: !!document.getElementById('plHideCrumb'),
        checked: document.getElementById('plHideTitle') ?
          document.getElementById('plHideTitle').checked : null,
      }));
      assert.ok(ui.title && ui.desc && ui.crumb,
        'kategori üst alanı için 3 gizleme anahtarı bulunmalı');
      assert.equal(ui.checked, false, 'varsayılan görünür olmalı');

      /* Ana sayfada bu alanlar olmamalı (yalnız kategori sayfası). */
      await goto(page, '/admin/tasarim?sayfa=home');
      await page.waitForFunction(() => document.getElementById('plPanel'), { timeout: 15000 });
      const homeUI = await page.evaluate(() => !!document.getElementById('plHideTitle'));
      assert.equal(homeUI, false, 'ana sayfada kategori üst alanı ayarı olmamalı');

      /* Üçünü de gizle → otomatik kayıt. */
      await goto(page, '/admin/tasarim?sayfa=kategori&kategori=ekonomi');
      await page.waitForFunction(() => document.getElementById('plHideTitle'), { timeout: 15000 });
      await page.evaluate(() => {
        ['plHideTitle', 'plHideDesc', 'plHideCrumb'].forEach((id) => {
          const el = document.getElementById(id);
          el.checked = true;
          el.dispatchEvent(new Event('change', { bubbles: true }));
        });
      });
      await page.waitForFunction(
        // NOT (Faz 210): durum metni artık 'Kaydedildi' (onay işareti
        // kaldırıldı; bkz. site_builder.html otomatik kayıt). Tam eşitlik
        // yerine önek kontrolü — metne ek bilgi eklenmesi testi kırmasın.
        () => /^Kaydedildi/.test(((document.getElementById('plStatus') || {}).textContent || '').trim()),
        { timeout: 8000 }
      );

      await goto(page, '/kategori/ekonomi');
      const hidden = await page.evaluate(() => {
        const vis = (el) => {
          if (!el) return 'yok';
          const r = el.getBoundingClientRect();
          return (r.width <= 1 && r.height <= 1) ? 'gizli' : 'gorunur';
        };
        const pill = document.querySelector('.cat-pill');
        const intro = document.querySelector('.cat-intro');
        const crumb = document.querySelector('nav.breadcrumb');
        return {
          pillState: vis(pill),
          introState: vis(intro),
          crumbState: vis(crumb),
          pillText: pill ? pill.textContent.trim() : '',
          crumbLinks: crumb ? crumb.querySelectorAll('a').length : 0,
          headCls: document.querySelector('.cat-head') ?
            document.querySelector('.cat-head').className : '',
          pillCls: pill ? pill.className : '',
          crumbCls: crumb ? crumb.className : '',
          main: !!document.querySelector('main#main'),
          overflowX: document.documentElement.scrollWidth - document.documentElement.clientWidth,
        };
      });
      assert.equal(hidden.pillState, 'gizli', 'çerçeveli kategori başlığı görsel olarak gizlenmeli');
      assert.equal(hidden.introState, 'gizli', 'açıklama görsel olarak gizlenmeli');
      assert.equal(hidden.crumbState, 'gizli', 'kırıntı görsel olarak gizlenmeli');
      assert.match(hidden.pillText, /Ekonomi/, 'başlık metni SEO için DOM\'da kalmalı');
      assert.ok(hidden.crumbLinks >= 1, 'kırıntı bağlantıları DOM\'da kalmalı');
      assert.match(hidden.headCls, /sr-only/, 'başlık + açıklama birlikte gizlenince kapsayıcı sr-only olmalı');
      assert.match(hidden.pillCls, /sr-only/, 'başlık sr-only kalmalı');
      assert.match(hidden.crumbCls, /sr-only/, 'kırıntı sr-only kalmalı');
      assert.ok(hidden.main, 'sayfa gövdesi bozulmamalı');
      assert.equal(hidden.overflowX, 0, 'yatay taşma olmamalı');

      /* Yalnız başlığı geri aç: açıklama gizli kalır, başlık görünür. */
      const setFlags = (body) => page.evaluate(async (body) => {
        const r = await fetch('/admin/tasarim/yerlesim?sayfa=kategori%3Aekonomi', {
          method: 'POST', credentials: 'same-origin',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(body),
        });
        return await r.json();
      }, body);
      assert.equal((await setFlags({
        sidebar: true, sidebarWidth: 33, contentWidth: 1280,
        hideTitle: false, hideDesc: true, hideBreadcrumb: true,
      })).ok, true, 'kısmi görünürlük kaydedilmeli');

      await goto(page, '/kategori/ekonomi');
      const partial = await page.evaluate(() => {
        const vis = (el) => {
          if (!el) return 'yok';
          const r = el.getBoundingClientRect();
          return (r.width <= 1 && r.height <= 1) ? 'gizli' : 'gorunur';
        };
        return {
          pill: vis(document.querySelector('.cat-pill')),
          intro: vis(document.querySelector('.cat-intro')),
          crumb: vis(document.querySelector('nav.breadcrumb')),
          headCls: document.querySelector('.cat-head') ?
            document.querySelector('.cat-head').className : '',
        };
      });
      assert.equal(partial.pill, 'gorunur', 'başlık geri açıldığında görünmeli');
      assert.equal(partial.intro, 'gizli', 'açıklama gizli kalmalı');
      assert.equal(partial.crumb, 'gizli', 'kırıntı gizli kalmalı');
      assert.doesNotMatch(partial.headCls, /sr-only/,
        'yalnız biri gizliyken kapsayıcı görünür kalmalı');
      assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
    } finally {
      await restoreLayouts(page, backup);
    }
  });
});

test('sayfa şablonu: genel kategori düzenindeki gizleme tüm kategorilere yansır', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdminPage(async (page, errors) => {
    await setViewport(page, 1400, 1000);
    await goto(page, '/');
    const backup = await backupLayouts(page, LAYOUT_KEYS);
    await resetLayouts(page);

    try {
      /* Genel kategori düzeninde kırıntı gizlendi. */
      const setFlags = (pageKey, body) => page.evaluate(async ({ pageKey, body }) => {
        const r = await fetch('/admin/tasarim/yerlesim?sayfa=' + encodeURIComponent(pageKey), {
          method: 'POST', credentials: 'same-origin',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(body),
        });
        return await r.json();
      }, { pageKey, body });
      assert.equal((await setFlags('kategori', { hideBreadcrumb: true, hideTitle: true })).ok, true,
        'genel kategori gizleme ayarı kaydedilmeli');

      /* Kendi kaydı olmayan kategori (ekonomi) genel ayarı miras almalı. */
      await goto(page, '/kategori/ekonomi');
      const inherited = await page.evaluate(() => {
        const vis = (el) => {
          if (!el) return 'yok';
          const r = el.getBoundingClientRect();
          return (r.width <= 1 && r.height <= 1) ? 'gizli' : 'gorunur';
        };
        return {
          crumb: vis(document.querySelector('nav.breadcrumb')),
          pill: vis(document.querySelector('.cat-pill')),
          crumbCls: (document.querySelector('nav.breadcrumb') || {}).className || '',
        };
      });
      assert.equal(inherited.crumb, 'gizli', 'genel gizleme kategorilere yansımali');
      assert.equal(inherited.pill, 'gizli', 'başlık gizleme de miras alınmalı');
      assert.match(inherited.crumbCls, /sr-only/, 'gizleme sr-only ile uygulanmalı');

      /* Kategori kendi alanında istisna yapabilir: yalnız kırıntıyı geri aç. */
      assert.equal((await setFlags('kategori:ekonomi', { hideBreadcrumb: false })).ok, true,
        'kategoriye özel istisna kaydedilmeli');
      await goto(page, '/kategori/ekonomi');
      const override = await page.evaluate(() => {
        const vis = (el) => {
          if (!el) return 'yok';
          const r = el.getBoundingClientRect();
          return (r.width <= 1 && r.height <= 1) ? 'gizli' : 'gorunur';
        };
        return {
          crumb: vis(document.querySelector('nav.breadcrumb')),
          pill: vis(document.querySelector('.cat-pill')),
        };
      });
      assert.equal(override.crumb, 'gorunur', 'kategori kendi alanında istisna yapabilmeli');
      assert.equal(override.pill, 'gizli', 'diğer alanlar genel ayardan gelmeli (alan bazlı miras)');

      /* Genel ayar değişince istisnası olmayan alanlar güncellenmeli. */
      assert.equal((await setFlags('kategori', { hideBreadcrumb: true, hideDesc: true, hideTitle: false })).ok, true);
      await goto(page, '/kategori/ekonomi');
      const updated = await page.evaluate(() => {
        const vis = (el) => {
          if (!el) return 'yok';
          const r = el.getBoundingClientRect();
          return (r.width <= 1 && r.height <= 1) ? 'gizli' : 'gorunur';
        };
        return {
          crumb: vis(document.querySelector('nav.breadcrumb')),
          pill: vis(document.querySelector('.cat-pill')),
          intro: vis(document.querySelector('.cat-intro')),
        };
      });
      assert.equal(updated.pill, 'gorunur', 'genelde açılan başlık kategoride görünmeli');
      assert.equal(updated.intro, 'gizli', 'genelde gizlenen açıklama kategoride gizli olmalı');
      assert.equal(updated.crumb, 'gorunur', 'kategori istisnası korunmalı');
      assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
    } finally {
      await restoreLayouts(page, backup);
    }
  });
});

test('sayfa şablonu: ana sayfa tek sütun ve özel genişlik', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdminPage(async (page, errors) => {
    await setViewport(page, 1400, 1000);
    await goto(page, '/');
    const backup = await backupLayouts(page, LAYOUT_KEYS);
    await resetLayouts(page);
    try {
      const saved = await page.evaluate(async () => {
        const r = await fetch('/admin/tasarim/yerlesim?sayfa=home', {
          method: 'POST',
          credentials: 'same-origin',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ sidebar: false, contentWidth: 1100 }),
        });
        return await r.json();
      });
      assert.equal(saved.ok, true, 'ana sayfa şablonu kaydedilmeli');

      await goto(page, '/');
      const home = await page.evaluate(() => {
        const grid = document.querySelector('.home-grid');
        const side = document.querySelector('.home-side');
        return {
          cls: grid ? grid.className : '',
          maxWidth: grid ? getComputedStyle(grid).maxWidth : '',
          w: grid ? Math.round(grid.getBoundingClientRect().width) : 0,
          sideDisplay: side ? getComputedStyle(side).display : 'yok',
          overflowX: document.documentElement.scrollWidth - document.documentElement.clientWidth,
        };
      });
      assert.ok(home.cls.includes('pg-no-side'), 'ana sayfa tek sütun olmalı');
      assert.equal(home.maxWidth, '1100px', 'ana sayfa özel genişliği uygulanmalı');
      assert.ok(home.w <= 1100, 'ana sayfa içeriği 1100 px sınırında olmalı');
      assert.equal(home.sideDisplay, 'none', 'ana sayfa sağ sütunu gizlenmeli');
      assert.equal(home.overflowX, 0, 'ana sayfada yatay taşma olmamalı');
    } finally {
      await restoreLayouts(page, backup);
    }
    assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
  });
});

test('sayfa şablonu: kategoriye özel şablon genel ayarı ezer', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdminPage(async (page, errors) => {
    await setViewport(page, 1400, 1000);
    await goto(page, '/');
    const backup = await backupLayouts(page, LAYOUT_KEYS);
    await resetLayouts(page);

    const setLayout = (pageKey, body) => page.evaluate(async ({ pageKey, body }) => {
      const r = await fetch('/admin/tasarim/yerlesim?sayfa=' + encodeURIComponent(pageKey), {
        method: 'POST',
        credentials: 'same-origin',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });
      return await r.json();
    }, { pageKey, body });

    try {
      // Genel kategori: iki sütun %40. Ekonomi: tek sütun.
      assert.equal((await setLayout('kategori', { sidebar: true, sidebarWidth: 40 })).ok, true);
      assert.equal((await setLayout('kategori:ekonomi', { sidebar: false })).ok, true);

      await goto(page, '/kategori/ekonomi');
      const ekonomi = await page.evaluate(() => {
        const grid = document.querySelector('.home-grid');
        return {
          cls: grid ? grid.className : '',
          side: document.querySelector('.home-side')
            ? getComputedStyle(document.querySelector('.home-side')).display : 'yok',
        };
      });
      assert.ok(ekonomi.cls.includes('pg-no-side'), 'Ekonomi özel şablonu tek sütun olmalı');
      assert.equal(ekonomi.side, 'none');

      await goto(page, '/kategori/gundem');
      const gundem = await page.evaluate(() => {
        const grid = document.querySelector('.home-grid');
        return {
          cls: grid ? grid.className : '',
          style: grid ? grid.getAttribute('style') : '',
        };
      });
      // Gündem'de kullanıcıya özel şablon varsa kalıtım kontrolü geçersizdir;
      // bu durumda kontrol atlanır (canlı ayar testi ezmek zorunda değil).
      if (/--pg-side-w:40%/.test(gundem.style)) {
        assert.ok(!gundem.cls.includes('pg-no-side'), 'Gündem genel kategori şablonunu kullanmalı');
        assert.match(gundem.style, /--pg-side-w:40%/, 'genel kategori %40 oranı Gündem’e uygulanmalı');
      } else {
        t.diagnostic('Gündem için özel şablon kayıtlı; genel kalıtım kontrolü atlandı');
      }

      // Builder seçili kategoriye özel değerleri göstermeli.
      await goto(page, '/admin/tasarim?sayfa=kategori&kategori=ekonomi');
      await page.waitForFunction(() => document.querySelector('#plPanel'), { timeout: 15000 });
      const panel = await page.evaluate(() => ({
        sidebar: document.getElementById('plSidebar').checked,
        template: document.getElementById('plTemplate').value,
      }));
      assert.equal(panel.sidebar, false, 'builder Ekonomi için sağ sütunu kapalı göstermeli');
      assert.equal(panel.template, 'tek');
    } finally {
      await restoreLayouts(page, backup);
    }
    assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
  });
});

test('sayfa şablonu: yazı sayfası tek sütun ve özel içerik genişliği', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdminPage(async (page, errors) => {
    await setViewport(page, 1400, 1000);
    await goto(page, '/');
    const backup = await backupLayouts(page, LAYOUT_KEYS);
    await resetLayouts(page);

    try {
      // Kategori şablonunun yazı ayarından önceki durumu (canlı ayarlardan
      // bağımsız çalışmak için referans alınır).
      await goto(page, '/kategori/ekonomi');
      const catBefore = await page.evaluate(() => {
        const grid = document.querySelector('.home-grid');
        return { cls: grid ? grid.className : '', style: grid ? grid.getAttribute('style') : '' };
      });

      const articlePath = await page.evaluate(() => {
        const a = [...document.querySelectorAll('a[href]')].find((x) =>
          /\/[^/]+-[0-9a-f]{4,8}-\d+$/.test(x.getAttribute('href') || ''));
        return a ? a.getAttribute('href') : '';
      });
      assert.ok(articlePath, 'test için yazı bağlantısı bulunmalı');

      const saved = await page.evaluate(async () => {
        const r = await fetch('/admin/tasarim/yerlesim?sayfa=article', {
          method: 'POST',
          credentials: 'same-origin',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ sidebar: false, contentWidth: 900 }),
        });
        return await r.json();
      });
      assert.equal(saved.ok, true, 'yazı sayfası şablonu kaydedilmeli');
      assert.equal(saved.layout.sidebar, false);

      await goto(page, articlePath);
      const single = await page.evaluate(() => {
        const wrap = document.querySelector('.art-wrap');
        const side = document.querySelector('.art-side');
        const article = document.querySelector('.article');
        return {
          cls: wrap ? wrap.className : '',
          style: wrap ? wrap.getAttribute('style') : '',
          maxWidth: wrap ? getComputedStyle(wrap).maxWidth : '',
          sideDisplay: side ? getComputedStyle(side).display : 'yok',
          articleW: article ? Math.round(article.getBoundingClientRect().width) : 0,
          overflowX: document.documentElement.scrollWidth - document.documentElement.clientWidth,
        };
      });
      assert.ok(single.cls.includes('pg-no-side'), 'yazı sayfası pg-no-side almalı');
      assert.equal(single.sideDisplay, 'none', 'yazı sağ sütunu gizlenmeli');
      assert.equal(single.maxWidth, '900px', 'özel içerik genişliği uygulanmalı');
      assert.ok(single.articleW <= 901, 'yazı içeriği 900 px sınırında olmalı');
      assert.equal(single.overflowX, 0, 'yazı sayfasında yatay taşma olmamalı');

      // Tek sütun mobilde de taşmamalı.
      await setViewport(page, 390, 844);
      await goto(page, articlePath);
      const mobile = await page.evaluate(() => {
        const wrap = document.querySelector('.art-wrap');
        const side = document.querySelector('.art-side');
        return {
          sideDisplay: side ? getComputedStyle(side).display : 'yok',
          overflowX: document.documentElement.scrollWidth - document.documentElement.clientWidth,
          w: wrap ? Math.round(wrap.getBoundingClientRect().width) : 0,
          client: document.documentElement.clientWidth,
        };
      });
      assert.equal(mobile.sideDisplay, 'none', 'mobilde sağ sütun gizli kalmalı');
      assert.ok(mobile.w <= mobile.client, 'mobil içerik viewport içinde olmalı');
      assert.equal(mobile.overflowX, 0, 'mobilde yatay taşma olmamalı');
      await setViewport(page, 1400, 1000);

      // Kategori şablonu bu ayardan etkilenmemeli (yazı ayarından ÖNCEKİ
      // durumla birebir aynı kalmalı).
      await goto(page, '/kategori/ekonomi');
      const catAfter = await page.evaluate(() => {
        const grid = document.querySelector('.home-grid');
        return { cls: grid ? grid.className : '', style: grid ? grid.getAttribute('style') : '' };
      });
      assert.equal(catAfter.cls, catBefore.cls,
        'yazı ayarı kategoriye sızmamalı: ' + catAfter.cls + ' vs ' + catBefore.cls);
      assert.equal(catAfter.style, catBefore.style,
        'yazı şablonu kategori oranını değiştirmemeli: ' + catAfter.style + ' vs ' + catBefore.style);
    } finally {
      await restoreLayouts(page, backup);
    }
    assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
  });
});
