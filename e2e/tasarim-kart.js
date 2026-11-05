const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 1000 });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });
  await p.goto(BASE + '/admin/tasarim', { waitUntil: 'networkidle2' });
  await new Promise((x) => setTimeout(x, 900));

  const d = await p.evaluate(() => {
    const kart = document.querySelector('.admin-body > .card');
    const cs = getComputedStyle(kart);
    const r = kart.getBoundingClientRect();
    const cocuklar = Array.from(kart.childNodes).map((n) => {
      if (n.nodeType === 3) return { tur: 'TEXT', icerik: JSON.stringify(n.textContent.slice(0, 20)) };
      const rr = n.getBoundingClientRect();
      const ncs = getComputedStyle(n);
      return {
        tur: n.tagName, cls: n.className.toString().slice(0, 24),
        h: Math.round(rr.height), y: Math.round(rr.y - r.y),
        display: ncs.display, mt: ncs.marginTop, mb: ncs.marginBottom,
      };
    });
    // ic ice: ilk cocuk div'in cocuklari
    const ilk = kart.children[0];
    const ic = ilk ? Array.from(ilk.children).map((n) => {
      const rr = n.getBoundingClientRect();
      const ncs = getComputedStyle(n);
      return { tur: n.tagName, cls: n.className.toString().slice(0, 20), h: Math.round(rr.height), lh: ncs.lineHeight, fs: ncs.fontSize, mb: ncs.marginBottom };
    }) : [];
    return {
      kart: { h: Math.round(r.height), w: Math.round(r.width), pad: cs.padding, lh: cs.lineHeight, disp: cs.display, minH: cs.minHeight, scrollH: kart.scrollHeight },
      cocuklar, ilkCocukH: ilk ? Math.round(ilk.getBoundingClientRect().height) : 0, ilkIci: ic,
    };
  });
  console.log(JSON.stringify(d, null, 1));
  await b.close();
})();
