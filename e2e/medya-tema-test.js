const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'], protocolTimeout: 90000 });
  const p = await b.newPage();
  await p.setViewport({ width: 1400, height: 1000, deviceScaleFactor: 2 });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });
  await p.goto(BASE + '/admin/dosyalar', { waitUntil: 'networkidle2' });
  await new Promise((x) => setTimeout(x, 1100));

  const grid = await p.$('#medya-panel .media-grid');
  const clip = await grid.boundingBox();
  const kirp = (h) => ({ x: Math.max(0, clip.x - 8), y: Math.max(0, clip.y - 8), width: Math.min(1380, clip.width + 16), height: Math.min(h, clip.height) });

  await p.screenshot({ path: '/tmp/mg-light.png', clip: kirp(760) });

  await p.evaluate(() => document.documentElement.setAttribute('data-theme', 'dark'));
  await new Promise((x) => setTimeout(x, 400));
  await p.screenshot({ path: '/tmp/mg-dark.png', clip: kirp(760) });

  // Liste gorunumu regresyonu
  await p.evaluate(() => document.documentElement.setAttribute('data-theme', 'light'));
  await p.click('[data-media-view="list"]');
  await new Promise((x) => setTimeout(x, 400));
  const liste = await p.evaluate(() => {
    const item = document.querySelector('#medya-panel .media-item');
    const r = item.getBoundingClientRect();
    const img = item.querySelector('img').getBoundingClientRect();
    const body = item.querySelector('.mi-body');
    const cap = item.querySelector('.mi-cap');
    return {
      panelListe: document.getElementById('medya-panel').classList.contains('media-list-view'),
      itemW: Math.round(r.width),
      itemH: Math.round(r.height),
      imgW: Math.round(img.width),
      imgH: Math.round(img.height),
      bodyGorunur: body ? getComputedStyle(body).display !== 'none' : null,
      capGorunur: cap ? getComputedStyle(cap).display !== 'none' : null,
      span: item.style.gridRowEnd,
    };
  });
  console.log('LISTE:', JSON.stringify(liste, null, 1));

  // Koyu tema kontrast: kart alt yazisi
  await p.evaluate(() => document.documentElement.setAttribute('data-theme', 'dark'));
  await p.click('[data-media-view="grid"]');
  await new Promise((x) => setTimeout(x, 400));
  const kontrast = await p.evaluate(() => {
    const lin = (v) => { v /= 255; return v <= 0.03928 ? v / 12.92 : Math.pow((v + 0.055) / 1.055, 2.4); };
    const pr = (c) => { const m = c.match(/rgba?\(([\d.]+),\s*([\d.]+),\s*([\d.]+)(?:,\s*([\d.]+))?\)/); return m ? { r: +m[1], g: +m[2], b: +m[3] } : null; };
    const lum = (c) => 0.2126 * lin(c.r) + 0.7152 * lin(c.g) + 0.0722 * lin(c.b);
    const k = (a, b) => { const A = pr(a), B = pr(b); if (!A || !B) return null; const l1 = lum(A), l2 = lum(B); return (Math.max(l1, l2) + 0.05) / (Math.min(l1, l2) + 0.05); };
    const cap = document.querySelector('.mi-cap b');
    const s = getComputedStyle(cap);
    // alt yazi gradyan uzerinde: en kotu durum gradyanin acik ucu
    return { renk: s.color, gradyanZemin: 'rgba(12,14,16,.72) uzerinde', oran: Math.round(k(s.color, 'rgb(12,14,16)') * 10) / 10 };
  });
  console.log('KOYU KAPAK KONTRAST:', JSON.stringify(kontrast));
  await b.close();
})();
