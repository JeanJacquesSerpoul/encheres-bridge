//go:build !(js && wasm)

// Package bids ne fait qu'une chose : embarquer le client web (cli/) dans le
// serveur. Tout le reste du code Go vit dans engine/.
//
// Ce fichier reste à la racine parce que //go:embed ne remonte jamais dans un
// dossier parent : depuis engine/, « ../cli » serait refusé. Le serveur
// (engine/main.go) l'importe sous le chemin « bids ».
//
// Exclu de la cible WebAssembly, comme main.go qui l'importe : sinon le moteur
// compilé en .wasm embarquerait cli/bids.wasm, donc lui-même, et doublerait de
// taille à chaque recompilation.
package bids

import "embed"

// CLI contient le dossier cli/ tel qu'il était au moment de la compilation, y
// compris cli/bids.wasm. Ses chemins commencent par « cli/ ».
//
//go:embed all:cli
var CLI embed.FS
