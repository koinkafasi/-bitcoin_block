// Üretim panosunu gerçek admin oturumuyla ölçer ve ekran görüntüsü alır.
const puppeteer = require('puppeteer');
const fs = require('fs');

const BASE = 'http://127.0.0.1:8070';
const SID = process.argv[2];
const OUT = '/tmp/pano-canli';

(async () => {
  fs.mkdirSync(OUT, { recursive: true });
  const browser = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox', '--disable-dev-shm-usage'] });
  const page = await browser.newPage();
  await page.setViewport({ width: 1500, height: 1100 });
  const hatalar = [];
  page.on('pageerror', (e) => hatalar.push(String(e.message)));
  page.on('console', (m) => { if (m.type() === 'error') hatalar.push('console: ' + m.text()); });
  if (SID) await page.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  await page.goto(BASE + '/admin/pano', { waitUntil: 'networkidle2', timeout: 60000 });
  await new Promise((r) => setTimeout(r, 1200));

  const o = await page.evaluate(() => {
    const b = (s) => { const e = document.querySelector(s); if (!e) return null; const r = e.getBoundingClientRect(); return { w: Math.round(r.width), h: Math.round(r.height) }; };
    return {
      url: location.pathname,
      baslik: (document.querySelector('.pn-head h1') || {}).textContent,
      kpi: [...document.querySelectorAll('.pn-kpi')].map((k) => ({
        ad: k.querySelector('.pn-kpi__label span').textContent,
        deger: k.querySelector('.pn-kpi__value').textContent,
        not: k.querySelector('.pn-kpi__note').textContent.replace(/\s+/g, ' ').trim(),
      })),
      alan: b('#pnChartBox .pn-seriwrap.on svg'),
      nokta: document.querySelectorAll('#pnChartBox .pn-seriwrap.on .pn-nokta').length,
      seriDegerleri: [...document.querySelectorAll('#pnChartBox .pn-seriwrap.on .pn-nokta')].map((n) => +n.getAttribute('data-v')),
      halka: b('.pn-halka'),
      dilim: document.querySelectorAll('.pn-dilim').length,
      lejant: [...document.querySelectorAll('.pn-lejant li')].slice(0, 4).map((l) => l.textContent.replace(/\s+/g, ' ').trim()),
      sutun: [...document.querySelectorAll('.pn-cubuk')].map((r) => r.getAttribute('data-v')),
      yatay: document.querySelectorAll('.pn-yatay__satir').length,
      serit: b('.pn-plan__ray'),
      kart: document.querySelectorAll('.pn-visit').length,
      aktivite: document.querySelectorAll('.pn-akt li').length,
      nav: document.querySelectorAll('.pn-item').length,
      hedef: (document.querySelector('.pn-hedef small') || {}).textContent,
    };
  });

  console.log('url        :', o.url);
  console.log('başlık     :', o.baslik);
  console.log('nav öğe    :', o.nav, '· hedef:', o.hedef);
  console.log('--- KPI ---');
  o.kpi.forEach((k) => console.log('  ', k.ad.padEnd(16), k.deger.padStart(5), '·', k.not));
  console.log('--- grafikler ---');
  console.log('  alan      :', o.alan && o.alan.w + 'x' + o.alan.h, '·', o.nokta, 'nokta');
  console.log('  seri      :', o.seriDegerleri.join(','));
  console.log('  halka     :', o.halka && o.halka.w + 'px ·', o.dilim, 'dilim');
  o.lejant.forEach((l) => console.log('     ', l));
  console.log('  sütun     :', o.sutun.join(','));
  console.log('  yatay     :', o.yatay, 'satır');
  console.log('  şerit     :', o.serit && o.serit.w + 'x' + o.serit.h, '·', o.kart, 'kart');
  console.log('  aktivite  :', o.aktivite, 'satır');

  await page.screenshot({ path: OUT + '/pano.png', fullPage: false });
  await page.screenshot({ path: OUT + '/pano-tam.png', fullPage: true });

  // Etkileşim: sekme + sürükleme (DB'ye yazmasın diye sürükleme YAPILMIYOR)
  await page.click('#pnSeg button[data-pn-seri="30"]');
  await new Promise((r) => setTimeout(r, 400));
  const o30 = await page.evaluate(() => ({
    aktif: document.querySelector('#pnSeg button[aria-pressed="true"]').textContent,
    nokta: document.querySelectorAll('#pnChartBox .pn-seriwrap.on .pn-nokta').length,
  }));
  console.log('--- sekme ---');
  console.log('  30 gün:', o30.aktif, '·', o30.nokta, 'nokta');

  console.log('--- JS hatası ---');
  console.log(hatalar.length ? hatalar.slice(0, 4).join('\n') : 'yok');
  await browser.close();
})();
