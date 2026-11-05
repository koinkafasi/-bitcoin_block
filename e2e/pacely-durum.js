const puppeteer = require('puppeteer');
const BASE = 'https://demo.codeandchill.store/pacely/';

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1440, height: 1000 });
  const hata = [];
  p.on('pageerror', (e) => hata.push(String(e.message)));
  await p.goto(BASE + '#/overview', { waitUntil: 'networkidle2', timeout: 60000 });
  await new Promise((r) => setTimeout(r, 3000));

  const d = await p.evaluate(() => ({
    signinGorunur: (() => {
      const s = document.getElementById('signin');
      return s ? !s.hidden && getComputedStyle(s).display !== 'none' : 'yok';
    })(),
    appGorunur: (() => {
      const a = document.querySelector('.app');
      return a ? getComputedStyle(a).display : 'yok';
    })(),
    tablar: document.querySelectorAll('.tab').length,
    htmlSinif: document.documentElement.className,
    govdeMetin: document.body.innerText.slice(0, 200),
    signinButonlari: [...document.querySelectorAll('#signin button, .signin button')].map((x) => x.textContent.trim().slice(0, 40)).slice(0, 6),
  }));
  console.log(JSON.stringify(d, null, 1));
  console.log('JS hatası:', hata.slice(0, 3));
  await b.close();
})();
