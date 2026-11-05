const puppeteer = require('puppeteer');
const BASE = process.argv[2], SID = process.argv[3];

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 1000 });
  const hata = [];
  p.on('pageerror', (e) => hata.push(String(e.message)));
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  // 1) Eski adres yönlendiriyor mu?
  await p.goto(BASE + '/admin/merkez', { waitUntil: 'networkidle2' });
  await new Promise((r) => setTimeout(r, 400));
  console.log('[1] /admin/merkez →', p.url().replace(BASE, ''), '· başlık:',
    await p.evaluate(() => (document.querySelector('.admin-top h1') || {}).textContent));

  // 2) yan menüde bağlantı var mı?
  await p.goto(BASE + '/admin/icerikler', { waitUntil: 'networkidle2' });
  await new Promise((r) => setTimeout(r, 500));
  const menu = await p.evaluate(() => ({
    merkezLink: !!document.querySelector('.side-scroll a[href="/admin/merkez"]'),
    metin: document.querySelector('.side-scroll').textContent.includes('Site Merkezi'),
    ogeSayisi: document.querySelectorAll('.side-scroll a').length,
    ustOge: [...document.querySelectorAll('.side-scroll > ul.menu > li > a')].map((a) => a.textContent.trim()).slice(0, 6),
  }));
  console.log('[2] yan menü: merkez bağlantısı', menu.merkezLink, '· "Site Merkezi" metni', menu.metin);
  console.log('    öğe sayısı:', menu.ogeSayisi, '·', menu.ustOge.join(' | '));

  // 3) pano menüsü
  await p.goto(BASE + '/admin', { waitUntil: 'networkidle2' });
  await new Promise((r) => setTimeout(r, 500));
  const pano = await p.evaluate(() => ({
    merkezLink: !!document.querySelector('.admin-side a[href="/admin/merkez"]'),
    nav: [...document.querySelectorAll('.side-scroll a')].map((a) => a.textContent.trim()).slice(0, 12),
  }));
  console.log('[3] pano sayfası merkez bağlantısı:', pano.merkezLink);
  console.log('    menü:', pano.nav.join(' | '));

  // 4) sekme çubuğu: /admin/merkez yoksa hangi alan?
  const sekme = await p.evaluate(() => ({
    var: !!document.getElementById('wsTabs'),
    metin: (document.getElementById('wsTabs') || {}).textContent,
  }));
  console.log('[4] pano sekme çubuğu:', sekme.var, '·', (sekme.metin || '').trim().slice(0, 50));

  // 5) Yönlendirme sonrası diğer sayfalar sağlam mı — birkaç kritik sayfa
  for (const y of ['/admin', '/admin/icerikler', '/admin/ayarlar', '/admin/kategoriler', '/admin/marka', '/admin/menu']) {
    const r = await p.goto(BASE + y, { waitUntil: 'domcontentloaded' });
    console.log('[5]', y.padEnd(22), r.status());
  }

  console.log('\nJS hatası:', hata.length ? [...new Set(hata)].slice(0, 3).join(' | ') : 'yok');
  await b.close();
})();
