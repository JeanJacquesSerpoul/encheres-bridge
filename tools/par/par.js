"use strict";

// Barème du bridge de duplicate et calcul du « par » d'une donne à partir de
// sa table de levées double-mort. Aucune entrée-sortie ici : audit.js fournit
// la table (les vingt entiers de dds_web_calc_table) et la vulnérabilité.

// Ordre des dénominations côté moteur : 0=♣ 1=♦ 2=♥ 3=♠ 4=SA.
// Ordre des couleurs côté DDS : 0=♠ 1=♥ 2=♦ 3=♣ 4=SA, et res_table[couleur*4+main]
// avec les mains dans l'ordre N E S W.
const DDS_STRAIN = [3, 2, 1, 0, 4];
const SIDE_SEATS = [[0, 2], [1, 3]]; // NS, EO

// tricksIn renvoie les levées double-mort d'un camp dans une dénomination :
// le meilleur des deux sièges, puisque le camp choisit son déclarant.
function tricksIn(table, strain, side) {
  const row = DDS_STRAIN[strain] * 4;
  return Math.max(table[row + SIDE_SEATS[side][0]], table[row + SIDE_SEATS[side][1]]);
}

// trickScore : valeur des levées du contrat annoncé (hors surlevées).
function trickScore(level, strain) {
  if (strain === 4) return 40 + 30 * (level - 1);
  if (strain >= 2) return 30 * level; // majeures
  return 20 * level; // mineures
}

const overtrickValue = (strain) => (strain <= 1 ? 20 : 30);

// undoubledPenalty / doubledPenalty : le prix de la chute, non contrée puis
// contrée (100, 300, 500, 800, 1100… non vulnérable ; 200, 500, 800… vulnérable).
const undoubledPenalty = (down, vul) => (vul ? 100 : 50) * down;

function doubledPenalty(down, vul) {
  if (vul) return 200 + 300 * (down - 1);
  if (down === 1) return 100;
  if (down === 2) return 300;
  return 500 + 300 * (down - 3);
}

// score renvoie la marque d'un contrat joué, du point de vue du camp qui le
// joue, pour un nombre de levées donné.
function score(level, strain, tricks, vul, doubled) {
  const needed = level + 6;
  if (tricks < needed) {
    const down = needed - tricks;
    return -(doubled ? doubledPenalty(down, vul) : undoubledPenalty(down, vul));
  }
  const ts = doubled ? 2 * trickScore(level, strain) : trickScore(level, strain);
  let total = ts;
  total += doubled
    ? (tricks - needed) * (vul ? 200 : 100)
    : (tricks - needed) * overtrickValue(strain);
  total += ts >= 100 ? (vul ? 500 : 300) : 50;
  if (level >= 6) total += vul ? 750 : 500;
  if (level >= 7) total += vul ? 750 : 500;
  if (doubled) total += 50;
  return total;
}

// Les trente-cinq paliers, du 1♣ au 7SA : index = (palier-1)*5 + dénomination.
const CONTRACTS = 35;
const levelOf = (idx) => Math.floor(idx / 5) + 1;
const strainOf = (idx) => idx % 5;

// par calcule le contrat du par et sa marque, vue de NS.
//
// Convention retenue : chacun leur tour, en partant du donneur, les camps font
// l'enchère la moins chère qui améliore leur marque -- un contrat qu'ils
// tiennent en double-mort, ou un sacrifice contré qui coûte moins que le
// contrat adverse. Les défenseurs contrent tout contrat qui chute et ne
// contrent jamais un contrat qui passe : contrer un contrat qui passe ne
// ferait que payer le déclarant. L'enchère montant strictement, la boucle
// s'arrête ; le dernier contrat annoncé est celui du par.
function par(table, vul /* [nsVul, ewVul] */, dealerSide) {
  // signed : marque du contrat idx joué par side, vue de NS.
  const signed = (idx, side) => {
    const strain = strainOf(idx);
    const tricks = tricksIn(table, strain, side);
    const made = tricks >= levelOf(idx) + 6;
    const s = score(levelOf(idx), strain, tricks, vul[side], !made);
    return side === 0 ? s : -s;
  };

  let last = -1;          // dernier palier annoncé
  let value = 0;          // marque courante, vue de NS
  let contract = null;    // contrat du par, null si personne n'ouvre
  let turn = dealerSide;
  let idle = 0;           // camps consécutifs sans enchère : deux et c'est fini

  while (idle < 2) {
    let found = -1;
    for (let idx = last + 1; idx < CONTRACTS; idx++) {
      const v = signed(idx, turn);
      if (turn === 0 ? v > value : v < value) {
        found = idx;
        break;
      }
    }
    if (found < 0) {
      idle++;
      turn = 1 - turn;
      continue;
    }
    idle = 0;
    last = found;
    value = signed(found, turn);
    const strain = strainOf(found);
    const tricks = tricksIn(table, strain, turn);
    contract = {
      level: levelOf(found),
      strain,
      side: turn,
      tricks,
      makes: tricks >= levelOf(found) + 6,
    };
    turn = 1 - turn;
  }
  return { score: value, contract };
}

// Barème IMP officiel : bornes inférieures des 0, 1, 2… IMP.
const IMP_TABLE = [
  0, 10, 40, 80, 120, 160, 210, 260, 310, 360, 420, 490, 590, 740, 890,
  1090, 1290, 1490, 1740, 1990, 2240, 2490, 2990, 3490, 3990,
];

function imps(diff) {
  const a = Math.abs(diff);
  let i = 0;
  while (i < IMP_TABLE.length - 1 && a >= IMP_TABLE[i + 1]) i++;
  return Math.sign(diff) * i;
}

// Levées nécessaires à la manche, par dénomination (5♣ 5♦ 4♥ 4♠ 3SA).
const GAME_TRICKS = [11, 11, 10, 10, 9];

const canMakeGame = (table, side) =>
  GAME_TRICKS.some((need, strain) => tricksIn(table, strain, side) >= need);

const canMakeSlam = (table, side) =>
  [0, 1, 2, 3, 4].some((strain) => tricksIn(table, strain, side) >= 12);

module.exports = {
  DDS_STRAIN, SIDE_SEATS, GAME_TRICKS,
  tricksIn, trickScore, score, par, imps, canMakeGame, canMakeSlam,
};
