const puppeteer = require('puppeteer');
const BASE = process.argv[2] || 'http://127.0.0.1:8070';

// Dil gecisi + tasarim olcumu (public site).
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'], protocolTimeout: 90000 });
  const p = await b.newPage();
  await p.setViewport({ width: 1440, height: 1000 });
  const hata = [];
  p.on('pageerror', (e) => hata.push(e.message));

  const olc = async (etiket) => {
    const d = await p.evaluate(() => {
      const q = (s) => document.querySelector(s);
      const manset = document.querySelectorAll('.hero-title, .hero-slide, [class*="hero"]').length;
      const kart = document.querySelectorAll('.m6-title, .ek-card, .sb-card, .u3-card').length;
      const docW = document.documentElement.scrollWidth;
      const winW = window.innerWidth;
      // Bos bolum: basligi olup icerigi olmayan
      const bosBolum = [];
      document.querySelectorAll('section, .hp-section, .galeri-etiket').forEach((el) => {
        const baslik = el.querySelector('h2, h3');
        if (!baslik) return;
        const link = el.querySelectorAll('a[href]').length;
        if (link === 0) bosBolum.push((baslik.textContent || '').trim().slice(0, 30));
      });
      return {
        url: location.pathname,
        lang: document.documentElement.lang,
        baslik: (document.title || '').slice(0, 60),
        manset,
        kart,
        yatayTasma: Math.max(0, docW - winW),
        bosBolumler: [...new Set(bosBolum)].slice(0, 6),
        seciliDil: (q('[data-lang-select]') || {}).value || '',
        langBase: (q('[data-lang-select]') || {}).getAttribute ? q('[data-lang-select]').getAttribute('data-base') : '',
      };
    });
    console.log(etiket + ':', JSON.stringify(d, null, 1));
    return d;
  };

  await p.goto(BASE + '/', { waitUntil: 'networkidle2', timeout: 60000 });
  await new Promise((r) => setTimeout(r, 1200));
  const tr = await olc('TR anasayfa');

  // Dil secicisini "en" yap (gercek kullanici akisi)
  const varMi = await p.evaluate(() => !!document.querySelector('[data-lang-select]'));
  console.log('\ndil secici var mi:', varMi);
  if (varMi) {
    await p.select('[data-lang-select]', 'en');
    await new Promise((r) => setTimeout(r, 2500));
    await olc('EN gecis sonrasi');
  }

  // Dogrudan /en
  await p.goto(BASE + '/en', { waitUntil: 'networkidle2', timeout: 60000 });
  await new Promise((r) => setTimeout(r, 1200));
  await olc('EN dogrudan');

  // /de
  await p.goto(BASE + '/de', { waitUntil: 'networkidle2', timeout: 60000 });
  await new Promise((r) => setTimeout(r, 1200));
  await olc('DE dogrudan');

  // EN makale (yonlendirme)
  const link = await p.evaluate(() => {
    const a = document.querySelector('.m6-title a, .ek-card a, .sb-card a, a[href*="-"]');
    return a ? a.getAttribute('href') : null;
  }).catch(() => null);
  if (link) {
    const enLink = '/en' + (link.startsWith('/') ? link : '/' + link);
    const r = await p.goto(BASE + enLink, { waitUntil: 'networkidle2', timeout: 60000 }).catch(() => null);
    await new Promise((x) => setTimeout(x, 800));
    console.log('\nEN makale:', enLink, '->', r ? r.status() : '?', '· son url:', new URL(p.url()).pathname);
    const md = await p.evaluate(() => ({
      lang: document.documentElement.lang,
      baslik: (document.title || '').slice(0, 60),
      canonical: (document.querySelector('link[rel=canonical]') || {}).href || '',
      hreflang: [...document.querySelectorAll('link[rel=alternate][hreflang]')].map((l) => l.hreflang + '=' + l.href),
    }));
    console.log('  ', JSON.stringify(md, null, 1));
  }

  console.log('\nJS hatasi:', hata.length ? [...new Set(hata)].slice(0, 3).join(' | ') : 'yok');
  await b.close();
})();
