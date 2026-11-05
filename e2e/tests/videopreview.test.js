'use strict';
/**
 * "Önerilen Video" seçicide kapak önizlemesi testleri (Faz 12c).
 *
 * Kapsam:
 *  1) Düzenleme formunda video seçici + önizleme kutusu var
 *  2) Seçim değişince önizleme açılır ve kapak görseli seçilen videonunki olur
 *  3) "Otomatik seç"e dönünce önizleme gizlenir
 *  4) Önizleme görseli gerçekten çizilir (0 boyut = kırık)
 *
 * Not: admin gerektirir; TB_SESSION verilmezse atlanır.
 */

const test = require('node:test');
const assert = require('node:assert/strict');
const { goto, withAdminPage } = require('../lib/harness');

const SID = process.env.TB_SESSION;

/** İçerik listesinden ilk düzenlenebilir içeriğin URL'sini bulur. */
async function firstContentEditUrl(page) {
  await goto(page, '/admin/icerikler');
  if (page.url().includes('/admin/giris')) return null;
  return page.evaluate(() => {
    const links = [...document.querySelectorAll('a[href^="/admin/icerikler/"]')];
    const hit = links.find((a) => {
      const m = a.getAttribute('href').match(/^\/admin\/icerikler\/(\d+)$/);
      return m && Number(m[1]) > 0;
    });
    return hit ? hit.getAttribute('href') : null;
  });
}

async function openForm(page) {
  const url = await firstContentEditUrl(page);
  if (!url) return false;
  await goto(page, url);
  return !page.url().includes('/admin/giris');
}

test('video seçici kapak önizlemesi: seçim değişince kapak güncellenir', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdminPage(async (page, errors) => {
    const ok = await openForm(page);
    assert.ok(ok, 'içerik düzenleme formu açıldı: ' + page.url());

    assert.ok(await page.$('#relVideoSel'), 'video seçici (#relVideoSel) olmalı');
    assert.ok(await page.$('#relVideoPrev'), 'önizleme kutusu (#relVideoPrev) olmalı');

    const opts = await page.evaluate(() =>
      [...document.querySelectorAll('#relVideoSel option')]
        .filter((o) => o.value)
        .map((o) => ({
          value: o.value,
          cover: o.getAttribute('data-cover') || '',
          slug: o.getAttribute('data-slug') || '',
          title: o.textContent.trim(),
        }))
    );
    assert.ok(opts.length > 0, 'en az bir video içeriği listelenmeli');

    // 1) boş seçim -> önizleme gizli
    await page.evaluate(() => {
      const s = document.getElementById('relVideoSel');
      s.value = '';
      s.dispatchEvent(new Event('change'));
    });
    assert.equal(
      await page.evaluate(() => document.getElementById('relVideoPrev').hidden),
      true,
      'boş seçimde önizleme gizli olmalı'
    );

    // 2) ilk videoyu seç -> önizleme açılır, kapak ve başlık eşleşir
    const first = opts[0];
    await page.evaluate((v) => {
      const s = document.getElementById('relVideoSel');
      s.value = v;
      s.dispatchEvent(new Event('change'));
    }, first.value);

    const st = await page.evaluate(() => {
      const box = document.getElementById('relVideoPrev');
      const img = document.getElementById('relVideoImg');
      const th = document.getElementById('relVideoThumb');
      return {
        hidden: box.hidden,
        imgHidden: img.hidden,
        src: img.getAttribute('src') || '',
        href: th.getAttribute('href') || '',
        title: document.getElementById('relVideoTitle').textContent.trim(),
      };
    });
    assert.equal(st.hidden, false, 'video seçilince önizleme görünür olmalı');
    assert.equal(st.title, first.title, 'önizleme başlığı seçilen videonun başlığı olmalı');

    if (first.cover) {
      assert.equal(st.imgHidden, false, 'kapak varsa görsel görünür olmalı');
      assert.equal(st.src, first.cover, 'kapak görseli seçilen videonunki olmalı');
      // 3) görsel gerçekten çizilir
      const size = await page.evaluate(() => {
        const r = document.getElementById('relVideoImg').getBoundingClientRect();
        return { w: Math.round(r.width), h: Math.round(r.height) };
      });
      assert.ok(size.w > 40 && size.h > 20, `kapak önizlemesi çizilmeli: ${JSON.stringify(size)}`);
    }
    if (first.slug) {
      assert.equal(st.href, '/' + first.slug, 'önizleme bağlantısı videonun sayfasına gitmeli');
    }

    // 4) ikinci (varsa) videoya geç -> kapak değişir
    if (opts.length > 1 && opts[1].cover && opts[1].cover !== first.cover) {
      await page.evaluate((v) => {
        const s = document.getElementById('relVideoSel');
        s.value = v;
        s.dispatchEvent(new Event('change'));
      }, opts[1].value);
      const src2 = await page.evaluate(
        () => document.getElementById('relVideoImg').getAttribute('src') || ''
      );
      assert.equal(src2, opts[1].cover, 'seçim değişince kapak güncellenmeli');
    }

    assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
  });
});

test('video seçici kategori filtresi: listeyi daraltır ve aramayla birleşir', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdminPage(async (page, errors) => {
    const ok = await openForm(page);
    assert.ok(ok, 'içerik düzenleme formu açıldı: ' + page.url());

    const cats = await page.evaluate(() => {
      const s = document.getElementById('relVideoCat');
      if (!s) return null;
      return [...s.options].filter((o) => o.value).map((o) => ({ v: o.value, ad: o.textContent.trim() }));
    });
    if (!cats) return t.skip('video seçicide kategori filtresi yok (tek kategori/veri yok)');

    const opts = await page.evaluate(() =>
      [...document.querySelectorAll('#relVideoSel option')]
        .filter((o) => o.value)
        .map((o) => ({ v: o.value, cat: o.getAttribute('data-cat') || '' }))
    );
    if (opts.length < 2) {
      return t.skip('filtreyi sınamak için en az 2 video gerekli');
    }

    // Filtrelenebilir bir kategori seç (listede gerçekten bulunan)
    const secilebilir = cats.find((c) => opts.some((o) => o.cat === c.v));
    assert.ok(secilebilir, 'listede geçen en az bir kategori olmalı');

    await page.evaluate((v) => {
      const s = document.getElementById('relVideoCat');
      s.value = v;
      s.dispatchEvent(new Event('change'));
    }, secilebilir.v);

    const sonra = await page.evaluate(() => ({
      items: [...document.querySelectorAll('#relVideoSel option')]
        .filter((o) => o.value)
        .map((o) => ({ cat: o.getAttribute('data-cat') || '', slug: o.getAttribute('data-slug') || '' })),
      count: document.getElementById('relVideoCount').textContent.trim(),
    }));
    assert.ok(sonra.items.length >= 1, 'filtre en az bir eşleşme bırakmalı');
    assert.ok(
      sonra.items.every((o) => o.cat === secilebilir.v),
      'yalnız seçilen kategorinin videoları kalmalı'
    );
    assert.ok(sonra.items.every((o) => o.slug.length > 0), 'data-slug filtrelemeden sonra da korunmalı');
    assert.match(sonra.count, /video eşleşti/, 'sayaç eşleşme adedini göstermeli');

    // Arama + kategori birlikte çalışmalı (AND)
    await page.evaluate(() => {
      const s = document.getElementById('relVideoSearch');
      s.value = 'zzzzzbulunamaz';
      s.dispatchEvent(new Event('input'));
    });
    const birlikte = await page.evaluate(
      () => [...document.querySelectorAll('#relVideoSel option')].filter((o) => o.value).length
    );
    assert.equal(birlikte, 0, 'arama + kategori birlikte sonuç vermediğinde liste boşalmalı');

    // Filtreyi kaldırınca her şey geri dönmeli
    await page.evaluate(() => {
      const c = document.getElementById('relVideoCat');
      const s = document.getElementById('relVideoSearch');
      c.value = '';
      s.value = '';
      c.dispatchEvent(new Event('change'));
      s.dispatchEvent(new Event('input'));
    });
    const geri = await page.evaluate(
      () => [...document.querySelectorAll('#relVideoSel option')].filter((o) => o.value).length
    );
    assert.equal(geri, opts.length, 'filtreler temizlenince seçenekler geri gelmeli');

    assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
  });
});

test('düzenleyici otomatik taslak kaydetme: gösterge ve kirli takibi (POST atmaz)', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdminPage(async (page, errors) => {
    // beforeunload uyarısı testte engel oluşturmasın
    page.on('dialog', (d) => d.accept());
    const ok = await openForm(page);
    assert.ok(ok, 'içerik düzenleme formu açıldı: ' + page.url());

    const kutu = await page.evaluate(() => {
      const k = document.getElementById('asDurum');
      return k
        ? {
            url: k.getAttribute('data-autosave-url') || '',
            id: k.getAttribute('data-content-id') || '',
            metin: (document.getElementById('asDurumMetin') || {}).textContent.trim(),
          }
        : null;
    });
    assert.ok(kutu, 'otomatik kaydetme göstergesi (#asDurum) olmalı');
    assert.match(kutu.id, /^\d+$/, 'gösterge içerik kimliğini taşımalı');
    assert.equal(
      kutu.url,
      '/admin/icerikler/' + kutu.id + '/otomatik-kaydet',
      'sessiz kaydetme adresi doğru olmalı'
    );
    assert.match(kutu.metin, /otomatik kaydedilmedi/, 'başlangıç metni beklenen olmalı: ' + kutu.metin);

    // Yazmaya başlayınca "değişiklik var" durumuna geçmeli (kaydetme BEKLENMEZ:
    // 30 sn debounce var, test bu sürede tamamlanır → canlı veriye dokunulmaz)
    //
    // DİKKAT (Faz 210): Ürün, "kirli" durumuna geçmek için GERÇEK kullanıcı
    // girdisi arar (`dokundu` bayrağı yalnız pointerdown/keydown/paste/cut/drop
    // ile kurulur) — Faz 191 kararı: editörün ve ortak çalışma odasının
    // programatik yazımları sayfayı "değişti" göstermemeli.
    // Bu yüzden sentetik `input` yerine GERÇEK yazma kullanılır.
    await page.click('#f_title');
    await page.keyboard.type('x');
    const kirli = await page.$eval('#asDurumMetin', (e) => e.textContent.trim());
    assert.match(kirli, /değişiklik var/, 'yazınca gösterge "değişiklik var" demeli: ' + kirli);

    assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
  });
});

test('video süresi alanı yalnız "video" türünde görünür', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdminPage(async (page, errors) => {
    // TARİHÇE (Faz 210): Alan artık yalnız "Gelişmiş" grubu OLAN türlerde
    // basılıyor (video → "video süresi", hesaplayici → "hesaplama şeması").
    // Diğer türlerde bölüm hiç çizilmiyor (eskiden boş bir "Gelişmiş"
    // başlığı kalıyordu). Bu yüzden kural şöyle doğrulanır:
    //   video  → alan VAR ve görünür,
    //   diğer  → alan YOK (ya da varsa gizli).
    await goto(page, '/admin/icerikler?tip=video');
    const videoUrl = await page.evaluate(() => {
      const a = [...document.querySelectorAll('a[href^="/admin/icerikler/"]')]
        .find((x) => /^\/admin\/icerikler\/\d+$/.test(x.getAttribute('href') || ''));
      return a ? a.getAttribute('href') : null;
    });
    assert.ok(videoUrl, 'video içeriği bulunmalı');

    await goto(page, videoUrl);
    const videoDurum = await page.evaluate(() => {
      const alan = document.getElementById('vidSureField');
      const girdi = document.querySelector('[name="video_sure"]');
      const tip = document.querySelector('select[name="tip"]');
      return {
        tip: tip ? tip.value : null,
        alan: !!alan,
        gizli: alan ? alan.hidden : null,
        girdi: !!girdi,
      };
    });
    assert.equal(videoDurum.tip, 'video', 'video türü açılmalı: ' + videoDurum.tip);
    assert.ok(videoDurum.alan, 'video türünde süre alanı (#vidSureField) olmalı');
    assert.equal(videoDurum.gizli, false, 'video türünde süre alanı görünmeli');
    assert.ok(videoDurum.girdi, 'süre girdisi (video_sure) olmalı');

    // Video dışı türde alan görünür OLMAMALI (yok ya da gizli).
    const digerUrl = await (async () => {
      await goto(page, '/admin/icerikler?tip=haber');
      return page.evaluate(() => {
        const a = [...document.querySelectorAll('a[href^="/admin/icerikler/"]')]
          .find((x) => /^\/admin\/icerikler\/\d+$/.test(x.getAttribute('href') || ''));
        return a ? a.getAttribute('href') : null;
      });
    })();
    assert.ok(digerUrl, 'haber içeriği bulunmalı');
    await goto(page, digerUrl);
    const digerDurum = await page.evaluate(() => {
      const alan = document.getElementById('vidSureField');
      return { tip: (document.querySelector('select[name="tip"]') || {}).value, var: !!alan, gizli: alan ? alan.hidden : null };
    });
    assert.notEqual(digerDurum.tip, 'video', 'haber türü açılmalı');
    assert.ok(
      !digerDurum.var || digerDurum.gizli === true,
      'video dışı türde süre alanı görünmeli OLMAMALI (var=' + digerDurum.var + ', gizli=' + digerDurum.gizli + ')',
    );

    assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
  });
});

test('video seçici arama: filtre seçenekleri daraltır ve geri alınabilir', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdminPage(async (page, errors) => {
    const ok = await openForm(page);
    assert.ok(ok, 'içerik düzenleme formu açıldı: ' + page.url());

    assert.ok(await page.$('#relVideoSearch'), 'arama kutusu (#relVideoSearch) olmalı');
    assert.ok(await page.$('#relVideoCount'), 'eşleşme sayacı (#relVideoCount) olmalı');

    const opts = await page.evaluate(() =>
      [...document.querySelectorAll('#relVideoSel option')]
        .filter((o) => o.value)
        .map((o) => ({ value: o.value, title: o.textContent.trim(), spot: o.getAttribute('data-spot') || '' }))
    );
    if (opts.length < 2) {
      // Tek videoda filtre anlamlı değil; sayacın yine de çalıştığını doğrula.
      const c1 = await page.$eval('#relVideoCount', (e) => e.textContent.trim());
      assert.match(c1, /Toplam \d+ video/, 'sayaç toplamı göstermeli: ' + c1);
      return;
    }

    // Arama kutusuna ilk videonun başlığından bir kelime yaz -> liste daralmalı
    const word = opts[0].title.split(/\s+/)[0].slice(0, 4);
    assert.ok(word.length >= 2, 'arama için kullanılabilir kelime olmalı: ' + word);

    await page.type('#relVideoSearch', word);
    await new Promise((r) => setTimeout(r, 120));

    const filtered = await page.evaluate(() => {
      const list = [...document.querySelectorAll('#relVideoSel option')].filter((o) => o.value);
      return {
        n: list.length,
        items: list.map((o) => ({
          title: o.textContent.trim(),
          spot: o.getAttribute('data-spot') || '',
        })),
        count: document.getElementById('relVideoCount').textContent.trim(),
      };
    });
    const w = word.toLocaleLowerCase('tr');
    assert.ok(filtered.n >= 1, 'filtre en az bir eşleşme bırakmalı');
    assert.ok(filtered.n <= opts.length, 'filtre seçenek sayısını artırmamalı');
    assert.ok(
      filtered.items.every(
        (it) =>
          it.title.toLocaleLowerCase('tr').includes(w) ||
          it.spot.toLocaleLowerCase('tr').includes(w)
      ),
      'kalan seçenekler terimi başlıkta ya da özette içermeli: ' + JSON.stringify(filtered.items)
    );
    assert.match(filtered.count, /video eşleşti/, 'sayaç eşleşme adedini göstermeli');

    // Önizleme verileri filtrelemeden sonra da korunmalı (data-* kaybolmamalı)
    const attrs = await page.evaluate(() => {
      const list = [...document.querySelectorAll('#relVideoSel option')].filter((o) => o.value);
      return list.map((o) => ({
        cover: o.getAttribute('data-cover'),
        slug: o.getAttribute('data-slug'),
      }));
    });
    assert.ok(
      attrs.every((a) => a.slug && a.slug.length > 0),
      'filtre sonrası data-slug korunmalı'
    );

    // Arama temizlenince tüm seçenekler geri gelmeli
    await page.evaluate(() => {
      const s = document.getElementById('relVideoSearch');
      s.value = '';
      s.dispatchEvent(new Event('input'));
    });
    const restored = await page.evaluate(() => ({
      n: [...document.querySelectorAll('#relVideoSel option')].filter((o) => o.value).length,
      count: document.getElementById('relVideoCount').textContent.trim(),
    }));
    assert.equal(restored.n, opts.length, 'arama temizlenince seçenekler geri gelmeli');
    assert.match(restored.count, /Toplam \d+ video/, 'sayaç toplama dönmeli');

    assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
  });
});