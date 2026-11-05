const puppeteer = require('puppeteer');
const BASE = process.argv[2], SID = process.argv[3];

// Pano içi etkileşimler. Sürükle-bırak DB'ye yazar: YALNIZ izole örnekte
// koşulmalı (üretimde gerçek bir içeriğin yayın saatini değiştirir).
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 1000 });
  const hata = [];
  p.on('pageerror', (e) => hata.push(String(e.message)));
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  await p.goto(BASE + '/admin', { waitUntil: 'networkidle2', timeout: 60000 });
  await new Promise((r) => setTimeout(r, 700));

  // 1) Sekme: 7 → 30 gün
  const s1 = await p.evaluate(() => ({
    seriler: [...document.querySelectorAll('.pn-seriwrap')].map((w) => w.classList.contains('on')),
    thumb: document.getElementById('pnSegThumb'),
  }));
  const btn = await p.$('#pnSeg button[data-pn-seri="30"]');
  await btn.click();
  await new Promise((r) => setTimeout(r, 450));
  const s2 = await p.evaluate(() => ({
    seriler: [...document.querySelectorAll('.pn-seriwrap')].map((w) => w.classList.contains('on')),
    nokta: document.querySelectorAll('#pnChartBox .pn-seriwrap.on .pn-nokta').length,
  }));
  console.log('sekme      :', s1.seriler.join(','), '→', s2.seriler.join(','), '·', s2.nokta, 'nokta');

  // 2) Balon
  const balon = await p.evaluate(() => {
    const n = document.querySelector('#pnChartBox .pn-seriwrap.on .pn-nokta');
    if (!n) return 'nokta yok';
    const r = n.getBoundingClientRect();
    n.dispatchEvent(new MouseEvent('mouseover', { bubbles: true, clientX: r.x + 1, clientY: r.y + 1 }));
    const t = document.getElementById('pnBalon');
    return t.hidden ? 'gizli' : t.textContent;
  });
  console.log('balon      :', balon);

  // 3) Sürükle-bırak (yalnız izole; DB'ye yazar)
  const drag = await p.evaluate(async () => {
    const ray = document.getElementById('pnPlanRay');
    if (!ray) return 'şerit yok';
    const el = ray.querySelector('.pn-visit');
    if (!el) return 'kart yok';
    const once = el.style.left;
    const r = ray.getBoundingClientRect();
    const dt = new DataTransfer();
    el.dispatchEvent(new DragEvent('dragstart', { bubbles: true, dataTransfer: dt }));
    const x = r.left + r.width * 0.62;
    ray.dispatchEvent(new DragEvent('dragover', { bubbles: true, cancelable: true, dataTransfer: dt, clientX: x }));
    ray.dispatchEvent(new DragEvent('drop', { bubbles: true, cancelable: true, dataTransfer: dt, clientX: x }));
    await new Promise((res) => setTimeout(res, 900));
    return {
      once, sonra: el.style.left,
      saat: (el.querySelector('small') || {}).textContent,
      toast: (document.querySelector('.pn-toast') || {}).textContent || null,
    };
  });
  console.log('sürükle    :', JSON.stringify(drag));

  // 4) Ortak kabuk: ray katlama (admin-ui.js)
  const once = await p.evaluate(() => Math.round(document.querySelector('.admin-side').getBoundingClientRect().width));
  await p.click('#adminRail');
  await new Promise((r) => setTimeout(r, 400));
  const sonra = await p.evaluate(() => Math.round(document.querySelector('.admin-side').getBoundingClientRect().width));
  console.log('ray (ortak):', once, '→', sonra);

  console.log('JS hatası  :', hata.length ? [...new Set(hata)].slice(0, 3).join(' | ') : 'yok');
  await b.close();
})();
