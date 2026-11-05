const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

// Ikon teshisi: her .ti icin sembol var mi, olcusu ne, rengi ne,
// gercekten ciziliyor mu.
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1440, height: 1000 });
  const hata = [];
  p.on('pageerror', (e) => hata.push(e.message));
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  const sayfalar = [
    ['admin', '/admin'],
    ['içerik formu', '/admin/icerikler/yeni'],
    ['icerikler', '/admin/icerikler'],
    ['kategoriler', '/admin/kategoriler'],
    ['public ana', '/'],
  ];

  for (const [ad, yol] of sayfalar) {
    const r = await p.goto(BASE + yol, { waitUntil: 'networkidle2' }).catch(() => null);
    await new Promise((x) => setTimeout(x, 700));
    if (!r || r.status() !== 200) { console.log(ad, 'atlandı', r ? r.status() : ''); continue; }

    const d = await p.evaluate(() => {
      const ikonlar = Array.from(document.querySelectorAll('svg.ti, svg.pn-ico, svg[class*="ti"]'));
      const eksik = [];
      const sifirBoyut = [];
      const gorunmez = [];
      let toplam = 0;
      const ornek = [];

      // Sprite'taki sembolleri oku
      const spriteKok = document.querySelector('body > svg[aria-hidden="true"]');
      for (const svg of ikonlar) {
        toplam++;
        const use = svg.querySelector('use');
        const href = use ? (use.getAttribute('href') || use.getAttribute('xlink:href') || '') : '';
        const kim = href.split('#')[1] || '';
        const s = getComputedStyle(svg);
        const rect = svg.getBoundingClientRect();
        if (!kim) { eksik.push('(use yok)'); continue; }
        if (rect.width < 2 || rect.height < 2) {
          sifirBoyut.push(kim + ' ' + Math.round(rect.width) + 'x' + Math.round(rect.height));
        }
        // renk zemine esit mi (gorunmez)
        const renk = s.color;
        let bg = null, c = svg;
        while (c) {
          const cs = getComputedStyle(c);
          if (cs.backgroundColor && !/rgba\(0, 0, 0, 0\)/.test(cs.backgroundColor)) { bg = cs.backgroundColor; break; }
          c = c.parentElement;
        }
        if (bg && renk === bg) gorunmez.push(kim + ' renk=zemin');
        if (ornek.length < 6) {
          ornek.push({ kim, w: Math.round(rect.width), h: Math.round(rect.height), stroke: s.stroke, fill: s.fill, color: s.color, display: s.display });
        }
      }

      // Sembol var mi: sprite'i fetch edip id seti cikar
      return { toplam, eksik: eksik.slice(0, 5), sifirBoyut: sifirBoyut.slice(0, 5), gorunmez: gorunmez.slice(0, 5), ornek };
    });

    console.log('\n### ' + ad + ' (' + yol + ')  ikon: ' + d.toplam);
    console.log('  sifir boyut:', d.sifirBoyut.length ? d.sifirBoyut : 'yok');
    console.log('  gorunmez:', d.gorunmez.length ? d.gorunmez : 'yok');
    console.log('  use yok:', d.eksik.length ? d.eksik : 'yok');
    d.ornek.forEach((x) => console.log('   ', JSON.stringify(x)));
  }
  console.log('\nJS hatası:', hata.length ? [...new Set(hata)].slice(0, 3).join(' | ') : 'yok');
  await b.close();
})();
