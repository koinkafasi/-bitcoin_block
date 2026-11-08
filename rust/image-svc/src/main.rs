// image-svc v0.2 — Rust görsel dönüşüm mikroservisi (sıcak yol).
// GET  /healthz
// POST /v1/process?ratio=1.7778&width=1200&format=jpeg
//   Gövde: ham görsel (jpg/png/webp). Yanıt: JPEG veya PNG baytları.
// Not: WebP/AVIF çıkışı sırada (cwebp / libvips-nasm yığını kurulunca).
use image::GenericImageView;
use image::ImageEncoder;
use std::io::{Read, Write};
use std::net::{TcpListener, TcpStream};

fn main() -> std::io::Result<()> {
    let port = std::env::var("IMAGE_SVC_PORT").unwrap_or_else(|_| "8170".into());
    // Kap icinde pod disindan erisilebilmesi icin adres yapilandirilabilir.
    // Varsayilan 127.0.0.1: systemd altindaki yerel davranis degismez.
    let bind = std::env::var("IMAGE_SVC_BIND").unwrap_or_else(|_| "127.0.0.1".into());
    let listener = TcpListener::bind((bind.as_str(), port.parse::<u16>().unwrap()))?;
    eprintln!("image-svc v0.2 dinliyor: {bind}:{port}");
    for stream in listener.incoming() {
        let Ok(mut stream) = stream else {
            continue;
        };
        if let Err(e) = handle(&mut stream) {
            eprintln!("hata: {e}");
        }
    }
    Ok(())
}

fn respond(stream: &mut TcpStream, status: u16, ctype: &str, body: &[u8]) -> std::io::Result<()> {
    let reason = if status == 200 {
        "OK"
    } else if status == 400 {
        "Bad Request"
    } else {
        "Internal Server Error"
    };
    let head = format!(
        "HTTP/1.1 {status} {reason}\r\nContent-Type: {ctype}\r\nContent-Length: {}\r\nConnection: close\r\n\r\n",
        body.len()
    );
    stream.write_all(head.as_bytes())?;
    stream.write_all(body)?;
    Ok(())
}

fn handle(stream: &mut TcpStream) -> std::io::Result<()> {
    let mut buf = [0u8; 8192];
    let n = stream.read(&mut buf)?;
    let raw = &buf[..n];
    let text = String::from_utf8_lossy(raw);
    let mut lines = text.lines();
    let first = lines.next().unwrap_or("");
    let mut parts = first.split_whitespace();
    let method = parts.next().unwrap_or("").to_string();
    let target = parts.next().unwrap_or("/").to_string();

    if target.starts_with("/healthz") {
        return respond(
            stream,
            200,
            "application/json",
            br#"{"status":"ok","service":"image-svc","version":"0.2.0"}"#,
        );
    }

    let mut clen = 0usize;
    for line in lines {
        if line.trim().is_empty() {
            break;
        }
        let lower = line.to_ascii_lowercase();
        if let Some(v) = lower.strip_prefix("content-length:") {
            clen = v.trim().parse().unwrap_or(0);
        }
    }

    if method == "POST" && target.starts_with("/v1/process") {
        let start = find_body_start(raw);
        let mut body = Vec::new();
        body.extend_from_slice(&raw[start.min(raw.len())..]);
        while body.len() < clen {
            let mut more = [0u8; 8192];
            let k = stream.read(&mut more)?;
            if k == 0 {
                break;
            }
            body.extend_from_slice(&more[..k]);
        }
        body.truncate(clen);
        if body.is_empty() {
            return respond(stream, 400, "application/json", br#"{"error":"bos_beden"}"#);
        }
        return process(stream, &target, &body);
    }

    respond(stream, 404, "application/json", br#"{"error":"not_found"}"#)
}

fn find_body_start(raw: &[u8]) -> usize {
    if let Some(p) = raw.windows(4).position(|w| w == b"\r\n\r\n") {
        return p + 4;
    }
    if let Some(p) = raw.windows(2).position(|w| w == b"\n\n") {
        return p + 2;
    }
    0
}

fn process(stream: &mut TcpStream, target: &str, body: &[u8]) -> std::io::Result<()> {
    let (_, q) = target.split_once('?').unwrap_or((target, ""));
    let mut ratio = 16.0f64 / 9.0f64;
    let mut width = 1200usize;
    let mut format = "jpeg";
    let mut quality = 85u8;
    for pair in q.split('&') {
        if pair.is_empty() {
            continue;
        }
        let (k, v) = pair.split_once('=').unwrap_or((pair, ""));
        match k {
            "ratio" => {
                if let Ok(f) = v.parse::<f64>() {
                    ratio = f;
                }
            }
            "width" => {
                if let Ok(u) = v.parse::<usize>() {
                    width = u;
                }
            }
            "format" => format = v,
            // Go tarafı yerel yolda q=82 kullanıyor; çıktı boyutu ve kalite
            // birebir aynı kalsın diye çağıran kaliteyi geçebilir.
            "quality" => {
                if let Ok(u) = v.parse::<u8>() {
                    quality = u.clamp(1, 100);
                }
            }
            _ => {}
        }
    }

    let img = match image::load_from_memory(body) {
        Ok(i) => i,
        Err(e) => {
            return respond(
                stream,
                400,
                "application/json",
                format!(r#"{{"error":"decode:{e}"}}"#).as_bytes(),
            );
        }
    };
    let (w, h) = img.dimensions();
    if w == 0 || h == 0 {
        return respond(stream, 400, "application/json", br#"{"error":"boyut_yok"}"#);
    }

    let crop = if w as f64 / h as f64 > ratio {
        let cw = (h as f64 * ratio) as u32;
        let x = (w - cw) / 2;
        img.crop_imm(x, 0, cw, h)
    } else {
        let ch = (w as f64 / ratio) as u32;
        let y = (h - ch) / 2;
        img.crop_imm(0, y, w, ch)
    };

    let out = if width > 0 && width != crop.width() as usize {
        // CatmullRom: Go tarafındaki xdraw.CatmullRom ile aynı çekirdek.
        // Triangle (bilinear) daha yumuşak sonuç veriyor ve küçültmede
        // srcset varyantları arasında görünür keskinlik farkı yaratıyordu.
        crop.resize(
            width as u32,
            (width as f64 / ratio) as u32,
            image::imageops::FilterType::CatmullRom,
        )
    } else {
        crop
    };

    match format {
        "png" => {
            let rgb = out.to_rgb8();
            let mut buf = Vec::new();
            let enc = image::codecs::png::PngEncoder::new(&mut buf);
            if enc
                .write_image(
                    &rgb,
                    rgb.width(),
                    rgb.height(),
                    image::ExtendedColorType::Rgb8,
                )
                .is_ok()
            {
                respond(stream, 200, "image/png", &buf)
            } else {
                respond(stream, 500, "application/json", br#"{"error":"png"}"#)
            }
        }
        _ => {
            let mut buf = Vec::new();
            let mut enc = image::codecs::jpeg::JpegEncoder::new_with_quality(&mut buf, quality);
            if enc.encode_image(&out).is_ok() {
                respond(stream, 200, "image/jpeg", &buf)
            } else {
                respond(stream, 500, "application/json", br#"{"error":"jpeg"}"#)
            }
        }
    }
}

#[cfg(test)]
#[path = "main_test.rs"]
mod tests;
