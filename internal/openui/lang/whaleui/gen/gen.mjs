#!/usr/bin/env node
// Generates internal/openui/lang/whaleui/library.json, the Whale UI library
// definition consumed by the native OpenUI Lang runtime.
//
// Sources (both owned by the bundled Whale UI viewer):
//   * host.cjs  internal/desktop/openui/viewer/v2/host.cjs. Evaluated in an
//               isolated module scope to read the library JSON Schema
//               (`library.toJSONSchema()`), the genui component schema table,
//               the specialized-component type map and the genui limits.
//   * openui-guide.js  the DSH-generated prompt guide (OPENUI_GUIDE). Its
//               signatures provide prompt signatures and the $binding<T>
//               markers the JSON Schema does not carry.
//
// Usage:
//   node internal/openui/lang/whaleui/gen/gen.mjs [--host <host.cjs>] [--guide <openui-guide.js>] [--out <library.json>]
// Defaults: host from the repository, guide from $WHALE_OPENUI_GUIDE, else
// <repo>/runtime-rc4/vendor/deepseek-harness/packages/client/ui-chat/lib/types/openui-guide.js,
// else the same path under the main worktree of this repository.
import { createHash } from "node:crypto";
import { execFileSync } from "node:child_process";
import fs from "node:fs";
import { createRequire } from "node:module";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const here = path.dirname(fileURLToPath(import.meta.url));
const repo = path.resolve(here, "../../../../..");
const GUIDE_REL = "runtime-rc4/vendor/deepseek-harness/packages/client/ui-chat/lib/types/openui-guide.js";

function arg(name) {
  const i = process.argv.indexOf(name);
  return i > 0 ? process.argv[i + 1] : undefined;
}

function mainWorktree() {
  try {
    const common = execFileSync("git", ["-C", repo, "rev-parse", "--path-format=absolute", "--git-common-dir"], { encoding: "utf8" }).trim();
    return path.dirname(common);
  } catch {
    return undefined;
  }
}

function findGuide() {
  const explicit = arg("--guide") ?? process.env.WHALE_OPENUI_GUIDE;
  if (explicit) return explicit;
  for (const root of [repo, mainWorktree()].filter(Boolean)) {
    const p = path.join(root, GUIDE_REL);
    if (fs.existsSync(p)) return p;
  }
  throw new Error(`openui-guide.js not found; pass --guide or set WHALE_OPENUI_GUIDE (looked for ${GUIDE_REL})`);
}

const sha256 = (buf) => createHash("sha256").update(buf).digest("hex");

// ── host.cjs ──

function loadHost(hostPath) {
  let src = fs.readFileSync(hostPath, "utf8");
  const raw = src;
  const lib = /var (\w+)=(\w+)\.toJSONSchema\(\),/.exec(src);
  const genui = /var (\w+)=\{arrow:\w+\(\[\]/.exec(src);
  const types = /var (\w+)=\[\["Plot","plot",/.exec(src);
  const limits = /(\w+)=\{maxDepth:\d+,maxNodes:\d+/.exec(src);
  if (!lib || !genui || !types || !limits) throw new Error("host.cjs layout changed: library/genui markers not found");
  const inject = `var ${lib[1]}=${lib[2]}.toJSONSchema();globalThis.__whaleGen={library:${lib[1]},library0:${lib[2]},genui:${genui[1]},types:${types[1]},limits:${limits[1]}};var `;
  src = src.replace(lib[0], inject);
  const require = createRequire(hostPath);
  const Module = require("module");
  const m = new Module(hostPath);
  m.filename = hostPath;
  m.paths = [];
  m._compile(src, hostPath);
  const out = globalThis.__whaleGen;
  delete globalThis.__whaleGen;
  return { ...out, sha: sha256(raw) };
}

// ── openui-guide.js ──

function splitSignature(line) {
  const open = line.indexOf("(");
  if (open < 0) return null;
  let depth = 0;
  for (let i = open; i < line.length; i++) {
    if (line[i] === "(") depth++;
    else if (line[i] === ")" && --depth === 0) {
      const sig = line.slice(0, i + 1);
      let rest = line.slice(i + 1);
      let returns;
      const ret = /^\s*→\s*(.*?)(?:\s+—\s+|$)/.exec(rest);
      if (ret) {
        returns = ret[1];
        rest = rest.slice(ret[0].length - (rest.slice(ret[0].length - 3).startsWith(" — ") ? 3 : 0));
      }
      const desc = /—\s*(.*)$/.exec(rest);
      return { name: line.slice(0, open).replace(/^@/, ""), signature: sig.replace(/^@/, "") + (returns ? ` → ${returns}` : ""), description: desc ? desc[1] : "" };
    }
  }
  return null;
}

async function loadGuide(guidePath) {
  const mod = await import(pathToFileURL(guidePath).href);
  const guide = mod.OPENUI_GUIDE;
  if (typeof guide !== "string") throw new Error("openui-guide.js does not export OPENUI_GUIDE");
  const components = {};
  const functions = {};
  const actions = {};
  let section = "";
  for (const line of guide.split("\n")) {
    if (line.startsWith("## ")) section = line.slice(3).trim();
    if (section === "Component Signatures" && /^[A-Z]\w*\(/.test(line)) {
      const s = splitSignature(line);
      if (s) components[s.name] = s;
    } else if (section === "Built-in Functions" && /^@[A-Z]\w*\(/.test(line)) {
      const s = splitSignature(line);
      if (s) functions[s.name] = s;
    } else if (section.startsWith("Action") && /^- @[A-Z]\w*\(/.test(line)) {
      const s = splitSignature(line.slice(2));
      if (s) actions[s.name] = s;
    }
  }
  return { components, functions, actions, sha: sha256(fs.readFileSync(guidePath)) };
}

// ── main ──

const hostPath = path.resolve(arg("--host") ?? path.join(repo, "internal/desktop/openui/viewer/v2/host.cjs"));
const guidePath = path.resolve(findGuide());
const outPath = path.resolve(arg("--out") ?? path.join(here, "..", "library.json"));

const host = loadHost(hostPath);
const guide = await loadGuide(guidePath);
const schema = host.library;
const root = host.library0.root;

const components = {};
for (const name of Object.keys(schema.properties ?? {})) {
  const g = guide.components[name];
  const bindable = g ? [...g.signature.matchAll(/(\w+)\??: \$binding</g)].map((m) => m[1]) : [];
  components[name] = {
    signature: g?.signature ?? "",
    description: schema.$defs?.[name]?.description ?? g?.description ?? "",
    ...(bindable.length ? { bindable } : {}),
  };
}
const functions = {};
for (const name of Object.keys(schema.functions ?? {})) {
  const g = guide.functions[name];
  functions[name] = { signature: g?.signature ?? "", description: schema.functions[name].description ?? "" };
}
const builtinFunctions = Object.keys(guide.functions).filter((n) => !(n in functions));
const actions = {};
for (const name of Object.keys(schema.actions ?? {})) {
  const g = guide.actions[name];
  actions[name] = { signature: g?.signature ?? "", description: schema.actions[name].description ?? "" };
}

const missing = Object.keys(components).filter((n) => !components[n].signature);
const doc = {
  id: "whale-ui",
  version: "1",
  root,
  generator: "internal/openui/lang/whaleui/gen/gen.mjs",
  sources: {
    host: { path: "internal/desktop/openui/viewer/v2/host.cjs", sha256: host.sha },
    guide: { path: GUIDE_REL, sha256: guide.sha },
  },
  schema,
  components,
  functions,
  actions,
  builtins: builtinFunctions,
  genui: {
    types: Object.fromEntries(host.types.map(([name, type, description]) => [name, { type, description }])),
    schemas: host.genui,
    limits: host.limits,
  },
};
fs.writeFileSync(outPath, JSON.stringify(doc, null, 1) + "\n");
console.log(
  `wrote ${path.relative(repo, outPath)}: ${Object.keys(components).length} components (${missing.length} without guide signature), ` +
    `${Object.keys(functions).length} library functions, ${builtinFunctions.length} built-ins in guide, ${Object.keys(actions).length} actions, ` +
    `${Object.keys(host.genui).length} genui schemas, ${host.types.length} specialized components`,
);
