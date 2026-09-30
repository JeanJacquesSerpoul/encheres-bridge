// Prend les captures d'écran du tutoriel (cli/tutorial/<langue>/NN.jpg) avec
// Playwright, sur le client servi en local. Voir tools/tutorial/README.md.
//
// Chaque étape du tutoriel (TUTORIAL_STEPS dans cli/app.js) a sa capture, du
// même numéro. Les scènes sont rejouées pour chaque langue : l'interface est
// prise telle qu'elle s'affiche en français, puis en anglais.
//
// Usage : node capture.js <url du client> <dossier de sortie>

const path = require("path");
const fs = require("fs");
const { execSync } = require("child_process");

let playwright;
try {
  playwright = require("playwright");
} catch {
  playwright = require(path.join(execSync("npm root -g").toString().trim(), "playwright"));
}

const [base, outDir] = process.argv.slice(2);
if (!base || !outDir) {
  console.error("usage : node capture.js <url du client> <dossier de sortie>");
  process.exit(2);
}

const DESKTOP = { width: 1280, height: 820 };
const PHONE = { width: 390, height: 780 };
const QUALITY = 82;

// Une donne fixe : la donne exemple du client, que la première visite affiche.
// Les captures ne dépendent ainsi ni du hasard ni de la dernière donne vue.

async function openPage(browser, lang, viewport, extra = {}) {
  const ctx = await browser.newContext({
    viewport,
    deviceScaleFactor: 1.5,
    colorScheme: "light",
    locale: lang === "fr" ? "fr-FR" : "en-GB",
    ...extra,
  });
  await ctx.addInitScript((l) => {
    try {
      localStorage.clear();
      localStorage.setItem("bids.welcomed", "1");
      localStorage.setItem("bids.lang", l);
      localStorage.setItem("bids.theme", "light");
    } catch (e) { /* stockage indisponible : les valeurs par défaut suffisent */ }
  }, lang);
  const page = await ctx.newPage();
  page.on("pageerror", (e) => console.error(`[${lang}] erreur JS :`, e.message));
  await page.goto(base);
  await page.waitForSelector("#boot-loading", { state: "detached" });
  // Le moteur répond de lui-même sur la donne exemple : on attend sa séquence.
  await page.waitForSelector("#comments li", { timeout: 30000 });
  await page.mouse.move(0, 0);
  return { ctx, page };
}

// Capture d'une zone de la page, élargie d'une marge, bornée à la fenêtre.
async function shoot(page, file, selectors, pad = 12) {
  await page.waitForTimeout(250);
  const boxes = [];
  for (const sel of [].concat(selectors)) {
    const box = await page.locator(sel).first().boundingBox();
    if (box) boxes.push(box);
  }
  const vp = page.viewportSize();
  const x0 = Math.max(0, Math.min(...boxes.map((b) => b.x)) - pad);
  const y0 = Math.max(0, Math.min(...boxes.map((b) => b.y)) - pad);
  const x1 = Math.min(vp.width, Math.max(...boxes.map((b) => b.x + b.width)) + pad);
  const y1 = Math.min(vp.height, Math.max(...boxes.map((b) => b.y + b.height)) + pad);
  await page.screenshot({
    path: file,
    type: "jpeg",
    quality: QUALITY,
    clip: { x: x0, y: y0, width: x1 - x0, height: y1 - y0 },
  });
}

async function scenes(browser, lang, dir) {
  const out = (n) => path.join(dir, `${String(n).padStart(2, "0")}.jpg`);
  let { ctx, page } = await openPage(browser, lang, DESKTOP);

  // 1. L'écran : la table à gauche, les onglets à droite.
  await page.screenshot({ path: out(1), type: "jpeg", quality: QUALITY });

  // 2. Obtenir une donne : le menu de la flèche de « Donne aléatoire ».
  await page.click("#new-deal-more-btn");
  await shoot(page, out(2), ["#deal-actions", "#new-deal-pop"]);
  await page.keyboard.press("Escape");

  // 3. Partager la donne.
  await page.click("#share-menu-btn");
  await shoot(page, out(3), ["#deal-actions", "#share-pop"]);
  await page.keyboard.press("Escape");

  // 4. Modifier la donne : la table en édition. Plus haute que l'écran : la
  // fenêtre grandit, sinon la rangée collée du bas masquerait les commandes.
  await page.setViewportSize({ width: DESKTOP.width, height: 1400 });
  await page.click("#edit-btn");
  await shoot(page, out(4), ["#edit-btn", ".cons-layout", "#cons-actions"]);

  // 5. Déplacer plusieurs cartes : une couleur et une carte prises.
  await page.click('.suit-pick[data-owner="E"][data-suit-pick="hearts"]');
  await page.click('.card[data-owner="E"][data-suit="spades"]');
  await page.mouse.move(0, 0);
  await shoot(page, out(5), [".cons-layout", "#cons-selbar"]);

  // 6. Glisser-déposer : la sélection part au centre.
  const src = page.locator('.card[data-owner="E"].picked').first();
  const dst = page.locator("#cons-center");
  const a = await src.boundingBox();
  const d = await dst.boundingBox();
  await page.mouse.move(a.x + a.width / 2, a.y + a.height / 2);
  await page.mouse.down();
  await page.mouse.move(a.x + 30, a.y + 20, { steps: 4 });
  await page.mouse.move(d.x + d.width / 2, d.y + d.height / 2, { steps: 10 });
  await shoot(page, out(6), [".cons-layout"]);
  await page.mouse.up();

  // 7. Bornes et Compléter : la donne incomplète, les bornes ouvertes.
  await page.click("#cons-bounds-btn");
  await page.locator("#cons-N .cons-input").first().fill("15");
  await page.mouse.move(0, 0);
  await shoot(page, out(7), [".cons-layout", "#cons-actions", "#bid-row"]);
  await ctx.close();

  ({ ctx, page } = await openPage(browser, lang, DESKTOP));

  // 8. Les enchères : la séquence, une enchère survolée.
  const bid = page.locator("#auction-body td .bid, #auction-body td").nth(1);
  await bid.hover().catch(() => {});
  await shoot(page, out(8), ["#side-pane"]);
  await page.mouse.move(0, 0);

  // 9. L'arbre de décision d'une enchère.
  // Celui de l'ouverture : l'arbre y parcourt toute l'échelle des ouvertures.
  await page.locator("#comments .tree-icon").nth(1).click();
  await page.mouse.move(0, 0);
  await page.keyboard.press("Escape");
  await page.waitForTimeout(200);
  await shoot(page, out(9), ["#comments"]);

  // 10. Le PAR.
  await page.click("#tab-par");
  await page.waitForSelector("#par-table tbody tr, #par-body tr", { timeout: 60000 });
  await page.waitForTimeout(400);
  // Une case survolée : les entames qui tiennent le contrat.
  await page.locator("#par-table tbody tr, #par-body tr").nth(2).locator("td").last().hover();
  await page.waitForTimeout(300);
  await shoot(page, out(10), ["#tabpanel-par", "#par-tip"]);
  await page.mouse.move(0, 0);

  // 11. S'entraîner : le choix de la main.
  await page.click("#train-btn");
  await page.waitForSelector("#train-seat-dialog[open]");
  await page.waitForTimeout(200);
  await shoot(page, out(11), ["#train-seat-dialog .train-seat-box"]);

  // 12. La table : votre main en bas, la boîte à enchères à votre tour. Les
  // donnes sont tirées au hasard : la capture change d'une fois à l'autre.
  await page.click('.seat-choice[data-seat="S"]');
  await page.waitForSelector("#bidding-box .bb-btn:not([disabled])", { timeout: 30000 });
  await page.waitForTimeout(300);
  await page.screenshot({ path: out(12), type: "jpeg", quality: QUALITY });

  // 13. La réponse, après une erreur : la bonne enchère ne s'affiche pas, la
  // donne continue. Passe, sauf si c'est justement l'enchère attendue : alors
  // la première couleur permise, au palier présélectionné.
  const passIsRight = await page.evaluate(() =>
    quiz.result.auction[quiz.idx].bid === document.querySelector("#bidding-box .bb-pass").dataset.bid);
  if (passIsRight) await page.locator("#bidding-box .bb-strain:not([disabled])").first().click();
  else await page.locator("#bidding-box .bb-pass").click();
  await page.waitForSelector("#quiz-feedback:not(.hidden)", { timeout: 10000 }).catch(() => {});
  await page.waitForTimeout(300);
  await page.screenshot({ path: out(13), type: "jpeg", quality: QUALITY });

  // 14. Le score de la séance, après une donne jouée jusqu'au bout.
  for (let i = 0; i < 60; i++) {
    if (await page.locator("#train-new-btn:not(.hidden)").isVisible().catch(() => false)) break;
    const cont = page.locator("#quiz-continue-btn:not(.hidden)");
    if (await cont.isVisible().catch(() => false)) { await cont.click(); continue; }
    const pass = page.locator("#bidding-box:not(.hidden) .bb-pass:not([disabled])");
    if (await pass.isVisible().catch(() => false)) { await pass.click(); continue; }
    await page.waitForTimeout(400);
  }
  await page.click("#train-finish");
  await page.waitForSelector("#train-summary:not([hidden])");
  await page.waitForTimeout(300);
  await page.screenshot({ path: out(14), type: "jpeg", quality: QUALITY });
  await page.click("#train-end-btn");

  // 15. Réglages.
  await page.click("#settings-btn");
  await shoot(page, out(15), ["#settings-btn", "#settings-pop"]);
  await ctx.close();

  // 16. Sur téléphone : la barre d'onglets au bas de l'écran.
  ({ ctx, page } = await openPage(browser, lang, PHONE, { isMobile: true, hasTouch: true }));
  await page.screenshot({ path: out(16), type: "jpeg", quality: QUALITY });
  await ctx.close();
}

(async () => {
  const browser = await playwright.chromium.launch();
  for (const lang of ["fr", "en"]) {
    const dir = path.join(outDir, lang);
    fs.mkdirSync(dir, { recursive: true });
    await scenes(browser, lang, dir);
    console.log(`${lang} : captures écrites dans ${dir}`);
  }
  await browser.close();
})().catch((e) => {
  console.error(e);
  process.exit(1);
});
