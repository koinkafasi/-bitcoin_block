/* media.js — medya kütüphanesi paneli etkileşimleri (templ'den ayrıldı).
 * Düzenleme modalı, dönüşümler, URL kopyalama, sürükle-bırak yükleme,
 * ızgara/liste görünümü. HTMX swap sonrası otomatik yeniden bağlanır. */
(function () {
  'use strict';

  function init() {
    const panel = document.getElementById('medya-panel');
    if (!panel || panel.dataset.ready === '1') return;
    panel.dataset.ready = '1';
    const modal = document.getElementById('mediaEditModal');
    const form = document.getElementById('mediaEditForm');
    const preview = document.getElementById('mediaEditPreview');
    const facts = document.getElementById('mediaEditFacts');
    const close = () => { modal.hidden = true; };
    const open = (item) => {
      const d = item.dataset;
      document.getElementById('mediaEditFilename').value = d.filename || '';
      document.getElementById('mediaEditName').value = d.title || '';
      document.getElementById('mediaEditCategory').value = d.category || 'genel';
      document.getElementById('mediaEditAlt').value = d.alt || '';
      document.getElementById('mediaEditCaption').value = d.caption || '';
      document.getElementById('mediaEditDescription').value = d.description || '';
      document.getElementById('mediaEditSEOTitle').value = d.seoTitle || '';
      document.getElementById('mediaEditSEODesc').value = d.seoDescription || '';
      preview.src = d.url || d.thumb || '';
      preview.alt = d.alt || d.title || '';
      facts.textContent = [d.dimensions, d.size, d.created].filter(Boolean).join(' · ');
      form.action = '/admin/dosyalar/' + d.mediaId + '/meta';
      modal.dataset.mediaId = d.mediaId;
      modal.hidden = false;
      document.getElementById('mediaEditName').focus();
    };
    panel.addEventListener('click', (event) => {
      const transformBtn = event.target.closest('[data-media-transform]');
      if (transformBtn) {
        event.preventDefault();
        const mediaId = modal.dataset.mediaId;
        if (!mediaId || !window.confirm('Görsel kalıcı olarak döndürülecek/çevrilecek. Devam?')) return;
        const fd = new FormData();
        fd.append('islem', transformBtn.dataset.mediaTransform);
        fetch('/admin/dosyalar/' + mediaId + '/duzenle', {
          method: 'POST', body: fd, credentials: 'same-origin',
        }).then((res) => {
          if (!res.ok) throw new Error('düzenleme başarısız');
          location.reload();
        }).catch(() => {
          transformBtn.textContent = 'Hata';
          setTimeout(() => { location.reload(); }, 900);
        });
        return;
      }
      const copyBtn = event.target.closest('[data-copy-url]');
      if (copyBtn) {
        event.preventDefault();
        const absolute = new URL(copyBtn.dataset.copyUrl || '', location.href).href;
        const old = copyBtn.textContent;
        const done = () => { copyBtn.textContent = '✓'; setTimeout(() => { copyBtn.textContent = old; }, 1200); };
        if (navigator.clipboard && navigator.clipboard.writeText) {
          navigator.clipboard.writeText(absolute).then(done).catch(() => {});
        }
        return;
      }
      const opener = event.target.closest('[data-media-open]');
      if (opener) {
        const item = opener.closest('.media-item');
        if (item && (event.shiftKey || event.ctrlKey || event.metaKey)) {
          const checkbox = item.querySelector('.medyaSec');
          if (checkbox) {
            checkbox.dispatchEvent(new MouseEvent('click', {
              bubbles: true, cancelable: true,
              shiftKey: event.shiftKey, ctrlKey: event.ctrlKey, metaKey: event.metaKey,
            }));
          }
          event.preventDefault();
          return;
        }
        event.preventDefault();
        if (item) open(item);
        return;
      }
      if (event.target.closest('#mediaEditClose, #mediaEditCancel')) close();
      if (event.target === modal) close();
    });
    document.addEventListener('keydown', (event) => {
      if (event.key === 'Escape' && !modal.hidden) close();
    });
    // Yıl/ay seçimi değişince filtre kendiliğinden uygulanır.
    panel.querySelectorAll('[data-otomatik]').forEach((el) => {
      el.addEventListener('change', () => {
        const form = el.form;
        if (!form) return;
        if (form.requestSubmit) form.requestSubmit(); else form.submit();
      });
    });
    /* ---- Faz 186: kart galerisi ---- */
    const durum = { surukle: 0 };
    const hareketAz = window.matchMedia('(prefers-reduced-motion: reduce)').matches;

    // Masonry: her karta satır sayısı yaz. Referansın yöntemi: ızgara
    // satır yüksekliği 4px, kart yüksekliği / 4 = span.
    const ROW = 4;
    const GAP = 14;

    /* Oran sınırları.
     *
     * Bir kart sütun genişliğinin en çok 2 katı yüksek (ORAN_MIN = .5),
     * en çok 2.5 katı geniş (ORAN_MAX) olabilir.
     *
     * Sınır neden gerekli: bir medyanın ÖZGÜN ölçüsü 750×15873 (uzun
     * şerit ekran görüntüsü), thumb'ı ise 422×422. Sınırsız oranla kart
     * 215×4554 px oluyordu; ızgarada tek başına bir sütun boyu boşluk
     * (ölçüldü: span 1142). */
    const ORAN_MIN = 0.5;
    const ORAN_MAX = 2.5;
    function sinirla(r) {
      if (!isFinite(r) || r <= 0) return 1;
      return Math.min(ORAN_MAX, Math.max(ORAN_MIN, r));
    }

    function oran(el) {
      // 1) Ekranda duran GÖRSELİN gerçek oranı en doğru kaynak: thumb
      //    özgün dosyadan farklı oranda üretilebiliyor.
      const img = el.querySelector('img');
      if (img && img.naturalWidth > 0 && img.naturalHeight > 0) {
        return sinirla(img.naturalWidth / img.naturalHeight);
      }
      // 2) Önceki ölçümden kalan oran (görsel henüz yüklenmemişse).
      const ar = Number(el.dataset.ar);
      if (ar > 0) return sinirla(ar);
      // 3) Sunucunun verdiği özgün ölçü.
      const ham = el.dataset.dimensions || '';
      const m = ham.match(/(\d+)\s*[x×]\s*(\d+)/);
      if (m) {
        const g = Number(m[1]);
        const y = Number(m[2]);
        if (g > 0 && y > 0) return sinirla(g / y);
      }
      return 1; // kare varsayılan (ses/video/belge)
    }
    function masonry() {
      const liste = panel.classList.contains('media-list-view');
      panel.querySelectorAll('.media-grid').forEach((grid) => {
        const ogeler = grid.querySelectorAll('.media-item, .mgal-drop, .mgal-up');
        if (liste) {
          ogeler.forEach((el) => { el.style.gridRowEnd = ''; });
          return;
        }
        const ilk = grid.querySelector('.media-item, .mgal-drop');
        if (!ilk) return;
        const cw = ilk.getBoundingClientRect().width;
        if (!cw) return;
        grid.querySelectorAll('.media-item').forEach((el) => {
          const h = cw / oran(el) + GAP;
          el.style.gridRowEnd = 'span ' + Math.max(1, Math.round(h / ROW));
        });
        grid.querySelectorAll('.mgal-drop').forEach((el) => {
          const h = Math.max(176, cw) + GAP;
          el.style.gridRowEnd = 'span ' + Math.max(1, Math.round(h / ROW));
        });
        grid.querySelectorAll('.mgal-up').forEach((el) => {
          const h = cw + GAP;
          el.style.gridRowEnd = 'span ' + Math.max(1, Math.round(h / ROW));
        });
      });
    }
    masonry();
    let boyutZaman = null;
    window.addEventListener('resize', () => {
      clearTimeout(boyutZaman);
      boyutZaman = setTimeout(masonry, 120);
    });

    /* Görsel yüklenince gerçek oranı karta yaz ve ızgarayı tazele.
     *
     * Izgara ilk kurulduğunda görseller henüz yüklenmemiş olabiliyor; o
     * anda oran sunucunun verdiği özgün ölçüden geliyor ki bu thumb'la
     * uyuşmayabiliyor. 'load' olayı kabarcıklanmadığı için yakalama
     * fazında dinleniyor (aynı 'error' gibi). */
    let oranZaman = null;
    const oranTazele = () => {
      clearTimeout(oranZaman);
      oranZaman = setTimeout(masonry, 60);
    };
    panel.addEventListener('load', (ev) => {
      const img = ev.target;
      if (!img || img.tagName !== 'IMG') return;
      const item = img.closest('.media-item');
      if (!item) return;
      if (img.naturalWidth > 0 && img.naturalHeight > 0) {
        item.dataset.ar = String(img.naturalWidth / img.naturalHeight);
      }
      oranTazele();
    }, true);

    /* Kırık görsel yedeği.
     * Thumb üretilemeyen/404 olan medyada tarayıcı kırık ikon + alt metni
     * basıyor; alt metin kart başlığının üstüne biniyordu (ölçüldü: izole
     * ortamda 60 kartın 8'i). Kart yerine nötr bir yer tutucu gösterilir.
     * 'error' olayı kabarcıklanmadığı için yakalama fazında dinleniyor. */
    panel.addEventListener('error', (ev) => {
      const img = ev.target;
      if (!img || img.tagName !== 'IMG') return;
      const item = img.closest('.media-item');
      if (!item) return;
      item.classList.add('mi-kirik');
      img.style.visibility = 'hidden';
    }, true);

    // Seçim durumu: kart çerçevesi ve daire görünürlüğü.
    const secimGuncelle = () => {
      const secili = panel.querySelectorAll('.medyaSec:checked').length;
      panel.classList.toggle('is-selecting', secili > 0);
      panel.querySelectorAll('.media-item').forEach((item) => {
        const k = item.querySelector('.medyaSec');
        item.classList.toggle('is-sel', !!(k && k.checked));
      });
    };
    panel.addEventListener('change', (ev) => {
      if (ev.target && ev.target.classList && ev.target.classList.contains('medyaSec')) secimGuncelle();
    });
    secimGuncelle();

    /* ---- Sürükle-bırak: galerinin her yeri hedef ---- */
    const dropInput = panel.querySelector('[data-mg-file]');
    const browseBtn = panel.querySelector('[data-mg-browse]');
    if (browseBtn && dropInput) {
      browseBtn.addEventListener('click', () => dropInput.click());
      dropInput.addEventListener('change', () => {
        const dosyalar = dropInput.files ? [...dropInput.files] : [];
        dropInput.value = '';
        if (dosyalar.length) yukle(dosyalar);
      });
    }
    const dosyaVar = (ev) => ev.dataTransfer && [...(ev.dataTransfer.types || [])].includes('Files');
    panel.addEventListener('dragenter', (ev) => {
      if (!dosyaVar(ev)) return;
      ev.preventDefault();
      durum.surukle += 1;
      panel.classList.add('is-drag');
    });
    panel.addEventListener('dragover', (ev) => {
      if (!dosyaVar(ev)) return;
      ev.preventDefault();
      if (ev.dataTransfer) ev.dataTransfer.dropEffect = 'copy';
    });
    panel.addEventListener('dragleave', () => {
      durum.surukle = Math.max(0, durum.surukle - 1);
      if (!durum.surukle) panel.classList.remove('is-drag');
    });
    panel.addEventListener('drop', (ev) => {
      if (!ev.dataTransfer) return;
      const dosyalar = [...ev.dataTransfer.files];
      if (!dosyalar.length) return;
      ev.preventDefault();
      durum.surukle = 0;
      panel.classList.remove('is-drag');
      yukle(dosyalar);
    });

    /* ---- Yükleme: halka + yüzde + dosya adı ---- */
    const CEVRE = 2 * Math.PI * 22; // referansla aynı: r=22
    function yuklemeKarti(ad) {
      const el = document.createElement('figure');
      el.className = 'mgal-up';
      el.innerHTML =
        '<div class="mgal-up__card">' +
        '<svg class="mgal-up__ring" viewBox="0 0 52 52" aria-hidden="true">' +
        '<circle cx="26" cy="26" r="22" class="mgal-up__track"></circle>' +
        '<circle cx="26" cy="26" r="22" class="mgal-up__arc" stroke-dasharray="' + CEVRE + '" stroke-dashoffset="' + CEVRE + '"></circle>' +
        '</svg>' +
        '<b class="mgal-up__pct">0%</b>' +
        '<small class="mgal-up__name"></small>' +
        '<span class="mgal-sr" role="status"></span>' +
        '</div>';
      el.querySelector('.mgal-up__name').textContent = ad;
      el.__p = 0;
      return el;
    }
    function yuzdeYaz(el, p) {
      const p2 = Math.max(0, Math.min(1, p));
      el.__p = p2;
      const yay = el.querySelector('.mgal-up__arc');
      const pct = el.querySelector('.mgal-up__pct');
      const durumEl = el.querySelector('.mgal-sr');
      if (yay) yay.style.strokeDashoffset = String(CEVRE * (1 - p2));
      if (pct) pct.textContent = Math.round(p2 * 100) + '%';
      if (durumEl) durumEl.textContent = Math.round(p2 * 100) + '% yüklendi';
    }
    // Kısa bir tırmanış: 0 → .3 (referansta .5s), sonra gerçek ilerleme.
    function tirman(el, hedef, ms) {
      if (hareketAz) { yuzdeYaz(el, hedef); return; }
      const bas = el.__p;
      const t0 = performance.now();
      (function adim(t) {
        const k = Math.min(1, (t - t0) / ms);
        yuzdeYaz(el, bas + (hedef - bas) * k);
        if (k < 1) requestAnimationFrame(adim);
      })(t0);
    }
    function panelYenile() {
      if (window.htmx && window.htmx.ajax) {
        window.htmx.ajax('GET', location.pathname + location.search, {
          target: '#medya-panel', swap: 'outerHTML', indicator: '#medya-yukleniyor',
        });
      } else {
        location.reload();
      }
    }
    function yukle(dosyalar) {
      const grid = panel.querySelector('.media-grid');
      if (!grid) return;
      // Faz 187: üstteki yükleme formu kaldırıldı. Başlık/kategori/alt metin
      // artık burada sorulmuyor; sunucu dosya adından türetiyor, alanlar
      // resim detayında düzenleniyor.
      const fd0 = new FormData();
      const kartlar = dosyalar.map((d) => {
        const el = yuklemeKarti(d.name);
        grid.insertBefore(el, grid.querySelector('.mgal-drop') ? grid.querySelector('.mgal-drop').nextSibling : grid.firstChild);
        return el;
      });
      masonry();
      let hata = 0;
      let bitti = 0;
      const sirayla = (i) => {
        if (i >= dosyalar.length) {
          if (hata) {
            const uyari = document.createElement('p');
            uyari.className = 'small';
            uyari.style.margin = '8px 0 0';
            uyari.textContent = hata + ' dosya yüklenemedi.';
            panel.appendChild(uyari);
          }
          if (bitti > hata) setTimeout(panelYenile, 320);
          return;
        }
        const dosya = dosyalar[i];
        const kart = kartlar[i];
        tirman(kart, 0.3, 500);
        const fd = new FormData();
        fd0.forEach((v, k) => fd.append(k, v));
        fd.append('dosya', dosya, dosya.name);
        const xhr = new XMLHttpRequest();
        xhr.open('POST', '/admin/dosyalar?json=1', true);
        xhr.withCredentials = true;
        xhr.responseType = 'text';
        xhr.upload.onprogress = (ev) => {
          if (!ev.lengthComputable) return;
          // Gerçek ilerleme 0.3 → 0.95 aralığına ölçeklenir.
          const p = 0.3 + 0.65 * (ev.loaded / ev.total);
          if (p > kart.__p) yuzdeYaz(kart, p);
        };
        xhr.onload = () => {
          let ok = false;
          try { ok = xhr.status >= 200 && xhr.status < 300 && !!JSON.parse(xhr.responseText).ok; } catch (_) { ok = false; }
          if (ok) {
            bitti += 1;
            tirman(kart, 1, 300);
            setTimeout(() => { kart.style.opacity = '0'; kart.style.transition = 'opacity .25s'; }, 380);
          } else {
            hata += 1;
            kart.classList.add('is-hata');
            const o = kart.querySelector('.mgal-up__pct');
            if (o) o.textContent = 'Hata';
          }
        };
        xhr.onerror = () => {
          hata += 1;
          kart.classList.add('is-hata');
          const o = kart.querySelector('.mgal-up__pct');
          if (o) o.textContent = 'Hata';
        };
        const son = () => setTimeout(() => sirayla(i + 1), 260);
        xhr.onloadend = son;
        xhr.send(fd);
      };
      sirayla(0);
    }

    /* Görünüm anahtarı EN SONDA: applyView masonry() çağırıyor ve masonry
     * yukarıdaki ROW/GAP sabitlerini kullanıyor. Bu blok daha önce
     * çalıştığında "Cannot access 'GAP' before initialization" hatası
     * veriyordu (TDZ): init yarıda kalıyor, ne ızgara kuruluyor ne de
     * görünüm düğmeleri bağlanıyordu. */
    const viewButtons = panel.querySelectorAll('[data-media-view]');
    const applyView = (view) => {
      panel.classList.toggle('media-list-view', view === 'list');
      viewButtons.forEach((btn) => btn.classList.toggle('is-active', btn.dataset.mediaView === view));
      // Görünüm değişince masonry span'ları tazelenir: liste görünümünde
      // span kalırsa kart 800px yüksekliğe uzuyordu.
      if (typeof masonry === 'function') masonry();
    };
    /* ---- Sonsuz kaydırma sonrası bakım (Faz 187) ----
     *
     * Sunucu sayfaları yıl/ay gruplarına böler. Sayfa sınırı bir ayın
     * ortasına düşebiliyor; o zaman aynı ay etiketi iki bölüm olarak
     * geliyor. Burada ikinci bölümün kartları birinciye taşınır, bölüm
     * silinir ve sayaç yeniden yazılır — kullanıcı tek grup görür.
     *
     * Ayrıca yeni kartlar için masonry span'ları ve seçim durumu
     * tazelenir. */
    const gruplariBirlestir = () => {
      const liste = document.getElementById('medyaListe');
      if (!liste) return;
      const gorulen = new Map();
      liste.querySelectorAll('.media-group').forEach((bolum) => {
        const anahtar = bolum.dataset.grup || '';
        const onceki = gorulen.get(anahtar);
        if (onceki && onceki !== bolum) {
          const hedef = onceki.querySelector('.media-grid');
          const kaynak = bolum.querySelector('.media-grid');
          if (hedef && kaynak) {
            [...kaynak.querySelectorAll('.media-item')].forEach((k) => hedef.appendChild(k));
          }
          bolum.remove();
        } else {
          gorulen.set(anahtar, bolum);
        }
      });
      gorulen.forEach((bolum) => {
        const sayi = bolum.querySelectorAll('.media-item').length;
        const etiket = bolum.querySelector('h3 span');
        if (etiket) etiket.textContent = String(sayi);
      });
    };

    const listeTazele = () => {
      gruplariBirlestir();
      masonry();
      secimGuncelle();
    };
    // afterSwap dinleyicisi panel dışından çağırır.
    window.__tbMediaTazele = listeTazele;

    let savedView = 'grid';
    try { savedView = localStorage.getItem('tb-media-view') || 'grid'; } catch (_) { /* yoksay */ }
    applyView(savedView);
    viewButtons.forEach((btn) => {
      btn.onclick = () => {
        const view = btn.dataset.mediaView === 'list' ? 'list' : 'grid';
        applyView(view);
        try { localStorage.setItem('tb-media-view', view); } catch (_) { /* yoksay */ }
      };
    });
  }

  window.__tbMediaInit = init;
  if (document.readyState !== 'loading') init();
  else document.addEventListener('DOMContentLoaded', init);
  // HTMX panel değişimlerinde yeniden bağla.
  // Sonsuz kaydırma dilimi paneli DEĞİŞTİRMEZ (gözcünün yerine geçer), bu
  // yüzden init() erken döner; dilim sonrası bakım ayrı çağrılır.
  document.addEventListener('htmx:afterSwap', function (ev) {
    if (!document.getElementById('medya-panel')) return;
    // Dilim isteği mi? Yanıt adresindeki `parca=1` bunu kesin söyler;
    // outerHTML sonrası detail.target ayrılmış (detached) düğüm olabiliyor.
    const xhr = ev.detail && ev.detail.xhr;
    const adres = (xhr && xhr.responseURL) || '';
    const dilim = adres.indexOf('parca=1') !== -1;
    init();
    if (dilim && typeof window.__tbMediaTazele === 'function') {
      window.__tbMediaTazele();
    }
  });
})();