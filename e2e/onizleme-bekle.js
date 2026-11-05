const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'] });
  const p = await b.newPage();
  await p.setViewport({ width: 1440, height: 1000 });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });
  await p.goto(BASE + '/admin/tasarim', { waitUntil: 'networkidle2' });

  // Onizlemelerin YUKLENMESINI bekle (onceki olcum yukleme anini yakalamis olabilir)
  for (let i = 0; i < 30; i++) {
    const hazir = await p.evaluate(() => {
      const lar = Array.from(document.querySelectorAll('.bp-prev'));
      if (!lar.length) return false;
      return lar.every((el) => !el.classList.contains('bp-prev-loading'));
    });
    if (hazir) break;
    await new Promise((x) => setTimeout(x, 400));
  }
  await new Promise((x) => setTimeout(x, 1200));

  const d = await p.evaluate(() => {
    const out = [];
    document.querySelectorAll('.manset-cat, .hero-cat, .card-badge').forEach((el) => {
      if (el.getClientRects().length === 0) return;
      const s = getComputedStyle(el);
      out.push({ sinif: el.className, renk: s.color, zemin: s.backgroundColor, metin: el.textContent.trim().slice(0, 20) });
    });
    // Yorum formu ipucu
    const yorum = Array.from(document.querySelectorAll('.bp-prev *')).find((el) => (el.textContent || '').trim() === 'Yorumunuzu ekleyin');
    const ys = yorum ? getComputedStyle(yorum) : null;
    return {
      rozetler: out.slice(0, 5),
      rozetToplam: out.length,
      yuklemedeOlan: document.querySelectorAll('.bp-prev-loading').length,
      yorum: yorum ? { renk: ys.color, zemin: ys.backgroundColor, boyut: ys.fontSize, sinif: yorum.className } : null,
    };
  });
  console.log(JSON.stringify(d, null, 1));
  await b.close();
})();
