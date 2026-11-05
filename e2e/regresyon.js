const puppeteer = require('puppeteer');

const BASE = process.argv[2];
const SID = process.argv[3];

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 1000 });
  const hata = [];
  p.on('pageerror', (e) => hata.push(String(e.message)));
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  const sayfalar = [
    ['/admin/ayarlar', 'ayarlar'],
    ['/admin/marka', 'marka'],
    ['/admin/menu', 'menu'],
    ['/admin/icerikler/yeni', 'yeni'],
    ['/admin/resmi-ilanlar', 'ilanlar'],
  ];
  for (const [yol, ad] of sayfalar) {
    const r = await p.goto(BASE + yol, { waitUntil: 'networkidle2' });
    await new Promise((x) => setTimeout(x, 600));
    await p.screenshot({ path: `/tmp/reg-${ad}.png` });
    // Yatay taşma var mı?
    const tasma = await p.evaluate(() => document.documentElement.scrollWidth - window.innerWidth);
    console.log(ad.padEnd(10), r.status(), '· yatay taşma:', tasma, 'px');
  }
  console.log('JS hatası:', hata.length ? [...new Set(hata)].join(' | ') : 'yok');
  await b.close();
})();
