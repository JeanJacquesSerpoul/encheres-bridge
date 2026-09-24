# Serveur IA — reconnaissance des cartes

Ce dossier contient le serveur Go réduit à **un seul fournisseur : OpenRouter**. Il expose l'API que le client
appelle pour lire les cartes sur une photo (voir la section « Reconnaissance des
cartes par photo » du [README](../README.md)) et garde la clé d'API hors du
navigateur : le client, lui, n'envoie qu'une image et un prompt.


## Démarrage

```bash
cp .env.example .env      # puis renseigner OPENROUTER_API_KEY
go run .                  # écoute sur http://localhost:9013
```

```bash
docker compose up --build -d
```

Sans Go ni Docker, les exécutables précompilés de [bin/](bin/) suffisent
(Linux et Windows, amd64). Après avoir créé `.env` comme ci-dessus :

```bash
bin/server_ai.sh            # Linux (ou Git Bash sous Windows) ; -p <port> pour changer de port
```

```powershell
.\bin\server_ai.ps1         # Windows ; -Port <port> pour changer de port
```

Les scripts se placent dans ce dossier pour que `.env` soit lu. Après une
modification des sources, `make bin` recompile les deux exécutables.

Le client vise ce port par défaut : dans l'en-tête, **Serveur IA → Local** vaut
`http://localhost:9013`. Tant que `/health` répond, les boutons appareil photo
s'allument.

## Configuration

Tout passe par l'environnement ; `.env` est lu au démarrage s'il existe.

| Variable             | Défaut                          | Obligatoire | Description |
|----------------------|---------------------------------|-------------|-------------|
| `OPENROUTER_API_KEY` | —                               | **Oui**     | Clé d'API OpenRouter |
| `DEFAULT_MODEL`      | `google/gemini-3.1-flash-lite`  | Non         | Modèle interrogé quand la requête n'en nomme pas |
| `PORT`               | `9013`                          | Non         | Port d'écoute |
| `MAX_TOKENS`         | `10000`                         | Non         | Plafond de tokens par réponse |
| `TIMEOUT`            | `60`                            | Non         | Délai d'attente d'OpenRouter, en secondes |
| `CORS_ORIGINS`       | `*`                             | Non         | Origines autorisées, séparées par des virgules (`CORS_ALLOWED_ORIGINS`, le nom d'aiproxy, est accepté aussi) |
| `ALLOWED_MODELS`     | —                               | Non         | Modèles acceptés en plus de `DEFAULT_MODEL`, séparés par des virgules |
| `RATE_LIMIT_PER_MIN` | `20`                            | Non         | Requêtes `/api/*` par IP et par minute ; `0` désactive la limite |
| `RATE_LIMIT_BURST`   | `5`                             | Non         | Rafale tolérée au-delà du débit |
| `TRUST_PROXY`        | `false`                         | Non         | Croire `X-Forwarded-For` / `X-Real-IP` (seulement derrière un reverse proxy) |
| `ENABLE_MODELS_ENDPOINT` | `false`                     | Non         | Exposer `GET /api/models` |
| `BIND_ADDR`          | `127.0.0.1`                     | Non         | Interface où `docker compose` publie le port |

Le modèle par défaut est un modèle de vision « flash » : il lit des cartes en
quelques secondes, et c'est celui que le client demande explicitement
(`IA_MODEL` dans [cli/app.js](../cli/app.js)). Le changer ici suffit à servir un
autre modèle aux requêtes qui n'en précisent pas — celles du client, elles,
continueront de nommer le leur.

## Sécurité

Le serveur dépense la clé d'OpenRouter pour quiconque le joint. Pour éviter
qu'il serve de relais ouvert :

- **Modèles** : seuls `DEFAULT_MODEL` et ceux de `ALLOWED_MODELS` sont
  acceptés. Un modèle hors liste reçoit un `400`.
- **Débit** : chaque IP a droit à `RATE_LIMIT_PER_MIN` requêtes `/api/*` par
  minute, avec une rafale de `RATE_LIMIT_BURST`. Au-delà, le serveur répond
  `429` avec `Retry-After`. `/health` n'est pas limité.
- **Entrées** : l'image doit être un data-URI base64 (jpeg, png, webp ou gif).
  Les URL distantes sont refusées, pour qu'on ne puisse pas faire récupérer
  n'importe quelle adresse par OpenRouter. Le PDF doit être un data-URI
  `application/pdf`, le prompt fait 16 Kio au plus et le corps 10 Mio au plus.
- **CORS** : `*` par défaut pour l'usage local, et le serveur le signale au
  démarrage. En production, mettez l'origine du client dans `CORS_ORIGINS`.
- **Reverse proxy** : derrière Caddy, mettez `TRUST_PROXY=true` pour que la
  limite porte sur l'IP réelle du client. Sans proxy, laissez `false` : sinon
  un appelant choisirait lui-même l'IP qu'on limite.
- **Docker** : le port n'est publié que sur `127.0.0.1`, et le conteneur tourne
  en lecture seule, sans capacités, sous un UID non root.

## API

### `GET /health`

```json
{"status": "ok"}
```

Sondé par le client avant d'allumer les boutons photo, et par Docker.

### `POST /api/chat`

Corps `application/json`, 10 Mio au plus (la photo y voyage en base64) :

| Champ      | Type   | Obligatoire | Description |
|------------|--------|-------------|-------------|
| `text`     | string | **Oui**     | Le prompt (16 Kio au plus) |
| `model`    | string | Non         | ID du modèle, parmi `DEFAULT_MODEL` et `ALLOWED_MODELS` ; `DEFAULT_MODEL` à défaut |
| `image`    | string | Non         | Data-URI base64 (`data:image/jpeg;base64,...`) ; les URL sont refusées |
| `pdf`      | string | Non         | PDF en data-URI base64 |
| `provider` | string | Non         | `OPENROUTER` uniquement ; toléré pour les clients venus d'aiproxy |

```bash
curl -X POST http://localhost:9013/api/chat \
  -H "Content-Type: application/json" \
  -d '{"text": "Combien de cartes vois-tu ?", "image": "data:image/jpeg;base64,..."}'
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
`400` requête invalide (JSON, `Content-Type`, `text` manquant ou trop long,
modèle non autorisé, image qui n'est pas un data-URI, fournisseur inconnu),
`413` corps trop gros, `429` limite de débit atteinte, `502` erreur
d'OpenRouter, `504` dépassement de `TIMEOUT`. Le détail
de l'erreur amont est journalisé mais pas renvoyé : il nommerait la clé, le
compte ou le modèle.

### `GET /api/models`

Liste les modèles d'OpenRouter — utile pour vérifier l'identifiant exact d'un
modèle avant de le mettre dans `DEFAULT_MODEL`. La route n'existe que si
`ENABLE_MODELS_ENDPOINT=true` (sinon `404`).

```bash
curl "http://localhost:9013/api/models" | head
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
    reverse_proxy localhost:9013
}
```

### À faire en production

Derrière Caddy, deux variables du `.env` du serveur doivent quitter leur valeur
par défaut :

```dotenv
TRUST_PROXY=true
CORS_ORIGINS=https://<domaine>
```

- **`TRUST_PROXY=true`** : vu du serveur, toutes les requêtes viennent de Caddy
  (`127.0.0.1`). Sans cette variable, les visiteurs partagent tous la même
  limite de débit, et le premier qui l'épuise bloque les autres. Avec elle, le
  serveur lit l'IP réelle dans `X-Forwarded-For`, que Caddy réécrit. Laissez-la
  à `false` si le serveur est joignable sans passer par Caddy : un appelant
  pourrait sinon choisir lui-même l'IP qu'on limite.
- **`CORS_ORIGINS`** : l'origine du client, c'est-à-dire schéma et domaine,
  sans chemin ni `/` final (`https://<domaine>`, même si le client est servi
  sous `/bridgequizz/`). Plusieurs origines se séparent par des virgules. Avec
  `*`, la valeur par défaut, n'importe quelle page web peut appeler le serveur
  depuis le navigateur de ses visiteurs, et le serveur le signale au démarrage.

Redémarrez ensuite le serveur (`docker compose up -d`). Le journal de
démarrage ne doit plus afficher l'avertissement sur `CORS_ORIGINS`.

Le serveur d'IA reste facultatif : sans lui, seuls les boutons photo du client
s'éteignent.
