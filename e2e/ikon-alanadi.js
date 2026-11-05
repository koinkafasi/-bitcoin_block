const puppeteer = require('puppeteer');
const BASE = process.argv[2];

// Alan adindan (Cloudflare uzerinden) ikonlar gercekten ciziliyor mu?
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1440, height: 1000 });
  const hata = [];
  p.on('pageerror', (e) => hata.push(e.message));

  const r = await p.goto(BASE + '/', { waitUntil: 'networkidle2' });
  await new Promise((x) => setTimeout(x, 900));

  const d = await p.evaluate(async () => {
    // Sprite hangi URL'den geldi ve kac sembol var?
    const kaynaklar = performance.getEntriesByType('resource')
      .map((e) => e.name).filter((n) => n.includes('tabler-icons.svg'));
    let sembolSayi = 0;
    let surum = '';
    if (kaynaklar.length) {
      surum = (kaynaklar[0].match(/\?v=[0-9a-f]+/) || [''])[0];
      try {
        const t = await (await fetch(kaynaklar[0])).text();
        sembolSayi = (t.match(/<symbol/g) || []).length;
      } catch (e) { sembolSayi = -1; }
    }

    const ikonlar = Array.from(document.querySelectorAll('svg.ti, svg.pn-ico'));
    let cizilen = 0, sifir = 0;
    for (const s of ikonlar) {
      const r = s.getBoundingClientRect();
      const u = s.querySelector('use');
      const href = u ? (u.getAttribute('href') || '') : '';
      if (!href.includes('#')) continue;
      // Gercekten boyandi mi: sembolun kendi kutusu var mi
      const bb = s.getBBox ? s.getBBox() : null;
      if (r.width > 2 && r.height > 2 && bb && bb.width > 0) cizilen++;
      else sifir++;
    }
    return { kaynaklar: kaynaklar.slice(0, 2), surum, sembolSayi, toplam: ikonlar.length, cizilen, sifir };
  });

  console.log(JSON.stringify(d, null, 1));
  console.log('JS hatası:', hata.length ? hata[0] : 'yok');
  await b.close();
})();
