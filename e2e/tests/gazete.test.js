'use strict';
/**
 * Gazete modülü (e-gazete):
 *
 * - Site lisansı aç/kapat (ticari paket anahtarı).
 * - Yeni sayı oluşturma + otomatik doldurma (10 sayfa).
 * - Canva benzeri editör: tuval, katmanlar, blok kaydetme.
 * - PDF üretimi + yayına alma.
 * - Arşiv filtresi (periyot/tarih) + flip görüntüleyici.
 * - Abone ekleme.
 *
 * Test kendi verisini üretir ve sonunda siler; lisans durumu GERÇEK
 * değerine döndürülür. 2099 tarihli sayılar gerçek baskılara dokunmaz.
 */

const test = require('node:test');
const assert = require('node:assert/strict');
const { BASE_URL, goto, setViewport, withAdminPage } = require('../lib/harness');

const ISSUE_DATE = '2099-01-01';
const SUB_EMAIL = 'gazete-e2e@example.com';

async function post(page, path, data) {
  return page.evaluate(async ({ path, data }) => {
    const fd = new FormData();
    Object.entries(data).forEach(([k, v]) => fd.append(k, v));
    const r = await fetch(path, { method: 'POST', body: fd, credentials: 'same-origin' });
    return { status: r.status, url: r.url, body: await r.text() };
  }, { path, data });
}

async function deleteSubscriberIfExists(page, email) {
  // Panelden abone satırını bulur ve KVKK silme formunu fetch ile gönderir.
  return page.evaluate(async (email) => {
    const rows = [...document.querySelectorAll('#gazete-panel table tr')];
    const row = rows.find((tr) => tr.textContent.includes(email));
    if (!row) return false;
    const delForm = [...row.querySelectorAll('form')].find((f) => f.action.includes('/sil'));
    if (!delForm) return false;
    const fd = new FormData(delForm);
    await fetch(delForm.action, { method: 'POST', body: fd, credentials: 'same-origin' });
    return true;
  }, email);
}

async function backupLicense(page) {
  // Lisans tablosunda ilk satır (site 1) durumunu okur.
  await goto(page, '/admin/gazete');
  return page.evaluate(() => {
    const row = document.querySelector('#gazete-panel table tbody tr');
    if (!row) return null;
    return row.textContent.includes('Açık');
  });
}

test('gazete: lisans anahtarı, sayı, editör, PDF, arşiv, flip, abone', async () => {
  await withAdminPage(async (page) => {
    // Editör, kaydedilmemiş değişiklik varken sekme kapatmayı onaya bağlar.
    // Test kaydı doğrudan fetch ile yaptığı için editörün `kirli` bayrağı
    // temizlenmez; yeniden yüklemede çıkan beforeunload onayını kabul et.
    // (Veri zaten POST edildi; onay yalnız sayfadan ayrılma iznidir.)
    page.on('dialog', (d) => d.accept().catch(() => {}));

    // ---------- Lisans yedeği ----------
    const lisansOnce = await backupLicense(page);

    // Lisans aç
    let r = await post(page, '/admin/gazete/lisans', { site_id: '1', acik: '1' });
    assert.equal(r.status, 200);
    assert.match(r.body, /Gazete lisansı güncellendi/);

    // ---------- Yeni sayı + otomatik doldurma ----------
    r = await post(page, '/admin/gazete/sayilar', {
      tarih: ISSUE_DATE, periyot: 'gunluk', doldur: '1', baslik: 'E2E Gazete',
    });
    assert.equal(r.status, 200);
    const m = r.url.match(/sayilar\/(\d+)\/duzenle/);
    assert.ok(m, 'yönlendirme editör sayfasına olmalı: ' + r.url);
    const issueId = m[1];

    // ---------- Editör ----------
    await goto(page, '/admin/gazete/sayilar/' + issueId + '/duzenle');
    assert.ok(await page.$('#gz-canvas'), 'tuval yok');
    assert.ok(await page.$('#gz-layers'), 'katman paneli yok');
    assert.ok(await page.$('#gz-pages'), 'sayfa şeridi yok');
    const boot = await page.evaluate(() => {
      return JSON.parse(document.getElementById('gz-boot').textContent);
    });
    assert.equal(Number(boot.issueId), Number(issueId));
    assert.ok(boot.sayfalar.length >= 10, '10 sayfa kurulmalı');
    const dolu = boot.sayfalar.filter((p) => Array.isArray(p.bloklar) && p.bloklar.length > 0);
    assert.ok(dolu.length >= 10, 'otomatik doldurma tüm sayfaları doldurmalı');

    // Blok ekle + seç + özellikler paneli
    //
    // NOT (Faz 210): tuval bloklarının sarmalayıcı sınıfı `gb`; seçim sınıfı
    // `gz-secili` (birincil seçim `gz-birincil`). Test eski `gz-blok` /
    // `secili` adlarını arıyordu ve bu yüzden hiçbir zaman eşleşmiyordu.
    // Sarmalayıcı ile iç gövde sınıflarının çakışmaması bilinçli bir karardır
    // (bkz. gazeteBlockHTML yorumu).
    const oncekiBlok = await page.$$eval('#gz-canvas .gb', (e) => e.length);
    await page.click('.gz-add[data-tip="baslik"]');
    await page.waitForFunction((n) => document.querySelectorAll('#gz-canvas .gb').length > n, { timeout: 5000 }, oncekiBlok);
    assert.ok(await page.$('#gz-canvas .gb.gz-secili'), 'eklenen blok seçili olmalı');
    assert.ok(await page.$('#gz-props-form .gz-num'), 'özellik formu gelmeli');

    // Kaydet (tüm sayfalar) — editörün POST akışı
    const kaydet = await page.evaluate(async (id) => {
      const boot = JSON.parse(document.getElementById('gz-boot').textContent);
      const sayfa = boot.sayfalar[0];
      // Başlık bloğunu ekle (UI tıklaması boot verisini güncellemez; elle POST ederiz)
      const bloklar = JSON.stringify(sayfa.bloklar.concat([{ id: 'e2e-b1', tip: 'baslik', x: 5, y: 5, w: 50, h: 6, baslik: 'E2E BAŞLIK' }]));
      const fd = new FormData();
      fd.append('sayfa', String(sayfa.no));
      fd.append('bloklar', bloklar);
      const r = await fetch('/admin/gazete/sayilar/' + id + '/sayfa', { method: 'POST', body: fd, credentials: 'same-origin' });
      return { status: r.status, body: await r.text() };
    }, issueId);
    assert.equal(kaydet.status, 200);
    assert.match(kaydet.body, /"ok":true/);

    // Kaydedilen blok editörde görünmeli
    await goto(page, '/admin/gazete/sayilar/' + issueId + '/duzenle');
    const boot2 = await page.evaluate(() => JSON.parse(document.getElementById('gz-boot').textContent));
    const ilk = boot2.sayfalar.find((p) => p.no === boot.sayfalar[0].no);
    assert.ok(JSON.stringify(ilk.bloklar).includes('e2e-b1'), 'kayıtlı blok geri gelmeli');

    // ---------- Şablon: kaydet → galeri → JSON ----------
    r = await post(page, '/admin/gazete/sayilar/' + issueId + '/sablona-kaydet', {
      ad: 'E2E Şablon', aciklama: 'e2e', varsayilan: '0',
    });
    assert.match(r.body, /Şablon kaydedildi/);
    await goto(page, '/admin/gazete/sablonlar');
    const galeri = await page.evaluate(() => document.body.textContent);
    assert.ok(galeri.includes('E2E Şablon'), 'şablon galeride görünmeli');
    assert.ok(await page.$('.gz-thumb'), 'mini sayfa önizleme yok');
    const tplJSON = await page.evaluate(async () => {
      const f = [...document.querySelectorAll('form')].find((x) => x.querySelector('input[name=sablon_id]'));
      const id = f ? f.querySelector('input[name=sablon_id]').value : '';
      const resp = await fetch('/admin/gazete/sablonlar/' + id, { credentials: 'same-origin' });
      return { id, status: resp.status, body: await resp.json() };
    });
    assert.equal(tplJSON.status, 200);
    assert.ok(tplJSON.body.sayfalar.length >= 10, 'şablon sayfaları JSON gelmeli');
    const templateId = tplJSON.id;

    // ---------- PDF ----------
    // `zorla=1`: otomatik doldurma, kapak görseli ve uzun başlıklarla
    // dolduğunda taşma geçidi PDF'i bilinçli olarak durdurur (SPEC K3;
    // geçit `gazete_tasma_test.go` ile birim testinde doğrulanıyor).
    // Bu test PDF üretim/indirme hattını sınar; düzeni zorla üretir.
    r = await post(page, '/admin/gazete/sayilar/' + issueId + '/pdf', { zorla: '1' });
    assert.equal(r.status, 200);
    assert.match(r.body, /PDF üretildi/);

    // ---------- Yayına alma ----------
    r = await post(page, '/admin/gazete/sayilar/' + issueId + '/durum', { durum: 'yayinda' });
    assert.match(r.body, /Durum güncellendi/);

    // ---------- Arşiv + filtre ----------
    await page.goto(BASE_URL + '/gazete', { waitUntil: 'domcontentloaded' });
    const arsivText = await page.evaluate(() => document.body.textContent);
    assert.match(arsivText, /Gazete Arşivi/);
    assert.ok(arsivText.includes(ISSUE_DATE), 'yeni sayı arşivde görünmeli');

    // Periyot filtresi
    await page.goto(BASE_URL + '/gazete?periyot=haftalik', { waitUntil: 'domcontentloaded' });
    const filtre = await page.evaluate(() => document.body.textContent);
    assert.ok(!filtre.includes(ISSUE_DATE), 'haftalık filtre günlük sayıyı gizlemeli');

    // ---------- Flip görüntüleyici ----------
    await page.goto(BASE_URL + '/gazete/' + ISSUE_DATE, { waitUntil: 'domcontentloaded' });
    assert.ok(await page.$('#gz-flip'), 'flip konteyneri yok');
    const pages = await page.$$('#gz-flip .gz-flip-page');
    assert.ok(pages.length >= 10, 'flip sayfaları: ' + pages.length);
    const okuyucuText = await page.evaluate(() => document.body.textContent);
    assert.match(okuyucuText, /Çevirerek Oku|Önceki/);

    // PDF indirme
    const pdf = await page.evaluate(async () => {
      const r = await fetch('/gazete/2099-01-01/pdf?periyot=gunluk');
      const b = await r.arrayBuffer();
      const head = new Uint8Array(b.slice(0, 5));
      return { status: r.status, type: r.headers.get('content-type'), magic: String.fromCharCode(...head) };
    });
    assert.equal(pdf.status, 200);
    assert.equal(pdf.type, 'application/pdf');
    assert.equal(pdf.magic, '%PDF-');

    // ---------- Abone ekleme ----------
    // Önceki koşulardan kalan kaydı temizle (idempotent).
    await goto(page, '/admin/gazete');
    await deleteSubscriberIfExists(page, SUB_EMAIL);
    r = await post(page, '/admin/gazete/aboneler', {
      email: SUB_EMAIL, kanal: 'email', siklik: 'gunluk',
    });
    assert.match(r.body, /Abone eklendi/);

    // ---------- Temizlik ----------
    r = await post(page, '/admin/gazete/sayilar/' + issueId + '/sil', {});
    assert.match(r.body, /Sayı silindi/);
    r = await post(page, '/admin/gazete/sablonlar/' + templateId + '/sil', {});
    assert.match(r.body, /Şablon silindi/);
    await goto(page, '/admin/gazete');
    const subDeleted = await deleteSubscriberIfExists(page, SUB_EMAIL);
    assert.ok(subDeleted, 'abone satırı bulunup silinmeli');

    // Lisansı GERÇEK değerine döndür
    if (lisansOnce !== null && lisansOnce === false) {
      await post(page, '/admin/gazete/lisans', { site_id: '1', acik: '0' });
    }
  });
});