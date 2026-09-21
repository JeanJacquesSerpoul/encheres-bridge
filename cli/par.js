"use strict";

// « PAR » de la donne affichée : les levées double-mort (double dummy) de
// chaque camp dans chaque couleur, calculées par le solveur DDS compilé en
// WebAssembly. Le module et ses octets .wasm sont repris tels quels de
// dds-fork/web (dds_web_wasm.js / dds_web_wasm_bin.js) ; on appelle ici
// dds_web_calc_table pour le tableau, exactement comme dds_web.html, et
// dds_web_solve_leads pour l'entame qui tient chaque case à son chiffre.
//
// DDS utilise des threads : il lui faut l'isolation cross-origine
// (SharedArrayBuffer). Le serveur Go pose les en-têtes COOP/COEP ; à défaut,
// coi-serviceworker.js recharge la page une fois pour les fournir.
//
// Ce fichier se charge après app.js et réutilise ses globales ($, esc,
// UI_TEXT, SEAT_SHORT, RANK_MAP). renderResult() appelle parSetDeal() à chaque
// donne.

(function () {
  // Colonnes du tableau, dans l'ordre d'affichage, et index de couleur DDS
  // (res_table[strain][hand], strain 0..4 = ♠ ♥ ♦ ♣ SA).
  const COLS = ["♣", "♦", "♥", "♠", "SA"];
  const STRAIN_BY_COL = { "♣": 3, "♦": 2, "♥": 1, "♠": 0, "SA": 4 };
  const RED_COLS = new Set(["♥", "♦"]);
  // Lignes : ordre du modèle dds_web (N, S, E, O) et index de main DDS.
  const ROWS = ["N", "S", "E", "W"];
  const HAND_BY_ROW = { N: 0, E: 1, S: 2, W: 3 };
  const ROW_BY_HAND = ["N", "E", "S", "W"]; // réciproque de HAND_BY_ROW
  // Clés de couleur d'une main renvoyée par le serveur, dans l'ordre PBN.
  const SUIT_KEYS_PBN = ["spades", "hearts", "diamonds", "clubs"];
  const DIR_ORDER = ["N", "E", "S", "W"]; // ordre des mains dans « N:… »
  // Couleurs telles que dds_web_solve_leads les numérote (0..3), soit l'ordre
  // habituel d'affichage d'une main.
  const SUIT_SYMBOLS = ["♠", "♥", "♦", "♣"];
  const RED_SUITS = new Set([1, 2]);
  const RANK_PIPS = { 14: "A", 13: "K", 12: "Q", 11: "J", 10: "T" };

  let dealHands = null; // { N: {spades:"AKQ",…}, E, S, W } de la dernière donne
  let dealLang = "fr";
  let lastTable = null; // 20 entiers renvoyés par dds_web_calc_table, ou null
  let modulePromise = null;

  // Entames résolues pour la donne courante : « main:couleur » → tableau de
  // { suit, rank, score }. leadPending garde les résolutions en cours, pour ne
  // pas relancer le solveur si le pointeur revient sur la case.
  const leadResults = new Map();
  const leadPending = new Map();
  // Les appels au WASM sont synchrones : on les met à la queue leu leu plutôt
  // que d'empiler deux SolveBoard sur le même contexte.
  let solveQueue = Promise.resolve();

  function tr(key) {
    const dict = (typeof UI_TEXT === "object" && UI_TEXT[dealLang]) || {};
    return dict[key] || key;
  }

  function strainLabel(lang) {
    return lang === "en" ? "NT" : "SA";
  }

  // « N:♠.♥.♦.♣ … » attendu par DdTableDealPBN, dans l'ordre N E S W.
  function pbnFromHands(hands) {
    const oneSeat = (seat) =>
      SUIT_KEYS_PBN.map((k) => (hands[seat] && hands[seat][k]) || "").join(".");
    return "N:" + DIR_ORDER.map(oneSeat).join(" ");
  }

  function loadModule() {
    if (typeof createDdsModule !== "function" ||
        typeof ddsWebWasmBytes !== "function") {
      return Promise.reject(new Error(tr("parUnavailable")));
    }
    if (typeof SharedArrayBuffer === "undefined") {
      return Promise.reject(new Error(tr("parNoIsolation")));
    }
    if (!modulePromise) {
      modulePromise = createDdsModule({ wasmBinary: ddsWebWasmBytes() })
        .catch((err) => { modulePromise = null; throw err; });
    }
    return modulePromise;
  }

  async function calcTable(pbn) {
    const module = await loadModule();
    const outPtr = module._malloc(20 * 4);
    try {
      const rc = module.ccall(
        "dds_web_calc_table", "number", ["string", "number"], [pbn, outPtr]);
      if (rc !== 1) throw new Error("DDS (code " + rc + ")");
      const out = [];
      for (let i = 0; i < 20; i++) {
        out.push(module.getValue(outPtr + i * 4, "i32"));
      }
      return out;
    } finally {
      module._free(outPtr);
    }
  }

  // Toutes les entames possibles depuis la main `first`, atout `trump`. Le C++
  // développe déjà les cartes équivalentes une par une ; `score` est le nombre
  // de levées du camp qui entame, donc de la défense.
  async function calcLeads(pbn, trump, first) {
    const module = await loadModule();
    const outPtr = module._malloc((1 + 13 * 3) * 4);
    try {
      const rc = module.ccall(
        "dds_web_solve_leads", "number",
        ["string", "number", "number", "number"], [pbn, trump, first, outPtr]);
      if (rc !== 1) throw new Error("DDS (code " + rc + ")");
      const n = module.getValue(outPtr, "i32");
      if (n < 1 || n > 13) throw new Error("DDS (" + n + ")");
      const leads = [];
      for (let i = 0; i < n; i++) {
        const base = outPtr + (1 + 3 * i) * 4;
        leads.push({
          suit: module.getValue(base, "i32"),
          rank: module.getValue(base + 4, "i32"),
          score: module.getValue(base + 8, "i32"),
        });
      }
      return leads;
    } finally {
      module._free(outPtr);
    }
  }

  // Une case du tableau, c'est un déclarant et une couleur ; l'entameur est à
  // sa gauche, soit la main suivante dans l'ordre N E S W.
  function leadsFor(hand, strain) {
    const key = hand + ":" + strain;
    if (leadPending.has(key)) return leadPending.get(key);

    const pbn = pbnFromHands(dealHands);
    const first = (hand + 1) % 4;
    const run = () => calcLeads(pbn, strain, first);
    const p = solveQueue.then(run, run);
    // La file doit survivre à un échec, et un échec ne doit pas être mémorisé.
    solveQueue = p.catch(() => {});
    p.then(
      (leads) => { leadResults.set(key, leads); },
      () => { leadPending.delete(key); });
    leadPending.set(key, p);
    return p;
  }

  function renderTable() {
    const table = $("#par-table");
    const head = $("#par-head");
    const body = $("#par-body");
    const hint = $("#par-hint");
    if (!table || !head || !body) return;
    hideTip();
    if (!lastTable) {
      table.classList.add("hidden");
      head.innerHTML = "";
      body.innerHTML = "";
      if (hint) hint.classList.add("hidden");
      return;
    }
    const lang = dealLang;

    head.innerHTML =
      `<th>${esc(tr("parContracts"))}</th>` +
      COLS.map((c) => {
        const label = c === "SA" ? strainLabel(lang) : c;
        const cls = RED_COLS.has(c) ? ' class="red"' : "";
        return `<th${cls}>${label}</th>`;
      }).join("");

    body.innerHTML = ROWS.map((row) => {
      const hand = HAND_BY_ROW[row];
      const cells = COLS.map((c) => {
        const strain = STRAIN_BY_COL[c];
        const cls = "par-cell" + (RED_COLS.has(c) ? " red" : "");
        const tricks = lastTable[strain * 4 + hand];
        return `<td class="${cls}" tabindex="0" data-hand="${hand}"` +
          ` data-strain="${strain}">${tricks}</td>`;
      }).join("");
      const seat = (SEAT_SHORT[lang] && SEAT_SHORT[lang][row]) || row;
      return `<tr><th>${esc(seat)}</th>${cells}</tr>`;
    }).join("");

    table.classList.remove("hidden");
    if (hint) hint.classList.remove("hidden");
  }

  // ---------- infobulle d'entame ----------

  // « A », « R », « 10 »… : DDS numérote les rangs de 2 à 14, app.js tient la
  // traduction des figures.
  function rankLabel(rank, lang) {
    const pip = RANK_PIPS[rank] || String(rank);
    const map = (typeof RANK_MAP === "object" && RANK_MAP[lang]) || {};
    return map[pip] || pip;
  }

  // « ♠ A 7  ♦ 5 » : les cartes groupées par couleur, de la plus forte à la
  // plus faible.
  function cardsHTML(leads, lang) {
    const bySuit = [[], [], [], []];
    for (const l of leads) bySuit[l.suit].push(l.rank);
    return bySuit.map((ranks, suit) => {
      if (!ranks.length) return "";
      ranks.sort((a, b) => b - a);
      const cls = RED_SUITS.has(suit) ? "par-tip-suit red" : "par-tip-suit";
      return `<span class="${cls}">${SUIT_SYMBOLS[suit]} ` +
        ranks.map((r) => esc(rankLabel(r, lang))).join(" ") + "</span>";
    }).filter(Boolean).join(" ");
  }

  function tricksLabel(n, lang) {
    if (lang === "en") return n + (n > 1 ? " tricks" : " trick");
    return n + (n > 1 ? " levées" : " levée");
  }

  function tipHTML(hand, strain, leads, lang) {
    const seat = (SEAT_SHORT[lang] && SEAT_SHORT[lang][ROW_BY_HAND[hand]]) ||
      ROW_BY_HAND[hand];
    const leader = ROW_BY_HAND[(hand + 1) % 4];
    const leaderSeat = (SEAT_SHORT[lang] && SEAT_SHORT[lang][leader]) || leader;
    const col = COLS.find((c) => STRAIN_BY_COL[c] === strain);
    const colLabel = col === "SA" ? strainLabel(lang) : col;
    const colCls = RED_COLS.has(col) ? ' class="red"' : "";

    // La meilleure défense entame de la carte qui lui rapporte le plus ; le
    // déclarant fait le reste, ce qui redonne le chiffre de la case.
    const best = leads.reduce((m, l) => Math.max(m, l.score), -1);
    const killers = leads.filter((l) => l.score === best);
    const others = leads.filter((l) => l.score < best);

    let lead;
    let alt = "";
    if (!others.length) {
      // Cas fréquent des contrats sans enjeu : l'entame ne change rien.
      lead = `${esc(tr("parLeadVerb"))} ${esc(tr("parLeadAny"))}`;
    } else {
      // Les cartes équivalentes sont développées une par une : la liste des
      // entames qui tiennent peut couvrir presque la main. On montre alors le
      // camp le plus court, qui dit la même chose en trois cartes.
      const worst = others.reduce((m, l) => Math.min(m, l.score), 14);
      const low = 13 - best + 1;
      const high = 13 - worst;
      const range = low === high ? String(low) : low + "–" + high;
      if (killers.length > others.length) {
        lead = `${esc(tr("parLeadVerb"))} ${esc(tr("parLeadExcept"))} ` +
          cardsHTML(others, lang);
        alt = `${esc(tr("parLeadThose"))} : ${esc(range)}`;
      } else {
        lead = `${esc(tr("parLeadVerb"))} ${cardsHTML(killers, lang)}`;
        alt = `${esc(tr("parLeadOther"))} : ${esc(range)}`;
      }
    }

    let html =
      `<div class="par-tip-head"><b>${esc(seat)}</b>` +
      `<span${colCls}>${colLabel}</span>` +
      `<span class="muted">${esc(tricksLabel(13 - best, lang))}</span></div>` +
      `<div class="par-tip-lead">${esc(leaderSeat)} ${lead}</div>`;
    if (alt) html += `<div class="par-tip-alt muted">${alt}</div>`;
    return html;
  }

  let openCell = null;   // case dont l'infobulle est affichée
  let solveTimer = 0;    // délai avant de lancer le solveur (intention de survol)
  let tipToken = 0;      // jeton anti-course : seul le dernier survol écrit
  let lastPointerType = "mouse";

  function tipEl() { return $("#par-tip"); }

  // Sous la largeur du responsive, l'infobulle flottante ne tient plus : elle
  // devient un bandeau collé en bas de l'écran. Idem sans survol (tactile), où
  // le doigt masque justement la case et ses voisines.
  function sheetMode() {
    const noHover = !!(window.matchMedia &&
      window.matchMedia("(hover: none)").matches);
    return noHover || window.innerWidth <= 720;
  }

  function placeTip(tip, cell) {
    if (sheetMode()) {
      tip.classList.add("sheet");
      tip.style.left = "";
      tip.style.top = "";
      return;
    }
    tip.classList.remove("sheet");
    const r = cell.getBoundingClientRect();
    const w = tip.offsetWidth;
    const h = tip.offsetHeight;
    const left = Math.max(8, Math.min(r.left + r.width / 2 - w / 2,
      window.innerWidth - w - 8));
    let top = r.bottom + 8;
    if (top + h > window.innerHeight - 8) top = Math.max(8, r.top - h - 8);
    tip.style.left = Math.round(left) + "px";
    tip.style.top = Math.round(top) + "px";
  }

  function setTip(cell, html) {
    const tip = tipEl();
    if (!tip || openCell !== cell) return;
    tip.innerHTML = html;
    tip.classList.remove("hidden");
    placeTip(tip, cell);
  }

  function hideTip() {
    tipToken++;
    clearTimeout(solveTimer);
    if (openCell) openCell.classList.remove("is-open");
    openCell = null;
    const tip = tipEl();
    if (tip) {
      tip.classList.add("hidden");
      tip.innerHTML = "";
    }
  }

  function showTip(cell) {
    if (!cell || !dealHands || !lastTable) return;
    if (openCell === cell) return;
    hideTip();
    openCell = cell;
    cell.classList.add("is-open");

    const hand = Number(cell.dataset.hand);
    const strain = Number(cell.dataset.strain);
    const key = hand + ":" + strain;
    if (leadResults.has(key)) {
      setTip(cell, tipHTML(hand, strain, leadResults.get(key), dealLang));
      return;
    }

    setTip(cell,
      `<div class="par-tip-wait">${esc(tr("parLeadComputing"))}</div>`);
    // Un SolveBoard bloque le fil principal une fraction de seconde : on attend
    // une intention de survol avant de le lancer, sinon un simple balayage du
    // tableau résoudrait les vingt cases.
    const token = ++tipToken;
    solveTimer = setTimeout(() => {
      leadsFor(hand, strain).then(
        (leads) => {
          if (token !== tipToken) return;
          setTip(cell, tipHTML(hand, strain, leads, dealLang));
        },
        (e) => {
          if (token !== tipToken) return;
          const msg = (e && e.message) ? e.message : String(e);
          setTip(cell, `<div class="par-tip-err">${esc(msg)}</div>`);
        });
    }, 140);
  }

  function cellOf(node) {
    return (node && node.closest) ? node.closest("td.par-cell") : null;
  }

  function bindTip() {
    const body = $("#par-body");
    if (!body) return;

    // Souris : survol. Le doigt, lui, passe par le clic, plus bas.
    body.addEventListener("pointerover", (e) => {
      if (e.pointerType && e.pointerType !== "mouse") return;
      const cell = cellOf(e.target);
      if (cell) showTip(cell);
    });
    body.addEventListener("pointerout", (e) => {
      if (e.pointerType && e.pointerType !== "mouse") return;
      const cell = cellOf(e.target);
      if (cell && cell === openCell && !cell.contains(e.relatedTarget)) {
        hideTip();
      }
    });

    // Tactile : une tape ouvre, une deuxième referme. À la souris on ne
    // bascule pas, sinon un clic effacerait l'infobulle que le survol vient
    // d'ouvrir.
    body.addEventListener("click", (e) => {
      const cell = cellOf(e.target);
      if (!cell) return;
      if (cell === openCell && lastPointerType !== "mouse") hideTip();
      else showTip(cell);
    });

    // Clavier : les cases sont dans l'ordre de tabulation.
    body.addEventListener("focusin", (e) => {
      const cell = cellOf(e.target);
      if (cell) showTip(cell);
    });
    body.addEventListener("focusout", (e) => {
      const cell = cellOf(e.target);
      if (cell && cell === openCell) hideTip();
    });

    // Toucher ailleurs referme ; l'infobulle elle-même n'intercepte pas les
    // pointeurs (pointer-events: none).
    document.addEventListener("pointerdown", (e) => {
      lastPointerType = e.pointerType || "mouse";
      if (openCell && !cellOf(e.target)) hideTip();
    });
    document.addEventListener("keydown", (e) => {
      if (e.key === "Escape") hideTip();
    });
    // L'infobulle flottante se décrocherait de sa case ; le bandeau bas, lui,
    // est collé à l'écran et survit au défilement — précieux au doigt, qui
    // fait défiler la page rien qu'en lisant.
    window.addEventListener("scroll", () => {
      if (!sheetMode()) hideTip();
    }, { passive: true });
    window.addEventListener("resize", hideTip);
  }

  // Appelé par renderResult() : nouvelle donne affichée, on repart de zéro.
  window.parSetDeal = function (hands, lang) {
    dealHands = hands || null;
    dealLang = lang || dealLang;
    lastTable = null;
    leadResults.clear();
    leadPending.clear();
    const err = $("#par-error");
    const status = $("#par-status");
    if (err) err.textContent = "";
    if (status) status.textContent = "";
    renderTable();
  };

  async function onCompute() {
    if (!dealHands) return;
    const btn = $("#par-btn");
    const err = $("#par-error");
    const status = $("#par-status");
    if (err) err.textContent = "";
    if (status) status.textContent = tr("parComputing");
    if (btn) btn.disabled = true;
    try {
      lastTable = await calcTable(pbnFromHands(dealHands));
      renderTable();
    } catch (e) {
      lastTable = null;
      renderTable();
      if (err) err.textContent = (e && e.message) ? e.message : String(e);
    } finally {
      if (status) status.textContent = "";
      if (btn) btn.disabled = false;
    }
  }

  const btn = $("#par-btn");
  if (btn) btn.addEventListener("click", onCompute);
  bindTip();

  // applyLang() ne retouche que le texte statique (data-i18n) ; le tableau,
  // lui, est reconstruit ici quand la langue change s'il porte un résultat.
  const langSel = $("#lang");
  if (langSel) {
    langSel.addEventListener("change", () => {
      dealLang = langSel.value;
      if (lastTable) renderTable();
    });
  }
})();
