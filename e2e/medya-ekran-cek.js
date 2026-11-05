const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'], protocolTimeout: 60000 });
  const p = await b.newPage();
  await p.setViewport({ width: 1400, height: 1000, deviceScaleFactor: 2 });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });
  await p.goto(BASE + '/admin/dosyalar', { waitUntil: 'networkidle2' });
  await new Promise((x) => setTimeout(x, 1200));

  async function cek(dosya, yukseklik) {
    const el = await p.$('#medya-panel .media-grid');
    if (!el) return;
    const c = await el.boundingBox();
    if (!c) return;
    await p.screenshot({
      path: dosya,
      clip: {
        x: Math.max(0, c.x - 6),
        y: Math.max(0, c.y - 6),
        width: Math.min(1390, c.width + 12),
        height: Math.min(yukseklik, c.height),
      },
    });
  }

  await cek('/tmp/mg-kart.png', 620);
  await p.click('[data-media-view="list"]');
  await new Promise((x) => setTimeout(x, 500));
  await cek('/tmp/mg-liste.png', 520);
  await p.evaluate(() => { document.documentElement.setAttribute('data-theme', 'dark'); });
  await p.click('[data-media-view="grid"]');
  await new Promise((x) => setTimeout(x, 500));
  await cek('/tmp/mg-koyu.png', 620);
  console.log('ekranlar alindi');
  await b.close();
})();
