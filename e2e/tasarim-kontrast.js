const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1440, height: 1000 });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });
  await p.goto(BASE + '/admin/tasarim', { waitUntil: 'networkidle2' });
  // Blok önizlemeleri bitsin: yüklenirken rozetlerin zemini henüz boyanmamış
  // oluyor ve "beyaz üstüne beyaz" gibi yanlış bulgular çıkıyordu.
  for (let i = 0; i < 30; i++) {
    const hazir = await p.evaluate(() => {
      const lar = Array.from(document.querySelectorAll('.bp-prev'));
      return lar.length > 0 && lar.every((el) => !el.classList.contains('bp-prev-loading'));
    });
    if (hazir) break;
    await new Promise((x) => setTimeout(x, 400));
  }
  await new Promise((x) => setTimeout(x, 1200));

  // Tum details'leri ac ki gizli alanlar da olculsun
  await p.evaluate(() => document.querySelectorAll('details').forEach((d) => { d.open = true; }));
  await new Promise((x) => setTimeout(x, 500));

  for (const tema of ['light', 'dark']) {
    await p.evaluate((t) => document.documentElement.setAttribute('data-theme', t), tema);
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
          const imza = el.tagName + '|' + el.className.toString().slice(0, 24) + '|' + (Math.round(kr * 10) / 10);
          if (gorulen.has(imza)) continue;
          gorulen.add(imza);
          dusuk.push({
            metin: t.slice(0, 40), sinif: (el.className.toString() || el.tagName).slice(0, 32),
            renk: s.color, zemin: bg, oran: Math.round(kr * 10) / 10, boyut: s.fontSize,
            ic: !!el.closest('.bp-prev'),
          });
        }
      }

      // Tasma (acik panellerde)
      const tasan = [];
      [['#plPanel', '.ly-grid'], ['#bpTokens', '.ly-grid'], ['#ptPanel', '.ly-grid']].forEach(([panel, grid]) => {
        const pEl = document.querySelector(panel);
        if (!pEl) return;
        const pr2 = pEl.getBoundingClientRect();
        pEl.querySelectorAll('input, select, textarea, button').forEach((el) => {
          const r = el.getBoundingClientRect();
          if (r.width < 1 || !gorunur(el)) return;
          if (r.right > pr2.right + 2) tasan.push({ panel, el: el.id || el.tagName, fazla: Math.round(r.right - pr2.right) });
        });
      });
      return { dusuk: dusuk.slice(0, 16), dusukSayi: dusuk.length, tasan: tasan.slice(0, 8), yatay: document.documentElement.scrollWidth - window.innerWidth };
    });

    // Panelin KENDI arayuzu ile blok ONIZLEMESI ayri degerlendirilir:
    // onizleme gercek sitenin isaretlemesi, stilleri public.css'ten geliyor.
    const kendi = d.dusuk.filter((x) => !x.ic);
    const onizleme = d.dusuk.filter((x) => x.ic);
    console.log('\n===== ' + tema.toUpperCase() + ' =====');
    console.log('  panel arayüzü düşük kontrast:', kendi.length,
      '| önizleme içi:', onizleme.length,
      '| yatay taşma:', d.yatay,
      '| panel taşması:', d.tasan.length ? JSON.stringify(d.tasan) : 'yok');
    kendi.forEach((x) => console.log('   PANEL ' + JSON.stringify(x)));
    onizleme.forEach((x) => console.log('   ÖNİZLEME ' + JSON.stringify(x)));
  }
  await b.close();
})();
