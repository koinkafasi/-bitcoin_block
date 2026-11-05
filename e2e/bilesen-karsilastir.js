const puppeteer = require('puppeteer');

// Kalan bileşenleri yan yana ölç: KPI, tablo başlığı, düğme, rozet, sayfa başlığı.
const PACELY = 'https://demo.codeandchill.store/pacely/#/overview';
const BIZ = process.argv[2];
const SID = process.argv[3];

const stil = (e) => {
  if (!e) return null;
  const c = getComputedStyle(e);
  const r = e.getBoundingClientRect();
  return {
    bg: c.backgroundColor,
    color: c.color,
    radius: c.borderRadius,
    font: c.fontSize + '/' + c.fontWeight,
    lh: c.lineHeight,
    pad: c.padding,
    shadow: c.boxShadow.slice(0, 60),
    border: c.borderColor + ' ' + c.borderWidth,
    w: Math.round(r.width),
    h: Math.round(r.height),
  };
};

const KISA = (v) => (v === null ? '—' : String(v).replace(/\s+/g, ' ').slice(0, 38));

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 1000 });

  await p.goto(PACELY, { waitUntil: 'networkidle2', timeout: 60000 });
  await new Promise((r) => setTimeout(r, 2500));
  const pac = await p.evaluate((fn) => {
    const s = eval('(' + fn + ')');
    return {
      kpi: s(document.querySelector('.kpi')),
      kpiEtiket: s(document.querySelector('.kpi .lbl, .kpi small, .kpi .t')),
      tabloBaslik: s(document.querySelector('table thead th')),
      tabloHucre: s(document.querySelector('table tbody td')),
      dugme: s(document.querySelector('.btn')),
      dugmeBirincil: s(document.querySelector('.btn.primary, .btn--primary, .btn.pri')),
      rozet: s(document.querySelector('.pill, .badge, .tag')),
      sayfaBaslik: s(document.querySelector('.phead h1, h1')),
      navAktif: s(document.querySelector('.nav__item.on, .nav .on, a.on')),
      navOge: s(document.querySelector('.nav__item, .nav a')),
      kartBaslik: s(document.querySelector('.card h3, .card .h')),
      girdi: s(document.querySelector('input[type=text], input:not([type])')),
    };
  }, stil.toString());

  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });
  await p.goto(BIZ + '/admin/icerikler', { waitUntil: 'networkidle2' });
  await new Promise((r) => setTimeout(r, 800));
  const biz = await p.evaluate((fn) => {
    const s = eval('(' + fn + ')');
    return {
      kpi: s(document.querySelector('.kpi')),
      kpiEtiket: s(document.querySelector('.kpi .kpi-l, .kpi small, .kpi .lbl')),
      tabloBaslik: s(document.querySelector('table thead th')),
      tabloHucre: s(document.querySelector('table tbody td')),
      dugme: s(document.querySelector('.btn')),
      dugmeBirincil: s(document.querySelector('.btn.primary, .btn--primary, .btn.pri')),
      rozet: s(document.querySelector('.badge, .pill, .tag')),
      sayfaBaslik: s(document.querySelector('.admin-top h1, .admin-body h1')),
      navAktif: s(document.querySelector('.admin-side .menu a.active')),
      navOge: s(document.querySelector('.admin-side .menu a')),
      kartBaslik: s(document.querySelector('.card h3, .card .card-h')),
      girdi: s(document.querySelector('input[type=text], input:not([type])')),
    };
  }, stil.toString());

  const alanlar = ['bg', 'color', 'radius', 'font', 'lh', 'pad', 'w', 'h', 'border', 'shadow'];
  for (const og of Object.keys(pac)) {
    console.log('\n=== ' + og + ' ===');
    if (!pac[og] || !biz[og]) {
      console.log('  pacely:', !!pac[og], '· biz:', !!biz[og]);
      continue;
    }
    for (const a of alanlar) {
      const e = pac[og][a], s = biz[og][a];
      const ayni = String(e) === String(s);
      if (ayni) console.log('  = ' + a.padEnd(7) + KISA(e));
      else console.log('  ! ' + a.padEnd(7) + 'pacely ' + KISA(e).padEnd(40) + 'biz ' + KISA(s));
    }
  }
  await b.close();
})();
