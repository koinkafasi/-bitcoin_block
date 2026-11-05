const puppeteer = require('puppeteer');

// Pacely demo: navigasyon ve ekran envanteri.
const URL = 'https://demo.codeandchill.store/pacely/#/home';

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1600, height: 1000 });
  const istek = [];
  p.on('response', (r) => {
    const u = r.url();
    if (/\.(js|json)(\?|$)/.test(u)) istek.push({ u, s: r.status() });
  });

  await p.goto(URL, { waitUntil: 'networkidle2', timeout: 90000 });
  await new Promise((r) => setTimeout(r, 3000));

  // 1) Hash rotalari: bağlantılar
  const bilgi = await p.evaluate(() => {
    const bag = [...document.querySelectorAll('a[href]')]
      .map((a) => a.getAttribute('href'))
      .filter((h) => h && h.includes('#/'))
      .map((h) => h.replace(/^.*#/, ''));
    const nav = [...document.querySelectorAll('nav a, aside a, .side a')].map((a) => ({
      metin: (a.textContent || '').trim().replace(/\s+/g, ' ').slice(0, 40),
      href: a.getAttribute('href'),
      sinif: a.className,
      rozet: (a.querySelector('.nav__count, .badge') || {}).textContent || '',
    }));
    // üst bar
    const top = [...document.querySelectorAll('header *, .topbar *, .top *')]
      .filter((e) => e.tagName === 'BUTTON' || e.tagName === 'A')
      .map((e) => ({
        etiket: (e.getAttribute('aria-label') || e.title || e.textContent || '').trim().slice(0, 34),
        sinif: (e.className || '').toString().slice(0, 40),
      }));
    return {
      rotalar: [...new Set(bag)].sort(),
      nav,
      ustBar: top.slice(0, 30),
      baslik: document.title,
      h1: (document.querySelector('h1') || {}).textContent,
    };
  });

  console.log('BAŞLIK:', bilgi.baslik, '|', bilgi.h1);
  console.log('\n=== HASH ROTALARI (' + bilgi.rotalar.length + ') ===');
  bilgi.rotalar.forEach((r) => console.log('  ' + r));
  console.log('\n=== YAN MENÜ (' + bilgi.nav.length + ') ===');
  bilgi.nav.forEach((n) => console.log('  ' + (n.metin || '(' + n.href + ')').padEnd(30) + ' ' + (n.href || '') + (n.rozet ? '  [' + n.rozet.trim() + ']' : '')));
  console.log('\n=== ÜST BAR ===');
  bilgi.ustBar.forEach((t) => console.log('  ' + t.etiket.padEnd(28) + t.sinif));

  console.log('\n=== JS/JSON istekleri ===');
  [...new Set(istek.map((i) => i.u))].slice(0, 20).forEach((u) => console.log('  ' + u));

  await p.screenshot({ path: '/tmp/pacely-home.png', fullPage: false });
  console.log('\nscreenshot: /tmp/pacely-home.png');
  await b.close();
})();
