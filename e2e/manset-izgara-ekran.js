const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 1100, deviceScaleFactor: 2 });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });
  await p.goto(BASE + '/admin/manset-editor', { waitUntil: 'networkidle2' });
  await new Promise((x) => setTimeout(x, 1100));

  // Izgarayi ac
  await p.evaluate(() => {
    const k = document.getElementById('me-grid');
    k.checked = true;
    k.dispatchEvent(new Event('change', { bubbles: true }));
  });
  await new Promise((x) => setTimeout(x, 400));

  const el = await p.$('.me-stage');
  const clip = await el.boundingBox();
  if (clip) {
    await p.screenshot({
      path: '/tmp/manset-izgara-acik.png',
      clip: { x: clip.x, y: clip.y, width: Math.min(1440, clip.width), height: Math.min(720, clip.height) },
    });
  }
  console.log('ekran alindi');
  await b.close();
})();
