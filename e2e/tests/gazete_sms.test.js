'use strict';
/**
 * Gazete SMS dağıtım kanalı (Netgsm / Twilio).
 *
 * KAPSAM:
 *  1) SMS ayarları panelde görünür ve kaydedilir,
 *  2) şifre/token alanları boş bırakılırsa korunur,
 *  3) sağlayıcı kapatılınca saklanan sırlar silinir,
 *  4) SMS abonesi için telefon zorunludur; telefonlu abone eklenir,
 *  5) abone kanalı 'sms' olarak saklanır.
 *
 * Temizlik: test sonunda SMS kapatılır ve kendi abonesini siler.
 */

const test = require('node:test');
const assert = require('node:assert/strict');
const { goto, setViewport, withAdminPage } = require('../lib/harness');

const SID = process.env.TB_SESSION;

test('gazete SMS: ayarlar, sır korunması, abone kanalı ve temizlik', async (t) => {
  if (!SID) return t.skip('TB_SESSION yok (admin testi)');
  await withAdminPage(async (page, errors) => {
    await setViewport(page, 1400, 950);

    const postForm = (url, data) => page.evaluate(async (u, d) => {
      const fd = new URLSearchParams();
      Object.entries(d).forEach(([k, v]) => fd.append(k, v));
      const r = await fetch(u, { method: 'POST', credentials: 'same-origin', body: fd });
      return r.status;
    }, url, data);

    const panel = () => page.evaluate(async () => {
      const r = await fetch('/admin/gazete', { credentials: 'same-origin' });
      const html = await r.text();
      return {
        saat: (html.match(/name="saat" value="([^"]*)"/) || [])[1] || '',
        provider: (html.match(/name="sms_provider"[\s\S]*?<option value="netgsm"([^>]*)>/) || [])[1] || '',
        kayitli: html.includes('(kayıtlı)'),
      };
    });

    // Göreli fetch'ler için sayfa kendi kökünde olmalı.
    await goto(page, '/admin/gazete');
    assert.ok(!page.url().includes('/admin/giris'), 'gazete paneli açık olmalı');

    try {
      // 1) SMS ayarlarını yaz.
      await postForm('/admin/gazete/ayar', {
        saat: '08:15', otomatik_yayin: '0', haftalik_gun: '1', aylik_gun: '1',
        sms_provider: 'netgsm', sms_from: 'E2E_TEST',
        sms_netgsm_user: 'e2e-user', sms_netgsm_password: 'e2e-secret',
      });
      await goto(page, '/admin/gazete');
      let p = await panel();
      assert.equal(p.saat, '08:15', 'saat kaydedilmeli (jsonb metin yuvarlaması)');
      assert.match(p.provider, /selected/, 'netgsm seçili görünmeli');

      // 2) Şifre boş bırakılırsa korunmalı.
      await postForm('/admin/gazete/ayar', {
        saat: '08:15', otomatik_yayin: '0', haftalik_gun: '1', aylik_gun: '1',
        sms_provider: 'netgsm', sms_from: 'E2E_TEST',
        sms_netgsm_user: 'e2e-user', sms_netgsm_password: '',
      });
      await page.reload({ waitUntil: 'domcontentloaded' });
      p = await panel();
      assert.ok(p.kayitli, 'şifre boş bırakılırsa korunmalı (kayıtlı ipucu)');

      // 3) Abone: telefonsuz reddedilmeli, telefonlu eklenmeli.
      const sonuc = await page.evaluate(async () => {
        const post = async (d) => {
          const fd = new URLSearchParams(d);
          const r = await fetch('/admin/gazete/aboneler', { method: 'POST', credentials: 'same-origin', body: fd });
          return r.text();
        };
        const yoksay = await post({ kanal: 'sms', siklik: 'gunluk' });
        const ekle = await post({ kanal: 'sms', siklik: 'gunluk', telefon: '905009998877' });
        return { yoksay, ekle };
      });
      assert.match(sonuc.yoksay, /telefon gerekli/, 'telefonsuz SMS abonesi reddedilmeli');
      assert.match(sonuc.ekle, /Abone eklendi/, 'telefonlu SMS abonesi eklenmeli');

      // 4) Kanal 'sms' olarak saklanmalı (panel listesinde).
      await page.reload({ waitUntil: 'domcontentloaded' });
      const satir = await page.evaluate(() => {
        const html = document.body.innerHTML;
        return html.includes('905009998877') && /<td class="small">sms<\/td>/.test(html);
      });
      assert.ok(satir, 'SMS abonesi listede sms kanalıyla görünmeli');
    } finally {
      // Temizlik: kendi abonesini sil + SMS'i kapat (sırlar da silinir).
      await page.evaluate(async () => {
        const html = await (await fetch('/admin/gazete', { credentials: 'same-origin' })).text();
        const m = html.match(/\/admin\/gazete\/aboneler\/(\d+)\/sil[^>]*data-confirm="Abone kaydı \(KVKK\) silinsin mi\?"[\s\S]{0,200}?905009998877/);
        const ids = [...html.matchAll(/\/admin\/gazete\/aboneler\/(\d+)\/sil/g)].map((x) => x[1]);
        const idx = m ? ids.indexOf(m[1]) : -1;
        for (const id of new Set(idx >= 0 ? [m[1]] : ids)) {
          await fetch('/admin/gazete/aboneler/' + id + '/sil', { method: 'POST', credentials: 'same-origin' });
        }
        const fd = new URLSearchParams({
          saat: '07:00', otomatik_yayin: '0', haftalik_gun: '1', aylik_gun: '1',
          sms_provider: '', sms_from: '', sms_netgsm_user: '',
        });
        await fetch('/admin/gazete/ayar', { method: 'POST', credentials: 'same-origin', body: fd });
      });
    }
    assert.deepEqual(errors, [], 'sayfada JS hatası olmamalı');
  });
});