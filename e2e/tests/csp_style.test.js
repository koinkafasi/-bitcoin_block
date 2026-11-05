'use strict';
/**
 * CSP `style-src` sertleştirmesi (Faz 211) — kalıcı regresyon kapısı.
 *
 * Politika: `style-src` 'unsafe-inline' İÇERMEZ; sunucunun ürettiği
 * `<style>` blokları sha256 hash'iyle izinlidir. `style="..."` öznitelikleri
 * ayrı direktifle (`style-src-attr`) bilinçli olarak açık kalır.
 *
 * Bu test iki şeyi kilitler:
 *   1. Hiçbir sayfada `style-src-elem` ihlali YOK — yani hiçbir stil bloğu
 *      hash'siz kalmamış. (Hash'siz kalan bir blok sessizce düşer ve panel
 *      görünümü bozulur; bu test onu yakalar.)
 *   2. `style-src` başlığında 'unsafe-inline' yok, `style-src-attr` var.
 *
 * Neden tarayıcı: hash'ler yanıtın kendi baytlarından hesaplanıyor. Yalnız
 * başlığa bakmak "hash var" der ama "doğru hash" demez; ihlal olayı ise
 * tarayıcının gerçek kararını verir.
 */

const test = require('node:test');
const assert = require('node:assert/strict');
const { BASE_URL, withPage, withAdminPage } = require('../lib/harness');

// Genel sayfalar (oturumsuz).
const GENEL = [
  ['ana sayfa', '/'],
  ['kategori', '/kategori/ekonomi'],
  ['arama', '/arama?q=test'],
  ['fikstur', '/fikstur'],
  ['borsa', '/borsa'],
  ['hava', '/hava-durumu'],
];

// Panel sayfaları (oturumlu). İçerik formu TipTap/ProseMirror yükler —
// sabit enjeksiyonun hash'i csp.go'daki vendorStyleHashes'te.
const PANEL = [
  ['pano', '/admin'],
  ['icerikler', '/admin/icerikler'],
  ['icerik formu', '/admin/icerikler/yeni'],
  ['tasarim', '/admin/tasarim'],
  ['ayarlar', '/admin/ayarlar'],
  ['gazete', '/admin/gazete'],
];

/** Sayfada style ile ilgili CSP ihlallerini toplar. */
async function stilIhlalleri(page, yol) {
  await page.goto(BASE_URL + yol, { waitUntil: 'networkidle2', timeout: 60000 });
  // Gecikmeli enjeksiyonlar (editör, htmx) için kısa bekleme.
  await new Promise((r) => setTimeout(r, 700));
  return page.evaluate(() => (window.__cspIhlal || []).filter(
    (i) => String(i.directive || '').startsWith('style')));
}

/** Her sayfa için ihlal dinleyicisi kurar. */
async function dinleyiciKur(page) {
  await page.evaluateOnNewDocument(() => {
    window.__cspIhlal = [];
    document.addEventListener('securitypolicyviolation', (e) => {
      window.__cspIhlal.push({
        directive: e.effectiveDirective || e.violatedDirective,
        blocked: e.blockedURI,
      });
    });
  });
}

test('CSP: genel sayfalarda style ihlali yok', async () => {
  for (const [ad, yol] of GENEL) {
    await withPage(async (page) => {
      await dinleyiciKur(page);
      const ihlal = await stilIhlalleri(page, yol);
      assert.deepEqual(ihlal, [],
        `${ad} (${yol}): stil bloğu hash'siz kalmış — CSP onu engelliyor`);
    });
  }
});

test('CSP: panel sayfalarında style ihlali yok', async (t) => {
  if (!process.env.TB_SESSION) return t.skip('TB_SESSION yok (admin testi)');
  for (const [ad, yol] of PANEL) {
    await withAdminPage(async (page) => {
      await dinleyiciKur(page);
      const ihlal = await stilIhlalleri(page, yol);
      // İçerik formu yoksa (id yok) 404 olabilir; yine de ihlal sayılmamalı.
      assert.deepEqual(ihlal, [],
        `${ad} (${yol}): stil bloğu hash'siz kalmış — CSP onu engelliyor`);
    });
  }
});

test('CSP: style-src sertleştirildi, öznitelikler ayrı direktifte', async () => {
  // Başlığı doğrudan çek (tarayıcı önbelleği devre dışı).
  const r = await fetch(BASE_URL + '/', { headers: { 'Cache-Control': 'no-cache' } });
  assert.equal(r.status, 200);
  const csp = r.headers.get('content-security-policy') || '';
  assert.ok(csp, 'CSP başlığı yayımlanmalı');

  const direktif = (ad) => {
    for (const p of csp.split(';')) {
      const t = p.trim();
      if (t.startsWith(ad)) return t;
    }
    return '';
  };

  const styleSrc = direktif('style-src ');
  assert.ok(styleSrc, 'style-src direktifi olmalı');
  assert.doesNotMatch(styleSrc, /'unsafe-inline'/,
    "style-src 'unsafe-inline' içermemeli (enjekte <style> çalışmasın)");
  assert.match(styleSrc, /'sha256-/,
    "style-src sayfanın <style> hash'ini içermeli");
  assert.match(styleSrc, /'self'/, "style-src 'self' içermeli");

  const attr = direktif('style-src-attr ');
  assert.match(attr, /'unsafe-inline'/,
    "style-src-attr 'unsafe-inline' içermeli (style= öznitelikleri)");
  assert.doesNotMatch(attr, /sha256/,
    'style-src-attr hash içermemeli');

  // script tarafı da kilitli kalsın.
  assert.doesNotMatch(direktif('script-src '), /'unsafe-inline'/);
  assert.doesNotMatch(csp, /'unsafe-eval'/);
});

test('CSP: politika iki istekte bayt-aynı (önbellek güvenli)', async () => {
  const al = async () => {
    const r = await fetch(BASE_URL + '/', { headers: { 'Cache-Control': 'no-cache' } });
    return r.headers.get('content-security-policy') || '';
  };
  const a = await al();
  const b = await al();
  assert.equal(a, b, 'CSP deterministik olmalı; nonce gibi rastgelelik yok');
  assert.doesNotMatch(a, /'nonce-/, 'nonce kullanılmamalı (önbellek)');
});
