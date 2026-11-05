'use strict';
/**
 * Metin Kalitesi paneli — Faz 115/119/121/122.
 *
 * Doğrulananlar:
 *  - Düzenleme ekranında "📐 Metin Kalitesi" kartı var.
 *  - Not (A-E) ve skor (0-100) gösterilir; metrikler basılır.
 *  - "↻ Yeniden ölç" HTMX ile paneli tazeler, sayfa yenilenmez.
 *  - Panel bir AI dedektörü OLMADIĞINI açıkça söyler (yanlış beklenti önlenir).
 *  - Yeni içerik formunda panel yok ("önce kaydedin").
 *  - Panel hiçbir şeyi otomatik kaydetmez.
 *
 * Test salt-okunurdur: yalnız mevcut bir içeriği açar, veri değiştirmez.
 */
const test = require('node:test');
const assert = require('node:assert/strict');
const { goto, setViewport, withAdminPage } = require('../lib/harness');

const SID = process.env.TB_SESSION;

async function firstContentID(page) {
  await goto(page, '/admin/icerikler');
  return page.evaluate(() => {
    const ids = [...document.querySelectorAll('a[href*="/admin/icerikler/"]')]
      .map((el) => el.getAttribute('href') || '')
      .map((h) => (h.match(/^\/admin\/icerikler\/(\d+)$/) || [])[1])
      .filter(Boolean);
    return ids[0] || '';
  });
}

test('kalite paneli: kart, not/skor, metrikler ve yeniden ölçme', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');

  await withAdminPage(async (page, errors) => {
    await setViewport(page, 1440, 950);

    const id = await firstContentID(page);
    if (!id) return t.skip('düzenlenecek içerik yok');
    await goto(page, '/admin/icerikler/' + id);

    await page.waitForSelector('#kalitePanel', { timeout: 15000 });

    const info = await page.evaluate(() => {
      const panel = document.getElementById('kalitePanel');
      if (!panel) return null;
      const grade = panel.querySelector('.kalite-grade');
      const score = panel.querySelector('.kalite-score');
      const metrics = panel.querySelector('.kalite-metrics');
      const refresh = panel.querySelector('[hx-post]');
      return {
        exists: true,
        title: (panel.querySelector('.kalite-title') || {}).textContent || '',
        help: (panel.querySelector('.kalite-help') || {}).textContent || '',
        grade: grade ? grade.textContent.trim() : '',
        gradeClass: grade ? grade.className : '',
        score: score ? score.textContent.trim() : '',
        metricsText: metrics ? metrics.textContent.replace(/\s+/g, ' ').trim() : '',
        refresh: refresh ? refresh.getAttribute('hx-post') : '',
        refreshTarget: refresh ? refresh.getAttribute('hx-target') : '',
      };
    });

    assert.ok(info, 'kalite paneli bulunmalı');
    assert.ok(/Metin Kalitesi/.test(info.title), 'panel başlığı doğru olmalı: ' + info.title);
    assert.ok(/AI dedektörü değildir/i.test(info.help),
      'panel AI dedektörü olmadığını açıkça söylemeli: ' + info.help);

    assert.ok(/^[A-E]$/.test(info.grade), 'not A-E arasında olmalı: ' + info.grade);
    assert.ok(/kalite-grade-[a-e]/.test(info.gradeClass), 'not rozeti sınıfı olmalı');
    assert.ok(/^\d{1,3}\/100$/.test(info.score), 'skor N/100 biçiminde olmalı: ' + info.score);

    const scoreNum = parseInt(info.score, 10);
    assert.ok(scoreNum >= 0 && scoreNum <= 100, 'skor 0-100 aralığında olmalı: ' + scoreNum);

    assert.ok(/sözcük/.test(info.metricsText), 'sözcük sayısı gösterilmeli: ' + info.metricsText);
    assert.ok(/cümle/.test(info.metricsText), 'cümle sayısı gösterilmeli');
    assert.ok(/çeşitlilik/.test(info.metricsText), 'sözcük çeşitliliği gösterilmeli');

    assert.equal(info.refresh, '/admin/icerikler/' + id + '/kalite',
      'yeniden ölçme ucu doğru olmalı');
    assert.equal(info.refreshTarget, '#kalitePanel', 'HTMX hedefi panel olmalı');

    // "Yeniden ölç" → panel tazelenir (sayfa yenilenmez).
    const urlBefore = page.url();
    await page.evaluate(() => {
      const b = document.querySelector('#kalitePanel [hx-post]');
      if (b) b.click();
    });
    await page.waitForFunction(
      () => !!document.querySelector('#kalitePanel .kalite-grade'),
      { timeout: 8000 },
    );
    assert.equal(page.url(), urlBefore, 'yeniden ölçme sayfayı yenilememeli');

    const afterRefresh = await page.evaluate(() => {
      const panel = document.getElementById('kalitePanel');
      return {
        exists: !!panel,
        score: panel ? (panel.querySelector('.kalite-score') || {}).textContent.trim() : '',
      };
    });
    assert.ok(afterRefresh.exists, 'tazeleme sonrası panel korunmalı');
    assert.ok(/\/100$/.test(afterRefresh.score), 'tazeleme sonrası skor gösterilmeli');

    assert.deepEqual(errors, [], 'sayfa hatası olmamalı: ' + errors.join(' | '));
  });
});

test('kalite paneli: yeni içerik formunda yok, kaydetmeyi tetiklemez', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');

  await withAdminPage(async (page) => {
    await setViewport(page, 1440, 950);
    await goto(page, '/admin/icerikler/yeni');

    const yeni = await page.evaluate(() => ({
      panel: !!document.getElementById('kalitePanel'),
      hint: (document.body.innerText || '').includes('önce haberi kaydedin'),
    }));
    assert.equal(yeni.panel, false, 'yeni içerikte kalite paneli olmamalı');
    assert.ok(yeni.hint, 'yeni içerikte "önce kaydedin" açıklaması olmalı');

    // Mevcut içerikte panel, form gönderimini tetiklemez.
    const id = await firstContentID(page);
    if (!id) return;
    await goto(page, '/admin/icerikler/' + id);
    await page.waitForSelector('#kalitePanel', { timeout: 15000 });

    const before = page.url();
    await page.evaluate(() => {
      const panel = document.getElementById('kalitePanel');
      if (panel) panel.scrollIntoView();
    });
    assert.equal(page.url(), before, 'panel görüntülemesi navigasyon yapmamalı');
  });
});
