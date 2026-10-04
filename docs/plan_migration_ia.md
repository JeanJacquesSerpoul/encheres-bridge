# Migration des enchères vers OpenRouter

## Le problème et l'approche

Les enchères sont aujourd'hui calculées **dans le navigateur** par un moteur Go
compilé en WebAssembly ([cli/bids.wasm](cli/bids.wasm), 4,5 Mo) lisant les règles
YAML d'un système ([cli/systems/sef/rules.yaml](cli/systems/sef/rules.yaml),
280 Ko). Le moteur, sa référence Python, ses PDF, ses données de test et ses
générateurs forment un écosystème de ~14 Mo ; ses enchères de compétition
s'arrêtent souvent au passe par défaut ; et toute la page en dépend.

On remplace ce moteur par un **appel au serveur IA d'openrouter_proxy**, déjà en
place pour la reconnaissance des cartes par photo. Le SEF n'est plus un programme
mais une **phrase dans le prompt**. Le client ne calcule plus d'enchères : il
demande, **valide**, et **écrit le résultat dans le PBN** de la donne.

Trois conséquences à assumer :

- **plus d'enchères hors ligne** pour une donne qui n'en a pas encore ;
- les enchères deviennent une **opinion de modèle**, non un jeu de règles
  reproductible — d'où l'empreinte et l'écriture dans le PBN, qui les figent ;
- le client publié sur GitHub Pages ne calcule plus rien sans un serveur IA
  joignable (aujourd'hui il est entièrement autonome).

## Décisions arrêtées

| Question | Décision |
|---|---|
| Sélecteur « Système » | Remplacé par un **champ de texte libre** : la fin du prompt d'enchères, valeur par défaut `Conforme au système d'enchère français 2024 (SEF 2024)`, mémorisée dans le navigateur |
| Chaîne Python (`tools/python_tools`) | **Supprimée entièrement** ; les thèmes deviennent des données figées |
| Enrichissement des 8 000 donnes | **Un thème échantillon d'abord** (500 donnes), puis décision |
| Points HL et type de main | **Recalculés dans le client**, à l'identique du moteur, dans un module testable |
| Langue des commentaires écrits en fichier | **Une seule**, choisie par l'outil (`--lang`, `fr` par défaut), **déclarée en tête de fichier** |

## Le contrat d'échange avec l'IA

### Ce qui part

`POST /api/chat` d'openrouter_proxy ([openrouter_proxy/internal/handler/chat.go](openrouter_proxy/internal/handler/chat.go))
— `{ text, model }` → `{ success, response, model, usage }`. Le corps accepte en
plus `image`, `pdf`, `provider` ; `DisallowUnknownFields` interdit tout autre
champ. Aucun mode JSON structuré n'est disponible : la sortie est du **texte
libre**, donc validée côté client.

Prompt d'enchères, en trois morceaux :

1. la **donne** : `[Dealer]`, `[Vulnerable]`, `[Deal]` du bloc courant ;
2. la **consigne de forme** : rendre le PBN complété d'une section
   `[Auction "<donneur>"]`, **un appel par ligne**, notation PBN standard
   (`Pass`, `X`, `XX`, `1C` … `7NT`), une ligne `% signification` facultative
   après l'appel concerné, et **rien d'autre** (ni préambule, ni bloc de code) ;
3. la **fin réglée** : le texte du champ libre, `Conforme au système d'enchère
   français 2024 (SEF 2024)` par défaut.

La consigne est dans la langue active de l'interface ; les **appels** sont en
notation PBN standard (indépendante de la langue), les **commentaires** dans la
langue active au moment de la demande.

### Ce qui revient, et ce qui est refusé

Réponse refusée **sans rien écrire** si :

| Cas | Détail |
|---|---|
| Texte parasite | ce qui entoure le PBN (préambule, ```, explication) est retiré ; s'il ne reste pas un PBN lisible, refus |
| Donne divergente | `[Dealer]`/`[Vulnerable]`/`[Deal]` renvoyés ≠ ceux envoyés → refus (la règle PBN veut que la valeur d'`[Auction]` soit le donneur) |
| `[Auction]` absente | refus (l'IA n'a pas rendu d'enchère) |
| Notation ambiguë | mélange de jetons français et anglais dans la même enchère → refus |
| Appel illégal | surcontre sans contre, contre d'un partenaire, palier inférieur ou égal au dernier contrat dans une dénomination plus basse |
| Enchère non terminée | la séquence ne finit pas par trois passes après la dernière enchère, ni quatre passes d'entrée, ni `AP` |
| `%` avant tout appel | un commentaire sans appel au-dessus ne se rattache à rien → refus |

**Rattachement des commentaires.** Une ligne `%` se rattache au **dernier appel
précédent**. C'est le seul lien : l'ordre du fichier fait foi.

**Notation acceptée en lecture, normalisée en écriture.** Le validateur accepte
`Pass`/`Passe`, `NT`/`SA`, `C`/`D`/`H`/`S`/`T`/`K`/`P`, mais `C` vaut ♣ en
anglais et ♥ en français. La convention se déduit donc de l'enchère entière :
les jetons décisifs sont `Passe`, `SA`, `T`, `K` (français) et `Pass`, `NT`
(anglais). Aucun des deux, ou les deux → refus. Ce qui est écrit en fichier est
toujours la **notation standard**.

### Ce qui est écrit dans le fichier

Après `[Deal]`, dans le même bloc, **sans ligne vide** (le client découpe les
fichiers sur les lignes vides, [splitPbnGames](cli/app.js:1645)) :

```
[Dealer "N"]
[Vulnerable "None"]
[Deal "N:AQ4.KJ3.98.KQT65 …"]
[Auction "N"]
1SA
% 15-17 HL, main régulière
Pass
2T
% Stayman, 4 réponses
…
Pass
Pass
Pass
```

Le format de référence du dépôt ([docs/pbn.txt](docs/pbn.txt) § 3.5) définit une
convention d'annotation **standard** par tags `[Note "1:…"]` et références
`=1=` sur les appels. Elle est plus interopérable que la ligne `%`, mais elle est
plus verbeuse et impose un compteur. **Le validateur la lira aussi** s'il la
rencontre ; l'écriture reste en `%`, comme convenu. *Point à confirmer à la
relecture : basculer l'écriture sur `[Note]` serait le choix le plus conforme.*

## Fichiers créés

| Fichier | Rôle |
|---|---|
| `cli/pbn.js` | PBN, sans DOM : découper en blocs, lire les tags, lire/écrire la section `[Auction]`, lire les remarques `%`, canonicaliser `[Deal]` (rangs triés), calculer l'empreinte |
| `cli/bids.js` | Sémantique, sans DOM : jetons d'appel, convention fr/en, légalité, fin d'enchère, contrat, déclarant, contre/surcontre, points H, **points HL**, **type de main** |
| `tests/pbn.test.js`, `tests/bids.test.js`, `tests/ai.test.js`, `tests/enrich.test.js` | Tests `node --test`, sans dépendance |
| `serve.py` | Serveur local : sert `cli/`, pose COOP/COEP et `Cache-Control: no-cache` (remplace `serve/main.go`) |
| `tools/enrich/enrich.js` | Outil d'enrichissement par thème, appel par donne, reprise, écriture dans le fichier |
| `tools/enrich/run.ps1`, `tools/enrich/run.sh` | Lancement de l'outil |

Chacun des deux modules du client est exporté à la fois comme globale (pour
`index.html`, sans étape de build) et via `module.exports` (pour les tests Node
et pour l'outil). Aucun `package.json` n'est ajouté : `node --test` fait partie
de Node.

## Détail par lot

### L1 — Vecteurs de test extraits du moteur (à faire en premier)

Avant toute suppression, tirer du moteur les valeurs de référence qui serviront
de fixtures aux tests JS :

- points **HL** et **type de main** pour les mains de `engine/testdata/` ;
- **contrat** et **déclarant** des enchères de `engine/testdata/systems/*/golden.json`
  (315 donnes) ;
- quelques séquences complètes, pour la lecture de tokens.

Sans ce lot, les points HL et le type de main ne seraient vérifiés que contre
eux-mêmes.

### L2 — `cli/pbn.js`

- `splitBlocks(text)`, `dealTags(block)`, `canonicalDeal(tag)`,
  `readAuction(block)`, `writeAuction(block, calls, comments)`.
- `readAuction` tolère l'absence de section, une section mal formée, une ligne
  vide au milieu (refus), les remarques `%` alors que le bloc en porte déjà en
  tête (`% Titre-FR:` des fichiers thématiques ne doit pas être pris pour un
  commentaire d'appel — [pbnTitles](cli/app.js:1270) reste le seul à les lire).
- `writeAuction` **remplace** une section existante au lieu de s'ajouter : un
  second clic ne doit pas empiler deux `[Auction]`.
- `fingerprint(block)` = `donneur | vulnérabilité | Deal canonicalisé`. La
  canonicalisation (rangs triés par couleur) est indispensable : la règle PBN
  accepte les rangs dans n'importe quel ordre à l'import et le client réécrit
  `[Deal]` dans son propre ordre, ce qui ferait croire à un changement de donne.

### L3 — `cli/bids.js`

- `parseCall`, `callToken`, `detectConvention`, `isLegal`, `isAuctionComplete`,
  `contract(auction)`, `declarer(auction)`, `doubled(auction)`.
- `hPoints(hand)` — déjà présent côté client sous le nom
  [handHCP](cli/app.js:2357), à déplacer ici ;
- `hlPoints(hand)`, `handType(hand)` — portage de
  [engine/features.go](engine/features.go), vérifié par les vecteurs de L1 ;
- `analyzeHand(hand)` → `{ h_points, hl_points, type }`.

### L4 — Action « Afficher les enchères »

Dans [cli/app.js](cli/app.js), remplace [simulate](cli/app.js:4878) et supprime
[scheduleAutoBid](cli/app.js:3115) et [autoBidAllowed](cli/app.js:3108).

- Aucun calcul automatique : ni au chargement, ni après un tirage, un
  déplacement de carte ou un changement de langue.
- Au clic : si le bloc courant porte une `[Auction]` **valide et concordante**,
  l'afficher sans réseau ; sinon appeler l'IA, valider, puis écrire.
- Le bouton est actif s'il y a une enchère locale utilisable **ou** si le serveur
  IA répond ; sinon il est éteint, avec la raison en clair sous lui.
- Le résultat est mis en forme **dans le même objet** que celui que rendait le
  moteur (`dealer`, `vulnerable`, `hands`, `auction`, `contract`, `declarer`,
  `doubled`) : [renderResult](cli/app.js:4402), [renderTrainTable](cli/app.js:5220)
  et [parSetDeal](cli/par.js) continuent de fonctionner sans être réécrits.
- `auction[].trace` disparaît (le moteur seul traçait un arbre) ;
  `auction[].comment` prend sa place.

### L5 — Empreinte et état périmé

- `shownResult = { fingerprint, result, source: "local" | "ia" }`.
- Quand `Dealer`/`Vulnerable`/`Deal` changent, l'ancienne analyse est **conservée
  dans le PBN** (la section `[Auction]` reste écrite) mais **marquée périmée** :
  `hideResult()` masque le panneau, la table revient en lecture **sans contrat**,
  et une ligne signale que l'enchère enregistrée ne correspond plus à la donne,
  avec le bouton pour la refaire.
- L'écriture de `[Auction]` ne rend **pas** périmé : la section ne touche ni
  `[Dealer]`, ni `[Vulnerable]`, ni `[Deal]`.

*Risque assumé, à confirmer :* un fichier sauvegardé après modification de la
donne portera une `[Auction]` qui ne correspond plus à son `[Deal]`. Conserver
l'analyse est ce qui a été demandé ; si ce risque gêne, la variante est de
**retirer** la section au moment de « Sauver le PBN » quand l'empreinte diffère.

### L6 — Réglages

[index.html](cli/index.html) : retrait de `#rules-system`, `#rules-help`, de la
fenêtre PDF (`#rules-pdf-dialog`) et des trois lignes d'état du moteur
(`#health-dot`, `#health-text`, `#app-version`, `#engine-error`,
`#engine-loading`).

Ajout, autour du serveur IA unique :

| Champ | Clé `localStorage` |
|---|---|
| Serveur IA (local/distant + URL) | `ia.mode`, `ia.local`, `ia.remote` *(existants)* |
| Modèle de reconnaissance des cartes | `ia.model` *(nouveau ; remplace la constante [IA_MODEL](cli/app.js:3706))* |
| Modèle d'enchères | `bids.model` *(nouveau)* |
| Fin du prompt d'enchères | `bids.bidSystem` *(nouveau)* |
| Reconnaissance par photo (boutons appareil photo) | `ia.enabled` *(existant, ne cache plus le serveur)* |

La clé `bids.rules` (système choisi) n'est plus lue ni écrite. Le bloc serveur
n'est plus masqué par la case photo : les enchères en dépendent aussi.

### L7 — Affichage du résultat

- Chaque main reçoit `analyzeHand` (L3) côté client : [handHTML](cli/app.js:1955)
  et [renderTrainTable](cli/app.js:5224) ne changent pas de forme.
- Le commentaire d'un appel n'est **plus écrit dans la liste** : l'icône d'arbre
  de décision, qui dépliait la trace du moteur, déplie désormais la ligne `%`
  rattachée à cet appel. [decisionTreeHTML](cli/app.js:5290) et
  `commentsTree` sont remplacés par un rendu de commentaire.
- L'entraînement garde son verdict « enchère attendue + explication » : c'est
  tout l'intérêt de l'exercice.

### L8 — Thèmes

- `cli/systems/sef/pbn/*.pbn` (16 fichiers, 8 000 donnes) → `cli/themes/`, avec
  son propre `index.json`. Les thèmes cessent d'appartenir à un système.
- `cli/systems/` disparaît (règles, PDF, `index.json`) ; `forgetThemes`,
  `systemDir`, `initRulesSystems`, `renderRulesSystems`, `rulesSystemPdf`,
  `openRulesPdf` et la fenêtre PDF sont retirés.
- `cli/themes/index.json` gagne le champ `lang` (langue des commentaires,
  décidée au lot L5 de l'outil) et le nombre de donnes enrichies.

### L9 — Entraînement

- [drawPracticePBN](cli/app.js:5520) : la branche `random` disparaît
  (`bids.trainSource` valant `random` retombe sur `all`).
- [pickThemedBlock](cli/app.js) ne retient que les blocs dont la section
  `[Auction]` est complète, valide et concordante avec la donne. Un thème sans
  aucune donne enrichie le dit clairement.
- [newTrainTable](cli/app.js:5553) n'appelle plus le moteur : l'objet `result`
  est construit avec `cli/bids.js` depuis l'enchère enregistrée.
- `declaringSeat` (main tirée dans le camp du déclarant) fonctionne sur le
  contrat calculé localement, sans changement.

### L10 — Outil d'enrichissement

`tools/enrich/enrich.js`, exécuté par Node :

```
node tools/enrich/enrich.js --theme cli/themes/drury.pbn \
    --server http://localhost:9013 --model <modèle> --lang fr [--limit 500]
```

- lit le fichier thème, garde les donnes **déjà enrichies et valides** telles
  quelles (c'est la reprise) ;
- appelle le proxy **une donne à la fois**, valide la réponse avec `cli/bids.js`,
  refuse tout ce qui ne passe pas (la donne reste non enrichie, à la prochaine
  passe) ;
- écrit dans le fichier au fil de l'eau, en **préservant les octets d'origine**
  pour les donnes inchangées ;
- cadence les appels sous la limite du proxy (`RATE_LIMIT_PER_MIN`, 20/min) ;
- journalise : donnes traitées, refusées, temps écoulé, jetons consommés.

`run.ps1` / `run.sh` lancent l'outil et rappellent le prérequis (`.env` du proxy
avec la clé, `ALLOWED_MODELS` contenant le modèle d'enchères).

### L11 — Serveur local Python

`serve.py` : sert `cli/` sur `localhost:9015`, pose `Cross-Origin-Opener-Policy`,
`Cross-Origin-Embedder-Policy` et `Cache-Control: no-cache` (ce que faisait
[serve/main.go](serve/main.go)). `run.ps1`, `run.sh` et `run-macos.command`
l'appellent et vérifient la présence de Python au lieu de Go ; l'option
`-ForceWasm` disparaît, `-Port` et `-NoBrowser` restent.

*Simplification possible :* `python -m http.server` seul, l'isolation étant déjà
assurée par [cli/coi-serviceworker.js](cli/coi-serviceworker.js) au prix d'un
rechargement à la première visite. Le serveur dédié évite ce rechargement.

### L12 — Suppression de l'ancien moteur

Retirés une fois les lots ci-dessus posés :

| Chemin | Taille |
|---|---|
| `engine/` (moteur, tests, `testdata`) | 12,9 Mo |
| `wasm/`, `serve/` | 5,7 Ko |
| `cli/bids.wasm`, `cli/wasm_exec.js`, `cli/bids-wasm.js` | 4,5 Mo |
| `cli/systems/` (règles 280 Ko, PDF 875 Ko, thèmes 1,3 Mo) | 2,4 Mo |
| `tools/python_tools/`, `tools/par/` | — |
| `docs/regles_moteur.md`, `Anatomie de rules*`, `donne.png`, `resultat.png` | 1,7 Mo |
| `go.mod`, `go.sum`, `build-wasm.{ps1,sh}`, `update-system.{ps1,sh}` | — |
| `.github/workflows/wasm.yml` | — |

Conservés : `docs/pbn.txt` (format de référence), `cli/par.js` et le solveur DDS
(le PAR ne dépend pas du moteur), `openrouter_proxy/`, `tools/tutorial/`.

À ajuster : `.gitignore` (entrées `bids`, `bids.exe`, `cli/bids_wasm_bin.js`)
et [THIRD-PARTY-NOTICES.md](THIRD-PARTY-NOTICES.md) — la section **yaml.v3**
tombe avec le moteur, celle de **DDS** reste.

### L13 — CI

- `tests.yml` (sur `pull_request`) : `node --test` sur `tests/`, puis
  `go test ./...` **dans `openrouter_proxy/`** — ses tests ne tournent
  aujourd'hui dans aucune CI.
- `pages.yml` : plus de `setup-go`, plus de `build-wasm.sh`, plus de
  `go test` ; la liste des actifs exigés est mise à jour (`cli/pbn.js`,
  `cli/bids.js`, `cli/themes/index.json` et les fichiers qu'il déclare).
- `wasm.yml` : supprimé.

### L14 — Documentation

- [README.md](README.md) : « Le moteur dans la page » devient « Le serveur IA
  d'enchères » ; « Voir les enchères », « S'entraîner », « Ce que le navigateur
  retient », « Reconnaissance des cartes par photo », « Tests », « Structure du
  projet », « Démarrage » (prérequis : Python, plus Go) et « Hébergement
  statique » (plus de type MIME `.wasm`, mais un serveur IA à joindre) sont
  repris. « Audit du par » et « Format de la réponse du moteur » disparaissent,
  remplacés par la description du prompt, du format `[Auction]` et de
  l'empreinte.
- Aide intégrée d'[index.html](cli/index.html) (42 mentions du moteur ou du
  système) et `TUTORIAL_STEPS` (129 mentions dans
  [app.js](cli/app.js)) réécrits.
- **Captures du tutoriel** (`cli/tutorial/{fr,en}/01..16.jpg`, 32 fichiers) :
  à reprendre, l'interface change visiblement. À lancer après les lots L4 à L9.

### L15 — Échantillon d'enrichissement

Un seul thème (500 donnes), pour mesurer coût, durée et qualité :

- qualité : taux de réponses valides du premier coup, cohérence des
  commentaires, enchères terminées ;
- coût : jetons consommés par donne ;
- durée : à 20 requêtes/minute, ≥ 25 min pour 500 donnes.

Puis décision pour les 15 autres thèmes (7 500 donnes, ≥ 6 h de cadence).

## Ordre d'exécution

```
L1 vecteurs ─┬─ L3 bids.js ─────────────┬─ L7 affichage ──┐
L2 pbn.js ───┴─ L4 action IA ── L5 périmé ──┘              │
                                        │                  │
L6 réglages ────────────────────────────┤                  │
L8 thèmes ── L9 entraînement ───────────┤                  │
L2 ──────── L10 outil ── L15 échantillon┘                  │
L11 serveur Python ─────────────────────────────────────────┤
                                        L12 suppression ── L13 CI ── L14 docs
```

## Risques et points ouverts

| # | Risque | Traitement proposé |
|---|---|---|
| 1 | **Le site publié ne calcule plus rien sans serveur IA.** GitHub Pages est aujourd'hui autonome. | Le dire en tête du README et dans l'aide ; garder l'analyse locale quand le PBN en porte une |
| 2 | **Non-reproductibilité** : deux appels sur la même donne peuvent differer. | L'enchère écrite dans le PBN fige le résultat ; un nouveau clic la remplace et le dit |
| 3 | **Qualité pédagogique** : l'enchère attendue devient l'avis d'un modèle, pas une règle. | Mesurer sur l'échantillon (L15) avant d'engager les 7 500 autres donnes |
| 4 | **Portée du retrait** : `docs/pbn.txt` reste, mais `Anatomie de rules.yaml` et `regles_moteur.md` documentent un langage de règles qui n'existe plus. | Supprimés avec le moteur |
| 5 | **Tutoriel** : 32 captures et les textes des 16 étapes. | Reprises au lot L14, après les changements d'interface |
| 6 | **Modèle d'enchères autorisé** : le proxy refuse (HTTP 400) tout modèle hors `DEFAULT_MODEL`/`ALLOWED_MODELS`. | Un modèle textuel doit être ajouté à `ALLOWED_MODELS` ; le message d'erreur du proxy est affiché tel quel |
| 7 | **Délais** : le proxy coupe à 60 s (`TIMEOUT`), le client à 90 s pour la vision. | Constante dédiée pour les enchères, sous le délai du serveur |
| 8 | **Clé `bids.rules`** laissée dans le navigateur des utilisateurs existants. | Ignorée ; suppression au passage |
| 9 | **Convention d'annotation** : ligne `%` (retenue) contre tags `[Note]` du standard. | `[Note]` lu, `%` écrit — bascule possible à la relecture |
| 10 | **Enrichissement partiel** : l'entraînement n'offre que les donnes enrichies. | Déjà prévu : les donnes non enrichies sont simplement écartées, et le compte est affiché par thème |

## Non-objectifs

- Ne pas changer le calcul du PAR ni le solveur DDS.
- Ne pas changer la reconnaissance des cartes par photo, sinon pour sortir
  `IA_MODEL` en réglage.
- Ne pas ajouter d'étape de build ni de paquet npm au client.
- Ne pas toucher au serveur `openrouter_proxy` : le contrat `/api/chat` suffit.
