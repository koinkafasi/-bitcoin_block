const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 1000 });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });
  await p.goto(BASE + '/admin/tasarim', { waitUntil: 'networkidle2' });
  await new Promise((x) => setTimeout(x, 1200));

  const d = await p.evaluate(() => {
    const out = [];
    document.querySelectorAll('.manset-cat, .card-badge, .cat-shape-badge').forEach((el) => {
      const s = getComputedStyle(el);
      if (el.getClientRects().length === 0) return;
      out.push({
        html: el.outerHTML.slice(0, 160),
        renk: s.color, zemin: s.backgroundColor,
        ic: !!el.closest('.bp-prev'),
      });
    });
    return out.slice(0, 6);
  });
  console.log(JSON.stringify(d, null, 1));
  await b.close();
})();
