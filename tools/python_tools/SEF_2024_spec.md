# SEF 2024 — Spécification du moteur d'enchères

Ce document décrit comment une application doit interpréter les règles d'enchères SEF 2024. Il se suffit à lui-même : les règles sont dans `cli/systems/sef/rules.yaml` (source) et `rules.json` (généré), la description bridge dans `SEF_2024.md`.

## 1. Fichiers livrés

| Fichier | Rôle | Utilisé par l'application |
|---|---|---|
| `cli/systems/sef/rules.yaml` | **Source unique des règles**, éditée à la main (modèles `for`, commentaires) | Non (sert à générer le JSON) |
| `rules.json` | Les règles, expansées et ordonnées (données) | **Oui**, chargé au démarrage |
| `sef_rules.schema.json` | Schéma JSON (draft 2020-12) de `rules.json` | Oui, pour valider le fichier au chargement |
| `SEF_2024_spec.md` | Ce document : sémantique des règles | Par le développeur |
| `tests.json` | Cas de test de référence | Par les tests automatiques de l'application |
| `sef_rules.py` | Implémentation de référence (Python 3) et outil de génération | Facultatif |
| `sef_engine.js` | Seconde implémentation (Node.js), écrite d'après ce document seul ; rejoue `tests.json` | Facultatif (exemple de portage) |
| `SEF_2024.md` | Description bridge du système, pour la lecture (sans règles) | Non |

Chaîne de production : on modifie **uniquement** `cli/systems/sef/rules.yaml`, puis on régénère :

```
python3 sef_rules.py ../../cli/systems/sef/rules.yaml --validate
python3 sef_rules.py ../../cli/systems/sef/rules.yaml --json ../../engine/testdata/systems/sef/rules.json
python3 sef_rules.py ../../cli/systems/sef/rules.yaml --gen-tests 1500 ../../engine/testdata/systems/sef/tests.json --seed 2024
```

`tests.json` porte l'empreinte SHA-256 du fichier de règles d'origine (`source_sha256`) : les tests et les règles doivent provenir de la même version.

### Substitution `for` (fichier YAML seulement)

Dans `cli/systems/sef/rules.yaml`, une règle peut porter `for:` (liste de dictionnaires). Elle est dupliquée pour chaque dictionnaire, et chaque `{clé}` est remplacée par sa valeur dans tous les champs texte (y compris les éléments d'une `seq` en liste). `rules.json` contient les règles déjà expansées, sans `for`.

## 2. Structure d'une règle

`rules.json` est un **tableau ordonné** de règles. L'ordre est une priorité : c'est la première règle applicable qui donne l'enchère.

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
| `natural` | non | booléen | `false` pour une enchère à la couleur conventionnelle qui n'est pas alertée (rectification de Texas, réponse à l'As, réponse au Blackwood…) : elle ne compte pas comme couleur du partenaire (`p_suit`, § 4.3). Défaut `true` |
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
| `hl` | int | `hcp` + points de longueur − dévaluation. Points de longueur : max(0, n − 4) dans chaque couleur qui contient au moins deux honneurs parmi A R D V (« commandée par au moins D V »). Dévaluation : 1 point par couleur faite d'un honneur sec (As compris) ou de deux honneurs secs (A R D V) |
| `dh` | int | `hcp` + points de courte : chicane 3, singleton 2, doubleton 1 (toutes couleurs) − même dévaluation que `hl` |
| `hld` | int | `hl` + mêmes points de courte + 4 pour un bicolore 6-5 ou plus (les deux couleurs les plus longues totalisent au moins 11 cartes, la seconde en a au moins 5) |
| `shape` | str | Les 4 longueurs triées par ordre décroissant, concaténées (ex. `"5332"`) |
| `balanced` | bool | `shape` ∈ {`4333`, `4432`, `5332`} |
| `semibalanced` | bool | `shape` ∈ {`5422`, `6322`} |
| `aces`, `kings` | int | Nombre d'As, de Rois |
| `losers` | int | Somme par couleur : on prend les min(n, 3) premières cartes et on compte combien des min(n, 3) premiers honneurs de `A K D` y manquent |
| `ptricks` | float | Somme par couleur de min(n, t) + 0,5 si n ≥ 4 + max(0, n − 4) (demi-levée pour la 4e carte, une levée par carte à partir de la 5e), où t = 1 pour l'As ; + 1 pour le Roi si As aussi, + 0,5 si Roi sans As (Roi au moins second) ; + 1 pour la Dame si A et R, + 0,5 si A ou R (Dame au moins troisième) |
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
| `seat` | int | Rang du joueur qui parle dans le tour d'enchères, compté depuis le donneur : 1 à 4 (4 = 4e position) |
| `lvl` | int | Palier le plus élevé atteint dans l'enchère (les deux camps), 0 avant toute enchère |
| `p_suit` | chaîne | Dernière couleur nommée naturellement par le partenaire : `'S'`, `'H'`, `'D'`, `'C'`, ou `''` (§ 4.3) |
| `p_len` | int | Longueur promise dans `p_suit` (§ 4.3), 0 sans couleur |
| `fit` | int | Atouts du camp dans `p_suit` : longueur de la main dans `p_suit` + `p_len` ; 0 sans couleur |
| `p_forcing` | bool | La dernière enchère du partenaire est `F1`, `FM`, `REL`, `ASK` ou `TO` |

Les conditions portent sur la main du joueur qui parle, sur la vulnérabilité (`vul`, `opp_vul`), sur son rang (`seat`) et sur le contexte de l'enchère (§ 4.3) : le reste de ce que le partenaire a montré est contenu dans `seq`. Les passes adverses n'apparaissant pas dans `seq`, seul `seat` distingue une ouverture en 1re, 3e ou 4e position ; exemple, la règle des 15 : `cond: "seat == 4 and 10 <= hcp <= 11 and hcp + S >= 15"`. Exemple, un barrage plus léger en vulnérabilité favorable : `cond: "{X} == 7 and (hcp <= 10 or (opp_vul and not vul and hcp <= 11))"`.

Dans `tests.json`, chaque cas porte `vul` et `opp_vul` (les quatre combinaisons en rotation) et `seat` (Nord parle en 1re ou 2e position, Sud en 3e ou 4e) ; `sef_rules.py --hand` accepte `--vul`, `--opp-vul` et `--seat`.

### 4.3 Contexte de l'enchère

`lvl`, `p_suit`, `p_len`, `fit` et `p_forcing` se calculent sur toute l'enchère déjà faite (les quatre joueurs), avec la règle choisie pour chaque appel :

- **Couleur naturelle** : un appel `1C` … `7S` dont la règle n'est ni `alert: true` ni `natural: false`. Un passe par défaut (aucune règle) ou une enchère illégale refusée n'ont pas de règle.
- `p_suit` : la plus récente des couleurs nommées naturellement par le partenaire (une enchère conventionnelle ou à SA plus récente ne l'efface pas).
- `p_len` : si cette enchère est l'**ouverture** (première enchère de l'enchère), 5 pour 1♥/1♠, 4 pour 1♣/1♦, 6 au palier de 2 ou plus ; si le camp du partenaire n'a pas ouvert (**intervention**), 5 ; sinon 4. Si le partenaire a nommé naturellement cette couleur au moins deux fois, `p_len` devient `min(max(p_len + 1, 5), 6)`.
- `p_forcing` : porte sur le **dernier appel** du partenaire, quel qu'il soit (faux s'il n'a pas de règle).

Le moteur Go (`setContext`), `sef_rules.py` (`context`) et `pbn_auction.py` calculent ce contexte ; `golden.json` le donne pour chaque appel (`ctx` : `[lvl, p_suit, p_len, p_forcing]`) et `TestGoldenPython` vérifie que Go retrouve les mêmes valeurs. Sans historique (`sef_rules.py --hand`), `lvl` est lu dans `seq` et il n'y a pas de couleur du partenaire.

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

## 7. Fichier de tests `tests.json`

En-tête : `format` (`sef-tests/1`), `source`, `source_sha256`, `seed`, `deals` (donnes tirées), `strong_deals` (dont donnes fortes), `count` (nombre de cas), puis `cases`, un objet par ligne :

```json
{"id":6,"hand":"KQJ3.9654.Q86.AK","seq":"1C","trump":null,"options":[],
 "expected":{"call":"1H","rule":"1C.r.1H"},
 "features":{"S":4,"H":4,"D":3,"C":2,"hcp":15,"hl":14,"dh":15,"hld":15,"shape":"4432",
             "balanced":true,"semibalanced":false,"aces":1,"kings":2,"losers":6,"ptricks":4.0,
             "ace":{"S":false,"H":false,"D":false,"C":true}, "…":"…"}}
```

| Champ | Contenu |
|---|---|
| `hand` | Main `♠.♥.♦.♣`, `T` pour le 10, `-` pour une chicane |
| `seq`, `trump`, `options` | Entrées de l'algorithme (§ 6) |
| `vul`, `opp_vul`, `seat` | Contexte de la table (§ 4.2) |
| `lvl`, `p_suit`, `p_len`, `p_forcing` | Contexte de l'enchère (§ 4.3), calculé pendant le tirage ; `fit` s'en déduit avec la main |
| `expected.call`, `expected.rule` | Enchère et `id` de la règle attendus ; `null` pour une séquence non codée |
| `features` | Valeurs attendues du vocabulaire (§ 4.2) pour cette main, fonctions données couleur par couleur |

Une réimplémentation est conforme quand, pour **chaque** cas, elle calcule les mêmes `features` et renvoie la même règle. Vérifier d'abord `features` (erreurs de calcul), puis `expected` (erreurs de correspondance ou de priorité). La version de référence se contrôle par :

```
python3 sef_rules.py ../../cli/systems/sef/rules.yaml --check-tests ../../engine/testdata/systems/sef/tests.json
node sef_engine.js ../../engine/testdata/systems/sef/rules.json ../../engine/testdata/systems/sef/tests.json
```

Les deux implémentations fournies passent tous les cas sans écart.

Le jeu fourni (graine 2024) est tiré de 2 250 donnes, dont 750 donnes fortes (31+ H dans la paire) pour couvrir Blackwood, contrôles et chelems ; un quart des donnes active l'option `checkback2018`. Son en-tête donne le nombre de cas (`count`), qui suit les règles.

## 8. Limites connues

- Plus de la moitié des règles (environ 55 %) sont des complétions (`status: infere`) : suites naturelles non décrites par les fiches, à revoir en priorité. Sur le banc du par, elles ne donnent que 28 % des enchères, mais 62 % (en IMP) des dernières enchères des manches et chelems manqués ou surenchéris.
- La compétition n'est codée qu'en partie (Landy, Michaëls précisés, Rubensohl, Truscott, Spoutnik simple) ; Joséphine, le Blackwood d'exclusion et le Blackwood sur barrage ne sont pas codés.
- Mesures sur donnes aléatoires, adversaires muets : 100 % des enchères arrivent à un contrat ; au double mort, 70 % des chelems demandés sont réussis.
