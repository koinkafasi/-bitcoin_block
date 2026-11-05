const puppeteer = require('puppeteer');
const BASE = process.argv[2] || 'http://127.0.0.1:8091';

// Faz 189: Ingilizce icerik gercekten yayinda mi, canonical/hreflang dogru mu,
// tema bozuluyor mu?
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'], protocolTimeout: 90000 });
  const p = await b.newPage();
  await p.setViewport({ width: 1440, height: 1000 });
  const hata = [];
  p.on('pageerror', (e) => hata.push(e.message));

  const oku = async (yol) => {
    const r = await p.goto(BASE + yol, { waitUntil: 'networkidle2', timeout: 60000 }).catch(() => null);
    await new Promise((x) => setTimeout(x, 700));
    return p.evaluate((istek) => ({
      istek,
      sonUrl: location.pathname,
      durum: 0,
      lang: document.documentElement.lang,
      baslik: (document.title || '').slice(0, 80),
      canonical: (document.querySelector('link[rel=canonical]') || {}).href || '',
      robots: (document.querySelector('meta[name=robots]') || {}).content || '',
      hreflang: [...document.querySelectorAll('link[rel=alternate][hreflang]')]
        .map((l) => l.hreflang + '→' + l.href.replace(location.origin, '')),
      h1: (document.querySelector('h1') || {}).textContent || '',
      govdeParagraf: document.querySelectorAll('.article-body p, .a-body p, article p').length,
      icerikUzunluk: (document.querySelector('.article-body, .a-body, article') || {}).textContent?.trim().length || 0,
      yatayTasma: Math.max(0, document.documentElement.scrollWidth - window.innerWidth),
      dilSecenekleri: [...document.querySelectorAll('[data-lang-select] option')].map((o) => o.value),
      etkinDilLinkleri: [...document.querySelectorAll('a[hreflang]')].map((a) => a.hreflang + '=' + a.getAttribute('href')),
    }), yol).then((d) => { d.durum = r ? r.status() : 0; return d; });
  };

  await oku('/');
  const en = await oku('/en');
  console.log('\n=== /en ===');
  console.log(JSON.stringify(en, null, 1));

  // EN ana sayfadaki bir makale baglantisini bul ve ac
  const bag = await p.evaluate(() => {
    const as = [...document.querySelectorAll('a[href]')];
    const en = as.find((a) => /^\/en\/[a-z0-9-]+-\d+$/.test(a.getAttribute('href') || ''));
    const herhangi = as.find((a) => /^\/[a-z0-9-]+-\d+$/.test(a.getAttribute('href') || ''));
    return en || null;
  });
  console.log('\nEN makale baglantisi:', bag);
  if (bag) {
    const m = await oku(bag);
    console.log(JSON.stringify(m, null, 1));
  }

  console.log('\nJS hatasi:', hata.length ? [...new Set(hata)].slice(0, 3).join(' | ') : 'yok');
  await b.close();
})();
