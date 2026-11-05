const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'], protocolTimeout: 60000 });
  const p = await b.newPage();
  await p.setViewport({ width: 1440, height: 1000 });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });
  await p.goto(BASE + '/admin/dosyalar', { waitUntil: 'networkidle2' });
  await new Promise((x) => setTimeout(x, 900));

  const d = await p.evaluate(() => {
    const grid = document.querySelector('#medya-panel .media-grid');
    const kartlar = [...grid.querySelectorAll('.media-item')].map((el) => el.getBoundingClientRect());
    const k0 = kartlar[0];
    const altlar = kartlar
      .filter((r) => Math.abs(r.left - k0.left) < 2 && r.top > k0.top + 4)
      .sort((a, b) => a.top - b.top);
    const dropEl = grid.querySelector('.mgal-drop');
    const drop = dropEl.getBoundingClientRect();
    const dropAlt = kartlar
      .filter((r) => Math.abs(r.left - drop.left) < 2 && r.top > drop.top + 4)
      .sort((a, b) => a.top - b.top);
    return {
      kart: { w: Math.round(k0.width), h: Math.round(k0.height) },
      dikeyBosluk: altlar.length ? Math.round(altlar[0].top - k0.bottom) : null,
      drop: { w: Math.round(drop.width), h: Math.round(drop.height) },
      dropIleKart: dropAlt.length ? Math.round(dropAlt[0].top - drop.bottom) : null,
    };
  });
  console.log(JSON.stringify(d, null, 1));
  await b.close();
})();
