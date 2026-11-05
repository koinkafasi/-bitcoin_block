'use strict';
/**
 * İçerik editörü: TipTap tabanlı blok editörünün yüklenmesi ve temel
 * biçimlendirme araçlarının çalışması. Test kendi geçici taslağını oluşturur
 * ve sonunda siler; gerçek bir haberi değiştirmez.
 */
const test = require('node:test');
const assert = require('node:assert/strict');
const { goto, setViewport, withAdminPage, BASE_URL } = require('../lib/harness');

const SID = process.env.TB_SESSION;

/**
 * Editörü odaklar ve odak GERÇEKTEN editöre düşene kadar bekler.
 *
 * Neden: tiptap'ın `commands.focus()` komutu bu ortamda DOM odağını
 * `requestAnimationFrame` içinde alır (senkron `dom.focus()` yalnızca
 * Safari/Android dallarında çalışır). Hemen ardından yazmak tuşu kaybettirir
 * (keydown body'ye gider). İnsan kullanımında fark edilmez; testte deterministik
 * olması için gerçek odak beklenir.
 */
async function focusEditor(page) {
  await page.evaluate(() => {
    if (window.__contentEditor) window.__contentEditor.commands.focus();
  });
  await page.waitForFunction(
    () => {
      const el = document.activeElement;
      return !!el && !!el.closest('#blockEditor');
    },
    { timeout: 4000 }
  );
}

async function waitEditor(page) {
  await page.waitForFunction(
    () => window.__contentEditor && document.querySelector('#blockEditor .ProseMirror'),
    { timeout: 15000 },
  );
  await page.waitForFunction(
    () => window.__collabActive !== true || window.__collabSynced === true,
    { timeout: 15000 },
  ).catch(() => {});
}

test('içerik editörü TipTap olarak yüklenir', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');

  await withAdminPage(async (page, errors) => {
    let contentID = '';

    try {
      await setViewport(page, 1440, 950);
      await page.goto(BASE_URL + '/admin/icerikler/yeni', { waitUntil: 'networkidle2' });
      await waitEditor(page);

      const title = 'Editör testi ' + Date.now();
      await page.$eval('input[name="baslik"]', (el, value) => {
        el.value = value;
        el.dispatchEvent(new Event('input', { bubbles: true }));
      }, title);

      await page.evaluate(() => {
        const btn = document.querySelector('button[name="kaydet"][value="taslak"]');
        if (btn) btn.click();
      });
      await page.waitForFunction(() => {
        const f = document.querySelector('#content-form-panel form[action]');
        return !!f && /\/admin\/icerikler\/\d+$/.test(f.getAttribute('action') || '');
      }, { timeout: 20000 });
      contentID = await page.evaluate(() => {
        const f = document.querySelector('#content-form-panel form[action]');
        const m = (f && f.getAttribute('action') || '').match(/\/admin\/icerikler\/(\d+)$/);
        return m ? m[1] : '';
      });
      assert.ok(contentID, 'geçici içerik oluşturulmalı');

      await page.goto(BASE_URL + '/admin/icerikler/' + contentID, { waitUntil: 'networkidle2' });
      await waitEditor(page);

      const info = await page.evaluate(() => {
        const hidden = document.querySelector('[name="detay"]');
        let doc = null;
        try { doc = JSON.parse(hidden.value); } catch (_) {}
        return {
          editor: !!window.__contentEditor,
          toolbarButtons: document.querySelectorAll('.be-toolbar [data-cmd]').length,
          hiddenIsDoc: !!doc && doc.type === 'doc',
          status: (document.getElementById('editorStatus') || {}).textContent || '',
        };
      });
      assert.ok(info.editor, 'TipTap editörü başlatılmalı');
      assert.ok(info.toolbarButtons >= 10, 'araç çubuğu düğmeleri olmalı: ' + info.toolbarButtons);
      assert.ok(info.hiddenIsDoc, 'gizli alan TipTap dokümanı taşımalı');
      assert.ok(info.status.includes('kelime'), 'kelime sayacı görünmeli');

      await page.evaluate(() => document.querySelector('.be-toolbar [data-cmd="image"]').click());
      await page.waitForSelector('#editorImageDialog:not([hidden])', { timeout: 4000 });
      const picker = await page.evaluate(() => ({
        count: document.querySelectorAll('#editorImageGrid .be-media-choice').length,
        thumb: (document.querySelector('#editorImageGrid .be-media-choice img') || {}).src || '',
      }));
      assert.ok(picker.count > 0, 'görsel seçicide medya önizlemeleri olmalı');
      assert.ok(picker.thumb.includes('/media/thumb/'), 'seçici thumbnail ucu kullanmalı: ' + picker.thumb);
      await page.$eval('#editorImageGrid .be-media-choice', (el) => el.click());
      const previewVisible = await page.evaluate(() => {
        const img = document.getElementById('editorImagePreview');
        return !!img && !img.hidden && !!img.src;
      });
      assert.ok(previewVisible, 'seçilen görselin büyük önizlemesi görünmeli');
      await page.click('#editorImageCancel');

      const slotPreview = await page.evaluate(() => {
        const select = document.querySelector('select.media-slot-select');
        if (!select) return null;
        const opt = [...select.options].find((o) => o.value);
        if (!opt) return null;
        select.value = opt.value;
        select.dispatchEvent(new Event('change', { bubbles: true }));
        const img = select.parentElement.querySelector('.media-slot-preview');
        return { thumb: opt.dataset.thumb || '', src: img ? img.src : '', hidden: img ? img.hidden : true };
      });
      assert.ok(slotPreview && slotPreview.thumb.includes('/media/thumb/') && !slotPreview.hidden && slotPreview.src.includes('/media/thumb/'), 'kapak/manşet seçiminde thumbnail önizlemesi görünmeli');

      // H2 komutu editör durumunu değiştirmeli.
      await page.evaluate(() => {
        const btn = document.querySelector('.be-toolbar [data-cmd="h2"]');
        if (btn) btn.click();
        if (window.__contentEditorSync) window.__contentEditorSync();
      });
      const hasHeading = await page.evaluate(() => {
        const hidden = document.querySelector('[name="detay"]');
        return hidden.value.includes('"heading"');
      });
      assert.ok(hasHeading, 'H2 komutu dokümana heading eklemeli');

      await page.click('#blockEditor .ProseMirror');
      await page.keyboard.press('Enter');
      await page.keyboard.press('/');
      await page.waitForSelector('.be-slash', { visible: true, timeout: 4000 });
      const slashCount = await page.$$eval('.be-slash-item', (els) => els.length);
      assert.ok(slashCount >= 8, 'slash menüde bloklar olmalı: ' + slashCount);
      await page.keyboard.press('Escape');

      const blocks = await page.evaluate(() => {
        const ed = window.__contentEditor;
        ed.chain().focus('end').insertTable({ rows: 2, cols: 2, withHeaderRow: true }).run();
        ed.chain().focus('end').insertContent({
          type: 'callout', attrs: { kind: 'warning' },
          content: [{ type: 'paragraph', content: [{ type: 'text', text: 'Uyarı metni' }] }],
        }).run();
        ed.chain().focus('end').insertContent({
          type: 'embed', attrs: { provider: 'youtube', url: 'https://youtu.be/abc12345' },
        }).run();
        ed.chain().focus('end').insertContent({ type: 'gallery', attrs: { items: [] } }).run();
        ed.chain().focus('end').insertContent({
          type: 'toggle', attrs: { title: 'Detay', open: true },
          content: [{ type: 'paragraph', content: [{ type: 'text', text: 'Toggle içeriği' }] }],
        }).run();
        ed.chain().focus('end').insertContent({
          type: 'codeBlock', attrs: { language: 'js' },
          content: [{ type: 'text', text: 'const x = 1' }],
        }).run();
        ed.chain().focus('end').insertContent({
          type: 'bookmark',
          attrs: { url: 'https://example.com/haber', title: 'Yer imi', description: 'Açıklama', site: 'Example', image: '' },
        }).run();
        ed.chain().focus('end').insertContent({
          type: 'mediaCard',
          attrs: { kind: 'file', url: '/media/2026/10/e2e.pdf', title: 'E2E dosya', filename: 'e2e.pdf' },
        }).run();
        ed.chain().focus('end').insertContent({
          type: 'mediaCard',
          attrs: { kind: 'audio', url: '/media/2026/10/e2e.mp3', title: 'E2E ses' },
        }).run();
        ed.chain().focus('end').insertContent({
          type: 'mediaCard',
          attrs: { kind: 'video', url: '/media/2026/10/e2e.mp4', title: 'E2E video' },
        }).run();
        ed.chain().focus('end').insertContent({
          type: 'image',
          attrs: { src: '/media/202610/test.png', alt: 'Test görseli', mediaId: 1, width: 50, layout: 'normal', crop: '3/2', focalX: 20, focalY: 30, overlay: 'E2E üstü' },
        }).run();
        if (window.__contentEditorSync) window.__contentEditorSync();
        const json = document.querySelector('[name="detay"]').value;
        return {
          table: json.includes('"table"'),
          callout: json.includes('"callout"'),
          embed: json.includes('"embed"'),
          gallery: json.includes('"gallery"'),
          toggle: json.includes('"toggle"'),
          bookmark: json.includes('"bookmark"'),
          mediaCard: json.includes('"mediaCard"'),
          imageWidth: json.includes('"width":50'),
          code: json.includes('"codeBlock"') && json.includes('language'),
          domTable: !!document.querySelector('#blockEditor .ProseMirror table'),
          domCallout: !!document.querySelector('#blockEditor .be-callout'),
          domEmbed: !!document.querySelector('#blockEditor .be-embed iframe'),
          domToggle: !!document.querySelector('#blockEditor .be-toggle'),
          domBookmark: !!document.querySelector('#blockEditor .editor-bookmark'),
          domMediaFile: !!document.querySelector('#blockEditor .be-media-file'),
          domMediaAudio: !!document.querySelector('#blockEditor .be-media-audio audio'),
          domMediaVideo: !!document.querySelector('#blockEditor .be-media-video video'),
          domImageResize: !!document.querySelector('#blockEditor .be-image-resize') && document.querySelector('#blockEditor .be-image').style.width === '50%',
          domImageCrop: document.querySelector('#blockEditor .be-image img').style.aspectRatio.replace(/\s/g, '') === '3/2',
          domImageOverlay: (document.querySelector('#blockEditor .be-image-overlay') || {}).textContent === 'E2E üstü',
          domCode: !!document.querySelector('#blockEditor .ProseMirror pre code'),
        };
      });
      assert.ok(blocks.table && blocks.domTable, 'tablo eklenmeli');
      assert.ok(blocks.callout && blocks.domCallout, 'callout eklenmeli');
      assert.ok(blocks.embed && blocks.domEmbed, 'YouTube embed eklenmeli');
      assert.ok(blocks.gallery, 'galeri düğümü eklenmeli');
      assert.ok(blocks.toggle && blocks.domToggle, 'toggle bloğu eklenmeli');
      assert.ok(blocks.bookmark && blocks.domBookmark, 'yer imi kartı eklenmeli');
      assert.ok(blocks.mediaCard && blocks.domMediaFile && blocks.domMediaAudio && blocks.domMediaVideo, 'dosya/ses/video kartları eklenmeli');
      assert.ok(blocks.imageWidth && blocks.domImageResize, 'görsel genişliği ve resize tutamacı olmalı');
      assert.ok(blocks.domImageCrop, 'görsel kırpma oranı uygulanmalı');
      assert.ok(blocks.domImageOverlay, 'görsel üstü metin uygulanmalı');
      assert.ok(blocks.code && blocks.domCode, 'kod bloğu diliyle eklenmeli');

      await page.evaluate(() => document.querySelector('.be-toolbar [data-cmd="filecard"]').click());
      await page.waitForSelector('#editorMediaCardDialog:not([hidden])', { timeout: 4000 });
      await page.type('#editorMediaCardUrl', '/media/2026/10/dialog.pdf');
      await page.type('#editorMediaCardTitle', 'Diyalog dosyası');
      await page.type('#editorMediaCardFilename', 'dialog.pdf');
      await page.click('#editorMediaCardConfirm');
      await page.waitForFunction(() => {
        const json = document.querySelector('[name="detay"]').value;
        return json.includes('Diyalog dosyası') && json.includes('dialog.pdf');
      }, { timeout: 4000 });

      await page.evaluate(() => document.querySelector('.be-toolbar [data-cmd="remoteimage"]').click());
      await page.waitForSelector('#editorRemoteImageDialog:not([hidden])', { timeout: 4000 });
      await page.type('#editorRemoteImageUrl', 'https://images.example.com/e2e-uzak.jpg');
      await page.type('#editorRemoteImageAlt', 'Uzak görsel');
      await page.type('#editorRemoteImageCredit', 'Fotoğraf: E2E');
      await page.click('#editorRemoteImageConfirm');
      await page.waitForFunction(() => {
        const json = document.querySelector('[name="detay"]').value;
        return json.includes('e2e-uzak.jpg') && json.includes('Uzak görsel');
      }, { timeout: 4000 });

      const failedUpload = await page.evaluate(async () => {
        const originalFetch = window.fetch;
        window.fetch = () => Promise.reject(new Error('E2E ağ hatası'));
        try {
          const file = new File([new Uint8Array([137, 80, 78, 71])], 'e2e-hata.png', { type: 'image/png' });
          await window.__editorUploadFiles([file]);
          for (let i = 0; i < 30; i += 1) {
            if (document.querySelector('#blockEditor .be-upload-card.is-error')) break;
            await new Promise((resolve) => setTimeout(resolve, 50));
          }
          const card = document.querySelector('#blockEditor .be-upload-card.is-error');
          return {
            hasCard: !!card,
            text: card ? card.textContent : '',
            cards: document.querySelectorAll('#blockEditor .be-upload-card').length,
            placeholderNodes: window.__contentEditor.getJSON().content.filter((n) => n.type === 'uploadPlaceholder').length,
          };
        } finally {
          window.fetch = originalFetch;
        }
      });
      assert.ok(failedUpload.hasCard && failedUpload.text.includes('e2e-hata.png'), 'başarısız yükleme kart içi hata göstermeli');
      await page.evaluate(() => document.querySelector('#blockEditor .be-upload-card.is-error button')?.click());
      await page.waitForFunction(() => !document.querySelector('#blockEditor .be-upload-card.is-error'), { timeout: 4000 });

      const upload = await page.evaluate(async () => {
        const canvas = document.createElement('canvas');
        canvas.width = 64;
        canvas.height = 40;
        const ctx = canvas.getContext('2d');
        ctx.fillStyle = '#d33';
        ctx.fillRect(0, 0, 64, 40);
        const blob = await new Promise((resolve) => canvas.toBlob(resolve, 'image/png'));
        const fd = new FormData();
        fd.append('dosya', new File([blob], 'e2e-editor-upload.png', { type: 'image/png' }));
        const res = await fetch('/admin/dosyalar?json=1', { method: 'POST', body: fd, credentials: 'same-origin' });
        const data = await res.json().catch(() => null);
        if (data && data.ok) {
          const token = document.querySelector('input[name=_csrf]')?.value || '';
          await fetch('/admin/dosyalar/' + data.id + '/sil', {
            method: 'POST', credentials: 'same-origin',
            headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
            body: '_csrf=' + encodeURIComponent(token),
          }).catch(() => {});
        }
        return data;
      });
      assert.ok(upload && upload.ok && upload.url, 'editör görsel yükleme ucu çalışmalı');

      await page.evaluate(() => {
        const ed = window.__contentEditor;
        ed.chain().focus('end').insertContent({
          type: 'paragraph',
          content: [{ type: 'text', text: 'Öneri tabanı' }],
        }).run();
        ed.commands.setTextSelection(ed.state.doc.content.size - 2);
      });
      await page.evaluate(() => document.querySelector('.be-toolbar [data-cmd="suggestion"]').click());
      await focusEditor(page);
      await page.keyboard.type(' yeni');
      await page.waitForFunction(() => document.querySelectorAll('#blockEditor .suggestion-insert').length > 0, { timeout: 4000 });
      assert.ok(await page.$('#suggestionBar:not([hidden])'), 'öneri çubuğu görünmeli');
      for (let i = 0; i < 5; i += 1) await page.keyboard.press('Backspace');
      await page.waitForFunction(() => document.querySelectorAll('#blockEditor .suggestion-insert').length === 0, { timeout: 4000 });
      await page.keyboard.press('Backspace');
      await page.waitForFunction(() => document.querySelectorAll('#blockEditor .suggestion-delete').length > 0, { timeout: 4000 });
      await page.click('#suggestionList');
      await page.waitForSelector('#editorSuggestionDialog:not([hidden])', { timeout: 4000 });
      const suggestionRows = await page.$$eval('#editorSuggestionList .be-suggestion-row', (rows) => rows.length);
      assert.ok(suggestionRows >= 1, 'öneri listesinde bekleyen değişiklikler görünmeli');
      await page.click('#editorSuggestionClose');
      await page.click('#suggestionAccept');
      await page.waitForFunction(() => document.querySelectorAll('#blockEditor .suggestion-delete').length === 0, { timeout: 4000 });
      await page.evaluate(() => document.querySelector('.be-toolbar [data-cmd="suggestion"]').click());

      await page.evaluate(() => {
        const ed = window.__contentEditor;
        ed.chain().focus('end').insertContent({
          type: 'paragraph',
          content: [{ type: 'text', text: 'ABC' }],
        }).run();
        let range = null;
        ed.state.doc.descendants((node, pos) => {
          if (!range && node.isText && node.text === 'ABC') range = { from: pos, to: pos + 3 };
        });
        if (range) ed.commands.setTextSelection(range);
      });
      await page.evaluate(() => document.querySelector('.be-toolbar [data-cmd="suggestion"]').click());
      await page.waitForFunction(() => document.body.classList.contains('be-suggestion-mode'), { timeout: 4000 });
      await page.evaluate(() => window.__contentEditor.commands.focus());
      await focusEditor(page);
      await page.keyboard.type('Z');
      await page.waitForFunction(() => document.querySelectorAll('#blockEditor .suggestion-insert').length > 0 &&
        document.querySelectorAll('#blockEditor .suggestion-delete').length > 0, { timeout: 4000 });
      await page.click('#suggestionRejectAll');
      await page.waitForFunction(() => document.querySelectorAll('#blockEditor .suggestion-delete').length === 0, { timeout: 4000 });
      const replacedText = await page.evaluate(() => document.querySelector('#blockEditor').textContent);
      assert.ok(replacedText.includes('ABC') && !replacedText.includes('Z'), 'öneri reddi metni eski haline getirmeli');

      await page.evaluate(() => document.querySelector('.be-toolbar [data-cmd="suggestion"]').click());

      await page.evaluate(() => {
        const ed = window.__contentEditor;
        ed.chain().focus('end').insertContent({
          type: 'paragraph',
          content: [{ type: 'text', text: 'Blok seçimi' }],
        }).run();
        document.querySelector('.be-toolbar [data-cmd="blockselect"]').click();
      });
      const blockSelection = await page.evaluate(() => {
        const sel = window.__contentEditor.state.selection;
        return { ranges: sel.ranges ? sel.ranges.length : 0, type: sel.constructor.name };
      });
      assert.ok(blockSelection.ranges >= 1, 'blok aralığı seçilmeli: ' + blockSelection.type);

      await page.evaluate(() => document.querySelector('.be-toolbar [data-cmd="shortcuts"]').click());
      assert.ok(await page.$('#editorShortcuts:not([hidden])'), 'kısayol penceresi açılmalı');
      await page.click('#editorShortcutsClose');

      // İki sekmede ortak düzenleme imleci görünmeli.
      const second = await page.browser().newPage();
      try {
        await second.goto(BASE_URL + '/admin/icerikler/' + contentID, { waitUntil: 'networkidle2' });
        await waitEditor(second);
        await new Promise((r) => setTimeout(r, 1500));
        for (const [i, p] of [page, second].entries()) {
          await p.evaluate((idx) => {
            const ed = window.__contentEditor;
            if (!ed) return;
            ed.commands.focus();
            ed.commands.setTextSelection(Math.min(2 + idx * 4, ed.state.doc.content.size - 1));
          }, i);
        }
        await new Promise((r) => setTimeout(r, 1200));
        // KOŞULLU (Faz 210): uzak imleç yalnız ortak düzenleme (Yjs/Hocuspocus)
        // odası kurulduğunda çizilir. Sunucu COLLAB_URL olmadan açıldıysa
        // editör yerel kipte çalışır — bu bir arıza değil, yapılandırma
        // eksikliğidir. Test o durumda imleç ARAMAZ ama yerel kipin gerçekten
        // kurulduğunu doğrular; aksi hâlde sessiz bir "hiçbir şey çalışmıyor"
        // durumu testten kaçardı.
        const durum = await page.evaluate(() => ({
          collab: !!window.__collabActive,
          cursor: document.querySelectorAll('.ProseMirror-yjs-cursor').length,
          editor: !!window.__contentEditor,
        }));
        assert.equal(durum.editor, true, 'editör kurulmalı');
        if (durum.collab) {
          assert.ok(durum.cursor >= 1, 'uzak düzenleme imleci görünmeli (collab açık)');
        } else {
          // Yerel kip: editör çalışıyor, ortak düzenleme katmanı kapalı.
          assert.equal(durum.cursor, 0, 'collab kapalıyken imleç çizilmemeli');
        }
      } finally {
        await second.close();
      }

      await page.evaluate(() => document.querySelector('.be-toolbar [data-cmd="focus"]').click());
      assert.ok(await page.evaluate(() => document.body.classList.contains('be-focus')), 'odak modu açılmalı');
      await page.keyboard.press('Escape');
      assert.ok(!(await page.evaluate(() => document.body.classList.contains('be-focus'))), 'Esc odak modundan çıkmalı');
      assert.ok(await page.$('.be-toolbar [data-cmd="internallink"]'), 'iç link düğmesi olmalı');
      assert.ok(await page.$('[data-social-preview]'), 'sosyal kart önizlemesi olmalı');

      await page.click('.be-toolbar [data-cmd="comments"]');
      await page.waitForSelector('#editorComments:not([hidden])', { timeout: 5000 });
      await page.type('#editorNoteBody', 'E2E editör notu');
      await page.click('#editorNoteAdd');
      await page.waitForFunction(() => document.body.textContent.includes('E2E editör notu'), { timeout: 5000 });
      page.on('dialog', (d) => { d.accept().catch(() => {}); });
      await page.evaluate(() => {
        const btn = [...document.querySelectorAll('.be-note button')].find((b) => b.textContent === 'Sil');
        if (btn) btn.click();
      });
      await page.waitForFunction(() => !document.body.textContent.includes('E2E editör notu'), { timeout: 5000 });
      await page.click('#editorCommentsClose');

      await page.setViewport({ width: 390, height: 844 });
      const mobileToolbar = await page.evaluate(() => {
        const secondary = document.querySelector('.be-secondary-group');
        const toggle = document.querySelector('.be-more-toggle');
        return {
          secondaryHidden: secondary ? getComputedStyle(secondary).display === 'none' : false,
          toggleVisible: toggle ? getComputedStyle(toggle).display !== 'none' : false,
        };
      });
      assert.ok(mobileToolbar.secondaryHidden && mobileToolbar.toggleVisible, 'mobilde ikincil araçlar menüde olmalı');
      await page.click('.be-more-toggle');
      assert.ok(await page.$('#editorMoreMenu:not([hidden])'), 'mobil daha fazla menüsü açılmalı');
      await page.evaluate(() => document.querySelector('#editorMoreMenu [data-cmd="shortcuts"]').click());
      assert.ok(await page.$('#editorShortcuts:not([hidden])'), 'menüden kısayol penceresi açılmalı');
      await page.click('#editorShortcutsClose');
      await page.setViewport({ width: 1280, height: 900 });

      assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
    } finally {
      if (contentID) {
        await page.evaluate((id) => fetch('/admin/icerikler/' + id + '/sil', {
          method: 'POST', credentials: 'same-origin',
        }).catch(() => {}), contentID).catch(() => {});
      }
    }
  });
});
