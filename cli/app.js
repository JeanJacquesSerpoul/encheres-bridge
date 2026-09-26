"use strict";

const $ = (sel) => document.querySelector(sel);

// ---------- configuration ----------

// Les enchères sont toujours calculées dans la page, par le moteur Go compilé
// en WebAssembly (bids-wasm.js) : il n'y a aucun serveur d'enchères à viser.
// Seul le serveur IA, facultatif, se choisit encore (voir plus bas).

// Le serveur d'IA (openrouter_proxy) prête son /api/chat à la lecture des
// cartes sur une photo. Il est facultatif : sans lui tout le reste du client fonctionne, seuls
// les boutons photo s'éteignent (voir renderPhotoButtons). En local il écoute
// sur 9013 ; en production il est proxifié sous un préfixe propre au
// déploiement, d'où l'absence de valeur par défaut en distant : la figer ferait
// pointer n'importe quelle copie du client vers un serveur qui n'est pas le
// sien.
const DEFAULT_LOCAL_IA = "http://localhost:9013";
const IA_LOCAL_KEY = "ia.local";
const IA_REMOTE_KEY = "ia.remote";
const IA_MODE_KEY = "ia.mode";
const IA_ENABLED_KEY = "ia.enabled";

const iaModeSelect = $("#ia-mode");
iaModeSelect.value = readIaMode();
$("#ia-server").value =
  iaModeSelect.value === "remote" ? readIaRemote() : readIaLocal();
iaModeSelect.addEventListener("change", applyIaMode);

// Option générale, par défaut OFF. Tant qu'elle l'est, la reconnaissance des
// cartes par photo est entièrement retirée de l'interface : 2ᵉ barre serveur,
// boutons appareil photo (celui de la donne comme ceux des mains) et état du
// serveur IA au pied de page. Cochée, on retrouve l'affichage habituel.
const iaEnabledToggle = $("#ia-enabled");
iaEnabledToggle.checked = readIaEnabled();
iaEnabledToggle.addEventListener("change", applyIaFeature);

function iaEnabled() {
  return iaEnabledToggle.checked;
}

function readIaEnabled() {
  return readStored(IA_ENABLED_KEY, "") === "1";
}

// Applique l'option : montre ou cache tout ce qui touche au serveur IA, puis
// redessine les mains (les boutons appareil photo n'y sont émis que si l'option
// est active) et ne sonde le serveur que lorsqu'il sert à quelque chose.
function applyIaFeature() {
  const on = iaEnabled();
  saveStored(IA_ENABLED_KEY, on ? "1" : "0");
  for (const sel of ["#ia-server-bar", "#ia-health-check", "#photo-status"]) {
    $(sel).classList.toggle("hidden", !on);
  }
  renderBoundsCards(); // réémet (ou retire) les boutons photo des mains
  renderPhotoButtons();
  // Rien ne doit survivre à la bascule : une photo en cours d'édition non plus.
  if (!on) closePhotoEditor();
  if (on) checkIaHealth();
}

$("#ia-server").addEventListener("change", () => {
  saveStored(iaModeSelect.value === "remote" ? IA_REMOTE_KEY : IA_LOCAL_KEY, iaURL());
  renderIaHint();
  checkIaHealth();
});

// Même bascule que pour le serveur d'enchères, et mêmes raisons.
function applyIaMode() {
  const mode = iaModeSelect.value;
  saveStored(IA_MODE_KEY, mode);
  $("#ia-server").value = mode === "remote" ? readIaRemote() : readIaLocal();
  renderIaHint();
  // Même abstention que pour le serveur d'enchères, avec une conséquence de
  // plus : l'état « inconnu » éteint les boutons photo, qui n'auraient rien à
  // interroger.
  if (mode === "remote" && !iaURL()) $("#ia-server").focus();
  checkIaHealth();
}

function renderIaHint() {
  const t = UI_TEXT[$("#lang").value];
  const remote = iaModeSelect.value === "remote";
  const missing = remote && !iaURL();
  const hint = $("#ia-hint");
  hint.textContent = missing ? t.serverRemoteHint : "";
  hint.classList.toggle("hidden", !missing);
  $("#ia-server").placeholder = remote ? t.serverIaPlaceholder : "";
  $("#ia-mode").querySelector('option[value="local"]').textContent =
    `${t.serverLocal} (${readIaLocal().replace(/^https?:\/\//, "")})`;
}

// La donne exemple, rappelée par son bouton : une manche à SA par Stayman.
const EXAMPLE_PBN = `[Dealer "S"]
[Vulnerable "All"]
[Deal "S:943.T3.Q753.Q983 J8.A54.AKT8.K765 AKT.J92.J964.J42 Q7652.KQ876.2.AT"]`;


// La dernière donne complète de l'utilisateur — quatre mains de 13 cartes —
// est retenue d'une visite à l'autre, et c'est elle qui revient au chargement
// suivant. Une donne en cours de composition ne l'écrase pas, la donne
// exemple non plus : elle n'est qu'à un clic, et la retenir ferait perdre la
// donne de l'utilisateur. Déplacer une de ses cartes en fait, elle, une donne
// de l'utilisateur.
const LAST_DEAL_KEY = "bids.lastDeal";

function dealTagOf(block) {
  const m = block.match(/\[Deal\s+"([^"]*)"\]/i);
  return m ? m[1].trim().toUpperCase() : "";
}

function rememberCompleteDeal() {
  const block = pbnGames[selectedGameIdx];
  if (!block || zoneCount(UNASSIGNED) > 0) return;
  if (SEATS.some((seat) => zoneCount(seat) !== HAND_SIZE)) return;
  if (dealTagOf(block) === dealTagOf(EXAMPLE_PBN)) return;
  saveStored(LAST_DEAL_KEY, block);
}

// Vrai quand la donne affichée vient d'un fichier : c'est lui qui dit qui
// donne et qui est vulnérable, les sélecteurs le recopient et se verrouillent.
// Générer une donne les rend à l'utilisateur.
let dealFromFile = false;

// Les textes de l'interface elle-même. Les éléments statiques les réclament
// par `data-i18n` (contenu) ou `data-i18n-title` (infobulle) ; les libellés
// qui existent déjà ailleurs — sièges, vulnérabilités — sont repris de
// SEAT_LABEL et VUL_LABEL plutôt que redits ici. Voir applyLang.
const UI_TEXT = {
  fr: {
    serverLocal: "Local",
    serverRemote: "Distant (production)",
    serverRemoteHint: "Saisissez l'URL du serveur distant.",
    errTimeout: "Le serveur n'a pas répondu à temps. Vérifiez son URL.",
    errUnreachable: "Serveur injoignable. Vérifiez son URL.",
    wasmMissing: "Moteur d'enchères (WASM) introuvable — lancez build-wasm.sh.",
    wasmMissingExec: "wasm_exec.js absent — lancez build-wasm.sh pour le moteur du navigateur.",
    wasmFailed: "Le moteur d'enchères n'a pas répondu.",
    wasmUnsupported: "Ce navigateur ne gère pas WebAssembly.",
    serverIa: "Serveur IA",
    serverIaPlaceholder: "https://exemple.net/openrouter-proxy",
    iaFeature: "Serveur IA de reconnaissance des cartes",
    serverTest: "Tester",
    healthUnknown: "état inconnu",
    versionWasm: "Moteur d'enchères",
    versionWasmTitle: "Version du moteur exécuté dans la page",
    versionModified: "compilé sur un dépôt modifié",
    online: "en ligne",
    offline: "injoignable",
    engineLoading: "Chargement du moteur d'enchères…",
    dealPanel: "Donne",
    fileLoad: "Charger un fichier .pbn",
    randomDeal: "Donne aléatoire",
    exampleDeal: "Donne exemple",
    fileSave: "Sauver le PBN",
    fileSaveStem: "donne",
    photoDeal: "Photographier les quatre mains",
    photoOffline: "Serveur IA injoignable : reconnaissance des cartes indisponible",
    photoUnknown: "État du serveur IA inconnu : reconnaissance des cartes indisponible",
    photoNoUrl: "Renseignez l'URL du serveur IA.",
    photoBusy: "Lecture des cartes…",
    photoBusyHint: "Cela peut prendre jusqu'à une minute et demie.",
    photoElapsed: (s) => `${s} s écoulée${s > 1 ? "s" : ""}`,
    photoCancelled: "Lecture des cartes annulée.",
    photoFailed: "Le serveur IA n'a pas su lire les cartes.",
    photoBadAnswer: "Réponse du serveur IA illisible.",
    photoNone: "Aucune carte reconnue sur la photo.",
    photoRead: (n) => `${n} carte${n > 1 ? "s" : ""} reconnue${n > 1 ? "s" : ""}.`,
    photoDropped: (n) => ` ${n} écartée${n > 1 ? "s" : ""} (doublon ou main pleine).`,
    photoEditTitle: "Recadrer et pivoter la photo",
    photoEditHint: "Faites glisser sur l'image pour choisir la zone à lire ; les poignées ajustent le cadre. Au clavier : flèches pour déplacer le cadre, Maj + flèches pour le redimensionner.",
    photoRotateLeft: "Pivoter d'un quart de tour à gauche",
    photoRotateRight: "Pivoter d'un quart de tour à droite",
    photoCropReset: "Image entière",
    photoCropLabel: "Zone à lire : flèches pour la déplacer, Maj + flèches pour la redimensionner.",
    photoCancel: "Annuler",
    photoConfirm: "Lire les cartes",
    photoCropTooSmall: "Zone de recadrage trop petite.",
    dealer: "Donneur",
    dealerTitle: "Donneur des donnes générées ; « Aléatoire » en tire un à chaque fois.",
    vulnerability: "Vulnérabilité",
    vulTitle: "Vulnérabilité des donnes générées ; « Aléatoire » en tire une à chaque fois.",
    lockedTitle: "Donné par le fichier PBN chargé. Modifier directement le texte la donne (PBN).",
    optRandom: "Aléatoire",
    language: "Langue",
    theme: "Thème",
    themeTitle: "Thème du système — c'est le choix par défaut — ou thème imposé à la page.",
    themeAuto: "Automatique",
    themeLight: "Clair",
    themeDark: "Sombre",
    helpOpen: "Mode d'emploi",
    helpTitle: "Mode d'emploi",
    helpClose: "Fermer le mode d'emploi",
    intro:
      "Composez une donne — l'application déroule les enchères du système " +
      "français et les commente, enchère par enchère.",
    pbnToggle: "Texte de la donne (format PBN)",
    moreDeals: "Autres façons d'obtenir une donne",
    editDeal: "Modifier la donne",
    editDone: "Terminer",
    editDoneBlocked: "Complétez les quatre mains pour revenir à la table.",
    shareMenu: "Partager",
    settings: "Réglages",
    tabsLabel: "Que faire de la donne",
    tabBids: "Enchères",
    tabTrain: "S'entraîner",
    tabPar: "PAR",
    parEmpty: "Le PAR s'affiche ici pour une donne complète dont les enchères sont calculées.",
    bidsEmpty: "Composez une donne complète, puis « Afficher les enchères » : la séquence et ses commentaires s'affichent ici.",
    pbnCopy: "Copier le texte PBN",
    pbnClose: "Masquer le texte PBN",
    shareLink: "Copier le lien de cette donne",
    shareCopied: "Lien de la donne copié",
    shareFailed: "Copie impossible : le lien n'a pas pu être placé dans le presse-papiers.",
    printResult: "Imprimer le résultat",
    pbnCopied: "Texte PBN copié",
    pbnCopyFailed: "Copie impossible : sélectionnez le texte et copiez-le à la main.",
    pbnNote: "Format texte standard des donnes de bridge. Collez-en une reçue par courriel, ou corrigez celle-ci à la main : le tableau de cartes suit.",
    dealToUse: "Donne à utiliser",
    dealWord: "Donne",
    boardWord: "plateau",
    dealerWord: "donneur",
    runAuction: "Afficher les enchères",
    yourHand: "Votre main",
    seatTip: (seat) => `Votre main est en ${seat}`,
    startQuiz: "Commencer le questionnaire",
    quizPanel: "Questionnaire d'enchères",
    auctionSeq: "Séquence d'enchères",
    continue: "Continuer",
    // Le questionnaire, jusqu'ici écrit en ternaires dans renderQuizStep,
    // chooseBid et finishQuiz. « Passe », « Contre » et « Surcontre » restent
    // hors d'ici à dessein : parseBidToken s'en sert comme jetons de
    // comparaison, les déplacer découplerait l'affichage de la valeur.
    quizYourTurn: (seat) => `À vous de parler (${seat}) — choisissez votre enchère.`,
    bbLevel: "Palier",
    bbStrain: "Dénomination",
    bbKeys: "Clavier : 1 à 7 pour le palier, C D H S N pour ♣ ♦ ♥ ♠ SA, P pour passe, X pour contre.",
    quizAboutToBid: (seat) => `${seat} va annoncer.`,
    quizCorrect: "✓ Correct !",
    quizWrong: "✗ Différent du système SEF",
    quizExpected: "Enchère attendue",
    treeShow: "Voir l'arbre de décision",
    treeHide: "Masquer l'arbre de décision",
    treeWhy: "Pourquoi",
    treeHand: (seat, h, hl, shape) => `main de ${seat} : ${h} H, ${hl} HL, ${shape}`,
    quizDone: "Questionnaire terminé.",
    quizScore: "Score",
    quizFinalContract: "Contrat final",
    quizShowDetail: "Afficher le détail complet",
    quizRecapTitle: "Vos erreurs",
    quizRecapNone: "Aucune erreur : toutes vos enchères sont celles du SEF.",
    appTitle: "Enchères au bridge",
    backToDeal: "Revenir à la donne",
    quizDealHidden: "Donne masquée.",
    quizMode: "Mode questionnaire",
    quizModeHint: "La donne reste masquée dès qu'elle est tirée ou chargée : vous enchérissez sans la connaître.",
    quizShowDeal: "Afficher la donne",
    quizCancel: "Annuler",
    welcomeTitle: "Bienvenue",
    welcomeText: "Composez une donne de bridge : l'application déroule ses enchères selon le Système d'Enchères Français et les commente, ou vous fait enchérir à la place d'un joueur.",
    welcomeTutorial: "Afficher le tutoriel",
    welcomeSkip: "Continuer sans le tutoriel",
    welcomeNote: "Le tutoriel et le mode d'emploi restent accessibles à tout moment par les boutons en haut de la page.",
    tutorialTitle: "Tutoriel",
    tutorialOpen: "Tutoriel",
    tutorialPrev: "‹ Précédent",
    tutorialNext: "Suivant ›",
    tutorialDone: "Terminer",
    tutorialZoom: "Agrandir l'image",
    quizReplay: "Rejouer cette donne",
    quizNewDeal: "Nouvelle donne",
    hiddenHand: "main cachée",
    dealerCap: "Donneur",
    byWord: "par",
    result: "Résultat",
    comments: "Séquence commentée",
    parCompute: "Calcul du PAR",
    parContracts: "Contrats",
    parComputing: "Calcul en cours…",
    parLoading: "Chargement du solveur…",
    parUnavailable: "Solveur double-mort (WASM) introuvable.",
    parNoIsolation: "Isolation cross-origine requise (SharedArrayBuffer) ; rechargez la page.",
    parLeadHint: "Survolez une case — ou touchez-la — pour voir l'entame qui tient le contrat à ce nombre de levées.",
    // Encadre le lien vers le projet DDS : son nom n'est pas traduit, et
    // applyLang ne peut pas poser de lien dans un [data-i18n].
    // Licence de l'application et lien vers son code, au pied de page.
    // Même découpe que parCredit ci-dessous, pour la même raison.
    appAuthor: "Bridge Bidding de Jean-Jacques Serpoul",
    // Majuscule : « Licence » ouvre sa propre ligne, ce n'est plus la suite
    // d'une phrase commencée au-dessus.
    appLicence: "Licence",
    appSource: "Code source sur GitHub",
    parCredit: "Levées double-mort calculées par",
    parCreditAuthors: "de Bo Haglund et Søren Hein — licence",
    parLeadComputing: "Recherche de l'entame…",
    parLeadVerb: "entame",
    parLeadAny: "ce qu'il veut",
    parLeadExcept: "tout sauf",
    parLeadOther: "Autre entame",
    parLeadThose: "Avec ces cartes",
  },
  en: {
    serverLocal: "Local",
    serverRemote: "Remote (production)",
    serverRemoteHint: "Enter the remote server URL.",
    errTimeout: "The server did not answer in time. Check its URL.",
    errUnreachable: "Server unreachable. Check its URL.",
    wasmMissing: "Bidding engine (WASM) not found — run build-wasm.sh.",
    wasmMissingExec: "wasm_exec.js is missing — run build-wasm.sh for the in-browser engine.",
    wasmFailed: "The bidding engine did not answer.",
    wasmUnsupported: "This browser does not support WebAssembly.",
    serverIa: "AI server",
    serverIaPlaceholder: "https://example.net/openrouter-proxy",
    iaFeature: "AI card-recognition server",
    serverTest: "Test",
    healthUnknown: "unknown state",
    versionWasm: "Bidding engine",
    versionWasmTitle: "Version of the engine running in the page",
    versionModified: "built from a modified tree",
    online: "online",
    offline: "unreachable",
    engineLoading: "Loading the bidding engine…",
    dealPanel: "Deal",
    fileLoad: "Load a .pbn file",
    randomDeal: "Random deal",
    exampleDeal: "Example deal",
    fileSave: "Save the PBN",
    fileSaveStem: "deal",
    photoDeal: "Photograph the four hands",
    photoOffline: "AI server unreachable: card recognition unavailable",
    photoUnknown: "AI server state unknown: card recognition unavailable",
    photoNoUrl: "Enter the AI server URL.",
    photoBusy: "Reading the cards…",
    photoBusyHint: "This may take up to a minute and a half.",
    photoElapsed: (s) => `${s} s elapsed`,
    photoCancelled: "Card reading cancelled.",
    photoFailed: "The AI server could not read the cards.",
    photoBadAnswer: "Unreadable answer from the AI server.",
    photoNone: "No card recognised in the photo.",
    photoRead: (n) => `${n} card${n > 1 ? "s" : ""} recognised.`,
    photoDropped: (n) => ` ${n} discarded (duplicate or full hand).`,
    photoEditTitle: "Crop and rotate the photo",
    photoEditHint: "Drag on the image to choose the area to read; the handles adjust the frame. By keyboard: arrows move the frame, Shift + arrows resize it.",
    photoRotateLeft: "Rotate a quarter turn to the left",
    photoRotateRight: "Rotate a quarter turn to the right",
    photoCropReset: "Whole image",
    photoCropLabel: "Area to read: arrows move it, Shift + arrows resize it.",
    photoCancel: "Cancel",
    photoConfirm: "Read the cards",
    photoCropTooSmall: "The crop area is too small.",
    dealer: "Dealer",
    dealerTitle: "Dealer of the generated deals; “Random” draws one every time.",
    vulnerability: "Vulnerability",
    vulTitle: "Vulnerability of the generated deals; “Random” draws one every time.",
    lockedTitle: "Set by the loaded PBN file. Directly edit the deal text (PBN).",
    optRandom: "Random",
    language: "Language",
    theme: "Theme",
    themeTitle: "Use the system theme — the default — or force one for the page.",
    themeAuto: "Automatic",
    themeLight: "Light",
    themeDark: "Dark",
    helpOpen: "How to use",
    helpTitle: "How to use",
    helpClose: "Close the guide",
    intro:
      "Build a deal — the application runs the French system's auction and " +
      "comments on it, call by call.",
    pbnToggle: "Deal as text (PBN format)",
    moreDeals: "Other ways to get a deal",
    editDeal: "Edit the deal",
    editDone: "Done",
    editDoneBlocked: "Complete all four hands to go back to the table.",
    shareMenu: "Share",
    settings: "Settings",
    tabsLabel: "What to do with the deal",
    tabBids: "Auction",
    tabTrain: "Practise",
    tabPar: "Par",
    parEmpty: "The par shows up here for a complete deal whose auction has been computed.",
    bidsEmpty: "Build a complete deal, then \u201cRun the auction\u201d: the calls and their meaning show up here.",
    pbnCopy: "Copy the PBN text",
    pbnClose: "Hide the PBN text",
    shareLink: "Copy a link to this deal",
    shareCopied: "Deal link copied",
    shareFailed: "Could not copy: the link could not be put on the clipboard.",
    printResult: "Print the result",
    pbnCopied: "PBN text copied",
    pbnCopyFailed: "Could not copy: select the text and copy it by hand.",
    pbnNote: "The standard text format for bridge deals. Paste one you received by e-mail, or fix this one by hand: the card table follows.",
    dealToUse: "Deal to use",
    dealWord: "Deal",
    boardWord: "board",
    dealerWord: "dealer",
    runAuction: "Run the auction",
    yourHand: "Your hand",
    seatTip: (seat) => `Your hand is ${seat}`,
    startQuiz: "Start the quiz",
    quizPanel: "Bidding quiz",
    auctionSeq: "Auction",
    continue: "Continue",
    quizYourTurn: (seat) => `Your turn to bid (${seat}) — choose your call.`,
    bbLevel: "Level",
    bbStrain: "Strain",
    bbKeys: "Keyboard: 1 to 7 for the level, C D H S N for ♣ ♦ ♥ ♠ NT, P to pass, X to double.",
    quizAboutToBid: (seat) => `${seat} is about to bid.`,
    quizCorrect: "✓ Correct!",
    quizWrong: "✗ Not what the SEF system bids",
    quizExpected: "Expected call",
    treeShow: "Show the decision tree",
    treeHide: "Hide the decision tree",
    treeWhy: "Why",
    treeHand: (seat, h, hl, shape) => `${seat}'s hand: ${h} H, ${hl} HL, ${shape}`,
    quizDone: "Quiz complete.",
    quizScore: "Score",
    quizFinalContract: "Final contract",
    quizShowDetail: "Show full detail",
    quizRecapTitle: "Your mistakes",
    quizRecapNone: "No mistakes: every call matches the SEF.",
    appTitle: "Bridge Bidding",
    backToDeal: "Back to the deal",
    quizDealHidden: "Deal hidden.",
    quizMode: "Quiz mode",
    quizModeHint: "The deal stays hidden as soon as it is drawn or loaded: you bid without knowing it.",
    quizShowDeal: "Show the deal",
    quizCancel: "Cancel",
    welcomeTitle: "Welcome",
    welcomeText: "Build a bridge deal: the app runs its auction following the French bidding system (SEF) and explains every call, or has you bid in place of one player.",
    welcomeTutorial: "Show the tutorial",
    welcomeSkip: "Continue without the tutorial",
    welcomeNote: "The tutorial and the guide are always available from the buttons at the top of the page.",
    tutorialTitle: "Tutorial",
    tutorialOpen: "Tutorial",
    tutorialPrev: "‹ Previous",
    tutorialNext: "Next ›",
    tutorialDone: "Finish",
    tutorialZoom: "Enlarge the picture",
    quizReplay: "Replay this deal",
    quizNewDeal: "New deal",
    hiddenHand: "hidden hand",
    dealerCap: "Dealer",
    byWord: "by",
    result: "Result",
    comments: "Annotated auction",
    parCompute: "Compute the par",
    parContracts: "Contracts",
    parComputing: "Computing…",
    parLoading: "Loading the solver…",
    parUnavailable: "Double-dummy solver (WASM) not found.",
    parNoIsolation: "Cross-origin isolation required (SharedArrayBuffer); reload the page.",
    parLeadHint: "Hover a cell — or tap it — to see the lead that holds declarer to that many tricks.",
    // « license » et non « licence » : c'est l'orthographe que l'attribution
    // du solveur emploie déjà en anglais, quelques lignes plus bas.
    appAuthor: "Bridge Bidding by Jean-Jacques Serpoul",
    // « Licensed under GPL-3.0 » : « Licence GPL-3.0 » se dit tel quel en
    // français, pas en anglais, où il faut la préposition.
    appLicence: "Licensed under",
    appSource: "Source code on GitHub",
    parCredit: "Double dummy tricks computed by",
    parCreditAuthors: "from Bo Haglund and Søren Hein — license",
    parLeadComputing: "Solving the lead…",
    parLeadVerb: "leads",
    parLeadAny: "anything",
    parLeadExcept: "anything but",
    parLeadOther: "Any other lead",
    parLeadThose: "With those",
  },
};

const LANG_KEY = "bids.lang";

// La langue de départ : celle que l'utilisateur a choisie la dernière fois,
// sinon la première de ses langues de navigation que l'on sait parler,
// l'anglais servant de repli. Un choix explicite gagne donc toujours sur le
// navigateur, y compris pour « en » sur un navigateur en français.
function initialLang() {
  const saved = readLang();
  if (saved) return saved;
  const preferred = navigator.languages && navigator.languages.length
    ? navigator.languages
    : [navigator.language || ""];
  for (const tag of preferred) {
    const base = tag.toLowerCase().split("-")[0];
    if (base in UI_TEXT) return base;
  }
  return "en";
}

// Le stockage local peut être refusé (navigation privée, page ouverte en
// file://) : la langue n'est alors simplement pas mémorisée.
function readLang() {
  try {
    const saved = localStorage.getItem(LANG_KEY);
    return saved in UI_TEXT ? saved : null;
  } catch (err) {
    return null;
  }
}

function saveLang(lang) {
  try {
    localStorage.setItem(LANG_KEY, lang);
  } catch (err) {
    /* rien à faire : la langue vaudra pour cette session seulement */
  }
}

// Mémorisation des réglages, même repli que pour la langue : sans stockage
// accessible, ils ne valent que pour la session.
function readStored(key, fallback) {
  try {
    return localStorage.getItem(key) || fallback;
  } catch (err) {
    return fallback;
  }
}

function saveStored(key, value) {
  try {
    localStorage.setItem(key, value);
  } catch (err) {
    /* rien à faire : le réglage vaudra pour cette session seulement */
  }
}

// Le thème de la page, choisi dans le bandeau et mémorisé comme la langue.
// Trois choix : « auto », celui par défaut, qui suit la préférence du système,
// puis « light » et « dark », qui la forcent. C'est le choix qui est conservé ;
// l'attribut posé sur <html>, lui, dit toujours un thème résolu — clair ou
// sombre — car c'est de lui que style.css tire sa palette entière, et que
// celle-ci n'est écrite qu'une fois.
const THEME_KEY = "bids.theme";
const THEME_CHOICES = ["auto", "light", "dark"];

function readThemeChoice() {
  const saved = readStored(THEME_KEY, "auto");
  return THEME_CHOICES.includes(saved) ? saved : "auto";
}

function saveTheme(choice) {
  saveStored(THEME_KEY, choice);
}

// Ce que le système préfère ; un navigateur sans matchMedia — ou qui n'en dit
// rien — préfère le clair, qui est le défaut de la feuille.
function systemTheme() {
  return window.matchMedia && window.matchMedia("(prefers-color-scheme: dark)").matches
    ? "dark"
    : "light";
}

// Le thème effectivement affiché, résolu en clair ou en sombre.
function activeTheme() {
  const choice = readThemeChoice();
  return choice === "auto" ? systemTheme() : choice;
}

// Le thème vit sur <html data-theme="…">, dont style.css tire sa palette
// entière et son color-scheme. Le script en tête de index.html a déjà posé
// l'attribut, lu du même stockage, pour que la page s'ouvre sur le bon thème
// dès le premier rendu ; ici, on ne fait que suivre le sélecteur. L'indice
// color-scheme du <head> est reposé avec lui : il ne sert qu'avant l'arrivée
// de la feuille — c'est la règle CSS qui décide ensuite — mais le laisser
// mentir sur le thème affiché serait une tromperie de plus à relire.
function applyTheme() {
  const theme = activeTheme();
  document.documentElement.dataset.theme = theme;
  const meta = document.querySelector('meta[name="color-scheme"]');
  if (meta) meta.setAttribute("content", theme);
  $("#theme").value = readThemeChoice();
}

$("#theme").addEventListener("change", () => {
  saveTheme($("#theme").value);
  applyTheme();
});

// En mode automatique, le thème suit le système d'un changement à l'autre :
// basculer le mode sombre de Windows ne doit pas demander un rechargement. Le
// choix est relu à chaque fois — un thème forcé n'a rien à suivre — et
// applyTheme se charge du reste.
if (window.matchMedia) {
  window.matchMedia("(prefers-color-scheme: dark)").addEventListener("change", () => {
    if (readThemeChoice() === "auto") applyTheme();
  });
}

// URL du serveur IA. Faute de valeur retenue, le local retombe sur son port
// d'écoute par défaut ; le distant sur une chaîne vide, qui force la saisie.
function readIaLocal() {
  return readStored(IA_LOCAL_KEY, DEFAULT_LOCAL_IA);
}

function readIaRemote() {
  return readStored(IA_REMOTE_KEY, "");
}

function readIaMode() {
  return readStored(IA_MODE_KEY, "local") === "remote" ? "remote" : "local";
}

// Capitalise un libellé écrit pour couler dans une phrase (« personne »),
// afin qu'il tienne seul dans un menu déroulant.
function capitalize(s) {
  return s.charAt(0).toUpperCase() + s.slice(1);
}

// Réécrit toute l'interface dans la langue choisie. Le questionnaire en cours
// garde la sienne : il a été construit avec les commentaires que le serveur a
// renvoyés à son démarrage, et le relancer effacerait la progression.
// Ses boutons (« Continuer », « Annuler »), eux, suivent la langue de la page
// comme le reste des libellés fixes.

function applyLang() {
  const lang = $("#lang").value;
  const t = UI_TEXT[lang];
  document.documentElement.lang = lang;
  for (const el of document.querySelectorAll("[data-i18n]")) {
    el.textContent = t[el.dataset.i18n];
  }
  for (const el of document.querySelectorAll("[data-i18n-title]")) {
    el.title = t[el.dataset.i18nTitle];
  }
  // Un bouton qui n'a qu'une icône n'a pas de texte à lire : son nom vient
  // d'ici. Le `title` seul y suppléait, mais il dépend d'un survol.
  for (const el of document.querySelectorAll("[data-i18n-label]")) {
    el.setAttribute("aria-label", t[el.dataset.i18nLabel]);
  }
  // Le mode d'emploi est écrit dans les deux langues : seule la sienne se
  // montre.
  for (const el of document.querySelectorAll("[data-help-lang]")) {
    el.hidden = el.dataset.helpLang !== lang;
  }
  // Le titre de l'onglet suit la langue, comme celui du bandeau ([data-i18n]).
  document.title = t.appTitle;
  renderDealActions();
  renderIaHint();
  renderSeatCompass();
  for (const opt of $("#dealer").options) {
    if (opt.value) opt.textContent = SEAT_LABEL[lang][opt.value];
  }
  for (const opt of $("#vul").options) {
    if (opt.value) opt.textContent = capitalize(VUL_LABEL[lang][opt.value]);
  }
  for (const opt of $("#deal-select").options) {
    opt.textContent = dealLabel(pbnGames[+opt.value], +opt.value);
  }
  syncTagSelects(); // les infobulles des deux sélecteurs en dépendent
}

// Loads a PBN text into the textarea and resets everything that depends on
// the previous deal (quiz in progress, displayed result, deal picker).
function loadPbn(text, fileName) {
  pushUndo();
  $("#pbn").value = text;
  // Quitter une donne chargée remet les sélecteurs sur « Aléatoire » : ce
  // qu'ils affichaient venait du fichier, ce n'était pas une préférence de
  // l'utilisateur, et la garder figerait toutes les donnes suivantes dessus.
  if (dealFromFile && !fileName) {
    $("#dealer").value = "";
    $("#vul").value = "";
  }
  dealFromFile = !!fileName;
  editRequested = false;
  resetQuiz();
  hideNewDealInQuizMode();
  hideResult();
  refreshDealSelector(true);
  setPbnOpen(false);
  $("#file-name").textContent = fileName || "";
  settlePbn();
}

$("#file").addEventListener("change", (ev) => {
  const f = ev.target.files[0];
  if (!f) return;
  f.text().then((t) => loadPbn(t, f.name));
});

// Nom de fichier par défaut, horodaté : deux donnes sauvées d'affilée ne se
// recouvrent pas dans le dossier de téléchargement.
function defaultPbnName(lang) {
  const d = new Date();
  const p = (n) => String(n).padStart(2, "0");
  const stamp = `${d.getFullYear()}${p(d.getMonth() + 1)}${p(d.getDate())}-${p(d.getHours())}${p(d.getMinutes())}`;
  return `${UI_TEXT[lang].fileSaveStem}-${stamp}.pbn`;
}

// Hands the PBN text back as a .pbn file. What is saved is the whole textarea
// — toutes ses donnes, tags compris — et non la seule donne sélectionnée : les
// zones y réécrivent les cartes à chaque déplacement, donc le fichier obtenu
// est bien ce que la page affiche. Un fichier chargé garde son nom.
$("#save-btn").addEventListener("click", () => {
  const lang = $("#lang").value;
  // #cons-error et non #error : celui-ci vit tout en bas du panneau, à côté
  // du questionnaire. Un message sur la donne se lit près de la donne.
  const errEl = $("#cons-error");
  const text = $("#pbn").value.trim();
  if (!text) {
    setError(errEl, CONS_TEXT[lang].errNoPbn);
    return;
  }
  setError(errEl, "");
  const url = URL.createObjectURL(new Blob([text + "\n"], { type: "text/plain" }));
  const a = document.createElement("a");
  a.href = url;
  a.download = $("#file-name").textContent.trim() || defaultPbnName(lang);
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(url);
});

// ---------- random deal generation ----------

const RANKS = "AKQJT98765432".split("");
const SUIT_CODES = ["S", "H", "D", "C"]; // PBN dot order: spades.hearts.diamonds.clubs
const RANK_ORDER = RANKS.join("");

// Fisher-Yates, in place.
function shuffle(arr) {
  for (let i = arr.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1));
    [arr[i], arr[j]] = [arr[j], arr[i]];
  }
  return arr;
}

function shuffledDeck() {
  const deck = [];
  for (const suit of SUIT_CODES) for (const rank of RANKS) deck.push({ suit, rank });
  return shuffle(deck);
}

function handString(cards) {
  const bySuit = { S: [], H: [], D: [], C: [] };
  for (const c of cards) bySuit[c.suit].push(c.rank);
  return SUIT_CODES.map((s) =>
    bySuit[s].sort((a, b) => RANK_ORDER.indexOf(a) - RANK_ORDER.indexOf(b)).join("")
  ).join(".");
}

// Time budget for the rejection sampling below: tight bounds can make matching
// deals rare, so we give up rather than freeze the page.
const RANDOM_DEAL_BUDGET_MS = 1500;

// Les deux tirages — une donne entière, ou les cartes qui manquent — sont des
// boucles qui redessinent jusqu'à tomber sur un résultat dans les bornes. Avec
// des bornes serrées elles vont au bout de leur budget, une seconde et demie
// pendant laquelle la page était entièrement figée : ni le bouton grisé, ni le
// moindre message ne pouvaient s'afficher, puisque rien n'était peint.
//
// Elles rendent maintenant la main au navigateur de temps en temps. Le pas est
// court — l'œil ne voit pas 40 ms — et il ne coûte qu'un tour de boucle
// d'événements toutes les quelques centaines d'essais.
const YIELD_EVERY_MS = 40;

async function breathe(since) {
  if (Date.now() - since < YIELD_EVERY_MS) return since;
  await new Promise((resolve) => setTimeout(resolve, 0));
  return Date.now();
}

// Sous ce seuil, l'attente ne se fait pas sentir : un message qui paraît et
// disparaît dans le même souffle n'est que du clignotement.
const BUSY_AFTER_MS = 200;

// Grise le bouton, et n'annonce l'attente que si elle dure. Rend une fonction
// à appeler quand c'est fini, dans un `finally`.
function showBusy(btn, message) {
  btn.disabled = true;
  const timer = setTimeout(() => setStatus($("#cons-busy"), message), BUSY_AFTER_MS);
  return () => {
    clearTimeout(timer);
    setStatus($("#cons-busy"), "");
    btn.disabled = false;
  };
}

// Valeurs du tag PBN [Vulnerable] que le serveur reconnaît (voir VulString).
const VULS = ["None", "NS", "EW", "All"];

function pickRandom(values) {
  return values[Math.floor(Math.random() * values.length)];
}

// Ce que l'utilisateur demande pour la donne à générer, ou null quand il ne
// demande rien — sélecteur sur « Aléatoire », ou verrouillé sur la donne d'un
// fichier, ce qui ne dit rien de la prochaine donne. La génération tire alors
// au hasard, à chaque donne.
function chosenDealer() {
  return (!dealFromFile && $("#dealer").value) || null;
}

function chosenVul() {
  return (!dealFromFile && $("#vul").value) || null;
}

// Builds a fresh, valid PBN deal for the given dealer and vulnerability: 52
// shuffled cards dealt 13 each, hands listed clockwise from the dealer as
// required by the PBN format. Deals are redrawn until every seat's honour-point
// count fits its bounds; returns null if none was found within the time budget.
async function randomPBN(dealer, vul) {
  const startIdx = SEATS.indexOf(dealer);
  const order = [0, 1, 2, 3].map((i) => SEATS[(startIdx + i) % 4]);
  const deadline = Date.now() + RANDOM_DEAL_BUDGET_MS;
  let souffle = Date.now();

  for (let attempt = 1; ; attempt++) {
    const deck = shuffledDeck();
    const hands = {};
    SEATS.forEach((seat, i) => { hands[seat] = deck.slice(i * 13, i * 13 + 13); });
    if (SEATS.every((seat) => isWithinBounds(cardsHCP(hands[seat]), hcpBounds[seat]))) {
      const dealStr = order.map((seat) => handString(hands[seat])).join(" ");
      return `[Dealer "${dealer}"]\n[Vulnerable "${vul}"]\n[Deal "${dealer}:${dealStr}"]`;
    }
    if (attempt % 256 === 0) {
      if (Date.now() > deadline) return null;
      souffle = await breathe(souffle);
    }
  }
}

// Tire une donne et la charge. Rend vrai une fois la donne en place : « Nouvelle
// donne », en fin de questionnaire, l'attend avant de relancer — sans cela, le
// questionnaire partait sur la donne précédente, lue avant l'arrivée de la
// nouvelle.
async function drawRandomDeal() {
  const lang = $("#lang").value;
  const errEl = $("#cons-error");
  setBoundsError(errEl, validateBounds(lang));
  if (errEl.textContent) return false;
  const fini = showBusy($("#random-btn"), CONS_TEXT[lang].busyDeal);
  try {
    const pbn = await randomPBN(chosenDealer() || pickRandom(SEATS), chosenVul() || pickRandom(VULS));
    if (!pbn) {
      setBoundsError(errEl, CONS_TEXT[lang].errNoDeal);
      return false;
    }
    loadPbn(pbn);
    return true;
  } finally {
    fini();
  }
}

$("#random-btn").addEventListener("click", drawRandomDeal);

$("#example-btn").addEventListener("click", () => {
  setError($("#cons-error"), "");
  loadPbn(EXAMPLE_PBN);
});

$("#dealer").addEventListener("change", () => {
  const seat = chosenDealer();
  if (!seat) return; // « Aléatoire » : le donneur sera tiré à la génération.
  setBlockDealer(seat);
  resetQuiz();
  hideResult();
});

$("#vul").addEventListener("change", () => {
  const vul = chosenVul();
  if (!vul) return; // Idem : tirée à la génération.
  setBlockVul(vul);
  resetQuiz();
  hideResult();
});

// ---------- multi-deal PBN files ----------

// A PBN file may hold several games, each separated by a blank line (PBN
// "semi-empty line" rule). Splits the raw text into individual game blocks.
function splitPbnGames(text) {
  const trimmed = text.trim();
  if (!trimmed) return [];
  const blocks = trimmed
    .split(/\r?\n[ \t]*\r?\n+/)
    .map((b) => b.trim())
    .filter((b) => /\[\s*Deal\b/i.test(b));
  return blocks.length ? blocks : [trimmed];
}

function dealLabel(block, idx) {
  const t = UI_TEXT[$("#lang").value];
  const dealer = block.match(/\[Dealer\s+"([^"]+)"\]/);
  const board = block.match(/\[Board\s+"([^"]+)"\]/);
  let label = `${t.dealWord} ${idx + 1}`;
  if (board) label += ` (${t.boardWord} ${board[1]})`;
  if (dealer) label += ` — ${t.dealerWord} ${dealer[1]}`;
  return label;
}

let pbnGames = [];
let selectedGameIdx = 0;

// Marks the currently selected game in the PBN textarea, so a multi-deal
// file shows at a glance which block is active — only meaningful once
// there's more than one deal to tell apart. Uses the browser's own text
// selection (styled via CSS) rather than editing the text, so the value
// sent to the server is never altered.
//
// `reveal` additionally focuses the textarea and scrolls to the block: a
// browser only paints a text selection when the field is focused, so without
// this the marker is invisible. It never opens the raw-text panel: a pick in
// the deal selector reveals the block only if the panel is already open, and
// opening it with #pbn-toggle-btn reveals the current block.
// Locates the selected block inside the textarea. Blocks are matched in
// order, so identical deals in one file still resolve to distinct ranges.
function selectedGameRange() {
  if (!pbnGames.length) return null;
  const text = $("#pbn").value;
  let searchFrom = 0;
  let start = -1;
  for (let i = 0; i <= selectedGameIdx; i++) {
    start = text.indexOf(pbnGames[i], searchFrom);
    if (start === -1) return null;
    searchFrom = start + pbnGames[i].length;
  }
  return { start, end: start + pbnGames[selectedGameIdx].length };
}

function highlightSelectedDeal(reveal) {
  const textarea = $("#pbn");
  if (pbnGames.length <= 1) {
    textarea.setSelectionRange(0, 0);
    return;
  }
  const range = selectedGameRange();
  if (!range) return;
  const text = textarea.value;
  const { start, end } = range;
  if (reveal && $("#pbn-details").hidden) reveal = false;
  if (reveal) textarea.focus({ preventScroll: true });
  textarea.setSelectionRange(start, end);
  if (reveal) {
    const lineIndex = (text.slice(0, start).match(/\n/g) || []).length;
    const lineHeight = parseFloat(getComputedStyle(textarea).lineHeight) || 18;
    textarea.scrollTop = Math.max(0, lineIndex * lineHeight - lineHeight);
  }
}

// Re-parses the PBN textarea and shows/hides the deal picker accordingly.
// `resetSelection` forces the picker back to the first game (used when a
// brand-new file/example is loaded rather than hand-edited).
function refreshDealSelector(resetSelection) {
  pbnGames = splitPbnGames($("#pbn").value);
  const row = $("#deal-select-row");
  const sel = $("#deal-select");
  if (pbnGames.length <= 1) {
    row.classList.add("hidden");
    sel.innerHTML = "";
    selectedGameIdx = 0;
    if (resetSelection) highlightSelectedDeal(false);
  } else {
    if (resetSelection || selectedGameIdx >= pbnGames.length) selectedGameIdx = 0;
    sel.innerHTML = pbnGames
      .map((b, i) => `<option value="${i}">${esc(dealLabel(b, i))}</option>`)
      .join("");
    sel.value = String(selectedGameIdx);
    row.classList.remove("hidden");
    if (resetSelection) highlightSelectedDeal(false);
  }
  syncTagSelects();
  syncZonesFromPbn();
}

// Remplace le bloc PBN sélectionné dans le textarea sans réveiller le
// gestionnaire `input`, qui prendrait la réécriture pour une saisie.
function replaceSelectedBlock(newBlock) {
  const range = selectedGameRange();
  if (!range) return;
  const textarea = $("#pbn");
  const text = textarea.value;
  writingPbn = true;
  textarea.value = text.slice(0, range.start) + newBlock + text.slice(range.end);
  writingPbn = false;
  pbnGames[selectedGameIdx] = newBlock;
  rememberCompleteDeal();
}

function tagRe(name) {
  return new RegExp(`\\[${name}\\s+"[^"]*"\\]`, "i");
}

// Écrit un tag dans le bloc PBN sélectionné, en l'ajoutant s'il manque : juste
// après `afterTag` quand ce dernier est présent, en tête du bloc sinon. Le tag
// [Deal] n'est jamais touché : le siège qu'il nomme en tête dit dans quel ordre
// les mains sont listées, pas qui donne.
function setBlockTag(name, value, afterTag) {
  const block = pbnGames[selectedGameIdx];
  if (!block) return;
  const line = `[${name} "${value}"]`;
  const own = tagRe(name);
  let newBlock;
  if (own.test(block)) {
    newBlock = block.replace(own, line);
  } else {
    const anchor = afterTag && block.match(tagRe(afterTag));
    newBlock = anchor
      ? block.replace(anchor[0], `${anchor[0]}\n${line}`)
      : `${line}\n${block}`;
  }
  if (newBlock === block) return;
  replaceSelectedBlock(newBlock);
  // Le libellé du sélecteur de donne cite le donneur : il doit suivre.
  const opt = $("#deal-select").options[selectedGameIdx];
  if (opt) opt.textContent = dealLabel(newBlock, selectedGameIdx);
}

function setBlockDealer(seat) {
  setBlockTag("Dealer", seat, null);
}

// L'ordre des tags est libre en PBN, mais [Vulnerable] se place après
// [Dealer] et avant [Deal] dans les exemples du format.
function setBlockVul(vul) {
  setBlockTag("Vulnerable", vul, "Dealer");
}

// Aligne les sélecteurs sur la donne courante, pour qu'une valeur affichée
// soit toujours celle de la donne. Une donne chargée depuis un fichier les
// impose et les verrouille : elle se lit telle qu'elle a été écrite.
//
// Hors fichier, « Aléatoire » n'affirme rien sur la donne : c'est l'absence de
// préférence, et rien ne l'en déloge — sans quoi la première donne générée
// hériterait du donneur de la donne préchargée.
//
// Le donneur retombe sur « Aléatoire » quand la donne ne déclare pas de tag
// [Dealer] ; une donne sans tag [Vulnerable] est en revanche une donne où
// personne n'est vulnérable, et c'est ce que le sélecteur affiche.
function syncTagSelects() {
  const block = pbnGames[selectedGameIdx] || "";
  if (dealFromFile || $("#dealer").value) {
    const dealer = block.match(/\[Dealer\s+"([NESW])"\]/i);
    $("#dealer").value = dealer ? dealer[1].toUpperCase() : "";
  }
  if (dealFromFile || $("#vul").value) {
    const vul = block.match(/\[Vulnerable\s+"([^"]*)"\]/i);
    $("#vul").value = vul ? normalizeVul(vul[1]) : "None";
  }
  const t = UI_TEXT[$("#lang").value];
  $("#dealer").title = dealFromFile ? t.lockedTitle : t.dealerTitle;
  $("#vul").title = dealFromFile ? t.lockedTitle : t.vulTitle;
  $("#dealer").disabled = dealFromFile;
  $("#vul").disabled = dealFromFile;
}

// Ramène les synonymes admis par le tag [Vulnerable] aux quatre valeurs du
// sélecteur (voir parseVulnerable côté serveur).
function normalizeVul(value) {
  switch (value.trim().toUpperCase()) {
    case "NS": return "NS";
    case "EW": return "EW";
    case "ALL": case "BOTH": return "All";
    default: return "None"; // "None", "Love", "-"…
  }
}

$("#deal-select").addEventListener("change", () => {
  selectedGameIdx = +$("#deal-select").value;
  resetQuiz();
  hideNewDealInQuizMode();
  hideResult();
  syncTagSelects();
  syncZonesFromPbn();
  highlightSelectedDeal(true);
});

// Hand-editing the PBN text wins over the cards laid out in the panel; our own
// write-backs (writeZonesToPbn) are skipped, or they would empty the neutral
// zone as soon as a card is moved.
$("#pbn").addEventListener("input", () => {
  if (writingPbn) return;
  refreshDealSelector(false);
});

// ---------- display helpers ----------

const SUIT_SYMBOLS = { spades: "♠", hearts: "♥", diamonds: "♦", clubs: "♣" };
// Suit keys in display order, matching the PBN dot order (♠.♥.♦.♣).
const SUIT_KEYS = ["spades", "hearts", "diamonds", "clubs"];
const RED_SUITS = new Set(["hearts", "diamonds"]);
// Strain letters used by the server, per language, mapped to symbols.
const STRAIN_MAP = {
  fr: { SA: "SA", P: "♠", C: "♥", K: "♦", T: "♣" },
  en: { NT: "NT", S: "♠", H: "♥", D: "♦", C: "♣" },
};
const SEAT_LABEL = { fr: { N: "Nord", E: "Est", S: "Sud", W: "Ouest" },
                     en: { N: "North", E: "East", S: "South", W: "West" } };
const SEAT_SHORT = { fr: { N: "N", E: "E", S: "S", W: "O" },
                     en: { N: "N", E: "E", S: "S", W: "W" } };
// Les quatre vulnérabilités, telles que le serveur les renvoie (VulString).
const VUL_LABEL = { fr: { None: "personne", NS: "N-S", EW: "E-O", All: "tous" },
                    en: { None: "none", NS: "NS", EW: "EW", All: "all" } };
// Strain letters used by the server, per language, ordered low → high rank.
const STRAIN_ORDER = { fr: ["T", "K", "C", "P", "SA"], en: ["C", "D", "H", "S", "NT"] };
const TEAM = { N: "NS", S: "NS", E: "EW", W: "EW" };
// Card rank letters as sent by the server (PBN notation), per display language.
const RANK_MAP = {
  fr: { A: "A", K: "R", Q: "D", J: "V", T: "10" },
  en: { A: "A", K: "K", Q: "Q", J: "J", T: "10" },
};

function rankHTML(rank, lang) {
  return esc(RANK_MAP[lang][rank] || rank);
}

function esc(s) {
  return String(s).replace(/[&<>"]/g, (c) =>
    ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" }[c]));
}

// Un intitulé suivi de son deux-points. Le français demande une espace avant,
// l'anglais n'en veut pas : les gabarits partagés écrivaient « Contract : 3NT »
// là où il faut « Contract: 3NT ». L'espace française est insécable, pour que
// le deux-points ne parte jamais seul à la ligne suivante.
function withColon(label, lang) {
  return lang === "fr" ? `${label}\u00a0:` : `${label}:`;
}

// « Vulnérabilité : N-S », à poser dans une puce ou au centre de la table.
function vulHTML(vul, lang) {
  const label = VUL_LABEL[lang][vul] || vul;
  return `${withColon(lang === "fr" ? "Vulnérabilité" : "Vulnerable", lang)} <b>${esc(label)}</b>`;
}

// Turns "4P" / "3NT" / "Passe" into HTML with suit symbols.
function bidHTML(bidText, lang) {
  const map = STRAIN_MAP[lang];
  const match = bidText.match(/^([1-7])(SA|NT|[A-Z]+)$/);
  if (!match) return esc(bidText); // Pass / X / XX / Contre...
  const sym = map[match[2]];
  if (!sym) return esc(bidText);
  if (sym === "♥" || sym === "♦") {
    return `${match[1]}<span class="red">${sym}</span>`;
  }
  return `${match[1]}${esc(sym)}`;
}

function isPass(bidText) {
  return bidText === "Pass" || bidText === "Passe";
}

// One line per suit, in ♠ ♥ ♦ ♣ order. `hand` may be partial/empty: missing
// suits show a dash.
function suitLinesHTML(hand, lang) {
  return SUIT_KEYS.map((suit) => {
    const sym = SUIT_SYMBOLS[suit];
    const cls = RED_SUITS.has(suit) ? "red" : "";
    const cards = hand && hand[suit]
      ? hand[suit].split("").map((r) => rankHTML(r, lang)).join(" ")
      : '<span class="void">—</span>';
    return `<div class="suitline"><span class="${cls}">${sym}</span> ${cards}</div>`;
  }).join("");
}

function handHTML(seat, hand, lang) {
  return `
    <div class="seat-name">
      <span>${esc(SEAT_LABEL[lang][seat])}</span>
      <span class="pts">${hand.h_points}H / ${hand.hl_points}HL · ${esc(hand.type)}</span>
    </div>
    ${suitLinesHTML(hand, lang)}`;
}

// ---------- honour-point bounds per hand ----------

// Bornes de PH (points d'honneur) saisies par l'utilisateur pour chaque main :
// elles contraignent la génération de donne aléatoire et signalent, sur la
// donne courante, les mains hors bornes.
const SEATS = ["N", "E", "S", "W"];
const HONOUR_POINTS = { A: 4, K: 3, Q: 2, J: 1 };
const TOTAL_HCP = 40;
// Maximum tenable dans une seule main : 4 As + 4 Rois + 4 Dames + 1 Valet.
const MAX_HAND_HCP = 37;

const CONS_TEXT = {
  fr: {
    ph: "PH", min: "Mini", max: "Maxi",
    reset: "Effacer les bornes",
    boundsToggle: "Bornes de points (PH mini/maxi)",
    undo: "Annuler la dernière action (Ctrl+Z)",
    undone: "Dernière action annulée.",
    neutral: "Cartes non affectées",
    // Le clic-puis-clic existe depuis toujours (voir le gestionnaire
    // pointerup) mais n'était annoncé nulle part : sur écran tactile, viser
    // une case vide au glissé est ingrat, et personne ne devinait le repli.
    neutralHint:
      "Glissez une carte d'une main à l'autre, ou ici pour la retirer. " +
      "Au clic : touchez une ou plusieurs cartes, puis leur destination. " +
      "Le symbole ♠ ♥ ♦ ♣ prend toute la couleur.",
    clear: "Retirer toutes les cartes",
    clearHand: "Retirer les cartes de cette main",
    photoHand: (seat) =>
      `Photographier la main ${/^[AEIOU]/.test(seat) ? "d'" : "de "}${seat}`,
    fill: "Compléter les mains",
    // Les noms courts des commandes de la table en édition, écrits sur les
    // boutons ; le nom complet reste dans leur infobulle.
    short: { undo: "Annuler", clear: "Tout retirer", fill: "Compléter", bounds: "Bornes", reset: "Effacer les bornes" },
    incomplete: (n) => n === 1 ? "Une main est incomplète." : `${n} mains sont incomplètes.`,
    errRange: (seat) => `${seat} : les bornes doivent être comprises entre 0 et ${MAX_HAND_HCP} PH.`,
    errOrder: (seat) => `${seat} : le mini dépasse le maxi.`,
    errSumMin: (lo) => `La somme des minis (${lo}) dépasse les ${TOTAL_HCP} PH du jeu.`,
    errSumMax: (hi) => `La somme des maxis (${hi}) n'atteint pas les ${TOTAL_HCP} PH du jeu.`,
    busyDeal: "Tirage en cours…",
    busyFill: "Distribution en cours…",
    errNoDeal: "Aucune donne trouvée avec ces bornes : élargissez-les.",
    errFull: (seat) => `${seat} a déjà 13 cartes.`,
    errNoFill: "Impossible de distribuer les cartes restantes avec ces bornes : élargissez-les.",
    errIncomplete: "Chaque main doit contenir 13 cartes : distribuez les cartes non affectées.",
    errNoPbn: "Saisissez ou chargez une donne PBN.",
    // Noms parlés des cartes : « R » et « ♠ » ne se lisent pas à voix haute.
    // Seules les honneurs ont un nom ; les chiffres se lisent d'eux-mêmes.
    rankName: { A: "As", K: "Roi", Q: "Dame", J: "Valet", T: "10" },
    suitName: { spades: "pique", hearts: "cœur", diamonds: "carreau", clubs: "trèfle" },
    cardLabel: (rank, suit, zone) => `${rank} de ${suit}, ${zone}`,
    emptyZoneLabel: (zone) => `${zone}, aucune carte`,
    // Annoncés dans la zone de politesse, à chaque geste au clavier.
    cardHeld: (card, n) => n > 1
      ? `${card} : pris, ${n} cartes sélectionnées. Allez à la destination, puis Entrée.`
      : `${card} : pris. Allez à la destination, puis Entrée.`,
    cardDropped: (card, zone) => `${card} : déposé dans ${zone}.`,
    cardsDropped: (n, zone) => `${n} cartes déposées dans ${zone}.`,
    cardReleased: (card) => `${card} : reposé.`,
    selectionCleared: "Sélection annulée.",
    // Le bandeau qui suit la sélection : combien de cartes, et quoi en faire.
    selCount: (n) => n === 1 ? "1 carte sélectionnée" : `${n} cartes sélectionnées`,
    selHint: "touchez une main pour l'y déposer",
    selHintMany: "touchez une main pour les y déposer",
    selClear: "Désélectionner",
    suitSelect: (suit, zone) => `Sélectionner les ${suit} (${zone})`,
    errRoom: (seat, room, n) =>
      `${seat} n'a de place que pour ${room} carte${room > 1 ? "s" : ""} : ${n} sont sélectionnées.`,
    cardsHelp:
      "Tableau des quatre mains. Tabulation pour passer d'une main à l'autre, " +
      "flèches gauche et droite pour parcourir ses cartes, haut et bas pour " +
      "changer de couleur. Entrée ou Espace prend une carte ; d'autres cartes " +
      "de la même main s'ajoutent à la sélection, Maj + Entrée prend toutes " +
      "celles qui séparent de la précédente, Ctrl + Entrée ajoute celle d'une " +
      "autre main. Entrée sur une autre main y dépose la sélection. Échap la repose.",
  },
  en: {
    ph: "HCP", min: "Min", max: "Max",
    reset: "Clear bounds",
    boundsToggle: "Point bounds (min/max HCP)",
    undo: "Undo the last action (Ctrl+Z)",
    undone: "Last action undone.",
    neutral: "Unassigned cards",
    neutralHint:
      "Drag a card from one hand to another, or here to take it out. " +
      "By click: tap one or more cards, then their destination. " +
      "The ♠ ♥ ♦ ♣ symbol takes the whole suit.",
    clear: "Take out every card",
    clearHand: "Take this hand's cards out",
    photoHand: (seat) => `Photograph ${seat}'s hand`,
    fill: "Fill the hands",
    short: { undo: "Undo", clear: "Take all out", fill: "Fill", bounds: "Bounds", reset: "Clear bounds" },
    incomplete: (n) => n === 1 ? "One hand is incomplete." : `${n} hands are incomplete.`,
    errRange: (seat) => `${seat}: bounds must be between 0 and ${MAX_HAND_HCP} HCP.`,
    errOrder: (seat) => `${seat}: min is greater than max.`,
    errSumMin: (lo) => `Minimums add up to ${lo}, more than the ${TOTAL_HCP} HCP in play.`,
    errSumMax: (hi) => `Maximums add up to ${hi}, less than the ${TOTAL_HCP} HCP in play.`,
    busyDeal: "Drawing a deal…",
    busyFill: "Dealing the cards…",
    errNoDeal: "No deal matches these bounds: widen them.",
    errFull: (seat) => `${seat} already holds 13 cards.`,
    errNoFill: "The remaining cards cannot be dealt within these bounds: widen them.",
    errIncomplete: "Every hand must hold 13 cards: deal the unassigned ones.",
    errNoPbn: "Type in or load a PBN deal.",
    rankName: { A: "Ace", K: "King", Q: "Queen", J: "Jack", T: "10" },
    suitName: { spades: "spades", hearts: "hearts", diamonds: "diamonds", clubs: "clubs" },
    cardLabel: (rank, suit, zone) => `${rank} of ${suit}, ${zone}`,
    emptyZoneLabel: (zone) => `${zone}, no cards`,
    cardHeld: (card, n) => n > 1
      ? `${card}: picked up, ${n} cards selected. Go to the destination, then press Enter.`
      : `${card}: picked up. Go to the destination, then press Enter.`,
    cardDropped: (card, zone) => `${card}: dropped in ${zone}.`,
    cardsDropped: (n, zone) => `${n} cards dropped in ${zone}.`,
    cardReleased: (card) => `${card}: put back.`,
    selectionCleared: "Selection cleared.",
    selCount: (n) => n === 1 ? "1 card selected" : `${n} cards selected`,
    selHint: "tap a hand to drop it there",
    selHintMany: "tap a hand to drop them there",
    selClear: "Deselect",
    suitSelect: (suit, zone) => `Select the ${suit} (${zone})`,
    errRoom: (seat, room, n) =>
      `${seat} only has room for ${room} card${room > 1 ? "s" : ""}: ${n} are selected.`,
    cardsHelp:
      "Table of the four hands. Tab moves between hands, left and right arrows " +
      "walk through a hand's cards, up and down change suit. Enter or Space " +
      "picks a card up; more cards of the same hand join the selection, " +
      "Shift+Enter takes every card back to the previous one, Ctrl+Enter adds " +
      "one from another hand. Enter on another hand drops the selection there. " +
      "Escape puts it back.",
  },
};

const hcpBounds = {};
for (const seat of SEATS) hcpBounds[seat] = { min: null, max: null };

// ---------- editable deal: four hands + a neutral zone ----------

// Les cartes de la donne courante, réparties entre les quatre mains et une
// zone « neutre » (UNASSIGNED) où atterrissent les cartes non affectées.
// C'est la source de vérité du glisser-déposer : chaque déplacement réécrit
// le tag [Deal] de la donne sélectionnée (voir writeZonesToPbn).
const UNASSIGNED = "X";
const ZONES = [...SEATS, UNASSIGNED];
const HAND_SIZE = 13;

let dealZones = emptyZones();
// Cartes désignées par un simple clic, en attente d'une zone de destination
// (voir le glisser-déposer plus bas). Elles peuvent venir de plusieurs mains :
// le dépôt les rassemble toutes dans la zone visée.
let selection = [];
// La dernière carte prise : l'autre bout d'une plage prise avec Maj.
let selAnchor = null;

function isSelected(card) {
  return selection.some((c) => sameCard(c, card));
}

function clearSelection() {
  selection = [];
  selAnchor = null;
}

// ---------- le clavier ----------
// Une carte par zone porte le tabindex ; c'est celle-ci. Mémorisée par zone,
// elle survit aux redessins, qui réécrivent tout l'innerHTML du tableau.
const rovingByZone = {};
// La carte à refocaliser juste après le prochain redessin. Sans elle, chaque
// geste renverrait le focus au début du document.
let refocusCard = null;

// La carte de `zone` qui doit porter le tabindex : celle retenue si elle y est
// encore, la première sinon.
function rovingCardOf(zone) {
  const list = zoneCardList(zone);
  if (!list.length) return null;
  const kept = rovingByZone[zone];
  if (kept && list.some((c) => c.suit === kept.suit && c.rank === kept.rank)) return kept;
  return list[0];
}

// Les cartes d'une zone, dans l'ordre où elles s'affichent : ♠ ♥ ♦ ♣, chaque
// couleur de l'as au 2. C'est l'ordre que suivent les flèches.
function zoneCardList(zone) {
  const hand = dealZones[zone];
  const out = [];
  for (const suit of SUIT_KEYS) {
    for (const rank of (hand[suit] || "").split("")) out.push({ zone, suit, rank });
  }
  return out;
}
// Vrai pendant que l'on réécrit nous-mêmes le textarea PBN : l'événement
// `input` doit alors être ignoré, sinon la zone neutre serait vidée.
let writingPbn = false;

function emptyHand() {
  const hand = {};
  for (const suit of SUIT_KEYS) hand[suit] = "";
  return hand;
}

function emptyZones() {
  const zones = {};
  for (const zone of ZONES) zones[zone] = emptyHand();
  return zones;
}

// Keeps a suit's ranks in A→2 order, dropping anything that isn't a rank.
function sortRanks(ranks) {
  return RANKS.filter((r) => ranks.includes(r)).join("");
}

function zoneCount(zone) {
  return handLength(dealZones[zone]);
}

// The zone's cards as a flat list, ♠ ♥ ♦ ♣ then A→2 within each suit.
function zoneCards(zone) {
  const cards = [];
  for (const suit of SUIT_KEYS) {
    for (const rank of dealZones[zone][suit]) cards.push({ suit, rank });
  }
  return cards;
}

function sameCard(a, b) {
  return !!a && !!b && a.zone === b.zone && a.suit === b.suit && a.rank === b.rank;
}

// Moves one card from its current zone to `to`. No-op if it isn't there.
function moveCard(card, to) {
  const from = dealZones[card.zone];
  if (!from || !from[card.suit].includes(card.rank)) return false;
  from[card.suit] = from[card.suit].replace(card.rank, "");
  dealZones[to][card.suit] = sortRanks(dealZones[to][card.suit] + card.rank);
  return true;
}

// Rebuilds the zones from the selected PBN block: whatever the [Deal] tag
// holds goes to the seats, and the neutral zone starts out empty.
function syncZonesFromPbn() {
  const hands = currentDealHands();
  dealZones = emptyZones();
  if (hands) {
    for (const seat of SEATS) {
      if (!hands[seat]) continue;
      for (const suit of SUIT_KEYS) {
        dealZones[seat][suit] = sortRanks(hands[seat][suit] || "");
      }
    }
  }
  clearSelection();
  renderBoundsCards();
  rememberCompleteDeal();
}

// Les cartes qu'aucune main ne tient vont au centre. Utile au démarrage : le
// [Deal] de la donne vide ne cite aucune carte, et syncZonesFromPbn laisse la
// zone neutre vide.
function gatherMissingCards() {
  for (const suit of SUIT_KEYS) {
    for (const rank of RANKS) {
      if (!ZONES.some((zone) => dealZones[zone][suit].includes(rank))) {
        dealZones[UNASSIGNED][suit] = sortRanks(dealZones[UNASSIGNED][suit] + rank);
      }
    }
  }
}

// Rewrites the [Deal] tag of the selected game from the current zones and
// pushes it back into the textarea, leaving the block's other tags (Dealer,
// Board…) alone. Unassigned cards are simply absent from the tag, so a
// partial deal stays a syntactically valid PBN block.
function writeZonesToPbn() {
  const block = pbnGames[selectedGameIdx];
  if (!block) return;
  const dealTag = block.match(/\[Deal\s+"([NESW]):[^"]*"\]/i);
  const dealer = block.match(/\[Dealer\s+"([NESW])"\]/i);
  const first = (dealTag ? dealTag[1] : dealer ? dealer[1] : "N").toUpperCase();
  const startIdx = SEATS.indexOf(first);
  const order = [0, 1, 2, 3].map((i) => SEATS[(startIdx + i) % 4]);
  const value = `${first}:${order
    .map((seat) => SUIT_KEYS.map((suit) => dealZones[seat][suit]).join("."))
    .join(" ")}`;
  const newBlock = dealTag
    ? block.replace(/\[Deal\s+"[^"]*"\]/i, `[Deal "${value}"]`)
    : `${block}\n[Deal "${value}"]`;
  replaceSelectedBlock(newBlock);
}

// ---------- annuler ----------
//
// Chaque action qui change la donne — carte déplacée, poubelle, table vidée ou
// complétée, photo, nouvelle donne tirée, exemple ou fichier — retient d'abord
// l'état d'avant. Le texte PBN suffit à le décrire : les cartes qu'il ne cite
// pas sont celles du centre (voir gatherMissingCards).
const UNDO_LIMIT = 50;
const undoStack = [];

function undoSnapshot() {
  return {
    pbn: $("#pbn").value,
    idx: selectedGameIdx,
    fromFile: dealFromFile,
    fileName: $("#file-name").textContent,
  };
}

function pushUndo() {
  const snap = undoSnapshot();
  const last = undoStack[undoStack.length - 1];
  // Deux instantanés identiques de suite n'annuleraient rien.
  if (last && last.pbn === snap.pbn && last.idx === snap.idx) return;
  undoStack.push(snap);
  if (undoStack.length > UNDO_LIMIT) undoStack.shift();
  renderUndoButton();
}

function renderUndoButton() {
  $("#cons-undo-btn").disabled = undoStack.length === 0;
}

function undo() {
  const snap = undoStack.pop();
  if (!snap) return;
  $("#pbn").value = snap.pbn;
  selectedGameIdx = snap.idx;
  dealFromFile = snap.fromFile;
  $("#file-name").textContent = snap.fileName;
  refreshDealSelector(false);
  gatherMissingCards();
  clearSelection();
  renderBoundsCards();
  resetQuiz();
  hideResult();
  setError($("#cons-error"), "");
  settlePbn();
  renderUndoButton();
  announce(CONS_TEXT[$("#lang").value].undone);
}

$("#cons-undo-btn").addEventListener("click", undo);

// Ctrl+Z (Cmd+Z sur Mac) hors des champs de saisie, qui gardent leur propre
// annuler — celui du texte PBN notamment. Rien quand la donne est masquée :
// on changerait une donne que l'on ne voit pas.
document.addEventListener("keydown", (ev) => {
  if (!(ev.ctrlKey || ev.metaKey) || ev.shiftKey || ev.altKey) return;
  if (ev.key !== "z" && ev.key !== "Z") return;
  const target = ev.target;
  if (target.closest && target.closest("input, textarea, select, [contenteditable]")) return;
  if (document.querySelector("dialog[open]")) return;
  if ($("#input-panel").classList.contains("quiz-running")) return;
  if (!undoStack.length) return;
  ev.preventDefault();
  undo();
});

// La saisie directe dans le texte PBN n'appelle ni commitZones ni loadPbn.
// L'état « posé » — celui qu'a laissé la dernière action, ou le démarrage — est
// donc tenu à jour à part, et empilé quand on quitte le champ après l'avoir
// modifié.
let settledPbn = null;

function settlePbn() {
  settledPbn = undoSnapshot();
}

$("#pbn").addEventListener("change", () => {
  if (!settledPbn || settledPbn.pbn === $("#pbn").value) return;
  undoStack.push(settledPbn);
  if (undoStack.length > UNDO_LIMIT) undoStack.shift();
  settlePbn();
  renderUndoButton();
});

// Applies a change made in the constraints panel: PBN text, display, and any
// result computed from the previous cards, which no longer describes the deal.
function commitZones() {
  // Le texte PBN décrit encore la donne d'avant ce changement : c'est lui que
  // l'on retient pour pouvoir l'annuler.
  pushUndo();
  writeZonesToPbn();
  settlePbn();
  renderBoundsCards();
  resetQuiz();
  hideResult();
}

function suitHCP(cards) {
  let n = 0;
  for (const rank of cards) n += HONOUR_POINTS[rank] || 0;
  return n;
}

function handHCP(hand) {
  return SUIT_KEYS.reduce((n, suit) => n + suitHCP(hand[suit] || ""), 0);
}

function handLength(hand) {
  return SUIT_KEYS.reduce((n, suit) => n + (hand[suit] || "").length, 0);
}

function cardsHCP(cards) {
  return cards.reduce((n, c) => n + (HONOUR_POINTS[c.rank] || 0), 0);
}

// Reads the [Deal "N:..."] tag of a PBN block into { N: {spades, ...}, ... }.
// Hands are listed clockwise from the seat named before the colon.
function parseDealHands(block) {
  const m = block && block.match(/\[Deal\s+"([NESW]):([^"]*)"\]/i);
  if (!m) return null;
  const startIdx = SEATS.indexOf(m[1].toUpperCase());
  const hands = {};
  m[2].trim().split(/\s+/).forEach((part, i) => {
    const suits = part.split(".");
    const hand = {};
    SUIT_KEYS.forEach((suit, s) => { hand[suit] = (suits[s] || "").toUpperCase(); });
    hands[SEATS[(startIdx + i) % 4]] = hand;
  });
  return hands;
}

function currentDealHands() {
  return parseDealHands(pbnGames[selectedGameIdx]);
}

function isWithinBounds(hcp, bounds) {
  if (bounds.min !== null && hcp < bounds.min) return false;
  if (bounds.max !== null && hcp > bounds.max) return false;
  return true;
}

function boundsSums() {
  let lo = 0;
  let hi = 0;
  for (const seat of SEATS) {
    lo += hcpBounds[seat].min ?? 0;
    hi += hcpBounds[seat].max ?? MAX_HAND_HCP;
  }
  return { lo, hi };
}

// Returns a user-facing message, or "" when the bounds are satisfiable.
function validateBounds(lang) {
  const t = CONS_TEXT[lang];
  for (const seat of SEATS) {
    const name = SEAT_LABEL[lang][seat];
    const { min, max } = hcpBounds[seat];
    for (const v of [min, max]) {
      if (v !== null && (v < 0 || v > MAX_HAND_HCP)) return t.errRange(name);
    }
    if (min !== null && max !== null && min > max) return t.errOrder(name);
  }
  const { lo, hi } = boundsSums();
  if (lo > TOTAL_HCP) return t.errSumMin(lo);
  if (hi < TOTAL_HCP) return t.errSumMax(hi);
  return "";
}

function boundsInputHTML(seat, kind, label) {
  const value = hcpBounds[seat][kind];
  return `
    <label class="cons-bound">
      <span>${esc(label)}</span>
      <input type="number" class="cons-input" inputmode="numeric" placeholder="–"
             min="0" max="${MAX_HAND_HCP}" step="1"
             data-seat="${seat}" data-bound="${kind}"
             value="${value === null ? "" : value}">
    </label>`;
}

// Nom parlé d'une carte : « Roi de pique, Nord ». Le glyphe affiché — « R »,
// « ♠ » — ne veut rien dire à voix haute.
function cardLabel(card, lang) {
  const t = CONS_TEXT[lang];
  const rank = t.rankName[card.rank] || card.rank;
  const zone = card.zone === UNASSIGNED ? t.neutral : SEAT_LABEL[lang][card.zone];
  return t.cardLabel(rank, t.suitName[card.suit], zone);
}

// One line per suit, in ♠ ♥ ♦ ♣ order, each rank its own draggable chip.
// Au clavier, une seule carte par zone porte le tabindex : la tabulation passe
// de main en main, les flèches parcourent la main. Cinquante-deux arrêts de
// tabulation seraient intraversables.
function zoneCardsHTML(zone, lang) {
  const hand = dealZones[zone];
  const roving = rovingCardOf(zone);
  return SUIT_KEYS.map((suit) => {
    const cls = RED_SUITS.has(suit) ? " red" : "";
    const ranks = hand[suit];
    const cards = ranks
      ? ranks
          .split("")
          .map((rank) => {
            const card = { zone, suit, rank };
            const held = isSelected(card);
            const sel = held ? " picked" : "";
            const tab = roving && roving.suit === suit && roving.rank === rank ? 0 : -1;
            return `<span class="card${cls}${sel}" data-owner="${zone}" data-suit="${suit}" ` +
              `data-rank="${rank}" role="button" tabindex="${tab}" ` +
              `aria-pressed="${held}" aria-label="${esc(cardLabel(card, lang))}">` +
              `${rankHTML(rank, lang)}</span>`;
          })
          .join("")
      : '<span class="void" aria-hidden="true">—</span>';
    // Le symbole prend toute la couleur de la zone d'un seul geste : au
    // pointeur seulement, le clavier a Maj + Entrée pour la même chose.
    const pick = ranks
      ? ` data-suit-pick="${suit}" data-owner="${zone}" title="${esc(suitPickLabel(zone, suit, lang))}"`
      : "";
    return `<div class="suitline"><span class="suitsym${cls}${ranks ? " suit-pick" : ""}"${pick} aria-hidden="true">${SUIT_SYMBOLS[suit]}</span>${cards}</div>`;
  }).join("");
}

function suitPickLabel(zone, suit, lang) {
  const t = CONS_TEXT[lang];
  const name = zone === UNASSIGNED ? t.neutral : SEAT_LABEL[lang][zone];
  return t.suitSelect(t.suitName[suit], name);
}

// Une zone vide n'a aucune carte à focaliser, et serait donc impossible à
// viser au clavier — or c'est précisément là qu'on veut déposer. Son corps
// devient alors la cible.
function emptyDropHTML(zone, lang) {
  const t = CONS_TEXT[lang];
  const name = zone === UNASSIGNED ? t.neutral : SEAT_LABEL[lang][zone];
  return `tabindex="0" role="button" aria-label="${esc(t.emptyZoneLabel(name))}"`;
}

// Les icônes des commandes de la donne. Toutes au même gabarit que celles des
// en-têtes de main : 24 unités, au trait, sans remplissage, la couleur venant
// du texte. Elles vont par paires de sens contraire — rassembler les cartes au
// centre, les redistribuer aux mains — pour que le geste se lise avant le mot.
const SVG_OPEN = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"
    stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">`;

// Quatre flèches qui rentrent : les cartes reviennent au centre.
const GATHER_SVG = `${SVG_OPEN}
  <path d="M9 3v6H3"/><path d="m3 3 6 6"/>
  <path d="M15 3v6h6"/><path d="m21 3-6 6"/>
  <path d="M9 21v-6H3"/><path d="m3 21 6-6"/>
  <path d="M15 21v-6h6"/><path d="m21 21-6-6"/>
</svg>`;

// Quatre flèches qui sortent : les cartes partent vers les mains.
const DEAL_SVG = `${SVG_OPEN}
  <path d="M3 9V3h6"/><path d="m3 3 6 6"/>
  <path d="M21 9V3h-6"/><path d="m21 3-6 6"/>
  <path d="M3 15v6h6"/><path d="m3 21 6-6"/>
  <path d="M21 15v6h-6"/><path d="m21 21-6-6"/>
</svg>`;

// Une gomme posée de biais : les bornes s'effacent.
const ERASER_SVG = `${SVG_OPEN}
  <path d="m5 16 7-7 6 6-4 4H8z"/>
  <path d="M12 9 16 5a2 2 0 0 1 3 0l3 3a2 2 0 0 1 0 3l-4 4"/>
  <path d="M4 21h16"/>
</svg>`;

// Une flèche qui revient sur elle-même : annuler.
const UNDO_SVG = `${SVG_OPEN}
  <path d="M9 14 4 9l5-5"/>
  <path d="M4 9h10.5a5.5 5.5 0 0 1 0 11H11"/>
</svg>`;

// Trois curseurs : les bornes de points de chaque main.
const BOUNDS_SVG = `${SVG_OPEN}
  <path d="M4 6h9"/><path d="M17 6h3"/><circle cx="15" cy="6" r="2"/>
  <path d="M4 12h3"/><path d="M11 12h9"/><circle cx="9" cy="12" r="2"/>
  <path d="M4 18h11"/><path d="M19 18h1"/><circle cx="17" cy="18" r="2"/>
</svg>`;

// Une flèche de lecture : la séquence se déroule.
const PLAY_SVG = `${SVG_OPEN}
  <circle cx="12" cy="12" r="9"/>
  <path d="m10 8.5 6 3.5-6 3.5z"/>
</svg>`;

// Une feuille qui entre : on charge un fichier.
const IMPORT_SVG = `${SVG_OPEN}
  <path d="M14 3H7a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V8z"/>
  <path d="M14 3v5h5"/>
  <path d="M12 18v-7"/><path d="m9 14 3-3 3 3"/>
</svg>`;

// Un dé : la donne est tirée au sort.
const DICE_SVG = `${SVG_OPEN}
  <rect x="3" y="3" width="18" height="18" rx="3"/>
  <path d="M8 8h.01"/><path d="M16 8h.01"/>
  <path d="M12 12h.01"/>
  <path d="M8 16h.01"/><path d="M16 16h.01"/>
</svg>`;

// Un livre ouvert : la donne exemple, celle du manuel.
const BOOK_SVG = `${SVG_OPEN}
  <path d="M2 4h6a4 4 0 0 1 4 4v13a3 3 0 0 0-3-3H2z"/>
  <path d="M22 4h-6a4 4 0 0 0-4 4v13a3 3 0 0 1 3-3h7z"/>
</svg>`;

// Deux chevrons : la donne sous sa forme de texte (PBN).
const CODE_SVG = `${SVG_OPEN}
  <path d="m8 7-5 5 5 5"/><path d="m16 7 5 5-5 5"/><path d="m14 4-4 16"/>
</svg>`;

// Un crayon : modifier la donne.
const PENCIL_SVG = `${SVG_OPEN}
  <path d="M12 20h9"/>
  <path d="M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4z"/>
</svg>`;

// Une croix : fermer, replier.
const CLOSE_SVG = `${SVG_OPEN}
  <path d="M18 6 6 18"/><path d="m6 6 12 12"/>
</svg>`;

// Une flèche vers le bas : le bouton ouvre un menu.
const CHEVRON_SVG = `${SVG_OPEN}
  <path d="m6 9 6 6 6-6"/>
</svg>`;

// Une boîte d'où sort une flèche : partager la donne hors de la page.
const SHARE_SVG = `${SVG_OPEN}
  <path d="M4 12v7a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-7"/>
  <path d="m16 6-4-4-4 4"/><path d="M12 2v13"/>
</svg>`;

// Deux maillons : le lien de partage.
const LINK_SVG = `${SVG_OPEN}
  <path d="M10 13a5 5 0 0 0 7.5.5l3-3a5 5 0 0 0-7-7l-1.7 1.7"/>
  <path d="M14 11a5 5 0 0 0-7.5-.5l-3 3a5 5 0 0 0 7 7l1.7-1.7"/>
</svg>`;

// Une imprimante : imprimer le résultat.
const PRINT_SVG = `${SVG_OPEN}
  <path d="M6 9V3h12v6"/>
  <rect x="3" y="9" width="18" height="8" rx="2"/>
  <path d="M6 14h12v7H6z"/>
</svg>`;

// Deux feuilles décalées : copier. Puis la coche qui la remplace un instant,
// une fois la copie faite.
const COPY_SVG = `${SVG_OPEN}
  <rect x="9" y="9" width="12" height="12" rx="2"/>
  <path d="M5 15H4a1 1 0 0 1-1-1V4a1 1 0 0 1 1-1h10a1 1 0 0 1 1 1v1"/>
</svg>`;
const CHECK_SVG = `${SVG_OPEN}
  <path d="M5 12.5 10 17 19 7"/>
</svg>`;

// Une flèche vers le bas au-dessus d'un plateau : le fichier est téléchargé.
const EXPORT_SVG = `${SVG_OPEN}
  <path d="M12 3v11"/><path d="m8 10 4 4 4-4"/>
  <path d="M4 17v2a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-2"/>
</svg>`;

// Une toque d'étudiant : on s'entraîne. Un point d'interrogation disait bien
// « questionnaire », mais c'est le dessin de l'aide partout ailleurs, et rien
// ne le distinguait d'un bouton « au secours ».
const QUIZ_SVG = `${SVG_OPEN}
  <path d="M2 9.2 12 4.4l10 4.8-10 4.8z"/>
  <path d="M6.5 11.4V16c0 1.4 2.5 2.6 5.5 2.6s5.5-1.2 5.5-2.6v-4.6"/>
  <path d="M21.4 9.5v4.6"/>
</svg>`;

// Dessins des deux icônes des en-têtes de main, et du bouton photo de la
// donne : une poubelle et un appareil photo, repris de l'application
// bridgeteacher.
const TRASH_SVG = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"
    stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
  <polyline points="3 6 5 6 21 6"/>
  <path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6"/>
  <path d="M10 11v6"/><path d="M14 11v6"/>
  <path d="M9 6V4a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v2"/>
</svg>`;
const CAMERA_SVG = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"
    stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
  <path d="M23 19a2 2 0 0 1-2 2H3a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h4l2-3h6l2 3h4a2 2 0 0 1 2 2z"/>
  <circle cx="12" cy="13" r="4"/>
</svg>`;

// Poubelle — seulement s'il y a des cartes à retirer — et appareil photo, dans
// l'en-tête d'une main. L'état du bouton photo est posé après coup, en un seul
// endroit (voir renderPhotoButtons).
function handActionsHTML(seat, lang) {
  const t = CONS_TEXT[lang];
  const trash = handLength(dealZones[seat])
    ? `<button type="button" class="cons-btn" data-clear-hand="${seat}"
         title="${esc(t.clearHand)}" aria-label="${esc(t.clearHand)}">${TRASH_SVG}</button>`
    : "";
  // Le bouton appareil photo n'existe que si l'option « Serveur IA » est active.
  const camera = iaEnabled()
    ? `<button type="button" class="cons-btn" data-photo="${seat}">${CAMERA_SVG}</button>`
    : "";
  return `${trash}${camera}`;
}

function boundsCardHTML(seat, lang) {
  const t = CONS_TEXT[lang];
  const hand = dealZones[seat];
  const hcp = handHCP(hand);
  const len = handLength(hand);
  const off = len > 0 && !isWithinBounds(hcp, hcpBounds[seat]);
  return `
    <div class="cons-head">
      <span class="cons-seat">${esc(SEAT_LABEL[lang][seat])}</span>
      <span class="cons-chips">
        ${handActionsHTML(seat, lang)}
        <span class="cons-chip${off ? " off" : ""}">${hcp} ${esc(t.ph)}</span>
        <span class="cons-chip">${len}/${HAND_SIZE}</span>
      </span>
    </div>
    <div class="cons-bounds">
      <span class="cons-ph">${esc(t.ph)}</span>
      ${boundsInputHTML(seat, "min", t.min)}
      ${boundsInputHTML(seat, "max", t.max)}
    </div>
    <div class="cons-cards"${handLength(hand) ? "" : " " + emptyDropHTML(seat, lang)}>${zoneCardsHTML(seat, lang)}</div>`;
}

// The centre of the table holds the cards that belong to no hand yet. Son
// en-tête porte la photo des quatre mains, qui refait la donne entière — à
// côté des cartes qui attendent leur main, là où la lecture les verse.
function neutralZoneHTML(lang) {
  const t = CONS_TEXT[lang];
  const count = zoneCount(UNASSIGNED);
  const camera = iaEnabled()
    ? `<button type="button" class="cons-btn" data-photo="deal">${CAMERA_SVG}</button>`
    : "";
  const body = count
    ? `<div class="cons-cards">${zoneCardsHTML(UNASSIGNED, lang)}</div>`
    : `<div class="neutral-hint" ${emptyDropHTML(UNASSIGNED, lang)}>${esc(t.neutralHint)}</div>`;
  return `
    <div class="cons-head">
      <span class="cons-seat">${esc(t.neutral)}</span>
      <span class="cons-chips">${camera}<span class="cons-chip">${count}</span></span>
    </div>
    ${body}`;
}

// Full redraw of the four bound cards + the neutral zone. Rebuilds the inputs,
// so only call it when the deal, the language or the bounds state changes as a
// whole — not on every keystroke (see refreshBoundsFlags).
// Un bouton sans texte : le dessin dedans, le nom dans l'infobulle et dans
// l'étiquette que lit un lecteur d'écran. Les trois viennent ensemble, sinon
// l'un des trois finit par manquer.
//
// Avec `labeled`, le nom s'écrit à côté du dessin : il se lit alors sans
// survol, et ni l'infobulle ni l'étiquette n'ont plus rien à ajouter.
function setCommandButton(sel, svg, name, labeled) {
  const btn = $(sel);
  if (labeled) {
    btn.classList.add("cmd-labeled");
    btn.innerHTML = `${svg}<span class="cmd-label">${esc(name)}</span>`;
    btn.removeAttribute("aria-label");
    delete btn.dataset.tip;
    return;
  }
  btn.innerHTML = svg;
  btn.setAttribute("aria-label", name);
  btn.dataset.tip = name;
}

// Les trois commandes du haut du panneau : charger, tirer, sauver. Même
// traitement que celles du bas — un dessin, un nom parlé, une infobulle.
// « Charger un fichier » est un <label> et non un <button> : son texte reste
// dans l'arbre, masqué à l'œil, car c'est lui qui nomme le champ fichier
// qu'il enveloppe.
function renderDealActions() {
  const lang = $("#lang").value;
  const t = UI_TEXT[lang];
  setCommandButton("#random-btn", DICE_SVG, t.randomDeal, true);
  setCommandButton("#new-deal-more-btn", CHEVRON_SVG, t.moreDeals);
  setCommandButton("#share-menu-btn", SHARE_SVG, t.shareMenu, true);
  setCommandButton("#example-btn", BOOK_SVG, t.exampleDeal, true);
  setCommandButton("#save-btn", EXPORT_SVG, t.fileSave, true);
  setCommandButton("#pbn-toggle-btn", CODE_SVG, t.pbnToggle, true);
  setCommandButton("#pbn-copy-btn", COPY_SVG, t.pbnCopy);
  setCommandButton("#pbn-close-btn", CLOSE_SVG, t.pbnClose);
  setCommandButton("#share-btn", LINK_SVG, t.shareLink, true);
  setCommandButton("#print-btn", PRINT_SVG, t.printResult);
}

// Place un texte dans le presse-papiers. Le presse-papiers moderne exige un
// contexte sûr (https ou localhost) et peut être refusé : l'ancienne commande
// de copie prend alors le relais, sur un champ temporaire hors de la vue.
async function copyToClipboard(text) {
  try {
    await navigator.clipboard.writeText(text);
    return true;
  } catch (err) {
    const tmp = document.createElement("textarea");
    tmp.value = text;
    tmp.setAttribute("readonly", "");
    tmp.style.position = "fixed";
    tmp.style.opacity = "0";
    document.body.appendChild(tmp);
    tmp.select();
    let ok = false;
    try {
      ok = document.execCommand("copy");
    } catch (err2) {
      ok = false;
    }
    tmp.remove();
    return ok;
  }
}

// Le retour d'un bouton de copie : une coche 1,5 s et l'annonce en cas de
// réussite, le message d'échec dans l'infobulle et en alerte sinon.
function flashCopied(btn, ok, okMsg, failMsg) {
  if (!ok) {
    btn.dataset.tip = failMsg;
    announceAlert(failMsg);
    return;
  }
  announce(okMsg);
  // Un article de menu disparaît avec son menu : c'est le bouton qui l'a
  // ouvert qui montre la coche, avec le message à côté.
  const trigger = btn.closest(".menu")?.querySelector(".menu-trigger");
  if (trigger) btn = trigger;
  btn.innerHTML = btn.classList.contains("cmd-labeled")
    ? `${CHECK_SVG}<span class="cmd-label">${esc(okMsg)}</span>`
    : CHECK_SVG;
  btn.dataset.tip = okMsg;
  btn.classList.add("copied");
  clearTimeout(btn.copyTimer);
  btn.copyTimer = setTimeout(() => {
    btn.classList.remove("copied");
    renderDealActions();
  }, 1500);
}

// Copie tout le texte PBN.
async function copyPbn() {
  const t = UI_TEXT[$("#lang").value];
  const ok = await copyToClipboard($("#pbn").value);
  flashCopied($("#pbn-copy-btn"), ok, t.pbnCopied, t.pbnCopyFailed);
}

// ---------- partage ----------
//
// Le lien porte la donne sélectionnée dans son fragment (#pbn=…) : le fragment
// n'est jamais envoyé au serveur, et le lien marche aussi bien sur GitHub Pages
// que dans un sous-répertoire ou en local.
const SHARE_PREFIX = "#pbn=";

function shareURL() {
  const block = pbnGames[selectedGameIdx] || $("#pbn").value.trim();
  return location.href.split("#")[0] + SHARE_PREFIX + encodeURIComponent(block);
}

$("#share-btn").addEventListener("click", async () => {
  const t = UI_TEXT[$("#lang").value];
  const ok = await copyToClipboard(shareURL());
  flashCopied($("#share-btn"), ok, t.shareCopied, t.shareFailed);
});

// La donne d'un lien de partage, ou null si l'adresse n'en porte pas — ou en
// porte une illisible, qui est alors ignorée plutôt que d'effacer la donne.
function sharedPbnFromHash() {
  if (!location.hash.startsWith(SHARE_PREFIX)) return null;
  let text;
  try {
    text = decodeURIComponent(location.hash.slice(SHARE_PREFIX.length));
  } catch (err) {
    return null;
  }
  return parseDealHands(text) ? text : null;
}

// Une fois lue, la donne quitte l'adresse : un rechargement ne doit pas écraser
// ce que l'on a fait depuis, et l'adresse affichée redevient celle de la page.
function clearShareHash() {
  history.replaceState(null, "", location.href.split("#")[0]);
}

// Un lien collé dans l'onglet déjà ouvert.
window.addEventListener("hashchange", () => {
  const shared = sharedPbnFromHash();
  if (!shared) return;
  loadPbn(shared);
  clearShareHash();
  revealPane($("#input-panel"));
});

// ---------- impression ----------
//
// Le Résultat s'imprime seul (voir @media print dans style.css), et toujours
// en thème clair : un fond sombre coûterait de l'encre pour rien.
$("#print-btn").addEventListener("click", () => window.print());
window.addEventListener("beforeprint", () => {
  document.documentElement.dataset.theme = "light";
  // Une enchère survolée au moment d'imprimer resterait éclairée sur papier.
  linkCall(null);
});
window.addEventListener("afterprint", applyTheme);

$("#pbn-copy-btn").addEventListener("click", copyPbn);
// Le texte disparaît sous le focus : il revient au menu qui l'a ouvert.
$("#pbn-close-btn").addEventListener("click", () => {
  setPbnOpen(false);
  $("#share-menu-btn").focus();
});

// Affiche ou masque le texte PBN sous la rangée du haut. Le bouton dit son
// état par aria-expanded, que style.css rend aussi visible (bouton enfoncé).
function setPbnOpen(open) {
  $("#pbn-details").hidden = !open;
  $("#pbn-toggle-btn").setAttribute("aria-expanded", String(open));
}

// À l'ouverture, un fichier de plusieurs donnes montre celle qui est choisie
// dans « Donne à utiliser » : c'est le seul moment où le texte apparaît pour
// elle, le sélecteur ne l'ouvrant plus de lui-même.
$("#pbn-toggle-btn").addEventListener("click", () => {
  const open = $("#pbn-details").hidden;
  setPbnOpen(open);
  if (open) highlightSelectedDeal(true);
});

function renderBoundsCards() {
  const lang = $("#lang").value;
  const t = CONS_TEXT[lang];
  // En édition, les commandes portent leur nom : c'est là qu'on les cherche.
  // Le nom complet, raccourci clavier compris, reste dans l'infobulle.
  for (const [sel, svg, key, long] of [
    ["#cons-undo-btn", UNDO_SVG, "undo", t.undo],
    ["#cons-clear-btn", GATHER_SVG, "clear", t.clear],
    ["#cons-fill-btn", DEAL_SVG, "fill", t.fill],
    ["#cons-bounds-btn", BOUNDS_SVG, "bounds", t.boundsToggle],
    ["#cons-reset-btn", ERASER_SVG, "reset", t.reset],
  ]) {
    setCommandButton(sel, svg, t.short[key], true);
    $(sel).title = long;
  }
  setCommandButton("#bid-btn", PLAY_SVG, UI_TEXT[lang].runAuction, true);
  setCommandButton("#quiz-btn", QUIZ_SVG, UI_TEXT[lang].startQuiz);
  // Posé ici et non par [data-i18n] : applyLang ne lit que UI_TEXT, et ce
  // texte appartient au panneau des contraintes, donc à CONS_TEXT.
  $("#cards-help").textContent = t.cardsHelp;
  // Désactivé plutôt que masqué : un bouton qui disparaît décale la rangée à
  // chaque carte déplacée, et l'on ne peut pas apprendre ce qu'il fait tant
  // qu'on ne l'a jamais vu. Éteint, il reste là et se laisse interroger.
  const fillBtn = $("#cons-fill-btn");
  fillBtn.classList.remove("hidden");
  fillBtn.disabled = zoneCount(UNASSIGNED) === 0;
  for (const seat of SEATS) {
    const el = $("#cons-" + seat);
    el.dataset.zone = seat;
    el.innerHTML = boundsCardHTML(seat, lang);
  }
  const center = $("#cons-center");
  center.dataset.zone = UNASSIGNED;
  center.classList.toggle("filled", zoneCount(UNASSIGNED) > 0);
  center.innerHTML = neutralZoneHTML(lang);
  renderPhotoButtons(); // les boutons des mains viennent d'être recréés
  renderSelectionBar();
  restoreCardFocus();
  renderBidReady();
  renderDealMode();
  renderReadTable();
}

// ---------- lecture et édition ----------
//
// La table se lit par défaut : quatre mains compactes autour du tapis, comme
// au résultat. On passe en édition pour déplacer des cartes ou poser des
// bornes, et la table y bascule d'elle-même tant que la donne est incomplète
// — il n'y a rien à lire sur une donne à moitié distribuée. Une nouvelle donne
// (tirage, exemple, fichier, lien) ramène à la lecture.
let editRequested = false;

function dealComplete() {
  return zoneCount(UNASSIGNED) === 0 && SEATS.every((seat) => zoneCount(seat) === HAND_SIZE);
}

function dealEditing() {
  return editRequested || !dealComplete();
}

function renderDealMode() {
  const t = UI_TEXT[$("#lang").value];
  const editing = dealEditing();
  $("#input-panel").classList.toggle("editing", editing);
  const btn = $("#edit-btn");
  btn.setAttribute("aria-pressed", String(editing));
  setCommandButton("#edit-btn", editing ? CHECK_SVG : PENCIL_SVG,
    editing ? t.editDone : t.editDeal, true);
  // Terminer sur une donne incomplète ne mènerait nulle part : le bouton dit
  // pourquoi il est éteint.
  // Donne masquée (questionnaire) : la modifier la dévoilerait.
  const hidden = $("#input-panel").classList.contains("quiz-running");
  btn.disabled = hidden || (editing && !dealComplete());
  btn.title = btn.disabled && !hidden ? t.editDoneBlocked : "";
}

$("#edit-btn").addEventListener("click", () => {
  editRequested = !dealEditing();
  renderDealMode();
});

// Le centre de la table sans résultat : donneur et vulnérabilité de la donne.
function readCenterHTML(lang) {
  const block = pbnGames[selectedGameIdx] || "";
  const dealer = (block.match(/\[Dealer\s+"([NESW])"\]/i) || [])[1];
  const vul = block.match(/\[Vulnerable\s+"([^"]*)"\]/i);
  const t = UI_TEXT[lang];
  return `
    ${dealer ? `<div>${withColon(t.dealer, lang)} <b>${esc(SEAT_SHORT[lang][dealer.toUpperCase()])}</b></div>` : ""}
    <div class="vul-line">${vulHTML(vul ? normalizeVul(vul[1]) : "None", lang)}</div>`;
}

// Remplit la table en lecture depuis la donne. Un résultat affiché l'a déjà
// remplie avec l'analyse du moteur (points H/HL, type de main, contrat) : on
// ne l'écrase pas.
function renderReadTable() {
  if (!$("#result-panel").classList.contains("hidden")) return;
  const lang = $("#lang").value;
  const hands = currentDealHands();
  const ph = CONS_TEXT[lang].ph;
  for (const seat of SEATS) {
    const hand = hands && hands[seat];
    $("#hand-" + seat).innerHTML = `
      <div class="seat-name">
        <span>${esc(SEAT_LABEL[lang][seat])}</span>
        <span class="pts">${hand ? handHCP(hand) : 0} ${esc(ph)}</span>
      </div>
      ${suitLinesHTML(hand, lang)}`;
  }
  $("#table-center").innerHTML = readCenterHTML(lang);
}

// « Afficher les enchères » ne s'allume que sur une donne complète. Éteint, il
// dit pourquoi à côté de lui, avec le remède à portée de clic : le bouton
// allumé sur une donne incomplète ne répondait que par un refus.
function renderBidReady() {
  const t = CONS_TEXT[$("#lang").value];
  const incomplete = SEATS.filter((seat) => zoneCount(seat) !== HAND_SIZE).length;
  const ready = !!pbnGames[selectedGameIdx] && incomplete === 0 && zoneCount(UNASSIGNED) === 0;
  $("#bid-btn").disabled = !ready;
  $("#bid-hint").classList.toggle("hidden", ready);
  $("#bid-hint-text").textContent = ready ? "" : t.incomplete(Math.max(incomplete, 1));
  const fill = $("#bid-fill-btn");
  fill.textContent = t.fill;
  fill.classList.toggle("hidden", zoneCount(UNASSIGNED) === 0);
  if (ready) scheduleAutoBid();
}

$("#bid-fill-btn").addEventListener("click", () => $("#cons-fill-btn").click());

// Le moteur tourne dans la page et répond sans délai : une donne complète qui
// change voit ses enchères recalculées d'elles-mêmes, sans attendre le
// bouton. Jamais pendant un questionnaire ni sur une donne masquée — ce
// serait la dévoiler —, ni avant que le moteur ne soit prêt, ni par-dessus un
// message que l'utilisateur doit lire.
let autoBidTimer = null;

function autoBidAllowed() {
  return healthState === "online" && !quiz && dealComplete() &&
    !!pbnGames[selectedGameIdx] &&
    $("#result-panel").classList.contains("hidden") &&
    !$("#input-panel").classList.contains("quiz-running") &&
    !$("#cons-error").textContent;
}

function scheduleAutoBid() {
  clearTimeout(autoBidTimer);
  autoBidTimer = setTimeout(() => {
    autoBidTimer = null;
    if (autoBidAllowed()) simulate({ auto: true });
  }, 150);
}

// Le redessin ci-dessus remplace tout le tableau : l'élément qui avait le
// focus n'existe plus. Sans cette reprise, chaque carte déplacée au clavier
// renverrait le focus en tête de document, et le suivi serait impraticable.
// On ne reprend le focus que s'il était déjà dans le tableau, pour ne pas
// l'arracher à qui tape dans un champ de bornes.
function restoreCardFocus() {
  const want = refocusCard;
  refocusCard = null;
  if (!want) return;
  const el = cardElement(want);
  if (!el) return;
  el.tabIndex = 0;
  el.focus({ preventScroll: true });
}

// Cheap update used while typing: refreshes the out-of-bounds highlight
// without touching the inputs the user is editing.
function refreshBoundsFlags() {
  for (const seat of SEATS) {
    const chip = $(`#cons-${seat} .cons-chip`);
    if (!chip) continue;
    const hand = dealZones[seat];
    chip.classList.toggle(
      "off",
      handLength(hand) > 0 && !isWithinBounds(handHCP(hand), hcpBounds[seat])
    );
  }
}

$("#constraints").addEventListener("input", (ev) => {
  const input = ev.target.closest(".cons-input");
  if (!input) return;
  const raw = input.value.trim();
  const value = raw === "" ? null : Number.parseInt(raw, 10);
  hcpBounds[input.dataset.seat][input.dataset.bound] =
    value === null || Number.isNaN(value) ? null : value;
  setError($("#cons-error"), validateBounds($("#lang").value));
  refreshBoundsFlags();
});

// Les champs Mini/Maxi des quatre mains, repliés par défaut : ils prenaient une
// ligne dans chaque main pour une fonction que la plupart n'utilisent pas. Un
// message d'erreur sur les bornes les rouvre, pour qu'on voie ce qu'il vise.
function setBoundsOpen(open) {
  $("#constraints").classList.toggle("show-bounds", open);
  $("#cons-bounds-btn").setAttribute("aria-expanded", String(open));
}

$("#cons-bounds-btn").addEventListener("click", () => {
  setBoundsOpen(!$("#constraints").classList.contains("show-bounds"));
});

// setError, et rouvre les bornes quand le message les concerne.
function setBoundsError(errEl, msg) {
  setError(errEl, msg);
  if (!msg) return;
  setBoundsOpen(true);
  // Les bornes ne se voient qu'en édition : on y passe pour les montrer.
  editRequested = true;
  renderDealMode();
}

$("#cons-reset-btn").addEventListener("click", () => {
  for (const seat of SEATS) hcpBounds[seat] = { min: null, max: null };
  setError($("#cons-error"), "");
  renderBoundsCards();
});

// ---------- moving cards between hands and the neutral zone ----------

// Distance (px) à parcourir avant qu'un appui devienne un glisser : en deçà,
// on considère que l'utilisateur a simplement cliqué la carte.
const DRAG_THRESHOLD = 5;

let dragState = null; // { card, cards, el, x0, y0, moved, pointerId, ghost }

function cardFromEl(el) {
  return { zone: el.dataset.owner, suit: el.dataset.suit, rank: el.dataset.rank };
}

function zoneUnder(x, y) {
  const el = document.elementFromPoint(x, y);
  const zoneEl = el && el.closest("[data-zone]");
  return zoneEl ? zoneEl.dataset.zone : null;
}

function clearDropHints() {
  for (const el of document.querySelectorAll("[data-zone]")) {
    el.classList.remove("drop-ok", "drop-no");
  }
}

// Whether `zone` can take every card of `cards` that isn't already there: the
// neutral zone is unbounded, a hand stops at 13.
function zoneAccepts(zone, cards) {
  return dropPlan(cards, zone).fits;
}

// ---------- ce qui est dit à voix haute ----------
//
// Les messages de l'application sont écrits dans des éléments que la feuille
// masque tant qu'ils sont vides, ou que le JS masque par `.hidden`. Or un
// élément en display:none est hors de l'arbre d'accessibilité : la zone
// vivante n'y est pas enregistrée, et le message arrive sans être annoncé.
// Les deux zones ci-dessous, elles, ne sont jamais masquées — chaque message
// y est recopié. Voir leur commentaire dans index.html.

function writeLive(id, msg) {
  const el = $(id);
  // Vidée puis remplie : deux messages identiques d'affilée — deux cartes du
  // même rang déposées au même endroit — ne seraient pas relus autrement.
  el.textContent = "";
  el.textContent = msg;
}

// Ce qui avance : une carte déposée, un calcul en cours. N'interrompt pas.
function announce(msg) {
  writeLive("#live-polite", msg);
}

// Ce qui bloque. Interrompt, à juste titre : il faut l'avoir lu pour
// continuer.
function announceAlert(msg) {
  writeLive("#live-alert", msg);
}

// Écrit un message d'erreur là où il s'affiche, et le fait annoncer.
//
// Un message inchangé n'est pas réannoncé : les bornes sont revalidées à
// chaque carte déplacée, et la même phrase serait répétée à chaque geste.
function setError(el, msg) {
  msg = msg || "";
  const changed = el.textContent !== msg;
  el.textContent = msg;
  if (msg && changed) announceAlert(msg);
}

// Même chose pour ce qui n'est pas une erreur, sur le ton poli.
function setStatus(el, msg) {
  msg = msg || "";
  const changed = el.textContent !== msg;
  el.textContent = msg;
  if (msg && changed) announce(msg);
}

// par.js est un module à part et n'a pas accès à ce qui précède, comme il
// n'avait pas accès à $ ni à UI_TEXT (voir window.parSetDeal).
window.a11ySetError = setError;
window.a11ySetStatus = setStatus;

// Prendre une carte, ou déposer la sélection sur la main de celle-ci.
// C'est ce que fait un clic simple, et c'est ce que fait Entrée : le geste est
// écrit une fois pour les deux, sans quoi les deux finiraient par diverger.

// Les cartes que `zone` recevrait (celles qui y sont déjà ne bougent pas), et
// la place qu'elle a pour elles.
function dropPlan(cards, zone) {
  const moving = cards.filter((c) => c.zone !== zone);
  const room = zone === UNASSIGNED ? Infinity : HAND_SIZE - zoneCount(zone);
  return { moving, room, fits: moving.length > 0 && moving.length <= room };
}

function zoneName(zone, lang) {
  return zone === UNASSIGNED ? CONS_TEXT[lang].neutral : SEAT_LABEL[lang][zone];
}

// Déplace `cards` dans `zone`, en un seul geste annulable. Tout ou rien : une
// main qui n'a pas la place pour toute la sélection n'en prend aucune, et la
// sélection reste en main pour viser ailleurs. Renvoie vrai si elles ont bougé.
function dropCards(cards, zone) {
  const lang = $("#lang").value;
  const t = CONS_TEXT[lang];
  const plan = dropPlan(cards, zone);
  if (!plan.moving.length) {
    renderBoundsCards();
    return false;
  }
  if (!plan.fits) {
    // setError annonce lui-même : un announce() de plus ferait dire deux fois
    // la même phrase, une fois poliment et une fois en interrompant.
    setError($("#cons-error"), plan.room === 0
      ? t.errFull(zoneName(zone, lang))
      : t.errRoom(zoneName(zone, lang), plan.room, plan.moving.length));
    renderBoundsCards();
    return false;
  }
  for (const card of plan.moving) moveCard(card, zone);
  setError($("#cons-error"), validateBounds(lang));
  const last = plan.moving[plan.moving.length - 1];
  refocusCard = { zone, suit: last.suit, rank: last.rank };
  rovingByZone[zone] = { suit: last.suit, rank: last.rank };
  announce(plan.moving.length === 1
    ? t.cardDropped(cardLabel(plan.moving[0], lang), zoneName(zone, lang))
    : t.cardsDropped(plan.moving.length, zoneName(zone, lang)));
  commitZones();
  return true;
}

// Dépose la sélection dans `zone`. `fallback` est la carte à refocaliser si
// le dépôt est refusé — celle sur laquelle on se trouvait.
//
// Un refus — la main est pleine — laisse les cartes en main : on les tient
// toujours, et on peut viser ailleurs sans les reprendre.
function dropHeld(zone, fallback) {
  const held = selection;
  clearSelection();
  if (dropCards(held, zone)) return;
  selection = held;
  refocusCard = fallback;
  renderBoundsCards();
}

// Un clic sur une carte :
// - sur une carte prise, la repose ;
// - avec Maj, prend toutes les cartes de la zone entre la précédente et elle ;
// - dans une main où l'on a déjà pris des cartes (ou avec Ctrl/Cmd, depuis
//   n'importe quelle main), l'ajoute à la sélection ;
// - dans une autre main, y dépose la sélection. C'est ce qui rend le geste
//   praticable au doigt : on touche les cartes, puis n'importe quelle carte de
//   la destination, plus facile à viser que le fond d'une main.
function activateCard(card, mods = {}) {
  const lang = $("#lang").value;
  const t = CONS_TEXT[lang];
  refocusCard = card;
  rovingByZone[card.zone] = { suit: card.suit, rank: card.rank };
  if (mods.range && selAnchor && selAnchor.zone === card.zone && !sameCard(selAnchor, card)) {
    const list = zoneCardList(card.zone);
    const at = (c) => list.findIndex((x) => x.suit === c.suit && x.rank === c.rank);
    const [a, b] = [at(selAnchor), at(card)].sort((x, y) => x - y);
    for (const c of list.slice(a, b + 1)) if (!isSelected(c)) selection.push(c);
    selAnchor = card;
    announce(t.cardHeld(cardLabel(card, lang), selection.length));
    renderBoundsCards();
    return;
  }
  if (isSelected(card)) {
    selection = selection.filter((c) => !sameCard(c, card));
    if (sameCard(selAnchor, card)) selAnchor = selection[selection.length - 1] || null;
    announce(t.cardReleased(cardLabel(card, lang)));
    renderBoundsCards();
    return;
  }
  if (selection.length && !mods.add && !selection.some((c) => c.zone === card.zone)) {
    dropHeld(card.zone, card);
    return;
  }
  selection.push(card);
  selAnchor = card;
  announce(t.cardHeld(cardLabel(card, lang), selection.length));
  renderBoundsCards();
}

// Le symbole d'une couleur : prend toutes ses cartes dans la zone, ou les
// repose si elles étaient déjà toutes prises. Il ne dépose jamais : sa cible
// est trop petite pour qu'on l'ait visée par hasard.
function toggleSuit(zone, suit) {
  const lang = $("#lang").value;
  const cards = zoneCardList(zone).filter((c) => c.suit === suit);
  if (!cards.length) return;
  if (cards.every(isSelected)) {
    selection = selection.filter((c) => !(c.zone === zone && c.suit === suit));
    if (selAnchor && selAnchor.zone === zone && selAnchor.suit === suit) {
      selAnchor = selection[selection.length - 1] || null;
    }
    announce(CONS_TEXT[lang].selCount(selection.length));
  } else {
    for (const c of cards) if (!isSelected(c)) selection.push(c);
    selAnchor = cards[cards.length - 1];
    announce(CONS_TEXT[lang].selCount(selection.length));
  }
  renderBoundsCards();
}

// Déposer la sélection dans une zone visée directement — le fond d'une main,
// ou une main vide, qui n'a aucune carte sur laquelle cliquer.
function activateZone(zone) {
  if (!selection.length) return;
  dropHeld(zone, selection[0]);
}

// Le bandeau de la sélection : combien de cartes on tient et où les poser,
// avec de quoi tout reposer. Collé au bas de l'écran sur téléphone, il reste
// en vue pendant qu'on fait défiler la table jusqu'à la main visée.
function renderSelectionBar() {
  const bar = $("#cons-selbar");
  if (!bar) return;
  const t = CONS_TEXT[$("#lang").value];
  const n = selection.length;
  bar.classList.toggle("hidden", n === 0);
  if (!n) return;
  $("#cons-selbar-text").innerHTML =
    `<b>${esc(t.selCount(n))}</b> — ${esc(n > 1 ? t.selHintMany : t.selHint)}`;
  $("#cons-selbar-clear").textContent = t.selClear;
}

$("#cons-selbar-clear").addEventListener("click", () => {
  clearSelection();
  announce(CONS_TEXT[$("#lang").value].selectionCleared);
  renderBoundsCards();
});

// Glisser une carte prise emporte toute la sélection ; glisser une autre carte
// n'emporte qu'elle, et la sélection est reposée — c'est ce qu'on attend d'un
// gestionnaire de fichiers, et cela évite d'emmener sans le voir des cartes
// prises plus tôt dans une autre main.
function startDrag(ev) {
  dragState.moved = true;
  const card = dragState.card;
  dragState.cards = isSelected(card) ? selection.slice() : [card];
  clearSelection();
  const cls = RED_SUITS.has(card.suit) ? " red" : "";
  const ghost = document.createElement("div");
  ghost.className = "card-ghost";
  const more = dragState.cards.length - 1;
  ghost.innerHTML =
    `<span class="suitsym${cls}">${SUIT_SYMBOLS[card.suit]}</span>` +
    `<span class="${cls.trim()}">${rankHTML(card.rank, $("#lang").value)}</span>` +
    (more ? `<span class="ghost-count">+${more}</span>` : "");
  document.body.appendChild(ghost);
  dragState.ghost = ghost;
  dragState.els = dragState.cards.map(cardElement).filter(Boolean);
  for (const el of dragState.els) el.classList.add("dragging");
  renderSelectionBar();
}

function endDrag() {
  if (dragState && dragState.ghost) dragState.ghost.remove();
  if (dragState && dragState.els) for (const el of dragState.els) el.classList.remove("dragging");
  clearDropHints();
  dragState = null;
}

$("#constraints").addEventListener("pointerdown", (ev) => {
  if (ev.pointerType === "mouse" && ev.button !== 0) return;
  if (ev.target.closest("input, button, select")) return;
  const suitEl = ev.target.closest("[data-suit-pick]");
  const cardEl = ev.target.closest(".card");
  const zoneEl = ev.target.closest("[data-zone]");
  if (!suitEl && !cardEl && !zoneEl) return;
  dragState = {
    suitPick: suitEl ? { zone: suitEl.dataset.owner, suit: suitEl.dataset.suitPick } : null,
    card: cardEl ? cardFromEl(cardEl) : null,
    el: cardEl,
    zone: zoneEl ? zoneEl.dataset.zone : null,
    x0: ev.clientX,
    y0: ev.clientY,
    moved: false,
    pointerId: ev.pointerId,
  };
});

window.addEventListener("pointermove", (ev) => {
  if (!dragState || ev.pointerId !== dragState.pointerId || !dragState.card) return;
  if (!dragState.moved) {
    if (Math.hypot(ev.clientX - dragState.x0, ev.clientY - dragState.y0) < DRAG_THRESHOLD) return;
    startDrag(ev);
  }
  dragState.ghost.style.transform = `translate(${ev.clientX}px, ${ev.clientY}px)`;
  clearDropHints();
  // La zone d'origine ne s'illumine pas : y relâcher la carte ne fait rien,
  // ce n'est pas un refus.
  const zone = zoneUnder(ev.clientX, ev.clientY);
  if (zone && dragState.cards.some((c) => c.zone !== zone)) {
    const target = document.querySelector(`[data-zone="${zone}"]`);
    target.classList.add(zoneAccepts(zone, dragState.cards) ? "drop-ok" : "drop-no");
  }
  ev.preventDefault();
}, { passive: false });

window.addEventListener("pointerup", (ev) => {
  if (!dragState || ev.pointerId !== dragState.pointerId) return;
  const st = dragState;
  endDrag();

  if (st.moved) {
    const zone = zoneUnder(ev.clientX, ev.clientY);
    if (zone) {
      // Refusé, le glisser rend la sélection qu'il emportait : on vise
      // ailleurs sans reprendre les cartes une à une.
      if (!dropCards(st.cards, zone) && st.cards.length > 1) {
        selection = st.cards;
        renderBoundsCards();
      }
    } else if (st.cards.length > 1) {
      selection = st.cards;
      renderBoundsCards();
    } else {
      renderBoundsCards();
    }
    return;
  }

  // Simple click: pick a card up, then click its destination. Clicking a card
  // of another zone drops the selection there — handy on touch screens, where
  // an empty spot can be hard to aim at. Le clavier passe par les mêmes
  // fonctions : un seul modèle pour les deux entrées.
  const mods = { add: ev.ctrlKey || ev.metaKey, range: ev.shiftKey };
  if (st.suitPick) toggleSuit(st.suitPick.zone, st.suitPick.suit);
  else if (st.card) activateCard(st.card, mods);
  else if (st.zone) activateZone(st.zone);
});

window.addEventListener("pointercancel", endDrag);

// Déplace le focus d'une carte à l'autre à l'intérieur d'une main, et déplace
// le tabindex avec lui : la main se souvient de l'endroit où on l'a quittée.
function focusCardAt(zone, index) {
  const list = zoneCardList(zone);
  if (!list.length) return;
  const card = list[Math.max(0, Math.min(index, list.length - 1))];
  rovingByZone[zone] = { suit: card.suit, rank: card.rank };
  const el = cardElement(card);
  if (!el) return;
  for (const other of $(`[data-zone="${zone}"]`).querySelectorAll(".card")) {
    other.tabIndex = -1;
  }
  el.tabIndex = 0;
  el.focus();
}

function cardElement(card) {
  return document.querySelector(
    `.card[data-owner="${card.zone}"][data-suit="${card.suit}"][data-rank="${card.rank}"]`
  );
}

$("#constraints").addEventListener("keydown", (ev) => {
  if (ev.key === "Escape" && selection.length) {
    const lang = $("#lang").value;
    const card = selection[selection.length - 1];
    const many = selection.length > 1;
    clearSelection();
    refocusCard = card;
    announce(many ? CONS_TEXT[lang].selectionCleared : CONS_TEXT[lang].cardReleased(cardLabel(card, lang)));
    renderBoundsCards();
    ev.preventDefault();
    return;
  }

  const cardEl = ev.target.closest(".card");
  // Une main vide : son corps porte le tabindex, et Entrée y dépose.
  if (!cardEl) {
    const dropEl = ev.target.closest("[role='button'][tabindex='0']");
    const zoneEl = dropEl && dropEl.closest("[data-zone]");
    if (zoneEl && (ev.key === "Enter" || ev.key === " ")) {
      ev.preventDefault();
      activateZone(zoneEl.dataset.zone);
    }
    return;
  }

  const card = cardFromEl(cardEl);
  const list = zoneCardList(card.zone);
  const at = list.findIndex((c) => c.suit === card.suit && c.rank === card.rank);

  switch (ev.key) {
    case "Enter":
    case " ":
      ev.preventDefault();
      activateCard(card, { add: ev.ctrlKey || ev.metaKey, range: ev.shiftKey });
      return;
    case "ArrowRight":
      ev.preventDefault();
      focusCardAt(card.zone, at + 1);
      return;
    case "ArrowLeft":
      ev.preventDefault();
      focusCardAt(card.zone, at - 1);
      return;
    case "Home":
      ev.preventDefault();
      focusCardAt(card.zone, 0);
      return;
    case "End":
      ev.preventDefault();
      focusCardAt(card.zone, list.length - 1);
      return;
    case "ArrowDown":
    case "ArrowUp": {
      // D'une couleur à l'autre, sur la première carte de la suivante qui en
      // a : sauter sur une chicane laisserait le focus nulle part.
      ev.preventDefault();
      const step = ev.key === "ArrowDown" ? 1 : -1;
      const from = SUIT_KEYS.indexOf(card.suit);
      for (let i = from + step; i >= 0 && i < SUIT_KEYS.length; i += step) {
        const next = list.findIndex((c) => c.suit === SUIT_KEYS[i]);
        if (next !== -1) {
          // Vers le haut, on vise la dernière carte de la couleur atteinte.
          const last = list.reduce((acc, c, j) => (c.suit === SUIT_KEYS[i] ? j : acc), next);
          focusCardAt(card.zone, step === 1 ? next : last);
          return;
        }
      }
      return;
    }
  }
});

// Deals the neutral zone at random over the seats that still have room,
// redrawing until every hand's honour-point count fits its bounds.
$("#cons-fill-btn").addEventListener("click", async () => {
  const lang = $("#lang").value;
  const errEl = $("#cons-error");
  setBoundsError(errEl, validateBounds(lang));
  if (errEl.textContent) return;

  const pool = zoneCards(UNASSIGNED);
  if (!pool.length) return;
  const slots = [];
  for (const seat of SEATS) {
    for (let i = zoneCount(seat); i < HAND_SIZE; i++) slots.push(seat);
  }
  if (slots.length < pool.length) {
    setError(errEl, CONS_TEXT[lang].errNoFill);
    return;
  }

  const base = {};
  for (const seat of SEATS) base[seat] = handHCP(dealZones[seat]);
  const deadline = Date.now() + RANDOM_DEAL_BUDGET_MS;
  let souffle = Date.now();
  const fini = showBusy($("#cons-fill-btn"), CONS_TEXT[lang].busyFill);
  try {
    for (let attempt = 1; ; attempt++) {
      shuffle(slots);
      const added = { N: 0, E: 0, S: 0, W: 0 };
      pool.forEach((card, i) => { added[slots[i]] += HONOUR_POINTS[card.rank] || 0; });
      if (SEATS.every((seat) => isWithinBounds(base[seat] + added[seat], hcpBounds[seat]))) {
        pool.forEach((card, i) => moveCard({ ...card, zone: UNASSIGNED }, slots[i]));
        commitZones();
        // Une distribution partielle vaut génération : donneur et vulnérabilité
        // choisis s'appliquent aussi ici, et « Aléatoire » les tire. Sur une
        // donne chargée, en revanche, on ne redistribue que les cartes : ses
        // tags sont ceux du fichier.
        if (!dealFromFile) {
          setBlockDealer(chosenDealer() || pickRandom(SEATS));
          setBlockVul(chosenVul() || pickRandom(VULS));
        }
        return;
      }
      if (attempt % 256 === 0) {
        if (Date.now() > deadline) {
          setError(errEl, CONS_TEXT[lang].errNoFill);
          return;
        }
        souffle = await breathe(souffle);
      }
    }
  } finally {
    fini();
  }
});

// Sends every card to the neutral zone, to rebuild the deal by hand.
$("#cons-clear-btn").addEventListener("click", () => {
  for (const seat of SEATS) {
    for (const card of zoneCards(seat)) moveCard({ ...card, zone: seat }, UNASSIGNED);
  }
  setError($("#cons-error"), "");
  clearSelection();
  commitZones();
});

// La poubelle d'une main fait de même pour ce seul siège : ses cartes restent
// dans le jeu, en attente d'une autre main ou de « Compléter les mains ».
$("#constraints").addEventListener("click", (ev) => {
  const btn = ev.target.closest("[data-clear-hand]");
  if (!btn) return;
  emptyHandToNeutral(btn.dataset.clearHand);
  setError($("#cons-error"), "");
  clearSelection();
  commitZones();
});

function emptyHandToNeutral(seat) {
  for (const card of zoneCards(seat)) moveCard({ ...card, zone: seat }, UNASSIGNED);
}

// ---------- reconnaissance des cartes par photo (serveur IA) ----------

// Modèle de vision interrogé à travers le serveur IA. Un modèle « flash » de
// petite taille suffit à lire des cartes et répond en quelques secondes ;
// c'est celui que l'application bridgeteacher utilise pour la même tâche.
const IA_MODEL = "google/gemini-3.1-flash-lite";

// Côté serveur, /api/chat rend { success, response } : la réponse utile est du
// texte, que l'on demande donc en JSON strict. Prompts repris de bridgeteacher,
// où ils ont fait leurs preuves : le modèle traduit lui-même les figures
// écrites dans une autre langue (R/D/V → K/Q/J).
const PROMPT_DEAL = `Analyze this image of a bridge deal showing 4 hands in compass layout: North at top, South at bottom, West at left, East at right.
For each direction (N, E, S, W), list all visible card ranks per suit.
Return ONLY a single valid JSON object (no markdown, no explanation, no code block):
{"N":{"S":[],"H":[],"D":[],"C":[]},"E":{"S":[],"H":[],"D":[],"C":[]},"S":{"S":[],"H":[],"D":[],"C":[]},"W":{"S":[],"H":[],"D":[],"C":[]}}
Rules:
- Suit keys: S=Spades/Piques, H=Hearts/Coeurs, D=Diamonds/Carreaux, C=Clubs/Trefles
- Rank values must be: A K Q J T 9 8 7 6 5 4 3 2  (use T for 10, A for Ace)
- Convert from ANY language: French (R=K, D=Q, V=J, A stays A), Spanish (C=K, D=Q, J=J), German (B=J, D=Q, K=K), etc.
- Each array contains rank strings in the order they appear in the image
- If a suit is missing from a hand, use an empty array []
Return ONLY the raw JSON object, nothing else.`;

const PROMPT_HAND = `Analyze this image of playing cards. Identify every visible playing card.
Return ONLY a single valid JSON object (no markdown, no explanation, no code block):
{"card_count":<number>,"cards":[{"rank":"<rank>","suit":"<suit>"},...]}
Rules:
- rank must be one of: A K Q J T 9 8 7 6 5 4 3 2  (use T for 10, A for Ace)
- suit must be one of: S H D C  (S=Spades/Piques, H=Hearts/Coeurs, D=Diamonds/Carreaux, C=Clubs/Trefles)
- Cards may appear in any language: French (R=K, D=Q, V=J, Coeur=H, Carreau=D, Trefle=C, Pique=S), or other languages - always convert to English codes above.
Return ONLY the raw JSON object, nothing else.`;

// Le JSON de l'IA nomme les couleurs comme le PBN ; les zones, elles, ont des
// clés en clair (voir SUIT_KEYS).
const SUIT_FROM_CODE = { S: "spades", H: "hearts", D: "diamonds", C: "clubs" };

// Cible de la photo en cours : « deal » pour les quatre mains, sinon le siège.
// Un seul champ fichier sert tous les boutons, il faut donc retenir lequel a
// déclenché la prise de vue.
let photoTarget = null;
// Vrai pendant l'aller-retour avec le serveur IA : les boutons attendent.
let photoBusy = false;
// Orientation de l'appareil au déclenchement, pour redresser la photo.
let photoAngle = 0;

// Les boutons photo n'ont de sens que si le serveur IA répond : sinon ils
// s'éteignent, l'infobulle disant pourquoi. Appelée à chaque sondage du
// serveur (renderIaHealth) et après chaque redessin des mains.
function renderPhotoButtons() {
  const lang = $("#lang").value;
  const t = UI_TEXT[lang];
  const off = !iaOnline();
  for (const btn of document.querySelectorAll("[data-photo]")) {
    const target = btn.dataset.photo;
    const name = target === "deal"
      ? t.photoDeal
      : CONS_TEXT[lang].photoHand(SEAT_LABEL[lang][target]);
    btn.disabled = off || photoBusy;
    // Éteint, le bouton dit pourquoi : serveur muet, ou pas encore sondé —
    // faute d'URL en distant, par exemple.
    btn.title = off
      ? (iaHealthState === "offline" ? t.photoOffline : t.photoUnknown)
      : name;
    btn.setAttribute("aria-label", name);
    // La même infobulle que ses voisins, et le même repli en bandeau bas :
    // le `title` natif ne se montre pas au doigt.
    btn.dataset.tip = btn.title;
  }
}

// Un clic sur n'importe quel bouton photo ouvre l'appareil (sur mobile,
// capture="environment" visant la caméra arrière) ou le sélecteur de fichiers.
// Les boutons des mains sont recréés à chaque redessin, d'où l'écoute posée à
// la racine du document.
document.addEventListener("click", (ev) => {
  const btn = ev.target.closest("[data-photo]");
  if (!btn || btn.disabled) return;
  photoTarget = btn.dataset.photo;
  photoAngle = deviceAngle();
  $("#photo-input").click();
});

// La photo passe d'abord par l'éditeur : le modèle lit d'autant mieux qu'on lui
// épargne le décor autour des cartes, et l'orientation relevée à la volée ne
// suffit pas toujours à la redresser.
$("#photo-input").addEventListener("change", (ev) => {
  const file = ev.target.files[0];
  // Réinitialisé tout de suite : reprendre la même photo doit relancer la
  // lecture, y compris après un abandon dans l'éditeur.
  ev.target.value = "";
  if (file) openPhotoEditor(file);
});

function showPhotoStatus(text, isError) {
  const el = $("#photo-status");
  // Le même élément porte les deux tons : « Lecture des cartes… » n'est pas
  // une erreur, « Aucune carte reconnue » en est une. Chacun son annonce.
  if (isError) setError(el, text);
  else setStatus(el, text);
  el.classList.toggle("error", !!isError);
  el.classList.toggle("muted", !isError);
}

// Envoie au serveur IA la photo sortie de l'éditeur (data URL déjà redressée,
// recadrée et réduite) et verse les cartes lues dans la donne.
async function recognizePhoto(image) {
  const t = UI_TEXT[$("#lang").value];
  const target = photoTarget;
  if (!target) return;
  if (!iaURL()) {
    showPhotoStatus(t.photoNoUrl, true);
    return;
  }
  photoBusy = true;
  renderPhotoButtons();
  showPhotoStatus(t.photoBusy, false);
  const cancel = new AbortController();
  const hideWait = showPhotoWait(cancel);
  try {
    const resp = await fetchWithTimeout(iaURL() + "/api/chat", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        text: target === "deal" ? PROMPT_DEAL : PROMPT_HAND,
        model: IA_MODEL,
        image,
      }),
      signal: cancel.signal,
    }, TIMEOUT_VISION);
    let body;
    try {
      body = await resp.json();
    } catch (err) {
      throw new Error(t.photoBadAnswer);
    }
    // Une réponse arrivée juste après « Annuler » ne touche plus à la donne.
    if (cancel.signal.aborted) throw new Error(t.photoCancelled);
    if (!resp.ok || !body.success) throw new Error(body.error || t.photoFailed);
    const answer = parseIaJSON(body.response);
    const report = target === "deal"
      ? applyPhotoDeal(answer)
      : applyPhotoHand(target, answer);
    showPhotoStatus(photoReport(report), report.read === 0);
  } catch (err) {
    // Renoncer n'est pas une erreur : le compte rendu le dit sans l'alarme.
    if (cancel.signal.aborted) showPhotoStatus(t.photoCancelled, false);
    else showPhotoStatus(err.message, true);
  } finally {
    hideWait();
    photoBusy = false;
    renderPhotoButtons();
  }
}

// Voile d'attente de la lecture : il paraît aussitôt (l'attente se compte en
// dizaines de secondes, pas en clignements), égrène les secondes et offre
// « Annuler », qui interrompt la requête. Rend la fonction qui le retire.
function showPhotoWait(cancel) {
  const veil = $("#photo-loading");
  const elapsed = $("#photo-loading-elapsed");
  const button = $("#photo-loading-cancel");
  const start = Date.now();
  const tick = () => {
    const t = UI_TEXT[$("#lang").value];
    elapsed.textContent = t.photoElapsed(Math.floor((Date.now() - start) / 1000));
  };
  tick();
  const timer = setInterval(tick, 1000);
  const onCancel = () => cancel.abort();
  const onKey = (ev) => {
    if (ev.key === "Escape") cancel.abort();
    // Tabuler ne mène nulle part ailleurs : la page est sous le voile.
    if (ev.key === "Tab") {
      ev.preventDefault();
      button.focus();
    }
  };
  button.addEventListener("click", onCancel);
  document.addEventListener("keydown", onKey);
  const opener = document.activeElement;
  veil.classList.remove("hidden");
  // Le seul geste possible sous le voile : le focus y va.
  button.focus();
  return () => {
    clearInterval(timer);
    button.removeEventListener("click", onCancel);
    document.removeEventListener("keydown", onKey);
    veil.classList.add("hidden");
    elapsed.textContent = "";
    if (opener && document.contains(opener)) opener.focus();
  };
}

function photoReport(report) {
  const t = UI_TEXT[$("#lang").value];
  if (!report.read) return t.photoNone;
  return t.photoRead(report.read) + (report.dropped ? t.photoDropped(report.dropped) : "");
}

// Le modèle promet du JSON nu, mais l'entoure parfois d'un bloc de code.
function parseIaJSON(response) {
  const t = UI_TEXT[$("#lang").value];
  const text = String(response || "")
    .trim()
    .replace(/^```(?:json)?/, "")
    .replace(/```$/, "")
    .trim();
  try {
    return JSON.parse(text);
  } catch (err) {
    throw new Error(t.photoBadAnswer);
  }
}

// Rangs tels que l'IA les rend : on accepte « 10 » et « 1 » par prudence, on
// rejette le reste.
function normalizeRank(raw) {
  const rank = String(raw == null ? "" : raw).trim().toUpperCase();
  if (rank === "10") return "T";
  if (rank === "1") return "A";
  return RANKS.includes(rank) ? rank : null;
}

// { card_count, cards: [{ rank, suit }] } → la main d'un siège. Les cartes lues
// remplacent celles qu'il détenait : les anciennes repartent en zone neutre, et
// celles qu'un autre siège détenait lui sont retirées — une carte n'est jamais
// à deux endroits.
function applyPhotoHand(seat, data) {
  const seen = new Set();
  const kept = [];
  let dropped = 0;
  for (const raw of (data && data.cards) || []) {
    const suit = SUIT_FROM_CODE[String(raw && raw.suit).trim().toUpperCase()];
    const rank = normalizeRank(raw && raw.rank);
    if (!suit || !rank || seen.has(suit + rank) || kept.length >= HAND_SIZE) {
      dropped++;
      continue;
    }
    seen.add(suit + rank);
    kept.push({ suit, rank });
  }
  emptyHandToNeutral(seat);
  for (const card of kept) {
    takeCard(card);
    dealZones[seat][card.suit] = sortRanks(dealZones[seat][card.suit] + card.rank);
  }
  clearSelection();
  setError($("#cons-error"), "");
  commitZones();
  return { read: kept.length, dropped };
}

// { N: { S: [...], ... }, ... } → les quatre mains, la donne étant refaite de
// zéro. Les cartes que la photo n'a pas livrées attendent en zone neutre :
// « Compléter les mains » achève alors la donne.
function applyPhotoDeal(data) {
  const zones = emptyZones();
  const placed = new Set();
  let read = 0;
  let dropped = 0;
  for (const seat of SEATS) {
    const hand = (data && data[seat]) || {};
    for (const code of SUIT_CODES) {
      const suit = SUIT_FROM_CODE[code];
      for (const raw of Array.isArray(hand[code]) ? hand[code] : []) {
        const rank = normalizeRank(raw);
        if (!rank || placed.has(suit + rank) || handLength(zones[seat]) >= HAND_SIZE) {
          dropped++;
          continue;
        }
        placed.add(suit + rank);
        zones[seat][suit] = sortRanks(zones[seat][suit] + rank);
        read++;
      }
    }
  }
  for (const suit of SUIT_KEYS) {
    for (const rank of RANKS) {
      if (!placed.has(suit + rank)) {
        zones[UNASSIGNED][suit] = sortRanks(zones[UNASSIGNED][suit] + rank);
      }
    }
  }
  dealZones = zones;
  clearSelection();
  setError($("#cons-error"), "");
  commitZones();
  return { read, dropped };
}

// Retire une carte de partout où elle se trouve : la photo dit à qui elle est,
// pas d'où elle vient.
function takeCard(card) {
  for (const zone of ZONES) {
    dealZones[zone][card.suit] = dealZones[zone][card.suit].replace(card.rank, "");
  }
}

// ---------- photo : lecture et mise en forme de l'image ----------

// Orientation de l'appareil, en degrés (0, 90, 180, 270) : screen.orientation
// est le standard, window.orientation le repli de Safari iOS.
function deviceAngle() {
  if (screen.orientation && screen.orientation.angle != null) {
    return ((screen.orientation.angle % 360) + 360) % 360;
  }
  const a = window.orientation || 0;
  return ((a % 360) + 360) % 360;
}

function readDataURL(file) {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = (ev) => resolve(ev.target.result);
    reader.onerror = () => reject(new Error(UI_TEXT[$("#lang").value].photoFailed));
    reader.readAsDataURL(file);
  });
}

function loadImage(dataUrl) {
  return new Promise((resolve, reject) => {
    const img = new Image();
    img.onerror = () => reject(new Error(UI_TEXT[$("#lang").value].photoFailed));
    img.onload = () => resolve(img);
    img.src = dataUrl;
  });
}

// Réduit la photo et la redresse selon l'angle relevé au déclenchement (sans
// lire l'EXIF) : une image de 12 Mpx est inutilement lourde pour le modèle, et
// une photo prise appareil tourné arrive couchée. Le reste des quarts de tour
// est laissé à l'éditeur, où l'utilisateur voit ce qu'il redresse.
async function shrinkImage(dataUrl, angle, maxDim = 2560) {
  const img = await loadImage(dataUrl);
  const srcW = img.naturalWidth;
  const srcH = img.naturalHeight;
  const scale = Math.min(1, maxDim / Math.max(srcW, srcH));
  const w = Math.round(srcW * scale);
  const h = Math.round(srcH * scale);
  // Quarts de tour à rendre pour compenser la rotation de l'appareil.
  const turns = -quarterTurns(angle / 90);
  const canvas = document.createElement("canvas");
  drawTurned(canvas, img, turns, w, h);
  return canvas.toDataURL("image/jpeg", 0.92);
}

// Nombre de quarts de tour, ramené dans 0..3 quel que soit le signe.
function quarterTurns(n) {
  return ((Math.round(n) % 4) + 4) % 4;
}

// Dessine l'image dans le canevas en lui faisant faire `turns` quarts de tour
// dans le sens des aiguilles d'une montre, à la taille w × h demandée (celle de
// l'image avant rotation ; le canevas, lui, échange ses côtés sur un quart ou
// trois quarts de tour).
function drawTurned(canvas, img, turns, w, h) {
  const steps = quarterTurns(turns);
  const swap = steps % 2 === 1;
  canvas.width = swap ? h : w;
  canvas.height = swap ? w : h;
  const ctx = canvas.getContext("2d");
  ctx.setTransform(1, 0, 0, 1, 0, 0);
  ctx.clearRect(0, 0, canvas.width, canvas.height);
  ctx.translate(canvas.width / 2, canvas.height / 2);
  ctx.rotate((steps * Math.PI) / 2);
  ctx.drawImage(img, -w / 2, -h / 2, w, h);
}

// ---------- éditeur de photo : recadrage et rotation ----------

// La photo est rarement cadrée comme le modèle l'aimerait : le décor autour des
// cartes lui coûte des cartes lues, et l'orientation relevée à la prise de vue
// ne redresse pas une photo prise à plat sur la table. L'éditeur intercale donc
// deux gestes entre l'appareil et le serveur : des quarts de tour, et un
// rectangle de recadrage.

// Image sortie de la prise de vue, déjà réduite et redressée selon l'appareil.
let photoEditImage = null;
// Quarts de tour ajoutés à la main, dans le sens des aiguilles d'une montre.
let photoEditTurns = 0;
// Rectangle de recadrage, en fractions de l'image affichée (donc invariant à la
// taille à laquelle le canevas est présenté).
let photoCrop = { x: 0, y: 0, w: 1, h: 1 };
// Geste en cours sur le rectangle : { mode, start, base }.
let cropDrag = null;

// En deçà, le geste n'était pas un recadrage mais une tape : le rectangle
// repart alors sur l'image entière plutôt que de disparaître.
const CROP_MIN = 0.03;

const FULL_CROP = { x: 0, y: 0, w: 1, h: 1 };

function clamp01(v) {
  return Math.min(1, Math.max(0, v));
}

// Le rectangle couvre toute l'image : il n'y a alors rien à déplacer, et le
// premier glissement doit tracer une sélection plutôt que de ne rien faire —
// le cadre occupant l'image entière, il intercepterait sinon tous les gestes.
function isFullCrop() {
  return photoCrop.w > .999 && photoCrop.h > .999;
}

// Prépare la photo et ouvre l'éditeur. L'échec de lecture du fichier se dit à
// l'endroit habituel, dans la barre d'état sous les boutons.
// Le bouton qui a ouvert l'éditeur : le focus lui revient à la fermeture,
// sans quoi il repartirait en tête de document.
let photoOpener = null;

// Les éléments focalisables de la boîte, dans l'ordre. Recalculé à chaque
// tabulation : le bouton photo d'une main peut apparaître ou disparaître.
function photoEditorStops() {
  return [...$("#photo-editor").querySelectorAll(
    'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
  )].filter((el) => el.offsetParent !== null && !el.disabled);
}

async function openPhotoEditor(file) {
  const t = UI_TEXT[$("#lang").value];
  try {
    showPhotoStatus(t.photoBusy, false);
    photoEditImage = await loadImage(await shrinkImage(await readDataURL(file), photoAngle));
    photoEditTurns = 0;
    photoCrop = { ...FULL_CROP };
    setError($("#photo-edit-error"), "");
    photoOpener = document.activeElement;
    $("#photo-editor").classList.remove("hidden");
    renderPhotoImage();
    // Le focus entre dans la boîte : aria-modal la déclare modale, mais ne
    // déplace rien de lui-même. Sans cela le focus restait derrière le voile,
    // sur une page que l'on ne peut plus voir ni atteindre.
    $("#photo-crop").focus();
    showPhotoStatus("", false);
  } catch (err) {
    closePhotoEditor();
    showPhotoStatus(err.message || t.photoFailed, true);
  }
}

function closePhotoEditor() {
  $("#photo-editor").classList.add("hidden");
  photoEditImage = null;
  cropDrag = null;
  // Rendre le focus à son point de départ : c'est de là qu'on est parti, et
  // c'est là qu'on reprend.
  if (photoOpener && document.contains(photoOpener)) photoOpener.focus();
  photoOpener = null;
}

// Redessine le canevas dans son orientation courante, puis replace le cadre.
function renderPhotoImage() {
  if (!photoEditImage) return;
  drawTurned($("#photo-edit-canvas"), photoEditImage, photoEditTurns,
    photoEditImage.naturalWidth, photoEditImage.naturalHeight);
  renderPhotoCrop();
}

function renderPhotoCrop() {
  // Le reproche fait au cadre ne survit pas au geste qui le corrige.
  setError($("#photo-edit-error"), "");
  const box = $("#photo-crop");
  box.style.left = `${photoCrop.x * 100}%`;
  box.style.top = `${photoCrop.y * 100}%`;
  box.style.width = `${photoCrop.w * 100}%`;
  box.style.height = `${photoCrop.h * 100}%`;
}

// Fait tourner l'image d'un quart de tour, et le rectangle avec elle : celui qui
// a déjà cadré ses cartes ne recommence pas parce qu'il redresse ensuite.
function turnPhoto(dir) {
  if (!photoEditImage) return;
  photoEditTurns = quarterTurns(photoEditTurns + dir);
  const { x, y, w, h } = photoCrop;
  photoCrop = dir > 0
    ? { x: 1 - y - h, y: x, w: h, h: w }
    : { x: y, y: 1 - x - w, w: h, h: w };
  renderPhotoImage();
}

$("#photo-rot-left").addEventListener("click", () => turnPhoto(-1));
$("#photo-rot-right").addEventListener("click", () => turnPhoto(1));
$("#photo-crop-reset").addEventListener("click", () => {
  photoCrop = { ...FULL_CROP };
  renderPhotoCrop();
});
$("#photo-cancel").addEventListener("click", closePhotoEditor);

// Le fond ferme l'éditeur, la boîte non : un clic à côté abandonne la photo.
$("#photo-editor").addEventListener("click", (ev) => {
  if (ev.target === $("#photo-editor")) closePhotoEditor();
});

// Déplace ou redimensionne le rectangle de recadrage, en fractions du cadre.
// Les mêmes bornes qu'au pointeur : jamais hors de l'image, jamais plus petit
// que CROP_MIN.
const CROP_STEP = .02;

function nudgeCrop(dx, dy, resize) {
  const c = photoCrop;
  if (resize) {
    photoCrop = {
      x: c.x,
      y: c.y,
      w: Math.min(1 - c.x, Math.max(CROP_MIN, c.w + dx)),
      h: Math.min(1 - c.y, Math.max(CROP_MIN, c.h + dy)),
    };
  } else {
    photoCrop = {
      x: Math.min(1 - c.w, Math.max(0, c.x + dx)),
      y: Math.min(1 - c.h, Math.max(0, c.y + dy)),
      w: c.w,
      h: c.h,
    };
  }
  renderPhotoCrop();
}

// Le rectangle au clavier : les flèches le déplacent, Maj + flèches le
// redimensionnent. Les huit poignées restent au pointeur — les rendre
// focalisables ferait huit arrêts de tabulation pour un seul rectangle, et
// deux touches suffisent à faire le même travail.
$("#photo-crop").addEventListener("keydown", (ev) => {
  if (!photoEditImage) return;
  const map = {
    ArrowLeft: [-CROP_STEP, 0],
    ArrowRight: [CROP_STEP, 0],
    ArrowUp: [0, -CROP_STEP],
    ArrowDown: [0, CROP_STEP],
  };
  const d = map[ev.key];
  if (!d) return;
  ev.preventDefault();
  nudgeCrop(d[0], d[1], ev.shiftKey);
});

document.addEventListener("keydown", (ev) => {
  const editor = $("#photo-editor");
  if (editor.classList.contains("hidden")) return;
  if (ev.key === "Escape") {
    closePhotoEditor();
    return;
  }
  // Le piège : la tabulation tourne dans la boîte. Sans lui, elle en sortait
  // vers une page que le voile rend inatteignable — on tabulait à l'aveugle.
  if (ev.key !== "Tab") return;
  const stops = photoEditorStops();
  if (!stops.length) return;
  const premier = stops[0];
  const dernier = stops[stops.length - 1];
  const actif = document.activeElement;
  if (ev.shiftKey && (actif === premier || !editor.contains(actif))) {
    ev.preventDefault();
    dernier.focus();
  } else if (!ev.shiftKey && (actif === dernier || !editor.contains(actif))) {
    ev.preventDefault();
    premier.focus();
  }
});

// ---------- gestes sur le rectangle de recadrage ----------

// Position du pointeur en fractions du cadre, qui épouse exactement le canevas.
function framePoint(ev) {
  const r = $("#photo-edit-frame").getBoundingClientRect();
  return {
    x: clamp01((ev.clientX - r.left) / r.width),
    y: clamp01((ev.clientY - r.top) / r.height),
  };
}

$("#photo-edit-frame").addEventListener("pointerdown", (ev) => {
  if (!photoEditImage) return;
  const p = framePoint(ev);
  const handle = ev.target.dataset && ev.target.dataset.handle;
  if (handle) {
    cropDrag = { mode: handle, start: p, base: { ...photoCrop } };
  } else if (ev.target.closest("#photo-crop") && !isFullCrop()) {
    cropDrag = { mode: "move", start: p, base: { ...photoCrop } };
  } else {
    // Ailleurs sur l'image : on trace un nouveau rectangle depuis ce point,
    // comme si l'on tirait sa poignée sud-est.
    photoCrop = { x: p.x, y: p.y, w: 0, h: 0 };
    cropDrag = { mode: "se", start: p, base: { ...photoCrop } };
    renderPhotoCrop();
  }
  // Pas de capture du pointeur : la suite du geste est écoutée sur la fenêtre,
  // qui la reçoit de toute façon, et la capture échoue sur un pointeur déjà
  // relâché. preventDefault, lui, empêche le navigateur de comprendre le
  // glissement comme une sélection de texte ou un défilement.
  ev.preventDefault();
});

// La suite du geste est écoutée sur la fenêtre, et non sur le cadre : le doigt
// ou la souris sort volontiers de l'image en tirant une poignée, et un
// relâchement perdu en route laisserait le rectangle figé au premier point.
window.addEventListener("pointermove", (ev) => {
  if (!cropDrag) return;
  const p = framePoint(ev);
  const dx = p.x - cropDrag.start.x;
  const dy = p.y - cropDrag.start.y;
  const b = cropDrag.base;
  if (cropDrag.mode === "move") {
    // Déplacement : le rectangle garde sa taille et reste dans l'image.
    photoCrop = {
      x: Math.min(Math.max(b.x + dx, 0), 1 - b.w),
      y: Math.min(Math.max(b.y + dy, 0), 1 - b.h),
      w: b.w,
      h: b.h,
    };
  } else {
    // Redimensionnement : seuls les bords nommés par la poignée bougent. Les
    // bords croisés sont remis d'aplomb, si bien que tirer une poignée au-delà
    // du bord opposé retourne le rectangle au lieu de l'annuler.
    let left = b.x;
    let top = b.y;
    let right = b.x + b.w;
    let bottom = b.y + b.h;
    if (cropDrag.mode.includes("w")) left = clamp01(b.x + dx);
    if (cropDrag.mode.includes("e")) right = clamp01(b.x + b.w + dx);
    if (cropDrag.mode.includes("n")) top = clamp01(b.y + dy);
    if (cropDrag.mode.includes("s")) bottom = clamp01(b.y + b.h + dy);
    photoCrop = {
      x: Math.min(left, right),
      y: Math.min(top, bottom),
      w: Math.abs(right - left),
      h: Math.abs(bottom - top),
    };
  }
  renderPhotoCrop();
});

for (const type of ["pointerup", "pointercancel"]) {
  window.addEventListener(type, () => {
    if (!cropDrag) return;
    cropDrag = null;
    // Une simple tape sur l'image laisserait un rectangle vide : on la lit
    // comme un abandon du recadrage.
    if (photoCrop.w < CROP_MIN || photoCrop.h < CROP_MIN) photoCrop = { ...FULL_CROP };
    renderPhotoCrop();
  });
}

// Découpe le canevas selon le rectangle et envoie le résultat au serveur IA.
$("#photo-confirm").addEventListener("click", () => {
  const t = UI_TEXT[$("#lang").value];
  const src = $("#photo-edit-canvas");
  if (!photoEditImage) return;
  const sx = Math.round(photoCrop.x * src.width);
  const sy = Math.round(photoCrop.y * src.height);
  const sw = Math.round(photoCrop.w * src.width);
  const sh = Math.round(photoCrop.h * src.height);
  // Un rectangle réduit à rien ne porterait aucune carte : on le dit plutôt que
  // d'envoyer une image vide et de laisser le modèle répondre à côté.
  if (sw < 32 || sh < 32) {
    setError($("#photo-edit-error"), t.photoCropTooSmall);
    return;
  }
  const out = document.createElement("canvas");
  out.width = sw;
  out.height = sh;
  out.getContext("2d").drawImage(src, sx, sy, sw, sh, 0, 0, sw, sh);
  const image = out.toDataURL("image/jpeg", 0.92);
  closePhotoEditor();
  recognizePhoto(image);
});

$("#lang").addEventListener("change", () => {
  const err = $("#cons-error").textContent;
  saveLang($("#lang").value);
  applyLang();
  renderVersion();
  renderBoundsCards();
  if (err) setError($("#cons-error"), validateBounds($("#lang").value));
  renderHealth();
  renderIaHealth();
  // Le résultat affiché a été calculé dans l'autre langue : on le redemande,
  // commentaires et types de mains sont traduits côté serveur.
  // Pas pendant ni après un questionnaire : il garde sa langue, et le calcul le
  // refermerait, score compris (voir resetQuiz).
  if (!$("#result-panel").classList.contains("hidden") && !quiz) simulate();
});

// Builds a S/W/N/E auction grid (head + body rows) from a list of calls,
// padding the first row up to the dealer's column. `cellRenderer(call)`
// returns the <td> HTML for a played call.
function auctionGridHTML(dealer, calls, lang, cellRenderer) {
  const columns = ["S", "W", "N", "E"];
  const headHTML = columns.map((s) => `<th>${esc(SEAT_SHORT[lang][s])}</th>`).join("");
  const cells = [];
  for (let i = 0; i < columns.indexOf(dealer); i++) cells.push(null);
  for (const c of calls) cells.push(c);
  while (cells.length % 4 !== 0) cells.push(undefined);

  const rows = [];
  for (let i = 0; i < cells.length; i += 4) {
    const tds = cells.slice(i, i + 4).map((c) => (c ? cellRenderer(c) : "<td></td>"));
    rows.push(`<tr>${tds.join("")}</tr>`);
  }
  return { headHTML, bodyHTML: rows.join("") };
}

function renderResult(r) {
  const lang = r.lang;
  $("#result-panel").classList.remove("hidden");

  // Hands around the table.
  for (const seat of ["N", "E", "S", "W"]) {
    $("#hand-" + seat).innerHTML = handHTML(seat, r.hands[seat], lang);
  }

  // Centre de la table : le contrat et son déclarant d'abord — c'est la
  // réponse, tout le reste n'est que le contexte qui la date. Donneur et
  // vulnérabilité suivent, en retrait. Rien n'est repris dans une rangée de
  // pastilles au-dessus du tableau : elle répétait mot pour mot ce qui se lit
  // ici, et « Contré » y figurait en clair alors que le contrat porte son X.
  const passedOut = isPass(r.contract);
  const contractHTML = passedOut
    ? esc(r.contract)
    : bidHTML(r.contract, lang) + (r.doubled ? " X" : "");
  $("#table-center").innerHTML = `
    <div class="big">${contractHTML}</div>
    <div>${passedOut ? "" : (lang === "fr" ? "par " : "by ") + esc(SEAT_SHORT[lang][r.declarer])}</div>
    <div>${withColon(lang === "fr" ? "Donneur" : "Dealer", lang)} <b>${esc(SEAT_SHORT[lang][r.dealer])}</b></div>
    <div class="vul-line">${vulHTML(r.vulnerable, lang)}</div>`;

  // Auction grid: columns S W N E, first row padded up to the dealer.
  // Une enchère commentée porte son commentaire en infobulle, au dessin de
  // celle du PAR (auctionTip, plus bas).
  auctionTip.hide();
  auctionTip.calls = r.auction;
  auctionTip.lang = lang;
  // Chaque case porte l'indice de son enchère : la case, la ligne commentée
  // et la main de l'enchérisseur s'éclairent ensemble (voir linkCall).
  const grid = auctionGridHTML(r.dealer, r.auction, lang, (a) => {
    if (!a.comment) {
      return `<td class="bid-cell" data-i="${r.auction.indexOf(a)}">${bidHTML(a.bid, lang)}</td>`;
    }
    return `<td class="bid-cell has-tip" tabindex="0" data-i="${r.auction.indexOf(a)}">` +
      `${bidHTML(a.bid, lang)}</td>`;
  });
  $("#auction-head").innerHTML = grid.headHTML;
  $("#auction-body").innerHTML = grid.bodyHTML;

  // Séquence commentée : toute l'enchère, dans l'ordre. Les enchères que le
  // moteur ne commente pas — les passes d'attente, les trois passes finales —
  // y figurent sans texte plutôt que d'en être retirées : filtrées, elles
  // laissaient une liste qui se lit comme une séquence mais en saute des tours,
  // et un 1♦ suivi d'un Contre du camp d'en face donnait l'ordre des joueurs
  // pour faux. Un passe reste en gris, sans texte : le moteur lui donne
  // désormais toujours un commentaire (ne serait-ce que « pas de quoi
  // intervenir »), mais l'afficher noierait la liste sous des banalités.
  // Chaque ligne porte à gauche l'icône de son arbre de décision, quand le
  // moteur en a tracé un ; l'arbre se déplie sous la ligne (commentsTree).
  commentsTree.auction = r.auction;
  commentsTree.hands = r.hands;
  commentsTree.lang = lang;
  const tip = UI_TEXT[lang].treeShow;
  $("#comments").innerHTML = r.auction
    .map((a, i) => {
      const icon = a.trace && a.trace.length
        ? `<button type="button" class="tree-icon" data-i="${i}" data-tip="${esc(tip)}" ` +
          `aria-label="${esc(tip)}" aria-expanded="false">${TREE_ICON}</button>`
        : `<span class="tree-icon-gap" aria-hidden="true"></span>`;
      const head = `<span class="who">${esc(SEAT_SHORT[lang][a.player])}</span> - ` +
        bidHTML(a.bid, lang);
      return a.comment && !isPass(a.bid)
        ? `<li data-i="${i}">${icon}${withColon(head, lang)} ${esc(a.comment)}</li>`
        : `<li class="silent" data-i="${i}">${icon}${head}</li>`;
    })
    .join("") || `<li class="muted">${lang === "fr" ? "aucune" : "none"}</li>`;

  // Tableau du « PAR » (levées double-mort) : remis à zéro pour la donne qu'on
  // vient d'afficher, puis calculé à la demande par le bouton (voir par.js).
  if (typeof parSetDeal === "function") parSetDeal(r.hands, lang);
  setParReady(true);
  // L'onglet du PAR ouvert suit la donne : il se recalcule avec elle.
  if (!$("#tabpanel-par").hidden && typeof parCompute === "function") parCompute();
}

// Infobulle de la séquence d'enchères : même boîte que celle du PAR (styles
// .par-tip), même conduite — survol à la souris, tape au doigt (une deuxième
// referme), focus au clavier, Échap ; bandeau en bas d'écran quand l'écran est
// étroit ou sans survol.
// La position d'une infobulle, et son repli en bandeau bas quand il n'y a pas
// de survol — un doigt n'en a pas. Partagé par l'infobulle des enchères et
// celle des boutons, qui la plaçaient sinon deux fois du même code.
function tipSheetMode() {
  const noHover = !!(window.matchMedia && window.matchMedia("(hover: none)").matches);
  return noHover || window.innerWidth <= 720;
}

function placeTip(tip, anchor) {
  if (tipSheetMode()) {
    tip.classList.add("sheet");
    tip.style.left = "";
    tip.style.top = "";
    return;
  }
  tip.classList.remove("sheet");
  const r = anchor.getBoundingClientRect();
  const w = tip.offsetWidth;
  const h = tip.offsetHeight;
  const left = Math.max(8, Math.min(r.left + r.width / 2 - w / 2, window.innerWidth - w - 8));
  let top = r.bottom + 8;
  if (top + h > window.innerHeight - 8) top = Math.max(8, r.top - h - 8);
  tip.style.left = Math.round(left) + "px";
  tip.style.top = Math.round(top) + "px";
}

// Les commandes de la donne n'ont plus de texte : leur nom est dans cette
// infobulle, comme le commentaire d'une enchère est dans la sienne. Le titre
// natif ne suffisait pas — il tarde, il ne se montre pas au doigt, et il
// disparaît au clavier.
const buttonTip = {
  open: null,

  el() { return $("#hint-tip"); },

  targetOf(node) {
    return (node && node.closest) ? node.closest("[data-tip]") : null;
  },

  show(btn) {
    const tip = this.el();
    const text = btn && btn.dataset.tip;
    if (!tip || !text || this.open === btn) return;
    this.hide();
    this.open = btn;
    tip.textContent = text;
    tip.classList.remove("hidden");
    placeTip(tip, btn);
  },

  hide() {
    this.open = null;
    const tip = this.el();
    if (!tip) return;
    tip.classList.add("hidden");
    tip.textContent = "";
  },

  bind() {
    document.addEventListener("pointerover", (e) => {
      if (e.pointerType && e.pointerType !== "mouse") return;
      const btn = this.targetOf(e.target);
      if (btn) this.show(btn);
    });
    document.addEventListener("pointerout", (e) => {
      if (e.pointerType && e.pointerType !== "mouse") return;
      const btn = this.targetOf(e.target);
      if (btn && btn === this.open && !btn.contains(e.relatedTarget)) this.hide();
    });
    // Au clavier, l'infobulle suit le focus : c'est le seul moyen de savoir
    // sur quel bouton on se trouve quand il n'a pas de texte.
    document.addEventListener("focusin", (e) => {
      const btn = this.targetOf(e.target);
      if (btn) this.show(btn);
      else this.hide();
    });
    // Le clic fait l'action : l'infobulle n'a plus lieu d'être, et elle
    // masquerait le message qui suit.
    document.addEventListener("click", (e) => {
      if (this.targetOf(e.target)) this.hide();
    });
    document.addEventListener("keydown", (e) => {
      if (e.key === "Escape") this.hide();
    });
    window.addEventListener("scroll", () => {
      if (!tipSheetMode()) this.hide();
    }, { passive: true });
    window.addEventListener("resize", () => this.hide());
  },
};
buttonTip.bind();

const auctionTip = {
  calls: [],
  lang: "fr",
  open: null,
  pointer: "mouse",

  el() { return $("#auction-tip"); },

  cellOf(node) {
    return (node && node.closest) ? node.closest("#auction-body td.has-tip") : null;
  },

  sheetMode() { return tipSheetMode(); },

  place(tip, cell) { placeTip(tip, cell); },

  show(cell) {
    const tip = this.el();
    const call = this.calls[Number(cell && cell.dataset.i)];
    if (!tip || !call || this.open === cell) return;
    // Sur grand écran, la ligne commentée est juste sous la grille et
    // s'éclaire avec la case (voir linkCall) : l'infobulle ne ferait que
    // masquer la grille.
    if (wideLayout.matches) return;
    this.hide();
    this.open = cell;
    cell.classList.add("is-open");
    const lang = this.lang;
    const seat = (SEAT_SHORT[lang] && SEAT_SHORT[lang][call.player]) || call.player;
    tip.innerHTML =
      `<div class="par-tip-head"><b>${esc(seat)}</b><span>${bidHTML(call.bid, lang)}</span></div>` +
      `<div>${esc(call.comment)}</div>`;
    tip.classList.remove("hidden");
    this.place(tip, cell);
  },

  hide() {
    if (this.open) this.open.classList.remove("is-open");
    this.open = null;
    const tip = this.el();
    if (tip) {
      tip.classList.add("hidden");
      tip.innerHTML = "";
    }
  },

  bind() {
    const body = $("#auction-body");
    if (!body) return;
    body.addEventListener("pointerover", (e) => {
      if (e.pointerType && e.pointerType !== "mouse") return;
      const cell = this.cellOf(e.target);
      if (cell) this.show(cell);
    });
    body.addEventListener("pointerout", (e) => {
      if (e.pointerType && e.pointerType !== "mouse") return;
      const cell = this.cellOf(e.target);
      if (cell && cell === this.open && !cell.contains(e.relatedTarget)) this.hide();
    });
    body.addEventListener("click", (e) => {
      const cell = this.cellOf(e.target);
      if (!cell) return;
      if (cell === this.open && this.pointer !== "mouse") this.hide();
      else this.show(cell);
    });
    body.addEventListener("focusin", (e) => {
      const cell = this.cellOf(e.target);
      if (cell) this.show(cell);
    });
    body.addEventListener("focusout", (e) => {
      const cell = this.cellOf(e.target);
      if (cell && cell === this.open) this.hide();
    });
    document.addEventListener("pointerdown", (e) => {
      this.pointer = e.pointerType || "mouse";
      if (this.open && !this.cellOf(e.target)) this.hide();
    });
    document.addEventListener("keydown", (e) => {
      if (e.key === "Escape") this.hide();
    });
    window.addEventListener("scroll", () => {
      if (!this.sheetMode()) this.hide();
    }, { passive: true });
    window.addEventListener("resize", () => this.hide());
  },
};
auctionTip.bind();

// ---------- appels réseau (serveur IA) ----------

// Un fetch qui renonce. Sans cela, un serveur qui accepte la connexion puis se
// tait laisse le bouton grisé indéfiniment : rien ne revient, ni réponse ni
// erreur, et l'application reste en attente jusqu'au rechargement de la page.
//
// Les délais diffèrent selon ce qu'on attend : une sonde doit répondre tout de
// suite, alors que le modèle de vision du serveur IA met des dizaines de
// secondes à lire une photo.
const TIMEOUT_PROBE = 8000;
const TIMEOUT_VISION = 90000;

// options.signal, s'il est fourni, laisse l'appelant renoncer lui-même : son
// abandon remonte tel quel (AbortError), à distinguer du délai dépassé.
async function fetchWithTimeout(url, options, ms = TIMEOUT_PROBE) {
  const ctrl = new AbortController();
  const timer = setTimeout(() => ctrl.abort(), ms);
  const outer = options.signal;
  const relay = () => ctrl.abort();
  if (outer) outer.addEventListener("abort", relay, { once: true });
  try {
    return await fetch(url, { ...options, signal: ctrl.signal });
  } catch (err) {
    if (outer && outer.aborted) throw err;
    // Le renoncement se lit comme une erreur d'abandon : on la retraduit,
    // « The user aborted a request » n'apprenant rien à personne.
    const t = UI_TEXT[$("#lang").value];
    if (err && err.name === "AbortError") throw new Error(t.errTimeout);
    // Un échec réseau arrive en TypeError, dont le message — « Failed to
    // fetch » — ne dit rien à qui ne lit pas l'anglais des navigateurs.
    if (err instanceof TypeError) throw new Error(t.errUnreachable);
    throw err;
  } finally {
    clearTimeout(timer);
    if (outer) outer.removeEventListener("abort", relay);
  }
}

// Dernier état connu du moteur, gardé pour le réafficher tel quel quand la
// langue change, sans le redemander.
let healthState = null; // "online", "offline", ou null tant qu'on ne sait pas

function renderHealth() {
  const t = UI_TEXT[$("#lang").value];
  const dot = $("#health-dot");
  dot.className =
    healthState === "online" ? "dot ok" : healthState === "offline" ? "dot ko" : "dot";
  dot.title = healthState ? t[healthState] : t.healthUnknown;
  // Rempli mais jamais vu : le libellé est réservé aux lecteurs d'écran (la
  // pastille et son infobulle portent l'état pour l'œil), et il se réécrit
  // quand même à chaque changement de langue.
  $("#health-text").textContent = healthState ? t[healthState] : "";
  // Rangé dans les réglages, l'état ne se verrait plus : un moteur injoignable
  // se signale sur le bouton qui y mène.
  $("#settings-btn").classList.toggle("warn", healthState === "offline");
}

// Version du moteur, telle que bids-wasm.js la renvoie. null tant qu'on ne l'a
// pas obtenue : le pied de page nomme alors le moteur sans lui prêter une
// révision qu'il ne connaît pas.
let engineVersion = null;

function renderVersion() {
  const el = $("#app-version");
  const t = UI_TEXT[$("#lang").value];
  // Le nom du moteur reste affiché même sans version : il dit à quelle ligne
  // du pied de page appartient la pastille d'état (l'autre étant celle de l'IA).
  if (!engineVersion) {
    el.textContent = t.versionWasm;
    el.removeAttribute("title");
    return;
  }
  // L'étoile signale un moteur compilé sur un dépôt modifié : la révision
  // seule prétendrait alors correspondre à un commit qu'il ne reflète pas.
  el.textContent = `${t.versionWasm} ${engineVersion.revision}${engineVersion.modified ? " *" : ""}`;
  const parts = [t.versionWasmTitle];
  if (engineVersion.time) parts.push(engineVersion.time);
  if (engineVersion.go) parts.push(engineVersion.go);
  if (engineVersion.modified) parts.push(t.versionModified);
  el.title = parts.join(" · ");
}

async function fetchVersion() {
  try {
    const body = await bidsLocal.version();
    engineVersion = body && body.revision ? body : null;
  } catch (err) {
    engineVersion = null;
  }
  renderVersion();
}

// Voile d'attente du moteur, posé et retiré par les crochets de bids-wasm.js.
// Le module étant préchargé dès l'ouverture, ce voile ne paraît que si l'on
// demande un calcul avant la fin de ce préchargement : bids-wasm.js ne signale
// que l'attente réelle, jamais le téléchargement lui-même (voir awaitModule).
window.onBidsEngineLoad = function (state) {
  $("#engine-loading").classList.toggle("hidden", state !== "start");
};

// Contrôle du moteur au chargement : il rejoue sa donne de référence. Cela
// instancie le module au passage, donc dès l'ouverture de la page — le premier
// calcul est ainsi immédiat.
//
// Ce préchargement a un temps été différé au premier calcul, quand
// l'hébergement mettait ~9 s à répondre à la moindre requête. Sur GitHub
// Pages, bids.wasm arrive compressé (1,3 Mo) en moins d'une seconde, hors du
// chemin critique de l'affichage : le coût ne justifie plus l'attente.
//
// S'il ne se charge pas (fichier absent, page ouverte en file://, navigateur
// sans WebAssembly), la raison s'affiche au pied de page : sans moteur, rien
// ne peut être calculé.
async function checkHealth() {
  healthState = null;
  renderHealth();
  const errEl = $("#engine-error");
  errEl.textContent = "";
  $("#health-text").textContent = "…";
  try {
    healthState = (await bidsLocal.selfCheck()) ? "online" : "offline";
    // La donne affichée au chargement attendait le moteur.
    scheduleAutoBid();
  } catch (err) {
    healthState = "offline";
    errEl.textContent = err.message;
  }
  renderHealth();
  fetchVersion();
}

// ---------- santé du serveur IA ----------

function iaURL() {
  return $("#ia-server").value.trim().replace(/\/+$/, "");
}

// Seule condition pour proposer la reconnaissance des cartes : un serveur IA
// qui a répondu.
function iaOnline() {
  return iaHealthState === "online";
}

let iaHealthState = null; // "online", "offline", ou null tant qu'on ne sait pas

function renderIaHealth() {
  const t = UI_TEXT[$("#lang").value];
  const dot = $("#ia-health-dot");
  dot.className =
    iaHealthState === "online" ? "dot ok" : iaHealthState === "offline" ? "dot ko" : "dot";
  dot.title = iaHealthState ? t[iaHealthState] : t.healthUnknown;
  // Même libellé invisible que pour les enchères : l'état se lit à la couleur
  // de la pastille, et les lecteurs d'écran gardent les mots.
  $("#ia-health-text").textContent = iaHealthState ? t[iaHealthState] : "";
  renderPhotoButtons();
}

// Même sondage que pour le serveur d'enchères, y compris le refus de résoudre
// « /health » contre l'origine de la page quand l'URL manque.
async function checkIaHealth() {
  iaHealthState = null;
  renderIaHealth();
  if (!iaURL()) {
    renderIaHint();
    return;
  }
  $("#ia-health-text").textContent = "…";
  try {
    const resp = await fetchWithTimeout(iaURL() + "/health", {}, TIMEOUT_PROBE);
    const body = await resp.json();
    iaHealthState = resp.ok && body.status === "ok" ? "online" : "offline";
  } catch (err) {
    iaHealthState = "offline";
  }
  renderIaHealth();
}

// Reads/validates the deal and asks the in-page engine for the full
// bidding sequence. Throws with a user-facing message on failure.
async function fetchBid(lang) {
  // The zones are the authoritative view of the deal, and they already track
  // the textarea, so they are checked before any re-parse: re-syncing an
  // incomplete deal would silently drop the unassigned cards.
  if (!pbnGames[selectedGameIdx]) {
    throw new Error(CONS_TEXT[lang].errNoPbn);
  }
  if (zoneCount(UNASSIGNED) > 0 || SEATS.some((seat) => zoneCount(seat) !== HAND_SIZE)) {
    throw new Error(CONS_TEXT[lang].errIncomplete);
  }
  refreshDealSelector(false);
  const pbn = pbnGames[selectedGameIdx];
  if (!pbn) {
    throw new Error(CONS_TEXT[lang].errNoPbn);
  }
  return bidsLocal.bid(pbn, lang);
}

// `auto` : le calcul relancé de lui-même sur une donne qui vient de changer
// (voir scheduleAutoBid). Il ne change ni d'onglet ni de mode, ne fait pas
// défiler la page et ne dit rien s'il échoue : l'utilisateur n'a rien demandé.
async function simulate(opts) {
  const auto = !!(opts && opts.auto === true);
  const btn = $("#bid-btn");
  // Le message va dans #cons-error, qui est sur la même ligne que le bouton.
  // Il partait dans #error, une rangée plus bas, à côté du questionnaire :
  // une donne incomplète produisait un refus qu'on ne voyait pas, et le clic
  // paraissait sans effet.
  const errEl = $("#cons-error");
  if (!auto) {
    setError(errEl, "");
    resetQuiz();
  }
  const lang = $("#lang").value;
  btn.disabled = true;
  try {
    const body = await fetchBid(lang);
    renderResult(body);
    if (!auto) {
      // Le contrat s'affiche au centre de la table en lecture : on y revient.
      editRequested = false;
      renderDealMode();
      // Sur écran étroit, le panneau est sous la table : on y descend, sans
      // quoi le calcul semble n'avoir rien donné. Sur grand écran, il est à
      // côté.
      selectTab("bids", true);
    }
  } catch (err) {
    if (!auto) setError(errEl, err.message);
    hideResult();
  } finally {
    renderBidReady();
  }
}

// ---------- le siège du questionnaire ----------

// Le siège retenu. Un bouton radio est toujours coché — celui de Nord au
// départ — donc ce repli ne sert qu'à se garder d'un document à moitié bâti.
function chosenSeat() {
  const picked = document.querySelector('input[name="seat"]:checked');
  return picked ? picked.value : "N";
}

// Nomme les quatre sièges. En bandeau, la pastille porte le nom entier plutôt
// que son initiale : la place ne manque plus, et « Ouest » se lit sans avoir à
// deviner ce que « O » désigne.
function renderSeatCompass() {
  const lang = $("#lang").value;
  const t = UI_TEXT[lang];
  for (const input of document.querySelectorAll('input[name="seat"]')) {
    const seat = input.value;
    const name = SEAT_LABEL[lang][seat];
    input.setAttribute("aria-label", name);
    input.nextElementSibling.textContent = name;
    const pick = input.closest(".seat-pick");
    // L'infobulle dit la phrase entière — « Votre main est en Nord » — là où
    // le bandeau ne montre que les quatre noms. C'est elle qui porte ce que
    // le libellé « Votre main » disait avant. L'étiquette entière la porte :
    // la cible du survol est la pastille, pas le bouton radio qu'elle cache.
    pick.dataset.tip = t.seatTip(name);
    // La pastille retenue porte une classe, et non un `:has(input:checked)` :
    // ce sélecteur-là n'est pas toujours réévalué quand la case est cochée
    // par le code, et la marque restait sur le siège précédent.
    pick.classList.toggle("is-picked", input.checked);
  }
}

document.addEventListener("change", (ev) => {
  if (ev.target.name === "seat") renderSeatCompass();
});

// ---------- quiz mode: guess your own seat's calls ----------

let quiz = null; // { result, seat, lang, calls, idx, correctCount, totalUser }

// Enchaînement des enchères adverses, toujours actif. Sans lui, chaque tour
// de table coûtait trois clics « Révéler » qui n'apprennent rien : on les
// subissait pour revenir à sa propre enchère.
// Assez long pour lire qui vient de parler, assez court pour ne pas attendre.
const AUTO_REVEAL_MS = 700;

// Le minuteur de l'enchaînement, gardé à part pour pouvoir l'annuler : un
// questionnaire relancé ou abandonné ne doit pas voir une enchère surgir
// après coup.
let autoRevealTimer = null;

function cancelAutoReveal() {
  if (autoRevealTimer === null) return;
  clearTimeout(autoRevealTimer);
  autoRevealTimer = null;
}

// Clears any quiz in progress, e.g. when a different PBN is loaded.
function resetQuiz() {
  cancelAutoReveal();
  quiz = null;
  $("#quiz-panel").classList.add("hidden");
  $("#quiz-launch-panel").classList.remove("quiz-running");
  // En mode questionnaire, la donne garde l'état où elle est : masquée tant
  // qu'on ne l'a pas demandée, affichée si on l'a fait.
  if (!quizMode()) setDealHidden(false);
}

// Mode questionnaire : chaque nouvelle donne arrive masquée, pour enchérir sur
// une donne que l'on ne connaît pas. Mémorisé comme la langue.
const QUIZ_MODE_KEY = "bids.quizMode";

function quizMode() {
  return $("#quiz-mode").checked;
}

// Une donne vient d'arriver (tirage, exemple, fichier, choix dans un fichier,
// démarrage) : en mode questionnaire, elle se cache avant d'avoir été vue.
// Déplacer une carte n'en est pas une : on ne re-masque pas une donne que
// l'on est en train de composer.
function hideNewDealInQuizMode() {
  if (quizMode()) setDealHidden(true);
}

$("#quiz-mode").checked = readStored(QUIZ_MODE_KEY, "") === "1";
$("#quiz-mode").addEventListener("change", () => {
  saveStored(QUIZ_MODE_KEY, quizMode() ? "1" : "0");
  // Activé : la donne affichée se cache aussitôt. Désactivé : elle revient,
  // sauf pendant un questionnaire, qui la masque de toute façon.
  if (quizMode()) setDealHidden(true);
  else if (!quiz) setDealHidden(false);
});

// Le questionnaire cache les mains adverses : la donne composée au-dessus les
// montrerait toutes, et son texte PBN aussi. Elles sont masquées pendant qu'on
// enchérit, avec les commandes qui modifient la donne sous la table (voir
// style.css), et le restent une fois le questionnaire terminé. Elles
// reviennent quand on l'annule, quand on change de donne, ou à la demande.
function setDealHidden(hidden) {
  $("#input-panel").classList.toggle("quiz-running", hidden);
  // Ouvrir le texte PBN ne montrerait rien — il est masqué lui aussi — et le
  // bouton resterait enfoncé sur un panneau invisible : il s'éteint.
  $("#pbn-toggle-btn").disabled = hidden;
  renderDealMode();
  if (hidden) setPbnOpen(false);
  else scheduleAutoBid();
}

$("#quiz-show-deal-btn").addEventListener("click", () => setDealHidden(false));

// Abandonne le questionnaire en cours : il se referme, la donne composée
// revient, et la page remonte jusqu'à elle.
function cancelQuiz() {
  resetQuiz();
  setError($("#error"), "");
  revealPane($("#input-panel"));
}

$("#quiz-cancel-btn").addEventListener("click", cancelQuiz);

$("#back-to-deal-btn").addEventListener("click", () => {
  revealPane($("#input-panel"));
});

// Hides the "Résultat" auction display, e.g. when a different deal is picked.
// L'onglet du PAR ne montre que la donne dont il a reçu les mains (voir
// parSetDeal) : celle d'un résultat affiché, ou celle d'un questionnaire
// terminé. Jamais pendant l'exercice, où il dévoilerait la donne.
function setParReady(on) {
  $("#tabpanel-par").classList.toggle("par-ready", on);
}

function hideResult() {
  $("#result-panel").classList.add("hidden");
  setParReady(false);
  // La table en lecture perd le contrat et l'analyse du moteur, qui ne
  // décrivent plus la donne.
  renderReadTable();
  linkCall(null);
}

// ---------- la grille, les commentaires et la table, liés ----------
//
// Survoler ou atteindre au clavier une enchère — dans la grille ou dans la
// liste commentée — éclaire sa case, sa ligne et la main de celui qui l'a
// faite. Un clic dans la grille amène la ligne commentée à l'écran.
function linkCall(i) {
  for (const el of document.querySelectorAll(".linked")) el.classList.remove("linked");
  for (const el of document.querySelectorAll(".read-layout .hand.speaking")) {
    el.classList.remove("speaking");
  }
  const call = i == null ? null : (commentsTree.auction || [])[i];
  if (!call) return;
  $(`#auction-body td[data-i="${i}"]`)?.classList.add("linked");
  $(`#comments > li[data-i="${i}"]`)?.classList.add("linked");
  $("#hand-" + call.player)?.classList.add("speaking");
}

for (const sel of ["#auction-body", "#comments"]) {
  const root = $(sel);
  const pick = (ev) => {
    const el = ev.target.closest("[data-i]");
    // Dans la liste, l'icône d'arbre porte aussi un data-i : c'est la ligne
    // qui compte.
    const row = sel === "#comments" ? ev.target.closest("li[data-i]") : el;
    linkCall(row ? +row.dataset.i : null);
  };
  root.addEventListener("mouseover", pick);
  root.addEventListener("focusin", pick);
  root.addEventListener("mouseleave", () => linkCall(null));
  root.addEventListener("focusout", (ev) => {
    if (!root.contains(ev.relatedTarget)) linkCall(null);
  });
}

$("#auction-body").addEventListener("click", (ev) => {
  const cell = ev.target.closest("td[data-i]");
  if (!cell) return;
  $(`#comments > li[data-i="${cell.dataset.i}"]`)
    ?.scrollIntoView({ behavior: "smooth", block: "nearest" });
});

function hiddenHandHTML(seat, lang) {
  return `
    <div class="seat-name">
      <span>${esc(SEAT_LABEL[lang][seat])}</span>
      <span class="pts muted">${esc(UI_TEXT[lang].hiddenHand)}</span>
    </div>
    <div class="hidden-hand">🂠 🂠 🂠 🂠</div>`;
}

function team(seat) {
  return TEAM[seat];
}

// Turns a bid string ("3SA", "Passe", "Contre"...) into a comparable token.
function parseBidToken(text, lang) {
  if (isPass(text)) return { type: "pass" };
  if (text === (lang === "fr" ? "Contre" : "X")) return { type: "x" };
  if (text === (lang === "fr" ? "Surcontre" : "XX")) return { type: "xx" };
  const m = text.match(/^([1-7])(.+)$/);
  if (!m) return { type: "unknown" };
  const strainIdx = STRAIN_ORDER[lang].indexOf(m[2]);
  if (strainIdx === -1) return { type: "unknown" };
  return { type: "bid", level: +m[1], strainIdx, rank: (+m[1]) * 5 + strainIdx };
}

// Scans the calls made so far (in order) to find the last real bid and the
// last non-pass call, to decide which buttons are legal for `seat`.
function computeLegalCalls(calls, seat, lang) {
  let lastBid = null;
  let lastAny = null;
  for (let i = calls.length - 1; i >= 0; i--) {
    const c = calls[i];
    const p = parseBidToken(c.bid, lang);
    if (!lastAny && p.type !== "pass") lastAny = { ...p, player: c.player };
    if (!lastBid && p.type === "bid") lastBid = { ...p, player: c.player };
    if (lastAny && lastBid) break;
  }
  return {
    isBidLegal: (rank) => !lastBid || rank > lastBid.rank,
    isXLegal: !!(lastAny && lastAny.type === "bid" && team(lastAny.player) !== team(seat)),
    isXXLegal: !!(lastAny && lastAny.type === "x" && team(lastAny.player) !== team(seat)),
  };
}

// La boîte à enchères, à la manière des boîtes de club : on choisit un palier,
// puis une dénomination. Quinze boutons au lieu de trente-huit, et la boîte
// tient à l'écran sans défiler. Le palier le plus bas encore permis est
// présélectionné : l'enchère la moins chère reste à un clic. La légalité vient
// toujours de computeLegalCalls.
let bbLegal = null;
let bbLevel = null;

function lowestLegalLevel(legal) {
  for (let level = 1; level <= 7; level++) {
    if (legal.isBidLegal(level * 5 + 4)) return level;
  }
  return null;
}

function buildBiddingBoxHTML(lang, legal) {
  const t = UI_TEXT[lang];
  bbLegal = legal;
  bbLevel = lowestLegalLevel(legal);
  const levels = [];
  for (let level = 1; level <= 7; level++) {
    const disabled = legal.isBidLegal(level * 5 + 4) ? "" : "disabled";
    levels.push(`<button type="button" class="bb-btn bb-level" data-level="${level}" ` +
      `aria-pressed="${level === bbLevel}" ${disabled}>${level}</button>`);
  }
  const strains = STRAIN_ORDER[lang].map((code, idx) =>
    `<button type="button" class="bb-btn bb-strain" data-strain="${code}" data-idx="${idx}"></button>`);
  const passText = lang === "fr" ? "Passe" : "Pass";
  const xText = lang === "fr" ? "Contre" : "X";
  const xxText = lang === "fr" ? "Surcontre" : "XX";
  return `
    <div class="bb-grid">
      <div class="bb-row bb-levels" role="group" aria-label="${esc(t.bbLevel)}">${levels.join("")}</div>
      <div class="bb-row bb-strains" role="group" aria-label="${esc(t.bbStrain)}">${strains.join("")}</div>
      <div class="bb-row bb-special">
        <button type="button" class="bb-btn bb-pass" data-bid="${passText}">${esc(passText)}</button>
        <button type="button" class="bb-btn bb-x" data-bid="${xText}" ${legal.isXLegal ? "" : "disabled"}>${esc(xText)}</button>
        <button type="button" class="bb-btn bb-xx" data-bid="${xxText}" ${legal.isXXLegal ? "" : "disabled"}>${esc(xxText)}</button>
      </div>
      <p class="bb-keys muted">${esc(t.bbKeys)}</p>
    </div>`;
}

// Pose le palier choisi : les dénominations deviennent les enchères de ce
// palier, chacune permise ou non.
function renderBiddingStrains() {
  const box = $("#bidding-box");
  const lang = quiz ? quiz.lang : $("#lang").value;
  for (const b of box.querySelectorAll(".bb-level")) {
    b.setAttribute("aria-pressed", String(+b.dataset.level === bbLevel));
  }
  for (const b of box.querySelectorAll(".bb-strain")) {
    const code = b.dataset.strain;
    if (!bbLevel) {
      b.disabled = true;
      delete b.dataset.bid;
      b.innerHTML = bidHTML(`1${code}`, lang).replace(/^1/, "");
      continue;
    }
    const bid = `${bbLevel}${code}`;
    b.dataset.bid = bid;
    b.innerHTML = bidHTML(bid, lang);
    b.disabled = !bbLegal.isBidLegal(bbLevel * 5 + +b.dataset.idx);
  }
}

function renderQuizHands(revealAll) {
  for (const seat of ["N", "E", "S", "W"]) {
    $("#qhand-" + seat).innerHTML =
      revealAll || seat === quiz.seat
        ? handHTML(seat, quiz.result.hands[seat], quiz.lang)
        : hiddenHandHTML(seat, quiz.lang);
  }
}

function renderQuizAuction() {
  // La dernière enchère adverse vient d'apparaître d'elle-même : un effet la
  // signale, faute de quoi elle se glisserait dans la grille sans qu'on la voie.
  const last = quiz.calls[quiz.calls.length - 1];
  const grid = auctionGridHTML(quiz.result.dealer, quiz.calls, quiz.lang, (c) => {
    let cls = "bid-cell";
    if (c.isUser) cls += c.isCorrect ? " correct" : " incorrect";
    else if (c === last) cls += " just-revealed";
    // Mauvaise réponse : l'enchère jouée est barrée, la bonne s'affiche à
    // côté plutôt que dans le seul texte de la rétroaction en dessous.
    const content = c.expected
      ? `<s>${bidHTML(c.bid, quiz.lang)}</s> <span class="expected">${bidHTML(c.expected, quiz.lang)}</span>`
      : bidHTML(c.bid, quiz.lang);
    return `<td class="${cls}">${content}</td>`;
  });
  $("#quiz-auction-head").innerHTML = grid.headHTML;
  $("#quiz-auction-body").innerHTML = grid.bodyHTML;
}

function renderQuizStep() {
  // Un enchaînement pouvait être en attente : le clic qui nous amène ici
  // l'a devancé, il ne doit pas se déclencher une seconde fois.
  cancelAutoReveal();
  $("#quiz-feedback").classList.add("hidden");
  $("#quiz-continue-btn").classList.add("hidden");
  renderQuizAuction();

  const lang = quiz.lang;
  const t = UI_TEXT[lang];
  if (quiz.idx >= quiz.result.auction.length) {
    finishQuiz();
    return;
  }

  const entry = quiz.result.auction[quiz.idx];
  const seatName = SEAT_LABEL[lang][entry.player];
  $("#qtable-center").innerHTML = `
    <div>${withColon(esc(t.dealerCap), lang)} <b>${esc(SEAT_SHORT[lang][quiz.result.dealer])}</b></div>
    <div class="vul-line">${vulHTML(quiz.result.vulnerable, lang)}</div>
    <div class="big">${esc(SEAT_SHORT[lang][entry.player])}</div>`;

  if (entry.player === quiz.seat) {
    $("#quiz-turn").textContent = t.quizYourTurn(seatName);
    const legal = computeLegalCalls(quiz.calls, quiz.seat, lang);
    const box = $("#bidding-box");
    box.innerHTML = buildBiddingBoxHTML(lang, legal);
    renderBiddingStrains();
    box.classList.remove("hidden");
  } else {
    $("#quiz-turn").textContent = t.quizAboutToBid(seatName);
    $("#bidding-box").classList.add("hidden");
    // Aucun bouton : l'enchère des autres sièges s'affiche d'elle-même.
    const pending = quiz;
    autoRevealTimer = setTimeout(() => {
      autoRevealTimer = null;
      // Le questionnaire a pu être relancé ou abandonné pendant l'attente.
      if (quiz !== pending) return;
      revealOpponentCall();
    }, AUTO_REVEAL_MS);
  }
}

// ---------- arbre de décision ----------
//
// Le chemin que le moteur a suivi pour une enchère, test par test, évalué sur
// la main (champ « trace » de la réponse, voir engine/trace.go). Seules les
// situations déjà instrumentées en portent un : sans trace, pas de lien.

// Le lien qui déplie l'arbre, suivi de l'arbre replié.
function decisionTreeToggleHTML(entry, lang) {
  if (!entry.trace || !entry.trace.length) return "";
  const t = UI_TEXT[lang];
  return `
    <button type="button" class="link-btn tree-toggle" aria-expanded="false">${esc(t.treeShow)}</button>
    <div class="decision-tree" hidden>${decisionTreeHTML(entry, lang)}</div>`;
}

function decisionTreeHTML(entry, lang, hands = quiz && quiz.result.hands) {
  const t = UI_TEXT[lang];
  const hand = hands && hands[entry.player];
  const handLine = hand
    ? t.treeHand(
      SEAT_LABEL[lang][entry.player], hand.h_points, hand.hl_points,
      [hand.spades, hand.hearts, hand.diamonds, hand.clubs].map((s) => s.length).join("-"))
    : "";
  const q = lang === "fr" ? " ?" : "?";
  const steps = entry.trace.map((s) => {
    const cls = s.note ? "note" : s.ok ? "ok" : "ko";
    const mark = s.note ? "•" : s.ok ? "✓" : "✗";
    return `
      <li class="${cls}" style="--depth:${s.depth || 0}">
        <span class="tree-mark" aria-hidden="true">${mark}</span>
        <span class="tree-label">${esc(s.label)}</span>
        ${s.value ? `<span class="tree-value">${esc(s.value)}</span>` : ""}
      </li>`;
  }).join("");
  return `
    <div class="tree-head">${esc(t.treeWhy)} <b>${bidHTML(entry.bid, lang)}</b>${q}
      ${handLine ? `<span class="muted">(${esc(handLine)})</span>` : ""}</div>
    <ol class="tree-steps">${steps}</ol>
    <div class="tree-end">⇒ <b>${bidHTML(entry.bid, lang)}</b></div>`;
}

// Un seul gestionnaire pour tous les liens : sous le verdict comme dans le
// récapitulatif, recréés à chaque tour.
document.addEventListener("click", (ev) => {
  const btn = ev.target.closest && ev.target.closest(".tree-toggle");
  if (!btn) return;
  const tree = btn.nextElementSibling;
  const open = tree.hidden;
  tree.hidden = !open;
  btn.setAttribute("aria-expanded", String(open));
  btn.textContent = UI_TEXT[quiz ? quiz.lang : $("#lang").value][open ? "treeHide" : "treeShow"];
});

// L'icône d'arbre de la séquence commentée : un embranchement, lu comme
// « le chemin qui a mené à cette enchère ».
const TREE_ICON = `<svg viewBox="0 0 16 16" width="15" height="15" aria-hidden="true" fill="none" ` +
  `stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">` +
  `<circle cx="8" cy="3" r="1.7"/><circle cx="3.5" cy="13" r="1.7"/><circle cx="12.5" cy="13" r="1.7"/>` +
  `<path d="M8 4.7V8M3.5 11.3V8h9v3.3"/></svg>`;

// La donne affichée dans la séquence commentée, pour construire un arbre à la
// demande : il n'est calculé qu'au premier clic sur son icône.
const commentsTree = { auction: [], hands: null, lang: "fr" };

document.addEventListener("click", (ev) => {
  const btn = ev.target.closest && ev.target.closest(".tree-icon");
  if (!btn) return;
  const li = btn.closest("li");
  const t = UI_TEXT[commentsTree.lang];
  let tree = li.querySelector(".decision-tree");
  if (!tree) {
    const entry = commentsTree.auction[Number(btn.dataset.i)];
    if (!entry) return;
    tree = document.createElement("div");
    tree.className = "decision-tree";
    tree.hidden = true;
    tree.innerHTML = decisionTreeHTML(entry, commentsTree.lang, commentsTree.hands);
    li.appendChild(tree);
  }
  const open = tree.hidden;
  tree.hidden = !open;
  const label = open ? t.treeHide : t.treeShow;
  btn.setAttribute("aria-expanded", String(open));
  btn.setAttribute("aria-label", label);
  btn.dataset.tip = label;
});

function chooseBid(bidText) {
  const entry = quiz.result.auction[quiz.idx];
  const isCorrect = bidText === entry.bid;
  quiz.totalUser++;
  if (isCorrect) quiz.correctCount++;
  quiz.calls.push({
    player: entry.player,
    bid: bidText,
    isUser: true,
    isCorrect,
    expected: isCorrect ? null : entry.bid,
  });

  $("#bidding-box").classList.add("hidden");
  renderQuizAuction();

  const lang = quiz.lang;
  const t = UI_TEXT[lang];
  const fb = $("#quiz-feedback");
  fb.className = "quiz-feedback " + (isCorrect ? "correct" : "incorrect");
  const verdict = esc(isCorrect ? t.quizCorrect : t.quizWrong);
  const refLine = isCorrect
    ? ""
    : `<div>${withColon(esc(t.quizExpected), lang)} <b>${bidHTML(entry.bid, lang)}</b></div>`;
  const commentLine = entry.comment
    ? `<div class="muted">${esc(entry.comment)}</div>`
    : "";
  const treeLine = isCorrect ? "" : decisionTreeToggleHTML(entry, lang);
  fb.innerHTML = `<span class="verdict">${verdict}</span>${refLine}${commentLine}${treeLine}`;
  fb.classList.remove("hidden");

  // Le verdict se lit : l'enchaînement ne s'applique qu'aux enchères des
  // autres, jamais au sien, d'où ce bouton.
  $("#quiz-continue-btn").classList.remove("hidden");
}

// L'enchère d'un autre siège, jouée par le moteur, entre dans la séquence.
function revealOpponentCall() {
  const entry = quiz.result.auction[quiz.idx];
  quiz.calls.push({ player: entry.player, bid: entry.bid, isUser: false });
  quiz.idx++;
  renderQuizStep();
}

// « Continuer », après le verdict de sa propre enchère.
function onQuizContinue() {
  quiz.idx++;
  renderQuizStep();
}

// Les enchères ratées, relues une à une : la vôtre barrée, l'attendue, et ce
// qu'elle signifie dans le SEF. quiz.calls suit quiz.result.auction pas à pas,
// donc un même indice désigne le même tour.
function quizRecapHTML(t, lang) {
  const misses = [];
  quiz.calls.forEach((c, i) => {
    if (!c.isUser || c.isCorrect) return;
    const entry = quiz.result.auction[i];
    const comment = entry && entry.comment;
    misses.push(`
      <li>
        <span class="recap-bids"><s>${bidHTML(c.bid, lang)}</s> → <b>${bidHTML(c.expected, lang)}</b></span>
        ${comment ? `<span class="muted">${esc(comment)}</span>` : ""}
        ${entry ? decisionTreeToggleHTML(entry, lang) : ""}
      </li>`);
  });
  if (!misses.length) return `<p class="quiz-recap-none">${esc(t.quizRecapNone)}</p>`;
  return `
    <div class="quiz-recap">
      <h3>${esc(t.quizRecapTitle)}</h3>
      <ol>${misses.join("")}</ol>
    </div>`;
}

function finishQuiz() {
  renderQuizHands(true);
  // La donne composée reste masquée : les mains se lisent dans le
  // questionnaire, et le lien « Afficher la donne » la rend à la demande.
  // Plus rien à annuler : le score propose ses propres suites, et la rangée de
  // lancement revient pour changer de main.
  $("#quiz-cancel-btn").classList.add("hidden");
  $("#quiz-launch-panel").classList.remove("quiz-running");
  // Les mains sont dévoilées : les enchères commentées et le PAR de la donne
  // jouée n'ont plus rien à cacher, même si la donne composée reste masquée à
  // gauche. Leurs onglets se remplissent sans qu'on y soit emmené — le score
  // reste sous les yeux ; renderResult lance aussi le PAR si son onglet est
  // ouvert.
  renderResult(quiz.result);
  const lang = quiz.lang;
  const t = UI_TEXT[lang];
  const r = quiz.result;
  const passedOut = isPass(r.contract);
  const contractHTML = passedOut ? esc(r.contract) : bidHTML(r.contract, lang) + (r.doubled ? " X" : "");
  $("#qtable-center").innerHTML = `
    <div>${withColon(esc(t.dealerCap), lang)} <b>${esc(SEAT_SHORT[lang][r.dealer])}</b></div>
    <div class="vul-line">${vulHTML(r.vulnerable, lang)}</div>
    <div class="big">${contractHTML}</div>
    <div>${passedOut ? "" : esc(t.byWord) + " " + esc(SEAT_SHORT[lang][r.declarer])}</div>`;

  $("#quiz-turn").textContent = t.quizDone;
  $("#bidding-box").classList.add("hidden");
  $("#quiz-feedback").classList.add("hidden");
  $("#quiz-continue-btn").classList.add("hidden");

  const pct = quiz.totalUser ? Math.round((100 * quiz.correctCount) / quiz.totalUser) : 0;
  const scoreEl = $("#quiz-score");
  // Le score fermait la marche : pour rejouer, il fallait remonter la page
  // jusqu'aux commandes de la donne. Les deux suites naturelles — refaire
  // celle-ci, en tirer une autre — sont offertes ici, avec le détail.
  scoreEl.innerHTML = `
    <div class="score-big">${withColon(esc(t.quizScore), lang)} ${quiz.correctCount} / ${quiz.totalUser} (${pct}%)</div>
    <div>${withColon(esc(t.quizFinalContract), lang)} <b>${contractHTML}</b></div>
    ${quizRecapHTML(t, lang)}
    <div class="quiz-score-actions">
      <button type="button" id="quiz-replay-btn">${esc(t.quizReplay)}</button>
      <button type="button" id="quiz-new-deal-btn">${esc(t.quizNewDeal)}</button>
      <button type="button" id="quiz-show-detail-btn">${esc(t.quizShowDetail)}</button>
    </div>`;
  scoreEl.classList.remove("hidden");
  // La donne n'a pas bougé : la redemander la rejoue à l'identique.
  $("#quiz-replay-btn").addEventListener("click", startQuiz);
  $("#quiz-new-deal-btn").addEventListener("click", async () => {
    // Le tirage refuse quand les bornes ne laissent aucune donne, et le dit
    // dans #cons-error : on ne lance pas un questionnaire sur la donne d'avant.
    if (!(await drawRandomDeal())) {
      $("#cons-error").scrollIntoView({ behavior: "smooth", block: "center" });
      return;
    }
    startQuiz();
  });
  $("#quiz-show-detail-btn").addEventListener("click", () => {
    renderResult(quiz.result);
    selectTab("bids", true);
  });
}

async function startQuiz() {
  const btn = $("#quiz-btn");
  const errEl = $("#error");
  setError(errEl, "");
  $("#result-panel").classList.add("hidden");
  setParReady(false);
  const lang = $("#lang").value;
  const seat = chosenSeat();
  btn.disabled = true;
  try {
    const result = await fetchBid(lang);
    quiz = { result, seat, lang, calls: [], idx: 0, correctCount: 0, totalUser: 0 };
    $("#quiz-panel").classList.remove("hidden");
    $("#quiz-score").classList.add("hidden");
    setDealHidden(true);
    // Un seul questionnaire à l'écran : la rangée de lancement s'efface.
    $("#quiz-launch-panel").classList.add("quiz-running");
    $("#quiz-cancel-btn").classList.remove("hidden");
    renderQuizHands();
    renderQuizStep();
    // La table du questionnaire est dans le panneau de la donne, au-dessus de
    // l'onglet sur écran étroit : c'est à elle qu'on descend.
    selectTab("train");
    revealPane($("#quiz-table"));
  } catch (err) {
    setError(errEl, err.message);
    $("#quiz-panel").classList.add("hidden");
  } finally {
    btn.disabled = false;
  }
}

$("#bidding-box").addEventListener("click", (ev) => {
  const level = ev.target.closest("button[data-level]");
  if (level && !level.disabled) {
    bbLevel = +level.dataset.level;
    renderBiddingStrains();
    return;
  }
  const b = ev.target.closest("button[data-bid]");
  if (!b || b.disabled) return;
  chooseBid(b.dataset.bid);
});

// Au clavier, tant que la boîte est ouverte : un chiffre choisit le palier,
// C D H S N la dénomination (♣ ♦ ♥ ♠ SA), P passe, X contre ou surcontre.
const BB_STRAIN_KEYS = { c: 0, d: 1, h: 2, s: 3, n: 4 };
document.addEventListener("keydown", (ev) => {
  const box = $("#bidding-box");
  if (box.classList.contains("hidden") || !box.querySelector(".bb-grid")) return;
  if (ev.ctrlKey || ev.metaKey || ev.altKey) return;
  if (ev.target.closest && ev.target.closest("input, select, textarea, dialog[open]")) return;
  if ($("#tabpanel-train").hidden) return;
  const key = ev.key.toLowerCase();
  let btn = null;
  if (/^[1-7]$/.test(key)) btn = box.querySelector(`.bb-level[data-level="${key}"]`);
  else if (key in BB_STRAIN_KEYS) btn = box.querySelector(`.bb-strain[data-idx="${BB_STRAIN_KEYS[key]}"]`);
  else if (key === "p") btn = box.querySelector(".bb-pass");
  else if (key === "x") {
    const xx = box.querySelector(".bb-xx");
    btn = xx && !xx.disabled ? xx : box.querySelector(".bb-x");
  }
  if (!btn || btn.disabled) return;
  ev.preventDefault();
  btn.click();
});
$("#quiz-continue-btn").addEventListener("click", onQuizContinue);

$("#ia-health-btn").addEventListener("click", checkIaHealth);
$("#bid-btn").addEventListener("click", simulate);
$("#quiz-btn").addEventListener("click", startQuiz);

// ---------- menus ----------
//
// Un bouton .menu-trigger ouvre le .menu-pop que nomme son aria-controls. Le
// menu se referme au clic d'un de ses articles, au clic ailleurs et sur Échap,
// qui rend le focus au bouton. Les flèches parcourent ses articles. Un seul
// menu ouvert à la fois.
function menuItems(pop) {
  return [...pop.querySelectorAll(".menu-item:not(.hidden):not(:disabled)")];
}

function closeMenus(except) {
  for (const trigger of document.querySelectorAll(".menu-trigger[aria-expanded=\"true\"]")) {
    if (trigger === except) continue;
    trigger.setAttribute("aria-expanded", "false");
    document.getElementById(trigger.getAttribute("aria-controls")).hidden = true;
  }
}

function openMenu(trigger, focusFirst) {
  closeMenus(trigger);
  const pop = document.getElementById(trigger.getAttribute("aria-controls"));
  trigger.setAttribute("aria-expanded", "true");
  pop.hidden = false;
  if (focusFirst) {
    const first = menuItems(pop)[0] || pop.querySelector("select, input, button");
    if (first) first.focus();
  }
}

for (const trigger of document.querySelectorAll(".menu-trigger")) {
  trigger.addEventListener("click", (ev) => {
    if (trigger.getAttribute("aria-expanded") === "true") closeMenus();
    // Ouvert au clavier (detail === 0), le focus entre dans le menu.
    else openMenu(trigger, ev.detail === 0);
  });
  trigger.addEventListener("keydown", (ev) => {
    if (ev.key !== "ArrowDown") return;
    ev.preventDefault();
    openMenu(trigger, true);
  });
}

document.addEventListener("click", (ev) => {
  const item = ev.target.closest(".menu-item");
  // Un article agit, puis le menu se retire. Le choix d'un fichier se fait
  // dans la fenêtre du système : son <label> ferme aussi le menu.
  if (item) {
    closeMenus();
    return;
  }
  if (!ev.target.closest(".menu")) closeMenus();
});

// Le <label> de chargement n'est pas un bouton : Entrée et Espace l'ouvrent.
$(".file-btn").addEventListener("keydown", (ev) => {
  if (ev.key !== "Enter" && ev.key !== " ") return;
  ev.preventDefault();
  $("#file").click();
  closeMenus();
});

document.addEventListener("keydown", (ev) => {
  const pop = ev.target.closest && ev.target.closest(".menu-pop");
  if (ev.key === "Escape") {
    const open = document.querySelector(".menu-trigger[aria-expanded=\"true\"]");
    if (!open) return;
    closeMenus();
    open.focus();
    return;
  }
  if (!pop || (ev.key !== "ArrowDown" && ev.key !== "ArrowUp")) return;
  const items = menuItems(pop);
  const i = items.indexOf(ev.target.closest(".menu-item"));
  if (i < 0) return;
  ev.preventDefault();
  const next = (i + (ev.key === "ArrowDown" ? 1 : -1) + items.length) % items.length;
  items[next].focus();
});

// ---------- onglets ----------
//
// Le panneau de droite : Enchères et S'entraîner. Sur grand écran il est à
// côté de la table, sur écran étroit dessous — c'est alors seulement que
// revealPane fait défiler la page jusqu'à lui.
const wideLayout = window.matchMedia("(min-width: 1100px)");

function revealPane(el) {
  if (wideLayout.matches) return;
  el.scrollIntoView({ behavior: "smooth", block: "start" });
}

function selectTab(name, reveal) {
  for (const tab of document.querySelectorAll('[role="tab"]')) {
    const on = tab.dataset.tab === name;
    tab.setAttribute("aria-selected", String(on));
    tab.tabIndex = on ? 0 : -1;
    document.getElementById(tab.getAttribute("aria-controls")).hidden = !on;
  }
  if (reveal) revealPane($("#side-pane"));
  // Le PAR se calcule à l'ouverture de son onglet, une fois par donne.
  if (name === "par" && typeof parCompute === "function") parCompute();
}

for (const tab of document.querySelectorAll('[role="tab"]')) {
  // Sur écran étroit, l'onglet est une barre au bas de l'écran : le choisir
  // mène aussi au panneau.
  tab.addEventListener("click", () => selectTab(tab.dataset.tab, true));
  tab.addEventListener("keydown", (ev) => {
    const tabs = [...document.querySelectorAll('[role="tab"]')];
    const i = tabs.indexOf(tab);
    let next = null;
    if (ev.key === "ArrowRight") next = tabs[(i + 1) % tabs.length];
    else if (ev.key === "ArrowLeft") next = tabs[(i - 1 + tabs.length) % tabs.length];
    else if (ev.key === "Home") next = tabs[0];
    else if (ev.key === "End") next = tabs[tabs.length - 1];
    if (!next) return;
    ev.preventDefault();
    selectTab(next.dataset.tab);
    next.focus();
  });
}

$("#nav-deal-btn").addEventListener("click", () => revealPane($("#input-panel")));

// ---------- mode d'emploi ----------

// Les icônes du mode d'emploi sont celles des boutons eux-mêmes : dessinées
// une fois, au démarrage, depuis les mêmes constantes.
const HELP_ICONS = {
  file: IMPORT_SVG, dice: DICE_SVG, book: BOOK_SVG, save: EXPORT_SVG,
  pbn: CODE_SVG, link: LINK_SVG, gather: GATHER_SVG, deal: DEAL_SVG, bounds: BOUNDS_SVG, undo: UNDO_SVG, eraser: ERASER_SVG, play: PLAY_SVG,
  quiz: QUIZ_SVG, trash: TRASH_SVG, camera: CAMERA_SVG, edit: PENCIL_SVG,
};

function renderHelpIcons() {
  for (const el of document.querySelectorAll("[data-help-icon]")) {
    el.className = "help-icon";
    el.innerHTML = HELP_ICONS[el.dataset.helpIcon] || "";
  }
}

const helpDialog = $("#help-dialog");
function openHelp() {
  closeMenus();
  helpDialog.showModal();
  // Le texte repart du haut à chaque ouverture, pas de là où on l'a quitté.
  helpDialog.querySelector(".help-box").scrollTop = 0;
}
$("#help-btn").addEventListener("click", openHelp);
$("#help-close").addEventListener("click", () => helpDialog.close());
// Un clic sur le voile referme : il atteint le <dialog> lui-même, alors que
// tout son contenu est dans .help-box.
helpDialog.addEventListener("click", (ev) => {
  if (ev.target === helpDialog) helpDialog.close();
});

// ---------- tutoriel ----------
//
// Une étape par écran : une capture (cli/tutorial/<langue>/NN.jpg, NN étant
// le rang de l'étape) et son explication. Les captures sont prises par
// tools/tutorial/capture.js, qui rejoue les mêmes scènes dans le même ordre :
// une étape ajoutée ici demande sa scène là-bas.
const TUTORIAL_STEPS = [
  {
    fr: ["L'écran", "À gauche, la table : la donne et les boutons qui la font naître ou la modifient.\nÀ droite, trois onglets : Enchères, PAR et S'entraîner.\nEn haut, la roue dentée ouvre les réglages, l'écran ▶ ce tutoriel et ? le mode d'emploi."],
    en: ["The screen", "On the left, the table: the deal and the buttons that create or change it.\nOn the right, three tabs: Auction, Par and Practise.\nAt the top, the cog opens the settings, the ▶ screen this tutorial and ? the guide."],
  },
  {
    fr: ["Obtenir une donne", "Donne aléatoire tire une donne complète.\nLa flèche à côté propose la donne exemple ou un fichier .pbn (un fichier de tournoi aussi : un sélecteur choisit alors la donne)."],
    en: ["Getting a deal", "Random deal draws a complete deal.\nThe arrow next to it offers the example deal or a .pbn file (a tournament file too: a selector then picks the deal)."],
  },
  {
    fr: ["Partager la donne", "Copiez un lien qui contient la donne, sauvez-la dans un fichier .pbn, ou affichez son texte PBN pour le copier ou y coller une donne reçue.\nDonneur et Vulnérabilité s'imposent ou se tirent au sort."],
    en: ["Sharing the deal", "Copy a link that holds the deal, save it to a .pbn file, or show its PBN text to copy it or paste a deal you received.\nDealer and Vulnerability can be set or drawn at random."],
  },
  {
    fr: ["Modifier la donne", "Modifier la donne passe la table en édition ; Terminer revient à la lecture.\nChaque main affiche ses points d'honneur (PH) et son nombre de cartes. La corbeille renvoie ses cartes au centre, dans Cartes non affectées.\nAvec la lecture par photo (voir Réglages), l'appareil photo d'une main lit cette main, celui de Cartes non affectées les quatre mains."],
    en: ["Editing the deal", "Edit the deal switches the table to editing; Done goes back to reading.\nEach hand shows its high-card points (HCP) and card count. The bin sends its cards to the centre, into Unassigned cards.\nWith photo reading on (see Settings), a hand's camera reads that hand, and the one on Unassigned cards reads all four."],
  },
  {
    fr: ["Prendre plusieurs cartes", "Touchez (ou cliquez) les cartes l'une après l'autre dans une main : elles se remplissent de vert.\nLe symbole ♠ ♥ ♦ ♣ prend toute la couleur d'un coup.\nTouchez ensuite une carte ou le fond de la main visée : toute la sélection y part. Le bandeau du bas compte les cartes prises et permet de tout reposer."],
    en: ["Picking several cards", "Tap (or click) cards one after the other in a hand: they fill in green.\nThe ♠ ♥ ♦ ♣ symbol takes the whole suit at once.\nThen tap a card or the background of the target hand: the whole selection goes there. The bar at the bottom counts the cards and lets you put them all back."],
  },
  {
    fr: ["Glisser-déposer", "Glissez une carte d'une main à l'autre, ou vers le centre pour la retirer. Une carte prise emporte toute la sélection (+5 ici).\nLa main survolée s'entoure de vert si elle a la place, de rouge sinon.\nAu clavier et à la souris : Maj prend une plage, Ctrl (⌘) ajoute une carte d'une autre main."],
    en: ["Drag and drop", "Drag a card from one hand to another, or to the centre to take it out. A selected card carries the whole selection (+5 here).\nThe hand under the pointer gets a green outline if it has room, a red one if not.\nWith keyboard and mouse: Shift takes a range, Ctrl (⌘) adds a card from another hand."],
  },
  {
    fr: ["Bornes, Compléter, Annuler", "Bornes affiche les PH mini et maxi de chaque main : les tirages et Compléter les respectent, et le compteur rougit si la main en sort.\nCompléter distribue les cartes du centre. Annuler (Ctrl+Z) revient sur la dernière action.\nTant qu'une main est incomplète, Afficher les enchères est éteint et propose Compléter les mains."],
    en: ["Bounds, Fill, Undo", "Bounds shows each hand's min and max HCP: draws and Fill respect them, and the counter turns red when the hand is outside.\nFill deals the cards from the centre. Undo (Ctrl+Z) reverts the last action.\nWhile a hand is incomplete, Show the auction is off and offers Fill the hands."],
  },
  {
    fr: ["Les enchères", "Dès que la donne est complète, le moteur déroule les enchères du Système d'Enchères Français.\nLa séquence commentée explique chaque enchère. Survolez-en une (ou touchez-la) : sa ligne s'éclaire et la main de son auteur est cerclée sur la table.\nL'imprimante imprime la table, la séquence et le PAR."],
    en: ["The auction", "As soon as the deal is complete, the engine runs the auction in the French bidding system (SEF).\nThe commented auction explains every call. Hover one (or tap it): its line lights up and its bidder's hand is circled on the table.\nThe printer prints the table, the auction and the par."],
  },
  {
    fr: ["L'arbre de décision", "L'icône en tête d'une ligne déplie l'arbre de décision : les règles examinées par le moteur sur la main, dans l'ordre, avec ✓ ou ✗ et la valeur mesurée, jusqu'à l'enchère choisie.\nC'est le meilleur moyen de comprendre pourquoi une enchère a été faite."],
    en: ["The decision tree", "The icon at the start of a line opens the decision tree: the rules the engine checked on the hand, in order, with ✓ or ✗ and the measured value, down to the chosen call.\nIt is the best way to understand why a call was made."],
  },
  {
    fr: ["Le PAR", "L'onglet PAR donne les levées que chaque camp réalise dans chaque couleur, cartes sur table (calcul double-mort).\nSurvolez ou touchez une case : les entames qui tiennent le déclarant à ce nombre de levées, et ce que coûtent les autres."],
    en: ["The par", "The Par tab gives the tricks each side makes in each denomination, cards face up (double dummy).\nHover or tap a cell: the leads that hold declarer to that number of tricks, and what the others cost."],
  },
  {
    fr: ["S'entraîner", "Choisissez votre main (Nord, Est, Sud ou Ouest), puis la toque pour commencer le questionnaire sur la donne de la table.\nLe mode questionnaire fait arriver masquée chaque nouvelle donne : vous enchérissez sans la connaître."],
    en: ["Practise", "Choose your hand (North, East, South or West), then the cap to start the quiz on the deal on the table.\nQuiz mode brings every new deal in hidden: you bid without knowing it."],
  },
  {
    fr: ["Le questionnaire", "La table ne montre que votre main. Les enchères des autres arrivent d'elles-mêmes.\nÀ votre tour, la boîte à enchères : un palier (1 à 7), puis une couleur ou SA ; ou Passe, Contre, Surcontre. Au clavier : 1 à 7, puis C D H S N, P, X."],
    en: ["The quiz", "The table shows only your hand. The other players' calls come in by themselves.\nOn your turn, the bidding box: a level (1 to 7), then a suit or NT; or Pass, Double, Redouble. By keyboard: 1 to 7, then C D H S N, P, X."],
  },
  {
    fr: ["La réponse", "Vous voyez aussitôt si votre enchère est celle du SEF ; sinon, l'enchère attendue et son explication, et l'arbre de décision sur votre main.\nContinuer passe à la suite ; Annuler arrête le questionnaire."],
    en: ["The answer", "You see at once whether your call is the SEF one; if not, the expected call with its explanation, and the decision tree on your hand.\nContinue moves on; Cancel stops the quiz."],
  },
  {
    fr: ["Le score", "À la fin : votre score, vos erreurs avec l'enchère attendue et le contrat final.\nRejouez la donne, tirez-en une nouvelle, ou affichez le détail complet : les onglets Enchères et PAR se remplissent avec la donne jouée."],
    en: ["The score", "At the end: your score, your mistakes with the expected call, and the final contract.\nReplay the deal, draw a new one, or show the full detail: the Auction and Par tabs fill in with the deal you played."],
  },
  {
    fr: ["Réglages", "La roue dentée : langue, thème (automatique, clair ou sombre) et l'option de lecture des cartes sur une photo, qui demande un serveur IA : elle ajoute un appareil photo aux mains et à Cartes non affectées.\nLa pastille verte dit que le moteur d'enchères est prêt."],
    en: ["Settings", "The cog: language, theme (automatic, light or dark) and the option to read cards from a photo, which needs an AI server: it adds a camera to the hands and to Unassigned cards.\nThe green dot says the bidding engine is ready."],
  },
  {
    fr: ["Sur téléphone", "La table passe en une colonne et les onglets forment une barre au bas de l'écran, précédée de Donne qui remonte à la table.\nAfficher les enchères reste collé en bas pendant que vous faites défiler les mains.\nBon jeu ! Ce tutoriel se rouvre par l'écran ▶ du bandeau."],
    en: ["On a phone", "The table goes to one column and the tabs become a bar at the bottom of the screen, led by Deal, which takes you back to the table.\nShow the auction stays at the bottom while you scroll through the hands.\nEnjoy! This tutorial opens again from the ▶ screen in the top bar."],
  },
];

const tutorialDialog = $("#tutorial-dialog");
let tutorialStep = 0;

function tutorialImage(i, lang) {
  return `tutorial/${lang}/${String(i + 1).padStart(2, "0")}.jpg`;
}

function renderTutorial() {
  const lang = $("#lang").value;
  const t = UI_TEXT[lang];
  const n = TUTORIAL_STEPS.length;
  const i = tutorialStep;
  const [title, text] = TUTORIAL_STEPS[i][lang];
  const src = tutorialImage(i, lang);
  $("#tutorial-img").src = src;
  $("#tutorial-img").alt = title;
  $("#tutorial-img-link").href = src;
  $("#tutorial-img-link").title = t.tutorialZoom;
  $("#tutorial-step-title").textContent = title;
  $("#tutorial-step-text").textContent = text;
  $("#tutorial-count").textContent = `${i + 1} / ${n}`;
  $("#tutorial-bar").style.width = `${((i + 1) / n) * 100}%`;
  $("#tutorial-prev").textContent = t.tutorialPrev;
  $("#tutorial-prev").disabled = i === 0;
  $("#tutorial-next").textContent = i === n - 1 ? t.tutorialDone : t.tutorialNext;
  const dots = $("#tutorial-dots");
  if (dots.children.length !== n) {
    dots.innerHTML = TUTORIAL_STEPS.map((_, k) =>
      `<button type="button" class="tutorial-dot" data-step="${k}"></button>`).join("");
  }
  [...dots.children].forEach((dot, k) => {
    const label = `${k + 1}. ${TUTORIAL_STEPS[k][lang][0]}`;
    dot.setAttribute("aria-label", label);
    dot.title = label;
    if (k === i) dot.setAttribute("aria-current", "step");
    else dot.removeAttribute("aria-current");
  });
  // L'étape suivante se charge pendant qu'on lit celle-ci : le passage à elle
  // est alors immédiat.
  if (i + 1 < n) new Image().src = tutorialImage(i + 1, lang);
  $(".tutorial-body").scrollTop = 0;
}

function tutorialGo(i) {
  if (i < 0 || i >= TUTORIAL_STEPS.length) return;
  tutorialStep = i;
  renderTutorial();
}

function openTutorial() {
  closeMenus();
  if (helpDialog.open) helpDialog.close();
  tutorialStep = 0;
  renderTutorial();
  tutorialDialog.showModal();
  $("#tutorial-next").focus();
}

$("#tutorial-prev").addEventListener("click", () => tutorialGo(tutorialStep - 1));
$("#tutorial-next").addEventListener("click", () => {
  if (tutorialStep === TUTORIAL_STEPS.length - 1) tutorialDialog.close();
  else tutorialGo(tutorialStep + 1);
});
$("#tutorial-dots").addEventListener("click", (ev) => {
  const dot = ev.target.closest("[data-step]");
  if (dot) tutorialGo(Number(dot.dataset.step));
});
$("#tutorial-close").addEventListener("click", () => tutorialDialog.close());
$("#help-tutorial-btn").addEventListener("click", openTutorial);
$("#tutorial-btn").addEventListener("click", openTutorial);
// Un clic sur le voile referme, comme pour le mode d'emploi.
tutorialDialog.addEventListener("click", (ev) => {
  if (ev.target === tutorialDialog) tutorialDialog.close();
});
// Flèches gauche et droite, Début et Fin — sauf sur un bouton de la rangée des
// points, où elles gardent leur rôle d'aller d'un point au suivant.
tutorialDialog.addEventListener("keydown", (ev) => {
  const moves = {
    ArrowLeft: tutorialStep - 1,
    ArrowRight: tutorialStep + 1,
    Home: 0,
    End: TUTORIAL_STEPS.length - 1,
  };
  if (!(ev.key in moves) || ev.altKey || ev.ctrlKey || ev.metaKey) return;
  ev.preventDefault();
  tutorialGo(moves[ev.key]);
  const dot = ev.target.closest && ev.target.closest(".tutorial-dot");
  if (dot) $(`.tutorial-dot[data-step="${tutorialStep}"]`).focus();
});
// Au doigt : un balayage horizontal franc change d'étape. Un geste surtout
// vertical est un défilement du texte, et reste à lui.
{
  let x0 = null;
  let y0 = 0;
  let swipedAt = 0;
  const body = $(".tutorial-body");
  body.addEventListener("pointerdown", (ev) => {
    if (ev.pointerType === "mouse") return;
    x0 = ev.clientX;
    y0 = ev.clientY;
  });
  body.addEventListener("pointerup", (ev) => {
    if (x0 === null) return;
    const dx = ev.clientX - x0;
    const dy = ev.clientY - y0;
    x0 = null;
    if (Math.abs(dx) < 50 || Math.abs(dx) < 1.5 * Math.abs(dy)) return;
    swipedAt = Date.now();
    tutorialGo(tutorialStep + (dx < 0 ? 1 : -1));
  });
  body.addEventListener("pointercancel", () => { x0 = null; });
  // Un balayage qui finit sur l'image ne doit pas l'ouvrir en grand.
  $("#tutorial-img-link").addEventListener("click", (ev) => {
    if (Date.now() - swipedAt < 400) ev.preventDefault();
  });
}

// ---------- écran d'accueil ----------

// Au premier lancement, un écran propose le tutoriel, sans l'imposer. Le
// choix, quel qu'il soit — Échap compris —, est retenu : l'écran ne revient
// plus. Un navigateur qui a déjà une donne retenue n'en est pas à son premier
// lancement : l'écran est apparu après lui, et l'y montrer serait une gêne.
const WELCOME_KEY = "bids.welcomed";
const welcomeDialog = $("#welcome-dialog");
// Retenu au moment même du choix, et non sur l'événement « close » : celui-ci
// n'arrive qu'après coup, et une page rechargée entre-temps rouvrait l'écran.
function dismissWelcome() {
  saveStored(WELCOME_KEY, "1");
  welcomeDialog.close();
}
$("#welcome-skip-btn").addEventListener("click", dismissWelcome);
$("#welcome-tutorial-btn").addEventListener("click", () => {
  dismissWelcome();
  openTutorial();
});
// Échap ferme l'écran lui aussi : c'est un choix de passer le tutoriel.
welcomeDialog.addEventListener("cancel", () => saveStored(WELCOME_KEY, "1"));

// Lu avant que la donne restaurée ne soit réécrite : c'est son absence qui
// signe une première visite.
const firstLaunch = !readStored(WELCOME_KEY, "") && !readStored(LAST_DEAL_KEY, "");

// Reprend la dernière donne complète, ou la donne exemple à la première
// visite — une table vide n'apprenait rien à qui découvre l'application —,
// puis sonde le serveur.
$("#lang").value = initialLang();
applyTheme();
// Un lien de partage (#pbn=…) l'emporte sur la dernière donne retenue.
const sharedAtLoad = sharedPbnFromHash();
if (sharedAtLoad) clearShareHash();
$("#pbn").value = sharedAtLoad || readStored(LAST_DEAL_KEY, "") || EXAMPLE_PBN;
refreshDealSelector(true);
gatherMissingCards();
renderBoundsCards();
renderHelpIcons();
hideNewDealInQuizMode();
settlePbn();
applyLang();
setPbnOpen(false);
// Sonde l'état et, en mode navigateur, instancie le moteur au passage : le
// premier calcul demandé est alors immédiat. Le mode navigateur reste celui
// par défaut, un mode déjà choisi primant toujours.
checkHealth();
// applyIaFeature affiche ou masque tout le bloc IA selon l'option (OFF par
// défaut) et ne sonde le serveur IA que lorsqu'elle est active.
applyIaFeature();

// L'interface est en place : le voile de démarrage posé par index.html n'a
// plus lieu d'être. Il est retiré ici, en toute fin d'initialisation, pour ne
// découvrir qu'une page déjà remplie et non un squelette à moitié construit.
(() => {
  const boot = document.getElementById("boot-loading");
  if (boot) boot.remove();
})();

// Sur une page désormais visible, pas sous le voile de démarrage.
if (firstLaunch) welcomeDialog.showModal();
