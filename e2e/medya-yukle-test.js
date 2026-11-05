const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

// Faz 186: gercek PNG ile uctan uca yukleme.
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'], protocolTimeout: 120000 });
  const p = await b.newPage();
  await p.setViewport({ width: 1440, height: 1000 });
  const hata = [];
  p.on('pageerror', (e) => hata.push(e.message));
  const yanitlar = [];
  p.on('response', async (res) => {
    const u = res.url();
    if (!u.includes('/admin/dosyalar')) return;
    let govde = '';
    try { govde = (await res.text()).slice(0, 160); } catch (e) {}
    yanitlar.push(res.status() + ' ' + u.replace(BASE, '') + ' :: ' + govde.replace(/\s+/g, ' ').slice(0, 90));
  });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  await p.goto(BASE + '/admin/dosyalar', { waitUntil: 'networkidle2' });
  await new Promise((x) => setTimeout(x, 800));
  const once = await p.evaluate(() => document.querySelectorAll('#medya-panel .media-item').length);

  const input = await p.$('#medya-panel [data-mg-file]');
  await input.uploadFile('/tmp/mg-test.png', '/tmp/mg-test-2.png');

  await new Promise((x) => setTimeout(x, 150));
  const kart = await p.evaluate(() => {
    const el = document.querySelector('.mgal-up');
    if (!el) return null;
    const arc = el.querySelector('.mgal-up__arc');
    return {
      pct: el.querySelector('.mgal-up__pct').textContent,
      ad: el.querySelector('.mgal-up__name').textContent,
      dashoffset: arc.getAttribute('stroke-dashoffset'),
      span: el.style.gridRowEnd,
      sayi: document.querySelectorAll('.mgal-up').length,
    };
  });

  await new Promise((x) => setTimeout(x, 7000));
  const sonra = await p.evaluate(() => ({
    sayi: document.querySelectorAll('#medya-panel .media-item').length,
    kalanKart: document.querySelectorAll('.mgal-up').length,
    hataKarti: document.querySelectorAll('.mgal-up.is-hata').length,
    ilkBaslik: (document.querySelector('#medya-panel .mi-cap b') || {}).textContent || '',
  }));

  console.log('ONCE :', once);
  console.log('KART :', JSON.stringify(kart));
  console.log('SONRA:', JSON.stringify(sonra));
  console.log('yanitlar:');
  yanitlar.slice(0, 5).forEach((x) => console.log('  ' + x));
  console.log('JS hatası:', hata.length ? [...new Set(hata)].slice(0, 2).join(' | ') : 'yok');
  await b.close();
})();
