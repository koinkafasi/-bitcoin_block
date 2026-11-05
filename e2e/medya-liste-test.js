const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

function okuSayfa() {
  const panel = document.getElementById('medya-panel');
  const item = document.querySelector('#medya-panel .media-item');
  return {
    sinif: panel ? panel.className : 'yok',
    span: item ? item.style.gridRowEnd : null,
    h: item ? Math.round(item.getBoundingClientRect().height) : null,
  };
}

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'], protocolTimeout: 60000 });
  const p = await b.newPage();
  await p.setViewport({ width: 1400, height: 1000 });
  const hata = [];
  p.on('pageerror', (e) => hata.push(e.message));
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });
  await p.goto(BASE + '/admin/dosyalar', { waitUntil: 'networkidle2' });
  await new Promise((x) => setTimeout(x, 900));

  console.log('baslangic :', JSON.stringify(await p.evaluate(okuSayfa)));
  await p.click('[data-media-view="list"]');
  await new Promise((x) => setTimeout(x, 120));
  console.log('list 120ms:', JSON.stringify(await p.evaluate(okuSayfa)));
  await new Promise((x) => setTimeout(x, 600));
  console.log('list 720ms:', JSON.stringify(await p.evaluate(okuSayfa)));

  await p.click('[data-media-view="grid"]');
  await new Promise((x) => setTimeout(x, 500));
  console.log('grid      :', JSON.stringify(await p.evaluate(okuSayfa)));

  console.log('JS hatası:', hata.length ? [...new Set(hata)].slice(0, 3).join(' | ') : 'yok');
  await b.close();
})();
