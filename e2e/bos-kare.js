const puppeteer = require('puppeteer');
const BASE = process.argv[2], SID = process.argv[3];

// Geçiş sırasında .admin-main BOŞ kalıyor mu? (eski içerik silindi, yenisi
// henüz boyanmadı → zemin bir an boş görünür = "sert" his)
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 1000 });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  await p.goto(BASE + '/admin/icerikler', { waitUntil: 'networkidle2' });
  await new Promise((r) => setTimeout(r, 600));

  const d = await p.evaluate(async () => {
    const main = document.querySelector('.admin-main');
    const kareler = [];
    let calisti = true;
    const ornekle = () => {
      if (!calisti) return;
      const yeniMain = document.querySelector('.admin-main');
      kareler.push({
        t: Math.round(performance.now()),
        cocuk: yeniMain ? yeniMain.children.length : -1,
        dugum: yeniMain ? yeniMain.querySelectorAll('*').length : 0,
        yukseklik: yeniMain ? Math.round(yeniMain.getBoundingClientRect().height) : 0,
        h1: (document.querySelector('.admin-top h1') || {}).textContent || '',
      });
      requestAnimationFrame(ornekle);
    };
    requestAnimationFrame(ornekle);
    document.querySelector('.ws-links a[href="/admin/kategoriler"]').click();
    await new Promise((r) => setTimeout(r, 1200));
    calisti = false;
    return { kareler, ilkDugum: main.querySelectorAll('*').length };
  });

  const k = d.kareler;
  const bosKare = k.filter((x) => x.dugum < 10).length;
  const sifirYukseklik = k.filter((x) => x.yukseklik === 0).length;
  console.log('toplam kare :', k.length, '(ilk içerik düğümü', d.ilkDugum + ')');
  console.log('boş kare    :', bosKare, '(< 10 düğüm)');
  console.log('sıfır yükseklik:', sifirYukseklik);
  console.log('\nkare  düğüm  yükseklik  h1');
  k.slice(0, 34).forEach((x, i) => {
    console.log(String(i).padStart(4), String(x.dugum).padStart(6), String(x.yukseklik).padStart(10), ' ', x.h1.slice(0, 22));
  });
  await b.close();
})();
