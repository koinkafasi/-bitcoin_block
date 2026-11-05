'use strict';
/**
 * AI editörü — onay paneli ve seçili metin menüsü regresyonu.
 *
 * Eski akış AI sonucunu doğrudan alana yazıyordu. Yeni akışta sonuç önce
 * panelde gösterilir; kullanıcı uygulamadan haber alanları değişmez.
 * Sağlayıcı erişilemezse panel açık hata gösterir; sahte sonuç yazılmaz.
 */

const test = require('node:test');
const assert = require('node:assert/strict');
const { goto, setViewport, withAdminPage } = require('../lib/harness');

const SID = process.env.TB_SESSION;

/** İçerik listesinden ilk düzenlenebilir içeriğin URL'sini bulur. */
async function firstContentEditUrl(page) {
  await goto(page, '/admin/icerikler');
  if (page.url().includes('/admin/giris')) return null;
  return page.evaluate(() => {
    const links = [...document.querySelectorAll('a[href^="/admin/icerikler/"]')];
    const hit = links.find((a) => /^\/admin\/icerikler\/\d+$/.test(a.getAttribute('href') || ''));
    return hit ? hit.getAttribute('href') : null;
  });
}

/** Görünür AI başlık düğmesine basar ve panelin açılmasını bekler. */
async function openTitleAI(page) {
  await page.evaluate(() => {
    const btns = [...document.querySelectorAll('button[data-ai="title"][data-target="#f_title"]')];
    const b = btns.find((x) => x.offsetParent !== null) || btns[0];
    b.click();
  });
  await page.waitForFunction(() => {
    const d = document.getElementById('aiEditor');
    return d && !d.hidden;
  }, { timeout: 8000 });
}

test('AI başlık butonu panel açar, haberi sessizce değiştirmez', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdminPage(async (page, errors) => {
    await setViewport(page, 1400, 900);
    const href = await firstContentEditUrl(page);
    assert.ok(href, 'düzenlenecek içerik bulunmalı');
    await goto(page, href);
    assert.ok(!page.url().includes('/admin/giris'), 'içerik formu açık: ' + page.url());

    const before = await page.evaluate(() => {
      const el = document.getElementById('f_title');
      const value = 'Türkçe şğüöçİı başlık denemesi';
      el.value = value;
      el.dispatchEvent(new Event('input', { bubbles: true }));
      return {
        title: el.value,
        detail: (document.querySelector('[name="detay"]') || {}).value || '',
      };
    });

    await openTitleAI(page);
    const ui = await page.evaluate(() => {
      const title = document.getElementById('f_title');
      const detail = document.querySelector('[name="detay"]');
      const body = document.getElementById('aiEditorBody');
      return {
        title: title ? title.value : '',
        detail: detail ? detail.value : '',
        actions: document.querySelectorAll('#aiSelectMenu [data-ai-open]').length,
        panelText: body ? body.innerText : '',
        hasRetry: !!document.querySelector('[data-ai-retry]'),
      };
    });

    assert.equal(ui.title, before.title, 'AI paneli başlığı doğrudan değiştirmemeli');
    assert.equal(ui.detail, before.detail, 'AI paneli gövdeyi doğrudan değiştirmemeli');
    assert.equal(ui.actions, 11, 'menüde 11 eylem olmalı');

    // Sağlayıcı yanıtı gelene kadar bekle; sonuç/err durumunda JS hatası olmamalı.
    await page.waitForFunction(() => {
      const l = document.getElementById('aiEditorLoading');
      const b = document.getElementById('aiEditorBody');
      if (!l || l.hidden) return true;
      return !!(b && b.innerText.trim().length > 0);
    }, { timeout: 30000 }).catch(() => {});
    assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
  });
});

test('metin seçiminde 11 eylemli AI menüsü açılır', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdminPage(async (page, errors) => {
    await setViewport(page, 1400, 900);
    const href = await firstContentEditUrl(page);
    assert.ok(href, 'düzenlenecek içerik bulunmalı');
    await goto(page, href);
    assert.ok(!page.url().includes('/admin/giris'), 'içerik formu açık');

    // Seçim, tiptap editörünün kendi durumunda yapılır: ai_editor.js
    // selectedText() ile textarea'yı değil editör seçimini okur (tiptap
    // geçişinden sonra gizli textarea'da seçim yapmak menüyü açmaz).
    const selected = await page.evaluate(() => {
      const ed = window.__contentEditor;
      if (!ed) return '';
      let range = null;
      ed.state.doc.descendants((node, pos) => {
        if (!range && node.isText && node.text.trim().length >= 3) {
          range = { from: pos, to: pos + Math.min(node.text.length, 20) };
        }
      });
      if (!range) {
        // belge boşsa metin EKLEMEZ; yalnızca son paragrafın sonunu seçer
        const size = ed.state.doc.content.size;
        range = { from: Math.max(0, size - 2), to: Math.max(1, size - 1) };
      }
      ed.commands.setTextSelection(range);
      return ed.state.doc.textBetween(range.from, range.to, '\n');
    });
    assert.ok(selected.trim().length >= 3, 'seçilecek metin olmalı');

    await page.evaluate(() => {
      const pm = document.querySelector('#blockEditor .ProseMirror');
      const r = pm.getBoundingClientRect();
      pm.dispatchEvent(new MouseEvent('mouseup', {
        bubbles: true, clientX: r.left + 30, clientY: r.top + 30,
      }));
    });

    await page.waitForFunction(() => {
      const m = document.getElementById('aiSelectMenu');
      return m && !m.hidden;
    }, { timeout: 4000 });
    const count = await page.$$eval('#aiSelectMenu [data-ai-open]', (els) => els.length);
    assert.equal(count, 11, 'seçim menüsünde 11 eylem olmalı');

    await page.keyboard.press('Escape');
    const hidden = await page.$eval('#aiSelectMenu', (m) => m.hidden);
    assert.equal(hidden, true, 'Escape menüyü kapatmalı');
    assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
  });
});
