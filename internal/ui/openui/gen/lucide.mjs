// Extracts the Lucide icon set (ISC) that the Whale UI viewer bundles into
// lucide.json.gz: {"PascalName": "<svg inner markup>"}. Run from this
// directory: node lucide.mjs
import { readFileSync, writeFileSync } from "node:fs";
import { gzipSync } from "node:zlib";

const src = readFileSync(new URL("../../../desktop/openui/viewer/v2/host.cjs", import.meta.url), "utf8");
const out = {};
const re = /=f\("([a-z0-9-]+)",([A-Za-z0-9_$]+)\)/g;
let m;
while ((m = re.exec(src))) {
  const [, name, v] = m;
  const head = src.slice(0, m.index);
  let at = -1;
  for (let i = head.length; i > 0; ) {
    const j = head.lastIndexOf(v + "=[[", i - 1);
    if (j < 0) break;
    if (",; ".includes(head[j - 1])) { at = j + v.length + 1; break; }
    i = j;
  }
  if (at < 0) continue;
  let depth = 0, k = at, inStr = false;
  for (; k < src.length; k++) {
    const ch = src[k];
    if (inStr) { if (ch === "\\") k++; else if (ch === '"') inStr = false; continue; }
    if (ch === '"') inStr = true;
    else if (ch === "[") depth++;
    else if (ch === "]") { depth--; if (depth === 0) break; }
  }
  let arr;
  try { arr = Function("return " + src.slice(at, k + 1))(); } catch { continue; }
  out[name] = arr.map(([tag, attrs]) => "<" + tag + Object.entries(attrs).filter(([a]) => a !== "key").map(([a, val]) => " " + a + '="' + String(val).replace(/"/g, "&quot;") + '"').join("") + "/>").join("");
}
const pascal = (k) => k.replace(/(^|-)([a-z0-9])/g, (_, __, c) => c.toUpperCase());
const icons = {};
for (const n of Object.keys(out).sort()) icons[pascal(n)] = out[n];
writeFileSync(new URL("../lucide.json.gz", import.meta.url), gzipSync(JSON.stringify(icons), { level: 9 }));
console.log(Object.keys(icons).length, "icons");
