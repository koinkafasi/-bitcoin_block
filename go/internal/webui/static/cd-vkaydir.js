/* cd-vkaydir.js — video sayfası SAĞ KOLON akışı.
 *
 * KULLANICI İSTEĞİ (11 Kasım 2026): "sağda scroll yok ve sağdaki video alta
 * yukarı doğru bir hafif transparan geçiş var tasarımda ve sağdaki içerikler
 * mouse ile alta doğru çok profesyonel bir akış alıyor" (CoinDesk referansı).
 *
 * ÜÇ PARÇA:
 *   1) Kaydırma çubuğu GİZLİ          → CSS (scrollbar-width: none)
 *   2) Üst/altta saydam geçiş (maske) → CSS (mask-image gradyanı)
 *   3) Fare tekerleğiyle YUMUŞAK akış → bu dosya
 *
 * NEDEN JS: tarayıcının varsayılan tekerlek kaydırması "sıçrar" (tek tıkla
 * ~100 px, animasyon yok). CSS `scroll-behavior: smooth` yalnız PROGRAMATİK
 * kaydırmayı yumuşatır, tekerleği DEĞİL — tekerlek sıçramaya devam ederdi.
 *
 * ZAMANA BAĞLI YUMUŞATMA (ölçüm dersi): ilk sürüm kare başına sabit oranla
 * (`fark * 0.14`) yaklaşıyordu. Kare hızı düştüğünde (arka plan sekmesi,
 * güç tasarrufu) akış BİTİYORDU — ölçümde 15 karede 1 px ilerledi. Artık
 * ilerleme GEÇEN ZAMANA göre hesaplanıyor: 60 fps'te de 10 fps'te de aynı
 * sürede aynı mesafe. Kare hızı yalnız akıcılığı etkiler, hızı değil.
 *
 * DOĞRU DAVRANIŞ (aksini yapmak rahatsız edici olurdu):
 *   • Liste başındayken YUKARI, sonundayken AŞAĞI tekerlek → olay
 *     yakalanmaz, SAYFA kayar (kullanıcı listede mahsur kalmaz).
 *   • Kaydırma sırasında sayfa kaymaz (preventDefault) — çift hareket yok.
 *   • Dokunmatik cihazda DEVRE DIŞI: tarayıcının ataletli kaydırması daha
 *     iyi; çift davranış olmasın.
 *   • `prefers-reduced-motion: reduce` → yumuşatma YOK, doğrudan atlar
 *     (hareket hassasiyeti olan kullanıcı yavaş akış istemez).
 */
(function () {
  'use strict';

  var SURE = 260;          // hedefe varış süresi (ms) — his: "akıyor"
  var ESIK = 0.5;          // px; bu kadar yaklaşınca dur

  function azaltHareket() {
    return window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches;
  }

  function bagla(el) {
    if (el.dataset.vkaydir === '1') return;
    el.dataset.vkaydir = '1';

    var hedef = el.scrollTop;
    var baslangic = el.scrollTop;
    var basZaman = 0;
    var calisiyor = false;

    function sinirla(v) {
      if (v < 0) return 0;
      var enFazla = el.scrollHeight - el.clientHeight;
      return v > enFazla ? enFazla : v;
    }

    function kare(simdi) {
      if (!basZaman) basZaman = simdi;
      var gecen = simdi - basZaman;
      var oran = gecen / SURE;
      if (oran >= 1) oran = 1;

      // ease-out cubic: hızlı başlar, yumuşak oturur (doğal akış hissi).
      var e = 1 - Math.pow(1 - oran, 3);
      el.scrollTop = baslangic + (hedef - baslangic) * e;

      if (oran < 1 && Math.abs(hedef - el.scrollTop) > ESIK) {
        requestAnimationFrame(kare);
      } else {
        el.scrollTop = hedef;
        calisiyor = false;
      }
    }

    // GÜVENLİK AĞI (ölçüm dersi): requestAnimationFrame bazı ortamlarda
    // (arka plan sekmesi, güç tasarrufu, kısıtlı headless) neredeyse hiç
    // çalışmıyor. Ölçümde 260 px'lik hedefte 47 px'de takılı kaldı.
    // Zamanlayıcı, süre dolduğunda hedefi ZORLA uygular: kullanıcı yarım
    // kalmış bir kaydırma görmez, en kötü ihtimalle animasyon "atlar".
    function emniyet() {
      if (!calisiyor) return;
      if (Math.abs(hedef - el.scrollTop) > ESIK) {
        el.scrollTop = hedef;
      }
      calisiyor = false;
    }

    function git(yeniHedef) {
      yeniHedef = sinirla(yeniHedef);
      if (azaltHareket()) {
        el.scrollTop = yeniHedef;
        hedef = yeniHedef;
        return;
      }
      // Zaten akış varsa yeni hedefe yumuşakça devam et (sıçrama yok).
      baslangic = el.scrollTop;
      basZaman = 0;
      hedef = yeniHedef;
      if (!calisiyor) {
        calisiyor = true;
        requestAnimationFrame(kare);
        setTimeout(emniyet, SURE + 120);
      }
    }

    el.addEventListener('wheel', function (e) {
      if (e.ctrlKey) return; // tarayıcı yakınlaştırması: karışma

      var ust = el.scrollTop <= 0;
      var alt = el.scrollTop + el.clientHeight >= el.scrollHeight - 1;

      // Uçlarda sayfaya devret: kullanıcı listede kilitli kalmasın.
      if ((ust && e.deltaY < 0) || (alt && e.deltaY > 0)) return;

      e.preventDefault();
      git(el.scrollTop + e.deltaY);
    }, { passive: false });

    // Liste dışına çıkınca hedefi eşitle (kayma birikmesin).
    el.addEventListener('mouseleave', function () {
      hedef = el.scrollTop;
    });

    // Klavye: odak listedeyken ok tuşları/PageUp/PageDown yumuşak kaysın.
    el.addEventListener('keydown', function (e) {
      var adim = 0;
      if (e.key === 'ArrowDown') adim = 80;
      else if (e.key === 'ArrowUp') adim = -80;
      else if (e.key === 'PageDown' || e.key === ' ') adim = el.clientHeight * 0.85;
      else if (e.key === 'PageUp') adim = -el.clientHeight * 0.85;
      else if (e.key === 'Home') { e.preventDefault(); git(0); return; }
      else if (e.key === 'End') {
        e.preventDefault();
        git(el.scrollHeight - el.clientHeight);
        return;
      }
      if (adim === 0) return;
      e.preventDefault();
      git(el.scrollTop + adim);
    });
  }

  function baslat() {
    var listeler = document.querySelectorAll('.cd-vband__liste');
    for (var i = 0; i < listeler.length; i++) bagla(listeler[i]);
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', baslat);
  } else {
    baslat();
  }
})();
