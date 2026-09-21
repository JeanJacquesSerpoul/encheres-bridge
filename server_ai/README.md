# Serveur IA — reconnaissance des cartes

Ce dossier contient le serveur Go réduit à **un seul fournisseur : OpenRouter**. Il expose l'API que le client
appelle pour lire les cartes sur une photo (voir la section « Reconnaissance des
cartes par photo » du [README](../README.md)) et garde la clé d'API hors du
navigateur : le client, lui, n'envoie qu'une image et un prompt.


## Démarrage

```bash
cp .env.example .env      # puis renseigner OPENROUTER_API_KEY
go run .                  # écoute sur http://localhost:9009
```

```bash
docker compose up --build -d
```

Le client vise ce port par défaut : dans l'en-tête, **Serveur IA → Local** vaut
`http://localhost:9009`. Tant que `/health` répond, les boutons appareil photo
s'allument.

## Configuration

Tout passe par l'environnement ; `.env` est lu au démarrage s'il existe.

| Variable             | Défaut                          | Obligatoire | Description |
|----------------------|---------------------------------|-------------|-------------|
| `OPENROUTER_API_KEY` | —                               | **Oui**     | Clé d'API OpenRouter |
| `DEFAULT_MODEL`      | `google/gemini-3.1-flash-lite`  | Non         | Modèle interrogé quand la requête n'en nomme pas |
| `PORT`               | `9009`                          | Non         | Port d'écoute |
| `MAX_TOKENS`         | `10000`                         | Non         | Plafond de tokens par réponse |
| `TIMEOUT`            | `60`                            | Non         | Délai d'attente d'OpenRouter, en secondes |
| `CORS_ORIGINS`       | `*`                             | Non         | Origines autorisées, séparées par des virgules (`CORS_ALLOWED_ORIGINS`, le nom d'aiproxy, est accepté aussi) |

Le modèle par défaut est un modèle de vision « flash » : il lit des cartes en
quelques secondes, et c'est celui que le client demande explicitement
(`IA_MODEL` dans [cli/app.js](../cli/app.js)). Le changer ici suffit à servir un
autre modèle aux requêtes qui n'en précisent pas — celles du client, elles,
continueront de nommer le leur.

## API

### `GET /health`

```json
{"status": "ok"}
```

Sondé par le client avant d'allumer les boutons photo, et par Docker.

### `POST /api/chat`

Corps `application/json`, 20 Mio au plus (la photo y voyage en base64) :

| Champ      | Type   | Obligatoire | Description |
|------------|--------|-------------|-------------|
| `text`     | string | **Oui**     | Le prompt |
| `model`    | string | Non         | ID du modèle ; `DEFAULT_MODEL` à défaut |
| `image`    | string | Non         | URL HTTPS ou data-URI base64 (`data:image/jpeg;base64,...`) |
| `pdf`      | string | Non         | PDF en data-URI base64 |
| `provider` | string | Non         | `OPENROUTER` uniquement ; toléré pour les clients venus d'aiproxy |

```bash
curl -X POST http://localhost:9009/api/chat \
  -H "Content-Type: application/json" \
  -d '{"text": "Combien de cartes vois-tu ?", "image": "https://example.com/main.jpg"}'
```

Réponse :

```json
{
  "success": true,
  "response": "{\"card_count\":13, ...}",
  "model": "google/gemini-3.1-flash-lite",
  "usage": { "prompt_tokens": 1520, "completion_tokens": 210, "total_tokens": 1730 }
}
```

En cas d'échec, `{"success": false, "error": "..."}` avec le statut voulu :
`400` requête invalide (JSON, `Content-Type`, `text` manquant, fournisseur
inconnu), `502` erreur d'OpenRouter, `504` dépassement de `TIMEOUT`. Le détail
de l'erreur amont est journalisé mais pas renvoyé : il nommerait la clé, le
compte ou le modèle.

### `GET /api/models`

Liste les modèles d'OpenRouter — utile pour vérifier l'identifiant exact d'un
modèle avant de le mettre dans `DEFAULT_MODEL`.

```bash
curl "http://localhost:9009/api/models" | head
```

Un `?provider=` est toléré s'il vaut `OPENROUTER`.

## Build et tests

```bash
go test ./...
make build          # dist/server_ai et dist/server_ai.exe
```

## Déploiement

Derrière un reverse proxy, le serveur se monte à côté de celui des enchères ;
le client attend alors l'URL complète dans **Serveur IA → Distant**. Avec Caddy :

```caddy
handle_path /aiproxy* {
    reverse_proxy localhost:9009
}
```

Le serveur d'IA reste facultatif : sans lui, seuls les boutons photo du client
s'éteignent.
