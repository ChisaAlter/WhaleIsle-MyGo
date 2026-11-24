import fs from 'node:fs'; import path from 'node:path';
import * as E from './engine.mjs';
import * as C from './color.mjs';
import { evaluate, subVars } from './cssval.mjs';
const OUT = process.argv[2]; const SHA = 'b2336c45f2b7355c42678ce786b6f14d62b819be';
const usage = JSON.parse(fs.readFileSync('usage.json', 'utf8'));
const KF = JSON.parse(fs.readFileSync('keyframes.json', 'utf8'));
const raw = t => { const m = {}; for (const x of t.matchAll(/(--[\w-]+)\s*:\s*([^;{}]+);/g)) m[x[1]] = x[2].replace(/\s+/g, ' ').trim(); return m; };
function block(text, opener) { const i = text.indexOf(opener); let d = 0; const st = text.indexOf('{', i); for (let j = st; j < text.length; j++) { if (text[j] === '{') d++; else if (text[j] === '}') { d--; if (d === 0) return text.slice(st + 1, j); } } return ''; }
const RAW_ROOT = raw(block(E.globalsCss, ':root {')), RAW_DARK = raw(block(E.globalsCss, '.dark {'));
function colorInfo(expr, mode) {
  const lk = E.themeLookup(mode); const missing = new Set(); const sub = subVars(expr, lk, 0, missing); const ev = evaluate(sub);
  const atom = ev.parts.length === 1 && ev.parts[0].length === 1 ? ev.parts[0][0] : null;
  if (!atom || atom.t !== 'color') return { value: ev.text };
  const c = atom.c; const rgb = C.toSrgb(c); const [L, Ch, H] = C.toOklch(c);
  const r = x => +x.toFixed(4);
  return { hex: C.hex(c), hexGamutMapped: C.hex(c, 'map'), rgba: [...rgb.map(v => r(Math.min(1, Math.max(0, v)) * 255)), r(c.alpha)], oklch: [r(L), r(Ch), r(H)], outOfSrgbGamut: C.isOutOfGamut(c) };
}
const tokens = { $schema: 'coss-ui-tokens/1', meta: { source: 'https://github.com/cosscom/ui', commit: SHA, files: ['packages/ui/src/styles/globals.css', 'tailwindcss@4.1.17 theme.css (default theme)', 'packages/ui/src/fonts/index.ts'], notes: ['hex = sRGB, per-channel clipped (matches Chromium on sRGB displays); hexGamutMapped = CSS Color 4 OKLCh chroma-reduction gamut mapping; both equal for in-gamut colors', 'rgba = 0-255 channels + 0-1 alpha (clipped)', 'colors with alpha are to be composited over whatever is beneath (they are NOT pre-blended)', 'oklch = [L 0-1, C, H deg] of the resolved color'] } };
// palette
tokens.palette = {};
for (const [k, v] of Object.entries(E.THEME_DEFAULT)) if (k.startsWith('--color-')) tokens.palette[k.slice(8)] = { css: v, ...colorInfo(v, 'light') };
// semantic
tokens.semantic = { light: {}, dark: {} };
for (const [k, v] of Object.entries(RAW_ROOT)) { if (k === '--radius') continue; tokens.semantic.light[k.slice(2)] = { css: v, ...colorInfo(E.ROOT_VARS[k], 'light') }; }
for (const k of Object.keys(RAW_ROOT)) { if (k === '--radius') continue; const v = RAW_DARK[k]; tokens.semantic.dark[k.slice(2)] = { css: v ?? `(inherits :root) ${RAW_ROOT[k]}`, ...colorInfo(E.DARK_VARS[k] ?? E.ROOT_VARS[k], 'dark') }; }
tokens.semantic.tailwindColorAliases = Object.fromEntries(Object.entries(E.THEME_INLINE).filter(([k]) => k.startsWith('--color-')).map(([k, v]) => [k.slice(8), v]));
tokens.semantic.notes = ['Page body in apps/ui/app/layout.tsx uses `bg-sidebar font-sans text-foreground antialiased` (the site canvas is --sidebar; content surfaces use --background).', 'Light --destructive-foreground is red-700 in globals.css, but alert.mdx manual-install snippet says red-400 (button/badge/menu/field mdx say red-700). globals.css is treated as authoritative.', 'The host app maps coss `primary` to its own accent at runtime; coss own primary is neutral-800 (light) / neutral-100 (dark).', 'globals sets `--tw-shadow-color: #000` in @theme inline; it lands on :root only, and because Tailwind registers --tw-shadow-color with `inherits: false` it never reaches elements, so it has no visible effect. Shadows use each utility\'s own color fallback.'];
// radius
const px = e => E.resolveExpr(e, E.themeLookup('light')).text;
tokens.radius = { base: { css: RAW_ROOT['--radius'], px: px('var(--radius)') } };
for (const n of ['xs', 'sm', 'md', 'lg', 'xl', '2xl', '3xl', '4xl']) { const k = '--radius-' + n; const src = E.THEME_INLINE[k] !== undefined ? 'globals @theme inline' : 'tailwind default'; const css = E.THEME_INLINE[k] ?? E.THEME_DEFAULT[k]; tokens.radius[n] = { css, px: px(css), source: src }; }
tokens.radius.full = { css: 'calc(infinity * 1px)', px: '9999px (pill)', source: 'tailwind rounded-full' };
tokens.radius.innerPattern = 'Most bordered controls draw an inner ::before with radius calc(<outer radius> - 1px) (e.g. 9px for rounded-lg) carrying the 1px highlight shadow.';
tokens.radius.usage = usage.radius;
// spacing & sizes
tokens.spacing = { unit: { css: E.THEME_DEFAULT['--spacing'], px: 4 }, rule: 'Tailwind v4: <utility>-n = calc(var(--spacing) * n) = 4px × n (fractions allowed, e.g. 4.5 → 18px)' };
tokens.breakpoints = { sm: 640, md: 768, lg: 1024, xl: 1280, '2xl': 1536, '3xl': 1600, '4xl': 2000, note: 'px, min-width; 3xl/4xl added in globals @theme inline' };
tokens.containers = Object.fromEntries(Object.entries(E.THEME_DEFAULT).filter(([k]) => k.startsWith('--container-')).map(([k, v]) => [k.slice(12), { css: v, px: px(v) }]));
// typography
tokens.typography = {
  fontFamilies: {
    sans: { family: 'Cal Sans (variable, v2.010)', files: ['packages/ui/src/fonts/CalSansVF.woff2 (normal)', 'packages/ui/src/fonts/CalSansVF-Italic.woff2 (italic)'], weightRange: '400 700', cssVar: '--font-sans (set by next/font/local)', fallbackStackIfMissing: E.THEME_DEFAULT['--font-sans'] },
    heading: { aliasOf: 'sans', cssVar: '--font-heading: var(--font-sans)', usage: 'font-heading class on titles, with font-semibold/bold' },
    mono: { family: 'Paper Mono', files: ['packages/ui/src/fonts/PaperMono-Regular.woff2'], cssVar: '--font-mono', usage: 'code, kbd, samp, pre (globals @layer base)' },
    rendering: '-webkit-font-smoothing: antialiased (body class antialiased)',
    fontGuide: 'packages/ui/src/fonts/font-guide.md: Cal Sans guide; no letter-spacing anywhere. Components here only use the single variable Cal Sans file for all sizes.',
  },
  scale: Object.fromEntries(['xs', 'sm', 'base', 'lg', 'xl', '2xl', '3xl', '4xl', '5xl', '6xl', '7xl', '8xl', '9xl'].map(n => { const fs = px(E.THEME_DEFAULT['--text-' + n]); const lhr = E.resolveExpr(E.THEME_DEFAULT['--text-' + n + '--line-height'], E.themeLookup('light')).text; const lh = /^[\d.]+$/.test(lhr) ? +(parseFloat(lhr) * parseFloat(fs)).toFixed(3) + 'px' : lhr; return [n, { fontSize: fs, lineHeight: lh, lineHeightRatio: lhr, css: E.THEME_DEFAULT['--text-' + n] + ' / ' + E.THEME_DEFAULT['--text-' + n + '--line-height'] }]; })),
  weights: Object.fromEntries(Object.entries(E.THEME_DEFAULT).filter(([k]) => k.startsWith('--font-weight-')).map(([k, v]) => [k.slice(14), +v])),
  tracking: Object.fromEntries(Object.entries(E.THEME_DEFAULT).filter(([k]) => k.startsWith('--tracking-')).map(([k, v]) => [k.slice(11), v])),
  leading: Object.fromEntries(Object.entries(E.THEME_DEFAULT).filter(([k]) => k.startsWith('--leading-')).map(([k, v]) => [k.slice(10), +v])),
  responsiveRule: 'Most controls use text-base (16px/24px) below 640px and sm:text-sm (14px/20px) at ≥640px. Desktop values are the sm: ones.',
  usedCombinations: { format: 'font-size / line-height / font-weight / letter-spacing (desktop, light) → parts', pairs: usage.typography },
};
// shadows
tokens.shadows = { scale: {}, composition: 'Tailwind composes box-shadow = inset-shadow, inset-ring, ring-offset, ring, shadow (in that order). `shadow-X/N` replaces each layer color alpha with N%. `shadow-<color>/<a>` sets the shadow color (alpha multiplied). Rings: ring-N → 0 0 0 (N + offset)px ring-color; ring-offset-N → 0 0 0 Npx offset-color.', usedResolved: usage.shadows };
for (const [k, v] of Object.entries(E.THEME_DEFAULT)) if (/^--(shadow|inset-shadow|drop-shadow)-/.test(k)) tokens.shadows.scale[k.slice(2)] = { css: v, resolved: E.resolveExpr(v, E.themeLookup('light')).text };
for (const a of [5, 8]) for (const n of ['xs', 'sm', 'md', 'lg', 'xl', '2xl']) { const e = E.entriesFor([`shadow-${n}/${a}`]).entries; const { props } = E.cascade(e, 'self', new Set(), 'desktop'); tokens.shadows.scale[`shadow-${n}/${a}`] = { resolved: E.resolveProps(props, 'light')['box-shadow'].value, note: 'alpha modifier form used by coss' }; }
// focus
const sig = new Map(); for (const f of usage.focusVisible) { const k = JSON.stringify(f.props); if (!sig.has(k)) sig.set(k, { props: f.props, parts: [] }); sig.get(k).parts.push(f.part); }
tokens.focusRing = {
  summary: [
    'Buttons / toggles / tabs / checkboxes / switches (focus-visible): `ring-2 ring-ring ring-offset-1 ring-offset-background` → box-shadow `0 0 0 1px <background>, 0 0 0 3px <ring>` (1px gap in background color, then 2px ring). ring = neutral-400 #a1a1a1 light / neutral-500 #737373 dark.',
    'Text inputs / textarea / select trigger / input-group (control has :focus-visible child): border-color → ring, plus `ring-[3px] ring-ring/24` → `0 0 0 3px` ring at 24% alpha, outer 1px highlight ::before shadow removed. Invalid+focus: border destructive/64, ring destructive/16 (dark /24).',
    'Global default outline color (any element): outline-ring/50 (ring at 50%), but nearly all components set outline-none and draw rings with box-shadow.',
  ],
  ringToken: { light: tokens.semantic.light.ring, dark: tokens.semantic.dark.ring },
  offsetColorToken: 'background',
  derivedSignatures: [...sig.values()].sort((a, b) => b.parts.length - a.parts.length),
};
tokens.borders = { defaultWidth: '1px (class `border`)', defaultColorToken: 'border (globals: `* { border-color: var(--border) }`)', inputBorderToken: 'input', widthsUsed: usage.borderWidth };
tokens.opacity = { disabled: '64% (opacity-64 on disabled / data-disabled / has-disabled; most components)', placeholder: 'muted-foreground at 72% (placeholder:text-muted-foreground/72)', svgIconDefault: '80% (icons without explicit opacity class: [&_svg:not([class*=opacity-])]:opacity-80)', pressedOnTouch: '72% on a:active/button:active below md (apps/ui/app/globals.css, site-only)', used: usage.opacity };
tokens.zIndex = usage.zIndex;
// motion
tokens.motion = {
  defaults: { duration: E.THEME_DEFAULT['--default-transition-duration'], easing: E.THEME_DEFAULT['--default-transition-timing-function'], note: 'Tailwind `transition-*` without duration/ease classes use these' },
  easings: { 'ease-in': E.THEME_DEFAULT['--ease-in'], 'ease-out': E.THEME_DEFAULT['--ease-out'], 'ease-in-out': E.THEME_DEFAULT['--ease-in-out'], linear: 'linear', toastFeedback: 'cubic-bezier(0.5, 1, 0.89, 1)' },
  animations: { ...Object.fromEntries(Object.entries(E.THEME_DEFAULT).filter(([k]) => k.startsWith('--animate-')).map(([k, v]) => [k.slice(10), { css: v, source: 'tailwind default' }])), ...Object.fromEntries(Object.entries(E.THEME_INLINE).filter(([k]) => k.startsWith('--animate-')).map(([k, v]) => [k.slice(10), { css: v, source: 'globals @theme inline' }])) },
  keyframes: { fromGlobals: block(E.globalsCss, '@theme inline').match(/@keyframes[\s\S]*?\n  }\n/g).map(s => s.replace(/\n\s*/g, ' ').trim()), fromTailwindDefault: (fs.readFileSync('node_modules/tailwindcss/theme.css', 'utf8').match(/@keyframes[\s\S]*?\n  }\n/g) || []).map(s => s.replace(/\n\s*/g, ' ').trim()) },
  reducedMotion: 'globals: toast feedback animations disabled under prefers-reduced-motion: reduce; many components gate transforms with motion-safe:/motion-reduce: (see enterExit).',
  transitionsUsed: usage.transitions,
  enterExit: usage.enterExit,
  baseUiMechanics: 'Base UI sets data-starting-style on mount (one frame) and data-ending-style while unmounting, waiting for transitions/animations to finish. A native implementation should animate from the starting-style values to the default values on open and from default to ending-style values on close, using the part\'s transition-property/duration/easing.',
};
fs.writeFileSync(path.join(OUT, 'tokens.json'), JSON.stringify(tokens, null, 1));
const n = Object.keys(tokens.palette).length + Object.keys(tokens.semantic.light).length * 2 + Object.keys(tokens.radius).length + Object.keys(tokens.typography.scale).length + Object.keys(tokens.shadows.scale).length;
console.log('palette', Object.keys(tokens.palette).length, 'semantic', Object.keys(tokens.semantic.light).length, 'token count approx', n);
