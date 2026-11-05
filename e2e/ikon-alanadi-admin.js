const puppeteer = require('puppeteer');
const BASE = 'https://theblock.com.tr';
const SID = process.argv[2];

// Alan adindan ADMIN sayfasi: ikonlar gercekten ciziliyor mu?
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1440, height: 1000 });
  const hata = [];
  p.on('pageerror', (e) => hata.push(e.message));
  await p.setCookie({ name: 'tb_session', value: SID, domain: 'theblock.com.tr', path: '/', secure: true });

  const sonuc = [];
  for (const yol of ['/admin', '/admin/icerikler/yeni', '/admin/kategoriler']) {
    const r = await p.goto(BASE + yol, { waitUntil: 'networkidle2' }).catch(() => null);
    await new Promise((x) => setTimeout(x, 2200));
    if (!r || r.status() !== 200) { sonuc.push({ yol, durum: r ? r.status() : 'yok' }); continue; }

    const d = await p.evaluate(async () => {
      const html = document.documentElement.outerHTML;
      const surum = (html.match(/tabler-icons\.svg\?v=([0-9a-f]+)/) || [])[1] || '';
      // Sprite'i indir: kac sembol var
      let sembol = 0;
      try {
        const t = await (await fetch('/static/vendor/tabler-icons.svg?v=' + surum)).text();
        sembol = (t.match(/<symbol/g) || []).length;
      } catch (e) { sembol = -1; }

      const ikonlar = Array.from(document.querySelectorAll('svg.ti, svg.pn-ico'));
      let cizilen = 0, bos = 0;
      const ornekBos = [];
      for (const s of ikonlar) {
        const r = s.getBoundingClientRect();
        if (r.width < 2 || r.height < 2) continue; // gizli paneller
        const bb = s.getBBox ? s.getBBox() : null;
        if (bb && bb.width > 0 && bb.height > 0) cizilen++;
        else {
          bos++;
          if (ornekBos.length < 4) {
            const u = s.querySelector('use');
            ornekBos.push(u ? u.getAttribute('href').split('#')[1] : '?');
          }
        }
      }
      return { surum, sembol, gorunen: cizilen + bos, cizilen, bos, ornekBos };
    });
    sonuc.push({ yol, durum: r.status(), ...d });
  }

  sonuc.forEach((x) => console.log(JSON.stringify(x)));
  console.log('JS hatası:', hata.length ? [...new Set(hata)].slice(0, 2).join(' | ') : 'yok');
  await b.close();
})();
