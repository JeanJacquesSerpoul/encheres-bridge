# Audit du moteur contre le par

Tire un lot de donnes aléatoires, fait jouer l'enchère complète par le moteur,
calcule le **par** de chaque donne en double-mort, et publie un rapport HTML qui
isole les écarts entre les deux.

```bash
tools/par/run.sh              # 1 000 donnes, graine 20260906
tools/par/run.sh 5000 42      # 5 000 donnes, graine 42
```

```powershell
tools\par\run.ps1                        # 1 000 donnes, graine 20260906
tools\par\run.ps1 -Deals 5000 -Seed 42   # 5 000 donnes, graine 42
```

La sortie va dans `tools/par/out/` (non versionné) :

| Fichier | Contenu |
|---------|---------|
| `auctions.jsonl` | une ligne par donne : PBN, enchère, contrat final |
| `par.json` | l'analyse complète : table de levées, par, écarts, catégories |
| `rapport.html` | le rapport lisible, à ouvrir dans un navigateur |

Une même graine redonne exactement les mêmes donnes, donc les mêmes enchères et
le même rapport : deux révisions du moteur se comparent ligne à ligne.

## Les deux moitiés

**Les enchères** viennent de `TestParAuditDump` (voir
[`audit_par_test.go`](../../engine/audit_par_test.go)), un harnais et non une
assertion : il est ignoré tant que `PAR_AUDIT_OUT` ne désigne pas un fichier, si
bien que `go test ./...` n'en voit rien. Les donnes sortent de `randomDeal`, le
donneur et la vulnérabilité du numéro d'étui (rotation de seize), l'enchère de
`NewEngine(d).Run()`.

**Le par** est calculé par [`audit.js`](audit.js) et [`par.js`](par.js). Les
levées double-mort viennent du solveur DDS compilé en WebAssembly qu'embarque
déjà le client (`cli/dds_web_wasm.js`) : il s'exporte en UMD, donc `require()`
suffit sous Node, sans navigateur ni isolation cross-origine. Comptez environ
quatre minutes pour mille donnes.

## Comment le par est calculé

Chacun leur tour, en partant du donneur, les camps font **l'enchère la moins
chère qui améliore leur marque** : un contrat qu'ils tiennent en double-mort, ou
un sacrifice contré qui coûte moins que le contrat adverse. Les défenseurs
contrent tout contrat qui chute et ne contrent jamais un contrat qui passe —
contrer un contrat qui passe ne ferait que payer le déclarant. L'enchère montant
strictement, la boucle s'arrête ; le dernier contrat annoncé est celui du par.

`par.js` n'a aucune entrée-sortie : il prend la table de levées et la
vulnérabilité, et rend le contrat du par, sa marque et le barème (levées,
surlevées, chutes contrées ou non, primes de manche et de chelem, conversion en
IMP).

## Les six écarts du rapport

Une donne peut en cumuler plusieurs.

| # | Écart | Règle |
|---|-------|-------|
| 1 | Camp déclarant inversé | le contrat final et celui du par sont joués par des camps différents |
| 2 | Couleur différente du par | la dénomination du dernier contrat annoncé diffère de celle du par |
| 3 | Chelem demandé, impossible | palier 6 ou 7 annoncé, et le contrat chute en double-mort |
| 4 | Chelem manqué | arrêt sous le palier de 6 alors qu'un camp encaisse douze levées |
| 5 | Manche demandée, impossible | 3SA, 4 en majeure ou 5 en mineure annoncé, et le contrat chute |
| 6 | Manche manquée | arrêt sous la manche alors qu'un camp réalise une manche |

Les catégories 1 et 2 sont les plus peuplées, et une bonne part de leurs donnes
est saine : le par est souvent un **sacrifice** adverse, que le moteur n'a aucune
raison d'annoncer si personne ne l'y pousse. Le rapport affiche le nombre de pars
qui sont des sacrifices, pour situer ce bruit de fond.

## Lire le rapport

Le rapport s'ouvre sur la vue d'ensemble (contrats tenus, marque égale au par,
écart moyen en IMP, distribution des paliers), puis six onglets, un par écart.
Chaque ligne se déplie sur le diagramme des quatre mains, la séquence d'enchères
et la table des levées double-mort, où le contrat joué et celui du par sont
repérés.

`Δ IMP` compare la marque réellement obtenue à celle du par, vue de NS. Un écart
positif veut dire que NS a fait **mieux** que le par : le par suppose les deux
camps parfaitement compétitifs, or l'adversaire n'a pas toujours trouvé le
sacrifice que le double-mort autorisait.
