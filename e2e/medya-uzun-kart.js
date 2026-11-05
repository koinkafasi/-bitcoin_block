const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

// Uzun kartlar: hangi medya, hangi olcu, span kac?
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'], protocolTimeout: 60000 });
  const p = await b.newPage();
  await p.setViewport({ width: 1440, height: 1000 });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });
  await p.goto(BASE + '/admin/dosyalar', { waitUntil: 'networkidle2' });
  await new Promise((x) => setTimeout(x, 1400));

  const d = await p.evaluate(() => {
    const kartlar = [...document.querySelectorAll('#medya-panel .media-item')];
    return kartlar
      .map((el) => {
        const r = el.getBoundingClientRect();
        const img = el.querySelector('img');
        return {
          alt: img ? img.getAttribute('alt') : '',
          boy: Math.round(r.width) + 'x' + Math.round(r.height),
          span: el.style.gridRowEnd,
          dims: el.dataset.dimensions || '(yok)',
          nat: img ? img.naturalWidth + 'x' + img.naturalHeight : '',
          src: img ? img.getAttribute('src') : '',
        };
      })
      .filter((k) => parseInt(k.boy.split('x')[1], 10) > 400)
      .slice(0, 8);
  });

  console.log('uzun kartlar:', d.length);
  d.forEach((k) => console.log('  ' + JSON.stringify(k)));

  const ozet = await p.evaluate(() => {
    const kartlar = [...document.querySelectorAll('#medya-panel .media-item')];
    const boylar = kartlar.map((el) => Math.round(el.getBoundingClientRect().height)).sort((a, b) => a - b);
    return { adet: boylar.length, enKisa: boylar[0], ortanca: boylar[Math.floor(boylar.length / 2)], enUzun: boylar[boylar.length - 1] };
  });
  console.log('yukseklik ozeti:', JSON.stringify(ozet));
  await b.close();
})();
