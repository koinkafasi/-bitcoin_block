const puppeteer = require('puppeteer');

const BASE = process.argv[2];
const SID = process.argv[3];

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 1000, deviceScaleFactor: 2 });
  const hata = [];
  p.on('pageerror', (e) => hata.push(e.message));
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  const r = await p.goto(BASE + '/admin/gelen-kutusu', { waitUntil: 'networkidle2' });
  await new Promise((x) => setTimeout(x, 700));
  console.log('HTTP:', r.status());

  const d = await p.evaluate(() => {
    const kart = [...document.querySelectorAll('.gelen-kart')].map((k) => ({
      ad: k.querySelector('.gelen-kart__ad').textContent,
      sayi: k.querySelector('.gelen-kart__sayi').textContent,
      on: k.classList.contains('on'),
      href: k.getAttribute('href'),
    }));
    return {
      kart,
      satir: document.querySelectorAll('.gelen-oge').length,
      bos: !!document.querySelector('.bos-durum'),
      turRozet: [...document.querySelectorAll('.gelen-oge__alt .badge')].slice(0, 4).map((x) => x.textContent),
      emoji: /[\u{1F000}-\u{1FAFF}\u{2600}-\u{27BF}]/u.test(document.body.innerText),
      tasma: document.documentElement.scrollWidth - window.innerWidth,
    };
  });
  console.log(JSON.stringify(d));

  // Süzgeç: tıklama süzüyor mu
  const link = await p.evaluate(() => {
    const k = [...document.querySelectorAll('.gelen-kart')].find((x) => x.getAttribute('href').includes('tur=iletisim'));
    return k ? k.getAttribute('href') : null;
  });
  if (link) {
    await p.goto(BASE + link, { waitUntil: 'networkidle2' });
    await new Promise((x) => setTimeout(x, 500));
    const s = await p.evaluate(() => ({
      satir: document.querySelectorAll('.gelen-oge').length,
      tekTur: [...new Set([...document.querySelectorAll('.gelen-oge__alt .badge')].map((x) => x.textContent))],
      onKart: (document.querySelector('.gelen-kart.on .gelen-kart__ad') || {}).textContent,
    }));
    console.log('süzgeç iletisim:', JSON.stringify(s));
  }

  await p.goto(BASE + '/admin/gelen-kutusu', { waitUntil: 'networkidle2' });
  await new Promise((x) => setTimeout(x, 400));
  await p.screenshot({ path: '/tmp/gelen.png', fullPage: true });
  console.log('JS hatası:', hata.length ? [...new Set(hata)].join(' | ') : 'yok');
  await b.close();
})();
