const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

// Sayfa Tasarimcisi teshisi: yukseklik, kirpilan dugmeler, kontrast.
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 1000 });
  const hata = [];
  p.on('pageerror', (e) => hata.push(e.message));
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  const r = await p.goto(BASE + '/admin/tasarim', { waitUntil: 'networkidle2' }).catch(() => null);
  await new Promise((x) => setTimeout(x, 1200));
  if (!r || r.status() !== 200) { console.log('atlandı', r ? r.status() : ''); await b.close(); return; }

  for (const tema of ['light', 'dark']) {
    await p.evaluate((t) => document.documentElement.setAttribute('data-theme', t), tema);
    await new Promise((x) => setTimeout(x, 400));

    const d = await p.evaluate(() => {
      const lin = (v) => { v /= 255; return v <= 0.03928 ? v / 12.92 : Math.pow((v + 0.055) / 1.055, 2.4); };
      const pr = (c) => { const m = c.match(/rgba?\(([\d.]+),\s*([\d.]+),\s*([\d.]+)(?:,\s*([\d.]+))?\)/); return m ? { r: +m[1], g: +m[2], b: +m[3], a: m[4] === undefined ? 1 : +m[4] } : null; };
      const lum = (c) => 0.2126 * lin(c.r) + 0.7152 * lin(c.g) + 0.0722 * lin(c.b);
      const kontrast = (a, bb) => {
        const A = pr(a), B = pr(bb);
        if (!A || !B) return null;
        const l1 = lum(A), l2 = lum(B);
        return (Math.max(l1, l2) + 0.05) / (Math.min(l1, l2) + 0.05);
      };
      const gorunur = (el) => {
        const s = getComputedStyle(el);
        return s.display !== 'none' && s.visibility !== 'hidden' && el.getClientRects().length > 0;
      };

      const bloklar = [];
      ['.bld-head', '.bld-tabs', '.bld-wrap', '.bld-pal', '.bld-canvas', '.bld-views', '.bld-history', '.bld-field'].forEach((sec) => {
        const el = document.querySelector(sec);
        if (!el) return;
        const rr = el.getBoundingClientRect();
        const s = getComputedStyle(el);
        bloklar.push({
          sec, h: Math.round(rr.height), w: Math.round(rr.width), y: Math.round(rr.y),
          overflow: s.overflow, maxH: s.maxHeight,
          scrollH: el.scrollHeight,
          kirpik: el.scrollHeight > rr.height + 2,
        });
      });

      const kirpilan = [];
      document.querySelectorAll('.bld-wrap button, .bld-pal .btn, .bld-canvas .btn, .bld-views button, .bld-wrap .bld-add').forEach((el) => {
        if (!gorunur(el)) return;
        const rr = el.getBoundingClientRect();
        let c = el.parentElement;
        while (c && c !== document.body) {
          const cs = getComputedStyle(c);
          if (cs.overflow !== 'visible' && cs.overflow !== '') {
            const cr = c.getBoundingClientRect();
            if (rr.bottom > cr.bottom + 2 || rr.top < cr.top - 2) {
              kirpilan.push({
                el: el.tagName + '.' + el.className.toString().split(' ')[0],
                metin: (el.textContent || '').trim().slice(0, 26),
                kap: c.tagName + '.' + c.className.toString().split(' ')[0],
                elAlt: Math.round(rr.bottom), kapAlt: Math.round(cr.bottom),
              });
              break;
            }
          }
          c = c.parentElement;
        }
      });

      const dusuk = [];
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
          const imza = el.tagName + '|' + el.className.toString().slice(0, 22) + '|' + (Math.round(kr * 10) / 10);
          if (gorulen.has(imza)) continue;
          gorulen.add(imza);
          dusuk.push({
            metin: t.slice(0, 42),
            sinif: (el.className.toString() || el.tagName).slice(0, 34),
            renk: s.color, zemin: bg,
            oran: Math.round(kr * 10) / 10, boyut: s.fontSize,
          });
        }
      }

      return { bloklar, kirpilan: kirpilan.slice(0, 10), dusuk: dusuk.slice(0, 14), dusukSayi: dusuk.length, sayfaH: document.body.scrollHeight };
    });

    console.log('\n========== ' + tema.toUpperCase() + ' ==========');
    console.log('sayfa yüksekliği:', d.sayfaH);
    console.log('--- bloklar ---');
    d.bloklar.forEach((x) => console.log('  ' + JSON.stringify(x)));
    console.log('--- kırpılan öğeler (' + d.kirpilan.length + ') ---');
    d.kirpilan.forEach((x) => console.log('  ' + JSON.stringify(x)));
    console.log('--- düşük kontrast (' + d.dusukSayi + ') ---');
    d.dusuk.forEach((x) => console.log('  ' + JSON.stringify(x)));
    await p.screenshot({ path: `/tmp/bld-${tema}.png`, fullPage: false });
  }
  console.log('\nJS hatası:', hata.length ? [...new Set(hata)].slice(0, 2).join(' | ') : 'yok');
  await b.close();
})();
