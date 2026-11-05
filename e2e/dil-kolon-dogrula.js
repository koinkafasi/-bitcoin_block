const puppeteer = require('puppeteer');
const BASE = process.argv[2];
const SID = process.argv[3];

// Faz 192c — Dil kolonu hizalama doğrulaması (tüm satırlar + sayfa 2).
(async () => {
  const b = await puppeteer.launch({ headless: 'new', args: ['--no-sandbox'], protocolTimeout: 90000 });
  const p = await b.newPage();
  await p.setViewport({ width: 1500, height: 1100 });
  if (SID) await p.setCookie({ name: 'tb_session', value: SID, domain: '127.0.0.1', path: '/' });

  const olc = async (url, etiket) => {
    await p.goto(BASE + url, { waitUntil: 'networkidle2', timeout: 60000 });
    await new Promise((r) => setTimeout(r, 500));
    const d = await p.evaluate(() => {
      const satirlar = [...document.querySelectorAll('table.table tbody tr')]
        .filter((x) => getComputedStyle(x).display !== 'none');
      const olcum = satirlar.map((tr) => {
        const tdler = [...tr.children];
        const dil = tr.querySelector('td.dil-hucre');
        const i = tdler.indexOf(dil);
        const r = tr.getBoundingClientRect();
        const dr = dil.getBoundingClientRect();
        const pills = [...dil.querySelectorAll('.dil-pill')].map((x) => x.getBoundingClientRect());
        const pillUst = pills.length ? Math.min(...pills.map((x) => x.top)) : 0;
        const pillAlt = pills.length ? Math.max(...pills.map((x) => x.bottom)) : 0;
        const pillGen = pills.length ? Math.max(...pills.map((x) => x.right)) - Math.min(...pills.map((x) => x.left)) : 0;
        return {
          satirYuk: Math.round(r.height),
          dilYuk: Math.round(dr.height),
          dilGen: Math.round(dr.width),
          dilUst: Math.round(dr.top),
          satirUst: Math.round(r.top),
          display: getComputedStyle(dil).display,
          pillSayisi: pills.length,
          pillYuk: Math.round(pillAlt - pillUst),
          pillGen: Math.round(pillGen),
          // dikey ortalama: etiket bloğunun satır içindeki konumu
          pillMerkezFark: Math.round(((pillUst + pillAlt) / 2) - (r.top + r.height / 2)),
          // satır yüksekliğini etiketler mi şişiriyor
          sisme: Math.round(dr.height - pillAlt + pillUst),
        };
      });
      return { basliklar: [...document.querySelectorAll('table.table thead th')].map((x) => (x.textContent || '').trim()), satirlar: olcum };
    });

    const hatalar = [];
    if (!d.satirlar.length) hatalar.push('görünen satır yok');
    d.satirlar.forEach((s, i) => {
      if (s.display !== 'table-cell') hatalar.push(`${i + 1}. satır display=${s.display}`);
      if (s.dilYuk !== s.satirYuk) hatalar.push(`${i + 1}. satır yükseklik farkı: dil ${s.dilYuk} vs satır ${s.satirYuk}`);
      if (Math.abs(s.pillMerkezFark) > 2) hatalar.push(`${i + 1}. satır dikey ortalama ${s.pillMerkezFark}px kayık`);
      if (s.pillGen > s.dilGen) hatalar.push(`${i + 1}. satır etiketler hücreden taşıyor (${s.pillGen} > ${s.dilGen})`);
    });
    const genler = [...new Set(d.satirlar.map((s) => s.dilGen))];
    if (genler.length > 1) hatalar.push('kolon genişliği satırlar arasında farklı: ' + genler.join(', '));
    const ustler = d.satirlar.map((s) => s.dilUst - s.satirUst);
    if (new Set(ustler).size > 1) hatalar.push('hücre üst ofseti farklı: ' + ustler.join(', '));

    console.log(`=== ${etiket} (${d.satirlar.length} satır) ===`);
    console.log('  başlıklar:', d.basliklar.filter(Boolean).join(' | '));
    d.satirlar.slice(0, 3).forEach((s, i) => {
      console.log(`  ${i + 1}. satır ${s.satirYuk}px · dil ${s.dilYuk}px/${s.dilGen}px · etiket ${s.pillSayisi} (${s.pillYuk}px) · merkez fark ${s.pillMerkezFark}px`);
    });
    console.log('  kolon genişlikleri:', genler.join(', '), '· üst ofsetleri:', [...new Set(ustler)].join(', '));
    console.log('  SONUÇ:', hatalar.length ? 'HATA → ' + hatalar.slice(0, 4).join(' | ') : 'hizalı ✓');
    return hatalar.length;
  };

  let hata = 0;
  hata += await olc('/admin/icerikler?durum=yayinda', 'Yayında · sayfa 1');
  hata += await olc('/admin/icerikler?durum=yayinda&sayfa=2', 'Yayında · sayfa 2');
  hata += await olc('/admin/icerikler?durum=taslak', 'Taslak · sayfa 1');

  // Dar ekran: etiketler taşıyor mu?
  await p.setViewport({ width: 900, height: 900 });
  hata += await olc('/admin/icerikler?durum=yayinda', 'Dar ekran 900px');

  console.log('\nTOPLAM:', hata === 0 ? 'tamam' : hata + ' bulgu');
  await b.close();
})();
