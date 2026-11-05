const puppeteer = require('puppeteer');

const BASE = process.argv[2];
const SID = process.argv[3];

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 1000 });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });
  await p.goto(BASE + '/admin/icerikler/yeni', { waitUntil: 'networkidle2' });
  await new Promise((r) => setTimeout(r, 1500));

  const d = await p.evaluate(() => {
    return [...document.querySelectorAll('svg.ti')]
      .filter((s) => {
        const r = s.getBoundingClientRect();
        return r.width === 0 || r.height === 0;
      })
      .map((s) => {
        const c = getComputedStyle(s);
        const par = s.parentElement;
        const pc = getComputedStyle(par);
        const gp = par.parentElement;
        const gc = gp ? getComputedStyle(gp) : null;
        return {
          ad: (s.querySelector('use') || {}).getAttribute
            ? (s.querySelector('use').getAttribute('href') || '').split('#')[1]
            : '?',
          kendi: { w: c.width, h: c.height, d: c.display },
          ebeveyn: par.tagName + '.' + par.className + ' {d:' + pc.display + ' w:' + pc.width + '}',
          buyuk: gp ? gp.tagName + '.' + gp.className + ' {d:' + gc.display + '}' : null,
          offsetParent: s.offsetParent ? s.offsetParent.tagName + '.' + s.offsetParent.className : null,
        };
      });
  });
  console.log(JSON.stringify(d, null, 1));
  await b.close();
})();
