# engine_bids — moteur d'enchères SEF 2024

Règles d'enchères du système SEF 2024 (1 189 règles après expansion), deux moteurs qui les appliquent (Python et JavaScript), et un programme qui lit un fichier PBN et sort les enchères d'une donne en JSON.

## Fichiers

| Fichier | Rôle |
|---|---|
| [`../../cli/rules/sef_rules.yaml`](../../cli/rules/sef_rules.yaml) | **Source unique des règles** : c'est le seul fichier de règles à modifier. Il vit dans le client web, où le moteur Go/WASM de la page le lit à chaque chargement |
| `sef_rules.json` | Règles expansées, générées depuis le YAML |
| `sef_tests.json` | Cas de test de référence (9 326 cas), générés depuis le YAML ; rejoués aussi par le moteur Go (`engine/rules_test.go`) |
| `sef_rules.py` | Moteur de référence (Python) et outil de validation, d'export et de test |
| `sef_engine.js` | Second moteur (Node.js), écrit d'après la spécification seule ; rejoue `sef_tests.json` |
| `pbn_auction.py` | Lit un fichier PBN et sort la séquence d'enchères d'une donne en JSON ; le moteur Go de l'application en est le portage |
| `gen_golden.py` | Génère `engine/testdata/golden_python.json` : des enchères complètes de `pbn_auction.py` que le moteur Go doit reproduire |
| `test_pbn_auction.py` | Tests de `pbn_auction.py` (38 tests, `python -m unittest test_pbn_auction -v`) |
| `test_chelems.py` | Évalue les chelems du système au double mort (nécessite `endplay`) |
| `SEF_2024.md`, `SEF_2024.pdf` | Description bridge des conventions |
| `SEF_2024_spec.md` | Sémantique des règles, à lire pour porter le moteur |
| `donne.pbn` | Exemple de fichier PBN |

Dépendances : Python 3 avec `pyyaml` ; Node.js pour `sef_engine.js` ; `endplay` pour `test_chelems.py`.

## Lire ou générer les enchères d'une donne (PBN → JSON)

```bash
python pbn_auction.py donne.pbn                  # donne 1 (défaut)
python pbn_auction.py donne.pbn 3                # donne n°3
python pbn_auction.py donne.pbn -o sortie.json   # écrit dans un fichier
python pbn_auction.py donne.pbn --generate       # génère même si [Auction] existe
python pbn_auction.py donne.pbn --lang EN        # textes en anglais
python pbn_auction.py donne.pbn --opponents-pass   # la paire qui n'a pas ouvert passe toujours
python pbn_auction.py donne.pbn --option checkback2018
```

- Si la donne a une section `[Auction]`, elle est **lue** (annonces, alertes `=n=` reliées aux `[Note "n:texte"]`, marque `!`, `AP`).
- Sinon (ou avec `--generate`), les enchères sont **générées** à partir des quatre mains `[Deal]` avec les règles SEF 2024. Les quatre joueurs enchérissent. Quand aucune règle ne s'applique, ou si elle donnerait une annonce illégale, le joueur passe : `rule` vaut alors `null` et `default_calls` compte ces annonces.
- `--opponents-pass` : une fois la donne ouverte, la paire adverse passe toujours (`rule` = `opponents.pass`, non comptée dans `default_calls`) ; on n'a alors que les suites sans intervention, que les règles couvrent le mieux.
- Langue : `--lang FR` (Français, défaut) ou `--lang EN` (Anglais). Elle change les textes des règles (`meaning`, `alert`), les messages et l'aide.

Chaque annonce du JSON contient `seat`, `call`, `alerted`, `alert` (texte de l'alerte), et, en mode généré, `rule` (identifiant de la règle) et `meaning`.

Limites : les règles couvrent surtout la paire Nord-Sud avec adversaires qui passent. La vulnérabilité est accessible aux conditions (`vul`, `opp_vul`).

## Le fichier `cli/rules/sef_rules.yaml`

C'est une **liste ordonnée** de règles : pour une main et une séquence d'enchères données, la **première règle applicable** donne l'enchère. L'ordre est donc une priorité. Les lignes qui commencent par `#` sont des commentaires (titres de section, explications).

### Une règle

```yaml
- id: 1S.r.2H
  seq: "1S"
  call: 2H
  cond: "H >= 5 and hl >= 11"
  forcing: F1
  meaning: "5+ cœurs, 11+ points (HL), forcing"
  meaning_en: "5+ hearts, 11+ points (HL), forcing"
  status: choix
  alert: false
```

| Champ | Obligatoire | Contenu |
|---|---|---|
| `id` | oui | Identifiant **unique** (convention : `ouverture.réponse.suite`, ex. `1S.r.2H`) |
| `seq` | oui | Séquence déjà enchérie avant l'enchère à choisir. `""` = ouverture. Une liste de séquences est possible : `["2NT", "2C 2D 2NT"]` |
| `call` | oui | L'enchère : `1C` … `7NT`, `P`, `X`, `XX` |
| `cond` | oui | Condition sur la main qui parle, en expression Python restreinte (voir plus bas) |
| `forcing` | oui | `NF` non forcing, `F1` forcing un tour, `FM` forcing de manche, `INV` invitation, `SO` conclusion, `REL` relais, `ASK` question, `TO` contre d'appel, `PEN` contre punitif |
| `meaning` | oui | Signification en français (texte d'alerte) |
| `meaning_en` | oui | La même en anglais |
| `status` | oui | `sef` (fiche SEF), `choix` (choix validé), `sef2018`, `infere` (complétion non écrite dans la fiche), `a_verifier` |
| `alert` | non | `true` si l'enchère est conventionnelle et doit être alertée (défaut `false`) |
| `option` | non | La règle n'existe que si cette option est activée (ex. `checkback2018`, avec `--option`) |
| `trump` | non | `S`, `H`, `D` ou `C` : cette enchère fixe l'atout convenu (utile pour le Blackwood) |

**Séquences (`seq`).** Les jetons sont séparés par des espaces, dans l'ordre, **du point de vue de la paire qui parle**. Les enchères adverses sont entre parenthèses et leurs passes sont omises ; les passes de la paire s'écrivent `P` :

| À la table (donneur N) | `seq` |
|---|---|
| N 1♥, E passe, S ? | `"1H"` |
| N 1SA, E 2♥, S ? | `"1NT (2H)"` |
| N passe, E passe, S 1♠, O passe, N ? | `"P 1S"` |
| O 1SA, N ? | `"(1NT)"` |

Jokers : `1C|1D` (l'un ou l'autre), `*` (n'importe quel jeton), `**` en tête (n'importe quel début). Les passes initiales de la paire sont ignorées sauf si le motif commence lui-même par `P`. Détail complet dans `SEF_2024_spec.md` § 3.

**Conditions (`cond`).** Opérateurs `and`, `or`, `not`, `==`, `!=`, `<`, `<=`, `>`, `>=`, `+`, `-`, `in`, comparaisons chaînées (`9 <= hl <= 12`). Longueurs : `S`, `H`, `D`, `C`. Points : `hcp` (honneurs), `hl` (honneurs + longueur), `dh`, `hld` (avec distribution). Forme : `shape` (ex. `'5332'`), `balanced`, `semibalanced`. Autres : `ptricks` (levées de jeu), `qtricks`, `sidetricks` (levées de défense, totales et annexes), `losers`, `aces`, `kings`. Fonctions par couleur : `ace('S')`, `king('H')`, `top('S')`, `solid('C')`, `stop('D')`, `short('S')`, `hcp_in('H')`, `keycards('S')`, `ctrl1('C')`, `ctrl2('C')`, plus `max(...)` et `min(...)`. Le vocabulaire exact est dans `SEF_2024_spec.md` § 4.

**Modèles `for`.** Pour éviter de recopier une règle par couleur, on écrit le modèle une fois ; chaque `{clé}` est remplacée dans tous les champs texte :

```yaml
- id: "1{M}.r.2{M}"
  for: [{M: H}, {M: S}]      # génère 1H.r.2H et 1S.r.2S
  seq: "1{M}"
  call: "2{M}"
  cond: "{M} >= 3 and 6 <= dh <= 10"
  forcing: NF
  meaning: "Soutien simple"
  meaning_en: "Simple raise"
  status: sef
```

### Ajouter une règle

Exemple : après 1♠, avec 6 cartes de trèfle et 11-15 HL, inviter à 3♣. On insère la règle **avant** celles qui prendraient déjà cette main (ici avant les réponses naturelles à 2 sur 1, `"1{M}.r.2{m}"`) :

```yaml
- id: 1S.r.3C.invit
  seq: "1S"
  call: 3C
  cond: "C >= 6 and 11 <= hl <= 15"
  forcing: INV
  meaning: "6+ trèfles, 11-15 HL : invitation"
  meaning_en: "6+ clubs, 11-15 HL: invitation"
  status: choix
```

Vérifier l'effet avec une main concrète :

```bash
python sef_rules.py ../../cli/rules/sef_rules.yaml --validate
python sef_rules.py ../../cli/rules/sef_rules.yaml --hand "32.32.32.AKQJ92" --seq 1S
# avant : 2C  [F1, sef, 1S.r.2C]      après : 3C  [INV, choix, 1S.r.3C.invit]
```

Si la règle ouvre une nouvelle suite (ici la réponse de l'ouvreur après `1S 3C`), il faut aussi écrire les règles de cette suite (`seq: "1S 3C"`) ; sinon le moteur ne trouve rien et le programme passe par défaut. `--simulate 1000` liste les séquences non codées les plus fréquentes.

### Modifier une règle

On change les champs voulus en gardant l'`id`. Exemple : demander 12 HL au lieu de 11 pour la réponse 2♥ après 1♠. Changer `cond` **et** les textes qui mentionnent le seuil, en français et en anglais :

```yaml
- id: 1S.r.2H
  seq: "1S"
  call: 2H
  cond: "H >= 5 and hl >= 12"            # était : hl >= 11
  forcing: F1
  meaning: "5+ cœurs, 12+ points (HL), forcing"
  meaning_en: "5+ hearts, 12+ points (HL), forcing"
  status: choix
```

```bash
python sef_rules.py ../../cli/rules/sef_rules.yaml --hand "K2.QJ432.KJ2.432" --seq 1S
# avant : 2H (1S.r.2H)      après : 2C  (1S.r.2m.fallback) : la main de 11 HL passe à la règle suivante
```

Pour **déplacer** une règle (changer sa priorité), on la coupe et on la recolle plus haut ou plus bas dans le fichier. Pour modifier une règle issue d'un modèle `for`, on modifie le modèle (toutes ses variantes changent) ; pour ne changer qu'une variante, on l'en retire (`for: [{M: H}]`) et on écrit à part la variante voulue.

### Supprimer une règle

On supprime le bloc entier (de `- id:` jusqu'à la règle suivante). Exemple : retirer `1S.r.3C.invit` ci-dessus. Les mains qu'elle traitait retombent sur les règles suivantes (ici `2C`). Attention :

- les règles de **suite** (`seq: "1S 3C"`) qui dépendaient de cette enchère deviennent inatteignables : les supprimer aussi ;
- il ne faut pas supprimer une règle de repli (`*.fallback`, `gen.*`) sans avoir vérifié avec `--simulate` que le nombre de séquences non codées n'augmente pas ;
- chercher les références à l'`id` supprimé avec `grep` avant de le faire (tests, docs).

## Modifier les règles

On ne modifie que `cli/rules/sef_rules.yaml` (chaque règle porte `meaning` en français et `meaning_en` en anglais, tous deux obligatoires). Puis on régénère et on vérifie :

```bash
python sef_rules.py ../../cli/rules/sef_rules.yaml --validate
python sef_rules.py ../../cli/rules/sef_rules.yaml --json sef_rules.json
python sef_rules.py ../../cli/rules/sef_rules.yaml --gen-tests 1500 sef_tests.json --seed 2024
python sef_rules.py ../../cli/rules/sef_rules.yaml --check-tests sef_tests.json
node sef_engine.js sef_rules.json sef_tests.json
python gen_golden.py
cd ../.. && go test ./engine
```

`--check-tests` et `sef_engine.js` doivent afficher 0 échec / 0 écart ; `gen_golden.py` régénère les enchères de référence du moteur Go, et `go test ./engine` vérifie que ce dernier reproduit à l'identique l'expansion des règles, les cas de test et ces enchères (la page, elle, n'a pas besoin d'être recompilée : elle relit le YAML à chaque chargement). `sef_tests.json` porte l'empreinte du YAML d'origine : règles et tests doivent venir de la même version.

Autres commandes utiles :

```bash
python sef_rules.py ../../cli/rules/sef_rules.yaml --hand "AK32.KQ4.A32.J32" --seq "1NT 2C" --lang EN   # enchère d'une main
python sef_rules.py ../../cli/rules/sef_rules.yaml --simulate 1000                                        # simulation sur donnes aléatoires
```
