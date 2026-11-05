const puppeteer = require('puppeteer');
const BASE = process.argv[2], SID = process.argv[3];
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 1000 });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  for (const yol of ['/admin/icerikler', '/admin/aboneler', '/admin/kategoriler', '/admin/bot']) {
    await p.goto(BASE + yol, { waitUntil: 'networkidle2', timeout: 45000 });
    await new Promise((r) => setTimeout(r, 400));
    const o = await p.evaluate(() => {
      const f = document.querySelector('.admin-body > form:has(> table.table)');
      const t = document.querySelector('.table');
      const th = document.querySelector('.table th');
      const kutu = document.querySelector('.admin-body');
      return {
        formKart: f ? { bg: getComputedStyle(f).backgroundColor, radius: getComputedStyle(f).borderRadius, pad: getComputedStyle(f).padding } : null,
        tabloGenislik: t ? Math.round(t.getBoundingClientRect().width) : 0,
        govdeGenislik: kutu ? Math.round(kutu.getBoundingClientRect().width) : 0,
        th: th ? { size: getComputedStyle(th).fontSize, weight: getComputedStyle(th).fontWeight, tt: getComputedStyle(th).textTransform } : null,
        yatay: document.documentElement.scrollWidth > document.documentElement.clientWidth,
        font: getComputedStyle(document.body).fontFamily.split(',')[0],
        kart: (() => { const c = document.querySelector('.card'); return c ? { bg: getComputedStyle(c).backgroundColor, radius: getComputedStyle(c).borderRadius } : null; })(),
        tab: document.querySelectorAll('.tabs a, .tabs button').length,
        tabAktif: (() => { const a = document.querySelector('.tabs a.active, .tabs button.active'); return a ? getComputedStyle(a).backgroundColor : null; })(),
      };
    });
    console.log(yol);
    console.log('  form-kart :', JSON.stringify(o.formKart));
    console.log('  tablo/gövde:', o.tabloGenislik, '/', o.govdeGenislik, o.yatay ? '⚠ TAŞMA' : 'ok');
    console.log('  th        :', JSON.stringify(o.th), '· font', o.font);
    console.log('  kart      :', JSON.stringify(o.kart), '· sekme', o.tab, 'aktif bg', o.tabAktif);
  }
  await b.close();
})();
