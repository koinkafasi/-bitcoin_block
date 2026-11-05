const puppeteer = require('puppeteer');

const BASE = process.argv[2];
const SID = process.argv[3];

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 900 });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });
  await p.goto(BASE + '/admin/icerikler/yeni', { waitUntil: 'networkidle2' });
  await new Promise((r) => setTimeout(r, 1200));

  const bilgi = await p.evaluate(() => {
    const drags = [...document.querySelectorAll('.be-drag')];
    const ed = document.getElementById('blockEditor');
    const zincir = [];
    let e = drags[0];
    while (e && e !== document.body) {
      const c = getComputedStyle(e);
      const r = e.getBoundingClientRect();
      zincir.push({
        tag: e.tagName + '.' + (typeof e.className === 'string' ? e.className : ''),
        pos: c.position,
        kutu: [Math.round(r.x), Math.round(r.y), Math.round(r.width), Math.round(r.height)],
      });
      e = e.parentElement;
    }
    return {
      dragSayisi: drags.length,
      dragKutu: drags.map((d) => { const r = d.getBoundingClientRect(); return [Math.round(r.x), Math.round(r.y)]; }),
      editorVar: !!ed,
      editorKutu: ed ? (() => { const r = ed.getBoundingClientRect(); return [Math.round(r.x), Math.round(r.y), Math.round(r.width), Math.round(r.height)]; })() : null,
      zincir,
    };
  });
  console.log(JSON.stringify(bilgi, null, 1));
  await b.close();
})();
