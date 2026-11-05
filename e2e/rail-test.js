const puppeteer = require('puppeteer');
const BASE = process.argv[2] || 'http://127.0.0.1:8070';
const SID = process.argv[3];

// Ray düğmesi: her genişlikte İKİ YÖNDE de çalışmalı.
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  const hata = [];
  p.on('pageerror', (e) => hata.push(String(e.message)));
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  const olc = () => p.evaluate(() => ({
    side: Math.round(document.querySelector('.admin-side').getBoundingClientRect().width),
    dar: document.documentElement.classList.contains('admin-rail'),
    kayit: localStorage.getItem('pn:rail'),
    etiketGorunur: (() => {
      const s = document.querySelector('.admin-side .menu a .ti');
      const a = document.querySelector('.admin-side .menu a');
      if (!a) return null;
      // öğenin font-size'ı 0 ise etiket gizli
      return getComputedStyle(a).fontSize !== '0px';
    })(),
  }));

  for (const gen of [1600, 1440, 1280, 1200, 1024, 960]) {
    await p.setViewport({ width: gen, height: 950 });
    await p.goto(BASE + '/admin/icerikler', { waitUntil: 'networkidle2', timeout: 45000 });
    await p.evaluate(() => localStorage.removeItem('pn:rail'));
    await p.reload({ waitUntil: 'networkidle2' });
    await new Promise((r) => setTimeout(r, 350));

    const a = await olc();
    await p.click('#adminRail');
    await new Promise((r) => setTimeout(r, 400));
    const b1 = await olc();
    await p.click('#adminRail');
    await new Promise((r) => setTimeout(r, 400));
    const c = await olc();

    const ok1 = b1.side !== a.side;
    const ok2 = c.side === a.side;
    console.log(`${String(gen).padStart(4)}px  ${a.side} → ${b1.side} → ${c.side}  ` +
      `${ok1 ? 'katlanıyor' : 'KATLANMIYOR'} · ${ok2 ? 'açılıyor' : 'AÇILMIYOR'}  ` +
      `etiket ${a.etiketGorunur ? 'açık' : 'gizli'}→${b1.etiketGorunur ? 'açık' : 'gizli'}`);
    if (!ok1 || !ok2) process.exitCode = 1;
  }

  // Kalıcılık: daralt, sayfa değiştir, korunuyor mu?
  await p.setViewport({ width: 1440, height: 950 });
  await p.goto(BASE + '/admin/icerikler', { waitUntil: 'networkidle2' });
  await p.evaluate(() => localStorage.removeItem('pn:rail'));
  await p.reload({ waitUntil: 'networkidle2' });
  await p.click('#adminRail');
  await new Promise((r) => setTimeout(r, 350));
  await p.goto(BASE + '/admin/ayarlar', { waitUntil: 'networkidle2' });
  await new Promise((r) => setTimeout(r, 350));
  const kal = await olc();
  console.log(`\nkalıcılık: /admin/ayarlar → ${kal.side}px · kayıt ${kal.kayit}`);
  await p.evaluate(() => localStorage.removeItem('pn:rail'));

  console.log('JS hatası:', hata.length ? hata.slice(0, 3).join(' | ') : 'yok');
  await b.close();
})();
