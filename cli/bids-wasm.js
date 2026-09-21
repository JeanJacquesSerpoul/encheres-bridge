"use strict";

// Le moteur d'enchères Go compilé en WebAssembly (bids.wasm) : les séquences
// sont calculées dans le navigateur, sans aucun appel réseau, avec le même
// code et le même encodeur JSON que /bid — donc les mêmes octets. Produit par
// build-wasm.sh, servi par le serveur Go (gzip + ETag, voir gzipStatic dans
// main.go), et absent tant que ce script n'a pas été lancé : dans ce cas le
// client le dit et les modes Local et Distant continuent de fonctionner.
//
// Une seule instanciation pour la page, lancée dès l'ouverture par le
// selfCheck de checkHealth, et un message clair quand le module manque. Le
// solveur DDS de par.js, lui, reste paresseux : il ne sert qu'au bouton
// « Calcul du PAR », quand celui-ci est commandé des enchères on ne peut se
// passer.
//
// Chargé APRÈS wasm_exec.js, qui définit Go, et AVANT app.js, dont la fin
// appelle checkHealth() — lequel passe par window.bidsLocal en mode
// « navigateur ». Les références à UI_TEXT ne sont résolues qu'à l'appel, donc
// bien après app.js : les scripts classiques partagent la même portée globale.

(function () {
  // Pas de `$` ici : app.js le déclare en const au niveau du script, une
  // seconde déclaration serait une SyntaxError de redéclaration.
  const q = (sel) => document.querySelector(sel);
  const WASM_URL = "bids.wasm";

  let modulePromise = null;
  let loaded = false;

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

  // Charge et instancie le module, une fois pour toutes. La promesse est
  // oubliée en cas d'échec : un nouveau clic réessaie au lieu de resservir
  // l'erreur indéfiniment.
  function loadModule() {
    if (typeof WebAssembly !== "object") {
      return Promise.reject(new Error(tr("wasmUnsupported")));
    }
    if (typeof Go !== "function") {
      return Promise.reject(new Error(tr("wasmMissingExec")));
    }
    if (!modulePromise) {
      modulePromise = instantiate().then(
        (api) => {
          loaded = true;
          return api;
        },
        (err) => {
          modulePromise = null;
          throw err;
        }
      );
    }
    return modulePromise;
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

    // instantiateStreaming exige un Content-Type application/wasm ; le serveur
    // Go le pose, mais file:// et certains hébergements non, d'où le repli.
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
    // Le JSON brut, tel que le serveur le met sur le fil : c'est lui que l'on
    // compare octet à octet avec /bid (voir tools/wasm-parity.mjs).
    bidRaw: (pbn, lang) => call("bid", pbn, lang),
    bid: (pbn, lang) => call("bid", pbn, lang).then(JSON.parse),
    bids: (pbn, lang) => call("bids", pbn, lang).then(JSON.parse),
    version: () => loadModule().then((api) => JSON.parse(api.version)),
    // Rejoue la donne de référence de /ready : la pastille d'état dit la même
    // chose dans les deux modes. C'est aussi ce qui instancie le module à
    // l'ouverture de la page, en silence — loadModule et non awaitModule :
    // personne n'attend encore, rien à afficher.
    selfCheck: () => loadModule().then((api) => api.selfCheck().ok === true),
  };
})();
