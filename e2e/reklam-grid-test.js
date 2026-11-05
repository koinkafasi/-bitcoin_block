const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

// /admin/reklamlar tarih alanlari 1068 px olculdu; komsulari ~170 px.
// Izgara sutunlari gercekten esit mi, alan tasiyor mu?
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1440, height: 1100 });
  const hata = [];
  p.on('pageerror', (e) => hata.push(e.message));
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  await p.goto(BASE + '/admin/reklamlar', { waitUntil: 'networkidle2' });
  await new Promise((r) => setTimeout(r, 600));

  const d = await p.evaluate(() => {
    const out = [];
    document.querySelectorAll('input[type=date]').forEach((el, i) => {
      if (i > 3) return;
      const kap = el.parentElement;
      const ks = getComputedStyle(kap);
      const r = el.getBoundingClientRect();
      const kr = kap.getBoundingClientRect();
      out.push({
        name: el.getAttribute('name'),
        alanGenislik: Math.round(r.width),
        kapGenislik: Math.round(kr.width),
        kapDisplay: ks.display,
        kolonlar: ks.gridTemplateColumns,
        sutunSayisi: ks.gridTemplateColumns.split(' ').length,
        // Komsu alanlarin genisligi
        komsular: Array.from(kap.children).slice(0, 8).map((c) => Math.round(c.getBoundingClientRect().width)),
      });
    });
    // Sayfada kac reklam karti var
    return { alanlar: out, tarihSayisi: document.querySelectorAll('input[type=date]').length };
  });
  console.log(JSON.stringify(d, null, 1));
  console.log('JS hatası:', hata.length ? hata[0] : 'yok');
  await b.close();
})();
