const puppeteer = require('puppeteer');
const BASE = process.argv[2], SID = process.argv[3];

const durum = (p) => p.evaluate(() => {
  const ray = document.querySelector('.ws-links');
  const pill = ray && ray.querySelector('.tb-hap');
  let clip = null;
  if (pill) {
    const m = /inset\(([-\d.]+)px ([-\d.]+)px ([-\d.]+)px ([-\d.]+)px/.exec(pill.style.clipPath || '');
    if (m) clip = { sol: Math.round(parseFloat(m[4])), sag: Math.round(parseFloat(m[2])), ust: Math.round(parseFloat(m[1])) };
  }
  const vt = document.querySelectorAll('::view-transition');
  return {
    hapVar: !!pill,
    clip,
    opak: pill ? pill.style.opacity : null,
    aktif: ray ? (ray.querySelector('.on') || {}).textContent : null,
    // view-transition etkin mi
    vtDestek: typeof document.startViewTransition === 'function',
  };
});

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 1000 });
  const hata = [];
  p.on('pageerror', (e) => hata.push(String(e.message)));
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  // 1) Hap "Yeni İçerik" sayfasında var mı?
  await p.goto(BASE + '/admin/icerikler/yeni', { waitUntil: 'networkidle2', timeout: 60000 });
  await new Promise((r) => setTimeout(r, 900));
  const yeni = await durum(p);
  console.log('=== /admin/icerikler/yeni ===');
  console.log('hap kuruldu :', yeni.hapVar, '· aktif:', yeni.aktif);
  console.log('clip        :', JSON.stringify(yeni.clip), '· opak:', yeni.opak);

  const hiza = await p.evaluate(() => {
    const ray = document.querySelector('.ws-links');
    const pill = ray && ray.querySelector('.tb-hap');
    const aktif = ray && ray.querySelector('.on');
    if (!pill || !aktif) return 'eksik';
    const pr = pill.getBoundingClientRect(), ar = aktif.getBoundingClientRect();
    const m = /inset\(([-\d.]+)px ([-\d.]+)px ([-\d.]+)px ([-\d.]+)px/.exec(pill.style.clipPath || '');
    return {
      sapmaSol: m ? Math.round(pr.left + parseFloat(m[4]) - ar.left) : 'clip yok',
      sapmaSag: m ? Math.round(pr.right - parseFloat(m[2]) - ar.right) : 'clip yok',
      sekmeSayisi: ray.querySelectorAll('a').length,
    };
  });
  console.log('hizalama    :', JSON.stringify(hiza));

  // 2) İçerikler → Yeni İçerik geçişini kare kare örnekle
  await p.goto(BASE + '/admin/icerikler', { waitUntil: 'networkidle2' });
  await new Promise((r) => setTimeout(r, 700));
  const once = await durum(p);
  console.log('\n=== İçerikler → Yeni İçerik ===');
  console.log('başlangıç   :', JSON.stringify(once.clip));

  p.click('.ws-links a[href="/admin/icerikler/yeni"]').catch(() => {});
  const kareler = [];
  for (let i = 0; i < 30; i++) {
    await new Promise((r) => setTimeout(r, 30));
    const d = await durum(p).catch(() => null);
    if (d && d.clip) kareler.push(d.clip.sol);
    else if (d) kareler.push(null);
  }
  console.log('sol kenar   :', kareler.map((k) => (k === null ? 'yok' : k)).join(' → '));
  const dolu = kareler.filter((k) => k !== null);
  const tekil = dolu.filter((k, i) => i === 0 || k !== dolu[i - 1]);
  console.log('farklı konum:', tekil.length, tekil.length > 1 ? '(animasyon var)' : '(ANİMASYON YOK)');
  if (tekil.length > 1) console.log('yol         :', tekil[0], '→', tekil[tekil.length - 1]);

  // 3) view-transition navigasyonda devrede mi? (stil kontrolü)
  const vt = await p.evaluate(() => {
    const out = {};
    for (const sh of document.styleSheets) {
      try {
        for (const r of sh.cssRules) {
          if (r.cssText && r.cssText.includes('view-transition')) {
            out[r.cssText.slice(0, 80)] = true;
          }
        }
      } catch (e) {}
    }
    return Object.keys(out);
  });
  console.log('\nview-transition kuralları:', vt.length ? vt.join(' | ') : 'yok');

  console.log('\nJS hatası:', hata.length ? [...new Set(hata)].slice(0, 3).join(' | ') : 'yok');
  await b.close();
})();
