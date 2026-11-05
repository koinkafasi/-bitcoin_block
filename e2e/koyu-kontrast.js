const puppeteer = require('puppeteer');

// Karanlık temada yıkıcı düğme + form alanı kontrastı.
const BASE = process.argv[2];
const SID = process.argv[3];

const parlaklik = (rgb) => {
  const [r, g, b] = rgb.match(/\d+/g).map(Number).map((v) => v / 255);
  const f = (c) => (c <= 0.03928 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4));
  return 0.2126 * f(r) + 0.7152 * f(g) + 0.0722 * f(b);
};

const kontrast = (a, b) => {
  const [x, y] = [parlaklik(a), parlaklik(b)].sort((p, q) => q - p);
  return ((x + 0.05) / (y + 0.05)).toFixed(2);
};

(async () => {
  const br = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await br.newPage();
  await p.setViewport({ width: 1500, height: 1000 });
  await p.emulateMediaFeatures([{ name: 'prefers-color-scheme', value: 'dark' }]);
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });
  await p.goto(BASE + '/admin/icerikler', { waitUntil: 'networkidle2' });
  // Tema `[data-theme="dark"]` ile açılıyor (medya sorgusu değil).
  await p.evaluate(() => document.documentElement.setAttribute('data-theme', 'dark'));
  await new Promise((r) => setTimeout(r, 400));

  const veri = await p.evaluate(() => {
    const top = (el) => {
      let e = el, bg = null;
      while (e && e !== document.documentElement) {
        const c = getComputedStyle(e).backgroundColor;
        if (c && c !== 'rgba(0, 0, 0, 0)') { bg = c; break; }
        e = e.parentElement;
      }
      return bg || 'rgb(0, 0, 0)';
    };
    const d = document.querySelector('.btn.danger');
    const i = [...document.querySelectorAll('.input')].find((e) => e.offsetHeight > 0);
    const out = {};
    if (d) out.danger = { color: getComputedStyle(d).color, zemin: top(d) };
    if (i) out.input = { renk: getComputedStyle(i).color, zemin: top(i), yuk: Math.round(i.getBoundingClientRect().height) };
    out.tema = document.documentElement.getAttribute('data-theme');
    return out;
  });

  if (veri.danger) {
    console.log('yıkıcı düğme kontrastı:', kontrast(veri.danger.color, veri.danger.zemin),
      '·', veri.danger.color, 'üstü', veri.danger.zemin);
  }
  if (veri.input) {
    console.log('form alanı kontrastı:', kontrast(veri.input.renk, veri.input.zemin),
      '· yükseklik', veri.input.yuk);
  }  console.log('tema özniteliği:', veri.tema);
  await p.screenshot({ path: '/tmp/koyu-icerikler.png' });
  await br.close();
})();
