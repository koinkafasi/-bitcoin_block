const puppeteer = require('puppeteer');

const BASE = process.argv[2];
const SID = process.argv[3];

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 1000, deviceScaleFactor: 2 });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });
  await p.goto(BASE + '/admin', { waitUntil: 'networkidle2' });
  await new Promise((r) => setTimeout(r, 600));

  // Arama terimini bul: veritabanındaki bir içerik başlığının ilk kelimesi.
  const kelime = await p.evaluate(async () => {
    const r = await fetch('/admin/ara?q=test', { credentials: 'same-origin' });
    const d = await r.json();
    const ilk = (d.gruplar || []).find((g) => g.ad === 'İçerik');
    return ilk ? ilk.oge[0].baslik.split(' ')[0] : 'test';
  });
  console.log('denenecek kelime:', kelime);

  await p.keyboard.down('Control'); await p.keyboard.press('KeyK'); await p.keyboard.up('Control');
  await new Promise((r) => setTimeout(r, 250));
  await p.type('#cmdkInput', kelime, { delay: 50 });
  await new Promise((r) => setTimeout(r, 800));

  const r2 = await p.evaluate(() => {
    const l = document.getElementById('cmdkList');
    return {
      grup: [...l.querySelectorAll('.cmdk-grup')].map((g) => g.textContent),
      oge: [...l.querySelectorAll('.cmdk-oge')].map((o) => ({
        ad: (o.querySelector('.cmdk-oge__ad') || {}).textContent,
        alt: (o.querySelector('.cmdk-oge__alt') || {}).textContent,
        rozet: (o.querySelector('.cmdk-oge__rozet') || {}).textContent,
        vurgu: !!o.querySelector('mark'),
      })),
    };
  });
  console.log('gruplar:', JSON.stringify(r2.grup));
  console.log('öğeler:', JSON.stringify(r2.oge.slice(0, 8), null, 1));
  await p.screenshot({ path: '/tmp/palet-arama.png' });

  // Son kullanılanlar: bir öğeyi seç, sonra yeniden aç.
  const ilkHref = await p.evaluate(() => (document.querySelector('.cmdk-oge') || {}).dataset?.href);
  if (ilkHref) {
    await p.evaluate((h) => {
      const li = [...document.querySelectorAll('.cmdk-oge')].find((x) => x.dataset.href === h);
      if (li) li.click();
    }, ilkHref);
    await new Promise((r) => setTimeout(r, 1200));
    await p.keyboard.down('Control'); await p.keyboard.press('KeyK'); await p.keyboard.up('Control');
    await new Promise((r) => setTimeout(r, 400));
    const son = await p.evaluate(() => {
      const l = document.getElementById('cmdkList');
      return {
        grup: [...l.querySelectorAll('.cmdk-grup')].map((g) => g.textContent),
        ilk: (l.querySelector('.cmdk-oge__ad') || {}).textContent,
      };
    });
    console.log('yeniden açınca:', JSON.stringify(son));
  }
  await b.close();
})();
