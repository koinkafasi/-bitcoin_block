// image-svc testleri: gerçek TCP üzerinden uçtan uca.
//
// Neden TCP: servis HTTP'yi elle ayrıştırıyor (başlık/gövde sınırı,
// `Content-Length` döngüsü, 8192 baytlık ilk tampon). Bu mantığı ayrı bir
// fonksiyona çıkarıp test etmek üretimde çalışan kodu değiştirmek olurdu;
// gerçek soket üzerinden sınamak aynı yolu kapsıyor ve hızlı (milisaniye).
//
// Faz 208: bu pakette hiç test yoktu (backlog ölçümü: 0).

use super::*;
use std::io::{Read, Write};
use std::net::TcpListener;

/// Tek bağlantı kabul edip `handle`'ı çalıştıran arka plan sunucusu.
fn sunucu() -> (u16, std::thread::JoinHandle<()>) {
    let l = TcpListener::bind(("127.0.0.1", 0)).expect("bağlan");
    let port = l.local_addr().unwrap().port();
    let h = std::thread::spawn(move || {
        if let Ok((mut s, _)) = l.accept() {
            let _ = handle(&mut s);
        }
    });
    (port, h)
}

struct Yanit {
    status: u16,
    ctype: String,
    body: Vec<u8>,
}

/// Ham istek gönderir, yanıtı başlık/gövde olarak ayrıştırır.
fn gonder(raw: &[u8]) -> Yanit {
    let (port, h) = sunucu();
    let mut s = std::net::TcpStream::connect(("127.0.0.1", port)).expect("bağlan");
    s.write_all(raw).expect("yaz");
    let mut out = Vec::new();
    // Sunucu `Connection: close` ile kapatıyor; EOF'a kadar oku.
    let _ = s.read_to_end(&mut out);
    let _ = h.join();

    let p = out
        .windows(4)
        .position(|w| w == b"\r\n\r\n")
        .expect("başlık sonu yok");
    let head = String::from_utf8_lossy(&out[..p]).to_string();
    let body = out[p + 4..].to_vec();

    let mut satirlar = head.lines();
    let ilk = satirlar.next().unwrap_or("");
    let status = ilk
        .split_whitespace()
        .nth(1)
        .and_then(|s| s.parse().ok())
        .unwrap_or(0);
    let ctype = satirlar
        .find_map(|l| {
            let low = l.to_ascii_lowercase();
            low.strip_prefix("content-type:")
                .map(|v| v.trim().to_string())
        })
        .unwrap_or_default();
    Yanit {
        status,
        ctype,
        body,
    }
}

/// POST isteği kurar (Content-Length doğru hesaplanır).
fn post(yol: &str, govde: &[u8]) -> Vec<u8> {
    let mut v = format!(
        "POST {yol} HTTP/1.1\r\nHost: localhost\r\nContent-Type: application/octet-stream\r\nContent-Length: {}\r\n\r\n",
        govde.len()
    )
    .into_bytes();
    v.extend_from_slice(govde);
    v
}

/// Verilen boyutta JPEG üretir.
fn jpeg_uret(w: u32, h: u32) -> Vec<u8> {
    let img = image::RgbImage::from_pixel(w, h, image::Rgb([30, 90, 180]));
    let mut buf = Vec::new();
    let mut enc = image::codecs::jpeg::JpegEncoder::new_with_quality(&mut buf, 90);
    enc.encode_image(&img).expect("kodla");
    buf
}

#[test]
fn healthz_ok_doner() {
    let y = gonder(b"GET /healthz HTTP/1.1\r\nHost: localhost\r\n\r\n");
    assert_eq!(y.status, 200);
    assert_eq!(y.ctype, "application/json");
    let s = String::from_utf8_lossy(&y.body);
    assert!(s.contains("\"status\":\"ok\""), "gövde: {s}");
    assert!(s.contains("image-svc"));
}

#[test]
fn bilinmeyen_yol_404() {
    let y = gonder(b"GET /v1/yok HTTP/1.1\r\nHost: localhost\r\n\r\n");
    assert_eq!(y.status, 404);
    assert!(String::from_utf8_lossy(&y.body).contains("not_found"));
}

#[test]
fn bos_govde_400() {
    let y = gonder(&post("/v1/process", b""));
    assert_eq!(y.status, 400);
    assert!(String::from_utf8_lossy(&y.body).contains("bos_beden"));
}

#[test]
fn bozuk_govde_400_decode() {
    let y = gonder(&post("/v1/process", b"bu bir gorsel degil"));
    assert_eq!(y.status, 400);
    let s = String::from_utf8_lossy(&y.body);
    assert!(s.contains("decode:"), "gövde: {s}");
}

/// 4:3 giriş 16:9'a kırpılır ve istenen genişliğe ölçeklenir.
#[test]
fn kirpma_ve_olcekleme_16_9() {
    let giris = jpeg_uret(400, 300); // 4:3
    let y = gonder(&post("/v1/process?ratio=1.7778&width=200&format=jpeg", &giris));
    assert_eq!(y.status, 200);
    assert_eq!(y.ctype, "image/jpeg");

    let out = image::load_from_memory(&y.body).expect("çıktı çözülemedi");
    let (w, h) = out.dimensions();
    assert_eq!(w, 200, "genişlik istenen değere ölçeklenmeli");
    // 200 / 1.7778 ≈ 112 (yuvarlama payı ile)
    assert!(
        (h as i64 - 112).abs() <= 2,
        "yükseklik 16:9 oranında olmalı, gelen {h}"
    );
    assert!(out.width() as f64 / out.height() as f64 > 1.7);
}

/// Oran verilmezse varsayılan 16:9 uygulanır (geniş giriş kırpılmaz, dar olan kırpılır).
#[test]
fn varsayilan_oran_16_9() {
    let giris = jpeg_uret(900, 300); // 3:1 — çok geniş
    let y = gonder(&post("/v1/process?width=300", &giris));
    assert_eq!(y.status, 200);
    let out = image::load_from_memory(&y.body).expect("çözülemedi");
    let (_, h) = out.dimensions();
    // 300 genişlikte 16:9 → ~169 yükseklik
    assert!((h as i64 - 169).abs() <= 2, "yükseklik {h}");
}

#[test]
fn png_cikisi() {
    let giris = jpeg_uret(320, 180);
    let y = gonder(&post("/v1/process?width=160&format=png", &giris));
    assert_eq!(y.status, 200);
    assert_eq!(y.ctype, "image/png");
    assert_eq!(&y.body[..4], b"\x89PNG", "PNG imzası yok");
    let out = image::load_from_memory(&y.body).expect("çözülemedi");
    assert_eq!(out.dimensions().0, 160);
}

/// Genişlik aynıysa yeniden ölçekleme yapılmaz (kırpma boyutu korunur).
#[test]
fn ayni_genislikte_olcekleme_yok() {
    let giris = jpeg_uret(160, 90); // tam 16:9
    let y = gonder(&post("/v1/process?width=160", &giris));
    assert_eq!(y.status, 200);
    let out = image::load_from_memory(&y.body).expect("çözülemedi");
    assert_eq!(out.dimensions(), (160, 90));
}

/// 8192 baytı aşan gövde Content-Length döngüsüyle eksiksiz okunmalı.
/// Bu, ilk tamponu aşan ilk gerçek yol.
#[test]
fn buyuk_govde_tamamen_okunur() {
    // Gürültülü büyük JPEG: sıkıştırma küçültmesin diye rastgele pikseller.
    let mut img = image::RgbImage::new(300, 200);
    let mut x: u32 = 12345;
    for p in img.pixels_mut() {
        x = x.wrapping_mul(1103515245).wrapping_add(12345);
        *p = image::Rgb([(x >> 16) as u8, (x >> 8) as u8, x as u8]);
    }
    let mut buf = Vec::new();
    let mut enc = image::codecs::jpeg::JpegEncoder::new_with_quality(&mut buf, 100);
    enc.encode_image(&img).expect("kodla");
    assert!(buf.len() > 8192, "test gövdesi 8192'yi aşmalı: {}", buf.len());

    let y = gonder(&post("/v1/process?width=120", &buf));
    assert_eq!(y.status, 200, "gövde: {}", String::from_utf8_lossy(&y.body));
    let out = image::load_from_memory(&y.body).expect("çözülemedi");
    assert_eq!(out.dimensions().0, 120);
}

/// Geçersiz sayısal parametreler varsayılana düşer, istek başarısız olmaz.
#[test]
fn gecersiz_parametreler_varsayilana_duser() {
    let giris = jpeg_uret(640, 360);
    let y = gonder(&post("/v1/process?ratio=abc&width=xyz&format=", &giris));
    assert_eq!(y.status, 200);
    assert_eq!(y.ctype, "image/jpeg", "boş format → jpeg");
    let out = image::load_from_memory(&y.body).expect("çözülemedi");
    // varsayılan genişlik 1200
    assert_eq!(out.dimensions().0, 1200);
}

/// Sorgu dizesindeki bilinmeyen anahtarlar yok sayılır.
#[test]
fn bilinmeyen_sorgu_anahtari_yoksayilir() {
    let giris = jpeg_uret(320, 180);
    let y = gonder(&post("/v1/process?bilinmeyen=1&width=100", &giris));
    assert_eq!(y.status, 200);
    assert_eq!(image::load_from_memory(&y.body).unwrap().dimensions().0, 100);
}

/// Body-start bulucu: CRLF ve LF sonlarını, başlık yokluğunu doğru ele alır.
#[test]
fn body_start_bulucu() {
    assert_eq!(find_body_start(b"GET / HTTP/1.1\r\n\r\nABC"), 18);
    assert_eq!(find_body_start(b"POST / HTTP/1.1\n\nABC"), 17);
    // Ayırıcı yoksa 0 (gövde baştan okunur).
    assert_eq!(find_body_start(b"bozuk"), 0);
}

/// Yanıt başlığı Content-Length'i gerçek gövde uzunluğuyla eşleşmeli.
#[test]
fn content_length_dogru() {
    let (port, h) = sunucu();
    let mut s = std::net::TcpStream::connect(("127.0.0.1", port)).unwrap();
    s.write_all(b"GET /healthz HTTP/1.1\r\nHost: localhost\r\n\r\n")
        .unwrap();
    let mut out = Vec::new();
    let _ = s.read_to_end(&mut out);
    let _ = h.join();
    let p = out.windows(4).position(|w| w == b"\r\n\r\n").unwrap();
    let head = String::from_utf8_lossy(&out[..p]).to_lowercase();
    let len: usize = head
        .lines()
        .find_map(|l| l.strip_prefix("content-length:"))
        .and_then(|v| v.trim().parse().ok())
        .expect("Content-Length yok");
    assert_eq!(len, out.len() - (p + 4));
}
