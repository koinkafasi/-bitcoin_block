const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

// CANLI: gazete "Sil" düğmesi. Silmeyi ONAYLAMIYORUZ (Vazgeç) — sadece
// akışın doğru olduğunu ölçüyoruz: modal çıkıyor, POST henüz gitmiyor,
// vazgeçince hiçbir istek gitmiyor.
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 1000 });
  const hata = [];
  p.on('pageerror', (e) => hata.push(e.message));
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  const istekler = [];
  p.on('request', (r) => { if (r.method() === 'POST') istekler.push(decodeURIComponent(r.url())); });
  const diyalog = [];
  p.on('dialog', async (d) => { diyalog.push(d.message().slice(0, 50)); await d.dismiss(); });

  const r = await p.goto(BASE + '/admin/gazete', { waitUntil: 'networkidle2' });
  console.log('gazete paneli:', r.status());
  await new Promise((x) => setTimeout(x, 700));

  const f = await p.evaluate(() => {
    const x = [...document.querySelectorAll('form[data-confirm]')]
      .find((y) => (y.getAttribute('data-confirm') || '').includes('Sayı silinsin mi'));
    return x ? { action: x.action, onay: x.getAttribute('data-confirm'), metot: x.method } : null;
  });
  console.log('sil formu:', JSON.stringify(f));
  if (!f) { console.log('form yok — dur'); await b.close(); return; }

  await p.evaluate(() => {
    [...document.querySelectorAll('form[data-confirm]')]
      .find((y) => (y.getAttribute('data-confirm') || '').includes('Sayı silinsin mi'))
      .querySelector('button').click();
  });
  await new Promise((x) => setTimeout(x, 600));

  const modal = await p.evaluate(() => {
    const m = document.querySelector('.ui-modal');
    return m ? {
      acik: m.classList.contains('open'),
      metin: (m.querySelector('#uiConfirmText') || {}).textContent,
      gorunur: (m.querySelector('.ui-box') || { getBoundingClientRect: () => ({ height: 0 }) })
        .getBoundingClientRect().height > 0,
    } : null;
  });
  console.log('modal:', JSON.stringify(modal));
  console.log('native onay ciktı mı:', diyalog.length > 0);
  console.log('onaydan ONCE POST:', istekler.length);

  await p.evaluate(() => document.querySelector('.ui-modal [data-ui-no]').click());
  await new Promise((x) => setTimeout(x, 700));
  console.log('Vazgec sonrasi POST:', istekler.length, istekler.length ? JSON.stringify(istekler) : '(yok — dogru)');
  console.log('bozuk adres var mi:', istekler.filter((u) => u.includes('silinsin')).length);

  await p.screenshot({ path: '/tmp/canli-gazete-sil.png' });
  console.log('JS hatasi:', hata.length ? [...new Set(hata)].join(' | ') : 'yok');
  await b.close();
})();
