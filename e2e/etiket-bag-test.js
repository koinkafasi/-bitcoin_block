const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

// A-8 dogrulamasi:
//  1) Sayfada CIFT id var mi? (literal id'yi dongude kullanmak bunu yaratir)
//  2) Her form kontrolu erisilebilir ad tasiyor mu?
//  3) <label for=X> gercekten var olan bir X'e isaret ediyor mu? (kirik bag)
const SAYFALAR = [
  '/admin', '/admin/icerikler', '/admin/icerikler/yeni', '/admin/kategoriler',
  '/admin/etiketler', '/admin/medya', '/admin/yorumlar', '/admin/members',
  '/admin/gazete', '/admin/reklamlar', '/admin/ayarlar', '/admin/marka',
  '/admin/gorunum', '/admin/yazarlar', '/admin/sistem', '/admin/ai',
  '/admin/raporlar', '/admin/manset-editor', '/admin/menuler', '/admin/yardim',
];

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1440, height: 1000 });
  const hata = [];
  p.on('pageerror', (e) => hata.push(e.message));
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  let toplamAlan = 0, adsiz = 0, kirik = 0, ciftIdSayfa = 0;
  const bulgular = [];

  for (const yol of SAYFALAR) {
    const r = await p.goto(BASE + yol, { waitUntil: 'networkidle2' }).catch(() => null);
    await new Promise((x) => setTimeout(x, 350));
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
      // 1) cift id
      const sayac = new Map();
      document.querySelectorAll('[id]').forEach((el) => sayac.set(el.id, (sayac.get(el.id) || 0) + 1));
      const cift = [...sayac.entries()].filter(([, n]) => n > 1).map(([k, n]) => k + '×' + n);

      // 2) adsiz kontroller (gizli ve readonly olmayan)
      const adsiz = [];
      document.querySelectorAll('input:not([type=hidden]), select, textarea').forEach((el) => {
        const s = getComputedStyle(el);
        if (s.display === 'none' || s.visibility === 'hidden' || !el.getClientRects().length) return;
        if (!ad(el)) adsiz.push({
          t: el.tagName.toLowerCase() + (el.type ? '[' + el.type + ']' : ''),
          n: el.getAttribute('name') || el.id || '',
        });
      });

      // 3) kirik label[for]
      const idler = new Set(Array.from(document.querySelectorAll('[id]')).map((e) => e.id));
      const kirik = [];
      document.querySelectorAll('label[for]').forEach((l) => {
        const hedef = l.getAttribute('for');
        if (!hedef) return;
        if (!idler.has(hedef)) kirik.push(hedef + ' <- ' + (l.textContent || '').trim().slice(0, 22));
      });

      return { cift, adsiz, kirik, alan: document.querySelectorAll('input:not([type=hidden]), select, textarea').length };
    });

    toplamAlan += d.alan;
    adsiz += d.adsiz.length;
    kirik += d.kirik.length;
    if (d.cift.length) ciftIdSayfa++;
    if (d.cift.length || d.adsiz.length || d.kirik.length) {
      bulgular.push({
        yol,
        cift: d.cift.slice(0, 6),
        adsiz: d.adsiz.slice(0, 6),
        kirik: d.kirik.slice(0, 6),
      });
    }
    console.log(yol.padEnd(26), r.status(), 'alan:' + String(d.alan).padStart(4),
      'ciftId:' + d.cift.length, 'adsiz:' + d.adsiz.length, 'kirikFor:' + d.kirik.length);
  }

  console.log('\n--- ÖZET ---');
  console.log('toplam alan', toplamAlan, '· adsiz', adsiz, '· kirik label[for]', kirik, '· cift id olan sayfa', ciftIdSayfa);
  if (bulgular.length) console.log(JSON.stringify(bulgular, null, 1));
  console.log('JS hatası:', hata.length ? [...new Set(hata)].slice(0, 2).join(' | ') : 'yok');
  await b.close();
})();
