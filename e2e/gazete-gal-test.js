const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

// Faz 184: gazete panelindeki sablon galerisi sigiyor mu?
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  const hata = [];
  p.on('pageerror', (e) => hata.push(e.message));

  // Isolated DB bos olabilir: hazir sablonlari yukle (idempotent).
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });
  await p.goto(BASE + '/admin/gazete', { waitUntil: 'networkidle2' });
  await p.evaluate(async (base) => {
    await fetch(base + '/admin/gazete/sablonlar/tohum', { method: 'POST', credentials: 'same-origin' });
  }, BASE);

  for (const g of [1500, 1280, 1024, 820]) {
    await p.setViewport({ width: g, height: 1000 });
    const r = await p.goto(BASE + '/admin/gazete', { waitUntil: 'networkidle2' }).catch(() => null);
    await new Promise((x) => setTimeout(x, 700));
    if (!r || r.status() !== 200) { console.log('atlandı', r ? r.status() : ''); await b.close(); return; }

    const d = await p.evaluate(() => {
      const gal = document.querySelector('.gz-gal');
      if (!gal) return { yok: true };
      const kartlar = Array.from(gal.querySelectorAll('.gz-gal__kart'));
      const gr = gal.getBoundingClientRect();
      const ilk = kartlar[0] ? kartlar[0].getBoundingClientRect() : null;
      const ilkY = ilk ? Math.round(ilk.y) : 0;
      const satir1 = kartlar.filter((k) => Math.abs(Math.round(k.getBoundingClientRect().y) - ilkY) < 4).length;
      const tasan = kartlar.filter((k) => k.getBoundingClientRect().right > gr.right + 2).length;
      const thumb = gal.querySelector('.gz-thumb');
      const tr = thumb ? thumb.getBoundingClientRect() : null;

      return {
        kart: kartlar.length,
        sutun: satir1,
        galeriW: Math.round(gr.width),
        kartW: ilk ? Math.round(ilk.width) : 0,
        kartH: ilk ? Math.round(ilk.height) : 0,
        tasanKart: tasan,
        onizlemeVar: !!thumb,
        onizleme: tr ? Math.round(tr.width) + 'x' + Math.round(tr.height) : null,
        onizlemeSayisi: gal.querySelectorAll('.gz-thumb').length,
        butonVar: gal.querySelectorAll('.gz-gal__alt a.btn').length,
        varsayilanRozet: gal.querySelectorAll('.gz-gal__rozet').length,
        yatayTasma: document.documentElement.scrollWidth - window.innerWidth,
      };
    });

    console.log(g + 'px  ' + JSON.stringify(d));
  }
  console.log('JS hatası:', hata.length ? [...new Set(hata)].slice(0, 2).join(' | ') : 'yok');
  await b.close();
})();
