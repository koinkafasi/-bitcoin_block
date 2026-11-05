const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

// Icerik formu: gercek agirlik + editorun calistigi + hata olmadigi.
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 1000 });
  const hata = [];
  p.on('pageerror', (e) => hata.push(e.message));
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  const kaynak = [];
  p.on('response', async (r) => {
    const u = r.url();
    // Varlıklar `?v=...` ile isteniyor; sorguyu da kabul et.
    if (!/\/static\/.*\.(js|css)(\?|$)/.test(u)) return;
    const h = r.headers();
    kaynak.push({
      dosya: u.split('/static/')[1].split('?')[0],
      enc: h['content-encoding'] || '-',
      len: Number(h['content-length'] || 0),
      durum: r.status(),
    });
  });

  const t0 = Date.now();
  const r = await p.goto(BASE + '/admin/icerikler/yeni', { waitUntil: 'networkidle2' });
  const yuklenme = Date.now() - t0;
  console.log('form:', r.status(), '· yüklenme:', yuklenme, 'ms');

  const duzenleyici = await p.evaluate(() => {
    const el = document.querySelector('.ProseMirror, .tiptap, [contenteditable="true"]');
    return {
      varMi: !!el,
      duzenlenebilir: el ? el.getAttribute('contenteditable') : null,
      // Araç çubuğu simgeleri sprite'tan mi (Faz 161 korunmuş mu)
      aracSimge: document.querySelectorAll('[class*="toolbar"] svg.ti, [class*="arac"] svg.ti').length,
      aracCubugu: (() => { const t = document.querySelector('[class*="toolbar"]'); return t ? t.className : null; })(),
      metin: el ? (el.textContent || '').trim().slice(0, 30) : null,
    };
  });
  console.log('editör:', JSON.stringify(duzenleyici));

  // Diske yaz: gerçekten düzenlenebiliyor mu?
  if (duzenleyici.varMi) {
    await p.click('.ProseMirror, [contenteditable="true"]');
    await p.keyboard.type('gzip testi', { delay: 15 });
    await new Promise((x) => setTimeout(x, 250));
    const yazildi = await p.evaluate(() => {
      const el = document.querySelector('.ProseMirror, [contenteditable="true"]');
      return (el.textContent || '').includes('gzip testi');
    });
    console.log('yazı yazılabiliyor:', yazildi);
  }

  const js = kaynak.filter((k) => k.dosya.endsWith('.js'));
  const toplam = js.reduce((a, k) => a + k.len, 0);
  const gzipli = js.filter((k) => k.enc === 'gzip').length;
  console.log('JS dosyası:', js.length, '· gzip:', gzipli, '· giden bayt (content-length toplamı):', toplam);
  const gziplenmemis = js.filter((k) => k.enc !== 'gzip').map((k) => k.dosya);
  console.log('sıkıştırılmayan:', gziplenmemis.length ? JSON.stringify(gziplenmemis) : 'yok');
  console.log('JS hatası:', hata.length ? [...new Set(hata)].slice(0, 3).join(' | ') : 'yok');
  await b.close();
})();
