# theblock image-svc — Rust görsel dönüşüm mikroservisi.
#
# `image` crate'i saf Rust (jpeg/png/webp özellikleriyle); libvips gibi
# sistem kütüphanesi gerekmiyor. Servis adı/açıklaması libvips diyor ama
# Cargo.toml'da öyle bir bağımlılık yok.
#
# Derleme bağlamı: depo kökü.
#   docker build -f deploy/docker/image-svc.Dockerfile -t <imaj> .

FROM rust:1-bookworm AS derle
WORKDIR /src

# Bağımlılıkları önce derle: kaynak değişince crates yeniden derlenmesin.
COPY rust/image-svc/Cargo.toml rust/image-svc/Cargo.lock ./
RUN mkdir -p src && echo 'fn main(){}' > src/main.rs \
 && cargo build --release \
 && rm -rf src

COPY rust/image-svc/src ./src
# Sahte main.rs'in derlenmiş hâli kalmasın:
RUN touch src/main.rs && cargo build --release

FROM debian:bookworm-slim
RUN apt-get update \
 && apt-get upgrade -y \
 && apt-get install -y --no-install-recommends ca-certificates \
 && rm -rf /var/lib/apt/lists/* \
 && useradd -u 10002 -r -s /usr/sbin/nologin imagesvc

COPY --from=derle /src/target/release/image-svc /usr/local/bin/image-svc

# Kap içinde pod dışından erişilebilmesi için 0.0.0.0.
# Yereldeki systemd kurulumunda bu değişken yok, 127.0.0.1'de kalır.
ENV IMAGE_SVC_PORT=8170 \
    IMAGE_SVC_BIND=0.0.0.0

EXPOSE 8170
USER imagesvc
ENTRYPOINT ["/usr/local/bin/image-svc"]
