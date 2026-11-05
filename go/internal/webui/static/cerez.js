/* Çerez onayı ve analitiğin rıza kapısı.
 *
 * NEDEN BU KADAR SIKI: Kişisel Verileri Koruma Kurulu'nun 08.08.2024 tarihli
 * 2024/1361 sayılı kararı, kesinlikle gerekli olmayan bir analitik çerez için
 * açık rıza arandığını ve sitenin İLK GİRİŞİNDE bu rızayı alacak bir
 * mekanizma bulunmamasını ihlal saydı. Kurulun Çerez Rehberi de işlevsel,
 * performans-analitik ve reklam çerezleri için varsayılanın "çalışmıyor"
 * olduğu bir opt-in mekanizması istiyor.
 *
 * Google Consent Mode yeterli DEĞİL: o yöntemde gtag.js yine de yüklenir ve
 * tarayıcı Google'a (ABD) istek atar; bu, çerez yazılmasa bile ziyaretçinin
 * IP adresinin aktarılması demektir (6698 s.K. m.9). Bu yüzden burada tam
 * erteleme uygulanır — rıza gelmeden gtag.js sayfaya hiç eklenmez.
 *
 * Reddetmek kabul etmek kadar kolay olmalı: iki düğme aynı görünürlükte ve
 * aynı tıklama sayısında. Rıza hizmetin ön şartı değildir; reddeden kullanıcı
 * siteyi aynen kullanır.
 */
(function () {
  'use strict';

  var COOKIE = 'tb_cerez';
  // Sürüm: çerez politikası veya izlenen araç listesi değişirse artır;
  // eski kararlar geçersizleşir ve rıza yeniden sorulur.
  var SURUM = '1';
  var GUN = 180;

  function cerezOku(ad) {
    try {
      var p = document.cookie.split(';');
      for (var i = 0; i < p.length; i++) {
        var s = p[i].trim();
        if (s.indexOf(ad + '=') === 0) return decodeURIComponent(s.substring(ad.length + 1));
      }
    } catch (e) {}
    return '';
  }

  function cerezYaz(ad, deger, gun) {
    try {
      var d = new Date();
      d.setTime(d.getTime() + gun * 24 * 60 * 60 * 1000);
      var guvenli = location.protocol === 'https:' ? '; Secure' : '';
      document.cookie = ad + '=' + encodeURIComponent(deger) +
        '; expires=' + d.toUTCString() + '; path=/; SameSite=Lax' + guvenli;
    } catch (e) {}
  }

  // Karar biçimi: "<surum>:<kabul|ret>:<ISO tarih>" — ispat yükü veri
  // sorumlusunda olduğu için hangi sürüme, ne zaman karar verildiği saklanır.
  function kararOku() {
    var ham = cerezOku(COOKIE);
    if (!ham) return null;
    var p = ham.split(':');
    if (p[0] !== SURUM) return null; // sürüm değişti, yeniden sor
    if (p[1] !== 'kabul' && p[1] !== 'ret') return null;
    return { karar: p[1], tarih: p[2] || '' };
  }

  function kararYaz(karar) {
    cerezYaz(COOKIE, SURUM + ':' + karar + ':' + new Date().toISOString(), GUN);
  }

  // --- Analitik ---------------------------------------------------------

  var olcumKimligi = '';
  var yuklendi = false;

  function analitigiBaslat() {
    if (yuklendi || !olcumKimligi) return;
    yuklendi = true;
    var s = document.createElement('script');
    s.async = true;
    s.src = 'https://www.googletagmanager.com/gtag/js?id=' + encodeURIComponent(olcumKimligi);
    document.head.appendChild(s);
    window.dataLayer = window.dataLayer || [];
    window.gtag = function () { window.dataLayer.push(arguments); };
    window.gtag('js', new Date());
    // IP anonimleştirme: GA4'te varsayılan olsa da açıkça bildiriyoruz.
    window.gtag('config', olcumKimligi, { anonymize_ip: true });
  }

  // Reddedildiğinde daha önce yazılmış Google çerezlerini temizle. Daha önce
  // üçüncü tarafa İLETİLMİŞ veriyi geri alamayız; burada yalnız bu tarayıcıda
  // kalan tanımlayıcılar silinir.
  function analitikCerezleriSil() {
    try {
      var hepsi = document.cookie.split(';');
      for (var i = 0; i < hepsi.length; i++) {
        var ad = hepsi[i].split('=')[0].trim();
        if (ad.indexOf('_ga') !== 0 && ad.indexOf('_gid') !== 0 && ad.indexOf('_gat') !== 0) continue;
        // Çerez ancak aynı ad + alan adı + yol üçlüsüyle silinebilir; hangi
        // nitelikle yazıldığını JavaScript okuyamadığı için olası
        // kombinasyonlar denenir. Alan adı niteliği IP adresli hostlarda
        // geçersizdir, o durumda yalnız host-only deneme işe yarar.
        var kokler = ['', '.' + location.hostname, location.hostname];
        var yollar = ['/', location.pathname];
        for (var k = 0; k < kokler.length; k++) {
          for (var y = 0; y < yollar.length; y++) {
            document.cookie = ad + '=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=' +
              yollar[y] + (kokler[k] ? '; domain=' + kokler[k] : '');
          }
        }
      }
    } catch (e) {}
  }


  // --- Stil ---------------------------------------------------------------
  //
  // Stiller `static/cerez.css` içinde. Önceden buradan bir `<style>` elementi
  // enjekte ediliyordu; CSP `style-src`ten 'unsafe-inline' kaldırıldığında
  // (Faz 211) JS'in oluşturduğu `<style>` de engellenir ve bant stilsiz
  // kalırdı. Statik dosya CSP'ye uygun, önbelleklenir; FOUC yok çünkü bant
  // zaten yalnız JS çalışınca görünür.

  // --- Bant -------------------------------------------------------------

  var bant = null;

  function bandiKapat() {
    if (bant && bant.parentNode) bant.parentNode.removeChild(bant);
    bant = null;
  }

  function kararVer(karar) {
    kararYaz(karar);
    bandiKapat();
    if (karar === 'kabul') {
      analitigiBaslat();
    } else {
      analitikCerezleriSil();
    }
  }

  function bandiGoster() {
    if (bant) return;
    bant = document.createElement('div');
    bant.className = 'cerez-bant';
    bant.setAttribute('role', 'dialog');
    bant.setAttribute('aria-live', 'polite');
    bant.setAttribute('aria-label', 'Çerez tercihi');

    var metin = document.createElement('div');
    metin.className = 'cerez-bant-metin';
    metin.innerHTML =
      '<strong>Çerezler</strong><p>Sitenin çalışması için gereken çerezleri ' +
      'kullanıyoruz. Ziyaret istatistiklerini ölçmek için ayrıca Google ' +
      'Analytics kullanmak istiyoruz; bu, verinizin yurt dışına aktarılması ' +
      'anlamına gelir ve yalnızca siz onay verirseniz çalışır. Reddederseniz ' +
      'site aynen çalışmaya devam eder. ' +
      '<a href="/cerez-politikasi">Çerez Politikası</a></p>';

    var dugmeler = document.createElement('div');
    dugmeler.className = 'cerez-bant-dugmeler';

    var ret = document.createElement('button');
    ret.type = 'button';
    ret.className = 'cerez-btn cerez-btn-ret';
    ret.textContent = 'Reddet';
    ret.addEventListener('click', function () { kararVer('ret'); });

    var kabul = document.createElement('button');
    kabul.type = 'button';
    kabul.className = 'cerez-btn cerez-btn-kabul';
    kabul.textContent = 'Kabul et';
    kabul.addEventListener('click', function () { kararVer('kabul'); });

    // Ret önce gelir ve kabul ile aynı boyuttadır: rıza özgür iradeyle
    // verilmiş sayılabilsin diye reddetmek zorlaştırılmaz.
    dugmeler.appendChild(ret);
    dugmeler.appendChild(kabul);
    bant.appendChild(metin);
    bant.appendChild(dugmeler);
    document.body.appendChild(bant);
    ret.focus();
  }

  // --- Giriş noktası ----------------------------------------------------

  function basla() {
    var kok = document.querySelector('[data-ga4]');
    olcumKimligi = kok ? (kok.getAttribute('data-ga4') || '') : '';

    // Tercihi sonradan değiştirme: KVKK rızanın geri alınmasını vermek kadar
    // kolay kılmayı gerektirir. Footer'daki bağlantı bu kancayı kullanır.
    document.addEventListener('click', function (e) {
      var t = e.target.closest ? e.target.closest('[data-cerez-ayar]') : null;
      if (!t) return;
      e.preventDefault();
      bandiGoster();
    });

    var k = kararOku();
    if (!k) {
      // Ölçüm kimliği tanımlı değilse sorulacak bir şey yok: site zaten
      // hiçbir üçüncü taraf kodu yüklemiyor.
      if (olcumKimligi) bandiGoster();
      return;
    }
    if (k.karar === 'kabul') {
      analitigiBaslat();
      return;
    }
    // Karar 'ret' ise temizliği HER sayfa yüklemesinde tekrarla. Tek seferlik
    // silmek yetmiyor: farklı yol/alan adı niteliğiyle yazılmış bir kopya
    // document.cookie üzerinden silinemeyip sonraki yüklemede geri
    // görünebiliyor (yerel testte _ga_<id> ile bu gözlendi). Burada her
    // açılışta süpürmek, reddi kalıcı hâle getirir.
    analitikCerezleriSil();
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', basla);
  } else {
    basla();
  }
})();
