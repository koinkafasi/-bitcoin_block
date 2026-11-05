const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

// A-7 son olcum: kucuk metin kontrasti. Logotype (marka yazisi) WCAG 1.4.3
// kapsami disinda - ayri raporlanir.
const SAYFALAR = [
  '/admin', '/admin/icerikler', '/admin/icerikler/yeni', '/admin/kategoriler',
  '/admin/etiketler', '/admin/yorumlar', '/admin/members', '/admin/gazete',
  '/admin/reklamlar', '/admin/ayarlar', '/admin/marka', '/admin/gorunum',
  '/admin/yazarlar', '/admin/raporlar', '/admin/manset-editor', '/admin/yardim',
  '/admin/trend', '/admin/aboneler', '/admin/seo', '/admin/kumeler',
];

function kontrastq(a, b) {
  const p = (c) => {
    const m = c.match(/rgba?\(([\d.]+),\s*([\d.]+),\s*([\d.]+)/);
    return m ? { r: +m[1], g: +m[2], b: +m[3] } : null;
  };
  const lin = (v) => { v /= 255; return v <= 0.03928 ? v / 12.92 : Math.pow((v + 0.055) / 1.055, 2.4); };
  const lum = (c) => 0.2126 * lin(c.r) + 0.7152 * lin(c.g) + 0.0722 * lin(c.b);
  const x = p(a), y = p(b);
  if (!x || !y) return null;
  const l1 = lum(x), l2 = lum(y);
  return (Math.max(l1, l2) + 0.05) / (Math.min(l1, l2) + 0.05);
}

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1440, height: 1000 });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  let kalan = 0, logo = 0;
  const bulgular = [];

  for (const yol of SAYFALAR) {
    const r = await p.goto(BASE + yol, { waitUntil: 'networkidle2' }).catch(() => null);
    await new Promise((x) => setTimeout(x, 300));
    if (!r || r.status() !== 200) continue;

    const d = await p.evaluate(() => {
      const gorunur = (el) => {
        const s = getComputedStyle(el);
        return s.display !== 'none' && s.visibility !== 'hidden' && el.getClientRects().length > 0;
      };
      const sonuc = [];
      const y = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT);
      let n;
      const gorulen = new Set();
      while ((n = y.nextNode())) {
        const t = n.textContent.replace(/\s+/g, ' ').trim();
        if (!t) continue;
        const el = n.parentElement;
        if (!el || !gorunur(el)) continue;
        if (['SCRIPT', 'STYLE', 'NOSCRIPT', 'OPTION'].includes(el.tagName)) continue;
        const s = getComputedStyle(el);
        let bg = null, c = el;
        while (c) {
          const cs = getComputedStyle(c);
          if (cs.backgroundColor && !/rgba\(0, 0, 0, 0\)|transparent/.test(cs.backgroundColor)) {
            const m = cs.backgroundColor.match(/rgba?\(([\d.]+),\s*([\d.]+),\s*([\d.]+)(?:,\s*([\d.]+))?\)/);
            if (m && (m[4] === undefined || +m[4] > 0.85)) { bg = cs.backgroundColor; break; }
          }
          if (cs.backgroundImage && cs.backgroundImage !== 'none') break;
          c = c.parentElement;
        }
        if (!bg) continue;
        const kr = (() => {
          const pr = (x2) => { const m = x2.match(/rgba?\(([\d.]+),\s*([\d.]+),\s*([\d.]+)/); return m ? { r: +m[1], g: +m[2], b: +m[3] } : null; };
          const lin = (v) => { v /= 255; return v <= 0.03928 ? v / 12.92 : Math.pow((v + 0.055) / 1.055, 2.4); };
          const lum = (x2) => 0.2126 * lin(x2.r) + 0.7152 * lin(x2.g) + 0.0722 * lin(x2.b);
          const A = pr(s.color), B = pr(bg);
          if (!A || !B) return null;
          const l1 = lum(A), l2 = lum(B);
          return (Math.max(l1, l2) + 0.05) / (Math.min(l1, l2) + 0.05);
        })();
        if (kr === null) continue;
        const kalin = parseInt(s.fontWeight, 10) >= 600;
        const buyuk = parseFloat(s.fontSize) >= 24 || (kalin && parseFloat(s.fontSize) >= 18.66);
        const esik = buyuk ? 3 : 4.5;
        if (kr < esik - 0.05) {
          // logotype: .brand icinde mi
          const marka = !!el.closest('.brand, .hp-logo-box, .side-head .brand');
          const imza = el.tagName + '|' + el.className.toString().slice(0, 24) + '|' + (Math.round(kr * 10) / 10);
          if (gorulen.has(imza)) continue;
          gorulen.add(imza);
          sonuc.push({ metin: t.slice(0, 34), sinif: el.className.toString().slice(0, 30), oran: Math.round(kr * 10) / 10, marka, renk: s.color, zemin: bg, etiket: el.tagName });
        }
      }
      return sonuc;
    });

    for (const x of d) {
      if (x.marka) logo++;
      else { kalan++; bulgular.push(yol + ' ' + JSON.stringify(x)); }
    }
  }

  console.log('KALAN (metin, esik alti):', kalan);
  console.log('LOGOTYPE (WCAG muaf, bilgi):', logo);
  bulgular.slice(0, 12).forEach((x) => console.log('  ' + x));
  await b.close();
})();
