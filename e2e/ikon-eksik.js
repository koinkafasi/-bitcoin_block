const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

// KESIN KONTROL: her <use href="...#ti-X"> hedefi sprite'ta tanimli mi?
// Eksik sembol = bos kutu (CSS boyut verdigi icin 18x18 gorunur ama ic bos).
const SAYFALAR = [
  '/admin', '/admin/icerikler', '/admin/icerikler/yeni', '/admin/kategoriler',
  '/admin/etiketler', '/admin/medya', '/admin/yorumlar', '/admin/members',
  '/admin/gazete', '/admin/reklamlar', '/admin/ayarlar', '/admin/marka',
  '/admin/gorunum', '/admin/yazarlar', '/admin/raporlar', '/admin/manset-editor',
  '/admin/yardim', '/admin/trend', '/admin/aboneler', '/admin/seo',
  '/admin/eski', '/admin/gelen-kutusu', '/admin/geri-bildirim',
  '/', '/gazete',
];

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1440, height: 1000 });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  // Sprite'taki sembol kimliklerini bir kez al (node tarafinda).
  const spriteMetin = await (await fetch(BASE + '/static/vendor/tabler-icons.svg')).text();
  const spr = new Set();
  {
    const re = /<symbol[^>]*\bid="([^"]+)"/g;
    let m;
    while ((m = re.exec(spriteMetin))) spr.add(m[1]);
  }
  console.log('sprite sembol sayisi:', spr.size);

  const eksikHarita = new Map();

  for (const yol of SAYFALAR) {
    const r = await p.goto(BASE + yol, { waitUntil: 'networkidle2' }).catch(() => null);
    await new Promise((x) => setTimeout(x, 350));
    if (!r || r.status() !== 200) continue;

    const hedefler = await p.evaluate(() => {
      const out = [];
      document.querySelectorAll('use').forEach((u) => {
        const h = u.getAttribute('href') || u.getAttribute('xlink:href') || '';
        if (!h.includes('#')) return;
        out.push({ kim: h.split('#')[1], dosya: h.split('#')[0] });
      });
      return out;
    });

    for (const h of hedefler) {
      if (!spr.has(h.kim)) {
        const anahtar = h.kim + ' | ' + h.dosya;
        if (!eksikHarita.has(anahtar)) eksikHarita.set(anahtar, []);
        eksikHarita.get(anahtar).push(yol);
      }
    }
  }

  console.log('\n=== SPRITE\'TA OLMAYAN SEMBOLLER ===');
  if (!eksikHarita.size) {
    console.log('  yok — tum use hedefleri tanimli');
  } else {
    for (const [k, v] of eksikHarita) {
      console.log('  ' + k + '   (' + [...new Set(v)].slice(0, 4).join(', ') + ')');
    }
  }
  await b.close();
})();
