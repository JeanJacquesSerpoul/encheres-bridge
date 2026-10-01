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
5. 5**forcing**La force de l'enchère.

   NFF1FMSOINVRELASKTOPEN
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

### Points

hcphldhhld

H (A 4, R 3, D 2, V 1), HL avec les cartes au-delà de la 4e, DH et HLD avec les courtes.

### Longueurs et forme

SHDCshapebalancedsemibalanced

`shape` vaut par exemple `'5332'`.

### Levées et contrôles

ptricksqtrickssidetrickslosersaceskings

Levées de jeu, de défense, perdantes.

### Par couleur

stop('H')keycards('S')top(s)hcp_in(s)short(s)ctrl1(s)

Arrêt, clés, honneurs, courte, contrôles.

### Contexte

vulopp_vulseat

Vulnérabilité des deux camps, rang du joueur depuis le donneur : la règle des 15 lit `seat == 4`.

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