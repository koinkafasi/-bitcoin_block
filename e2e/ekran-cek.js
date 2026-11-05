const puppeteer = require('puppeteer');
const fs = require('fs');

const BASE = process.argv[2];
const SID = process.argv[3];

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 1000, deviceScaleFactor: 1 });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  const sayfalar = [
    ['/admin', 'pano'],
    ['/admin/icerikler', 'icerikler'],
    ['/admin/kategoriler', 'kategoriler'],
  ];
  for (const [yol, ad] of sayfalar) {
    await p.goto(BASE + yol, { waitUntil: 'networkidle2' });
    await new Promise((r) => setTimeout(r, 700));
    await p.screenshot({ path: `/tmp/shot-${ad}.png` });
    console.log('çekildi:', ad);
  }

  // Menü etiketlerinde emoji kaldı mı?
  const metin = await p.evaluate(() =>
    [...document.querySelectorAll('.admin-side .menu > li > a')].map((a) => a.textContent.trim()));
  console.log('\nmenü etiketleri:', metin.slice(0, 6).join(' | '));
  console.log('emoji var mı:', metin.some((m) => /[\u{1F300}-\u{1FAFF}\u{2600}-\u{27BF}]/u.test(m)));

  const pin = await p.evaluate(() => {
    const b = document.querySelector('.admin-side .pin-btn');
    if (!b) return null;
    return { label: b.getAttribute('aria-label'), svg: !!b.querySelector('svg'), metin: b.textContent.trim() };
  });
  console.log('pin düğmesi:', JSON.stringify(pin));

  await b.close();
})();
