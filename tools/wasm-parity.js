"use strict";

// Parité du moteur compilé en WebAssembly avec le serveur HTTP.
//
//   node tools/wasm-parity.js [--server URL] [fichiers.pbn...]
//   node tools/wasm-parity.js --dump [--lang fr|en] [--boards] <fichier.pbn>
//
// Le client de test calcule désormais les enchères de deux façons : en
// interrogeant /bid, ou dans le navigateur avec cli/bids.wasm. Les deux
// passent par le même ParsePBN, le même moteur et le même encodeur JSON
// (encodeJSON, response.go), donc ils doivent rendre les mêmes octets. Ce
// script le vérifie : il compare, donne par donne et langue par langue, la
// réponse du serveur et celle du module — le texte brut, sans le reparser, ce
// qui viderait la comparaison de son sens.
//
// Sans fichier, il prend tout testdata/*.pbn. Sortie : une ligne par cas, et
// un code de retour non nul au premier écart. Le serveur doit tourner
// (go run .) ; --dump s'en passe et n'écrit que le JSON du module.
//
// Le module se charge comme la page le fait : wasm_exec.js dans le contexte
// global (il y pose Go), puis instanciation et attente du crochet que main()
// appelle une fois son API installée — cf. cli/bids-wasm.js.

const fs = require("fs");
const path = require("path");
const vm = require("vm");

const ROOT = path.join(__dirname, "..");
const CLI = path.join(ROOT, "cli");
const WASM = path.join(CLI, "bids.wasm");
const EXEC_JS = path.join(CLI, "wasm_exec.js");
const LANGS = ["en", "fr"];

function usage() {
  fs.readFileSync(__filename, "utf8")
    .split("\n")
    .slice(2, 21)
    .forEach((l) => console.log(l.replace(/^\/\/ ?/, "")));
}

async function loadEngine() {
  for (const f of [EXEC_JS, WASM]) {
    if (!fs.existsSync(f)) {
      throw new Error(path.relative(ROOT, f) + " manquant : lancez ./build-wasm.sh");
    }
  }
  vm.runInThisContext(fs.readFileSync(EXEC_JS, "utf8"), { filename: "wasm_exec.js" });
  const go = new globalThis.Go(); // eslint-disable-line no-undef
  const { instance } = await WebAssembly.instantiate(fs.readFileSync(WASM), go.importObject);
  const ready = new Promise((resolve) => {
    globalThis.__bidsWasmReady = resolve;
  });
  // Pas attendu : main() bloque pour garder ses fonctions appelables.
  go.run(instance).catch((err) => {
    console.error("bids.wasm :", err);
    process.exit(1);
  });
  await ready;
  return globalThis.bidsWasm;
}

// callWasm rend le JSON brut, exactement comme le serveur le met sur le fil.
function callWasm(api, entry, pbn, lang) {
  const res = api[entry](pbn, lang);
  if (!res || !res.ok) throw new Error((res && res.error) || "le moteur n'a pas répondu");
  return res.json;
}

// callServer poste la donne comme le client : un champ multipart « pbn ».
async function callServer(server, route, pbn, lang) {
  const fd = new FormData();
  fd.append("pbn", new Blob([pbn], { type: "text/plain" }), "deal.pbn");
  const resp = await fetch(`${server}/${route}?lang=${lang}`, { method: "POST", body: fd });
  return resp.text();
}

// Le premier caractère qui diffère en dit plus qu'un diff complet sur deux
// lignes JSON de plusieurs kilo-octets.
function firstDiff(a, b) {
  const n = Math.min(a.length, b.length);
  for (let i = 0; i < n; i++) {
    if (a[i] !== b[i]) {
      return `octet ${i} : serveur ${JSON.stringify(a.slice(i, i + 40))}` +
        ` / wasm ${JSON.stringify(b.slice(i, i + 40))}`;
    }
  }
  return `longueurs ${a.length} et ${b.length}`;
}

function pbnFiles(args) {
  if (args.length > 0) return args;
  const dir = path.join(ROOT, "testdata");
  return fs.readdirSync(dir).filter((f) => f.endsWith(".pbn")).map((f) => path.join(dir, f));
}

async function compare(server, files) {
  const api = await loadEngine();
  let failures = 0;
  for (const file of files) {
    const pbn = fs.readFileSync(file, "utf8");
    // Une donne isolée passe aussi par /bids, qui en accepte plusieurs : les
    // deux entrées du module doivent tenir, pas seulement la première.
    const routes = [["bid", "bid"], ["bids", "bids"]];
    for (const [route, entry] of routes) {
      for (const lang of LANGS) {
        const name = `${path.basename(file)} ${route} ${lang}`;
        let http;
        try {
          http = await callServer(server, route, pbn, lang);
        } catch (err) {
          console.error(`ERREUR ${name} : serveur injoignable (${err.message})`);
          return 1;
        }
        let wasm;
        try {
          wasm = callWasm(api, entry, pbn, lang);
        } catch (err) {
          // Le serveur répond { "error": ... } sur une donne invalide ; le
          // module lève. Les deux doivent porter le même message.
          wasm = JSON.stringify({ error: err.message }) + "\n";
        }
        if (http === wasm) {
          console.log(`ok    ${name}`);
        } else {
          failures++;
          console.log(`DIFF  ${name} — ${firstDiff(http, wasm)}`);
        }
      }
    }
  }
  console.log(failures === 0 ? "\nIdentique sur tous les cas." : `\n${failures} écart(s).`);
  return failures === 0 ? 0 : 1;
}

async function dump(args) {
  let lang = "fr";
  let entry = "bid";
  const files = [];
  for (let i = 0; i < args.length; i++) {
    if (args[i] === "--lang") lang = args[++i];
    else if (args[i] === "--boards") entry = "bids";
    else files.push(args[i]);
  }
  if (files.length !== 1) {
    usage();
    return 1;
  }
  const api = await loadEngine();
  process.stdout.write(callWasm(api, entry, fs.readFileSync(files[0], "utf8"), lang));
  return 0;
}

async function main() {
  const args = process.argv.slice(2);
  if (args.includes("-h") || args.includes("--help")) {
    usage();
    return 0;
  }
  if (args[0] === "--dump") return dump(args.slice(1));
  let server = "http://localhost:9015";
  const rest = [];
  for (let i = 0; i < args.length; i++) {
    if (args[i] === "--server") server = args[++i].replace(/\/+$/, "");
    else rest.push(args[i]);
  }
  return compare(server, pbnFiles(rest));
}

main().then((code) => process.exit(code), (err) => {
  console.error(err.message);
  process.exit(1);
});
