const puppeteer = require('puppeteer');

const BASE = process.argv[2];
const SID = process.argv[3];

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 1000 });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });
  await p.goto(BASE + '/admin', { waitUntil: 'networkidle2' });
  await new Promise((r) => setTimeout(r, 600));

  const sonuc = await p.evaluate(async () => {
    const out = {};
    for (const q of ['saglik', 'Sağlık', 'SAGLIK', 'yardim', 'bakanligi']) {
      const t0 = performance.now();
      const r = await fetch('/admin/ara?q=' + encodeURIComponent(q), { credentials: 'same-origin' });
      const d = await r.json();
      out[q] = {
        ms: Math.round(performance.now() - t0),
        gruplar: (d.gruplar || []).map((g) => g.ad + '(' + g.oge.length + ')'),
        ilk: ((d.gruplar || [])[0] || {}).oge?.[0]?.baslik || '',
      };
    }
    return out;
  });
  console.log(JSON.stringify(sonuc, null, 1));
  await b.close();
})();
