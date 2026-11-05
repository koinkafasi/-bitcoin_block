const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];
const ICERIK = process.argv[4] || '554';

// Faz 190: panelin TARAYICIDA olculmesi.
//
// Ölçülenler:
//   1. satırlar durum bilgisi taşıyor mu (var olan çeviri = Düzenle)
//   2. AI düğmesi doğru hedefe yazıyor mu (hx-target = eylem hücresi)
//   3. "Boş taslak" HTMX yolunda TAM SAYFA yüklemesi yapıyor mu
//      (eski kod düz 302 dönüyordu; htmx 302'yi takip edip dönen sayfayı
//      DÜĞMENİN İÇİNE yazıyordu → iç içe sayfa)
//
// Not: tıklamadan sonra evaluate ÇAĞRILMAZ. HX-Redirect tam sayfa yüklemesi
// başlatır; yürütme bağlamı yok olur ve bekleyen evaluate asılı kalır (ilk
// sürümde protocolTimeout'a düştü). Tıklama ElementHandle ile yapılır, sonuç
// yalnız navigasyondan sonra okunur.
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'], protocolTimeout: 120000 });
  const p = await b.newPage();
  await p.setViewport({ width: 1440, height: 1200 });
  const hata = [];
  p.on('pageerror', (e) => hata.push(e.message));
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  await p.goto(BASE + '/admin/icerikler/' + ICERIK, { waitUntil: 'networkidle2', timeout: 60000 });
  await new Promise((r) => setTimeout(r, 700));

  // Panel kapalı <details> içinde: açmadan satırlar görünmez.
  await p.evaluate(() => {
    const det = [...document.querySelectorAll('details')].find((d) => {
      const s = d.querySelector('summary');
      return s && (s.textContent || '').includes('Dil / Çeviriler');
    });
    if (det && !det.open) det.open = true;
  });
  await new Promise((r) => setTimeout(r, 250));

  const panel = await p.evaluate(() => {
    const det = [...document.querySelectorAll('details')].find((d) => {
      const s = d.querySelector('summary');
      return s && (s.textContent || '').includes('Dil / Çeviriler');
    });
    if (!det) return { var: false };
    const satirlar = [...det.querySelectorAll('.ceviri-satir')].map((r) => {
      const a = r.querySelector('a[href*="/admin/icerikler/"]');
      const btn = r.querySelector('button[hx-post*="/cevir/"]');
      return {
        dil: ((r.querySelector('.ceviri-satir__dil b') || {}).textContent || '').trim(),
        kod: ((r.querySelector('.ceviri-kod') || {}).textContent || '').trim(),
        durum: ((r.querySelector('.ceviri-rozet') || {}).textContent || '').trim(),
        ai: !!btn,
        duzenle: a ? a.getAttribute('href') : '',
        aiHedef: btn ? btn.getAttribute('hx-target') : null,
      };
    });
    const ilk = det.querySelector('button[hx-post*="/cevir/"]');
    return {
      var: true,
      satir: satirlar.length,
      aiDugme: satirlar.filter((s) => s.ai).length,
      duzenle: satirlar.filter((s) => s.duzenle).length,
      artiKalan: [...det.querySelectorAll('a')].filter((a) => (a.textContent || '').trim().endsWith('+')).length,
      ceviriler: satirlar.filter((s) => s.duzenle).map((s) => s.kod + ' → ' + s.duzenle),
      ilk3: satirlar.slice(0, 3),
      aiSwap: ilk ? ilk.getAttribute('hx-swap') : null,
      aiPost: ilk ? ilk.getAttribute('hx-post') : null,
    };
  });
  console.log('=== PANEL ===');
  console.log(JSON.stringify(panel, null, 1));

  // --- "Boş taslak" düğmesi (ilk çevirisi olmayan dil) ---
  const dugmeler = await p.$$('button[hx-vals*="kopya"]');
  console.log('\n"Boş taslak" düğmesi:', dugmeler.length);
  if (dugmeler.length) {
    const t0 = Date.now();
    const hedefUrl = new URL(p.url()).pathname;
    const gidis = p.waitForNavigation({ waitUntil: 'domcontentloaded', timeout: 60000 }).catch(() => null);
    await dugmeler[0].click();
    const gitti = await gidis;
    console.log('kaynak sayfa:', hedefUrl);
    console.log('navigasyon:', gitti ? 'oldu' : 'OLMADI', '·', Date.now() - t0, 'ms');
    console.log('yeni adres:', new URL(p.url()).pathname);

    // Yönlendirme sonrası sayfa: selector beklemesi (evaluate değil) —
    // ağır editör sayfasında callFunctionOn asılı kalabiliyor.
    const band = await p.waitForSelector('.ceviri-bant', { timeout: 30000 }).then(() => true).catch(() => false);
    console.log('dil bandı:', band);
    console.log('kaynak kıvrımı:', await p.waitForSelector('.ceviri-kaynak', { timeout: 8000 }).then(() => true).catch(() => false));
    console.log('satır sayısı:', (await p.$$('.ceviri-satir')).length);
    console.log('iç içe sayfa kalıntısı:', (await p.$$('button .ceviri-satir, button #f_title')).length);
    console.log('süre:', Date.now() - t0, 'ms');
    console.log('SONUC:', new URL(p.url()).pathname !== hedefUrl && band
      ? 'OK: tam sayfa yönlendirme + dil bandı' : 'HATA');
  }

  console.log('\nJS hatası:', hata.length ? [...new Set(hata)].slice(0, 3).join(' | ') : 'yok');
  await b.close();
})();
