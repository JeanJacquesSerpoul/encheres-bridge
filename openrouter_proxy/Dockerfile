# ── Stage 1: Build ──────────────────────────────────────────────────────────
FROM golang:1.22-alpine AS builder

WORKDIR /src

# Copy dependency manifests first to leverage layer cache
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Compile statically for Linux/amd64
RUN mkdir /out && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags "-s -w" -o /out/server_ai .

# ── Stage 2: Runtime ────────────────────────────────────────────────────────
FROM alpine:3.19

# ca-certificates: required for TLS calls to openrouter.ai
RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /out/server_ai .

# Run as a non-root user, named by UID so that runtimes enforcing
# runAsNonRoot can check it
RUN addgroup -S -g 10001 aiserver && adduser -S -u 10001 -G aiserver aiserver
USER 10001:10001

EXPOSE 9009

ENTRYPOINT ["/app/server_ai"]
