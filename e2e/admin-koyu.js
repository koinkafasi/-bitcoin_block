const puppeteer = require('puppeteer');
const BASE = process.argv[2], SID = process.argv[3];
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 1000 });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });
  await p.goto(BASE + '/admin/icerikler', { waitUntil: 'networkidle2' });
  // tema anahtarı düz metin ('dark'), JSON değil
  await p.evaluate(() => localStorage.setItem('tb_theme', 'dark'));
  await p.reload({ waitUntil: 'networkidle2' });
  await new Promise((r) => setTimeout(r, 500));
  const o = await p.evaluate(() => {
    const bg = (s) => { const e = document.querySelector(s); return e ? getComputedStyle(e).backgroundColor : null; };
    const fg = (s) => { const e = document.querySelector(s); return e ? getComputedStyle(e).color : null; };
    const th = document.querySelector('.table th');
    const kontrast = (a, c) => {
      if (!a || !c) return null;
      const p = (x) => { const mm = x.match(/\d+(\.\d+)?/g); if (!mm) return 0; const m = mm.slice(0, 3).map(Number).map((v) => { v /= 255; return v <= .03928 ? v / 12.92 : Math.pow((v + .055) / 1.055, 2.4); }); return .2126 * m[0] + .7152 * m[1] + .0722 * m[2]; };
      const l1 = p(a), l2 = p(c); const hi = Math.max(l1, l2), lo = Math.min(l1, l2);
      return ((hi + .05) / (lo + .05)).toFixed(1);
    };
    return {
      tema: document.documentElement.getAttribute('data-theme'),
      govde: bg('body'), govdeMetin: fg('body'),
      side: bg('.admin-side'), kart: bg('.card'), formKart: bg('.admin-body form:has(> table.table)'),
      input: bg('.input'), th: th ? getComputedStyle(th).color : null,
      thBg: th ? getComputedStyle(th).backgroundColor : null,
      kontrastGovde: kontrast(bg('body'), fg('body')),
      kontrastKart: (() => { const c = document.querySelector('.card'); return c ? kontrast(getComputedStyle(c).backgroundColor, getComputedStyle(c).color) : null; })(),
      kontrastTh: th ? kontrast(getComputedStyle(th).backgroundColor === 'rgba(0, 0, 0, 0)' ? bg('.card') : getComputedStyle(th).backgroundColor, getComputedStyle(th).color) : null,
      tasma: document.documentElement.scrollWidth > document.documentElement.clientWidth + 1,
    };
  });
  console.log(JSON.stringify(o, null, 1));
  await p.screenshot({ path: '/tmp/admin-skin/koyu.png' });
  await p.evaluate(() => localStorage.setItem('tb_theme', 'light'));
  await b.close();
})();
