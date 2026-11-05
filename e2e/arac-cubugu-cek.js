const puppeteer = require('puppeteer');

// Araç çubuğu ve blok paletinin yakın görüntüsü.
const BASE = process.argv[2];
const SID = process.argv[3];

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 1000, deviceScaleFactor: 2 });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });
  await p.goto(BASE + '/admin/icerikler/yeni', { waitUntil: 'networkidle2' });
  await new Promise((r) => setTimeout(r, 1500));

  const tb = await p.$('.be-toolbar');
  if (tb) {
    await tb.screenshot({ path: '/tmp/arac-cubugu.png' });
    console.log('araç çubuğu çekildi');
  } else {
    console.log('araç çubuğu YOK');
  }

  // "Daha fazla" menüsünü aç ve çek
  const more = await p.$('.be-more-toggle');
  // "Daha fazla" düğmesi dar ekranda görünür; geniş ekranda gizli
  // (CSS .be-more-toggle{display:none} → medya sorgusu). Görünmüyorsa atla.
  if (more && (await more.evaluate((e) => e.offsetParent !== null))) {
    await more.click();
    await new Promise((r) => setTimeout(r, 350));
    const menu = await p.$('#editorMoreMenu');
    if (menu) {
      await menu.screenshot({ path: '/tmp/arac-menu.png' });
      console.log('menü çekildi');
    }
  }

  const durum = await p.evaluate(() => {
    const tb = document.querySelector('.be-toolbar');
    if (!tb) return null;
    const btn = [...tb.querySelectorAll('.be-btn')].find((x) => x.querySelector('svg'));
    const s = btn.querySelector('svg').getBoundingClientRect();
    return {
      yukseklik: Math.round(tb.getBoundingClientRect().height),
      dugme: Math.round(btn.getBoundingClientRect().height),
      ikon: Math.round(s.width) + 'x' + Math.round(s.height),
      toplamDugme: tb.querySelectorAll('.be-btn').length,
      svgsizDugme: [...tb.querySelectorAll('.be-btn')].filter((x) => !x.querySelector('svg')).map((x) => x.textContent),
    };
  });
  console.log(JSON.stringify(durum));
  await b.close();
})();
