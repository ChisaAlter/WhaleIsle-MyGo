// Independent implementations of the remaining documented function contracts.
import { defineFunction } from "@openuidev/react-lang";
import { z } from "zod/v4";
const array = z.array(z.any());
const dateSchema = z.union([z.string(), z.number()]);
const aggregation = z.enum(["sum", "avg", "count", "min", "max"]).optional();
const MAX_ITEMS = 10000;
function fn(name: string, shape: Record<string, z.ZodType>, run: (args: any) => unknown) {
  return defineFunction({ name, params: z.object(shape), returns: z.any(), description: `Local ${name} operation.`, fn: args => {
    for (const value of Object.values(args)) if (Array.isArray(value) && value.length > MAX_ITEMS) {
      throw new Error(`${name} exceeds the 10,000-item processing budget.`);
    }
    return run(args);
  } });
}
function number(value: unknown): number {
  if (typeof value !== "number" || !Number.isFinite(value)) throw new Error("A finite number is required.");
  return value;
}
function key(value: string) {
  if (["__proto__", "prototype", "constructor"].includes(value)) throw new Error("Unsafe property name.");
  return value;
}
function get(row: any, field: string): any { return field === "" ? row : row != null && Object.hasOwn(row, key(field)) ? row[field] : undefined; }
function set(row: any, field: string, value: number): any { return field === "" ? value : { ...row, [key(field)]: value }; }
function reduce(values: number[], mode = "sum") {
  if (mode === "count") return values.length;
  if (!values.length) return 0;
  if (mode === "min") return values.reduce((a, b) => Math.min(a, b));
  if (mode === "max") return values.reduce((a, b) => Math.max(a, b));
  const sum = values.reduce((a, b) => number(a + b), 0);
  return mode === "avg" ? sum / values.length : sum;
}
function date(value: string | number): Date {
  if (typeof value !== "string" && typeof value !== "number") throw new Error("A date string or timestamp is required.");
  const match = typeof value === "string" ? /^(\d{4})-(\d{2})-(\d{2})$/.exec(value) : null;
  const result = match ? new Date(0) : new Date(value);
  if (match) { result.setFullYear(Number(match[1]), Number(match[2]) - 1, Number(match[3])); result.setHours(0, 0, 0, 0); }
  if (!Number.isFinite(result.getTime()) || (match && (result.getFullYear() !== Number(match[1]) || result.getMonth() !== Number(match[2]) - 1 || result.getDate() !== Number(match[3])))) throw new Error("Invalid calendar date.");
  return result;
}
function calendarYear(value: Date): string {
  if (!Number.isFinite(value.getTime())) throw new Error("Date exceeds the supported calendar range.");
  const year = value.getFullYear();
  return year >= 0 && year <= 9999 ? String(year).padStart(4, "0") : `${year < 0 ? "-" : "+"}${String(Math.abs(year)).padStart(6, "0")}`;
}
const isoDate = (value: Date) => `${calendarYear(value)}-${String(value.getMonth() + 1).padStart(2, "0")}-${String(value.getDate()).padStart(2, "0")}`;
const monthKey = (value: Date) => `${calendarYear(value)}-${String(value.getMonth() + 1).padStart(2, "0")}`;
function calendarTime(value: Date): number {
  const result = new Date(0); result.setUTCFullYear(value.getFullYear(), value.getMonth(), value.getDate()); result.setUTCHours(0, 0, 0, 0);
  return number(result.getTime());
}
function startOf(value: Date, unit: string) {
  const result = new Date(value); result.setHours(0, 0, 0, 0);
  if (unit === "year") { result.setMonth(0, 1); }
  else if (unit === "month") result.setDate(1);
  else if (unit === "week") result.setDate(result.getDate() - (result.getDay() + 6) % 7);
  if (!Number.isFinite(result.getTime())) throw new Error("Date exceeds the supported calendar range.");
  return result;
}
function addMonths(value: Date, amount: number) {
  const result = new Date(value); const day = result.getDate(); result.setDate(1);
  result.setMonth(result.getMonth() + Math.trunc(number(amount)));
  const end = new Date(result); end.setMonth(end.getMonth() + 1, 0); result.setDate(Math.min(day, end.getDate()));
  if (!Number.isFinite(result.getTime())) throw new Error("Date exceeds the supported calendar range."); return result;
}
function wholeMonths(from: Date, to: Date): number {
  if (to < from) return -wholeMonths(to, from);
  let months = (to.getFullYear() - from.getFullYear()) * 12 + to.getMonth() - from.getMonth();
  if (addMonths(from, months) > to) months -= 1; return months;
}
const units: Record<string, [string, number]> = {
  mm: ["length", .001], cm: ["length", .01], m: ["length", 1], km: ["length", 1000], in: ["length", .0254], ft: ["length", .3048], yd: ["length", .9144], mi: ["length", 1609.344],
  mg: ["mass", .000001], g: ["mass", .001], kg: ["mass", 1], oz: ["mass", .028349523125], lb: ["mass", .45359237], st: ["mass", 6.35029318],
  ml: ["volume", .001], l: ["volume", 1], tsp: ["volume", .00492892159375], tbsp: ["volume", .01478676478125], floz: ["volume", .0295735295625], cup: ["volume", .2365882365], pt: ["volume", .473176473], qt: ["volume", .946352946], gal: ["volume", 3.785411784],
  "m/s": ["speed", 1], "km/h": ["speed", 1 / 3.6], mph: ["speed", .44704], kn: ["speed", 1852 / 3600],
  m2: ["area", 1], km2: ["area", 1000000], ft2: ["area", .09290304], ha: ["area", 10000], acre: ["area", 4046.8564224],
};

export const extraFunctions = [
  fn("Pick", { list: array }, ({ list }) => list.length ? list[Math.floor(Math.random() * list.length)] : null),
  fn("Shuffle", { list: array }, ({ list }) => { const result = [...list]; for (let i = result.length - 1; i > 0; i--) { const j = Math.floor(Math.random() * (i + 1)); [result[i], result[j]] = [result[j], result[i]]; } return result; }),
  fn("Random", { min: z.number(), max: z.number(), integer: z.boolean().optional() }, ({ min, max, integer }) => {
    number(min); number(max); if (min > max) throw new Error("Random minimum exceeds maximum.");
    if (!integer) { const fraction = Math.random(); return number((1 - fraction) * min + fraction * max); }
    const low = Math.ceil(min), high = Math.floor(max); if (low > high || !Number.isSafeInteger(low) || !Number.isSafeInteger(high)) throw new Error("Random integer range is invalid.");
    const width = high - low + 1; if (!Number.isSafeInteger(width)) throw new Error("Random integer range exceeds exact integer precision.");
    return low + Math.floor(Math.random() * width);
  }),
  fn("Allocate", { total: z.number(), weights: z.array(z.number()), decimals: z.number().optional() }, ({ total, weights, decimals = 2 }) => {
    number(total); if (!Number.isInteger(decimals) || decimals < 0 || decimals > 10) throw new Error("Allocate decimals must be an integer from 0 to 10.");
    if (weights.some((value: number) => number(value) < 0)) throw new Error("Allocation weights cannot be negative.");
    const scale = 10 ** decimals, amount = Math.round(Math.abs(total) * scale);
    if (!Number.isSafeInteger(amount)) throw new Error("Allocation exceeds exact integer precision.");
    if (!weights.length) { if (total !== 0) throw new Error("Allocation requires weights."); return []; }
    const largest = reduce(weights, "max");
    if (largest <= 0) { if (total !== 0) throw new Error("Allocation weights sum to zero."); return weights.map(() => 0); }
    // Normalize before adding or multiplying; finite large weights must never
    // turn the remaining-unit loop into an effectively unbounded allocation.
    const normalized = weights.map((weight: number) => weight / largest), sum = reduce(normalized);
    const exact = normalized.map((weight: number) => amount * (weight / sum));
    const shares = exact.map(Math.floor); let rest = amount - reduce(shares);
    const order = exact.map((value: number, index: number) => ({ index, remainder: value - shares[index] })).sort((a: any, b: any) => b.remainder - a.remainder || a.index - b.index);
    if (!Number.isSafeInteger(rest) || rest < 0 || rest > weights.length) throw new Error("Allocation exceeds reliable rounding precision.");
    for (let i = 0; rest > 0; i++, rest--) shares[order[i % order.length].index] += 1;
    return shares.map((share: number) => (total < 0 ? -share : share) / scale);
  }),
  fn("Payment", { principal: z.number(), rate: z.number(), periods: z.number() }, ({ principal, rate, periods }) => {
    number(principal); number(rate); number(periods); if (periods <= 0 || rate <= -100) throw new Error("Payment periods/rate are outside the valid range.");
    const r = rate / 100; return number(r === 0 ? principal / periods : principal * r / -Math.expm1(-periods * Math.log1p(r)));
  }),
  fn("Convert", { value: z.number(), from: z.string(), to: z.string() }, ({ value, from, to }) => {
    number(value); from = from.trim().toLowerCase(); to = to.trim().toLowerCase();
    const temperatures = ["c", "f", "k"];
    if (temperatures.includes(from) && temperatures.includes(to)) { const c = from === "c" ? value : from === "f" ? (value - 32) * 5 / 9 : value - 273.15; return number(to === "c" ? c : to === "f" ? c * 9 / 5 + 32 : c + 273.15); }
    const a = Object.hasOwn(units, from) ? units[from] : undefined, b = Object.hasOwn(units, to) ? units[to] : undefined;
    if (!a || !b || a[0] !== b[0]) throw new Error("Unknown or incompatible conversion units."); return number(value * a[1] / b[1]);
  }),
  fn("Duration", { seconds: z.number(), style: z.enum(["short", "clock"]).optional() }, ({ seconds, style = "short" }) => {
    number(seconds); const sign = seconds < 0 ? "-" : ""; const value = Math.floor(Math.abs(seconds)); const hours = Math.floor(value / 3600), minutes = Math.floor(value % 3600 / 60), sec = value % 60;
    if (style === "clock") return sign + (hours ? `${hours}:${String(minutes).padStart(2, "0")}:${String(sec).padStart(2, "0")}` : `${minutes}:${String(sec).padStart(2, "0")}`);
    return sign + (hours ? `${hours}h ${minutes}m` : minutes ? `${minutes}m ${String(sec).padStart(2, "0")}s` : `${sec}s`);
  }),
  fn("MixColor", { from: z.string(), to: z.string(), t: z.number() }, ({ from, to, t }) => {
    const parse = (text: string) => { const match = /^#([a-f\d]{3}|[a-f\d]{6})$/i.exec(text); if (!match) throw new Error("MixColor requires 3- or 6-digit hex colors."); const hex = match[1].length === 3 ? [...match[1]].map(c => c + c).join("") : match[1]; return [0, 2, 4].map(i => parseInt(hex.slice(i, i + 2), 16)); };
    const a = parse(from), b = parse(to), fraction = Math.min(1, Math.max(0, number(t))); return "#" + a.map((v, i) => Math.round(v + (b[i] - v) * fraction).toString(16).padStart(2, "0")).join("");
  }),
  fn("Now", {}, () => new Date().toISOString()),
  fn("Today", {}, () => isoDate(new Date())),
  fn("FormatDate", { date: dateSchema, style: z.enum(["date", "short", "long", "weekday", "month", "time", "datetime", "iso"]).optional() }, ({ date: input, style = "date" }) => {
    const value = date(input); if (style === "iso") return isoDate(value);
    const formats: Record<string, Intl.DateTimeFormatOptions> = { date: { year: "numeric", month: "short", day: "numeric" }, short: { month: "short", day: "numeric" }, long: { weekday: "long", year: "numeric", month: "long", day: "numeric" }, weekday: { weekday: "short" }, month: { year: "numeric", month: "short" }, time: { hour: "numeric", minute: "2-digit" }, datetime: { year: "numeric", month: "short", day: "numeric", hour: "numeric", minute: "2-digit" } };
    return new Intl.DateTimeFormat(undefined, formats[style]).format(value);
  }),
  fn("AddDays", { date: dateSchema, days: z.number() }, ({ date: input, days }) => { const value = date(input); value.setDate(value.getDate() + Math.trunc(number(days))); if (!Number.isFinite(value.getTime())) throw new Error("Date exceeds the supported calendar range."); return isoDate(value); }),
  fn("AddMonths", { date: dateSchema, months: z.number() }, ({ date: input, months }) => isoDate(addMonths(date(input), months))),
  fn("DaysBetween", { from: dateSchema, to: dateSchema }, ({ from, to }) => Math.round((calendarTime(date(to)) - calendarTime(date(from))) / 86400000)),
  fn("MonthsBetween", { from: dateSchema, to: dateSchema }, ({ from, to }) => wholeMonths(date(from), date(to))),
  fn("StartOf", { date: dateSchema, unit: z.enum(["week", "month", "year"]) }, ({ date: input, unit }) => isoDate(startOf(date(input), unit))),
  fn("GroupBy", { rows: array, key: z.string(), field: z.string().optional(), agg: aggregation }, ({ rows, key: group, field, agg = "sum" }) => {
    const groups = new Map<any, number[]>(); for (const row of rows) { const k = get(row, group); if (typeof k !== "string" && typeof k !== "number") throw new Error("Group keys must be strings or numbers."); const values = groups.get(k) ?? []; values.push(field && agg !== "count" ? number(get(row, field)) : 1); groups.set(k, values); }
    return [...groups].map(([key, values]) => ({ key, value: reduce(values, !field ? "count" : agg) }));
  }),
  fn("Pivot", { rows: array, rowKey: z.string(), colKey: z.string(), field: z.string().optional(), agg: aggregation }, ({ rows, rowKey, colKey, field, agg = "sum" }) => {
    key(rowKey); const groups = new Map<any, Map<string, number[]>>(), columns = new Set<string>();
    for (const row of rows) { const r = get(row, rowKey), c = String(get(row, colKey)); if (typeof r !== "string" && typeof r !== "number") throw new Error("Pivot row keys must be strings or numbers."); key(c); if (c === rowKey) throw new Error("Pivot column collides with its row-key field."); columns.add(c); const cols = groups.get(r) ?? new Map(); const values = cols.get(c) ?? []; values.push(field && agg !== "count" ? number(get(row, field)) : 1); cols.set(c, values); groups.set(r, cols); }
    if (groups.size * (columns.size + 1) > MAX_ITEMS) throw new Error("Pivot exceeds the 10,000-cell allocation budget.");
    return [...groups].map(([r, cols]) => Object.fromEntries([[rowKey, r], ...[...columns].map(c => [c, reduce(cols.get(c) ?? [], !field ? "count" : agg)])]));
  }),
  fn("Top", { rows: array, field: z.string(), n: z.number(), direction: z.enum(["desc", "asc"]).optional() }, ({ rows, field, n, direction = "desc" }) => [...rows].sort((a, b) => (number(get(a, field)) - number(get(b, field))) * (direction === "asc" ? 1 : -1)).slice(0, Math.max(0, Math.trunc(number(n))))),
  fn("Between", { rows: array, field: z.string(), min: z.any().optional(), max: z.any().optional() }, ({ rows, field, min, max }) => rows.filter((row: any) => { const value = get(row, field); if (!["string", "number"].includes(typeof value)) return false; return (min == null || value >= min) && (max == null || value <= max); })),
  fn("Bucket", { rows: array, dateField: z.string(), unit: z.enum(["day", "week", "month"]), field: z.string().optional(), agg: aggregation }, ({ rows, dateField, unit, field, agg = "sum" }) => {
    const groups = new Map<string, number[]>(); const dates = new Map<string, Date>();
    for (const row of rows) { const d = startOf(date(get(row, dateField)), unit); const k = unit === "month" ? monthKey(d) : isoDate(d); dates.set(k, d); const values = groups.get(k) ?? []; values.push(field && agg !== "count" ? number(get(row, field)) : 1); groups.set(k, values); }
    const mode = !field ? "count" : agg; const keys = [...groups.keys()].sort((a, b) => dates.get(a)!.getTime() - dates.get(b)!.getTime()); if (!keys.length) return [];
    if (mode === "sum" || mode === "count") {
      const cursor = new Date(dates.get(keys[0])!), end = dates.get(keys[keys.length - 1])!;
      const span = unit === "month" ? (end.getFullYear() - cursor.getFullYear()) * 12 + end.getMonth() - cursor.getMonth() : Math.round((calendarTime(end) - calendarTime(cursor)) / 86400000) / (unit === "week" ? 7 : 1);
      if (!Number.isFinite(span) || span + 1 > MAX_ITEMS) throw new Error("Bucket exceeds the 10,000-period allocation budget.");
      let visited = 0;
      while (cursor <= end) {
        if (++visited > MAX_ITEMS) throw new Error("Bucket exceeds the 10,000-period allocation budget.");
        const k = unit === "month" ? monthKey(cursor) : isoDate(cursor);
        if (!groups.has(k)) { groups.set(k, []); dates.set(k, new Date(cursor)); }
        if (cursor.getTime() === end.getTime()) break;
        const previous = cursor.getTime();
        if (unit === "month") cursor.setMonth(cursor.getMonth() + 1); else cursor.setDate(cursor.getDate() + (unit === "week" ? 7 : 1));
        if (!Number.isFinite(cursor.getTime()) || cursor.getTime() <= previous) throw new Error("Bucket cannot advance beyond the supported calendar range.");
      }
    }
    return [...groups].sort(([a], [b]) => dates.get(a)!.getTime() - dates.get(b)!.getTime()).map(([key, values]) => ({ key, value: reduce(values, mode) }));
  }),
  fn("Cumulative", { rows: array, field: z.string() }, ({ rows, field }) => { let sum = 0; return rows.map((row: any) => set(row, field, sum = number(sum + number(get(row, field))))); }),
  fn("MovingAvg", { rows: array, field: z.string(), window: z.number() }, ({ rows, field, window }) => { const size = Math.trunc(number(window)); if (size < 1) throw new Error("Moving-average window must be positive."); const values: number[] = []; let sum = 0; return rows.map((row: any, i: number) => { const value = number(get(row, field)); values.push(value); if (i >= size) sum = number(sum - values[i - size]); sum = number(sum + value); return set(row, field, sum / Math.min(i + 1, size)); }); }),
  fn("Change", { current: z.number(), previous: z.number() }, ({ current, previous }) => number(previous) === 0 ? null : number((number(current) - previous) / Math.abs(previous) * 100)),
  fn("Histogram", { values: z.array(z.number()), bins: z.number(), min: z.number().optional(), max: z.number().optional() }, ({ values, bins, min, max }) => {
    if (!Number.isInteger(bins) || bins < 1 || bins > 10000) throw new Error("Histogram bins must be an integer from 1 to 10,000."); values.forEach(number); if (!values.length && (min === undefined || max === undefined)) return [];
    const low = min === undefined ? reduce(values, "min") : number(min), high = max === undefined ? reduce(values, "max") : number(max); if (high < low) throw new Error("Histogram minimum exceeds maximum.");
    const width = high === low ? 1 : number(high - low) / bins; if (width <= 0) throw new Error("Histogram range is below representable precision."); const result = Array.from({ length: bins }, (_, i) => ({ key: `${low + i * width}–${low + (i + 1) * width}`, value: 0, start: low + i * width, end: low + (i + 1) * width }));
    for (const value of values) if (value >= low && value <= high) result[Math.min(bins - 1, Math.floor((value - low) / width))].value += 1; return result;
  }),
  fn("Stats", { values: z.array(z.number()) }, ({ values }) => { values.forEach(number); if (!values.length) return { count: 0, min: 0, max: 0, mean: 0, median: 0, stdev: 0 }; const sorted = [...values].sort((a, b) => a - b), count = sorted.length, mean = reduce(sorted.map(value => value / count)), half = Math.floor(count / 2); return { count, min: sorted[0], max: sorted[count - 1], mean, median: count % 2 ? sorted[half] : sorted[half - 1] / 2 + sorted[half] / 2, stdev: number(Math.sqrt(sorted.reduce((sum, value) => sum + (value - mean) ** 2 / count, 0))) }; }),
];
