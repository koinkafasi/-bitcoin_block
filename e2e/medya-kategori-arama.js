const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

// Faz 187: kategori secili iken arama yapinca kategori korunuyor mu?
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'], protocolTimeout: 60000 });
  const p = await b.newPage();
  await p.setViewport({ width: 1440, height: 1000 });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });
  await p.goto(BASE + '/admin/dosyalar', { waitUntil: 'networkidle2' });
  await new Promise((x) => setTimeout(x, 700));

  const cipler = await p.$$('.media-cip');
  if (cipler.length > 1) await cipler[1].click();
  await new Promise((x) => setTimeout(x, 800));
  const once = await p.evaluate(() => ({ url: location.search, secili: (document.querySelector('.media-cip.is-on') || {}).textContent }));

  await p.click('#medyaAra');
  await p.type('#medyaAra', 'seed', { delay: 40 });
  await new Promise((x) => setTimeout(x, 1500));
  const sonra = await p.evaluate(() => ({
    url: location.search,
    secili: [...document.querySelectorAll('.media-cip.is-on')].map((c) => c.textContent.replace(/\s+/g, ' ').trim()),
    gizliKategori: (document.querySelector('input[name="kategori"]') || {}).value,
    kart: document.querySelectorAll('#medya-panel .media-item').length,
    toplam: (document.querySelector('.media-summary strong') || {}).textContent || '',
  }));
  console.log('once :', JSON.stringify(once));
  console.log('sonra:', JSON.stringify(sonra, null, 1));
  await b.close();
})();
