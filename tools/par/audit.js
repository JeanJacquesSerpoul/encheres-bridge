"use strict";

// Audit du moteur d'enchères contre le par de la donne.
//
//   node tools/par/audit.js <enchères.jsonl> [dossier de sortie]
//
// L'entrée est produite par TestParAuditDump (voir audit_par_test.go) : une
// ligne JSON par donne, avec la donne au format PBN, l'enchère complète et le
// contrat final. Pour chaque donne on calcule ici la table des levées
// double-mort, le contrat du par, puis on classe l'écart dans les six
// catégories décrites plus bas. Sortie : par.json (données brutes) et
// rapport.html (rapport lisible, gabarit template.html).
//
// Le solveur est celui du client embarqué (cli/dds_web_wasm.js) : il s'exporte
// en UMD, donc require() suffit sous Node ; seul le tableau d'octets .wasm
// s'évalue dans le contexte global, comme le fait la page.

const fs = require("fs");
const path = require("path");
const vm = require("vm");
const P = require("./par");

const CLI = path.join(__dirname, "..", "..", "cli");
const SEATS = ["N", "E", "S", "W"];

function loadSolver() {
  vm.runInThisContext(fs.readFileSync(path.join(CLI, "dds_web_wasm_bin.js"), "utf8"),
    { filename: "dds_web_wasm_bin.js" });
  const createDdsModule = require(path.join(CLI, "dds_web_wasm.js"));
  return createDdsModule({ wasmBinary: ddsWebWasmBytes() }); // eslint-disable-line no-undef
}

// calcTable renvoie les vingt levées double-mort de la donne (cinq
// dénominations x quatre mains), exactement comme le bouton « par » du client.
function calcTable(module, ptr, pbn) {
  const rc = module.ccall("dds_web_calc_table", "number",
    ["string", "number"], [pbn, ptr]);
  if (rc !== 1) throw new Error("DDS a renvoyé le code " + rc + " pour " + pbn);
  const table = [];
  for (let i = 0; i < 20; i++) table.push(module.getValue(ptr + i * 4, "i32"));
  return table;
}

// Un contrat de manche : 3SA, 4 en majeure, 5 en mineure. Le chelem est
// au-dessus, il a sa propre catégorie.
function isGameContract(level, strain) {
  if (level === 0 || level >= 6) return false;
  if (strain === 4) return level >= 3;
  if (strain >= 2) return level >= 4;
  return level >= 5;
}

// Les six écarts demandés. Chaque donne peut en cumuler plusieurs.
const CATEGORIES = [
  { id: 1, name: "Camp déclarant inversé" },
  { id: 2, name: "Couleur différente du par" },
  { id: 3, name: "Chelem demandé, impossible" },
  { id: 4, name: "Chelem manqué" },
  { id: 5, name: "Manche demandée, impossible" },
  { id: 6, name: "Manche manquée" },
];

function classify(d) {
  const cats = [];
  const played = d.level > 0;
  if (played && d.parLevel > 0 && d.simSide !== d.parSide) cats.push(1);
  if (played && d.parLevel > 0 && d.strain !== d.parStrain) cats.push(2);
  if (d.level >= 6 && d.result < 0) cats.push(3);
  if (d.level < 6 && (d.canSlam[0] || d.canSlam[1])) cats.push(4);
  if (isGameContract(d.level, d.strain) && d.result < 0) cats.push(5);
  if (!isGameContract(d.level, d.strain) && d.level < 6 &&
      (d.canGame[0] || d.canGame[1])) cats.push(6);
  return cats;
}

function analyse(row, table) {
  const vul = [row.vul === "NS" || row.vul === "All",
    row.vul === "EW" || row.vul === "All"];
  const dealerSide = SEATS.indexOf(row.dealer) % 2;
  const { score, contract } = P.par(table, vul, dealerSide);

  const d = {
    board: row.board, dealer: row.dealer, vul: row.vul, pbn: row.pbn,
    auction: row.auction, level: row.level, strain: row.strain,
    declarer: row.declarer, doubled: row.doubled, table,
    parScore: score,
    parLevel: contract ? contract.level : 0,
    parStrain: contract ? contract.strain : -1,
    parSide: contract ? contract.side : -1,
    parTricks: contract ? contract.tricks : null,
    parMakes: contract ? contract.makes : false,
    canGame: [0, 1].map((s) => P.canMakeGame(table, s)),
    canSlam: [0, 1].map((s) => P.canMakeSlam(table, s)),
    maxTricks: [0, 1].map((s) => [0, 1, 2, 3, 4].map((st) => P.tricksIn(table, st, s))),
  };

  if (row.level > 0) {
    d.simSide = row.declarer % 2;
    d.tricks = table[P.DDS_STRAIN[row.strain] * 4 + row.declarer];
    d.result = d.tricks - (row.level + 6);
    const s = P.score(row.level, row.strain, d.tricks, vul[d.simSide], row.doubled);
    d.simScore = d.simSide === 0 ? s : -s; // vu de NS, comme le par
  } else {
    d.simSide = -1;
    d.tricks = null;
    d.result = null;
    d.simScore = 0;
  }
  d.imp = P.imps(d.simScore - d.parScore); // > 0 : NS a fait mieux que le par
  d.cats = classify(d);
  return d;
}

function summarise(deals) {
  const n = deals.length;
  const absImp = deals.map((d) => Math.abs(d.imp));
  const count = (fn) => deals.filter(fn).length;
  const counts = {};
  for (const c of CATEGORIES) counts[c.id] = count((d) => d.cats.includes(c.id));
  return {
    total: n,
    flagged: count((d) => d.cats.length > 0),
    counts,
    passout: count((d) => d.level === 0),
    made: count((d) => d.level > 0 && d.result >= 0),
    parEqual: count((d) => d.simScore === d.parScore),
    nsAbove: count((d) => d.simScore > d.parScore),
    nsBelow: count((d) => d.simScore < d.parScore),
    meanImp: +(absImp.reduce((a, b) => a + b, 0) / n).toFixed(2),
    imp0: absImp.filter((v) => v === 0).length,
    imp1_3: absImp.filter((v) => v >= 1 && v <= 3).length,
    imp4_9: absImp.filter((v) => v >= 4 && v <= 9).length,
    imp10: absImp.filter((v) => v >= 10).length,
    sameSideSameStrain: count((d) => d.simSide >= 0 && d.simSide === d.parSide &&
      d.strain === d.parStrain),
    parSacrifice: count((d) => d.parLevel > 0 && !d.parMakes),
    levels: [0, 1, 2, 3, 4, 5, 6, 7].map((l) => count((d) => d.level === l)),
    parLevels: [0, 1, 2, 3, 4, 5, 6, 7].map((l) => count((d) => d.parLevel === l)),
  };
}

// Le rapport n'embarque que les donnes signalées, et sous forme abrégée : mille
// donnes complètes pèseraient le double pour rien.
function slim(d) {
  return {
    b: d.board, d: d.dealer, v: d.vul, p: d.pbn, a: d.auction,
    lv: d.level, st: d.strain, dc: d.declarer, db: d.doubled ? 1 : 0,
    tr: d.tricks, rs: d.result, ss: d.simSide, sc: d.simScore,
    pl: d.parLevel, ps: d.parStrain, pd: d.parSide,
    pm: d.parMakes ? 1 : 0, pt: d.parTricks, pv: d.parScore,
    mt: d.maxTricks, cg: d.canGame.map(Number), cs: d.canSlam.map(Number),
    im: d.imp, c: d.cats, t: d.table,
  };
}

function render(payload, outFile) {
  const tpl = fs.readFileSync(path.join(__dirname, "template.html"), "utf8");
  // Le JSON est inséré dans un <script type="application/json"> : seul « < »
  // doit être neutralisé, pour qu'un </script> imprévu ne ferme pas la balise.
  const json = JSON.stringify(payload).replace(/</g, "\\u003c");
  fs.writeFileSync(outFile, tpl.replace("__PAYLOAD__", json));
}

async function main() {
  const input = process.argv[2];
  if (!input) {
    console.error("usage: node tools/par/audit.js <enchères.jsonl> [dossier de sortie]");
    process.exit(2);
  }
  const outDir = process.argv[3] || path.join(__dirname, "out");
  fs.mkdirSync(outDir, { recursive: true });

  const rows = fs.readFileSync(input, "utf8").trim().split(/\r?\n/).map(JSON.parse);
  const module = await loadSolver();
  const ptr = module._malloc(20 * 4);
  const started = Date.now();
  const deals = [];
  for (const row of rows) {
    deals.push(analyse(row, calcTable(module, ptr, row.pbn)));
    if (deals.length % 100 === 0) {
      process.stderr.write("  " + deals.length + " / " + rows.length + " donnes\r");
    }
  }
  module._free(ptr);
  process.stderr.write("\n");

  const stats = summarise(deals);
  const meta = {
    deals: deals.length,
    date: new Date().toISOString().slice(0, 10),
    engine: process.env.PAR_AUDIT_ENGINE || "",
    seed: process.env.PAR_AUDIT_SEED || "20260906",
  };
  fs.writeFileSync(path.join(outDir, "par.json"),
    JSON.stringify({ meta, stats, deals }, null, 1));
  render({ meta, stats, deals: deals.filter((d) => d.cats.length).map(slim) },
    path.join(outDir, "rapport.html"));

  console.log(rows.length + " donnes analysées en " +
    ((Date.now() - started) / 1000).toFixed(1) + " s");
  for (const c of CATEGORIES) {
    console.log("  " + String(stats.counts[c.id]).padStart(4) + "  " + c.name);
  }
  console.log("  écart moyen au par : " + stats.meanImp.toFixed(2) + " IMP");
  console.log("sortie : " + path.join(outDir, "rapport.html"));
}

main().catch((err) => {
  console.error(err && err.stack ? err.stack : err);
  process.exit(1);
});
