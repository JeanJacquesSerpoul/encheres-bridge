# Les règles du moteur d'enchères — description complète

Ce document décrit **tout** ce que le moteur décide, tel qu'il est réellement
codé (et non tel qu'il devrait l'être). Il est destiné à la relecture par un
joueur de bridge : chaque règle porte un numéro stable — **[O-3]**, **[RM-7]**…
— pour pouvoir être citée dans un commentaire.

Les seuils sont donnés exactement comme le moteur les applique. Là où le code
s'écarte du SEF ou simplifie, le paragraphe est marqué **⚠**.

Sommaire :
[1](#1-conventions-de-lecture) · [2](#2-évaluation-des-mains) ·
[3](#3-les-ouvertures) · [4](#4-réponses-aux-ouvertures-de-1-à-la-couleur) ·
[5](#5-réponses-à-1sa-et-2sa) · [6](#6-réponses-aux-ouvertures-fortes-et-de-barrage) ·
[7](#7-redemandes-de-louvreur) · [8](#8-conventions-du-camp-de-louvreur) ·
[9](#9-le-camp-de-la-défense) · [10](#10-la-compétition) ·
[11](#11-la-zone-de-chelem) · [12](#12-la-décision-générique-de-fin-denchères) ·
[13](#13-filets-de-sécurité) · [14](#14-points-à-discuter-en-priorité)

---

## 1. Conventions de lecture

- **H** = points d'honneurs (A=4, R=3, D=2, V=1).
- **L** = points de longueur.
- **HL** = H + L. **HLD** = HL + points de distribution (voir §2).
- « le moins cher » = l'enchère la plus basse encore légale dans cette couleur.
- Les règles d'une même section sont **ordonnées** : le moteur applique la
  première qui s'applique et s'arrête. L'ordre est donc lui-même une règle.
- Le moteur joue les **deux camps** avec le même système : ce que montre une
  enchère est immédiatement connu du partenaire.

Ce que le moteur mémorise après chaque enchère : la fourchette de points
annoncée (plancher et plafond), la longueur minimale promise dans chaque
couleur, les arrêts garantis, les contrôles montrés ou niés, et une poignée de
drapeaux conventionnels (Texas en cours, Blackwood posé, etc.).

**Le moteur ne voit jamais la main du partenaire.** Toutes les décisions
n'utilisent que la main du joueur et ce que les enchères ont annoncé.

Ce document couvre **toutes** les fonctions de décision du moteur. La seule
classification non décrite ici (« régulière / unicolore / bicolore /
tricolore ») ne sert qu'à l'affichage du type de main dans l'interface : elle
n'intervient dans **aucune** décision d'enchère.

### 1.1 Le commentaire des passes

**Aucune enchère ne sort sans commentaire**, passes comprises. La plupart des
passes ont une règle qui les explique (« moins de 6 points », « arrêt, jeu
faible »…) ; ceux que le moteur atteint faute de règle applicable — il n'a
rien trouvé à dire, ce qui est le cas le plus fréquent — reçoivent la raison
que leur **position** garantit, et rien de plus :

| Situation | Commentaire |
|-----------|-------------|
| Personne n'a encore ouvert | pas les conditions d'ouverture |
| Notre camp tient la dernière enchère, dite par le partenaire | l'enchère du partenaire convient, rien à ajouter |
| …dite par soi-même | rien à ajouter à sa propre enchère |
| Les adversaires tiennent l'enchère, notre camp a déjà parlé | l'enchère est aux adversaires, pas de quoi pousser plus haut |
| …notre camp s'est tu, et la main a 12 H et plus | l'ouverture, mais aucune enchère ne décrit la main |
| …notre camp s'est tu, main plus faible | pas de quoi intervenir |

Ces textes ne disent **que** ce que la position prouve. En particulier, une
main d'ouverture qui passe n'est jamais décrite comme faible : le commentaire
dit que les points sont là et qu'aucune enchère du système ne décrit la main
— c'est une information utile au joueur, et souvent le signe d'une règle
manquante.

---

## 2. Évaluation des mains

### [E-1] Comptage des points
- **H** : A=4, R=3, D=2, V=1.
- **L** : +1 par carte au-delà de la quatrième, dans chaque couleur.
  (♠AR8654 = 2 points de longueur.)
- **HLD(atout)** : HL plus, **dans les trois couleurs autres que l'atout**,
  +3 par chicane, +2 par singleton, +1 par doubleton.
  ⚠ Le doubleton **d'atout** ne compte pas ; les points D ne sont ajoutés
  qu'une fois un atout choisi.

### [E-1c] Une courte face à la longueur du partenaire ne vaut pas ses points D
HLD paie la courte parce qu'une courte fait travailler les atouts. La prémisse
tombe exactement là où le partenaire est long : une chicane face à six Trèfles
n'est pas trois points de coupe, c'est trois points de **gaspillage** — sa
longueur est l'endroit où nos petits atouts auraient gagné leur vie, et les
honneurs qu'il y détient sont des levées que la courte jette. Le moteur
retranche donc les points D dans toute couleur où le partenaire a montré
**5 cartes et plus** (`hldFacingPartner`) — même instrument que
`hldAgainstTheirBidding`, tourné vers notre propre camp. Face à une couleur de
4 cartes — l'ouverture de 1♦ par exemple — la courte garde ses points : la
longueur d'en face n'est pas assez grande pour que nos atouts n'aient rien à
couper.

⚠ **Seulement au chelem.** La manche se gagne sur une coupe à laquelle le
compte ne croit qu'à moitié ; douze levées, non. Les points restent donc dans
`cMin`, sur lequel les décisions de manche sont étalonnées, et n'en sortent
que dans `cMinSlam`, où chaque point doit être une levée.

### [E-1b] Le passe d'ouverture est un plafond, et il tient
Passer à son premier tour **avant que quiconque ait ouvert** plafonne la main
sous le seuil d'ouverture (moins de 12 H et moins de 13 HL) : le moteur
enregistre 0-11. Ce plafond ne peut plus être relevé. Une enchère ultérieure
dont le plancher annoncé le dépasse — un Blackwood qui promet 14, par exemple
— réclame des points que le passe a déjà niés : la fourchette attachée à une
enchère est une description conventionnelle, le passe est une observation sur
les cartes réelles. En cas de contradiction c'est le passe qui gagne, et le
plancher est rogné au plafond au lieu que le plafond soit hissé au plancher.
Sans cette règle un joueur ayant passé à l'ouverture pouvait conduire son camp
au chelem en affichant quatorze points.

### [E-2] Types de mains
- **Régulière** : aucun singleton ni chicane, au plus un doubleton →
  4-3-3-3, 4-4-3-2, 5-3-3-2 (et 6-3-3-1 exclue car singleton).
- **Semi-régulière** : les **deux couleurs les plus courtes** comptent
  exactement 2 cartes → 5-4-2-2, 6-3-2-2 — **et aussi 7-2-2-2**, que le
  commentaire du code ne mentionne pas (vérifié). Conséquence : une main
  7-2-2-2 de 20-21 H ouvre de **2SA** [O-3] et non de 2♣.
- Une main « équilibrée » au sens des redemandes à SA = régulière **ou**
  semi-régulière **et** couleur la plus longue ≤ 5 cartes — ce plafond de 5
  écarte bien le 7-2-2-2 des redemandes à Sans-Atout, mais **pas** de
  l'ouverture de 2SA, qui ne l'applique pas.

### [E-3] Couleur la plus longue
À égalité de longueur, la couleur **la plus haute** l'emporte
(♠ > ♥ > ♦ > ♣). ⚠ Ce départage arbitraire est utilisé partout, y compris
pour choisir la couleur d'un barrage.

### [E-4] « Belle couleur » (`GoodSuit`)
Au moins 5 cartes **et** (au moins deux honneurs parmi A R D V 10,
**ou** au moins 5 points H dans la couleur). Sert aux interventions, aux
barrages, aux bicolores Michaël et aux rencontres.

### [E-5] Arrêt (`Stopper`)
A, Rx, Dxx ou Vxxx.

### [E-6] Double arrêt (`DoubleStopper`)
AR(x), ADx, RDxx ou Axxxx. Exigé quand les adversaires ont **nommé et
soutenu** une couleur (deux enchères ou plus dans cette couleur) : un arrêt
simple y saute et le reste de la couleur tombe. Exigé aussi par le **3SA** en
réponse au contre d'appel [D-3].

### [E-6b] Arrêt et demi (`StopperAndHalf`)
Un arrêt [E-5] renforcé par un **second honneur** (parmi A R D V 10) ou par
une **quatrième carte** — AVx, RVx, Rxxx — sans atteindre les deux arrêts
indépendants de [E-6], qui le satisfait aussi. C'est le prix du **2SA** en
réponse au contre d'appel [D-3].

### [E-7] Sécurité Sans-Atout (`ntSafe`)
On peut proposer Sans-Atout si, dans **chaque** couleur nommée par les
adversaires, on possède un arrêt (ou le partenaire en a garanti un) — et un
**double** arrêt si cette couleur a été nommée au moins deux fois.

### [E-7b] Ce que les enchères adverses dévaluent
Un **singleton honneur** (R, D ou V) dans une couleur **nommée par les
adversaires** ne reçoit **pas** les 2 points de courte que le HLD accorde :
leur As est au-dessus, l'honneur tombe dessous, et ce qui reste est un
singleton ordinaire dont la valeur de coupe est déjà payée par l'honneur
lui-même. Un singleton **petit** garde son crédit, et un singleton honneur
dans une couleur que personne n'a nommée aussi — l'As peut encore être en
face. Appliqué là où la différence se joue : la redemande de l'ouvreur après
un soutien [RO-9 à RO-13], où deux points font passer du barrage à l'essai de
manche.

### [E-8] Fit
Un fit est un total de **8 cartes ou plus** dans une couleur, en additionnant
sa propre longueur et la longueur **promise** par le partenaire. À égalité,
les majeures priment ; ensuite le fit le plus long.
⚠ Un fit 4-4 non annoncé n'existe donc pas pour le moteur tant que le
partenaire n'a pas promis 4 cartes.

### [E-9] Seuils de manche et de chelem
| Objectif | Compte combiné |
|---|---|
| Manche 4♥/4♠ (fit majeur) | 27 HLD |
| Manche 3SA | 25 H (honneurs purs) |
| Manche 5♣/5♦ (fit mineur) | 30 HLD |
| Essai par contrôles | 29 à 32, chelem en vue (max ≥ 33 ou min ≥ 30) |
| Petit chelem | 33 |
| Grand chelem | 37 |

Le compte combiné = sa propre main (HLD si fit, H pur pour Sans-Atout) plus
le **plancher** annoncé par le partenaire, jamais son maximum. Le maximum ne
sert qu'à une chose : vérifier qu'un chelem est **arithmétiquement possible**
avant de démarrer une sonde par contrôles ([S-1]) — jamais à conclure.

Une conclusion à la manche annonce son plancher **dans l'unité de la manche
choisie** : en H pur pour 3SA, en HLD pour une manche à la couleur. Un 3SA qui
annonçait un plancher HLD (la longueur d'une mineure comptée en points)
gonflait le compte combiné du partenaire et l'envoyait au chelem à SA sur une
force qui n'y était pas.

Ce calcul vaut aussi pour les **échelles conventionnelles** : les paliers
d'une réponse à 2SA, d'un développement du Stayman ou d'un transfert après les
deux majeures ne sont pas des nombres fixes, mais **33 moins le plancher promis
par l'ouverture** — 18 HLD face au 1SA de 15-17, 13 face au 2SA de 20-21, 11
face à la redemande 2SA (22-23) d'un 2♣ fort. Un seuil figé conclut à la manche
avec 33 points réunis sur la table.

Et là où le décalage de palier ne laisse plus de place — le **Stayman sur 2SA**,
dont la réponse occupe déjà le palier de 3 — l'enchère **artificielle** de
chelem est abandonnée : elle tomberait sur la manche ou juste sous elle, sans
rien laisser à contrôler. La main passe alors par la route générique
(contrôles [S-3], puis clefs), qui a encore de la place sous la manche.

Le seuil retenu est celui de **la manche réellement disponible**, et cela vaut
aussi bien pour la conclusion que pour la proposition et son acceptation :
avec un **fit mineur** et Sans-Atout jouable [E-7], la manche visée est 3SA et
le seuil est **25** ; sans les arrêts, il ne reste que 5♣/5♦ — onze levées — et
le seuil remonte à **30**. Un fit majeur joue toujours 4M : **27**.

### [E-9c] Fit majeur : 25 H suffisent aussi
Le seuil de **27 HLD** en majeure compte la distribution des deux mains, et le
plancher du partenaire a souvent déjà compté la sienne (soutien 12-16 **HLD**) :
une main plate n'y ajoute que sa propre courte et reste sous un seuil que les
seuls honneurs atteignent. Avec un fit majeur de huit cartes, **25 H** combinés
(ses honneurs + le plancher du partenaire) donnent aussi la manche.
`1♣ – 1♥ – 2♥` avec A842 AQ94 J6 QT6 : 13 + 12 = 25 → **4♥**, et non une
proposition.

### [E-9d] Le neuvième atout vaut un point
Dès que le fit **connu** — ses propres atouts plus la longueur que le
partenaire a promise — atteint **neuf cartes**, la décision de manche
(conclusion, proposition, acceptation d'une proposition) compte **+1 HLD**.
Mesuré sur 5 000 donnes en double-mort : une manche en majeure passe les 50 %
de réussite vers **28 HLD** (deux mains) avec huit atouts, vers **27** avec
neuf ; un dixième n'ajoute rien de mesurable, ses points de longueur étant
déjà dans le HL. Validé hors échantillon sur le banc de non-régression
(tools/par/README.md) : **+127 IMP** sur les 5 000 donnes de la graine
20261001, qui n'avaient servi à aucun réglage (57 donnes gagnées, 43 perdues),
+241 IMP sur les 12 000 donnes du banc. Le bonus ne vaut pas pour le chelem, dont le compte
(sans l'appoint des courtes face aux longues du partenaire) reste inchangé.

### [E-9b] La limite du partenaire est déjà la proposition
Quand l'enchère qui proposerait la manche est **celle que le partenaire vient
de faire** — sa redemande limite 2SA (18-19), ou sa proposition 3M dans le fit
majeur — elle ne peut pas être répétée, et la question n'est plus de proposer
mais d'accepter. Hors forcing de manche, la décision se prend sur le **milieu
de sa fourchette** (arrondi vers le haut) : la main plus ce milieu au seuil de
la manche [E-9] → la manche ; sinon passe. `1♣ – 1♠ – 2SA` avec six points :
6 + 19 = 25 → **3SA**, là où le moteur passait faute de pouvoir « proposer »
2SA une seconde fois.

---

## 3. Les ouvertures

Testées dans cet ordre ; la première qui s'applique gagne.

### [O-1] 2♦ — forcing de manche
**HL ≥ 24.** Artificiel, ne promet rien à Carreau. Engage le camp à la manche.

### [O-2] 2♣ — fort indéterminé
**18-23 H**, et (une couleur d'au moins 6 cartes **ou** 21 H et plus),
et **pas** une main de 20-21 H régulière ou semi-régulière (qui ouvre de 2SA).
Forcing.

### [O-3] 2SA
**20-21 H**, régulière ou semi-régulière.

### [O-4] 1SA
**15-17 H, régulière** (semi-régulière exclue : un 5-4-2-2 de 16 H ouvre d'1 à
la couleur).

### [O-5] 3SA — mineure septième affranchie
Exactement 7 cartes dans une mineure, contenant **A, R et D**, sans aucun
autre A ni R, et au plus une D extérieure.

### [O-6] Ouverture d'1 à la couleur
**H ≥ 12 ou HL ≥ 13.** La couleur choisie :
- ♠ si 5+ cartes et la plus longue — **sauf** exactement 5♠-5♣ avec 14 H et
  plus, qui ouvre d'**1♣** ;
- sinon ♥ si 5+ cartes et la plus longue ;
- sinon la mineure la plus longue ; à 3-3 → **1♣** ; à 4-4 et à 5-5 → **1♦**.

L'ouverture majeure promet **5 cartes**, l'ouverture mineure **3 cartes**.
Fourchette annoncée : 12-23 HL.

### [O-6b] Bicolore majeur à 11 H
Avec **11 H** et les deux majeures **5-4 ou 4-5**, on ouvre d'**1 dans la
majeure cinquième** (la plus longue), en toute position : la main tient les
deux couleurs qui achètent la partielle. Le 5-5 n'a pas besoin de cette règle :
à 11 H il vaut déjà 13 HL et s'ouvre par [O-6]. L'ouverture promet 5 cartes
dans la majeure nommée et 4 dans l'autre ; fourchette annoncée : 11-23.

### [O-7] Barrages
Seulement avec **5 à 10 H**, et à condition que **la moitié au moins des points
H de la main soit dans la couleur du barrage** (`SuitH×2 ≥ H`) : une main aux
points dispersés ne barre pas.
- **7 cartes**, belle couleur, sans 4 cartes dans une autre majeure → **3X**,
  avec 5-10 H.
- **8 cartes ou plus** → **4X**, avec 5-10 H (sans condition de qualité ni
  d'autre majeure).

### [O-7a] Le 2 majeur faible
Le barrage aux paliers de 3 et de 4 achète son palier avec la seule longueur.
Le palier de 2 n'achète presque rien : il se paie en qualité de main, et les
conditions y sont donc plus lourdes.

On ouvre d'un 2 majeur faible avec **6 à 10 H** et **les quatre conditions**
suivantes, sans dérogation :

- **une couleur convenable de 6 cartes**, **D109xxx au minimum** : deux des
  cinq gros honneurs avec au moins 3 H dans la couleur (DV, R10, AV et mieux),
  ou la Dame avec le 10 et le 9. V109xxx a la forme, pas la couleur ;
- **pas 4 cartes dans l'autre majeure**, que le partenaire ne retrouverait
  jamais ;
- **pas de mineure cinquième** : une mineure quatrième est tolérée, une
  cinquième carte fait de la main un bicolore, qui n'est pas ce que l'enchère
  dit ;
- **moins de deux levées de défense extérieures, ni deux As** : un As ou un
  Roi hors de la couleur compte une levée. Avec deux, la main défend mieux
  qu'elle ne barre.

Ces conditions suffisent à tenir la défense hors de la main : le test « la
moitié des points dans la couleur longue » ne vaut que pour les barrages aux
paliers de 3 et de 4. À 11 H, six cartes valent 13 HL : l'ouverture d'1 prend
la main [O-6].

**Pas de 2 faible en quatrième position** : trois passes ont fait le tour, il
n'y a plus personne à barrer, et la main qui ouvrirait d'1 passe.

### [O-7b] Ouverture légère de troisième et de quatrième
Deux passes (ou trois) ont précédé : le partenaire est **déjà passé**, il n'y a
donc **pas de manche à chercher**, et pas de manche à manquer. Ce qui reste à
gagner, c'est l'**indication d'entame** et le palier qu'on coûte aux
adversaires — qui sont, eux, ceux qui ont les points. La règle des 20 des deux
premiers sièges [O-6] ne s'applique plus telle quelle.

Sous le seuil de [O-6], avec **10 H et plus** :
- **troisième siège** — on ouvre **pour l'entame** : couleur **cinquième**
  (celle que désigne [O-6]), **belle** (deux des cinq gros honneurs, ou 5 points
  H dans la couleur), et la force de la main **dedans** plutôt qu'éparpillée
  (`SuitH×2 ≥ H`, la même concentration que réclament les barrages [O-7]). Une
  main dont les points sont ailleurs égare l'entame qu'elle prétend indiquer,
  et défend mieux qu'elle n'ouvre ;
- **quatrième siège** — il n'y a plus d'entame à indiquer : passer la donne
  marque zéro, ce qui vaut mieux qu'un score négatif. On n'ouvre que sur la
  **règle des 15** : **H + nombre de piques ≥ 15**, le Pique étant ce qui
  décide qui achète la partielle au palier de 1. L'ouverture est naturelle
  [O-6], mineure troisième comprise.

Zone annoncée : **10-11**. Par construction [O-6] a déjà pris toute main de
12 H ou 13 HL, donc le plafond est réel ; le partenaire, plafonné lui aussi par
son passe d'ouverture [E-1b], lit un total combiné qui n'invite jamais.

Les barrages [O-7] gardent la priorité : ils décrivent six ou sept cartes à un
palier que l'ouverture d'1 n'achète pas.

⚠ **La contrepartie** : ayant emprunté un Roi qu'elle n'a pas, l'ouverture
légère **ne reparle jamais d'elle-même**. Le partenaire étant déjà passé, rien
de ce qu'il dit au palier de 1 ou de 2 n'est forcing, et le second tour est
précisément l'endroit où l'emprunt se paie. Elle passe, sauf enchère forcing du
partenaire (Drury [RM-2b], rencontre [RM-2]) : elle répète alors sa couleur au
palier le moins cher — c'est le « non » que le Drury demandait.

### [O-8] Passe
Tout le reste.

> ⚠ **Trou connu** : dans les deux premiers sièges, une main de 11 H et 12 HL
> avec une couleur sixième médiocre ne peut ni ouvrir d'1 [O-6] ni barrer
> [O-7] : elle passe. Le 2 faible s'arrête à 10 H [O-7a], et il réclame
> de toute façon une couleur convenable, que cette main-là n'a pas — et l'ouverture légère [O-7b],
> qui la réclame aussi, ne la rattrape pas davantage en troisième.

---

## 4. Réponses aux ouvertures de 1 à la couleur

### 4.1 Réponses à 1♥ / 1♠ (silence adverse)

Dans l'ordre :

### [RM-1] 4 cartes à ♠ sur 1♥
Avec **4+ cartes à Pique, 6 HL et plus, et moins de 3 cartes à Cœur** → **1♠**,
forcing. Avec un fit ♥ troisième, on ne détourne pas par Pique.

### [RM-1b] Le 2 sur 1 de la main déjà passée n'est pas forcing
Une main qui a **passé** ne peut pas engager le camp à la manche : son passe la
plafonne à 11 [E-1b], et l'ouverture qu'elle affronte est de troisième ou de
quatrième, donc peut-être légère [O-7b]. Onze en face de dix ne fait pas une
manche. Son **changement de couleur au palier de 2** nomme donc simplement le
contrat que le camp jouera le mieux (11 H), **sans rien forcer** : l'ouverture,
seule à savoir si elle était légère, est libre de passer. Le Drury [RM-2b] est
la seule question qu'une main passée garde le droit de poser, et elle la pose
avec un fit.

### [RM-2] Rencontre du répondant déjà passé
Main ayant **passé avant l'ouverture**, 4+ atouts : saut d'un palier dans une
belle couleur cinquième, **8-11 H** — forcing. (Pas de rencontre pour une main
non passée.)

### [RM-2b] Drury (main déjà passée, ouverture de 3e ou 4e)
Main ayant **passé avant l'ouverture**, face à **1♥/1♠**, dans le **silence
adverse**, avec un **fit** et **11 HLD et plus** :
- **fit troisième**, ou **fit quatrième sans singleton** → **2♣**, artificiel et
  forcing ;
- **fit quatrième avec exactement une courte** → **2SA**, artificiel et forcing.

L'ouverture de troisième ou quatrième pouvant être légère, la question est
« as-tu tes points ? », et elle se pose **sous 2M**, avant que le compte brut du
fit n'achète un palier que l'ouverture ne peut pas payer.

⚠ La **rencontre** [RM-2] garde la priorité sur le Drury : elle montre le fit
**et** une source de levées d'un seul coup, ce que la demande artificielle ne
sait pas faire.

### [RM-2c] Redemandes de l'ouvreur sur le 2♣ Drury
Les zones se lisent en compte combiné [E-9] contre les **11 HLD** promis :
| Ouvreur (HLD dans sa majeure) | Redemande | Sens |
|---|---|---|
| ≤ 14 | **2M** | ouverture minimale, on s'arrête là |
| 15-16 | **2♦** | ambition de manche, artificiel : décris ton soutien |
| 17-18 | **4M** | je veux jouer la manche, pas de chelem |
| ≥ 19 | **3M** | ambition de chelem, contrôles [S-3] |

Avant cette échelle, un **beau bicolore** 5-4 se nomme au palier de 2 dans
l'autre majeure (2♥ sur 1♠ à partir de 15, 2♠ sur 1♥ à partir de 17 puisqu'il
achète le palier au-dessus du refuge). Une mineure annexe n'a pas de place : 2♣
était la demande et 2♦ est l'essai.

### [RM-2d] Réponses du répondant
- Sur **2♦** : le soutien se précise par sa longueur — **2M** avec 3 cartes,
  **3M** avec 4, **4M** avec 5 (sécurité distributionnelle). La réponse porte
  aussi la valeur réelle de la main, que le plancher de 11 sous-estime.
- Sur **2SA** puis **3♣** (« quel est ton singleton ? », posé à partir de
  15 HLD, sinon l'ouvreur s'arrête à **3M**) : **3♦** = singleton ♦, **3♥** =
  singleton ♥, **3♠** = singleton ♠, et le **retour à 3 dans la couleur
  d'ouverture** = singleton ♣, puisque 3♣ était la question.

Toute intervention adverse annule le schéma : 2♣ n'est alors pas une couleur à
soutenir, et la décision générique [§12] reprend la main.

### [RM-3] Splinter
**4+ atouts, 13-15 HLD**, un singleton ou une chicane, **pas** de belle couleur
annexe cinquième et **une seule** courte → double saut dans la couleur courte.
Engage le camp à la manche.

### [RM-4] Avec 3 atouts et plus
Dans l'ordre :
- **13 HLD et plus** → changement de couleur d'abord (couleur la plus longue de
  4+ cartes ; 2♥ sur 1♠ exige 5 cartes ; à défaut une mineure de 3 cartes),
  forcing. Si aucune couleur n'est annonçable → décision générique [§12], et en
  dernier recours **3M**.
- **11-12 HLD avec exactement 3 atouts** → **2SA conventionnel** (forcing).
- **11-12 HLD avec 4 atouts et plus** → **3M**, proposition.
- **6-10 HLD** → **2M**.
- moins de 6 → passe.

### [RM-4b] Le fit différé
Après le **changement de couleur avant soutien** au palier de 2 (réponse 2 sur
1 fittée, d'une main non passée, dans le silence adverse), le répondant dit
l'atout **au palier de 3**, quelle que soit la redemande de l'ouvreur :
`1♠ – 2♣ – 2♦ – 3♠`. L'enchère est **forcing de manche** et laisse l'espace du
chelem : seul l'ouvreur sait de combien il dépasse son minimum, il lance les
contrôles avec une main de chelem ou conclut à la manche avec un minimum.
Sauter à la manche (`4♠`) dirait au contraire « rien à ajouter » et fermerait
la porte. Pas au palier de 2 : `2♠` y est la simple préférence.
Sur intervention, ou si l'ouvreur a déjà dépassé 3 de la majeure, la décision
générique reprend la main.

### [RM-5] Sans fit, 11 HL et plus
Changement de couleur **2 sur 1** dans la meilleure couleur → **forcing de
manche** (le camp est engagé).

### [RM-6] Sans fit, 6-10 HL
**1SA** « poubelle », avec **6 H au moins** : les points de longueur viennent
d'une couleur que le partenaire ignorera, et à Sans-Atout une longue sans
honneur ne fait pas de levée. D2 2 8764 R109753 (5 H, 7 HL) passe sur 1♠.

### [RM-7] Moins de 6 HL
Passe.

### 4.2 Réponses à 1♣ / 1♦ (silence adverse)

### [Rm-1] Moins de 6 HL → passe.

### [Rm-2] Majeure quatrième — priorité absolue
Avec 4+ cartes dans une majeure : **1♥/1♠**, forcing.
Choix : si une majeure a 5 cartes ou plus, la plus longue (♠ à égalité) ;
sinon ♥ si 4 cartes, sinon ♠.
⚠ Cette règle passe **avant** tout soutien et avant 1SA : une main 4-4 en
majeures avec 5 cartes à l'ouverture nomme quand même une majeure.

### [Rm-3] Rencontre dans l'autre mineure
Saut d'un palier, 4+ cartes dans la mineure d'ouverture, belle couleur
cinquième dans l'autre mineure, **8-11 H**.

### [Rm-4] Sur 1♣ avec 4 cartes à ♦ → **1♦**, forcing.

### [Rm-5] Main régulière sans majeure quatrième
6-10 HL → **1SA** · 11-12 HL → **2SA** (proposition) · 13-15 HL → **3SA**.

### [Rm-6] 13 HL et plus, main irrégulière
- 4+ cartes dans l'autre mineure → changement de couleur forcing.
- Sinon, jusqu'à 18 HL, **avec un arrêt dans les trois autres couleurs** →
  **3SA**.
- Sinon → décision générique [§12].

### [Rm-7] Soutien de la mineure
Il faut **5 cartes à ♣** (ou 4 à ♦) : 11-12 HLD → **3m** (proposition) ;
6-10 HLD → **2m** (dénie une majeure quatrième).

### [Rm-8] 11 HL et plus avec 4 cartes dans l'autre mineure
Changement de couleur forcing.

### [Rm-9] Défaut → **1SA** (6-10 HL).

### 4.3 Réponses après intervention adverse

### [RC-1] Sur notre 1SA → Rubensohl [§8.6].

### [RC-1b] La majeure non nommée d'abord
Une majeure que ni l'ouverture ni les adversaires n'ont nommée se dit
**avant** la rencontre et le soutien, comme en silence [Rm-2] : soutenir la
dénierait, alors que le fit se trouve le plus souvent là. Le soutien de la
majeure d'ouverture à 3 cartes garde la priorité — ce fit-là est déjà connu.

| Longueur | Enchère | Plancher |
|---|---|---|
| 5 cartes et plus | la couleur, au palier de 1 ou de 2 | 6 HL au palier de 1, **11 HL** au palier de 2 |
| exactement 4 | **Contre Spoutnik** [RC-6] | 6 HL, 8 HL sur une intervention au palier de 2 |
| exactement 4, sans contre disponible (intervention par **Contre**) | la couleur au palier de 1 | 6 HL |

Choix entre les deux majeures : la plus longue dès cinq cartes (♠ à
égalité), ♥ avec deux majeures quatrièmes.
`1♣ – 1♠ – ?` avec ♠7 ♥DT942 ♦ARV ♣ADV6 → **2♥**, pas 2♣ : cinq cartes.

### [RC-6] Le contre Spoutnik
Sur l'intervention à la couleur d'un adversaire (palier de 1 ou de 2), le
**Contre** du répondant montre **exactement 4 cartes** dans la ou les
majeures que personne n'a nommées — la longueur qu'une enchère naturelle ne
peut plus promettre, puisque cinq cartes la nomment. Il demande 6 HL, et
**8 HL** sur une intervention au palier de 2, car l'ouvreur devra répondre au
palier de 2.

**L'ouvreur ne passe jamais** : le contre est une enchère, pas une punition.
- 4 cartes dans la majeure annoncée → il la nomme : au plus bas avec 12-16
  HLD, **à saut** avec 17-18, **à la manche** à partir de 19 ;
- sinon il décrit : Sans-Atout avec l'arrêt adverse (3SA à partir de 18 H),
  répétition d'une couleur d'ouverture sixième, ou une couleur quatrième.

`1♣ – 1♠ – Contre` avec ♠7 ♥DT94 ♦ARV2 ♣ADV6 : 4 Cœurs exactement.

### [RC-2] Rencontre en compétition
- Après un **Contre** sur une mineure : seulement l'autre mineure, saut d'un
  palier.
- Après un **Contre** sur une majeure : n'importe quelle couleur, saut d'un
  palier.
- Après une **intervention à la couleur** : double saut — sauf si
  l'intervention est déjà une majeure au palier de 2 sur une ouverture mineure,
  où un simple saut suffit.
- Une rencontre **en mineure s'arrête au palier de 4** : le saut est réduit à ce
  qu'il faut pour y arriver (1♠ 2♦ → **4♣**, pas 5♣). Le message — fit et belle
  couleur cinquième, 8-11 H — tient entièrement à 4m, alors que 5m vend un
  palier dont le camp aura besoin et dépasse la manche en majeure vers laquelle
  le fit se dirige le plus souvent. S'il ne reste plus de saut possible sous 4m,
  il n'y a pas de rencontre.

### [RC-3] Soutien en compétition
Fit requis : 3 cartes en majeure, 4 en mineure.
- **13 HLD et plus** → décision générique directe [§12] (pas de proposition
  qu'un ouvreur minimum passerait).
- **Loi des levées totales** : 9 atouts communs autorisent le palier de 3,
  10 atouts le palier de 4, quel que soit le nombre de points **au-dessus du
  plancher de réponse**. Ce plancher écrase le plafond en points s'il est plus
  haut, jamais le plancher [L-1].
- 11 HLD et plus → soutien invitationnel (ou soutien « loi » s'il est plus
  haut).
- 6-10 HLD → soutien simple, porté au palier de la loi s'il est plus haut.
- **Moins de 6 HLD → passe**, fit long compris.

### [RC-4] Changement de couleur en compétition
- Palier de 1, 4+ cartes, 6 HL et plus → forcing.
- Palier de 2, 5+ cartes, 11 HL et plus → forcing (on choisit la **plus
  longue**, pas la moins chère).
- 8-10 HL avec un arrêt dans la couleur adverse et 1SA disponible → **1SA**.
- 11 HL et plus sans rien d'autre → décision générique.
- Sinon passe. Ce passe **dénie 11 HL, rien de moins** : tant que 1SA reste
  disponible au palier de 1, une main de 8-10 avec un arrêt l'aurait dit, donc
  le passe est plafonné à **7** ; dès que l'intervention porte le Sans-Atout au
  palier de 2, une main de 8-10 sans couleur quatrième annonçable au palier de
  1 n'a plus aucune enchère et passe elle aussi — le plafond annoncé est alors
  **10**, sous peine de faire manquer au partenaire des manches que le camp
  détient.

### [RC-5] 3SA sur la couleur longue du partenaire
Le partenaire a **répété sa couleur au palier de 3** en compétition — donc
6 cartes promises — et nous en avons **2** : le fit est huitième, la couleur
tournera une fois la main prise. Avec un **arrêt dans chaque couleur nommée
par les adversaires** et **20 points combinés** (les nôtres plus le plancher
du partenaire), on conclut à **3SA**.

Le compte seul ne trouve jamais cette manche : les sixième et septième cartes
du partenaire sont des levées que l'échelle des honneurs ne voit pas, d'où un
plancher cinq points sous les 25 exigés ailleurs pour 3SA [E-9]. Un arrêt
**simple** suffit, là où [E-7] réclamerait un double arrêt d'une couleur
nommée deux fois : la couleur du partenaire s'affranchissant d'elle-même, il
n'y a pas de coup à blanc à jouer ni de main à rendre.

---

## 5. Réponses à 1SA et 2SA

### [N-1] Texas majeur
Main **unicolore** majeure : 5+ cartes dans une majeure **et moins de 4** dans
l'autre. Sur 1SA : 2♦ (♥) / 2♥ (♠). Sur 2SA : 3♦ / 3♥.
⚠ Un 5-4 majeur ne fait **pas** de Texas : il passe par le Stayman.

### [N-2] Misère dorée (sur 1SA seulement)
Main de **7-8 H** avec une majeure cinquième **et un vrai singleton** (pas une
chicane) : on passe par le **Stayman** au lieu du Texas, pour trouver le fit
neuvième avant de s'engager. Suites en [§8.5].

### [N-3] Stayman
4+ cartes dans une majeure et **8 H** (sur 1SA) / **4 H** (sur 2SA) → 2♣ / 3♣.

### [N-3b] Bicolore majeur 5-5 (sur 1SA seulement)
5 ♥ **et** 5 ♠ avec **9 H et plus** → **4♦**, forcing de manche : l'ouvreur
nomme sa meilleure majeure au palier de 4, et le répondant passe. L'ouvreur
d'1SA est régulier, il ne peut donc être court dans les deux majeures : sa
plus longue en fait un fit de **8 cartes au moins**, et la manche se joue
mieux là qu'à Sans-Atout. À longueur égale (3-3), ce sont les **honneurs** qui
tranchent, puis Cœur, la manche la moins chère.
⚠ Sous le seuil, l'échelle ordinaire s'applique : **Stayman** à partir de 8 H
[N-3], sinon l'échelle Sans-Atout [N-6]. Le seuil se lit en **H**, comme celui
du Stayman, et non en HL : un 5-5 porte toujours 2 points de longueur, qui
diraient la même chose de toutes les mains.

### [N-4] Texas mineurs (sur 1SA seulement)
2♠ = Trèfle, 3♣ = Carreau. Conditions : 6+ cartes dans la mineure avec
**HL ≤ 7** (main faible), **ou** 10 HL et plus **avec un singleton/chicane**.
Un 5-5 mineur passe toujours par le Texas Trèfle et exige 10 HL.
⚠ Entre les deux, la zone **8-9 HL ne fait pas de Texas, courte ou pas** :
trop forte pour la fuite, qui achète neuf levées à 3♣ là où 1SA n'en demande
que sept, et trop faible pour la manche. Elle prend l'échelle Sans-Atout
[N-6]. Le seuil se lit en **HL**, jamais en HLD : la distribution de la courte
ne se compte qu'une fois le fit trouvé.

### [N-5] Bicolore mineur fort (sur 2SA, ou sur la redemande 2SA d'un 2♣/2♦)
3♠ = Texas Trèfle, 4♣ = Texas Carreau. Exige **11 HL et plus** et soit un 5-5
mineur, soit une mineure sixième avec une courte.

### [N-6] Échelle Sans-Atout quantitative
Sur **1SA** : ≤7 H passe · 8-9 **2SA** (proposition) · 10-15 **3SA** ·
16-17 **4SA** quantitatif · 18+ **6SA**.
Sur **2SA** : ≤3 passe, puis l'arithmétique du combiné [E-9] plutôt qu'un
palier fixe — **le compte atteint 33 face au plancher** → route générique du
chelem (demande de clefs [S-8]) · **33 face au seul plafond**, et zone
d'ouverture **étroite** (amplitude ≤ 7 : le plancher de 24 non plafonné du 2♦
n'en est pas une) → **4SA quantitatif** · sinon **3SA**. Face à 20-21 cela
redonne 4-11 puis 12+ ; face à la redemande 2SA d'un 2♣ (22-23), 4-9 puis 10.
⚠ Le commentaire affiché sur le passe annonce « 0-8H » alors que le seuil réel
est **0-7 H** : simple erreur de libellé.

### [N-7] Réponses de l'ouvreur au Stayman
Sur 1SA : 2♦ = pas de majeure quatrième · 2♥ = 4 ♥ sans 4 ♠ · 2♠ = 4 ♠ sans
4 ♥ · 2SA = les deux. Sur 2SA, même échelle décalée (3♦/3♥/3♠/3SA).

### [N-7b] Suites du répondant fitté après le Stayman (sur 1SA)
Le fit trouvé (l'ouvreur a montré la majeure quatrième du répondant), le
compte combiné décide [E-9] : la manche en majeure demande **27 HLD**, mesurés
face à la fourchette de l'ouvreur (15-17).
- **12 HLD et plus** (la manche face au minimum) → **4M**.
- **10-11 HLD** (la manche seulement face au maximum) → **3M**, proposition ;
  l'ouvreur accepte quand son compte atteint 27, passe sinon.
- **Moins de 10 HLD** → **Passe** : même face à 17, la manche n'y est pas.

Les zones de chelem (splinter, convention 2012) restent au-dessus.
`1SA – 2♣ – 2♠ – 3♠ – Passe` avec ♠AV94 ♥92 ♦764 ♣RV75 (10 HLD) face à
♠RD107 ♥AR4 ♦R95 ♣1062 (15 H, régulier).

### [N-8] Rectification du Texas majeur
Rectification au plus bas. **Super-accept** (saut) avec **4 atouts** et un
maximum (au moins 2 H au-dessus du plancher annoncé), uniquement au palier
de 2.

Sur le super-accept, le répondant conclut à la manche dès 5 H, **sauf si un
chelem est en vue** au sens de [S-1] : il **lance les contrôles** (29 et plus
avec son HLD face au maximum de l'ouvreur). Le super-accept s'arrête à 3M et
laisse libres toutes les marches jusqu'à la manche : les contrôles n'y
coûtent rien, alors que toutes les réponses à 4SA sont déjà au-dessus de 4M.
Le passage au Blackwood se fait ensuite depuis l'échange [S-4].
`1SA – 2♦ – 3♥` avec ♠R105 ♥RV943 ♦9 ♣ADV6 (17 HLD) → **4♣**, puis 4♦ – 4SA
– 5♣ – 6♥.

### [N-9] Rectification du Texas mineur
Trèfle : avec un bon fit (4+ ♣, ou Rx, ou Dxx) l'ouvreur **refuse** par 2SA ;
sinon il rectifie. Carreau : rectification à 3♦ obligatoire.
Sur un Texas faible, le répondant qui reçoit 2SA **revient à 3♣** — arrêt
absolu.

### [N-10] Suites après la rectification du Texas mineur
Le répondant :
- **main faible** → **Passe** (arrêt dans la mineure) ;
- **courte**, annoncée **naturellement dans les majeures** : **3♥** =
  singleton/chicane ♥, **3♠** = singleton/chicane ♠, **3SA** =
  singleton/chicane dans **l'autre mineure** ;
- **bicolore 5-5 mineur** → **3♦**, forcing de manche ;
- pas de courte à montrer → **3SA**.
⚠ C'est bien la couleur *courte* qui est nommée, et non le « meilleur résidu »
(l'autre majeure) : `1SA – 3♣ – 3♦ – 3♥` montre une courte à **Cœur**.

---

## 6. Réponses aux ouvertures fortes et de barrage

### [F-1] Sur 2♣ — relais 2♦ obligatoire
Le répondant dit toujours **2♦** (forcing), puis :
- **Fit 3+ dans la couleur redemandée**, ≤7 HL :
  - en **majeure** → **4M**. Toute la zone tient sur cette enchère parce que
    cette enchère *est* la manche.
  - en **mineure**, la manche est un palier plus haut, donc la zone se scinde
    sur les **points d'honneurs** — ce qui manque à l'ouvreur, ce sont deux
    levées, et la longueur dans une couleur annexe face à un unicolore n'en
    fournit presque jamais : **5 H et plus → 5m**, la manche que ses neuf
    levées [O-2] et nos deux valent ; **moins de 5 H → soutien au plus bas**,
    en annonçant **0-4**, une zone que l'ouvreur peut juger.
- **Fit 3+**, 8 HL et plus → **3 de la couleur** (4 si l'ouvreur a déjà pris le
  palier de 3), espoir de chelem, forcing.
- Sinon, couleur cinquième personnelle et 5 HL → couleur la moins chère,
  forcing.
- Sinon **2SA** d'attente, forcing — mais c'est une enchère de **0-7** : elle
  dit « rien à décrire », et l'ouvreur se réévalue contre ce plafond. Deux
  mains ne doivent donc pas l'emprunter, et nomment **3SA** à la place :
  - **8 HL et plus** : annoncer un plafond qu'on n'a pas, c'est ce qui fait
    redemander l'ouvreur en main minimale ; et 8 en face des 18 promis atteint
    déjà la manche à Sans-Atout. Le 3SA annonce alors **8-11** — le plafond
    compte autant que le plancher, sans lui un jeu de 8 se lit comme un
    possible 15 et l'ouvreur part chercher un chelem qui n'existe pas. La main
    qui détient vraiment plus n'a pas de moyen de le dire ici : la route du
    chelem dans cette séquence, c'est le soutien fitté, forcing et illimité.
  - **quelle que soit la force, si l'ouvreur a redemandé au palier de 3**
    (couleur sixième, neuf levées de jeu à lui seul [O-2]) : il n'y a plus
    rien à attendre. Soutenir sa mineure sur deux cartes s'arrêterait sous
    toutes les manches.

  Sans-Atout injouable [E-7] → soutien de sa couleur, faute de mieux.
- Si l'ouvreur a redemandé **2SA** → on repart sur l'échelle de réponses à 2SA
  avec un plancher d'ouvreur à 22 H.

### [F-2] Sur 2♦ — réponses aux As
| Réponse | Signification | Plancher |
|---|---|---|
| 2♥ | pas d'As — 0-7 H, **ou 8 H et plus avec une courte** | non plafonné |
| 2SA | pas d'As, 8 H et plus, **aucune courte** | 8+ |

⚠ Le 2♥ **ne doit pas annoncer de plafond** : le palier couvre deux mains très
différentes, et une zone haute annoncée est irréversible — une enchère
ultérieure peut relever un plancher, jamais abaisser un plafond déjà posé. Le
répondant se précise lui-même ensuite (son 3SA le ramène à 4-11), ce qui rend
la proposition quantitative de nouveau calculable.
| 2♠ | l'As de ♥ **ou** de ♠ | 4+ |
| 3♣ | l'As de ♣ | 4+ |
| 3♦ | l'As de ♦ | 4+ |
| 3♥ | deux As de même **couleur** (♣+♠ ou ♦+♥) | 8+ |
| 3♠ | deux As de même **rang** (♣+♦ ou ♥+♠) | 8+ |
| 3SA | deux As mélangés — **ou trois, ou quatre** | 8+ (12+ si 3-4 As) |

### [F-3] Sur 3SA (mineure affranchie)
Le répondant **passe**, sauf avec **14 H et plus et une courte** : il nomme
**4♣** (ou 5♣ à partir de 17 H) comme ancre — « jouons là, quelle que soit
votre mineure ». L'ouvreur passe si sa couleur est Trèfle, sinon rectifie à
Carreau au même palier.

### [F-4] Sur un 2 faible majeur
- **4 atouts et plus** → **4M** directement, « attaque-défense », **quel que
  soit le nombre de points**.
- 2-3 atouts, **18-20 HLD** → **4M** directement.
- 2-3 atouts, **15-17 ou 21+ HLD** → **2SA** relais-fitté, forcing.
- 13 HL et plus avec une belle couleur dans l'autre majeure → changement de
  couleur forcing.
- Exactement 3 atouts : rencontre en mineure si possible, sinon **3M**
  (prolongement du barrage).
- 16 HL et plus, main régulière → **3SA**.
- Sinon passe.

### [F-5] Réponses de l'ouvreur au relais 2SA (2 faible)
- ≤ 11 HLD → **3M** (minimum).
- 12-14 HLD : d'abord une force extérieure (**A ou R dans une couleur de 3+
  cartes**) au palier de 3 ; sinon un **singleton** annoncé au palier de 4
  (après 2♥, un singleton ♠ se montre par **4♥**) ; sinon **3SA**.

### [F-6] Face au barrage du partenaire (règle générale)
Si le partenaire est plafonné à 10 et qu'on tient **16 H et plus** : 4M en
majeure (avec 2 atouts au moins), ou 3SA sur une mineure sixième avec arrêt
partout.

### [F-6b] Face au barrage à 3 en majeure : 3 atouts → 4M
Face au barrage à 3 en majeure (7 cartes), **3 atouts et plus** font **10
atouts** : la **loi des levées totales** place le camp au palier de 4, **quel
que soit le nombre de points**. Main faible, **4M** prolonge le barrage ; main
correcte, c'est la manche — la même enchère, comme les 4 atouts sur le 2 faible
[F-4]. Le compte combiné seul ne la trouve pas : 13 H face à un barrage de 5 à
10 n'atteignent jamais 27 HLD.
À partir de **16 H**, ou avec moins de 3 atouts, la règle générale [F-6] et la
machinerie du chelem reprennent la main.
`3♠ – Passe – 4♠` avec ♠1054 ♥R ♦A93 ♣AD8752 (13 H) face à ♠ARV9862 ♥1096 ♦6
♣V9.
⚠ Les barrages à 3 en **mineure** ne sont pas concernés : 4m par la loi y
passerait au-dessus de 3SA, que [F-6] propose avec les arrêts.

---

## 7. Redemandes de l'ouvreur

### 7.1 Après une réponse 1SA (1M – 1SA, 1m – 1SA)

Dans l'ordre :
### [RO-1] Majeure septième auto-suffisante
Majeure d'ouverture de **7 cartes** dont le HLD plus le plancher du partenaire
atteint le seuil de manche → **4M** directement.

### [RO-2] 18 H et plus, main équilibrée → **3SA**.
### [RO-3] 17-18 H, main équilibrée → **2SA**, proposition.
### [RO-4] 6 cartes et 17 HL et plus → **répétition à saut** (17-19), invite.
### [RO-5] 6 cartes → **répétition simple** (13-16).
### [RO-5b] Bicolore cher, 18 HL et plus
4+ cartes dans une couleur **plus chère** que l'ouverture, 18 HL et plus →
**2 de cette couleur**, forcing (18-23 HL), comme [RO-19] : l'ouverture y est
enregistrée à 4 cartes au moins. Sans lui, `1♦ – 1SA` avec ♠ARV5 ♥ADV7 ♦AD96 ♣5
(20 H) répétait ses ♦ « par défaut » en 12-14 [RO-7] et le répondant passait
sous la manche.
### [RO-6] Bicolore économique
Seulement avec une main **irrégulière** : 4+ cartes dans une couleur moins
chère que l'ouverture → 2 dans cette couleur (12-17).
Une main **régulière** ne nomme pas son 4-4 : elle passe.
### [RO-7] Répétition par défaut
Main irrégulière, 5 cartes à l'ouverture → **2 de la couleur** (12-14).
### [RO-8] Sinon **passe** (12-14, jeu régulier minimal).

Sur les répétitions de mineure [RO-4]/[RO-5], le répondant dispose de la
**demande d'arrêt** [§8.4].

### 7.2 Après un soutien du répondant

Fit simple (2M, 6-10 HLD montré) :
### [RO-9] 22 HLD et plus, majeure → **4M**.
### [RO-10] 17-21 HLD, majeure → **essai de manche** dans l'ordre : couleur
nécessitant un appui [§8.7], sinon **2SA** essai généralisé, sinon **3M**.
⚠ La simple répétition **3M** face à un soutien est un barrage, pas un essai.
### [RO-11] 17 HLD et plus, mineure → **3m**, essai.
### [RO-12] 12-16 HLD, majeure sixième → **3M**, barrage.
### [RO-13] Sinon passe.

Soutien à saut (3M, 11-12 HLD) :
### [RO-14] HLD + 11 ≥ 27 et majeure → **4M** ; sinon décision générique.

### 7.3 Après un changement de couleur du répondant

### [RO-15] Fit 4 cartes dans la couleur du répondant
- **20 HLD et plus**, majeure → **4** de cette couleur.
- Mineure dont le soutien à saut dépasserait 3SA : **22 HLD et plus** → saut
  (ambition de chelem, forcing) ; sinon **soutien au plus bas**, 3SA préservé.
- **17-19 HLD** → soutien à saut, invite.
- Sinon soutien simple (12-16).

### [RO-16] 18-19 H, main équilibrée → **2SA**.
Après une ouverture mineure et une réponse majeure au palier de 1, ce saut
arme le **Checkback 3♣** [§8.2].

### [RO-16b] Majeure sixième du répondant face au 2SA limite
Face à la redemande 2SA (18-19, équilibrée), une majeure **sixième déjà
nommée** par le répondant fait un fit de huit cartes au moins : la manche se
joue **4M**, et la sixième carte y est une levée, donc la main se compte en
**HL** ([E-9b], seuil 25). `1♥ – 1♠ – 2SA` avec Q98742 K8 JT86 5 : 7 HL + 19
→ **4♠**.

### [RO-17] 12-14 H, main équilibrée, réponse au palier de 1 → **1SA**.
Après 1m–1M et 1♥–1♠, ce 1SA arme le **Roudi 2♣** [§8.3].
Si 1SA est devenu illégal (intervention) : **2SA** à condition d'avoir un arrêt
dans chaque couleur adverse ; sinon on descend aux règles suivantes.

### [RO-18] 12-14 H équilibrée après une réponse 2 sur 1 → **2SA**, à condition
que Sans-Atout soit sûr [E-7].

### [RO-18b] Fit 3 cartes dans la majeure du répondant, en compétition
Réponse majeure au palier de 1 (donc forcing), fit **3 cartes**, adversaire
ayant parlé, et les descriptions équilibrées ci-dessus indisponibles : une main
avec des points ne peut pas passer une réponse forcing, elle soutient le fit
**4-3** connu. HLD [E-1] du fit, minoré des 2 points de singleton quand ce
singleton, dans leur couleur, est un honneur sec sans valeur défensive.
- **20 HLD et plus** → **4M**.
- **16-19 HLD** → **soutien à saut** (3M), essai de manche.
- **13-15 HLD** → **soutien simple** (2M), compétitif.
- Moins : on descend aux règles suivantes.

Même traitement après une **réponse 2 sur 1** : elle promet 11 HL et force
l'ouvreur à parler, donc le soutien au palier de 3 ne demande rien de plus
que l'ouverture (12-16 HLD), et le saut à la manche se fait à partir de
17 HLD.

### [RO-19] Deuxième couleur (4 cartes et plus)
- **Changement de couleur au palier de 1** (1♣ – 1♥ – 1♠) → montre la forme,
  dénie un saut ou un bicolore cher, **ne dit rien du niveau de points** :
  l'ouvreur précisera sa force au tour suivant. **Forcing un tour** face à un
  répondant non passé : il ne peut pas passer, et dispose toujours d'une
  enchère bon marché (1SA d'attente, préférence, soutien). Seul le passe lui
  est retiré ; l'enchère reste lue comme naturelle, propositions comprises.
  Face à un répondant passé, dont la réponse est déjà limitée, elle reste non
  forcing.
- **Bicolore économique** (palier de 2 dans une couleur moins chère que
  l'ouverture) → **12-17**, non forcing. Le partenaire est libre de le passer :
  l'enchère ne peut donc pas porter en plus les mains qui veulent l'entendre
  reparler.
- **Saut dans la seconde couleur** (les deux mêmes couleurs, un palier plus
  haut), **18 HL et plus** → forcing. Sans lui, un jeu de 19 et un jeu de 12
  faisaient la même enchère et le répondant n'avait aucun moyen de les
  distinguer. Exception : quand le **forcing de manche est déjà engagé** (une
  réponse 2 sur 1 d'une main non passée, dans le silence adverse, **fittée ou
  non** — le changement de couleur avant soutien de [RM-4] y compris :
  `1♠ – 2♣ – 2♦` et non `3♦` avec 18 HL), le saut n'a rien à distinguer — le partenaire ne peut pas
  passer l'enchère économique, qui porte alors toute la fourchette et
  n'annonce aucun plafond — et le palier qu'il dépenserait est celui dont
  l'exploration du chelem a besoin.
- **Bicolore cher** (palier ≤ 2), **18 HL et plus** → forcing. Il nomme sa
  seconde couleur au-dessus de la première, donc la première en compte au
  moins autant : **4 cartes** y sont enregistrées, et non les 3 que promet une
  ouverture mineure. Sans cela le partenaire évalue sa propre courte en face
  de la vraie couleur comme une valeur de coupe au lieu d'un gaspillage
  [E-1c].
- **Bicolore à saut** (palier ≤ 3), **20 HL et plus** → forcing de manche.
  Il passe **avant** le changement de couleur au palier de 1 : `1♣ – 1♦ – 2♥`
  et non `1♥` avec 20 HL. Le palier de 1, forcing un tour seulement, ne peut
  pas porter une main qui a la manche en face de n'importe quelle réponse.

### [RO-19c] Le bicolore économique allonge l'ouverture
La seconde couleur étant moins chère que la première, l'ouverture en compte au
moins autant : le bicolore économique (et son saut) enregistre **4 cartes**
dans la couleur d'ouverture, et non les 3 d'une ouverture mineure.
`1♦ – 1♠ – 2♣` promet quatre carreaux ; le répondant qui en a quatre et deux
trèfles trouve le fit de huit cartes (préférence ou proposition), au lieu de
passer dans un 4-2 à trèfle. La **préférence** pour la première couleur est
prise quand elle est le fit strictement meilleur ; à longueurs annoncées
égales (4-4), des longueurs égales chez le répondant laissent l'enchère où
elle est.

### [RO-19d] Refuser 2SA avec une main irrégulière
Sur la proposition 2SA du partenaire, l'ouvreur minimum qui a une **chicane**
(ou un singleton dans la couleur du partenaire) et une **sixième** déjà nommée
ne passe pas : il refuse en revenant à **3 de sa couleur**, non forcing.
`1♦ – 1♠ – 2♣ – 2SA` avec — K84 AT7654 AQT7 → **3♦** ; le Sans-Atout
partiel chute là où le carreau gagne.

### [RO-19b] Réponses au bicolore cher
Le bicolore cher est **auto-forcing**. Toutes les réponses sont **forcing de
manche**, à deux exceptions près, qui sont ce qui permet au camp de s'arrêter
sous la manche :
- **La répétition de la majeure du répondant quand elle est cinquième**, qui
  est prioritaire et garantit la cinquième carte ;
- **2SA modérateur** (« coup de frein »), **5-7 H**, quand aucune majeure
  cinquième n'est là.

Les deux sont **forcing pour un tour**. L'ouvreur y précise la force de son
bicolore : quand le maximum combiné — le partenaire désormais plafonné à 7 —
n'atteint pas le seuil, il se replie dans sa plus longue couleur au palier le
moins cher, en s'annonçant **18-19**, et le répondant est libre de passer.
Sans ce plafond le seul plancher de 18 ferait encore une manche face au haut
du frein.

Le frein est nécessaire parce que le compte générique lit les 18 du bicolore
cher comme un plancher, et conclut donc à la manche sur une main qui n'a rien :
`1♣ – 1♠ – 2♦ – 3SA` sur cinq points.

### [RO-20] Répétition à saut : 6 cartes et 17 HL → 17-19.
### [RO-21] Répétition simple : 5 cartes → 13-17.
⚠ Le moteur enregistre **5** cartes quand il n'en a que 5, même quand la
théorie en promet 6 : le partenaire ne compte jamais un fit fantôme.
### [RO-21b] Répétition **au palier de 3**, poussée par une intervention
Quand l'intervention a repoussé la répétition au palier de 3, elle reste
possible avec **6 cartes et plus** et **15 HLD** (mesurés contre leur
enchère). C'est le seul palier où une longue franche peut encore se dire :
les redemandes à Sans-Atout de [RO-18] et [RO-22] réclament un arrêt que la
main n'a pas forcément, et la deuxième couleur de [RO-19] n'existe pas.
Passer enterre une source de levées dont le partenaire n'entendra jamais
parler, et le laisse conclure à Sans-Atout une main qui appartient à la
couleur. Le prix se paie en **levées de jeu**, pas en honneurs : `1♦ – 1♠ –
(2♥)` avec ♦RDVT732 et l'As de Trèfle vaut **3♦**, sur quoi le partenaire
peut nommer 3SA avec l'arrêt Cœur.

### [RO-22] Défaut : Sans-Atout le moins cher (palier ≤ 2) si Sans-Atout est
sûr, 12-14. Sinon passe.

### 7.4 Redemandes après les ouvertures fortes

### [RO-23] Après 2♣ – 2♦
- Majeure sixième, ou majeure cinquième avec 21 H et plus → **2M**, forcing.
- Main (semi-)régulière de **22-23 H** → **2SA**.
- Couleur sixième → **3** de cette couleur, forcing.
- Couleur cinquième (main irrégulière) → **3** de cette couleur, forcing.
- Sinon (4-4-4-1) → **2SA**, 22-23.

**Jamais 3SA.** Aucun type de main ne reste pour cette redemande : l'équilibrée
de 22-23 dit 2SA, qui n'est pas une enchère à passer mais une marche — le
répondant y retrouve l'échelle des réponses à 2SA, Stayman et Texas compris
[F-1], toute cette place étant sous la manche. Sauter à 3SA la brûlerait pour
rien. Et une main équilibrée au-dessus de 23 H n'ouvre pas de 2♣ : à partir de
24 HL c'est 2♦ [O-1], où le 3SA existe bel et bien comme redemande par défaut
[RO-24].

### [RO-24] Après 2♦
- Couleur de 5 cartes et plus → couleur la moins chère, forcing de manche.
- Sinon Sans-Atout le moins cher. **Si ce Sans-Atout est déjà à la manche**
  (réponse au palier de 3) et que les quatre As sont localisés, et qu'**un
  seul Roi du partenaire suffirait à atteindre 33 H**, l'ouvreur pose
  **4SA appel aux Rois** au lieu de conclure [§11.4].
- **Les deux planchers additionnés valent 33 H** (l'ouverture et celui promis
  par le palier des As, par exemple 25 H face au 2SA qui garantit 8 H) →
  **6SA** directement ; **37 H** → **7SA**. Conclure à 3SA enterrerait la
  donne, le répondant passant une manche que le camp dépassait d'emblée.
- Sinon redemande par défaut (forcing tant qu'elle est sous la manche).

### [RO-25] Réveil de l'ouvreur (le répondant a passé sur une intervention)
18 H et plus, intervention au palier ≤ 3 → **Contre**, forcing.
Sinon 6 cartes → répétition au palier ≤ 3. Sinon passe.

---

## 8. Conventions du camp de l'ouvreur

### 8.1 Splinter
Voir [RM-3] (réponse) et [§8.5] (après Stayman, 15-17 HLD). Le camp est
engagé à la manche ; la suite passe par les contrôles [§11].

### 8.2 Checkback Stayman (1m – 1M – 2SA)
### [C-1] Le répondant dit **3♣** avec une majeure **cinquième** ou 4 cartes
dans l'autre majeure — sauf si le compte combiné atteint déjà 33 (on garde
alors la route générique du chelem).
### [C-2] Réponses de l'ouvreur : **3♦** = 3 cartes chez vous **et** 4 dans
l'autre · **3M (la vôtre)** = 3 cartes sans 4 dans l'autre · **3 autre M** =
4 dans l'autre sans fit troisième · **3SA** = ni l'un ni l'autre.
### [C-3] Conclusion : 4M sur le fit 5-3, 4 dans l'autre majeure sur le fit
4-4, sinon 3SA.
### [C-4] Sans forme pour le Checkback, une main régulière de **14 HL et plus**
dont le total reste sous 33 propose **4SA quantitatif**.

### 8.3 Roudi (1m – 1M – 1SA, et 1♥ – 1♠ – 1SA)
### [C-5] **2♣** avec **exactement 5 cartes** dans sa majeure et **11 H et
plus** (et un total combiné sous 33).
### [C-6] Réponses, **trois paliers** : **2♦** = 2 cartes, minimum **ou**
maximum · **2♥** = 3 cartes et minimum (12 H) · **2♠** = 3 cartes et maximum
(13-14). Sans le troisième atout, la zone n'est pas dévoilée : il n'y a pas de
fit à soutenir, et l'ouvreur reste dans les 12-14 de sa redemande à 1SA, où le
répondant peut le proposer.
### [C-7] Conclusion : avec 11 H face à **2♥** (fit et minimum), arrêt au
palier de 2 ; face à **2♦**, zone de l'ouvreur encore ouverte, **2SA**
proposition — l'ouvreur passe au minimum, conclut à 3SA au maximum. Sinon
**4M** sur le fit 5-3, **3SA** sans fit.

### [C-7b] **Un appel annoncé n'est jamais détourné.** Les deux redemandes à
Sans-Atout mettent une enchère au service d'une convention, et l'alertent :
**3♣** après le saut à 2SA [C-1], **2♣** après la redemande à 1SA [C-5]. Le
partenaire y lira la question, quoi que l'on ait voulu dire — aucune autre
enchère ne l'emprunte, et surtout pas un **contrôle**, qui ouvrirait
l'exploration du chelem sur un malentendu. Avec une **majeure sixième** et
l'ambition du chelem, on **répète sa majeure**, forcing : l'atout est fixé
avant les contrôles, et le premier d'entre eux se dira au tour suivant,
au-dessus de l'appel réservé. Sinon l'exploration ne démarre pas, et le compte
[§12] place le contrat.

### 8.4 Demande d'arrêt après répétition d'une mineure
### [C-8] Après 1m – 1SA – 2m/3m, le répondant qui a **au moins 2 points de
plus que son minimum annoncé** nomme, **au même palier**, une majeure où il
n'a **pas** d'arrêt. Avec les deux arrêts, il dit Sans-Atout directement.
### [C-9] L'ouvreur répond Sans-Atout au palier demandé avec l'arrêt, sinon il
**revient à sa mineure** au plus bas.
### [C-10] Après la dénégation, le contrat se juge au seul compte : passe,
**5m** à partir du seuil de manche — Sans-Atout étant exclu par la dénégation,
c'est **30** [E-9] — et **6m** à partir de 33.

### 8.5 Misère dorée (docs/addon_11.md)
### [C-11] Après le Stayman de misère dorée [N-2] :
- l'ouvreur montre **la même majeure** → **4M** (fit neuvième) ;
- l'ouvreur montre **les deux majeures** → Texas mineur au palier de 4 pour
  que l'ouvreur reste déclarant ;
- l'ouvreur nie (2♦ ou l'autre majeure) → on montre la misère dorée par
  **2M** si la majeure est encore disponible, sinon par **2SA**.
### [C-12] Sur ce 2SA, l'ouvreur maximum (2 H au-dessus de son plancher) dit
**3M** forcing avec 3 cartes, sinon **3SA** ; minimum, il passe. Sur le 3M
forcing, le répondant conclut **4M**.

### 8.6 Rubensohl (intervention à la couleur sur notre 1SA)
### [C-13] **Contre** = positif, 6 HL et plus, au moins 2 cartes dans
l'intervention, aucune couleur cinquième — « tendance Stayman ».
### [C-14] Couleur au palier de 2 **au-dessus** de l'intervention, 5+ cartes,
0-7 HL = naturel faible, non forcing.
### [C-15] À partir de **2SA**, échelle de Texas (♣, ♦, ♥, ♠) à partir de
8 HL. La marche qui tomberait sur l'intervention devient le **Texas
impossible** : singleton ou chicane dans l'intervention, demande les majeures.
### [C-16] Avec 10 HL et plus et rien d'autre : **3SA** avec l'arrêt (ou avec
un 4-3-3-3 dont la quatrième est mineure), sinon **3♠** = demande d'arrêt
générique pour 3SA, quelle que soit la couleur d'intervention.
### [C-17] Réponses : au Contre, la majeure quatrième la moins chère qui n'est
pas l'intervention ; sinon passe (punitif avec arrêt et 17 H et plus).
Au Texas impossible et à la demande d'arrêt, même logique : majeure quatrième,
sinon 3SA avec l'arrêt, sinon cue-bid de l'intervention pour nier.

### 8.7 Essai « couleur nécessitant un appui »
### [C-18] Après un soutien majeur simple, l'ouvreur de 17-21 HLD nomme une
couleur annexe de **2 à 4 cartes sans A ni R** au palier ≤ 3.
### [C-19] Le partenaire accepte **seulement avec une aide réelle** : A ou R,
ou D troisième, ou une courte (≤ 2 cartes) — pas sur le seul compte de points.
### [C-20] L'essai doit laisser la place de refuser : la couleur nommée doit
avoir **trois de l'atout au-dessus d'elle**. Un « essai » à 3♠ sur un fit à
Cœur ne laisse au partenaire sans aide que 4♥ ou le passe — et le passe fait
jouer l'essai lui-même, dans une couleur que personne n'a convenue. En silence
la question ne se pose pas (l'essai à Pique sort à 2♠) ; en compétition, si les
adversaires ont pris la place, il n'y a plus d'essai à faire et l'échelle passe
à la suite [RO-10].
Si un essai arrive malgré tout au-dessus de l'atout, le refus **retourne dans
la couleur au palier de 4** : c'est une manche non voulue, mais jamais aussi
mauvais que de jouer l'essai.

### 8.8 Quatrième couleur forcing
D'après la fiche « La Quatrième Couleur Forcing » .

### [C-21] Le camp a nommé **trois couleurs**, aucun fit n'est apparu : nommer
la quatrième — celle que personne ne peut vouloir jouer — est une **demande de
renseignement**, éventuellement purement artificielle, **à alerter**. Elle
demande **10 H et plus**. Elle est **forcing un tour** sur un bicolore
économique, et **forcing de manche** sur un bicolore cher, où le camp est
engagé de toute façon.

### [C-22] On ne demande que s'il y a une vraie question :
- **exactement 5 cartes** dans sa majeure — le fit troisième que la réponse au
  palier de 1 n'a pas pu montrer (une sixième se répète, elle se passe de
  soutien) ;
- ou **pas d'arrêt** dans la quatrième couleur — Sans-Atout a besoin de celui
  du partenaire.

Sans l'un ni l'autre, la main se décrit elle-même et n'a rien à relayer.

⚠ La convention lit **trois couleurs naturelles** et en déduit la quatrième.
Elle est donc désarmée dès qu'une des enchères du camp est artificielle
(ouverture de 2♣ et ses relais, Drury, Checkback…) : là, une couleur nommée ne
promet aucune longueur et la déduction serait fausse. Le moteur exige que
chaque couleur nommée par le camp l'ait été **en promettant sa longueur**.

### [C-23] **Quatrième couleur à Pique.** Au palier de 1, **1♠ reste naturel** :
avec quatre piques on les nommerait. La demande passe alors au **2♠
« impossible »**, qu'aucune main naturelle ne choisirait au-dessus du palier
de 1 disponible.

### [C-24] Réponses, dans l'ordre :
1. **3 cartes dans la majeure du demandeur** → on la soutient au plus bas,
   **à saut avec 17-19 HLD**. Si la première couleur du demandeur est une
   mineure, cette marche n'existe pas : la demande porte alors sur l'arrêt.
2. **5-4-3-1 avec l'As troisième dans la quatrième couleur** → on la
   **soutient**, conventionnellement. Le singleton interdit de proposer
   Sans-Atout, et l'As troisième est le seul jeu qui rende jouable une couleur
   que personne n'a nommée. Le soutien reste **forcing** : c'est une
   description, pas un contrat — le demandeur peut y être chicane.
3. **Arrêt dans la quatrième couleur** → Sans-Atout au plus bas avec le
   minimum, **3SA à partir de 15 H**. La zone est celle de la main, jamais
   celle du palier : quand les enchères sont déjà hautes, le minimum atterrit
   sur le 3SA que le maximum aurait sauté et ne s'en attribue pas les points.
4. Sinon la forme : **bicolore 5-5** → répétition de la deuxième couleur ;
   **couleur sixième** → répétition de la première.

### [C-25] La suite se joue au compte [§12], qui dispose maintenant de ce qui
manquait : le fit 5-3 majeur révélé par la première réponse, ou l'arrêt qui
autorise la manche à Sans-Atout.

### 8.9 Troisième couleur forcing
D'après la fiche « La 3ème Couleur Forcing ».

### [C-26] Une seule séquence y mène : l'ouvreur d'une **mineure** a **répété
sa couleur au palier de 2** (1m - 1M - 2m). Cette redemande le plafonne et
refuse en un seul coup le soutien à quatre atouts et la deuxième couleur ;
elle ne dit rien, en revanche, de sa **troisième carte** dans la majeure.
Nommer alors la troisième couleur est une **demande de renseignement**,
artificielle, **à alerter**, et **forcing de manche**. Elle promet
**exactement 5 cartes** dans la majeure déjà nommée — celle que la réponse au
palier de 1 n'a pas pu montrer — et **rien du tout** dans la couleur qu'elle
nomme.

### [C-27] La troisième couleur est toujours la **« collante »** de
l'ouverture : **♦ après 1♣**, **♥ après 1♦**. **♠ n'est en aucun cas une
troisième couleur forcing** : avec quatre piques on les aurait nommés au
palier de 1. Et quand la collante est justement la couleur que le répondant a
déjà nommée (1♦ - 1♥ - 2♦), le camp n'a plus de troisième couleur à sa
portée : la demande n'existe pas.

### [C-28] On ne demande qu'avec **11 H et plus** — la fiche dit « forcing de
manche » sans chiffrer le plancher, le moteur retient celui du 2 sur 1 — et
jamais d'une main qui a **passé d'entrée** : elle ne peut pas engager la
manche [G-4]. Le **fit mineur** que la répétition de l'ouvreur crée si souvent
n'y fait pas obstacle : c'est précisément le contrat à éviter, onze levées
quand neuf à Sans-Atout ou dix dans la majeure restent à trouver. Un **fit
majeur**, lui, aurait déjà tout réglé.

### [C-29] Triple question, à laquelle on répond **dans cet ordre**, et
toujours **au palier de 3** : la manche étant acquise, il n'y a plus de
partielle à protéger en dessous.
1. **3 cartes dans la majeure du demandeur** → on la **soutient** : c'est le
   fit 5-3 que toute la demande cherche.
2. **4 cartes dans la troisième couleur quand elle est majeure** → on la
   **nomme** : rien n'interdit au répondant d'avoir quatre cartes dans une
   couleur qu'il n'a pas promise, et le 4-4 vaut mieux que le Sans-Atout.
3. **Arrêt dans la troisième couleur** → **3SA**.
4. **4 cartes dans une troisième couleur mineure, sans l'arrêt** → on la
   nomme : ce n'est pas un contrat, c'est la longueur qui répond à la question
   posée — l'arrêt devra venir d'en face.
5. Sinon **répétition de la couleur d'ouverture**, le seul jeu qui reste.

### [C-30] Quand la réponse a dénié à la fois le soutien et l'arrêt, le
Sans-Atout ne tient plus que si le **demandeur** détient l'arrêt lui-même ;
sans lui, le camp — engagé à la manche — joue **5m** dans son fit mineur. Pour
le reste la suite se joue au compte [§12], qui dispose maintenant de ce qui
manquait : le fit 5-3 majeur, ou l'arrêt qui autorise le Sans-Atout.

---

## 9. Le camp de la défense

### 9.1 Interventions directes

Dans l'ordre, sur une ouverture adverse :

### [I-1] Réveil classique
Si l'ouverture adverse d'1 à la couleur revient après **deux passes**
(une seule enchère dans tout l'étui) → barème du réveil [§9.3].

### [I-2] Bicolore Michaël précisé (sur une ouverture d'1)
Toujours **5-5 au moins**, jamais 5-4, **9 HL et plus, sans limite
supérieure**. Les deux couleurs sont de qualité [E-4], **ou** l'une l'est et
les honneurs sont dans le bicolore (**6 H et plus** dans les deux couleurs) :
sur 1♣, AR1065 V9762 V5 4 dit **2♦**, pas 1♠ qui enterrerait les Cœurs ;
V9653 R10953 (4 H dans les couleurs) reste trop maigre.
- Sur une **majeure** : cue-bid = l'autre majeure + ♣ · **3♣** = l'autre
  majeure + ♦ · **2SA** = les deux mineures.
- Sur une **mineure** : **2♦** = les deux majeures · **2SA** = ♥ + l'autre
  mineure (un bicolore avec ♠ se dit naturellement).

### [I-3] Contre « toutes distributions »
**18 H et plus**, intervention au palier ≤ 4. Aucune promesse de forme.

### [I-3b] Landy (sur l'ouverture adverse de 1SA)
Intervention du joueur assis **juste après l'ouvreur de 1SA** : **2♣** annonce
en une seule enchère — donc de façon très économique — un **bicolore majeur au
moins 5-4** (4 cartes dans chaque majeure, 9 au total) et **une dizaine de
points** : **10-18 HL**. Le plafond vient de l'ordre des règles : à partir de
18 H la main est déjà partie par le contre [I-3].

L'enchère n'existe **que dans ce siège et sur cette ouverture** : plus loin
autour de la table, le 1SA a déjà reçu une réponse et la donne n'est plus celle
que la convention traite. Ce que le partenaire enregistre, c'est **4 cartes
dans chacune des deux majeures** — la cinquième existe, mais on ne sait pas
encore dans laquelle.

**2♣ est réservé au Landy dans ce siège** : l'intervention naturelle à Trèfle
n'y existe pas, un unicolore Trèfle passe. Le même camp ne peut pas jouer les
deux systèmes sur la même enchère, et le partenaire répondrait dans une
majeure. Le prix est réel — sur 20 000 donnes, le Landy parle 89 fois et le
2♣ naturel qu'il supprime aurait été dit 126 fois — mais un bicolore majeur
annoncé en une enchère vaut mieux qu'une partielle à Trèfle. Deux passes plus
loin, en réveil, le 2♣ naturel redevient disponible [I-5].

### [I-4] Intervention à 1SA
**16-18 H, main régulière, arrêt** dans la couleur adverse, sur une ouverture
au palier de 1. L'arrêt n'est pas qu'une condition : il fait partie de ce que
l'enchère **dit**, et il est donc **enregistré** — sans quoi l'arithmétique
Sans-Atout du partenaire refuserait le contrat faute d'un arrêt déjà promis.

La zone est un point au-dessus de celle de l'**ouverture** de 1SA (15-17), et
c'est voulu : l'intervention se fait devant une main qui a déjà annoncé
l'ouverture, donc avec les points adverses localisés à sa gauche.

Elle est essayée **avant le contre « toutes distributions »** [I-2], dont le
plancher de 18 H chevauche son plafond. Le contre n'y promet que des points ;
l'intervention à 1SA nomme en une enchère une forme, un arrêt et une fourchette
de deux points, et une main qui y entre n'a rien à gagner à la description la
plus vague. Au-dessus de 18, le contre reprend la main.


### [I-4b] 3SA sur un barrage de 3
Sur une **ouverture de barrage au palier de 3** : **16-20 H**, un **arrêt**
dans leur couleur, **sans chicane** → **3SA**, naturel. À partir de 18 H, une
main irrégulière avec un seul arrêt préfère le contre [I-2]. Le barrage a pris la
place de toute séquence descriptive, et l'arrêt dans la couleur du barreur est
justement ce que son partenaire ne peut pas avoir : la manche est là en face
de quelques points. Essayé **avant** le contre 18 H et plus, qui laisserait le
partenaire deviner la dénomination au palier de 4. `3♦ – ?` avec K854 AT5
KQJ42 A → **3SA** (le moteur passait).
### [I-5] Intervention naturelle à la couleur
Belle couleur d'au moins 5 cartes, avec **deux honneurs** au moins (A, R, D,
V ou 10), jamais une couleur nommée par les adversaires — ni **Trèfle dans le
siège du Landy** [I-3b], où 2♣ est conventionnel :
- palier de 1 → **9-18 HL et au moins 8 H** : les points de longueur seuls ne
  font pas une intervention. `(1♣)` avec ♠RV1076 ♥103 ♦D87654 ♣— (6 H, 9 HL)
  → **passe** ;
- palier de 2 → **11 HL et plus**, et (6 cartes **ou** 14 HL) ;
- palier de 3 → **12 HL et plus** et 6 cartes.

### [I-5b] Sur leur ouverture de 1SA : **6 cartes**, sans dispense
L'intervention naturelle à la couleur sur l'ouverture adverse de 1SA exige
**six cartes**, et aucune force d'honneurs n'en tient lieu — la clause « ou
14 HL » de [I-5] ne s'applique pas là. L'ouvreur a annoncé 15-17 réguliers :
les points sont chez eux, l'intervention n'est pas une proposition de contrat
mais un acte de gêne, et elle se paie en **levées de jeu**. Une cinquième au
palier de 2 devant une main qui connaît sa force à deux points près est
exactement ce qu'un contre punitif attend ; la sixième carte est ce qui
transforme les mêmes honneurs en levées qui y survivent.

⚠ Conséquence assumée : une main de 15 H avec une belle cinquième se tait
désormais sur 1SA. Le moteur n'ayant pas de **contre punitif de 1SA**, elle
n'a plus rien à dire — c'est le prix de la règle, pas un oubli.

### [I-5c] Cinquième sans les honneurs, mais avec l'ouverture
Au **palier de 1** seulement, une couleur cinquième qui n'a pas les **deux
honneurs** de [I-5] parle quand même dès **12 H**. La « belle couleur » est ce
qui paie une enchère faite sur la seule distribution ; une main qui tient
l'ouverture possède sa part de la donne et ne peut pas être condamnée au passe
faute d'un valet. `(1♦)` avec ♠V9754 ♥ADx6 ♦Rx ♣Dx (12 H) → **1♠**.

L'essai est fait **après** le contre d'appel [I-6], qui garde donc la priorité
quand sa distribution est réunie ; avec une cinquième en majeure il est de
toute façon interdit, et le choix n'est alors pas entre deux descriptions mais
entre **nommer la couleur et se taire avec une ouverture**.

Palier de 1 seulement : plus haut, ce sont la sixième carte et la qualité de
la couleur qui rendent l'enchère sûre, et aucune des deux n'est promise ici.

### [I-6] Contre d'appel, 12-17 H
Pour intervenir à la couleur au palier de 1 il faut une couleur cinquième ; le
contre d'appel est ce qui permet d'intervenir **sans** couleur cinquième et de
chercher le meilleur contrat pour la défense. Sur une ouverture au palier de 1,
avec la forme requise :
- sur une **mineure** : **12 HL et plus**, 3 ou 4 cartes dans **chaque**
  majeure, total **≥ 7** (donc 4-3 ou 4-4 ; jamais 2, jamais 5). Le **4-3-3-3
  reste valable** : seule une vraie longueur (4 cartes et plus) dans la mineure
  d'ouverture interdit le contre ;
- sur **1♥** : **12 H et plus**, 4 cartes et plus à ♠, **court (≤ 2 cartes)**
  dans la couleur d'ouverture ;
- sur **1♠** : **12 H et plus**, 4 ou 5 cartes à ♥, **court (≤ 2 cartes)** dans
  la couleur d'ouverture.

**Interdiction de déroger.** On ne contre pas parce qu'on a l'ouverture : sans
les quatre cartes promises dans l'autre majeure, le partenaire déclarera un
contrat qui chute. Passer avec un jeu fort est bien moins dangereux que de
promettre une distribution que l'on n'a pas — et ce passe n'est pas définitif,
le joueur placé devant l'ouvreur pourra encore s'exprimer en réveil [§9.3].

Certitudes de fit qui en découlent : sur une ouverture mineure, le contre
promet 3 cartes dans chaque majeure, donc 4 cartes chez le partenaire assurent
un total de 7, 5 cartes un fit huitième et 6 cartes un fit neuvième ; sur une
ouverture majeure, les 4 cartes promises dans l'autre majeure donnent le fit
huitième dès 4 cartes en face.

Sur une enchère au palier de 2 : 12-17 H, au plus 2 cartes dans leur couleur,
3+ partout ailleurs.

### [I-6b] Contre d'appel sur un barrage de 3
Sur une **ouverture de barrage au palier de 3** : **15-17 H**, au plus 2 cartes
dans leur couleur, 3 cartes et plus partout ailleurs. Le partenaire répondra au
palier de 3 ou de 4 : le plancher de 12 du palier de 2 ne suffit plus. À partir
de 18 H, le contre « toutes distributions » [I-2] s'applique.

### [I-7] Sinon passe.

### 9.2 Réponses au contre d'appel — la règle des trois zones

L'avancée est **obligatoire** tant que l'adversaire de droite s'est tu ; s'il a
parlé par-dessus le contre, la main faible peut passer.

Trois zones : **0-7 H** (faible), **8-10 H** (moyenne), **11-14 H** (forte).

L'échelle des réponses en majeure est ancrée sur la manche, pas sur l'enchère
la moins chère : **4 cartes au palier de 2, 5 cartes au palier de 3, 6 cartes
à la manche**. Derrière 1♣, 1♦ ou 1♥ cela donne les quatre paliers habituels —
sans saut de 0 à 7 H, saut simple (4 cartes), double saut (5) et triple saut
(6) en zone moyenne. Derrière **1♠** un palier manque : **2♥ n'est plus un
saut et couvre 0-10 H**, 3♥ montre cinq cartes en zone moyenne et 4♥ six.

### [D-1] Majeure septième → **4M**, quel que soit le nombre de points
(le contre garantit 3 atouts : fit dixième au moins, loi des levées totales).
### [D-2] Majeure sixième et 8 H et plus → **4M** : nommer une manche majeure
face au contre demande **six cartes en zone moyenne, cinq en zone forte**.
### [D-3] 11 H et plus : majeure cinquième → **4M** · sans majeure quatrième
et **main (semi-)régulière** : **3SA** avec 12-14 H et **deux arrêts**, **2SA**
avec 11 H et **un arrêt et demi** · sinon **cue-bid**, forcing et auto-forcing.
Au-delà de 14 H la main sort de la zone forte et passe elle aussi par le
cue-bid, plutôt que par une réponse qui la plafonnerait.
### [D-4] 8-10 H : les deux majeures quatrièmes sur une ouverture mineure →
cue-bid · majeure quatrième ou cinquième → l'échelle ci-dessus · sinon **1SA**
avec arrêt · sinon saut dans une mineure cinquième · sinon **1SA** « le moins
mauvais mensonge ».
### [D-5] 0-7 H : couleur **sans saut**, majeure quatrième avant mineure
cinquième — quitte à nommer une majeure troisième. Jamais 1SA, qui garantit
8-10 H : le partenaire dirait 2SA ou 3SA sur une main que l'on n'a pas. Toutes
les enchères à Sans-Atout sont positives et leur zone est précise.
### [D-6] Le cue-bid demande au contreur sa **majeure quatrième la moins
chère** ; à défaut, un arrêt dans la couleur adverse.
### [D-7] Sur 1♠, la réponse 2♥ couvrant 0-10 H, l'intervenant ne fait un
essai à **3♥** qu'avec **16 HLD** au moins et rien de perdu, notamment à ♠. Le
partenaire passe en zone faible et dit **4♥** en zone moyenne.

### 9.3 Le réveil (« Les enchères de réveil — après ouverture de 1 à la couleur »)

Quand l'ouverture adverse d'1 à la couleur revient après **deux passes**, un
passe de plus met fin à l'étui : les deux adversaires sont limités, et la force
manquante est chez le partenaire, qui n'a rien pu dire en position directe.
Trois familles se partagent le siège, et aucune ne veut dire ce que la même
enchère voudrait dire en intervention.

- **Le réveil par une couleur**, avec ou sans saut, **dénie une valeur
  d'ouverture**.
- **Le réveil à Sans-Atout** est limité : **9-13 HL** pour 1SA, **17-19 HL**
  pour 2SA, avec un arrêt dans la couleur d'ouverture — ou **trois petites
  cartes**, qui suffisent quand la force du partenaire est placée derrière
  l'ouvreur.
- **Le contre** provient de trois types de main très différents : la
  distribution du contre d'appel avec une **courte dans la couleur
  d'ouverture**, où **8 H suffisent** ; **la valeur de l'ouverture** (14 HL) et
  pas d'enchère à Sans-Atout possible ; le **jeu régulier de 14-16 HL**, qui
  fera suivre son contre d'une enchère à Sans-Atout au palier minimum.

| Réveil | Signification |
| --- | --- |
| Couleur au palier de 1 · `1♣ passe passe 1♠` | 5 cartes, **8-13 HL** (à partir de 14 HL, contre) |
| Couleur sans saut au palier de 2 · `1♠ passe passe 2♣` | idem |
| Majeure avec saut au palier de 2 · `1♦ passe passe 2♥` | barrage : **6 cartes**, l'équivalent d'un beau 2 faible |
| Majeure avec saut au palier de 3 · `1♥ passe passe 3♠` | barrage : **7 cartes**, l'équivalent d'une ouverture au palier de 3 |
| Mineure avec saut · `1♠ passe passe 3♦` | belle couleur **sixième à la limite de l'ouverture** (11 HL et plus) |
| **1SA** · `1♥ passe passe 1SA` | **9-13 HL** et un arrêt (ou trois petites cartes) dans la couleur d'ouverture |
| **2SA** · `1♦ passe passe 2SA` | **17-19 HL** et un arrêt dans la couleur d'ouverture |
| **Contre** · `1♠ passe passe contre` | les trois mains ci-dessus |

Les règles sont examinées dans cet ordre, et le premier qui répond l'emporte :
bicolores, 2SA, contre à partir de 14 HL, réveils à saut, couleur au palier de
1, 1SA, couleur au palier de 2, contre d'appel, couleur quatrième.

### [V-1] Bicolores de réveil : sur une majeure, cue-bid = autre majeure + ♣,
**3♣** = autre majeure + ♦ ; sur une mineure, **2♦** = les deux majeures, et
sur 1♣ le cue-bid = ♦ + ♥. (2SA est **naturel** en réveil, d'où le décalage.)
Ces enchères sont **réservées** : un réveil naturel ne les emprunte jamais,
le partenaire lirait le bicolore et nommerait une couleur que personne n'a.
### [V-2] **2SA** = 17-19 HL régulier avec arrêt.
### [V-3] **Contre obligatoire à partir de 14 HL** : la main possède
l'ouverture, et tout réveil par une couleur, plafonné à 13 HL, la
sous-évaluerait. Deux nuances :
- **jeu régulier de 14-16 H** avec un arrêt (ou trois petites) → contre, puis
  **Sans-Atout au palier le moins cher** sur la réponse minimale du
  partenaire. C'est cette deuxième enchère qui distingue la main du contre à
  8 H ; le contre seul ne la montre pas ;
- **belle couleur sixième** nommable au palier de 2 → on la nomme. Le contre
  demande au partenaire de choisir une couleur ; avec six cartes on n'a rien à
  demander, et sous-évaluer sa force coûte moins cher que promettre un soutien
  que l'on n'a pas dans trois couleurs.
### [V-4] **1SA** = 9-13 HL, arrêt **ou trois petites cartes** dans la couleur
d'ouverture. Une belle couleur cinquième nommable **au palier de 1** passe
avant : une couleur que le partenaire peut soutenir vaut mieux qu'un 1SA
limité. Une belle **majeure** cinquième passe avant **même au palier de 2** :
sur 1♠, V93 RV1053 AR10 64 réveille à **2♥**, pas à 1SA. Une mineure au
palier de 2, elle, reste derrière 1SA.
### [V-5] **Réveils à saut.** En majeure le saut est un **barrage** et rien
d'autre — 6 cartes au palier de 2, 7 cartes au palier de 3, la force dans la
couleur elle-même, l'ouverture déniée. En mineure c'est l'inverse : pas de
barrage à faire, mais une **belle couleur sixième à la limite de l'ouverture**,
**11 HL et plus**. Indisponible quand le saut est déjà pris par un bicolore
[V-1].
### [V-6] **Réveil par une couleur** : belle couleur cinquième, **8-13 HL** :
à partir de 14 HL, la main contre [V-3]. Elle peut valoir une ouverture
(12-13 H) ; le réveil ne la dénie pas, il la plafonne.
### [V-7] **Contre d'appel dès 8 H** avec la forme tricolore idéale : court
(≤ 2 cartes) dans leur couleur, 3 cartes et plus partout ailleurs.
### [V-8] Exceptionnellement, une **belle couleur quatrième** (5 H dans la
couleur) au palier de 1, à partir de 9 H.

### 9.4 L'avancée (le partenaire de l'intervenant)

### [A-1] Si les adversaires ont conclu à la manche et que nous avons un fit :
surenchère ou sacrifice [§10].
### [A-2] Après le contre d'appel du partenaire → [§9.2] ; après un **contre
de réveil**, dont la zone n'est pas la même → [R-3].
### [A-3] Après un bicolore Michaël : on choisit la **plus longue** des deux
couleurs (majeure à égalité), puis le palier suivant la loi des atouts —
13 HL ou 10 atouts → palier de 4 ; 10 HL ou 9 atouts → palier de 3 ; sinon 2.
Double misfit (≤ 2 cartes partout) et adversaires actifs → passe.
### [A-3b] Après un Landy [I-3b] : on nomme la **majeure la plus longue**,
**♥ à égalité** — et la réponse enregistre la longueur **réellement** détenue,
ce qui est précisément ce qui permet la suite : sur un 2♥ ne montrant pas plus
de **3 cartes** (choix par défaut, pas une préférence), l'intervenant qui a 5
cartes à ♠ et 4 à ♥ **rectifie à 2♠** — un fit de 8 cartes au prix d'un fit de
7. Sur un 2♥ montrant 4 cartes, il passe.
Le palier suit la loi des atouts, en comptant **un atout de moins** qu'après un
Michaël (seule la quatrième est promise) : 13 HLD ou 10 atouts → palier de 4 ;
10 HLD ou 9 atouts → palier de 3 ; sinon 2. La **zone annoncée** reste celle
réellement détenue (0-9 · 10-12 · 13 et plus HLD), même quand c'est la longueur
d'atout seule qui a poussé le palier.
Sans aucune majeure (2 cartes au plus dans chacune) et avec **5 Trèfles** :
**passe**, 2♣ se joue tel quel.

### [A-4] Après une intervention à 1SA
**Couleur sixième et belle** (deux gros honneurs) que les adversaires n'ont pas
nommée, et **HL + le plancher du partenaire ≥ 25** → on la **nomme**, au palier
le moins cher (≤ 3), **forcing**. Sinon → décision générique.

⚠ C'est la seule entorse au « H pur » du Sans-Atout [E-9], et elle est
motivée : la raison de ne pas compter la longueur à SA est qu'**une couleur
longue peut ne pas être franche** — or une sixième à deux gros honneurs est
précisément celle qui l'est, et elle vaut face à un 15-17 régulier les quatre
ou cinq levées que le compte en honneurs ne voit pas.

L'avancée nomme sa **couleur** et non Sans-Atout, car c'est en général elle qui
ne peut pas le dire : les adversaires ont parlé et elle n'a pas d'arrêt. C'est
l'intervenant — qui en a **garanti** un [I-4] et tient la main régulière
derrière — qui place le contrat. L'enchère décrit et rend la main ; elle ne
conclut jamais.
### [A-5] Après une intervention à la couleur (pour un **réveil** par une
couleur, voir [R-1] : les zones ne sont pas les mêmes) :
- 4+ atouts → rencontre possible (saut, belle couleur cinquième, 8-11 H) ;
- **avec le fit** : 13 HLD, **ou 11 H utiles**, et plus → **cue-bid de la
  couleur d'ouverture**, forcing [A-6], à défaut décision générique ·
  11-12 HLD → soutien à saut (plafonné au palier de 3) · 7-10 HLD → soutien
  simple. **Une avancée qui a
  déjà passé** — sur l'ouverture, ou d'entrée — ne fait **pas** le soutien à
  saut : le saut est une proposition, et le passe a déjà dénié les points
  qu'elle promet ; elle soutient au palier le moins cher et laisse le
  partenaire libre de passer ;
  - ⚠ **Les HLD se comptent en points « utiles ».** Les trois bandes sont
    celles de la fiche, et la fiche les assortit d'une condition : « avec des
    points *utiles* ». L'ouverture adverse a nommé une couleur, et le
    **singleton à honneur** qu'on y détient est le seul endroit où HLD paie
    deux fois pour rien — l'honneur tombe sous leur As, et la coupe a déjà
    été payée par l'honneur lui-même. Le moteur retire donc ce crédit de
    courte (`hldAgainstTheirBidding`, dont [E-1c] est le pendant tourné vers
    notre propre camp) avant de choisir la bande. Sans lui, une Dame sèche
    dans leur couleur vaut quatre points et fait franchir à la main la barre
    du cue-bid de force : sur 1♦ – 1♠ – passe, ♠R74 ♥AV32 ♦D ♣V8765 demandait
    la force de l'intervention et jouait la manche, là où la main vaut le
    soutien à saut et rien de plus.
  - ⚠ **Le fit se compte, il ne se suppose pas.** « 3 atouts et plus » est un
    compte pris face aux **5 cartes** que promet une intervention au palier
    de 1 : c'est le **fit de 8 cartes** qui est la vraie condition. Au palier
    de 2 ou de 3, où l'intervention promet **6 cartes** [I-5], un simple
    **doubleton** fait ce même fit huitième. Le moteur additionne donc sa
    longueur et celle qu'a promise le partenaire.
  - Les deux **soutiens** (7-10 et 11-12), eux, continuent d'exiger **3 atouts
    réels** : ce sont des enchères de compétition, leur sûreté est la longueur
    d'atout elle-même, et sur un doubleton ils achèteraient un palier que la
    loi des levées totales ne paie pas. La bande 13 HLD et plus n'est pas un
    soutien mais une **annonce de force** — face aux 6 cartes et 12 HL d'une
    intervention au palier de 3, le camp aligne deux ouvertures quelle que
    soit la répartition des atouts — et le fit huitième y suffit ;
- couleur propre belle et cinquième, 10-17 HL, palier ≤ 2 → naturel non
  forcing ;
- avec arrêt dans **toutes** les couleurs adverses : 11 HL et plus →
  **cue-bid** forcing ; 8-10 HL → **Sans-Atout** naturel au palier ≤ 2.
  Ce cue-bid-là est le seul qui se passe de fit : il vise 3SA, pas la couleur
  du partenaire, et c'est l'arrêt dans chaque couleur adverse qui le justifie.
- Sinon passe.

### [A-6] Cue-bid de force sur une intervention à la couleur au palier de 1
L'intervention couvre **9-18 HL**, une fourchette trop large pour que
l'avancée place le contrat seule : le **cue-bid de la couleur d'ouverture**
demande dans quelle moitié le partenaire se situe.

**Le fit est obligatoire : un fit de 8 cartes** — sa propre longueur plus
celle que l'intervention a promise, donc 3 cartes face à une intervention au
palier de 1, un simple doubleton face aux 6 cartes des paliers de 2 et 3
[I-5]. Sans ce fit, toutes les réponses codées — le retour à la couleur y
compris — posent le contrat dans une couleur que personne ne soutient : la
question ne vaut pas son prix.

Avec le fit, **deux portes d'entrée**, qui ne mesurent pas la même chose :
- **13 HLD** — de la force de jeu : les atouts et la courte feront les levées ;
- **11 H utiles** — le compte d'honneurs que le cue-bid annonce lui-même
  (« j'ai un fit avec toi et une main forte »). Une main qui les tient possède
  la moitié des honneurs du jeu face à une enchère qui peut couvrir 9 à 18 HL,
  et deviner dans quelle moitié se trouve le partenaire n'est pas une décision
  que l'avancée doit prendre seule — or le soutien à saut la prend, puisqu'il
  nomme la zone de manche et s'y plafonne.

En dessous des deux, le soutien simple ou à saut dit déjà la zone.

⚠ **Les H se comptent en points « utiles »**, comme les HLD. Un Roi ou une
Dame **sec dans la couleur d'ouverture** ne vaut pas ses points : il tombe sous
leur As. L'As sec, lui, garde tout — c'est un contrôle, et un contrôle se moque
de qui a nommé la couleur. Sans ce retrait, la main même pour laquelle la règle
des points utiles a été écrite — ♠R74 ♥AV32 ♦D ♣V8765, onze points d'honneur
imprimés — reviendrait droit au cue-bid.

**Le prix de la question, c'est sa pire réponse** : le retour du partenaire à
sa couleur. Sous la manche c'est une partielle qu'il peut passer, la question
est donc gratuite ; **la manche elle-même reste payable** — c'est là que le fit
et 13 HLD allaient de toute façon. Au-delà de la manche, la réponse achèterait
un palier que personne n'a demandé : la question ne se pose plus. C'est ce qui
autorise le cue-bid **au palier de 4** sur une intervention au palier de 3, où
le retour est exactement la manche. ⚠ Le cue-bid **sans fit** de [A-5], lui,
vise 3SA et non la couleur du partenaire : il reste plafonné au **palier de 3**.

L'avancée ne pose pas cette question-là à un **réveil** [V-6], qui a déjà dénié
l'ouverture : il n'y a plus rien à demander. Le cue-bid existe pourtant sur un
réveil, mais il y annonce au lieu d'interroger — c'est [R-4], et la réponse
codée y partage la fourchette 8-13 HL au lieu de 9-18.

La réponse est codée :
- **sans l'ouverture** (moins de 12 HL) → **retour à la couleur
  d'intervention**, au palier le plus bas ;
- **avec l'ouverture** → l'intervenant **décrit** : deuxième couleur
  quatrième, à défaut Sans-Atout avec les couleurs adverses tenues, à défaut
  saut dans sa propre couleur. Jamais le simple retour, réservé au minimum.

Sur la réponse **avec l'ouverture**, le camp est **en manche** : le cue-bid
tenait 11 H utiles ou 13 HLD. L'avancée la nomme, **3SA d'abord** — neuf
levées au lieu de onze pour la même prime — **sauf** couleur adverse non
arrêtée (ni chez elle, ni par le 2SA du partenaire, qui garantit l'arrêt) ou
main irrégulière : alors la manche dans le fit (4 en majeure, 5 en mineure).
Elle y annonce sa force exacte, que le cue-bid laissait sans plafond. Avec un
minimum combiné en zone de chelem (31), la décision générale reprend la main.
`1♣ – 1♦ – 2♣ – 2SA` avec ♠AR ♥RV85 ♦952 ♣10852 → **3SA**, pas 3♦.

### [A-7] La suite des enchères, après le soutien de l'avancée
Le soutien a **plafonné** l'avancée — 7-10 HLD pour le simple, 11-12 pour le
saut [A-5] — tandis que l'intervenant n'a toujours annoncé que les 9-18 HL de
son intervention. C'est donc à lui de placer le contrat, et il le fait sur une
arithmétique, pas sur une convention : le seuil de la manche majeure est de
**27 HLD** [E-9], et le partenaire vient d'annoncer un plancher et un plafond.

- **Les certitudes.** Le partenaire n'ayant pas plus de 10 HLD, tout compte qui
  laisse le **maximum** combiné sous 27 ne peut pas produire de manche :
  **passe**, jusqu'à 16 HLD. Et dès que le **minimum** combiné atteint 27 — à
  partir de 20 HLD — la manche ne se propose plus, elle **s'annonce**.
- **Le doute.** Entre les deux, 17-19 HLD : le maximum passe la barre, le
  minimum non. **Proposition de manche au palier de 3**, dans la couleur
  d'intervention.
- **La réponse.** L'avancée ajoute son compte réel au plancher que la
  proposition vient d'annoncer, et conclut à la manche s'il atteint le seuil :
  face à un soutien simple, **10 HLD acceptent**, 7-9 refusent.

Ces bornes sont celles de la fiche — passer jusqu'à 14/15 HLD, proposer de
15/16 à 18, annoncer la manche dès 19 — **recalculées sur les zones du
moteur**, dont le soutien simple s'arrête à 10 HLD là où la fiche le laisse
monter à 12 en se passant du soutien à saut. Le raisonnement, lui, est mot
pour mot le même, et c'est celui de la décision générique [§12] : il n'y a
aucune enchère particulière à retenir ici, seulement à constater que le compte
s'applique aussi de ce côté de la table.

### 9.5 Les réponses au réveil 

**Principe** : on répond **comme si le partenaire était faible**, car avec un
jeu valant l'ouverture le réveilleur, lui, doit reparler. Le réveil est plafonné
à 13 HL : les zones de §9.2 et de [A-5], taillées pour
une intervention directe de 9-18 HL, surenchériraient sur presque toutes les
mains.

### [R-1] Sur un **réveil par une couleur** (5 cartes, 8-13 HL)
- **Passe** — c'est la réponse la plus fréquente ;
- avec le **fit** (3 atouts et plus), en comptant **+1 HLD dès le neuvième
  atout** (le réveil en promet 5 ; voir [E-9d]) : 13 HLD et plus → **cue-bid** de la
  couleur d'ouverture [R-4] · 11-12 HLD → soutien à saut · 7-10 HLD → soutien
  simple. Contrairement à [A-5], le saut **survit au passe** de l'avancée : en
  réveil ce passe n'a rien dénié, c'est la raison même pour laquelle le
  partenaire a dû réveiller. `1♠ – passe – passe – 2♥` avec ♠A75 ♥A864 ♦V953
  ♣D8 : 11 H + 1 (doubleton) + 1 (neuvième atout) = 13 HLD → **2♠**, et le
  réveilleur au maximum conclut à **4♥** ;
- 4 atouts et une belle couleur cinquième, 8-11 H → **rencontre** [A-5] ;
- **2SA** = 13-15 H · **1SA** = 9-12 H, dans les deux cas avec l'arrêt (ou
  trois petites cartes) dans la couleur d'ouverture. 1SA occupant la zone
  basse, 2SA est un **saut** par-dessus ;
  - ⚠ Ce **2SA est une invitation**, et le réveilleur y répond depuis la zone
    qu'il a lui-même annoncée (8-13 HL), pas sur un compte absolu : **11 HL et
    plus → 3SA**, en dessous il passe. Le compte minimum (11 + 13) ne fait que
    24, mais le 2SA moyen vaut 14 et la couleur cinquième du réveil est
    précisément la source de levées que 3SA joue. `1♦ – passe – passe – 1♥ –
    passe – 2SA` avec ♠V82 ♥ADV65 ♦52 ♣R83 → **3SA** ;
- **nouvelle couleur** = **misfit, non forcing** : le partenaire est limité et
  reste libre de passer ;
- sinon **passe**.

### [R-2] Sur un **réveil à 1SA** (9-13 HL) ou **2SA** (17-19 HL)
Mêmes principes que sur une intervention à 1SA : la zone du partenaire est
connue et bornée, l'avancée additionne et conclut. Sur 2SA, il s'agit de juger
sa main pour une manche éventuelle — **6 H suffisent** à la chercher.

### [R-3] Sur un **contre de réveil**
L'échelle se lit à partir de l'enchère la moins chère dans la couleur, et non
à partir de la manche comme en §9.2 :

| Réponse | Signification |
| --- | --- |
| Couleur **sans saut** · `1♣ p p X p 1♦/1♥/1♠` | 4 cartes, **0-10 H** |
| Couleur **avec saut** · `1♣ p p X p 2♦/2♥/2♠` | 4 cartes et **11-12 H**, ou 5 cartes et **8-10 H** |
| Couleur avec **double saut** · `1♣ p p X p 3♦` | 5 cartes et **11-12 H** |
| **1SA** | **9-12 H** et l'arrêt (ou trois petites), sans majeure quatrième |
| **2SA** | **13-14 H** et l'arrêt, sans majeure quatrième |
| **Cue-bid** · `1♣ p p X p 2♣` | la valeur de l'ouverture et **pas d'enchère évidente**, forcing |

Ordre d'examen : 2SA et cue-bid à partir de 13 H, puis les sauts, puis 1SA,
puis la réponse minimale. Quand le palier manque pour le saut que la zone
demande (2♥ derrière 1♠), l'enchère retombe sur la moins chère et couvre alors
tout le bas de l'échelle, comme en [D-7].

### [R-4] Le cue-bid sur un réveil par une couleur
Il ne demande rien — contrairement au cue-bid de force [A-6], qui interroge une
intervention de 9-18 HL — il **annonce** : espoir de manche, la valeur de
l'ouverture et plus, forcing. La réponse est codée sur la fourchette 8-13 HL du
réveil, un cran plus bas que celle d'une intervention directe :
- **réveil minimum** (moins de 11 HL) → **retour à sa couleur** au palier le
  plus bas ;
- **le maximum du réveil** → il **décrit** : deuxième couleur quatrième, à
  défaut Sans-Atout avec les couleurs adverses tenues, à défaut saut dans sa
  propre couleur.

---

## 10. La compétition

### [L-1] Loi des levées totales (soutien)
9 atouts communs → palier de 3 ; 10 atouts → palier de 4. Ce plancher prime sur
le plafond en points [RC-3] : il fixe le **palier** du soutien, il n'ouvre pas
le droit de parler. Le soutien reste une réponse, et toute réponse demande
**6 points au moins** ; en dessous on passe, quelle que soit la longueur
d'atout. Le fit n'est pas perdu pour autant : c'est [L-1b] qui le reprendra
plus tard, en bataille de partielle, sur la seule longueur d'atout.

### [L-1b] La loi en bataille de partielle
Le soutien [L-1] se décide au moment de la réponse ; la partielle, elle, se
perd plus tard — quand le compte a renoncé à la manche et que les adversaires
restent maîtres du contrat. À ce moment, si notre camp détient un fit **connu**
[E-8] de 9 cartes, on ne leur laisse pas la partielle : on nomme le fit au
palier que la loi lui accorde (3 avec 9 atouts, 4 avec 10), **sans rien
promettre en points** — l'enchère est faite sur la seule longueur d'atout et ne
doit pas se lire comme des valeurs que le partenaire relancerait.

Conditions : leur enchère est la dernière sur la table, elle est **sous la
manche** et au plus au palier de 3 (au-delà, ce sont la surenchère [L-2] et le
sacrifice [L-3] qui jugent), et l'enchère la moins chère dans le fit ne dépasse
pas le palier de la loi. Cette dernière condition est aussi le frein : dès que
notre camp a dit son palier, la suivante passerait au-dessus de la loi et la
règle se tait.

### [L-2] Surenchère
Quand les adversaires ont annoncé la manche et que nous avons un fit : on
enchérit par-dessus si le compte combiné atteint le **seuil de manche plus
3 points par palier supplémentaire**.

### [L-3] Sacrifice
Sinon, on sacrifie dans un fit d'au moins **9 cartes** si :
- la force annoncée par les adversaires atteint **23 points combinés** (sinon
  leur contrat n'est pas donné pour gagnant) ;
- le nombre de levées attendues = **le nombre d'atouts communs** (loi) ;
- la chute **contrée** coûte moins que leur contrat, vulnérabilités des deux
  camps prises en compte ;
- au plus 3 de chute.

### [L-4] Contre punitif d'un sacrifice
Si notre camp avait annoncé la manche sur des valeurs réelles et que les
adversaires nous ont surenchéri avec **moins de 23 points combinés annoncés**,
et que ni la surenchère ni le sacrifice ne sont rentables → **Contre**.

### [L-5] Barème utilisé
Marque du contrat réussi exactement (levées + 50 en partielle, +300/+500 en
manche, +500/+750 petit chelem, +1000/+1500 grand chelem) contre la pénalité
contrée (100/300/500 puis +300 par levée non vulnérable ; 200 puis +300 par
levée vulnérable).

---

## 11. La zone de chelem

### 11.1 Enchères de contrôle

Source : *Les enchères de contrôle* (document de référence joint au projet).

### [S-0] Quand une couleur est-elle un contrôle ?
Le fit doit d'abord avoir été **exprimé** — les deux mains ont promis de la
longueur dans l'atout — et l'enchère doit atteindre ou dépasser le palier
de la manche. Sous ce palier, une couleur nouvelle reste un **essai**.
Le moteur applique ce critère (`fitExpressed`) à **tout** déclenchement du
mécanisme. Une main qui connaît le fit sans l'avoir dit — le partenaire a
promis cinq cartes, elle en tient quatre — sait quelque chose que le
partenaire ignore : celui-ci n'a jamais entendu la couleur de sa bouche, ne
peut donc pas distinguer un contrôle d'un naturel, et répondrait à une
question qui ne lui a pas été posée. Le fit implicite suffit à **choisir un
contrat**, pas à **ouvrir un dialogue**.

Le moteur nomme donc l'atout d'abord (`expressFit`, « soutien forcing :
l'atout avant les contrôles ») sous la même condition de chelem en vue qui
arme les contrôles, et l'échange démarre au tour suivant. Trois réserves :
le soutien ne descend jamais sous le palier de 3 (il se lirait comme une
simple préférence au moment même où le camp ouvre une sonde de chelem), il
s'efface devant une enchère que le partenaire a réservée à une convention
(`reservedByOffer` — un Checkback, un relais), et il ne se substitue pas à la
conclusion : au palier de la manche ou au-dessus, c'est la valeur qui place
le contrat.

Une fois l'échange commencé, le contrôle **à l'atout** montre une clef, pas
une longueur, et n'alimente donc pas `shownLens` : `trumpNamedByBoth` accepte
aussi bien le fit exprimé que l'échange déjà entamé (`sideHasCued`), sans quoi
le camp perdrait le fil de sa propre conversation en cours de phrase.

⚠ **Deux portes, une seule règle.** Le mécanisme des contrôles a un second
point d'entrée, `slam-explore-with-fit`, réservé au compte de 33 et plus.
Pour **demander** (Blackwood) il se contente de `trumpAgreed` — les réponses
nomment une couleur que le camp a bidée, donc lisible. Pour **ouvrir
l'échange des contrôles** il exige `trumpNamedByBoth` comme l'autre porte :
un contrôle est une question, et le partenaire ne peut pas répondre à une
question portant sur un atout qu'il n'a jamais entendu de notre bouche.
Exception : quand il ne reste plus de place pour soutenir sous la manche, le
soutien serait la conclusion et le contrôle est tout ce qui subsiste —
l'atout est alors convenu de fait, puisqu'il est la seule couleur du camp sur
la table, et l'échange s'ouvre quand même.

### [S-1] Déclenchement avec fit
- ⚠ **Les points des adversaires comptent aussi.** Le jeu n'a que 40 H : ce
  qu'ils ont annoncé (leur plancher en HL, moins 2 points de longueur chacun
  — une ouverture de 12 HL, c'est au moins 10 H) est retiré du plafond de
  notre camp. Sous **33 H**, pas de chelem sur les honneurs : pas d'essai par
  contrôles, pas de Blackwood. `1♣ – 1♦ – 2♣ – 2SA – 3SA` : l'ouverture d'Est
  laisse au plus 30 H à Nord-Sud, Sud (♠D3 ♥A3 ♦ARV86 ♣R743, 17 H) passe.
  Le plafond s'efface devant une **courte** (singleton ou chicane) tenue ou
  annoncée par le partenaire : un chelem de coupes ne se compte pas en
  honneurs, et 25 H avec un fit de 9 et deux singletons le font après
  l'ouverture adverse.
- **29 à 32** combinés **et un chelem en vue** → essai par contrôles. « On ne
  démarre les enchères de contrôle que si on envisage un chelem » : il faut
  que le **maximum** combiné atteigne la zone (**33**) — un partenaire à
  amplitude large, contre d'appel ou intervention, peut cacher sous son
  plancher de quoi conclure — **ou** que le **minimum** soit déjà à un Roi de
  la zone : un Roi vaut 3 points, donc **30**. Un camp dont le plafond est un
  29 sec, face à un soutien
  parfaitement limité, n'a pas de chelem à explorer : la sonde ne fait que
  brouiller une manche normale. Avec un **fit mineur**, il faut en plus **31**
  (ou que Sans-Atout soit injouable) : sinon un contrôle au-dessus de 3SA
  condamne le camp à 5m (onze levées) là où 3SA (neuf levées) suffisait.
- ⚠ Ce maximum se compte **en valeur chelem** : on en retire, comme au
  minimum, les points de courte **face à la longueur annoncée par le
  partenaire**. Douze levées ne se font pas sur une coupe à laquelle le compte
  ne croit qu'à moitié. `1♦ – 1♥ – 1♠ – 2♠` avec ♠ARV7 ♥3 ♦ARDT962 ♣9 compte
  23 HLD et tombe pile sur 33 face au plafond du soutien — mais le singleton
  Cœur est en face des Cœurs du partenaire et n'achète rien : 31 en valeur
  chelem, pas de chelem en vue, pas d'échange de contrôles.
- ⚠ Ce maximum ne dispense pas non plus **à lui seul** : l'amplitude large
  qui le justifie (« contre d'appel ou intervention, peut cacher sous son
  plancher de quoi conclure ») ne décrit pas un soutien simple codifié. Un
  partenaire à plancher connu et étroit (**écart ≤ 4**, comme un « soutien
  simple, 12-16HLD ») a déjà tout dit : effleurer 33 sur son seul plafond
  n'est alors que le cas chanceux où il est au sommet exact d'une enchère qui,
  en moyenne, en est loin — pas un chelem en vue. `1♣ – 1♠ – 2♠` avec ♠AD862
  ♥R9873 ♦98 ♣R compte 17 en soutien du fit, tombe pile sur 33 face au
  plafond du soutien (12-16, écart 4) — mais 2♠ est un soutien étroit, pas une
  main à amplitude large : pas de chelem en vue, pas d'échange de contrôles.
  Le maximum ne fait passer la porte qu'avec un partenaire dont l'écart
  annoncé dépasse ce seuil (contre d'appel, intervention…) ; sinon seul le
  minimum (**30**) l'ouvre. `1♣ – 1♠ – 2♥ – 3♥` avec ♠7 ♥DT942 ♦ARV ♣ADV6
  compte 18 HLD en soutien du fit : 30 face au plancher du soutien, un Roi
  sous la zone — la sonde s'ouvre, et c'est le partenaire qui la referme s'il
  n'a pas les contrôles.
- **33 et plus** → Blackwood directement, **sauf** si une couleur annexe est
  non contrôlée (voir [S-3]).
- **Avec un fit mineur, la demande attend elle aussi les 33.** On la croyait
  gratuite — la manche est déjà au palier de 5, donc 4SA ne coûte rien. C'est
  faux à moitié : l'échelle des réponses est 5♣-5♦-5♥-5♠ quel que soit
  l'atout, donc sur fit Carreau les deux marches hautes sont **au-dessus de
  5♦**. Qui entend « 2 clefs sans la Dame » et veut s'arrêter n'a plus d'arrêt
  à nommer, et `keycardAnswer` hisse son propre repli dans le chelem qu'il
  refusait. Demander en mineure engage donc le camp, et cela ne se fait qu'avec
  le compte qui veut le chelem.
- **Sous 33, si aucune clef ne manque** : une main de **20 HLD et plus** dont
  les clefs propres — les quatre As et le Roi d'atout — complétées par celles
  que les cue-bids du partenaire ont révélées font **cinq**, explore quand
  même. Le chelem ne dépend plus du total mais de la Dame d'atout et des Rois,
  et la demande ne peut pas revenir courte. C'est le compte qui est aveugle
  ici : un Roi montré par un cue-bid ne relève le plancher du partenaire que
  de 3, si bien qu'un camp détenant tous les As et trois Rois totalise encore
  32 et s'arrête à la manche. Le **passe du partenaire sur la manche** ne dit
  pas qu'il n'y a pas de chelem : il dit qu'il n'a plus de contrôle à montrer.
- Sur un **soutien limite au palier de 3** dans une majeure (zone codifiée,
  amplitude ≤ 4 points) qui hisse le compte à **29 et plus** *et laisse un
  chelem en vue* (maximum combiné ≥ 33, ou minimum ≥ 30 — cf. [S-1]), le
  moteur n'accepte plus simplement la manche : il passe la main au mécanisme
  des contrôles. « La décision d'explorer le chelem est uniquement le fait du
  joueur qui vient de recevoir l'information de ce soutien limite. » Sans
  chelem en vue, il conclut à la manche.

### [S-2] Ce qu'est un contrôle
Dans une couleur annexe : **premier tour** l'As et la chicane, **deuxième
tour** le Roi et le singleton. **Dans l'atout** : seulement l'As ou le Roi —
une courte à l'atout n'est pas un contrôle.

### [S-2b] Le point de départ, et ce qu'un saut dénie
Le joueur qui **déclenche** le processus commence au palier de son choix : il
nomme **le contrôle immédiatement inférieur à celui qu'il est urgent de
découvrir**, pour que la réponse la moins chère du partenaire soit précisément
cette carte. Exemple du document : ♠AR8654 ♥93 ♦A ♣ARD2, séquence 1♠ – 3♠ ;
le Cœur est le seul contrôle inconnu, donc **4♦**.

À partir de cette marche, l'ordre économique est **incontournable** et **tout
contrôle sauté est dénié**. En revanche, les contrôles situés **sous** la
marche de départ ne sont pas déniés : dans l'exemple, 4♦ ne dit rien du Trèfle.

**« Le moins cher » se lit sur l'échelle des enchères, pas sur l'ordre des
couleurs.** Après un soutien à 3♥, l'enchère de contrôle la moins chère est
**3♠** — elle est sous 3SA, donc sous 4♣ : c'est elle qui se nomme en premier,
et une main sans contrôle à Pique la saute, ce qui le **dénie**. Après un
soutien à 3♠, rien ne passe sous 4♣ et l'ordre des couleurs et celui de
l'échelle coïncident.

`1♦ – 1♥ – 3♥` avec ♠DV8 ♥AV96 ♦96 ♣ADV3 en Sud : pas de contrôle à Pique,
donc **4♣** (l'As), qui dénie le Pique du même coup. Le partenaire — ici
chicane à Pique — n'en est pas gêné, et il sait à quoi s'en tenir. 3♠ aurait
dit l'inverse : « j'ai le contrôle à Pique ».

L'ordre économique prime sans exception : le contrôle le moins cher se nomme
en premier, **même** s'il s'agit d'une courte dans la couleur du partenaire.
Un singleton dans sa longueur reste un contrôle de deuxième tour — le taire
pour sauter à un contrôle plus cher le **dénierait**. (Sur soutien *forcing*,
la réserve que cette courte inspire s'exprime par le 3SA « oui mais » [S-2d],
jamais par un saut dans l'ordre des contrôles.)

Garde-fou : l'échange ne dépasse jamais le palier de la **manche** dans
l'atout — sauf si le compte atteint 34, ou 33 quand le palier de la manche
est déjà consommé.

**L'atout au-dessus de la manche.** Quand l'échange a passé la manche et
qu'il ne reste plus de contrôle annexe à montrer, **5 dans l'atout** n'est
plus « la manche » :
- avec l'**As** ou le **Roi** d'atout et un chelem en vue (33 combinés), c'est
  le **contrôle à l'atout** et une **invitation au chelem**. Les couleurs
  annexes que personne n'a contrôlées sont déniées, et le partenaire décide.
  Il les nomme s'il les tient (6♥ sur 5♠) ; sinon il **passe**, parce que le
  chelem donnerait deux levées rapides dans cette couleur ;
- sans honneur d'atout, c'est un simple **retour à l'atout**.

`1♥ – 1♠ – 4♠ – 5♣ – 5♦ – 5♠ – 6♥ – 6♠` avec ♠AV98 ♥74 ♦DV42 ♣AR8 en Sud :
5♠ montre l'As d'atout sans contrôle à Cœur, et Nord (♥ARD) cue 6♥.

### [S-2c] Le « relais contrôle » 3SA (soutien **non forcing** au palier de 3)
Le Trèfle est sous tous les cue-bids : une main qui n'y a pas de contrôle n'a
aucune enchère naturelle pour en demander un — 4♣ prétendrait le détenir, et
ouvrir à 4♦ ne dit rien du Trèfle [S-2b]. D'où une enchère conventionnelle :

| Enchère | Demande | Disponible |
|---|---|---|
| **3SA** | le contrôle à **Trèfle** | fit Cœur et fit Pique |

Un contrôle à **Carreau** manquant n'a besoin d'aucune convention — 4♣ le
demande.

**Réponse positive** : l'enchère immédiatement supérieure (3SA → **4♣**).
**Réponse négative** : on saute la marche — ce qui **dénie** le contrôle
demandé — et on nomme ses propres contrôles dans l'ordre économique (4♦, puis
4♥…), ou l'on conclut à la manche s'il n'y en a aucun.

⚠ **Le document donne un second relais, 3♠ sur fit Cœur, demandant le contrôle
à Pique ; le moteur ne l'implémente pas.** Il contredit le principe des
contrôles, qui est plus fort que lui : sur fit Cœur 3♠ est simplement
l'enchère de contrôle **la moins chère** [S-2b], donc elle **montre** le
contrôle à Pique. Le lire comme une demande obligerait une main sans contrôle
à Pique à l'annoncer, et priverait celle qui l'a du moyen de le dire. Rien
n'est perdu : la main qui n'a pas ce contrôle saute la marche, ce qui le
dénie — l'information que la demande cherchait, obtenue sans enchère
conventionnelle.

Un fit Cœur teinte donc aussi le relais : **3SA saute 3♠**, et dénie par là le
contrôle à Pique. Une main qui le détient nomme 3♠ au lieu de relayer, et
l'ordre économique de la réponse du partenaire placera le Trèfle de toute
façon.

### [S-2d] Le 3SA « oui mais » (soutien **forcing** au palier de 3)
Face au soutien **forcing**, l'ouvreur a trois attitudes : **non** (il nomme la
manche), **oui** (une enchère de contrôle), ou **oui mais** — **3SA** : une
main avec laquelle on est intéressé par le chelem, donc jamais minimale, mais
qui porte une contre-indication.

⚠ Le document propose une version large (toute contre-indication : qualité
d'atout, distribution régulière, qualité des honneurs…) et une version
étroite, pour laquelle l'auteur exprime « une nette préférence ». **Le moteur
n'implémente que le critère étroit : un singleton dans la couleur du
partenaire.** La « mauvaise qualité d'atout » a été essayée puis retirée : le
test se déclenche sur ♠AR9853 ♥V643 ♦A ♣A3 — quatre atouts médiocres, mais
trois As et une sixième à côté — où nommer un contrôle est manifestement
meilleur. Un singleton face à la longueur du partenaire n'a pas ce défaut :
c'est un fait sur le fit qu'aucune enchère de contrôle ne peut exprimer, et il
reste vrai quelle que soit la force du reste.

**Sauf un singleton honneur** (As, Roi ou Dame sec) : face à la longueur du
partenaire, il ne se perd pas, il complète sa couleur (♣D sec en face de
♣AR542). Il ne déclenche pas le « oui mais », et l'évaluation pour le chelem
(le minimum combiné sans l'appoint des courtes face aux longues du partenaire)
ne le décompte pas. `1♠ – 2♣ – 2♦ – 3♠` avec ♠RV9762 ♥D4 ♦ARVX ♣D : Nord
lance les contrôles (`4♦`) au lieu de 3SA.

### [S-3] Couleur annexe non contrôlée
Une couleur (hors atout) où l'on a **2 cartes ou plus sans A ni R**, que le
camp n'a jamais nommée et où le partenaire n'a montré aucun contrôle :
Blackwood est interdit (il compte les cartes clefs, pas les contrôles de
deuxième tour). On sonde alors par contrôles :
- **sous le palier de la manche**, librement — la sonde ne coûte rien, le
  partenaire peut toujours conclure à 4M. Il suffit que l'atout soit
  **convenu** au sens de [S-4b] : le contrôle nomme une couleur que le
  partenaire sait lire même s'il est seul à avoir montré la longueur d'atout.
  Exiger que les deux mains l'aient montrée ne laissait à la main de chelem
  que la manche à annoncer ;
- **au-dessus**, seulement si le compte atteint **34** et que la dernière
  enchère du partenaire délimitait une zone **étroite** (amplitude ≤ 4 points),
  car la sonde achète alors le palier de 5 si le contrôle ne vient pas.
  `1♦ – 1♥ – 4♥` (20-23 HLD) avec ♠DV5 ♥D9762 ♦— ♣A7542 : 14 HLD, la chicane
  comptée face aux quatre Carreaux de l'ouverture, soit 34 → **5♣**, et le
  camp atteint **6♥**.

### [S-3b] Quand il n'y a plus de contrôle à montrer
On revient à **la manche dans l'atout**. Ce n'est pas un arrêt : l'enchère ne
dit rien d'autre que « je n'ai plus de contrôle à montrer », elle ne plafonne
pas la main et ne ferme pas l'étui. Le partenaire garde la parole et peut
poursuivre — c'est précisément ce que fait la main qui voit les cinq clefs
réunies [S-1].

### [S-4] Passage au Blackwood
Depuis l'échange de contrôles, et **selon ce que la question coûte** :
- **atout mineur** : **29 combinés**. La manche est déjà au palier de 5, donc
  4SA et ses réponses ne coûtent rien.
- **atout majeur** : **33 combinés** — la zone de chelem. Là, 4SA et *toutes*
  ses réponses sont au-dessus de 4M : une demande qui revient courte plante le
  camp à 5M, onze levées pour une prime de manche, et rien ne permet de
  redescendre. Sous 33, la réponse ne peut même pas changer la décision :
  quatre clefs demandent 33 pour nommer le chelem [S-7], seul « les cinq »
  ferait la différence, et c'est un pari sur ce que les enchères n'ont jamais
  promis. L'échange s'arrête alors à **4M**, là où les contrôles allaient.
- dans les deux cas, une main de **20 HLD** dont le camp contrôle **toutes**
  les couleurs annexes demande quand même : sur ces jeux distributionnels ce
  sont les contrôles, pas les points, qui décident.
- **Avec un fit mineur, la main qui n'a plus rien à découvrir demande avant de
  cue-bidder.** La manche y est au-dessus de 4SA : quel que soit le contrôle
  qu'elle nomme, le partenaire peut conclure à 5m et enterrer la demande, sans
  retour possible. Une main qui détient — ou dont le partenaire a montré — un
  contrôle dans **chaque** couleur annexe n'apprendrait rien de l'échange :
  elle pose la question tant qu'elle est là. Tant qu'une couleur reste
  inconnue, le cue garde la priorité : le Blackwood compte les clefs, pas les
  contrôles de deuxième tour [S-3].
- une grosse main (**20 HLD**) qui voit **les cinq clefs** chez elle ou
  révélées par les contrôles du partenaire peut demander à **32 combinés**, un
  point sous la zone : le chelem tient alors à la Dame d'atout et aux Rois,
  pas au total. **Pas en dessous** : les cinq clefs disent que les As sont là,
  pas que douze levées le sont, et la réponse est connue d'avance (« 0 ou 3 »
  ne peut valoir que 0). `2♦ – (2♠) – Passe – 3♥ – 4♥` avec ♠A1043 ♥AR532
  ♦AR ♣AD face à ♠V ♥10987 ♦763 ♣98652 (30 combinés) → **Passe** à 4♥, et
  non 4SA – 5♣ – 6♥.
- à partir de **29 combinés**, quel que soit l'atout, quand le partenaire a
  **nié tout contrôle** dans une couleur où l'on n'a pas l'As (un seul As
  annexe ainsi localisé chez l'adversaire) et que **toutes** les couleurs
  annexes sont contrôlées. Les clefs restantes du partenaire ne peuvent plus
  être que l'As et le Roi d'atout : s'il les a toutes, la seule clef absente
  est cet As annexe, couvert par le singleton ou le Roi, et le chelem est
  demandé ; sinon on s'arrête au palier de 5. `1♠ – 2♣ – 2♥ – 3♠ – 3SA – 4♥`
  avec ♠109543 ♥ADV72 ♦A7 ♣5 face à ♠ARDV86 ♥— ♦84 ♣DV1084 (le 4♥ saute 4♣
  et 4♦) → **4SA** – 5♠ – **6♠**.

À défaut, on s'arrête à la manche dans l'atout.

### 11.2 Blackwood aux cartes clefs (atout convenu)

### [S-4b] Ce que « atout convenu » veut dire
Les réponses comptant le **Roi et la Dame d'atout**, la demande n'a de sens
que si le partenaire sait de quelle couleur on parle. Le moteur l'exige :
- soit **les deux mains ont montré de la longueur** dans la couleur ;
- soit c'est la **dernière couleur nommée par le camp**.

La longueur montrée doit l'avoir été par une enchère **naturelle**. Une enchère
artificielle promet sans proposer : le cue-bid qui demande la force d'une
intervention garantit trois cartes dans cette couleur [A-6], mais son sujet est
la force du partenaire, pas le contrat. Aussi, tant que **notre camp a deux
couleurs à lui sur la table** et que le partenaire n'en a **nommé aucune**,
l'atout n'est pas choisi et la demande attend : après 1♥ – 2♣ – (cue-bid 2♥) –
2♠, l'intervenant a offert Trèfle et Pique, l'avancée n'a rien tranché, et
« deux clefs et la Dame d'atout » désignerait une couleur que personne n'a
retenue. Le camp nomme d'abord, la demande vient ensuite.

Un fit seulement **calculé** — la promesse minimale de l'ouverture plus sa
propre longueur [E-8] — ne suffit pas : après 1♦ – 1♠ – 1SA, une main qui
détient six Carreaux n'a jamais dit un mot de Carreau, et un 4SA laisserait le
partenaire répondre « deux clefs et la Dame d'atout » sur une couleur que
personne n'a convenue. Dans ce cas le moteur **nomme la couleur** d'abord (avec
un saut s'il y a la place), forcing ; la demande vient au tour suivant, une
fois l'atout sur la table.

La main **courte** — celle qui soutient la couleur du partenaire et n'a aucune
cinquième à nommer — met le même atout sur la table par le **soutien**, gardé
**sous le palier de la manche** et **forcing** : à la zone de chelem c'est un
essai, pas la conclusion que la même enchère serait un cran plus haut. Sans
cela, une main de 23 H en face d'une ouverture n'avait plus que la manche à
annoncer.

### [S-4c] Qui demande, après une ouverture de 2♦
Le répondant qui a **répondu aux As** ne pose pas le Blackwood : il a déjà dit
où sont ses As et n'apprend rien en retour, tandis que l'ouvreur, qui connaît
sa propre main **et** l'As que le palier a nommé, détient tout le tableau.
L'exploration est donc la sienne — et son 4SA devient l'**appel aux Rois** dès
que le camp tient les quatre As [S-10]. Une demande venue de l'autre côté
achète une réponse dont l'ambiguïté 0/3 ou 1/4 n'est levable par rien dans
l'enchère : le demandeur retient alors le plus petit compte [S-6] et s'arrête
au palier de 5 avec toutes les clefs dans les deux mains.

### [S-5] **4SA**. Réponses : 5♣ = 0 ou 3 · 5♦ = 1 ou 4 · 5♥ = 2 sans la Dame
d'atout · 5♠ = 2 avec la Dame. (Clefs = les 4 As + le Roi d'atout.)
### [S-6] L'ambiguïté 0/3 et 1/4 se lève **uniquement** sur ce que le demandeur
sait : son propre compte (le total ne peut dépasser 5), les clefs déjà
révélées par un contrôle avant la demande, et le **plancher que les enchères
du partenaire ont promis**. Ce dernier est une impossibilité arithmétique, pas
un pari : les clefs qui manqueraient au petit compte sont alors chez les
adversaires, et ce qui reste des 40 points — 40 moins sa propre force, moins
la valeur de ces clefs (4 par As, 3 pour le Roi d'atout) — plafonne le
partenaire. Si le plancher qu'il a annoncé **atteint** ce plafond, le petit
compte est exclu et la réponse se lit au grand : à l'égalité, le petit compte
obligerait le partenaire à détenir **tout** ce qui reste en Rois, Dames et
Valets sans une seule clef, main que ses enchères viennent de démentir.
Redouter cette main-là revient à demander les clefs pour n'en rien faire. Dans
le doute — plancher sous le plafond — le demandeur retient le **plus petit**
compte.

`1♥ – 1♠ – 3♥ – 3SA – 4SA – 5♣` avec ♠R9 ♥ARVT9862 ♦4 ♣RD : 16 H et 2 clefs,
donc un partenaire sans clef plafonne à 40 − 16 − 12 = 12 — exactement le
plancher que son 3SA a promis. La réponse est donc 3 clefs, les cinq sont
réunies, et c'est **6♥**, pas 5♥.
### [S-7] Conclusion : **5 clefs** → petit chelem, et grand chelem à partir de
**37** combinés. **4 clefs** → petit chelem si le compte atteint 33, ou si la
clef manquante est un As annexe couvert par les contrôles montrés. Sinon arrêt
au palier de 5.
### [S-7b] Après une ouverture de 2♦, les As sont déjà comptés
Le palier des As [F-2] a dit à l'ouvreur **combien d'As** tient le répondant :
ses propres clefs plus ces As donnent le compte **sans aucune demande**. Quand
le fit se trouve au-dessus de 4SA — une mineure soutenue à la manche, comme
`2♦ – 2♥ – 2♠ – 3SA – 4♣ – 5♣` — ni le Blackwood ni l'appel aux Rois n'ont plus
la place, et l'échange de contrôles s'arrête à la manche où l'on est déjà. Avec
**4 clefs connues** et le compte de chelem atteint (33, points de courte face à
la longueur du partenaire retranchés [E-1c]), l'ouvreur conclut au **petit
chelem**, la conclusion même de [S-7].

Mesure seule, sur 30 000 donnes comportant une main de 21 H et plus : 95 donnes
changent, **+467 IMP** en double-mort (51 gains, 13 pertes).

### 11.3 Blackwood sans atout convenu
### [S-8] Déclenché sans fit à partir de **33** combinés. Simple décompte des
As : 5♣ = 0 · 5♦ = 1 · 5♥ = 2 · 5♠ = 3 · 5SA = 4.
### [S-9] Conclusion : 3 As ou plus → **6SA** ; 4 As et 37 combinés → **7SA** ;
sinon arrêt à **5SA**.

### 11.4 L'appel aux Rois (après une ouverture de 2♦)
### [S-10] Les réponses aux As ayant localisé les As, dès que l'ouvreur
constate que le camp les détient **tous les quatre**, son **4SA demande les
Rois**. Sinon 4SA reste un Blackwood ordinaire.
L'appel vaut aussi **sur la conclusion du répondant** : le fit trouvé sous la
manche, celui-ci conclut à 4M — il a répondu au palier des As quel que soit son
compte, donc son plancher est 0 et le compte combiné ne bougera plus. La route
des contrôles [S-1] plafonne à la manche, là où l'enchère se trouve déjà : seul
le 4SA rouvre la donne. Même condition qu'en [RO-24] : il ne se pose que si un
**seul Roi du partenaire suffirait à atteindre 33** [E-9].
### [S-10b] Le répondant aux As corrige la conclusion de l'ouvreur
L'échelle des As dit les As, pas les points : elle promet un **plancher sans
plafond** — 0 pour la marche de 2♥, 4 pour un As isolé, 8 pour les autres. Le
capitaine qui compte à partir de ce plancher se pose donc à 3SA sur 35 points
réels, et la seule main qui puisse voir les 33, c'est celle qui sait lesquels
de ces points elle détient.

Quand l'ouvreur a **conclu à la manche** — sa description est finie, rien
d'autre ne viendra — le répondant aux As **corrige à 6SA** si son propre compte
ajouté au plancher de l'ouverture atteint 33 [E-9]. Il ne *demande* rien : ses
As sont sur la table et 4SA appartient au capitaine [S-10].

⚠ **Seulement sur une conclusion à la manche.** Tant que l'ouvreur décrit
encore — une couleur au palier de 3, sous la manche — la main qui a répondu aux
As reste le second : elle pose sa manche et laisse parler le capitaine. Mesure :
sans cette réserve, le 6SA du répondant coupe l'appel aux Rois de l'ouvreur et
ramène **dix grands chelems à 6SA** sur mille donnes à 33 points et plus, pour
seize gains — un troc perdant. Avec elle, six donnes changent, toutes vers le
haut, et **aucune sur cinq mille donnes neutres**.

### [S-11] **Avec atout convenu** : la Dame d'atout remplace le Roi d'atout
comme cinquième clef ; réponses 5♣ = 0/3, 5♦ = 1/4, 5♥ = 2/5. Cinq clefs →
grand chelem ; quatre clefs et 37 combinés → grand chelem ; sinon petit chelem.
### [S-12] **Sans atout** : décompte des 4 Rois (5♣ = 0 · 5♦ = 1 · 5♥ = 2 ·
5♠ = 3 · 5SA = 4). Les 4 Rois → **7SA** ; 3 Rois et 37 combinés → 7SA ;
**si le compte combiné reste sous 33 → arrêt à 5SA** ; sinon **6SA**.

### 11.5 Le 4SA quantitatif
### [S-13] Posé quand le compte minimum reste sous 33 mais que le maximum
l'atteint, face à une zone **étroite** (amplitude ≤ 7) délimitée par une
enchère à Sans-Atout du partenaire, avec une main (semi-)régulière — et à
partir de **28 combinés**. Le plancher compte autant que le plafond : la
question s'achète au palier de 4 à Sans-Atout, une manche qui doit désormais
prendre dix levées au lieu de neuf, donc elle ne vaut d'être posée que si le
partenaire n'a que quelques points à trouver au-dessus de son plancher, et non
toute la largeur de sa zone. Mesure sur 40 000 donnes : à partir de 28, 129
propositions pour 55 chelems ; en dessous, 98 pour 6, dont 85 revenues se
poser à 4SA.
### [S-13b] La forme n'est pas le bon garde-fou
La règle réclamait une main **régulière ou semi-régulière**. Elle protégeait
ainsi un contrat à Sans-Atout que le camp allait jouer de toute façon — il n'a
de fit nulle part — alors que ce que douze levées à Sans-Atout ne supportent
pas, c'est une **chicane** : une couleur entière offerte à la défense. ♠ADV87
♥ARV ♦AD63 ♣4 face à une redemande à 1SA de 12-14 posait un 3SA plat sur 32
points combinés, quand le partenaire au sommet de sa zone en donne 34.

Sont donc admises les mains régulières et semi-régulières d'avant, **plus**
celles à singleton isolé (5-4-3-1, 4-4-4-1). Restent exclues la **chicane** et
le **singleton à côté d'une couleur sixième** : là, le chelem appartient à la
couleur, pas aux Sans-Atout. La proposition nommant un chelem à Sans-Atout,
elle exige en outre les **arrêts** (`ntOK`).

⚠ Deux conditions d'apparence aussi arbitraire ont été mesurées, et gardées.

- **La dernière enchère du partenaire doit être à Sans-Atout.** Sans elle, la
  règle se déclenche au milieu d'une séquence en couleur et l'enterre : sur
  5 000 donnes neutres, un chelem perdu et un grand ramené à 6SA, parce qu'un
  3SA modeste laisse au partenaire la place de reparler là où 4SA clôt les
  enchères.
- **La zone étroite (amplitude ≤ 7).** Elle tient la règle à l'écart des
  séquences de 2♦ fort et des 2 sur 1 forcing de manche. Face à un partenaire
  qu'une enchère forcing laisse sans plafond, **la proposition ne
  m'appartient pas** : il connaît mon plancher et son propre compte, je ne
  connais que le mien. Lui passer devant avec 4SA achète le petit chelem là où
  son propre appel aux Rois trouvait le grand.

C'est pourquoi la famille la plus peuplée des chelems manqués — celle où c'est
la main faible qui pose 3SA face à une ouverture de 2♦ — **n'est pas traitée
ici** : elle demande que la main forte réévalue par-dessus le 3SA, ce qui est
un autre mécanisme. [S-13c] en traite un cas : la main forte bicolore qui
nomme sa deuxième couleur par-dessus ce 3SA.

### [S-14] Réponses : c'est une **arithmétique**, pas une position dans sa
propre zone annoncée — se croire « au milieu » d'une zone de 4-11 ne dit rien
sur les 33 [E-9]. Ce qui tranche est son propre compte ajouté à ce que le
demandeur a promis : **33 face à son plancher → 6** · 33 face à son plafond
seulement → **5**, acceptation partielle · sinon **passe**. Après une
acceptation partielle, qui a fixé le compte du répondant exactement, le
demandeur refait la même soustraction : 33 atteints → 6, sinon arrêt.

### [S-13c] La deuxième couleur par-dessus le 3SA du partenaire
Une main qui a déjà montré une couleur **cinquième** et en tient une
**deuxième cinquième**, jamais nommée, n'a pas fini de se décrire quand le
partenaire conclut à 3SA : il a choisi Sans-Atout sans savoir que la seconde
couleur existe. Dans la zone du 4SA quantitatif (**28 combinés** au moins,
**33** atteints face au plafond du partenaire), elle **nomme cette couleur au
palier de 4**, naturel et forcing, au lieu du 4SA quantitatif. L'enchère coûte
le même palier, offre un atout où les points de distribution comptent enfin,
et c'est la main longue qui nomme la couleur, donc qui jouera le contrat.

`2♦ – 2♥ – 2♠ – 3SA` avec ♠AR1072 ♥A ♦R8 ♣ARV52 face à ♠9 ♥RDV97 ♦975 ♣D1094
→ **4♣** – 5♣ – **6♣** joué par Sud [S-7b], au lieu de 4SA – 5SA joué par
Nord. Joué par Nord, l'entame Carreau d'Est à travers ♦975 vers ♦R8 fait
perdre deux levées avant que les Cœurs ne soient affranchis.

**Sans fit** dans la deuxième couleur, le partenaire répond **au compte**, en
Sans-Atout, comme au quantitatif [S-14] : 33 face au plancher du demandeur →
**6SA** ; 33 face à son seul plafond, et une zone étroite (amplitude ≤ 7) →
**5SA**, acceptation partielle ; sinon **4SA**. Laissé à la décision générique,
il revenait toujours à 4SA et perdait le chelem que le quantitatif trouvait.

Mesure sur 30 000 donnes comportant une main de 21 H et plus, avec [S-7b] :
213 donnes changent, **+1 083 IMP** face au jeu en double-mort (119 gains pour
36 pertes), et les chelems manqués passent de 134 à 17 sur ces donnes. Aucune
donne ne change sur 40 000 donnes neutres.

---

## 12. La décision générique de fin d'enchères

Quand aucune convention ne s'applique, les règles conventionnelles sont
essayées **dans cet ordre** (la première qui produit une enchère légale gagne) :

1. répondre au Contre du partenaire · 2. répondre au Blackwood ·
3. conclure son propre Blackwood · 4-6. répondre à un Stayman/Texas tardif ·
7. Stayman pour les majeures après 2♦–2SA · 8. proposer sa couleur longue
après une réponse aux As · 9. répondre au cue-bid de force sur une
intervention [A-6] · 10. répondre au cue-bid du contre d'appel [D-6] ·
11-12. Checkback · 13. Roudi disponible · 14. rectification de la mineure
affranchie · 15-16. réponses au Drury · 17. répondre au Roudi ·
18. répondre à la troisième couleur forcing [§8.9] ·
19. répondre à la quatrième couleur forcing [§8.8] ·
20-22. demande d'arrêt après répétition d'une mineure ·
23. conclure sur la dénégation de la troisième couleur forcing [§8.9] ·
24. répondre à un relais contrôle [S-2c] · 25. poursuivre les contrôles ·
26. rectifier le transfert Stayman · 27-28. misère dorée ·
29-30. proposition de chelem quantitative · 31. le 3SA « oui mais » [S-2d] ·
32. essai par contrôles · 33. réponse à l'essai « couleur nécessitant un
appui » · 34. réponse à une proposition de manche · 35-37. exploration de
chelem · 38. troisième couleur forcing [§8.9] ·
39. quatrième couleur forcing [§8.8] · 40. bicolore du répondant ·
41. 3SA sur la couleur longue du partenaire [RC-5] · 42-43. sacrifice et
contre punitif · 44. fuite sur misfit · 45. manche sur le barrage du
partenaire.

Puis, à défaut, le **compte** :

| Compte combiné | Action |
|---|---|
| ≥ seuil de manche [E-9] | **Conclusion directe** — jamais de proposition quand le seuil est atteint face au **plancher** du partenaire |
| Maximum combiné ≥ seuil | **Proposition** : 3M fitté, répétition d'une majeure sixième, 2SA, ou 3m |
| Fit, 6-10, palier de 2 | **Soutien simple** [G-0] |
| Séquence forcing | enchère constructive la moins chère, jamais de passe |
| Sinon | **Loi des levées totales** si l'adversaire tient une partielle et que nous avons un fit neuvième [L-1b] · sinon **Passe**, ou **préférence** pour la première couleur du partenaire |

### [G-0] Le plancher de la proposition, et le soutien simple qui la remplace
Une proposition exige **11 points** en propre — le plancher de la zone
invitationnelle, le même partout ([RM-4], [Rm-7], [RC-3]). La seule dispense
est un partenaire qui s'est déjà plafonné assez haut pour que les points
manquants soient vraiment les siens (**minimum combiné à moins de 2 du
seuil**).

Sans ce plancher, le maximum du partenaire suffisait à tout : un changement de
couleur au palier de 1 ne dit **rien** de la force de l'ouvreur [RO-19], donc
son plafond reste le 23 de l'ouverture, et 9 + 23 franchissait le seuil de la
manche majeure. `1♦ – 1♥ – 1♠` sur ♠V854 ♥R9642 ♦– ♣VT53 — 9 HLD — produisait
**3♠ « proposition de manche »** là où la main n'a que le soutien simple.

Quand le compte renonce ainsi à proposer, le fit doit quand même se dire :
avec **6-10** et un fit non encore montré, on **soutient au palier de 2** la
couleur que le partenaire vient de nommer. Ce n'est pas un lot de consolation,
c'est l'enchère que la main devait faire ; passer enterre huit atouts au
palier où le partenaire s'est arrêté. Une seule fois — la main qui a déjà
montré son soutien ne le remonte pas — et jamais au-dessus du palier de 2, que
la zone n'achète pas.

⚠ La répétition d'une majeure sixième au palier de 2 échappe au plancher : ce
n'est pas une proposition mais une description que le partenaire peut passer.

### [G-1] Acceptation d'une proposition
On accepte si la valeur de la main atteint **le milieu de la fourchette déjà
annoncée** (ou le plancher + 2, le plus bas des deux). Avec un fit, on
revalorise par HL **plus les seules vraies courtes** (singleton +2, chicane
+3) : un doubleton ne fait pas monter à la manche. Sans fit, on compte les
**honneurs purs** — sauf, sur une proposition **à Sans-Atout**, les points de
longueur d'une couleur **sixième tenue par deux des trois gros honneurs**
(AR, AD, RD) : c'est la source de levées que 3SA va jouer, et la fourchette
annoncée les comptait déjà. `1♦ – 1♠ – 2♦ – 2SA` avec ♠R10 ♥A94 ♦AR7643 ♣87
(14 H, 16 HL, haut de 13-17) → **3SA**, et non passe. Sur une proposition en
couleur sans fit, on reste aux honneurs purs.
En compétition, l'acceptation exige que **plancher du partenaire + valeur
propre** atteigne réellement le seuil de manche — sauf après la réponse large
au contre d'appel (2♥ sur 1♠, 0-10 H), où l'essai du contreur ne pose qu'une
question de zone : on accepte à partir de **8 H** [D-7].

⚠ **Une exception à la règle du doubleton, en compétition seulement.** Là on
compare à un seuil absolu, et ce seuil est exprimé en **HLD** [E-9] alors que
la valeur offerte est HL + courtes, c'est-à-dire HLD **moins le doubleton** :
l'écart n'est pas de la prudence, c'est une erreur d'unité qui coûte
exactement un point — et une manche se joue à un point. Le moteur le comble,
mais **uniquement dans la main qui sera le mort** : celle dont la longueur à
l'atout est *inférieure* à celle que le partenaire a montrée. La distribution
paie des coupes ; le doubleton vaut donc son point à la main de soutien, dont
les atouts sont en surnombre, et rien du tout à la main longue, dont les
atouts partent à tirer ceux de l'adversaire. C'est ainsi qu'un 3-5-3-2 plat
se convainquait de jouer une manche qui chute.

Exemple : `1♣ – (1♠) – 2♣ – (2♠) – 3♣ – (3♠)`, Est tient ♠RT9 ♥AT854 ♦V83
♣T2 face à une intervention cinquième. 8 H, 9 HL, **10 HLD** — le doubleton
Trèfle est dans la main courte, il compte, et 17 + 10 atteint les 27 de la
manche majeure. Est accepte.

### [G-2] Préférence pour la première couleur du partenaire
Avant de laisser mourir un bicolore non forcing, on revient à la première
couleur au même palier si le fit y est **au moins aussi long** que dans la
seconde, avec au moins 2 cartes.

### [G-3] Fuite sur misfit
Si la dernière enchère non forcing du partenaire trouve **au plus 1 carte** et
moins de 7 cartes communes, on revient à sa propre couleur **sixième** déjà
annoncée, sans dépasser sa manche.

### [G-4] Une main qui a passé d'entrée (moins de 6 H) ne peut plus proposer.
### [G-5] Une main déjà plafonnée sous le seuil ne propose pas.
### [G-6] En forcing de manche, une « proposition » devient une enchère
descriptive forcing.
### [G-7] Au-dessus de la manche hors zone de chelem, on se cale sur le
contrat le plus bas sûr.
### [G-8] On ne retire pas le **3SA du partenaire** pour cinq d'une mineure :
deux levées de plus pour la même prime, et c'est le partenaire qui vient de
dire que Sans-Atout tient. Hors zone de chelem, on passe.

---

## 13. Filets de sécurité

Ces règles ne sont pas des règles de bridge : elles rattrapent les trous du
moteur. Un joueur qui en voit une se déclencher a trouvé un vrai bug.

### [Z-1] Une enchère devenue illégale (intervention) est remontée d'un palier
si la main le mérite (11 HL, ou un fit huitième connu) et que cela ne dépasse
pas le palier de 4 ; sinon **passe**.
### [Z-2] Une enchère **forcing** du partenaire, l'adversaire de droite s'étant
tu, n'est jamais passée : on produit l'enchère constructive la moins chère.
### [Z-3] Un camp engagé à la manche ne laisse jamais mourir les enchères
au-dessous : manche dans le fit majeur, sinon 3SA derrière les arrêts, sinon
manche dans le fit mineur.
### [Z-4] Au-delà de **40 enchères**, tout le monde passe.

---

## 14. Points à discuter en priorité

Les endroits où le moteur tranche d'une manière qui mérite l'avis d'un joueur.

| # | Sujet | Ce que fait le moteur |
|---|---|---|
| 1 | **Ouverture 5♠-5♣** | Ouvre **1♣** à partir de 14 H, 1♠ en dessous [O-6] |
| 2 | **Barrage** | Exige que la moitié des H soit dans la couleur ; une main de 11 H / 12 HL ne peut ni ouvrir ni barrer [O-7] |
| 3 | **1SA** | Réservé aux mains **strictement régulières** : un 5-4-2-2 de 16 H ouvre d'1 à la couleur [O-4] |
| 4 | **Majeure quatrième sur mineure** | Prime sur tout, y compris sur un soutien 5 cartes de la mineure [Rm-2] |
| 5 | **Bicolore économique** | Interdit aux mains régulières, qui passent [RO-6] |
| 6 | **Répétition** | Le moteur enregistre la longueur réelle (5) même quand la théorie promet 6 [RO-21] |
| 7 | **Fit** | Uniquement 8 cartes **promises** : un fit 4-4 non annoncé n'existe pas [E-8] |
| 8 | **Essai face à un soutien** | 3M est un barrage, l'essai passe par une couleur d'appui ou 2SA [RO-10] |
| 9 | **Texas mineur** | Une mineure sixième de force moyenne (8-9 HL) ne fait pas de Texas, **même avec une courte** : elle passe, ou prend l'échelle Sans-Atout [N-4] |
| 10 | **Loi des levées totales** | Fixe le **palier** du soutien, pas le droit de répondre : sous 6 HLD on passe [L-1] |
| 11 | **Sacrifice** | Exige 23 points annoncés en face, et calcule sur le barème exact [L-3] |
| 12 | **Contrôles avec fit mineur** | Barre relevée à 31 pour ne pas condamner 3SA [S-1] |
| 13 | **Blackwood 0/3 et 1/4** | Retient toujours le **plus petit** compte dans le doute [S-6] |
| 14 | **Appel aux Rois sans atout** | Peut s'arrêter à **5SA** si le compte reste sous 33 [S-12] |
| 15 | **Acceptation d'une proposition** | Un doubleton ne compte pas ; seules les vraies courtes revalorisent [G-1]. **Sauf en compétition dans la main de soutien**, où le seuil comparé est en HLD : voir l'exception de [G-1] |
| 16 | **Échelle sur 1SA** | Le libellé annonce « 0-8H » là où le seuil est 0-7 [N-6] |
| 17 | **7-2-2-2 « semi-régulière »** | Une 7-2-2-2 de 20-21 H ouvre **2SA** au lieu de 2♣ [E-2] — cas limite non voulu, à trancher |
| 18 | **Quand un contrôle est-il un contrôle ?** | ~~Le document exige un fit **exprimé** et le palier de la manche [S-0]. Le moteur ne l'exige que pour les contrôles qu'il déclenche sous la manche.~~ **Tranché en faveur du document** : le fit exprimé est exigé partout, et le moteur nomme l'atout d'abord (voir [S-0]). Coût constaté : sur la donne de `TestControlBidThirdAceViaBlackwood`, le tour dépensé à dire l'atout n'enfle plus le plancher promis, le minimum combiné plafonne à 35 au lieu de 37 et le camp s'arrête au petit chelem là où le grand était sur table. Si ce grand pèse plus lourd que la règle, le levier est la barre des 37 points dans `keycardAnswer`, pas l'exigence de fit |
| 19 | **3SA « oui mais »** | Version étroite retenue (singleton dans la couleur du partenaire), la version large du document écartée [S-2d] |
| 20 | **Suite du 3SA « oui mais »** | Le moteur marque l'enchère forcing et laisse la machinerie générique placer le contrat ; le document ne décrit pas la suite |
| 21 | **Quatrième couleur à saut** | La fiche en fait un bicolore 5-5 naturel à honneurs concentrés, forcing de manche. Le moteur ne produit pas ce saut : toute quatrième couleur qu'il nomme est la demande [C-21], au palier le plus bas |
| 22 | **Troisième couleur forcing** | La fiche la dit « en principe forcing de manche » sans donner de plancher ; le moteur exige **11 H** et interdit la demande à une main déjà passée [C-28] |
| 23 | **Manche mineure après la dénégation d'arrêt** | [C-10] la prévoit, mais le seuil de 30 la met hors d'atteinte : le répondant est plafonné à 10 H par sa réponse d'1SA et l'ouvreur à 16 par sa redemande. La branche est donc théorique — à trancher : abaisser le seuil pour cette séquence, ou l'assumer |
| 24 | **Bicolore majeur 5-5 sur 1SA** | Le 4♦ est un **choix de manche**, pas une enchère de chelem : l'ouvreur nomme sa majeure et l'échange s'arrête là. Une 5-5 majeure très forte n'a pas d'autre route que celle-ci [N-3b] |
