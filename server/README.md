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

2. **Ouvrir [`cli/index.html`](../cli/index.html)** dans un navigateur, par
   simple double-clic. Ouvert de cette façon, le client vise **Local** par
   défaut, c'est-à-dire `http://localhost:9015` : rien à saisir ni à
   configurer tant que le serveur écoute sur ce port.

Pour arrêter le serveur : `Ctrl+C` dans sa fenêtre.

### Variantes

Le serveur sert aussi le client à sa racine : **http://localhost:9015/** affiche
la même application sans passer par le fichier local — et, là, elle calcule les
enchères elle-même, avec le moteur WebAssembly servi par le binaire : c'est le
mode **Navigateur (hors ligne)**, choisi par défaut. Le fichier ouvert en
`file://`, lui, n'a pas accès à ce moteur et s'en tient au serveur.

N'ayant alors aucun serveur à choisir, le client n'affiche plus de barre de
serveur dans son en-tête. Pour l'y ramener — viser la production, un autre
port, comparer les deux chemins — ouvrez-le avec **`?serveur=1`** :

```
http://localhost:9015/?serveur=1
```

Le mode sélectionné est mémorisé comme d'habitude : les visites suivantes se
passent du paramètre, la barre restant visible tant que le mode n'est pas
revenu à « Navigateur ». « `?serveur=0` » la referme.

Si 9015 est déjà pris, lancez-le sur un autre port : **http://localhost:9415/**
sert alors le même client, qui n'a rien à savoir de ce port puisqu'il calcule
sur place.

```bash
PORT=9415 ./server/bids-linux
```

```powershell
$env:PORT="9415"; .\server\bids-windows.exe
```

Ouvert en `file://`, le client ne peut pas deviner ce port : sélectionnez alors
**Distant** dans l'en-tête — qu'il affiche, faute de moteur — et saisissez
`http://localhost:9415`. Servi par le binaire, c'est `?serveur=1` qui ramène
ce choix.

Les autres variables d'environnement (`CORS_ORIGINS`, `SERVE_CLI`, `LOG_LEVEL`,
`LOG_FORMAT`) sont décrites dans le [README](../README.md).

### Windows

Le binaire n'est pas signé : au premier lancement, SmartScreen peut afficher
« Windows a protégé votre ordinateur ». *Informations complémentaires* →
*Exécuter quand même*. Le pare-feu peut aussi demander l'autorisation d'ouvrir
le port ; l'accès **réseau privé** suffit.
