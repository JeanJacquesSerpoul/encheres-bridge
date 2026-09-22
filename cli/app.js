"use strict";

const $ = (sel) => document.querySelector(sel);

// ---------- configuration ----------

// Serveur local (go run . / docker run, voir server_bids/README.md) et serveur
// de production (proxifié par Caddy sous /api/bidings, voir front/app/src/api.js).
//
// Le serveur local écoute par défaut sur le port 9015 ; un PORT=9415... ou un
// hôte distinct se saisit dans le champ et se mémorise (voir readLocal), comme
// l'URL distante. Celle-ci n'a pas de valeur par défaut : la figer ferait
// pointer n'importe quelle copie du client vers un serveur qui n'est pas le
// sien (voir readRemote).
const DEFAULT_LOCAL_SERVER = "http://localhost:9015";
const LOCAL_KEY = "bids.local";
const REMOTE_KEY = "bids.remote";
const MODE_KEY = "bids.mode";

// Porte de service : « ?serveur=1 » dans l'URL garde la barre du serveur
// visible même quand le moteur tourne dans la page, où elle disparaît faute
// d'avoir un serveur à choisir. Sans elle, le client de test n'aurait plus
// aucun moyen de viser la production ou un autre port. « ?serveur=0 » la
// referme, pour pouvoir l'écrire dans un signet sans se demander laquelle des
// deux formes l'emporte.
//
// Déclarée avant le premier syncServerField, quelques lignes plus bas : un
// const lu avant sa déclaration lèverait.
const SERVER_BAR_FORCED = (() => {
  try {
    const v = new URLSearchParams(location.search).get("serveur");
    return v !== null && v !== "0";
  } catch (err) {
    return false;
  }
})();

const serverModeSelect = $("#server-mode");
serverModeSelect.value = readMode();
syncServerField();
serverModeSelect.addEventListener("change", applyServerMode);

// L'URL saisie est mémorisée dans le mode courant : la retaper à chaque
// visite serait pénible et, pour le distant, le client ne peut pas la deviner.
$("#server").addEventListener("change", () => {
  if (serverModeSelect.value === "remote") saveRemote(serverURL());
  else saveLocal(serverURL());
  renderServerHint();
  checkHealth();
});

// En mode « navigateur », les enchères sont calculées sur place par
// bids-wasm.js : il n'y a aucun serveur à viser, donc aucune URL à saisir.
function wasmMode() {
  return serverModeSelect.value === "wasm";
}

// Met l'en-tête en accord avec le mode. En mode navigateur, toute la barre
// disparaît — champ d'URL comme sélecteur : il n'y a aucun serveur à viser,
// donc rien à choisir. renderHealth la ramène si le moteur fait défaut, seul
// cas où il reste quelque chose à décider.
function syncServerField() {
  const wasm = wasmMode();
  const field = $("#server");
  field.disabled = wasm;
  field.classList.toggle("hidden", wasm);
  if (!wasm) {
    field.value = serverModeSelect.value === "remote" ? readRemote() : readLocal();
  }
  // healthState n'est pas lisible ici : cette fonction tourne dès le haut du
  // script, avant sa déclaration. On masque donc sur le seul mode, et c'est
  // renderHealth qui révèle la barre en cas de panne du moteur.
  $("#bids-server-bar").classList.toggle("hidden", wasm && !SERVER_BAR_FORCED);
}

// Bascule le champ d'URL sur le mode choisi. En distant, l'URL n'est pas connue
// d'avance : on restitue celle déjà saisie, sinon on prévient qu'il faut la
// renseigner au lieu de sonder une adresse vide.
function applyServerMode() {
  const mode = serverModeSelect.value;
  saveMode(mode);
  syncServerField();
  renderServerHint();
  // Sans URL, checkHealth s'abstient de sonder et l'état retombe à
  // « inconnu » : le laisser sur son verdict précédent annoncerait en ligne un
  // serveur que l'on n'a pas contacté.
  if (mode === "remote" && !serverURL()) $("#server").focus();
  checkHealth();
}

// Le message n'a de sens qu'en distant tant que l'URL manque : ailleurs, le
// champ se suffit à lui-même, et en mode navigateur l'option choisie dit déjà
// que rien n'est interrogé. Le bandeau reste libre pour les erreurs du moteur
// (voir checkHealth).
function renderServerHint() {
  const t = UI_TEXT[$("#lang").value];
  const remote = serverModeSelect.value === "remote";
  const missing = remote && !serverURL();
  const hint = $("#server-hint");
  hint.textContent = missing ? t.serverRemoteHint : "";
  hint.classList.toggle("hidden", !hint.textContent);
  $("#server").placeholder = remote ? t.serverRemotePlaceholder : "";
  // L'option « Local » porte l'hôte réellement visé : celui qui est mémorisé,
  // ou localhost:9015 par défaut (voir readLocal). On le rafraîchit ici car il
  // est désormais modifiable.
  $("#server-mode").querySelector('option[value="local"]').textContent =
    `${t.serverLocal} (${readLocal().replace(/^https?:\/\//, "")})`;
}

// Le serveur d'IA (aiproxy) prête son /api/chat à la lecture des cartes sur une
// photo. Il est facultatif : sans lui tout le reste du client fonctionne, seuls
// les boutons photo s'éteignent (voir renderPhotoButtons). En local il écoute
// sur 9009 ; en production il est proxifié sous un préfixe propre au
// déploiement, d'où l'absence de valeur par défaut en distant — même raison
// que pour le serveur d'enchères.
const DEFAULT_LOCAL_IA = "http://localhost:9009";
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
  for (const sel of ["#ia-server-bar", "#ia-health-check",
                     "#deal-photo-btn", "#photo-status"]) {
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

// Donne affichée au démarrage, le temps d'en charger ou d'en générer une :
// une manche à SA par Stayman.
const DEFAULT_PBN = `[Dealer "S"]
[Vulnerable "All"]
[Deal "S:943.T3.Q753.Q983 J8.A54.AKT8.K765 AKT.J92.J964.J42 Q7652.KQ876.2.AT"]`;

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
    server: "Serveur",
    serverLocal: "Local",
    serverRemote: "Distant (production)",
    serverRemoteHint: "Saisissez l'URL du serveur distant.",
    errNoServer: "Aucun serveur visé : saisissez l'URL du serveur distant, ou repassez en mode Navigateur.",
    errTimeout: "Le serveur n'a pas répondu à temps. Vérifiez son URL, ou repassez en mode Navigateur.",
    errUnreachable: "Serveur injoignable. Vérifiez son URL, ou repassez en mode Navigateur.",
    errNotJson: "Réponse inattendue : ce n'est pas un serveur d'enchères. Vérifiez son URL.",
    serverRemotePlaceholder: "https://exemple.net/api/bidings",
    serverWasm: "Navigateur (hors ligne)",
    wasmMissing: "Moteur d'enchères (WASM) introuvable — lancez build-wasm.sh.",
    wasmMissingExec: "wasm_exec.js absent — lancez build-wasm.sh pour le moteur du navigateur.",
    wasmFailed: "Le moteur d'enchères n'a pas répondu.",
    wasmUnsupported: "Ce navigateur ne gère pas WebAssembly.",
    serverIa: "Serveur IA",
    serverIaPlaceholder: "https://exemple.net/aiproxy",
    iaFeature: "Serveur IA de reconnaissance des cartes",
    serverTest: "Tester",
    healthUnknown: "état inconnu",
    version: "Serveur enchères",
    versionWasm: "Moteur d'enchères (navigateur)",
    versionTitle: "Version du serveur interrogé",
    versionWasmTitle: "Version du moteur exécuté dans la page",
    versionModified: "compilé sur un dépôt modifié",
    online: "en ligne",
    offline: "injoignable",
    engineLoading: "Chargement du moteur d'enchères…",
    dealPanel: "Donne",
    fileLoad: "Charger un fichier .pbn",
    randomDeal: "Donne aléatoire",
    fileSave: "Sauver le PBN",
    fileSaveStem: "donne",
    photoDeal: "Photographier les quatre mains",
    photoOffline: "Serveur IA injoignable : reconnaissance des cartes indisponible",
    photoUnknown: "État du serveur IA inconnu : reconnaissance des cartes indisponible",
    photoNoUrl: "Renseignez l'URL du serveur IA.",
    photoBusy: "Lecture des cartes…",
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
    lockedTitle: "Donné par le fichier PBN chargé. Générez une donne pour reprendre la main.",
    optRandom: "Aléatoire",
    language: "Langue",
    theme: "Thème",
    themeTitle: "Thème du système — c'est le choix par défaut — ou thème imposé à la page.",
    themeAuto: "Automatique",
    themeLight: "Clair",
    themeDark: "Sombre",
    intro:
      "Composez une donne — l'application déroule les enchères du système " +
      "français et les commente, enchère par enchère.",
    pbnToggle: "Texte de la donne (format PBN)",
    pbnToggleTitle: "Réservé aux initiés : la donne se compose très bien sans ouvrir ce repli.",
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
    quizAboutToBid: (seat) => `${seat} va annoncer.`,
    // Même élision que photoHand : « d'Ouest », « d'Est », mais « de Nord ».
    quizReveal: (seat) =>
      `Révéler l'enchère ${/^[AEIOU]/.test(seat) ? "d'" : "de "}${seat}`,
    quizAutoReveal: "Enchaîner les enchères adverses",
    quizAutoRevealTitle: "Les enchères des trois autres sièges se dévoilent d'elles-mêmes ; la boîte à enchères revient dès que c'est à vous.",
    quizCorrect: "✓ Correct !",
    quizWrong: "✗ Différent du système SEF",
    quizExpected: "Enchère attendue",
    quizDone: "Questionnaire terminé.",
    quizScore: "Score",
    quizFinalContract: "Contrat final",
    quizShowDetail: "Afficher le détail complet",
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
    server: "Server",
    serverLocal: "Local",
    serverRemote: "Remote (production)",
    serverRemoteHint: "Enter the remote server URL.",
    errNoServer: "No server to call: enter the remote server URL, or switch back to In-browser mode.",
    errTimeout: "The server did not answer in time. Check its URL, or switch back to In-browser mode.",
    errUnreachable: "Server unreachable. Check its URL, or switch back to In-browser mode.",
    errNotJson: "Unexpected answer: this is not a bidding server. Check its URL.",
    serverRemotePlaceholder: "https://example.net/api/bidings",
    serverWasm: "In-browser (offline)",
    wasmMissing: "Bidding engine (WASM) not found — run build-wasm.sh.",
    wasmMissingExec: "wasm_exec.js is missing — run build-wasm.sh for the in-browser engine.",
    wasmFailed: "The bidding engine did not answer.",
    wasmUnsupported: "This browser does not support WebAssembly.",
    serverIa: "AI server",
    serverIaPlaceholder: "https://example.net/aiproxy",
    iaFeature: "AI card-recognition server",
    serverTest: "Test",
    healthUnknown: "unknown state",
    version: "Bidding server",
    versionWasm: "Bidding engine (in-browser)",
    versionTitle: "Version of the server being queried",
    versionWasmTitle: "Version of the engine running in the page",
    versionModified: "built from a modified tree",
    online: "online",
    offline: "unreachable",
    engineLoading: "Loading the bidding engine…",
    dealPanel: "Deal",
    fileLoad: "Load a .pbn file",
    randomDeal: "Random deal",
    fileSave: "Save the PBN",
    fileSaveStem: "deal",
    photoDeal: "Photograph the four hands",
    photoOffline: "AI server unreachable: card recognition unavailable",
    photoUnknown: "AI server state unknown: card recognition unavailable",
    photoNoUrl: "Enter the AI server URL.",
    photoBusy: "Reading the cards…",
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
    lockedTitle: "Set by the loaded PBN file. Generate a deal to take over.",
    optRandom: "Random",
    language: "Language",
    theme: "Theme",
    themeTitle: "Use the system theme — the default — or force one for the page.",
    themeAuto: "Automatic",
    themeLight: "Light",
    themeDark: "Dark",
    intro:
      "Build a deal — the application runs the French system's auction and " +
      "comments on it, call by call.",
    pbnToggle: "Deal as text (PBN format)",
    pbnToggleTitle: "For the initiated: a deal is built just fine without ever opening this.",
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
    quizAboutToBid: (seat) => `${seat} is about to bid.`,
    quizReveal: (seat) => `Reveal ${seat}'s call`,
    quizAutoReveal: "Play the opponents' calls through",
    quizAutoRevealTitle: "The other three seats reveal their calls on their own; the bidding box comes back as soon as it is your turn.",
    quizCorrect: "✓ Correct!",
    quizWrong: "✗ Not what the SEF system bids",
    quizExpected: "Expected call",
    quizDone: "Quiz complete.",
    quizScore: "Score",
    quizFinalContract: "Final contract",
    quizShowDetail: "Show full detail",
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

// URL des serveurs. Faute de valeur retenue, le serveur local retombe sur son
// port d'écoute par défaut ; le distant sur une chaîne vide, qui force la
// saisie.
function readLocal() {
  return readStored(LOCAL_KEY, DEFAULT_LOCAL_SERVER);
}

function saveLocal(url) {
  saveStored(LOCAL_KEY, url);
}

function readRemote() {
  return readStored(REMOTE_KEY, "");
}

function saveRemote(url) {
  saveStored(REMOTE_KEY, url);
}

function readIaLocal() {
  return readStored(IA_LOCAL_KEY, DEFAULT_LOCAL_IA);
}

function readIaRemote() {
  return readStored(IA_REMOTE_KEY, "");
}

// Le mode (navigateur, local ou distant) est mémorisé de la même façon ; à
// défaut, le client calcule les enchères lui-même, n'ayant alors besoin de
// rien ni de personne. Si le moteur n'est pas servi, c'est son échec de
// chargement qui le dira, et la barre du serveur repararaîtra.
function readMode() {
  const mode = readStored(MODE_KEY, "wasm");
  return mode === "remote" || mode === "local" ? mode : "wasm";
}

function saveMode(mode) {
  saveStored(MODE_KEY, mode);
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
// Le bouton du questionnaire porte deux libellés — « Continuer » après une
// réponse, « Révéler l'enchère de X » avant celle d'un adversaire — et c'est
// renderQuizStep qui les pose, pas un [data-i18n]. Il en portait pourtant un,
// et applyLang écrasait donc le second par le premier : changer de langue en
// plein questionnaire affichait « Continuer » sur un bouton qui révélait une
// enchère. Le libellé est relu ici, dans la langue du questionnaire en cours
// — lui ne change pas de langue en route (voir le gestionnaire de #lang).
function relabelQuizContinue() {
  const btn = $("#quiz-continue-btn");
  if (btn.classList.contains("hidden")) return;
  const t = UI_TEXT[quiz ? quiz.lang : $("#lang").value];
  const seat = btn.dataset.seat;
  btn.textContent = btn.dataset.mode === "reveal" && seat
    ? t.quizReveal(SEAT_LABEL[quiz ? quiz.lang : $("#lang").value][seat])
    : t.continue;
}

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
  relabelQuizContinue();
  renderDealActions();
  renderServerHint(); // messages du champ d'URL et libellé de l'option locale
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
  $("#pbn").value = text;
  // Quitter une donne chargée remet les sélecteurs sur « Aléatoire » : ce
  // qu'ils affichaient venait du fichier, ce n'était pas une préférence de
  // l'utilisateur, et la garder figerait toutes les donnes suivantes dessus.
  if (dealFromFile && !fileName) {
    $("#dealer").value = "";
    $("#vul").value = "";
  }
  dealFromFile = !!fileName;
  resetQuiz();
  hideResult();
  refreshDealSelector(true);
  $("#pbn-details").open = false;
  $("#file-name").textContent = fileName || "";
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

$("#random-btn").addEventListener("click", async () => {
  const lang = $("#lang").value;
  const errEl = $("#cons-error");
  setError(errEl, validateBounds(lang));
  if (errEl.textContent) return;
  const fini = showBusy($("#random-btn"), CONS_TEXT[lang].busyDeal);
  try {
    const pbn = await randomPBN(chosenDealer() || pickRandom(SEATS), chosenVul() || pickRandom(VULS));
    if (!pbn) {
      setError(errEl, CONS_TEXT[lang].errNoDeal);
      return;
    }
    loadPbn(pbn);
  } finally {
    fini();
  }
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
// `reveal` additionally opens the raw-text panel, focuses the textarea and
// scrolls to the block: a browser only paints a text selection when the
// field is focused, so without this the marker is invisible. Used for an
// explicit pick in the deal selector; left off while just loading a file,
// so the panel stays collapsed as before.
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
  if (reveal) {
    $("#pbn-details").open = true;
    textarea.focus({ preventScroll: true });
  }
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
    neutral: "Cartes non affectées",
    // Le clic-puis-clic existe depuis toujours (voir le gestionnaire
    // pointerup) mais n'était annoncé nulle part : sur écran tactile, viser
    // une case vide au glissé est ingrat, et personne ne devinait le repli.
    neutralHint:
      "Glissez une carte d'une main à l'autre, ou ici pour la retirer. " +
      "Au clic : touchez la carte, puis sa destination.",
    clear: "Retirer toutes les cartes",
    clearHand: "Retirer les cartes de cette main",
    photoHand: (seat) =>
      `Photographier la main ${/^[AEIOU]/.test(seat) ? "d'" : "de "}${seat}`,
    fill: "Compléter les mains",
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
    cardHeld: (card) => `${card} : pris. Allez à la destination, puis Entrée.`,
    cardDropped: (card, zone) => `${card} : déposé dans ${zone}.`,
    cardReleased: (card) => `${card} : reposé.`,
    cardsHelp:
      "Tableau des quatre mains. Tabulation pour passer d'une main à l'autre, " +
      "flèches gauche et droite pour parcourir ses cartes, haut et bas pour " +
      "changer de couleur. Entrée ou Espace prend une carte, puis la dépose " +
      "sur la main où vous êtes. Échap la repose.",
  },
  en: {
    ph: "HCP", min: "Min", max: "Max",
    reset: "Clear bounds",
    neutral: "Unassigned cards",
    neutralHint:
      "Drag a card from one hand to another, or here to take it out. " +
      "By click: tap the card, then its destination.",
    clear: "Take out every card",
    clearHand: "Take this hand's cards out",
    photoHand: (seat) => `Photograph ${seat}'s hand`,
    fill: "Fill the hands",
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
    cardHeld: (card) => `${card}: picked up. Go to the destination, then press Enter.`,
    cardDropped: (card, zone) => `${card}: dropped in ${zone}.`,
    cardReleased: (card) => `${card}: put back.`,
    cardsHelp:
      "Table of the four hands. Tab moves between hands, left and right arrows " +
      "walk through a hand's cards, up and down change suit. Enter or Space " +
      "picks a card up, then drops it on the hand you are on. Escape puts it back.",
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
// Carte désignée par un simple clic, en attente d'une zone de destination
// (voir le glisser-déposer plus bas).
let selectedCard = null;

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
  selectedCard = null;
  renderBoundsCards();
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

// Applies a change made in the constraints panel: PBN text, display, and any
// result computed from the previous cards, which no longer describes the deal.
function commitZones() {
  writeZonesToPbn();
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
            const held = sameCard(selectedCard, card);
            const sel = held ? " picked" : "";
            const tab = roving && roving.suit === suit && roving.rank === rank ? 0 : -1;
            return `<span class="card${cls}${sel}" data-owner="${zone}" data-suit="${suit}" ` +
              `data-rank="${rank}" role="button" tabindex="${tab}" ` +
              `aria-pressed="${held}" aria-label="${esc(cardLabel(card, lang))}">` +
              `${rankHTML(rank, lang)}</span>`;
          })
          .join("")
      : '<span class="void" aria-hidden="true">—</span>';
    return `<div class="suitline"><span class="suitsym${cls}" aria-hidden="true">${SUIT_SYMBOLS[suit]}</span>${cards}</div>`;
  }).join("");
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

// The centre of the table holds the cards that belong to no hand yet.
function neutralZoneHTML(lang) {
  const t = CONS_TEXT[lang];
  const count = zoneCount(UNASSIGNED);
  const body = count
    ? `<div class="cons-cards">${zoneCardsHTML(UNASSIGNED, lang)}</div>`
    : `<div class="neutral-hint" ${emptyDropHTML(UNASSIGNED, lang)}>${esc(t.neutralHint)}</div>`;
  return `
    <div class="cons-head">
      <span class="cons-seat">${esc(t.neutral)}</span>
      <span class="cons-chips"><span class="cons-chip">${count}</span></span>
    </div>
    ${body}`;
}

// Full redraw of the four bound cards + the neutral zone. Rebuilds the inputs,
// so only call it when the deal, the language or the bounds state changes as a
// whole — not on every keystroke (see refreshBoundsFlags).
// Un bouton sans texte : le dessin dedans, le nom dans l'infobulle et dans
// l'étiquette que lit un lecteur d'écran. Les trois viennent ensemble, sinon
// l'un des trois finit par manquer.
function setCommandButton(sel, svg, name) {
  const btn = $(sel);
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
  const load = $(".file-btn");
  load.dataset.tip = t.fileLoad;
  setCommandButton("#random-btn", DICE_SVG, t.randomDeal);
  setCommandButton("#save-btn", EXPORT_SVG, t.fileSave);
}

function renderBoundsCards() {
  const lang = $("#lang").value;
  const t = CONS_TEXT[lang];
  setCommandButton("#cons-reset-btn", ERASER_SVG, t.reset);
  setCommandButton("#cons-clear-btn", GATHER_SVG, t.clear);
  setCommandButton("#cons-fill-btn", DEAL_SVG, t.fill);
  setCommandButton("#bid-btn", PLAY_SVG, UI_TEXT[lang].runAuction);
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
  restoreCardFocus();
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

$("#cons-reset-btn").addEventListener("click", () => {
  for (const seat of SEATS) hcpBounds[seat] = { min: null, max: null };
  setError($("#cons-error"), "");
  renderBoundsCards();
});

// ---------- moving cards between hands and the neutral zone ----------

// Distance (px) à parcourir avant qu'un appui devienne un glisser : en deçà,
// on considère que l'utilisateur a simplement cliqué la carte.
const DRAG_THRESHOLD = 5;

let dragState = null; // { card, el, x0, y0, moved, pointerId, ghost }

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

// Whether `zone` can take one more card: the neutral zone is unbounded, a
// hand stops at 13.
function zoneAccepts(zone, card) {
  if (zone === card.zone) return false;
  return zone === UNASSIGNED || zoneCount(zone) < HAND_SIZE;
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

// Prendre une carte, ou déposer celle qu'on tient sur la main de celle-ci.
// C'est ce que fait un clic simple, et c'est ce que fait Entrée : le geste est
// écrit une fois pour les deux, sans quoi les deux finiraient par diverger.
// Dépose la carte tenue dans `zone`. `fallback` est la carte à refocaliser si
// le dépôt est refusé — celle sur laquelle on se trouvait.
//
// Un refus — la main est pleine — laisse la carte dans la main : on la tient
// toujours, et on peut viser ailleurs sans la reprendre. Elle était reposée
// jusqu'ici, y compris à la souris, ce qui obligeait à recommencer.
function dropHeld(zone, fallback) {
  const lang = $("#lang").value;
  const t = CONS_TEXT[lang];
  const held = selectedCard;
  const name = cardLabel(held, lang);
  const zoneName = zone === UNASSIGNED ? t.neutral : SEAT_LABEL[lang][zone];
  if (!zoneAccepts(zone, held)) {
    // setError annonce lui-même : un announce() de plus ferait dire deux fois
    // la même phrase, une fois poliment et une fois en interrompant.
    setError($("#cons-error"), t.errFull(zoneName));
    refocusCard = fallback;
    renderBoundsCards();
    return;
  }
  selectedCard = null;
  dropCard(held, zone);
  refocusCard = { zone, suit: held.suit, rank: held.rank };
  rovingByZone[zone] = { suit: held.suit, rank: held.rank };
  announce(t.cardDropped(name, zoneName));
  renderBoundsCards();
}

function activateCard(card) {
  const lang = $("#lang").value;
  const t = CONS_TEXT[lang];
  if (selectedCard && selectedCard.zone !== card.zone) {
    dropHeld(card.zone, card);
    return;
  }
  const wasHeld = sameCard(selectedCard, card);
  selectedCard = wasHeld ? null : card;
  refocusCard = card;
  rovingByZone[card.zone] = { suit: card.suit, rank: card.rank };
  announce(wasHeld ? t.cardReleased(cardLabel(card, lang)) : t.cardHeld(cardLabel(card, lang)));
  renderBoundsCards();
}

// Déposer la carte tenue dans une zone visée directement — le fond d'une main,
// ou une main vide, qui n'a aucune carte sur laquelle cliquer.
function activateZone(zone) {
  if (!selectedCard) return;
  dropHeld(zone, selectedCard);
}

function dropCard(card, zone) {
  if (zone === card.zone) {
    renderBoundsCards();
    return;
  }
  const lang = $("#lang").value;
  if (!zoneAccepts(zone, card)) {
    setError($("#cons-error"), CONS_TEXT[lang].errFull(SEAT_LABEL[lang][zone]));
    renderBoundsCards();
    return;
  }
  setError($("#cons-error"), validateBounds(lang));
  moveCard(card, zone);
  commitZones();
}

function startDrag(ev) {
  dragState.moved = true;
  selectedCard = null;
  const card = dragState.card;
  const ghost = document.createElement("div");
  const cls = RED_SUITS.has(card.suit) ? " red" : "";
  ghost.className = "card-ghost";
  ghost.innerHTML =
    `<span class="suitsym${cls}">${SUIT_SYMBOLS[card.suit]}</span>` +
    `<span class="${cls.trim()}">${rankHTML(card.rank, $("#lang").value)}</span>`;
  document.body.appendChild(ghost);
  dragState.ghost = ghost;
  dragState.el.classList.add("dragging");
}

function endDrag() {
  if (dragState && dragState.ghost) dragState.ghost.remove();
  if (dragState && dragState.el) dragState.el.classList.remove("dragging");
  clearDropHints();
  dragState = null;
}

$("#constraints").addEventListener("pointerdown", (ev) => {
  if (ev.pointerType === "mouse" && ev.button !== 0) return;
  if (ev.target.closest("input, button, select")) return;
  const cardEl = ev.target.closest(".card");
  const zoneEl = ev.target.closest("[data-zone]");
  if (!cardEl && !zoneEl) return;
  dragState = {
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
  if (zone && zone !== dragState.card.zone) {
    const target = document.querySelector(`[data-zone="${zone}"]`);
    target.classList.add(zoneAccepts(zone, dragState.card) ? "drop-ok" : "drop-no");
  }
  ev.preventDefault();
}, { passive: false });

window.addEventListener("pointerup", (ev) => {
  if (!dragState || ev.pointerId !== dragState.pointerId) return;
  const st = dragState;
  endDrag();

  if (st.moved) {
    const zone = zoneUnder(ev.clientX, ev.clientY);
    if (zone) dropCard(st.card, zone);
    else renderBoundsCards();
    return;
  }

  // Simple click: pick a card up, then click its destination. Clicking a card
  // of another zone drops the held one there — handy on touch screens, where
  // an empty spot can be hard to aim at. Le clavier passe par les mêmes deux
  // fonctions : un seul modèle pour les deux entrées.
  if (st.card) activateCard(st.card);
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
  if (ev.key === "Escape" && selectedCard) {
    const lang = $("#lang").value;
    const card = selectedCard;
    selectedCard = null;
    refocusCard = card;
    announce(CONS_TEXT[lang].cardReleased(cardLabel(card, lang)));
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
      activateCard(card);
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
  setError(errEl, validateBounds(lang));
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
  selectedCard = null;
  commitZones();
});

// La poubelle d'une main fait de même pour ce seul siège : ses cartes restent
// dans le jeu, en attente d'une autre main ou de « Compléter les mains ».
$("#constraints").addEventListener("click", (ev) => {
  const btn = ev.target.closest("[data-clear-hand]");
  if (!btn) return;
  emptyHandToNeutral(btn.dataset.clearHand);
  setError($("#cons-error"), "");
  selectedCard = null;
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
  try {
    const resp = await fetchWithTimeout(iaURL() + "/api/chat", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        text: target === "deal" ? PROMPT_DEAL : PROMPT_HAND,
        model: IA_MODEL,
        image,
      }),
    }, TIMEOUT_VISION);
    let body;
    try {
      body = await resp.json();
    } catch (err) {
      throw new Error(t.photoBadAnswer);
    }
    if (!resp.ok || !body.success) throw new Error(body.error || t.photoFailed);
    const answer = parseIaJSON(body.response);
    const report = target === "deal"
      ? applyPhotoDeal(answer)
      : applyPhotoHand(target, answer);
    showPhotoStatus(photoReport(report), report.read === 0);
  } catch (err) {
    showPhotoStatus(err.message, true);
  } finally {
    photoBusy = false;
    renderPhotoButtons();
  }
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
  selectedCard = null;
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
  selectedCard = null;
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
  if (!$("#result-panel").classList.contains("hidden")) simulate();
});

// Builds a W/N/E/S auction grid (head + body rows) from a list of calls,
// padding the first row up to the dealer's column. `cellRenderer(call)`
// returns the <td> HTML for a played call.
function auctionGridHTML(dealer, calls, lang, cellRenderer) {
  const columns = ["W", "N", "E", "S"];
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

  // Center of the table + summary chips.
  const passedOut = isPass(r.contract);
  const contractHTML = passedOut
    ? esc(r.contract)
    : bidHTML(r.contract, lang) + (r.doubled ? " X" : "");
  $("#table-center").innerHTML = `
    <div>${withColon(lang === "fr" ? "Donneur" : "Dealer", lang)} <b>${esc(SEAT_SHORT[lang][r.dealer])}</b></div>
    <div class="vul-line">${vulHTML(r.vulnerable, lang)}</div>
    <div class="big">${contractHTML}</div>
    <div>${passedOut ? "" : (lang === "fr" ? "par " : "by ") + esc(SEAT_SHORT[lang][r.declarer])}</div>`;

  $("#summary").innerHTML = `
    <span class="chip contract">${withColon(lang === "fr" ? "Contrat" : "Contract", lang)} <b>${contractHTML}</b></span>
    <span class="chip">${withColon(lang === "fr" ? "Déclarant" : "Declarer", lang)} <b>${passedOut ? "—" : esc(SEAT_SHORT[lang][r.declarer])}</b></span>
    <span class="chip">${withColon(lang === "fr" ? "Donneur" : "Dealer", lang)} <b>${esc(SEAT_SHORT[lang][r.dealer])}</b></span>
    <span class="chip">${vulHTML(r.vulnerable, lang)}</span>
    <span class="chip">${withColon(lang === "fr" ? "Contré" : "Doubled", lang)} <b>${r.doubled ? (lang === "fr" ? "oui" : "yes") : (lang === "fr" ? "non" : "no")}</b></span>`;

  // Auction grid: columns W N E S, first row padded up to the dealer.
  // Une enchère commentée porte son commentaire en infobulle, au dessin de
  // celle du PAR (auctionTip, plus bas).
  auctionTip.hide();
  auctionTip.calls = r.auction;
  auctionTip.lang = lang;
  const grid = auctionGridHTML(r.dealer, r.auction, lang, (a) => {
    if (!a.comment) return `<td class="bid-cell">${bidHTML(a.bid, lang)}</td>`;
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
  $("#comments").innerHTML = r.auction
    .map((a) => {
      const head = `<span class="who">${esc(SEAT_SHORT[lang][a.player])}</span> - ` +
        bidHTML(a.bid, lang);
      return a.comment && !isPass(a.bid)
        ? `<li>${withColon(head, lang)} ${esc(a.comment)}</li>`
        : `<li class="silent">${head}</li>`;
    })
    .join("") || `<li class="muted">${lang === "fr" ? "aucune" : "none"}</li>`;

  // Tableau du « PAR » (levées double-mort) : remis à zéro pour la donne qu'on
  // vient d'afficher, puis calculé à la demande par le bouton (voir par.js).
  if (typeof parSetDeal === "function") parSetDeal(r.hands, lang);
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

// ---------- server calls ----------

function serverURL() {
  return $("#server").value.trim().replace(/\/+$/, "");
}

// Un fetch qui renonce. Sans cela, un serveur qui accepte la connexion puis se
// tait laisse le bouton grisé indéfiniment : rien ne revient, ni réponse ni
// erreur, et l'application reste en attente jusqu'au rechargement de la page.
//
// Les délais diffèrent selon ce qu'on attend : une sonde doit répondre tout de
// suite, un calcul d'enchères prend le temps qu'il faut, et le modèle de vision
// du serveur IA met des dizaines de secondes à lire une photo.
const TIMEOUT_PROBE = 8000;
const TIMEOUT_BID = 20000;
const TIMEOUT_VISION = 90000;

async function fetchWithTimeout(url, options, ms = TIMEOUT_PROBE) {
  const ctrl = new AbortController();
  const timer = setTimeout(() => ctrl.abort(), ms);
  try {
    return await fetch(url, { ...options, signal: ctrl.signal });
  } catch (err) {
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
  }
}

// Dernier état connu du serveur, gardé pour le réafficher tel quel quand la
// langue change, sans le redemander.
let healthState = null; // "online", "offline", ou null tant qu'on ne sait pas

function renderHealth() {
  const t = UI_TEXT[$("#lang").value];
  const dot = $("#health-dot");
  dot.className =
    healthState === "online" ? "dot ok" : healthState === "offline" ? "dot ko" : "dot";
  dot.title = healthState ? t[healthState] : t.healthUnknown;
  $("#health-text").textContent = healthState ? t[healthState] : "";
  // Moteur en panne (module absent, navigateur sans WebAssembly...) : la barre
  // revient, faute de quoi le client resterait bloqué sur un mode qui ne
  // calcule rien, sans aucun moyen de viser un serveur.
  if (wasmMode() && healthState === "offline") {
    $("#bids-server-bar").classList.remove("hidden");
  }
}

// Version du serveur, telle que /version la renvoie. null tant qu'on ne l'a
// pas obtenue : le pied de page nomme alors le serveur sans lui prêter une
// révision qu'il ne connaît pas.
let serverVersion = null;

function renderVersion() {
  const el = $("#app-version");
  const t = UI_TEXT[$("#lang").value];
  // En mode navigateur, nommer « serveur » ce qui tourne dans la page ferait
  // croire à un serveur que l'on ne contacte pas.
  const label = wasmMode() ? t.versionWasm : t.version;
  // Le nom du serveur reste affiché même sans version : il dit à quelle ligne
  // du pied de page appartient la pastille d'état (l'autre étant celle de l'IA).
  if (!serverVersion) {
    el.textContent = label;
    el.removeAttribute("title");
    return;
  }
  // L'étoile signale un binaire compilé sur un dépôt modifié : la révision
  // seule prétendrait alors correspondre à un commit qu'il ne reflète pas.
  el.textContent = `${label} ${serverVersion.revision}${serverVersion.modified ? " *" : ""}`;
  const parts = [wasmMode() ? t.versionWasmTitle : t.versionTitle];
  if (serverVersion.time) parts.push(serverVersion.time);
  if (serverVersion.go) parts.push(serverVersion.go);
  if (serverVersion.modified) parts.push(t.versionModified);
  el.title = parts.join(" · ");
}

async function fetchVersion() {
  try {
    if (wasmMode()) {
      const body = await bidsLocal.version();
      serverVersion = body && body.revision ? body : null;
    } else {
      const resp = await fetchWithTimeout(serverURL() + "/version", {}, TIMEOUT_PROBE);
      const body = await resp.json();
      serverVersion = resp.ok && body.revision ? body : null;
    }
  } catch (err) {
    serverVersion = null;
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

async function checkHealth() {
  healthState = null;
  renderHealth();
  // En mode navigateur, l'équivalent de /ready : le moteur rejoue sa donne de
  // référence. Cela instancie le module au passage, donc dès l'ouverture de la
  // page — le premier calcul est ainsi immédiat.
  //
  // Ce préchargement a un temps été différé au premier calcul, quand
  // l'hébergement mettait ~9 s à répondre à la moindre requête. Sur GitHub
  // Pages, bids.wasm arrive compressé (1,3 Mo) en moins d'une seconde, hors du
  // chemin critique de l'affichage : le coût ne justifie plus l'attente.
  if (wasmMode()) {
    $("#health-text").textContent = "…";
    try {
      healthState = (await bidsLocal.selfCheck()) ? "online" : "offline";
      renderServerHint();
    } catch (err) {
      healthState = "offline";
      const hint = $("#server-hint");
      hint.textContent = err.message;
      hint.classList.remove("hidden");
    }
    renderHealth();
    fetchVersion();
    return;
  }
  // Sans URL, fetch() résoudrait « /health » contre l'origine de la page, donc
  // contre le serveur local : le distant serait annoncé en ligne sans avoir
  // jamais été contacté. On s'en tient à l'état inconnu.
  if (!serverURL()) {
    renderServerHint();
    return;
  }
  $("#health-text").textContent = "…";
  try {
    const resp = await fetchWithTimeout(serverURL() + "/health", {}, TIMEOUT_PROBE);
    const body = await resp.json();
    healthState = resp.ok && body.status === "ok" ? "online" : "offline";
  } catch (err) {
    healthState = "offline";
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

// Le mode par défaut étant « navigateur », on a longtemps sondé bids.wasm en
// HEAD à l'ouverture pour retomber sur le serveur local quand le moteur n'est
// pas servi (ouvert en file://, ou binaire compilé sans build-wasm.sh).
//
// Cette sonde a été retirée : sur un hébergement statique lent, chaque requête
// coûte sa latence complète — mesurée à plusieurs secondes — et celle-ci ne
// servait qu'à choisir un mode par avance. Le repli se fait désormais quand il
// se justifie vraiment, c'est-à-dire si le moteur refuse de se charger : le
// message d'erreur s'affiche et renderHealth fait réapparaître la barre du
// serveur, qui laisse en viser un.

// Reads/validates the PBN textarea and asks the server for the full
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
  // Seul le transport change : les vérifications ci-dessus valent pour les
  // deux modes, et le moteur WebAssembly rend exactement le même JSON que /bid.
  if (wasmMode()) {
    return bidsLocal.bid(pbn, lang);
  }
  // Sans URL, fetch() résoudrait « /bid » contre l'origine de la page : la
  // réponse serait le HTML d'un 404, et resp.json() échouerait sur un message
  // du moteur JavaScript — « Unexpected token '<' » — que personne ne peut
  // comprendre. checkHealth a ce garde-fou depuis toujours ; il manquait ici.
  if (!serverURL()) {
    throw new Error(UI_TEXT[lang].errNoServer);
  }
  const fd = new FormData();
  fd.append("pbn", new Blob([pbn], { type: "text/plain" }), "deal.pbn");
  const resp = await fetchWithTimeout(`${serverURL()}/bid?lang=${lang}`, {
    method: "POST",
    body: fd,
  }, TIMEOUT_BID);
  let body;
  try {
    body = await resp.json();
  } catch (err) {
    // L'URL répond, mais pas en JSON : ce n'est pas un serveur d'enchères.
    // Le 404 d'un hébergeur statique donnait « Unexpected token '<' ».
    throw new Error(UI_TEXT[lang].errNotJson);
  }
  if (!resp.ok) {
    throw new Error(body.error || `HTTP ${resp.status}`);
  }
  return body;
}

async function simulate() {
  const btn = $("#bid-btn");
  // Le message va dans #cons-error, qui est sur la même ligne que le bouton.
  // Il partait dans #error, une rangée plus bas, à côté du questionnaire :
  // une donne incomplète produisait un refus qu'on ne voyait pas, et le clic
  // paraissait sans effet.
  const errEl = $("#cons-error");
  setError(errEl, "");
  resetQuiz();
  const lang = $("#lang").value;
  btn.disabled = true;
  try {
    const body = await fetchBid(lang);
    renderResult(body);
    // Le panneau s'ouvre sous le pli : sans cela, rien ne bouge à l'écran et
    // le calcul semble n'avoir rien donné. Même geste qu'en fin de
    // questionnaire et au lancement de celui-ci.
    $("#result-panel").scrollIntoView({ behavior: "smooth", block: "start" });
  } catch (err) {
    setError(errEl, err.message);
    $("#result-panel").classList.add("hidden");
  } finally {
    btn.disabled = false;
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

// Enchaînement des enchères adverses. Sans lui, chaque tour de table coûte
// trois clics « Révéler » qui n'apprennent rien : on les subit pour revenir à
// sa propre enchère. Le réglage est mémorisé comme la langue, et décoché par
// défaut — le rythme d'origine reste celui de qui n'y touche pas.
const AUTO_REVEAL_KEY = "bids.quizAuto";
// Assez long pour lire qui vient de parler, assez court pour ne pas attendre.
const AUTO_REVEAL_MS = 700;
const autoRevealToggle = $("#quiz-auto");
autoRevealToggle.checked = readStored(AUTO_REVEAL_KEY, "") === "1";
autoRevealToggle.addEventListener("change", () => {
  saveStored(AUTO_REVEAL_KEY, autoRevealToggle.checked ? "1" : "0");
  // Cocher la case en plein questionnaire doit valoir tout de suite : si l'on
  // attend devant un « Révéler », l'enchère s'enchaîne sans le clic.
  if (quiz && $("#quiz-continue-btn").dataset.mode === "reveal") renderQuizStep();
});

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
}

// Hides the "Résultat" auction display, e.g. when a different deal is picked.
function hideResult() {
  $("#result-panel").classList.add("hidden");
}

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

function buildBiddingBoxHTML(lang, legal) {
  const strains = STRAIN_ORDER[lang];
  const rows = [];
  for (let level = 1; level <= 7; level++) {
    const cells = strains.map((code, strainIdx) => {
      const rank = level * 5 + strainIdx;
      const disabled = legal.isBidLegal(rank) ? "" : "disabled";
      return `<button type="button" class="bb-btn" data-bid="${level}${code}" ${disabled}>${bidHTML(`${level}${code}`, lang)}</button>`;
    });
    rows.push(`<div class="bb-row">${cells.join("")}</div>`);
  }
  const passText = lang === "fr" ? "Passe" : "Pass";
  const xText = lang === "fr" ? "Contre" : "X";
  const xxText = lang === "fr" ? "Surcontre" : "XX";
  rows.push(`
    <div class="bb-row bb-special">
      <button type="button" class="bb-btn bb-pass" data-bid="${passText}">${esc(passText)}</button>
      <button type="button" class="bb-btn bb-x" data-bid="${xText}" ${legal.isXLegal ? "" : "disabled"}>${esc(xText)}</button>
      <button type="button" class="bb-btn bb-xx" data-bid="${xxText}" ${legal.isXXLegal ? "" : "disabled"}>${esc(xxText)}</button>
    </div>`);
  return `<div class="bb-grid">${rows.join("")}</div>`;
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
  const grid = auctionGridHTML(quiz.result.dealer, quiz.calls, quiz.lang, (c) => {
    let cls = "bid-cell";
    if (c.isUser) cls += c.isCorrect ? " correct" : " incorrect";
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
    box.classList.remove("hidden");
  } else {
    $("#quiz-turn").textContent = t.quizAboutToBid(seatName);
    $("#bidding-box").classList.add("hidden");
    const btn = $("#quiz-continue-btn");
    btn.textContent = t.quizReveal(seatName);
    btn.dataset.mode = "reveal";
    // Le siège est gardé sur le bouton : applyLang en a besoin pour réécrire
    // ce libellé-là, qu'aucun [data-i18n] ne porte.
    btn.dataset.seat = entry.player;
    btn.classList.remove("hidden");
    // Le bouton reste offert même quand l'enchaînement est coché : qui ne
    // veut pas attendre les 700 ms clique, et renderQuizStep annule alors le
    // minuteur en tête de son prochain passage.
    if (autoRevealToggle.checked) {
      const pending = quiz;
      autoRevealTimer = setTimeout(() => {
        autoRevealTimer = null;
        // Le questionnaire a pu être relancé ou abandonné pendant l'attente.
        if (quiz !== pending) return;
        onQuizContinue();
      }, AUTO_REVEAL_MS);
    }
  }
}

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
  fb.innerHTML = `<span class="verdict">${verdict}</span>${refLine}${commentLine}`;
  fb.classList.remove("hidden");

  const btn = $("#quiz-continue-btn");
  btn.textContent = t.continue;
  btn.dataset.mode = "next";
  // Le verdict se lit : l'enchaînement ne s'applique qu'aux enchères des
  // autres, jamais au sien.
  delete btn.dataset.seat;
  btn.classList.remove("hidden");
}

function onQuizContinue() {
  const btn = $("#quiz-continue-btn");
  if (btn.dataset.mode === "reveal") {
    const entry = quiz.result.auction[quiz.idx];
    quiz.calls.push({ player: entry.player, bid: entry.bid, isUser: false });
  }
  quiz.idx++;
  renderQuizStep();
}

function finishQuiz() {
  renderQuizHands(true);
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
    <div class="quiz-score-actions">
      <button type="button" id="quiz-replay-btn">${esc(t.quizReplay)}</button>
      <button type="button" id="quiz-new-deal-btn">${esc(t.quizNewDeal)}</button>
      <button type="button" id="quiz-show-detail-btn">${esc(t.quizShowDetail)}</button>
    </div>`;
  scoreEl.classList.remove("hidden");
  // La donne n'a pas bougé : la redemander la rejoue à l'identique.
  $("#quiz-replay-btn").addEventListener("click", startQuiz);
  $("#quiz-new-deal-btn").addEventListener("click", () => {
    $("#random-btn").click();
    // Le tirage refuse quand les bornes ne laissent aucune donne, et le dit
    // dans #cons-error : on ne lance pas un questionnaire sur la donne d'avant.
    if ($("#cons-error").textContent) {
      $("#cons-error").scrollIntoView({ behavior: "smooth", block: "center" });
      return;
    }
    startQuiz();
  });
  $("#quiz-show-detail-btn").addEventListener("click", () => {
    renderResult(quiz.result);
    $("#result-panel").scrollIntoView({ behavior: "smooth", block: "start" });
  });
}

async function startQuiz() {
  const btn = $("#quiz-btn");
  const errEl = $("#error");
  setError(errEl, "");
  $("#result-panel").classList.add("hidden");
  const lang = $("#lang").value;
  const seat = chosenSeat();
  btn.disabled = true;
  try {
    const result = await fetchBid(lang);
    quiz = { result, seat, lang, calls: [], idx: 0, correctCount: 0, totalUser: 0 };
    $("#quiz-panel").classList.remove("hidden");
    $("#quiz-score").classList.add("hidden");
    renderQuizHands();
    renderQuizStep();
    $("#quiz-panel").scrollIntoView({ behavior: "smooth", block: "start" });
  } catch (err) {
    setError(errEl, err.message);
    $("#quiz-panel").classList.add("hidden");
  } finally {
    btn.disabled = false;
  }
}

$("#bidding-box").addEventListener("click", (ev) => {
  const b = ev.target.closest("button[data-bid]");
  if (!b || b.disabled) return;
  chooseBid(b.dataset.bid);
});
$("#quiz-continue-btn").addEventListener("click", onQuizContinue);

$("#health-btn").addEventListener("click", checkHealth);
$("#ia-health-btn").addEventListener("click", checkIaHealth);
$("#bid-btn").addEventListener("click", simulate);
$("#quiz-btn").addEventListener("click", startQuiz);

// Prefill with the default deal and ping the server on load.
$("#lang").value = initialLang();
applyTheme();
$("#pbn").value = DEFAULT_PBN;
refreshDealSelector(true);
applyLang();
$("#pbn-details").open = false;
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
