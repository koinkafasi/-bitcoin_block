const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1400, height: 1000, deviceScaleFactor: 2 });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });
  await p.goto(BASE + '/admin/gazete', { waitUntil: 'networkidle2' });
  await p.evaluate(async (base) => {
    await fetch(base + '/admin/gazete/sablonlar/tohum', { method: 'POST', credentials: 'same-origin' });
  }, BASE);
  await p.goto(BASE + '/admin/gazete', { waitUntil: 'networkidle2' });
  await new Promise((x) => setTimeout(x, 800));

  for (const tema of ['light', 'dark']) {
    await p.evaluate((t) => document.documentElement.setAttribute('data-theme', t), tema);
    await new Promise((x) => setTimeout(x, 350));
    const el = await p.$('.gz-gal');
    if (!el) { console.log(tema, 'galeri yok'); continue; }
    const clip = await el.boundingBox();
    if (clip) {
      await p.screenshot({
        path: `/tmp/gz-gal-${tema}.png`,
        clip: { x: Math.max(0, clip.x - 10), y: Math.max(0, clip.y - 10), width: Math.min(1380, clip.width + 20), height: Math.min(700, clip.height + 20) },
      });
    }
    // koyu temada kontrast
    const kontrast = await p.evaluate(() => {
      const lin = (v) => { v /= 255; return v <= 0.03928 ? v / 12.92 : Math.pow((v + 0.055) / 1.055, 2.4); };
      const pr = (c) => { const m = c.match(/rgba?\((\d+),\s*(\d+),\s*(\d+)(?:,\s*([\d.]+))?\)/); return m ? { r: +m[1], g: +m[2], b: +m[3], a: m[4] === undefined ? 1 : +m[4] } : null; };
      const lum = (c) => 0.2126 * lin(c.r) + 0.7152 * lin(c.g) + 0.0722 * lin(c.b);
      const k = (a, b) => { const A = pr(a), B = pr(b); if (!A || !B) return null; const l1 = lum(A), l2 = lum(B); return (Math.max(l1, l2) + 0.05) / (Math.min(l1, l2) + 0.05); };
      const sonuc = [];
      document.querySelectorAll('.gz-gal__ad, .gz-gal__aciklama, .gz-gal__rozet, .gz-gal__alt .small').forEach((el) => {
        const s = getComputedStyle(el);
        let bg = null, c = el;
        while (c) {
          const cs = getComputedStyle(c);
          if (cs.backgroundColor && !/rgba\(0, 0, 0, 0\)/.test(cs.backgroundColor)) { bg = cs.backgroundColor; break; }
          c = c.parentElement;
        }
        const oran = bg ? k(s.color, bg) : null;
        if (oran !== null && oran < 4.5) sonuc.push({ sinif: el.className, oran: Math.round(oran * 10) / 10 });
      });
      return sonuc;
    });
    console.log(tema, 'ekran alindi · düşük kontrast:', kontrast.length ? JSON.stringify(kontrast) : 'yok');
  }
  await b.close();
})();
