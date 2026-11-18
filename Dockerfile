# syntax=docker/dockerfile:1
FROM --platform=$BUILDPLATFORM oven/bun:1-alpine AS ui-builder
WORKDIR /app/ui
COPY ui/package.json ui/bun.lock ./
RUN bun install --frozen-lockfile
COPY ui/ ./
RUN bun run build

FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS go-builder
ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=ui-builder /app/ui/build/client ./ui/build/client
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build -trimpath -ldflags="-s -w -X main.Version=${VERSION}" -o /opus ./cmd/opus

FROM debian:bookworm-slim

ARG TARGETARCH
ARG GH_VERSION=2.105.0
ARG GH_SHA256_AMD64=79e7ab5db738c48777b696e374f8b428f0647debfb424f44d2f426370407fe2f
ARG GH_SHA256_ARM64=b1278c75107423d818f1937fd92cafc9e5e9b606631568f68d7fb816f5885a02

ARG AGY_VERSION=1.5.11-6132547114106880
ARG AGY_SHA512_AMD64=dad3c015e8964fcdc1639b13b7152f4cb84cb1bde385707e2bcdf095d6f1831f4f6521199aec499cc052f9e99906b7fdbc52c273df064d5cb7c84b068ba3ce50
ARG AGY_SHA512_ARM64=5f703c88e01851061c87c07b1a4ca54c1f5a6826ffada9fb005974a099e9a552ddb1f9ba2fe1194e2829803482c86084babf8cf91530a528bc7f659613600585

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates git curl \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd --gid 65532 opus \
    && useradd --uid 65532 --gid 65532 --home-dir /data --shell /usr/sbin/nologin --no-create-home opus \
    && mkdir -p /data \
    && chown 65532:65532 /data

RUN set -eu; \
    arch="${TARGETARCH:-amd64}"; \
    case "$arch" in \
      amd64) gh_sha="$GH_SHA256_AMD64" ;; \
      arm64) gh_sha="$GH_SHA256_ARM64" ;; \
      *) echo "Unsupported arch: $arch" >&2; exit 1 ;; \
    esac; \
    curl -fsSL "https://github.com/cli/cli/releases/download/v${GH_VERSION}/gh_${GH_VERSION}_linux_${arch}.tar.gz" -o /tmp/gh.tar.gz; \
    echo "${gh_sha}  /tmp/gh.tar.gz" | sha256sum -c -; \
    tar -xzf /tmp/gh.tar.gz -C /tmp "gh_${GH_VERSION}_linux_${arch}/bin/gh"; \
    install -m 0755 "/tmp/gh_${GH_VERSION}_linux_${arch}/bin/gh" /usr/local/bin/gh; \
    rm -rf /tmp/gh.tar.gz "/tmp/gh_${GH_VERSION}_linux_${arch}"

RUN set -eu; \
    arch="${TARGETARCH:-amd64}"; \
    case "$arch" in \
      amd64) agy_dir="linux-x64"; agy_file="cli_linux_x64.tar.gz"; agy_sha="$AGY_SHA512_AMD64" ;; \
      arm64) agy_dir="linux-arm"; agy_file="cli_linux_arm64.tar.gz"; agy_sha="$AGY_SHA512_ARM64" ;; \
      *) echo "Unsupported arch: $arch" >&2; exit 1 ;; \
    esac; \
    curl -fsSL "https://storage.googleapis.com/antigravity-public/antigravity-cli/${AGY_VERSION}/${agy_dir}/${agy_file}" -o /tmp/agy.tar.gz; \
    echo "${agy_sha}  /tmp/agy.tar.gz" | sha512sum -c -; \
    tar -xzf /tmp/agy.tar.gz -C /tmp antigravity; \
    install -m 0755 /tmp/antigravity /usr/local/bin/agy; \
    rm -f /tmp/agy.tar.gz /tmp/antigravity

COPY --from=go-builder /opus /usr/local/bin/opus

ENV HOME=/data \
    APP_HOME=/data \
    APP_DATABASE_URL="file:/data/opus.db?_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)"

WORKDIR /data
VOLUME /data
EXPOSE 7880
USER 65532:65532

# TODO: switch to /readyz once implemented
HEALTHCHECK --interval=30s --timeout=3s --start-period=15s \
    CMD curl -fsS http://127.0.0.1:7880/api/health || exit 1

ENTRYPOINT ["/usr/local/bin/opus"]
CMD ["serve"]
