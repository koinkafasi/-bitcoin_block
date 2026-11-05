'use strict';
/**
 * E2E ortak yardımcıları.
 *
 * Tasarım ilkesi: testler deterministik olmalı. Bu yüzden
 *  - sabit viewport + deviceScaleFactor kullanılır,
 *  - ağ sessizleşene kadar beklenir (networkidle),
 *  - ölçüm sayfa içinde (browser context) yapılır; "geçici" DOM durumu
 *    dışarıdan yorumlanmaz.
 */

const puppeteer = require('puppeteer');

const BASE_URL = process.env.E2E_BASE_URL || 'http://127.0.0.1:8070';

/** Testlerde kullanılacak masaüstü genişlikleri. */
const DESKTOP_WIDTHS = [1280, 1440];
/** Tablet / mobil genişlikler. */
const MOBILE_WIDTHS = [320, 360, 390, 414, 768];

/** Görsel yükleme bitene kadar bekleyip gezin. */
async function goto(page, path, { wait = 'networkidle2' } = {}) {
  const url = path.startsWith('http') ? path : BASE_URL + path;
  const resp = await page.goto(url, { waitUntil: 'domcontentloaded', timeout: 45000 });
  try {
    await page.waitForNetworkIdle({ idleTime: 600, timeout: 20000 });
  } catch (_) {
    /* dış kaynaklar (yahoo/espn vb.) yavaş olabilir; sessiz geç */
  }
  await page.evaluate(() => document.fonts && document.fonts.ready).catch(() => {});
  return resp;
}

/**
 * Sayfadaki yatay taşma ölçümü.
 * Dönüş: { clientW, scrollW, overflowing: [{tag, cls, w, right}] }
 *
 * `overflow-x:hidden` ile kırpılan öğeler gerçek hata değildir; bu yüzden
 * taşan bir ataya sahip öğeler listelenmez (görsel olarak taşma yaratmazlar).
 */
async function measureOverflow(page) {
  return page.evaluate(() => {
    const cw = document.documentElement.clientWidth;
    const overflowing = [];
    const seen = new Set();

    document.querySelectorAll('body *').forEach((el) => {
      const r = el.getBoundingClientRect();
      if (r.width === 0 && r.height === 0) return;
      const cs = getComputedStyle(el);
      if (cs.position === 'fixed') return;
      if (r.right <= cw + 1) return;

      // görünür alanı kırpabilen bir ata var mı?
      let p = el.parentElement;
      let clipped = false;
      while (p) {
        const pcs = getComputedStyle(p);
        if (['hidden', 'auto', 'scroll', 'clip'].includes(pcs.overflowX)) {
          clipped = true;
          break;
        }
        p = p.parentElement;
      }
      if (clipped) return;

      const key = el.tagName + '.' + el.className + '@' + Math.round(r.right);
      if (seen.has(key)) return;
      seen.add(key);
      overflowing.push({
        tag: el.tagName.toLowerCase(),
        cls: String(el.className || '').slice(0, 60),
        w: Math.round(r.width),
        right: Math.round(r.right),
      });
    });

    return {
      clientW: cw,
      scrollW: document.documentElement.scrollWidth,
      bodyScrollW: document.body.scrollWidth,
      overflowing: overflowing.slice(0, 10),
    };
  });
}

/** Görünür <img> öğelerinin gerçek çizilmiş boyutları. */
async function measureImages(page) {
  return page.evaluate(() => {
    const out = [];
    document.querySelectorAll('img').forEach((img) => {
      const r = img.getBoundingClientRect();
      if (r.width < 2 || r.height < 2) return;
      const cs = getComputedStyle(img);
      if (cs.display === 'none' || cs.visibility === 'hidden') return;
      out.push({
        w: Math.round(r.width),
        h: Math.round(r.height),
        cls: String(img.className || '').slice(0, 40),
        parent: String(img.parentElement && img.parentElement.className || '').slice(0, 40),
      });
    });
    return out;
  });
}

async function setViewport(page, width, height = 900) {
  await page.setViewport({ width, height, deviceScaleFactor: 1 });
  // yeniden düzen (layout) tamamen otursun
  await new Promise((r) => setTimeout(r, 150));
}

/** Tek bir sayfayı verilen tüm genişliklerde gezip taşma ölçer. */
async function checkNoOverflowAcrossWidths(page, path, widths, height = 900) {
  const results = [];
  for (const w of widths) {
    await setViewport(page, w, height);
    await goto(page, path);
    const m = await measureOverflow(page);
    results.push({ width: w, ...m });
  }
  return results;
}

/** Yüklenemeyen (kırık) görseller — yalnızca kendi kaynaklarımız. */
async function measureBrokenImages(page, origin) {
  return page.evaluate((o) => {
    return [...document.querySelectorAll('img')]
      .filter((i) => {
        const src = i.getAttribute('src') || '';
        return src.startsWith(o) && i.complete && i.naturalWidth === 0;
      })
      .map((i) => i.getAttribute('src'));
  }, origin);
}

/** Tarayıcı başlat (CI'da --no-sandbox gerekir). */
async function launchBrowser() {
  return puppeteer.launch({
    headless: 'new',
    args: [
      '--no-sandbox',
      '--disable-dev-shm-usage',
      '--disable-gpu',
      '--font-render-hinting=none',
    ],
  });
}

/**
 * Dış kaynakları (ESPN logoları, Yahoo Finance vb.) engeller.
 *
 * Gerekçe: testler yalnızca kendi sunucumuzun çıktısını doğrulamalı. Dış
 * ağa bağımlılık hem yavaş hem kararsızdır (zaman aşımı/rate-limit) ve
 * "Navigating frame detached" gibi tarayıcı hatalarına yol açar.
 * `resourceTypes: ['image']` dışında tüm çapraz kaynak istekleri kesilir.
 */
async function blockExternal(page, origin) {
  await page.setRequestInterception(true);
  page.on('request', (req) => {
    try {
      const url = req.url();
      if (url.startsWith(origin) || url.startsWith('data:') || url.startsWith('blob:')) {
        req.continue();
      } else {
        req.abort();
      }
    } catch (_) {
      // istek iptal edilmiş olabilir
    }
  });
}

/** Test sayfası + tarayıcı kurar; kapanışı test içinde garanti eder. */
async function withPage(fn, { width = 1280, height = 900, blockExternalResources = true } = {}) {
  const browser = await launchBrowser();
  try {
    const page = await browser.newPage();
    await page.setCacheEnabled(false);
    if (blockExternalResources) {
      await blockExternal(page, BASE_URL);
    }
    await setViewport(page, width, height);
    const errors = [];
    page.on('pageerror', (e) => errors.push(String(e.message)));
    return await fn(page, errors);
  } finally {
    await browser.close();
  }
}

/**
 * Admin oturumu kurulmuş sayfa (TB_SESSION çevre değişkeninden).
 * TB_SESSION yoksa sayfa /admin/giris'e yönlenir; testler bunu atlamalıdır.
 */
async function withAdminPage(fn, opts = {}) {
  const browser = await launchBrowser();
  try {
    const sid = process.env.TB_SESSION;
    if (sid) {
      await browser.setCookie({
        name: 'tb_session',
        value: sid,
        domain: new URL(BASE_URL).hostname,
        path: '/',
      });
    }
    const page = await browser.newPage();
    await page.setCacheEnabled(false);
    if (opts.blockExternalResources !== false) {
      await blockExternal(page, BASE_URL);
    }
    await setViewport(page, opts.width || 1400, opts.height || 900);
    const errors = [];
    page.on('pageerror', (e) => errors.push(String(e.message)));
    return await fn(page, errors);
  } finally {
    await browser.close();
  }
}

/**
 * Kategori testleri için bilinen tek bloklu eski düzen (kategori_liste).
 * Global kategori düzeni bağımsız modüllere ayrıldığında eski şablon
 * testlerinin veri kaynağı olarak kullanılır.
 */
function legacyCategoryBlock(extra = {}) {
  return {
    id: 'e2e-legacy-kategori',
    type: 'kategori_liste',
    visible: true,
    opts: Object.assign({
      zone: 'govde', view: 'klasik', hero_style: 'portre', hero_count: 5,
      hero_width: 58, hero_ratio: '4/5', list_cols: '2', count: 24,
      overlay_gradient: 'bottom', overlay_color: '#0b0a11',
      overlay_opacity: '62', overlay_height: 55,
      pager_shape: 'outline', pager_pos: 'center', pager_size: 30,
      akis_limit: 60, akis_per: 12, akis_auto: true,
    }, extra),
  };
}

/**
 * Kategori tasarımcısında kategori_liste bloğu yoksa geçici bir eski düzen
 * kaydeder. Dönüş: { synthetic, blocks } — blocks JSON'dur; testler bunu
 * temel alabilir. Çağıran taraf özgün düzeni geri yüklemelidir.
 */
async function ensureLegacyCategory(page, { pageKey, builderUrl, opts = {} } = {}) {
  const has = await page.evaluate(
    () => (window.__bldInit || []).some((b) => b.type === 'kategori_liste')
  );
  if (has) {
    return {
      synthetic: false,
      blocks: await page.evaluate(() => JSON.stringify(window.__bldInit || [])),
    };
  }
  const block = legacyCategoryBlock(opts);
  const url = '/admin/tasarim/kaydet?sayfa=' + encodeURIComponent(pageKey);
  const res = await page.evaluate(async ({ url, block }) => {
    const r = await fetch(url, {
      method: 'POST',
      credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ blocks: [block] }),
    });
    return await r.json();
  }, { url, block });
  if (!res.ok) throw new Error('geçici kategori_liste düzeni kaydedilemedi');
  let blocks = JSON.stringify([block]);
  if (builderUrl) {
    await goto(page, builderUrl);
    await page.waitForFunction(() => window.__bldInit, { timeout: 15000 });
    blocks = await page.evaluate(() => JSON.stringify(window.__bldInit || []));
  }
  return { synthetic: true, blocks };
}

/**
 * Tasarımcıda belirli bir modül tipi yoksa GEÇİCİ olarak ekler, işi yapar ve
 * özgün düzeni geri yükler.
 *
 * Neden: "vitrin düzeni seçilebilir" testleri `select[data-k="duzen"]`
 * arıyor; o alan yalnız `kategori_panel` modülünde var. Taze E2E
 * veritabanında ana sayfa düzeninde bu modül YOK, dolayısıyla test hiçbir
 * şey ölçemiyordu (Faz 208: dolu bir veritabanında geçiyor, temizde
 * düşüyordu — testin kendi kurmadığı sahneye bağlı kalması).
 *
 * Sahne testin kendisi tarafından kurulur ve `finally` ile geri alınır;
 * kalıcı veri değişmez.
 */
async function withTempBlock(page, { pageKey = 'ana', type, opts = {}, builderUrl }, fn) {
  const kaydetURL = '/admin/tasarim/kaydet?sayfa=' + encodeURIComponent(pageKey);
  const kaydet = (blocks) => page.evaluate(async ({ url, blocks }) => {
    const r = await fetch(url, {
      method: 'POST',
      credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ blocks }),
    });
    return await r.json();
  }, { url: kaydetURL, blocks });

  await goto(page, builderUrl || ('/admin/tasarim?sayfa=' + pageKey));
  await page.waitForFunction(() => window.__bldInit, { timeout: 15000 });
  const ozgun = await page.evaluate(() => window.__bldInit || []);

  const varMi = ozgun.some((b) => b && b.type === type);
  let eklenen = null;
  if (!varMi) {
    eklenen = Object.assign({
      id: 'e2e-gecici-' + type,
      type,
      visible: true,
      opts: Object.assign({ zone: 'govde' }, opts),
    });
    const sonuc = await kaydet(ozgun.concat([eklenen]));
    if (!sonuc.ok) throw new Error(type + ' modülü eklenemedi');
    // Yeni düzeni sunucudan yükle (tasarımcı blokları boot verisinden okur).
    await goto(page, builderUrl || ('/admin/tasarim?sayfa=' + pageKey));
    await page.waitForFunction(() => window.__bldInit, { timeout: 15000 });
  }

  try {
    return await fn({ eklendi: !varMi });
  } finally {
    if (eklenen) {
      await kaydet(ozgun); // sahneyi temizle: kalıcı değişiklik bırakma
    }
  }
}

module.exports = {
  BASE_URL,
  DESKTOP_WIDTHS,
  MOBILE_WIDTHS,
  goto,
  measureOverflow,
  measureImages,
  measureBrokenImages,
  setViewport,
  checkNoOverflowAcrossWidths,
  launchBrowser,
  blockExternal,
  withPage,
  withAdminPage,
  legacyCategoryBlock,
  ensureLegacyCategory,
  withTempBlock,
};
