const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

// Faz 185: secim balonu sayfa sonunda gorunmemeli; secim yapinca cikmali.
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'], protocolTimeout: 90000 });
  const p = await b.newPage();
  await p.setViewport({ width: 1440, height: 900 });
  const hata = [];
  p.on('pageerror', (e) => hata.push(e.message));
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });
  await p.goto(BASE + '/admin/icerikler/yeni', { waitUntil: 'networkidle2' });
  await new Promise((x) => setTimeout(x, 3500));

  const kapali = await p.evaluate(() => {
    const el = document.querySelector('.be-bubble');
    if (!el) return { yok: true };
    const s = getComputedStyle(el);
    const r = el.getBoundingClientRect();
    return {
      konum: s.position,
      gorunurluk: s.visibility,
      display: s.display,
      w: Math.round(r.width),
      h: Math.round(r.height),
      gorunurMu: s.visibility !== 'hidden' && s.display !== 'none' && r.width > 2,
      sayfaSonundaMi: Math.round(r.y) > document.documentElement.scrollHeight - 120,
    };
  });

  // Editore yazi yaz ve sec
  const yazildi = await p.evaluate(() => {
    const ed = document.querySelector('#blockEditor .ProseMirror') || document.querySelector('.ProseMirror');
    if (!ed) return false;
    ed.focus();
    document.execCommand('insertText', false, 'Secim balonu denemesi icin bir cumle.');
    const sel = window.getSelection();
    const r = document.createRange();
    const t = ed.querySelector('p') || ed;
    if (t.firstChild) r.setStart(t.firstChild, 0); else r.selectNodeContents(t);
    r.setEnd(t.firstChild || t, Math.min(6, (t.textContent || '').length));
    sel.removeAllRanges();
    sel.addRange(r);
    ed.dispatchEvent(new Event('selectionchange', { bubbles: true }));
    return true;
  });
  await new Promise((x) => setTimeout(x, 900));

  const acik = await p.evaluate(() => {
    const el = document.querySelector('.be-bubble');
    if (!el) return { yok: true };
    const s = getComputedStyle(el);
    const r = el.getBoundingClientRect();
    return {
      gorunurluk: s.visibility,
      display: s.display,
      w: Math.round(r.width),
      h: Math.round(r.height),
      x: Math.round(r.x),
      y: Math.round(r.y),
      gorunurMu: s.visibility !== 'hidden' && s.display !== 'none' && r.width > 2,
      tamGenislikMi: r.width > window.innerWidth * 0.85,
      dugme: el.querySelectorAll('button').length,
    };
  });

  console.log('KAPALI (yükleme sonrası):', JSON.stringify(kapali));
  console.log('yazıldı:', yazildi);
  console.log('AÇIK (seçim sonrası):   ', JSON.stringify(acik));
  console.log('JS hatası:', hata.length ? [...new Set(hata)].slice(0, 2).join(' | ') : 'yok');
  await b.close();
})();
