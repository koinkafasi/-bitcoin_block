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

  const liste = await p.evaluate(() => {
    const out = [];
    document.querySelectorAll('svg.ti').forEach((s) => {
      const r = s.getBoundingClientRect();
      const c = getComputedStyle(s);
      const par = s.parentElement;
      out.push({
        kutu: Math.round(r.width) + 'x' + Math.round(r.height),
        w: c.width, h: c.height,
        ebeveyn: par.tagName + '.' + (typeof par.className === 'string' ? par.className : ''),
      });
    });
    // aynı ebeveynleri tekilleştir
    const gor = new Map();
    out.forEach((o) => { gor.set(o.ebeveyn, o); });
    return [...gor.values()].slice(0, 12);
  });
  console.log(JSON.stringify(liste, null, 1));
  await b.close();
})();
