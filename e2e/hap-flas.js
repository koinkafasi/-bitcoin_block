const puppeteer = require('puppeteer');
const BASE = process.argv[2], SID = process.argv[3];

// Her karede URL + clip. Amaç: hedefte başlayıp geri dönme (flaş) var mı?
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 1000 });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  await p.goto(BASE + '/admin/icerikler', { waitUntil: 'networkidle2' });
  await new Promise((r) => setTimeout(r, 700));

  const oku = () => p.evaluate(() => {
    const ray = document.querySelector('.ws-links');
    const pill = ray && ray.querySelector('.tb-hap');
    let sol = null;
    if (pill) {
      const m = /inset\([-\d.]+px [-\d.]+px [-\d.]+px ([-\d.]+)px/.exec(pill.style.clipPath || '');
      if (m) sol = Math.round(parseFloat(m[1]));
    }
    return { yol: location.pathname, sol, var: !!pill };
  }).catch(() => null);

  console.log('tıklama öncesi:', JSON.stringify(await oku()));
  p.click('.ws-links a[href="/admin/icerikler/yeni"]').catch(() => {});

  const kareler = [];
  for (let i = 0; i < 34; i++) {
    await new Promise((r) => setTimeout(r, 22));
    const o = await oku();
    if (o) kareler.push(o);
  }

  // Belge değişimini işaretle
  let oncekiYol = null;
  const goster = kareler.map((k) => {
    const yeniBelge = k.yol !== oncekiYol;
    oncekiYol = k.yol;
    return (yeniBelge ? '‖' : '') + (k.sol === null ? '-' : k.sol);
  });
  console.log('kareler:', goster.join(' '));

  const yeniBelgeIlk = kareler.findIndex((k, i) => i > 0 && k.yol !== kareler[i - 1].yol);
  if (yeniBelgeIlk > 0) {
    const yeniKareler = kareler.slice(yeniBelgeIlk).map((k) => k.sol);
    console.log('yeni belge kareleri:', yeniKareler.join(' → '));
    const tekil = yeniKareler.filter((v, i) => i === 0 || v !== yeniKareler[i - 1]);
    console.log('tekil konum:', tekil.length, tekil.length > 1 ? '· animasyon var' : '· ANİMASYON YOK');
    console.log('yol:', tekil[0], '→', tekil[tekil.length - 1]);
    const geri = tekil.some((v, i) => i > 0 && v < tekil[i - 1] - 2);
    console.log('geri sıçrama (flaş):', geri ? 'VAR ⚠' : 'yok');
  } else {
    console.log('yeni belge yakalanamadı — yeniden deneyin');
  }
  await b.close();
})();
