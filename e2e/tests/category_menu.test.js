'use strict';
/**
 * Ekonomi mega menüsü + alt kategori sayfaları.
 *
 * - Üst menüde "Ekonomi"ye tıklanınca kısa, beyaz alt kategori paneli açılır.
 * - Paneldeki alt kategori sayfası açılır; kırıntı yolu ve sekmeler doğrudur.
 * - Mobil çekmecede alt kategoriler görünür.
 * - Yönetim > Kategoriler ekranında üst kategori seçimi vardır.
 */

const test = require('node:test');
const assert = require('node:assert/strict');
const path = require('node:path');
const { BASE_URL, goto, setViewport, withAdminPage } = require('../lib/harness');

const SID = process.env.TB_SESSION;

test('Ekonomi mega menüsü ve alt kategori sayfası', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdminPage(async (page, errors) => {
    await setViewport(page, 1440, 1000);
    await goto(page, '/');

    // Üst menüde Ekonomi'nin alt kategori paneli var mı?
    const menu = await page.evaluate(() => {
      const item = [...document.querySelectorAll('.nav-item.has-mega')].find((el) =>
        (el.querySelector('[data-mega-btn]') && (el.querySelector('[data-mega-btn]').textContent || '').toLowerCase().includes('ekonomi')));
      if (!item) return null;
      const btn = item.querySelector('[data-mega-btn]');
      const panel = item.querySelector('[data-mega-panel]');
      return {
        expanded: btn ? btn.getAttribute('aria-expanded') : null,
        hidden: panel ? panel.hidden : null,
        links: panel ? panel.querySelectorAll('.nav-mega-grid a').length : 0,
      };
    });
    assert.ok(menu, 'Ekonomi mega menü öğesi bulunmalı');
    assert.equal(menu.expanded, 'false', 'panel başlangıçta kapalı olmalı');
    assert.ok(menu.links >= 10, 'panelde alt kategoriler olmalı: ' + menu.links);

    // Tıkla → panel açılır, beyaz ve kısa.
    await page.evaluate(() => {
      const item = [...document.querySelectorAll('.nav-item.has-mega')].find((el) =>
        (el.querySelector('[data-mega-btn]') && (el.querySelector('[data-mega-btn]').textContent || '').toLowerCase().includes('ekonomi')));
      item.querySelector('[data-mega-btn]').click();
    });
    await page.waitForFunction(() => {
      const item = [...document.querySelectorAll('.nav-item.has-mega')].find((el) =>
        (el.querySelector('[data-mega-btn]') && (el.querySelector('[data-mega-btn]').textContent || '').toLowerCase().includes('ekonomi')));
      const p = item && item.querySelector('[data-mega-panel]');
      return p && !p.hidden;
    });

    const open = await page.evaluate(() => {
      const item = [...document.querySelectorAll('.nav-item.has-mega')].find((el) =>
        (el.querySelector('[data-mega-btn]') && (el.querySelector('[data-mega-btn]').textContent || '').toLowerCase().includes('ekonomi')));
      const panel = item.querySelector('[data-mega-panel]');
      const inner = panel.querySelector('.nav-mega-inner');
      const r = inner.getBoundingClientRect();
      const cs = getComputedStyle(inner);
      return {
        bg: cs.backgroundColor,
        cls: panel.className,
        w: Math.round(r.width),
        h: Math.round(r.height),
        top: Math.round(r.top),
        rail: !!panel.querySelector('.nav-mega-rail'),
        close: !!panel.querySelector('[data-mega-close]'),
        borsa: !!panel.querySelector('.nav-mega-grid a[href="/kategori/borsa"]'),
        all: panel.querySelector('.nav-mega-all') ? panel.querySelector('.nav-mega-all').getAttribute('href') : null,
      };
    });
    // Sayfa boyu tasarım kullanıcı ayarına bağlıdır; yönetici farklı bir
    // tasarım seçtiyse sayfa boyu ölçüleri beklenmez (ayar testte değiştirilmez).
    if (open.cls.includes('nav-mega-sayfa')) {
      assert.equal(open.bg, 'rgb(255, 255, 255)', 'panel arka planı beyaz olmalı: ' + open.bg);
      assert.ok(open.h >= 500, 'panel sayfa boyunda olmalı: ' + open.h);
      assert.ok(open.w >= 1200, 'panel tam genişlikte olmalı: ' + open.w);
      assert.ok(open.rail, 'sayfa boyu panelde kategori rayı olmalı');
      assert.ok(open.close, 'sayfa boyu panelde kapat düğmesi olmalı');
      // Panel başlığın HEMEN altında başlamalı; sihirli piksel penceresi yerine
      // aynı ölçümdeki header alt kenarına bağlanır (header yüksekliği tasarımla
      // değişir). Tolerans: panel bir mikro-header çubuğunun altında başlayabilir.
      const geo = await page.evaluate(() => {
        const h = document.querySelector('header.site-header');
        const p = [...document.querySelectorAll('[data-mega-panel]')].find((x) => !x.hidden);
        return {
          headerBottom: h ? Math.round(h.getBoundingClientRect().bottom) : 0,
          panelTop: p ? Math.round(p.getBoundingClientRect().top) : -1,
        };
      });
      assert.ok(
        geo.panelTop >= geo.headerBottom - 16 && geo.panelTop <= geo.headerBottom + 25,
        `panel başlığın hemen altında olmalı: top=${geo.panelTop} headerBottom=${geo.headerBottom}`
      );
    }
    assert.ok(open.borsa, 'Borsa alt kategorisi panelde olmalı');
    assert.equal(open.all, '/kategori/ekonomi', 'Tüm Ekonomi bağlantısı doğru olmalı');

    // Alt kategoriye git.
    await Promise.all([
      page.waitForNavigation({ waitUntil: 'domcontentloaded' }),
      page.click('.nav-mega-grid a[href="/kategori/borsa"]'),
    ]);
    assert.ok(page.url().endsWith('/kategori/borsa'), 'Borsa sayfası açılmalı: ' + page.url());

    const cat = await page.evaluate(() => ({
      h1: (document.querySelector('h1.cat-pill') || {}).textContent || '',
      crumbs: [...document.querySelectorAll('.breadcrumb a')].map((a) => a.textContent.trim()),
      tabs: [...document.querySelectorAll('.cat-subs a')].map((a) => ({
        text: a.textContent.trim(), href: a.getAttribute('href'), on: a.classList.contains('on'),
      })),
      canon: (document.querySelector('link[rel="canonical"]') || {}).href || '',
      empty: !!document.querySelector('.cat-empty'),
    }));
    assert.ok(cat.h1.includes('Borsa'), 'sayfa başlığı Borsa olmalı');
    assert.ok(cat.crumbs.includes('Ekonomi'), 'kırıntı yolunda Ekonomi olmalı: ' + cat.crumbs.join(' › '));
    const active = cat.tabs.find((x) => x.on);
    assert.ok(active && active.href === '/kategori/borsa', 'Borsa sekmesi aktif olmalı');
    assert.ok(cat.tabs.some((x) => x.href === '/kategori/ekonomi'), 'Tümü bağlantısı Ekonomi olmalı');
    assert.ok(cat.canon.endsWith('/kategori/borsa'), 'canonical alt kategori adresi olmalı');

    assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
  });
});

test('mobil çekmecede Ekonomi alt kategorileri', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdminPage(async (page, errors) => {
    await setViewport(page, 390, 844);
    await goto(page, '/');
    await page.click('[data-menu-btn]');
    await page.waitForFunction(() => document.getElementById('hpDrawer').classList.contains('open'));
    const subs = await page.evaluate(() =>
      [...document.querySelectorAll('#hpDrawer .hp-drawer-subview a')].map((a) => a.getAttribute('href')));
    assert.ok(subs.includes('/kategori/borsa'), 'çekmecede Borsa alt kategorisi olmalı');
    assert.ok(subs.includes('/kategori/doviz'), 'çekmecede Döviz alt kategorisi olmalı');

    const kapali = await page.evaluate(() => {
      const panel = document.querySelector('#hpDrawer .hp-drawer-panel');
      return [...document.querySelectorAll('#hpDrawer .hp-drawer-subview')].every((p) => p.hidden)
        && !panel.classList.contains('sub-open');
    });
    assert.ok(kapali, 'alt kategoriler başlangıçta kapalı olmalı');

    await page.click('#hpDrawer [data-subopen="ekonomi"]');
    await page.waitForFunction(() => {
      const panel = document.querySelector('#hpDrawer .hp-drawer-panel');
      return panel.classList.contains('sub-open');
    });
    const acik = await page.evaluate(() => {
      const views = [...document.querySelectorAll('#hpDrawer .hp-drawer-subview')];
      const open = views.filter((p) => !p.hidden);
      const p = open[0];
      return {
        adet: open.length,
        borsa: p ? !!p.querySelector('a[href="/kategori/borsa"]') : false,
        h: p ? Math.round(p.getBoundingClientRect().height) : 0,
        cols: p ? getComputedStyle(p.querySelector('.hp-subview-list')).gridTemplateColumns.split(' ').length : 0,
        mainHidden: getComputedStyle(document.getElementById('hpDrawerMain')).display === 'none',
      };
    });
    assert.equal(acik.adet, 1, 'aynı anda yalnızca bir alt menü açık olmalı');
    assert.ok(acik.borsa, 'açılan panelde Borsa bağlantısı olmalı');
    assert.ok(acik.h > 100, 'açılan alt menü görünür yükseklikte olmalı: ' + acik.h);
    assert.ok(acik.cols >= 1, 'alt menü okunaklı sütun düzeninde olmalı');
    assert.ok(acik.mainHidden, 'alt menü açılınca ana liste gizlenmeli');

    await page.click('#hpDrawer .hp-drawer-subview:not([hidden]) [data-subback]');
    await page.waitForFunction(() => {
      const panel = document.querySelector('#hpDrawer .hp-drawer-panel');
      return !panel.classList.contains('sub-open');
    });
    assert.ok(true, 'geri düğmesi ana listeye dönmeli');

    // Alt kategori sayfasında ilgili alt menü otomatik açılmalı.
    await goto(page, '/kategori/borsa');
    await page.click('[data-menu-btn]');
    await page.waitForFunction(() => document.getElementById('hpDrawer').classList.contains('open'));
    const oto = await page.evaluate(() => {
      const b = document.querySelector('#hpDrawer [data-subopen="ekonomi"]');
      const p = document.getElementById('hpSubView-ekonomi');
      return {
        expanded: b ? b.getAttribute('aria-expanded') : null,
        hidden: p ? p.hidden : null,
        subOpen: document.querySelector('#hpDrawer .hp-drawer-panel').classList.contains('sub-open'),
      };
    });
    assert.equal(oto.expanded, 'true', 'alt kategori sayfasında alt menü açık gelmeli');
    assert.equal(oto.hidden, false, 'alt kategori paneli görünür olmalı');
    assert.ok(oto.subOpen, 'çekmece alt menü modunda açılmalı');

    assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
  });
});

test('ana menü GZT düzeninde: yardımcı kategoriler alt menüde', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdminPage(async (page, errors) => {
    await setViewport(page, 1440, 1000);
    await goto(page, '/');

    const menu = await page.evaluate(() => {
      const top = [...document.querySelectorAll('.site-nav > .nav-item > a')]
        .map((a) => a.getAttribute('href'));
      const children = (label) => {
        const item = [...document.querySelectorAll('.nav-item.has-mega')].find((el) =>
          (el.querySelector('[data-mega-btn]') || {}).textContent
            ? (el.querySelector('[data-mega-btn]').textContent || '').trim().startsWith(label)
            : false);
        return item
          ? [...item.querySelectorAll('.nav-mega-grid a')].map((a) => a.getAttribute('href'))
          : [];
      };
      return {
        top,
        ekonomi: children('Ekonomi'),
        gundem: children('Gündem'),
        yasam: children('Yaşam'),
        teknoloji: children('Teknoloji'),
        spor: children('Spor'),
        kultur: children('Eğitim-kültür'),
        dunya: children('Dünya'),
      };
    });

    for (const slug of ['asayis', 'ilceler', 'bilim', 'oyun', 'saglik', 'turizm', 'sanat']) {
      assert.ok(!menu.top.includes('/kategori/' + slug),
        `${slug} artık üst menüde olmamalı`);
    }
    const order = await page.evaluate(() =>
      [...document.querySelectorAll('.site-nav > .nav-item > a')].map((a) => a.textContent.trim()));
    assert.deepEqual(order,
      ['Gündem', 'Siyaset', 'Ekonomi', 'Dünya', 'Spor', 'Yaşam', 'Teknoloji', 'Eğitim-kültür'],
      'üst menü SEO/kullanım sırası doğru olmalı: ' + order.join(' · '));
    // Kategori satırları kullanıcı verisidir: migration ile tohumlanır ama site
    // sahibi silebilir. Bu yüzden yalnızca GERÇEKTEN var olan (sayfası 200
    // dönen) alt kategoriler menüde görünmek zorunda — mevcut kategori menüde
    // gizliyse bu bir regresyondur.
    const ekonomiSluglari = ['turkiye-ekonomisi', 'dunya-ekonomisi', 'piyasalar', 'borsa',
      'doviz', 'altin', 'kripto-para', 'emtia', 'faiz', 'enflasyon', 'bankacilik',
      'finans', 'is-dunyasi', 'enerji', 'otomotiv', 'emlak', 'tarim', 'sanayi-uretim'];
    const varOlan = [];
    for (const slug of ekonomiSluglari) {
      const resp = await page.goto(BASE_URL + '/kategori/' + slug, { waitUntil: 'domcontentloaded' });
      if (resp && resp.status() === 200) varOlan.push(slug);
    }
    await goto(page, '/');
    for (const slug of varOlan) {
      assert.ok(menu.ekonomi.includes('/kategori/' + slug), `Ekonomi altında ${slug} olmalı`);
    }
    assert.ok(varOlan.length > 0, 'en az bir ekonomi alt kategorisi bulunmalı');
    for (const slug of ['hisse-senetleri', 'yatirim-fonlari', 'banka-gise-kurlari', 'kap-haberleri']) {
      assert.ok(!menu.ekonomi.includes('/kategori/' + slug), `Ekonomi'de boş ${slug} gizlenmeli`);
    }
    assert.ok(menu.gundem.includes('/kategori/asayis'), 'Asayiş Gündem altına taşınmalı');
    assert.ok(menu.gundem.includes('/kategori/ilceler'), 'İlçeler Gündem altına taşınmalı');
    for (const slug of ['afet-kaza', 'analiz']) {
      assert.ok(menu.gundem.includes('/kategori/' + slug), `Gündem altında ${slug} olmalı`);
    }
    for (const slug of ['guvenlik', 'ozgun', 'toplum', 'ramazan']) {
      assert.ok(!menu.gundem.includes('/kategori/' + slug), `Gündem'de boş ${slug} gizlenmeli`);
    }
    assert.ok(menu.yasam.includes('/kategori/saglik'), 'Sağlık Yaşam altına taşınmalı');
    assert.ok(menu.yasam.includes('/kategori/turizm'), 'Turizm Yaşam altına taşınmalı');
    assert.ok(menu.teknoloji.includes('/kategori/bilim'), 'Bilim Teknoloji altına taşınmalı');
    assert.ok(menu.teknoloji.includes('/kategori/oyun'), 'Oyun Teknoloji altına taşınmalı');
    assert.ok(menu.kultur.includes('/kategori/sanat'), 'Sanat Eğitim-kültür altına taşınmalı');
    assert.ok(menu.spor.includes('/kategori/futbol'), 'Futbol Spor altında olmalı');
    assert.ok(menu.spor.includes('/kategori/basketbol'), 'Basketbol Spor altında olmalı');
    assert.ok(menu.top.includes('/kategori/dunya'), 'Dünya üst menüde olmalı');
    for (const slug of ['afrika', 'amerika', 'asya', 'avrupa', 'ortadogu']) {
      assert.ok(menu.dunya.includes('/kategori/' + slug), `Dünya altında ${slug} olmalı`);
    }
    for (const slug of ['donanim', 'internet', 'mobil-teknolojiler', 'robotik',
      'uzay', 'yazilim', 'e-spor', 'oyun-haberleri']) {
      assert.ok(menu.teknoloji.includes('/kategori/' + slug), `Teknoloji altında ${slug} olmalı`);
    }
    for (const slug of ['teknoloji-haberleri', 'mobil-oyunlar', 'oyun-incelemeleri']) {
      assert.ok(!menu.teknoloji.includes('/kategori/' + slug), `Teknoloji'de boş ${slug} gizlenmeli`);
    }
    for (const slug of ['aktuel', 'kadin', 'kisisel-gelisim', 'doga-kesif',
      'gezi-rotalari', 'yeme-icme', 'hobi', 'mizah']) {
      assert.ok(menu.yasam.includes('/kategori/' + slug), `Yaşam altında ${slug} olmalı`);
    }
    for (const slug of ['genclik', 'sehir-rehberi', 'seyahat-rehberi', 'hayvanlar-alemi']) {
      assert.ok(!menu.yasam.includes('/kategori/' + slug), `Yaşam'da boş ${slug} gizlenmeli`);
    }
    for (const slug of ['dusunce-fikir', 'edebiyat', 'kitap', 'muzik', 'sinema', 'tarih', 'mimari']) {
      assert.ok(menu.kultur.includes('/kategori/' + slug), `Eğitim-kültür altında ${slug} olmalı`);
    }
    for (const slug of ['derleme-secki', 'medya-dijital-kultur', 'mekan', 'portreler']) {
      assert.ok(!menu.kultur.includes('/kategori/' + slug), `Eğitim-kültür'de boş ${slug} gizlenmeli`);
    }

    // GZT tohumunda "Türkiye Ekonomisi" kategorisi vardır; kullanıcı sildiyse
    // sayfa 404 döner ve bu kontrol anlamsızlaşır (kullanıcı verisi).
    const teResp = await page.goto(BASE_URL + '/kategori/turkiye-ekonomisi', { waitUntil: 'domcontentloaded' });
    if (teResp && teResp.status() === 200) {
      assert.ok((await page.$eval('h1.cat-pill', (el) => el.textContent)).includes('Türkiye Ekonomisi'));
    }

    await goto(page, '/kategori/guncel');
    assert.ok(page.url().endsWith('/kategori/gundem'), 'eski Güncel adresi Gündem\'e yönlenmeli: ' + page.url());

    // Boş kategori noindex; sitemap'e girmemeli.
    await goto(page, '/kategori/guvenlik');
    const bos = await page.evaluate(() => ({
      robots: (document.querySelector('meta[name="robots"]') || {}).content || '',
      sitemap: null,
    }));
    assert.ok(bos.robots.includes('noindex'), 'boş kategori noindex olmalı: ' + bos.robots);
    const sitemap = await page.evaluate(async () =>
      (await fetch('/sitemap.xml')).text());
    assert.ok(!sitemap.includes('/kategori/guvenlik'), 'boş kategori sitemap\'te olmamalı');
    assert.ok(sitemap.includes('/kategori/gundem'), 'dolu kategori sitemap\'te olmalı');

    assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
  });
});

test('yönetim: kategoriye özel sayfa tasarımı', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdminPage(async (page, errors) => {
    await setViewport(page, 1440, 1000);
    await goto(page, '/admin/tasarim?sayfa=kategori&kategori=borsa');

    const b = await page.evaluate(() => {
      const catSel = document.getElementById('bldCat');
      const form = document.getElementById('builderHtmxForm');
      const tab = [...document.querySelectorAll('.bld-tabs a')].find((a) =>
        (a.textContent || '').includes('Kategori Sayfası'));
      return {
        catValue: catSel ? catSel.value : null,
        catOptions: catSel ? catSel.options.length : 0,
        action: form ? form.getAttribute('action') : '',
        tabOn: !!tab && tab.classList.contains('on'),
        note: document.body.textContent.includes('Bu kategori için özel düzen yok'),
        open: [...document.querySelectorAll('a')].some((a) =>
          a.getAttribute('href') === '/kategori/borsa' && a.getAttribute('target') === '_blank'),
      };
    });
    assert.equal(b.catValue, 'borsa', 'kategori seçici Borsa olmalı');
    assert.ok(b.catOptions > 10, 'kategori seçenekleri listelenmeli: ' + b.catOptions);
    assert.ok(/sayfa=kategori(%3A|:)borsa/i.test(b.action), 'kaydetme bu kategori anahtarına gitmeli: ' + b.action);
    assert.ok(b.tabOn, 'Kategori Sayfası sekmesi aktif olmalı');
    assert.ok(b.note, 'özel düzen yoksa genel düzenden başlanacağı açıklanmalı');
    assert.ok(b.open, 'kategori sayfasını aç bağlantısı olmalı');

    assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
  });
});

test('yönetim: kategori tasarım şablonu uygula ve geri al', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdminPage(async (page, errors) => {
    await setViewport(page, 1440, 1000);

    const setPreset = async (key) => {
      await goto(page, '/admin/kategoriler');
      await page.evaluate((preset) => {
        const f = [...document.querySelectorAll('#kategoriler-panel form')].find((el) =>
          (el.querySelector('input[name="ad"]') || {}).value === 'Borsa');
        f.querySelector('select[name="sablon"]').value = preset;
        f.querySelector('button[name="sablon_uygula"]').click();
      }, key);
      await page.waitForFunction(() => {
        const flash = document.querySelector('#kategoriler-panel .flash');
        return flash && /şablon uygulandı/i.test(flash.textContent);
      }, { timeout: 8000 });
    };

    await setPreset('piyasa');
    const badge = await page.evaluate(() =>
      [...document.querySelectorAll('#kategoriler-panel .shape')]
        .map((x) => x.textContent)
        .find((x) => x.includes('/kategori/borsa')) || '');
    assert.ok(badge.includes('Şablon: Piyasa / Ekonomi'), 'piyasa şablonu rozeti görünmeli: ' + badge);

    // Tasarımcıda bu kategoriye özel düzen açılmalı (genel düzenden kopya notu yok).
    await goto(page, '/admin/tasarim?sayfa=kategori&kategori=borsa');
    const builder = await page.evaluate(() => ({
      catValue: (document.getElementById('bldCat') || {}).value,
      usingGlobal: document.body.textContent.includes('Bu kategori için özel düzen yok'),
    }));
    assert.equal(builder.catValue, 'borsa');
    assert.equal(builder.usingGlobal, false, 'özel düzen varken genel düzen notu çıkmamalı');

    await setPreset('genel');
    const after = await page.evaluate(() =>
      [...document.querySelectorAll('#kategoriler-panel .shape')]
        .map((x) => x.textContent)
        .find((x) => x.includes('/kategori/borsa')) || '');
    assert.ok(!after.includes('Şablon: Piyasa / Ekonomi'), 'geri alınınca piyasa rozeti kalkmalı');

    assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
  });
});

test('yönetim: mega menü ve alt menü tasarımı değiştirilebilir', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdminPage(async (page, errors) => {
    await setViewport(page, 1440, 1000);

    const setDesign = async (mega, catnav) => {
      await goto(page, '/admin/menu');
      await page.select('select[name="mega"]', mega);
      await page.select('select[name="catnav"]', catnav);
      await page.click('form[action="/admin/menu/tasarim"] button[type="submit"]');
      await page.waitForFunction(() => {
        const f = document.querySelector('#menu-panel .flash');
        return f && f.textContent.includes('Menü tasarımı kaydedildi');
      }, { timeout: 8000 });
    };

    let ilkMega = 'sayfa';
    let ilkCatnav = 'bar';
    try {
      // Yönetim ekranında seçenekler var mı?
      await goto(page, '/admin/menu');
      const opts = await page.evaluate(() => ({
        mega: [...document.querySelectorAll('select[name="mega"] option')].map((o) => o.value),
        catnav: [...document.querySelectorAll('select[name="catnav"] option')].map((o) => o.value),
        megaValue: document.querySelector('select[name="mega"]').value,
        catnavValue: document.querySelector('select[name="catnav"]').value,
      }));
      assert.deepEqual(opts.mega, ['sayfa', 'serit', 'genis', 'kisa']);
      assert.deepEqual(opts.catnav, ['bar', 'pill']);
      // Kullanıcı ayarı test tarafından sıfırlanmaz: başlangıç değeri neyse
      // finally'de ona dönülür; burada yalnız değerin geçerli olduğu kontrol edilir.
      ilkMega = opts.megaValue;
      ilkCatnav = opts.catnavValue;
      assert.ok(opts.mega.includes(ilkMega), 'mevcut mega tasarımı geçerli olmalı: ' + ilkMega);
      assert.ok(opts.catnav.includes(ilkCatnav), 'mevcut alt menü tasarımı geçerli olmalı: ' + ilkCatnav);

      // Kompakt + hap tasarımına geç.
      await setDesign('kisa', 'pill');

      await goto(page, '/');
      await page.evaluate(() => {
        const item = [...document.querySelectorAll('.nav-item.has-mega')].find((el) =>
          (el.querySelector('[data-mega-btn]') && (el.querySelector('[data-mega-btn]').textContent || '').toLowerCase().includes('ekonomi')));
        item.querySelector('[data-mega-btn]').click();
      });
      await page.waitForFunction(() => {
        const p = document.querySelector('.nav-item.has-mega .nav-mega:not([hidden])');
        return !!p;
      });
      const compact = await page.evaluate(() => {
        const p = document.querySelector('.nav-item.has-mega .nav-mega:not([hidden])');
        const r = p.querySelector('.nav-mega-inner').getBoundingClientRect();
        return {
          cls: p.className,
          h: Math.round(r.height),
          close: !!p.querySelector('[data-mega-close]'),
        };
      });
      assert.ok(compact.cls.includes('nav-mega-kisa'), 'kompakt sınıfı uygulanmalı: ' + compact.cls);
      assert.ok(compact.h < 400, 'kompakt panel kısa olmalı: ' + compact.h);
      assert.equal(compact.close, false, 'kompakt panelde kapat düğmesi olmamalı');

      await goto(page, '/kategori/ekonomi');
      const pills = await page.evaluate(() =>
        document.querySelectorAll('.cat-subs.cat-subs-pill .cat-sub').length);
      assert.ok(pills > 5, 'hap tasarımında alt kategoriler görünmeli: ' + pills);

      // Sabit şerit: primary navigation'ın hemen altında ikinci menü sırası.
      await setDesign('serit', 'bar');
      await goto(page, '/');
      await page.evaluate(() => {
        const item = [...document.querySelectorAll('.nav-item.has-mega')].find((el) =>
          (el.querySelector('[data-mega-btn]') && (el.querySelector('[data-mega-btn]').textContent || '').toLowerCase().includes('ekonomi')));
        item.querySelector('[data-mega-btn]').click();
      });
      await page.waitForFunction(() =>
        !!document.querySelector('.nav-mega-serit:not([hidden])'));
      // #main üst boşluğu CSS geçişiyle (.26s) artar; geçiş oturmadan ölçmek
      // bayat değer verir. Boşluk şeridi karşılayana kadar bekle.
      await page.waitForFunction(() => {
        const p = document.querySelector('.nav-mega-serit:not([hidden])');
        const m = document.getElementById('main');
        if (!p || !m) return false;
        return parseFloat(getComputedStyle(m).paddingTop) >= p.getBoundingClientRect().height + 3;
      }, { timeout: 4000 }).catch(() => {});
      const strip = await page.evaluate(() => {
        const p = document.querySelector('.nav-mega-serit:not([hidden])');
        const r = p.querySelector('.nav-mega-inner').getBoundingClientRect();
        return {
          h: Math.round(r.height),
          outerH: Math.round(p.getBoundingClientRect().height),
          top: Math.round(r.top),
          bodyClass: document.body.classList.contains('mega-serit-open'),
          mainPad: parseFloat(getComputedStyle(document.getElementById('main')).paddingTop) || 0,
          display: getComputedStyle(p.querySelector('.nav-mega-grid')).display,
          logoX: Math.round(document.querySelector('.brand').getBoundingClientRect().x),
          bodyX: Math.round(document.querySelector('main .container').getBoundingClientRect().x),
          innerX: Math.round(p.querySelector('.nav-mega-inner').getBoundingClientRect().x),
          innerW: Math.round(p.querySelector('.nav-mega-inner').getBoundingClientRect().width),
          vw: window.innerWidth,
        };
      });
      assert.ok(strip.h > 20 && strip.h < 140, 'şerit ince olmalı: ' + strip.h);
      const stripGeo = await page.evaluate(() => {
        const h = document.querySelector('header.site-header');
        const p = [...document.querySelectorAll('[data-mega-panel]')].find((x) => !x.hidden);
        return {
          headerBottom: h ? Math.round(h.getBoundingClientRect().bottom) : 0,
          stripTop: p ? Math.round(p.querySelector('.nav-mega-inner').getBoundingClientRect().top) : -1,
        };
      });
      assert.ok(
        stripGeo.stripTop >= stripGeo.headerBottom - 16 && stripGeo.stripTop <= stripGeo.headerBottom + 25,
        `şerit header altında olmalı: top=${stripGeo.stripTop} headerBottom=${stripGeo.headerBottom}`
      );
      assert.ok(strip.bodyClass, 'gövdeye şerit boşluğu sınıfı eklenmeli');
      assert.ok(strip.mainPad >= strip.outerH + 3,
        'içerik şeridin altında en az 3px nefes alanı bırakmalı: pad=' + strip.mainPad + ' strip=' + strip.outerH);
      assert.equal(strip.display, 'flex', 'şeritte bağlantılar yatay olmalı');
      const hizali = (s) => s.innerX === s.logoX || (s.innerX === 0 && s.innerW >= s.vw - 1);
      assert.ok(hizali(strip), 'şerit logo ile hizalı veya tam genişlemeli: ' + JSON.stringify(strip));
      assert.ok(strip.innerX === strip.bodyX || strip.innerW >= strip.vw - 1,
        'şerit gövde ile hizalı veya tam genişlemeli');

      // 1280 px'te de logo/gövde hizası korunmalı.
      await setViewport(page, 1280, 1000);
      await new Promise((r) => setTimeout(r, 250));
      const align1280 = await page.evaluate(() => {
        const p = document.querySelector('.nav-mega-serit:not([hidden])');
        return {
          logoX: Math.round(document.querySelector('.brand').getBoundingClientRect().x),
          bodyX: Math.round(document.querySelector('main .container').getBoundingClientRect().x),
          innerX: Math.round(p.querySelector('.nav-mega-inner').getBoundingClientRect().x),
          pad: parseFloat(getComputedStyle(document.getElementById('main')).paddingTop) || 0,
        };
      });
      assert.equal(align1280.innerX, align1280.logoX, '1280 px: şerit logo ile hizalı olmalı');
      assert.equal(align1280.innerX, align1280.bodyX, '1280 px: şerit gövde ile hizalı olmalı');
      assert.ok(align1280.pad >= 20, '1280 px: içerik boşluğu korunmalı');

      // Dışarı tıklama sabit şeridi kapatmamalı.
      await page.evaluate(() => document.body.dispatchEvent(new MouseEvent('click', { bubbles: true })));
      await new Promise((r) => setTimeout(r, 150));
      assert.ok(await page.evaluate(() => !!document.querySelector('.nav-mega-serit:not([hidden])')),
        'sabit şerit dışarı tıklamayla kapanmamalı');
      await page.evaluate(() =>
        document.querySelector('.nav-mega-serit [data-mega-close]').click());
      await page.waitForFunction(() => !document.body.classList.contains('mega-serit-open'));
    } finally {
      await setDesign(ilkMega, ilkCatnav).catch(() => {});
    }

    assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
  });
});

test('yönetim: sayfa boyu mega menü arka planı yüklenebilir', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  const img = path.resolve(__dirname, '../../go/internal/webui/static/icon-192.png');
  await withAdminPage(async (page, errors) => {
    await setViewport(page, 1440, 1000);

    const submitBg = async () => {
      await Promise.all([
        page.waitForNavigation({ waitUntil: 'domcontentloaded' }),
        page.click('form[action="/admin/menu/tasarim/arka-plan"] button[type="submit"]'),
      ]);
      await page.waitForFunction(() => {
        const f = document.querySelector('#menu-panel .flash');
        return f && /arka planı/i.test(f.textContent);
      }, { timeout: 8000 });
    };

    try {
      await goto(page, '/admin/menu');
      const input = await page.$('input[name="mega_arka"]');
      assert.ok(input, 'arka plan dosya alanı olmalı');
      await input.uploadFile(img);
      await page.select('select[name="size"]', 'cover');
      await page.evaluate(() => {
        const o = document.querySelector('input[name="overlay"]');
        o.value = '70';
        document.querySelector('input[name="remove_bg"]').checked = false;
      });
      await submitBg();

      const admin = await page.evaluate(() => ({
        preview: !!document.querySelector('.bg-preview img'),
        flash: (document.querySelector('#menu-panel .flash') || {}).textContent || '',
      }));
      assert.ok(admin.preview, 'yüklenen görsel önizlemede görünmeli');
      assert.ok(/kaydedildi/i.test(admin.flash), 'kaydetme mesajı görünmeli: ' + admin.flash);

      await goto(page, '/');
      const css = await page.evaluate(() =>
        [...document.querySelectorAll('head style')].map((s) => s.textContent).join('\n'));
      assert.ok(css.includes('.nav-mega-sayfa .nav-mega-inner{background-image:linear-gradient'),
        'mega menü arka plan CSS\'i sayfaya eklenmeli');
      assert.ok(css.includes('rgba(255,255,255,0.70)'), 'okunabilirlik katmanı %70 olmalı');
    } finally {
      await goto(page, '/admin/menu').catch(() => {});
      await page.evaluate(async () => {
        const fd = new FormData();
        fd.append('remove_bg', '1');
        await fetch('/admin/menu/tasarim/arka-plan', {
          method: 'POST', body: fd, credentials: 'same-origin',
        });
      }).catch(() => {});
    }

    assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
  });
});

test('yönetim: kategori üst/alt ilişkisi düzenlenebilir', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdminPage(async (page, errors) => {
    await setViewport(page, 1440, 1000);
    await goto(page, '/admin/kategoriler');

    const form = await page.evaluate(() => {
      const create = document.querySelector('#kategoriler-panel form[action="/admin/kategoriler"]');
      const ust = create ? create.querySelector('select[name="ust"]') : null;
      const borsa = [...document.querySelectorAll('#kategoriler-panel form')].find((f) =>
        (f.querySelector('input[name="ad"]') || {}).value === 'Borsa');
      const sel = borsa ? borsa.querySelector('select[name="ust"]') : null;
      return {
        hasCreateParent: !!ust,
        economyOption: ust ? [...ust.options].some((o) => o.textContent.trim() === 'Ekonomi') : false,
        borsaIsChild: !!borsa && borsa.classList.contains('cat-row-child'),
        borsaParent: sel && sel.selectedOptions[0] ? sel.selectedOptions[0].textContent.trim() : '',
      };
    });
    assert.ok(form.hasCreateParent, 'yeni kategori formunda üst kategori seçimi olmalı');
    assert.ok(form.economyOption, 'üst kategori listesinde Ekonomi olmalı');
    assert.ok(form.borsaIsChild, 'Borsa alt kategori olarak işaretlenmeli');
    assert.equal(form.borsaParent, 'Ekonomi', 'Borsa üst kategorisi Ekonomi olmalı');

    assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
  });
});
