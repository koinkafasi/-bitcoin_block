'use strict';
/** Genel site duman testleri: temel sayfalar yükleniyor mu? */

const test = require('node:test');
const assert = require('node:assert/strict');
const { BASE_URL, goto, withPage } = require('../lib/harness');

test('anasayfa yükleniyor ve başlık/üst yazı var', async () => {
  await withPage(async (page) => {
    const resp = await goto(page, '/');
    assert.equal(resp.status(), 200, 'anasayfa 200 dönmeli');
    assert.ok(await page.title(), '<title> boş olmamalı');

    const header = await page.$('header');
    assert.ok(header, 'üst çubuk (header) bulunamadı');

    const linkCount = await page.$$eval('a[href]', (as) => as.length);
    assert.ok(linkCount > 10, `makul bağlantı olmalı, bulunan: ${linkCount}`);
  });
});

test('viewport meta etiketi tanımlı (mobil ölçekleme)', async () => {
  await withPage(async (page) => {
    await goto(page, '/');
    const meta = await page.$eval('meta[name="viewport"]', (m) => m.getAttribute('content'));
    assert.ok(meta, 'viewport meta etiketi yok — mobilde sayfa 980px ölçeklenir');
    assert.match(meta, /width=device-width/, 'viewport genişliği device-width olmalı');
  });
});

test('statik CSS sunuluyor ve sürüm parametresi taşıyor', async () => {
  await withPage(async (page) => {
    await goto(page, '/');
    const href = await page.$eval('link[href*="public.css"]', (l) => l.getAttribute('href'));
    assert.ok(href, 'public.css bağlantısı yok');
    assert.match(href, /public\.css\?v=/, 'önbellek kırıcı sürüm parametresi olmalı');

    const status = await page.evaluate(async (u) => {
      const r = await fetch(u);
      return r.status;
    }, href);
    assert.equal(status, 200, 'public.css sunulmalı');
  });
});

test('robots.txt ve rss.xml erişilebilir', async () => {
  await withPage(async (page) => {
    for (const path of ['/robots.txt', '/rss.xml']) {
      const resp = await goto(page, path);
      assert.equal(resp.status(), 200, `${path} 200 dönmeli`);
    }
  });
});

test('arama sayfası boş sorguyla bozulmaz', async () => {
  await withPage(async (page) => {
    const resp = await goto(page, '/arama');
    assert.equal(resp.status(), 200, 'boş arama sayfası 200 dönmeli');
  });
});

test('site haritası üretiliyor', async () => {
  await withPage(async (page) => {
    const resp = await goto(page, '/sitemap.xml');
    assert.equal(resp.status(), 200);
    const body = await page.content();
    assert.match(body, /<(urlset|url)/, 'geçerli sitemap olmalı');
  });
});

test('kategori sayfası içerik listeliyor', async () => {
  await withPage(async (page) => {
    const resp = await goto(page, '/kategori/gundem');
    assert.equal(resp.status(), 200, 'kategori sayfası 200 dönmeli');

    // Haber bağlantıları slug-id biçiminde biter: /<slug>-<hex>-<id>
    const items = await page.$$eval('a[href]', (as) =>
      as
        .map((a) => a.getAttribute('href') || '')
        .filter((h) => /\/[^/]+-[0-9a-f]{4,8}-\d+$/.test(h))
    );
    assert.ok(items.length > 0, 'kategoride en az bir haber bağlantısı olmalı');

    // İlk haber bağlantısı gerçekten bir yazı sayfasına götürmeli
    const first = await page.evaluate(() => {
      const a = [...document.querySelectorAll('a[href]')].find((x) =>
        /\/[^/]+-[0-9a-f]{4,8}-\d+$/.test(x.getAttribute('href') || '')
      );
      return a ? a.getAttribute('href') : null;
    });
    assert.ok(first, 'haber bağlantısı bulunamadı');

    const artResp = await goto(page, first);
    assert.equal(artResp.status(), 200, `yazı sayfası açılmalı: ${first}`);
    const text = await page.evaluate(() => document.body.innerText);
    assert.ok(text.length > 200, 'yazı sayfası içerik göstermeli');
  });
});

test('yazı sayfasında Önerilen Video kartı görünür (video haber varsa)', async () => {
  await withPage(async (page) => {
    await goto(page, '/');
    const first = await page.evaluate(() => {
      const a = [...document.querySelectorAll('a[href]')].find((x) =>
        /\/[^/]+-[0-9a-f]{4,8}-\d+$/.test(x.getAttribute('href') || '')
      );
      return a ? a.getAttribute('href') : null;
    });
    assert.ok(first, 'haber bağlantısı bulunamadı');
    await goto(page, first);

    const card = await page.$('.rel-video .rv-card');
    if (!card) {
      // Video yayını yoksa üretilmez — test isteğe bağlı kabul edilir.
      assert.ok(true, 'video yayını yok, kart isteğe bağlı');
      return;
    }
    const label = await page.$$eval('.rel-video .rv-label', (els) => els.map((e) => e.textContent).join(''));
    assert.match(label, /ÖNERİLEN VİDEO/, 'kart etiketi "ÖNERİLEN VİDEO" olmalı');
    const href = await page.$eval('.rel-video .rv-card', (e) => e.getAttribute('href'));
    assert.ok(href && href.length > 0, 'kart bir video haberine götürmeli');
    // Faz 16: video kategorisi varsa tam olarak bir rozet basılır.
    const badges = await page.$$eval('.rel-video .rv-cat', (els) => els.map((e) => e.textContent.trim()));
    assert.ok(badges.length <= 1, 'kartta en fazla bir kategori rozeti olmalı');
    for (const b of badges) assert.ok(b.length > 0, 'rozet metni boş olmamalı');
  });
});

test('yazı sayfasında kategori rozetleri basılır', async () => {
  await withPage(async (page) => {
    await goto(page, '/');
    const first = await page.evaluate(() => {
      const a = [...document.querySelectorAll('a[href]')].find((x) =>
        /\/[^/]+-[0-9a-f]{4,8}-\d+$/.test(x.getAttribute('href') || '')
      );
      return a ? a.getAttribute('href') : null;
    });
    assert.ok(first, 'haber bağlantısı bulunamadı');
    await goto(page, first);

    const all = await page.evaluate(() => ({
      rv: [...document.querySelectorAll('.rv-cat')].map((e) => e.textContent.trim()),
      nh: [...document.querySelectorAll('.nh-cat')].map((e) => e.textContent.trim()),
      nested: document.querySelectorAll('a a').length,
    }));
    for (const b of all.rv.concat(all.nh)) {
      assert.ok(b.length > 0, 'kategori rozeti boş metin içermemeli');
      assert.ok(b.length <= 40, 'kategori rozeti makul uzunlukta olmalı: ' + b);
    }
    // Süre rozeti (Faz 17): varsa mm:ss / sa:dk:ss biçiminde olmalı.
    const sureler = await page.evaluate(() =>
      [...document.querySelectorAll('.rv-sure')].map((e) => e.textContent.trim())
    );
    for (const s of sureler) {
      assert.match(s, /^\d{1,2}:\d{2}(:\d{2})?$/, 'süre rozeti biçimi geçersiz: ' + s);
      const parcalar = s.split(':').map(Number);
      assert.ok(parcalar[1] < 60, 'dakika 60 altında olmalı: ' + s);
      if (parcalar.length === 3) assert.ok(parcalar[2] < 60, 'saniye 60 altında olmalı: ' + s);
    }
    assert.ok(all.nh.length <= 2, 'en fazla iki haber bandı olmalı (önceki + sonraki)');
    // Kart/band zaten <a>; içlerine <a> gömmek geçersiz HTML'dir ve DOM'u bozar.
    assert.equal(all.nested, 0, 'iç içe <a> oluşmamalı');
  });
});

test('yazı sayfasında Google News site kaynağı rozeti (site adresi bağlantıda)', async () => {
  await withPage(async (page) => {
    await goto(page, '/');
    const first = await page.evaluate(() => {
      const a = [...document.querySelectorAll('a[href]')].find((x) =>
        /\/[^/]+-[0-9a-f]{4,8}-\d+$/.test(x.getAttribute('href') || '')
      );
      return a ? a.getAttribute('href') : null;
    });
    assert.ok(first, 'haber bağlantısı bulunamadı');
    await goto(page, first);

    // Rozet Google News kaynak ekleme adresine gitmeli ve SİTE ADRESİ bağlantıya
    // gömülü olmalı (?q=<site>). Yazının orijinal kaynağına gitmek hatalıydı.
    const badge = await page.evaluate(() => {
      const a = document.querySelector('a.src-badge');
      return a
        ? {
            href: a.getAttribute('href'),
            resolved: a.href,
            rel: a.getAttribute('rel'),
            text: a.textContent.trim(),
          }
        : null;
    });
    assert.ok(badge, 'Google News rozeti (a.src-badge) olmalı');
    assert.match(
      badge.href,
      /^https:\/\/www\.google\.com\/preferences\/source\?q=/,
      'rozet Google News kaynak adresine gitmeli: ' + badge.href
    );
    // ?q= değeri YALNIZCA domain olmalı: şema ve yol yok.
// (Geliştirme sunucusunda port host'un parçası olduğu için :port tolere edilir;
//  canlıda theblock.com.tr gibi çıplak domain basılır.)
    const q = badge.href.split('?q=')[1] || '';
    assert.ok(q.length > 0, 'bağlantıda domain olmalı: ' + badge.href);
    assert.doesNotMatch(q, /^https?:/i, 'sorgu değerinde şema (http/https) olmamalı: ' + q);
    assert.doesNotMatch(q, /\//, 'sorgu değerinde yol olmamalı: ' + q);
    assert.doesNotMatch(q, /%3A|%2F/i, 'sorgu değeri kaçışlanmamalı (okunur kalmalı): ' + q);
    assert.match(q, /^[a-z0-9.-]+(:\d+)?$/i, 'sorgu değeri çıplak domain olmalı: ' + q);
    assert.match(
      badge.resolved,
      /^https:\/\/www\.google\.com\/preferences\/source\?q=[a-z0-9.-]+(:\d+)?$/i,
      'tarayıcıda çözümlenen adres doğru olmalı: ' + badge.resolved
    );
    assert.match(badge.rel || '', /noopener/, 'yeni sekme güvenliği (noopener) olmalı');
    assert.match(badge.text, /Google News/, 'rozet etiketi Google News içermeli');

    // RSS kopyala düğmesi kaldırıldı (Faz 20): geri gelmemeli.
    const kopya = await page.evaluate(() => ({
      btn: document.querySelectorAll('#rssCopyBtn').length,
      srcCopy: document.querySelectorAll('.src-copy').length,
    }));
    assert.equal(kopya.btn, 0, 'RSS kopyala düğmesi kaldırılmış olmalı');
    assert.equal(kopya.srcCopy, 0, '.src-copy sınıfı kalmamalı');

    // Orijinal haber kaynağı bağlantısı yerinde olmalı (ayrı basılır).
    const kaynak = await page.evaluate(() => {
      const a = [...document.querySelectorAll('a')].find((x) =>
        /^Kaynak:/.test(x.textContent.trim())
      );
      return a ? { href: a.getAttribute('href'), text: a.textContent.trim() } : null;
    });
    if (kaynak) {
      assert.match(kaynak.href || '', /^https?:\/\//, 'Kaynak bağlantısı dış adres olmalı');
      assert.ok(kaynak.text.length > 'Kaynak:'.length, 'Kaynak metni kaynak adı içermeli');
    }
  });
});

test('yazı sayfasında Sıradaki Haber bandı (son içerik değilse) görünür', async () => {
  await withPage(async (page) => {
    await goto(page, '/');
    // "İlginizi çekebilir" benzeri makale linklerinden farklı: yazı sonundaki
    // sonraki-haber bandı, ilk iki makalede olmayabilir (en yenide yoktur).
    const first = await page.evaluate(() => {
      const a = [...document.querySelectorAll('a[href]')].find((x) =>
        /\/[^/]+-[0-9a-f]{4,8}-\d+$/.test(x.getAttribute('href') || '')
      );
      return a ? a.getAttribute('href') : null;
    });
    assert.ok(first, 'haber bağlantısı bulunamadı');
    await goto(page, first);
    // En yeni makale ise band olmaz (istemeye bağlı); var ise link doğrulanır.
    const band = await page.$('.next-haber:not(.prev-haber)');
    if (band) {
      const href = await page.$eval('.next-haber:not(.prev-haber)', (e) => e.getAttribute('href'));
      assert.ok(href && href.length > 0, 'band bir sonraki habere götürmeli');
      const label = await page.$eval('.next-haber:not(.prev-haber) .nh-path', (e) => e.textContent.trim());
      assert.match(label, /SONRAKİ HABER/i, 'band etiketi "SONRAKİ HABER" olmalı');
    } else {
      assert.ok(true, 'en yeni makale — band bu örnekte beklenmez (isteğe bağlı)');
    }
  });
});

test('yazı sayfasında Önceki Haber bandı (ilk içerik değilse) görünür', async () => {
  await withPage(async (page) => {
    await goto(page, '/');
    const first = await page.evaluate(() => {
      const a = [...document.querySelectorAll('a[href]')].find((x) =>
        /\/[^/]+-[0-9a-f]{4,8}-\d+$/.test(x.getAttribute('href') || '')
      );
      return a ? a.getAttribute('href') : null;
    });
    assert.ok(first, 'haber bağlantısı bulunamadı');
    await goto(page, first);
    // En eski makale ise band olmaz (isteğe bağlı); varsa çift yönlü nav doğrulanır.
    const prev = await page.$('.prev-haber');
    if (!prev) {
      assert.ok(true, 'en eski makale — önceki band beklenmez (isteğe bağlı)');
      return;
    }
    const href = await page.$eval('.prev-haber', (e) => e.getAttribute('href'));
    assert.ok(href && href.length > 0, 'band bir önceki habere götürmeli');
    const label = await page.$eval('.prev-haber .nh-path', (e) => e.textContent.trim());
    assert.match(label, /ÖNCEKİ HABER/i, 'band etiketi "ÖNCEKİ HABER" olmalı');
    const arrow = await page.$eval('.prev-haber .nh-arrow', (e) => e.textContent.trim());
    assert.match(arrow, /←/, 'önceki band oku sola (←) göstermeli');
    // Önceki/sonraki ikilisi aynı sayfada birlikte görünebilmeli
    const counts = await page.evaluate(() => ({
      prev: document.querySelectorAll('.prev-haber').length,
      next: document.querySelectorAll('.next-haber:not(.prev-haber)').length,
    }));
    assert.ok(counts.prev <= 1 && counts.next <= 1, 'her band en fazla bir kez görünmeli');
  });
});

test('makale sayfasında kanal paylaşım düğmeleri doğru adresle basılır (Faz 44)', async () => {
  await withPage(async (page) => {
    await goto(page, '/');
    const first = await page.evaluate(() => {
      const a = [...document.querySelectorAll('a[href]')].find((x) =>
        /\/[^/]+-[0-9a-f]{4,8}-\d+$/.test(x.getAttribute('href') || '')
      );
      return a ? a.getAttribute('href') : null;
    });
    assert.ok(first, 'haber bağlantısı bulunamadı');
    await goto(page, first);

    const durum = await page.evaluate(() => {
      const grup = document.querySelector('.art-share-menu');
      if (!grup) return { grup: false };
      const links = [...grup.querySelectorAll('a[href]')].map((a) => a.getAttribute('href'));
      return {
        grup: true,
        fb: links.some((h) => h.includes('facebook.com/sharer')),
        x: links.some((h) => h.includes('twitter.com/intent')),
        wa: links.some((h) => h.includes('wa.me')),
        tg: links.some((h) => h.includes('t.me/share')),
        li: links.some((h) => h.includes('linkedin.com/sharing')),
        bs: links.some((h) => h.includes('bsky.app/intent')),
        adet: links.length,
        tekKodlama: links.every((h) => !/%25/.test(h)),
        kopyala: !!document.querySelector('.art-copy-link[data-kopyala]'),
      };
    });
    assert.equal(durum.grup, true, 'paylaşım menüsü (.art-share-menu) olmalı');
    assert.ok(durum.fb && durum.x && durum.wa && durum.tg && durum.li && durum.bs,
      'altı kanal bağlantısı olmalı: ' + JSON.stringify(durum));
    assert.ok(durum.adet >= 6, 'en az 6 paylaşım bağlantısı olmalı (' + durum.adet + ')');
    assert.equal(durum.tekKodlama, true, 'paylaşım linkleri çifte-kodlu (%25) olmamalı');
    assert.equal(durum.kopyala, true, '"bağlantıyı kopyala" düğmesi mutlak adresle olmalı');
  });
});

test('yazı sayfasında yorum formunda spam tuzağı gizli ve doldurulabilir değil (Faz 35)', async () => {
  await withPage(async (page) => {
    await goto(page, '/');
    const first = await page.evaluate(() => {
      const a = [...document.querySelectorAll('a[href]')].find((x) =>
        /\/[^/]+-[0-9a-f]{4,8}-\d+$/.test(x.getAttribute('href') || '')
      );
      return a ? a.getAttribute('href') : null;
    });
    assert.ok(first, 'haber bağlantısı bulunamadı');
    await goto(page, first);

    const tuzak = await page.evaluate(() => {
      const form = document.querySelector('form[action^="/yorum/"]');
      if (!form) return null; // yorumlar kapalıysa form yoktur
      const input = form.querySelector('input[name="web"]');
      if (!input) return { formVar: true, tuzakVar: false };
      const r = input.getBoundingClientRect();
      const cs = getComputedStyle(input);
      return {
        formVar: true,
        tuzakVar: true,
        gizli: input.closest('[aria-hidden="true"]') !== null,
        disariTasinda: r.width < 5 && r.height < 5 || r.bottom < 0 || r.right < 0 || cs.position === 'absolute',
        tabindex: input.getAttribute('tabindex'),
        otomatikDoldurmaKapali: input.autocomplete === 'off',
      };
    });
    if (!tuzak) {
      assert.ok(true, 'bu makalede yorum formu yok (yorumlar kapalı olabilir) — isteğe bağlı');
      return;
    }
    assert.equal(tuzak.tuzakVar, true, 'yorum formunda gizli spam tuzağı (name="web") olmalı');
    assert.equal(tuzak.gizli, true, 'tuzak kapsayıcısı aria-hidden olmalı');
    assert.equal(tuzak.disariTasinda, true, 'tuzak görünür alan dışında olmalı (ekran dışına taşınmış)');
    assert.equal(tuzak.tabindex, '-1', 'tuzak Tab sırası dışında olmalı');
    assert.equal(tuzak.otomatikDoldurmaKapali, true, 'tuzakta otomatik doldurma kapalı olmalı');
  });
});
