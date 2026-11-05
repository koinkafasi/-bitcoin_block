const puppeteer = require('puppeteer');
const BASE = 'https://demo.codeandchill.store/pacely/';

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1440, height: 1000 });

  const cdp = await p.createCDPSession();
  await cdp.send('Network.enable');
  let adet = 0, bayt = 0;
  cdp.on('Network.loadingFinished', (e) => { adet++; bayt += e.encodedDataLength || 0; });

  await p.goto(BASE + '#/overview', { waitUntil: 'networkidle2', timeout: 60000 });
  await new Promise((r) => setTimeout(r, 3000));

  // Yan menüdeki gerçek bağlantılar
  const linkler = await p.evaluate(() =>
    [...document.querySelectorAll('.side a[href^="#/"]')].map((a) => ({
      href: a.getAttribute('href'),
      ad: a.textContent.trim().slice(0, 18),
    })).slice(0, 8));
  console.log('yan menü bağlantıları:', linkler.map((l) => l.href).join(' '));

  await p.evaluate(() => { window.__belge = 'ayni'; });
  const hedefler = ['#/schedule/day', '#/team', '#/billing', '#/settings'];
  for (const h of hedefler) {
    adet = 0; bayt = 0;
    const t = Date.now();
    await p.evaluate((x) => { location.hash = x; }, h);
    const t0 = Date.now();
    let degisti = false;
    while (Date.now() - t0 < 4000) {
      degisti = await p.evaluate(() => {
        const h1 = document.querySelector('h1');
        return h1 && h1.textContent.trim() !== 'Good morning, Noor';
      }).catch(() => false);
      if (degisti) break;
      await new Promise((r) => setTimeout(r, 20));
    }
    await new Promise((r) => setTimeout(r, 200));
    const belge = await p.evaluate(() => window.__belge || 'YENİ');
    console.log('%s → istek %d · %s KB · %d ms · belge %s · h1 değişti %s',
      h.padEnd(16), adet, (bayt / 1024).toFixed(1), Date.now() - t,
      belge === 'ayni' ? 'aynı' : 'YENİ', degisti);
    // geri dön
    await p.evaluate(() => { location.hash = '#/overview'; });
    await new Promise((r) => setTimeout(r, 800));
  }
  await b.close();
})();
