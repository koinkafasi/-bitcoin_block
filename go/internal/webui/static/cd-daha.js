// cd-daha.js — "Daha fazla haber": parça yükleme + otomatik yükleme.
//
// NEDEN: sayfalama eski hâliyle TAM SAYFA GEÇİŞİYDİ (`/business?sayfa=2`
// → sayfa baştan yükleniyor). CoinDesk'te (8 Kasım 2026 ölçümü) buton
// `<button>`; tıklayınca 5 haber aynı sayfaya ekleniyor (21 → 26 kart),
// URL hiç değişmiyor, sayfa yenilenmiyor.
//
// Bizde `<a href>` KORUNUR: JS olmayan tarayıcı ve arama motoru normal
// sayfa geçişi yapar (rel=next bağlantısı yaşar). JS varsa preventDefault
// ile `?parca=1` parçası çekilir, satırlar listenin sonuna eklenir,
// nav yeni href'lerle yenilenir.
//
// OTOMATİK YÜKLEME: IntersectionObserver buton 600px yakına gelince
// tetikler — kullanıcı aşağı indikçe haberler akar, ekstra tıklama
// gerekmez. Son sayfada buton kaldırılır.
//
// CSP: `script-src 'self'` → harici dosya (inline yok). Harici kütüphane
// (fetch/IntersectionObserver) yüklenmez.
(function () {
  'use strict';

  var gozlemci = null;
  var nav = null;
  var btn = null;
  var yukleniyor = false;
  var bitti = false;

  function gozlemciKur() {
    if (bitti || !btn) return;
    if (gozlemci) gozlemci.disconnect();
    // Eski tarayıcıda otomatik yükleme yok; tıklama yine çalışır.
    if (!('IntersectionObserver' in window)) return;
    gozlemci = new IntersectionObserver(
      function (giris) {
        if (giris[0] && giris[0].isIntersecting) yukle();
      },
      { rootMargin: '600px 0px' }
    );
    gozlemci.observe(btn);
  }

  function durum(metin) {
    if (btn) btn.textContent = metin;
  }

  function yukle() {
    if (yukleniyor || bitti || !btn) return;
    var href = btn.getAttribute('href');
    if (!href) return;
    yukleniyor = true;
    btn.setAttribute('aria-busy', 'true');
    durum('Yükleniyor…');
    var parcaURL = href + (href.indexOf('?') >= 0 ? '&' : '?') + 'parca=1';
    fetch(parcaURL, { credentials: 'same-origin' })
      .then(function (r) {
        if (!r.ok) throw new Error('HTTP ' + r.status);
        return r.text();
      })
      .then(function (html) {
        parcaEkle(html);
      })
      .catch(function () {
        // Ağ hatası: buton tekrar denenebilir bir bağlantı olarak kalır.
        if (btn) {
          btn.removeAttribute('aria-busy');
          durum('Tekrar dene');
        }
      })
      // `.finally` — zincirin SONUNDA reddi yakalamayan bir `.then`
      // bırakmamak için (Sonar S9383). Her iki durumda da çalışır.
      .finally(function () {
        yukleniyor = false;
      });
  }

  function parcaEkle(html) {
    var kap = document.createElement('div');
    // Parçada <script> YOK (sunucu yalnız satır+nav basar) → innerHTML
    // güvenli; yine de nav dışında kalan her şey olduğu gibi eklenir.
    kap.innerHTML = html;
    var yeniNav = kap.querySelector('nav.cd-daha');

    if (nav && nav.parentNode) {
      var parca = document.createDocumentFragment();
      while (kap.firstChild) {
        if (kap.firstChild === yeniNav) {
          kap.removeChild(yeniNav);
          continue;
        }
        parca.appendChild(kap.firstChild);
      }
      // Satırlar nav'dan ÖNCE eklenir: sıra korunur, buton hep en altta.
      nav.parentNode.insertBefore(parca, nav);
    }

    if (yeniNav && nav && nav.parentNode) {
      // Nav yenilendi: yeni sayfa href'leri (rel=next yaşar).
      nav.parentNode.replaceChild(yeniNav, nav);
      nav = yeniNav;
      btn = nav.querySelector('a.cd-daha__btn');
      if (btn) {
        btn.removeAttribute('aria-busy');
        gozlemciKur();
        return;
      }
      // Son sayfa: buton kalktı ama "← Önceki" KALIR (kullanıcı eklenen
      // içeriğin üstündeki bağlantıyı kaybederse geri dönüşsüz olur).
      bitti = true;
      if (gozlemci) gozlemci.disconnect();
      return;
    }

    // Parça nav içermiyorsa (beklenmeyen): nav'u tümüyle kaldır.
    bitti = true;
    if (gozlemci) gozlemci.disconnect();
    if (nav && nav.parentNode) nav.parentNode.removeChild(nav);
    nav = null;
    btn = null;
  }

  function baslat() {
    nav = document.querySelector('nav.cd-daha');
    if (!nav) return;
    btn = nav.querySelector('a.cd-daha__btn');
    if (!btn) return;
    gozlemciKur();
  }

  // Tıklama: modifier tuşlu (yeni sekme vb.) bırakılır — o hâlde normal
  // bağlantı çalışır. Düz tıklama parça yükler.
  document.addEventListener('click', function (e) {
    if (e.defaultPrevented || e.button !== 0 ||
        e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return;
    var hedef = e.target;
    if (!hedef || !hedef.closest) return;
    var a = hedef.closest('a.cd-daha__btn');
    if (!a || a !== btn) return;
    e.preventDefault();
    yukle();
  });

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', baslat);
  } else {
    baslat();
  }
})();
