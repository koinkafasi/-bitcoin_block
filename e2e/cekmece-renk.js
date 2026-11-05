const puppeteer = require('puppeteer');
const BASE = process.argv[2];

// Çekmece (hamburger menü) bağlantı renklerini ÖLÇ.
//
// Şüphe: `.hp-drawer a` hiç `color` vermiyor; genel `a { color: inherit }`
// panelden miras alıyor olmalı — ama alt kategori listesi
// `--hp-text-muted` (#71717a) kullanıyor ve 0.72rem'de kontrast düşük.
// Tahmin etmek yerine tarayıcıdan okunuyor.

function parlaklik(rgb) {
  const [r, g, b] = rgb.map((v) => {
    const s = v / 255;
    return s <= 0.03928 ? s / 12.92 : Math.pow((s + 0.055) / 1.055, 2.4);
  });
  return 0.2126 * r + 0.7152 * g + 0.0722 * b;
}

function kontrast(a, b) {
  const l1 = parlaklik(a), l2 = parlaklik(b);
  const acik = Math.max(l1, l2), koyu = Math.min(l1, l2);
  return (acik + 0.05) / (koyu + 0.05);
}

function ayikla(s) {
  const m = (s || '').match(/rgba?\(([^)]+)\)/);
  if (!m) return null;
  const p = m[1].split(',').map((x) => parseFloat(x.trim()));
  return [p[0], p[1], p[2]];
}

(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'], protocolTimeout: 90000 });
  const p = await b.newPage();
  await p.setViewport({ width: 1440, height: 1000 });
  await p.goto(BASE + '/', { waitUntil: 'networkidle2', timeout: 60000 });

  // Çekmeceyi aç.
  await p.evaluate(() => {
    const btn = document.querySelector('[data-menu-btn]');
    if (btn) btn.click();
    const d = document.getElementById('hpDrawer');
    if (d) d.classList.add('open');
  });
  await new Promise((r) => setTimeout(r, 400));

  const koyu = process.argv[3] === 'koyu';
  if (koyu) {
    await p.evaluate(() => document.documentElement.setAttribute('data-theme', 'dark'));
    await new Promise((r) => setTimeout(r, 300));
  }

  const veri = await p.evaluate(() => {
    const panel = document.querySelector('.hp-drawer-panel');
    if (!panel) return { yok: true };
    const pr = panel.getBoundingClientRect();
    const zemin = getComputedStyle(panel).backgroundColor;

    const ornekle = (sel, ad) => {
      const el = panel.querySelector(sel);
      if (!el) return null;
      const s = getComputedStyle(el);
      const r = el.getBoundingClientRect();
      return {
        ad,
        secici: sel,
        metin: (el.textContent || '').trim().slice(0, 24),
        renk: s.color,
        boyut: s.fontSize,
        agirlik: s.fontWeight,
        opaklik: s.opacity,
        zemin: s.backgroundColor,
        gorunur: r.width > 0 && r.height > 0,
      };
    };

    return {
      yok: false,
      panelZemin: zemin,
      panelRenk: getComputedStyle(panel).color,
      panelGenislik: Math.round(pr.width),
      ornekler: [
        ornekle('.hp-drawer-title', 'başlık'),
        ornekle('.hp-drawer-group', 'grup etiketi'),
        ornekle('.hp-drawer-cats > .hp-drawer-list li a', 'kategori linki'),
        ornekle('.hp-drawer-grid > a', 'içerik/servis linki'),
        ornekle('.hp-drawer-subtoggle', 'alt kategori düğmesi'),
        ornekle('.hp-drawer .select', 'tercih select'),
        ornekle('.hp-drawer-close', 'kapat'),
      ].filter(Boolean),
    };
  });

  if (veri.yok) {
    console.log('çekmece bulunamadı');
    await b.close();
    return;
  }

  console.log('panel zemin:', veri.panelZemin, '· metin:', veri.panelRenk, '· genişlik:', veri.panelGenislik);
  const zeminRGB = ayikla(veri.panelZemin);
  console.log('');
  console.log('--- ölçüm ---');
  veri.ornekler.forEach((o) => {
    const c = ayikla(o.renk);
    const k = c && zeminRGB ? kontrast(c, zeminRGB).toFixed(2) : '?';
    const bayrak = k !== '?' && k < 4.5 ? '  ← KONTRAST DÜŞÜK' : '';
    console.log(
      `${o.ad.padEnd(20)} renk ${o.renk.padEnd(22)} ${o.boyut.padEnd(7)} w${o.agirlik} ` +
      `opaklık ${o.opaklik} · kontrast ${k}:1${bayrak} · "${o.metin}"`
    );
  });

  // Alt kategori panelini de aç (varsa düğme).
  const alt = await p.evaluate(() => {
    const t = document.querySelector('.hp-drawer-subtoggle');
    if (!t) return null;
    t.click();
    return true;
  });
  if (alt) {
    await new Promise((r) => setTimeout(r, 350));
    const altVeri = await p.evaluate(() => {
      const panel = document.querySelector('.hp-drawer-panel');
      const a = panel.querySelector('.hp-drawer-subs a');
      if (!a) return null;
      const s = getComputedStyle(a);
      return { renk: s.color, boyut: s.fontSize, metin: (a.textContent || '').trim().slice(0, 24) };
    });
    if (altVeri) {
      const c = ayikla(altVeri.renk);
      const k = c && zeminRGB ? kontrast(c, zeminRGB).toFixed(2) : '?';
      console.log(
        `${'alt kategori'.padEnd(20)} renk ${altVeri.renk.padEnd(22)} ${altVeri.boyut.padEnd(7)} ` +
        `· kontrast ${k}:1${k !== '?' && k < 4.5 ? '  ← KONTRAST DÜŞÜK' : ''} · "${altVeri.metin}"`
      );
    }
  }

  await b.close();
})();
