const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'], protocolTimeout: 60000 });
  const p = await b.newPage();
  await p.setViewport({ width: 1440, height: 1000 });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });
  await p.goto(BASE + '/admin/dosyalar', { waitUntil: 'networkidle2' });
  await new Promise((x) => setTimeout(x, 700));

  const d = await p.evaluate(async () => {
    const panel = document.getElementById('medya-panel');
    const btn = panel.querySelector('.mgal-drop__btn');
    const bekle = (ms) => new Promise((r) => setTimeout(r, ms));

    const once = {
      bg: getComputedStyle(btn).backgroundColor,
      kenar: getComputedStyle(btn).borderStyle,
      kenarRenk: getComputedStyle(btn).borderColor,
    };
    panel.classList.add('is-drag');
    await bekle(320);
    const sonra = {
      bg: getComputedStyle(btn).backgroundColor,
      kenar: getComputedStyle(btn).borderStyle,
      kenarRenk: getComputedStyle(btn).borderColor,
      ico: getComputedStyle(panel.querySelector('.mgal-drop__ico')).transform,
    };
    panel.classList.remove('is-drag');
    await bekle(320);
    const geri = getComputedStyle(btn).backgroundColor;
    return { once, sonra, geri };
  });
  console.log(JSON.stringify(d, null, 1));
  await b.close();
})();
