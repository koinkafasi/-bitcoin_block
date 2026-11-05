// Faz 192 — collab tohumlama yarışı için duman testi.
//
// Node test çatısı yok; bu betik doğrudan çalışır:
//   node tohum-test.mjs
//
// Ölçülen: aynı belge için EŞZAMANLI iki tohumlama çağrısı belgeyi kaç kez
// ekliyor? Düzeltme öncesi 2 (yinelenen gövde), sonrası 1.
import * as Y from 'yjs'
import { getSchema } from '@tiptap/core'
import { prosemirrorJSONToYDoc, yDocToProsemirrorJSON } from 'y-prosemirror'
import { schemaExtensions } from './schema.js'

const schema = getSchema(schemaExtensions)

const seed = {
  type: 'doc',
  content: [
    { type: 'paragraph', content: [{ type: 'text', text: 'Birinci paragraf.' }] },
    { type: 'paragraph', content: [{ type: 'text', text: 'İkinci paragraf.' }] },
    { type: 'paragraph', content: [{ type: 'text', text: 'Üçüncü paragraf.' }] },
  ],
}

function uygula(doc) {
  const seeded = prosemirrorJSONToYDoc(schema, seed, 'default')
  Y.applyUpdate(doc, Y.encodeStateAsUpdate(seeded))
}

// --- 1) DÜZELTME ÖNCESİ davranış: eski sunucu kodu birebir modellenir ---
// Eski kod: kontrol `await fetchSeed`TEN ÖNCE yapılıyordu; iki eşzamanlı
// çağrı da "boş" görüp tohumu uyguluyordu.
{
  const doc = new Y.Doc()
  async function eskiTohumla(gecikme) {
    if (doc.getXmlFragment('default').length > 0) return
    await new Promise((r) => setTimeout(r, gecikme)) // fetchSeed
    uygula(doc)
  }
  await Promise.all([eskiTohumla(20), eskiTohumla(1)])
  const j = yDocToProsemirrorJSON(doc, 'default')
  console.log('düzeltme öncesi (naif yarış):', j.content.length, 'blok  ← 6 = yineleme')
  if (j.content.length !== 6) {
    console.error('  UYARI: naif yarış yeniden üretilemedi (test modeli zayıf)')
  }
}

// --- 2) DÜZELTME SONRASI: sunucudaki tohumla() mantığı ---
{
  const tohumlanan = new Map()
  const seedlenmis = new WeakSet()
  async function tohumla(document, ad, gecikme) {
    if (seedlenmis.has(document)) return
    if (document.getXmlFragment('default').length > 0) { seedlenmis.add(document); return }
    const ucus = tohumlanan.get(ad)
    if (ucus) { await ucus; return }
    const is = (async () => {
      await new Promise((r) => setTimeout(r, gecikme)) // fetchSeed
      if (document.getXmlFragment('default').length > 0) return
      uygula(document)
      seedlenmis.add(document)
    })()
    tohumlanan.set(ad, is)
    try { await is } finally { tohumlanan.delete(ad) }
  }

  const doc = new Y.Doc()
  await Promise.all([
    tohumla(doc, 'content-1', 30),
    tohumla(doc, 'content-1', 1),
    tohumla(doc, 'content-1', 5),
  ])
  const j = yDocToProsemirrorJSON(doc, 'default')
  console.log('düzeltme sonrası (kilitli):    ', j.content.length, 'blok')

  // İkinci kez çağrı (WeakSet) hâlâ tekrar etmemeli.
  await tohumla(doc, 'content-1', 1)
  const j2 = yDocToProsemirrorJSON(doc, 'default')
  console.log('tekrar çağrı sonrası:          ', j2.content.length, 'blok')

  if (j.content.length !== 3 || j2.content.length !== 3) {
    console.error('BAŞARISIZ: yineleme hâlâ oluşuyor')
    process.exit(1)
  }
  console.log('GEÇTİ: 3 paragraf, yineleme yok')
}
