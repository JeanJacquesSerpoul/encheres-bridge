# syntax=docker/dockerfile:1
# Image de production autonome : le dossier prod/ contient le binaire Linux
# pré-compilé, aucune étape de compilation Go. Le client de test cli/ est
# compilé dans le binaire (//go:embed) mais désactivé par défaut ici via
# SERVE_CLI=false (voir docker-compose.yml) : seuls /health, /ready,
# /version, /bid et /bids sont exposés.
#
# Préparer le dossier depuis le poste de développement (Windows) :
#   .\build-prod.ps1
# puis recopier prod/ sur le serveur Linux et y lancer :
#   docker compose up -d --build

FROM alpine:3.21
RUN adduser -D -H app
WORKDIR /app
# --chmod : le bit d'exécution est perdu quand le binaire transite par Windows.
COPY --chmod=0755 bids-linux ./bids
USER app
ENV PORT=9015
ENV SERVE_CLI=false
EXPOSE 9015
HEALTHCHECK --interval=30s --timeout=3s --start-period=2s --retries=3 \
  CMD wget -qO- "http://localhost:${PORT}/ready" || exit 1
ENTRYPOINT ["./bids"]
