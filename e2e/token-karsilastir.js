const puppeteer = require('puppeteer');

// İki panelin GERÇEK hesaplanmış stillerini yan yana ölç.
// Pacely: canlı demo. Bizim: izole örnek.
const PACELY = 'https://demo.codeandchill.store/pacely/#/overview';
const BIZ = process.argv[2];
const SID = process.argv[3];

const TOPLA = {
  pacely: () => {
    const g = (s) => {
      const e = document.querySelector(s);
      if (!e) return null;
      const c = getComputedStyle(e);
      const r = e.getBoundingClientRect();
      return {
        bg: c.backgroundColor, color: c.color, radius: c.borderRadius,
        font: c.fontSize + '/' + c.fontWeight, ls: c.letterSpacing,
        lh: c.lineHeight, pad: c.padding, shadow: c.boxShadow,
        w: Math.round(r.width), h: Math.round(r.height),
        border: c.borderColor,
      };
    };
    return {
      govde: g('body'),
      kart: g('.card'),
      kpi: g('.kpi'),
      baslik: g('h1'),
      yanMenu: g('.side'),
      ustBar: g('.top'),
      arama: g('.searchpill'),
      navAktif: g('.nav__item.on, .nav__item[aria-current]'),
    };
  },
  biz: () => {
    const g = (s) => {
      const e = document.querySelector(s);
      if (!e) return null;
      const c = getComputedStyle(e);
      const r = e.getBoundingClientRect();
      return {
        bg: c.backgroundColor, color: c.color, radius: c.borderRadius,
        font: c.fontSize + '/' + c.fontWeight, ls: c.letterSpacing,
        lh: c.lineHeight, pad: c.padding, shadow: c.boxShadow,
        w: Math.round(r.width), h: Math.round(r.height),
        border: c.borderColor,
      };
    };
    return {
      govde: g('body'),
      kart: g('.card'),
      kpi: g('.kpi'),
      baslik: g('.admin-body h1, .admin-top h1'),
      yanMenu: g('.admin-side'),
      ustBar: g('.admin-top'),
      arama: g('.search-pill'),
      navAktif: g('.admin-side .menu a.active'),
    };
  },
};

const KISA = (v) => (v === null ? '—' : String(v).replace(/\s+/g, ' ').slice(0, 34));

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 1000 });

  await p.goto(PACELY, { waitUntil: 'networkidle2', timeout: 60000 });
  await new Promise((r) => setTimeout(r, 2500));
  const pac = await p.evaluate(TOPLA.pacely);

  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });
  await p.goto(BIZ + '/admin/kategoriler', { waitUntil: 'networkidle2' });
  await new Promise((r) => setTimeout(r, 800));
  const biz = await p.evaluate(TOPLA.biz);

  const alanlar = ['bg', 'color', 'radius', 'font', 'ls', 'lh', 'pad', 'w', 'h', 'border'];
  for (const og of Object.keys(pac)) {
    if (!pac[og] || !biz[og]) { console.log(`\n${og}: ${pac[og] ? 'bizde yok' : 'pacelyde yok'}`); continue; }
    console.log(`\n=== ${og} ===`);
    for (const a of alanlar) {
      const e = pac[og][a], s = biz[og][a];
      const ayni = String(e) === String(s);
      console.log(`  ${ayni ? '✓' : '·'} ${a.padEnd(7)} pacely ${KISA(e).padEnd(34)} biz ${KISA(s)}`);
    }
  }
  await b.close();
})();
