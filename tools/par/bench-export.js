"use strict";

// Construit le jeu de données du banc de non-régression (voir README.md,
// « Banc de non-régression ») à partir d'audits déjà calculés.
//
//   node tools/par/bench-export.js <sortie.jsonl.gz> <graine>=<par.json> [<graine>=<par.json>…]
//
// Chaque par.json vient de audit.js. On n'en garde que ce qui ne dépend que de
// la donne : la donne elle-même, le donneur, la vulnérabilité, les vingt levées
// double-mort et le contrat du par. Les enchères, qui dépendent du moteur, sont
// rejouées par le test Go (engine/par_bench_test.go) à chaque exécution.
//
// La graine est donnée explicitement : le champ meta.seed de par.json reflète
// l'environnement de audit.js, pas celui de TestParAuditDump qui a tiré les
// donnes. Un audit découpé en plusieurs morceaux se passe morceau par morceau
// avec la même graine : l'identifiant d'une donne est « graine-étui ».

const fs = require("fs");
const zlib = require("zlib");

function main() {
  const [out, ...inputs] = process.argv.slice(2);
  if (!out || inputs.length === 0) {
    console.error("usage : node tools/par/bench-export.js <sortie.jsonl.gz> <graine>=<par.json> …");
    process.exit(2);
  }
  const seen = new Set();
  const lines = [];
  for (const arg of inputs) {
    const eq = arg.indexOf("=");
    if (eq <= 0) {
      console.error(`argument ${arg} : attendu <graine>=<par.json>`);
      process.exit(2);
    }
    const seed = arg.slice(0, eq);
    const data = JSON.parse(fs.readFileSync(arg.slice(eq + 1), "utf8"));
    for (const d of data.deals) {
      const id = `${seed}-${d.board}`;
      if (seen.has(id)) {
        console.error(`donne ${id} en double`);
        process.exit(1);
      }
      seen.add(id);
      lines.push({
        id,
        dealer: d.dealer,
        vul: d.vul,
        deal: d.pbn,
        dd: d.table,
        par: { score: d.parScore, level: d.parLevel, strain: d.parStrain, side: d.parSide },
      });
    }
  }
  // Ordre stable, indépendant de l'ordre des arguments : graine puis étui.
  lines.sort((a, b) => {
    const [sa, ba] = a.id.split("-").map(Number);
    const [sb, bb] = b.id.split("-").map(Number);
    return sa - sb || ba - bb;
  });
  const text = lines.map((l) => JSON.stringify(l)).join("\n") + "\n";
  fs.writeFileSync(out, zlib.gzipSync(text, { level: 9 }));
  console.log(`${lines.length} donnes écrites dans ${out}`);
}

main();
