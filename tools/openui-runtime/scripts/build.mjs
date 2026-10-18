import "./bundle.mjs";
import { createHash } from "node:crypto";
import { readFile, readdir, writeFile } from "node:fs/promises";
import { join } from "node:path";
import { resolve } from "node:path";
import { execFileSync } from "node:child_process";
import { build } from "esbuild";

// Generate the distributed schema/guide from the same registry as the viewer.
await build({ entryPoints: ["scripts/catalogue-entry.ts"], outfile: "dist/catalogue-tool.cjs", bundle: true,
  platform: "node", target: "node22", format: "cjs", jsx: "automatic", logLevel: "warning",
  define: { "process.env.NODE_ENV": '"production"' },
  alias: { "@openuidev/lang-core": resolve("src/runtime-core.ts"), "@whaleisle/official-lang-core": resolve("node_modules/@openuidev/lang-core/dist/index.mjs") },
});
execFileSync(process.execPath, ["dist/catalogue-tool.cjs"], { stdio: "inherit" });

const sha256 = value => createHash("sha256").update(value).digest("hex");
const supplemental = JSON.parse(await readFile("licenses/supplemental/index.json", "utf8"));
const paths = JSON.parse(await readFile("dist/used-package-paths.json", "utf8"));
const packages = [];
const sections = ["THIRD-PARTY NOTICES\nWhaleIsle-MyGo independently authored OpenUI viewer\n\nThese terms cover their respective dependencies. They do not license the excluded open-intelligent-ui application."];
for (const path of paths) {
  const info = JSON.parse(await readFile(join(path, "package.json"), "utf8"));
  const key = `${info.name}@${info.version}`;
  const noticeFiles = (await readdir(path, { withFileTypes: true }))
    .filter(entry => entry.isFile() && /^(licen[cs]e|copying|notice|copyright)([._-]|$)/i.test(entry.name))
    .map(entry => entry.name).sort();
  const extra = supplemental.packages[key];
  if (noticeFiles.length === 0 && !extra?.status?.startsWith("verified-")) throw new Error(`No verified complete notice for ${key}`);
  const records = [];
  for (const file of noticeFiles) {
    const text = await readFile(join(path, file), "utf8");
    sections.push(`${key}\nPublished package file: ${file}\n\n${text}`);
    records.push({ file, sha256: sha256(text) });
  }
  if (extra && (extra.supplementalToShippedLicense || noticeFiles.length === 0)) {
    for (const item of extra.files) {
      const text = await readFile(join("licenses/supplemental", item.file), "utf8");
      if (sha256(text) !== item.sha256) throw new Error(`Supplemental notice changed: ${item.file}`);
      sections.push(`${key}\nSource: ${item.source}\nProvenance: ${item.provenance}\n\n${text}`);
      records.push(item);
    }
  }
  packages.push({ name: info.name, version: info.version, license: info.license ?? null, path, notices: records });
}
const legal = await readFile("dist/viewer.js.LEGAL.txt", "utf8").catch(() => "");
if (legal) sections.push(`Preserved production-bundle legal comments\n\n${legal}`);
const notices = sections.join("\n\n" + "=".repeat(72) + "\n\n") + "\n";
await writeFile("dist/THIRD-PARTY-NOTICES.txt", notices);

const script = (await readFile("dist/viewer.js", "utf8")).replace(/<\/script/gi, "<\\/script");
const style = (await readFile("dist/viewer.css", "utf8")).replace(/<\/style/gi, "<\\/style");
const embeddedNotices = JSON.stringify({ notices }).replace(/</g, "\\u003c");
const csp = "default-src 'none'; script-src 'unsafe-inline' data:; style-src 'unsafe-inline'; img-src data:; font-src data:; connect-src 'none'; frame-src 'none'; worker-src 'none'; object-src 'none'; media-src 'none'; base-uri 'none'; form-action 'none'";
const html = `<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><meta http-equiv="Content-Security-Policy" content="${csp}"><title>WhaleIsle independent OpenUI viewer</title><style>${style}</style></head><body><div id="fatal" hidden role="alert"></div><div id="viewer"></div><script type="application/json" id="third-party-notices">${embeddedNotices}</script><script>${script}</script></body></html>\n`;
if (Buffer.byteLength(html) > 12 * 1024 * 1024) throw new Error("Viewer exceeds the existing 12 MiB host resource limit");
await writeFile("dist/viewer.html", html);
const provenance = {
  runtime: "whaleisle-independent-col-v1",
  packaging: "Production asset generation only; tests and browser/native acceptance not run.",
  originalApplicationIncluded: false,
  lockfileSHA256: sha256(await readFile("package-lock.json")),
  assets: ["viewer.html", "viewer.js", "viewer.css", "THIRD-PARTY-NOTICES.txt", "library-spec.json", "library-guide.txt"].map(name => ({ name })),
  packages,
};
for (const asset of provenance.assets) {
  const bytes = await readFile(join("dist", asset.name));
  Object.assign(asset, { bytes: bytes.length, sha256: sha256(bytes) });
}
await writeFile("dist/provenance.json", JSON.stringify(provenance, null, 2) + "\n");
console.log(`Generated self-contained viewer.html (${Buffer.byteLength(html)} bytes), including complete notices for ${packages.length} packages.`);
