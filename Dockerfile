# Single-arch (linux/amd64 by default) build of the private fork.
# Runtime layout matches the upstream "lite" image: /config, /media, port 19035,
# PUID/PGID/PERMS/UMASK/TZ handled by docker/lite-entrypoint.sh.

FROM node:22-alpine AS frontend
WORKDIR /src/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY frontend/ ./
RUN npm run build

FROM golang:1.26-alpine AS backend
RUN apk add --no-cache gcc g++ musl-dev
ENV CGO_ENABLED=1 \
    CGO_CFLAGS="-D_LARGEFILE64_SOURCE"
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /src/frontend/dist/spa ./frontend/dist/spa
ARG VERSION=dev
RUN go build \
      -ldflags="-s -w -linkmode external -extldflags '-static' -X main.AppVersion=${VERSION} -X main.LiteMode=true" \
      -o /out/chinesesubfinder ./cmd/chinesesubfinder

FROM alpine:3.22
ENV TZ=Asia/Shanghai \
    PERMS=true \
    PUID=1026 \
    PGID=100 \
    UMASK=022 \
    PS1="\u@\h:\w \$ "
RUN apk add --no-cache bash ffmpeg ca-certificates tini su-exec tzdata \
    && ln -sf /usr/share/zoneinfo/${TZ} /etc/localtime \
    && echo "${TZ}" > /etc/timezone
COPY --from=backend /out/chinesesubfinder /usr/bin/chinesesubfinder
COPY docker/lite-entrypoint.sh /usr/bin/entrypoint.sh
RUN chmod +x /usr/bin/entrypoint.sh
VOLUME ["/config", "/media"]
WORKDIR /config
EXPOSE 19035
# /system-status needs no login. The port can be changed with a /config/CustomPort file.
# -Y off: do not send the check through an http_proxy set on the container.
# With PERMS=true the app only starts after the ownership pass over /media, so a first
# start on a large library can show "unhealthy" for a while; nothing acts on that.
HEALTHCHECK --interval=30s --timeout=5s --start-period=60s --retries=3 \
    CMD wget -q -Y off -O /dev/null "http://127.0.0.1:$(cat /config/CustomPort 2>/dev/null || echo 19035)/system-status" || exit 1
ENTRYPOINT ["tini", "entrypoint.sh"]
