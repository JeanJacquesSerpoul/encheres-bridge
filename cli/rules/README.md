# Règles d'enchères

`default.yaml` contient toutes les règles d'enchères de l'application : les 1 189 règles du SEF 2024 une fois les modèles `for:` expansés. Le moteur d'enchères (`../bids.wasm`) ne contient **aucune** règle de bridge. À chaque chargement de la page, `../bids-wasm.js` télécharge ce fichier et le transmet au moteur.

**Pour modifier une règle**, on édite ce fichier puis on recharge la page. Il n'y a rien à recompiler.

## Plusieurs systèmes

Le dossier peut contenir plusieurs fichiers de règles. Chacun est un système d'enchères que l'on choisit dans **Réglages › Système**. Le choix est mémorisé dans le navigateur, et `default.yaml` est utilisé tant que rien d'autre n'a été choisi.

Un site statique ne sait pas lister un dossier : les systèmes proposés sont donc ceux que déclare `index.json`. Pour en ajouter un :

1. copier `default.yaml` sous un autre nom, par exemple `mon-systeme.yaml`. Seuls les lettres, chiffres, `.`, `-` et `_` sont permis, avec l'extension `.yaml` ou `.yml` ;
2. le modifier ;
3. l'ajouter à `index.json`, avec son nom dans les deux langues :

```json
{
  "systems": [
    { "file": "default.yaml", "name": { "fr": "SEF 2024", "en": "SEF 2024" } },
    { "file": "mon-systeme.yaml", "name": { "fr": "Mon système", "en": "My system" } }
  ]
}
```

4. produire ses données de test et ses PDF (voir [Garder les tests à jour](#garder-les-tests-à-jour)) :

```bash
cd tools/python_tools
python regen_system.py ../../cli/rules/mon-systeme.yaml
```

Si un système mémorisé disparaît de `index.json`, la page revient à `default.yaml`.

Les tests valables pour tout système tournent une fois par système déclaré, chacun sur ses propres données : la conformité à la référence Python, les enchères complètes, la légalité des enchères, le banc du par et les PDF. Les autres tests du moteur décrivent le SEF lui-même (telle main ouvre de 1SA…) et ne portent que sur `default.yaml`.

## Description en PDF

Le bouton **?** à côté de la liste des systèmes ouvre la description du système choisi : un PDF lisible, règle par règle, section par section, avec les conditions traduites en clair. Il est produit à partir du fichier de règles, et `index.json` le déclare par système, dans chaque langue :

```json
{ "file": "mon-systeme.yaml", "name": { "fr": "Mon système", "en": "My system" },
  "pdf": { "fr": "mon-systeme.pdf", "en": "mon-systeme.en.pdf" } }
```

Pour le produire ou le mettre à jour, sans dépendance à installer :

```bash
cd tools/python_tools
python rules_pdf.py ../../cli/rules/mon-systeme.yaml --title "Mon système"
python rules_pdf.py ../../cli/rules/mon-systeme.yaml --title "Mon système" --lang EN
```

Les titres de section et de sous-section du fichier, ainsi que chaque ligne de commentaire, ont leur traduction anglaise sur la ligne suivante, `# en: …` : le PDF anglais l'utilise à la place du texte français. Un nouveau titre ou un nouveau commentaire prend la même convention.

Le PDF porte l'empreinte du fichier de règles dont il est issu. `go test ./engine` échoue quand un PDF déclaré ne correspond plus à ses règles : il faut alors le régénérer.

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

Si le fichier est invalide, par exemple avec un champ manquant, une enchère mal écrite, un nom inconnu dans une condition ou un id en double, le moteur le refuse en entier. Le pied de page de l'application liste alors chaque erreur avec l'id de la règle concernée, et aucune enchère n'est calculée.

Avant de publier une modification, on peut valider le fichier depuis la racine du dépôt :

```bash
python tools/python_tools/sef_rules.py cli/rules/default.yaml --validate
```

## Garder les tests à jour

Les tests du moteur (`go test ./...`) le comparent à la référence Python sur des données générées depuis chaque fichier de règles. Après une modification, on régénère ces données, et les PDF, depuis `tools/python_tools/` :

```bash
cd tools/python_tools
python regen_system.py ../../cli/rules/default.yaml
cd ../.. && go test ./...
```

Sans argument, `regen_system.py` traite tous les systèmes de `index.json`. Il valide le fichier, puis écrit :

| Fichier | `default.yaml` | `mon-systeme.yaml` |
|---|---|---|
| Règles expansées | `tools/python_tools/sef_rules.json` | `tools/python_tools/sef_rules.mon-systeme.json` |
| Cas de test | `tools/python_tools/sef_tests.json` | `tools/python_tools/sef_tests.mon-systeme.json` |
| Enchères complètes | `engine/testdata/golden_python.json` | `engine/testdata/golden_python.mon-systeme.json` |
| Référence du banc du par | `engine/testdata/par_bench_baseline.json` | `engine/testdata/par_bench_baseline.mon-systeme.json` |
| PDF | ceux que déclare `index.json` | ceux que déclare `index.json` |

Chaque système a sa propre référence du banc du par : deux systèmes enchérissent différemment, et aucun n'est une régression de l'autre. Le script crée la référence d'un nouveau système, mais ne réécrit jamais une référence existante. Si une modification éloigne les contrats du par, `TestParBenchmark` échoue pour ce système. Une dégradation voulue s'accepte avec :

```bash
PAR_BENCH_UPDATE=1 go test -run TestParBenchmark/mon-systeme ./engine
```

ou `python regen_system.py ../../cli/rules/mon-systeme.yaml --par-update`.
