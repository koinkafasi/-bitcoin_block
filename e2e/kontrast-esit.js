const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

// Kontrast=1 cikan oge hangisi? (renk == zemin) ve gercekten gorunur mu?
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1440, height: 1000, deviceScaleFactor: 2 });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });
  await p.goto(BASE + '/admin', { waitUntil: 'networkidle2' });
  await new Promise((r) => setTimeout(r, 900));

  const d = await p.evaluate(() => {
    const out = [];
    const y = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT);
    let n;
    while ((n = y.nextNode())) {
      const t = n.textContent.trim();
      if (!t) continue;
      const el = n.parentElement;
      if (!el) continue;
      const s = getComputedStyle(el);
      let bg = null, c = el;
      while (c) {
        const cs = getComputedStyle(c);
        if (cs.backgroundColor && !/rgba\(0, 0, 0, 0\)/.test(cs.backgroundColor)) { bg = cs.backgroundColor; break; }
        c = c.parentElement;
      }
      if (!bg) continue;
      // kaba esitlik: renk == zemin
      if (s.color.replace(/\s/g, '') === bg.replace(/\s/g, '')) {
        const r = el.getBoundingClientRect();
        out.push({
          metin: t.slice(0, 20), etiket: el.tagName + '.' + el.className.toString().split(' ')[0],
          renk: s.color, bg, boyut: s.fontSize,
          rect: [Math.round(r.x), Math.round(r.y), Math.round(r.width), Math.round(r.height)],
          opaklik: s.opacity, ebeveynOpaklik: getComputedStyle(el.parentElement || el).opacity,
          gorunur: s.visibility + '/' + s.display,
          html: el.outerHTML.slice(0, 120),
        });
      }
    }
    return out;
  });
  console.log(JSON.stringify(d, null, 1));
  await b.close();
})();
