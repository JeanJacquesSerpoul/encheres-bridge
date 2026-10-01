"use strict";

// Le moteur d'enchères Go compilé en WebAssembly (bids.wasm) : les séquences
// sont calculées dans le navigateur, sans aucun appel réseau : c'est le seul
// moteur de l'application. Produit par build-wasm.sh (point d'entrée wasm/,
// moteur engine/) et versionné dans cli/ ; s'il ne se charge pas, le client
// le dit au pied de page, et rien ne peut être calculé.
//
// Une seule instanciation pour la page, lancée dès l'ouverture par le
// selfCheck de checkHealth, et un message clair quand le module manque. Le
// solveur DDS de par.js, lui, reste paresseux : il ne sert qu'au bouton
// « Calcul du PAR », quand celui-ci est commandé des enchères on ne peut se
// passer.
//
// Chargé APRÈS wasm_exec.js, qui définit Go, et AVANT app.js, dont la fin
// appelle checkHealth() — lequel passe par window.bidsLocal. Les références à UI_TEXT ne sont résolues qu'à l'appel, donc
// bien après app.js : les scripts classiques partagent la même portée globale.

(function () {
  // Pas de `$` ici : app.js le déclare en const au niveau du script, une
  // seconde déclaration serait une SyntaxError de redéclaration.
  const q = (sel) => document.querySelector(sel);
  const WASM_URL = "bids.wasm";
  // Les règles d'enchères, lues par le moteur à chaque chargement de la page :
  // on les modifie sans recompiler bids.wasm (cli/systems/README.md). Chaque
  // système est un dossier de systems/, désigné par son identifiant ; celui
  // choisi dans les réglages est mémorisé, le SEF sinon.
  const SYSTEMS_DIR = "systems/";
  const DEFAULT_SYSTEM = "sef";
  const RULES_KEY = "bids.rules";
  // Un nom de dossier de systems/, rien d'autre : ni chemin, ni URL.
  const SYSTEM_ID = /^[A-Za-z0-9][A-Za-z0-9_-]*$/;

  function rulesUrl(id) {
    return SYSTEMS_DIR + id + "/rules.yaml";
  }

  let modulePromise = null; // le module WASM, instancié une fois pour toutes
  let loaded = false;
  let rules = null; // { id, promise } : les règles installées dans le module

  function rulesChoice() {
    try {
      let saved = localStorage.getItem(RULES_KEY);
      // Avant les dossiers par système, le choix retenu était un fichier de
      // rules/ : default.yaml (le SEF) ou new.yaml.
      if (saved && /\.ya?ml$/.test(saved)) {
        saved = saved.replace(/\.ya?ml$/, "");
        if (saved === "default") saved = DEFAULT_SYSTEM;
      }
      if (saved && SYSTEM_ID.test(saved)) return saved;
    } catch (err) {
      /* stockage inaccessible : le système par défaut */
    }
    return DEFAULT_SYSTEM;
  }

  function saveRulesChoice(id) {
    try {
      if (id === DEFAULT_SYSTEM) localStorage.removeItem(RULES_KEY);
      else localStorage.setItem(RULES_KEY, id);
    } catch (err) {
      /* rien à faire : le choix vaudra pour cette session seulement */
    }
  }

  // Le module pèse plusieurs mégaoctets : sur un lien lent son arrivée prend
  // des secondes, pendant lesquelles un bouton grisé ne dit rien. On prévient
  // l'interface, qui pose un voile d'attente (voir onBidsEngineLoad dans
  // app.js). Un simple crochet plutôt qu'une dépendance : ce fichier n'a pas
  // à connaître le DOM du client.
  function notify(state) {
    if (typeof window.onBidsEngineLoad === "function") {
      window.onBidsEngineLoad(state);
    }
  }

  function tr(key) {
    const sel = q("#lang");
    const lang = (sel && sel.value) || "fr";
    const dict = (typeof UI_TEXT === "object" && UI_TEXT[lang]) || {};
    return dict[key] || key;
  }

  // Le module prêt à calculer : instancié une fois pour toutes, avec les
  // règles du système choisi. Chaque promesse est oubliée en cas d'échec : un
  // nouveau clic réessaie au lieu de resservir l'erreur indéfiniment.
  // Le module seul, sans règles : ce qu'il faut pour lire sa version.
  function loadWasm() {
    if (typeof WebAssembly !== "object") {
      return Promise.reject(new Error(tr("wasmUnsupported")));
    }
    if (typeof Go !== "function") {
      return Promise.reject(new Error(tr("wasmMissingExec")));
    }
    if (!modulePromise) {
      modulePromise = instantiate().catch((err) => {
        modulePromise = null;
        throw err;
      });
    }
    return modulePromise;
  }

  function loadModule() {
    const id = rulesChoice();
    // Les règles se téléchargent pendant que le module s'instancie ; leur
    // erreur éventuelle n'est levée qu'après, pour ne pas laisser de rejet
    // sans gestionnaire.
    const text = rules && rules.id === id ? null : fetchRules(id).then(
      (body) => ({ body }),
      (error) => ({ error })
    );
    return loadWasm().then((api) => {
      if (!rules || rules.id !== id) {
        const promise = text.then((got) => {
          if (got.error) throw got.error;
          const res = api.loadRules(got.body);
          if (!res || !res.ok) {
            const detail = (res && res.error) || tr("wasmFailed");
            console.error(rulesUrl(id) + " :", detail);
            throw new Error(tr("rulesInvalid").replace("{file}", rulesUrl(id)) + " " + detail);
          }
        });
        rules = { id, promise };
        promise.catch(() => {
          if (rules && rules.promise === promise) rules = null;
        });
      }
      return rules.promise.then(() => {
        loaded = true;
        return api;
      });
    });
  }

  // Le voile d'attente ne suit pas le téléchargement mais l'attente : le
  // module est préchargé en silence à l'ouverture de la page, et rien ne doit
  // s'afficher tant que personne ne patiente. Un clic qui arrive avant la fin
  // du préchargement, lui, est bien une attente — d'où ce détour plutôt
  // qu'un signal posé dans loadModule.
  async function awaitModule() {
    if (loaded) return modulePromise;
    notify("start");
    try {
      return await loadModule();
    } finally {
      notify("done");
    }
  }

  // Le texte des règles. no-cache : une règle modifiée doit servir dès le
  // rechargement de la page, pas quand le cache du navigateur expire.
  async function fetchRules(id) {
    const resp = await fetch(rulesUrl(id), { cache: "no-cache" });
    if (!resp.ok) {
      throw new Error(tr("rulesMissing").replace("{file}", rulesUrl(id)) + " (HTTP " + resp.status + ")");
    }
    return resp.text();
  }

  async function instantiate() {
    const go = new Go();
    const resp = await fetch(WASM_URL);
    if (!resp.ok) {
      throw new Error(tr("wasmMissing") + " (HTTP " + resp.status + ")");
    }

    // main() pose globalThis.bidsWasm puis appelle ce crochet : c'est le seul
    // signal fiable que l'API est en place. La promesse de go.run(), elle, ne
    // se résout qu'à la sortie du programme — c'est-à-dire jamais, puisque le
    // module doit rester vivant pour que ses fonctions restent appelables.
    const ready = new Promise((resolve) => {
      window.__bidsWasmReady = resolve;
    });

    // instantiateStreaming exige un Content-Type application/wasm ; le
    // mini-serveur de run.* et GitHub Pages le posent, mais certains
    // hébergements non, d'où le repli.
    const type = resp.headers.get("content-type") || "";
    const streamable =
      typeof WebAssembly.instantiateStreaming === "function" &&
      type.includes("application/wasm");
    const result = streamable
      ? await WebAssembly.instantiateStreaming(resp, go.importObject)
      : await WebAssembly.instantiate(await resp.arrayBuffer(), go.importObject);

    // Volontairement pas attendu (voir ci-dessus) : un rejet ne peut venir que
    // d'un plantage du runtime Go, qu'on ne veut pas perdre en silence.
    go.run(result.instance).catch((err) => console.error("bids.wasm :", err));
    await ready;
    return window.bidsWasm;
  }

  // Un appel au moteur. Le code Go est synchrone et tient la boucle
  // d'événements le temps du calcul : on rend la main une fois avant, pour que
  // le bouton grisé et le message d'attente s'affichent d'abord.
  async function call(name, pbn, lang) {
    const api = await awaitModule();
    await new Promise((resolve) => setTimeout(resolve, 0));
    const res = api[name](pbn, lang || "en");
    if (!res || !res.ok) {
      throw new Error((res && res.error) || tr("wasmFailed"));
    }
    return res.json;
  }

  window.bidsLocal = {
    // Le système d'enchères, par son identifiant : le dossier de systems/
    // dont le moteur applique les règles. set() mémorise le choix ; les
    // règles sont (re)chargées au prochain appel au moteur — selfCheck,
    // d'ordinaire, que la page relance aussitôt.
    rules: {
      defaultId: DEFAULT_SYSTEM,
      current: rulesChoice,
      set(id) {
        if (!SYSTEM_ID.test(id)) throw new Error("invalid system id: " + id);
        saveRulesChoice(id);
      },
    },
    bid: (pbn, lang) => call("bid", pbn, lang).then(JSON.parse),
    bids: (pbn, lang) => call("bids", pbn, lang).then(JSON.parse),
    version: () => loadWasm().then((api) => JSON.parse(api.version)),
    // Rejoue la donne de référence de /ready : la pastille d'état dit la même
    // chose dans les deux modes. C'est aussi ce qui instancie le module à
    // l'ouverture de la page, en silence — loadModule et non awaitModule :
    // personne n'attend encore, rien à afficher.
    selfCheck: () => loadModule().then((api) => api.selfCheck().ok === true),
  };
})();
