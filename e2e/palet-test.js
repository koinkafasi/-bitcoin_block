const puppeteer = require('puppeteer');

// Faz 162: komut paleti 2.0 doğrulaması.
const BASE = process.argv[2];
const SID = process.argv[3];

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 1000 });
  const hata = [];
  p.on('pageerror', (e) => hata.push(e.message));
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });
  await p.goto(BASE + '/admin', { waitUntil: 'networkidle2' });
  await new Promise((r) => setTimeout(r, 700));

  // Aç: ⌘K
  await p.keyboard.down('Control'); await p.keyboard.press('KeyK'); await p.keyboard.up('Control');
  await new Promise((r) => setTimeout(r, 300));
  const acik = await p.evaluate(() => document.getElementById('cmdk').classList.contains('open'));
  console.log('Ctrl+K açtı:', acik);

  const bos = await p.evaluate(() => {
    const l = document.getElementById('cmdkList');
    return {
      grup: [...l.querySelectorAll('.cmdk-grup')].map((g) => g.textContent),
      oge: l.querySelectorAll('.cmdk-oge').length,
    };
  });
  console.log('boş sorgu grupları:', JSON.stringify(bos.grup), '· öğe:', bos.oge);

  // "kadıköy" gibi bir içerik başlığı yerine, veritabanında kesin olan bir
  // kelimeyi dene: önce içerik başlıklarını çek.
  const ornek = await p.evaluate(async () => {
    const r = await fetch('/admin/icerikler?durum=&q=', { credentials: 'same-origin' });
    return r.status;
  });
  console.log('icerikler durumu:', ornek);

  // Sunucu aramasını doğrudan sına
  const arama = await p.evaluate(async () => {
    const sonuc = {};
    for (const q of ['a', 'e', 'test']) {
      const r = await fetch('/admin/ara?q=' + encodeURIComponent(q), { credentials: 'same-origin' });
      const d = await r.json();
      sonuc[q] = { durum: r.status, ok: d.ok, gruplar: (d.gruplar || []).map((g) => g.ad + '(' + g.oge.length + ')') };
    }
    return sonuc;
  });
  console.log('sunucu araması:', JSON.stringify(arama));

  // Arayüze yaz
  await p.click('#cmdkInput');
  await p.type('#cmdkInput', 'e', { delay: 60 });
  await new Promise((r) => setTimeout(r, 600));
  const arayuz = await p.evaluate(() => {
    const l = document.getElementById('cmdkList');
    return {
      grup: [...l.querySelectorAll('.cmdk-grup')].map((g) => g.textContent),
      oge: [...l.querySelectorAll('.cmdk-oge')].slice(0, 6).map((o) => ({
        ad: (o.querySelector('.cmdk-oge__ad') || {}).textContent,
        alt: (o.querySelector('.cmdk-oge__alt') || {}).textContent,
        rozet: (o.querySelector('.cmdk-oge__rozet') || {}).textContent,
        href: o.dataset.href,
        vurgu: !!o.querySelector('mark'),
      })),
      secili: (l.querySelector('.cmdk-oge.on .cmdk-oge__ad') || {}).textContent,
    };
  });
  console.log('yazınca gruplar:', JSON.stringify(arayuz.grup));
  console.log('ilk öğeler:', JSON.stringify(arayuz.oge, null, 1));
  console.log('seçili:', arayuz.secili);

  // Klavye gezinme
  await p.keyboard.press('ArrowDown');
  const sonra = await p.evaluate(() => (document.querySelector('.cmdk-oge.on .cmdk-oge__ad') || {}).textContent);
  console.log('ArrowDown sonrası seçili:', sonra);

  // Emoji var mı?
  // Emoji: yalnız SONUÇ listesinde aranır. Altbilgideki ↑ ↓ klavye
  // göstergeleri meşru (⌘K gibi); onlar içerik değil, tuş etiketi.
  const emoji = await p.evaluate(() => {
    const re = /[\u{1F000}-\u{1FAFF}\u{2600}-\u{27BF}\u{2300}-\u{23FF}]/u;
    return re.test(document.getElementById('cmdkList').textContent);
  });
  console.log('palette emoji:', emoji);

  // Ölçü
  const olcu = await p.evaluate(() => {
    const bx = document.querySelector('.cmdk-box').getBoundingClientRect();
    const c = getComputedStyle(document.querySelector('.cmdk-box'));
    return { w: Math.round(bx.width), h: Math.round(bx.height), radius: c.borderRadius, bg: c.backgroundColor };
  });
  console.log('kutu:', JSON.stringify(olcu));

  await p.screenshot({ path: '/tmp/palet.png' });
  await p.keyboard.press('Escape');
  const kapandi = await p.evaluate(() => !document.getElementById('cmdk').classList.contains('open'));
  console.log('Esc kapattı:', kapandi);
  console.log('JS hatası:', hata.length ? [...new Set(hata)].join(' | ') : 'yok');
  await b.close();
})();
