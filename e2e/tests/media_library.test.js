'use strict';
/**
 * Medya kütüphanesi: thumbnail, metadata, kategori ve yıl/ay gruplama.
 * Admin oturumu gerektirir; TB_SESSION yoksa atlanır.
 */

const test = require('node:test');
const assert = require('node:assert/strict');
const { goto, setViewport, withAdminPage } = require('../lib/harness');

const SID = process.env.TB_SESSION;

test('medya kütüphanesi: çoklu dosya yükleme', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdminPage(async (page) => {
    let ids = [];
    try {
      await goto(page, '/admin/dosyalar');
      ids = await page.evaluate(async () => {
        const fd = new FormData();
        for (const name of ['e2e-coklu-1.png', 'e2e-coklu-2.png']) {
          const canvas = document.createElement('canvas');
          canvas.width = 40;
          canvas.height = 30;
          const ctx = canvas.getContext('2d');
          ctx.fillStyle = name.endsWith('1.png') ? '#059669' : '#7c3aed';
          ctx.fillRect(0, 0, 40, 30);
          const blob = await new Promise((resolve) => canvas.toBlob(resolve, 'image/png'));
          fd.append('dosya', new File([blob], name, { type: 'image/png' }));
        }
        fd.append('kategori', 'E2E Çoklu');
        const res = await fetch('/admin/dosyalar?json=1', { method: 'POST', body: fd, credentials: 'same-origin' });
        const data = await res.json().catch(() => null);
        if (!res.ok || !data || !data.ok || !Array.isArray(data.items)) {
          throw new Error((data && data.error) || 'çoklu yükleme başarısız');
        }
        return data.items.map((item) => item.id);
      });
      assert.equal(ids.length, 2, 'iki dosya birlikte yüklenmeli');
      const thumbs = await page.evaluate(async (list) => {
        const out = [];
        for (const id of list) {
          const res = await fetch('/media/thumb/' + id);
          out.push({ id, status: res.status, type: res.headers.get('content-type') || '' });
        }
        return out;
      }, ids);
      thumbs.forEach((item) => {
        assert.equal(item.status, 200, 'çoklu yükleme thumb ' + item.id);
        assert.match(item.type, /image\/jpeg/);
      });
    } finally {
      if (ids.length) {
        await page.evaluate(async (list) => {
          const token = document.querySelector('input[name=_csrf]')?.value || '';
          for (const id of list) {
            await fetch('/admin/dosyalar/' + id + '/sil', {
              method: 'POST', credentials: 'same-origin',
              headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
              body: '_csrf=' + encodeURIComponent(token),
            }).catch(() => {});
          }
        }, ids).catch(() => {});
      }
    }
  });
});

test('medya kütüphanesi: thumbnail, metadata ve kategori akışı', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdminPage(async (page, errors) => {
    let mediaID = 0;
    const title = 'E2E medya ' + Date.now();
    const category = 'E2E Kategori';
    try {
      await setViewport(page, 1440, 950);
      await goto(page, '/admin/dosyalar');
      assert.ok(!page.url().includes('/admin/giris'), 'medya sayfası açık: ' + page.url());
      // Faz 187: sayfalama KALDIRILDI, yerine sonsuz kaydırma geldi.
      // Gözcü (#medyaSon) görünür olunca sonraki dilimi kendisi ister.
      // Eski `.media-pager` / `a[href*="sayfa=2"]` bu yüzden hiç yok.
      const pager = await page.evaluate(() => {
        const son = document.getElementById('medyaSon');
        return son
          ? {
              var: true,
              hxGet: !!son.getAttribute('hx-get'),
              tetik: son.getAttribute('hx-trigger') || '',
              takas: son.getAttribute('hx-swap') || '',
            }
          : null;
      });
      assert.ok(pager, 'medya kütüphanesi sonsuz kaydırma gözcüsü olmalı (#medyaSon)');
      assert.ok(pager.hxGet && /revealed/.test(pager.tetik),
        'gözcü görününce sonraki dilimi istemeli: ' + JSON.stringify(pager));
      const csv = await page.evaluate(async () => {
        const res = await fetch('/admin/dosyalar/disa-aktar.csv');
        const text = await res.text();
        return { status: res.status, type: res.headers.get('content-type') || '', head: text.slice(0, 40) };
      });
      assert.equal(csv.status, 200, 'CSV dışa aktarma 200 dönmeli');
      assert.match(csv.type, /text\/csv/);
      assert.match(csv.head, /id,baslik,dosya/);

      const boxes = await page.$$('#medyaTopluForm .medyaSec');
      assert.ok(boxes.length >= 3, 'seçim kutuları olmalı');
      await boxes[0].click();
      await page.keyboard.down('Shift');
      await boxes[2].click();
      await page.keyboard.up('Shift');
      const rangeSelection = await page.evaluate(() => ({
        checked: document.querySelectorAll('#medyaTopluForm .medyaSec:checked').length,
        selected: document.querySelectorAll('#medyaTopluForm .media-item.selected').length,
        count: (document.getElementById('medyaSayi') || {}).textContent || '',
      }));
      assert.equal(rangeSelection.checked, 3, 'Shift aralık seçimi üç kaydı seçmeli');
      assert.equal(rangeSelection.selected, 3, 'seçili kartlar işaretlenmeli');
      assert.match(rangeSelection.count, /3 medya seçili/, 'sayaç güncellenmeli');
      await page.keyboard.down('Control');
      await boxes[1].click();
      await page.keyboard.up('Control');
      const toggled = await page.evaluate(() => document.querySelectorAll('#medyaTopluForm .medyaSec:checked').length);
      assert.equal(toggled, 2, 'Ctrl/Cmd tıklaması tek kaydı çıkarmalı');
      await page.click('#medyaSecimTemizle');
      const cleared = await page.evaluate(() => ({
        checked: document.querySelectorAll('#medyaTopluForm .medyaSec:checked').length,
        hidden: document.getElementById('medyaBar').hidden,
      }));
      assert.equal(cleared.checked, 0, 'seçim temizlenmeli');
      assert.equal(cleared.hidden, true, 'seçim yokken toplu çubuk gizlenmeli');
      await page.keyboard.down('Shift');
      await page.click('#medyaTopluForm .media-item .media-open');
      await page.keyboard.up('Shift');
      const cardSelect = await page.evaluate(() => ({
        checked: document.querySelectorAll('#medyaTopluForm .medyaSec:checked').length,
        modal: !document.getElementById('mediaEditModal').hidden,
      }));
      assert.equal(cardSelect.checked, 1, 'Shift + kart tıklaması seçim yapmalı');
      assert.equal(cardSelect.modal, false, 'modifier ile kart tıklaması modal açmamalı');
      await page.click('#medyaSecimTemizle');
      await page.click('[data-media-view="list"]');
      const listMode = await page.evaluate(() => document.getElementById('medya-panel').classList.contains('media-list-view'));
      assert.equal(listMode, true, 'liste görünümü uygulanmalı');
      await page.click('[data-media-view="grid"]');
      // Sürükle-bırak: yeni tasarımda bırakma anında yüklenir (önizleme
      // adımı yok). Bu yüzden yalnız bırakma alanının ve dosya girdisinin
      // doğru bağlandığını doğruluyoruz — veri yazmadan.
      const dropAlan = await page.evaluate(() => {
        const alan = document.querySelector('.mgal-drop[data-drop]');
        const girdi = alan ? alan.querySelector('[data-mg-file]') : null;
        return {
          alan: !!alan,
          girdi: !!girdi,
          coklu: girdi ? girdi.multiple : false,
          kabul: girdi ? (girdi.getAttribute('accept') || '') : '',
        };
      });
      assert.equal(dropAlan.alan, true, 'sürükle-bırak alanı (.mgal-drop[data-drop]) olmalı');
      assert.equal(dropAlan.girdi, true, 'alan içinde dosya girdisi ([data-mg-file]) olmalı');
      assert.equal(dropAlan.coklu, true, 'girdi çoklu dosya kabul etmeli');
      assert.match(dropAlan.kabul, /image\//, 'girdi görsel türlerini kabul etmeli');

      const created = await page.evaluate(async ({ title, category }) => {
        const canvas = document.createElement('canvas');
        canvas.width = 80;
        canvas.height = 60;
        const ctx = canvas.getContext('2d');
        ctx.fillStyle = '#1d4ed8';
        ctx.fillRect(0, 0, 80, 60);
        const blob = await new Promise((resolve) => canvas.toBlob(resolve, 'image/png'));
        const fd = new FormData();
        fd.append('dosya', new File([blob], 'e2e-medya-kutuphane.png', { type: 'image/png' }));
        fd.append('baslik', title);
        fd.append('kategori', category);
        fd.append('aciklama', 'E2E açıklama');
        const res = await fetch('/admin/dosyalar?json=1', {
          method: 'POST', body: fd, credentials: 'same-origin',
        });
        const data = await res.json().catch(() => null);
        if (!res.ok || !data || !data.ok) throw new Error((data && data.error) || 'yükleme başarısız');
        return { id: data.id, url: data.url, thumb: data.thumb };
      }, { title, category });
      mediaID = created.id;
      assert.ok(mediaID > 0, 'test medyası yüklenmeli');

      const beforeTransform = await page.evaluate(async ({ url }) => {
        const img = new Image();
        img.src = url + '?v=' + Date.now();
        await img.decode();
        return { w: img.naturalWidth, h: img.naturalHeight };
      }, created);
      assert.deepEqual(beforeTransform, { w: 80, h: 60 }, 'yüklenen görsel boyutu 80×60 olmalı');

      const transform = await page.evaluate(async (id) => {
        const fd = new FormData();
        fd.append('islem', 'rotate-right');
        const res = await fetch('/admin/dosyalar/' + id + '/duzenle', {
          method: 'POST', body: fd, credentials: 'same-origin',
        });
        const text = await res.text();
        return { ok: res.ok, status: res.status, flash: (text.match(/class="flash[^"]*"[^>]*>([^<]+)/) || [])[1] || '' };
      }, mediaID);
      assert.ok(transform.ok, 'görsel döndürme ucu çalışmalı');

      const afterTransform = await page.evaluate(async ({ url }) => {
        const img = new Image();
        img.src = url + '?v=' + Date.now();
        await img.decode();
        return { w: img.naturalWidth, h: img.naturalHeight };
      }, created);
      assert.deepEqual(afterTransform, { w: 60, h: 80 }, '90° döndürmede boyutlar yer değiştirmeli');

      const thumb = await page.evaluate(async (id) => {
        const res = await fetch('/media/thumb/' + id);
        return { status: res.status, type: res.headers.get('content-type') || '' };
      }, mediaID);
      assert.equal(thumb.status, 200, 'thumbnail ucu 200 dönmeli');
      assert.match(thumb.type, /image\/jpeg/, 'thumbnail JPEG olmalı');

      await goto(page, '/admin/dosyalar?q=' + encodeURIComponent(title));
      const card = await page.evaluate((id) => {
        const item = document.querySelector('.media-item[data-media-id="' + id + '"]');
        if (!item) return null;
        return {
          title: item.dataset.title,
          category: item.dataset.category,
          thumb: item.querySelector('img') ? item.querySelector('img').getAttribute('src') : '',
          grouped: !!item.closest('.media-group'),
        };
      }, mediaID);
      assert.ok(card, 'yüklenen medya kütüphanede görünmeli');
      assert.equal(card.title, title, 'başlık kartta görünmeli');
      assert.equal(card.category, category, 'kategori kartta görünmeli');
      assert.ok(card.thumb.includes('/media/thumb/'), 'kart thumbnail kullanmalı');
      assert.ok(card.grouped, 'medya yıl/ay grubunda olmalı');

      await page.click(`.media-item[data-media-id="${mediaID}"] [data-media-open]`);
      await page.waitForSelector('#mediaEditModal:not([hidden])', { timeout: 4000 });
      const modal = await page.evaluate(() => ({
        title: document.getElementById('mediaEditName').value,
        alt: document.getElementById('mediaEditAlt').value,
        category: document.getElementById('mediaEditCategory').value,
        preview: document.getElementById('mediaEditPreview').src,
      }));
      assert.equal(modal.title, title, 'modal başlığı dolmalı');
      assert.equal(modal.category, category, 'modal kategorisi dolmalı');
      assert.ok(modal.preview.includes('/media/'), 'modal büyük önizleme göstermeli');

      await page.$eval('#mediaEditAlt', (el) => { el.value = 'E2E alt metin'; });
      await page.$eval('#mediaEditCaption', (el) => { el.value = 'E2E fotoğraf altı'; });
      await page.$eval('#mediaEditDescription', (el) => { el.value = 'E2E açıklama güncellendi'; });
      await page.$eval('#mediaEditSEOTitle', (el) => { el.value = 'E2E SEO başlığı'; });
      await page.$eval('#mediaEditSEODesc', (el) => { el.value = 'E2E SEO açıklaması'; });
      await page.click('#mediaEditForm button[type="submit"]');
      await page.waitForFunction(() => {
        const flash = document.querySelector('#medya-panel .flash');
        return !!flash && flash.textContent.includes('güncellendi');
      }, { timeout: 8000 });

      await goto(page, '/admin/dosyalar?kategori=' + encodeURIComponent(category));
      const updated = await page.evaluate((id) => {
        const item = document.querySelector('.media-item[data-media-id="' + id + '"]');
        return item ? {
          alt: item.dataset.alt,
          caption: item.dataset.caption,
          description: item.dataset.description,
          seoTitle: item.dataset.seoTitle,
          seoDescription: item.dataset.seoDescription,
        } : null;
      }, mediaID);
      assert.ok(updated, 'kategori filtresinde medya bulunmalı');
      assert.equal(updated.alt, 'E2E alt metin');
      assert.equal(updated.caption, 'E2E fotoğraf altı');
      assert.equal(updated.description, 'E2E açıklama güncellendi');
      assert.equal(updated.seoTitle, 'E2E SEO başlığı');
      assert.equal(updated.seoDescription, 'E2E SEO açıklaması');

      assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
    } finally {
      if (mediaID > 0) {
        await page.evaluate(async (id) => {
          const token = document.querySelector('input[name=_csrf]')?.value || '';
          await fetch('/admin/dosyalar/' + id + '/sil', {
            method: 'POST', credentials: 'same-origin',
            headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
            body: '_csrf=' + encodeURIComponent(token),
          }).catch(() => {});
        }, mediaID).catch(() => {});
      }
    }
  });
});
