import fs from 'node:fs'; import path from 'node:path';
import * as E from './engine.mjs';
import * as Col from './color.mjs';
import { KEYBOARD } from './keyboard.mjs';
const OUT = process.argv[2]; const SHA = 'b2336c45f2b7355c42678ce786b6f14d62b819be';
const ex = JSON.parse(fs.readFileSync('extracted.json', 'utf8'));
const DOCS = path.join(E.SRCROOT, 'apps/ui/content/docs');

// ---------- named states ----------
const NAMED = [
  ['hover', a => a === ':hover'],
  ['pressed', a => a === ':active' || a === '[data-pressed]', [':hover']],
  ['focus-visible', a => /^:focus(-visible|-within)?$/.test(a) || (/^has\((?!\+|peer)/.test(a) && /:focus-(visible|within)/.test(a) && !/:not\(:focus/.test(a) && !/aria-invalid|disabled/.test(a))],
  ['disabled', a => /^(:disabled|\[data-disabled\]|\[aria-disabled=true\])$/.test(a) || (/^has\((?!\+)/.test(a) && /:disabled/.test(a) && !/focus|invalid/.test(a))],
  ['invalid', a => /^\[(aria-invalid(=true)?|data-invalid)\]$/.test(a) || (/^has\((?!\+)/.test(a) && /aria-invalid|data-invalid/.test(a) && !/focus|disabled/.test(a))],
  ['open', a => /^\[(data-open|data-popup-open|data-panel-open|data-expanded|data-state=open|aria-expanded=true)\]$/.test(a)],
  ['checked', a => /^(\[data-checked\]|:checked|\[data-state=checked\]|\[aria-checked=true\])$/.test(a)],
  ['unchecked', a => a === '[data-unchecked]'],
  ['indeterminate', a => a === '[data-indeterminate]'],
  ['selected', a => /^\[(data-selected|aria-selected=true|data-active|data-active=true|aria-current=page|data-state=active|data-state=selected)\]$/.test(a)],
  ['highlighted', a => a === '[data-highlighted]'],
  ['starting-style (enter from)', a => a === '[data-starting-style]' || a === '@starting-style'],
  ['ending-style (exit to)', a => a === '[data-ending-style]'],
  ['rtl', a => /dir\(rtl\)|dir=rtl/.test(a)],
  ['placeholder', a => a === '[data-placeholder]' || a === ':placeholder-shown'],
];
const MULTI = { 'has(:disabled,:focus-visible,[aria-invalid])': ['disabled', 'focus-visible', 'invalid'], 'has(:disabled,:focus-within,[aria-invalid])': ['disabled', 'focus-visible', 'invalid'], 'has(input:disabled,textarea:disabled,input:focus-visible,textarea:focus-visible,input[aria-invalid],textarea[aria-invalid])': ['disabled', 'focus-visible', 'invalid'] };
function keyOfAtom(a) { if (a === 'in(.dark)' || a === ':enabled') return null; for (const [n, f] of NAMED) if (f(a)) return n; return a; }
function scenarios(entries) {
  const atoms = new Set(); entries.forEach(e => E.atomsOf(e.pred, atoms));
  const groups = new Map(); // key -> set of atoms
  const add = (k, a) => { if (!groups.has(k)) groups.set(k, new Set()); groups.get(k).add(a); };
  for (const a of atoms) { if (MULTI[a]) { MULTI[a].forEach(k => add(k, a)); continue; } const k = keyOfAtom(a); if (k) add(k, a); }
  for (const [n, , extra] of NAMED) if (groups.has(n) && extra) extra.forEach(x => groups.get(n).add(x));
  const list = [...groups.entries()].map(([k, s]) => ({ name: k, atoms: [...s] }));
  // combos from entries requiring several keys at once
  const combos = new Map();
  for (const e of entries) { const pos = [...requiredKeys(e.pred)]; const u = [...new Set(pos)].sort(); if (u.length >= 2 && u.length <= 3) combos.set(u.join(' + '), u); }
  for (const [name, keys] of combos) { const s = new Set(); keys.forEach(k => (groups.get(k) || [k]).forEach(x => s.add(x))); list.push({ name, atoms: [...s] }); }
  return list.slice(0, 60);
}
function requiredKeys(n) { if (n.atom) { if (MULTI[n.atom]) return new Set(); const k = keyOfAtom(n.atom); return new Set(k ? [k] : []); } if (n.and) { const s = new Set(); n.and.forEach(x => requiredKeys(x).forEach(a => s.add(a))); return s; } if (n.or) { const sets = n.or.map(requiredKeys); return new Set([...(sets[0] || [])].filter(a => sets.every(s => s.has(a)))); } return new Set(); }
const DEFAULT_SET = [':enabled'];
let INHERITED = { desktop: {}, mobile: {} };
function compute(entries, target, atoms, mode, env = 'desktop') {
  const set = new Set([...DEFAULT_SET, ...atoms]); if (atoms.some(a => /disabled/.test(a))) set.delete(':enabled'); if (mode === 'dark') set.add('in(.dark)');
  const { props, unknownEnv } = E.cascade(entries, target, set, env); const r = E.resolveProps(props, mode, INHERITED[env]); return { r, unknownEnv };
}
const simp = r => Object.fromEntries(Object.entries(r).map(([k, v]) => [k, v]));
function merge(l, d) { const out = {}; for (const k of new Set([...Object.keys(l), ...Object.keys(d)])) { const a = l[k], b = d[k]; out[k] = { light: a ? a.value : null, dark: b ? b.value : null, expr: (a || b).expr, src: (a || b).src }; const tr = E.tokenRef((a || b).expr); if (tr) out[k].token = tr; if ((a || b).unresolved) out[k].unresolved = (a || b).unresolved; if ((a || b).inherited) out[k].inherited = (a || b).inherited; } return out; }
const isBase = e => e && typeof e.src === 'string' && e.src.startsWith('(');
function prune(m) { if (m['border-width'] && m['border-width'].light === '0' && m['border-width'].dark === '0' && isBase(m['border-width'])) { for (const k of ['border-width', 'border-style', 'border-color']) if (isBase(m[k])) delete m[k]; } if (isBase(m['outline-color']) && !m['outline-width'] && !m['outline']) delete m['outline-color']; return m; }
function diff(base, cur) { const out = {}; for (const k of new Set([...Object.keys(base), ...Object.keys(cur)])) { const a = base[k], b = cur[k]; if (!b) { out[k] = { light: '(unset)', dark: '(unset)', src: '(removed)' }; continue; } if (!a || a.light !== b.light || a.dark !== b.dark) out[k] = b; } return out; }
function analyzeClasses(classes) {
  const { entries, invalid } = E.entriesFor(classes);
  const targets = [...new Set(entries.map(e => e.target))].sort((a, b) => (a === 'self' ? -1 : b === 'self' ? 1 : a.localeCompare(b)));
  const sc = scenarios(entries); const res = {};
  const unknownEnv = new Set();
  for (const t of targets) {
    const L = compute(entries, t, [], 'light'), D = compute(entries, t, [], 'dark'); L.unknownEnv.forEach(x => unknownEnv.add(x));
    const def = prune(merge(L.r, D.r)); const states = {};
    for (const s of sc) { const l = compute(entries, t, s.atoms, 'light').r, d = compute(entries, t, s.atoms, 'dark').r; const df = diff(def, prune(merge(l, d))); if (Object.keys(df).length) states[s.name] = { atoms: s.atoms, props: df }; }
    const mob = prune(merge(compute(entries, t, [], 'light', 'mobile').r, compute(entries, t, [], 'dark', 'mobile').r)); const md = diff(def, mob);
    if (!Object.keys(def).length && !Object.keys(states).length && !Object.keys(md).length) continue;
    res[t] = { default: def, states, mobile: md };
  }
  return { targets: res, invalid, unknownEnv: [...unknownEnv] };
}
// ---------- part configs ----------
function partConfigs(part, cvas) {
  const base = []; const conds = []; const dyn = []; let cvaRef = null;
  for (const f of part.fragments) {
    if (f.dynamic) { dyn.push(f); continue; }
    if (f.cva) { if (!f.when.length) cvaRef = f; else conds.push({ when: f.when.join(' && '), cva: f }); continue; }
    if (!f.when.length) base.push(...f.classes); else conds.push({ when: f.when.join(' && '), classes: f.classes });
  }
  const configs = [];
  const cvaClasses = (cv, sel) => { const out = cv.base.split(/\s+/).filter(Boolean); for (const dim in cv.variants) { const v = sel[dim] ?? cv.defaultVariants[dim]; if (v !== undefined && cv.variants[dim][v] !== undefined) out.push(...cv.variants[dim][v].split(/\s+/).filter(Boolean)); }
    for (const cp of cv.compoundVariants) { if (Object.entries(cp.when).every(([k, v]) => String(sel[k] ?? cv.defaultVariants[k]) === String(v).replace(/^"|"$/g, ''))) out.push(...cp.className.split(/\s+/).filter(Boolean)); } return out; };
  const cv = cvaRef ? cvas[cvaRef.cva] : null;
  const baseAll = [...(cv ? cvaClasses(cv, {}) : []), ...base];
  configs.push({ id: 'base', label: cv ? 'base (cva defaults: ' + (Object.entries(cv.defaultVariants).map(([k, v]) => k + '=' + v).join(', ') || 'none') + ')' : 'base', classes: baseAll, sourceClasses: base });
  if (cv) for (const dim in cv.variants) for (const val in cv.variants[dim]) { if (String(cv.defaultVariants[dim]) === val) continue; configs.push({ id: `${dim}=${val}`, label: `${cvaRef.cva}: ${dim}=${val}`, classes: [...cvaClasses(cv, { [dim]: val }), ...base], variantClasses: cv.variants[dim][val] }); }
  if (cv) cv.compoundVariants.forEach((cp, i) => configs.push({ id: 'compound:' + Object.entries(cp.when).map(([k, v]) => k + '=' + v).join(','), label: 'compoundVariant ' + JSON.stringify(cp.when), classes: [...cvaClasses(cv, cp.when), ...base], variantClasses: cp.className }));
  for (const c of conds) { const extra = c.classes || cvaClasses(cvas[c.cva.cva], {}); configs.push({ id: 'when:' + c.when, label: 'when `' + c.when + '`', classes: [...baseAll, ...extra], variantClasses: extra.join(' ') }); }
  return { configs, dyn, cv, cvaRef };
}
// ---------- docs ----------
function docsFor(name) {
  const f = path.join(DOCS, 'components', name + '.mdx'); if (!fs.existsSync(f)) return null; const t = fs.readFileSync(f, 'utf8');
  const fm = /^---\n([\s\S]*?)\n---/.exec(t); const meta = {}; if (fm) { for (const l of fm[1].split('\n')) { const m = /^(\w+):\s*(.*)$/.exec(l.trim()); if (m && m[2]) meta[m[1]] = m[2].replace(/^"|"$/g, ''); } const d = /doc:\s*(\S+)/.exec(fm[1]); if (d) meta.baseUiDoc = d[1]; }
  const kb = t.split('\n').filter(l => /keyboard|arrow key|escape|enter key|tab key|shortcut|⌘|ctrl/i.test(l) && !/^import|<|```|npx/.test(l.trim())).map(l => l.trim()).slice(0, 12);
  return { ...meta, file: `apps/ui/content/docs/components/${name}.mdx`, keyboardMentions: kb };
}
// ---------- outputs ----------
const esc = s => String(s ?? '').replace(/\|/g, '\\|').replace(/\n/g, ' ');
const sv = x => x === null || x === undefined || x === '' ? '—' : '`' + esc(x) + '`'; const showVal = v => v.light === v.dark ? sv(v.light) : sv(v.light) + ' / ' + sv(v.dark);
const index = { meta: { source: 'https://github.com/cosscom/ui', commit: SHA, generated: new Date().toISOString().slice(0, 10), env: { desktop: 'viewport 1280px, pointer fine, hover capable, motion allowed', mobile: 'viewport 375px, pointer coarse, no hover' }, units: 'px (1rem=16px)', valueFormat: 'colors are sRGB hex (#rrggbb or #rrggbbaa, per-channel clipped); every value has light/dark; expr = Tailwind-generated CSS before var resolution; src = class that won the cascade' }, components: {} };
const KEYMETRICS = ['display', 'height', 'min-height', 'max-height', 'width', 'min-width', 'max-width', 'padding', 'padding-inline', 'padding-block', 'padding-inline-start', 'padding-inline-end', 'padding-top', 'padding-bottom', 'gap', 'column-gap', 'row-gap', 'border-width', 'border-top-width', 'border-bottom-width', 'border-color', 'border-radius', 'background-color', 'color', 'font-size', 'line-height', 'font-weight', 'letter-spacing', 'box-shadow', 'opacity', 'z-index', 'transition-property', 'transition-duration', 'transition-timing-function', 'animation'];
const usage = { typography: new Map(), opacity: new Map(), transitions: new Map(), radius: new Map(), z: new Map(), shadows: new Map(), borderWidth: new Map(), enterExit: [], focus: [] };
const bump = (m, k, who) => { if (!m.has(k)) m.set(k, new Set()); m.get(k).add(who); };
let compCount = 0; const mdFiles = [];
fs.mkdirSync(path.join(OUT, 'components'), { recursive: true });
for (const [key, d] of Object.entries(ex)) {
  INHERITED = { desktop: {}, mobile: {} };
  for (const c of d.components) for (const p of c.parts) { const { configs } = partConfigs(p, d.cvas); const cfg = configs[0]; if (!cfg || !cfg.classes.length) continue; const { entries } = E.entriesFor(cfg.classes);
    for (const env of ['desktop', 'mobile']) { const { props } = E.cascade(entries, 'self', new Set(DEFAULT_SET), env); for (const k in props) if (k.startsWith('--') && !k.startsWith('--tw-') && !INHERITED[env][k]) INHERITED[env][k] = { expr: props[k].value, from: (p.attrs['data-slot'] || c.name) + ' [' + props[k].cls + ']' }; } }
  const name = key.replace(/^chat\//, ''); compCount++;
  const docs = docsFor(name) || (key.startsWith('chat/') ? { title: name, file: 'apps/ui/content/docs/(root)/chat.mdx' } : null);
  const srcText = fs.readFileSync(path.join(E.SRCROOT, d.file), 'utf8');
  const primitives = [...srcText.matchAll(/from "(@base-ui\/react\/[\w-]+|lucide-react|react-day-picker|cmdk|vaul|input-otp|[^"]*registry\/default\/(?:ui|chat)\/[\w-]+)"/g)].map(m => m[1].replace(/.*registry\/default\//, '@/')); 
  const comp = { file: d.file, docs, dependsOn: [...new Set(primitives)], constants: d.constants, cva: d.cvas, exports: d.components.map(c => c.name), parts: [] };
  let md = `# ${docs?.title || name}\n\n`;
  md += `Source: \`${d.file}\` @ \`${SHA.slice(0, 12)}\`` + (docs?.file ? ` · docs: \`${docs.file}\`` : '') + (docs?.baseUiDoc ? ` · Base UI: ${docs.baseUiDoc}` : '') + `\n\n`;
  if (docs?.description) md += `> ${docs.description}\n\n`;
  md += `Depends on: ${comp.dependsOn.map(x => '`' + x + '`').join(', ') || '—'}\n\nExports: ${comp.exports.map(x => '`' + x + '`').join(', ')}\n\n`;
  if (Object.keys(d.constants).length) md += `Module constants: ${Object.entries(d.constants).map(([k, v]) => '`' + k + ' = ' + esc(v) + '`').join(', ')}\n\n`;
  md += `## Behavior & keyboard\n\n${KEYBOARD[name] || '_No interactive keyboard behavior beyond native focus order (inferred: presentational component)._'}\n\n`;
  if (docs?.keyboardMentions?.length) md += `Docs mentions:\n\n${docs.keyboardMentions.map(l => '- ' + l).join('\n')}\n\n`;
  md += `## How to read\n\nEach part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all \`sm:\` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show \`light\` / \`dark\` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. \`src\` is the class that wins the cascade. See \`../README.md\` for conventions.\n\n`;
  if (Object.keys(d.cvas).length) { md += `## Variant definitions (cva)\n\n`; for (const cv of Object.values(d.cvas)) { md += `### \`${cv.name}\` (line ${cv.line})\n\nDefaults: ${JSON.stringify(cv.defaultVariants)}\n\n\`\`\`txt\nbase: ${cv.base}\n`; for (const dim in cv.variants) for (const v in cv.variants[dim]) md += `${dim}.${v}: ${cv.variants[dim][v]}\n`; cv.compoundVariants.forEach(c => md += `compound ${JSON.stringify(c.when)}: ${c.className}\n`); md += '```\n\n'; } }
  md += `## Parts\n\n`;
  for (const c of d.components) {
    if (!c.parts.length) { md += `### \`${c.name}\`\n\nNo own classes (re-export / wrapper of the primitive)${Object.keys(c.params).length ? '; prop defaults: ' + Object.entries(c.params).map(([k, v]) => '`' + k + '=' + esc(v) + '`').join(', ') : ''}.\n\n`; comp.parts.push({ component: c.name, line: c.line, params: c.params, element: null, configs: [] }); continue; }
    for (const p of c.parts) {
      const slot = p.attrs['data-slot'] || null; const { configs, dyn, cv } = partConfigs(p, d.cvas);
      const partRec = { component: c.name, slot, element: p.tag === '(props object)' ? (c.defaultTagName ? 'useRender <' + c.defaultTagName + '>' : 'props object') : '<' + p.tag + '>', line: p.line, attrs: p.attrs, params: c.params, dynamicClassNames: dyn.map(x => ({ expr: x.dynamic, when: x.when })), configs: [] };
      md += `### \`${c.name}\` → ${slot ? '`[data-slot=' + slot + ']`' : '(no data-slot)'} · \`${partRec.element}\` (line ${p.line})\n\n`;
      const attrs = Object.entries(p.attrs).filter(([k]) => k !== 'data-slot'); if (attrs.length) md += `Attributes: ${attrs.map(([k, v]) => '`' + k + '=' + esc(typeof v === 'string' ? v : JSON.stringify(v)).slice(0, 120) + '`').join(', ')}\n\n`;
      if (Object.keys(c.params).length) md += `Prop defaults: ${Object.entries(c.params).map(([k, v]) => '`' + k + '=' + esc(v) + '`').join(', ')}\n\n`;
      if (dyn.length) md += `Dynamic className inputs (not statically resolvable): ${dyn.map(x => '`' + esc(x.dynamic).slice(0, 140) + '`' + (x.when.length ? ' when `' + esc(x.when.join(' && ')) + '`' : '')).join(', ')}\n\n`;
      let baseDef = null;
      for (const cfg of configs) {
        if (!cfg.classes.length) continue;
        const an = analyzeClasses(cfg.classes);
        const rec = { id: cfg.id, label: cfg.label, classes: cfg.classes.join(' '), invalidClasses: an.invalid, unknownEnv: an.unknownEnv, targets: an.targets };
        partRec.configs.push(rec);
        const who = `${name}/${slot || c.name}${cfg.id === 'base' ? '' : '[' + cfg.id + ']'}`;
        for (const [t, tv] of Object.entries(an.targets)) { const all = [tv.default, ...Object.values(tv.states).map(s => s.props)];
          for (const pr of all) { if (pr['font-size']) bump(usage.typography, [pr['font-size'].light, pr['line-height']?.light ?? 'inherit', pr['font-weight']?.light ?? 'inherit', pr['letter-spacing']?.light ?? 'normal'].join(' / '), who); if (pr['opacity']) bump(usage.opacity, pr['opacity'].light + ' ← ' + pr['opacity'].src, who); if (pr['border-radius']) bump(usage.radius, pr['border-radius'].light + ' ← ' + pr['border-radius'].src, who); if (pr['z-index']) bump(usage.z, pr['z-index'].light, who); if (pr['box-shadow'] && pr['box-shadow'].light !== 'none') bump(usage.shadows, pr['box-shadow'].src + ' ⇒ ' + pr['box-shadow'].light + (pr['box-shadow'].dark !== pr['box-shadow'].light ? ' | dark: ' + pr['box-shadow'].dark : ''), who); if (pr['border-width'] && pr['border-width'].light !== '0') bump(usage.borderWidth, pr['border-width'].light, who); }
          const tp = tv.default['transition-property']; if (tp) bump(usage.transitions, [tp.light, tv.default['transition-duration']?.light, tv.default['transition-timing-function']?.light].join(' · '), who + (t === 'self' ? '' : ' ' + t));
          if (tv.default['transition']) bump(usage.transitions, 'transition: ' + tv.default['transition'].light, who + (t === 'self' ? '' : ' ' + t));
          if (tv.default['animation']) bump(usage.transitions, 'animation: ' + tv.default['animation'].light, who);
          for (const [sn, s] of Object.entries(tv.states)) if (/starting-style|ending-style/.test(sn)) usage.enterExit.push({ part: who, target: t, state: sn, props: Object.fromEntries(Object.entries(s.props).map(([k, v]) => [k, v.light === v.dark ? v.light : v])) });
          const fv = tv.states['focus-visible']; if (fv && t === 'self') usage.focus.push({ part: who, props: Object.fromEntries(Object.entries(fv.props).map(([k, v]) => [k, v.light === v.dark ? v.light : v])) });
        }
        md += `#### ${cfg.label}\n\n\`\`\`txt\n${cfg.id === 'base' ? cfg.classes.join(' ') : cfg.variantClasses}\n\`\`\`\n\n`;
        if (an.invalid.length) md += `Classes that produce no CSS (custom markers / group names): ${an.invalid.map(x => '`' + x + '`').join(' ')}\n\n`;
        if (an.unknownEnv.length) md += `Rules behind unevaluated conditions (container queries etc.): ${an.unknownEnv.map(x => '`' + x + '`').join(', ')}\n\n`;
        for (const [t, tv] of Object.entries(an.targets)) {
          const isBase = cfg.id === 'base'; let def = tv.default;
          if (!isBase && baseDef && baseDef[t]) def = diff(baseDef[t].default, tv.default);
          const tname = t === 'self' ? 'element' : '`' + t + '`';
          if (Object.keys(def).length) {
            md += `**${tname} — ${isBase ? 'default' : 'default (changes vs base)'}**\n\n`;
            if (isBase && t === 'self') { const km = KEYMETRICS.filter(k => def[k]).map(k => `${k} ${showVal(def[k])}${def[k].token ? ' (' + def[k].token + ')' : ''}`); if (km.length) md += `Key metrics: ${km.join(' · ')}\n\n`; }
            md += `| property | value (light / dark) | token / expr | src |\n|---|---|---|---|\n`;
            for (const [k, v] of Object.entries(def)) md += `| ${k} | ${showVal(v)} | ${v.token ? '`' + v.token + '`' : '`' + esc(v.expr).slice(0, 110) + '`'} | \`${esc(v.src)}\` |\n`;
            md += '\n';
          }
          const sts = Object.entries(tv.states); if (sts.length) { md += `**${tname} — states**\n\n| state | property | value (light / dark) | src |\n|---|---|---|---|\n`; for (const [sn, s] of sts) for (const [k, v] of Object.entries(s.props)) md += `| ${esc(sn)} | ${k} | ${showVal(v)} | \`${esc(v.src)}\` |\n`; md += '\n'; }
          const mob = Object.entries(tv.mobile); if (mob.length) md += `**${tname} — mobile differences**: ${mob.map(([k, v]) => k + ' ' + showVal(v)).join(' · ')}\n\n`;
        }
        if (cfg.id === 'base') baseDef = an.targets;
      }
      comp.parts.push(partRec);
    }
  }
  // orphan class strings
  const orphans = d.orphans.filter(o => { const toks = o.text.trim().split(/\s+/); return toks.length && toks.every(t => E.classRules(t)) && toks.some(t => /[-:\[]/.test(t)); });
  if (orphans.length) { md += `## Other class strings in this file\n\nClass strings not bound directly to a JSX \`className\` (lookup tables, maps, helpers). Resolved per class (desktop, light / dark, no state):\n\n`; comp.otherClassStrings = [];
    for (const o of orphans) { md += `- line ${o.line}: context \`${esc(o.ctx).slice(0, 140)}\`\n\n  \`\`\`txt\n  ${o.text}\n  \`\`\`\n\n`; const an = analyzeClasses(o.text.trim().split(/\s+/)); comp.otherClassStrings.push({ line: o.line, context: o.ctx, classes: o.text, targets: an.targets });
      const flat = []; for (const [t, tv] of Object.entries(an.targets)) { for (const [k, v] of Object.entries(tv.default)) if (!/^\(/.test(v.src)) flat.push(`${t === 'self' ? '' : t + ' '}${k}: ${showVal(v)}`); for (const [sn, s] of Object.entries(tv.states)) for (const [k, v] of Object.entries(s.props)) flat.push(`[${sn}]${t === 'self' ? '' : ' ' + t} ${k}: ${showVal(v)}`); }
      if (flat.length) md += `  Resolved: ${flat.join(' · ')}\n\n`; } }
  index.components[key] = comp;
  const fname = (key.startsWith('chat/') ? name : name) + '.md';
  fs.writeFileSync(path.join(OUT, 'components', fname), md); mdFiles.push(fname);
}
const setObj = m => Object.fromEntries([...m.entries()].sort((a, b) => b[1].size - a[1].size).map(([k, v]) => [k, [...v].sort()]));
index.usage = { typography: setObj(usage.typography), opacity: setObj(usage.opacity), radius: setObj(usage.radius), zIndex: setObj(usage.z), shadows: setObj(usage.shadows), borderWidth: setObj(usage.borderWidth), transitions: setObj(usage.transitions), enterExit: usage.enterExit, focusVisible: usage.focus };
fs.writeFileSync(path.join(OUT, 'index.json'), JSON.stringify(index, null, 1));
fs.writeFileSync('usage.json', JSON.stringify(index.usage, null, 1));
fs.writeFileSync('keyframes.json', JSON.stringify(E.KEYFRAMES, null, 1));
console.log('components', compCount, 'md', mdFiles.length);
