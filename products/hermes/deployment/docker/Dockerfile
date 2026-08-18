FROM golang:1.26-bookworm AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TIME=unknown
RUN CGO_ENABLED=1 go test ./... && \
    go build -trimpath -ldflags "-s -w -X github.com/mca-rolando/HermesDDNS/internal/buildinfo.Version=${VERSION} -X github.com/mca-rolando/HermesDDNS/internal/buildinfo.Commit=${COMMIT} -X github.com/mca-rolando/HermesDDNS/internal/buildinfo.BuildTime=${BUILD_TIME}" -o /out/hermesddns ./cmd/hermesddns && \
    go build -trimpath -ldflags "-s -w -X github.com/mca-rolando/HermesDDNS/internal/buildinfo.Version=${VERSION} -X github.com/mca-rolando/HermesDDNS/internal/buildinfo.Commit=${COMMIT} -X github.com/mca-rolando/HermesDDNS/internal/buildinfo.BuildTime=${BUILD_TIME}" -o /out/hermesctl ./cmd/hermesctl

FROM debian:12-slim
RUN DEBIAN_FRONTEND=noninteractive apt-get update && \
    apt-get install -y --no-install-recommends bind9 bind9-utils dnsutils ca-certificates curl && \
    rm -rf /var/lib/apt/lists/*
COPY --from=builder /out/hermesddns /usr/local/bin/hermesddns
COPY --from=builder /out/hermesctl /usr/local/bin/hermesctl
COPY deployment/docker/setup-bind.sh /usr/local/lib/hermesddns/setup-bind.sh
RUN chmod +x /usr/local/lib/hermesddns/setup-bind.sh && mkdir -p /opt/hermesddns/data /opt/hermesddns/config /opt/hermesddns/backups /var/cache/bind
WORKDIR /opt/hermesddns
EXPOSE 53/tcp 53/udp 8080/tcp
HEALTHCHECK --interval=30s --timeout=5s --retries=3 CMD curl -fsS http://127.0.0.1:8080/health || exit 1
CMD ["/bin/sh", "-c", "/usr/local/lib/hermesddns/setup-bind.sh && service named start && exec /usr/local/bin/hermesddns"]
