# SEF 2024 — Spécification du moteur d'enchères

Ce document décrit comment une application doit interpréter les règles d'enchères SEF 2024. Il se suffit à lui-même : les règles sont dans `cli/rules/default.yaml` (source) et `sef_rules.json` (généré), la description bridge dans `SEF_2024.md`.

## 1. Fichiers livrés

| Fichier | Rôle | Utilisé par l'application |
|---|---|---|
| `cli/rules/default.yaml` | **Source unique des règles**, éditée à la main (modèles `for`, commentaires) | Non (sert à générer le JSON) |
| `sef_rules.json` | Les règles, expansées et ordonnées (données) | **Oui**, chargé au démarrage |
| `sef_rules.schema.json` | Schéma JSON (draft 2020-12) de `sef_rules.json` | Oui, pour valider le fichier au chargement |
| `SEF_2024_spec.md` | Ce document : sémantique des règles | Par le développeur |
| `sef_tests.json` | Cas de test de référence | Par les tests automatiques de l'application |
| `sef_rules.py` | Implémentation de référence (Python 3) et outil de génération | Facultatif |
| `sef_engine.js` | Seconde implémentation (Node.js), écrite d'après ce document seul ; rejoue `sef_tests.json` | Facultatif (exemple de portage) |
| `SEF_2024.md` | Description bridge du système, pour la lecture (sans règles) | Non |

Chaîne de production : on modifie **uniquement** `cli/rules/default.yaml`, puis on régénère :

```
python3 sef_rules.py ../../cli/rules/default.yaml --validate
python3 sef_rules.py ../../cli/rules/default.yaml --json sef_rules.json
python3 sef_rules.py ../../cli/rules/default.yaml --gen-tests 1500 sef_tests.json --seed 2024
```

`sef_tests.json` porte l'empreinte SHA-256 du fichier de règles d'origine (`source_sha256`) : les tests et les règles doivent provenir de la même version.

### Substitution `for` (fichier YAML seulement)

Dans `cli/rules/default.yaml`, une règle peut porter `for:` (liste de dictionnaires). Elle est dupliquée pour chaque dictionnaire, et chaque `{clé}` est remplacée par sa valeur dans tous les champs texte (y compris les éléments d'une `seq` en liste). `sef_rules.json` contient les règles déjà expansées, sans `for`.

## 2. Structure d'une règle

`sef_rules.json` est un **tableau ordonné** de règles. L'ordre est une priorité : c'est la première règle applicable qui donne l'enchère.

| Champ | Obligatoire | Type | Description |
|---|---|---|---|
| `id` | oui | chaîne | Identifiant unique |
| `seq` | oui | chaîne ou tableau de chaînes | Séquence déjà enchérie avant l'enchère à choisir (§ 3). `""` = ouverture. Un tableau = plusieurs séquences équivalentes |
| `call` | oui | chaîne | Enchère proposée : `1C` … `7NT`, `P`, `X`, `XX` |
| `cond` | oui | chaîne | Condition sur la main du joueur qui parle (§ 4) |
| `forcing` | oui | énumération | `NF` non forcing · `F1` forcing un tour · `FM` forcing de manche · `SO` conclusion · `INV` invitation · `REL` relais / Texas · `ASK` question (Blackwood…) · `TO` contre d'appel · `PEN` contre punitif |
| `meaning` | oui | chaîne | Signification en clair (texte d'alerte à afficher) |
| `meaning_en` | oui | chaîne | Traduction anglaise de `meaning` (langue `EN` ; `FR` = `meaning`, langue par défaut) |
| `status` | oui | énumération | `sef` fiche SEF 2024 · `choix` choix validé · `sef2018` convention SEF 2018 · `infere` complétion non écrite dans la fiche · `a_verifier` |
| `alert` | non | booléen | Enchère conventionnelle, à alerter (défaut `false`) |
| `option` | non | chaîne | La règle n'existe que si cette option est activée pour la paire (ex. `checkback2018`) |
| `trump` | non | `S`/`H`/`D`/`C` | Quand cette enchère est choisie, l'atout convenu de la paire devient cette couleur (§ 5) |

Couleurs : `C` ♣, `D` ♦, `H` ♥, `S` ♠, `NT` sans-atout.

## 3. Séquences

### 3.1 Construction de la séquence courante

La séquence est construite **du point de vue de la paire qui parle** :

- jetons séparés par un espace, dans l'ordre chronologique ;
- on commence à la **première enchère autre que passe** de la table ;
- les enchères adverses sont **entre parenthèses** : `1NT (2H) X` ;
- les **passes adverses sont omis** ; les passes de la paire s'écrivent `P`.

Exemples (Nord-Sud parlent, Est-Ouest passent sauf mention) :

| Enchères à la table (donneur Nord) | Séquence vue par le prochain joueur NS |
|---|---|
| N 1♥, E passe, S ? | `1H` |
| N passe, E passe, S 1♠, O passe, N ? | `P 1S` |
| N 1SA, E 2♥, S ? | `1NT (2H)` |
| O 1SA, N ? | `(1NT)` |
| N 1SA, E passe, S 2♣, O passe, N 2♦, E passe, S ? | `1NT 2C 2D` |

### 3.2 Motif de `seq`

Un motif est une suite de jetons :

| Jeton | Correspond à |
|---|---|
| `1H` | exactement ce jeton |
| `1C\|1D` | l'un des jetons listés |
| `(2H)` | cette enchère adverse (les parenthèses doivent correspondre) |
| `(2H\|2S)` | l'une de ces enchères adverses |
| `*` | n'importe quel jeton (de la paire ou adverse) |
| `**` (en tête seulement) | n'importe quel début : 0 jeton ou plus ; le reste du motif doit correspondre à la **fin** de la séquence |
| `BW:X` (en tête seulement) | comme `**`, mais seulement si l'atout convenu de la paire est `X` (§ 5) |

Hors `**` et `BW:X`, le motif et la séquence doivent avoir **le même nombre de jetons**. Le motif `""` ne correspond qu'à la séquence vide.

### 3.3 Mains passées

Avant de comparer, on retire de la séquence courante ses `P` **initiaux** (passes de la paire avant l'ouverture). Exception : si le motif commence lui-même par `P`, on le compare à la séquence **complète** (conventions de main passée, ex. Drury `P 1S`).

Conséquence : une main qui a passé puis ouvre utilise les règles d'ouverture (`seq = ""`).

## 4. Conditions (`cond`)

### 4.1 Syntaxe

Sous-ensemble d'expressions Python :

- littéraux : entiers, décimaux, chaînes entre apostrophes (`'H'`, `'5422'`), tuples `('H', 'S')`, `true`, `false` ;
- opérateurs : `and`, `or`, `not`, `==`, `!=`, `<`, `<=`, `>`, `>=` (comparaisons chaînées permises : `9 <= hl <= 12`), `+`, `-`, `in` (appartenance à un tuple) ;
- appels : uniquement les fonctions du § 4.2 ;
- noms : uniquement ceux du § 4.2.

Sémantique Python : `and` / `or` court-circuités ; un booléen vaut 0 ou 1 dans une addition ou une soustraction (ex. `kings - king('S')`). Toute autre construction est invalide.

### 4.2 Vocabulaire

La main est donnée par couleur (`S`, `H`, `D`, `C`), cartes triées de la plus forte à la plus faible (`AKQJT98765432`). `n` désigne la longueur de la couleur.

| Nom | Type | Définition exacte |
|---|---|---|
| `S`, `H`, `D`, `C` | int | Longueur de la couleur |
| `hcp` | int | A = 4, R = 3, D = 2, V = 1 |
| `hl` | int | `hcp` + somme sur les couleurs de max(0, n − 4) |
| `dh` | int | `hcp` + points de courte : chicane 3, singleton 2, doubleton 1 (toutes couleurs) |
| `hld` | int | `hl` + mêmes points de courte |
| `shape` | str | Les 4 longueurs triées par ordre décroissant, concaténées (ex. `"5332"`) |
| `balanced` | bool | `shape` ∈ {`4333`, `4432`, `5332`} |
| `semibalanced` | bool | `shape` ∈ {`5422`, `6322`} |
| `aces`, `kings` | int | Nombre d'As, de Rois |
| `losers` | int | Somme par couleur : on prend les min(n, 3) premières cartes et on compte combien des min(n, 3) premiers honneurs de `A K D` y manquent |
| `ptricks` | float | Somme par couleur de min(n, t) + max(0, n − 3), où t = 1 pour l'As ; + 1 pour le Roi si As aussi, + 0,5 si Roi sans As (Roi au moins second) ; + 1 pour la Dame si A et R, + 0,5 si A ou R (Dame au moins troisième) |
| `qtricks` | float | Levées de défense : somme par couleur de 2 (A R), 1,5 (A D), 1 (A seul ou avec V…), 1 (R D), 0,5 (R au moins second sans A ni D), 0 sinon |
| `sidetricks` | float | Levées de défense annexes : `qtricks` sans la couleur la plus longue (à égalité de longueur, la plus haute : ♠ > ♥ > ♦ > ♣) |
| `ace(s)`, `king(s)`, `queen(s)` | bool | La couleur `s` contient l'As / le Roi / la Dame |
| `top(s)` | int | Nombre de cartes parmi A, R, D dans `s` |
| `solid(s)` | bool | A, R et D dans `s` |
| `stop(s)` | bool | A ; ou R avec n ≥ 2 ; ou D avec n ≥ 3 ; ou V avec n ≥ 4 |
| `short(s)` | bool | n ≤ 1 |
| `hcp_in(s)` | int | Points d'honneur dans `s` |
| `keycards(s)` | int | `aces` + 1 si le Roi de `s` est présent |
| `ctrl1(s)` | bool | As dans `s`, ou n = 0 |
| `ctrl2(s)` | bool | As ou Roi dans `s`, ou n ≤ 1 |
| `max(…)`, `min(…)` | fonction | Maximum / minimum de ses arguments |
| `vul` | bool | Le camp du joueur qui parle est vulnérable (tag PBN `[Vulnerable]` : `NS`, `EW`, `All`/`Both`) |
| `opp_vul` | bool | Le camp adverse est vulnérable |

Les conditions ne portent que sur la main du joueur qui parle et sur la vulnérabilité (`vul`, `opp_vul`) : ce que le partenaire a montré est contenu dans `seq`. Exemple, un barrage plus léger en vulnérabilité favorable : `cond: "{X} == 7 and (hcp <= 10 or (opp_vul and not vul and hcp <= 11))"`.

Dans `sef_tests.json`, chaque cas porte `vul` et `opp_vul` (les quatre combinaisons en rotation) ; `sef_rules.py --hand` accepte `--vul` et `--opp-vul`.

## 5. État de l'enchère : atout convenu

Chaque paire a un **atout convenu**, initialement indéfini. Quand un joueur choisit une règle qui porte `trump: X`, l'atout convenu de sa paire devient `X` (il peut être redéfini plus tard). Les motifs `BW:X …` ne s'appliquent que si l'atout convenu est `X` : c'est ainsi qu'un 4SA est traité comme Blackwood pour la bonne couleur, après n'importe quelle séquence.

## 6. Algorithme de choix

```
entrée : main, séquence courante, atout convenu de la paire, options de la paire
pour chaque règle r dans l'ordre du fichier :
    si r.option est défini et n'est pas dans les options : continuer
    si aucun motif de r.seq ne correspond (§ 3, avec l'atout pour BW:) : continuer
    si cond(r) est vraie pour la main : renvoyer r
renvoyer « séquence non codée »
```

- Le résultat est la **règle** trouvée : l'application utilise `call`, affiche `meaning` si `alert` est vrai, et met à jour l'atout si `trump` est présent.
- **Séquence non codée** : aucune règle ne s'applique. L'application doit le signaler (et, par exemple, laisser l'utilisateur choisir) ; ce n'est pas une passe.
- Les règles ne vérifient pas la légalité : une enchère renvoyée qui ne serait pas supérieure à la précédente est un défaut des règles et doit être signalée.
- Les adversaires : le moteur ne décrit que les enchères de la paire. Leur comportement relève de l'application (dans les tests, ils passent toujours).

## 7. Fichier de tests `sef_tests.json`

En-tête : `format` (`sef-tests/1`), `source`, `source_sha256`, `seed`, `deals` (donnes tirées), `strong_deals` (dont donnes fortes), `count` (nombre de cas), puis `cases`, un objet par ligne :

```json
{"id":6,"hand":"KQJ3.9654.Q86.AK","seq":"1C","trump":null,"options":[],
 "expected":{"call":"1H","rule":"1C.r.1H"},
 "features":{"S":4,"H":4,"D":3,"C":2,"hcp":15,"hl":15,"dh":16,"hld":16,"shape":"4432",
             "balanced":true,"semibalanced":false,"aces":1,"kings":2,"losers":6,"ptricks":5.0,
             "ace":{"S":false,"H":false,"D":false,"C":true}, "…":"…"}}
```

| Champ | Contenu |
|---|---|
| `hand` | Main `♠.♥.♦.♣`, `T` pour le 10, `-` pour une chicane |
| `seq`, `trump`, `options` | Entrées de l'algorithme (§ 6) |
| `expected.call`, `expected.rule` | Enchère et `id` de la règle attendus ; `null` pour une séquence non codée |
| `features` | Valeurs attendues du vocabulaire (§ 4.2) pour cette main, fonctions données couleur par couleur |

Une réimplémentation est conforme quand, pour **chaque** cas, elle calcule les mêmes `features` et renvoie la même règle. Vérifier d'abord `features` (erreurs de calcul), puis `expected` (erreurs de correspondance ou de priorité). La version de référence se contrôle par :

```
python3 sef_rules.py ../../cli/rules/default.yaml --check-tests sef_tests.json
node sef_engine.js sef_rules.json sef_tests.json
```

Les deux implémentations fournies passent les 9 331 cas sans écart.

Le jeu fourni (graine 2024) compte 9 331 cas issus de 2 250 donnes, dont 750 donnes fortes (31+ H dans la paire) pour couvrir Blackwood, contrôles et chelems ; un quart des donnes active l'option `checkback2018`. Il utilise 665 règles différentes.

## 8. Limites connues

- Environ 57 % des règles sont des complétions (`status: infere`) : suites naturelles non décrites par les fiches, à revoir en priorité.
- La compétition n'est codée qu'en partie (Landy, Michaëls précisés, Rubensohl, Truscott, Spoutnik simple) ; Joséphine, le Blackwood d'exclusion et le Blackwood sur barrage ne sont pas codés.
- Mesures sur donnes aléatoires, adversaires muets : 100 % des enchères arrivent à un contrat ; au double mort, 70 % des chelems demandés sont réussis.
