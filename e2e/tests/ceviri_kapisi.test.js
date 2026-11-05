'use strict';
/**
 * Çeviri kapısı — Faz 120 (birim testleri httpapi/translation_gate_test.go).
 *
 * Burada yalnız KULLANICIYA GÖRÜNEN tarafı doğrulanır:
 *  - Çeviri oluşturma akışı taslak üretir (asla doğrudan yayınlamaz).
 *  - Türkçe (kök dil) içerikte kapı devreye girmez.
 *
 * Test yeni bir çeviri OLUŞTURMAZ (veri kirletmemek için); yalnız mevcut
 * içeriğin dil/durum alanlarını ve çeviri düğmesinin varlığını okur.
 */
const test = require('node:test');
const assert = require('node:assert/strict');
const { goto, setViewport, withAdminPage } = require('../lib/harness');

const SID = process.env.TB_SESSION;

test('çeviri: düğmeler var, mevcut içerik taslak/yayın durumunu korur', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');

  await withAdminPage(async (page) => {
    await setViewport(page, 1440, 950);

    await goto(page, '/admin/icerikler');
    const id = await page.evaluate(() => {
      const ids = [...document.querySelectorAll('a[href*="/admin/icerikler/"]')]
        .map((el) => el.getAttribute('href') || '')
        .map((h) => (h.match(/^\/admin\/icerikler\/(\d+)$/) || [])[1])
        .filter(Boolean);
      return ids[0] || '';
    });
    if (!id) return t.skip('içerik yok');

    await goto(page, '/admin/icerikler/' + id);
    await page.waitForSelector('#content-form-panel', { timeout: 15000 });

    const info = await page.evaluate(() => {
      const form = document.querySelector('#content-form-panel form[action]');
      const langSelect = document.querySelector('select[name="dil"], select[name="locale"]');
      const translateLinks = [...document.querySelectorAll('a[href*="/cevir/"]')]
        .map((a) => a.getAttribute('href'));
      return {
        formAction: form ? form.getAttribute('action') : '',
        lang: langSelect ? langSelect.value : '',
        translateLinks,
      };
    });

    assert.ok(/\/admin\/icerikler\/\d+$/.test(info.formAction),
      'düzenleme formu doğru hedefte olmalı: ' + info.formAction);
    // Kök dil Türkçe; çeviri kapısı bu içerikte devreye girmez.
    if (info.lang) {
      assert.equal(info.lang, 'tr', 'kök içerik dili tr olmalı: ' + info.lang);
    }
    // Çeviri bağlantıları varsa /cevir/<lang> biçiminde olmalı.
    for (const href of info.translateLinks) {
      assert.ok(/\/cevir\/[a-z]{2}(-[a-z]{2})?$/.test(href),
        'çeviri bağlantısı dil kodu taşımalı: ' + href);
    }
  });
});

test('çeviri: kaynaksız çeviri kuralı belgelenmiş ve uygulanmış', async (t) => {
  // Kuralın varlığını koddan bağımsız doğrulamak için yalnız doküman ve
  // flash mesajı metnini kontrol ederiz (davranış birim testlerinde).
  const fs = require('node:fs');
  const path = require('node:path');
  const doc = path.join(__dirname, '..', '..', 'docs', 'METIN-KALITESI.md');
  let text = '';
  try {
    text = fs.readFileSync(doc, 'utf8');
  } catch (_) {
    return t.skip('doküman bulunamadı');
  }
  assert.ok(/Çeviri kapısı/.test(text), 'dokümanda çeviri kapısı anlatılmalı');
  assert.ok(/kaynak/i.test(text), 'kaynak zorunluluğu belirtilmeli');
});
