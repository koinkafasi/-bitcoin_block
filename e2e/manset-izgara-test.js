const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

// Faz 178 dogrulamasi: "Izgara" dugmesi gercekten ciziyor mu ve
// yakinlastirmayla olcekleniyor mu?
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 1100 });
  const hata = [];
  p.on('pageerror', (e) => hata.push(e.message));
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  const r = await p.goto(BASE + '/admin/manset-editor', { waitUntil: 'networkidle2' }).catch(() => null);
  await new Promise((x) => setTimeout(x, 1100));
  if (!r || r.status() !== 200) { console.log('atlandı', r ? r.status() : ''); await b.close(); return; }

  const d = await p.evaluate(async () => {
    const kaplama = document.getElementById('me-grid-overlay');
    const kutu = document.getElementById('me-grid');
    const zoom = document.getElementById('me-zoom');
    const olc = () => {
      const s = getComputedStyle(kaplama);
      const r = kaplama.getBoundingClientRect();
      const c = document.getElementById('me-canvas').getBoundingClientRect();
      return {
        sinif: kaplama.className,
        opaklik: s.opacity,
        bgGorsel: s.backgroundImage !== 'none',
        bgSize: s.backgroundSize,
        w: Math.round(r.width), h: Math.round(r.height),
        canvas: [Math.round(c.width), Math.round(c.height)],
        kaplamaCanvasla: Math.abs(r.width - c.width) < 3 && Math.abs(r.height - c.height) < 3,
      };
    };
    const kapali = olc();
    kutu.checked = true;
    kutu.dispatchEvent(new Event('change', { bubbles: true }));
    await new Promise((x) => setTimeout(x, 250));
    const acik = olc();

    // Zoom: gercek secenek degerlerini kullan.
    const secenek = Array.from(zoom.options).map((o) => o.value);
    zoom.value = '1';
    zoom.dispatchEvent(new Event('change', { bubbles: true }));
    await new Promise((x) => setTimeout(x, 250));
    const yakin = olc();

    zoom.value = '0.5';
    zoom.dispatchEvent(new Event('change', { bubbles: true }));
    await new Promise((x) => setTimeout(x, 250));
    const uzak = olc();

    return { kapali, acik, yakin, uzak, secenek };
  });

  console.log(JSON.stringify(d, null, 1));
  const ok = d.acik.opaklik === '1' && d.acik.bgGorsel && d.acik.kaplamaCanvasla && d.acik.h > 0;
  const olcek = d.yakin.bgSize !== d.uzak.bgSize;
  console.log(ok ? 'IZGARA: ciziliyor ve canvas ile hizali' : 'IZGARA: SORUN');
  console.log(olcek ? 'OLCEK: zoom ile degisiyor' : 'OLCEK: zoom degismedi');
  console.log('JS hatası:', hata.length ? [...new Set(hata)].slice(0, 2).join(' | ') : 'yok');
  await b.close();
})();
