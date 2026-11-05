const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

// "Ikon gorunmuyor" sikayeti icin ikinci gecis: ikonu OLMASI GEREKEN ama
// bos kalan dugmeler, editörun kendi araclari ve ikon-font kullanan yerler.
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 1100 });
  const hata = [];
  p.on('pageerror', (e) => hata.push(e.message));
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  const r = await p.goto(BASE + '/admin/icerikler/yeni', { waitUntil: 'networkidle2' });
  await new Promise((x) => setTimeout(x, 2500)); // editor.js + ai_editor.js

  const d = await p.evaluate(() => {
    const out = {};

    // 1) Ikonu olmasi gereken ama BOS dugmeler
    const bosDugme = [];
    document.querySelectorAll('button, a.btn, .btn').forEach((el) => {
      const metin = (el.textContent || '').replace(/\s+/g, ' ').trim();
      const svg = el.querySelector('svg');
      if (metin) return;
      if (svg) return;
      const r = el.getBoundingClientRect();
      if (r.width < 4 || r.height < 4) return;
      bosDugme.push({
        sinif: el.className.toString().slice(0, 40),
        id: el.id || '',
        title: el.getAttribute('title') || '',
        aria: el.getAttribute('aria-label') || '',
        boyut: Math.round(r.width) + 'x' + Math.round(r.height),
        html: el.innerHTML.slice(0, 60),
      });
    });
    out.bosDugme = bosDugme.slice(0, 12);

    // 2) Editör araclari: TipTap toolbar / slash menu / bubble menu
    const editorlar = [];
    ['.be-toolbar', '.block-editor', '.tiptap', '.ProseMirror', '.be-status', '.editor-toolbar', '.ed-toolbar'].forEach((sec) => {
      const el = document.querySelector(sec);
      if (!el) return;
      editorlar.push({
        sec,
        var: true,
        ikon: el.querySelectorAll('svg').length,
        dugme: el.querySelectorAll('button').length,
        metin: (el.textContent || '').replace(/\s+/g, ' ').trim().slice(0, 50),
      });
    });
    out.editorlar = editorlar;

    // 3) Ikon fontu kullanan siniflar (glyph bekleyen)
    const fontIkon = [];
    document.querySelectorAll('[class*="icon-"], [class*="ico-"], [class*="ti-"]:not(svg):not(use)').forEach((el) => {
      if (el.tagName === 'use' || el.tagName === 'svg') return;
      if (el.querySelector('svg')) return;
      const s = getComputedStyle(el);
      if (!/icon|ti-|ico-/.test(s.fontFamily) && !/icon/.test(el.className.toString())) return;
      fontIkon.push({ sinif: el.className.toString().slice(0, 50), metin: (el.textContent || '').trim().slice(0, 20), font: s.fontFamily.slice(0, 40) });
    });
    out.fontIkon = fontIkon.slice(0, 8);

    // 4) Yüklenen fontlar
    out.fontlar = Array.from(document.fonts || []).map((f) => f.family + ':' + f.status).slice(0, 10);

    // 5) Sprite gercekten kullanilabilir mi: bir sembol cizelim
    const deneme = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
    deneme.setAttribute('width', '24');
    deneme.setAttribute('height', '24');
    deneme.style.cssText = 'position:fixed;left:-9999px';
    const use = document.createElementNS('http://www.w3.org/2000/svg', 'use');
    use.setAttribute('href', '/static/vendor/tabler-icons.svg#ti-news');
    deneme.appendChild(use);
    document.body.appendChild(deneme);
    const kutu = deneme.getBBox ? deneme.getBBox() : null;
    out.sembolTest = kutu ? { w: Math.round(kutu.width), h: Math.round(kutu.height) } : 'bbox yok';
    deneme.remove();

    return out;
  });

  console.log(JSON.stringify(d, null, 1));
  console.log('JS hatası:', hata.length ? [...new Set(hata)].slice(0, 3).join(' | ') : 'yok');
  await p.screenshot({ path: '/tmp/editor-tam.png', fullPage: false });
  await b.close();
})();
