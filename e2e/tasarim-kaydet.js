const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'], protocolTimeout: 60000 });
  const p = await b.newPage();
  await p.setViewport({ width: 1440, height: 1000 });
  const hata = [];
  p.on('pageerror', (e) => hata.push(e.message));
  // 'varsayılana dön' confirm() açar; testte onayla.
  p.on('dialog', (d) => d.accept());
  const istekler = [];
  p.on('request', (r) => {
    const u = r.url();
    if (u.includes('/admin/tasarim/') || u.includes('/admin/arka-plan/')) istekler.push(r.method() + ' ' + u.replace(BASE, ''));
  });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });
  await p.goto(BASE + '/admin/tasarim', { waitUntil: 'networkidle2' });
  await new Promise((x) => setTimeout(x, 1300));

  const sonuc = await p.evaluate(async () => {
    const out = {};
    const pl = document.getElementById('plPanel');
    pl.open = true;
    await new Promise((r) => setTimeout(r, 200));

    // Kontrolu degistir -> otomatik kayit tetiklenmeli
    const sw = document.getElementById('plSideWidth');
    if (sw) {
      sw.value = '40';
      sw.dispatchEvent(new Event('change', { bubbles: true }));
      sw.dispatchEvent(new Event('input', { bubbles: true }));
    }
    await new Promise((r) => setTimeout(r, 900));
    out.status = (document.getElementById('plStatus') || {}).textContent || '';

    // Reset dugmesi
    const btn = document.getElementById('plReset');
    out.resetVar = !!btn;
    if (btn) btn.click();
    await new Promise((r) => setTimeout(r, 900));
    out.sonraStatus = (document.getElementById('plStatus') || {}).textContent || '';
    return out;
  });

  console.log(JSON.stringify(sonuc, null, 1));
  console.log('istekler:', istekler.length ? istekler : 'yok');
  console.log('JS hatası:', hata.length ? [...new Set(hata)].slice(0, 2).join(' | ') : 'yok');
  await b.close();
})();
