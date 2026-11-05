const puppeteer = require('puppeteer');
const BASE = process.argv[2], SID = process.argv[3], GEN = +(process.argv[4] || 880);
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: GEN, height: 900 });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  for (const yol of ['/admin', '/admin/icerikler', '/admin/ayarlar', '/admin/bot', '/admin/kategoriler', '/admin/dosyalar']) {
    await p.goto(BASE + yol, { waitUntil: 'networkidle2', timeout: 45000 });
    await new Promise((r) => setTimeout(r, 400));
    const o = await p.evaluate((gen) => {
      const doc = document.documentElement;
      const tasan = [];
      if (doc.scrollWidth > doc.clientWidth) {
        for (const el of document.querySelectorAll('body *')) {
          const r = el.getBoundingClientRect();
          if (r.right > gen + 1 && r.width > 40) {
            tasan.push({
              sel: el.tagName.toLowerCase() + (el.id ? '#' + el.id : '') + (el.className && typeof el.className === 'string' ? '.' + el.className.split(/\s+/).slice(0, 2).join('.') : ''),
              sag: Math.round(r.right), w: Math.round(r.width),
            });
          }
        }
      }
      return {
        scroll: doc.scrollWidth, client: doc.clientWidth,
        tasan: tasan.slice(0, 5),
        formKart: (() => { const f = document.querySelector('.admin-body form:has(> table.table)'); return f ? getComputedStyle(f).borderRadius : null; })(),
        menü: getComputedStyle(document.querySelector('.admin-side')).display,
      };
    }, GEN);
    console.log(yol.padEnd(20), `${o.scroll}/${o.client}`, o.scroll > o.client ? 'TAŞMA' : 'ok',
      '· menü', o.menü, '· tablo-kart', o.formKart || '-');
    o.tasan.forEach((t) => console.log('    ↳', t.sel, 'sağ', t.sag, 'gen', t.w));
  }
  await b.close();
})();
