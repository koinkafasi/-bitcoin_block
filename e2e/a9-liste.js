const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

// A-9 ground truth: hangi ekranda hangi kontrol adsiz?
const SAYFALAR = [
  '/admin', '/admin/icerikler', '/admin/icerikler/yeni', '/admin/kategoriler',
  '/admin/etiketler', '/admin/yorumlar', '/admin/members', '/admin/gazete',
  '/admin/reklamlar', '/admin/ayarlar', '/admin/marka', '/admin/gorunum',
  '/admin/yazarlar', '/admin/raporlar', '/admin/manset-editor', '/admin/yardim',
  '/admin/seo', '/admin/backlink', '/admin/trend', '/admin/aboneler',
  '/admin/bot', '/admin/servisler', '/admin/kumeler', '/admin/mahrecler',
  '/admin/geri-bildirim', '/admin/gelen-kutusu', '/admin/ai', '/admin/araclar',
  '/admin/sinemalar', '/admin/resmi-ilanlar', '/admin/gazeteler', '/admin/etkinlikler',
  '/admin/topluluk', '/admin/seo-servis', '/admin/tasarim', '/admin/iletisim',
];

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1440, height: 1000 });
  const hata = [];
  p.on('pageerror', (e) => hata.push(e.message));
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  const tumu = [];
  let toplamAlan = 0;

  for (const yol of SAYFALAR) {
    const r = await p.goto(BASE + yol, { waitUntil: 'networkidle2' }).catch(() => null);
    await new Promise((x) => setTimeout(x, 320));
    if (!r || r.status() !== 200) continue;

    const d = await p.evaluate(() => {
      const ad = (el) => {
        const a = el.getAttribute('aria-label');
        if (a && a.trim()) return a.trim();
        const lb = el.getAttribute('aria-labelledby');
        if (lb) {
          const t = lb.split(/\s+/).map((i) => document.getElementById(i)?.textContent || '').join(' ').trim();
          if (t) return t;
        }
        if (el.id) {
          const l = document.querySelector('label[for="' + CSS.escape(el.id) + '"]');
          if (l && l.textContent.trim()) return l.textContent.trim();
        }
        const anc = el.closest('label');
        if (anc && anc.textContent.trim()) return anc.textContent.trim();
        return '';
      };
      const out = [];
      let sayi = 0;
      document.querySelectorAll('input:not([type=hidden]), select, textarea').forEach((el) => {
        const s = getComputedStyle(el);
        if (s.display === 'none' || s.visibility === 'hidden' || !el.getClientRects().length) return;
        sayi++;
        if (ad(el)) return;
        out.push({
          tur: el.tagName.toLowerCase() + (el.type ? '[' + el.type + ']' : ''),
          name: el.getAttribute('name') || '',
          id: el.id || '',
          ph: el.getAttribute('placeholder') || '',
          // En yakin baslik/metin: kart basligi ya da onundeki kisa metin
          yakin: (() => {
            let c = el.parentElement;
            for (let i = 0; i < 4 && c; i++) {
              const t = (c.textContent || '').replace(/\s+/g, ' ').trim();
              if (t) return t.slice(0, 60);
              c = c.parentElement;
            }
            return '';
          })(),
        });
      });
      return { alan: sayi, adsiz: out };
    });

    toplamAlan += d.alan;
    for (const x of d.adsiz) tumu.push({ yol, ...x });
  }

  console.log('TOPLAM ALAN:', toplamAlan, '· ADSIZ:', tumu.length);
  console.log('');
  for (const x of tumu) {
    console.log(
      x.yol.padEnd(24),
      x.tur.padEnd(18),
      ('name=' + (x.name || '-')).padEnd(24),
      ('ph="' + x.ph + '"').padEnd(30),
      '| ' + x.yakin
    );
  }
  console.log('\nJS hatası:', hata.length ? [...new Set(hata)].slice(0, 2).join(' | ') : 'yok');
  await b.close();
})();
