const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

// Tum ekranlarda tarih alanlarini topla: genislik, kapsayici display,
// aria-label. 900+ px genislik flex satirinda tasma isaretidir.
const SAYFALAR = [
  '/admin/icerikler', '/admin/icerikler/yeni', '/admin/gazeteler', '/admin/sinemalar',
  '/admin/resmi-ilanlar', '/admin/reklamlar', '/admin/gazete', '/admin/etkinlikler',
  '/admin/akis', '/admin/topluluk',
];

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1440, height: 1000 });
  const hata = [];
  p.on('pageerror', (e) => hata.push(e.message));
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  let toplam = 0, adsiz = 0, genis = 0, placeholder = 0;
  for (const yol of SAYFALAR) {
    const r = await p.goto(BASE + yol, { waitUntil: 'networkidle2' }).catch(() => null);
    await new Promise((x) => setTimeout(x, 300));
    if (!r || r.status() !== 200) { console.log(yol.padEnd(26), 'atlandı', r ? r.status() : ''); continue; }

    const d = await p.evaluate(() => {
      const ad = (el) => {
        const a = el.getAttribute('aria-label');
        if (a && a.trim()) return a.trim();
        const lb = el.getAttribute('aria-labelledby');
        if (lb) {
          const t = lb.split(/\s+/).map((i) => document.getElementById(i)?.textContent || '').join(' ').trim();
          if (t) return t;
        }
        if (el.id) {
          const l = document.querySelector('label[for="' + CSS.escape(el.id) + '"]');
          if (l && l.textContent.trim()) return l.textContent.trim();
        }
        const anc = el.closest('label');
        if (anc && anc.textContent.trim()) return anc.textContent.trim();
        return '';
      };
      return Array.from(document.querySelectorAll('input[type=date], input[type=datetime-local]')).map((el) => {
        const r = el.getBoundingClientRect();
        const kap = el.parentElement;
        const ks = kap ? getComputedStyle(kap) : null;
        return {
          n: el.getAttribute('name'),
          w: Math.round(r.width),
          kap: ks ? ks.display : '?',
          aria: ad(el),
          ph: el.getAttribute('placeholder'),
          gorunur: r.width > 0,
        };
      });
    });
    const kotu = [];
    d.forEach((x) => {
      toplam++;
      if (x.gorunur && !x.aria) { adsiz++; kotu.push('ETİKETSİZ:' + x.n); }
      if (x.ph) { placeholder++; kotu.push('PLACEHOLDER:' + x.n); }
      if (x.w > 400) { genis++; kotu.push('GENİŞ:' + x.n + '=' + x.w + 'px/' + x.kap); }
    });
    console.log(yol.padEnd(26), r.status(), 'tarih:', d.length,
      d.length ? d.map((x) => x.n + '=' + x.w).join(' ') : '', kotu.length ? ' <<< ' + kotu.join(' ') : '');
  }
  console.log('\nTOPLAM', toplam, '· etiketsiz', adsiz, '· placeholder', placeholder, '· taşan', genis);
  console.log('JS hatası:', hata.length ? hata[0] : 'yok');
  await b.close();
})();
