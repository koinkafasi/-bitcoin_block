const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

// Bos liste durumunda colspan, tablo genisligini kapsiyor mu?
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 1000 });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  // Filtreyi imkansiz bir terimle daralt: liste bosalsin.
  await p.goto(BASE + '/admin/icerikler?q=zzzzqqqxyz', { waitUntil: 'networkidle2' });
  await new Promise((r) => setTimeout(r, 700));

  const d = await p.evaluate(() => {
    const t = document.querySelector('#icerik-panel table.table, table.table');
    if (!t) return null;
    const th = t.querySelectorAll('thead th').length;
    const bos = t.querySelector('.bos-durum');
    const td = bos ? bos.closest('td') : null;
    return {
      thSayisi: th,
      colspan: td ? td.getAttribute('colspan') : null,
      tdGenislik: td ? Math.round(td.getBoundingClientRect().width) : null,
      tabloGenislik: Math.round(t.getBoundingClientRect().width),
      // Bos durum kutusu td'nin tamamini kapliyor mu
      bosGenislik: bos ? Math.round(bos.getBoundingClientRect().width) : null,
      // Son sutun bosluğu: tablo genisligi - td genisligi
      eksik: td ? Math.round(t.getBoundingClientRect().width - td.getBoundingClientRect().width) : null,
    };
  });
  console.log(JSON.stringify(d, null, 1));
  await p.screenshot({ path: '/tmp/bos-durum-colspan.png', fullPage: false });
  await b.close();
})();
