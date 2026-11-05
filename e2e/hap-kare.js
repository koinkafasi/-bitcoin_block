const puppeteer = require('puppeteer');
const BASE = process.argv[2], SID = process.argv[3];

const oku = (p) => p.evaluate(() => {
  const pill = document.querySelector('.ws-links .tb-hap');
  if (!pill) return null;
  const m = /inset\([-\d.]+px ([-\d.]+)px [-\d.]+px ([-\d.]+)px/.exec(pill.style.clipPath || '');
  return m ? { sol: Math.round(parseFloat(m[2])), sag: Math.round(parseFloat(m[1])) } : null;
});

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 1000 });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  await p.goto(BASE + '/admin/icerikler', { waitUntil: 'networkidle2' });
  await new Promise((r) => setTimeout(r, 500));
  const once = await oku(p);
  console.log('başlangıç (İçerikler) :', JSON.stringify(once));

  // Tıkla, hemen örnekle (navigation'ı bekleme).
  p.click('.ws-links a[href="/admin/kategoriler"]').catch(() => {});
  const kareler = [];
  for (let i = 0; i < 26; i++) {
    await new Promise((r) => setTimeout(r, 35));
    const o = await oku(p).catch(() => null);
    if (o) kareler.push(o);
  }
  console.log('sol kenar :', kareler.map((k) => k.sol).join(' → '));
  console.log('sağ inset :', kareler.map((k) => k.sag).join(' → '));

  const tekil = kareler.filter((k, i) => i === 0 || k.sol !== kareler[i - 1].sol);
  console.log('farklı konum sayısı   :', tekil.length, '(1 ise animasyon yok)');
  if (tekil.length > 1) {
    console.log('ilk → son             :', tekil[0].sol, '→', tekil[tekil.length - 1].sol);
    const artan = tekil.every((k, i) => i === 0 || k.sol >= tekil[i - 1].sol);
    console.log('tek yönlü             :', artan);
  }
  await b.close();
})();
