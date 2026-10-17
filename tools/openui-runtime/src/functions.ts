// Independently implemented functions from documented positional contracts.
import { defineFunction } from "@openuidev/react-lang";
import { z } from "zod/v4";

const list = z.array(z.any());
const finite = (value: unknown) => {
  if (typeof value !== "number" || !Number.isFinite(value)) throw new Error("A finite number is required.");
  return value;
};
const field = (value: unknown, name: string): unknown => {
  if (!name) return value;
  if (["__proto__", "prototype", "constructor"].includes(name)) throw new Error("Unsafe property name.");
  return value !== null && typeof value === "object" && Object.hasOwn(value, name) ? (value as Record<string, unknown>)[name] : undefined;
};
function fn(name: string, shape: Record<string, z.ZodType>, run: (args: any) => unknown) {
  return defineFunction({ name, params: z.object(shape), returns: z.any(), description: `Local ${name} operation.`, fn: run });
}

export const compatibleFunctions = [
  fn("Append", { list, item: z.any() }, ({ list, item }) => [...list, item]),
  fn("Prepend", { list, item: z.any() }, ({ list, item }) => [item, ...list]),
  fn("Concat", { list, more: list }, ({ list, more }) => [...list, ...more]),
  fn("Remove", { list, field: z.string(), value: z.any() }, ({ list, field: name, value }) => list.filter((item: unknown) => field(item, name) !== value)),
  fn("RemoveAt", { list, index: z.number() }, ({ list, index }) => {
    const at = Math.trunc(finite(index)); const resolved = at < 0 ? list.length + at : at;
    return list.filter((_: unknown, i: number) => i !== resolved);
  }),
  fn("Update", { list, field: z.string(), value: z.any(), setField: z.string(), setValue: z.any() }, ({ list, field: name, value, setField, setValue }) => {
    field({}, setField);
    return list.map((item: unknown) => field(item, name) === value && item !== null && typeof item === "object" && !Array.isArray(item) ? { ...item, [setField]: setValue } : item);
  }),
  fn("Toggle", { list, item: z.any() }, ({ list, item }) => list.includes(item) ? list.filter((value: unknown) => value !== item) : [...list, item]),
  fn("Includes", { list: z.union([list, z.string()]), item: z.any() }, ({ list, item }) => list.includes(item)),
  fn("Find", { list, field: z.string(), value: z.any() }, ({ list, field: name, value }) => list.find((item: unknown) => field(item, name) === value) ?? null),
  fn("IndexOf", { list, item: z.any() }, ({ list, item }) => list.indexOf(item)),
  fn("Take", { list, n: z.number() }, ({ list, n }) => n < 0 ? list.slice(Math.trunc(n)) : list.slice(0, Math.trunc(n))),
  fn("Slice", { list: z.union([list, z.string()]), start: z.number(), end: z.number().optional() }, ({ list, start, end }) => list.slice(start, end)),
  fn("Unique", { list }, ({ list }) => [...new Set(list)]),
  fn("Reverse", { list }, ({ list }) => [...list].reverse()),
  fn("Range", { start: z.number(), end: z.number(), step: z.number().optional() }, ({ start, end, step = 1 }) => {
    finite(start); finite(end); finite(step);
    if (step === 0) throw new Error("Range step cannot be zero.");
    const size = Math.max(0, Math.ceil((end - start) / step));
    if (size > 10000) throw new Error("Range exceeds the 10,000-item allocation budget.");
    return Array.from({ length: size }, (_, i) => start + i * step);
  }),
  fn("Flatten", { list }, ({ list }) => list.flat()),
  fn("Chunk", { list, size: z.number() }, ({ list, size }) => {
    const width = Math.trunc(finite(size)); if (width < 1) throw new Error("Chunk size must be positive.");
    const result = []; for (let index = 0; index < list.length; index += width) result.push(list.slice(index, index + width)); return result;
  }),
  fn("Pow", { base: z.number(), exponent: z.number() }, ({ base, exponent }) => finite(Math.pow(finite(base), finite(exponent)))),
  fn("Sqrt", { number: z.number() }, ({ number }) => finite(number) < 0 ? null : Math.sqrt(number)),
  fn("Clamp", { number: z.number(), min: z.number(), max: z.number() }, ({ number, min, max }) => {
    finite(number); finite(min); finite(max); if (min > max) throw new Error("Clamp minimum exceeds maximum."); return Math.min(max, Math.max(min, number));
  }),
  fn("Percent", { part: z.number(), total: z.number() }, ({ part, total }) => finite(total) === 0 ? 0 : finite(finite(part) / total * 100)),
  fn("Lerp", { a: z.number(), b: z.number(), t: z.number() }, ({ a, b, t }) => finite(finite(a) + (finite(b) - a) * finite(t))),
  fn("Default", { value: z.any().optional(), fallback: z.any() }, ({ value, fallback }) => value == null || value === "" || (typeof value === "number" && Number.isNaN(value)) ? fallback : value),
  fn("Join", { list, sep: z.string().optional(), field: z.string().optional() }, ({ list, sep = ", ", field: name }) => list.map((item: unknown) => name ? field(item, name) : item).filter((item: unknown) => item != null).join(sep)),
  fn("Plural", { n: z.number(), word: z.string(), plural: z.string().optional() }, ({ n, word, plural }) => `${n} ${n === 1 ? word : (plural ?? `${word}s`)}`),
  fn("Pad", { value: z.union([z.string(), z.number()]), length: z.number(), char: z.string().optional() }, ({ value, length, char = "0" }) => {
    if (length > 16384) throw new Error("Pad exceeds the string allocation budget."); return String(value).padStart(length, char);
  }),
  fn("Upper", { text: z.string() }, ({ text }) => text.toUpperCase()),
  fn("Lower", { text: z.string() }, ({ text }) => text.toLowerCase()),
  fn("Trim", { text: z.string() }, ({ text }) => text.trim()),
  fn("Capitalize", { text: z.string() }, ({ text }) => text.replace(/\b\p{L}/gu, (letter: string) => letter.toLocaleUpperCase())),
  fn("Split", { text: z.string(), sep: z.string().optional() }, ({ text, sep = "," }) => text.split(sep).map((part: string) => part.trim()).filter(Boolean)),
  fn("Replace", { text: z.string(), find: z.string(), replacement: z.string() }, ({ text, find, replacement }) => text.replaceAll(find, replacement)),
  fn("Format", { value: z.number(), kind: z.enum(["number", "currency", "percent", "compact"]).optional(), decimals: z.number().optional(), currency: z.string().optional() }, ({ value, kind = "number", decimals, currency }) => {
    finite(value);
    const options: Intl.NumberFormatOptions = {};
    if (kind === "currency" || (kind === "compact" && currency)) { options.style = "currency"; options.currency = currency ?? "USD"; }
    if (kind === "percent") { options.style = "percent"; value /= 100; }
    if (kind === "compact") options.notation = "compact";
    if (decimals !== undefined) { if (!Number.isInteger(decimals) || decimals < 0 || decimals > 20) throw new Error("Decimals must be between 0 and 20."); options.minimumFractionDigits = decimals; options.maximumFractionDigits = decimals; }
    return new Intl.NumberFormat(undefined, options).format(value);
  }),
];
