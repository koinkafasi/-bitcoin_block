const puppeteer = require('puppeteer');

const BASE = process.argv[2];
const SID = process.argv[3];

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 1000 });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });
  await p.goto(BASE + '/admin/icerikler/yeni', { waitUntil: 'networkidle2' });
  await new Promise((r) => setTimeout(r, 700));

  const bilgi = await p.evaluate(() => {
    const lbl = [...document.querySelectorAll('label')].find((l) => /İçerik Türü|Başlık/.test(l.textContent));
    if (!lbl) return 'etiket yok';
    const c = getComputedStyle(lbl);
    const r = lbl.getBoundingClientRect();
    const par = lbl.parentElement;
    const pr = par.getBoundingClientRect();
    const pc = getComputedStyle(par);
    const grand = par.parentElement;
    return {
      etiket: lbl.textContent.trim().slice(0, 20),
      sinif: lbl.className,
      etiketKutu: [Math.round(r.x), Math.round(r.y), Math.round(r.width), Math.round(r.height)],
      etiketStil: { pos: c.position, display: c.display, margin: c.margin, lh: c.lineHeight, fs: c.fontSize, transform: c.transform },
      ebeveyn: par.tagName + '.' + par.className,
      ebeveynKutu: [Math.round(pr.x), Math.round(pr.y), Math.round(pr.width), Math.round(pr.height)],
      ebeveynStil: { pos: pc.position, display: pc.display, overflow: pc.overflow, height: pc.height, padding: pc.padding, margin: pc.margin },
      buyuk: grand ? grand.tagName + '.' + grand.className : null,
    };
  });
  console.log(JSON.stringify(bilgi, null, 1));

  // Üstteki küçük kutu ne?
  const ust = await p.evaluate(() => {
    const els = [...document.querySelectorAll('.admin-main *')].filter((e) => {
      const r = e.getBoundingClientRect();
      return r.top > 100 && r.top < 200 && r.height > 0;
    }).slice(0, 6);
    return els.map((e) => ({
      tag: e.tagName + '.' + e.className,
      kutu: [Math.round(e.getBoundingClientRect().x), Math.round(e.getBoundingClientRect().y), Math.round(e.getBoundingClientRect().width), Math.round(e.getBoundingClientRect().height)],
    }));
  });
  console.log('üst öğeler:', JSON.stringify(ust, null, 1));

  await b.close();
})();
