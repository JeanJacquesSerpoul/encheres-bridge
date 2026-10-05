# Rapport de cohérence du moteur d'enchères

Les tests disent si une enchère est légale, si les trois moteurs (Go, Python,
Node) concordent et si l'écart au par ne se dégrade pas. Ce rapport cherche ce
qu'aucun d'eux ne voit : les règles qui ne servent jamais, les séquences sans
règle, les libellés que la condition contredit, les thèmes PBN que le moteur
n'enchérit pas comme le fichier.

```bash
COHERENCE_OUT=tools/coherence/out go test -run TestCoherenceReport ./engine
```

```powershell
$env:COHERENCE_OUT="tools/coherence/out"; go test -run TestCoherenceReport ./engine
```

C'est un harnais, pas un test : sans `COHERENCE_OUT`, il est sauté, et
`go test ./...` n'en est pas changé. Le chemin relatif se lit depuis la racine
du dépôt. Il écrit, pour chaque système de `cli/systems/index.json`,
`coherence-<id>.md` (le rapport) et `coherence-<id>.json` (les totaux), dans
un dossier non versionné. Durée : un peu plus d'une minute.

## Corpus

Les 12 000 donnes du banc du par (`engine/testdata/par_bench.jsonl.gz`), plus
`COHERENCE_DEALS` donnes tirées au hasard (20 000 par défaut, graine
`COHERENCE_SEED`). À chaque décision, toute la liste des règles est rejouée,
pour voir aussi celles qui conviendraient mais viennent trop tard. Le relevé
est vérifié contre le moteur, décision par décision et enchère par enchère.

## Sections

| | Contrôle |
|---|---|
| A | Règles jamais choisies : masquées par une règle antérieure (nommée), condition jamais vraie, séquence jamais rencontrée ; règles d'option, non évaluées |
| B | Séquences sans règle (passe par défaut) : les plus fréquentes, celles des mains de 12 H et plus, celles qui suivent un forcing |
| C | Passe donné par une règle juste après un forcing du partenaire |
| D | Libellé contre condition : tranche de points ou levées de jeu du libellé qu'aucune borne de la condition ne reprend ; tranche de points du français absente de l'anglais |
| E | Traductions vides, identiques au français ou en français |
| F | Les tableaux « enchère \| signification » de `SEF_2024.md`, chaque ligne à côté des règles aux mêmes chiffres, pour la relecture |
| G | Thèmes PBN : l'enchère du fichier contre celle du moteur, avec le premier appel divergent |
| H | Les règles qui coûtent le plus d'IMP au banc du par |

Les sections B et D sont des pistes à trier, pas des fautes : une séquence
sans règle est souvent un passe normal, et un libellé peut citer les points
du partenaire.
