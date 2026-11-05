import { Server } from '@hocuspocus/server'
import * as Y from 'yjs'
import { getSchema } from '@tiptap/core'
import { prosemirrorJSONToYDoc, yDocToProsemirrorJSON } from 'y-prosemirror'
import { schemaExtensions } from './schema.js'
import crypto from 'node:crypto'

const schema = getSchema(schemaExtensions)
const GO_URL = process.env.GO_URL || 'http://127.0.0.1:8070'
const SECRET = process.env.COLLAB_SECRET || ''
const PORT = Number(process.env.PORT || 1234)

function contentID(documentName) {
  const m = String(documentName || '').match(/^content-(\d+)$/)
  return m ? Number(m[1]) : 0
}

function verifyToken(token, documentName) {
  if (!SECRET) throw new Error('COLLAB_SECRET yok')
  const parts = String(token || '').split(':')
  if (parts.length !== 4) throw new Error('geçersiz token')
  const [cid, uid, exp, sig] = parts
  const payload = `${cid}:${uid}:${exp}`
  const expected = crypto.createHmac('sha256', SECRET).update(payload).digest('hex')
  const a = Buffer.from(sig)
  const b = Buffer.from(expected)
  if (a.length !== b.length || !crypto.timingSafeEqual(a, b)) throw new Error('imza geçersiz')
  if (Number(exp) < Math.floor(Date.now() / 1000)) throw new Error('token süresi doldu')
  if (Number(cid) !== contentID(documentName)) throw new Error('belge uyuşmuyor')
  return { userID: Number(uid) }
}

async function fetchSeed(id) {
  if (!SECRET) return null
  const res = await fetch(`${GO_URL}/internal/collab/${id}`, {
    headers: { 'X-Collab-Secret': SECRET },
  })
  if (!res.ok) return null
  const json = await res.json()
  json.__version = Number(res.headers.get('X-Collab-Version') || 0)
  return json
}

// saveToGo: iyimser eşzamanlılık. Okunan sürümü gönderir; Go 409 dönerse
// arada panel formu yazmış demektir. Bu durumda güncel sürümü alıp BİR kez
// yeniden dener ve çakışmayı günlüğe yazar (sessiz "son yazan kazanır"
// yerine görünür kayıt).
async function saveToGo(id, json, version) {
  if (!SECRET) return 0
  const dene = async (surum) => fetch(`${GO_URL}/internal/collab/${id}`, {
    method: 'POST',
    headers: {
      'X-Collab-Secret': SECRET,
      'Content-Type': 'application/json',
      ...(surum > 0 ? { 'X-Collab-Version': String(surum) } : {}),
    },
    body: JSON.stringify(json),
  })

  let res = await dene(version)
  if (res.status === 409) {
    const guncel = Number(res.headers.get('X-Collab-Current-Version') || 0)
    console.error(`collab: SÜRÜM ÇAKIŞMASI içerik=${id} beklenen=${version} güncel=${guncel} ` +
      '— güncel sürümle yeniden deneniyor. Arada panel formundan kayıt yapılmış olabilir.')
    if (guncel > 0) res = await dene(guncel)
  }
  if (res.status === 404) return 0
  if (!res.ok) {
    console.error('collab save failed', id, res.status, await res.text())
    return 0
  }
  try {
    const govde = await res.json()
    return Number(govde.version || 0)
  } catch {
    return 0
  }
}

/* ---- Faz 192: tohumlama yarışı ----
 *
 * BULGU (üretim verisi): `contents.id=554` gövdesi 56 bloktu ve 1..28 ile
 * 29..56 BİREBİR aynıydı; EN çevirisi de aynı yinelemeyi taşıyordu.
 *
 * Kök neden: `onLoadDocument` async. Tohum kontrolü
 * (`getXmlFragment('default').length > 0`) ile `fetchSeed` (Go'ya ağ isteği)
 * arasında bir `await` var. Aynı belge için iki çağrı bu aralıkta gelirse
 * İKİSİ de "boş" görüp tohumu uygular → belge iki kez eklenir. Yinelemenin
 * tam olarak 2× ve bitişik yarımlar olması bununla uyuşuyor.
 *
 * Düzeltme iki katmanlı:
 *   1) `tohumlanan`: aynı belge adı için uçuşta olan tohumlama sözü; ikinci
 *      çağrı kendi tohumunu uygulamaz, birincisini bekler.
 *   2) `seedlenmis`: Y.Doc örneği başına işaret — aynı örnek ikinci kez
 *      tohumlanmaz (WeakSet: doc serbest bırakılınca kayıt da düşer).
 */
const tohumlanan = new Map()   // documentName -> Promise
const seedlenmis = new WeakSet() // Y.Doc örnekleri
// Belge başına son bilinen sürüm (iyimser eşzamanlılık için).
const surumler = new Map()     // documentName -> number

async function tohumla(document, documentName, id) {
  if (seedlenmis.has(document)) return
  if (document.getXmlFragment('default').length > 0) {
    seedlenmis.add(document)
    return
  }
  const ucus = tohumlanan.get(documentName)
  if (ucus) {
    // Başka bir çağrı şu anda tohumluyor: bitmesini bekle, KENDİ tohumunu
    // uygulama. (Aksi hâlde tam olarak yinelenen gövde oluşuyordu.)
    await ucus
    return
  }
  const is = (async () => {
    const json = await fetchSeed(id)
    if (json && json.__version) surumler.set(documentName, json.__version)
    if (!json || json.type !== 'doc') return
    // Kontrolü await'TEN SONRA tekrarla: bu arada başka bir yol yazmış olabilir.
    if (document.getXmlFragment('default').length > 0) return
    const seeded = prosemirrorJSONToYDoc(schema, json, 'default')
    Y.applyUpdate(document, Y.encodeStateAsUpdate(seeded))
    seedlenmis.add(document)
  })()
  tohumlanan.set(documentName, is)
  try {
    await is
  } catch (err) {
    console.error('collab seed error', id, err.message)
  } finally {
    tohumlanan.delete(documentName)
  }
}

// yinelenmisGovde: blok listesi tam iki kez yazilmis mi?
//
// Kayıt sırasında görülürse günlüğe düşer: sessiz veri bozulması yerine
// görünür uyarı. Otomatik DÜZELTME yapılmaz (meşru içerik de olabilir).
function yinelenmisGovde(json) {
  const b = json && Array.isArray(json.content) ? json.content : null
  if (!b || b.length < 4 || b.length % 2 !== 0) return false
  const yari = b.length / 2
  for (let i = 0; i < yari; i++) {
    if (JSON.stringify(b[i]) !== JSON.stringify(b[i + yari])) return false
  }
  return true
}

const server = new Server({
  port: PORT,
  debounce: 1500,
  maxDebounce: 8000,
  unloadImmediately: false,
  async onAuthenticate({ token, documentName }) {
    return verifyToken(token, documentName)
  },
  async onLoadDocument({ document, documentName }) {
    const id = contentID(documentName)
    if (!id) return document
    await tohumla(document, documentName, id)
    return document
  },
  async onStoreDocument({ document, documentName }) {
    const id = contentID(documentName)
    if (!id) return
    try {
      const json = yDocToProsemirrorJSON(document, 'default')
      if (yinelenmisGovde(json)) {
        console.error(`collab: YİNELENMİŞ GÖVDE içerik=${id} blok=${json.content.length} ` +
          '(ilk yarı = ikinci yarı). Kaydediliyor ama incelenmeli.')
      }
      const surum = surumler.get(documentName) || 0
      const yeni = await saveToGo(id, json, surum)
      if (yeni > 0) surumler.set(documentName, yeni)
    } catch (err) {
      console.error('collab store error', id, err.message)
    }
  },
  async onDisconnect() {
    // Hocuspocus stores via debounced onStoreDocument.
  },
})

await server.listen()
console.log(`collab server listening on :${PORT}`)
