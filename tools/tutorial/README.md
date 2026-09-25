# Captures du tutoriel

Le tutoriel du client (bouton **Tutoriel** du mode d'emploi, et **Afficher le
tutoriel** de l'écran d'accueil) fait défiler des étapes. Chacune a une capture
d'écran et une explication :

- les explications sont dans `cli/app.js`, tableau `TUTORIAL_STEPS` (français
  et anglais) ;
- les captures sont dans `cli/tutorial/<langue>/NN.jpg`, où `NN` est le rang de
  l'étape (01, 02…).

Les captures sont prises par `capture.js` avec Playwright. Il rejoue, pour
chaque langue, les scènes de l'application dans l'ordre des étapes : la donne
exemple, le menu des donnes, l'édition, la sélection de cartes, le glisser, les
bornes, les enchères, l'arbre de décision, le PAR, le questionnaire, les
réglages et la vue téléphone.

## Reprendre les captures

```sh
tools/tutorial/run.sh
```

Le script sert `cli/` en local (port 9377, ou `TUTORIAL_PORT`), prend les
captures, puis remplace `cli/tutorial/`. Il faut Node et Playwright avec
Chromium (`npm i -g playwright`, ou le Playwright déjà installé).

À relancer après tout changement visible de l'interface. **Une étape ajoutée
ou retirée dans `TUTORIAL_STEPS` demande la scène correspondante dans
`capture.js`**, au même rang : sinon les images et les textes se décalent.
