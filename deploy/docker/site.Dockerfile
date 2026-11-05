# theblock Go web/API — çok aşamalı derleme.
#
# Şablonlar, statik dosyalar ve migration'lar binary'ye gömülü
# (`//go:embed templates static` ve `migrations/*.sql`), yani çalışma
# imajına kopyalanacak ek dosya yok.
#
# Çalışma tabanı neden distroless değil: internal/transcribe/ffmpeg.go
# ffprobe/ffmpeg'i alt süreç olarak çağırıyor (röportaj transkripsiyonu).
# ASSEMBLYAI_API_KEY dolu olduğu için bu yol canlıda etkin.
#
# Derleme bağlamı: depo kökü.
#   docker build -f deploy/docker/site.Dockerfile -t <imaj> .

FROM golang:1.27-bookworm AS derle
WORKDIR /src

# Bağımlılıklar ayrı katmanda: kaynak değişince yeniden indirilmez.
COPY go/go.mod go/go.sum ./
RUN go mod download

COPY go/ ./
# -trimpath: yerel yollar imaja sızmasın. -s -w: sembol tablosu atılır.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/site    ./cmd/site \
 && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate

FROM debian:bookworm-slim
RUN apt-get update \
 && apt-get upgrade -y \
 && apt-get install -y --no-install-recommends ffmpeg ca-certificates tzdata \
 && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=derle /out/site    /app/site
COPY --from=derle /out/migrate /app/migrate

# Medya PVC'si /data altına bağlanır; storage-private kardeş dizin olur
# (bkz. internal/platform/storage.go: filepath.Dir(base)+"/storage-private").
ENV PORT=8070 \
    STORAGE_DIR=/data/storage \
    PRIVATE_STORAGE_DIR=/data/storage-private \
    TZ=Europe/Istanbul

RUN useradd -u 10001 -r -s /usr/sbin/nologin theblock \
 && mkdir -p /data/storage /data/storage-private \
 && chown -R theblock:theblock /data

EXPOSE 8070
USER theblock
ENTRYPOINT ["/app/site"]
