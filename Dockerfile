# syntax=docker/dockerfile:1

# ---- build ----
FROM golang:1.26-alpine AS build
WORKDIR /src
ARG REVISION=docker
COPY go.mod ./
COPY *.go ./
COPY cli ./cli
# Le moteur en WebAssembly d'abord : //go:embed all:cli le fige dans le binaire
# à l'étape suivante, l'ordre n'est donc pas négociable. build-wasm.sh n'est pas
# utilisable ici (.dockerignore exclut *.sh), d'où les deux commandes en clair.
RUN cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" ./cli/wasm_exec.js \
 && GOOS=js GOARCH=wasm CGO_ENABLED=0 go build -trimpath \
      -ldflags="-s -w -X main.buildRevision=${REVISION}" -o ./cli/bids.wasm .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.buildRevision=${REVISION}" -o /out/bids .

# ---- runtime ----
# The cli/ test client is compiled into the binary via //go:embed, so the
# runtime stage doesn't need its own copy of cli/. Set SERVE_CLI=false to
# stop serving it (e.g. in production, see docker-compose.yml).
FROM alpine:3.21
RUN adduser -D -H app
WORKDIR /app
COPY --from=build /out/bids ./bids
USER app
ENV PORT=9015
EXPOSE 9015
HEALTHCHECK --interval=30s --timeout=3s --start-period=2s --retries=3 \
  CMD wget -qO- "http://localhost:${PORT}/ready" || exit 1
ENTRYPOINT ["./bids"]
