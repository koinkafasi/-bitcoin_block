const puppeteer = require('puppeteer');

const BASE = process.argv[2];
const SID = process.argv[3];
const SAYI = process.argv[4] || '78';

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1600, height: 1000 });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });
  const r = await p.goto(BASE + '/admin/gazete/sayilar/' + SAYI + '/duzenle', { waitUntil: 'networkidle2' });
  await new Promise((x) => setTimeout(x, 1200));
  console.log('durum:', r.status());
  console.log('son URL:', p.url());
  const ozet = await p.evaluate(() => ({
    baslik: (document.querySelector('.admin-top h1') || {}).textContent,
    varMi: {
      gzToolbar: !!document.querySelector('.gz-toolbar'),
      gzPalette: !!document.querySelector('.gz-palette'),
      gzCanvas: !!document.querySelector('#gz-canvas'),
      formPanel: !!document.querySelector('#content-form-panel'),
    },
    govdeBas: (document.querySelector('.admin-body') || document.body).innerText.slice(0, 300),
  }));
  console.log(JSON.stringify(ozet, null, 1));
  await b.close();
})();
