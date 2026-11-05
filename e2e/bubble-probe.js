const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

// .be-bubble gercek durumu: inline stil, data-show, konum
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'], protocolTimeout: 90000 });
  const p = await b.newPage();
  await p.setViewport({ width: 1440, height: 900 });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });
  await p.goto(BASE + '/admin/icerikler/yeni', { waitUntil: 'networkidle2' });
  await new Promise((x) => setTimeout(x, 3200));

  const d = await p.evaluate(() => {
    const el = document.querySelector('.be-bubble');
    if (!el) return { yok: true };
    const s = getComputedStyle(el);
    const r = el.getBoundingClientRect();
    return {
      varMi: true,
      inlineStyle: el.getAttribute('style'),
      dataShow: el.getAttribute('data-show'),
      sinif: el.className,
      konum: s.position,
      display: s.display,
      visibility: s.visibility,
      opacity: s.opacity,
      w: Math.round(r.width),
      h: Math.round(r.height),
      y: Math.round(r.y),
      parent: el.parentElement.tagName,
      inlineProps: Array.from(el.style).slice(0, 10),
    };
  });
  console.log(JSON.stringify(d, null, 1));
  await b.close();
})();
