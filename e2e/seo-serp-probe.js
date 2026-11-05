const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

// Mevcut davranis: baslik degisince SERP onizlemesi guncelleniyor mu?
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'], protocolTimeout: 60000 });
  const p = await b.newPage();
  await p.setViewport({ width: 1440, height: 1000 });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });
  await p.goto(BASE + '/admin/icerikler', { waitUntil: 'networkidle2' });
  const link = await p.evaluate(() => {
    const a = [...document.querySelectorAll('a[href*="/admin/icerikler/"]')]
      .find((x) => /\/admin\/icerikler\/\d+$/.test(x.getAttribute('href') || ''));
    return a ? a.getAttribute('href') : null;
  });
  await p.goto(BASE + link, { waitUntil: 'networkidle2' });
  await new Promise((x) => setTimeout(x, 800));

  const d = await p.evaluate(() => {
    const serp = document.querySelector('.serp .t');
    const before = serp ? serp.textContent : '(yok)';
    const t = document.querySelector('#f_title');
    t.value = 'TAMAMEN YENI BIR BASLIK DENEMESI 12345';
    t.dispatchEvent(new Event('input', { bubbles: true }));
    const after = serp ? serp.textContent : '(yok)';
    return {
      kapsayiciVar: !!document.querySelector('[data-seo-preview]'),
      once: before.slice(0, 60),
      sonra: after.slice(0, 60),
      guncellendi: before !== after,
      sosyalGuncel: (document.querySelector('[data-bind=social-title]') || {}).textContent || '',
      gauge: (document.querySelector('.gauge') || {}).getAttribute ? document.querySelector('.gauge').getAttribute('data-score') : '',
    };
  });
  console.log(JSON.stringify(d, null, 1));
  await b.close();
})();
