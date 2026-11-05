const puppeteer = require('puppeteer');

// Faz 168 + 170: tüm panelde emoji taraması, konsol hataları ve yatay taşma.
const BASE = process.argv[2];
const SID = process.argv[3];

const EMOJI = /[\u{1F000}-\u{1FAFF}\u{2600}-\u{27BF}\u{2B00}-\u{2BFF}\u{2300}-\u{23FF}\u{25A0}-\u{25FF}]/u;

const SAYFALAR = [
  ['/admin/pano', 'pano'],
  ['/admin/icerikler', 'icerikler'],
  ['/admin/kategoriler', 'kategoriler'],
  ['/admin/etiketler', 'etiketler'],
  ['/admin/ceviriler', 'ceviriler'],
  ['/admin/sinemalar', 'sinemalar'],
  ['/admin/resmi-ilanlar', 'resmi-ilan'],
  ['/admin/dosyalar', 'medya'],
  ['/admin/yorumlar', 'yorumlar'],
  ['/admin/aboneler', 'aboneler'],
  ['/admin/bulten', 'bulten'],
  ['/admin/ayarlar', 'ayarlar'],
  ['/admin/gorunum', 'gorunum'],
  ['/admin/tasarim', 'tasarim'],
  ['/admin/seo', 'seo'],
  ['/admin/servisler', 'servisler'],
  ['/admin/iletisim', 'iletisim'],
  ['/admin/geri-bildirim', 'geribildirim'],
  ['/admin/gazete', 'gazete'],
  ['/admin/gazeteler', 'gazeteler'],
  ['/admin/anketler', 'anketler'],
  ['/admin/etkinlikler', 'etkinlikler'],
  ['/admin/bilesenler', 'bilesenler'],
  ['/admin/menu', 'menu'],
  ['/admin/yardim', 'yardim'],
  ['/admin/gelen-kutusu', 'gelen-kutu'],
  ['/admin/raporlar', 'raporlar'],
  ['/admin/paylasim', 'paylasim'],
  ['/admin/backlink', 'backlink'],
];

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 1000 });
  const hata = [];
  p.on('pageerror', (e) => hata.push(e.message));
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  let toplamEmoji = 0, toplamTasma = 0;
  for (const [yol, ad] of SAYFALAR) {
    const r = await p.goto(BASE + yol, { waitUntil: 'networkidle2' }).catch(() => null);
    await new Promise((x) => setTimeout(x, 380));
    if (!r || r.status() >= 400) { console.log(ad.padEnd(12), 'atlandı', r ? r.status() : 'yok'); continue; }
    const d = await p.evaluate((src) => {
      const re = new RegExp(src, 'u');
      const govde = document.querySelector('.admin-body') || document.body;
      const bulunan = [];
      const y = document.createTreeWalker(govde, NodeFilter.SHOW_TEXT);
      while (y.nextNode()) {
        const t = y.currentNode.nodeValue || '';
        const par = y.currentNode.parentElement;
        if (re.test(t) && par && par.getClientRects().length) bulunan.push(t.trim().slice(0, 28));
      }
      // Yatay taşma: belge genişliği görünür alanı aşıyor mu.
      const tasma = document.documentElement.scrollWidth - document.documentElement.clientWidth;
      return { emoji: bulunan, tasma };
    }, EMOJI.source);
    toplamEmoji += d.emoji.length;
    if (d.tasma > 2) toplamTasma++;
    console.log(ad.padEnd(12), r.status(), 'emoji:', d.emoji.length,
      d.emoji.length ? JSON.stringify(d.emoji.slice(0, 3)) : '',
      '· taşma:', d.tasma);
  }
  console.log('\nTOPLAM emoji:', toplamEmoji, '· taşan sayfa:', toplamTasma);
  console.log('JS hatası:', hata.length ? [...new Set(hata)].slice(0, 3).join(' | ') : 'yok');
  await b.close();
})();
