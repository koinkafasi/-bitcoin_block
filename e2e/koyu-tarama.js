const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

// KOYU TEMA taramasi: tum panelde esik alti metin var mi?
const SAYFALAR = [
  '/admin', '/admin/icerikler', '/admin/icerikler/yeni', '/admin/kategoriler',
  '/admin/etiketler', '/admin/yorumlar', '/admin/members', '/admin/gazete',
  '/admin/reklamlar', '/admin/ayarlar', '/admin/marka', '/admin/gorunum',
  '/admin/yazarlar', '/admin/raporlar', '/admin/tasarim', '/admin/yardim',
  '/admin/trend', '/admin/aboneler', '/admin/seo', '/admin/kumeler',
  '/admin/gelen-kutusu', '/admin/geri-bildirim', '/admin/medya',
];

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1440, height: 1000 });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  let toplam = 0;
  const bulgular = [];

  for (const yol of SAYFALAR) {
    const r = await p.goto(BASE + yol, { waitUntil: 'networkidle2' }).catch(() => null);
    if (!r || r.status() !== 200) continue;
    await p.evaluate(() => document.documentElement.setAttribute('data-theme', 'dark'));
    await new Promise((x) => setTimeout(x, 450));

    const d = await p.evaluate(() => {
      const lin = (v) => { v /= 255; return v <= 0.03928 ? v / 12.92 : Math.pow((v + 0.055) / 1.055, 2.4); };
      const pr = (c) => { const m = c.match(/rgba?\((\d+),\s*(\d+),\s*(\d+)(?:,\s*([\d.]+))?\)/); return m ? { r: +m[1], g: +m[2], b: +m[3], a: m[4] === undefined ? 1 : +m[4] } : null; };
      const lum = (c) => 0.2126 * lin(c.r) + 0.7152 * lin(c.g) + 0.0722 * lin(c.b);
      const kontrast = (a, bb) => {
        const A = pr(a), B = pr(bb);
        if (!A || !B) return null;
        const l1 = lum(A), l2 = lum(B);
        return (Math.max(l1, l2) + 0.05) / (Math.min(l1, l2) + 0.05);
      };
      const gorunur = (el) => {
        if (el.closest('details:not([open])')) return false;
        const s = getComputedStyle(el);
        return s.display !== 'none' && s.visibility !== 'hidden' && el.getClientRects().length > 0;
      };
      const out = [];
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
        if (parseFloat(s.fontSize) < 9) continue;
        let bg = null, c = el;
        while (c) {
          const cs = getComputedStyle(c);
          if (cs.backgroundColor && !/rgba\(0, 0, 0, 0\)|transparent/.test(cs.backgroundColor)) {
            const m = pr(cs.backgroundColor);
            if (m && m.a > 0.85) { bg = cs.backgroundColor; break; }
          }
          if (cs.backgroundImage && cs.backgroundImage !== 'none') break;
          c = c.parentElement;
        }
        if (!bg) continue;
        const kr = kontrast(s.color, bg);
        if (kr === null) continue;
        const kalin = parseInt(s.fontWeight, 10) >= 600;
        const buyuk = parseFloat(s.fontSize) >= 24 || (kalin && parseFloat(s.fontSize) >= 18.66);
        const esik = buyuk ? 3 : 4.5;
        if (kr < esik - 0.05) {
          const imza = el.tagName + '|' + el.className.toString().slice(0, 20) + '|' + Math.round(kr * 10) / 10;
          if (gorulen.has(imza)) continue;
          gorulen.add(imza);
          out.push({ metin: t.slice(0, 30), sinif: (el.className.toString() || el.tagName).slice(0, 26), renk: s.color, zemin: bg, oran: Math.round(kr * 10) / 10 });
        }
      }
      return { tokenText: getComputedStyle(document.documentElement).getPropertyValue('--text').trim(), out };
    });

    if (d.out.length) { toplam += d.out.length; bulgular.push({ yol, tokenText: d.tokenText, ornek: d.out.slice(0, 4) }); }
  }

  console.log('KOYU TEMA — eşik altı metin:', toplam, '| etkilenen sayfa:', bulgular.length);
  bulgular.slice(0, 10).forEach((x) => console.log('  ' + x.yol + ' --text=' + x.tokenText + ' ' + JSON.stringify(x.ornek)));
  await b.close();
})();
