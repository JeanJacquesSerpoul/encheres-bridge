"use strict";

// Compare deux audits du par, pour mesurer l'effet d'une modification du
// moteur.
//
//   node tools/par/compare.js <avant/par.json> <après/par.json> [--all]
//
// Les deux audits doivent avoir été tirés avec la même graine et le même
// nombre de donnes (voir README.md) : les donnes sont alors identiques et se
// comparent une à une, par numéro d'étui. Sortie : l'écart moyen au par, les
// six catégories (nombre de donnes et IMP qu'elles coûtent, en valeur
// absolue), puis les donnes dont le contrat a changé, les plus grands gains
// d'abord. Avec --all, toutes les donnes changées sont listées, sinon les
// vingt premières gagnantes et les vingt premières perdantes.

const fs = require("fs");

const STRAINS = ["T", "K", "C", "P", "SA"];
const CATEGORIES = {
  1: "Camp déclarant inversé",
  2: "Couleur différente du par",
  3: "Chelem demandé, impossible",
  4: "Chelem manqué",
  5: "Manche demandée, impossible",
  6: "Manche manquée",
};

function load(file) {
  const data = JSON.parse(fs.readFileSync(file, "utf8"));
  const byBoard = new Map();
  for (const d of data.deals) byBoard.set(d.board, d);
  return { meta: data.meta || {}, deals: data.deals, byBoard };
}

function contract(d) {
  if (!d.level) return "Passe";
  const seat = "NESW"[d.declarer];
  return d.level + STRAINS[d.strain] + (d.doubled ? "X" : "") + " " + seat;
}

function categoryCost(deals) {
  const out = {};
  for (const id of Object.keys(CATEGORIES)) out[id] = { n: 0, imp: 0 };
  for (const d of deals) {
    for (const c of d.cats) {
      out[c].n++;
      out[c].imp += Math.abs(d.imp);
    }
  }
  return out;
}

const meanAbs = (deals) => deals.reduce((a, d) => a + Math.abs(d.imp), 0) / deals.length;
const signed = (v) => (v > 0 ? "+" : "") + v;
const pad = (s, n) => String(s).padStart(n);

function main() {
  const args = process.argv.slice(2);
  const all = args.includes("--all");
  const files = args.filter((a) => a !== "--all");
  if (files.length !== 2) {
    console.error("usage : node tools/par/compare.js <avant/par.json> <après/par.json> [--all]");
    process.exit(2);
  }
  const a = load(files[0]);
  const b = load(files[1]);
  const common = a.deals.filter((d) => b.byBoard.has(d.board));
  if (common.length !== a.deals.length || common.length !== b.deals.length) {
    console.warn(`attention : ${a.deals.length} donnes avant, ${b.deals.length} après, ${common.length} en commun`);
  }
  for (const d of common) {
    if (d.pbn !== b.byBoard.get(d.board).pbn) {
      console.error(`étui ${d.board} : les donnes diffèrent, les audits n'ont pas la même graine`);
      process.exit(1);
    }
  }
  const aDeals = common;
  const bDeals = common.map((d) => b.byBoard.get(d.board));

  const ma = meanAbs(aDeals);
  const mb = meanAbs(bDeals);
  console.log(`donnes comparées : ${common.length}`);
  console.log(`écart moyen au par : ${ma.toFixed(2)} → ${mb.toFixed(2)} IMP (${signed(+(mb - ma).toFixed(2))})`);
  console.log("");
  console.log("catégorie                          donnes (avant → après)   IMP (avant → après)");
  const ca = categoryCost(aDeals);
  const cb = categoryCost(bDeals);
  for (const id of Object.keys(CATEGORIES)) {
    const name = (id + " " + CATEGORIES[id]).padEnd(34);
    console.log(`${name} ${pad(ca[id].n, 5)} → ${pad(cb[id].n, 5)} (${pad(signed(cb[id].n - ca[id].n), 4)})` +
      `   ${pad(ca[id].imp, 5)} → ${pad(cb[id].imp, 5)} (${pad(signed(cb[id].imp - ca[id].imp), 5)})`);
  }

  const changed = [];
  for (const da of aDeals) {
    const db = b.byBoard.get(da.board);
    if (da.auction === db.auction) continue;
    changed.push({
      board: da.board,
      before: contract(da),
      after: contract(db),
      impA: Math.abs(da.imp),
      impB: Math.abs(db.imp),
      gain: Math.abs(da.imp) - Math.abs(db.imp),
      auction: db.auction,
    });
  }
  changed.sort((x, y) => y.gain - x.gain || x.board - y.board);
  const won = changed.filter((c) => c.gain > 0);
  const lost = changed.filter((c) => c.gain < 0);
  const net = changed.reduce((s, c) => s + c.gain, 0);
  console.log("");
  console.log(`enchères changées : ${changed.length} (gagnées ${won.length}, perdues ${lost.length}, ` +
    `neutres ${changed.length - won.length - lost.length}), bilan ${signed(net)} IMP`);

  const show = (list) => {
    for (const c of list) {
      console.log(`  étui ${pad(c.board, 4)}  ${c.before.padEnd(9)} → ${c.after.padEnd(9)}` +
        ` ${pad(c.impA, 3)} → ${pad(c.impB, 3)} IMP   ${c.auction}`);
    }
  };
  if (all) {
    show(changed);
    return;
  }
  if (won.length) {
    console.log("\nles plus grands gains :");
    show(won.slice(0, 20));
  }
  if (lost.length) {
    console.log("\nles plus grandes pertes :");
    show(lost.slice(-20).reverse());
  }
}

main();
