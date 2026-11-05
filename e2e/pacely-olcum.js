const puppeteer = require('puppeteer');

// Pacely'de rota geçişinin gerçek maliyeti.
// Ölçülen: ağ isteği sayısı/baytı, içerik değişim süresi, belge yeniden yükleniyor mu.
const BASE = 'https://demo.codeandchill.store/pacely/';
const ROTALAR = ['overview', 'compose', 'calendar', 'inbox'];

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1440, height: 1000 });

  const cdp = await p.createCDPSession();
  await cdp.send('Network.enable');
  let adet = 0, bayt = 0;
  cdp.on('Network.loadingFinished', (e) => { adet++; bayt += e.encodedDataLength || 0; });
  const sifirla = () => { adet = 0; bayt = 0; };

  // İlk yükleme
  let t = Date.now();
  await p.goto(BASE + '#/overview', { waitUntil: 'networkidle2', timeout: 60000 });
  await new Promise((r) => setTimeout(r, 2500));
  console.log('İLK YÜKLEME :', adet, 'istek ·', (bayt / 1024).toFixed(0), 'KB ·', Date.now() - t, 'ms');
  console.log('  (belge + React + chunk\'lar; sonrasında hepsi bellekte)');

  // Belge kimliği işareti
  await p.evaluate(() => { window.__pacelyBelge = 'ayni'; document.body.dataset.test = 'x'; });

  // Rota geçişleri
  for (let i = 0; i < ROTALAR.length - 1; i++) {
    const hedef = ROTALAR[i + 1];
    sifirla();
    t = Date.now();
    await p.evaluate((h) => { location.hash = '#/' + h; }, hedef);
    // İçerik görünene kadar bekle
    const t0 = Date.now();
    while (Date.now() - t0 < 5000) {
      const ok = await p.evaluate((h) => {
        const aktif = document.querySelector('.tab[aria-current="page"]');
        return !!aktif && aktif.getAttribute('data-route') === h;
      }, hedef).catch(() => false);
      if (ok) break;
      await new Promise((r) => setTimeout(r, 20));
    }
    const sure = Date.now() - t;
    const belge = await p.evaluate(() => window.__pacelyBelge || 'YENİ');
    console.log('\n', ROTALAR[i], '→', hedef);
    console.log('   istek', adet, '· bayt', (bayt / 1024).toFixed(1), 'KB · süre', sure, 'ms');
    console.log('   belge aynı :', belge === 'ayni' ? '✓ (SPA)' : '✗ yeniden yükleme');
  }

  // Ön yükleme: fare üzerine gelince chunk isteniyor mu?
  await p.evaluate(() => { location.hash = '#/overview'; });
  await new Promise((r) => setTimeout(r, 1200));
  sifirla();
  const tab = await p.$('.tab[data-route="team"]');
  if (tab) {
    await tab.hover();
    await new Promise((r) => setTimeout(r, 1500));
  }
  console.log('\nFARE ÜZERİNE GELME (team sekmesi)');
  console.log('   istek', adet, '· bayt', (bayt / 1024).toFixed(1), 'KB',
    adet > 0 ? '→ ön yükleme VAR' : '→ ön yükleme yok');

  await b.close();
})();
