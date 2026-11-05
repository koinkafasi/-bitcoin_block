const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'], protocolTimeout: 60000 });
  const p = await b.newPage();
  await p.setViewport({ width: 1400, height: 1000 });
  if (SID) {
    await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });
  }

  const yanitlar = [];
  p.on('response', (res) => {
    const u = res.url();
    if (u.indexOf('/media/thumb/') === -1) return;
    yanitlar.push({
      durum: res.status(),
      url: u.replace(BASE, ''),
      tur: res.headers()['content-type'] || '',
    });
  });

  await p.goto(BASE + '/admin/dosyalar', { waitUntil: 'networkidle2' });
  await new Promise((x) => setTimeout(x, 2500));

  const d = await p.evaluate(() => {
    const out = { toplam: 0, kirik: [], oranlar: {} };
    document.querySelectorAll('#medya-panel .media-item img').forEach((img) => {
      out.toplam += 1;
      const r = img.getBoundingClientRect();
      if (!img.complete || img.naturalWidth === 0) {
        out.kirik.push({ src: img.getAttribute('src'), alt: img.getAttribute('alt') });
      }
      const k = Math.round(r.width) + 'x' + Math.round(r.height);
      out.oranlar[k] = (out.oranlar[k] || 0) + 1;
    });
    out.kirik = out.kirik.slice(0, 8);
    return out;
  });

  console.log('img toplam:', d.toplam, '· kirik:', d.kirik.length);
  console.log('kirik ornek:', JSON.stringify(d.kirik, null, 1));
  console.log('boyut dagilimi:', JSON.stringify(d.oranlar));

  const kodlar = {};
  yanitlar.forEach((y) => {
    kodlar[y.durum] = (kodlar[y.durum] || 0) + 1;
  });
  console.log('thumb yanit kodlari:', JSON.stringify(kodlar));
  const kotu = yanitlar.filter((y) => y.durum !== 200).slice(0, 5);
  console.log('200 olmayan ornek:', JSON.stringify(kotu, null, 1));
  await b.close();
})();
