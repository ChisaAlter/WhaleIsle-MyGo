// Renders the coss reference side of the component gallery in headless
// Chromium: every specimen of the manifest the Go gallery wrote, as coss's
// own markup and class strings (resolved from docs/design/coss/index.json,
// merged with tailwind-merge as coss's cn() does) compiled by
// tailwindcss@4.1.17 against coss's globals.css. States are data
// attributes in the markup, plus forced pseudo-classes (:hover, :active,
// :focus-visible, :focus-within) through CDP.
//
//   node ref.mjs <gallery-dir> <globals.css>
//
// Needs tailwindcss@4.1.17, tailwind-merge@3 and playwright-core resolvable
// from NODE_PATH or the working directory, and a Chromium headless shell
// (CHROMIUM_PATH, else the newest one in %LOCALAPPDATA%\ms-playwright).
// Headless only: it never opens a window or takes input focus.
import fs from 'node:fs';
import path from 'node:path';
import { createRequire } from 'node:module';

const require = createRequire(path.join(process.cwd(), 'noop.js'));
const { compile } = require('tailwindcss');
const { twMerge } = require('tailwind-merge');
const { chromium } = require('playwright-core');

const [dir, globalsPath] = process.argv.slice(2);
if (!dir || !globalsPath) {
  console.error('usage: node ref.mjs <gallery-dir> <globals.css>');
  process.exit(2);
}
const repo = path.resolve(path.dirname(new URL(import.meta.url).pathname.replace(/^\/([A-Za-z]:)/, '$1')), '../..');
const index = JSON.parse(fs.readFileSync(path.join(repo, 'docs/design/coss/index.json'), 'utf8'));
const manifest = JSON.parse(fs.readFileSync(path.join(dir, 'manifest.json'), 'utf8'));

function cva(spec) {
  // cva:<component>.<cvaName>:k=v,k=v
  const [ref, args = ''] = spec.split(':');
  const [comp, name] = ref.split('.');
  const def = index.components[comp]?.cva?.[name];
  if (!def) throw new Error('no cva ' + spec);
  const chosen = { ...(def.defaultVariants || {}) };
  for (const kv of args.split(',').filter(Boolean)) {
    const [k, v] = kv.split('=');
    chosen[k] = v;
  }
  const parts = [def.base];
  for (const [k, v] of Object.entries(chosen)) if (def.variants?.[k]?.[v] !== undefined) parts.push(def.variants[k][v]);
  for (const cv of def.compoundVariants || []) {
    const { class: cls, className, ...cond } = cv;
    if (Object.entries(cond).every(([k, v]) => (Array.isArray(v) ? v.includes(chosen[k]) : chosen[k] === v))) parts.push(cls || className || '');
  }
  return twMerge(...parts);
}

function part(spec) {
  // part:<component>/<slot or component name>[@configId[@configId…]]: the
  // configs' classes merged in order (base when none is named).
  const [ref, ...cfgIds] = spec.split('@');
  const cut = ref.lastIndexOf('/');
  const comp = ref.slice(0, cut), slotRef = ref.slice(cut + 1);
  const [slot, nth] = slotRef.split('#');
  const c = index.components[comp];
  if (!c) throw new Error('no component ' + spec);
  // slot#n picks the n-th (0-based) part of that component name.
  const p = nth !== undefined ? c.parts.filter((x) => x.component === slot)[Number(nth)] : c.parts.find((x) => x.slot === slot) || c.parts.find((x) => x.component === slot);
  if (!p) throw new Error('no part ' + spec);
  const classes = (cfgIds.length ? cfgIds : ['base']).map((id) => {
    const cfg = p.configs.find((x) => x.id === id);
    if (!cfg) throw new Error('no config ' + id + ' in ' + ref + ': ' + p.configs.map((x) => x.id).join(' | '));
    return cfg.classes;
  });
  return twMerge(...classes);
}

function expand(html) {
  return html.replace(/\{\{(cva|part):([^}|]+)(?:\|([^}]*))?\}\}/g, (_, kind, spec, extra) => {
    const base = kind === 'cva' ? cva(spec) : part(spec);
    return extra ? twMerge(base, extra) : base;
  });
}

function classesOf(html) {
  const set = new Set();
  for (const m of html.matchAll(/class="([^"]*)"/g)) for (const c of m[1].split(/\s+/)) if (c) set.add(c);
  return [...set];
}

// theme(static) keeps every palette variable: globals.css's .dark block
// mixes --color-neutral-950 and friends, which no class names.
let globals = fs.readFileSync(globalsPath, 'utf8').replace(/^@source.*$/gm, '').replace('@import "tailwindcss";', '@import "tailwindcss" theme(static);');
const twDir = path.dirname(require.resolve('tailwindcss/package.json'));
async function css(candidates) {
  const compiler = await compile(globals, {
    base: twDir,
    loadStylesheet: async (id, base) => {
      let p = id === 'tailwindcss' ? path.join(twDir, 'index.css') : path.resolve(base, id);
      if (id.startsWith('tailwindcss/')) p = path.join(twDir, id.slice('tailwindcss/'.length));
      return { path: p, base: path.dirname(p), content: fs.readFileSync(p, 'utf8') };
    },
  });
  return compiler.build(candidates);
}

function chromiumPath() {
  if (process.env.CHROMIUM_PATH) return process.env.CHROMIUM_PATH;
  const root = path.join(process.env.LOCALAPPDATA || '', 'ms-playwright');
  const dirs = fs.readdirSync(root).filter((d) => d.startsWith('chromium_headless_shell-')).sort();
  for (const d of dirs.reverse()) {
    for (const sub of fs.readdirSync(path.join(root, d))) {
      const exe = path.join(root, d, sub, 'chrome-headless-shell.exe');
      if (fs.existsSync(exe)) return exe;
    }
  }
  throw new Error('no chromium headless shell; set CHROMIUM_PATH');
}

const browser = await chromium.launch({ executablePath: chromiumPath(), headless: true, args: ['--force-color-profile=srgb', '--font-render-hinting=none'] });
const out = { sheets: [] };
fs.mkdirSync(path.join(dir, 'ref'), { recursive: true });
try {
  for (const sheet of manifest.sheets) {
    const cells = sheet.cells.filter((c) => c.html);
    const bodies = cells.map((c) => {
      const html = expand(c.html);
      return `<div class="g-cell" data-id="${c.id}" style="left:${c.x}px;top:${c.y}px;width:${c.w}px;height:${c.h}px">${html}</div>`;
    });
    const all = bodies.join('\n');
    const style = await css([...classesOf(all), 'bg-background', 'text-foreground']);
    const page = await browser.newPage({ viewport: { width: sheet.width, height: sheet.height }, deviceScaleFactor: 1 });
    const cdp = await page.context().newCDPSession(page);
    await cdp.send('Emulation.setEmulatedMedia', { features: [{ name: 'hover', value: 'hover' }, { name: 'pointer', value: 'fine' }, { name: 'prefers-reduced-motion', value: 'reduce' }] });
    await page.setContent(`<!doctype html><html class="${sheet.scheme === 'dark' ? 'dark' : ''}" style="${sheet.rootStyle || ''}"><head><style>${style}
      *,*::before,*::after{transition:none!important;animation-play-state:paused!important}
      body{margin:0;font-family:${sheet.font};-webkit-font-smoothing:antialiased;font-size:14px;line-height:20px}
      .g-cell{position:absolute;box-sizing:border-box;padding:${sheet.pad}px;display:flex;align-items:flex-start;justify-content:flex-start;overflow:visible}
      </style></head><body class="bg-background text-foreground" style="width:${sheet.width}px;height:${sheet.height}px">${all}</body></html>`);
    await page.evaluate(() => document.fonts.ready);
    if (process.env.REF_DEBUG) console.log(await page.evaluate(() => [document.documentElement.outerHTML.slice(0, 200).replace(/<style>[^]*/, ''), getComputedStyle(document.body).backgroundColor]));
    // Forced pseudo-classes.
    const { root } = await cdp.send('DOM.getDocument', { depth: -1 });
    const { nodeIds } = await cdp.send('DOM.querySelectorAll', { nodeId: root.nodeId, selector: '[data-force]' });
    await cdp.send('CSS.enable');
    for (const id of nodeIds) {
      const { attributes } = await cdp.send('DOM.getAttributes', { nodeId: id });
      const i = attributes.indexOf('data-force');
      const states = attributes[i + 1].split(',').filter(Boolean);
      await cdp.send('CSS.forcePseudoState', { nodeId: id, forcedPseudoClasses: states });
    }
    await page.waitForTimeout(50);
    const geo = await page.evaluate(() => {
      const res = {};
      for (const cell of document.querySelectorAll('.g-cell')) {
        const el = cell.querySelector('[data-g]') || cell.firstElementChild;
        if (!el) continue;
        const r = el.getBoundingClientRect();
        const cs = getComputedStyle(el);
        let tr = null;
        const walker = document.createTreeWalker(el, NodeFilter.SHOW_TEXT);
        for (let n = walker.nextNode(); n; n = walker.nextNode()) {
          if (!n.textContent.trim()) continue;
          const range = document.createRange();
          range.selectNodeContents(n);
          const b = range.getBoundingClientRect();
          if (b.width > 0) {
            tr = { x: b.x, y: b.y, w: b.width, h: b.height };
            break;
          }
        }
        res[cell.dataset.id] = {
          x: r.x, y: r.y, w: r.width, h: r.height,
          padL: parseFloat(cs.paddingLeft), padR: parseFloat(cs.paddingRight), padT: parseFloat(cs.paddingTop), padB: parseFloat(cs.paddingBottom),
          radius: parseFloat(cs.borderTopLeftRadius), border: parseFloat(cs.borderTopWidth),
          bg: cs.backgroundColor,
          color: cs.color, borderColor: cs.borderTopColor, shadow: cs.boxShadow,
          text: tr,
        };
      }
      return res;
    });
    await page.screenshot({ path: path.join(dir, 'ref', sheet.name + '.png'), animations: 'disabled' });
    out.sheets.push({ name: sheet.name, cells: geo });
    await page.close();
    process.stdout.write(`ref ${sheet.name}: ${cells.length} specimens\n`);
  }
} finally {
  await browser.close();
}
fs.writeFileSync(path.join(dir, 'ref-geometry.json'), JSON.stringify(out, null, 1));
