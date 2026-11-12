// =====================================================================
// Yapıştırma temizliği ve Türkçe tipografi düzeltmesi
// =====================================================================
// SORUN: başka bir siteden kopyalanan metin editöre olduğu gibi düşüyordu.
// Kaynak sayfanın <div> yığınları, boş <span>'leri, "Paylaş / İlgili
// Haberler / Reklam" blokları ve bozuk tipografisi (düz tırnak, üç ayrı
// nokta, çift boşluk, kelimeye yapışık kesme işareti) içerik gövdesine
// giriyordu. Eski temizleyici yalnız script/style siliyor, geri kalanı
// olduğu gibi bırakıyordu.
//
// BURADA YAPILAN ÜÇ AŞAMA:
//   1. Yapı:      istenmeyen düğümleri at, kalanı <p>/<h2>/<ul>'a indir
//   2. Tipografi: tırnak, çizgi, üç nokta, boşluk, kesme işareti
//   3. Yazım:     Türkçede sık yapılan ayrı/bitişik yazım hataları
//
// TASARIM KURALI: temizlik ANLAMI DEĞİŞTİRMEZ. Yalnız biçim düzeltilir;
// cümle kurulmaz, kelime atılmaz, özet çıkarılmaz. Yazarın metni neyse o
// kalır — sadece okunur hâle gelir.

// Gövdeye ait olmayan, kaynak sayfadan sürüklenen blokların metin
// imzaları. Tamamı küçük harfe çevrilip karşılaştırılır.
const COP_KALIPLARI = [
  'paylaş', 'share this', 'ilgili haberler', 'related posts', 'read more',
  'devamını oku', 'advertisement', 'reklam', 'sponsored content',
  'abone ol', 'subscribe to', 'newsletter', 'bülten', 'yorum yaz',
  'leave a comment', 'tags:', 'etiketler:', 'kaynak:', 'bu yazıyı paylaş',
]

// Hiç korunmayan etiketler (içerikleriyle birlikte silinir).
const SIL_ETIKET = 'script,style,meta,link,noscript,iframe,form,button,input,select,textarea,nav,aside,footer,header,svg,video,audio,object,embed'

// İçeriği korunup kendisi kaldırılan sarmalayıcılar.
const SOY_ETIKET = 'div,section,article,main,span,font,center,small,time,label'

/**
 * temizlePastedHTML yapıştırılan HTML'i gövdeye uygun hâle getirir.
 * @param {string} html
 * @returns {string}
 */
export function temizlePastedHTML(html) {
  const doc = new DOMParser().parseFromString(html || '', 'text/html')

  doc.querySelectorAll(SIL_ETIKET).forEach((el) => el.remove())

  // Yorum düğümleri (kaynak CMS işaretleri) atılır.
  const yurutucu = doc.createTreeWalker(doc.body, NodeFilter.SHOW_COMMENT)
  const yorumlar = []
  while (yurutucu.nextNode()) yorumlar.push(yurutucu.currentNode)
  yorumlar.forEach((n) => n.remove())

  // Öznitelikler: yalnız bağlantı adresi ve görsel kaynağı kalır.
  doc.querySelectorAll('*').forEach((el) => {
    const tut = []
    if (el.tagName === 'A' && el.getAttribute('href')) tut.push(['href', el.getAttribute('href')])
    if (el.tagName === 'IMG') {
      if (el.getAttribute('src')) tut.push(['src', el.getAttribute('src')])
      if (el.getAttribute('alt')) tut.push(['alt', el.getAttribute('alt')])
    }
    ;[...el.attributes].forEach((a) => el.removeAttribute(a.name))
    tut.forEach(([ad, deger]) => el.setAttribute(ad, deger))
  })

  // Kaynak sayfanın gövde dışı blokları.
  doc.querySelectorAll('p,li,h1,h2,h3,h4,h5,h6').forEach((el) => {
    const metin = (el.textContent || '').trim().toLowerCase()
    if (!metin) return
    if (metin.length < 60 && COP_KALIPLARI.some((k) => metin.startsWith(k))) el.remove()
  })

  // Başlık düzeyi: sayfada H1 zaten yazının başlığı. Yapıştırılan H1
  // ikinci bir birinci düzey başlık yaratmasın diye H2'ye iner; H5/H6
  // gövdede okunmuyor, onlar da H4'e toplanır.
  doc.querySelectorAll('h1').forEach((el) => yenidenAdlandir(doc, el, 'h2'))
  doc.querySelectorAll('h5,h6').forEach((el) => yenidenAdlandir(doc, el, 'h4'))

  // Sarmalayıcıları soy: <div><span>metin</span></div> → metin.
  let guvenlik = 0
  while (doc.querySelector(SOY_ETIKET) && guvenlik++ < 50) {
    doc.querySelectorAll(SOY_ETIKET).forEach((el) => {
      const ebeveyn = el.parentNode
      if (!ebeveyn) return
      while (el.firstChild) ebeveyn.insertBefore(el.firstChild, el)
      ebeveyn.removeChild(el)
    })
  }

  // Metin düğümlerinde tipografi + yazım.
  const metinler = doc.createTreeWalker(doc.body, NodeFilter.SHOW_TEXT)
  const dugumler = []
  while (metinler.nextNode()) dugumler.push(metinler.currentNode)
  dugumler.forEach((n) => { n.nodeValue = duzeltMetin(n.nodeValue) })

  // Boş kalan düğümler (görselsiz p, metinsiz li).
  doc.querySelectorAll('p,li,h2,h3,h4,blockquote').forEach((el) => {
    if (!(el.textContent || '').trim() && !el.querySelector('img')) el.remove()
  })

  return doc.body.innerHTML
}

function yenidenAdlandir(doc, el, yeniAd) {
  const yeni = doc.createElement(yeniAd)
  while (el.firstChild) yeni.appendChild(el.firstChild)
  el.parentNode.replaceChild(yeni, el)
}

/**
 * duzeltMetin tipografi ve yazım düzeltmelerini uygular.
 * Dışa açık: editörün "Metni düzelt" düğmesi de bunu çağırıyor.
 * @param {string} s
 * @returns {string}
 */
export function duzeltMetin(s) {
  let t = String(s || '')

  // --- görünmez ve tekrar eden boşluklar ---
  t = t.replace(/ /g, ' ')          // sabit boşluk → normal
  t = t.replace(/[​-‍﻿]/g, '') // sıfır genişlikli karakterler
  t = t.replace(/[ \t]{2,}/g, ' ')       // çoklu boşluk
  t = t.replace(/ +\n/g, '\n')

  // --- tipografi ---
  t = t.replace(/\.{3,}/g, '…')          // üç nokta tek karakter
  t = t.replace(/ -- /g, ' — ')          // çift tire → uzun çizgi
  t = t.replace(/(\d)\s*-\s*(\d)/g, '$1-$2') // sayı aralığı bitişik

  // Noktalama öncesi boşluk (kaynak sitelerde sık).
  t = t.replace(/\s+([,.;:!?%])/g, '$1')
  // Noktalama sonrası boşluk eksikse ekle (ondalık sayıya dokunma).
  t = t.replace(/([,;:])(?=[^\s\d])/g, '$1 ')

  // --- Türkçe kesme işareti ---
  // Düz tırnak yerine tipografik kesme; "Türkiye 'nin" gibi kopuk
  // yazımları birleştirir.
  t = t.replace(/\s+'\s*/g, '’')
  t = t.replace(/'/g, '’')

  // --- sık yapılan yazım hataları ---
  const YAZIM = [
    [/\bbirşey\b/gi, 'bir şey'],
    [/\bherşey\b/gi, 'her şey'],
    [/\bhiçbirşey\b/gi, 'hiçbir şey'],
    [/\bbirkere\b/gi, 'bir kere'],
    [/\bbirçoğu\b/g, 'birçoğu'],
    [/\byalnış\b/gi, 'yanlış'],
    [/\byanlız\b/gi, 'yalnız'],
    [/\bheralde\b/gi, 'herhâlde'],
    [/\bbukadar\b/gi, 'bu kadar'],
    [/\bşukadar\b/gi, 'şu kadar'],
    [/\bdahaönce\b/gi, 'daha önce'],
    [/\bbilgisi\s+dahilinde\b/gi, 'bilgisi dâhilinde'],
    [/\bdolayısı\s*ile\b/gi, 'dolayısıyla'],
    [/\bbir\s+çok\b/gi, 'birçok'],
    [/\bbir\s+kaç\b/gi, 'birkaç'],
    [/\bher\s+hangi\b/gi, 'herhangi'],
    [/\bhiç\s+bir\b/gi, 'hiçbir'],
  ]
  YAZIM.forEach(([desen, yeni]) => { t = t.replace(desen, yeni) })

  return t
}
