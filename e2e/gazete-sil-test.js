const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

// Gazete "Sil": (1) onay modali çıkıyor mu (2) onaylayınca GERÇEK URL'e
// POST gidiyor mu — /admin/Sayı silinsin mi? değil.
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 1000 });
  const hata = [];
  p.on('pageerror', (e) => hata.push(e.message));
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  await p.goto(BASE + '/admin/gazete', { waitUntil: 'networkidle2' });
  await new Promise((r) => setTimeout(r, 600));

  const liste = await p.evaluate(() => {
    const f = [...document.querySelectorAll('form[data-confirm]')].find((x) =>
      x.getAttribute('data-confirm').includes('Sayı silinsin mi'));
    if (!f) return null;
    return { action: f.action, onay: f.getAttribute('data-confirm'), dugme: !!f.querySelector('button') };
  });
  console.log('sil formu:', JSON.stringify(liste));
  if (!liste) { console.log('FORM YOK — dur'); await b.close(); return; }

  // Giden istekleri yakala.
  const istekler = [];
  p.on('request', (r) => { if (r.method() === 'POST') istekler.push(r.url()); });

  // Sil'e bas.
  await p.evaluate(() => {
    const f = [...document.querySelectorAll('form[data-confirm]')].find((x) =>
      x.getAttribute('data-confirm').includes('Sayı silinsin mi'));
    f.querySelector('button').click();
  });
  await new Promise((r) => setTimeout(r, 500));

  const modal = await p.evaluate(() => {
    const m = document.querySelector('.ui-modal');
    if (!m) return { varMi: false };
    const acik = m.classList.contains('open');
    const kutu = m.querySelector('.ui-box');
    return {
      varMi: true,
      acik,
      metin: (m.querySelector('#uiConfirmText') || {}).textContent,
      // Görünür mü (open sınıfı gerçekten ekrana getiriyor mu)
      gorunur: kutu ? kutu.getBoundingClientRect().height > 0 : false,
    };
  });
  console.log('onay modali:', JSON.stringify(modal));
  await p.screenshot({ path: '/tmp/gazete-sil-modal.png' });

  // Onayla: "Evet" düğmesi.
  const tiklandi = await p.evaluate(() => {
    const m = document.querySelector('.ui-modal');
    if (!m) return 'modal yok';
    const evet = m.querySelector('[data-ui-yes]');
    if (!evet) return 'Evet dugmesi yok';
    evet.click();
    return 'tiklandi';
  });
  console.log('onaylama:', tiklandi);
  await new Promise((r) => setTimeout(r, 900));

  console.log('POST edilen URL\'ler:', JSON.stringify(istekler));
  const kotu = istekler.filter((u) => decodeURIComponent(u).includes('silinsin'));
  console.log('HATALI URL var mı:', kotu.length ? JSON.stringify(kotu) : 'yok');
  console.log('JS hatası:', hata.length ? [...new Set(hata)].join(' | ') : 'yok');
  await b.close();
})();
