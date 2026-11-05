const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 1000, deviceScaleFactor: 2 });
  const hata = [];
  p.on('pageerror', (e) => hata.push(e.message));
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  await p.goto(BASE + '/admin/tasarim', { waitUntil: 'networkidle2' });
  await new Promise((r) => setTimeout(r, 1200));

  const d = await p.evaluate(() => {
    const cipler = [...document.querySelectorAll('.bp-chip, .bld-block')];
    const ilk = cipler[0];
    const butonlar = ilk ? [...ilk.querySelectorAll('button[data-act]')] : [];
    return {
      cipSayisi: cipler.length,
      ilkCip: ilk ? ilk.className : null,
      tutamac: ilk ? (() => { const g = ilk.querySelector('.grip'); const s = g && g.querySelector('svg.ti'); return { varMi: !!s, ad: s ? (s.querySelector('use').getAttribute('href') || '').split('#')[1] : null, kutu: s ? [Math.round(s.getBoundingClientRect().width), Math.round(s.getBoundingClientRect().height)] : null }; })() : null,
      dugmeler: butonlar.map((b) => ({ act: b.dataset.act, svg: !!b.querySelector('svg.ti'), ad: b.querySelector('svg.ti') ? (b.querySelector('use').getAttribute('href') || '').split('#')[1] : null, metin: b.textContent.trim().slice(0, 12) })),
    };
  });
  console.log(JSON.stringify(d, null, 1));
  await p.screenshot({ path: '/tmp/tasarim-son.png', fullPage: false });
  console.log('JS hatası:', hata.length ? [...new Set(hata)].join(' | ') : 'yok');
  await b.close();
})();
