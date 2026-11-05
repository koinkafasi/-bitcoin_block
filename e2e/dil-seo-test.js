const puppeteer = require('puppeteer');
const BASE = process.argv[2] || 'http://127.0.0.1:8091';

// Faz 189: dil sayfalarinda canonical/hreflang/robots + dil gecisi.
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'], protocolTimeout: 90000 });
  const p = await b.newPage();
  await p.setViewport({ width: 1440, height: 1000 });
  const hata = [];
  p.on('pageerror', (e) => hata.push(e.message));

  const oku = async (yol) => {
    const r = await p.goto(BASE + yol, { waitUntil: 'networkidle2', timeout: 60000 }).catch(() => null);
    await new Promise((x) => setTimeout(x, 600));
    const d = await p.evaluate(() => ({
      url: location.pathname,
      lang: document.documentElement.lang,
      baslik: (document.title || '').slice(0, 70),
      canonical: (document.querySelector('link[rel=canonical]') || {}).href || '',
      robots: (document.querySelector('meta[name=robots]') || {}).content || '',
      hreflang: [...document.querySelectorAll('link[rel=alternate][hreflang]')].map((l) => l.hreflang + ' → ' + l.href.replace(location.origin, '')),
      manset: document.querySelectorAll('.hero-title, .hero-slide, [class*="hero"]').length,
      yatayTasma: Math.max(0, document.documentElement.scrollWidth - window.innerWidth),
      bosBolum: [...document.querySelectorAll('h2')].filter((h) => {
        const sec = h.closest('section');
        return sec && sec.querySelectorAll('a[href]').length === 0;
      }).map((h) => h.textContent.trim().slice(0, 24)).slice(0, 4),
      // gorunen menu adlari
      menu: [...document.querySelectorAll('nav a, .hp-nav a')].slice(0, 8).map((a) => a.textContent.trim()).filter(Boolean),
    }));
    console.log('\n=== ' + yol + ' (' + (r ? r.status() : '?') + ') ===');
    console.log(JSON.stringify(d, null, 1));
    return d;
  };

  await oku('/');
  await oku('/en');
  await oku('/de');

  // Gercek makale baglantisi bul (TR), sonra /en ile dene
  const makale = await p.evaluate(() => {
    const a = [...document.querySelectorAll('a[href]')].find((x) => /\/[a-z0-9-]+-\d+$/.test(x.getAttribute('href') || ''));
    return a ? a.getAttribute('href') : null;
  });
  console.log('\nmakale baglantisi:', makale);
  if (makale) {
    await oku(makale);
    await oku('/en' + makale);
    // Dil gecisi: makale sayfasindan EN'e gecince yol korunuyor mu?
    await p.goto(BASE + makale, { waitUntil: 'networkidle2' });
    await new Promise((x) => setTimeout(x, 700));
    const varMi = await p.evaluate(() => !!document.querySelector('[data-lang-select]'));
    if (varMi) {
      await p.select('[data-lang-select]', 'en');
      await new Promise((x) => setTimeout(x, 2500));
      console.log('\n=== makaleden EN gecisi ===');
      console.log(JSON.stringify({
        sonUrl: new URL(p.url()).pathname,
        yolKorundu: new URL(p.url()).pathname === '/en' + makale,
        lang: await p.evaluate(() => document.documentElement.lang),
      }, null, 1));
    }
  }

  console.log('\nJS hatasi:', hata.length ? [...new Set(hata)].slice(0, 3).join(' | ') : 'yok');
  await b.close();
})();
