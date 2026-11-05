const puppeteer = require('puppeteer');

const BASE = process.argv[2];
const SID = process.argv[3];

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 1000 });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });
  await p.goto(BASE + '/admin/icerikler', { waitUntil: 'networkidle2' });
  await new Promise((r) => setTimeout(r, 600));

  const bilgi = await p.evaluate(() => {
    const sels = [...document.querySelectorAll('select')];
    return sels.map((s) => {
      const c = getComputedStyle(s);
      const r = s.getBoundingClientRect();
      const wrap = s.parentElement;
      return {
        name: s.name || s.id || '(adsız)',
        cls: s.className,
        w: Math.round(r.width),
        h: Math.round(r.height),
        appearance: c.appearance + '/' + c.webkitAppearance,
        bgImage: c.backgroundImage.slice(0, 30),
        wrapTag: wrap ? wrap.tagName + '.' + wrap.className : null,
        wrapBg: wrap ? getComputedStyle(wrap).backgroundImage.slice(0, 30) : null,
        secenek: s.options.length ? s.options[s.selectedIndex].text : '',
      };
    });
  });
  console.log(JSON.stringify(bilgi, null, 1));

  // tarih girdileri
  const tarih = await p.evaluate(() =>
    [...document.querySelectorAll('input[type=date]')].map((i) => ({
      name: i.name, lang: i.lang, value: i.value, type: i.type,
    })));
  console.log('tarih:', JSON.stringify(tarih));

  // sayfa dili
  console.log('html lang:', await p.evaluate(() => document.documentElement.lang));
  await b.close();
})();
