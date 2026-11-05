'use strict';
/**
 * Dikte (sesle yazma) — Faz 117.
 *
 * Doğrulananlar:
 *  - Araç çubuğunda "Sesle yaz (dikte)" düğmesi var ve erişilebilir adı doğru.
 *  - Tıklanınca durum çubuğu açılır ve KULLANICIYA AÇIK bir mesaj gösterir:
 *    ya "Dinliyorum…" (API varsa) ya da neden çalışmadığı (desteklenmiyor /
 *    izin yok). Sessiz başarısızlık olmamalı.
 *  - Durdur / Escape çubuğu kapatır (mikrofon açık kalmaz).
 *  - Dikte hiçbir şeyi otomatik KAYDETMEZ: gizli `detay` alanı ve içerik
 *    değişmez.
 *
 * Not: headless Chrome'da SpeechRecognition çoğu ortamda yoktur veya ağ
 * hatası verir. Test her iki dalı da kabul eder; amaç hata yolunun sessiz
 * kalmamasını garanti etmek.
 */
const test = require('node:test');
const assert = require('node:assert/strict');
const { BASE_URL, goto, setViewport, withAdminPage } = require('../lib/harness');

const SID = process.env.TB_SESSION;

async function waitEditor(page) {
  await page.waitForFunction(
    () => window.__contentEditor && document.querySelector('#blockEditor .ProseMirror'),
    { timeout: 20000 },
  );
  await page.waitForFunction(
    () => window.__collabActive !== true || window.__collabSynced === true,
    { timeout: 15000 },
  ).catch(() => {});
}

/** İlk düzenlenebilir içeriğin kimliğini bulur (yeni kayıt oluşturmaz). */
async function firstContentID(page) {
  await goto(page, '/admin/icerikler');
  const id = await page.evaluate(() => {
    const a = [...document.querySelectorAll('a[href*="/admin/icerikler/"]')]
      .map((el) => el.getAttribute('href') || '')
      .map((h) => (h.match(/^\/admin\/icerikler\/(\d+)$/) || [])[1])
      .filter(Boolean);
    return a[0] || '';
  });
  return id;
}

test('dikte: düğme var, durum çubuğu açılır, sessiz başarısız olmaz', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');

  await withAdminPage(async (page, errors) => {
    await setViewport(page, 1440, 950);

    const id = await firstContentID(page);
    if (!id) return t.skip('düzenlenecek içerik yok');
    await goto(page, '/admin/icerikler/' + id);
    await waitEditor(page);

    // 1) Düğme var ve erişilebilir.
    const btn = await page.evaluate(() => {
      const b = document.querySelector('.be-toolbar [data-cmd="dictate"]');
      if (!b) return null;
      return {
        title: b.getAttribute('title') || '',
        aria: b.getAttribute('aria-label') || '',
        visible: !!(b.offsetWidth || b.offsetHeight),
      };
    });
    assert.ok(btn, 'araç çubuğunda dikte düğmesi olmalı');
    assert.ok(/dikte/i.test(btn.title) || /dikte/i.test(btn.aria), 'düğme "dikte" olarak adlandırılmalı');
    assert.ok(btn.visible, 'dikte düğmesi görünür olmalı');

    // 2) Dikte öncesi metin ve gizli alan.
    const before = await page.evaluate(() => ({
      detail: (document.querySelector('[name="detay"]') || {}).value || '',
    }));

    // 3) Tıkla → durum çubuğu görünür + anlamlı mesaj.
    await page.evaluate(() => document.querySelector('.be-toolbar [data-cmd="dictate"]').click());
    await page.waitForFunction(
      () => {
        const bar = document.getElementById('dictationBar');
        if (!bar || bar.hidden) return false;
        const label = bar.querySelector('.be-dict-label');
        return !!label && label.textContent.trim().length > 0;
      },
      { timeout: 8000 },
    ).catch(() => {});

    const opened = await page.evaluate(() => {
      const bar = document.getElementById('dictationBar');
      if (!bar) return { exists: false };
      const label = bar.querySelector('.be-dict-label');
      return {
        exists: true,
        hidden: bar.hidden,
        label: (label && label.textContent || '').trim(),
        isError: bar.classList.contains('be-dict-error'),
        hasStop: !!bar.querySelector('#dictationStop'),
        hasLang: !!bar.querySelector('#dictationLang'),
        langValue: (bar.querySelector('#dictationLang') || {}).value || '',
        listening: !!(document.querySelector('.be-toolbar [data-cmd="dictate"]') || {}).classList?.contains('is-active'),
        shellLang: document.documentElement.lang || '',
      };
    });

    assert.ok(opened.exists, 'dikte durum çubuğu oluşturulmalı');
    assert.equal(opened.hidden, false, 'durum çubuğu görünmeli');
    assert.ok(opened.label.length > 0, 'kullanıcıya mesaj gösterilmeli (sessiz başarısızlık yok)');
    assert.ok(opened.hasStop, 'durdurma düğmesi olmalı');
    assert.ok(opened.hasLang, 'dil seçici olmalı');
    assert.ok(opened.langValue.startsWith('tr'), 'varsayılan dikte dili Türkçe olmalı: ' + opened.langValue);

    // Mesaj ya "dinliyorum" ya da açıklayıcı bir hata olmalı.
    const saysListening = /dinliyorum/i.test(opened.label);
    assert.ok(
      saysListening || opened.isError,
      'çubuk ya dinlemeyi ya da açık bir hata bildirmeli: ' + opened.label,
    );
    if (opened.isError) {
      assert.ok(/destekle|izin|HTTPS|ulaş|Mikrofon|tanınamadı/i.test(opened.label),
        'hata mesajı nedeni açıklamalı: ' + opened.label);
    }

    // 4) Durdur → çubuk kapanır (mikrofon açık kalmaz).
    await page.evaluate(() => {
      const stop = document.getElementById('dictationStop');
      if (stop) stop.click();
    });
    const closed = await page.evaluate(() => {
      const bar = document.getElementById('dictationBar');
      const active = (document.querySelector('.be-toolbar [data-cmd="dictate"]') || {}).classList?.contains('is-active');
      return { hidden: !bar || bar.hidden, active: !!active };
    });
    assert.ok(closed.hidden, 'durdurunca çubuk kapanmalı');
    assert.equal(closed.active, false, 'durdurunca düğme pasif olmalı');

    // 5) Otomatik kayıt yok: gizli alan ve içerik değişmemeli.
    const after = await page.evaluate(() => ({
      detail: (document.querySelector('[name="detay"]') || {}).value || '',
      title: (document.querySelector('input[name="baslik"]') || {}).value || '',
    }));
    assert.equal(after.detail, before.detail, 'dikte gizli detay alanını kendiliğinden değiştirmemeli');

    assert.deepEqual(errors, [], 'sayfa hatası olmamalı: ' + errors.join(' | '));
  });
});

test('dikte: konuşma metni editöre yazılır, noktalama komutu işlenir, otomatik kaydetmez', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');

  await withAdminPage(async (page) => {
    await setViewport(page, 1440, 950);
    const id = await firstContentID(page);
    if (!id) return t.skip('düzenlenecek içerik yok');
    await goto(page, '/admin/icerikler/' + id);
    await waitEditor(page);

    // SpeechRecognition'ı sahtele: deterministik "konuşma" üret.
    await page.evaluate(() => {
      class FakeSR {
        constructor() {
          this.lang = ''; this.continuous = false; this.interimResults = false; this.maxAlternatives = 1
        }
        start() {
          if (FakeSR._ran) return
          FakeSR._ran = true
          setTimeout(() => {
            if (!this.onresult) return
            const mk = (t) => Object.assign([{ transcript: t }], { isFinal: true })
            const results = Object.assign([mk('Belediye yeni projeyi duyurdu'), mk('nokta')], { length: 2 })
            this.onresult({ resultIndex: 0, results })
          }, 80)
        }
        stop() { if (this.onend) this.onend() }
        abort() {}
      }
      window.SpeechRecognition = FakeSR
      window.webkitSpeechRecognition = FakeSR
    })

    await page.evaluate(() => document.querySelector('.be-toolbar [data-cmd="dictate"]').click());
    await page.waitForFunction(() => {
      const bar = document.getElementById('dictationBar');
      return !!bar && !bar.hidden;
    }, { timeout: 8000 });

    // Metnin VE noktalama komutunun işlenmesini bekle (ara sonuç ile final
    // sonuç ayrı olaylarda gelir; yalnız uzunluk artışını beklemek yarış
    // koşulu yaratıyordu).
    await page.waitForFunction(
      () => {
        const ed = window.__contentEditor;
        if (!ed) return false;
        const t = ed.getText();
        return t.includes('Belediye yeni projeyi duyurdu') && t.includes('duyurdu.');
      },
      { timeout: 8000 },
    );

    const after = await page.evaluate(() => ({
      text: window.__contentEditor.getText(),
      detail: (document.querySelector('[name="detay"]') || {}).value || '',
      active: (document.querySelector('.be-toolbar [data-cmd="dictate"]') || {}).classList?.contains('is-active'),
    }));

    assert.ok(after.text.includes('Belediye yeni projeyi duyurdu'),
      'konuşma metni editöre yazılmalı: ' + after.text.slice(-120));
    assert.ok(after.text.includes('duyurdu.'),
      '“nokta” komutu noktaya dönüşmeli: ' + after.text.slice(-120));
    assert.ok(!after.text.includes('duyurdu .'),
      'noktalamadan önce boşluk kalmamalı: ' + JSON.stringify(after.text.slice(-160)));
    assert.equal(after.active, true, 'dikte sürerken düğme aktif görünmeli');

    // Gizli alan editörü yansıtır (form gönderimi için); "otomatik kaydetme"
    // ise sunucuya POST ile olur ve aşağıda fetch ile doğrulanır.

    // Durdur (mikrofon açık kalmamalı) — fetch'ten ÖNCE.
    await page.evaluate(() => {
      const stop = document.getElementById('dictationStop');
      if (stop) stop.click();
    });
    await page.waitForFunction(() => {
      const bar = document.getElementById('dictationBar');
      return !bar || bar.hidden;
    }, { timeout: 5000 });

    // Sunucuda kayıt OLMAMALI. Sayfa yenilemek yerine içeriği fetch ile
    // sunucudan oku (navigasyon, collab/beforeunload yüzünden takılabiliyor).
    const serverHasDictation = await page.evaluate(async (url) => {
      const r = await fetch(url, { credentials: 'same-origin' });
      const html = await r.text();
      return html.includes('Belediye yeni projeyi duyurdu');
    }, '/admin/icerikler/' + id);
    assert.equal(serverHasDictation, false,
      'dikte sunucudaki içeriği kendiliğinden değiştirmemeli (kaydedilmedi)');
  });
});

test('dikte: Escape çubuğu kapatır', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');

  await withAdminPage(async (page) => {
    await setViewport(page, 1440, 950);
    const id = await firstContentID(page);
    if (!id) return t.skip('düzenlenecek içerik yok');
    await goto(page, '/admin/icerikler/' + id);
    await waitEditor(page);

    await page.evaluate(() => document.querySelector('.be-toolbar [data-cmd="dictate"]').click());
    await page.waitForFunction(() => {
      const bar = document.getElementById('dictationBar');
      return !!bar && !bar.hidden;
    }, { timeout: 8000 }).catch(() => {});

    const barBefore = await page.evaluate(() => {
      const bar = document.getElementById('dictationBar');
      return !!bar && !bar.hidden;
    });
    if (!barBefore) return; // tarayıcı hiç açmadıysa (düğme yok) atla

    await page.keyboard.press('Escape');
    await page.waitForFunction(() => {
      const bar = document.getElementById('dictationBar');
      return !bar || bar.hidden;
    }, { timeout: 5000 });
    const hidden = await page.evaluate(() => {
      const bar = document.getElementById('dictationBar');
      return !bar || bar.hidden;
    });
    assert.ok(hidden, 'Escape dikte çubuğunu kapatmalı');
  });
});
