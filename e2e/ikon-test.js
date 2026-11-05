const puppeteer = require('puppeteer');
const BASE = process.argv[2], SID = process.argv[3];
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox', '--disable-dev-shm-usage'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 1000 });
  const hata = [], kotu = [];
  p.on('pageerror', (e) => hata.push(String(e.message)));
  p.on('console', (m) => { if (m.type() === 'error') hata.push('console: ' + m.text()); });
  p.on('response', (r) => { if (r.status() >= 400) kotu.push(r.status() + ' ' + r.url()); });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  await p.goto(BASE + '/admin/ikonlar', { waitUntil: 'networkidle2', timeout: 60000 });
  await new Promise((r) => setTimeout(r, 700));

  const o = await p.evaluate(() => {
    const kartlar = [...document.querySelectorAll('.ik-kart')];
    return {
      baslik: (document.querySelector('.admin-top h1') || {}).textContent,
      kart: kartlar.length,
      ilk5: kartlar.slice(0, 5).map((k) => k.getAttribute('data-ikon')),
      anim: document.querySelectorAll('.ik-anim').length,
      animAnimasyonlu: [...document.querySelectorAll('.ik-anim svg')].filter((s) => {
        const c = getComputedStyle(s); return c.animationName && c.animationName !== 'none';
      }).length,
      spinner: document.querySelectorAll('.ik-spin').length,
      spinStroke: (() => {
        const s = document.querySelector('.ik-spin .ikon-spin-svg svg');
        return s ? getComputedStyle(s).stroke : null;
      })(),
      spinStrokeAttr: (() => {
        const s = document.querySelector('.ik-spin .ikon-spin-svg svg');
        return s ? s.getAttribute('stroke') : null;
      })(),
      sayi: (document.getElementById('ikSayi') || {}).textContent,
      tasma: document.documentElement.scrollWidth > document.documentElement.clientWidth + 1,
      cssYuklendi: [...document.styleSheets].some((s) => (s.href || '').includes('ikonlar.css')),
    };
  });

  // Sprite'taki sembol sayısı + kullanılan adların hepsi var mı
  const sprite = await p.evaluate(async () => {
    const hrefs = [...document.querySelectorAll('use')].map((u) => u.getAttribute('href')).filter(Boolean);
    const kullanilan = [...new Set(hrefs.filter((h) => h.includes('tabler-icons.svg')).map((h) => h.split('#')[1]))];
    const txt = await (await fetch('/static/vendor/tabler-icons.svg')).text();
    const varOlan = new Set([...txt.matchAll(/id="(ti-[a-z0-9-]+)"/g)].map((m) => m[1]));
    return {
      sembolSayisi: varOlan.size,
      kullanilan: kullanilan.length,
      eksik: kullanilan.filter((k) => !varOlan.has(k)),
    };
  });

  console.log('başlık        :', o.baslik);
  console.log('ikon kartı    :', o.kart, '·', o.sayi);
  console.log('ilk 5         :', o.ilk5.join(', '));
  console.log('animasyon     :', o.anim, 'kart ·', o.animAnimasyonlu, 'gerçekten animasyonlu');
  console.log('spinner       :', o.spinner, 'adet · stroke attr', o.spinStrokeAttr, '· hesaplanan', o.spinStroke);
  console.log('ikonlar.css   :', o.cssYuklendi, '· yatay taşma:', o.tasma ? 'VAR' : 'yok');
  console.log('sprite sembol :', sprite.sembolSayisi, '· sayfada kullanılan', sprite.kullanilan);
  console.log('EKSİK sembol  :', sprite.eksik.length ? sprite.eksik.join(', ') : 'yok');

  // arama
  await p.type('#ikAra', 'bell');
  await new Promise((r) => setTimeout(r, 350));
  const ara = await p.evaluate(() => ({
    sayi: document.getElementById('ikSayi').textContent,
    gorunen: [...document.querySelectorAll('.ik-kart')].filter((k) => !k.hidden).map((k) => k.getAttribute('data-ikon')),
  }));
  console.log('arama "bell"  :', ara.sayi, '·', ara.gorunen.join(', '));

  // pano sayfasında da ikonlar çözülüyor mu (regresyon: PanoIkon adları)
  await p.goto(BASE + '/admin/pano', { waitUntil: 'networkidle2' });
  await new Promise((r) => setTimeout(r, 500));
  const pano = await p.evaluate(async () => {
    const kullanilan = [...new Set([...document.querySelectorAll('use')]
      .map((u) => u.getAttribute('href')).filter((h) => h && h.includes('tabler-icons.svg'))
      .map((h) => h.split('#')[1]))];
    const txt = await (await fetch('/static/vendor/tabler-icons.svg')).text();
    const varOlan = new Set([...txt.matchAll(/id="(ti-[a-z0-9-]+)"/g)].map((m) => m[1]));
    return { kullanilan: kullanilan.length, eksik: kullanilan.filter((k) => !varOlan.has(k)) };
  });
  console.log('pano ikonları :', pano.kullanilan, 'adet · eksik:', pano.eksik.length ? pano.eksik.join(', ') : 'yok');

  await p.goto(BASE + '/admin/ikonlar', { waitUntil: 'networkidle2' });
  await p.screenshot({ path: '/tmp/ikon-galeria.png' });
  console.log('JS hatası     :', hata.length ? [...new Set(hata)].slice(0, 3).join(' | ') : 'yok');
  console.log('hatalı istek  :', kotu.length ? kotu.slice(0, 3).join(' | ') : 'yok');
  await b.close();
})();
