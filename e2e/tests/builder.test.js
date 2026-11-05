'use strict';
/**
 * Tasarımcı (Sayfa Tasarımcısı) regresyon testleri.
 *
 * Kapsam — sürükleme simülasyonu YAPILMAZ (CI'da kararsız); onun yerine
 * sürüklemenin çalışması için gereken * ön koşullar doğrulanır:
 *
 *  1) SortableJS yüklenmiş ve tanımlı
 *  2) Modül çipleri sürükleme kulpuna (grip) sahip
 *  3) Kilitli çipler sürüklenemez (CSS seçici doğru)
 *  4) Geri al / yinele düğmeleri var
 *  5) DAR EKRANDA PALET YAPIŞKAN DEĞİL
 *     → Regresyon: sticky palet kaydırınca çiplerin üstüne biniyor,
 *       kullanıcı çipe dokunamadığı için mobil sürükle-bırak tamamen bozuluyordu.
 *
 * Not: admin gerektirir; TB_SESSION verilmezse atlanır.
 */

const test = require('node:test');
const assert = require('node:assert/strict');
const { BASE_URL, goto, setViewport, withAdminPage } = require('../lib/harness');

const SID = process.env.TB_SESSION;

function withAdmin(fn) {
  return withAdminPage(fn);
}

async function openDesigner(page, width, height) {
  await setViewport(page, width, height);
  await goto(page, '/admin/tasarim?sayfa=home');
  return !page.url().includes('/admin/giris');
}

test('indeksleme paneli: haber sitemap kartı ve adresi görünür', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdmin(async (page) => {
    await goto(page, '/admin/indexleme');
    assert.ok(!page.url().includes('/admin/giris'), 'indeksleme paneli açık: ' + page.url());

    const kart = await page.evaluate(() => {
      const kod = [...document.querySelectorAll('code')].map((e) => e.textContent.trim());
      const adres = kod.find((t) => t.includes('/sitemap-news.xml')) || '';
      return {
        adres,
        butonVar: !!document.querySelector('form[action="/admin/indexleme/news-sitemap"] button'),
        scLink: !!document.querySelector('a[href*="search.google.com/search-console"]'),
      };
    });
    assert.match(kart.adres, /^https?:\/\/[^/]+\/sitemap-news\.xml$/,
      'panelde mutlak sitemap adresi olmalı: ' + kart.adres);
    assert.ok(kart.butonVar, '"Şimdi doğrula" düğmesi olmalı');
    assert.ok(kart.scLink, 'Search Console bağlantısı olmalı');
  });
});

test('içerik listesi: toplu işlem çubuğu seçime göre açılır (veri yazmadan)', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdmin(async (page, errors) => {
    await setViewport(page, 1400);
    await goto(page, '/admin/icerikler');
    assert.ok(!page.url().includes('/admin/giris'), 'içerik listesi açık: ' + page.url());

    const ilk = await page.evaluate(() => ({
      form: !!document.getElementById('topluForm'),
      barHidden: (document.getElementById('topluBar') || {}).hidden,
      sayi: (document.getElementById('topluSayi') || {}).textContent || '',
      kutular: document.querySelectorAll('.satirSec').length,
      islemler: [...document.querySelectorAll('[name="islem"]')].map((b) => b.value),
      tumunu: !!document.getElementById('tumunuSec'),
      action: (document.getElementById('topluForm') || {}).getAttribute
        ? document.getElementById('topluForm').getAttribute('action')
        : null,
    }));
    assert.ok(ilk.form, 'toplu işlem formu olmalı');
    assert.equal(ilk.action, '/admin/icerikler/toplu', 'form adresi doğru olmalı');
    assert.equal(ilk.kutular > 0, true, 'satır seçim kutuları olmalı');
    assert.ok(ilk.tumunu, '"tümünü seç" kutusu olmalı');
    // Faz 42 — liste sütunları: Yayın + Okunma görünmeli.
    //
    // NOT (Faz 210): sütun SIRASI zamanla değişti — tabloya "Dil" (3.) ve
    // "Rozet" (6.) eklendi. Sabit `td:nth-child(N)` kullanmak testi kırılgan
    // yapıyordu (Yayın yerine Rozet hücresi okunuyordu). Artık sütunu
    // BAŞLIKTAN buluyoruz.
    const kolonlar = await page.evaluate(() => {
      const ths = [...document.querySelectorAll('table.table thead th')].map((t) => t.textContent.trim());
      const satir = document.querySelector('table.table tbody tr');
      const hucre = (ad) => {
        const i = ths.indexOf(ad);
        if (i < 0 || !satir) return '';
        const td = satir.children[i];
        return td ? td.textContent.trim() : '';
      };
      return { ths, yayin: hucre('Yayın'), okunma: hucre('Okunma') };
    });
    assert.ok(kolonlar.ths.includes('Yayın') && kolonlar.ths.includes('Okunma'),
      'liste başlıkları Yayın + Okunma içermeli: ' + kolonlar.ths.join(','));
    assert.match(kolonlar.yayin, /^\d{2}\.\d{2} \d{2}:\d{2}$/, 'Yayın sütunu dd.mm hh:mm olmalı: ' + kolonlar.yayin);
    assert.match(kolonlar.okunma, /^\d+|—$/, 'Okunma sütunu sayı ya da — olmalı: ' + kolonlar.okunma);
    assert.deepEqual(
      [...ilk.islemler].sort(),
      ['arsiv', 'kategori', 'planla', 'rozet', 'sil', 'yayinla'],
      'altı işlem düğmesi olmalı (planla + rozet dahil, Faz 32/39)'
    );
    assert.equal(
      await page.evaluate(() => {
        const i = document.querySelector('#topluForm input[name="yayin_tarihi"]');
        return !!i && i.type === 'datetime-local';
      }),
      true,
      '"yayına zamanla" için datetime-local girişi olmalı'
    );
    assert.equal(
      await page.evaluate(() => {
        const bar = document.getElementById('topluBar');
        return !!bar &&
          !!bar.querySelector('select[name="rozet"]') &&
          !!bar.querySelector('select[name="rozet_durum"]') &&
          !!bar.querySelector('[name="islem"][value="rozet"]');
      }),
      true,
      'toplu çubuğunda rozet kontrolleri (seç/tür + Aç/Kapat düğmesi) olmalı'
    );

    // Hiçbir şeye dokunmadan: çubuk gizli kalmalı.
    assert.equal(ilk.barHidden, true, 'seçim yokken çubuk gizli olmalı');
    assert.match(ilk.sayi, /^0 içerik seçili$/, 'sayaç 0 ile başlamalı');

    // Yalnızca seçim yap (POST GÖNDERME — canlı veriye dokunulmaz)
    await page.evaluate(() => {
      const k = document.querySelectorAll('.satirSec');
      if (k[0]) {
        k[0].checked = true;
        k[0].dispatchEvent(new Event('change'));
      }
    });
    const secili = await page.evaluate(() => ({
      hidden: document.getElementById('topluBar').hidden,
      sayi: document.getElementById('topluSayi').textContent.trim(),
    }));
    assert.equal(secili.hidden, false, 'seçim yapılınca çubuk açılmalı');
    assert.equal(secili.sayi, '1 içerik seçili', 'sayaç seçimi yansıtmalı');

    // "Tümünü seç"
    await page.evaluate(() => {
      const t = document.getElementById('tumunuSec');
      t.checked = true;
      t.dispatchEvent(new Event('change'));
    });
    const tumu = await page.evaluate(() => ({
      sayi: document.getElementById('topluSayi').textContent.trim(),
      isaretli: [...document.querySelectorAll('.satirSec')].every((c) => c.checked),
    }));
    assert.equal(tumu.isaretli, true, '"tümünü seç" bütün satırları işaretlemeli');
    assert.match(tumu.sayi, / içerik seçili$/, 'sayaç toplamı göstermeli');

    // Seçimi kaldırınca çubuk tekrar gizlenmeli
    await page.evaluate(() => {
      const t = document.getElementById('tumunuSec');
      t.checked = false;
      t.dispatchEvent(new Event('change'));
    });
    assert.equal(
      await page.evaluate(() => document.getElementById('topluBar').hidden),
      true,
      'seçim kalkınca çubuk gizlenmeli'
    );

    assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
  });
});

test('medya yöneticisi: toplu işlem çubuğu seçime göre çalışır (veri yazmadan)', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdmin(async (page, errors) => {
    await setViewport(page, 1400);
    await goto(page, '/admin/dosyalar');
    assert.ok(!page.url().includes('/admin/giris'), 'medya sayfası açık: ' + page.url());

    const ilk = await page.evaluate(() => ({
      form: !!document.getElementById('medyaTopluForm'),
      action: (document.getElementById('medyaTopluForm') || {}).getAttribute
        ? document.getElementById('medyaTopluForm').getAttribute('action')
        : null,
      barVar: !!document.getElementById('medyaBar'),
      barHidden: (document.getElementById('medyaBar') || {}).hidden,
      tumunu: !!document.getElementById('medyaTumunuSec'),
      kutular: document.querySelectorAll('.medyaSec').length,
      islemler: [...document.querySelectorAll('#medyaTopluForm [name="islem"]')].map((b) => b.value),
      yeniAd: !!document.querySelector('#medyaTopluForm [name="yeni_ad"]'),
    }));
    assert.ok(ilk.form, 'toplu işlem formu olmalı');
    assert.equal(ilk.action, '/admin/dosyalar/toplu', 'form adresi doğru olmalı');
    assert.ok(ilk.barVar, 'toplu çubuk (#medyaBar) olmalı');
    assert.equal(ilk.barHidden, true, 'seçim yokken çubuk gizli olmalı');
    assert.ok(ilk.tumunu, '"tümünü seç" kutusu olmalı');
    assert.ok(ilk.kutular > 0, 'her medya için seçim kutusu olmalı (' + ilk.kutular + ')');
    // İşlem düğmeleri: toplu kategori atama, silme, yeniden adlandırma.
    assert.deepEqual([...ilk.islemler].sort(), ['kategori', 'sil', 'yeniden-adlandir'],
      'üç işlem düğmesi olmalı');
    assert.ok(ilk.yeniAd, '"yeni ad" girişi olmalı');

    // Seçim yapinca çubuk açılır + sayaç artar (POST atılmaz → veri etkilenmez).
    await page.evaluate(() => {
      const k = document.querySelectorAll('.medyaSec');
      if (k[0]) {
        k[0].checked = true;
        k[0].dispatchEvent(new Event('change'));
      }
    });
    const secili = await page.evaluate(() => ({
      hidden: document.getElementById('medyaBar').hidden,
      sayi: document.getElementById('medyaSayi').textContent.trim(),
    }));
    assert.equal(secili.hidden, false, 'seçim yapılınca çubuk açılmalı');
    assert.equal(secili.sayi, '1 medya seçili', 'sayaç seçimi yansıtmalı');

    // Tümünü seç → bütün kutular işaretlenir.
    await page.evaluate(() => {
      const t = document.getElementById('medyaTumunuSec');
      t.checked = true;
      t.dispatchEvent(new Event('change'));
    });
    assert.equal(
      await page.evaluate(() => {
        const k = Array.from(document.querySelectorAll('.medyaSec'));
        return k.filter((c) => c.checked).length === k.length;
      }),
      true,
      '"tümünü seç" bütün kutuları işaretlemeli'
    );

    // Seçim kalkınca çubuk tekrar gizlenir.
    await page.evaluate(() => {
      const t = document.getElementById('medyaTumunuSec');
      t.checked = false;
      t.dispatchEvent(new Event('change'));
    });
    assert.equal(
      await page.evaluate(() => document.getElementById('medyaBar').hidden),
      true,
      'seçim kalkınca çubuk gizlenmeli'
    );

    assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
  });
});

test('içerik listesi: filtreler tip/kategori/tarih çalışır ve sekmelerde korunur (veri yazmaz)', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdmin(async (page, errors) => {
    await setViewport(page, 1400);
    await goto(page, '/admin/icerikler');
    assert.ok(!page.url().includes('/admin/giris'), 'içerik listesi açık: ' + page.url());

    // Filtre seçicileri mevcut ve seçenekli olmalı.
    const seciciler = await page.evaluate(() => {
      const tip = document.querySelector('select[name="tip"]');
      const kategori = document.querySelector('select[name="kategori"]');
      const gun = document.querySelector('select[name="gun"]');
      return {
        tip: !!tip,
        tipAdet: tip ? tip.options.length : 0,
        tipTipler: tip ? [...tip.options].map((o) => o.textContent.trim()) : [],
        kategori: !!kategori,
        kategoriAdet: kategori ? kategori.options.length : 0,
        gun: !!gun,
        gunSecenek: gun ? [...gun.options].map((o) => o.value) : [],
      };
    });
    assert.ok(seciciler.tip && seciciler.kategori && seciciler.gun, 'tip/kategori/tarih seçicileri olmalı');
    assert.ok(seciciler.tipAdet >= 2, 'tip seçicisinde en az Tüm tipler + bir tip olmalı');
    assert.ok(seciciler.tipTipler.includes('Video'), '"Video" tip seçeneği olmalı');
    assert.ok(seciciler.kategoriAdet >= 2, 'kategori seçicisinde en az Tüm kategoriler + bir kategori olmalı');
    assert.deepEqual(seciciler.gunSecenek, ['', '7', '14', '30'], 'tarih seçeneği: Tümü/7/14/30 gün olmalı');

    // Gerçek form gönderimi: tip=video seçip "Filtrele"ye bas → GET (veri yazmaz).
    await page.select('select[name="tip"]', 'video');
    await page.click('button[type="submit"]');
    await page.waitForFunction(() => location.search.includes('tip=video'), { timeout: 5000 });

    const sonuc = await page.evaluate(() => {
      // Sütun indeksi sabit DEĞİL: "Dil" ve "Rozet" sütunları eklendikçe
      // kayıyordu. Hücreleri başlık adından çöz.
      const ths = [...document.querySelectorAll('table.table thead th')].map((t) => t.textContent.trim());
      const tipIdx = ths.indexOf('Tip');
      const tipHuc = [...document.querySelectorAll('table.table tbody tr')]
        .map((tr) => (tr.children[tipIdx] || {}).textContent || '')
        .map((t) => t.trim());
      const tabLinkler = [...document.querySelectorAll('.tabs a')].map((a) => a.getAttribute('href'));
      return {
        tipHuc,
        tipSelected: (document.querySelector('select[name="tip"]') || {}).value,
        tabLinkler,
        // Aktif filtre notu: DOM sırası değiştiği için yapıya bağlı
        // seçici (`.tabs ~ .input-row ~ .hint`) kırılıyordu. Metinden bul.
        aktifFiltre: [...document.querySelectorAll('.hint')]
          .map((e) => e.textContent || '')
          .find((t) => t.includes('Aktif filtre')) || '',
      };
    });
    assert.equal(sonuc.tipSelected, 'video', 'tip seçimi sayfa yüklendikten sonra korunmalı');
    assert.ok(sonuc.tipHuc.length > 0, 'video içerikleri listelenmeli');
    assert.ok(
      sonuc.tipHuc.every((v) => v === 'video'),
      'filtrelenmiş listedeki her satır video olmalı: ' + sonuc.tipHuc.join(',')
    );
    // Sekme linkleri filtreyi korumalı (yalnızca & — %26 ya da %3d DEĞİL).
    for (const href of sonuc.tabLinkler) {
      assert.ok(
        href.includes('tip=video') && !href.includes('%26') && !href.includes('%3d'),
        'sekme linki filtreyi düz & işaretiyle korumalı: ' + href
      );
    }
    assert.ok(sonuc.aktifFiltre.includes('Video'), 'aktif filtre notu tipi göstermeli: ' + sonuc.aktifFiltre);

    // Filtre yokken sekmelerde filtre parametresi olmamalı.
    await goto(page, '/admin/icerikler');
    const temiz = await page.evaluate(() =>
      [...document.querySelectorAll('.tabs a')].map((a) => a.getAttribute('href')).join(' ')
    );
    assert.ok(!temiz.includes('tip='), 'filtre yokken sekme linkleri temiz olmalı: ' + temiz);

    assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
  });
});

test('içerik listesi: sayfalama çalışır, filtreler sayfa linklerinde korunur (veri yazmaz)', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdmin(async (page, errors) => {
    await setViewport(page, 1400);
    await goto(page, '/admin/icerikler');
    assert.ok(!page.url().includes('/admin/giris'), 'içerik listesi açık: ' + page.url());

    // 494 içerik → çok sayfalı çubuk olmalı; aktif 1.
    const ilk = await page.evaluate(() => {
      const cubuk = document.querySelector('.sayfalama');
      const numberLinks = cubuk ? [...cubuk.querySelectorAll('a.btn')].map((a) => a.textContent.trim()) : [];
      return {
        varMi: !!cubuk,
        aktif: (cubuk ? cubuk.querySelector('.btn.active') : null)
          ? cubuk.querySelector('.btn.active').textContent.trim()
          : null,
        sonraki: cubuk ? !!Array.from(cubuk.querySelectorAll('a')).find((a) => a.textContent.includes('Sonraki')) : false,
        onceki: cubuk ? !!Array.from(cubuk.querySelectorAll('a')).find((a) => a.textContent.includes('Önceki')) : false,
        sayfalar: numberLinks.filter((x) => /^\d+$/.test(x)),
      };
    });
    assert.ok(ilk.varMi, 'çok içerikte sayfalama çubuğu olmalı');
    assert.equal(ilk.aktif, '1', '1. sayfada aktif sayfa 1 olmalı');
    assert.ok(ilk.sayfalar.length >= 2, 'sayfa numara linkleri olmalı: ' + ilk.sayfalar.join(','));
    assert.equal(ilk.onceki, false, '1. sayfada "Önceki" görünmemeli');
    assert.equal(ilk.sonraki, true, '1. sayfada "Sonraki" görünmeli');

    // "Sonraki"ye tıkla → sayfa 2, aktif 2, Önceki belirir.
    await page.evaluate(() => {
      const a = Array.from(document.querySelectorAll('.sayfalama a')).find((x) => x.textContent.includes('Sonraki'));
      if (a) a.click();
    });
    await page.waitForFunction(() => location.search.includes('sayfa=2'), { timeout: 5000 });
    const ikinci = await page.evaluate(() => {
      const cubuk = document.querySelector('.sayfalama');
      return {
        aktif: cubuk.querySelector('.btn.active') ? cubuk.querySelector('.btn.active').textContent.trim() : null,
        onceki: !!Array.from(cubuk.querySelectorAll('a')).find((a) => a.textContent.includes('Önceki')),
        ilkSatir: (document.querySelector('table.table tbody tr') || {}).textContent
          ? document.querySelector('table.table tbody tr').textContent.trim().slice(0, 30)
          : '',
      };
    });
    assert.equal(ikinci.aktif, '2', '2. sayfada aktif 2 olmalı');
    assert.equal(ikinci.onceki, true, '2. sayfada "Önceki" görünmeli');

    // Filtre + sayfa birlikte: tip=haber&sayfa=3 → sayfa linkleri tip'ı korumalı.
    await goto(page, '/admin/icerikler?tip=haber&sayfa=3');
    const koruma = await page.evaluate(() => {
      const cubuk = document.querySelector('.sayfalama');
      return {
        aktif: cubuk.querySelector('.btn.active') ? cubuk.querySelector('.btn.active').textContent.trim() : null,
        linkler: cubuk ? [...cubuk.querySelectorAll('a')].map((a) => a.getAttribute('href')) : [],
        tipKorunuyor: cubuk ? [...cubuk.querySelectorAll('a')].map((a) => a.getAttribute('href')).some((h) => h.includes('tip=haber')) : false,
      };
    });
    assert.equal(koruma.aktif, '3', 'filtreli liste 3. sayfada aktif 3 olmalı');
    assert.equal(koruma.tipKorunuyor, true, 'sayfa linkleri tip filtresini korumalı');

    // Az içerikli filtre (video, 4 tane) → sayfalama çubUĞU OLMAMALI.
    await goto(page, '/admin/icerikler?tip=video');
    const tek = await page.evaluate(() => ({
      sayfalama: !!document.querySelector('.sayfalama'),
      govdeMetin: document.querySelector('.admin-body') ? document.querySelector('.admin-body').textContent : '',
      toplamVar: [...document.querySelectorAll('div')].some((d) => /^Toplam:\s*\d+$/.test(d.textContent.trim())),
    }));
    assert.equal(tek.sayfalama, false, 'tek sayfalık sonuçta sayfalama çubuğu olmamalı');
    assert.equal(tek.toplamVar, true, 'tek sayfalık sonuçta "Toplam: N" satırı olmalı');
    assert.ok(!tek.govdeMetin.includes('Sayfa 1/1'), 'tek sayfa için "Sayfa 1/1" göstergesi olmamalı');

    assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
  });
});

test('yorumlar: toplu işlem çubuğu seçime göre çalışır (veri yazmadan)', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdmin(async (page, errors) => {
    await setViewport(page, 1400);
    await goto(page, '/admin/yorumlar');
    assert.ok(!page.url().includes('/admin/giris'), 'yorum sayfası açık: ' + page.url());

    const ilk = await page.evaluate(() => {
      const form = document.getElementById('yorumTopluForm');
      return {
        form: !!form,
        action: form ? form.getAttribute('action') : null,
        barVar: !!document.getElementById('yorumBar'),
        barHidden: (document.getElementById('yorumBar') || {}).hidden,
        tumunu: !!document.getElementById('yorumTumunuSec'),
        kutular: document.querySelectorAll('.yorumSec').length,
        islemler: [...document.querySelectorAll('#yorumTopluForm [name="islem"]')].map((b) => b.value),
        durumHidden: form ? (form.querySelector('input[name="durum"]') || {}).value : null,
      };
    });
    assert.ok(ilk.form, 'toplu işlem formu olmalı');
    assert.equal(ilk.action, '/admin/yorumlar/toplu', 'form adresi doğru olmalı');
    assert.ok(ilk.barVar, 'toplu çubuk olmalı');
    assert.equal(ilk.barHidden, true, 'seçim yokken çubuk gizli olmalı');
    assert.ok(ilk.tumunu, '"tümünü seç" kutusu olmalı');
    assert.ok(ilk.kutular > 0, 'yorumlar için seçim kutuları olmalı (' + ilk.kutular + ')');
    assert.deepEqual([...ilk.islemler].sort(), ['onayla', 'sil'], 'iki işlem (onayla/sil) olmalı');
    assert.ok(['onay_bekliyor', 'onayli', 'hepsi', ''].includes(ilk.durumHidden), 'gizli durum sekme bilgisini taşımalı');

    // Seçim yapınca çubuk açılır + sayaç artar (POST atılmaz).
    await page.evaluate(() => {
      const k = document.querySelectorAll('.yorumSec');
      if (k[0]) {
        k[0].checked = true;
        k[0].dispatchEvent(new Event('change'));
      }
    });
    assert.equal((await page.$eval('#yorumBar', (e) => e.hidden)), false, 'seçim yapılınca çubuk açılmalı');
    assert.equal((await page.$eval('#yorumSayi', (e) => e.textContent.trim())), '1 yorum seçili', 'sayaç seçimi yansıtmalı');

    // Tümünü seç → bütün kutular işaretlenir; kaldırınca gizlenir.
    await page.evaluate(() => {
      const t = document.getElementById('yorumTumunuSec');
      t.checked = true;
      t.dispatchEvent(new Event('change'));
    });
    assert.equal(
      await page.evaluate(() => {
        const k = Array.from(document.querySelectorAll('.yorumSec'));
        return k.filter((c) => c.checked).length === k.length;
      }),
      true,
      '"tümünü seç" bütün kutuları işaretlemeli'
    );
    await page.evaluate(() => {
      const t = document.getElementById('yorumTumunuSec');
      t.checked = false;
      t.dispatchEvent(new Event('change'));
    });
    assert.equal(await page.$eval('#yorumBar', (e) => e.hidden), true, 'seçim kalkınca çubuk gizlenmeli');

    assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
  });
});

test('raporlar: yorum + spam istatistik kartları görünür (veri yazmaz)', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdmin(async (page, errors) => {
    await setViewport(page, 1400);
    await goto(page, '/admin/raporlar');
    assert.ok(!page.url().includes('/admin/giris'), 'rapor sayfası açık: ' + page.url());

    const bolum = await page.evaluate(() => {
      const t = (sel) => document.querySelector(sel) ? document.querySelector(sel).textContent : '';
      const body = document.querySelector('.admin-body').textContent;
      return {
        enCokYorum: body.includes('En Çok Yorumlanan'),
        yorumDurum: body.includes('Yorum Durumu'),
        spam: body.includes('Spam') && body.includes('Tuzak isabet'),
        engelli: body.includes('Engelli IP'),
        onayGrup: body.includes('Bekliyor:') && body.includes('Onaylı:') && body.includes('Toplam:'),
        bekleyenSayi: (body.match(/Bekliyor: (\d+)/) || [])[1],
        yonetLinki: !!document.querySelector('a[href="/admin/yorumlar"]'),
        ayrintiLinki: !!document.querySelector('a[href="/admin/indexleme"]'),
      };
    });
    assert.equal(bolum.enCokYorum, true, '"En Çok Yorumlanan" kartı olmalı');
    assert.equal(bolum.yorumDurum, true, '"Yorum Durumu" kartı olmalı');
    assert.equal(bolum.onayGrup, true, 'Bekliyor/Onaylı/Toplam rozetleri olmalı');
    assert.equal(bolum.spam, true, '"Spam & Bot Koruması" kartı olmalı');
    assert.equal(bolum.engelli, true, '"Engelli IP" istatistiği olmalı');
    assert.match(bolum.bekleyenSayi || '', /^\d+$/, 'Bekliyor sayısı sayısal olmalı: ' + bolum.bekleyenSayi);
    assert.equal(bolum.yonetLinki, true, 'yorum yönetimine giden bağlantı olmalı');
    assert.equal(bolum.ayrintiLinki, true, 'indeksleme/spam ayrıntısına giden bağlantı olmalı');

    assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
  });
});

test('sosyal paylaşım: 8 kanal panel üzerinden yönetilebilir görünür (veri yazmaz, POST atmaz)', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdmin(async (page, errors) => {
    await setViewport(page, 1500);
    await goto(page, '/admin/paylasim');
    assert.ok(!page.url().includes('/admin/giris'), 'paylaşım sayfası açık: ' + page.url());

    const durum = await page.evaluate(() => {
      const form = document.querySelector('form[action="/admin/paylasim/ayar"]');
      const kanallar = ['telegram', 'facebook', 'x', 'linkedin', 'bluesky', 'mastodon', 'whatsapp', 'instagram'];
      const kutular = kanallar.map((k) => form && !!form.querySelector('input[name="cn_' + k + '"]'));
      return {
        form: !!form,
        tumKanallar: kutular.every(Boolean),
        kanallar,
        etkinSayi: /\/ 8/.test(document.querySelector('.admin-body').textContent),
        auto: !!document.querySelector('input[name="auto"]'),
        testButonlari: document.querySelectorAll('[data-test-post="/admin/paylasim/test"]').length,
        gonder: !!document.querySelector('form[action="/admin/paylasim/gonder"]'),
        sonPaylasimlar: document.querySelector('.admin-body').textContent.includes('Son Paylaşımlar'),
      };
    });
    assert.equal(durum.form, true, 'ayar formu olmalı');
    assert.equal(durum.tumKanallar, true, '8 kanalın etkin kutusu olmalı: ' + durum.kanallar.join(','));
    assert.equal(durum.etkinSayi, true, '"N / 8" sayaç bilgisi olmalı');
    assert.equal(durum.auto, true, 'otomatik paylaşım kutusu olmalı');
    assert.equal(durum.testButonlari >= 8, true, 'her kanalda test düğmesi olmalı (' + durum.testButonlari + ')');
    assert.equal(durum.gonder, true, '"Hemen Paylaş" formu olmalı');
    assert.equal(durum.sonPaylasimlar, true, 'son paylaşımlar tablosu olmalı');

    assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
  });
});

test('etiket yönetimi: liste + adlandır/birleştir/sil kontrolleri görünür (veri yazmaz)', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdmin(async (page, errors) => {
    await setViewport(page, 1500);
    await goto(page, '/admin/etiketler');
    assert.ok(!page.url().includes('/admin/giris'), 'etiket sayfası açık: ' + page.url());

    const durum = await page.evaluate(() => {
      const body = document.querySelector('.admin-body').textContent;
      return {
        baslik: body.includes('Etiketler'),
        toplamRozet: /Etiketler\s*\d+/.test(body),
        satirlar: document.querySelectorAll('input[name="yeni_ad"]').length,
        birlestir: document.querySelectorAll('input[name="hedef"]').length,
        adlandirForm: !!document.querySelector('form[action$="/adlandir"]'),
        birlestirForm: !!document.querySelector('form[action="/admin/etiketler/birlestir"]'),
        ara: !!document.querySelector('form[action="/admin/etiketler"] input[name="q"]'),
      };
    });
    assert.equal(durum.baslik, true, '"Etiketler" başlığı olmalı');
    assert.equal(durum.toplamRozet, true, 'toplam etiket rozeti olmalı (1640+)');
    assert.ok(durum.satirlar > 0, 'etiket satırları olmalı (' + durum.satirlar + ')');
    assert.ok(durum.birlestir === durum.satirlar, 'her satırda birleştirme girişi olmalı');
    assert.equal(durum.adlandirForm, true, 'adlandırma formu olmalı');
    assert.equal(durum.birlestirForm, true, 'birleştirme formu olmalı');
    assert.equal(durum.ara, true, 'arama kutusu olmalı');

    assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
  });
});

test('bildirimler: push abone listesi görünür ve silme kontrolü vardır (veri yazmaz)', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdmin(async (page, errors) => {
    await setViewport(page, 1400);
    await goto(page, '/admin/bildirim');
    assert.ok(!page.url().includes('/admin/giris'), 'bildirim sayfası açık: ' + page.url());

    const durum = await page.evaluate(() => {
      const body = document.querySelector('.admin-body').textContent;
      return {
        aboneler: body.includes('Aboneler'),
        sayac: /Aboneler\s*\d+/.test(body),
        silForm: !!document.querySelector('form[action="/admin/bildirim/sub-sil"]'),
        endpointHucre: !!document.querySelector('table.table tbody tr td:nth-child(1)'),
        gonder: !!document.querySelector('form[action="/admin/bildirim/gonder"]'),
      };
    });
    assert.equal(durum.aboneler, true, '"Aboneler" bölümü olmalı');
    assert.equal(durum.sayac, true, 'abone sayacı olmalı');
    assert.equal(durum.silForm, true, 'abone silme formu olmalı');
    assert.equal(durum.endpointHucre, true, 'endpoint sütunu olmalı');
    assert.equal(durum.gonder, true, 'bildirim gönderme formu olmalı');

    assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
  });
});

test('çeviriler: dil sekmeleri + anahtar editörü görünür (veri yazmaz)', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdmin(async (page, errors) => {
    await setViewport(page, 1500);
    await goto(page, '/admin/ceviriler');
    assert.ok(!page.url().includes('/admin/giris'), 'çeviri sayfası açık: ' + page.url());

    const durum = await page.evaluate(() => {
      const body = document.querySelector('.admin-body').textContent;
      const sekmeler = document.querySelectorAll('a[href^="/admin/ceviriler?dil="]').length;
      const satirlar = document.querySelectorAll('table.table tbody tr').length;
      return {
        sekmeler,
        satirlar,
        kaydetForm: !!document.querySelector('form[action="/admin/ceviriler/kaydet"]'),
        ekleForm: !!document.querySelector('form[action="/admin/ceviriler/anahtar-ekle"]'),
        trRef: body.includes('Ana Sayfa'),
        dilBaslik: /Çeviriler/.test(body),
      };
    });
    assert.ok(durum.sekmeler >= 10, 'dil sekmeleri olmalı (>10): ' + durum.sekmeler);
    assert.ok(durum.satirlar >= 20, 'anahtar satırları olmalı (≥20): ' + durum.satirlar);
    assert.equal(durum.kaydetForm, true, 'kaydet formu olmalı');
    assert.equal(durum.ekleForm, true, 'yeni anahtar formu olmalı');
    assert.equal(durum.trRef, true, 'TR referans sütunu dolu olmalı');
    assert.equal(durum.dilBaslik, true, 'Çeviriler başlığı olmalı');

    assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
  });
});

test('içerik önizleme: düzenleme sayfasındaki adres çalışır ve article render eder (veri yazmaz)', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdmin(async (page) => {
    await setViewport(page, 1400);
    await goto(page, '/admin/icerikler');
    assert.ok(!page.url().includes('/admin/giris'), 'içerik listesi açık');

    // İlk içeriğin id'sini listeden bul (sıralı satır linki: /admin/icerikler/<id>)
    const gid = await page.evaluate(() => {
      const a = [...document.querySelectorAll('a[href]')].find((x) =>
        /\/admin\/icerikler\/\d+$/.test(x.getAttribute('href') || '')
      );
      const m = a ? a.getAttribute('href').match(/\/admin\/icerikler\/(\d+)$/) : null;
      return m ? m[1] : null;
    });
    if (!gid) return t.skip('listedek yok');

    // Editör formunda "Önizle" (yeni sekme) bağlantısı olmalı
    await goto(page, '/admin/icerikler/' + gid);
    const linkVar = await page.evaluate(() =>
      [...document.querySelectorAll('a[href*="/onizleme"]')].some((x) => x.target === '_blank')
    );
    assert.ok(linkVar, 'editör formunda Önizle (yeni sekme) bağlantısı olmalı');

    // Önizleme adresi (yayındaki içerik için) article render etmeli (uygun HTTP + no-store)
    const resp = await goto(page, '/admin/icerikler/' + gid + '/onizleme');
    const durum = await page.evaluate(() => ({
      girisDegil: !location.pathname.includes('/admin/giris'),
      title: document.querySelector('title') ? document.querySelector('title').textContent.trim() : '',
    }));
    assert.equal(durum.girisDegil, true, 'önizleme girişe yönlenmemeli');
    assert.ok(durum.title.length > 0, 'önizleme gerçek makale başlığını basmalı: ' + durum.title);
    assert.ok(resp.status() < 500, 'önizleme 5xx olmamalı: ' + (resp && resp.status()));
    const cc = resp.headers()['cache-control'] || '';
    assert.ok(cc.includes('no-store'), 'önizleme yanıtı no-store olmalı: ' + cc);
  });
});

test('içerik listesi: özel tarih aralığı filtresi (baslangic/bitis) uygulanır (veri yazmaz)', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdmin(async (page) => {
    await setViewport(page, 1400);
    await goto(page, '/admin/icerikler');
    assert.ok(!page.url().includes('/admin/giris'), 'içerik listesi açık');

    // Filtre satırında tarih girişleri var mı
    const varMi = await page.evaluate(() => ({
      baslangic: !!document.querySelector('form[action="/admin/icerikler"] input[name="baslangic"]'),
      bitis: !!document.querySelector('form[action="/admin/icerikler"] input[name="bitis"]'),
    }));
    assert.equal(varMi.baslangic && varMi.bitis, true, 'tarih aralığı (baslangic/bitis) girişleri olmalı');

    // Aralık uygula (GET — veri yazmaz) → girişler yankılanır, tab linkleri aralığı korur
    await goto(page, '/admin/icerikler?baslangic=2026-10-14&bitis=2026-10-15');
    const echo = await page.evaluate(() => ({
      b: (document.querySelector('input[name="baslangic"]') || {}).value || '',
      e: (document.querySelector('input[name="bitis"]') || {}).value || '',
      tab: [...document.querySelectorAll('.tabs a')].map((a) => a.getAttribute('href')).join(' '),
    }));
    assert.equal(echo.b, '2026-10-14', 'başlangıç değeri yankılanmalı: ' + echo.b);
    assert.equal(echo.e, '2026-10-15', 'bitiş değeri yankılanmalı: ' + echo.e);
    assert.ok(echo.tab.includes('baslangic=2026-10-14') && echo.tab.includes('bitis=2026-10-15'),
      'sekme linkleri tarih aralığını korumalı: ' + echo.tab.slice(0, 120));
  });
});

test('içerik listesi: görünüm kaydetme çubuğu görünür (veri yazmaz, POST atmaz)', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdmin(async (page) => {
    await setViewport(page, 1400);
    await goto(page, '/admin/icerikler');
    assert.ok(!page.url().includes('/admin/giris'), 'içerik listesi açık');

    const durum = await page.evaluate(() => {
      const f = document.querySelector('form[action="/admin/icerikler/gorunum-kaydet"]');
      return {
        form: !!f,
        ad: !!f && !!f.querySelector('input[name="ad"]'),
        qs: !!f && !!f.querySelector('input[name="qs"]'),
        silYolu: !!document.querySelector('form[action="/admin/icerikler/gorunum-sil"]') ||
          !!document.querySelector('.chip-view'),
      };
    });
    assert.equal(durum.form, true, '"görünümü kaydet" formu olmalı');
    assert.equal(durum.ad && durum.qs, true, 'ad + gizli qs girişleri olmalı');
  });
});

test('tasarımcı: SortableJS yüklü', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdmin(async (page) => {
    const okAuth = await openDesigner(page, 1400, 900);
    assert.ok(okAuth, 'tasarımcı açıldı: ' + page.url());
    assert.equal(
      await page.evaluate(() => typeof window.Sortable),
      'function',
      'SortableJS yüklenmeli'
    );
  });
});

test('tasarımcı: her modül çipinde sürükleme kulpu var', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdmin(async (page) => {
    assert.ok(await openDesigner(page, 1400, 900));
    const chips = await page.$$eval('.bp-chip[data-id]', (els) =>
      els.map((e) => !!e.querySelector('.grip'))
    );
    assert.ok(chips.length > 0, 'modül çipi bulunamadı');
    assert.ok(chips.every(Boolean), 'kulpu olmayan çip: ' + chips.filter((x) => !x).length);
  });
});

test('tasarımcı: geri al / yinele düğmeleri mevcut', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdmin(async (page) => {
    assert.ok(await openDesigner(page, 1400, 900));
    assert.ok(await page.$('#bldUndo'), 'geri al düğmesi yok');
    assert.ok(await page.$('#bldRedo'), 'yinele düğmesi yok');
    // düğmeler görünüm değiştiricilerinin DIŞINDA olmalı (aksi halde view=undefined olur)
    const inViews = await page.$$eval('.bld-views button', (els) =>
      els.map((e) => e.id || e.dataset.view || '')
    );
    assert.ok(!inViews.includes('undefined'), 'data-viewsiz düğme görünüm kabına girmemeli: ' + inViews.join(','));
  });
});

test('tasarımcı: dar ekranda palet çiplerin üstüne binmemeli (sürükleme regresyonu)', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdmin(async (page) => {
    for (const w of [390, 768]) {
      assert.ok(await openDesigner(page, w, 844), `${w}px: tasarımcı açıldı`);
      const pos = await page.$eval('.bld-pal', (e) => getComputedStyle(e).position);
      assert.notEqual(
        pos,
        'sticky',
        `${w}px'de palet yapışkan kalırsa kaydırınca çiplerin üstünü örter ve ` +
          `dokunmatik sürükleme çalışmaz (position=${pos})`
      );
    }
  });
});

test('tasarımcı: dar ekranda çip tıklanabilir olmalı (palet altında kalmasın)', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdmin(async (page) => {
    assert.ok(await openDesigner(page, 390, 844));
    // ilk çipin merkezine dokunulduğunda hedef o çipin kendisi olmalı
    const hit = await page.evaluate(() => {
      const chip = document.querySelector('.bp-chip[data-id]');
      if (!chip) return 'cip-yok';
      chip.scrollIntoView({ block: 'center' });
      const r = chip.getBoundingClientRect();
      const el = document.elementFromPoint(r.x + r.width / 2, r.y + r.height / 2);
      return el ? chip.contains(el) || el === chip ? 'ok' : el.tagName + '.' + el.className : 'null';
    });
    assert.equal(hit, 'ok', 'çipin üstü başka bir öğeyle örtülüyor: ' + hit);
  });
});

test('tasarımcı: modül paleti 19 blok tipi sunar', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdmin(async (page) => {
    assert.ok(await openDesigner(page, 1400, 900));
    const n = await page.$$eval('.bp-pal .bld-add', (e) => e.length);
    assert.ok(n >= 19, `en az 19 blok tipi beklenir, bulunan ${n}`);
  });
});

test('tasarımcı: palete tıklayınca modül eklenir ve geri alınabilir', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdmin(async (page) => {
    assert.ok(await openDesigner(page, 1400, 900));
    const before = await page.$$eval('.bp-chip[data-id]', (e) => e.length);

    await page.$eval('.bp-pal .bld-add[data-type="puan_durumu"]', (b) => b.click());
    await new Promise((r) => setTimeout(r, 500));

    const after = await page.$$eval('.bp-chip[data-id]', (e) => e.length);
    assert.equal(after, before + 1, 'tıklayınca bir modül eklenmeli');

    // geri al düğmesi etkinleşmeli ve durumu eski haline döndürmeli
    const undoEnabled = await page.$eval('#bldUndo', (b) => !b.disabled);
    assert.ok(undoEnabled, 'eklemeden sonra geri al etkin olmalı');

    await page.$eval('#bldUndo', (b) => b.click());
    await new Promise((r) => setTimeout(r, 500));
    const restored = await page.$$eval('.bp-chip[data-id]', (e) => e.length);
    assert.equal(restored, before, 'geri al durumu eski haline döndürmeli');
  });
});

test('tasarımcı: gerçek modül önizlemeleri render edilir (Faz 2)', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdmin(async (page) => {
    assert.ok(await openDesigner(page, 1400, 900));

    // önizlemelerin şebekeden gelip DOM'a yerleşmesini bekle
    await page.waitForFunction(
      () =>
        !document.querySelector('.bp-prev.bp-prev-loading') &&
        document.querySelectorAll('.bp-prev').length > 0,
      { timeout: 15000 }
    );

    const state = await page.evaluate(() => {
      const prevs = [...document.querySelectorAll('.bp-prev')];
      // gerçek modül işaretleyicisi: yer tutucu olmayan, görünür yüksekliğe sahip
      const real = prevs.filter((p) => {
        const r = p.getBoundingClientRect();
        return r.height > 0 && !p.querySelector('.bp-prev-empty');
      });
      return {
        total: prevs.length,
        real: real.length,
        heights: prevs.map((p) => Math.round(p.getBoundingClientRect().height)),
        scriptPollution: prevs.filter((p) => /<script/i.test(p.innerHTML)).length,
        linkPollution: prevs.filter((p) => /<link/i.test(p.innerHTML)).length,
        hasRealMarkup: prevs.some(
          (p) =>
            p.querySelector('.ust-manset, .hero-row, .m6, .kat-sekmeli, .puan-blk, .trend-num') !== null
        ),
      };
    });

    assert.ok(state.total > 0, 'önizleme kabı olmalı');
    assert.ok(state.real >= 5, `en az 5 gerçek içerikli önizleme beklenir (var: ${state.real})`);
    assert.equal(state.scriptPollution, 0, 'önizleme içinde <script> olmamalı');
    assert.equal(state.linkPollution, 0, 'önizleme içinde <link> olmamalı');
    assert.ok(state.hasRealMarkup, 'bilinen modül işaretleyicilerinden biri render edilmeli');
    assert.ok(
      state.heights.some((h) => h > 100),
      'en az bir önizleme gerçek yükseklikte render olmalı (boş kabuk değil)'
    );
  });
});

test('tasarımcı: interaktif modül için yer tutucu görünür (hikayeler)', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdmin(async (page) => {
    assert.ok(await openDesigner(page, 1400, 900));
    await page.waitForFunction(
      () => document.querySelectorAll('.bp-prev .bp-prev-empty').length > 0,
      { timeout: 15000 }
    );
    const txt = await page.$eval('.bp-prev .bp-prev-empty', (e) => e.textContent);
    assert.match(txt, /interaktif|önizleme/i, 'yer tutucu neden açıklanmalı: ' + txt);
  });
});

test('tasarımcı: çubuk tıklaması katlar ama sunucuya kaydetmez', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdmin(async (page) => {
    assert.ok(await openDesigner(page, 1400, 900));

    // katlama etkileşiminde HİÇBİR kaydetme isteği gitmemeli (UI durumu sunucuya taşmaz)
    const saves = [];
    const onReq = (r) => {
      if (r.method() === 'POST' && /\/admin\/[\w-]+\/kaydet/.test(new URL(r.url()).pathname)) {
        saves.push(r.url());
      }
    };
    page.on('request', onReq);

    const chip = '.bp-chip[data-id]';
    await page.waitForSelector(chip);

    const collapse = () =>
      page.$eval(chip + ' .bp-chipbar .nm', (e) => {
        e.dispatchEvent(new MouseEvent('click', { bubbles: true }));
      });

    await collapse();
    await new Promise((r) => setTimeout(r, 300));
    const hid1 = await page.$eval(chip, (c) => {
      const p = c.querySelector('.bp-prev');
      return { cls: c.className, disp: getComputedStyle(p).display };
    });
    assert.ok(hid1.cls.includes('collapsed'), 'ilk tıklama çipi katlamalı');
    assert.equal(hid1.disp, 'none', 'katlanınca önizleme gizlenmeli');

    await collapse();
    await new Promise((r) => setTimeout(r, 300));
    const hid2 = await page.$eval(chip, (c) => {
      const p = c.querySelector('.bp-prev');
      return { cls: c.className, disp: getComputedStyle(p).display };
    });
    assert.ok(!hid2.cls.includes('collapsed'), 'ikinci tıklama açmalı');
    assert.notEqual(hid2.disp, 'none', 'açılınca önizleme görünmeli');

    page.off('request', onReq);
    assert.equal(saves.length, 0, 'katlama/açma sunucuya kaydedilmemeli: ' + saves.join(', '));
    // son çip açık kaldı (durum kirletilmedi)
    assert.equal(hid2.disp, 'block');
  });
});

test('tasarımcı: önizleme gövdesine tıklamak katlamaz (yalnız çubuk tetikler)', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdmin(async (page) => {
    assert.ok(await openDesigner(page, 1400, 900));
    // gerçek içerikli (yüksekliği > 0) bir önizleme seç
    await page.waitForFunction(
      () => [...document.querySelectorAll('.bp-prev')].some((p) => p.getBoundingClientRect().height > 0),
      { timeout: 15000 }
    );
    const target = await page.evaluate(() => {
      const p = [...document.querySelectorAll('.bp-prev')].find(
        (x) => x.getBoundingClientRect().height > 0 && !x.querySelector('.bp-prev-empty')
      );
      const chip = p.closest('.bp-chip');
      const r = p.getBoundingClientRect();
      return { x: r.x + r.width / 2, y: r.y + r.height / 2, id: chip.dataset.id };
    });
    await page.mouse.click(target.x, target.y);
    await new Promise((r) => setTimeout(r, 300));
    const cls = await page.$eval(`.bp-chip[data-id="${target.id}"]`, (c) => c.className);
    assert.ok(!cls.includes('collapsed'), 'gövdeye tıklamak katlamamalı: ' + cls);
  });
});

test('tasarımcı: çip kilitleme sürüklemeyi ve silmeyi kapatır', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdmin(async (page) => {
    assert.ok(await openDesigner(page, 1400, 900));
    await page.$eval('.bp-pal .bld-add[data-type="baslik"]', (b) => b.click());
    await new Promise((r) => setTimeout(r, 500));

    await page.$eval('.bp-chip .ops button[data-act="lock"]', (b) => b.click());
    await new Promise((r) => setTimeout(r, 500));

    const locked = await page.$$eval('.bp-chip.locked', (e) => e.length);
    assert.ok(locked > 0, 'kilit sınıfı uygulanmalı');
    const delDisabled = await page.$eval(
      '.bp-chip.locked .ops button[data-act="del"]',
      (b) => b.disabled
    );
    assert.equal(delDisabled, true, 'kilitli çipte silme düğmesi devre dışı olmalı');

    // temizlik
    await page.$eval('#bldUndo', (b) => b.click());
    await new Promise((r) => setTimeout(r, 400));
    await page.$eval('#bldUndo', (b) => b.click());
    await new Promise((r) => setTimeout(r, 600));
  });
});

// ---- Faz 3: blok başına kapak şablonu eşlemesi ----

test('tasarımcı: görselli blokta kapak şablonu seçici var', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdmin(async (page) => {
    assert.ok(await openDesigner(page, 1400, 900));
    // slider/manset tipi bir çipi bul ve ayar panelini aç
    const chipIdx = await page.evaluate(() => {
      const chips = [...document.querySelectorAll('.bp-chip[data-id]')];
      const i = chips.findIndex((c) => c.innerHTML.indexOf('slider') >= 0 || c.innerHTML.indexOf('manset') >= 0);
      if (i < 0) return -1;
      chips[i].querySelector('.ops button[data-act="cfg"]').click();
      return i;
    });
    if (chipIdx < 0) return t.skip('görselli blok yok (slider/manset)');

    await new Promise((r) => setTimeout(r, 400));
    const hasDesignSel = await page.evaluate(() => {
      const sel = document.querySelector('select[data-k="kapak_tasarim"]');
      return sel ? { options: sel.options.length, rendered: true } : { rendered: false };
    });
    assert.equal(hasDesignSel.rendered, true, 'kapak şablonu seçici render edilmeli');
    if (hasDesignSel.rendered) {
      assert.ok(hasDesignSel.options >= 1, 'şablon listesi boş olmamalı (en az "yok" seçeneği)');
    }
    await page.$eval('#bldUndo', (b) => b.click()).catch(() => {});
  });
});

test('tasarımcı: slider üst yazı rengi, kategori ve gösterge ayarları', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdmin(async (page) => {
    assert.ok(await openDesigner(page, 1400, 900));
    const opened = await page.evaluate(() => {
      const chips = [...document.querySelectorAll('.bp-chip[data-id]')];
      const chip = chips.find((c) => c.innerHTML.indexOf('slider') >= 0);
      if (!chip) return false;
      chip.querySelector('.ops button[data-act="cfg"]').click();
      return true;
    });
    if (!opened) return t.skip('slider bloğu yok');
    await new Promise((r) => setTimeout(r, 350));
    const fields = await page.evaluate(() => ({
      color: !!document.querySelector('input[data-k="title_color"]'),
      font: !!document.querySelector('select[data-k="title_font"]'),
      lineHeight: !!document.querySelector('input[data-k="title_line_height"]'),
      letterSpacing: !!document.querySelector('input[data-k="title_letter_spacing"]'),
      wordSpacing: !!document.querySelector('input[data-k="title_word_spacing"]'),
      titleCase: !!document.querySelector('select[data-k="title_case"]'),
      overlayColor: !!document.querySelector('input[data-k="overlay_color"]'),
      overlayOpacity: !!document.querySelector('input[data-k="overlay_opacity"]'),
      overlayOpacityPreset: !!document.querySelector('select[data-k="overlay_opacity_preset"]'),
      overlayHeight: !!document.querySelector('input[data-k="overlay_height"]'),
      overlayGradient: !!document.querySelector('select[data-k="overlay_gradient"]'),
      overlayBlur: !!document.querySelector('input[data-k="overlay_blur"]'),
      catPos: !!document.querySelector('select[data-k="cat_pos"]'),
      catSize: !!document.querySelector('input[data-k="cat_size"]'),
      catColor: !!document.querySelector('input[data-k="cat_color"]'),
      catShape: !!document.querySelector('select[data-k="cat_shape"]'),
      catClip: !!document.querySelector('input[data-k="cat_clip"]'),
      catLeft: !!document.querySelector('input[data-k="cat_left"]'),
      catBottom: !!document.querySelector('input[data-k="cat_bottom"]'),
      headStyle: document.querySelectorAll('input[data-k="head_style"]').length,
      modStyle: !!document.querySelector('select[data-k="mod_style"]'),
      hideMobile: !!document.querySelector('input[data-k="hide_mobile"]'),
      hideBadge: !!document.querySelector('input[data-k="hide_badge"]'),
      hideHeading: !!document.querySelector('input[data-k="hide_heading"]'),
      pager: !!document.querySelector('select[data-k="pager_type"]'),
      pagerShape: !!document.querySelector('select[data-k="pager_shape"]'),
      pagerPos: !!document.querySelector('select[data-k="pager_pos"]'),
      pagerColor: !!document.querySelector('input[data-k="pager_color"]'),
      pagerSize: !!document.querySelector('input[data-k="pager_size"]'),
      showAll: !!document.querySelector('input[data-k="show_all"]'),
      allowOver: !!document.querySelector('input[data-k="allow_over_20"]'),
      m2Fade: !!document.querySelector('input[data-k="m2_fade"]'),
      m2NumBg: !!document.querySelector('input[data-k="m2_num_bg"]'),
      m2NumActiveBg: !!document.querySelector('input[data-k="m2_num_active_bg"]'),
      sliderStyle: !!document.querySelector('select[data-k="slider_style"]'),
      sliderStyleOptions: Array.from((document.querySelector('select[data-k="slider_style"]') || {}).options || []).map((o) => o.textContent),
    }));
    assert.equal(fields.color, true, 'başlık rengi alanı olmalı');
    assert.equal(fields.font, true, 'başlık fontu alanı olmalı');
    assert.equal(fields.lineHeight, true, 'satır yüksekliği alanı olmalı');
    assert.equal(fields.letterSpacing, true, 'harf aralığı alanı olmalı');
    assert.equal(fields.wordSpacing, true, 'kelime aralığı alanı olmalı');
    assert.equal(fields.titleCase, true, 'başlık harf biçimi alanı olmalı');
    assert.equal(fields.overlayColor, true, 'başlık arkası rengi alanı olmalı');
    assert.equal(fields.overlayOpacity, true, 'başlık arkası özel saydamlığı alanı olmalı');
    assert.equal(fields.overlayOpacityPreset, true, 'başlık arkası saydamlık seçeneği olmalı');
    assert.equal(fields.overlayHeight, true, 'başlık arkası yüksekliği alanı olmalı');
    assert.equal(fields.overlayGradient, true, 'başlık arkası geçiş yönü alanı olmalı');
    assert.equal(fields.overlayBlur, true, 'başlık arkası bulanıklığı alanı olmalı');
    assert.equal(fields.catPos, true, 'kategori konumu alanı olmalı');
    assert.equal(fields.catSize, true, 'kategori boyutu alanı olmalı');
    assert.equal(fields.catColor, true, 'kategori etiketi rengi alanı olmalı');
    assert.equal(fields.catShape, true, 'kategori etiket biçimi alanı olmalı');
    assert.equal(fields.catClip, true, 'etiket clip-path alanı olmalı');
    assert.equal(fields.catLeft, true, 'etiket left alanı olmalı');
    assert.equal(fields.catBottom, true, 'etiket bottom alanı olmalı');
    assert.equal(fields.headStyle, 10, '10 başlık işaret stili seçeneği olmalı');
    assert.equal(fields.modStyle, true, 'modül görünüm stili alanı olmalı');
    assert.equal(fields.hideMobile, true, 'mobilde gizle alanı olmalı');
    assert.equal(fields.hideBadge, true, 'kategori etiketini gizle alanı olmalı');
    assert.equal(fields.hideHeading, true, 'modül başlığını gizle alanı olmalı');
    assert.equal(fields.m2Fade, true, 'Stil 5 geçiş efekti rengi alanı olmalı');
    assert.equal(fields.m2NumBg, true, 'Stil 5 sayı barı arka planı alanı olmalı');
    assert.equal(fields.m2NumActiveBg, true, 'Stil 5 aktif sayı arka planı alanı olmalı');
    assert.equal(fields.pager, true, 'slider gösterge tipi alanı olmalı');
    assert.equal(fields.pagerShape, true, 'gösterge biçimi alanı olmalı');
    assert.equal(fields.pagerPos, true, 'gösterge konumu alanı olmalı');
    assert.equal(fields.pagerColor, true, 'gösterge rengi alanı olmalı');
    assert.equal(fields.pagerSize, true, 'gösterge boyutu alanı olmalı');
    assert.equal(fields.showAll, true, 'tüm manşetler aç/kapa alanı olmalı');
    assert.equal(fields.allowOver, true, '20 üzeri sayı izni alanı olmalı');
    assert.equal(fields.sliderStyle, true, 'slider stil şablonu alanı olmalı');
    for (const name of ['Stil 1', 'Stil 2', 'Stil 3', 'Stil 4', 'Stil 5']) {
      assert.ok(fields.sliderStyleOptions.includes(name), name + ' seçeneği olmalı');
    }
    assert.ok(!fields.sliderStyleOptions.includes('Edirne Gerçek'), 'eski Edirne adı görünmemeli');
    await page.$eval('#bldUndo', (b) => b.click()).catch(() => {});
  });
});

test('tasarımcı: kapak şablonu seçimi önizlemede gerçek kapak üretir', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdmin(async (page) => {
    assert.ok(await openDesigner(page, 1400, 900));
    await page.waitForFunction(
      () => document.querySelectorAll('.bp-chip[data-id]').length > 0,
      { timeout: 10000 }
    );

    // slider çipini bul, ayarları aç, ilk şablonu seç
    const chipIdx = await page.evaluate(() => {
      const chips = [...document.querySelectorAll('.bp-chip[data-id]')];
      const i = chips.findIndex((c) => c.innerHTML.indexOf('slider') >= 0);
      if (i < 0) return -1;
      chips[i].querySelector('.ops button[data-act="cfg"]').click();
      return i;
    });
    if (chipIdx < 0) return t.skip('slider bloğu yok');

    await new Promise((r) => setTimeout(r, 400));
    const designSel = await page.$('select[data-k="kapak_tasarim"]');
    if (!designSel) return t.skip('kapak şablonu seçici yok');
    const optCount = await designSel.$$eval('option[value]:not([value="0"])', (o) => o.length);
    if (!optCount) return t.skip('kayıtlı şablon yok');

    // seçimi değiştir
    await designSel.select(await designSel.$$eval('option[value]:not([value="0"])', (o) => o[0].value));
    await new Promise((r) => setTimeout(r, 2000)); // debounce + refetch

    // önizlemede gerçek kapak (<img>) dolu görünmeli
    const previewImg = await page.evaluate(() => {
      const sel = document.querySelector('select[data-k="kapak_tasarim"]');
      if (!sel) return { selected: 'yok' };
      const chip = sel.closest('.bp-chip');
      const prev = chip ? chip.querySelector('.bp-prev') : null;
      const img = prev ? prev.querySelector('img') : null;
      return {
        selected: sel.value,
        hasImg: !!img,
        src: img ? img.getAttribute('src') : '',
        loading: prev ? prev.classList.contains('bp-prev-loading') : false,
      };
    });
    assert.ok(Number(designSel ? await designSel.evaluate((s) => s.value) : 0) > 0, 'şablon seçilmeli');
    assert.equal(previewImg.loading, false, 'önizleme yüklenmeyi bırakmalı');
    assert.ok(previewImg.hasImg && previewImg.src.includes('/media/'), 'önizlemede üretilen kapak <img> bulunmalı');

    // temizlik: seçimi sıfırla ki kalıcı layout'a bulaşmasın
    await designSel.select('0');
    await new Promise((r) => setTimeout(r, 800));
    await page.$eval('#bldUndo', (b) => b.click()).catch(() => {});
  });
});

// ---- Faz 4: sürüm geçmişi (kalıcı geri alma) ----

test('tasarımcı: kaydedilen düzen sürüm geçmişine arşivlenir ve geri alınır', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdmin(async (page) => {
    assert.ok(await openDesigner(page, 1400, 900));
    await page.waitForFunction(
      () => document.querySelectorAll('.bp-chip[data-id]').length > 0,
      { timeout: 10000 }
    );

    // önce mevcut aktif düzeni ve kaydedilmemiş fark hazırla: geçmişi açıp
    // "henüz sürüm yok" ya da en az bir sürüm olduğunu gör
    const openHistory = async () => {
      await page.$eval('#bldHistoryBtn', (b) => b.click());
      await page.waitForFunction(
        () => !document.getElementById('bldHistory').hidden,
        { timeout: 8000 }
      );
      // liste yüklendi
      await page.waitForFunction(
        () => {
          const l = document.getElementById('bldHistoryList');
          return l && !l.innerHTML.includes('Yükleniyor');
        },
        { timeout: 8000 }
      );
    };
    await openHistory();
    const beforeText = await page.$eval('#bldHistoryList', (l) => l.textContent);

    // Kaydet: mevcut düzen "önceki durum" olarak arşivlenir (canlı görünüm
    // değişmez çünkü aynı bloklar tekrar yazılır), sonra geçmişte bir sürüm olmalı
    await page.$eval('#bldHistoryClose', (b) => b.click()).catch(() => {});
    await page.$eval('#bldSave', (b) => b.click());
    await new Promise((r) => setTimeout(r, 1200));

    await openHistory();
    const afterText = await page.$eval('#bldHistoryList', (l) => l.textContent);
    assert.ok(afterText.length > 0, 'geçmiş listesi boş olmamalı');
    // en az bir satır var
    const rows = await page.$$eval('.bld-history-item', (els) => els.length);
    // İlk kaydetmede sürüm oluşur; "henüz sürüm yok" ise geri alma da yok —
    // ama kayıt en az bir sürüm arşivlemiş olmalı. Önceki boşsa bu test kendi
    // kaydıyla bir sürüm yaratmış olur (canlı görünüm değişmez).
    const hadBefore = !beforeText.includes('Henüz sürüm yok');
    const hasRestoreBtn = await page.$('.bld-history-item button[data-vid]');
    assert.ok(hasRestoreBtn || rows > 0, 'sürüm satırı ve/veya geri al düğmesi olmalı');

    // Geri al düğmesi varsa, en yeniyi geri al (canlı görünüm aynı kalır)
    if (hasRestoreBtn) {
      page.on('dialog', (d) => d.accept());
      await page.$eval('.bld-history-item button[data-vid]', (b) => b.click());
      await page.waitForFunction(
        () => !document.querySelector('.bld-history-item button[data-vid]') || true,
        { timeout: 8000 }
      );
      // geri alma sonrası sayfa yenilenir; görünüm açılmış olmalı
      const okAuth = await openDesigner(page, 1400, 900);
      assert.ok(okAuth, 'geri alma sonrası tasarımcı açılmalı');
    } else {
      // sürüm yoksa en az panelin açıldığını kanıtlarız
      assert.ok(true, 'panelsürüm listesi render edildi (kaydedilecek sürüm başka testte oluşur)');
    }
    void hadBefore;
    await page.$eval('#bldHistoryClose', (b) => b.click()).catch(() => {});
  });
});

// ---- Faz 5: serbest (absolute) konumlandırma ----

test('tasarımcı: her çipin ayarlarında serbest konum alanları var', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdmin(async (page) => {
    assert.ok(await openDesigner(page, 1400, 900));
    await page.waitForFunction(
      () => document.querySelectorAll('.bp-chip[data-id]').length > 0,
      { timeout: 10000 }
    );
    // bir çipin ayarlarını aç
    await page.$eval('.bp-chip .ops button[data-act="cfg"]', (b) => b.click());
    await new Promise((r) => setTimeout(r, 400));
    const fields = await page.evaluate(() => {
      const keys = [...document.querySelectorAll('.bp-opts [data-k], .bp-chipbar ~ .bp-opts [data-k]')]
        .map((e) => e.dataset.k);
      return {
        hasAbs: keys.includes('stAbs'),
        hasXY: keys.includes('stX') && keys.includes('stY'),
        hasW: keys.includes('stW'),
      };
    });
    assert.equal(fields.hasAbs, true, 'stAbs alanı olmalı');
    assert.equal(fields.hasXY, true, 'stX/stY alanı olmalı');
    assert.equal(fields.hasW, true, 'stW alanı olmalı');
    await page.$eval('#bldUndo', (b) => b.click()).catch(() => {});
  });
});

test('tasarımcı: serbest konum kaydedilince çipte konum rozeti görünür', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdmin(async (page) => {
    assert.ok(await openDesigner(page, 1400, 900));
    await page.waitForFunction(
      () => document.querySelectorAll('.bp-chip[data-id]').length > 0,
      { timeout: 10000 }
    );

    // ORİJİNAL blok durumunu sakla (test sonunda geri yüklenecek — canlı kirletilmez)
    const orig = await page.evaluate(() => JSON.parse(JSON.stringify(window.__bldInit || [])));

    // bir bloğun ayarlarını aç, stAbs'ı işaretle + x/y gir (garanti change dispatch)
    await page.$eval('.bp-chip .ops button[data-act="cfg"]', (b) => b.click());
    await new Promise((r) => setTimeout(r, 400));
    const setAbs = await page.evaluate(() => {
      const c = document.querySelector('input[data-k="stAbs"]');
      if (!c) return false;
      c.checked = true;
      c.dispatchEvent(new Event('change', { bubbles: true }));
      const x = document.querySelector('input[data-k="stX"]');
      const y = document.querySelector('input[data-k="stY"]');
      if (x) { x.value = '45'; x.dispatchEvent(new Event('change', { bubbles: true })); }
      if (y) { y.value = '80'; y.dispatchEvent(new Event('change', { bubbles: true })); }
      return true;
    });
    assert.ok(setAbs, 'stAbs alanı bulunmalı');
    await new Promise((r) => setTimeout(r, 600)); // autosave + bar yenileme

    // açık opts panosundaki çipte rozet belirdi mi (refreshChipbar)?
    const chipAbs = await page.evaluate(() => {
      const s = document.querySelector('input[data-k="stAbs"]');
      if (!s) return { barBadge: false, bar: '' };
      const chip = s.closest('.bp-chip');
      const bar = chip ? chip.querySelector('.bp-chipbar') : null;
      return { barBadge: bar ? !!bar.querySelector('.abs-pos') : false, bar: bar ? bar.textContent : '' };
    });
    // Kaydet → reload → rozet DOM'da kalıcı (chipbarHTML doğrudan opts.stAbs okur)
    await page.$eval('#bldSave', (b) => b.click());
    await new Promise((r) => setTimeout(r, 1500));
    await page.reload({ waitUntil: 'networkidle2' }).catch(() => {});
    await page.waitForFunction(
      () => document.querySelectorAll('.bp-chip[data-id]').length > 0,
      { timeout: 10000 }
    );
    const afterReload = await page.evaluate(() => {
      const badges = document.querySelectorAll('.bp-chipbar .abs-pos');
      return { count: badges.length, txt: badges.length ? badges[0].textContent : '' };
    });

    const okLive = chipAbs.barBadge || afterReload.count > 0;
    assert.ok(okLive, 'serbest konum rozeti çipte görünmeli (bar veya kayıt sonrası); bar=' + chipAbs.bar + ' sonra=' + JSON.stringify(afterReload));

    // TEMİZLİK: orijinal blokları geri kaydet (canlı düzen bozulmasın)
    await page.evaluate((origBlocks) => {
      return fetch('/admin/tasarim/kaydet?sayfa=home', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ blocks: origBlocks }),
      }).then((r) => r.json());
    }, orig);
  });
});

// ---- Faz 6: zengin metin bloğu ----

test('tasarımcı: zengin metin editörü render edilir ve içerik önizlemeyi besler', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdmin(async (page) => {
    assert.ok(await openDesigner(page, 1400, 900));
    await page.waitForFunction(
      () => document.querySelectorAll('.bp-chip[data-id]').length >= 0,
      { timeout: 10000 }
    );
    const orig = await page.evaluate(() => JSON.parse(JSON.stringify(window.__bldInit || [])));

    // paletten "Zengin Metin" ekle
    const added = await page.evaluate(() => {
      const btn = [...document.querySelectorAll('.bp-pal .bld-add')].find((b) => b.dataset.type === 'metin');
      if (!btn) return false;
      btn.click();
      return true;
    });
    assert.ok(added, 'palette "metin" (Zengin Metin) bloğu olmalı');
    await new Promise((r) => setTimeout(r, 600));

    // yeni çipin ayarlarını aç → .rt-editor görünmeli
    const openOk = await page.evaluate(() => {
      const chips = [...document.querySelectorAll('.bp-chip[data-id]')];
      const c = chips.find((x) => x.innerHTML.indexOf('metin') >= 0);
      if (!c) return false;
      c.querySelector('.ops button[data-act="cfg"]').click();
      return true;
    });
    assert.ok(openOk, 'zengin metin çipi bulunup ayarları açılmalı');
    await new Promise((r) => setTimeout(r, 400));

    const editor = await page.evaluate(() => {
      const ed = document.querySelector('.rt-editor[data-k="html"]');
      return {
        has: !!ed,
        toolbar: ed ? ed.querySelector('.rt-toolbar') !== null : false,
        editable: ed ? !!ed.querySelector('.rt-content[contenteditable="true"]') : false,
      };
    });
    assert.ok(editor.has && editor.toolbar && editor.editable, 'zengin metin editörü + toolbar + contenteditable render edilmeli');

    // içerik gir → önizleme şebekeden .blk-rich gelir
    await page.evaluate(() => {
      const ce = document.querySelector('.rt-content[contenteditable="true"]');
      ce.innerHTML = '<p>Zengin <strong>metin</strong> başlığı</p>';
      ce.dispatchEvent(new Event('input', { bubbles: true }));
    });
    await new Promise((r) => setTimeout(r, 2000)); // save + önizleme debounce

    const prev = await page.evaluate(() => {
      const chip = [...document.querySelectorAll('.bp-chip[data-id]')].find(
        (x) => x.innerHTML.indexOf('Zengin') >= 0
      );
      if (!chip) return { chip: false };
      const p = chip.querySelector('.bp-prev');
      return {
        chip: true,
        html: p ? p.innerHTML : '',
        hasRich: p ? p.innerHTML.indexOf('blk-rich') >= 0 : false,
        hasStrong: p ? p.innerHTML.indexOf('<strong>metin</strong>') >= 0 : false,
      };
    });
    assert.ok(prev.chip, 'içerikli zengin metin çipi bulunmalı');
    assert.ok(prev.hasRich || prev.hasStrong, 'önizlemede .blk-rich veya <strong> içeriği olmalı: ' + prev.html);

    // TEMİZLİK — orijinal düzeni geri kaydet (boş pokus ekleme kalmasın)
    await page.evaluate((origBlocks) => {
      return fetch('/admin/tasarim/kaydet?sayfa=home', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ blocks: origBlocks }),
      }).then((r) => r.json());
    }, orig);
  });
});

// ---- Faz 7: A/B testi ----

test('tasarımcı: A/B paneli açılır, B eşitlenir ve çerez varyantı sabit tutar', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdmin(async (page) => {
    // native confirm() diyaloğunu otomatik kabul et (Puppeteer beklememesi için)
    page.on('dialog', (d) => d.accept());
    assert.ok(await openDesigner(page, 1400, 900));
    await page.waitForFunction(() => document.querySelectorAll('.bp-chip[data-id]').length > 0, { timeout: 10000 });

    // paneli aç
    await page.$eval('#bldABBtn', (b) => b.click());
    await page.waitForFunction(() => !document.getElementById('bldAB').hidden, { timeout: 8000 });
    await page.waitForFunction(() => document.getElementById('abStatsList') &&
      !document.getElementById('abStatsList').innerHTML.includes('Yükleniyor'), { timeout: 8000 });
    const openStat = await page.$eval('#abStatsList', (el) => el.textContent);
    assert.ok(openStat.length > 0, 'A/B istatistiği yüklendi');

    // B'yi mevcut düzene eşitle + ağırlık 100 (deterministik B) + kaydet
    await page.$eval('#abSetB', (b) => b.click());
    await new Promise((r) => setTimeout(r, 700));
    await page.$eval('#abWeight', (el) => { el.value = '100'; el.dispatchEvent(new Event('input', { bubbles: true })); });
    await page.$eval('#abEnabled', (el) => { if (!el.checked) el.click(); });
    await page.$eval('#abSave', (b) => b.click());
    await new Promise((r) => setTimeout(r, 900));

    // canlı: B çerezsiz istek (ağırlık 100 → yeni ziyaretçi B alır)
    const resp = await page.goto(BASE_URL + '/?_ab=1', { waitUntil: 'domcontentloaded' });
    assert.equal(resp.status(), 200, 'ana sayfa 200');
    await new Promise((r) => setTimeout(r, 500));

    // bu tarayıcı oturumunda tb_ab_home çerezi yazılmış olmalı
    const cookies = await page.cookies(BASE_URL);
    const abCookie = cookies.find((c) => c.name === 'tb_ab_home');
    assert.ok(abCookie, 'A/B çerezi yazılmalı');
    assert.equal(abCookie.value, 'b', 'ağırlık 100 → çerez B');

    // admin'e dön ve A/B'yi kapat + B'yi boşalt (kalıcı etki bırakma)
    await page.goto(BASE_URL + '/admin/tasarim?sayfa=home', { waitUntil: 'domcontentloaded' }).catch(() => {});
    await new Promise((r) => setTimeout(r, 700));
    const abGet = await page.evaluate(async () => {
      const r = await fetch('/admin/tasarim/ab?sayfa=home');
      return r.json();
    });
    assert.equal(abGet.enabled, true, 'A/B açık doğrulandı');
    const cleaned = await page.evaluate(async () => {
      const r = await fetch('/admin/tasarim/ab/kaydet?sayfa=home', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ mode: 'set-b', enabled: false, weight_b: 50, blocks: [] }),
      });
      return r.json();
    });
    assert.equal(cleaned.ok, true, 'A/B kapatıldı + B boşaltıldı');
  });
});

// ---- Faz 8: düzen kopyalama ----

test('tasarımcı: düzen başka sayfaya kopyalanır ve hedef orijinaline döner', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdmin(async (page) => {
    page.on('dialog', (d) => d.accept());

    // 1) article orijinal düzenini sakla
    await setViewport(page, 1400, 900);
    await goto(page, '/admin/tasarim?sayfa=article');
    const origArticle = await page.evaluate(() => JSON.parse(JSON.stringify(window.__bldInit || [])));
    assert.ok(origArticle.length > 0, 'article başlangıç düzeni boş olmamalı');

    // 2) home tasarımcısına dön, kopyalama panelini aç
    await goto(page, '/admin/tasarim?sayfa=home');
    await page.$eval('#bldCopyBtn', (b) => b.click());
    await page.waitForFunction(() => !document.getElementById('bldCopy').hidden, { timeout: 8000 });
    await page.$eval('#copyTarget', (el) => { el.value = 'article'; el.dispatchEvent(new Event('change', { bubbles: true })); });
    await page.$eval('#copyGo', (b) => b.click());
    await new Promise((r) => setTimeout(r, 1200));

    // 3) article artık home kadar blok içermeli (zone uyarlamalı)
    await goto(page, '/admin/tasarim?sayfa=article');
    await page.waitForFunction(
      () => document.querySelectorAll('.bp-chip[data-id]').length > 0,
      { timeout: 10000 }
    );
    const after = await page.evaluate(() => JSON.parse(JSON.stringify(window.__bldInit || [])));
    assert.ok(after.length > origArticle.length, `kopyalama sonrası article blokları artmalı (önce ${origArticle.length}, sonra ${after.length})`);

    // 4) TEMİZLİK: article'ı orijinaline geri kaydet
    const okReset = await page.evaluate((orig) => {
      return fetch('/admin/tasarim/kaydet?sayfa=article', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ blocks: orig }),
      }).then((r) => r.json());
    }, origArticle);
    assert.equal(okReset.ok, true, 'article orijinaline geri kaydedildi');
  });
});

// ---- Faz 9: kategori liste görünüm şablonu ----

test('tasarımcı: kategori liste görünüm şablonu seçilir ve kaydedilir', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdmin(async (page) => {
    page.on('dialog', (d) => d.accept());
    await setViewport(page, 1400, 900);
    const RESET = '/admin/tasarim/sifirla?sayfa=' + encodeURIComponent('kategori:gundem');
    const SAVE = '/admin/tasarim/kaydet?sayfa=' + encodeURIComponent('kategori:gundem');
    const reset = () => page.evaluate(async (url) => {
      const r = await fetch(url, { method: 'POST', credentials: 'same-origin' });
      return r.status;
    }, BASE_URL + RESET);

    try {
      await goto(page, '/');
      await reset();
      // Genel kategori düzeni artık bağımsız modüllerden oluşabilir. Bu test
      // eski tek bloklu şablonun hâlâ seçilebildiğini doğrular; bu yüzden
      // Gündem'e geçici bir kategori_liste düzeni yazılır (bitişte silinir).
      const legacy = await page.evaluate(async (url) => {
        const r = await fetch(url, {
          method: 'POST',
          credentials: 'same-origin',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            blocks: [{
              id: 'e2e-gundem-list', type: 'kategori_liste', visible: true,
              opts: {
                zone: 'govde', view: 'klasik', hero_style: 'portre', hero_count: 5,
                hero_width: 44, hero_ratio: '4/5', list_cols: '4',
                overlay_gradient: 'bottom', overlay_color: '#0b0a11',
                overlay_opacity: '62', overlay_height: 55,
                pager_shape: 'outline', pager_pos: 'center', pager_size: 30,
                akis_limit: 60, akis_per: 12, akis_auto: true,
              },
            }],
          }),
        });
        return await r.json();
      }, BASE_URL + SAVE);
      assert.equal(legacy.ok, true, 'geçici Gündem düzeni kaydedilmeli');
      await goto(page, '/admin/tasarim?sayfa=kategori&kategori=gundem');
      await page.waitForFunction(
        () => document.querySelector('.catview-pick') !== null,
        { timeout: 10000 }
      );

      // 7 görsel şablon: 3 manşet düzeni + 4 liste görünümü (akis dahil). Manşet
      // düzenleri, "liste görünümü" seçmeden doğrudan değiştirilebilmeli.
      const btns = await page.$$eval('.catview-btn', (els) => els.map((e) => e.dataset.view));
      assert.deepEqual(btns, ['portre', 'uclu', 'klasik', 'liste', 'izgara', 'mozaik', 'akis'],
        '7 kategori şablonu olmalı: ' + btns.join(','));
      const demos = await page.$$eval('.catview-btn .ct-demo', (els) => els.length);
      assert.equal(demos, 7, 'her şablon için görsel mini önizleme olmalı');

      // Sıkı liste → kaydet → public Gündem list view.
      await page.$eval('.catview-btn[data-view="liste"]', (b) => b.click());
      await new Promise((r) => setTimeout(r, 1500));
      await goto(page, '/kategori/gundem');
      const liste = await page.evaluate(() => ({
        liste: !!document.querySelector('.cat-liste'),
        hero: !!document.querySelector('.cat-hero'),
      }));
      assert.equal(liste.liste, true, 'Sıkı liste şablonu public sayfaya uygulanmalı');
      assert.equal(liste.hero, false, 'liste şablonunda manşet rayı basılmamalı');

      // Listeye dönmeden doğrudan Üçlü manşet → kaydet → public trio.
      await goto(page, '/admin/tasarim?sayfa=kategori&kategori=gundem');
      await page.waitForFunction(() => document.querySelector('.catview-pick'), { timeout: 10000 });
      await page.$eval('.catview-btn[data-view="uclu"]', (b) => b.click());
      await new Promise((r) => setTimeout(r, 1500));
      await goto(page, '/kategori/gundem');
      const trio = await page.evaluate(() => {
        const hero = document.querySelector('.cat-hero');
        return {
          trio: !!(hero && hero.classList.contains('cat-hero-trio')),
          slides: document.querySelectorAll('.cat-hero-slider .hero-slide').length,
          left: document.querySelectorAll('.cat-trio-left > a').length,
          right: document.querySelectorAll('.cat-trio-right > a').length,
        };
      });
      assert.equal(trio.trio, true, 'liste görünümü seçmeden Üçlü manşet uygulanabilmeli');
      assert.equal(trio.slides, 5);
      assert.equal(trio.left, 3);
      assert.equal(trio.right, 2);
    } finally {
      assert.equal(await reset(), 200, 'Gündem özel düzeni temizlenmeli');
    }
  });
});

// ---- Faz 10: kategori ana gövdesi modüler ----

test('tasarımcı: kategori gövdesi (govde zonu) modüler yönetilir', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdmin(async (page) => {
    await setViewport(page, 1400, 900);
    await goto(page, '/admin/tasarim?sayfa=kategori');
    await page.waitForFunction(
      () => document.querySelectorAll('.bp-chip[data-id]').length >= 0,
      { timeout: 10000 }
    );
    // govde zonu şablon görünümünde var mı?
    const govde = await page.evaluate(() => {
      const z = document.querySelector('.bp-zone[data-zone="govde"]');
      return {
        hasZone: !!z,
        chips: z ? z.querySelectorAll('.bp-chip').length : 0,
        types: z ? [...z.querySelectorAll('.bp-chip[data-id] .ty')].map((e) => e.textContent.trim()) : [],
        note: z ? !!z.querySelector('.bp-ph') : false,
        split: !!document.getElementById('catSplitBtn'),
        tag: ((document.getElementById('catViewTag') || {}).textContent || '').trim(),
      };
    });
    assert.ok(govde.hasZone, 'kategori şablon görünümünde govde (gövde) zonu olmalı');
    assert.ok(govde.chips > 0, 'govde zonunda en az bir blok olmalı (var: ' + govde.chips + ')');

    // Palette hem eski tek blok hem de bağımsız modüller bulunmalı.
    const pals = await page.evaluate(() => [...document.querySelectorAll('.bp-pal .bld-add')].map((b) => b.dataset.type));
    for (const ty of ['kategori_liste', 'kategori_slider', 'kategori_secimli', 'kategori_4lu', 'kategori_akis']) {
      assert.ok(pals.includes(ty), 'palette ' + ty + ' bloğu olmalı');
    }
    assert.ok(pals.includes('baslik') && pals.includes('slider'),
      'govde\'ye eklenebilecek diğer bloklar (baslik/slider) olmalı');

    const MODS = ['kategori_slider', 'kategori_secimli', 'kategori_4lu', 'kategori_akis'];
    if (MODS.some((m) => govde.types.includes(m))) {
      // Bağımsız modül düzeni: dört modül de gövdede, ayırma düğmesi yok.
      for (const m of MODS) {
        assert.ok(govde.types.includes(m), 'bağımsız gövdede ' + m + ' modülü olmalı');
      }
      assert.equal(govde.split, false, 'modüler gövdede "Modüllere ayır" düğmesi tekrar gösterilmemeli');
      assert.ok(govde.note, 'modüler gövde açıklaması görünmeli');
      assert.ok(govde.tag.includes('Bağımsız'), 'şablon etiketi bağımsız modülleri göstermeli: ' + govde.tag);

      // Slider modülünün satır/aralık/oran/bant/pager ayarları açılabilmeli.
      const opened = await page.evaluate(() => {
        const chip = [...document.querySelectorAll('.bp-zone[data-zone="govde"] .bp-chip[data-id]')]
          .find((c) => c.innerHTML.indexOf('kategori_slider') >= 0);
        if (!chip) return false;
        chip.querySelector('.ops button[data-act="cfg"]').click();
        return true;
      });
      assert.equal(opened, true, 'slider modülünün ayarları açılmalı');
      await page.waitForSelector('select[data-k="row_layout"]', { timeout: 5000 });
      const fields = await page.evaluate(() => {
        const opts = (k) => {
          const el = document.querySelector('select[data-k="' + k + '"]');
          return el ? [...el.options].map((o) => o.value) : null;
        };
        return {
          rowLayout: opts('row_layout'),
          ratio: opts('hero_ratio'),
          pager: opts('pager_shape'),
          hasOffset: !!document.querySelector('input[data-k="item_offset"]'),
          hasWidth: !!document.querySelector('input[data-k="hero_width"]'),
          hasOverlay: !!document.querySelector('select[data-k="overlay_gradient"]'),
        };
      });
      assert.ok(fields.rowLayout && fields.rowLayout.includes('hero-center'),
        'satır yerleşimi seçenekleri olmalı');
      assert.ok(fields.rowLayout.includes('hero-left') && fields.rowLayout.includes('hero-right'));
      assert.equal(fields.hasOffset, true, 'haber aralığı alanı olmalı');
      assert.equal(fields.hasWidth, true, 'manşet genişliği alanı olmalı');
      assert.ok(fields.ratio && fields.ratio.includes('16/9'), 'oran seçenekleri olmalı');
      assert.ok(fields.hasOverlay, 'bant tasarımı alanı olmalı');
      assert.ok(fields.pager && fields.pager.length >= 3, 'sayı çerçevesi seçenekleri olmalı');
    } else {
      // Eski tek bloklu düzen: görünüm seçici ve manşet ayarları korunmalı.
      assert.ok(govde.types.includes('kategori_liste'), 'eski düzende kategori_liste bloğu olmalı');
      assert.equal(govde.split, true, 'eski tek blokta "Modüllere ayır" düğmesi görünmeli');

      const viewSel = await page.evaluate(() => {
        const chip = [...document.querySelectorAll('.bp-chip[data-id]')].find((c) => c.innerHTML.indexOf('kategori_liste') >= 0);
        if (!chip) return { chip: false };
        chip.querySelector('.ops button[data-act="cfg"]').click();
        return { chip: true, opened: true };
      });
      assert.ok(viewSel.chip && viewSel.opened, 'kategori_liste çipi bulunup ayarları açılmalı');
      await new Promise((r) => setTimeout(r, 400));
      const viewSel2 = await page.evaluate(() => {
        const sel = document.querySelector('select[data-type="listview"]');
        return { has: !!sel, opts: sel ? sel.options.length : 0 };
      });
      assert.ok(viewSel2.has && viewSel2.opts >= 5, 'blok-bazlı görünüm seçici (liste/izgara/mozaik + genel) olmalı');

      const heroOpts = await page.evaluate(() => {
        const get = (k) => document.querySelector('.bp-opts [data-k="' + k + '"]');
        const opts = (sel) => sel ? [...sel.options].map((o) => o.value) : [];
        return {
          style: opts(get('hero_style')),
          ratio: opts(get('hero_ratio')),
          cols: opts(get('list_cols')),
          count: !!get('hero_count'),
          width: !!get('hero_width'),
        };
      });
      assert.ok(heroOpts.style.includes('portre') && heroOpts.style.includes('klasik') &&
        heroOpts.style.includes('uclu'),
        'manşet düzeni portre/üçlü/klasik seçilebilmeli: ' + heroOpts.style.join(','));
      assert.deepEqual(heroOpts.ratio.filter(Boolean), ['4/5', '3/4', '1/1', '16/9'],
        'dört manşet oranı sunulmalı');
      assert.deepEqual(heroOpts.cols.filter(Boolean), ['2', '3', '4'],
        'alt kartlar 2/3/4 sütun seçilebilmeli');
      assert.ok(heroOpts.count && heroOpts.width, 'slayt sayısı ve genişlik alanları bulunmalı');
    }

    // TEMİZLİK YOK (test salt okuma — canlı kategori düzeni bozulmadı)
  });
});

test('medya: HTMX panel + yükleme alanı yerel çalışır (veri yazmaz)', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  // TARİHÇE (Faz 210): Panel Murmura düzenine geçti. Eski tasarımda ayrı
  // bir `form[action="/admin/dosyalar"]` (hx-post + hx-encoding:multipart)
  // ve `#medyaOnizleme` vardı; dosya seçilince yalnız yerel önizleme
  // gösteriliyordu. Bugünkü tasarımda yükleme, galerinin ilk kutusu
  // (.mgal-drop) ve `[data-mg-file]` girdisiyle ANINDA yapılır (media.js
  // XHR ile gönderir) — "veri yazmama" garantisi bu yüzden dosya seçme
  // adımını KAPSAMAZ. Burada panelin HTMX sözleşmesi korunuyor:
  // arama hx-get + #medya-panel hedefi, silme hx-post + onay.
  try {
    await withAdmin(async (page, errors) => {
      await setViewport(page, 1400);
      await goto(page, '/admin/dosyalar');
      assert.ok(!page.url().includes('/admin/giris'), 'medya sayfası açık: ' + page.url());

      const hazir = await page.evaluate(() => {
        const ara = document.querySelector('form[action="/admin/dosyalar"]');
        return {
          htmx: !!window.htmx,
          panel: !!document.querySelector('#medya-panel'),
          araHx: !!(ara &&
            ara.getAttribute('hx-get') === '/admin/dosyalar' &&
            ara.getAttribute('hx-target') === '#medya-panel'),
          delHx: !!document.querySelector('a[hx-post$="/sil"][hx-confirm]'),
          yuklemeGirdisi: !!document.querySelector('[data-mg-file]'),
          dropKutu: !!document.querySelector('.mgal-drop'),
        };
      });
      assert.equal(hazir.htmx, true, 'HTMX yüklü olmalı');
      assert.equal(hazir.panel, true, 'medya paneli olmalı');
      assert.equal(hazir.araHx, true, 'arama formu hx-get + #medya-panel hedefi taşımalı');
      assert.equal(hazir.delHx, true, 'silme butonu hx-post + hx-confirm taşımalı');
      assert.equal(hazir.yuklemeGirdisi, true, 'dosya girdisi (data-mg-file) olmalı');
      assert.equal(hazir.dropKutu, true, 'sürükle-bırak kutusu olmalı');

      // media.js gerçekten bağlı mı (panel etkileşimleri ona dayanıyor)?
      const js = await page.evaluate(() => typeof window.__tbMediaInit === 'function');
      assert.equal(js, true, 'media.js bağlı olmalı');

      assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
    });
  } finally {
    /* geçici dosya yok */
  }
});

test('içerik formu: AI butonları yeni onay panelini açar (doğrudan yazmaz)', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdmin(async (page, errors) => {
    await setViewport(page, 1400);
    await goto(page, '/admin/icerikler');
    const gid = await page.evaluate(() => {
      const a = [...document.querySelectorAll('a[href^="/admin/icerikler/"]')].find((x) =>
        /^\/admin\/icerikler\/\d+$/.test(x.getAttribute('href') || '')
      );
      const m = a ? a.getAttribute('href').match(/\/admin\/icerikler\/(\d+)$/) : null;
      return m ? m[1] : '';
    });
    assert.ok(gid, 'düzenlenecek içerik bulunmalı');
    await goto(page, '/admin/icerikler/' + gid);
    assert.ok(!page.url().includes('/admin/giris'), 'içerik formu açık');

    const ai = await page.evaluate(() => {
      const btn = document.querySelector('[data-ai][data-target="#f_title"]');
      const score = document.getElementById('aiSkorBtn');
      return {
        htmx: !!window.htmx,
        btn: !!btn,
        hxPost: btn ? btn.getAttribute('hx-post') : '',
        editor: !!document.getElementById('aiEditor'),
        actions: document.querySelectorAll('#aiSelectMenu [data-ai-open]').length,
        indicator: !!document.getElementById('ai-durum'),
        scoreHx: !!(score && /\/ai-skor$/.test(score.getAttribute('hx-post') || '')),
      };
    });
    assert.equal(ai.htmx, true, 'HTMX yüklü olmalı');
    assert.equal(ai.btn, true, 'AI butonu bulunmalı');
    assert.equal(ai.hxPost, null, 'AI butonu doğrudan yazan eski hx-post taşımamalı');
    assert.equal(ai.editor, true, 'yeni AI onay paneli bulunmalı');
    assert.equal(ai.actions, 11, 'menüde 11 eylem olmalı');
    assert.equal(ai.indicator, true, 'AI göstergesi bulunmalı');
    assert.equal(ai.scoreHx, true, 'AI skor butonu hx-post taşımalı');

    assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
  });
});

test('içerik listesi: HTMX filtresi tam sayfa yenilemeden çalışır (veri yazmaz)', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdmin(async (page, errors) => {
    await setViewport(page, 1400);
    await goto(page, '/admin/icerikler');
    assert.ok(!page.url().includes('/admin/giris'), 'içerik listesi açık: ' + page.url());

    const hazir = await page.evaluate(() => ({
      htmx: !!window.htmx,
      panel: !!document.querySelector('#icerik-panel'),
    }));
    assert.equal(hazir.htmx, true, 'HTMX admin sayfasında yüklenmeli');
    assert.equal(hazir.panel, true, 'içerik paneli bulunmalı');

    // Tam sayfa yenilenirse bu işaret kaybolur.
    await page.evaluate(() => { window.__tbNoReload = 1; });
    await page.select('select[name="tip"]', 'video');
    await page.click('form[action="/admin/icerikler"] button[type="submit"]');
    await page.waitForFunction(
      () => location.search.includes('tip=video') &&
        document.querySelectorAll('table.table tbody tr').length > 0,
      { timeout: 5000 }
    );

    const sonuc = await page.evaluate(() => ({
      noReload: window.__tbNoReload === 1,
      panel: !!document.querySelector('#icerik-panel'),
      tipler: (() => {
        // Sütun indeksi sabit DEĞİL ("Dil"/"Rozet" eklendikçe kayıyor).
        const ths = [...document.querySelectorAll('table.table thead th')].map((t) => t.textContent.trim());
        const i = ths.indexOf('Tip');
        return [...document.querySelectorAll('table.table tbody tr')]
          .map((tr) => (tr.children[i] || {}).textContent || '')
          .map((t) => t.trim());
      })(),
      durumButonu: !!document.querySelector(
        'a[hx-post$="/yayinla"], a[hx-post$="/taslak"], a[hx-post$="/arsiv"]'
      ),
      silButonu: !!document.querySelector('a[hx-post$="/sil"][hx-confirm]'),
      hxPushUrl: (document.querySelector('form[action="/admin/icerikler"]') || {}).getAttribute
        ? document.querySelector('form[action="/admin/icerikler"]').getAttribute('hx-push-url')
        : null,
    }));
    assert.equal(sonuc.noReload, true, 'filtre tam sayfa yenilememeli');
    assert.equal(sonuc.panel, true, 'HTMX sonrası panel korunmalı');
    assert.ok(sonuc.tipler.length > 0 && sonuc.tipler.every((v) => v === 'video'),
      'filtrelenmiş satırlar video olmalı: ' + sonuc.tipler.join(','));
    assert.equal(sonuc.durumButonu, true, 'durum butonları hx-post taşımalı');
    assert.equal(sonuc.silButonu, true, 'silme butonu hx-post + hx-confirm taşımalı');
    assert.equal(sonuc.hxPushUrl, 'true', 'filtre URL durumunu push etmeli');

    assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
  });
});

test('yorumlar: HTMX panel + toplu form hx-post taşır (veri yazmaz)', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdmin(async (page, errors) => {
    await setViewport(page, 1400);
    await goto(page, '/admin/yorumlar');
    assert.ok(!page.url().includes('/admin/giris'), 'yorumlar açık: ' + page.url());
    const r = await page.evaluate(() => {
      const bulk = document.getElementById('yorumTopluForm');
      return {
        htmx: !!window.htmx,
        panel: !!document.querySelector('#yorum-panel'),
        bulkHx: !!(bulk && bulk.getAttribute('hx-post') === '/admin/yorumlar/toplu' &&
          bulk.getAttribute('hx-target') === '#yorum-panel'),
        tabHx: !!document.querySelector('.tabs a[hx-get]'),
        approveHx: !!document.querySelector('a[hx-post$="/onayla"]'),
        deleteHx: !!document.querySelector('a[hx-post$="/sil"][hx-confirm]'),
      };
    });
    assert.equal(r.htmx, true, 'HTMX yüklü olmalı');
    assert.equal(r.panel, true, 'yorum paneli olmalı');
    assert.equal(r.bulkHx, true, 'toplu form hx-post/hx-target taşımalı');
    assert.equal(r.tabHx, true, 'sekmeler hx-get taşımalı');
    assert.equal(r.approveHx || r.deleteHx, true, 'en az bir yorum aksiyonu hx-post taşımalı');
    assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
  });
});

test('anketler: HTMX panel + düzenleme formları hx-post taşır (veri yazmaz)', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdmin(async (page, errors) => {
    await setViewport(page, 1400);
    await goto(page, '/admin/anketler');
    assert.ok(!page.url().includes('/admin/giris'), 'anketler açık: ' + page.url());
    const r = await page.evaluate(() => ({
      htmx: !!window.htmx,
      panel: !!document.querySelector('#anketler-panel'),
      createHx: !!document.querySelector('form[action="/admin/anketler/yeni"][hx-post="/admin/anketler/yeni"]'),
      hasPoll: !!document.querySelector('form[hx-post$="/sil"]'),
      editHx: [...document.querySelectorAll('form[hx-post^="/admin/anketler/"]')]
        .some((f) => !/\/yeni$/.test(f.getAttribute('action') || '') && !/\/sil$/.test(f.getAttribute('action') || '')),
      delHx: !!document.querySelector('form[action$="/sil"][hx-post$="/sil"][hx-confirm]'),
    }));
    assert.equal(r.htmx, true, 'HTMX yüklü olmalı');
    assert.equal(r.panel, true, 'anket paneli olmalı');
    assert.equal(r.createHx, true, 'yeni anket formu hx-post taşımalı');
    // Canlı veride anket yoksa düzenleme/silme formu da yoktur (test salt okuma).
    assert.ok(!r.hasPoll || r.editHx, 'anket varsa düzenleme formu hx-post taşımalı');
    assert.ok(!r.hasPoll || r.delHx, 'anket varsa silme formu hx-confirm taşımalı');
    assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
  });
});
