const puppeteer = require('puppeteer');
const BASE = process.argv[2], SID = process.argv[3];

// Kabuk korumalı gezinme testi.
// Ölçüt: belge YENİDEN YÜKLENMEMELİ. Kanıt: bir DOM düğümüne JS özelliği
// koyup sonrasında okuyoruz — tam yüklemede kaybolur.
const isaretle = (p) => p.evaluate(() => {
  document.querySelector('.admin-side').__tbIsaret = 'ayni-belge';
  window.__tbSayac = (window.__tbSayac || 0) + 1;
  return window.__tbSayac;
});
const oku = (p) => p.evaluate(() => ({
  isaret: (document.querySelector('.admin-side') || {}).__tbIsaret || null,
  sayac: window.__tbSayac,
  yol: location.pathname,
  baslik: document.title,
  h1: (document.querySelector('.admin-top h1') || {}).textContent,
  breadcrumb: ((document.getElementById('adminCrumb') || {}).textContent || '').trim(),
  sekmeVar: !!document.getElementById('wsTabs'),
  aktifSekme: ((document.querySelector('.ws-links .on') || {}).textContent || '').trim(),
  hap: !!document.querySelector('.ws-links .tb-hap'),
  tabloArac: document.querySelectorAll('.tbl-tools').length,
  satir: document.querySelectorAll('.table tbody tr').length,
  kart: document.querySelectorAll('.card').length,
  tasma: document.documentElement.scrollWidth > document.documentElement.clientWidth + 1,
}));

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 1000 });
  const hata = [];
  p.on('pageerror', (e) => hata.push(String(e.message)));
  p.on('console', (m) => { if (m.type() === 'error') hata.push('console: ' + m.text()); });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  await p.goto(BASE + '/admin/icerikler', { waitUntil: 'networkidle2', timeout: 60000 });
  await new Promise((r) => setTimeout(r, 800));
  await isaretle(p);
  console.log('başlangıç :', JSON.stringify(await oku(p)));

  // --- 1) Yan menü: çalışma alanı değişimi ---
  await p.click('.side-scroll a[href="/admin/dosyalar"]');
  await new Promise((r) => setTimeout(r, 1300));
  const a1 = await oku(p);
  console.log('\n[1] yan menü: /admin/icerikler → /admin/dosyalar');
  console.log('   ', JSON.stringify(a1));
  console.log('    belge aynı :', a1.isaret === 'ayni-belge' ? '✓ AJAX' : '✗ tam yükleme');

  // --- 2) Yan menü geri + sekme ile alt sayfa ---
  await p.click('.side-scroll a[href="/admin/icerikler"]');
  await new Promise((r) => setTimeout(r, 1300));
  const a2 = await oku(p);
  console.log('\n[2] yan menü: → /admin/icerikler');
  console.log('   ', JSON.stringify(a2));
  console.log('    belge aynı :', a2.isaret === 'ayni-belge' ? '✓ AJAX' : '✗ tam yükleme');

  await p.click('.ws-links a[href="/admin/kategoriler"]');
  await new Promise((r) => setTimeout(r, 1400));
  const a3 = await oku(p);
  console.log('\n[3] sekme: İçerikler → Kategoriler');
  console.log('   ', JSON.stringify(a3));
  console.log('    belge aynı :', a3.isaret === 'ayni-belge' ? '✓ AJAX' : '✗ tam yükleme');

  // --- 4) Geri tuşu ---
  await p.goBack();
  await new Promise((r) => setTimeout(r, 1300));
  const a4 = await oku(p);
  console.log('\n[4] geri tuşu');
  console.log('   ', JSON.stringify(a4));
  console.log('    belge aynı :', a4.isaret === 'ayni-belge' ? '✓ AJAX' : '✗ tam yükleme');

  // --- 4b) Sayfa içi HTMX (hx-get sekmeleri) boost'tan etkilenmemeli ---
  await p.goto(BASE + '/admin/icerikler', { waitUntil: 'networkidle2' });
  await new Promise((r) => setTimeout(r, 700));
  await isaretle(p);
  const oncekiSatir = await p.evaluate(() => document.querySelectorAll('.table tbody tr').length);
  const sek = await p.$('.tabs a:not(.active)');
  if (sek) { await sek.click(); await new Promise((r) => setTimeout(r, 1500)); }
  const a5 = await p.evaluate(() => ({
    isaret: (document.querySelector('.admin-side') || {}).__tbIsaret || null,
    url: location.pathname + location.search,
    panel: !!document.querySelector('#icerik-panel'),
    satir: document.querySelectorAll('.table tbody tr').length,
    arac: document.querySelectorAll('.tbl-tools').length,
  }));
  console.log('\n[4b] sayfa içi HTMX sekmesi (hx-get, boost değil)');
  console.log('    ', JSON.stringify(a5), '· önceki satır', oncekiSatir);
  console.log('     parça geldi :', a5.panel && a5.satir !== oncekiSatir ? '✓' : '(satır aynı olabilir)');
  console.log('     belge aynı  :', a5.isaret === 'ayni-belge' ? '✓' : '✗');

  // --- 5) Ağır sayfa boost dışı mı ---
  await p.goto(BASE + '/admin/icerikler', { waitUntil: 'networkidle2' });
  await new Promise((r) => setTimeout(r, 600));
  const agir = await p.evaluate(() => {
    const a = document.querySelector('.ws-links a[href="/admin/icerikler/yeni"]');
    return a ? a.getAttribute('hx-boost') : 'link yok';
  });
  await p.goto(BASE + '/admin/icerikler/yeni', { waitUntil: 'networkidle2' });
  const agirYuk = await p.evaluate(() => ({
    editor: !!document.querySelector('.ProseMirror, .block-editor'),
    form: !!document.getElementById('content-form-panel'),
  }));
  console.log('\n[5] ağır sayfa (içerik formu)');
  console.log('    hx-boost     :', agir, '(false olmalı)');
  console.log('    editör yüklü :', agirYuk.editor, '· form:', agirYuk.form);

  console.log('\nJS hatası:', hata.length ? [...new Set(hata)].slice(0, 4).join(' | ') : 'yok');
  await b.close();
})();
