const puppeteer = require('puppeteer');

const BASE = process.argv[2];
const SID = process.argv[3];

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 900 });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });
  await p.goto(BASE + '/admin/icerikler/yeni', { waitUntil: 'networkidle2' });
  await new Promise((r) => setTimeout(r, 800));

  const bulunan = await p.evaluate(() => {
    const hedef = { x: 292, y: 137 };
    const out = [];
    document.querySelectorAll('.admin-main *').forEach((e) => {
      const r = e.getBoundingClientRect();
      if (r.width === 0 || r.height === 0) return;
      if (r.x <= hedef.x && r.x + r.width >= hedef.x && r.y <= hedef.y && r.y + r.height >= hedef.y) {
        const c = getComputedStyle(e);
        out.push({
          tag: e.tagName + '.' + (e.className || ''),
          metin: (e.textContent || '').trim().slice(0, 30),
          kutu: [Math.round(r.x), Math.round(r.y), Math.round(r.width), Math.round(r.height)],
          pos: c.position, bg: c.backgroundColor, z: c.zIndex, opacity: c.opacity,
        });
      }
    });
    return out;
  });
  console.log(JSON.stringify(bulunan, null, 1));
  await b.close();
})();
