# theblock collab — Hocuspocus/Yjs ortak düzenleme servisi (Node).
#
# Go tarafına HTTP ile belge okuyup yazar (GO_URL) ve jetonu COLLAB_SECRET
# ile doğrular; ikisi de ortam değişkeni olarak verilir.
#
# Derleme bağlamı: depo kökü.
#   docker build -f deploy/docker/collab.Dockerfile -t <imaj> .

FROM node:22-bookworm-slim AS bagimlilik
WORKDIR /app
COPY collab-server/package.json collab-server/package-lock.json ./
# npm ci: package-lock.json'a birebir uyar, üretim bağımlılıkları yeter.
RUN npm ci --omit=dev

FROM node:22-bookworm-slim

# Güvenlik yaması + çalışma zamanında gerekmeyen paket yöneticilerini atma.
#
# apt: CVE-2026-109518 (2.6.1 → 2.6.2, bookworm-security'de mevcut).
#
# npm/npx/corepack/yarn: bağımlılıklar yukarıdaki `bagimlilik` aşamasından
# KOPYALANIYOR; çalışma imajında paket yöneticisi hiç çağrılmıyor. Silinince
# npm'in kendi bağımlılıklarındaki CVE'ler (brace-expansion, ip-address,
# pacote, picomatch, sigstore …) imajdan tümden çıkar — bastırılmaz, yok olur.
# Üretim konteynerinde paket yöneticisi bulunmaması ayrıca saldırı yüzeyini
# küçültür: ele geçirilen süreç `npm i <kötü-paket>` çalıştıramaz.
RUN apt-get update \
 && apt-get upgrade -y \
 && rm -rf /var/lib/apt/lists/* \
 && rm -rf /usr/local/lib/node_modules/npm \
           /usr/local/lib/node_modules/corepack \
           /usr/local/bin/npm \
           /usr/local/bin/npx \
           /usr/local/bin/corepack \
           /usr/local/bin/yarn \
           /usr/local/bin/yarnpkg

WORKDIR /app
ENV NODE_ENV=production \
    PORT=1234 \
    GO_URL=http://theblock-site:8070

COPY --from=bagimlilik /app/node_modules ./node_modules
COPY collab-server/package.json ./
COPY collab-server/server.js collab-server/schema.js ./

EXPOSE 1234
# node imajının hazır "node" kullanıcısı (uid 1000) — root çalıştırmıyoruz.
USER node
CMD ["node", "server.js"]
