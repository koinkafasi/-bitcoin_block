const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

const SAYFALAR = [
  '/admin', '/admin/icerikler', '/admin/icerikler/yeni', '/admin/kategoriler',
  '/admin/etiketler', '/admin/medya', '/admin/yorumlar', '/admin/members',
  '/admin/gazete', '/admin/reklamlar', '/admin/ayarlar', '/admin/yardim',
];

function kontrast(a, b) {
  const p = (c) => {
    const m = c.match(/rgba?\(([\d.]+),\s*([\d.]+),\s*([\d.]+)(?:,\s*([\d.]+))?\)/);
    if (!m) return null;
    return { r: +m[1], g: +m[2], b: +m[3], a: m[4] === undefined ? 1 : +m[4] };
  };
  const l = (c) => {
    const f = (v) => { v /= 255; return v <= 0.03928 ? v / 12.92 : Math.pow((v + 0.055) / 1.055, 2.4); };
    return 0.2126 * f(c.r) + 0.7152 * f(c.g) + 0.0722 * f(c.b);
  };
  const x = p(a), y = p(b);
  if (!x || !y) return null;
  const l1 = l(x), l2 = l(y);
  return (Math.max(l1, l2) + 0.05) / (Math.min(l1, l2) + 0.05);
}

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1440, height: 1000 });
  const jsHata = [];
  p.on('pageerror', (e) => jsHata.push(e.message));
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  const toplam = { adsiz: [], ikonsuz: [], ciftId: [], dusukKontrast: [], tasma: [] };

  for (const yol of SAYFALAR) {
    const r = await p.goto(BASE + yol, { waitUntil: 'networkidle2' }).catch(() => null);
    await new Promise((x) => setTimeout(x, 450));
    if (!r || r.status() !== 200) { console.log('atlandı', yol); continue; }

    const d = await p.evaluate(() => {
      const kontrast = (a, b) => {
        const p = (c) => {
          const m = c.match(/rgba?\(([\d.]+),\s*([\d.]+),\s*([\d.]+)(?:,\s*([\d.]+))?\)/);
          if (!m) return null;
          return { r: +m[1], g: +m[2], b: +m[3], a: m[4] === undefined ? 1 : +m[4] };
        };
        const l = (c) => {
          const f = (v) => { v /= 255; return v <= 0.03928 ? v / 12.92 : Math.pow((v + 0.055) / 1.055, 2.4); };
          return 0.2126 * f(c.r) + 0.7152 * f(c.g) + 0.0722 * f(c.b);
        };
        const x = p(a), y = p(b);
        if (!x || !y) return null;
        const l1 = l(x), l2 = l(y);
        return (Math.max(l1, l2) + 0.05) / (Math.min(l1, l2) + 0.05);
      };
      const gorunur = (el) => {
        const s = getComputedStyle(el);
        return s.display !== 'none' && s.visibility !== 'hidden' && el.getClientRects().length > 0;
      };
      const ad = (el) => {
        const aria = el.getAttribute('aria-label');
        if (aria && aria.trim()) return aria.trim();
        const lb = el.getAttribute('aria-labelledby');
        if (lb) {
          const t = lb.split(/\s+/).map((id) => document.getElementById(id)?.textContent || '').join(' ').trim();
          if (t) return t;
        }
        if (el.id) {
          const l = document.querySelector('label[for="' + CSS.escape(el.id) + '"]');
          if (l && l.textContent.trim()) return l.textContent.trim();
        }
        const anc = el.closest('label');
        if (anc && anc.textContent.trim()) return anc.textContent.trim();
        const t = el.getAttribute('title');
        if (t && t.trim()) return t.trim();
        return '';
      };

      const adsiz = [];
      document.querySelectorAll('input:not([type=hidden]), select, textarea').forEach((el) => {
        if (!gorunur(el)) return;
        if (!ad(el)) adsiz.push({
          tur: el.tagName.toLowerCase() + (el.type ? '[' + el.type + ']' : ''),
          ad: el.getAttribute('name') || el.id || '',
          yer: el.closest('form')?.getAttribute('action') || '',
        });
      });

      const ikonsuz = [];
      document.querySelectorAll('a, button').forEach((el) => {
        if (!gorunur(el)) return;
        const metin = (el.textContent || '').replace(/\s+/g, ' ').trim();
        if (metin) return;
        if (ad(el)) return;
        const img = el.querySelector('img[alt]:not([alt=""])');
        if (img) return;
        ikonsuz.push({ tur: el.tagName.toLowerCase(), sinif: el.className.toString().slice(0, 40), href: el.getAttribute('href') || '' });
      });

      const sayac = new Map();
      document.querySelectorAll('[id]').forEach((el) => sayac.set(el.id, (sayac.get(el.id) || 0) + 1));
      const ciftId = [...sayac.entries()].filter(([, n]) => n > 1).map(([id, n]) => id + '×' + n);

      // Kontrast: yalnız düz metin, arka planı saydam olmayan atalardan.
      const dusuk = [];
      const yuruyus = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT);
      let node;
      const gorulen = new Set();
      while ((node = yuruyus.nextNode())) {
        const t = node.textContent.replace(/\s+/g, ' ').trim();
        if (!t) continue;
        const el = node.parentElement;
        if (!el || !gorunur(el)) continue;
        if (['SCRIPT', 'STYLE', 'NOSCRIPT', 'OPTION'].includes(el.tagName)) continue;
        const s = getComputedStyle(el);
        if (parseFloat(s.fontSize) < 10) continue;
        // Arka planı bul
        let bg = null, cur = el;
        while (cur) {
          const cs = getComputedStyle(cur);
          if (cs.backgroundColor && !/rgba\(0, 0, 0, 0\)|transparent/.test(cs.backgroundColor)) {
            const m = cs.backgroundColor.match(/rgba?\(([\d.]+),\s*([\d.]+),\s*([\d.]+)(?:,\s*([\d.]+))?\)/);
            if (m && (m[4] === undefined || +m[4] > 0.85)) { bg = cs.backgroundColor; break; }
          }
          if (cs.backgroundImage && cs.backgroundImage !== 'none') break;
          cur = cur.parentElement;
        }
        if (!bg) continue;
        const cr = kontrast(s.color, bg);
        if (cr === null) continue;
        const kalin = parseInt(s.fontWeight, 10) >= 600;
        const buyuk = parseFloat(s.fontSize) >= 24 || (kalin && parseFloat(s.fontSize) >= 18.66);
        const esik = buyuk ? 3 : 4.5;
        if (cr < esik - 0.05) {
          const imza = el.tagName + '|' + (el.className.toString().slice(0, 30)) + '|' + Math.round(cr * 10) / 10;
          if (gorulen.has(imza)) continue;
          gorulen.add(imza);
          dusuk.push({ metin: t.slice(0, 45), sinif: el.className.toString().slice(0, 45), oran: Math.round(cr * 10) / 10, esik });
        }
      }

      const tasma = [];
      document.querySelectorAll('main *, .admin-main *').forEach((el) => {
        if (!gorunur(el)) return;
        const r = el.getBoundingClientRect();
        if (r.width > 0 && r.right > window.innerWidth + 2) {
          tasma.push({ etiket: el.tagName.toLowerCase() + '.' + el.className.toString().split(' ')[0], sag: Math.round(r.right) });
        }
      });

      return { adsiz, ikonsuz, ciftId, dusuk, tasma: tasma.slice(0, 4) };
    });

    const ekle = (k, v) => { if (v.length) toplam[k].push(yol + ' → ' + JSON.stringify(v)); };
    ekle('adsiz', d.adsiz);
    ekle('ikonsuz', d.ikonsuz);
    ekle('ciftId', d.ciftId);
    ekle('dusukKontrast', d.dusuk.slice(0, 6));
    ekle('tasma', d.tasma);
  }

  for (const k of Object.keys(toplam)) {
    console.log('\n=== ' + k + ' (' + toplam[k].length + ' sayfa) ===');
    toplam[k].slice(0, 10).forEach((x) => console.log('  ' + x));
  }
  console.log('\nJS hatası:', jsHata.length ? [...new Set(jsHata)].slice(0, 3).join(' | ') : 'yok');
  await b.close();
})();
