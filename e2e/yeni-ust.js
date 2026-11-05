const puppeteer = require('puppeteer');

const BASE = process.argv[2];
const SID = process.argv[3];

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 900, deviceScaleFactor: 2 });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });
  await p.goto(BASE + '/admin/icerikler/yeni', { waitUntil: 'networkidle2' });
  await new Promise((r) => setTimeout(r, 900));

  await p.screenshot({ path: '/tmp/yeni-ust.png', clip: { x: 260, y: 110, width: 820, height: 260 } });
  console.log('çekildi');

  // İlk details/summary ölçüleri
  const d = await p.evaluate(() => {
    const det = document.querySelector('details.form-grup');
    if (!det) return 'details yok';
    const s = det.querySelector('summary');
    const dc = getComputedStyle(det), sc = s ? getComputedStyle(s) : null;
    const dr = det.getBoundingClientRect(), sr = s ? s.getBoundingClientRect() : null;
    return {
      details: { kutu: [Math.round(dr.x), Math.round(dr.y), Math.round(dr.width), Math.round(dr.height)], padding: dc.padding, margin: dc.margin, border: dc.border },
      summary: s ? { metin: s.textContent.trim().slice(0, 30), kutu: [Math.round(sr.x), Math.round(sr.y), Math.round(sr.width), Math.round(sr.height)], margin: sc.margin, pos: sc.position, display: sc.display, height: sc.height, overflow: sc.overflow } : null,
      ilkCocuk: det.firstElementChild ? det.firstElementChild.tagName + '.' + det.firstElementChild.className : null,
    };
  });
  console.log(JSON.stringify(d, null, 1));

  // form-grup sayısı ve açık olanlar
  const gruplar = await p.evaluate(() => [...document.querySelectorAll('details.form-grup')].slice(0, 4).map((d) => ({
    ozet: (d.querySelector('summary') || {}).textContent || '',
    acik: d.open,
  })));
  console.log('gruplar:', JSON.stringify(gruplar, null, 1));

  await b.close();
})();
