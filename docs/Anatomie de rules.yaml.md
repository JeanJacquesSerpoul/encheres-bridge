Encheres-bridge · systèmes d'enchères

# Anatomie de `rules.yaml`

Le moteur ne connaît aucune règle de bridge. Tout ce qu'il enchérit vient d'un fichier : une liste ordonnée de règles, où la première qui convient à la main donne l'enchère. Voici comment ce fichier se lit.

cli/systems/ index.json les systèmes proposés sef/ SEF 2024, par défaut rules.yaml 830 règles écrites, 2 738 une fois expansées rules.pdf description, en clair rules.en.pdf pbn/ donnes thématiques new/ système en construction rules.yaml 1 règle

1 · La brique de base

## Une règle, champ par champ

Chaque règle répond à une seule question : *dans cette séquence, avec cette main, que dit-on ?* Voici une vraie règle du SEF, le Stayman après l'ouverture de 1SA.

```
1- id: 1NT.r.2C
2  seq: ["1NT", "1NT (X)"]
3  call: 2C
4  cond: "(H == 4 or S == 4)
        and max(H, S) < 5 and hl >= 8
        and not (shape == '4333'
                 and hcp < 12)"
5  forcing: F1
6  meaning: "Stayman 4 réponses : majeure
           4e, 8+ H (4333 : 12+ H)"
6  meaning_en: "4-response Stayman: …"
7  status: sef
8  alert: true
```

1. 1**id**Identifiant unique. Il nomme la règle dans les commentaires, les tests et l'arbre de décision.
2. 2**seq**Le motif de la séquence déjà enchérie, vue par la paire. `""` pour une ouverture ; une liste pour plusieurs séquences équivalentes, ici 1SA, contré ou non.
3. 3**call**L'enchère proposée : `1C` à `7NT`, `P`, `X`, `XX`.
4. 4**cond**La condition sur la main de celui qui parle. Une expression : points, longueurs, forme, levées…
5. 5**forcing**La force de l'enchère : ce qu'elle impose au partenaire.

   NF

   non forcing : le partenaire peut passer

   F1

   forcing un tour : le partenaire doit reparler

   FM

   forcing de manche : on ne s'arrête pas avant la manche

   SO

   conclusion : pour jouer, le partenaire passe

   INV

   invitation : le partenaire choisit entre s'arrêter et conclure

   REL

   relais ou Texas : demande au partenaire une enchère précise

   ASK

   question : Blackwood, demande de clés ou de contrôles

   TO

   contre d'appel : demande au partenaire de choisir une couleur

   PEN

   contre punitif : pour jouer le contrat adverse contré
6. 6**meaning, meaning_en**Le commentaire affiché, en français et en anglais, dans la séquence commentée.
7. 7**status**L'origine de la règle.

   sefchoixsef2018inferea_verifier
8. 8**alert, trump, option, for**Facultatifs : enchère à alerter, atout convenu (§ 6), règle réservée à une option, modèle à dupliquer (§ 5).

obligatoirefacultatif

2 · Ce que `seq` regarde

## La séquence vue par la paire

Le motif `seq` ne se compare pas aux enchères de la table telles quelles, mais à la séquence vue par la paire qui parle : ses propres enchères, et celles des adversaires entre parenthèses.

| NORD | EST | SUD | OUEST |
| --- | --- | --- | --- |
| 1♣ | 1♠ | X | Passe |
| ? |  |  |  |

→

1C(1S)X

Ce que voit Nord : 1C (1S) X

**Nos enchères, passes compris**Un passe du partenaire est une information : il s'écrit `P`.

**Les leurs, entre parenthèses**Une intervention adverse devient `(1S)`. Leurs passes sont omis.

**Les passes d'entrée s'effacent**Une main passée qui ouvre suit les règles d'ouverture, sauf si le motif commence lui-même par `P` : c'est ainsi que Drury s'écrit `P 1S`.

3 · Le langage des motifs

## Comment un motif reconnaît une séquence

Un motif est une suite de jetons séparés par des espaces. Sans joker de tête, il doit avoir exactement autant de jetons que la séquence.

| Jeton | Correspond à | Exemples |
| --- | --- | --- |
| 1H | exactement cette enchère de la paire | 1H1H ✓1S ✗ |
| 1C\|1D | l'une des enchères listées | 1C\|1D 1H1D 1H ✓1S 1H ✗ |
| (2H) | cette enchère adverse | 1S (2H)1S (2H) ✓1S 2H ✗ |
| * | une enchère quelconque, à nous ou à eux | 1NT * 1NT (2C) ✓1NT ✗ |
| \*\* | en tête : n'importe quel début, même vide ; le reste doit finir la séquence | \*\* 4NT1H 3H 4NT ✓4NT ✓ |
| BW:H | comme `**`, si l'atout convenu de la paire est ♥ | BW:H 4NTatout ♥ ✓atout ♠ ✗ |
| "" | la séquence vide : une ouverture | personne n'a parlé ✓ |

4 · Le langage des conditions

## Ce que `cond` peut lire

Une condition est un petit sous-ensemble d'expressions Python : `and`, `or`, `not`, comparaisons chaînées (`9 <= hl <= 12`), `+`, `-`, `in`. Elle ne lit que la main de celui qui parle. Ce que le partenaire a montré, la séquence le dit déjà.

Main d'exemple : les pastilles à droite de chaque nom donnent sa valeur pour cette main

♠ A R V 5 3♥ R D 4 3♦ 2♣ V 3 2

### Points

hcp

Points d'honneur, H : As 4, Roi 3, Dame 2, Valet 1.

14

hl

H + 1 point par carte à partir de la 5e dans une couleur commandée par au moins D V (5e pique : +1) ; − 1 par honneur sec ou paire d'honneurs secs.

15

dh

H + points de courte : chicane 3, singleton 2, doubleton 1 (singleton ♦ : +2) ; même dévaluation que HL.

16

hld

HL + les mêmes points de courte.

17

### Longueurs et forme

S H D C

Longueur à pique, cœur, carreau, trèfle.

5 4 1 3

shape

Les quatre longueurs, de la plus longue à la plus courte, écrites d'un bloc ; se compare à une chaîne.

'5431'

balanced

Main régulière : 4333, 4432 ou 5332.

false

semibalanced

Main semi-régulière : 5422 ou 6322.

false

### Levées et contrôles

ptricks

Levées de jeu estimées : par couleur, les honneurs qui font levée, une demi-levée pour la 4e carte et une levée par carte à partir de la 5e.

5

qtricks

Levées de défense : par couleur, A R 2, A D 1,5, A 1, R D 1, R second 0,5.

3

sidetricks

`qtricks` sans compter la couleur la plus longue.

1

losers

Perdantes : dans les trois premières cartes de chaque couleur, les As, Rois et Dames qui manquent.

6

aces, kings

Nombre d'As, nombre de Rois.

1, 2

### Par couleur

Ces fonctions prennent une couleur : `'S'`, `'H'`, `'D'` ou `'C'`.

stop(s)

Arrêt : As, Roi second, Dame troisième ou Valet quatrième.

stop('C') false

keycards(s)

Clés pour le Blackwood : les As, plus le Roi de la couleur `s`.

keycards('S') 2

ace(s), king(s), queen(s)

La couleur contient l'As, le Roi, la Dame.

queen('H') true

top(s)

Nombre d'honneurs parmi As, Roi, Dame.

top('S') 2

solid(s)

As, Roi et Dame ensemble.

solid('S') false

hcp_in(s)

Points d'honneur de la couleur.

hcp_in('S') 8

short(s)

Singleton ou chicane.

short('D') true

ctrl1(s)

Contrôle de premier tour : As ou chicane.

ctrl1('H') false

ctrl2(s)

Contrôle de deuxième tour : As, Roi, singleton ou chicane.

ctrl2('D') true

max(…), min(…)

Le plus grand, le plus petit de leurs arguments.

max(S, H) 5

### Contexte

Pas la main, mais la donne : vulnérabilité (tag PBN `[Vulnerable]`) et position.

vul

Le camp de celui qui parle est vulnérable.

true / false

opp_vul

Le camp adverse est vulnérable : un barrage peut être plus léger en vulnérabilité favorable, `opp_vul and not vul`.

true / false

seat

Rang de celui qui parle, compté depuis le donneur : 1 à 4. Les passes adverses n'étant pas dans la séquence, c'est lui qui distingue une ouverture en 1re ou en 4e position (règle des 15 : `seat == 4`).

1 à 4

5 · L'ordre décide

## La première règle applicable l'emporte

Le moteur parcourt le fichier de haut en bas et s'arrête à la première règle dont le motif et la condition conviennent. L'ordre est donc une priorité : les cas particuliers d'abord, la règle générale ensuite, et souvent un `cond: "true"` pour finir. Ci-dessous, les 21 règles du SEF qui répondent à une ouverture de 1SA, dans l'ordre du fichier. Choisissez une main de répondant.

Ou votre main : piques.cœurs.carreaux.trèfles

Étape 1

### Séquence

Construire la séquence vue par la paire, sans ses passes d'entrée.

Étape 2

### Motif

Pour chaque règle, dans l'ordre : son `seq` correspond-il ? Sinon, règle suivante.

Étape 3

### Condition

Son `cond` est-il vrai sur la main ? Sinon, règle suivante.

Résultat

### Enchère

La première règle retenue donne `call`. Aucune règle, ou une enchère illégale : passe par défaut, signalé comme tel.

6 · Écrire moins

## Les modèles `for:`

Une règle qui porte `for:` est un modèle : elle est dupliquée pour chaque ligne de la liste, et chaque `{clé}` est remplacée par sa valeur. Le SEF compte 830 règles écrites, et 2 738 une fois expansées.

```
- id: "open.4e.{c}"
  for: [{c: 1S, x: "S >= 5 and S >= H"},
        {c: 1H, x: "H >= 5"},
        {c: 1D, x: "D >= 3 and …"},
        {c: 1C, x: "true"}]
  seq: ""
  call: "{c}"
  cond: "seat == 4 and 10 <= hcp <= 11
         and hcp + S >= 15 and ({x})"
  # règle des 15, en 4e position
```

→

**open.4e.1S** · call 1S · … and (S >= 5 and S >= H)

**open.4e.1H** · call 1H · … and (H >= 5)

**open.4e.1D** · call 1D · … and (D >= 3 and …)

**open.4e.1C** · call 1C · … and (true)

7 · Une mémoire

## L'atout convenu

Les conditions ne voient que la main. Pour savoir quelle couleur est l'atout, une règle peut porter `trump:` : quand elle est choisie, l'atout convenu de la paire devient cette couleur. Les motifs `BW:X` le lisent, ce qui permet d'écrire une seule fois les réponses au Blackwood pour chaque atout, quelle que soit la séquence.

Nord1SA

15-17 HL régulier

Sud2♣

Stayman

Nord2SA

Pas de majeure 4e

Sud3♠

`1NT.2C.2NT.slamS` porte `trump: S` : l'atout convenu devient ♠

Plus tard4SA

La réponse suit les règles `BW:S 4NT` : clés à ♠

8 · Du fichier au site

## Modifier un système

On édite `rules.yaml` et on recharge la page : il n'y a rien à recompiler. Avant de publier, une commande met à jour tout ce qui dérive du fichier.

### Éditer

- `cli/systems/<id>/rules.yaml`
- Ordre : du particulier au général
- `meaning` et `meaning_en` obligatoires

### Mettre à jour

```
./update-system.sh sef
```

- Validation (un fichier invalide est refusé en entier)
- Cas de test et enchères de référence
- PDF français et anglais
- Tests, dont le banc du par

### Publier

```
./update-system.sh -p sef
```

- Branche, commit, push
- Pull request
- Le site suit à la fusion

Règles, exemples et décisions du simulateur : SEF 2024 de cli/systems/sef/rules.yaml, vérifiés avec le moteur de référence sef_rules.py. Sémantique complète : tools/python_tools/SEF_2024_spec.md.