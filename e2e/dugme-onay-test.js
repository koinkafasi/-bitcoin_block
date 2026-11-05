const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

// Düğme düzeyinde data-confirm: <button type=submit data-confirm="...">
// Önceden hiç onay sorulmuyordu.
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 1000 });
  const hata = [];
  p.on('pageerror', (e) => hata.push(e.message));
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  const istekler = [];
  p.on('request', (r) => { if (r.method() === 'POST') istekler.push(decodeURIComponent(r.url())); });

  await p.goto(BASE + '/admin/gorunum', { waitUntil: 'networkidle2' });
  await new Promise((r) => setTimeout(r, 600));

  const dugme = await p.evaluate(() => {
    const d = document.querySelector('button[data-confirm]');
    return d ? { metin: d.textContent.trim(), onay: d.getAttribute('data-confirm'), formda: !!d.closest('form') } : null;
  });
  console.log('düğme:', JSON.stringify(dugme));
  if (!dugme) { await b.close(); return; }

  await p.click('button[data-confirm]');
  await new Promise((r) => setTimeout(r, 500));

  const modal = await p.evaluate(() => {
    const m = document.querySelector('.ui-modal');
    return m ? { acik: m.classList.contains('open'), metin: (m.querySelector('#uiConfirmText') || {}).textContent } : null;
  });
  console.log('modal:', JSON.stringify(modal));
  console.log('onaydan ÖNCE POST:', istekler.length);

  // Vazgeç: hiçbir istek gitmemeli.
  await p.evaluate(() => document.querySelector('.ui-modal [data-ui-no]').click());
  await new Promise((r) => setTimeout(r, 500));
  console.log('Vazgeç sonrası POST:', istekler.length, istekler.length ? JSON.stringify(istekler) : '(yok — doğru)');

  console.log('JS hatası:', hata.length ? [...new Set(hata)].join(' | ') : 'yok');
  await b.close();
})();
