const puppeteer = require('puppeteer');
const fs = require('fs');
const BASE = process.argv[2];
const SID = process.argv[3];

(async () => {
  const png = Buffer.from(
    'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8DwHwAFAAH/q842iQAAAABJRU5ErkJggg==',
    'base64'
  );
  fs.writeFileSync('/tmp/mg-test.png', png);

  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'], protocolTimeout: 120000 });
  const p = await b.newPage();
  await p.setViewport({ width: 1440, height: 1000 });
  const yanitlar = [];
  p.on('response', async (res) => {
    const u = res.url();
    if (!u.includes('/admin/dosyalar')) return;
    let govde = '';
    try { govde = (await res.text()).slice(0, 300); } catch (e) {}
    yanitlar.push({ durum: res.status(), url: u.replace(BASE, ''), govde });
  });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });
  await p.goto(BASE + '/admin/dosyalar', { waitUntil: 'networkidle2' });
  await new Promise((x) => setTimeout(x, 600));

  const input = await p.$('#medya-panel [data-mg-file]');
  await input.uploadFile('/tmp/mg-test.png');
  await new Promise((x) => setTimeout(x, 4000));

  console.log(JSON.stringify(yanitlar, null, 1));
  await b.close();
})();
