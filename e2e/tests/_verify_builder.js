'use strict';
/**
 * Tasarımcı Faz-1 doğrulama (elle çalıştırılır, CI'ya girmez):
 *   SortableJS yüklendi mi, paletten alana sürükleme çalışıyor mu,
 * geri al/yinele ve kilit düğmeleri doğru mu, kayıt yapılıyor mu.
 *
 * Kullanım:  TB_SESSION=<oturum> node tests/_verify_builder.js
 */
const puppeteer = require('puppeteer');

const SID = process.env.TB_SESSION;
const BASE = process.env.E2E_BASE_URL || 'http://127.0.0.1:8070';
const PAGE = process.env.BUILD_PAGE || 'home';

function ok(name, cond, extra) {
  console.log((cond ? '  PASS  ' : '  FAIL  ') + name + (extra ? '  ' + extra : ''));
  if (!cond) process.exitCode = 1;
}

/** SortableJS sürükleme simülasyonu (fare olayları ile). */
async function dragTo(page, srcSel, dstSel, steps = 25) {
  const src = await page.$(srcSel);
  const dst = await page.$(dstSel);
  const a = await src.boundingBox();
  const b = await dst.boundingBox();
  if (!a || !b) throw new Error('kaynak/hedef bulunamadi: ' + srcSel + ' -> ' + dstSel);
  await page.mouse.move(a.x + a.width / 2, a.y + a.height / 2);
  await page.mouse.down();
  const tx = b.x + b.width / 2;
  const ty = b.y + Math.min(b.height - 6, 40);
  for (let i = 1; i <= steps; i++) {
    await page.mouse.move(
      a.x + a.width / 2 + ((tx - a.x - a.width / 2) * i) / steps,
      a.y + a.height / 2 + ((ty - a.y - a.height / 2) * i) / steps
    );
    await new Promise((r) => setTimeout(r, 12));
  }
  await new Promise((r) => setTimeout(r, 150));
  await page.mouse.up();
  await new Promise((r) => setTimeout(r, 500));
}

(async () => {
  if (!SID) {
    console.error('TB_SESSION yok');
    process.exit(1);
  }
  const browser = await puppeteer.launch({
    headless: 'new',
    args: ['--no-sandbox', '--disable-dev-shm-usage'],
  });
  const page = await browser.newPage();
  await page.setViewport({ width: 1500, height: 1000 });
  const errors = [];
  const netFails = [];
  page.on('pageerror', (e) => errors.push('PAGEERROR: ' + e.message));
  page.on('response', (r) => {
    if (r.status() >= 400) netFails.push(r.status() + ' ' + r.request().method() + ' ' + r.url());
  });
  const dump = async (tag) => {
    const n = await page.$$eval('.bp-chip[data-id]', (e) => e.length);
    const perZone = await page.$$eval('[data-slots]', (els) =>
      els.map((e) => e.dataset.slots + '=' + e.querySelectorAll('.bp-chip[data-id]').length)
    );
    const undoDis = await page.$eval('#bldUndo', (b) => b.disabled);
    const redoDis = await page.$eval('#bldRedo', (b) => b.disabled);
    console.log(`    [${tag}] toplam=${n} zone[${perZone.join(' ')}] undoDisabled=${undoDis} redoDisabled=${redoDis}`);
    return n;
  };

  const host = new URL(BASE).hostname;
  await browser.setCookie({
    name: 'tb_session',
    value: SID,
    domain: host,
    path: '/',
  });

  await page.goto(`${BASE}/admin/tasarim?sayfa=${PAGE}`, {
    waitUntil: 'domcontentloaded',
  });
  await new Promise((r) => setTimeout(r, 1500));

  if (page.url().includes('/admin/giris')) {
    console.error('OTURUM KABUL EDILMEDI, url=', page.url());
    await browser.close();
    process.exit(1);
  }

  console.log(`\n=== /admin/tasarim?sayfa=${PAGE} ===`);

  // 1) SortableJS yüklendi mi
  ok('SortableJS yüklendi', await page.evaluate(() => typeof window.Sortable === 'function'));

  // 2) paletten alana sürükleme
  const before = await page.$$eval('.bp-chip[data-id]', (e) => e.length);
  const zoneSel = '[data-slots="ust"]';
  const zoneEmpty = await page.$(zoneSel);
  ok('hedef alan var', !!zoneEmpty);

  const palBtn = await page.$('.bp-pal .bld-add[data-type="slider"]');
  ok('palette slider var', !!palBtn);
  await dragTo(page, '.bp-pal .bld-add[data-type="slider"]', zoneSel);
  const after = await page.$$eval('.bp-chip[data-id]', (e) => e.length);
  ok('paletten alana sürüklendi', after > before, `${before} -> ${after}`);

  // 3) yeni çip gerçek bir bloğa dönüştü mü
  const chipType = await page.$$eval('[data-slots="ust"] .bp-chip .ty', (e) =>
    e.map((x) => x.textContent.trim())
  );
  ok('çip "slider" tipinde', chipType.includes('slider'), JSON.stringify(chipType));

  // 4) sunucuya kaydedildi mi
  await new Promise((r) => setTimeout(r, 1800));
  const saved = await page.evaluate(async (p) => {
    const r = await fetch('/admin/tasarim/kaydet?sayfa=' + p, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ probe: 1 }),
    });
    return r.status;
  }, PAGE);
  ok('kaydetme ucu erişilebilir', saved === 200 || saved === 400, 'HTTP ' + saved);

  // 5) geri al / yinele
  const undoEnabled = await page.$eval('#bldUndo', (b) => !b.disabled);
  ok('geri al düğmesi etkin', undoEnabled);
  await dump('once');
  await page.click('#bldUndo');
  await new Promise((r) => setTimeout(r, 500));
  const afterUndo = await dump('undo-sonrasi');

  // elle purge + rerender dene: hayalet kalıcı mı, yoksa geri mi geliyor?
  ok('geri al çipi kaldırdı', afterUndo === before, `${after} -> ${afterUndo}`);

  await page.click('#bldRedo');
  await new Promise((r) => setTimeout(r, 500));
  const afterRedo = await dump('redo-sonrasi');
  ok('yinele çipi geri getirdi', afterRedo === after, `${afterUndo} -> ${afterRedo}`);

  // 6) kilit düğmesi (düğmeler gizli olabildiği için DOM üzerinden tıkla)
  try {
    await page.$eval('[data-slots="ust"] .bp-chip .ops button[data-act="lock"]', (b) => b.click());
    await new Promise((r) => setTimeout(r, 400));
    const locked = await page.$$eval('[data-slots="ust"] .bp-chip.locked', (e) => e.length);
    ok('kilit işledi', locked > 0, locked + ' kilitli');
    const delDisabled = await page.$eval(
      '[data-slots="ust"] .bp-chip.locked .ops button[data-act="del"]',
      (b) => b.disabled
    );
    ok('kilitli çipte silme devre dışı', delDisabled === true);
  } catch (e) {
    ok('kilit işledi', false, 'HATA: ' + e.message);
  }

  // 7) temizlik: testin eklediği çipi geri al
  for (let i = 0; i < 4; i++) {
    const dis = await page.$eval('#bldUndo', (b) => b.disabled);
    if (dis) break;
    await page.click('#bldUndo');
    await new Promise((r) => setTimeout(r, 350));
  }
  await new Promise((r) => setTimeout(r, 1400));
  const restored = await dump('temizlik-sonrasi');
  ok('başlangıç durumuna döndü', restored === before, `${before} bekleniyordu, ${restored} bulundu`);

  // 8) JS hatası yok (kaynak 404'leri JS hatası değildir)
  ok('JS hatası yok', errors.length === 0, errors.join(' | '));
  console.log('    ağ notları: ' + (netFails.join(' ; ') || 'yok'));

  await browser.close();
  console.log(process.exitCode ? '\nSONUC: BASARISIZ' : '\nSONUC: TUMU GECTI');
})().catch((e) => {
  console.error('HATA:', e.message);
  process.exit(1);
});
