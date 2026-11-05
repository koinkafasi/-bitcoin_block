const puppeteer = require('puppeteer');
const BASE = process.argv[2] || 'http://127.0.0.1:8091';
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'], protocolTimeout: 60000 });
  const p = await b.newPage();
  await p.setViewport({ width: 1440, height: 1000 });
  for (const yol of ['/', '/en']) {
    await p.goto(BASE + yol, { waitUntil: 'networkidle2' });
    await new Promise((r) => setTimeout(r, 800));
    const d = await p.evaluate(() => {
      const bolumler = [];
      document.querySelectorAll('section').forEach((s) => {
        const h = s.querySelector('h2, h3');
        if (!h) return;
        const link = s.querySelectorAll('a[href]').length;
        bolumler.push({ ad: (h.textContent || '').trim().slice(0, 26), link });
      });
      return {
        manset: document.querySelectorAll('.hero-title, .hero-slide, [class*="hero"]').length,
        bolumSayisi: bolumler.length,
        bosBolumler: bolumler.filter((x) => x.link === 0).map((x) => x.ad),
        toplamLink: document.querySelectorAll('a[href]').length,
        yatayTasma: Math.max(0, document.documentElement.scrollWidth - window.innerWidth),
        sayfaYuksekligi: document.body.scrollHeight,
      };
    });
    console.log('\n' + yol + ':', JSON.stringify(d, null, 1));
  }
  await b.close();
})();
