const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 1100, deviceScaleFactor: 1 });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  const r = await p.goto(BASE + '/admin/manset-editor', { waitUntil: 'networkidle2' });
  await new Promise((x) => setTimeout(x, 1200));

  const d = await p.evaluate(() => {
    const olc = (sec) => {
      const e = document.querySelector(sec);
      if (!e) return null;
      const s = getComputedStyle(e);
      const r = e.getBoundingClientRect();
      return {
        sec,
        display: s.display,
        position: s.position,
        w: Math.round(r.width),
        h: Math.round(r.height),
        x: Math.round(r.x),
        y: Math.round(r.y),
        overflow: s.overflow,
        bg: s.backgroundColor,
      };
    };
    const kurallar = [];
    for (const sheet of document.styleSheets) {
      let kurallar2;
      try { kurallar2 = sheet.cssRules; } catch (e) { continue; }
      for (const k of kurallar2) {
        if (k.selectorText && /\.me-/.test(k.selectorText)) kurallar.push(k.selectorText.slice(0, 60));
      }
    }
    const stiller = Array.from(document.querySelectorAll('style')).map((s) => s.textContent.length);
    return {
      durum: document.readyState,
      meKurallari: kurallar.slice(0, 12),
      meKuralSayisi: kurallar.length,
      styleBloklari: stiller,
      stage: olc('.me-stage'),
      wrap: olc('.me-canvaswrap'),
      canvas: olc('#me-canvas'),
      grid: olc('#me-grid-overlay'),
      panel: olc('.me-shell') || olc('main .card') || olc('main'),
      sayfaYuksekligi: document.body.scrollHeight,
    };
  });
  console.log(JSON.stringify(d, null, 1));
  await p.screenshot({ path: '/tmp/manset-editor.png', fullPage: false });
  await b.close();
})();
