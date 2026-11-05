const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1440, height: 1000 });
  const hata = [];
  p.on('pageerror', (e) => hata.push(e.message));
  const istekler = [];
  p.on('request', (r) => {
    const u = r.url();
    if (u.includes('/admin/tasarim/') || u.includes('/admin/arka-plan/')) {
      istekler.push(r.method() + ' ' + u.replace(BASE, ''));
    }
  });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });
  await p.goto(BASE + '/admin/tasarim', { waitUntil: 'networkidle2' });
  await new Promise((x) => setTimeout(x, 1200));

  const d = await p.evaluate(() => {
    const sonuc = {};

    // 1) Uc panel kapali mi?
    sonuc.kapali = ['plPanel', 'bpTokens'].map((id) => {
      const el = document.getElementById(id);
      return { id, var: !!el, tur: el ? el.tagName : '', open: el ? el.open : null };
    });

    // 2) Baslik + dugme ayni satirda mi
    const kart = document.querySelector('.admin-body > .card');
    const c = Array.from(kart.children).map((el) => {
      const r = el.getBoundingClientRect();
      return { y: Math.round(r.y), h: Math.round(r.height) };
    });
    sonuc.ustKartAyniSatir = c.length === 2 && Math.abs(c[0].y - c[1].y) < 4;
    sonuc.ustKartH = Math.round(kart.getBoundingClientRect().height);

    // 3) plPanel acilinca icerik gorunur mu
    const pl = document.getElementById('plPanel');
    pl.open = true;
    const grid = pl.querySelector('.ly-grid');
    const gr = grid.getBoundingClientRect();
    sonuc.plAcilH = Math.round(pl.getBoundingClientRect().height);
    sonuc.plGridGorunur = gr.height > 40;
    sonuc.plButonGorunur = (() => {
      const btn = document.getElementById('plReset');
      if (!btn) return false;
      const r = btn.getBoundingClientRect();
      return r.width > 10 && r.height > 6 && r.bottom <= window.innerHeight + 5000;
    })();

    // 4) bpTokens ayni
    const bt = document.getElementById('bpTokens');
    bt.open = true;
    sonuc.tokensAcilH = Math.round(bt.getBoundingClientRect().height);

    // 5) Ilk kontroller erisilebilir ad tasiyor mu
    sonuc.adsiz = Array.from(pl.querySelectorAll('input, select')).filter((el) => {
      const id = el.id;
      const lbl = id ? document.querySelector('label[for="' + CSS.escape(id) + '"]') : null;
      return !el.getAttribute('aria-label') && !(lbl && lbl.textContent.trim());
    }).map((el) => el.id || el.tagName);

    return sonuc;
  });
  console.log(JSON.stringify(d, null, 1));

  // 6) plReset gercekten istek atiyor mu
  await p.click('#plReset');
  await new Promise((x) => setTimeout(x, 900));
  console.log('istekler:', istekler.length ? istekler : 'yok');
  console.log('JS hatası:', hata.length ? [...new Set(hata)].slice(0, 2).join(' | ') : 'yok');
  await b.close();
})();
