const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

// app.js yolu: form İÇİNDEKİ silme bağlantıları (URL taşıyan data-confirm).
// Onayı reddederek gönderimi engelliyoruz; amaç silmek değil, davranışı
// ölçmek: native onay çıkmalı, POST adresi gerçek silme URL'i olmalı.
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 1000 });
  const hata = [];
  p.on('pageerror', (e) => hata.push(e.message));
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  const istekler = [];
  p.on('request', (r) => { if (r.method() === 'POST') istekler.push(decodeURIComponent(r.url())); });

  // Onay diyaloglarını reddet: hiçbir şey silinmesin.
  const diyaloglar = [];
  p.on('dialog', async (d) => { diyaloglar.push(d.message().slice(0, 60)); await d.dismiss(); });

  await p.goto(BASE + '/admin/icerikler', { waitUntil: 'networkidle2' });
  await new Promise((r) => setTimeout(r, 500));

  // İlk içeriğin düzenleme bağlantısı.
  const link = await p.evaluate(() => {
    const a = [...document.querySelectorAll('a[href^="/admin/icerikler/"]')]
      .find((x) => /^\/admin\/icerikler\/\d+$/.test(x.getAttribute('href')));
    return a ? a.getAttribute('href') : null;
  });
  if (!link) { console.log('düzenleme bağlantısı yok'); await b.close(); return; }
  console.log('içerik:', link);

  await p.goto(BASE + link, { waitUntil: 'networkidle2' });
  await new Promise((r) => setTimeout(r, 700));

  const durum = await p.evaluate(() => {
    const a = document.querySelector('a[data-confirm]');
    if (!a) return null;
    return {
      metin: a.textContent.trim(),
      url: a.getAttribute('data-confirm'),
      formda: !!a.closest('form'),
      hx: a.hasAttribute('hx-post'),
    };
  });
  console.log('bağlantı:', JSON.stringify(durum));
  if (!durum) { console.log('data-confirm bağlantısı yok'); await b.close(); return; }

  await p.click('a[data-confirm]');
  await new Promise((r) => setTimeout(r, 500));

  console.log('native onay çıktı mı:', diyaloglar.length > 0, diyaloglar.length ? JSON.stringify(diyaloglar) : '');
  console.log('POST sayısı (reddettik, 0 olmalı):', istekler.length, istekler.length ? JSON.stringify(istekler) : '');

  // Aynı testi "Kaynağı sil" için de yap (varsa).
  const kaynak = await p.evaluate(() => {
    const a = [...document.querySelectorAll('a[data-confirm]')]
      .find((x) => (x.getAttribute('data-confirm') || '').includes('kaynaklar/sil'));
    return a ? a.getAttribute('data-confirm') : null;
  });
  console.log('kaynak sil bağlantısı:', kaynak || '(bu içerikte yok)');

  console.log('JS hatası:', hata.length ? [...new Set(hata)].join(' | ') : 'yok');
  await b.close();
})();
