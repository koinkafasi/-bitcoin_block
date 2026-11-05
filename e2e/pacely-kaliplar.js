const puppeteer = require('puppeteer');

// Pacely: her rotada hangi ARAYÜZ KALIPLARI var?
const ROTALAR = ['/home', '/schedule', '/schedule/day', '/patients', '/patients/all/P-1040',
  '/patients/waitlist', '/plans', '/plans/library', '/outcomes', '/team', '/billing',
  '/billing/claims', '/messages', '/settings', '/help'];
const BASE = 'https://demo.codeandchill.store/pacely/#';

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1600, height: 1000 });
  const hatalar = [];
  p.on('pageerror', (e) => hatalar.push(e.message));

  const sonuc = {};
  for (const r of ROTALAR) {
    await p.goto(BASE + r, { waitUntil: 'networkidle2', timeout: 90000 });
    await new Promise((x) => setTimeout(x, 900));
    sonuc[r] = await p.evaluate(() => {
      const q = (s) => [...document.querySelectorAll(s)];
      const metin = (e) => (e.textContent || '').trim().replace(/\s+/g, ' ');
      const gorunur = (e) => e.getClientRects().length > 0;
      return {
        h1: metin(document.querySelector('h1') || document.createElement('h1')),
        // Kalıp izleri
        sekmeler: q('[role=tab], .seg button, .tabs button').filter(gorunur).map(metin).slice(0, 14),
        bolumBaslik: q('h2, .card h3, .sh').filter(gorunur).map(metin).slice(0, 10),
        tabloBasliklari: q('table thead th').map(metin).slice(0, 14),
        tabloSatir: q('table tbody tr').length,
        dugmeler: q('button, a.btn').filter(gorunur).map(metin).filter(Boolean).slice(0, 22),
        // Yapısal işaretler
        iskelet: q('[class*=skeleton], [class*=shimmer], [aria-busy=true]').length,
        bosDurum: q('[class*=empty], [class*=placeholder]').length,
        rozetler: q('.pill, .badge, .chip').filter(gorunur).length,
        kpi: q('.kpi, [class*=kpi]').filter(gorunur).length,
        grafik: q('canvas, svg[class*=chart], [class*=chart]').length,
        avatar: q('[class*=avatar]').length,
        zamanCizelgesi: q('[class*=timeline], [class*=activity], [class*=feed]').length,
        filtreCipleri: q('.chip[aria-pressed], [class*=filter]').length,
        sayfalama: q('[class*=pager], [class*=pagination], nav[aria-label*=Sayfa]').length,
        cekmece: q('[role=dialog], .drawer, .sheet, [class*=modal]').length,
        ariaLive: q('[aria-live]').length,
        stickyBaslik: q('[class*=sticky], [class*=head--]').length,
        satirSecim: q('table tbody input[type=checkbox]').length,
        ustAraclar: q('.tablewrap *, [class*=tablebar] *').filter((e) => e.tagName === 'BUTTON').map(metin).slice(0, 10),
        metinUzunluk: (document.querySelector('main') || document.body).innerText.length,
      };
    });
  }

  for (const [r, v] of Object.entries(sonuc)) {
    console.log('\n########## ' + r + ' ##########');
    console.log('  h1:', v.h1);
    if (v.sekmeler.length) console.log('  sekmeler:', JSON.stringify(v.sekmeler));
    if (v.bolumBaslik.length) console.log('  bölüm:', JSON.stringify(v.bolumBaslik));
    if (v.tabloBasliklari.length) console.log('  tablo başlık:', JSON.stringify(v.tabloBasliklari), '· satır:', v.tabloSatir);
    if (v.dugmeler.length) console.log('  düğmeler:', JSON.stringify(v.dugmeler));
    if (v.ustAraclar.length) console.log('  tablo araçları:', JSON.stringify(v.ustAraclar));
    console.log('  sayaçlar: iskelet=' + v.iskelet, 'boş=' + v.bosDurum, 'rozet=' + v.rozetler,
      'kpi=' + v.kpi, 'grafik=' + v.grafik, 'avatar=' + v.avatar, 'akış=' + v.zamanCizelgesi,
      'filtre=' + v.filtreCipleri, 'sayfalama=' + v.sayfalama, 'diyalog=' + v.cekmece,
      'ariaLive=' + v.ariaLive, 'satırSeçim=' + v.satirSecim, 'sticky=' + v.stickyBaslik);
  }
  console.log('\nJS hatası:', hatalar.length ? [...new Set(hatalar)].slice(0, 4) : 'yok');
  await b.close();
})();
