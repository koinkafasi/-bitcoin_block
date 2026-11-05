const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

// Faz 188b: alan adi korumasi canli dogrulama.
// Icerik metninde gecmeyen bir site adi SEO alanlarina girmemeli.
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'], protocolTimeout: 180000 });
  const p = await b.newPage();
  await p.setViewport({ width: 1440, height: 1000 });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  await p.goto(BASE + '/admin/icerikler', { waitUntil: 'networkidle2' });
  const link = await p.evaluate(() => {
    const a = [...document.querySelectorAll('a[href*="/admin/icerikler/"]')]
      .find((x) => /\/admin\/icerikler\/\d+$/.test(x.getAttribute('href') || ''));
    return a ? a.getAttribute('href') : null;
  });
  if (!link) { console.log('içerik yok'); await b.close(); return; }
  await p.goto(BASE + link, { waitUntil: 'networkidle2' });
  await new Promise((x) => setTimeout(x, 800));

  // Basligi ve govdeyi not et (kaynak metin)
  const kaynak = await p.evaluate(() => ({
    baslik: (document.querySelector('#f_title') || {}).value || '',
    spot: (document.querySelector('#f_spot') || {}).value || '',
    govdeUzunluk: (document.querySelector('#blockEditor') || {}).textContent?.length || 0,
  }));

  await p.evaluate(() => {
    const d = [...document.querySelectorAll('button')].find((x) => (x.textContent || '').includes('AI ile iki alanı doldur'));
    const det = d.closest('details');
    if (det && !det.open) det.open = true;
    d.click();
  });

  let sonuc = null;
  for (let i = 0; i < 60; i++) {
    await new Promise((x) => setTimeout(x, 1000));
    sonuc = await p.evaluate(() => {
      const panel = document.querySelector('#seoOneri');
      if (!panel) return null;
      const metinler = [...panel.querySelectorAll('.seo-oneri__metin')].map((x) => x.textContent);
      const hata = panel.querySelector('.seo-oneri__hata');
      if (!metinler.length && !hata) return null;
      return {
        hata: hata ? hata.textContent.trim() : '',
        metinler,
        uyarilar: [...panel.querySelectorAll('.seo-oneri__uyari')].map((x) => x.textContent.trim()),
      };
    });
    if (sonuc) break;
  }

  console.log('kaynak başlık:', kaynak.baslik.slice(0, 70));
  console.log('öneri:', JSON.stringify(sonuc, null, 1));

  if (sonuc && sonuc.metinler.length) {
    const hepsi = sonuc.metinler.join(' ').toLowerCase();
    const siteler = [...new Set((hepsi.match(/[a-z0-9-]+(?:\.[a-z0-9-]+)*\.(?:com|net|org|gov)(?:\.[a-z]{2})?/g) || []))];
    const kaynakKucuk = (kaynak.baslik + ' ' + kaynak.spot).toLowerCase();
    const uydurma = siteler.filter((s) => !kaynakKucuk.includes(s));
    console.log('\nSEO alanlarındaki alan adları:', JSON.stringify(siteler));
    console.log('kaynak metinde geçmeyen (uydurma):', JSON.stringify(uydurma));
    console.log('SONUC:', uydurma.length === 0 ? 'TEMIZ ✓' : 'UYDURMA VAR ✗');
  }
  await b.close();
})();
