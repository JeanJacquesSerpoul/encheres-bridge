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
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /out/server_ai .

# Run as non-root user
RUN addgroup -S aiserver && adduser -S aiserver -G aiserver
USER aiserver

EXPOSE 9009

ENTRYPOINT ["/app/server_ai"]
