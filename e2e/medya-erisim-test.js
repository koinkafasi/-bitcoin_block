const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

// Faz 187: medya ekraninda erisilebilirlik + kontrast (acik/koyu).
// Tema degisiminden SONRA beklenir: cip gecisi 0.15s, ayni karede okumak
// onceki temanin rengini veriyordu (olculdu: koyu temada 16.0 cikti).
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'], protocolTimeout: 90000 });
  const p = await b.newPage();
  await p.setViewport({ width: 1440, height: 1000 });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });
  await p.goto(BASE + '/admin/dosyalar', { waitUntil: 'networkidle2' });
  await new Promise((x) => setTimeout(x, 900));

  const erisim = await p.evaluate(() => {
    const adli = (el) => {
      if (el.getAttribute('aria-hidden') === 'true') return true; // AT'den gizli
      if (el.getAttribute('aria-label')) return true;
      if (el.getAttribute('aria-labelledby')) return true;
      if (el.id && document.querySelector('label[for="' + el.id + '"]')) return true;
      if (el.closest('label')) return true;
      if (el.type === 'hidden') return true;
      if (el.getAttribute('tabindex') === '-1' && el.type === 'file') return true;
      return false;
    };
    const alanlar = [...document.querySelectorAll('input, select, textarea')];
    const adsiz = alanlar.filter((el) => !adli(el)).map((el) => el.tagName + '[' + (el.name || el.id || el.type) + ']');

    const idler = {};
    document.querySelectorAll('[id]').forEach((el) => { idler[el.id] = (idler[el.id] || 0) + 1; });
    const cift = Object.keys(idler).filter((k) => idler[k] > 1);

    const kirikFor = [...document.querySelectorAll('label[for]')]
      .filter((l) => !document.getElementById(l.getAttribute('for')))
      .map((l) => l.getAttribute('for'));

    return { alan: alanlar.length, adsiz, cift, kirikFor };
  });
  console.log('=== ERISIM ===', JSON.stringify(erisim));

  const olc = () => p.evaluate(() => {
    const lin = (v) => { v /= 255; return v <= 0.03928 ? v / 12.92 : Math.pow((v + 0.055) / 1.055, 2.4); };
    const pr = (c) => { const m = c.match(/rgba?\(([\d.]+),\s*([\d.]+),\s*([\d.]+)/); return m ? [+m[1], +m[2], +m[3]] : null; };
    const lum = (c) => 0.2126 * lin(c[0]) + 0.7152 * lin(c[1]) + 0.0722 * lin(c[2]);
    const oran = (a, b) => { const A = pr(a), B = pr(b); if (!A || !B) return null; const l1 = lum(A), l2 = lum(B); return Math.round(((Math.max(l1, l2) + 0.05) / (Math.min(l1, l2) + 0.05)) * 10) / 10; };
    const zemin = (el) => {
      let n = el;
      while (n && n !== document.documentElement) {
        const s = getComputedStyle(n);
        if (s.backgroundColor && s.backgroundColor !== 'rgba(0, 0, 0, 0)' && s.backgroundColor !== 'transparent') return s.backgroundColor;
        n = n.parentElement;
      }
      return getComputedStyle(document.body).backgroundColor;
    };
    const tek = (sec, ad) => {
      const el = document.querySelector(sec);
      if (!el) return { ad, yok: true };
      const s = getComputedStyle(el);
      return { ad, yazi: s.color, zemin: zemin(el), oran: oran(s.color, zemin(el)) };
    };
    return [
      tek('.media-cip:not(.is-on)', 'cip (pasif)'),
      tek('.media-cip.is-on', 'cip (secili)'),
      tek('.media-cipler__baslik', 'cipler basligi'),
      tek('.media-sec > span', 'yil/ay etiketi'),
      tek('.media-son .small', 'gozcu metni'),
      tek('.media-summary .small', 'ozet ipucu'),
      tek('.media-ara__ikon', 'arama ikonu'),
      tek('.media-item .mi-cap b', 'kart basligi'),
    ];
  });

  for (const t of ['light', 'dark']) {
    await p.evaluate((tema) => document.documentElement.setAttribute('data-theme', tema), t);
    await new Promise((x) => setTimeout(x, 450));
    const r = await olc();
    console.log('\n=== KONTRAST ' + t + ' ===');
    r.forEach((x) => console.log('  ' + x.ad + ': ' + (x.yok ? 'yok' : x.oran + ' (' + x.yazi + ' / ' + x.zemin + ')')));
  }
  await b.close();
})();
