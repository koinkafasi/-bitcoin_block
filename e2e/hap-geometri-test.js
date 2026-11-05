const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

// Sekme hapı, gerçekten aktif sekmenin üstünde mi? Sekme çubuğu 7 öğeyle
// daralınca satır kayıyor (flex-wrap: wrap); hap kopyaları ayrı bir flex
// kabında olduğu için sarma noktası kayarsa hap yanlış sekmede durur.
const GENISLIKLER = [1440, 1280, 1100, 1024, 900, 820, 768, 640, 480, 390];

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  const hata = [];
  p.on('pageerror', (e) => hata.push(e.message));
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  for (const g of GENISLIKLER) {
    await p.setViewport({ width: g, height: 900 });
    await p.goto(BASE + '/admin/icerikler', { waitUntil: 'networkidle2' });
    await new Promise((r) => setTimeout(r, 500));

    const d = await p.evaluate(() => {
      const ray = document.querySelector('.ws-links');
      const pill = ray && ray.querySelector('.tb-hap');
      const aktif = ray && (ray.querySelector('a.on') || ray.querySelector('a.active'));
      if (!ray || !pill || !aktif) return { yok: true, ray: !!ray, pill: !!pill, aktif: !!aktif };

      const r = ray.getBoundingClientRect();
      const pr = pill.getBoundingClientRect();
      const ar = aktif.getBoundingClientRect();
      const cp = getComputedStyle(pill).clipPath;
      const m = cp.match(/inset\(([-\d.]+)px ([-\d.]+)px ([-\d.]+)px ([-\d.]+)px/);
      let gorunen = null;
      if (m) {
        gorunen = {
          sol: pr.left + parseFloat(m[4]),
          sag: pr.right - parseFloat(m[2]),
          ust: pr.top + parseFloat(m[1]),
          alt: pr.bottom - parseFloat(m[3]),
        };
      }
      // Sekme çubuğu kaç satıra sarıyor?
      const satirlar = new Set(
        Array.from(ray.querySelectorAll('a')).map((a) => Math.round(a.getBoundingClientRect().top))
      ).size;
      return {
        rayGenislik: Math.round(r.width),
        rayYukseklik: Math.round(r.height),
        satir: satirlar,
        aktif: { sol: Math.round(ar.left), sag: Math.round(ar.right), ust: Math.round(ar.top), alt: Math.round(ar.bottom) },
        hap: gorunen && { sol: Math.round(gorunen.sol), sag: Math.round(gorunen.sag), ust: Math.round(gorunen.ust), alt: Math.round(gorunen.alt) },
        fark: gorunen && {
          sol: Math.round(gorunen.sol - ar.left),
          sag: Math.round(gorunen.sag - ar.right),
          ust: Math.round(gorunen.ust - ar.top),
          alt: Math.round(gorunen.alt - ar.bottom),
        },
        clip: cp,
      };
    });
    const f = d.fark;
    const kotu = !f || Math.abs(f.sol) > 2 || Math.abs(f.sag) > 2 || Math.abs(f.ust) > 2 || Math.abs(f.alt) > 2;
    console.log(
      String(g).padStart(4),
      'ray', String(d.rayGenislik).padStart(4) + 'x' + d.rayYukseklik,
      'satir', d.satir || '-',
      'aktif', JSON.stringify(d.aktif),
      'hap', JSON.stringify(d.hap),
      kotu ? '  <<< KAYMA ' + JSON.stringify(f) : ''
    );
    if (g === 900 || g === 640 || g === 390) {
      await p.screenshot({ path: `/tmp/hap-${g}.png`, clip: { x: 0, y: 0, width: g, height: 420 } });
    }
  }
  console.log('JS hatası:', hata.length ? [...new Set(hata)].slice(0, 3).join(' | ') : 'yok');
  await b.close();
})();
