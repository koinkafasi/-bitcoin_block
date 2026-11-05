const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

// Tarih alanlari grid icinde "etiket" olarak placeholder kullaniyor. Yerel
// tarih alaninda placeholder HIC cizilmez; alan etiketsiz mi kaliyor?
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox', '--lang=en-US'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1440, height: 1100, deviceScaleFactor: 2 });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  const sayfalar = ['/admin/araclar', '/admin/sinemalar', '/admin/reklamlar', '/admin/gazete'];
  for (const yol of sayfalar) {
    const r = await p.goto(BASE + yol, { waitUntil: 'networkidle2' }).catch(() => null);
    await new Promise((x) => setTimeout(x, 500));
    if (!r || r.status() !== 200) { console.log('atlandı', yol, r ? r.status() : ''); continue; }

    const d = await p.evaluate(() => {
      const out = [];
      document.querySelectorAll('input[type=date], input[type=datetime-local]').forEach((el) => {
        const r = el.getBoundingClientRect();
        const s = getComputedStyle(el);
        // Ayni satirdaki komsu alanlarin placeholder'lari gorunuyor mu?
        const komsu = [];
        const kap = el.parentElement;
        if (kap) {
          kap.querySelectorAll('input, select').forEach((o) => {
            if (o === el) return;
            komsu.push((o.getAttribute('placeholder') || o.getAttribute('aria-label') || o.tagName).slice(0, 22));
          });
        }
        out.push({
          name: el.getAttribute('name'),
          ph: el.getAttribute('placeholder'),
          aria: el.getAttribute('aria-label'),
          title: el.getAttribute('title'),
          // Yerel alan: degeri bos olsa da tarayici kendi ipucu parcalarini cizer
          deger: el.value,
          genislik: Math.round(r.width),
          gorunur: s.display !== 'none' && r.width > 0,
          komsuPlaceholder: komsu,
        });
      });
      return out;
    });

    console.log('\n### ' + yol + '  (' + d.length + ' tarih alani)');
    d.forEach((x) => {
      console.log('  name=' + String(x.name).padEnd(12),
        'ph=' + JSON.stringify(x.ph),
        'aria=' + JSON.stringify(x.aria),
        'title=' + JSON.stringify(x.title),
        'w=' + x.genislik,
        x.aria || x.ph ? '' : '<<< ETIKETSIZ');
    });

    // Formu cek
    const el = await p.$('input[type=date]');
    if (el) {
      const box = await el.evaluateHandle((e) => e.closest('form') || e.parentElement);
      try {
        const h = await box.asElement();
        const clip = await h.boundingBox();
        if (clip) {
          await p.screenshot({
            path: `/tmp/tarih-${yol.replace(/\//g, '-')}.png`,
            clip: { x: Math.max(0, clip.x - 6), y: Math.max(0, clip.y - 6), width: Math.min(1440, clip.width + 12), height: Math.min(700, clip.height + 12) },
          });
        }
      } catch (e) {}
    }
  }
  await b.close();
})();
