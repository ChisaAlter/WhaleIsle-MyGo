import { build } from "esbuild";
import { mkdir, writeFile } from "node:fs/promises";
import { resolve } from "node:path";

await mkdir("dist", { recursive: true });
const bundle = await build({
  entryPoints: ["src/viewer.tsx"],
  outfile: "dist/viewer.js",
  bundle: true,
  platform: "browser",
  target: ["es2022"],
  format: "iife",
  jsx: "automatic",
  minify: true,
  sourcemap: false,
  metafile: true,
  legalComments: "linked",
  define: { "process.env.NODE_ENV": '"production"' },
  alias: {
    "@openuidev/lang-core": resolve("src/runtime-core.ts"),
    "@whaleisle/official-lang-core": resolve("node_modules/@openuidev/lang-core/dist/index.mjs"),
  },
  loader: { ".woff2": "dataurl", ".woff": "dataurl", ".ttf": "dataurl" },
  logLevel: "warning",
});
await writeFile("dist/bundle-metafile.json", JSON.stringify(bundle.metafile, null, 2) + "\n");
const used = new Set();
for (const output of Object.values(bundle.metafile.outputs)) {
  if (output.imports.some(item => item.external)) throw new Error("External runtime import was not bundled");
  for (const [input, details] of Object.entries(output.inputs)) {
    if (details.bytesInOutput > 0 && input.includes("node_modules/")) {
      const prefix = input.slice(0, input.lastIndexOf("node_modules/") + "node_modules/".length);
      const rest = input.slice(prefix.length).split("/");
      used.add(prefix + (rest[0].startsWith("@") ? rest.slice(0, 2).join("/") : rest[0]));
    }
  }
}
await writeFile("dist/used-package-paths.json", JSON.stringify([...used].sort(), null, 2) + "\n");
console.log(`Generated production JavaScript/CSS; ${used.size} package paths contribute bytes. No runtime acceptance performed.`);
