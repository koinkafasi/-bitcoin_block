const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

// CANLI: editor.js gzip'li mi, editor calisiyor mu, boyut ne.
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 1000 });
  const hata = [];
  p.on('pageerror', (e) => hata.push(e.message));
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  const varlik = [];
  let toplamByte = 0;
  p.on('response', (r) => {
    const u = r.url();
    if (!/\/static\/.*\.(js|css)(\?|$)/.test(u)) return;
    const h = r.headers();
    varlik.push({ d: u.split('/static/')[1].split('?')[0], enc: h['content-encoding'] || '-' });
  });

  const t0 = Date.now();
  const rr = await p.goto(BASE + '/admin/icerikler/yeni', { waitUntil: 'networkidle2' });
  const sure = Date.now() - t0;
  console.log('form:', rr.status(), '· networkidle:', sure, 'ms');

  const js = varlik.filter((x) => x.d.endsWith('.js'));
  console.log('JS:', js.length, '· gzip:', js.filter((x) => x.enc === 'gzip').length,
    '· sıkışmayan:', js.filter((x) => x.enc !== 'gzip').map((x) => x.d).join(',') || 'yok');

  const ed = await p.evaluate(() => {
    const el = document.querySelector('.ProseMirror, [contenteditable="true"]');
    return { varMi: !!el, arac: document.querySelectorAll('.be-toolbar svg.ti').length };
  });
  console.log('editör:', JSON.stringify(ed));

  // Gerçek aktarılan bayt: performance API (encodedBodySize).
  const boyut = await p.evaluate(() => {
    const g = performance.getEntriesByType('resource')
      .filter((e) => /\/static\/.*\.js/.test(e.name));
    return {
      adet: g.length,
      aktarilan: g.reduce((a, e) => a + (e.encodedBodySize || e.transferSize || 0), 0),
      decoder: g.reduce((a, e) => a + (e.decodedBodySize || 0), 0),
    };
  });
  console.log('aktarılan (encoded):', (boyut.aktarilan / 1024).toFixed(0), 'KB · çözülen (decoded):', (boyut.decoder / 1024).toFixed(0), 'KB');

  console.log('JS hatası:', hata.length ? [...new Set(hata)].slice(0, 3).join(' | ') : 'yok');
  await b.close();
})();
