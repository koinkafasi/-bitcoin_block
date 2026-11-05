# theblock search-svc — Rust/Tantivy arama mikroservisi.
#
# İndeks diskte tutulur (SEARCH_INDEX_DIR) ve pod yeniden başlayınca
# kaybolmamalı: Kubernetes'te buraya PVC bağlanır.
#
# Derleme bağlamı: depo kökü.
#   docker build -f deploy/docker/search-svc.Dockerfile -t <imaj> .

FROM rust:1-bookworm AS derle
WORKDIR /src

COPY rust/search-svc/Cargo.toml rust/search-svc/Cargo.lock ./
RUN mkdir -p src && echo 'fn main(){}' > src/main.rs \
 && cargo build --release \
 && rm -rf src

COPY rust/search-svc/src ./src
RUN touch src/main.rs && cargo build --release

FROM debian:bookworm-slim
RUN apt-get update \
 && apt-get upgrade -y \
 && apt-get install -y --no-install-recommends ca-certificates \
 && rm -rf /var/lib/apt/lists/* \
 && useradd -u 10003 -r -s /usr/sbin/nologin searchsvc \
 && mkdir -p /data/search-index \
 && chown -R searchsvc:searchsvc /data

COPY --from=derle /src/target/release/search-svc /usr/local/bin/search-svc

ENV SEARCH_PORT=8171 \
    SEARCH_BIND=0.0.0.0 \
    SEARCH_INDEX_DIR=/data/search-index

EXPOSE 8171
USER searchsvc
ENTRYPOINT ["/usr/local/bin/search-svc"]
