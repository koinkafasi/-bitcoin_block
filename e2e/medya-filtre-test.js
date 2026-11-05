const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

// Faz 187: chip filtresi, arama, liste gorunumu ve "liste bitti" durumu.
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'], protocolTimeout: 120000 });
  const p = await b.newPage();
  await p.setViewport({ width: 1440, height: 1000 });
  const hata = [];
  p.on('pageerror', (e) => hata.push(e.message));
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  await p.goto(BASE + '/admin/dosyalar', { waitUntil: 'networkidle2' });
  await new Promise((x) => setTimeout(x, 700));

  // 1) Cip tikla: genel
  const cipler = await p.$$('.media-cip');
  if (cipler.length > 1) await cipler[1].click();
  await new Promise((x) => setTimeout(x, 900));
  const cipSonuc = await p.evaluate(() => ({
    url: location.pathname + location.search,
    secili: [...document.querySelectorAll('.media-cip.is-on')].map((c) => c.textContent.replace(/\s+/g, ' ').trim()),
    ariaSecili: [...document.querySelectorAll('.media-cip[aria-current="page"]')].map((c) => c.textContent.replace(/\s+/g, ' ').trim()),
    kart: document.querySelectorAll('#medya-panel .media-item').length,
    kategoriMetinleri: [...document.querySelectorAll('.media-item .mi-cap small')].slice(0, 3).map((s) => s.textContent),
  }));
  console.log('=== CIP FILTRESI ===');
  console.log(JSON.stringify(cipSonuc, null, 1));

  // 2) Arama: yaz, bekle
  await p.click('#medyaAra');
  await p.type('#medyaAra', 'seed', { delay: 40 });
  await new Promise((x) => setTimeout(x, 1400));
  const aramaSonuc = await p.evaluate(() => {
    const ara = document.getElementById('medyaAra');
    return {
      url: location.pathname + location.search,
      odakArama: document.activeElement === ara,
      deger: ara ? ara.value : '(yok)',
      kart: document.querySelectorAll('#medya-panel .media-item').length,
      toplam: (document.querySelector('.media-summary strong') || {}).textContent || '',
      ilkBaslik: (document.querySelector('.media-item .mi-cap b') || {}).textContent || '',
      // Filtre etkinken "Temizle" gorunmeli
      temizle: [...document.querySelectorAll('.media-arac__eylem a')].some((a) => a.textContent.trim() === 'Temizle'),
    };
  });
  console.log('\n=== ARAMA (yazarken) ===');
  console.log(JSON.stringify(aramaSonuc, null, 1));

  // 3) Liste gorunumu
  await p.click('[data-media-view="list"]');
  await new Promise((x) => setTimeout(x, 500));
  const liste = await p.evaluate(() => {
    const it = document.querySelector('#medya-panel .media-item');
    const r = it.getBoundingClientRect();
    const img = it.querySelector('img').getBoundingClientRect();
    return {
      sinif: document.getElementById('medya-panel').className,
      kartH: Math.round(r.height), imgW: Math.round(img.width), imgH: Math.round(img.height),
      capGorunur: getComputedStyle(it.querySelector('.mi-cap')).display !== 'none',
    };
  });
  console.log('\n=== LISTE GORUNUMU ===');
  console.log(JSON.stringify(liste, null, 1));
  await p.click('[data-media-view="grid"]');
  await new Promise((x) => setTimeout(x, 400));

  // 4) Sonuna kadar kaydir: "Tumu gosterildi" cikmali
  await p.goto(BASE + '/admin/dosyalar', { waitUntil: 'networkidle2' });
  await new Promise((x) => setTimeout(x, 700));
  for (let i = 0; i < 16; i++) {
    await p.evaluate(() => window.scrollTo(0, document.body.scrollHeight));
    await new Promise((x) => setTimeout(x, 900));
    const bitti = await p.evaluate(() => !!document.querySelector('.media-son-bitti'));
    if (bitti) break;
  }
  const son = await p.evaluate(() => ({
    kart: document.querySelectorAll('#medya-panel .media-item').length,
    toplam: (document.querySelector('.media-summary strong') || {}).textContent || '',
    gozcu: !!document.getElementById('medyaSon'),
    bitti: (document.querySelector('.media-son-bitti') || {}).textContent || '',
  }));
  console.log('\n=== SON ===');
  console.log(JSON.stringify(son, null, 1));

  console.log('\nJS hatasi:', hata.length ? [...new Set(hata)].slice(0, 3).join(' | ') : 'yok');
  await b.close();
})();
