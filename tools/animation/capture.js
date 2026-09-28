// Prend les captures d'écran de la vidéo de présentation
// (cli/animation/donne-<langue>.jpg) avec Playwright, sur le client servi en
// local : la table en mode modification, sur la donne de l'animation.
//
// Même démarche que tools/tutorial/capture.js. À relancer quand la table
// d'édition change d'aspect, ou quand l'animation change de donne (la donne
// est aussi écrite dans cli/animation.html : les deux vont ensemble).
//
// Usage : node tools/animation/capture.js <url du client> [dossier de sortie]
//   ex. : go run ./serve, puis
//         node tools/animation/capture.js http://localhost:9015/index.html

const path = require("path");
const { execSync } = require("child_process");

let playwright;
try {
  playwright = require("playwright");
} catch {
  playwright = require(path.join(execSync("npm root -g").toString().trim(), "playwright"));
}

const [base, outArg] = process.argv.slice(2);
if (!base) {
  console.error("usage : node tools/animation/capture.js <url du client> [dossier de sortie]");
  process.exit(2);
}
const outDir = outArg || path.join(__dirname, "..", "..", "cli", "animation");

// La donne de l'animation, passée par un lien de partage (#pbn=…).
const PBN = [
  '[Dealer "S"]',
  '[Vulnerable "All"]',
  '[Deal "S:J76.AK2.K6432.AJ 8.J8543.QT9.QT83 AKT95.Q9.875.962 Q432.T76.AJ.K754"]',
].join("\n");

async function shoot(browser, lang) {
  // Plus haute que l'écran : la table en édition y tient sans défilement.
  const ctx = await browser.newContext({
    viewport: { width: 1280, height: 1400 },
    deviceScaleFactor: 1.5,
    colorScheme: "light",
    locale: lang === "fr" ? "fr-FR" : "en-GB",
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
  await page.goto(base + "#pbn=" + encodeURIComponent(PBN));
  await page.waitForSelector("#boot-loading", { state: "detached" });
  await page.waitForSelector("#comments li", { timeout: 30000 });
  await page.click("#edit-btn");
  await page.mouse.move(0, 0);
  await page.waitForTimeout(400);

  const box = await page.locator(".cons-layout").first().boundingBox();
  const pad = 10;
  const file = path.join(outDir, `donne-${lang}.jpg`);
  await page.screenshot({
    path: file,
    type: "jpeg",
    quality: 82,
    clip: { x: box.x - pad, y: box.y - pad, width: box.width + 2 * pad, height: box.height + 2 * pad },
  });
  console.log(file);
  await ctx.close();
}

(async () => {
  const browser = await playwright.chromium.launch();
  try {
    for (const lang of ["fr", "en"]) await shoot(browser, lang);
  } finally {
    await browser.close();
  }
})().catch((e) => {
  console.error(e);
  process.exit(1);
});
