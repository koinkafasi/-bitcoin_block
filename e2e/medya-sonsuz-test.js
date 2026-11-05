const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

// Faz 187 olcumu:
//  - ust kart / dosya sec dugmesi / tekrar alanlar kaldirildi mi
//  - kategori cipleri ve sayaclari
//  - sonsuz kaydirma: kaydirinca kart sayisi artiyor mu, sayfalama yok mu
//  - ay grubu birlesmesi (ayni etiket iki kez gorunmemeli)
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'], protocolTimeout: 90000 });
  const p = await b.newPage();
  await p.setViewport({ width: 1440, height: 1000 });
  const hata = [];
  p.on('pageerror', (e) => hata.push(e.message));
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  const dilimler = [];
  p.on('response', (res) => {
    const u = res.url();
    if (u.indexOf('/admin/dosyalar') === -1) return;
    if (u.indexOf('parca=1') !== -1) dilimler.push(res.status());
  });

  const r = await p.goto(BASE + '/admin/dosyalar', { waitUntil: 'networkidle2' });
  if (!r || r.status() !== 200) { console.log('atlandi', r && r.status()); await b.close(); return; }
  await new Promise((x) => setTimeout(x, 900));

  // 1) Tekrar alanlar gitti mi
  const gitti = await p.evaluate(() => ({
    ustKart: document.querySelectorAll('.media-upload').length,
    dosyaSecDugmesi: document.querySelectorAll('#medyaDosya').length,
    baslikAlani: document.querySelectorAll('input[name="baslik"]').length,
    altAlani: document.querySelectorAll('input[name="alt"]').length,
    captionAlani: document.querySelectorAll('input[name="caption"]').length,
    aciklamaAlani: document.querySelectorAll('textarea[name="aciklama"]').length,
    urlFormuUstte: document.querySelectorAll('.media-url').length,
    sayfalama: document.querySelectorAll('.media-pager').length,
    modalAlanlari: document.querySelectorAll('#mediaEditForm input, #mediaEditForm textarea').length,
  }));
  console.log('=== KALDIRILAN ALANLAR ===');
  console.log(JSON.stringify(gitti, null, 1));

  // 2) Cipler
  const cip = await p.evaluate(() => {
    const cipler = [...document.querySelectorAll('.media-cip')];
    return {
      adet: cipler.length,
      metinler: cipler.slice(0, 8).map((c) => c.textContent.replace(/\s+/g, ' ').trim()),
      secili: cipler.filter((c) => c.classList.contains('is-on')).map((c) => c.textContent.replace(/\s+/g, ' ').trim()),
      yukseklik: cipler[0] ? Math.round(cipler[0].getBoundingClientRect().height) : 0,
      yaricap: cipler[0] ? getComputedStyle(cipler[0]).borderRadius : '',
      gecis: cipler[0] ? getComputedStyle(cipler[0]).transition.slice(0, 60) : '',
    };
  });
  console.log('\n=== KATEGORI CIPLERI ===');
  console.log(JSON.stringify(cip, null, 1));

  // 3) Sonsuz kaydirma
  const ilk = await p.evaluate(() => ({
    kart: document.querySelectorAll('#medya-panel .media-item').length,
    grup: [...document.querySelectorAll('.media-group')].map((g) => g.dataset.grup),
    gozcu: !!document.getElementById('medyaSon'),
    toplam: (document.querySelector('.media-summary strong') || {}).textContent || '',
  }));
  console.log('\n=== ILK YUKLEME ===');
  console.log(JSON.stringify({ kart: ilk.kart, grupSayisi: ilk.grup.length, gruplar: ilk.grup, gozcu: ilk.gozcu, toplam: ilk.toplam }));

  // Sayfanin altina kaydir, birkac kez
  for (let i = 0; i < 3; i++) {
    await p.evaluate(() => window.scrollTo(0, document.body.scrollHeight));
    await new Promise((x) => setTimeout(x, 1500));
  }
  await p.evaluate(() => window.scrollTo(0, document.body.scrollHeight));
  await new Promise((x) => setTimeout(x, 1500));

  const sonra = await p.evaluate(() => {
    const gruplar = [...document.querySelectorAll('.media-group')].map((g) => g.dataset.grup);
    const etiketler = [...document.querySelectorAll('.media-group')].map((g) => g.dataset.grup);
    const tekil = new Set(etiketler);
    const sayaclar = [...document.querySelectorAll('.media-group')].map((g) => {
      const n = g.querySelectorAll('.media-item').length;
      const yazan = (g.querySelector('h3 span') || {}).textContent || '';
      return { grup: g.dataset.grup, gercek: n, yazan: Number(yazan) };
    });
    return {
      kart: document.querySelectorAll('#medya-panel .media-item').length,
      grup: gruplar.length,
      tekilGrup: tekil.size,
      sayacUyumlu: sayaclar.every((s) => s.gercek === s.yazan),
      sayaclar: sayaclar.slice(-4),
      gozcuVar: !!document.getElementById('medyaSon'),
      bittiYazisi: !!document.querySelector('.media-son-bitti'),
      ipucuKartSpan: document.querySelectorAll('.media-item[style*="grid-row-end"]').length,
    };
  });
  console.log('\n=== 4 DILIM SONRASI ===');
  console.log(JSON.stringify(sonra, null, 1));
  console.log('dilim yanitlari:', JSON.stringify(dilimler));

  console.log('\nJS hatasi:', hata.length ? [...new Set(hata)].slice(0, 3).join(' | ') : 'yok');
  await b.close();
})();
