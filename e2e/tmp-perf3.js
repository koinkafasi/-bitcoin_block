'use strict';
/**
 * Gerçek ağ baytı ölçümü — CDP encodedDataLength, ÖNBELLEK KAPALI.
 *
 * Önceki koşuda ikinci (mobil) ölçüm 0 bayt çıkıyordu: aynı tarayıcı
 * profilinde HTTP önbelleği devreye giriyordu. `setCacheEnabled(false)`
 * her sayfayı soğuk ziyaretçi gibi ölçer.
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

const kb = (n) => (n / 1024).toFixed(1);

(async () => {
  for (const [ad, yol] of SAYFALAR) {
    for (const [w, h, etiket] of [[1440, 1000, 'masaustu'], [390, 844, 'mobil']]) {
      // Her ölçüm TAZE tarayıcı: önbellek ve oturum durumu sızmasın.
      const browser = await puppeteer.launch({
        headless: 'new',
        args: ['--no-sandbox', '--disable-dev-shm-usage', '--disable-gpu', '--incognito'],
      });
      try {
        const page = await browser.newPage();
        await page.setCacheEnabled(false);
        await page.setViewport({ width: w, height: h });
        const cdp = await page.target().createCDPSession();
        await cdp.send('Network.enable');
        await cdp.send('Network.setCacheDisabled', { cacheDisabled: true });

        const istekler = new Map();
        const bitenler = [];
        cdp.on('Network.requestWillBeSent', (e) => {
          istekler.set(e.requestId, { url: e.request.url.replace(BASE, ''), type: e.type });
        });
        cdp.on('Network.loadingFinished', (e) => {
          const r = istekler.get(e.requestId);
          if (!r) return;
          bitenler.push({ ...r, bytes: e.encodedDataLength || 0 });
        });

        if (SID) await browser.setCookie({ name: 'tb_session', value: SID, domain: new URL(BASE).hostname, path: '/' });
        await page.setRequestInterception(true);
        page.on('request', (r) => {
          const u = r.url();
          if (u.startsWith(BASE) || u.startsWith('data:') || u.startsWith('about:')) r.continue(); else r.abort();
        });

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
            bitenler.length = 0;
            await page.goto(BASE + hedef, { waitUntil: 'networkidle2', timeout: 60000 });
          }
          await new Promise((r) => setTimeout(r, 1200));
          const m = await page.evaluate(() => ({ lcp: window.__lcp, cls: window.__cls }));

          const toplam = bitenler.reduce((s, k) => s + k.bytes, 0);
          const grup = {};
          for (const k of bitenler) {
            const g = k.type === 'Stylesheet' ? 'css' : k.type === 'Script' ? 'js'
              : k.type === 'Image' ? 'img' : k.type === 'Font' ? 'font'
              : k.type === 'Document' ? 'html' : 'diger';
            grup[g] = (grup[g] || 0) + k.bytes;
          }
          const buyuk = [...bitenler].sort((a, b) => b.bytes - a.bytes).slice(0, 6);

          console.log(`\n=== ${ad} (${etiket}) ${hedef} ===`);
          console.log(`  istek ${bitenler.length} · TOPLAM ${kb(toplam)} KB · LCP ${Math.round(m.lcp)} ms · CLS ${m.cls.toFixed(4)}`);
          console.log('  ' + Object.entries(grup).sort((a, b) => b[1] - a[1]).map(([k, v]) => `${k}=${kb(v)}`).join(' · '));
          for (const b of buyuk) console.log(`    ${String(Math.round(b.bytes / 1024)).padStart(5)} KB  ${b.type.padEnd(10)} ${b.url}`);
        } catch (e) {
          console.log('HATA', ad, etiket, String(e.message).slice(0, 100));
        }
      } finally {
        await browser.close();
      }
    }
  }
})();
