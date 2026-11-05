const puppeteer = require('puppeteer');
const BASE = process.argv[2] || 'http://127.0.0.1:8091';

// Faz 189: kategori + arama + arsiv sayfalarinin dil davranisi.
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'], protocolTimeout: 60000 });
  const p = await b.newPage();
  await p.setViewport({ width: 1440, height: 1000 });
  const hata = [];
  p.on('pageerror', (e) => hata.push(e.message));

  const oku = async (yol) => {
    const r = await p.goto(BASE + yol, { waitUntil: 'networkidle2', timeout: 60000 }).catch(() => null);
    await new Promise((x) => setTimeout(x, 500));
    const d = await p.evaluate(() => ({
      sonUrl: location.pathname,
      lang: document.documentElement.lang,
      baslik: (document.title || '').slice(0, 60),
      canonical: (document.querySelector('link[rel=canonical]') || {}).href || '',
      robots: (document.querySelector('meta[name=robots]') || {}).content || '',
      hreflangSayi: document.querySelectorAll('link[rel=alternate][hreflang]').length,
      kart: document.querySelectorAll('a[href*="-"]').length,
      yatayTasma: Math.max(0, document.documentElement.scrollWidth - window.innerWidth),
    }));
    d.kod = r ? r.status() : 0;
    d.istek = yol;
    console.log(JSON.stringify(d));
    return d;
  };

  // Kategori slug'ini ana sayfadan bul
  await p.goto(BASE + '/', { waitUntil: 'networkidle2' });
  const kat = await p.evaluate(() => {
    const a = [...document.querySelectorAll('a[href*="/kategori/"]')].find((x) => x.getAttribute('href'));
    return a ? a.getAttribute('href').replace(/^\/[a-z]{2}\//, '/') : '/kategori/gundem';
  });
  console.log('kategori:', kat);

  await oku(kat);
  await oku('/en' + kat);
  await oku('/de' + kat);
  await oku('/en/arama?q=istanbul');

  console.log('JS hatasi:', hata.length ? [...new Set(hata)].slice(0, 2).join(' | ') : 'yok');
  await b.close();
})();
