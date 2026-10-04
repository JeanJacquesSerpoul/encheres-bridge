# Systèmes d'enchères

Chaque système d'enchères est un dossier, que l'on choisit dans **Réglages › Système** :

```
systems/
  index.json            la liste des systèmes proposés, avec leur nom
  sef/                  le SEF 2024, système par défaut
    rules.yaml          ses règles : 1 189 une fois les modèles for: expansés
    rules.pdf           leur description en PDF (bouton ? des Réglages)
    rules.en.pdf        la même, en anglais
    pbn/                ses donnes thématiques : index.json et fichiers .pbn
  new/                  un second système, en construction
    …                   les mêmes fichiers
```

Le moteur d'enchères (`../bids.wasm`) ne contient **aucune** règle de bridge. À chaque chargement de la page, `../bids-wasm.js` télécharge le `rules.yaml` du système choisi et le transmet au moteur. Les **Donnes thématiques** et l'entraînement par thème proposent les donnes du dossier `pbn/` de ce même système.

**Pour modifier une règle**, on édite le `rules.yaml` du système puis on recharge la page. Il n'y a rien à recompiler. Avant de publier, une seule commande, depuis la racine du dépôt, met à jour tout ce qui en dérive et lance les tests :

```bash
./update-system.sh new            # .\update-system.ps1 new sous Windows
./update-system.sh -p new         # puis publie : branche, commit, push et pull request
```

`-a` (`-AcceptPar`) accepte une hausse voulue de l'écart au par, `-p` (`-Publish`) publie, `-m "…"` (`-Message`) donne le message du commit. Le détail est dans [Garder les tests à jour](#garder-les-tests-à-jour).

## Ajouter un système

Un site statique ne sait pas lister un dossier : les systèmes proposés sont ceux que déclare `index.json`. Pour en ajouter un :

1. créer son dossier, par exemple `mon-systeme/`. Son nom est l'identifiant du système : seuls les lettres, chiffres, `-` et `_` sont permis ;
2. y écrire `rules.yaml`, en partant de celui d'un autre système ou d'une seule règle ;
3. y créer `pbn/index.json`, vide au départ : `{ "files": [] }` ;
4. le déclarer dans `index.json`, avec son nom dans les deux langues :

```json
{
  "systems": [
    { "id": "sef", "name": { "fr": "SEF 2024", "en": "SEF 2024" } },
    { "id": "mon-systeme", "name": { "fr": "Mon système", "en": "My system" } }
  ]
}
```

5. produire ses PDF et ses données de test, depuis la racine du dépôt :

```bash
./update-system.sh mon-systeme    # .\update-system.ps1 mon-systeme sous Windows
```

Le choix du système est mémorisé dans le navigateur. Si un système mémorisé disparaît de `index.json`, la page revient au SEF, qui doit donc rester déclaré.

Les tests valables pour tout système tournent une fois par système déclaré, chacun sur ses propres données : la conformité à la référence Python, les enchères complètes, la légalité des enchères, le banc du par et les PDF. Les autres tests du moteur décrivent le SEF lui-même (telle main ouvre de 1SA…) et ne portent que sur `sef/rules.yaml`.

## Donnes thématiques

Le dossier `pbn/` d'un système contient ses séries de donnes thématiques, et son `index.json` les déclare (voir [Donnes thématiques](../../README.md#donnes-thématiques) dans le README du dépôt). Un thème appartient à un système : ses donnes ont été choisies parce que les enchères **de ce système** y emploient une convention donnée. [tools/python_tools/gen_theme_pbn.py](../../tools/python_tools/gen_theme_pbn.py) les produit, avec `--rules` pour désigner les règles du système :

```bash
cd tools/python_tools
python gen_theme_pbn.py '^drury\.[HS]\.(2C|2NT)$' ../../cli/systems/sef/pbn/drury.pbn \
    --rules ../../cli/systems/sef/rules.yaml --fr "Drury" --en "Drury"
```

Après une modification des règles, [tools/python_tools/regen_themes.py](../../tools/python_tools/regen_themes.py) régénère les séries d'un système à partir de leurs recettes (`python regen_themes.py sef`).

Une donne peut aussi porter son enchère : la section `[Auction "<donneur>"]` du bloc, ses appels en notation PBN standard, chacun suivi de son commentaire **entre accolades**. L'enchère, ses commentaires et le contrat sont alors lus dans le fichier, sans moteur : c'est ce que demande l'**entraînement**, qui ne tire que les donnes ainsi enrichies et n'offre pas les autres. `2-faible.pbn` est la série d'exemple ; la description du format est dans [Donnes thématiques](../../README.md#le-nouveau-format) du README du dépôt.

## Description en PDF

Le bouton **?** à côté de la liste des systèmes ouvre la description du système choisi, `rules.pdf` ou `rules.en.pdf` selon la langue : un PDF lisible, règle par règle, section par section, avec les conditions traduites en clair. `update-system` les produit à partir de `rules.yaml`, avec le nom que `index.json` donne au système.

Les titres de section et de sous-section du fichier, ainsi que chaque ligne de commentaire, ont leur traduction anglaise sur la ligne suivante, `# en: …` : le PDF anglais l'utilise à la place du texte français. Un nouveau titre ou un nouveau commentaire prend la même convention.

Le PDF porte l'empreinte du fichier de règles dont il est issu. `go test ./engine` échoue quand un PDF ne correspond plus à ses règles : il faut alors le régénérer.

## Principe

Le fichier est une **liste ordonnée**. À son tour de parole, un joueur prend la **première règle applicable** : son `option` est activée (aucune ne l'est dans l'application), son motif `seq` correspond à la séquence vue par sa paire, et sa condition `cond` est vraie sur sa main. Si aucune règle ne s'applique, ou si l'enchère de la règle est illégale, il passe, et le commentaire l'indique.

```yaml
- id: open.1NT
  seq: ""                     # début d'enchère
  call: 1NT
  cond: "15 <= hl <= 17 and balanced and not (shape == '5332' and max(S, H) == 5 and hcp == 17)"
  forcing: NF
  meaning: "15-17 HL régulier ; 5332 avec majeure 5e : 15-16 H"   # commentaire affiché (fr)
  meaning_en: "15-17 HL balanced; 5332 with 5-card major: 15-16 HCP"  # commentaire affiché (en)
  status: sef
```

- **La séquence vue par la paire** comprend ses propres enchères, passes comprises, et celles des adversaires entre parenthèses, sans leurs passes. Par exemple `1C (1S) X`.
- **Le motif `seq`** accepte :
  - `*` pour une enchère quelconque ;
  - `**` en tête pour n'importe quel début ;
  - `A|B` pour une alternative ;
  - `BW:S` en tête : n'importe quel début, à condition que l'atout convenu soit ♠.
- **La condition `cond`** est une expression qui porte sur la main :
  - points : `hcp`, `hl`, `hld` ;
  - longueurs : `S`, `H`, `D`, `C` ;
  - forme : `shape`, `balanced` ;
  - levées : `ptricks`, `qtricks` ;
  - fonctions par couleur : `stop('H')`, `keycards('S')`, etc.
- **`meaning` et `meaning_en`** sont les commentaires affichés dans la séquence commentée et dans l'arbre de décision.

La référence complète est dans [tools/python_tools/SEF_2024_spec.md](../../tools/python_tools/SEF_2024_spec.md) : champs, motifs, liste des caractéristiques de main, langage des conditions. On y trouve aussi des exemples de modification dans [tools/python_tools/README.md](../../tools/python_tools/README.md).

## Si le fichier est invalide

Si le fichier est invalide, par exemple avec un champ manquant, une enchère mal écrite, un nom inconnu dans une condition ou un id en double, le moteur le refuse en entier. Le pied de page de l'application liste alors chaque erreur avec l'id de la règle concernée, et aucune enchère n'est calculée. `update-system` le valide avant toute autre chose ; on peut aussi le valider seul, depuis la racine du dépôt :

```bash
python tools/python_tools/sef_rules.py cli/systems/sef/rules.yaml --validate
```

## Garder les tests à jour

Les tests du moteur (`go test ./...`) le comparent à la référence Python sur des données générées depuis les règles de chaque système. Après une modification, `update-system.sh` (ou `update-system.ps1`) les régénère, avec les PDF, puis lance les tests. Il appelle pour cela `tools/python_tools/regen_system.py`, que l'on peut aussi lancer seul ; sans argument, celui-ci traite tous les systèmes de `index.json`. Les règles sont validées, puis le script écrit, pour le système `<id>` :

| Fichier | Contenu |
|---|---|
| `cli/systems/<id>/rules.pdf`, `rules.en.pdf` | La description du système |
| `engine/testdata/systems/<id>/rules.json` | Les règles expansées |
| `engine/testdata/systems/<id>/tests.json` | Les cas de test : caractéristiques de main et règle choisie |
| `engine/testdata/systems/<id>/golden.json` | Des enchères complètes, les quatre mains enchérissant |
| `engine/testdata/systems/<id>/par_baseline.json` | La référence du banc du par, créée si elle manque |

Chaque système a sa propre référence du banc du par : deux systèmes enchérissent différemment, et aucun n'est une régression de l'autre. Le script crée la référence d'un nouveau système, mais ne réécrit jamais une référence existante. Si une modification éloigne les contrats du par, `TestParBenchmark` échoue pour ce système. Une dégradation voulue s'accepte avec :

```bash
./update-system.sh -a mon-systeme      # .\update-system.ps1 mon-systeme -AcceptPar
```

ou `PAR_BENCH_UPDATE=1 go test -run TestParBenchmark/mon-systeme ./engine`.
