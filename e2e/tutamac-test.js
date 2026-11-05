const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 1000 });
  const hata = [];
  p.on('pageerror', (e) => hata.push(e.message));
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  for (const yol of ['/admin/menu', '/admin/etiketler', '/admin/tab-kategorileri', '/admin/kategoriler', '/admin/ceviriler', '/admin/bulten', '/admin/gazeteler', '/admin/tasarim']) {
    const r = await p.goto(BASE + yol, { waitUntil: 'networkidle2' });
    await new Promise((x) => setTimeout(x, 700));
    const d = await p.evaluate(() => {
      const tutamaclar = [...document.querySelectorAll('span[title="Sürükle"], .m-drag, .grip')];
      return {
        sayi: tutamaclar.length,
        svg: tutamaclar.filter((s) => s.querySelector('svg.ti')).length,
        sifir: tutamaclar.filter((s) => {
          const v = s.querySelector('svg.ti');
          return v && v.getBoundingClientRect().width === 0;
        }).length,
        ornek: tutamaclar[0] ? (tutamaclar[0].querySelector('use') || {}).getAttribute?.('href') : null,
      };
    });
    console.log(yol.padEnd(16), r.status(), JSON.stringify(d));
  }
  console.log('JS hatası:', hata.length ? [...new Set(hata)].join(' | ') : 'yok');
  await b.close();
})();
