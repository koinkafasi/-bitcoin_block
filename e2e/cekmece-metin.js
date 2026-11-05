const puppeteer = require('puppeteer');
const BASE = process.argv[2];

// Üretimde çekmece bağlantı metinleri gerçekten boş mu?
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'], protocolTimeout: 60000 });
  const p = await b.newPage();
  await p.setViewport({ width: 1440, height: 1000 });
  await p.goto(BASE + '/', { waitUntil: 'networkidle2', timeout: 60000 });
  await p.evaluate(() => {
    const btn = document.querySelector('[data-menu-btn]');
    if (btn) btn.click();
  });
  await new Promise((r) => setTimeout(r, 500));

  const d = await p.evaluate(() => {
    const panel = document.querySelector('.hp-drawer-panel');
    if (!panel) return null;
    const baglar = [...panel.querySelectorAll('.hp-drawer-grid > a')].map((a) => ({
      metin: (a.textContent || '').replace(/\s+/g, ' ').trim(),
      uzunluk: (a.textContent || '').trim().length,
    }));
    return {
      baslik: ((panel.querySelector('.hp-drawer-title') || {}).textContent || '').trim(),
      gruplar: [...panel.querySelectorAll('.hp-drawer-group')].map((g) => g.textContent.trim()),
      baglar,
    };
  });
  console.log('başlık:', JSON.stringify(d.baslik));
  console.log('gruplar:', JSON.stringify(d.gruplar));
  console.log('--- bağlantılar ---');
  d.baglar.forEach((x) => console.log(`  [${String(x.uzunluk).padStart(2)}] ${JSON.stringify(x.metin)}`));
  await b.close();
})();
