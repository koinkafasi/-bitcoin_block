const puppeteer = require('puppeteer');

const BASE = process.argv[2];
const SID = process.argv[3];

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 1000, deviceScaleFactor: 2 });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });
  await p.goto(BASE + '/admin/icerikler', { waitUntil: 'networkidle2' });
  await new Promise((r) => setTimeout(r, 600));

  const form = await p.$('form.filtre-grid');
  if (form) {
    await form.screenshot({ path: '/tmp/shot-filtre.png' });
    console.log('filtre formu çekildi');
  }
  const el = await p.$('.ws-tabs');
  if (el) {
    await el.screenshot({ path: '/tmp/shot-sekme.png' });
    console.log('sekme çekildi');
  }
  const side = await p.$('.admin-side');
  if (side) {
    await side.screenshot({ path: '/tmp/shot-menu.png' });
    console.log('menü çekildi');
  }
  await b.close();
})();
