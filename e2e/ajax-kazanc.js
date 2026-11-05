const puppeteer = require('puppeteer');
const BASE = process.argv[2], SID = process.argv[3];

// Kazanç ölçümü. Bayt/istek sayısı CDP Network'ten (belgeler arası doğru),
// süre "içerik göründü" anından (sabit uyku yok).
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 1000 });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  const cdp = await p.createCDPSession();
  await cdp.send('Network.enable');
  let adet = 0, bayt = 0;
  cdp.on('Network.loadingFinished', (e) => { adet++; bayt += e.encodedDataLength || 0; });
  const sifirla = () => { adet = 0; bayt = 0; };
  const sonuc = (sure) => ({ adet, kb: +(bayt / 1024).toFixed(1), sure });

  const bekleBaslik = async (beklenen, ms = 10000) => {
    const t0 = Date.now();
    while (Date.now() - t0 < ms) {
      const h = await p.evaluate(() => (document.querySelector('.admin-top h1') || {}).textContent || '').catch(() => '');
      if (h.trim() === beklenen) return true;
      await new Promise((r) => setTimeout(r, 20));
    }
    return false;
  };

  // Isınma: varlıklar önbelleğe girsin (ikinci ziyaret gerçekçi).
  await p.goto(BASE + '/admin/icerikler', { waitUntil: 'networkidle2' });
  await p.goto(BASE + '/admin/kategoriler', { waitUntil: 'networkidle2' });
  await new Promise((r) => setTimeout(r, 400));

  // --- 1) TAM YÜKLEME ---
  await p.goto(BASE + '/admin/icerikler', { waitUntil: 'networkidle2' });
  await new Promise((r) => setTimeout(r, 400));
  sifirla();
  let t = Date.now();
  await p.goto(BASE + '/admin/kategoriler', { waitUntil: 'domcontentloaded' });
  await bekleBaslik('Kategoriler');
  const tam = sonuc(Date.now() - t);

  // --- 2) BOOST ---
  await p.goto(BASE + '/admin/icerikler', { waitUntil: 'networkidle2' });
  await new Promise((r) => setTimeout(r, 500));
  sifirla();
  t = Date.now();
  await p.click('.ws-links a[href="/admin/kategoriler"]');
  await bekleBaslik('Kategoriler');
  const boost = sonuc(Date.now() - t);

  console.log('TAM YÜKLEME  → /admin/kategoriler');
  console.log('   istek', tam.adet, '·', tam.kb, 'KB ·', tam.sure, 'ms');
  console.log('\nBOOST        → /admin/kategoriler');
  console.log('   istek', boost.adet, '·', boost.kb, 'KB ·', boost.sure, 'ms');
  console.log('\nKAZANÇ');
  console.log('   bayt :', tam.kb, 'KB →', boost.kb, 'KB',
    '(' + (100 - (boost.kb / Math.max(1, tam.kb)) * 100).toFixed(0) + '% az)');
  console.log('   süre :', tam.sure, 'ms →', boost.sure, 'ms', '(' + (tam.sure - boost.sure) + ' ms)');
  console.log('   istek:', tam.adet, '→', boost.adet);

  // --- 3) Parlama / menü sürekliliği ---
  await p.goto(BASE + '/admin/icerikler', { waitUntil: 'networkidle2' });
  await new Promise((r) => setTimeout(r, 400));
  const parlama = await p.evaluate(async () => {
    const kareler = [];
    const z = setInterval(() => {
      kareler.push({
        renk: getComputedStyle(document.body).backgroundColor,
        side: !!document.querySelector('.admin-side'),
        hap: !!document.querySelector('.ws-links .tb-hap'),
      });
    }, 16);
    document.querySelector('.ws-links a[href="/admin/kategoriler"]').click();
    await new Promise((r) => setTimeout(r, 900));
    clearInterval(z);
    return {
      kare: kareler.length,
      beyaz: kareler.filter((k) => k.renk === 'rgb(255, 255, 255)').length,
      menusuz: kareler.filter((k) => !k.side).length,
      hapsiz: kareler.filter((k) => !k.hap).length,
    };
  });
  console.log('\nPARLAMA (' + parlama.kare + ' kare, 16 ms)');
  console.log('   beyaz kare :', parlama.beyaz, '(0 olmalı)');
  console.log('   menüsüz kare:', parlama.menusuz, '(0 olmalı)');
  console.log('   hapsız kare :', parlama.hapsiz);

  // --- 4) Kaydırma konumu korunuyor mu ---
  await p.goto(BASE + '/admin/kategoriler', { waitUntil: 'networkidle2' });
  await new Promise((r) => setTimeout(r, 400));
  await p.evaluate(() => window.scrollTo(0, 600));
  await new Promise((r) => setTimeout(r, 150));
  const kaydirmaOnce = await p.evaluate(() => Math.round(window.scrollY));
  await p.click('.ws-links a[href="/admin/icerikler"]');
  await bekleBaslik('İçerikler');
  await new Promise((r) => setTimeout(r, 300));
  const kaydirmaSonra = await p.evaluate(() => Math.round(window.scrollY));
  console.log('\nKAYDIRMA: geçiş öncesi', kaydirmaOnce, '→ sonrası', kaydirmaSonra,
    kaydirmaSonra === 0 ? '(başa döndü — show:window:top)' : '');

  await b.close();
})();
