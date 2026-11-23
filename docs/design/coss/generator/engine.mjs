// Tailwind class -> flattened rules -> predicate/target classification -> cascade -> resolved values
import fs from 'node:fs'; import path from 'node:path';
import { loadDS } from './ds.mjs';
import { evaluate, subVars, splitTop } from './cssval.mjs';
import * as Col from './color.mjs';
export const SRCROOT = path.resolve('../ui-b2336c45f2b7355c42678ce786b6f14d62b819be');
export const ds = await loadDS();

// ---------------- theme ----------------
function cssFnRewrite(v) {
  // --alpha(X / N%) -> color-mix(in oklab, X N%, transparent); --spacing(n) -> calc(var(--spacing) * n)
  let s = v;
  for (let g = 0; g < 50; g++) {
    const k = s.indexOf('--alpha('); if (k < 0) break; let d = 0, e = -1;
    for (let j = k + 7; j < s.length; j++) { if (s[j] === '(') d++; else if (s[j] === ')') { d--; if (d === 0) { e = j; break; } } }
    const inner = s.slice(k + 8, e); const sl = inner.lastIndexOf('/');
    s = s.slice(0, k) + `color-mix(in oklab, ${inner.slice(0, sl).trim()} ${inner.slice(sl + 1).trim()}, transparent)` + s.slice(e + 1);
  }
  s = s.replace(/--spacing\(([^)]+)\)/g, 'calc(var(--spacing) * $1)');
  return s.replace(/\s+/g, ' ').trim();
}
function parseVarBlock(text) { const m = {}; for (const x of text.matchAll(/(--[\w-]+)\s*:\s*([^;{}]+);/g)) m[x[1]] = cssFnRewrite(x[2]); return m; }
function block(text, opener) { const i = text.indexOf(opener); if (i < 0) return ''; let d = 0; const st = text.indexOf('{', i); for (let j = st; j < text.length; j++) { if (text[j] === '{') d++; else if (text[j] === '}') { d--; if (d === 0) return text.slice(st + 1, j); } } return ''; }
const twTheme = fs.readFileSync('node_modules/tailwindcss/theme.css', 'utf8');
export const globalsCss = fs.readFileSync(path.join(SRCROOT, 'packages/ui/src/styles/globals.css'), 'utf8');
export const THEME_DEFAULT = parseVarBlock(block(twTheme, '@theme default').replace(/@keyframes[\s\S]*?\n  }\n/g, ''));
const inlineBlock = block(globalsCss, '@theme inline');
export const THEME_INLINE = parseVarBlock(inlineBlock.replace(/@keyframes[^{]*\{(?:[^{}]*\{[^{}]*\})*[^{}]*\}/g, ''));
export const ROOT_VARS = parseVarBlock(block(globalsCss, ':root {'));
export const DARK_VARS = parseVarBlock(block(globalsCss, '.dark {'));
export const PROPERTY_INIT = {};
for (const k of Object.keys(THEME_INLINE)) if (THEME_INLINE[k] === `var(${k})`) delete THEME_INLINE[k]; // self refs (font-sans etc. set by next/font)
export function themeLookup(mode) {
  return name => { if (name.startsWith("--tw-")) return PROPERTY_INIT[name]; if (mode === "dark" && DARK_VARS[name] !== undefined) return DARK_VARS[name]; if (ROOT_VARS[name] !== undefined) return ROOT_VARS[name]; if (THEME_INLINE[name] !== undefined) return THEME_INLINE[name]; if (THEME_DEFAULT[name] !== undefined) return THEME_DEFAULT[name]; if (PROPERTY_INIT[name] !== undefined) return PROPERTY_INIT[name]; return undefined; };
}
export function resolveExpr(expr, lookup) { const missing = new Set(); const sub = subVars(expr, lookup, 0, missing); const ev = evaluate(sub); return { text: ev.text.replace(/(-?\d*\.?\d+)rem\b/g, (_, n) => (+n * 16) + 'px'), parts: ev.parts, missing: [...missing] }; }

// ---------------- CSS parsing ----------------
export function parseCss(text) {
  let i = 0;
  function parseItems() {
    const items = []; let buf = ''; let q = null, par = 0;
    while (i < text.length) {
      const ch = text[i];
      if (q) { buf += ch; if (ch === q) q = null; i++; continue; }
      if (ch === '"' || ch === "'") { q = ch; buf += ch; i++; continue; }
      if (ch === '(') par++; if (ch === ')') par--;
      if (par === 0 && ch === '{') { i++; const prelude = buf.trim(); buf = ''; const children = parseItems(); items.push({ type: 'rule', prelude, children }); continue; }
      if (par === 0 && ch === '}') { i++; if (buf.trim()) items.push(decl(buf)); return items; }
      if (par === 0 && ch === ';') { i++; if (buf.trim()) items.push(decl(buf)); buf = ''; continue; }
      buf += ch; i++;
    }
    if (buf.trim()) items.push(decl(buf)); return items;
  }
  function decl(b) { const k = b.indexOf(':'); let prop = b.slice(0, k).trim(), value = b.slice(k + 1).trim(); let important = false; if (/!important$/.test(value)) { important = true; value = value.replace(/\s*!important$/, ''); } return { type: 'decl', prop, value: value.replace(/\s+/g, ' '), important }; }
  return parseItems();
}
const cssCache = new Map();
export const KEYFRAMES = {};
export function classRules(cls) {
  if (cssCache.has(cls)) return cssCache.get(cls);
  const css = ds.candidatesToCss([cls])[0]; let res = null;
  if (css) {
    const items = parseCss(css); const rules = [];
    for (const it of items) {
      if (it.type !== 'rule') continue;
      if (it.prelude.startsWith('@property')) { const n = it.prelude.split(/\s+/)[1]; const iv = it.children.find(d => d.prop === 'initial-value'); if (iv && PROPERTY_INIT[n] === undefined) PROPERTY_INIT[n] = iv.value; continue; }
      if (it.prelude.startsWith('@keyframes')) { KEYFRAMES[it.prelude.split(/\s+/)[1]] = it.children; continue; }
      if (it.prelude.startsWith('@layer') || it.prelude.startsWith(':root')) continue;
      const walk = (node, levels) => { const decls = node.children.filter(c => c.type === 'decl'); if (decls.length) rules.push({ levels, decls }); for (const c of node.children) if (c.type === 'rule') walk(c, [...levels, c.prelude]); };
      walk(it, []);
    }
    res = { css, rules };
  }
  cssCache.set(cls, res); return res;
}

// ---------------- selector predicates ----------------
const normAtom = s => s.trim().replace(/^\*(?=[:\[.])/, '').replace(/="([^"]*)"/g, '=$1').replace(/='([^']*)'/g, '=$1').replace(/\s+/g, ' ');
function scanCompound(str) { // returns list of simple selector tokens
  const out = []; let i = 0; str = str.trim();
  while (i < str.length) {
    const ch = str[i];
    if (ch === '*') { i++; continue; }
    if (ch === '[') { let d = 0, j = i; for (; j < str.length; j++) { if (str[j] === '[') d++; else if (str[j] === ']') { d--; if (!d) break; } } out.push(str.slice(i, j + 1)); i = j + 1; continue; }
    if (ch === ':' || ch === '.' || ch === '#') { let j = i + 1; if (str[j] === ':') j++; while (j < str.length && /[\w\-\\\/]/.test(str[j])) j++; if (str[j] === '(') { let d = 0; for (; j < str.length; j++) { if (str[j] === '(') d++; else if (str[j] === ')') { d--; if (!d) break; } } j++; } out.push(str.slice(i, j)); i = j; continue; }
    let j = i; while (j < str.length && !/[:\[.#]/.test(str[j])) j++; out.push(str.slice(i, j)); i = j;
  }
  return out;
}
function topCombinatorSplit(item) { // split complex selector on last top-level combinator
  let d = 0; let last = -1, comb = null; const s = item.trim();
  for (let i = 0; i < s.length; i++) { const c = s[i]; if (c === '(' || c === '[') d++; else if (c === ')' || c === ']') d--; else if (d === 0 && (c === ' ' || c === '>' || c === '~' || c === '+')) { last = i; comb = c; } }
  if (last < 0) return null; let left = s.slice(0, last).trim(); let right = s.slice(last + 1).trim(); if (/[>~+]$/.test(left)) { comb = left.slice(-1); left = left.slice(0, -1).trim(); }
  return { left, right, comb };
}
export function parseList(str) { const items = splitTop(str, ','); const nodes = items.map(parseComplex); return nodes.length === 1 ? nodes[0] : { or: nodes }; }
function parseComplex(item) {
  const sp = topCombinatorSplit(item);
  if (sp && (!sp.left || sp.left.includes('&'))) return { atom: normAtom(item) };
  if (sp) { const ctx = sp.comb === '~' || sp.comb === '+' ? 'peer(' : 'in('; const a = { atom: ctx + normAtom(stripWhere(sp.left)) + ')' }; const r = parseCompound(sp.right); return r.and.length ? { and: [a, ...r.and] } : a; }
  return parseCompound(item);
}
const stripWhere = s => s.replace(/:where\(([^()]*)\)/g, '$1').replace(/^\*/, '');
export function parseCompound(str) {
  const toks = scanCompound(str); const and = [];
  for (const t of toks) {
    const m = /^:(not|is|where|matches|has)\((.*)\)$/s.exec(t);
    if (m) { const inner = parseList(m[2]); if (m[1] === 'not') and.push({ not: inner }); else if (m[1] === 'has') and.push(hasify(inner, m[2])); else and.push(inner); continue; }
    if (/^class\*=/.test(t) || /class\*=/.test(t)) { and.push({ atom: 'userclass(' + t.replace(/.*class\*=/, '').replace(/['"\]]/g, '') + ')' }); continue; }
    and.push({ atom: normAtom(t) });
  }
  return { and };
}
function hasify(node, raw) { if (node.or) return { or: node.or.map(n => hasify(n)) }; const t = textOf(node); return { atom: 'has(' + t + ')' }; }
export function textOf(n) { if (n.atom) return n.atom; if (n.not) return ':not(' + textOf(n.not) + ')'; if (n.or) return n.or.map(textOf).join(','); if (n.and) return n.and.map(textOf).join(''); return ''; }
export function atomsOf(n, acc = new Set(), neg = false) { if (n.atom) acc.add(n.atom); if (n.not) atomsOf(n.not, acc, !neg); if (n.or) n.or.forEach(x => atomsOf(x, acc, neg)); if (n.and) n.and.forEach(x => atomsOf(x, acc, neg)); return acc; }
export function positiveAtoms(n, acc = new Set(), neg = false) { if (n.atom && !neg) acc.add(n.atom); if (n.not) positiveAtoms(n.not, acc, !neg); if (n.or) n.or.forEach(x => positiveAtoms(x, acc, neg)); if (n.and) n.and.forEach(x => positiveAtoms(x, acc, neg)); return acc; }
function atomTrue(a, set) {
  if (set.has(a)) return true;
  // presence selector [x] true when [x=...] present; has([x]) when has([x=...])
  const pm = /^(has\()?\[([\w-]+)\](\))?$/.exec(a); if (pm) { for (const s of set) if (s.startsWith((pm[1] || '') + '[' + pm[2] + '=')) return true; }
  return false;
}
export function requiredAtoms(n) { if (n.atom) return new Set([n.atom]); if (n.and) { const s = new Set(); n.and.forEach(x => requiredAtoms(x).forEach(a => s.add(a))); return s; } if (n.or) { const sets = n.or.map(requiredAtoms); return new Set([...(sets[0] || [])].filter(a => sets.every(s => s.has(a)))); } return new Set(); }
export function evalPred(n, set) { if (n.atom) return atomTrue(n.atom, set); if (n.not) return !evalPred(n.not, set); if (n.or) return n.or.some(x => evalPred(x, set)); if (n.and) return n.and.every(x => evalPred(x, set)); return true; }

// classify rule levels
const ENV_PROFILES = { desktop: { width: 1280, hover: true, pointer: 'fine', reduced: false, container: null }, mobile: { width: 375, hover: false, pointer: 'coarse', reduced: false, container: null } };
export function evalEnv(prelude, env) {
  const p = prelude.replace(/^@\w+(-\w+)*\s*/, ''); const kind = /^@([\w-]+)/.exec(prelude)[1];
  if (kind === 'supports') return !/-webkit-touch-callout/.test(p); if (kind === 'container') return { unknown: true };
  if (kind !== 'media') return true;
  let m;
  if ((m = /\(width\s*(>=|<|>|<=)\s*([\d.]+)(rem|px)\)/.exec(p))) { const v = +m[2] * (m[3] === 'rem' ? 16 : 1); const w = env.width; return m[1] === '>=' ? w >= v : m[1] === '<' ? w < v : m[1] === '>' ? w > v : w <= v; }
  if (/\(hover:\s*hover\)/.test(p)) return env.hover; if ((m = /\(pointer:\s*(\w+)\)/.exec(p))) return env.pointer === m[1]; if ((m = /\(any-pointer:\s*(\w+)\)/.exec(p))) return env.pointer === m[1];
  if ((m = /prefers-reduced-motion:\s*([\w-]+)/.exec(p))) return m[1] === 'reduce' ? env.reduced : !env.reduced;
  if (/prefers-color-scheme/.test(p)) return false; if (/print/.test(p)) return false;
  return { unknown: true };
}
export function classifyLevels(levels) {
  const env = []; const preds = []; let target = '';
  for (const lv of levels) {
    if (lv.startsWith('@')) { if (lv.startsWith('@starting-style')) { if (target) target += ' @starting-style'; else preds.push({ atom: '@starting-style' }); } else env.push(lv); continue; }
    let s = lv.trim();
    const items = splitTop(s, ',');
    if (!target && items.length > 1 && items.every(x => x.includes('&'))) { const ors = items.map(x => { const c = classifyLevels([x]); return c.target === 'self' ? c.pred : { atom: 'complex(' + normAtom(x) + ')' }; }); preds.push({ or: ors }); continue; }
    let m = /^:is\(&\s*([>~+]?)\s*(.*)\)$/.exec(s);
    if (m && !target) { target = (m[1] ? '> ' : ' ') + m[2]; continue; }
    if (target) { target += s.replace(/^&/, ''); continue; }
    // top-level & index
    let d = 0, amp = -1; for (let i = 0; i < s.length; i++) { const c = s[i]; if (c === '(' || c === '[') d++; else if (c === ')' || c === ']') d--; else if (c === '&' && d === 0) { amp = i; break; } }
    if (amp < 0) { target = ' ' + s; continue; }
    const before = s.slice(0, amp).trim(); let after = s.slice(amp + 1);
    if (before) { const rawB = s.slice(0, amp); const tb = rawB.trim(); const lastCh = tb.slice(-1);
      if (lastCh === '+' || lastCh === '~') preds.push({ atom: (lastCh === '+' ? 'prev(' : 'prevsib(') + normAtom(stripWhere(tb.slice(0, -1))) + ')' });
      else if (lastCh === '>' ) preds.push({ atom: 'parent(' + normAtom(stripWhere(tb.slice(0, -1))) + ')' });
      else if (/\s$/.test(rawB)) preds.push({ atom: 'in(' + normAtom(stripWhere(tb)) + ')' });
      else preds.push(...parseCompound(rawB).and.map(a => a.atom && /^[a-z]+$/.test(a.atom) ? { atom: 'tag(' + a.atom + ')' } : a)); }
    if (/^\s*[>~+\s]/.test(after) && after.trim()) { target = after.replace(/^\s+/, ' '); continue; }
    // pseudo element split
    const pe = after.indexOf('::'); let pred = after, tgt = '';
    if (pe >= 0) { pred = after.slice(0, pe); tgt = after.slice(pe); }
    // descendant combinator at top-level inside after?
    const sp = topCombinatorSplit(pred);
    if (sp && !pred.startsWith(':') && false) { }
    if (sp && /^[:\[.]/.test(pred.trim()) && !/^:(is|where|not|has)\(/.test(pred.trim())) { tgt = ' ' + sp.right + tgt; pred = sp.left; }
    if (pred.trim()) { const c = parseCompound(pred); preds.push(...c.and); }
    if (tgt) target = tgt;
  }
  return { env, pred: { and: preds }, target: target.trim() || 'self' };
}
// specificity
export function specificity(sel) {
  const items = splitTop(sel, ','); let best = [0, 0, 0];
  for (const it of items) { const s = specOne(it); if (cmpSpec(s, best) > 0) best = s; } return best;
}
function specOne(sel) {
  let a = 0, b = 0, c = 0; let i = 0; const s = sel;
  while (i < s.length) {
    const ch = s[i];
    if (ch === '#') { a++; i++; while (i < s.length && /[\w-]/.test(s[i])) i++; continue; }
    if (ch === '.') { b++; i++; while (i < s.length && /[\w\-\\]/.test(s[i])) { if (s[i] === '\\') i++; i++; } continue; }
    if (ch === '[') { b++; let d = 0; for (; i < s.length; i++) { if (s[i] === '[') d++; else if (s[i] === ']') { d--; if (!d) break; } } i++; continue; }
    if (ch === ':') { let j = i + 1; const pe = s[j] === ':'; if (pe) j++; let st = j; while (j < s.length && /[\w-]/.test(s[j])) j++; const name = s.slice(st, j);
      let args = null; if (s[j] === '(') { let d = 0, k = j; for (; k < s.length; k++) { if (s[k] === '(') d++; else if (s[k] === ')') { d--; if (!d) break; } } args = s.slice(j + 1, k); j = k + 1; }
      if (pe) c++; else if (name === 'where') { } else if (['is', 'not', 'has', 'matches'].includes(name)) { const sp = specificity(args); a += sp[0]; b += sp[1]; c += sp[2]; } else if (['before', 'after', 'placeholder', 'selection', 'marker', 'backdrop', 'first-line', 'first-letter'].includes(name)) c++; else b++;
      i = j; continue; }
    if (/[a-zA-Z]/.test(ch)) { c++; while (i < s.length && /[\w-]/.test(s[i])) i++; continue; }
    i++;
  }
  return [a, b, c];
}
const cmpSpec = (x, y) => x[0] - y[0] || x[1] - y[1] || x[2] - y[2];
export function fullSelector(levels) { let s = '.c'; for (const lv of levels) { if (lv.startsWith('@')) continue; s = lv.includes('&') ? lv.replace(/&/g, s) : s + ' ' + lv; } return s; }

// ---------------- class entries ----------------
export function splitVariants(cls) { const out = []; let d = 0, cur = ''; for (const ch of cls) { if (ch === '[' || ch === '(') d++; if (ch === ']' || ch === ')') d--; if (ch === ':' && d === 0) { out.push(cur); cur = ''; continue; } cur += ch; } out.push(cur); return { variants: out.slice(0, -1), utility: out[out.length - 1] }; }
export function entriesFor(classes) {
  // returns {entries, invalid}
  const valid = classes.filter(c => classRules(c)); const invalid = classes.filter(c => !classRules(c));
  let order = []; try { order = ds.getClassOrder(valid); } catch { order = valid.map((c, i) => [c, BigInt(i)]); }
  const ordMap = new Map(order.map(([c, o]) => [c, o === null ? -1n : o]));
  const entries = [];
  for (const cls of valid) {
    const { rules } = classRules(cls);
    rules.forEach((r, ri) => { const cl = classifyLevels(r.levels); const spec = specificity(fullSelector(r.levels)); r.decls.forEach((dcl, di) => entries.push({ cls, ...cl, spec, order: ordMap.get(cls) ?? 0n, idx: ri * 100 + di, prop: dcl.prop, value: dcl.value, important: dcl.important || cls.startsWith('!') || cls.endsWith('!') })); });
  }
  return { entries, invalid };
}
export const BASE_ENTRIES = [
  { cls: '(preflight) *', prop: 'border-width', value: '0', layer: 0 }, { cls: '(preflight) *', prop: 'border-style', value: 'solid', layer: 0 },
  { cls: '(globals @layer base) * border-border', prop: 'border-color', value: 'var(--border)', layer: 0 },
  { cls: '(globals @layer base) * outline-ring/50', prop: 'outline-color', value: 'color-mix(in oklab, var(--ring) 50%, transparent)', layer: 0 },
].map(e => ({ ...e, env: [], pred: { and: [] }, target: 'self', spec: [0, 0, 0], order: -1n, idx: 0, important: false }));
export function cascade(entries, target, set, envName) {
  const env = ENV_PROFILES[envName]; const unknownEnv = new Set();
  const applicable = [];
  for (const e of [...(target === 'self' || target.startsWith('::') ? BASE_ENTRIES : []), ...entries]) {
    if (e.target !== target) continue;
    let ok = true; for (const p of e.env) { const r = evalEnv(p, env); if (r && r.unknown) { unknownEnv.add(p); ok = false; } else if (!r) ok = false; }
    if (!ok) continue; if (!evalPred(e.pred, set)) continue; applicable.push(e);
  }
  applicable.sort((x, y) => (x.important - y.important) || ((x.layer ?? 1) - (y.layer ?? 1)) || cmpSpec(x.spec, y.spec) || (x.order < y.order ? -1 : x.order > y.order ? 1 : 0) || x.idx - y.idx);
  const props = {}; for (const e of applicable) props[e.prop] = e;
  return { props, unknownEnv: [...unknownEnv] };
}
export function resolveProps(props, mode, inherited = {}) {
  const theme = themeLookup(mode); const custom = {}; for (const k in props) if (k.startsWith('--')) custom[k] = props[k].value;
  const usedInh = new Set(); const lookup = n => { if (custom[n] !== undefined) return custom[n]; const t = theme(n); if (t !== undefined) return t; if (inherited[n]) { usedInh.add(n); return inherited[n].expr; } return undefined; };
  const out = {};
  for (const k in props) { if (k.startsWith('--')) continue; const e = props[k]; const r = resolveExpr(e.value, lookup); out[k] = { value: cleanShadow(k, r.text).replace(/calc\(infinity \* 1px\)/g, '9999px'), expr: e.value, src: e.cls }; if (r.missing.length) out[k].unresolved = r.missing; if (usedInh.size) { out[k].inherited = [...usedInh].map(n => n + ' from ' + inherited[n].from + ' (inferred: ancestor part)'); usedInh.clear(); } }
  const fs = out['font-size'] && /^[\d.]+px$/.test(out['font-size'].value) ? parseFloat(out['font-size'].value) : null;
  if (fs && out['line-height'] && /^[\d.]+$/.test(out['line-height'].value)) out['line-height'].value = +(parseFloat(out['line-height'].value) * fs).toFixed(3) + 'px';
  if (fs && out['letter-spacing'] && /^-?[\d.]+em$/.test(out['letter-spacing'].value)) out['letter-spacing'].value += ' (' + +(parseFloat(out['letter-spacing'].value) * fs).toFixed(3) + 'px)';
  return out;
}
function cleanShadow(prop, text) {
  if (prop !== 'box-shadow' && prop !== 'text-shadow') return text;
  const layers = splitTop(text, ',').filter(l => !/#[0-9a-f]{6}00$/.test(l.trim()) && !/^0 0 #0000$/.test(l.trim()) && !/^(inset )?0 0 0 0px #\w+$/.test(l.trim()));
  return layers.length ? layers.join(', ') : 'none';
}
export function tokenRef(expr) { // summarize token usage: var(--x) -> x ; color-mix(in oklab, var(--x) N%, transparent) -> x/N%
  let m = /^var\(--([\w-]+)\)$/.exec(expr); if (m) { const n = '--' + m[1]; if (m[1].startsWith('tw-')) return null; return (ROOT_VARS[n] !== undefined || THEME_INLINE[n] !== undefined || THEME_DEFAULT[n] !== undefined) ? m[1] : 'runtime var ' + n; }
  m = /^color-mix\(in oklab, var\(--([\w-]+)\) ([\d.]+%), transparent\)$/.exec(expr); if (m) return m[1] + '/' + m[2];
  return null;
}
