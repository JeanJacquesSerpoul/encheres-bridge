# Exécutables du serveur

Ce dossier reçoit les binaires produits par
[build-server.sh](../build-server.sh) (Linux, WSL, macOS) et
[build-server.ps1](../build-server.ps1) (Windows) :

| Fichier | Cible |
|---------|-------|
| `bids-linux` | Linux (`amd64` par défaut) |
| `bids-windows.exe` | Windows (`amd64` par défaut) |

Les binaires eux-mêmes ne sont pas versionnés : ils se recompilent en une
commande et pèsent une douzaine de Mo pièce — le client de test y est compilé,
moteur d'enchères en WebAssembly compris.

## Fabriquer les exécutables

```bash
./build-server.sh                 # les deux, en amd64
./build-server.sh -o linux        # seulement Linux
./build-server.sh -a arm64        # pour un Raspberry Pi, un Mac ARM...
```

```powershell
.\build-server.ps1
.\build-server.ps1 -Targets windows
.\build-server.ps1 -Arch arm64
```

Les deux scripts commencent par appeler [build-wasm.sh](../build-wasm.sh) /
[build-wasm.ps1](../build-wasm.ps1), qui compilent le moteur d'enchères en
WebAssembly dans `cli/` (`bids.wasm` et `wasm_exec.js`, versionnés et tenus
à jour sur `main` par le workflow wasm.yml). L'ordre n'est pas négociable : `//go:embed` fige le contenu de `cli/`
au moment où le serveur est compilé, le moteur doit donc exister avant. Sans
lui, le binaire servirait un client incapable de calculer les enchères tout
seul — d'où le refus de compiler quand il manque.

```bash
./build-wasm.sh                   # le moteur WebAssembly seul
./build-server.sh -w              # ne pas le recompiler (déjà à jour)
```

```powershell
.\build-wasm.ps1
.\build-server.ps1 -NoWasm
```

## Utiliser l'application

1. **Lancer l'exécutable de votre système.** Il écoute sur le port 9015 et n'a
   besoin de rien d'autre : le client de test est compilé dans le binaire
   (`//go:embed`), donc le fichier peut être déplacé ou copié où vous voulez.

   ```bash
   ./server/bids-linux
   ```

   ```powershell
   .\server\bids-windows.exe
   ```

2. **Ouvrir http://localhost:9015/** dans un navigateur. Le serveur sert le
   client à sa racine, et le client calcule les enchères lui-même avec le
   moteur WebAssembly compilé dans le binaire : rien à saisir ni à configurer.

Pour arrêter le serveur : `Ctrl+C` dans sa fenêtre.

### Variantes

Ouvrir [`cli/index.html`](../cli/index.html) par double-clic (`file://`) ne
suffit pas : le navigateur refuse alors de charger le moteur WebAssembly. Le
client le signale et fait reparaître la barre du serveur ; choisir **Local**
fait alors calculer les enchères par le serveur lancé à l'étape 1.

Servi par le binaire, le client calcule toujours sur place et n'affiche pas de
barre de serveur. Pour l'y ramener — viser la production, un autre port,
comparer les deux chemins — ouvrez-le avec **`?serveur=1`** :

```
http://localhost:9015/?serveur=1
```

Le choix fait dans cette barre n'est repris qu'avec `?serveur=1` : sans le
paramètre, le client revient au calcul dans la page. « `?serveur=0` » referme
la barre.

Si 9015 est déjà pris, lancez-le sur un autre port : **http://localhost:9415/**
sert alors le même client, qui n'a rien à savoir de ce port puisqu'il calcule
sur place.

```bash
PORT=9415 ./server/bids-linux
```

```powershell
$env:PORT="9415"; .\server\bids-windows.exe
```

Ouvert en `file://`, le client ne peut pas deviner ce port : dans la barre du
serveur — qu'il affiche, faute de moteur — sélectionnez **Local** et corrigez
l'URL en `http://localhost:9415`.

Les autres variables d'environnement (`CORS_ORIGINS`, `SERVE_CLI`, `LOG_LEVEL`,
`LOG_FORMAT`) sont décrites dans le [README](../README.md).

### Windows

Le binaire n'est pas signé : au premier lancement, SmartScreen peut afficher
« Windows a protégé votre ordinateur ». *Informations complémentaires* →
*Exécuter quand même*. Le pare-feu peut aussi demander l'autorisation d'ouvrir
le port ; l'accès **réseau privé** suffit.
