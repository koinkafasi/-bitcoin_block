'use strict';
/**
 * Üst manşet kart stilleri (haberturk tarzı görsel üstü başlık):
 * - ayarlarda düzen, kart stili, başlık/bant/gradyan renkleri görünür
 * - her stil doğru katmanı ve konumu üretir
 * - başlık rengi, gradyan rengi/saydamlığı ve gölge uygulanır
 * - ızgara düzeni sütun sayısına uyar
 * - başlık kapatılsa bile metin DOM'da kalır (SEO)
 *
 * Kalıcı düzen değiştirmez; önizleme HTML'i canlı sayfaya enjekte edilir.
 */

const test = require('node:test');
const assert = require('node:assert/strict');
const { withAdminPage, withPage, goto, setViewport } = require('../lib/harness');

const STYLES = ['klasik', 'overlay', 'overlay-top', 'band', 'frame', 'corner'];

test('üst manşet: kart stili alanları ve tüm stiller', async (t) => {
  if (!process.env.TB_SESSION) return t.skip('TB_SESSION yok (admin testi)');

  let previews = {};
  await withAdminPage(async (page, errors) => {
    await goto(page, '/admin/tasarim?sayfa=ana');
    await page.waitForSelector('.bp-chip[data-id]', { timeout: 15000 });

    const ui = await page.evaluate(() => {
      const chips = [...document.querySelectorAll('.bp-chip[data-id]')];
      for (const ch of chips) {
        const ty = ch.querySelector('.ty');
        if (ty && ty.textContent.trim() === 'ust_manset') {
          ch.querySelector('button[data-act="cfg"]').click();
          const q = (k) => ch.querySelector('[data-k="' + k + '"]');
          const style = q('um_style');
          return {
            ok: true,
            styles: style ? [...style.options].map((o) => o.value).filter(Boolean) : [],
            layout: q('um_layout') ? [...q('um_layout').options].map((o) => o.value).filter(Boolean) : [],
            cols: q('um_cols') ? [...q('um_cols').options].map((o) => o.value).filter(Boolean) : [],
            colors: ['um_title_color', 'um_title_bg', 'um_grad_color'].map((k) => !!q(k)),
            sizes: ['um_title_size', 'um_grad_opacity'].map((k) => !!q(k)),
            shadow: !!q('um_shadow'),
            card: ['um_card_bg', 'um_card_dark_bg', 'um_card_shadow', 'um_title_pad'].map((k) => !!q(k)),
          };
        }
      }
      return { ok: false };
    });
    assert.ok(ui.ok, 'üst manşet ayarları açılmalı');
    assert.deepEqual(ui.styles, STYLES, 'altı kart stili olmalı');
    assert.deepEqual(ui.layout, ['serit', 'izgara']);
    assert.deepEqual(ui.cols, ['2', '3', '4']);
    assert.deepEqual(ui.colors, [true, true, true], 'renk alanları olmalı');
    assert.deepEqual(ui.sizes, [true, true], 'boyut/saydamlık alanları olmalı');
    assert.ok(ui.shadow, 'gölge anahtarı olmalı');
    assert.deepEqual(ui.card, [true, true, true, true], 'kart arka planı/gölge/boşluk alanları olmalı');

    const blocks = STYLES.map((st) => ({
      id: 'um-' + st,
      type: 'ust_manset',
      opts: {
        um_style: st,
        um_layout: 'izgara',
        um_cols: 4,
        count: 4,
        um_title_color: '#ffcc00',
        um_grad_color: '#001122',
        um_grad_opacity: 75,
        um_title_size: 18,
      },
    }));
    const res = await page.evaluate(async (blocks) => {
      const r = await fetch('/admin/tasarim/onizleme?sayfa=ana', {
        method: 'POST',
        credentials: 'same-origin',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ blocks }),
      });
      return await r.json();
    }, blocks);
    assert.equal(res.ok, true, 'önizleme üretilmeli');
    for (const it of res.items) previews[it.id] = it.html;

    // Başlık kapatıldığında metin DOM'da kalmalı.
    const hidden = await page.evaluate(async () => {
      const r = await fetch('/admin/tasarim/onizleme?sayfa=ana', {
        method: 'POST',
        credentials: 'same-origin',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          blocks: [{ id: 'um-hidden', type: 'ust_manset', opts: { um_style: 'overlay', overlay_title: false, count: 2 } }],
        }),
      });
      return await r.json();
    });
    assert.equal(hidden.ok, true);
    previews['um-hidden'] = (hidden.items.find((x) => x.id === 'um-hidden') || {}).html || '';

    // Kart kabuğu: açık/koyu arka plan, gölge kapatma ve başlık yan boşluğu.
    const cardRes = await page.evaluate(async () => {
      const r = await fetch('/admin/tasarim/onizleme?sayfa=ana', {
        method: 'POST',
        credentials: 'same-origin',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          blocks: [
            {
              id: 'um-card-ayar',
              type: 'ust_manset',
              opts: {
                um_layout: 'serit',
                count: 3,
                card_style: 'bloomberg-cizgi',
                um_card_bg: '#123456',
                um_card_dark_bg: '#654321',
                um_card_shadow: false,
                um_title_pad: 24,
              },
            },
            {
              id: 'um-bloomberg-cizgi-temiz',
              type: 'ust_manset',
              opts: { um_layout: 'serit', count: 3, card_style: 'bloomberg-cizgi' },
            },
          ],
        }),
      });
      return await r.json();
    });
    assert.equal(cardRes.ok, true, 'kart kabuğu önizlemesi üretilmeli');
    for (const it of cardRes.items) previews[it.id] = it.html;

    assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
  });

  await withPage(async (page) => {
    await setViewport(page, 1280, 1000);
    await goto(page, '/');

    const measure = (id) => page.evaluate((id) => {
      const box = document.getElementById('um-test');
      const sec = box.querySelector('.ust-manset');
      const card = sec.querySelector('.um-card');
      const media = card.querySelector('.um-media');
      const title = card.querySelector('.um-t');
      const shade = card.querySelector('.um-shade');
      const s = getComputedStyle(title);
      return {
        cls: sec.className,
        cardCls: card.className,
        cards: sec.querySelectorAll('.um-card').length,
        titleInMedia: !!media.querySelector('.um-t'),
        titleText: (title.textContent || '').trim(),
        titleCls: title.className,
        pos: s.position,
        top: s.top,
        bottom: s.bottom,
        color: s.color,
        bg: s.backgroundColor,
        shadow: s.textShadow,
        borderLeft: s.borderLeftWidth,
        radius: s.borderRadius,
        fontSize: s.fontSize,
        shadeDisplay: shade ? getComputedStyle(shade).display : 'yok',
        shadeImg: shade ? getComputedStyle(shade).backgroundImage : '',
        shadeOpacity: shade ? getComputedStyle(shade).opacity : '',
        cols: getComputedStyle(sec).gridTemplateColumns.split(' ').filter(Boolean).length,
      };
    }, id);

    const inject = (html) => page.evaluate((html) => {
      let box = document.getElementById('um-test');
      if (!box) {
        box = document.createElement('div');
        box.id = 'um-test';
        document.body.appendChild(box);
      }
      box.innerHTML = html;
    }, html);

    const r = {};
    for (const st of STYLES) {
      await inject(previews['um-' + st]);
      r[st] = await measure();
      assert.ok(r[st].cards > 0, st + ': kart basılmalı');
    }

    // Ortak: görsel üstü stiller başlığı medya içinde ve mutlak konumda tutar.
    for (const st of ['overlay', 'overlay-top', 'band', 'frame', 'corner']) {
      assert.ok(r[st].cardCls.includes('um-ov'), st + ': kart görsel üstü sınıfı almalı');
      assert.ok(r[st].titleInMedia, st + ': başlık görselin içinde olmalı');
      assert.equal(r[st].pos, 'absolute', st + ': başlık mutlak konumda olmalı');
    }
    assert.ok(r.klasik.cardCls.includes('um-ov') === false, 'klasik: başlık görselin altında');
    assert.equal(r.klasik.pos, 'static', 'klasik: başlık akışta olmalı');

    // Alt hizalı stiller ve üst hizalı stil.
    assert.equal(r.overlay.bottom, '0px');
    assert.equal(r['overlay-top'].top, '0px');
    assert.ok(r['overlay-top'].shadeImg.includes('linear-gradient(to bottom')
      || r['overlay-top'].shadeImg.includes('linear-gradient(rgb'),
      'üst stil: gradyan yukarıdan aşağı olmalı: ' + r['overlay-top'].shadeImg);

    // Bant: gradyan katmanı basılmaz, arka plan renkli.
    assert.ok(['none', 'yok'].includes(r.band.shadeDisplay), 'bant: gradyan gizlenmeli');
    assert.match(r.band.bg, /rgba\(11, 10, 17, 0\.82\)/, 'bant: arka plan yarı saydam olmalı');

    // Çerçeve: içeriden boşluk + yuvarlak köşe.
    assert.equal(r.frame.bottom, '10px', 'çerçeve: alttan boşluk olmalı');
    assert.notEqual(r.frame.radius, '0px', 'çerçeve: yuvarlak köşe olmalı');

    // Köşe: sol vurgu çizgisi.
    assert.equal(r.corner.borderLeft, '4px', 'köşe: sol vurgu çizgisi olmalı');

    // Renk ve tipografi ayarları.
    assert.equal(r.overlay.color, 'rgb(255, 204, 0)', 'başlık rengi uygulanmalı');
    assert.equal(r.overlay.fontSize, '18px', 'başlık boyutu uygulanmalı');
    assert.ok(r.overlay.shadeImg.includes('rgb(0, 17, 34)'), 'gradyan rengi uygulanmalı');
    assert.equal(r.overlay.shadeOpacity, '0.75', 'gradyan saydamlığı uygulanmalı');
    assert.notEqual(r.overlay.shadow, 'none', 'başlık gölgesi varsayılan açık');

    // Izgara: 1280px'te 4 sütun.
    assert.equal(r.overlay.cols, 4, 'ızgara 4 sütun olmalı');

    // Mobil: ızgara 2 sütuna düşmeli ve taşma olmamalı.
    await setViewport(page, 390, 844);
    await inject(previews['um-overlay']);
    const mob = await measure();
    assert.equal(mob.cols, 2, 'mobilde 2 sütun olmalı');
    const overflow = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
    assert.equal(overflow, 0, 'mobilde yatay taşma olmamalı');

    // Başlık kapatma: sr-only ama metin korunur.
    await setViewport(page, 1280, 1000);
    await inject(previews['um-hidden']);
    const hide = await measure();
    assert.ok(hide.titleCls.includes('sr-only'), 'kapalı başlık sr-only olmalı');
    assert.ok(hide.titleText.length > 3, 'kapalı başlık metni DOM’da kalmalı');

    // Kart kabuğu: açık/koyu arka plan, gölge kapatma ve başlık yan boşluğu.
    const measureCard = (tema) => page.evaluate((tema) => {
      document.documentElement.setAttribute('data-theme', tema);
      const sec = document.querySelector('#um-test .ust-manset');
      const card = sec.querySelector('.um-card');
      const title = card.querySelector('.um-t');
      const s = getComputedStyle(sec);
      const probe = document.createElement('div');
      probe.style.background = 'var(--hp-surface)';
      document.body.appendChild(probe);
      const surface = getComputedStyle(probe).backgroundColor;
      probe.remove();
      return {
        tema,
        cardBg: getComputedStyle(card).backgroundColor,
        cardShadow: getComputedStyle(card).boxShadow,
        titlePadLeft: getComputedStyle(title).paddingLeft,
        titlePadRight: getComputedStyle(title).paddingRight,
        titleBg: getComputedStyle(title).backgroundColor,
        surface,
        bgLightVar: s.getPropertyValue('--um-card-bg-light').trim(),
        bgDarkVar: s.getPropertyValue('--um-card-bg-dark').trim(),
        shadowHoverVar: s.getPropertyValue('--um-card-shadow-hover').trim(),
        titlePadVar: s.getPropertyValue('--um-title-pad-x').trim(),
      };
    }, tema);

    await inject(previews['um-card-ayar']);
    const cardLight = await measureCard('light');
    const cardDark = await measureCard('dark');
    assert.equal(cardLight.bgLightVar, '#123456', 'açık tema kart arka planı değişkeni');
    assert.equal(cardLight.bgDarkVar, '#654321', 'koyu tema kart arka planı değişkeni');
    assert.equal(cardLight.cardBg, 'rgb(18, 52, 86)', 'açık tema kart arka planı uygulanmalı');
    assert.equal(cardDark.cardBg, 'rgb(101, 67, 33)', 'koyu tema kart arka planı uygulanmalı');
    assert.equal(cardLight.cardShadow, 'none', 'gölge kapatılınca kart gölgesi olmamalı');
    assert.equal(cardDark.cardShadow, 'none', 'koyu temada da gölge kapalı olmalı');
    assert.equal(cardLight.shadowHoverVar, 'none', 'hover gölgesi de kapanmalı');
    assert.equal(cardLight.titlePadLeft, '24px', 'başlık sol boşluğu uygulanmalı');
    assert.equal(cardLight.titlePadRight, '24px', 'başlık sağ boşluğu uygulanmalı');
    assert.equal(cardLight.titlePadVar, '24px', 'yan boşluk değişkeni');

    // Bloomberg II varsayılanı tema yüzeyi: koyu temada kategorideki tip5
    // kartıyla aynı renge geçer; varsayılan yan boşluk 16px.
    await inject(previews['um-bloomberg-cizgi-temiz']);
    const cleanLight = await measureCard('light');
    const cleanDark = await measureCard('dark');
    assert.equal(cleanLight.titleBg, cleanLight.surface, 'açık temada kutu tema yüzeyi olmalı');
    assert.equal(cleanDark.titleBg, cleanDark.surface, 'koyu temada kutu tema yüzeyi olmalı');
    assert.equal(cleanDark.cardBg, cleanDark.surface, 'koyu temada kart zemini tema yüzeyi olmalı');
    assert.equal(cleanLight.titlePadLeft, '16px', 'Bloomberg II varsayılan sol boşluğu 16px olmalı');

    await page.evaluate(() => document.documentElement.setAttribute('data-theme', 'light'));

    await page.evaluate(() => { const b = document.getElementById('um-test'); if (b) b.remove(); });
  });
});
