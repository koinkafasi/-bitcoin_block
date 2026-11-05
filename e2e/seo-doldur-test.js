const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

// Faz 188: AI SEO doldurma — canli/izole olcum.
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'], protocolTimeout: 180000 });
  const p = await b.newPage();
  await p.setViewport({ width: 1440, height: 1000 });
  const hata = [];
  p.on('pageerror', (e) => hata.push(e.message));
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  // 1) Icerik listesinden ilk haberi ac
  const r = await p.goto(BASE + '/admin/icerikler', { waitUntil: 'networkidle2' });
  if (!r || r.status() !== 200) { console.log('atlandi', r && r.status()); await b.close(); return; }

  const link = await p.evaluate(() => {
    const a = [...document.querySelectorAll('a[href*="/admin/icerikler/"]')]
      .find((x) => /\/admin\/icerikler\/\d+$/.test(x.getAttribute('href') || ''));
    return a ? a.getAttribute('href') : null;
  });
  if (!link) { console.log('içerik bağlantısı yok'); await b.close(); return; }
  console.log('içerik:', link);

  await p.goto(BASE + link, { waitUntil: 'networkidle2' });
  await new Promise((x) => setTimeout(x, 900));

  // 2) SEO grubunu ac, dugmeyi bul
  const dugme = await p.evaluate(() => {
    const d = [...document.querySelectorAll('button')].find((x) => (x.textContent || '').includes('AI ile iki alanı doldur'));
    if (!d) return { var: false };
    // <details> kapaliysa ac
    const det = d.closest('details');
    if (det && !det.open) det.open = true;
    return { var: true, hedef: d.getAttribute('hx-post'), gorunur: d.getBoundingClientRect().width > 0 };
  });
  console.log('düğme:', JSON.stringify(dugme));
  if (!dugme.var) { console.log('JS hatasi:', hata.join(' | ')); await b.close(); return; }

  // 3) Mevcut SEO degerlerini not et (DB'ye yazilmadigini dogrulamak icin)
  const once = await p.evaluate(() => ({
    baslik: (document.querySelector('[name=seo_baslik]') || {}).value || '',
    aciklama: (document.querySelector('[name=seo_aciklama]') || {}).value || '',
    panelVar: !!document.querySelector('#seoOneri'),
    panelBos: (document.querySelector('#seoOneri') || {}).innerHTML === '',
  }));
  console.log('önce:', JSON.stringify(once));

  // 4) Dugmeye bas, yaniti bekle
  const basladi = Date.now();
  await p.evaluate(() => {
    const d = [...document.querySelectorAll('button')].find((x) => (x.textContent || '').includes('AI ile iki alanı doldur'));
    d.click();
  });
  let sonuc = null;
  for (let i = 0; i < 60; i++) {
    await new Promise((x) => setTimeout(x, 1000));
    sonuc = await p.evaluate(() => {
      const panel = document.querySelector('#seoOneri');
      if (!panel) return null;
      const metin = panel.querySelector('.seo-oneri__metin');
      const hata = panel.querySelector('.seo-oneri__hata');
      if (!metin && !hata) return null;
      return {
        sure: 0,
        hata: hata ? hata.textContent.trim() : '',
        baslik: metin ? metin.textContent : '',
        alanlar: [...panel.querySelectorAll('.seo-oneri__metin')].map((x) => x.textContent),
        sayaclar: [...panel.querySelectorAll('.seo-oneri__ad b')].map((x) => x.textContent),
        anahtar: [...panel.querySelectorAll('.seo-oneri__cip')].map((x) => x.textContent),
        uyarilar: [...panel.querySelectorAll('.seo-oneri__uyari')].map((x) => x.textContent.trim()),
        dugmeVar: !!panel.querySelector('[data-seo-uygula]'),
      };
    });
    if (sonuc) break;
  }
  if (sonuc) sonuc.sure = Date.now() - basladi;
  console.log('\n=== ÖNERİ ===');
  console.log(JSON.stringify(sonuc, null, 1));

  // 5) Uygula: iki alan dolmali
  if (sonuc && sonuc.dugmeVar) {
    await p.evaluate(() => document.querySelector('[data-seo-uygula]').click());
    await new Promise((x) => setTimeout(x, 400));
    const sonra = await p.evaluate(() => ({
      baslik: (document.querySelector('[name=seo_baslik]') || {}).value || '',
      aciklama: (document.querySelector('[name=seo_aciklama]') || {}).value || '',
      serpBaslik: (document.querySelector('.serp .t') || {}).textContent || '',
      toast: (document.querySelector('.pn-toast') || {}).textContent || '',
    }));
    console.log('\n=== UYGULANDI ===');
    console.log(JSON.stringify(sonra, null, 1));
    console.log('serp guncel mi:', sonra.serpBaslik === sonra.baslik);
  }

  console.log('\nJS hatasi:', hata.length ? [...new Set(hata)].slice(0, 3).join(' | ') : 'yok');
  await b.close();
})();
