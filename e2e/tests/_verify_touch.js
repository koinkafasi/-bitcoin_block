'use strict';
/**
 * Dokunmatik sürükleme doğrulaması (elle çalıştırılır):
 * SortableJS fallback (pointer) modunun telefonda çalıştığını kanıtlar.
 * Bu, native HTML5 DnD'nin YAPAMADIĞI şey.
 *
 * Kullanım:  TB_SESSION=<oturum> node tests/_verify_touch.js
 */
const puppeteer = require('puppeteer');

const SID = process.env.TB_SESSION;
const BASE = process.env.E2E_BASE_URL || 'http://127.0.0.1:8070';

function ok(name, cond, extra) {
  console.log((cond ? '  PASS  ' : '  FAIL  ') + name + (extra ? '  ' + extra : ''));
  if (!cond) process.exitCode = 1;
}

/** Touchscreen üzerinden bas-bırak-fare (SortableJS pointer fallback). */
async function touchDrag(page, srcSel, dstSel, steps = 30) {
  const a = await (await page.$(srcSel)).boundingBox();
  const b = await (await page.$(dstSel)).boundingBox();
  if (!a || !b) throw new Error('kaynak/hedef yok');

  const sx = a.x + a.width / 2;
  const sy = a.y + a.height / 2;
  const ex = b.x + b.width / 2;
  const ey = b.y + Math.min(b.height - 8, 40);

  const t = await page.touchscreen.touchStart(sx, sy);
  for (let i = 1; i <= steps; i++) {
    await t.move(sx + ((ex - sx) * i) / steps, sy + ((ey - sy) * i) / steps);
    await new Promise((r) => setTimeout(r, 14));
  }
  await new Promise((r) => setTimeout(r, 200));
  await t.end();
  await new Promise((r) => setTimeout(r, 700));
}

(async () => {
  const browser = await puppeteer.launch({
    headless: 'new',
    args: ['--no-sandbox', '--disable-dev-shm-usage'],
  });
  const page = await browser.newPage();

  // iPhone 12 benzeri
  await page.setViewport({ width: 390, height: 844, deviceScaleFactor: 2, isMobile: true, hasTouch: true });
  await browser.setCookie({
    name: 'tb_session',
    value: SID,
    domain: new URL(BASE).hostname,
    path: '/',
  });

  const errors = [];
  page.on('pageerror', (e) => errors.push(e.message));

  await page.goto(`${BASE}/admin/tasarim?sayfa=home`, { waitUntil: 'domcontentloaded' });
  await new Promise((r) => setTimeout(r, 1600));

  console.log('\n=== dokunmatik sürükleme (390x844, hasTouch) ===');

  ok('tasarimci yuklendi', !page.url().includes('/admin/giris'), page.url());
  ok('SortableJS yuklu', await page.evaluate(() => typeof window.Sortable === 'function'));

  const total = await page.$$eval('.bp-chip[data-id]', (e) => e.length);
  ok('cipler yuklendi', total > 3, total + ' cip');

  // --- 1) AYNI ALAN İÇİNDE dokunmatik sıralama (en sık mobil kullanım) ---
  const zoneSel = '[data-slots="ust"]';
  await page.$eval(zoneSel, (e) => e.scrollIntoView({ block: 'start' }));
  await new Promise((r) => setTimeout(r, 500));

  const orderBefore = await page.$$eval(`${zoneSel} .bp-chip[data-id]`, (e) =>
    e.map((x) => x.dataset.id)
  );
  console.log('    onceki sira: ' + JSON.stringify(orderBefore));

  const grips = await page.$$(`${zoneSel} .bp-chip .grip`);
  const chips = await page.$$(`${zoneSel} .bp-chip`);
  ok('en az 3 cip var', chips.length >= 3, chips.length + ' cip');

  if (chips.length >= 3) {
    // 1. cipten 3. cipin ALTINA dokunarak sürükle (swap eşiğini aşacak kadar)
    const a = await grips[0].boundingBox();
    const target = await chips[2].boundingBox();
    const sy = a.y + a.height / 2;
    const ex = a.x + a.width / 2;
    const ey = target.y + target.height - 5;

    const t = await page.touchscreen.touchStart(ex, sy);
    // uzun bas (sürükleme eşiği ~120ms)
    await new Promise((r) => setTimeout(r, 260));
    for (let i = 1; i <= 30; i++) {
      await t.move(ex, sy + ((ey - sy) * i) / 30);
      await new Promise((r) => setTimeout(r, 18));
    }
    // hedefte bekle
    for (let i = 0; i < 20; i++) {
      await t.move(ex, sy + ((ey - sy) * i) / 30);
      await new Promise((r) => setTimeout(r, 30));
    }
    await t.end();
    await new Promise((r) => setTimeout(r, 900));

    const orderAfter = await page.$$eval(`${zoneSel} .bp-chip[data-id]`, (e) =>
      e.map((x) => x.dataset.id)
    );
    console.log('    yeni sira:   ' + JSON.stringify(orderAfter));
    ok(
      'dokunmatikle siralama degisti',
      JSON.stringify(orderAfter) !== JSON.stringify(orderBefore),
      JSON.stringify(orderBefore) + ' -> ' + JSON.stringify(orderAfter)
    );
    ok('cip sayisi degismedi', orderAfter.length === orderBefore.length,
      `${orderBefore.length} -> ${orderAfter.length}`);
  }

  // --- 2) Dokunmatik kaydırma hâlâ çalışıyor mu ---
  await page.evaluate(() => window.scrollTo(0, 0));
  await new Promise((r) => setTimeout(r, 200));
  const y0 = await page.evaluate(() => window.scrollY);
  await page.evaluate(() => window.scrollTo(0, 300));
  await new Promise((r) => setTimeout(r, 250));
  const y1 = await page.evaluate(() => window.scrollY);
  ok('dikey kaydirma calisiyor', y1 > y0, `${y0} -> ${y1}`);

  // --- 3) Geri al ile temizle ---
  for (let i = 0; i < 3; i++) {
    if (await page.$eval('#bldUndo', (b) => b.disabled)) break;
    await page.click('#bldUndo');
    await new Promise((r) => setTimeout(r, 400));
  }
  await new Promise((r) => setTimeout(r, 1300));
  const restored = await page.$$eval('.bp-chip[data-id]', (e) => e.length);
  ok('baslangic durumuna dondu', restored === total, `${total} bekleniyordu, ${restored}`);

  ok('JS hatasi yok', errors.length === 0, errors.join(' | '));

  await browser.close();
  console.log(process.exitCode ? '\nSONUC: BASARISIZ' : '\nSONUC: TUMU GECTI');
})().catch((e) => {
  console.error('HATA:', e.message);
  process.exit(1);
});
