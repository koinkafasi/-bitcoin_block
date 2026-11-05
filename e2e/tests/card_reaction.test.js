'use strict';
/**
 * Kart geri bildirimi: "Buna benzer daha fazla / daha az" düğmeleri,
 * seçim durumu, localStorage ve API çağrısı.
 */
const test = require('node:test');
const assert = require('node:assert/strict');
const { goto, setViewport, withAdminPage } = require('../lib/harness');

const SID = process.env.TB_SESSION;

test('kartlarda buna benzer daha fazla / daha az', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdminPage(async (page, errors) => {
    await setViewport(page, 1440, 1000);
    await goto(page, '/kategori/gundem');
    await page.evaluate(() => {
      try { localStorage.removeItem('tb_react_fazla'); localStorage.removeItem('tb_react_az'); } catch (e) {}
    });
    await page.reload({ waitUntil: 'networkidle2' });

    const info = await page.evaluate(() => {
      const bar = document.querySelector('.card-react');
      const btns = [...document.querySelectorAll('.cr-btn')];
      return {
        bars: document.querySelectorAll('.card-react').length,
        fazla: btns.filter((b) => b.getAttribute('data-yon') === 'fazla').length,
        az: btns.filter((b) => b.getAttribute('data-yon') === 'az').length,
        text: bar ? bar.textContent : '',
      };
    });
    assert.ok(info.bars >= 3, 'kartlarda geri bildirim çubuğu olmalı: ' + info.bars);
    assert.ok(info.fazla >= 3 && info.az >= 3, 'iki seçenek de olmalı');
    assert.ok(info.text.includes('Buna benzer daha fazla'), 'açıklamalı metin olmalı');
    assert.ok(info.text.includes('Buna benzer daha az'), 'açıklamalı metin olmalı');
    const commentHref = await page.$eval('.card-comment', (a) => a.getAttribute('href'));
    assert.ok(commentHref && commentHref.endsWith('#yorum-yaz'),
      'kartta yorum formuna bağlantı olmalı: ' + commentHref);

    await page.hover('.card-react .cr-btn[data-yon="fazla"]');
    await new Promise((r) => setTimeout(r, 220));
    const tooltip = await page.evaluate(() => {
      const s = document.querySelector('.card-react .cr-btn[data-yon="fazla"] span');
      return s ? { opacity: getComputedStyle(s).opacity, visibility: getComputedStyle(s).visibility } : null;
    });
    assert.ok(tooltip && tooltip.visibility === 'visible' && parseFloat(tooltip.opacity) > 0.8,
      'açıklama hover’da görünmeli');

    const btn = await page.$('.card-react .cr-btn[data-yon="fazla"]');
    await btn.click();
    await page.waitForFunction(() => {
      const b = document.querySelector('.card-react .cr-btn[data-yon="fazla"]');
      return b && b.classList.contains('on') && b.getAttribute('aria-pressed') === 'true';
    });
    const afterLike = await page.evaluate(() => ({
      fazla: JSON.parse(localStorage.getItem('tb_react_fazla') || '[]'),
      az: JSON.parse(localStorage.getItem('tb_react_az') || '[]'),
      toast: (document.getElementById('crToast') || {}).textContent || '',
    }));
    assert.ok(afterLike.fazla.length >= 1, 'beğeni localStorage’a yazılmalı');
    assert.ok(afterLike.toast.includes('daha fazla'), 'olumlu geri bildirim mesajı olmalı');

    const azBtn = await page.$('.card-react .cr-btn[data-yon="az"]');
    await azBtn.click();
    await page.waitForFunction(() => {
      const b = document.querySelector('.card-react .cr-btn[data-yon="az"]');
      return b && b.classList.contains('on');
    });
    const afterLess = await page.evaluate(() => ({
      fazla: JSON.parse(localStorage.getItem('tb_react_fazla') || '[]'),
      az: JSON.parse(localStorage.getItem('tb_react_az') || '[]'),
    }));
    assert.ok(afterLess.az.length >= 1, 'daha az seçimi kaydedilmeli');
    assert.equal(afterLess.fazla.length, 0, 'aynı kartta iki seçim aynı anda olmamalı');

    // Seçim sayfa yenilendiğinde korunmalı.
    await page.reload({ waitUntil: 'networkidle2' });
    const persisted = await page.evaluate(() => {
      const b = document.querySelector('.card-react .cr-btn[data-yon="az"]');
      return b && b.classList.contains('on') && b.getAttribute('aria-pressed') === 'true';
    });
    assert.ok(persisted, 'daha az seçimi yenilemede korunmalı');

    // Aynı düğmeye tekrar basınca geri alınmalı.
    const azAgain = await page.$('.card-react .cr-btn[data-yon="az"]');
    await azAgain.click();
    await page.waitForFunction(() => {
      const b = document.querySelector('.card-react .cr-btn[data-yon="az"]');
      return b && !b.classList.contains('on');
    });
    const cleared = await page.evaluate(() => JSON.parse(localStorage.getItem('tb_react_az') || '[]'));
    assert.equal(cleared.length, 0, 'tekrar tıklama geri bildirimi kaldırmalı');

    assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
  });
});
