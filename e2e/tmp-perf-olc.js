'use strict';
/**
 * Sayfa başına gerçek transfer ölçümü (Faz: performans bütçesi).
 *
 * Ölçülen: istek sayısı, toplam aktarılan bayt (sıkıştırılmış, ağdan geçen),
 * kaynak tipine göre dağılım, LCP ve CLS (PerformanceObserver).
 *
 * Neden tarayıcı: transfer boyutu sıkıştırmaya bağlı (gzip/br) ve LCP/CLS
 * yalnız gerçek bir render'da ölçülebilir.
 */
const puppeteer = require('puppeteer');

const BASE = process.env.E2E_BASE_URL || 'http://127.0.0.1:8099';
const SID = process.env.TB_SESSION || '';

const SAYFALAR = [
  ['ana', '/'],
  ['kategori', '/kategori/ekonomi'],
  ['makale', null],
  ['panel', '/admin'],
  ['panel-icerik-formu', '/admin/icerikler/yeni'],
];

function kb(n) { return (n / 1024).toFixed(1); }

(async () => {
  const browser = await puppeteer.launch({
    headless: 'new',
    args: ['--no-sandbox', '--disable-dev-shm-usage', '--disable-gpu'],
  });
  try {
    for (const [ad, yol] of SAYFALAR) {
      for (const [w, h, etiket] of [[1440, 1000, 'masaustu'], [390, 844, 'mobil']]) {
        const page = await browser.newPage();
        await page.setViewport({ width: w, height: h });
        if (SID) await browser.setCookie({ name: 'tb_session', value: SID, domain: new URL(BASE).hostname, path: '/' });
        // Yalnız bizim kaynaklarımız ölçülsün.
        await page.setRequestInterception(true);
        page.on('request', (r) => {
          const u = r.url();
          if (u.startsWith(BASE) || u.startsWith('data:') || u.startsWith('about:')) r.continue(); else r.abort();
        });

        const kaynaklar = [];
        page.on('response', async (res) => {
          try {
            const req = res.request();
            const len = Number(res.headers()['content-length'] || 0);
            kaynaklar.push({
              url: req.url().replace(BASE, ''),
              type: req.resourceType(),
              status: res.status(),
              bytes: len,
              encoding: res.headers()['content-encoding'] || '',
            });
          } catch (e) {}
        });

        // LCP + CLS
        await page.evaluateOnNewDocument(() => {
          window.__lcp = 0; window.__cls = 0;
          try {
            new PerformanceObserver((l) => {
              for (const e of l.getEntries()) window.__lcp = Math.max(window.__lcp, e.startTime);
            }).observe({ type: 'largest-contentful-paint', buffered: true });
            new PerformanceObserver((l) => {
              for (const e of l.getEntries()) if (!e.hadRecentInput) window.__cls += e.value;
            }).observe({ type: 'layout-shift', buffered: true });
          } catch (e) {}
        });

        let hedef = yol;
        try {
          await page.goto(BASE + (yol || '/'), { waitUntil: 'networkidle2', timeout: 60000 });
          if (yol === null) {
            hedef = await page.evaluate(() => {
              const a = [...document.querySelectorAll('a[href]')].find((x) =>
                /^\/[^/]+-\d+$/.test(x.getAttribute('href') || ''));
              return a ? a.getAttribute('href') : '/';
            });
            kaynaklar.length = 0;
            await page.goto(BASE + hedef, { waitUntil: 'networkidle2', timeout: 60000 });
          }
          await new Promise((r) => setTimeout(r, 1200));
          const metrik = await page.evaluate(() => ({ lcp: window.__lcp, cls: window.__cls }));

          const toplam = kaynaklar.reduce((s, k) => s + k.bytes, 0);
          const grup = {};
          for (const k of kaynaklar) {
            const g = k.type === 'stylesheet' ? 'css' : k.type === 'script' ? 'js'
              : k.type === 'image' ? 'img' : k.type === 'font' ? 'font' : 'diger';
            grup[g] = (grup[g] || 0) + k.bytes;
          }
          const buyuk = [...kaynaklar].sort((a, b) => b.bytes - a.bytes).slice(0, 5);

          console.log(`\n=== ${ad} (${etiket}) ${hedef} ===`);
          console.log(`  istek ${kaynaklar.length} · toplam ${kb(toplam)} KB · LCP ${Math.round(metrik.lcp)} ms · CLS ${metrik.cls.toFixed(4)}`);
          console.log('  ' + Object.entries(grup).map(([k, v]) => `${k}=${kb(v)}KB`).join(' · '));
          for (const b of buyuk) {
            console.log(`    ${String(Math.round(b.bytes / 1024)).padStart(5)} KB  ${b.encoding || 'yok'}  ${b.url}`);
          }
        } catch (e) {
          console.log('HATA', ad, etiket, String(e.message).slice(0, 100));
        } finally {
          await page.close();
        }
      }
    }
  } finally {
    await browser.close();
  }
})();
