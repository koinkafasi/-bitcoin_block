const puppeteer = require('puppeteer');
const BASE = 'https://demo.codeandchill.store/pacely/';

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1440, height: 1000 });
  await p.goto(BASE + '#/overview', { waitUntil: 'networkidle2', timeout: 60000 });
  await new Promise((r) => setTimeout(r, 2500));

  const d = await p.evaluate(() => ({
    hash: location.hash,
    aktifTab: [...document.querySelectorAll('.tab')].map((t) => ({
      rota: t.getAttribute('data-route'),
      current: t.getAttribute('aria-current'),
      sinif: t.className,
    })).slice(0, 4),
    baslik: document.querySelector('h1') && document.querySelector('h1').textContent,
    h1ler: [...document.querySelectorAll('h1,h2')].slice(0, 5).map((h) => h.textContent.trim().slice(0, 40)),
  }));
  console.log(JSON.stringify(d, null, 1));

  // Rota değiştir, sonra DOM'u incele
  await p.evaluate(() => { location.hash = '#/compose'; });
  await new Promise((r) => setTimeout(r, 1500));
  const d2 = await p.evaluate(() => ({
    hash: location.hash,
    current: [...document.querySelectorAll('[aria-current="page"]')].map((e) => e.getAttribute('data-route')),
    h1ler: [...document.querySelectorAll('h1')].map((h) => h.textContent.trim().slice(0, 50)),
    gorunurView: [...document.querySelectorAll('[data-view]')].filter((v) => !v.hidden).map((v) => v.getAttribute('data-view')),
  }));
  console.log('\ncompose sonrası:', JSON.stringify(d2, null, 1));
  await b.close();
})();
