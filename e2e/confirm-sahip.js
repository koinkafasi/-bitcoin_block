const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

// data-confirm'i URL olan, hx-post'u olmayan öğelerin form içinde olup
// olmadığını ve app.js'in onları gerçekten üstlenip üstlenmediğini ölçer.
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 1000 });
  const hata = [];
  p.on('pageerror', (e) => hata.push(e.message));
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  const sayfalar = ['/admin/icerikler', '/admin/ayarlar', '/admin/sistem',
    '/admin/siteler', '/admin/bot', '/admin/kumeler', '/admin/mahrecler',
    '/admin/gazete', '/admin/dosyalar', '/admin/kategoriler', '/admin/yorumlar'];

  for (const yol of sayfalar) {
    const r = await p.goto(BASE + yol, { waitUntil: 'networkidle2' }).catch(() => null);
    if (!r || r.status() !== 200) { console.log(yol, 'atlandı', r ? r.status() : 'yok'); continue; }
    await new Promise((x) => setTimeout(x, 350));
    const d = await p.evaluate(() => {
      const hepsi = [...document.querySelectorAll('[data-confirm]')];
      return hepsi.map((el) => {
        const v = el.getAttribute('data-confirm') || '';
        return {
          etiket: el.tagName.toLowerCase(),
          url: v.startsWith('/'),
          deger: v.slice(0, 34),
          formda: !!el.closest('form'),
          hx: el.hasAttribute('hx-post'),
          hxConfirm: el.hasAttribute('hx-confirm'),
          // Kim üstlenecek?
          sahip: v.startsWith('/') && el.tagName !== 'FORM' && !el.closest('form')
            ? 'app.js' : (el.hasAttribute('hx-post') ? 'htmx' : (el.tagName === 'FORM' || el.closest('form') ? 'admin-ui' : 'YOK')),
        };
      });
    });
    if (d.length) {
      console.log('---', yol);
      d.forEach((x) => console.log('   ', JSON.stringify(x)));
    }
  }
  console.log('JS hatası:', hata.length ? [...new Set(hata)].join(' | ') : 'yok');
  await b.close();
})();
