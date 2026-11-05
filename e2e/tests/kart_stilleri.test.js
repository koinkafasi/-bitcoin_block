'use strict';
/**
 * Kart stili ailesi (Bloomberg ve alternatifleri) — public yüzey denetimi.
 *
 * KAPSAM (salt-okunur; kullanıcı düzeni/ayarları DEĞİŞTİRİLMEZ):
 *  1) public.css tüm kart stillerini ve koyu tema kuralını içeriyor,
 *  2) tasarımcı panelinde "Kart stili" seçici ve Bloomberg seçeneği var,
 *  3) seçici kategori listesi + üst manşet + öne çıkanlar modüllerinde var,
 *  4) sayfada JS hatası yok.
 *
 * Gerçek render doğrulaması birim testlerinde (kart_stili_test.go) yapılır;
 * burada yalnız varlık/dağıtım kontrol edilir.
 */

const test = require('node:test');
const assert = require('node:assert/strict');
const { goto, setViewport, withAdminPage, BASE_URL } = require('../lib/harness');

const STILLER = [
  'cs-scope', 'cs-bloomberg', 'cs-bloomberg-cizgi', 'cs-bant-alt',
  'cs-vurgu', 'cs-cerceve', 'cs-kose', 'cs-cam', 'cs-alt-cizgi',
];

test('kart stilleri public.css içinde tanımlı', async () => {
  const css = await fetch(BASE_URL + '/static/public.css').then((r) => r.text());
  for (const s of STILLER) {
    assert.ok(css.includes('.' + s), 'public.css stil içermeli: ' + s);
  }
  assert.ok(css.includes('[data-theme="dark"] .cs-scope'), 'koyu tema kuralı olmalı');
  assert.ok(css.includes('--cs-dark-bg'), 'koyu tema band rengi değişkeni olmalı');
  assert.ok(css.includes('order: -1'), 'Bloomberg bandı görselin üstüne alınmalı');
  // tip5 akış kartı da stil ailesine girmeli (ak-fig5/ak-cap5/ak-img5).
  for (const sec of ['.cs-scope .ak-fig5', '.cs-scope .ak-cap5', '.cs-scope .ak-img5']) {
    assert.ok(css.includes(sec), 'akış tip5 stili public.css içinde olmalı: ' + sec);
  }
  assert.ok(css.includes('.cs-bloomberg .ak-cap5'), 'Bloomberg tip5 bandı üstte olmalı');
  assert.ok(css.includes('.cs-bloomberg-cizgi .ak-cap5'), 'Bloomberg II tip5 taşan kutusu olmalı');
  assert.ok(css.includes('margin: -48px 0 0'), 'Bloomberg II başlık kutusu görselin üstüne taşmalı');
  assert.ok(css.includes('--cs-light-bg: var(--hp-surface)'), 'Bloomberg II varsayılanı tema yüzeyi olmalı');
  assert.ok(css.includes('.ust-manset.um-title-pad > .um-card .um-t'), 'başlık yan boşluğu kuralı olmalı');
  assert.ok(css.includes('--um-card-bg-light'), 'kart arka planı değişkeni olmalı');
  // Kategori kart gövdesi/gölgesi: açık/koyu tema ayrı, gölge kapatılabilir.
  assert.ok(css.includes('--cs-card-light-bg'), 'kart gövdesi açık tema değişkeni olmalı');
  assert.ok(css.includes('--cs-card-dark-bg'), 'kart gövdesi koyu tema değişkeni olmalı');
  assert.ok(css.includes('--cs-shadow'), 'kart gölgesi değişkeni olmalı');
});

test('tasarımcı panelinde kart stili seçenekleri var', async (t) => {
  if (!process.env.TB_SESSION) return t.skip('TB_SESSION yok (admin testi)');
  await withAdminPage(async (page, errors) => {
    await setViewport(page, 1440, 1000);
    const r = await goto(page, '/admin/tasarim');
    assert.equal(r.status(), 200, 'tasarımcı açılmalı');

    const html = await page.content();
    assert.ok(html.includes('card_style'), 'kart stili alanı tanımlı olmalı');
    assert.ok(html.includes('Bloomberg — başlık bandı görselin ÜSTÜNDE'),
      'Bloomberg seçeneği görünmeli');
    // Koyu tema renk alanları yöneticiye açık.
    for (const alan of ['card_title_bg', 'card_dark_bg', 'card_accent',
      'card_box_bg', 'card_box_dark_bg', 'card_shadow']) {
      assert.ok(html.includes(alan), 'renk alanı bulunmalı: ' + alan);
    }
    // Üç modülde de seçilebilir olmalı (kategori listesi, manşet, öne çıkanlar).
    const sayi = (html.match(/card_style/g) || []).length;
    assert.ok(sayi >= 3, 'en az 3 modülde kart stili alanı olmalı, bulunan: ' + sayi);

    assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
  });
});

test('kart stili tüm seçenekleri listelenir', async (t) => {
  if (!process.env.TB_SESSION) return t.skip('TB_SESSION yok (admin testi)');
  await withAdminPage(async (page) => {
    await goto(page, '/admin/tasarim');
    const html = await page.content();
    for (const etiket of [
      'Klasik (varsayılan)',
      'Bloomberg II — başlık kutusu görselin üstüne taşar',
      'Bant — başlık bandı görselin ALTINDA',
      'Vurgu — aksan renginde dolu bant',
      'Çerçeve — başlık ince çerçeve içinde',
      'Köşe — köşe vurgulu bant',
      'Cam — yarı saydam bulanık bant',
      'Alt çizgi — bant + alt aksan çizgisi',
    ]) {
      assert.ok(html.includes(etiket), 'seçenek görünmeli: ' + etiket);
    }
    const bloombergII = (html.match(/Bloomberg II — başlık kutusu görselin üstüne taşar/g) || []).length;
    assert.ok(bloombergII >= 5,
      'Bloomberg II tüm kart modüllerinde listelenmeli (ev, kategori listesi, akış, 4\'lü, bağımsız akış); bulunan: ' + bloombergII);
  });
});
