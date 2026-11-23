#!/usr/bin/env node
// Records reference results of the Whale UI library functions by running the
// bundled viewer host.cjs implementations, for the Go differential test
// (testdata/function-cases.json). Determinism: Math.random is a fixed
// sequence, Intl uses en-US, the time zone is UTC and Date.now is fixed.
//
// Usage: node internal/openui/lang/whaleui/gen/fncases.mjs
import fs from "node:fs";
import { createRequire } from "node:module";
import path from "node:path";
import { fileURLToPath } from "node:url";

process.env.TZ = "UTC";
const here = path.dirname(fileURLToPath(import.meta.url));
const repo = path.resolve(here, "../../../../..");
const hostPath = path.join(repo, "internal/desktop/openui/viewer/v2/host.cjs");

export const RANDOM = [0.42, 0.07, 0.93, 0.5, 0.25, 0.75, 0.11, 0.99, 0.3, 0.6];
let ri = 0;
Math.random = () => RANDOM[ri++ % RANDOM.length];
const FIXED_NOW = Date.UTC(2026, 10, 23, 10, 30, 0);
const RealDate = Date;
globalThis.Date = class extends RealDate {
  constructor(...a) {
    super(...(a.length ? a : [FIXED_NOW]));
  }
  static now() {
    return FIXED_NOW;
  }
};
const NF = Intl.NumberFormat;
const DF = Intl.DateTimeFormat;
Intl.NumberFormat = function (_l, o) {
  return new NF("en-US", o);
};
Intl.DateTimeFormat = function (_l, o) {
  return new DF("en-US", { timeZone: "UTC", ...o });
};

let src = fs.readFileSync(hostPath, "utf8");
const lib = /var (\w+)=(\w+)\.toJSONSchema\(\),/.exec(src);
src = src.replace(lib[0], `var ${lib[1]}=${lib[2]}.toJSONSchema();globalThis.__fns=${lib[2]}.functions;var `);
const require = createRequire(hostPath);
const Module = require("module");
const m = new Module(hostPath);
m.paths = [];
m._compile(src, hostPath);
const fns = globalThis.__fns;

const rows = [
  { d: "2026-01-05", v: 10, k: "a" },
  { d: "2026-01-07", v: 5, k: "b" },
  { d: "2026-02-10", v: 7, k: "a" },
  { d: "2026-03-01", v: null, k: "c" },
];
const C = {
  Append: [{ list: [1, 2], item: 3 }, { list: [], item: { a: 1 } }],
  Prepend: [{ list: [1, 2], item: 0 }],
  Concat: [{ list: [1], more: [2, 3] }],
  Remove: [{ list: [{ id: 1 }, { id: 2 }], field: "id", value: 1 }, { list: [1, 2, 1], field: "", value: 1 }],
  RemoveAt: [{ list: [1, 2, 3], index: 1 }, { list: [1, 2, 3], index: -1 }, { list: [1, 2], index: 1.7 }],
  Update: [{ list: [{ id: 1, done: false }, { id: 2, done: false }], field: "id", value: 2, setField: "done", setValue: true }],
  Toggle: [{ list: [1, 2], item: 2 }, { list: [1, 2], item: 3 }],
  Includes: [{ list: [1, "a"], item: "a" }, { list: "hello", item: "ell" }, { list: "n1", item: 1 }, { list: [1], item: "1" }],
  Find: [{ list: [{ id: 1, n: "a" }, { id: 2, n: "b" }], field: "id", value: 2 }, { list: [{ id: 1 }], field: "id", value: 5 }],
  IndexOf: [{ list: ["a", "b"], item: "b" }, { list: [1], item: 2 }],
  Take: [{ list: [1, 2, 3, 4], n: 2 }, { list: [1, 2, 3, 4], n: -2 }, { list: [1, 2], n: 10 }],
  Slice: [{ list: [1, 2, 3, 4], start: 1, end: 3 }, { list: [1, 2, 3, 4], start: -2 }, { list: "héllo", start: 1, end: 3 }, { list: [1, 2, 3], start: 2, end: 1 }],
  Unique: [{ list: [1, 2, 1, "1", 2] }],
  Reverse: [{ list: [1, 2, 3] }],
  Range: [{ start: 0, end: 5 }, { start: 5, end: 0, step: -2 }, { start: 0, end: 1, step: 0.25 }, { start: 0, end: 5, step: 0 }],
  Pick: [{ list: ["a", "b", "c"] }, { list: [] }],
  Shuffle: [{ list: [1, 2, 3, 4, 5] }],
  Flatten: [{ list: [[1, 2], 3, [4, [5]]] }],
  Chunk: [{ list: [1, 2, 3, 4, 5], size: 2 }, { list: [1], size: 0 }],
  Pow: [{ base: 2, exponent: 10 }, { base: 9, exponent: 0.5 }],
  Sqrt: [{ number: 16 }, { number: -1 }],
  Random: [{ min: 1, max: 6, integer: true }, { min: 0, max: 10 }],
  Clamp: [{ number: 15, min: 0, max: 10 }, { number: -3, min: 0, max: 10 }],
  Percent: [{ part: 1, total: 4 }, { part: 1, total: 0 }],
  Lerp: [{ a: 0, b: 10, t: 0.25 }],
  Allocate: [{ total: 100, weights: [1, 1, 1] }, { total: 10, weights: [1, 2], decimals: 0 }, { total: -5, weights: [0, 0] }, { total: 7, weights: [] }],
  Payment: [{ principal: 1000, rate: 1, periods: 12 }, { principal: 1200, rate: 0, periods: 12 }, { principal: 1, rate: 1, periods: 0 }],
  Convert: [{ value: 1, from: "mi", to: "km" }, { value: 100, from: "C", to: "F" }, { value: 1, from: "kg", to: "m" }, { value: 32, from: "f", to: "c" }],
  Format: [
    { value: 1234.5678 },
    { value: 1234.5, kind: "currency", decimals: 2, currency: "USD" },
    { value: 45, kind: "percent" },
    { value: 1234567, kind: "compact" },
    { value: 1500, kind: "compact" },
    { value: -9.87654, decimals: 2 },
    { value: 1234, kind: "currency", currency: "EUR" },
  ],
  Join: [{ list: ["a", null, "b"] }, { list: [{ n: "x" }, { n: "y" }], sep: "/", field: "n" }],
  Plural: [{ n: 1, word: "item" }, { n: 2, word: "item" }, { n: 3, word: "child", plural: "children" }],
  Pad: [{ value: 7, length: 3 }, { value: "ab", length: 5, char: "xy" }, { value: "abc", length: 2 }],
  Upper: [{ text: "abc ß" }],
  Lower: [{ text: "ABC" }],
  Trim: [{ text: "  a b \n" }],
  Capitalize: [{ text: "hello  wide world" }],
  Split: [{ text: "a, b,,c " }, { text: "a|b", sep: "|" }],
  Replace: [{ text: "a-b-c", find: "-", replacement: "+" }],
  Duration: [{ seconds: 4800 }, { seconds: 245 }, { seconds: 45 }, { seconds: 4800, style: "clock" }, { seconds: 65, style: "clock" }],
  Default: [{ value: null, fallback: 1 }, { value: "", fallback: "x" }, { value: 0, fallback: 5 }, { value: false, fallback: true }, { fallback: 9 }],
  MixColor: [{ from: "#000", to: "#ffffff", t: 0.5 }, { from: "#ff0000", to: "#0000ff", t: 2 }, { from: "red", to: "#fff", t: 0 }],
  Today: [{}],
  Now: [{}],
  FormatDate: [{ date: "2026-01-05" }, { date: "2026-01-05", style: "long" }, { date: "2026-01-05", style: "iso" }, { date: "2026-01-05T15:04:00Z", style: "time" }, { date: "2026-01-05", style: "weekday" }, { date: "2026-01-05", style: "month" }, { date: "2026-01-05", style: "short" }, { date: "nope" }],
  AddDays: [{ date: "2026-01-30", days: 3 }, { date: "2026-03-01", days: -1 }],
  AddMonths: [{ date: "2026-01-31", months: 1 }, { date: "2024-02-29", months: 12 }, { date: "2026-05-15", months: -3 }],
  DaysBetween: [{ from: "2026-01-01", to: "2026-03-01" }, { from: "2026-03-01", to: "2026-01-01" }],
  MonthsBetween: [{ from: "2026-01-31", to: "2026-02-28" }, { from: "2026-01-15", to: "2026-04-14" }, { from: "2026-04-15", to: "2026-01-15" }],
  StartOf: [{ date: "2026-11-26", unit: "week" }, { date: "2026-11-26", unit: "month" }, { date: "2026-11-26", unit: "year" }],
  GroupBy: [{ rows, key: "k" }, { rows, key: "k", field: "v", agg: "avg" }, { rows, key: "k", field: "v", agg: "max" }],
  Pivot: [{ rows, rowKey: "k", colKey: "d", field: "v" }],
  Top: [{ rows, field: "v", n: 2 }, { rows, field: "d", n: 2, direction: "asc" }],
  Between: [{ rows, field: "d", min: "2026-01-06", max: "2026-02-28" }, { rows, field: "v", min: 6 }],
  Bucket: [{ rows, dateField: "d", unit: "month", field: "v" }, { rows, dateField: "d", unit: "week" }, { rows, dateField: "d", unit: "month", field: "v", agg: "avg" }],
  Cumulative: [{ rows, field: "v" }, { rows: [1, 2, 3], field: "" }],
  MovingAvg: [{ rows, field: "v", window: 2 }, { rows: [1, 2, 3, 4], field: "", window: 3 }],
  Change: [{ current: 120, previous: 100 }, { current: 1, previous: 0 }],
  Histogram: [{ values: [1, 2, 2, 3, 10], bins: 3 }, { values: [], bins: 2 }],
  Stats: [{ values: [3, 1, 2, 4] }, { values: [] }],
};

const out = [];
for (const [name, cases] of Object.entries(C)) {
  if (!fns[name]) throw new Error(`no function ${name}`);
  for (const args of cases) {
    ri = 0;
    let result, error;
    try {
      result = fns[name].fn(structuredClone(args));
    } catch (e) {
      error = String(e?.message ?? e);
    }
    out.push(error === undefined ? { fn: name, args, result: result === undefined ? null : result } : { fn: name, args, error });
  }
}
const missing = Object.keys(fns).filter((n) => !(n in C));
if (missing.length) throw new Error(`no cases for ${missing.join(", ")}`);
fs.writeFileSync(path.join(here, "..", "testdata", "function-cases.json"), JSON.stringify({ random: RANDOM, now: new RealDate(FIXED_NOW).toISOString(), cases: out }, null, 1) + "\n");
console.log(`wrote ${out.length} cases for ${Object.keys(C).length} functions`);
