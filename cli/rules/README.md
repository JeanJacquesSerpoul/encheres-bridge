# Règles d'enchères

`sef_rules.yaml` contient toutes les règles d'enchères de l'application : les 1 189 règles du SEF 2024 une fois les modèles `for:` expansés. Le moteur d'enchères (`../bids.wasm`) ne contient **aucune** règle de bridge. À chaque chargement de la page, `../bids-wasm.js` télécharge ce fichier et le transmet au moteur.

**Pour modifier une règle**, on édite ce fichier puis on recharge la page. Il n'y a rien à recompiler.

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
python tools/python_tools/sef_rules.py cli/rules/sef_rules.yaml --validate
```

## Garder les tests à jour

Les tests du moteur (`go test ./...`) le comparent à la référence Python sur des données générées depuis ce fichier. Après une modification, on régénère ces données depuis `tools/python_tools/` :

```bash
cd tools/python_tools
python sef_rules.py ../../cli/rules/sef_rules.yaml --json sef_rules.json
python sef_rules.py ../../cli/rules/sef_rules.yaml --gen-tests 1500 sef_tests.json --seed 2024
python gen_golden.py
cd ../.. && go test ./engine
```

Le benchmark du par (`TestParBenchmark`) échoue si la modification éloigne les contrats du par. Si cette dégradation est voulue, on l'accepte avec :

```bash
PAR_BENCH_UPDATE=1 go test -run TestParBenchmark ./engine
```
