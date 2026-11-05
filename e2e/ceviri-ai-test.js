const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];
const ICERIK = process.argv[4] || '554';

// Faz 190 — panelin TARAYICIDA tam akışı.
//
// Ölçülenler:
//   1. satırlar durum bilgisi taşıyor mu (var olan çeviri = Düzenle)
//   2. AI düğmesi doğru hedefe yazıyor mu (hx-target = eylem hücresi)
//   3. tıklama ANINDA dönüyor mu (kuyruk), bekleme paneli çıkıyor mu
//   4. yoklama bitince HX-Redirect ile TAM SAYFA yönlendirme oluyor mu
//   5. yeni sayfada dil bandı + kaynak kıvrımı var mı, iç içe sayfa yok mu
//
// DİKKAT — eski tuzak: content.editor sayfası "değişiklik var" sayarsa
// beforeunload diyaloğu çıkar ve JS ile yönlendirme (HX-Redirect) kilitlenir.
// Bu yüzden diyalog dinlenir ve raporlanır.
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'], protocolTimeout: 600000 });
  const p = await b.newPage();
  await p.setViewport({ width: 1440, height: 1100 });
  const jsHata = [];
  const diyaloglar = [];
  p.on('pageerror', (e) => jsHata.push(e.message));
  p.on('dialog', async (d) => { diyaloglar.push(d.type()); await d.accept().catch(() => {}); });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  await p.goto(BASE + '/admin/icerikler/' + ICERIK, { waitUntil: 'networkidle2', timeout: 60000 });
  await new Promise((r) => setTimeout(r, 700));

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
      aiSwap: ilk ? ilk.getAttribute('hx-swap') : null,
      aiPost: ilk ? ilk.getAttribute('hx-post') : null,
    };
  });
  console.log('=== PANEL ===');
  console.log(JSON.stringify(panel, null, 1));

  // --- AI ile çevir: tıkla, bekleme panelini bekle, yönlendirmeyi yakala ---
  const t0 = Date.now();
  const yanit = p.waitForResponse(
    (r) => r.request().method() === 'POST' && r.url().includes('/cevir/'), { timeout: 60000 }).catch(() => null);
  p.evaluate(() => {
    const det = [...document.querySelectorAll('details')].find((d) => {
      const s = d.querySelector('summary');
      return s && (s.textContent || '').includes('Dil / Çeviriler');
    });
    det.querySelector('button[hx-post*="/cevir/"]').click();
  }).catch(() => {});

  const r = await yanit;
  console.log('\n=== AI İSTEĞİ ===');
  console.log('yanıt:', r ? r.status() : 'yok', '·', Date.now() - t0, 'ms');
  if (r) {
    const h = r.headers();
    console.log('HX-Redirect:', h['hx-redirect'] || '(yok)');
  }

  // Bekleme paneli 2 sn içinde görünmeli.
  let bekleme = null;
  for (let i = 0; i < 20 && !bekleme; i++) {
    await new Promise((x) => setTimeout(x, 300));
    bekleme = await p.evaluate(() => {
      const d = document.querySelector('.ceviri-bekliyor');
      if (!d) return null;
      return {
        metin: (d.textContent || '').replace(/\s+/g, ' ').trim().slice(0, 60),
        yoklama: d.getAttribute('hx-get'),
        aralik: d.getAttribute('hx-trigger'),
      };
    }).catch(() => null);
  }
  console.log('bekleme paneli:', JSON.stringify(bekleme), '·', Date.now() - t0, 'ms');

  // Yönlendirme (yoklama sonucu). 6 dakikaya kadar bekle.
  let hedef = null;
  for (let i = 0; i < 180; i++) {
    const yol = new URL(p.url()).pathname;
    if (yol !== '/admin/icerikler/' + ICERIK) { hedef = yol; break; }
    await new Promise((x) => setTimeout(x, 2000));
  }
  console.log('\n=== YÖNLENDİRME ===');
  console.log('adres:', new URL(p.url()).pathname, '· toplam:', Math.round((Date.now() - t0) / 1000), 'sn');

  if (hedef) {
    const band = await p.waitForSelector('.ceviri-bant', { timeout: 20000 }).then(() => true).catch(() => false);
    const kaynak = await p.waitForSelector('.ceviri-kaynak', { timeout: 8000 }).then(() => true).catch(() => false);
    const icIce = await p.$$('button .ceviri-satir, button #f_title');
    const satir = await p.$$('.ceviri-satir');
    console.log('dil bandı:', band, '· kaynak kıvrımı:', kaynak, '· iç içe yerleşim:', icIce.length, '· satır:', satir.length);
  }

  console.log('\nJS hatası:', jsHata.length ? [...new Set(jsHata)].slice(0, 3).join(' | ') : 'yok');
  console.log('diyaloglar:', diyaloglar.length ? diyaloglar.join(', ') : 'yok');
  console.log('SONUC:', hedef && hedef !== '/admin/icerikler/' + ICERIK
    ? 'OK: kuyruk + yoklama + yönlendirme çalıştı' : 'HATA: yönlendirme olmadı');
  await b.close();
})();
