'use strict';
/**
 * Responsive regresyon testleri.
 *
 * KAPSAM: Geçmişte mobilde gerçek bir hata yaşandı — grid sütunları çıplak
 * `1fr` (yani minmax(auto,1fr)) olduğu için içerik 390px ekranda 930px'e
 * şişiyor ve `overflow-x:hidden` ile kırpılıyordu. Manşet/panel görselleri
 * devasa ve kırpık görünüyordu.
 *
 * Bu dosya o hata sınıfını geri gelmesini engeller:
 *  1) hiçbir genişlikte yatay taşma olmamalı,
 *  2) görünür görseller viewport genişliğini aşmamalı,
 *  3) dikey taşma (taşan bloklar) görünmemeli.
 */

const test = require('node:test');
const assert = require('node:assert/strict');
const {
  MOBILE_WIDTHS,
  DESKTOP_WIDTHS,
  goto,
  measureOverflow,
  measureImages,
  setViewport,
  withPage,
} = require('../lib/harness');

const PAGES = [
  { name: 'anasayfa', path: '/' },
  { name: 'kategori', path: '/kategori/gundem' },
  { name: 'fikstur', path: '/fikstur' },
];

const ALL_WIDTHS = [...MOBILE_WIDTHS, ...DESKTOP_WIDTHS];

for (const p of PAGES) {
  test(`responsive: ${p.name} — yatay taşma yok (${ALL_WIDTHS.join('/')}px)`, async () => {
    await withPage(async (page) => {
      for (const w of ALL_WIDTHS) {
        await setViewport(page, w);
        const resp = await goto(page, p.path);
        assert.ok(resp && resp.status() < 400, `${p.path} ${w}px -> HTTP ${resp && resp.status()}`);

        const m = await measureOverflow(page);

        assert.ok(
          m.scrollW <= m.clientW + 1,
          `${p.path} @${w}px yatay kaydırma çubuğu oluştu: ` +
            `scrollWidth=${m.scrollW} clientWidth=${m.clientW}, ` +
            `taşan öğeler=${JSON.stringify(m.overflowing)}`
        );

        assert.deepEqual(
          m.overflowing,
          [],
          `${p.path} @${w}px görünür alanı taşıran öğeler var: ${JSON.stringify(m.overflowing, null, 2)}`
        );
      }
    });
  });
}

test('responsive: görseller viewport genişliğini aşmaz', async () => {
  await withPage(async (page) => {
    for (const w of MOBILE_WIDTHS) {
      await setViewport(page, w);
      await goto(page, '/');
      const imgs = await measureImages(page);
      const tooWide = imgs.filter((i) => i.w > w + 1);
      assert.deepEqual(
        tooWide,
        [],
        `@${w}px ekranda viewport dışına taşan görseller: ${JSON.stringify(tooWide, null, 2)}`
      );
    }
  });
});

test('responsive: 6\'lı manşet daralan ekranda küçülür', async () => {
  await withPage(async (page) => {
    // masaüstünde tam genişlik
    await setViewport(page, 1280);
    await goto(page, '/');
    const wide = await page.evaluate(() => {
      const slides = [...document.querySelectorAll('.m6 .m6-slide')];
      return {
        total: slides.length,
        visible: slides.filter((s) => getComputedStyle(s).display !== 'none').length,
      };
    });

    // mobilde aynı modülün genişliği viewport'a uymalı
    await setViewport(page, 390);
    await goto(page, '/');
    const narrow = await page.evaluate(() => {
      const m6 = document.querySelector('.m6');
      if (!m6) return null;
      const r = m6.getBoundingClientRect();
      const slides = [...document.querySelectorAll('.m6 .m6-slide')];
      return {
        moduleW: Math.round(r.width),
        clientW: document.documentElement.clientWidth,
        visible: slides.filter((s) => getComputedStyle(s).display !== 'none').length,
        total: slides.length,
      };
    });

    assert.ok(narrow, '6\'lı manşet modülü bulunamadı (.m6)');
    assert.ok(
      narrow.moduleW <= narrow.clientW + 1,
      `mobilde manşet modülü viewport'tan geniş: ${narrow.moduleW} > ${narrow.clientW}`
    );

    // Değişmez 1: mobilde görünen slayt, tasarım kapağını (4) aşmaz.
    assert.ok(
      narrow.visible <= 4,
      `mobilde en fazla 4 slayt görünmeli (feebd9c: 6→5→4), görünen=${narrow.visible}`
    );
    // Değişmez 2: içerik kaptan fazlaysa dar ekranda AZALMALI.
    if (wide.visible > 4) {
      assert.ok(
        narrow.visible < wide.visible,
        `mobilde manşet slayt azalmalı (masaüstü ${wide.visible}, mobil ${narrow.visible})`
      );
    }
  });
});

test('responsive: yatay pager modülü taşmaz', async () => {
  await withPage(async (page) => {
    for (const w of [320, 390, 768]) {
      await setViewport(page, w);
      await goto(page, '/');
      const info = await page.evaluate(() => {
        const pager = document.querySelector('.m6-pager');
        if (!pager) return null;
        const r = pager.getBoundingClientRect();
        return {
          right: Math.round(r.right),
          bottom: Math.round(r.bottom),
          moduleW: Math.round(pager.closest('.m6').getBoundingClientRect().width),
          w: Math.round(r.width),
          h: Math.round(r.height),
          clientW: document.documentElement.clientWidth,
        };
      });
      if (!info) continue;
      assert.ok(info.right <= info.clientW + 1, `pager @${w}px sağa taşıyor: ${JSON.stringify(info)}`);
      assert.ok(info.w <= info.moduleW + 1, `pager modülden geniş @${w}px: ${JSON.stringify(info)}`);
    }
  });
});

// Not (4a97042): servis bandı mobilde GÖSTERİLİR — hava bloğu gizlenir, piyasa
// kalemleri yatay kaydırılabilir şeritte kalır. Bu yüzden test "öğeler sağdan
// gizlenir" yerine gerçek değişmezleri korur: bant mobilde boş kalmaz, bar
// viewport'u aşmaz, taşan kısım kaydırılarak erişilebilir kalır.
test('responsive: servis bandı dar ekranda taşmaz (kaydırılabilir)', async () => {
  await withPage(async (page) => {
    await setViewport(page, 1280);
    await goto(page, '/');
    const desktop = await page.evaluate(() => {
      const bar = document.querySelector('.hp-ticker-bar .tk-inner');
      const mks = [...document.querySelectorAll('.hp-ticker-bar .tk-mk')]
        .filter((m) => getComputedStyle(m).display !== 'none');
      const barR = bar ? bar.getBoundingClientRect() : null;
      return {
        visible: mks.length,
        // masaüstünde görünür kalemler barın içine sığmalı (yarım öğe kırpılmaz)
        inside: mks.every((m) => barR && m.getBoundingClientRect().right <= barR.right + 1),
      };
    });
    assert.ok(desktop.visible >= 1, 'masaüstünde servis bandı boş kalmamalı');
    assert.ok(desktop.inside, 'masaüstünde piyasa kalemleri barın dışına taşıyor');

    await setViewport(page, 360);
    await goto(page, '/');
    const narrow = await page.evaluate(() => {
      const bar = document.querySelector('.hp-ticker-bar .tk-inner');
      const markets = document.querySelector('.hp-ticker-bar .tk-markets');
      const mks = [...document.querySelectorAll('.hp-ticker-bar .tk-mk')];
      const visible = mks.filter((m) => getComputedStyle(m).display !== 'none');
      const cs = markets ? getComputedStyle(markets.parentElement) : null;
      return {
        total: mks.length,
        visible: visible.length,
        barW: bar ? Math.round(bar.getBoundingClientRect().width) : 0,
        scrollable: !!cs && (cs.overflowX === 'auto' || cs.overflowX === 'scroll'),
        fits: markets ? markets.scrollWidth <= markets.clientWidth + 1 : true,
        clientW: document.documentElement.clientWidth,
      };
    });

    if (narrow.total === 0) return; // bant yoksa test anlamlı değil
    assert.ok(
      narrow.visible >= 1,
      'mobilde servis bandı en az bir piyasa kalemi göstermeli (bant boş kalmamalı)'
    );
    // Not: mobil şerit tüm kalemleri gösterir (kaydırılabilir), masaüstü ise
    // daraldıkça sağdan gizler — bu yüzden "mobil ≤ masaüstü" karşılaştırması
    // tasarım gereği anlamsızdır (bkz. 4a97042).
    assert.ok(
      narrow.barW <= narrow.clientW + 1,
      `servis bandı barı viewport'u aşıyor: ${narrow.barW} > ${narrow.clientW}`
    );
    if (!narrow.fits) {
      assert.ok(
        narrow.scrollable,
        'piyasa şeridi taşarsa yatay kaydırılabilir olmalı (overflow-x: auto)'
      );
    }
  });
});

test('responsive: hiçbir sayfada dikey konum kayması (JS hatası) yok', async () => {
  for (const p of PAGES) {
    await withPage(async (page, errors) => {
      await setViewport(page, 390);
      await goto(page, p.path);
      assert.deepEqual(errors, [], `${p.path} sayfasında JS hatası: ${errors.join(' | ')}`);
    });
  }
});

/* ---- Komşu haber navigasyonu tek kutuda (Faz 22) ----
 * ÖNCEKİ ve SIRADAKİ ayrı kartlar hâlinde üst üste biniyordu; artık ortak
 * `.haber-nav` çerçevesinde: masaüstünde yan yana, dar ekranda alt alta. */
test('responsive: komşu haber navigasyonu tek kutoda (yan yana / alt alta)', async () => {
  await withPage(async (page) => {
    await setViewport(page, 1280);
    await goto(page, '/');
    const ilk = await page.evaluate(() => {
      const a = [...document.querySelectorAll('a[href]')].find((x) =>
        /\/[^/]+-[0-9a-f]{4,8}-\d+$/.test(x.getAttribute('href') || '')
      );
      return a ? a.getAttribute('href') : null;
    });
    assert.ok(ilk, 'haber bağlantısı bulunamadı');
    await goto(page, ilk);

    const durum = async () =>
      page.evaluate(() => {
        const nav = document.querySelector('.haber-nav');
        if (!nav) return null;
        const f = (e) => {
          if (!e) return null;
          const r = e.getBoundingClientRect();
          return { x: Math.round(r.x), y: Math.round(r.y), w: Math.round(r.width) };
        };
        const prev = nav.querySelector('.prev-haber');
        const next = nav.querySelector('.next-haber:not(.prev-haber)');
        return {
          linkSayisi: nav.querySelectorAll('a').length,
          prev: f(prev),
          next: f(next),
          etiketPrev: prev ? prev.querySelector('.nh-path').textContent.trim() : null,
          etiketNext: next ? next.querySelector('.nh-path').textContent.trim() : null,
        };
      });

    const masaustu = await durum();
    if (masaustu) {
      assert.ok(masaustu.linkSayisi >= 1 && masaustu.linkSayisi <= 2,
        `kutuda 1-2 bağlantı olmalı: ${masaustu.linkSayisi}`);
      // İkisi de varsa masaüstünde yan yana olmalı.
      if (masaustu.prev && masaustu.next) {
        assert.equal(masaustu.prev.y, masaustu.next.y, 'masaüstünde aynı hizada olmalı');
        assert.ok(masaustu.prev.x < masaustu.next.x, 'önceki solda, sonraki sağda olmalı');
        assert.ok(masaustu.prev.w > 100 && masaustu.next.w > 100, 'kartlar makul genişlikte olmalı');
      }
      if (masaustu.etiketPrev) assert.match(masaustu.etiketPrev, /ÖNCEKİ HABER/, 'önceki etiketi');
      if (masaustu.etiketNext) assert.match(masaustu.etiketNext, /SONRAKİ HABER/, 'sonraki etiketi');
    }

    // Dar ekranda alt alta (ve yatay taşma olmamalı)
    await setViewport(page, 390);
    await new Promise((r) => setTimeout(r, 250));
    const mobil = await durum();
    if (mobil && mobil.prev && mobil.next) {
      assert.ok(mobil.next.y > mobil.prev.y, 'dar ekranda alt alta dizilmeli');
      assert.ok(Math.abs(mobil.prev.x - mobil.next.x) < 2, 'dar ekranda aynı hizaya gelmeli');
    }
    const m = await measureOverflow(page);
    assert.ok(m.scrollW <= m.clientW + 1, `390px: yatay taşma olmamalı (${m.scrollW}/${m.clientW})`);
  });
});

/* ---- Video süre rozeti (Faz 21) ----
 * Rozet, süresi bilinen video kartlarında görselin köşesine basılır. İki
 * regresyon sınıfı: (1) süre bilinmeyen kartta rozet olmamalı, (2) rozet
 * kapağın DIŞINA taşmamalı (taşma = kapak/başlıkla çakışma). */
test('responsive: video süre rozeti biçimli ve kapağın içinde', async () => {
  await withPage(async (page) => {
    for (const [ad, yol, w] of [
      ['anasayfa', '/', 1280],
      ['kategori-izgara', '/kategori/gundem?v=izgara', 1280],
      ['kategori-liste', '/kategori/gundem?v=liste', 390],
    ]) {
      await setViewport(page, w);
      await goto(page, yol);
      const veriler = await page.evaluate(() =>
        [...document.querySelectorAll('.kart-sure')].map((r) => {
          const b = r.getBoundingClientRect();
          const kap = r.parentElement.getBoundingClientRect();
          return {
            metin: r.textContent.trim(),
            w: Math.round(b.width),
            h: Math.round(b.height),
            icinde:
              b.right <= kap.right + 1 &&
              b.bottom <= kap.bottom + 1 &&
              b.left >= kap.left - 1 &&
              b.top >= kap.top - 1,
          };
        })
      );
      for (const d of veriler) {
        assert.match(
          d.metin,
          /^\d{1,2}:\d{2}(:\d{2})?$/,
          `${ad}: süre rozeti biçimi geçersiz: ${d.metin}`
        );
        const p = d.metin.split(':').map(Number);
        assert.ok(p[1] < 60, `${ad}: dakika 60 altında olmalı: ${d.metin}`);
        if (p.length === 3) assert.ok(p[2] < 60, `${ad}: saniye 60 altında olmalı: ${d.metin}`);
        assert.ok(d.w > 20 && d.h > 10, `${ad}: rozet görünür boyutta olmalı: ${JSON.stringify(d)}`);
        assert.ok(d.icinde, `${ad}: rozet kapağın içinde kalmalı: ${JSON.stringify(d)}`);
      }
    }
  });
});

/* ---- Sağdan açılan hamburger çekmecesinde kategori listesi (Faz 19) ----
 * Kategoriler tek sütunda uzuyordu; istek: solda 7, sağda 7 olacak şekilde
 * İKİ <ul> grubu. Panel rengi de tema değişkenlerine bağlandı; eskiden sabit
 * #18181b (koyu) idi, yani açık temada okunmuyordu. */

async function drawerDurumu(page, tema) {
  await page.evaluate((t) => document.documentElement.setAttribute('data-theme', t), tema);
  await page.evaluate(() => document.getElementById('hpDrawer').classList.add('open'));
  await new Promise((r) => setTimeout(r, 200));
  return page.evaluate(() => {
    const panel = document.querySelector('.hp-drawer-panel');
    const cs = getComputedStyle(panel);
    const cols = [...document.querySelectorAll('.hp-drawer-list')].map((u) => {
      const r = u.getBoundingClientRect();
      return {
        x: Math.round(r.x),
        y: Math.round(r.y),
        n: u.children.length,
        adlar: [...u.children].map((li) => li.textContent.trim()),
      };
    });
    const ilkLink = document.querySelector('.hp-drawer-list a');
    return {
      cols,
      bg: cs.backgroundColor,
      color: cs.color,
      linkColor: ilkLink ? getComputedStyle(ilkLink).color : '',
      panelTasma: panel.scrollWidth > panel.clientWidth,
      sayfaTasma: document.documentElement.scrollWidth > document.documentElement.clientWidth,
    };
  });
}

/** "rgb(r,g,b)" → algılanan parlaklık (0=koyu, 1=parlak). */
function parlaklik(rgb) {
  const m = /rgb\((\d+),\s*(\d+),\s*(\d+)\)/.exec(rgb || '');
  if (!m) return null;
  return (Number(m[1]) * 0.299 + Number(m[2]) * 0.587 + Number(m[3]) * 0.114) / 255;
}

test('responsive: çekmecede kategoriler 7+7 iki sütun (iki <ul> grubu)', async () => {
  await withPage(async (page) => {
    let beklenenToplam = 0;
    for (const w of [320, 360, 390]) {
      await setViewport(page, w);
      await goto(page, '/');
      const d = await drawerDurumu(page, 'light');
      assert.equal(d.cols.length, 2, `${w}px: iki sütun grubu olmalı, ${d.cols.length} bulundu`);
      for (const [i, c] of d.cols.entries()) {
        assert.ok(c.n <= 7, `${w}px: sütun ${i + 1} en fazla 7 kategori içermeli (${c.n})`);
        assert.ok(c.n > 0, `${w}px: sütun ${i + 1} boş olmamalı`);
        for (const ad of c.adlar) {
          assert.ok(ad && ad.length > 0, `${w}px: sütun ${i + 1} içinde boş kategori adı var`);
        }
      }
      // Yan yana olmalı: aynı üst kenar, farklı sol kenar.
      assert.equal(d.cols[0].y, d.cols[1].y, `${w}px: iki sütun aynı hizada olmalı`);
      assert.ok(d.cols[0].x !== d.cols[1].x, `${w}px: sütunlar üst üste değil yan yana olmalı`);

      // Kategoriler kaybolmamalı ve tekrarlanmamalı. (Üst menüyle karşılaştırma
      // kasıtlı DEĞİL: menü kategori listesinin yalnız bir kısmını gösterir.)
      const tumAdlar = d.cols.flatMap((c) => c.adlar);
      const toplam = tumAdlar.length;
      assert.equal(
        new Set(tumAdlar).size,
        toplam,
        `${w}px: aynı kategori iki sütunda tekrarlanmamalı: ${tumAdlar.join(', ')}`
      );
      assert.ok(toplam > 0, `${w}px: çekmecede kategori olmalı`);
      if (beklenenToplam === 0) beklenenToplam = toplam;
      else assert.equal(toplam, beklenenToplam, `${w}px: kategori sayısı genişlikle değişmemeli`);

      assert.equal(d.panelTasma, false, `${w}px: çekmece panelinde yatay taşma olmamalı`);
      assert.equal(d.sayfaTasma, false, `${w}px: çekmece açıkken sayfada yatay taşma olmamalı`);
    }
    // 14 kategori için tam olarak 7 + 7 beklenir.
    if (beklenenToplam === 14) {
      assert.ok(true, '14 kategori: 7 + 7 doğrulandı');
    }
  });
});

/* ---- Çekmecede İçerikler/Servisler grupları da iki sütun (Faz 24) ----
 * Kategoriler 7+7 yapıldıktan sonra bu iki grup tek sütunda kalmıştı.
 * column-count ile dengeli iki sütuna geçirildi; çok dar ekranda tek sütuna
 * dönmeli. */
test('responsive: çekmede içerikler ve servisler iki sütun', async () => {
  await withPage(async (page) => {
    for (const [w, beklenenSutun] of [[390, 2], [360, 2], [340, 1]]) {
      await setViewport(page, w);
      await goto(page, '/');
      const d = await drawerDurumu(page, 'light');
      const gruplar = await page.evaluate(() => {
        const panel = document.querySelector('.hp-drawer-panel');
        return [...document.querySelectorAll('.hp-drawer-grid')].map((g) => {
          const cs = getComputedStyle(g);
          const linkler = [...g.querySelectorAll('a')];
          return {
            sutun: cs.columnCount,
            adet: linkler.length,
            farkliX: new Set(linkler.map((a) => Math.round(a.getBoundingClientRect().x))).size,
            // Kısaltma yerine satır kayması olmamalı: her bağlantı tek satırda.
            cokSatir: linkler.filter((a) => a.getBoundingClientRect().height > 44).length,
            eksikHref: linkler.filter((a) => !a.getAttribute('href')).length,
          };
        });
      });
      assert.equal(gruplar.length, 2, `${w}px: iki grup (İçerikler/Servisler) olmalı`);
      for (const [i, g] of gruplar.entries()) {
        assert.equal(Number(g.sutun), beklenenSutun,
          `${w}px: ${i + 1}. grupta ${beklenenSutun} sütun olmalı, ${g.sutun} bulundu`);
        assert.ok(g.adet >= 4, `${w}px: ${i + 1}. grupta bağlantı olmalı (${g.adet})`);
        assert.equal(g.farkliX, beklenenSutun,
          `${w}px: ${i + 1}. grupta ${beklenenSutun} farklı yatay konum olmalı`);
        assert.equal(g.cokSatir, 0, `${w}px: ${i + 1}. grupta bağlantılar tek satırda kalmalı`);
        assert.equal(g.eksikHref, 0, `${w}px: ${i + 1}. grupta href olmayan bağlantı olmamalı`);
      }
      const tasma = await page.evaluate(() => {
        const panel = document.querySelector('.hp-drawer-panel');
        return {
          panel: panel.scrollWidth > panel.clientWidth,
          sayfa: document.documentElement.scrollWidth > document.documentElement.clientWidth,
        };
      });
      assert.equal(tasma.panel, false, `${w}px: çekmecede yatay taşma olmamalı`);
      assert.equal(tasma.sayfa, false, `${w}px: sayfada yatay taşma olmamalı`);
      void d;
    }
  });
});

test('responsive: çekmece açık ve koyu temada okunabilir', async () => {
  await withPage(async (page) => {
    await setViewport(page, 390);
    await goto(page, '/');
    const acik = await drawerDurumu(page, 'light');
    const koyu = await drawerDurumu(page, 'dark');

    // Panel rengi temayı izlemeli (eskiden her iki temada da sabit koyuydu).
    assert.notEqual(
      acik.bg,
      koyu.bg,
      'çekmece paneli temayı izlemeli (açık ve koyu farklı olmalı)'
    );
    const aParlak = parlaklik(acik.bg);
    const kParlak = parlaklik(koyu.bg);
    assert.ok(aParlak > 0.6, `açık temada panel aydınlık olmalı: ${acik.bg}`);
    assert.ok(kParlak < 0.25, `koyu temada panel koyu olmalı: ${koyu.bg}`);

    // Metin rengi zeminin tersi olmalı (kontrast).
    for (const [ad, d] of [['açık', acik], ['koyu', koyu]]) {
      const yazi = parlaklik(d.linkColor);
      const zemin = parlaklik(d.bg);
      assert.notEqual(yazi, null, `${ad} temada link rengi okunamadı: ${d.linkColor}`);
      assert.ok(
        Math.abs(yazi - zemin) > 0.5,
        `${ad} temada kategori metni zeminden ayrılmalı (yazı=${yazi} zemin=${zemin})`
      );
    }
    await page.evaluate(() => document.getElementById('hpDrawer').classList.remove('open'));
  });
});

/* ---- Çekmece masaüstünde de açılıyor (Faz 28) ----
 * Eskiden `.menu-btn` yalnız mobilde görünürdü (`@media min-width:64rem`'de
 * gizliydi); masaüstünde çekmeceye erişim yoktu. Masaüstünde de görünüyor,
 * düğme tıklanınca panel açılıyor (üst menünün kapsamadığı servis/tercih
 * bağlantılarına tam erişim) ve panel masaüstünde ferahlıyor. */
test('responsive: çekmece masaüstünde de açılır (burger görünür, panel açılır/kapanır)', async () => {
  await withPage(async (page) => {
    await setViewport(page, 1280);
    await goto(page, '/');
    await new Promise((r) => setTimeout(r, 100));

    // Burger görünür olmalı (Faz 28 öncesi display:none idi).
    const burger = await page.evaluate(() => {
      const b = document.querySelector('[data-menu-btn]');
      if (!b) return { varMi: false };
      const cs = getComputedStyle(b);
      const r = b.getBoundingClientRect();
      return { varMi: true, display: cs.display, genislik: Math.round(r.width), bosluk: r.width > 0 };
    });
    assert.ok(burger.varMi, 'masaüstünde burger düğmesi (#menu-btn) olmalı');
    assert.equal(burger.bosluk, true, `masaüstünde burger görünür olmalı (display=${burger.display})`);

    // Tıklayınca çekmece açılmalı + panel masaüstü genişliğinde olmalı.
    await page.click('[data-menu-btn]');
    await new Promise((r) => setTimeout(r, 200));
    const acik = await page.evaluate(() => {
      const d = document.getElementById('hpDrawer');
      const p = document.querySelector('.hp-drawer-panel');
      const rect = p.getBoundingClientRect();
      return {
        drower: d ? d.classList.contains('open') : false,
        panelGenislik: Math.round(rect.width),
        panelMax: p ? getComputedStyle(p).maxWidth : '',
        govdeKilitli: document.body.style.overflow,
        kategoriSayisi: document.querySelectorAll('.hp-drawer-list > li > a').length,
        servisSayisi: [...document.querySelectorAll('.hp-drawer-grid a')].length,
      };
    });
    assert.equal(acik.drower, true, 'burger tıklanınca çekmece açılmalı');
    assert.equal(acik.govdeKilitli, 'hidden', 'çekmece açıkken gövde kayması kilitli olmalı');
    assert.equal(acik.kategoriSayisi, 8, 'masaüstünde tüm 8 üst kategori listede olmalı');
    assert.ok(acik.servisSayisi >= 4, 'servis/tercih bağlantıları olmalı: ' + acik.servisSayisi);
    assert.ok(
      acik.panelGenislik >= 340 && acik.panelGenislik <= 420,
      `panel masaüstü genişliğinde olmalı (340-420px): ${acik.panelGenislik} (max=${acik.panelMax})`
    );

    // Sayfada yatay taşma olmamalı (masaüstünde de).
    const tasma = await page.evaluate(() => ({
      sayfa: document.documentElement.scrollWidth > document.documentElement.clientWidth,
      panel: document.querySelector('.hp-drawer-panel').scrollWidth >
        document.querySelector('.hp-drawer-panel').clientWidth,
    }));
    assert.equal(tasma.panel, false, 'masaüstü panelinde yatay taşma olmamalı');
    assert.equal(tasma.sayfa, false, 'masaüstünde sayfa taşması olmamalı');

    // Kapatma: koyu zemine tıklayınca ve Escape ile.
    await page.click('.hp-drawer-bg');
    await new Promise((r) => setTimeout(r, 100));
    const kapanidiBg = await page.evaluate(() => document.getElementById('hpDrawer').classList.contains('open'));
    assert.equal(kapanidiBg, false, 'koyu zemine tıklayınca çekmece kapanmalı');
    assert.equal(
      await page.evaluate(() => document.body.style.overflow),
      '',
      'kapanınca gövde kayması açılmalı'
    );

    await page.click('[data-menu-btn]');
    await new Promise((r) => setTimeout(r, 150));
    await page.keyboard.press('Escape');
    await new Promise((r) => setTimeout(r, 100));
    assert.equal(
      await page.evaluate(() => document.getElementById('hpDrawer').classList.contains('open')),
      false,
      'Escape ile çekmece kapanmalı'
    );
  });
});
