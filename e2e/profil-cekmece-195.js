const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

// Faz 195 doğrulaması (düzeltilmiş ölçüm).
//
// Önceki sürüm iki yanlış alarm veriyordu:
//   1) Çekmece `.classList.add('open')` ile açılıyordu → JS işaretlemesi
//      çalışmıyordu. Artık gerçek düğme tıklanıyor.
//   2) "Tüm bağlantılar ≥28px" kontrolü GİZLİ alt kategori görünümlerini de
//      sayıyordu (yükseklik 0). Artık yalnız görünür olanlar ölçülüyor.
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'], protocolTimeout: 90000 });
  const p = await b.newPage();
  await p.setViewport({ width: 1440, height: 1000 });
  const hatalar = [];
  p.on('pageerror', (e) => hatalar.push(e.message.slice(0, 90)));
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  const coz = (s) => {
    const m = (s || '').match(/rgba?\(([^)]+)\)/);
    return m ? m[1].split(',').slice(0, 3).map((x) => parseFloat(x.trim())) : null;
  };
  const parlak = (rgb) => {
    const [r, g, bl] = rgb.map((v) => {
      const s = v / 255;
      return s <= 0.03928 ? s / 12.92 : Math.pow((s + 0.055) / 1.055, 2.4);
    });
    return 0.2126 * r + 0.7152 * g + 0.0722 * bl;
  };
  const kontrast = (a, b2) => {
    const l1 = parlak(a), l2 = parlak(b2);
    return (Math.max(l1, l2) + 0.05) / (Math.min(l1, l2) + 0.05);
  };

  const cekmece = async (tema) => {
    await p.evaluate((t) => document.documentElement.setAttribute('data-theme', t), tema);
    await new Promise((r) => setTimeout(r, 350));
    return p.evaluate(() => {
      const panel = document.querySelector('.hp-drawer-panel');
      if (!panel) return null;
      const gorunur = (el) => {
        const r = el.getBoundingClientRect();
        return r.width > 0 && r.height > 0;
      };
      const oku = (sel) => {
        const el = panel.querySelector(sel);
        if (!el) return null;
        return { renk: getComputedStyle(el).color, boyut: getComputedStyle(el).fontSize };
      };
      const baglar = [...panel.querySelectorAll('a')].filter(gorunur);
      const kisa = baglar
        .filter((a) => a.getBoundingClientRect().height < 28)
        .map((a) => (a.textContent || '').trim().slice(0, 18) + '=' + Math.round(a.getBoundingClientRect().height));
      const etiketsiz = baglar
        .filter((a) => (a.textContent || '').trim().length < 3)
        .map((a) => JSON.stringify((a.textContent || '').trim()));
      return {
        zemin: getComputedStyle(panel).backgroundColor,
        grup: oku('.hp-drawer-group'),
        alt: oku('.hp-drawer-cats .hp-drawer-subs a'),
        gorunenBag: baglar.length,
        kisa,
        etiketsiz,
      };
    });
  };

  // --- 1) Profil simgesi ---
  await p.goto(BASE + '/', { waitUntil: 'networkidle2', timeout: 60000 });
  const profil = await p.evaluate(() => {
    const a = document.querySelector('.member-btn');
    return a ? a.getAttribute('href') : null;
  });
  console.log('=== PROFIL SIMGESI ===');
  console.log('href:', profil, '·', profil === '/hesabim' ? 'OK' : 'HATA');

  // --- 2) Geçerli sayfa işareti: çekmecede OLAN bir sayfa (/akis) ---
  await p.goto(BASE + '/akis', { waitUntil: 'networkidle2', timeout: 60000 });
  await p.evaluate(() => {
    const btn = document.querySelector('[data-menu-btn]');
    if (btn) btn.click();
  });
  await new Promise((r) => setTimeout(r, 400));
  const akis = await p.evaluate(() => {
    const panel = document.querySelector('.hp-drawer-panel');
    return [...panel.querySelectorAll('a[aria-current="page"]')]
      .map((a) => (a.textContent || '').trim().slice(0, 22));
  });
  console.log('');
  console.log('=== GECERLI SAYFA ISARETI (/akis) ===');
  console.log('isaretli:', JSON.stringify(akis), '·', akis.length ? 'OK' : 'HATA');

  // --- 3) Açık + koyu tema ---
  console.log('');
  console.log('=== CEKMECE (ACIK TEMA) ===');
  const acik = await cekmece('light');
  const zg = coz(acik.zemin);
  console.log('zemin:', acik.zemin);
  [['grup', acik.grup], ['alt', acik.alt]].forEach(([ad, o]) => {
    if (!o) return;
    const k = kontrast(coz(o.renk), zg);
    console.log('  ' + ad.padEnd(5) + ' ' + o.renk.padEnd(20) + ' ' + o.boyut.padEnd(7) +
      ' kontrast ' + k.toFixed(2) + ':1' + (k < 4.5 ? '  <- DUSUK' : ''));
  });

  console.log('');
  console.log('=== CEKMECE (KOYU TEMA) ===');
  const koyu = await cekmece('dark');
  const zk = coz(koyu.zemin);
  console.log('zemin:', koyu.zemin);
  [['grup', koyu.grup], ['alt', koyu.alt]].forEach(([ad, o]) => {
    if (!o) return;
    const k = kontrast(coz(o.renk), zk);
    console.log('  ' + ad.padEnd(5) + ' ' + o.renk.padEnd(20) + ' ' + o.boyut.padEnd(7) +
      ' kontrast ' + k.toFixed(2) + ':1' + (k < 4.5 ? '  <- DUSUK' : ''));
  });

  console.log('');
  console.log('=== ERISILEBILIRLIK ===');
  console.log('gorunen baglanti:', koyu.gorunenBag);
  console.log('28px alti:', koyu.kisa.length ? JSON.stringify(koyu.kisa) : 'yok');
  console.log('etiketsiz (yalniz emoji):', koyu.etiketsiz.length ? JSON.stringify(koyu.etiketsiz) : 'yok');

  console.log('');
  console.log('JS hatasi:', hatalar.length ? [...new Set(hatalar)].slice(0, 3).join(' | ') : 'yok');
  await b.close();
})();
