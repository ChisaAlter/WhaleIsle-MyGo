// Minimal CSS value evaluator: var() substitution, calc(), color functions, color-mix, relative oklab alpha.
import * as C from './color.mjs';
const ROOT_PX = 16;
// split top-level by a separator char (comma or whitespace) respecting parens/quotes
export function splitTop(s, sep = ',') {
  const out = []; let depth = 0, cur = '', q = null;
  for (const ch of s) {
    if (q) { cur += ch; if (ch === q) q = null; continue; }
    if (ch === '"' || ch === "'") { q = ch; cur += ch; continue; }
    if (ch === '(' || ch === '[') depth++; if (ch === ')' || ch === ']') depth--;
    if (depth === 0 && (sep === ' ' ? /\s/.test(ch) : ch === sep)) { if (sep !== ' ' || cur.trim()) out.push(cur.trim()); cur = ''; continue; }
    cur += ch;
  }
  if (cur.trim() || sep !== ' ') out.push(cur.trim());
  return out;
}
function findClose(s, i) { let d = 0; for (let j = i; j < s.length; j++) { if (s[j] === '(') d++; else if (s[j] === ')') { d--; if (d === 0) return j; } } return -1; }
// substitute var() using lookup(name) -> string|undefined
export function subVars(s, lookup, depth = 0, missing = new Set()) {
  if (depth > 30) return s;
  let out = '', i = 0;
  while (i < s.length) {
    const k = s.indexOf('var(', i);
    if (k < 0 || (k > 0 && /[\w-]/.test(s[k - 1]))) { if (k < 0) { out += s.slice(i); break; } out += s.slice(i, k + 4); i = k + 4; continue; }
    const e = findClose(s, k + 3); if (e < 0) { out += s.slice(i); break; }
    const inner = s.slice(k + 4, e); const parts = splitTop(inner, ',');
    const name = parts[0].trim(); const fb = parts.length > 1 ? inner.slice(inner.indexOf(',') + 1) : undefined;
    let val = lookup(name);
    if (val === undefined || val === null) { if (fb !== undefined) val = fb.trim(); else { missing.add(name); val = null; } }
    out += s.slice(i, k);
    out += val === null ? `var(${name})` : subVars(val, lookup, depth + 1, missing);
    i = e + 1;
  }
  return out;
}
// value store
const parseNum = tok => { const m = /^(-?(?:\d+\.?\d*|\.\d+)(?:e-?\d+)?)([a-z%]*)$/i.exec(tok); if (!m) return null; const n = parseFloat(m[1]), u = m[2].toLowerCase();
  if (u === 'px') return { t: 'len', px: n }; if (u === 'rem') return { t: 'len', px: n * ROOT_PX }; if (u === '') return { t: 'num', n };
  if (u === '%') return { t: 'pct', p: n }; if (u === 'ms') return { t: 'time', ms: n }; if (u === 's') return { t: 'time', ms: n * 1000 }; if (u === 'deg') return { t: 'angle', deg: n };
  if (u === 'turn') return { t: 'angle', deg: n * 360 }; if (u === 'em') return { t: 'em', em: n }; return { t: 'raw', s: tok }; };
export function evaluate(input) {
  const store = []; const put = v => `\u27E6${store.push(v) - 1}\u27E7`;
  const get = tok => { const m = /^\u27E6(\d+)\u27E7$/.exec(tok.trim()); return m ? store[+m[1]] : null; };
  const atom = tok => { tok = tok.trim(); const g = get(tok); if (g) return g; if (/^#[0-9a-f]{3,8}$/i.test(tok)) return { t: 'color', c: C.parseHex(tok) };
    const lc = tok.toLowerCase(); if (C.NAMED[lc]) return { t: 'color', c: C.NAMED[lc] }; if (lc === 'currentcolor') return { t: 'raw', s: 'currentcolor', current: true }; if (lc === 'none') return { t: 'none' }; return parseNum(tok) || { t: 'raw', s: tok }; };
  let s = input;
  // wrap bare parenthesis groups: treat '(' not preceded by an identifier char as calc(
  for (let guard = 0; guard < 500; guard++) {
    const m = /([a-zA-Z-]*)\(([^()]*)\)/.exec(s); if (!m) break;
    const fname = m[1].toLowerCase(), args = m[2]; let rep;
    try { rep = fn(fname, args); } catch (e) { rep = null; }
    s = s.slice(0, m.index) + (rep ? put(rep) : put({ t: 'raw', s: m[0].replace(/\u27E6(\d+)\u27E7/g, (_, i) => fmt(store[+i])) })) + s.slice(m.index + m[0].length);
  }
  // finalize remaining tokens
  const parts = splitTop(s, ',').map(seg => splitTop(seg, ' ').map(t => { const a = atom(t); return a; }));
  const deph = t => t.replace(/\u27E6(\d+)\u27E7/g, (_, i) => fmt(store[+i]));
  return { parts, text: deph(parts.map(seg => seg.map(fmt).join(' ')).join(', ')) };
  function fn(name, args) {
    if (name === 'var') return null;
    if (name === '' || name === 'calc') return calc(args);
    if (name === 'rgb' || name === 'rgba') { const a = args.replace('/', ' / ').split(/[\s,\/]+/).filter(Boolean).map(atom); const ch = x => x.t === 'pct' ? x.p / 100 : x.n / 255; const al = a[3] ? (a[3].t === 'pct' ? a[3].p / 100 : a[3].n) : 1; return { t: 'color', c: C.fromSrgb(ch(a[0]), ch(a[1]), ch(a[2]), al) }; }
    if (name === 'oklch' || name === 'oklab') {
      if (/^\s*from\s/.test(args)) { // relative color: only "l a b / alpha" or "l c h / alpha" passthrough supported
        const rest = args.replace(/^\s*from\s+/, ''); const toks = splitTop(rest, ' '); const base = atom(toks[0]); if (base.t !== 'color') return null;
        const si = toks.indexOf('/'); const chans = toks.slice(1, si < 0 ? undefined : si).join(' '); if (!/^(l a b|l c h)$/.test(chans)) return null;
        let al = base.c.alpha; if (si >= 0) { const x = atom(toks[si + 1]); al = x.t === 'pct' ? x.p / 100 : x.t === 'num' ? x.n : (x.s === 'alpha' ? base.c.alpha : NaN); }
        if (Number.isNaN(al)) return null; return { t: 'color', c: { ...base.c, alpha: al } };
      }
      const [main, alp] = args.split('/'); const a = main.trim().split(/\s+/).map(atom); const al = alp ? (x => x.t === 'pct' ? x.p / 100 : x.n)(atom(alp)) : 1;
      const L = a[0].t === 'pct' ? a[0].p / 100 : a[0].n;
      if (name === 'oklch') { const Cc = a[1].t === 'pct' ? a[1].p / 100 * 0.4 : a[1].n; const H = a[2].t === 'angle' ? a[2].deg : (a[2].n || 0); return { t: 'color', c: C.fromOklch(L, Cc, H, al) }; }
      const v = x => x.t === 'pct' ? x.p / 100 * 0.4 : x.n; return { t: 'color', c: { L, a: v(a[1]), b: v(a[2]), alpha: al } };
    }
    if (name === 'color-mix') {
      const ps = splitTop(args, ','); const space = ps[0].replace(/^in\s+/, '').trim().split(/\s+/)[0];
      const it = ps.slice(1).map(p => { const t = splitTop(p, ' ').map(atom); const col = t.find(x => x.t === 'color'); const pc = t.find(x => x.t === 'pct'); return { col, pc: pc ? pc.p : undefined, bad: t.some(x => x.t === 'raw') }; });
      if (it.length !== 2 || it.some(x => !x.col || x.bad)) return null;
      let [p1, p2] = [it[0].pc, it[1].pc]; if (p1 === undefined && p2 === undefined) { p1 = 50; p2 = 50; } else if (p1 === undefined) p1 = 100 - p2; else if (p2 === undefined) p2 = 100 - p1;
      const sum = p1 + p2; if (sum === 0) return null; const mult = sum < 100 ? sum / 100 : 1;
      return { t: 'color', c: C.mix(space, it[0].col.c, p1 / sum, it[1].col.c, p2 / sum, mult) };
    }
    return null;
  }
  function calc(expr) {
    const toks = []; const re = /\s*(\u27E6\d+\u27E7|-?(?:\d+\.?\d*|\.\d+)(?:e-?\d+)?[a-z%]*|[+\-*/]|[^\s+*/]+)/gy; let m;
    while ((m = re.exec(expr)) && m[0].length) toks.push(m[1]);
    let i = 0; const vals = toks.map(t => /^[+\-*/]$/.test(t) ? t : atom(t));
    const peek = () => vals[i], next = () => vals[i++];
    const prim = () => { const v = next(); if (v === '-') { const x = prim(); return mul(x, { t: 'num', n: -1 }); } return v; };
    const term = () => { let a = prim(); while (peek() === '*' || peek() === '/') { const op = next(); const b = prim(); a = op === '*' ? mul(a, b) : div(a, b); } return a; };
    const exp = () => { let a = term(); while (peek() === '+' || peek() === '-') { const op = next(); const b = term(); a = add(a, b, op === '-' ? -1 : 1); } return a; };
    const r = exp(); if (i < vals.length || !r || r.t === 'raw' || r.t === 'color') throw new Error('calc');
    return r;
  }
  function key(v) { return v.t === 'len' ? 'px' : v.t === 'num' ? 'n' : v.t === 'pct' ? 'p' : v.t === 'time' ? 'ms' : v.t === 'angle' ? 'deg' : v.t === 'em' ? 'em' : null; }
  function mul(a, b) { if (a.t === 'num') [a, b] = [b, a]; if (b.t !== 'num') throw 0; const k = key(a); if (!k) throw 0; return { ...a, [k]: a[k] * b.n }; }
  function div(a, b) { if (b.t !== 'num') throw 0; const k = key(a); if (!k) throw 0; return { ...a, [k]: a[k] / b.n }; }
  function add(a, b, sgn) { if (a.t !== b.t) { if ((a.t === 'num' && a.n === 0)) return sgn < 0 ? mul(b, { t: 'num', n: -1 }) : b; if (b.t === 'num' && b.n === 0) return a; throw 0; } const k = key(a); if (!k) throw 0; return { ...a, [k]: a[k] + sgn * b[k] }; }
  function fmt(v) { if (!v) return '?'; const r = x => +(+x).toFixed(4); switch (v.t) { case 'len': return r(v.px) + 'px'; case 'num': return String(r(v.n)); case 'pct': return r(v.p) + '%'; case 'time': return r(v.ms) + 'ms'; case 'angle': return r(v.deg) + 'deg'; case 'em': return r(v.em) + 'em'; case 'color': return C.hex(v.c); case 'none': return 'none'; default: return v.s; } }
}
export const fmtColor = c => C.hex(c);
