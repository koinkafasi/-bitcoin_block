const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

// --muted-fg iki temada da ÇÖZÜLMELİ (undefined ise renk miras alınır ve
// "sönük metin" hiç sönük görünmez). Public + admin × açık/koyu.
const kontrast = (a, b) => {
  const p = (c) => {
    const m = c.match(/rgba?\(([\d.]+),\s*([\d.]+),\s*([\d.]+)/);
    return m ? { r: +m[1], g: +m[2], b: +m[3] } : null;
  };
  const l = (c) => {
    const f = (v) => { v /= 255; return v <= 0.03928 ? v / 12.92 : Math.pow((v + 0.055) / 1.055, 2.4); };
    return 0.2126 * f(c.r) + 0.7152 * f(c.g) + 0.0722 * f(c.b);
  };
  const x = p(a), y = p(b);
  if (!x || !y) return null;
  return (Math.max(l(x), l(y)) + 0.05) / (Math.min(l(x), l(y)) + 0.05);
};

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1440, height: 1000, deviceScaleFactor: 2 });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  const durumlar = [
    ['admin/açık', '/admin/icerikler/yeni', 'light', true],
    ['admin/koyu', '/admin/icerikler/yeni', 'dark', true],
    ['public/açık', '/', 'light', false],
    ['public/koyu', '/', 'dark', false],
    ['kurucu/açık', '/admin/gorunum/kurucu', 'light', true],
  ];

  for (const [ad, yol, tema, adminMi] of durumlar) {
    await p.goto(BASE + yol, { waitUntil: 'networkidle2' }).catch(() => null);
    // Temayı zorla: admin temayı <html data-theme> ile taşıyor.
    await p.evaluate((t) => {
      document.documentElement.setAttribute('data-theme', t);
    }, tema);
    await new Promise((r) => setTimeout(r, 400));

    const d = await p.evaluate(() => {
      const cs = getComputedStyle(document.documentElement);
      const token = cs.getPropertyValue('--muted-fg').trim();
      // Bu token'ı kullanan gerçek bir öğe bul ve ölç.
      let ornek = null;
      for (const sec of ['.as-durum', '.be-status', '.vid-count', '.hint', '.muted', '.muted-fg',
        '.bld-empty', '.pn-kpi__foot', '.hp-meta', '.cl-row-s']) {
        const el = document.querySelector(sec);
        if (el) {
          const s = getComputedStyle(el);
          let bg = null, c = el;
          while (c && !bg) {
            const bcs = getComputedStyle(c);
            if (bcs.backgroundColor && !/rgba\(0, 0, 0, 0\)/.test(bcs.backgroundColor)) bg = bcs.backgroundColor;
            c = c.parentElement;
          }
          ornek = { sec, renk: s.color, bg: bg || 'yok' };
          break;
        }
      }
      return {
        token, tokenCozuldu: token !== '' && token !== 'var(--muted-fg)',
        gövdeRenk: getComputedStyle(document.body).color,
        tema: document.documentElement.getAttribute('data-theme'),
        ornek,
      };
    });

    const kr = d.ornek && d.ornek.bg !== 'yok' ? kontrast(d.ornek.renk, d.ornek.bg) : null;
    const metindenFarkli = d.ornek ? d.ornek.renk !== d.gövdeRenk : null;
    console.log(
      ad.padEnd(14),
      'tema=' + d.tema,
      '--muted-fg=' + (d.token || 'TANIMSIZ'),
      d.ornek ? d.ornek.sec + ' renk=' + d.ornek.renk : 'örnek yok',
      kr ? 'kontrast=' + (Math.round(kr * 10) / 10) : '',
      metindenFarkli === false ? '<<< METINDEN FARKSIZ' : '',
      d.tokenCozuldu ? '' : '<<< TOKEN COZULMEDI'
    );
  }
  await b.close();
})();
