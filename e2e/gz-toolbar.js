const puppeteer = require('puppeteer');

const BASE = process.argv[2];
const SID = process.argv[3];
const SAYI = process.argv[4] || '78';

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1600, height: 1000, deviceScaleFactor: 2 });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });
  await p.goto(BASE + '/admin/gazete/sayilar/' + SAYI + '/duzenle', { waitUntil: 'networkidle2' });
  await new Promise((r) => setTimeout(r, 1500));

  const tb = await p.evaluate(() => {
    const bar = document.querySelector('.gz-toolbar');
    if (!bar) return null;
    return [...bar.querySelectorAll('.btn')].map((x) => {
      const s = x.querySelector('svg.ti');
      const r = s ? s.getBoundingClientRect() : null;
      return {
        metin: x.textContent.trim(),
        svg: s ? (s.querySelector('use').getAttribute('href') || '').split('#')[1] : null,
        kutu: r ? Math.round(r.width) + 'x' + Math.round(r.height) : null,
      };
    });
  });
  console.log(JSON.stringify(tb, null, 1));

  const el = await p.$('.gz-toolbar');
  if (el) await el.screenshot({ path: '/tmp/gz-tb.png' });
  const pal = await p.$('.gz-palette');
  if (pal) await pal.screenshot({ path: '/tmp/gz-pal.png' });
  await b.close();
})();
