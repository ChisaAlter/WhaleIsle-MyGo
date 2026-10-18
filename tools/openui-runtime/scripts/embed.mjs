// Install only audited build products into this repository's fixed Go embed path.
import { createHash } from "node:crypto";
import { readFile, writeFile, mkdir, rename, access } from "node:fs/promises";
import { fileURLToPath } from "node:url";
import { dirname, resolve, join } from "node:path";

const root = resolve(dirname(fileURLToPath(import.meta.url)), "../../..");
await access(join(root, "go.mod"));
await access(join(root, "internal/desktop/openui/runtime.go"));
const source = resolve(dirname(fileURLToPath(import.meta.url)), "../dist");
const target = join(root, "internal/desktop/openui/viewer");
const provenanceBytes = await readFile(join(source, "provenance.json"));
const provenance = JSON.parse(provenanceBytes.toString("utf8"));
if (provenance.runtime !== "whaleisle-independent-col-v1" || provenance.originalApplicationIncluded !== false) {
  throw new Error("The generated provenance is not this independently authored runtime.");
}
const names = ["THIRD-PARTY-NOTICES.txt", "library-spec.json", "library-guide.txt", "viewer.html"];
const contents = new Map();
for (const name of names) {
  const bytes = await readFile(join(source, name));
  const item = provenance.assets.find(asset => asset.name === name);
  if (!item || item.bytes !== bytes.length || item.sha256 !== createHash("sha256").update(bytes).digest("hex")) {
    throw new Error(`Asset no longer matches production provenance: ${name}`);
  }
  contents.set(name, bytes);
}
await mkdir(target, { recursive: true });
// Check the whole set first. Rename each completed file; install HTML last.
for (const [name, bytes] of [["provenance.json", provenanceBytes], ...contents]) {
  const staged = join(target, `.${name}.${process.pid}.tmp`);
  await writeFile(staged, bytes);
  await rename(staged, join(target, name));
}
console.log(`Installed five production assets into ${target}. No native build or runtime acceptance performed.`);
