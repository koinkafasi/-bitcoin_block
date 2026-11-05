const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

// Canli: dil gecisi makalede yol koruyor mu + /en icerik gorunuyor mu?
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'], protocolTimeout: 90000 });
  const p = await b.newPage();
  await p.setViewport({ width: 1440, height: 1000 });
  const hata = [];
  p.on('pageerror', (e) => hata.push(e.message));

  // /en ana sayfa: manset + bolum
  await p.goto(BASE + '/en', { waitUntil: 'networkidle2', timeout: 60000 });
  await new Promise((r) => setTimeout(r, 1200));
  const en = await p.evaluate(() => ({
    lang: document.documentElement.lang,
    manset: document.querySelectorAll('.hero-title, .hero-slide, [class*="hero"]').length,
    enLink: [...document.querySelectorAll('a[href]')].filter((a) => /^\/en\/[a-z0-9-]+-\d+$/.test(a.getAttribute('href') || '')).length,
    yatayTasma: Math.max(0, document.documentElement.scrollWidth - window.innerWidth),
    bosBolum: [...document.querySelectorAll('h2')].filter((h) => {
      const s = h.closest('section');
      return s && s.querySelectorAll('a[href]').length === 0;
    }).length,
  }));
  console.log('/en:', JSON.stringify(en));

  // TR makale ac, dil secici ile EN'e gec -> yol korunmali
  await p.goto(BASE + '/', { waitUntil: 'networkidle2' });
  const makale = await p.evaluate(() => {
    const a = [...document.querySelectorAll('a[href]')].find((x) => /^\/[a-z0-9-]+-\d+$/.test(x.getAttribute('href') || ''));
    return a ? a.getAttribute('href') : null;
  });
  console.log('TR makale:', makale);
  if (makale) {
    await p.goto(BASE + makale, { waitUntil: 'networkidle2' });
    await new Promise((r) => setTimeout(r, 700));
    const varMi = await p.evaluate(() => !!document.querySelector('[data-lang-select]'));
    if (varMi) {
      await p.select('[data-lang-select]', 'en');
      await new Promise((r) => setTimeout(r, 2500));
      const s = await p.evaluate(() => ({
        sonUrl: location.pathname,
        lang: document.documentElement.lang,
        baslik: (document.title || '').slice(0, 70),
      }));
      console.log('gecis sonrasi:', JSON.stringify(s));
      console.log('yol korundu:', s.sonUrl === '/en' + makale || s.sonUrl.startsWith('/en/'));
    }
  }
  console.log('JS hatasi:', hata.length ? [...new Set(hata)].slice(0, 2).join(' | ') : 'yok');
  await b.close();
})();
