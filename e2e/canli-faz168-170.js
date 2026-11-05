const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

const EMOJI = /[\u{1F000}-\u{1FAFF}\u{2600}-\u{27BF}\u{2B00}-\u{2BFF}\u{2300}-\u{23FF}\u{25A0}-\u{25FF}\u{2800}-\u{28FF}]/u;

const SAYFALAR = [
  '/admin/pano', '/admin/icerikler', '/admin/kategoriler', '/admin/etiketler',
  '/admin/ceviriler', '/admin/sinemalar', '/admin/resmi-ilanlar', '/admin/dosyalar',
  '/admin/yorumlar', '/admin/members', '/admin/aboneler', '/admin/bulten',
  '/admin/ayarlar', '/admin/gorunum', '/admin/tasarim', '/admin/seo',
  '/admin/servisler', '/admin/iletisim', '/admin/geri-bildirim', '/admin/gazete',
  '/admin/gazeteler', '/admin/anketler', '/admin/etkinlikler', '/admin/bilesenler',
  '/admin/menu', '/admin/yardim', '/admin/gelen-kutusu', '/admin/raporlar',
  '/admin/paylasim', '/admin/backlink', '/admin/tab-kategorileri',
];

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 1000 });
  const hata = [];
  p.on('pageerror', (e) => hata.push(e.message));
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  let emoji = 0, bosIkon = 0, tasan = 0, olmayan = [];
  for (const yol of SAYFALAR) {
    const r = await p.goto(BASE + yol, { waitUntil: 'networkidle2' }).catch(() => null);
    await new Promise((x) => setTimeout(x, 300));
    if (!r || r.status() !== 200) { olmayan.push(yol + ' ' + (r ? r.status() : 'yok')); continue; }
    const d = await p.evaluate((src) => {
      const re = new RegExp(src, 'u');
      const govde = document.querySelector('.admin-body') || document.body;
      let em = 0;
      const y = document.createTreeWalker(govde, NodeFilter.SHOW_TEXT);
      while (y.nextNode()) {
        const t = y.currentNode.nodeValue || '';
        const par = y.currentNode.parentElement;
        if (re.test(t) && par && par.getClientRects().length) em++;
      }
      // Sembolü sprite'ta bulunmayan ya da boyutsuz ikonlar.
      const bos = [...govde.querySelectorAll('svg.ti')].filter((s) => {
        const k = s.getBoundingClientRect();
        return s.getClientRects().length && k.width === 0;
      }).length;
      return { em, bos, tasma: document.documentElement.scrollWidth - document.documentElement.clientWidth };
    }, EMOJI.source);
    emoji += d.em; bosIkon += d.bos;
    if (d.tasma > 2) { tasan++; console.log('  TAŞMA', yol, d.tasma); }
  }
  console.log('sayfa:', SAYFALAR.length - olmayan.length, '/', SAYFALAR.length);
  console.log('UI emoji:', emoji, '· boyutsuz ikon:', bosIkon, '· taşan sayfa:', tasan);
  console.log('erişilemeyen:', olmayan.length ? olmayan.join(', ') : 'yok');
  console.log('JS hatası:', hata.length ? [...new Set(hata)].slice(0, 3).join(' | ') : 'yok');

  // Yeni menü canlıda
  await p.goto(BASE + '/admin/icerikler', { waitUntil: 'networkidle2' });
  await p.click('.new-dd > summary');
  await new Promise((r) => setTimeout(r, 300));
  const menu = await p.evaluate(() => ({
    acik: document.querySelector('.new-dd').open,
    oge: [...document.querySelectorAll('.new-dd__menu b')].map((x) => x.textContent),
    ikon: [...document.querySelectorAll('.new-dd__menu svg.ti')].filter((s) => s.getBoundingClientRect().width > 0).length,
  }));
  console.log('Yeni menüsü:', JSON.stringify(menu));
  await b.close();
})();
