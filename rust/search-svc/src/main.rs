// search-svc — Tantivy tabanlı Türkçe arama mikroservisi (spec 03 P4)
// HTTP API:
//   POST /v1/index  {"docs":[{"id":"..","title":"..","body":"..","spot":"..","tags":[..],"url":".."}]}
//   GET  /v1/search?q=kelime&limit=10   -> {"hits":[{"id","title","url","score"}],"total":N,"took_ms":X}
//   GET  /v1/stats                      -> {"docs":N,"took_ms":X}
//   GET  /healthz                       -> "ok"
use std::collections::BTreeMap;
use std::time::Instant;

use serde::{Deserialize, Serialize};
use tantivy::collector::TopDocs;
use tantivy::query::QueryParser;
use tantivy::schema::*;
use tantivy::{Index, IndexWriter, TantivyDocument};
use tiny_http::{Header, Response, Server};

const DEFAULT_PORT: u16 = 8171;

#[derive(Deserialize, Debug)]
struct IndexReq {
    #[serde(default)]
    docs: Vec<DocIn>,
}

#[derive(Deserialize, Debug, Clone)]
struct DocIn {
    id: String,
    title: String,
    #[serde(default)]
    body: String,
    #[serde(default)]
    spot: String,
    #[serde(default)]
    tags: Vec<String>,
    #[serde(default)]
    url: String,
    #[serde(default)]
    category: String,
}

#[derive(Serialize)]
struct Hit {
    id: String,
    title: String,
    url: String,
    score: f32,
}

#[derive(Serialize)]
struct SearchResp {
    hits: Vec<Hit>,
    total: usize,
    took_ms: u64,
}

#[derive(Serialize)]
struct StatsResp {
    docs: u64,
    took_ms: u64,
}

#[derive(Serialize)]
struct IndexResp {
    ok: bool,
    indexed: usize,
    received: usize,
}

#[derive(Serialize)]
struct ErrResp {
    ok: bool,
    error: String,
}

fn build_schema() -> Schema {
    let mut builder = Schema::builder();
    builder.add_text_field("id", STRING | STORED);
    // arama için normalize metin
    builder.add_text_field("title", TEXT);
    builder.add_text_field("body", TEXT);
    builder.add_text_field("spot", TEXT);
    builder.add_text_field("tags", TEXT);
    builder.add_text_field("category", TEXT);
    // gösterim için orijinal metin (STORED)
    builder.add_text_field("title_raw", STORED);
    builder.add_text_field("spot_raw", STORED);
    builder.add_text_field("category_raw", STORED);
    builder.add_text_field("url", STORED);
    builder.build()
}

/// Türkçe'ye uygun normalizasyon: küçük harf + diyakritik katlama.
fn normalize(s: &str) -> String {
    let mut out = String::with_capacity(s.len());
    for ch in s.chars() {
        let lower: char = ch.to_lowercase().next().unwrap_or(ch);
        let folded = match lower {
            'ç' => 'c',
            'ğ' => 'g',
            'ı' => 'i',
            'ö' => 'o',
            'ş' => 's',
            'ü' => 'u',
            'â' => 'a',
            'î' => 'i',
            'û' => 'u',
            other => other,
        };
        if folded.is_alphanumeric() {
            out.push(folded);
        } else {
            out.push(' ');
        }
    }
    out.split_whitespace().collect::<Vec<_>>().join(" ")
}

struct Service {
    index: Index,
    writer: IndexWriter,
}

/// Tantivy, mevcut indeksin şeması kodla uyuşmuyorsa bu hatayı döndürür.
fn is_schema_mismatch(e: &tantivy::TantivyError) -> bool {
    e.to_string().contains("schema does not match")
}

impl Service {
    /// İndeksi açar. Şema uyuşmazlığında eski indeksi yedekleyip sıfırdan
    /// oluşturur: aksi hâlde systemd Restart=always ile sonsuz çökme döngüsü
    /// oluşur (eski indeks şeması kod değişince açılamaz).
    fn open(dir: &str) -> tantivy::Result<Self> {
        let schema = build_schema();
        match Self::open_at(dir, schema.clone()) {
            Ok(svc) => Ok(svc),
            Err(e) if is_schema_mismatch(&e) => {
                let stamp = std::time::SystemTime::now()
                    .duration_since(std::time::UNIX_EPOCH)
                    .map(|d| d.as_secs())
                    .unwrap_or(0);
                let backup = format!("{}.bak-schema-{}", dir.trim_end_matches('/'), stamp);
                eprintln!(
                    "indeks şeması uyuşmuyor; eski indeks yedekleniyor: {} -> {}",
                    dir, backup
                );
                if let Err(rename_err) = std::fs::rename(dir, &backup) {
                    eprintln!(
                        "yedekleme başarısız ({}); indeks dizini siliniyor",
                        rename_err
                    );
                    std::fs::remove_dir_all(dir).expect("eski indeks silinemedi");
                }
                Self::open_at(dir, schema)
            }
            Err(e) => Err(e),
        }
    }

    fn open_at(dir: &str, schema: Schema) -> tantivy::Result<Self> {
        if !std::path::Path::new(dir).exists() {
            std::fs::create_dir_all(dir).expect("indeks dizini oluşturulamadı");
        }
        let mmap = tantivy::directory::MmapDirectory::open(dir).expect("indeks dizini açılamadı");
        let index = Index::open_or_create(mmap, schema)?;
        let writer: IndexWriter = index.writer(50_000_000)?;
        Ok(Service { index, writer })
    }

    fn field(&self, name: &str) -> Field {
        self.index.schema().get_field(name).expect("alan yok")
    }

    fn add(&mut self, d: &DocIn) -> tantivy::Result<()> {
        let mut doc = TantivyDocument::default();
        doc.add_text(self.field("id"), &d.id);
        doc.add_text(self.field("title"), normalize(&d.title));
        doc.add_text(self.field("title_raw"), &d.title);
        doc.add_text(self.field("body"), normalize(&d.body));
        doc.add_text(self.field("spot"), normalize(&d.spot));
        doc.add_text(self.field("spot_raw"), &d.spot);
        for t in &d.tags {
            doc.add_text(self.field("tags"), normalize(t));
        }
        doc.add_text(self.field("category"), normalize(&d.category));
        doc.add_text(self.field("category_raw"), &d.category);
        doc.add_text(self.field("url"), &d.url);
        self.writer.add_document(doc).map(|_| ())
    }

    fn commit(&mut self) -> tantivy::Result<()> {
        self.writer.commit()?;
        Ok(())
    }

    fn searcher(&self) -> tantivy::Result<tantivy::Searcher> {
        let r = self.index.reader()?;
        r.reload()?;
        Ok(r.searcher())
    }

    fn search(&self, q: &str, limit: usize) -> tantivy::Result<SearchResp> {
        let started = Instant::now();
        let nq = normalize(q);
        let reader = self.searcher()?;
        let schema = self.index.schema();
        let fields: Vec<Field> = ["title", "body", "spot", "tags", "category"]
            .iter()
            .filter_map(|f| schema.get_field(f).ok())
            .collect();
        let mut parser = QueryParser::for_index(&self.index, fields);
        parser.set_conjunction_by_default();
        let query = parser.parse_query(&nq)?;
        let collector = TopDocs::with_limit(limit).order_by_score();
        let top = reader.search(&query, &collector)?;
        let mut hits = Vec::with_capacity(top.len());
        for (score, addr) in top {
            if let Ok(d) = reader.doc::<TantivyDocument>(addr) {
                let get = |name: &str| -> String {
                    match schema.get_field(name).ok() {
                        Some(f) => d
                            .get_first(f)
                            .and_then(|v| v.as_str())
                            .unwrap_or_default()
                            .to_string(),
                        None => String::new(),
                    }
                };
                hits.push(Hit {
                    id: get("id"),
                    title: {
                        let raw = get("title_raw");
                        if raw.is_empty() {
                            get("title")
                        } else {
                            raw
                        }
                    },
                    url: get("url"),
                    score,
                });
            }
        }
        let took = started.elapsed().as_millis() as u64;
        Ok(SearchResp {
            total: hits.len(),
            hits,
            took_ms: took,
        })
    }

    fn stats(&self) -> tantivy::Result<StatsResp> {
        let started = Instant::now();
        let reader = self.searcher()?;
        let docs = reader.num_docs();
        Ok(StatsResp {
            docs,
            took_ms: started.elapsed().as_millis() as u64,
        })
    }
}

fn json_ok<R: Serialize>(req: tiny_http::Request, val: &R) {
    let body = serde_json::to_string(val).unwrap_or_else(|_| "{}".into());
    let resp = Response::from_string(body)
        .with_header(
            Header::from_bytes(
                &b"Content-Type"[..],
                &b"application/json; charset=utf-8"[..],
            )
            .unwrap(),
        )
        .with_status_code(200);
    let _ = req.respond(resp);
}

fn json_err(req: tiny_http::Request, code: u16, msg: &str) {
    let body = ErrResp {
        ok: false,
        error: msg.replace('"', "'"),
    };
    let resp = Response::from_string(serde_json::to_string(&body).unwrap_or_default())
        .with_header(
            Header::from_bytes(
                &b"Content-Type"[..],
                &b"application/json; charset=utf-8"[..],
            )
            .unwrap(),
        )
        .with_status_code(code);
    let _ = req.respond(resp);
}

fn main() {
    let dir = std::env::var("SEARCH_INDEX_DIR").unwrap_or_else(|_| "./data/index".to_string());
    let port = std::env::var("SEARCH_PORT")
        .ok()
        .and_then(|p| p.parse().ok())
        .unwrap_or(DEFAULT_PORT);
    let mut svc = Service::open(&dir).expect("indeks açılamadı");
    // Bkz. image-svc: kap icinde 0.0.0.0 gerekir, yerelde 127.0.0.1 kalir.
    let bind = std::env::var("SEARCH_BIND").unwrap_or_else(|_| "127.0.0.1".to_string());
    let server = Server::http((bind.as_str(), port)).expect("dinlenemedi");
    eprintln!(
        "search-svc hazır: http://{}:{} (indeks: {})",
        bind, port, dir
    );

    for mut request in server.incoming_requests() {
        let url = request.url().to_string();
        let method = request.method().clone();
        if url.starts_with("/healthz") {
            let resp = Response::from_string("ok").with_status_code(200);
            let _ = request.respond(resp);
            continue;
        }
        if url.starts_with("/v1/index") && method == tiny_http::Method::Post {
            let mut raw: Vec<u8> = Vec::new();
            let _ = request.as_reader().read_to_end(&mut raw);
            let body = String::from_utf8_lossy(&raw).to_string();
            match serde_json::from_str::<IndexReq>(&body) {
                Ok(req) => {
                    let n = req.docs.len();
                    let mut ok = 0usize;
                    for d in &req.docs {
                        if svc.add(d).is_ok() {
                            ok += 1;
                        }
                    }
                    let _ = svc.commit();
                    json_ok(
                        request,
                        &IndexResp {
                            ok: true,
                            indexed: ok,
                            received: n,
                        },
                    );
                }
                Err(e) => json_err(request, 400, &format!("geçersiz gövde: {}", e)),
            }
            continue;
        }
        if url.starts_with("/v1/search") {
            let params: BTreeMap<String, String> = url
                .split('?')
                .nth(1)
                .unwrap_or("")
                .split('&')
                .filter(|p| !p.is_empty())
                .filter_map(|kv| kv.split_once('='))
                .map(|(k, v)| (k.to_string(), url_decode(v)))
                .collect();
            let q = params.get("q").cloned().unwrap_or_default();
            let limit = params
                .get("limit")
                .and_then(|l| l.parse::<usize>().ok())
                .unwrap_or(10)
                .min(50);
            if q.trim().is_empty() {
                json_err(request, 400, "q gerekli");
                continue;
            }
            match svc.search(&q, limit) {
                Ok(res) => json_ok(request, &res),
                Err(e) => json_err(request, 500, &format!("sorgu hatası: {}", e)),
            }
            continue;
        }
        if url.starts_with("/v1/stats") {
            match svc.stats() {
                Ok(s) => json_ok(request, &s),
                Err(e) => json_err(request, 500, &e.to_string()),
            }
            continue;
        }
        json_err(request, 404, "bulunamadı");
    }
}

fn url_decode(s: &str) -> String {
    let b = s.as_bytes();
    let mut out = Vec::with_capacity(b.len());
    let mut i = 0;
    while i < b.len() {
        match b[i] {
            b'%' if i + 2 < b.len() => {
                let hex = std::str::from_utf8(&b[i + 1..i + 3]).unwrap_or("0");
                out.push(u8::from_str_radix(hex, 16).unwrap_or(b' '));
                i += 3;
            }
            b'+' => {
                out.push(b' ');
                i += 1;
            }
            c => {
                out.push(c);
                i += 1;
            }
        }
    }
    String::from_utf8_lossy(&out).to_string()
}

#[cfg(test)]
mod tests {
    use super::*;

    /// Eski şemalı bir indeks açıldığında servis çökmemeli; indeksi yedekleyip
    /// yenisini oluşturmalı (systemd çökme döngüsünün regresyon testi).
    #[test]
    fn recovers_from_schema_mismatch() {
        let stamp = std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap()
            .as_nanos();
        let root = std::env::temp_dir().join(format!("theblock-search-test-{}", stamp));
        let dir = root.join("index");
        std::fs::create_dir_all(&dir).unwrap();

        // Bilerek farklı şema: title STORED (güncel şemada yalnız title_raw STORED).
        let mut builder = Schema::builder();
        builder.add_text_field("id", STRING | STORED);
        builder.add_text_field("title", TEXT | STORED);
        Index::create_in_dir(&dir, builder.build()).unwrap();

        let svc = Service::open(dir.to_str().unwrap()).expect("şema toparlanmalı");
        assert!(svc.index.schema().get_field("title_raw").is_ok());
        assert!(svc.index.schema().get_field("category_raw").is_ok());
        drop(svc);

        let backups = std::fs::read_dir(&root)
            .unwrap()
            .filter_map(|e| e.ok())
            .filter(|e| {
                e.file_name()
                    .to_string_lossy()
                    .starts_with("index.bak-schema-")
            })
            .count();
        assert_eq!(backups, 1, "eski indeks yedeklenmeli");

        std::fs::remove_dir_all(&root).unwrap();
    }
}
