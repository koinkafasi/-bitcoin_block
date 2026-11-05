const puppeteer = require('puppeteer');
const BASE = process.argv[2], SID = process.argv[3];
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 1000 });
  p.on('console', (m) => console.log('  [sayfa]', m.text()));
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  await p.goto(BASE + '/admin/icerikler', { waitUntil: 'networkidle2' });
  await new Promise((r) => setTimeout(r, 600));

  const oncesi = await p.evaluate(() => {
    const ray = document.querySelector('.ws-links');
    return {
      hapId: ray && ray.dataset.hapId,
      tikVar: !!(ray && ray.__tbTik),
      izleVar: !!(ray && ray.__tbIzle),
      depo: sessionStorage.getItem('tb:hap-gecis'),
    };
  });
  console.log('tıklamadan önce:', JSON.stringify(oncesi));

  // Elle tıklama olayı gönder (navigation'ı engelle) ve depoyu oku.
  const sonrasi = await p.evaluate(() => {
    const ray = document.querySelector('.ws-links');
    const a = ray.querySelector('a[href="/admin/kategoriler"]');
    a.addEventListener('click', (e) => e.preventDefault(), { once: true });
    a.click();
    return { depo: sessionStorage.getItem('tb:hap-gecis'), hapId: ray.dataset.hapId };
  });
  console.log('tıklamadan sonra:', JSON.stringify(sonrasi));

  // Şimdi gerçek gezinme: depo dolu, yeni sayfada kullanılıyor mu?
  await p.evaluate(() => {
    const ray = document.querySelector('.ws-links');
    const a = ray.querySelector('a[href="/admin/kategoriler"]');
    sessionStorage.setItem('tb:hap-gecis', JSON.stringify({
      id: ray.dataset.hapId, l: a.offsetLeft, r: pillGenislik(ray) - (a.offsetLeft + a.offsetWidth), t: Date.now()
    }));
    function pillGenislik(r) {
      for (const c of r.children) if (c.classList.contains('tb-hap')) return c.offsetWidth;
      return 0;
    }
  });
  await p.goto(BASE + '/admin/kategoriler', { waitUntil: 'domcontentloaded' });
  const hemen = await p.evaluate(() => {
    const ray = document.querySelector('.ws-links');
    const pill = ray && ray.querySelector('.tb-hap');
    return {
      hapId: ray && ray.dataset.hapId,
      lr: pill && pill.__lr,
      clip: pill && pill.style.clipPath,
      depo: sessionStorage.getItem('tb:hap-gecis'),
    };
  });
  console.log('yeni sayfa anında:', JSON.stringify(hemen));
  await b.close();
})();
