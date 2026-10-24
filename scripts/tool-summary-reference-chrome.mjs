#!/usr/bin/env node
/** Round117: current installed file-summary links and diff totals, one isolated Chrome. */
import fs from 'node:fs/promises';
import path from 'node:path';
import {fileURLToPath, pathToFileURL} from 'node:url';
import {createHash, randomUUID} from 'node:crypto';
import net from 'node:net';
import vm from 'node:vm';
import {spawn} from 'node:child_process';

const ROOT = 'C:\\AI\\WhaleIsle-MyGo';
const SCRIPT = path.join(ROOT, 'scripts/tool-summary-reference-chrome.mjs');
const INSTALL = 'C:\\软件\\Whale Isle\\resources\\vendor\\deepseek-harness';
const TOOL = path.join(INSTALL, 'packages/client/ui-tool/lib');
const REF = path.join(ROOT, 'reference/electron/vendor/deepseek-harness/packages/client/ui-primitives/src');
const SOURCES = {
  runner: SCRIPT, row: path.join(TOOL, 'types/client/tool/components/ToolRow.js'),
  model: path.join(TOOL, 'types/client/tool/models/tool-call-model.js'), tool: path.join(TOOL, 'client.js'),
  terminal: path.join(TOOL, 'types/client/tool/models/terminal-card-model.js'),
  read: path.join(TOOL, 'types/client/tool/models/read-card-model.js'),
  raw: path.join(TOOL, 'types/client/tool/models/raw-tool-call.js'),
  recorded: path.join(TOOL, 'types/client/tool/models/recorded-path.js'),
  readRow: path.join(TOOL, 'types/client/tool/toolviews/read-row.js'),
  mutationRow: path.join(TOOL, 'types/client/tool/toolviews/file-mutation-row.js'),
  webRow: path.join(TOOL, 'types/client/tool/toolviews/web-row.js'),
  paths: path.join(INSTALL, 'packages/util/workspace-path/lib/types/index.js'),
  theme: path.join(INSTALL, 'packages/client/ui-theme/lib/client.js'),
  webJS: path.join(INSTALL, 'apps/web/dist/assets/index-wKUmVWbt.js'),
  webCSS: path.join(INSTALL, 'apps/web/dist/assets/index-l5jToo6l.css'),
  refDiff: path.join(REF, 'DiffBlock.tsx'), refShimmer: path.join(REF, 'TextShimmer.tsx'),
  icons: path.join(REF, 'icons/index.tsx'), prior: path.join(ROOT, 'scripts/tool-code-reference-chrome.mjs'),
};
const CHROME = 'C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe';
const MODE = process.argv[2];
const supplied = MODE === '--accept-prepared' ? process.argv[3] : null;
const suppliedRun = supplied && path.basename(supplied).match(/^r117-tool-summary-([0-9a-fTZ-]+)-preflight\.json$/)?.[1];
const RUN = suppliedRun ?? new Date().toISOString().replace(/[:.]/g, '-') + '-' + randomUUID().slice(0, 8);
const BASE = path.join(ROOT, '.devdata/r117-tool-summary-' + RUN);
const FIXTURE = BASE + '.html', PROFILE = BASE + '-profile', PREP = BASE + '-preflight.json';
const ATTEMPT = path.join(ROOT, '.devdata/r117-tool-summary-acceptance-attempt.json');
const PREFIX = path.join(ROOT, 'docs/review-20261022/evidence/tool-summary-pass261-reference-' + RUN);
const OUT = {source: PREFIX + '-source.json', json: PREFIX + '.json', report: PREFIX + '-report.md', text: PREFIX + '.txt', rest: PREFIX + '-rest.png', hover: PREFIX + '-hover.png', open: PREFIX + '-open.png', focus: PREFIX + '-focus.png'};
const PROTECTED = new Set([20012, 16008, 12516]);
const norm = value => path.resolve(value).toLowerCase();
const hash = value => createHash('sha256').update(value).digest('hex');
const invariant = (value, message) => { if (!value) throw Error(message); };
const near = (a, b) => Math.abs(a - b) < 0.04;
const delay = ms => { invariant(ms <= 1000, 'Long blocking wait forbidden'); return new Promise(resolve => setTimeout(resolve, ms)); };
const allowed = new Set([FIXTURE, PREP, ATTEMPT, ...Object.values(OUT)].map(norm));
const E = {
  schema: 'tool-summary-pass261-reference/v1', round: 117, runId: RUN, startedAt: new Date().toISOString(), status: 'running',
  attemptPolicy: 'One exclusive round-wide acceptance attempt; one Chrome startup; each case once. Preserve failures without an accepting rerun.',
  scope: {
    script: SCRIPT, fixture: FIXTURE, profile: PROFILE,
    description: 'Manual DOM reproducing current installed ToolRow/DisclosureRow/TextShimmer base containment with verbatim current installed CSS and light theme. Synthetic summaries/diffs and explicit local handlers. Source-only exact installed pure functions execute in a VM; no React or application execution.',
    input: 'Only CDP events in own exact PID/profile/port-verified headless Chrome. No hardware input, monitor, original Electron or application parity claim. Row keyboard handler is explicit local code. Path Space activation uses a documented explicit LOCAL keydown handler, not React/native-button default certification. No native-button Enter activation assertion.',
    network: 'No server; file CSP denies network and CDP blocks HTTP/HTTPS/WS/WSS. Only own loopback CDP endpoint contacted.',
    protected: {electronPID: 20012, backendPID: 16008, lockAppPID: 12516, guiPort: 3080, touched: false},
  },
  compactPolicy: 'Unique CSS/maps/hashes and pure source/table only once in sidecar; computed styles interned once; final boxes once per scenario; transitions scalar only; combined JSON <250KB.',
  sourceEvidence: OUT.source, cases: [], measurements: {}, screenshots: {}, errors: [], writes: [],
  limitations: [
    'Installed artifacts are authority. Older read-only TypeScript only locates and explains verified symbols; installed ui-primitives package is absent, current webdist supplies the primitive implementation.',
    'Manual DOM/local callbacks do not execute React, useDisclosure, application openFile dispatch, host file navigation, actual ReadBlock/DiffBlock bodies or session/model logic.',
    'Path click logs callback arguments only; no file is opened. URL anchor attributes/styles/keydown isolation are measured without navigating or opening a target.',
    'Running/preparing shimmer source and CSS are recorded only. Decoration timing, motion, reduced-motion behavior and React decorative duplication are not certified.',
    'System font resolution and baseline light tokens only. DPR1 is a startup flag, no Emulation or hardware-input assertion.',
  ],
};
let source, browser, page, child, exitInfo, port, verified = false, spawnCount = 0;
const requests = [], exceptions = [], protocolErrors = [], methods = {};

async function noLinks(target) {
  const absolute = path.resolve(target); let current = path.parse(absolute).root;
  for (const part of absolute.slice(current.length).split(path.sep)) {
    if (!part) continue; current = path.join(current, part);
    try { invariant(!(await fs.lstat(current)).isSymbolicLink(), 'Refuse symlink ' + current); }
    catch (error) { if (error.code !== 'ENOENT') throw error; }
  }
}
async function absent(target) {
  try { await fs.lstat(target); throw Error('Refuse existing ' + target); }
  catch (error) { if (error.code !== 'ENOENT') throw error; }
}
async function writeOwned(target, data) {
  invariant(allowed.has(norm(target)), 'Out of scope write ' + target); await noLinks(target); await absent(target);
  await fs.writeFile(target, data, {flag: 'wx'}); E.writes.push(target);
}
async function scope() {
  invariant(process.platform === 'win32' && Number(process.versions.node.split('.')[0]) >= 24, 'Windows Node24 required');
  invariant((MODE === '--prepare-only' && process.argv.length === 3) || (MODE === '--accept-prepared' && process.argv.length === 4 && suppliedRun && norm(supplied) === norm(PREP)), 'Only prepare/own-prepared modes');
  invariant(norm(fileURLToPath(import.meta.url)) === norm(SCRIPT) && norm(process.cwd()) === norm(ROOT), 'Wrong runner/workspace');
  for (const target of [...Object.values(SOURCES), CHROME, PROFILE, ...allowed]) await noLinks(target);
  for (const target of [ROOT, path.dirname(BASE), path.dirname(PREFIX)]) invariant(norm(await fs.realpath(target)) === norm(target), 'Redirected root ' + target);
  for (const target of [PROFILE, ATTEMPT, ...Object.values(OUT).filter(p => MODE === '--prepare-only' || p !== OUT.source), ...(MODE === '--prepare-only' ? [PREP, FIXTURE] : [])]) await absent(target);
}
function fragment(text, from, until, sourceID) {
  const start = text.indexOf(from), end = text.indexOf(until, start);
  invariant(start >= 0 && end > start, 'Missing installed fragment ' + from);
  return {source: sourceID, startUtf16Offset: start, endUtf16Offset: end, text: text.slice(start, end)};
}
function pureFunction(text, name) {
  const found = text.match(new RegExp('(?:export )?function ' + name + '\\([^]*?\\n}'));
  invariant(found, 'Missing function ' + name); return found[0].replace(/^export /, '');
}
function themeLiteral(text, name) {
  const prefix = 'var ' + name + '_css_default = ', line = text.split('\n').find(value => value.trimStart().startsWith(prefix));
  invariant(line, 'Missing installed theme ' + name);
  return {source: 'theme', line: text.slice(0, text.indexOf(line)).split('\n').length, text: JSON.parse(line.trim().slice(prefix.length).replace(/;$/, ''))};
}
function cssMap(css, suffix, names) {
  return Object.fromEntries(names.map(name => {
    const found = css.match(new RegExp('\\._' + name + '_' + suffix + '_\\d+'));
    invariant(found, 'Missing CSS class ' + name); return [name, found[0].slice(1)];
  }));
}
function diffTable(pure) {
  const entry = (oldText, newText, file = 'fixture.txt') => ({path: file, oldText, newText});
  const longOld = ['shared-prefix', ...Array.from({length: 129}, (_, i) => 'old-' + i), 'shared-suffix'].join('\n');
  const longNew = ['shared-prefix', ...Array.from({length: 129}, (_, i) => 'new-' + i), 'shared-suffix'].join('\n');
  const boundedOld = longOld.replace('\nold-128', ''), boundedNew = longNew.replace('\nnew-128', '');
  const samples = [
    ['empty-array', [], [0, 0]], ['empty-create', [entry(null, '')], [0, 0]],
    ['create-two', [entry(null, 'a\nb\n')], [2, 0]], ['delete-two', [entry('a\nb\n', '')], [0, 2]],
    ['identical', [entry('a\nb\n', 'a\nb\n')], [0, 0]], ['terminator-only', [entry('a\nb', 'a\nb\n')], [0, 0]],
    ['one-replacement-with-context', [entry('a\nb\nc\n', 'a\nB\nc\n')], [1, 1]],
    ['interior-blank', [entry('a\nb', 'a\n\nb')], [1, 0]], ['two-terminal-LF-retains-blank', [entry('a\n', 'a\n\n')], [1, 0]],
    ['single-LF-is-one-blank-line', [entry(null, '\n')], [1, 0]], ['CRLF-versus-LF', [entry('a\r\nb\r\n', 'a\nb\n')], [2, 2]],
    ['same-CRLF', [entry('a\r\nb\r\n', 'a\r\nb\r\n')], [0, 0]], ['literal-sign-content', [entry('+old\n-same\n', '+new\n-same\n')], [1, 1]],
    ['reorder-shares-LCS', [entry('a\nb\nc\n', 'b\na\nc\n')], [1, 1]],
    ['multiple-independent-diffs', [entry('a\nb\n', 'a\nB\nc\n', 'one'), entry('x\ny\n', 'y\n', 'two')], [2, 2]],
    ['exact256-edits-excludes-shared', [entry(boundedOld, boundedNew)], [128, 128]],
    ['over256-falls-back-including-shared', [entry(longOld, longNew)], [131, 131]],
  ];
  return samples.map(([id, diffs, expected]) => {
    const result = pure.QZ(diffs), oldLines = diffs.map(d => pure.L8(d.oldText ?? '')), newLines = diffs.map(d => pure.L8(d.newText));
    return {id, inputs: diffs.map(d => ({path: d.path, oldText: d.oldText?.length > 300 ? {chars: d.oldText.length, sha256: hash(d.oldText), recipe: 'shared-prefix + old-0..' + (id.startsWith('exact') ? 127 : 128) + ' + shared-suffix joined LF'} : d.oldText, newText: d.newText.length > 300 ? {chars: d.newText.length, sha256: hash(d.newText), recipe: 'shared-prefix + new-0..' + (id.startsWith('exact') ? 127 : 128) + ' + shared-suffix joined LF'} : d.newText})), splitCounts: {old: oldLines.map(v => v.length), new: newLines.map(v => v.length)}, result: {added: result.added, removed: result.removed}, expected: {added: expected[0], removed: expected[1]}, pass: result.added === expected[0] && result.removed === expected[1]};
  });
}
async function extract() {
  const texts = Object.fromEntries(await Promise.all(Object.entries(SOURCES).map(async ([id, file]) => [id, await fs.readFile(file, 'utf8')])));
  const {row, model, tool, theme, webCSS, webJS} = texts;
  const rowLiteral = tool.match(/^\t\tconst css\$3 = ("(?:[^"\\]|\\.)*");\r?$/m); invariant(rowLiteral, 'Missing installed row CSS');
  const map = fragment(tool, 'var ToolRow_module_css_default = ', '\n\t\t//#endregion', 'tool');
  const maps = {row: Object.fromEntries([...map.text.matchAll(/"([^"]+)": "([^"]+)"/g)].map(m => [m[1], m[2]]))};
  const disclosure = fragment(webCSS, '._root_1rdzk_1{', '._button_ll2mm_2{', 'webCSS');
  maps.disclosure = cssMap(disclosure.text, '1j0v6', ['root', 'header', 'row', 'leading', 'title']);
  maps.shimmer = cssMap(disclosure.text, '1rdzk', ['root', 'content', 'text']);
  const css = {row: {source: 'tool', line: tool.slice(0, rowLiteral.index).split('\n').length, text: JSON.parse(rowLiteral[1])}, disclosure,
    ...Object.fromEntries(['base', 'motion', 'corner_shape', 'design_platform', 'focus', 'scrollbar', 'gradient_shadow_text'].map(name => ['theme_' + name, themeLiteral(theme, name)]))};
  const tokenMap = {};
  for (const rule of css.theme_design_platform.text.matchAll(/([^{}]+)\{([^{}]*)\}/g)) if (rule[1] === 'body' || rule[1] === ':root') for (const value of rule[2].matchAll(/(--[\w-]+):([^;}]+)/g)) tokenMap[value[1]] = value[2];
  const lightColors = Object.fromEntries(['tertiary', 'secondary', 'success', 'error', 'warn'].map(key => {
    const token = key === 'tertiary' || key === 'secondary' ? '--dsw-alias-label-' + key : '--dsw-alias-state-' + key + (key === 'warn' ? '-label' : '-primary');
    let value = tokenMap[token]; for (let depth = 0; /^var\(/.test(value) && depth < 10; depth++) value = tokenMap[value.slice(4, -1)];
    invariant(/^#[0-9a-f]{6}$/i.test(value), 'Cannot resolve light color ' + token);
    return [key, {token, hex: value, rgb: 'rgb(' + [1, 3, 5].map(index => parseInt(value.slice(index, index + 2), 16)).join(', ') + ')'}];
  }));
  const engine = fragment(webJS, 'class QL{', 'const oT=', 'webJS');
  const hunks = fragment(webJS, 'function Bw(', 'function eV(', 'webJS');
  const split = fragment(webJS, 'function L8(', 'function tV(', 'webJS');
  const disclosureJS = fragment(webJS, 'const hU=$.memo', ',mU=400;', 'webJS');
  const shimmerJS = fragment(webJS, 'const Pc=$.createContext', ',pU=200;', 'webJS');
  const diffCode = engine.text + '\nconst YZ=256;\n' + hunks.text + '\n' + split.text + '\n({QZ,L8,Bw})';
  const pureDiff = vm.runInNewContext(diffCode, Object.create(null), {timeout: 1500});
  const table = diffTable(pureDiff); invariant(table.every(v => v.pass), 'Installed diffTotals source-only table differs: ' + JSON.stringify(table.filter(v => !v.pass)));
  const readFns = ['positiveInteger', 'validReadCall', 'readCallLine'].map(name => pureFunction(texts.read, name)).join('\n');
  const rawFn = pureFunction(texts.raw, 'parsedToolCall');
  const readPure = vm.runInNewContext('const parsedCalls=new WeakMap();\n' + rawFn + '\n' + readFns + '\n({readCallLine})', Object.create(null), {timeout: 1000});
  const readInputs = [undefined, 1, 37, 0, -1, 1.5, '37', null];
  const readTable = readInputs.map(offset => {
    const args = {file_path: 'C:\\fixture\\read.ts', ...(offset === undefined ? {} : {offset})};
    const result = readPure.readCallLine({phase: 'start', name: 'read', argsRaw: JSON.stringify(args)});
    const expected = typeof offset === 'number' && Number.isInteger(offset) && offset >= 1 ? offset : undefined;
    return {offset: offset ?? (offset === null ? null : 'omitted'), result: result ?? null, pass: result === expected};
  });
  invariant(readTable.every(v => v.pass), 'readCallLine source table differs');
  const pathFns = ['isWindowsStylePath', 'isAbsoluteWorkspacePath', 'abbreviateHomePath', 'relativizeToCwd'].map(name => pureFunction(texts.paths, name)).join('\n');
  const modelCode = model.replace(/^import .*;\r?\n/gm, '').replaceAll('export ', '');
  const modelPure = vm.runInNewContext(pathFns + '\n' + pureFunction(texts.recorded, 'recordedAbsolutePath') + '\n' + modelCode + '\n({toolRowModel})', Object.create(null), {timeout: 1000});
  const mkArgs = value => ({complete: key => Object.hasOwn(value, key), text: key => typeof value[key] === 'string' ? value[key] : undefined});
  const modelSamples = [
    {id: 'running-absolute', name: 'read', args: {file_path: 'C:\\fixture\\src\\read.ts'}, phase: 'start'},
    {id: 'running-relative', name: 'read', args: {file_path: 'src/read.ts'}, phase: 'start'},
    {id: 'settled-relative-success', name: 'edit', args: {file_path: 'src/edit.ts'}, done: true},
    {id: 'settled-relative-error', name: 'edit', args: {file_path: 'src/edit.ts'}, done: true, isError: true},
    {id: 'recorded-absolute-wins', name: 'write', args: {file_path: 'src/edit.ts'}, done: true, meta: {path: 'D:\\actual\\edit.ts'}},
    {id: 'web-url-not-file', name: 'web_fetch', args: {url: 'https://example.invalid/doc'}, done: true},
    {id: 'error-first-output-line', name: 'read', args: {file_path: 'C:\\fixture\\src\\read.ts'}, done: true, isError: true, content: [{type: 'text', text: 'Access denied\nsecond'}]},
    {id: 'interrupted-stopped', name: 'read', args: {file_path: 'C:\\fixture\\src\\read.ts'}, done: true, isError: true, error: {code: 'interrupted'}},
  ];
  const modelTable = modelSamples.map(s => {
    const raw = JSON.stringify(s.args), common = {callId: 'fixture-call', args: mkArgs(s.args)};
    const block = s.done ? {...common, kind: 'tool_result', call: {name: s.name, argsRaw: raw}, content: s.content ?? [], isError: s.isError ?? false, error: s.error, meta: s.meta} : {...common, phase: s.phase, name: s.name, argsRaw: raw};
    const r = modelPure.toolRowModel(s.name, block, 'C:\\fixture');
    return {id: s.id, result: {variant: r.variant, summary: r.summary, filePath: r.filePath ?? null, errorSummary: r.errorSummary, state: r.state}};
  });
  invariant(modelTable[0].result.summary === 'src\\read.ts' && modelTable[0].result.filePath === 'C:\\fixture\\src\\read.ts' && modelTable[1].result.filePath === null && modelTable[2].result.filePath === 'src/edit.ts' && modelTable[3].result.filePath === null && modelTable[4].result.filePath === 'D:\\actual\\edit.ts' && modelTable[5].result.filePath === null && modelTable[6].result.errorSummary === 'Access denied' && modelTable[7].result.state === 'stopped', 'Installed model table differs');
  const terminalFns = ['terminalFailed', 'terminalExitLabel'].map(name => pureFunction(texts.terminal, name)).join('\n');
  const terminalPure = vm.runInNewContext(terminalFns + '\n({terminalExitLabel})', Object.create(null), {timeout: 1000});
  const terminalTable = [{exitCode: 0}, {exitCode: 7}, {exitCode: null}, {signal: 'SIGTERM'}, {exitCode: 7, running: true}, {exitCode: 7, background: true}].map(card => ({card, label: terminalPure.terminalExitLabel({card}, (key, arg) => key + ':' + (arg?.code ?? arg?.signal ?? ''))}));
  const artwork = {};
  for (const [key, name] of [['up', 'IconChevronUpOutlineArtwork'], ['down', 'IconChevronDownOutlineArtwork']]) {
    const start = texts.icons.indexOf(name + ' = '), end = texts.icons.indexOf('\n)', start) + 2; invariant(start >= 0 && end > start, 'Missing reference artwork');
    const shapes = [...texts.icons.slice(start, end).matchAll(/<(path|rect)\s+([^>]+)\/>/g)].map(v => ({tag: v[1], attributes: Object.fromEntries([...v[2].matchAll(/([\w-]+)="([^"]+)"/g)].map(a => [a[1], a[2]]))}));
    invariant(shapes.length > 0 && shapes.every(s => s.tag !== 'path' || webJS.includes(s.attributes.d)), 'Reference artwork not in installed bundle');
    artwork[key] = shapes;
  }
  invariant(row.includes('event.stopPropagation();') && row.includes('onOpenFile(filePath, { line: filePathLine })') && row.includes("state === 'error' || state === 'stopped'") && row.includes('exitLabel === null && openFile !== undefined'), 'Installed ToolRow behavior changed');
  invariant(css.row.text.includes('flex:0 auto') && css.row.text.includes('text-decoration:underline 1px dotted') && hunks.text.includes('maxEditLength:YZ'), 'Installed CSS/algorithm changed');
  source = {
    schema: 'tool-summary-pass261-reference-source/v1', runId: RUN, authority: INSTALL,
    files: await Promise.all(Object.entries(SOURCES).map(async ([id, file]) => ({id, path: file, bytes: (await fs.stat(file)).size, sha256: hash(await fs.readFile(file))}))),
    css, maps, artwork, lightColors,
    installedJS: {diffEngine: engine, diffHunksAndTotals: hunks, splitLines: split, disclosure: disclosureJS, shimmer: shimmerJS},
    sourceOnly: {
      diffTotals: {symbols: {totals: 'QZ', hunks: 'Bw', lines: 'L8', structuredPatch: 'rT', maxEditLength: 256, context: 3}, table},
      readCallLine: {source: 'read', lines: [15, 18, 73], dependencies: rawFn, functions: readFns, table: readTable},
      toolRowModel: {source: 'model', lines: [200, 210, 248], table: modelTable},
      terminalExit: {source: 'terminal', line: 61, functions: terminalFns, table: terminalTable},
      render: {source: 'row', lines: [62, 65, 69, 74, 83, 89, 96], summary: 'terminal description, else open details.expandedSummary, else summary; errorSummary only error with no terminal exit', suppression: 'error/stopped => no path/href links, suffix, imageSuffix or totals; exitLabel additionally suppresses both link kinds; empty summary suppresses entire separator/link/suffix/totals fragment', callback: 'stopPropagation; onOpenFile(filePath) or onOpenFile(filePath,{line:filePathLine}); path wins href; Enter/Space keydown only stopPropagation', suffix: 'summarySuffix unless imageBody then image label/count; null omitted, empty string still element; totals follow suffix even0/0'},
      cssFacts: {link: 'font13/24;color inherit;dotted underline1px offset3;flex0 1 auto;min-width0;ellipsis;nowrap;overflow hidden;margin0;padding0;border0', summary: 'flex1 1 auto;ellipsis;no underline', suffix: 'flex:none;font13/24;margin-left4px;nowrap', totals: 'summarySuffix+diffStat;font code11/24;margin-left10;translateY(.5px);neutral inherited rest;added success and removed error on row:hover OR aria-expanded=true'},
    },
    referenceRisk: 'The older reference TS is read-only explanatory material. Current installed JS/CSS, hashes and extracted functions control every measured assertion; no package-local installed primitive source exists.',
  };
  return {maps, css, artwork, totals: diffs => ({...pureDiff.QZ(diffs)})};
}
const esc = value => String(value).replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('>', '&gt;').replaceAll('"', '&quot;');
function models(x) {
  const diffs = [{path: 'fixture.txt', oldText: 'same\nold\n', newText: 'same\nnew\nextra\n'}];
  return [
    {id: 'short-path', title: 'Edited', summary: 'src/a.ts', filePath: 'C:\\fixture\\src\\a.ts', diffs, suffix: '1 KB', w: 560},
    {id: 'short-plain', title: 'Edited', summary: 'src/a.ts', diffs, suffix: '1 KB', w: 560},
    {id: 'tiny-path', title: 'Edited', summary: 'src/very-long-folder/a-very-long-file-name-that-will-ellipsis.ts', filePath: 'C:\\fixture\\src\\long.ts', diffs, w: 190},
    {id: 'long-plain', title: 'Edited', summary: 'Long plain description that fills the entire available row width and must ellipsize beside totals', diffs, w: 360},
    {id: 'description-path', title: 'Read', summary: 'Read project configuration', filePath: 'C:\\fixture\\config.json', w: 560},
    {id: 'read-line', title: 'Read', summary: 'src/read.ts', filePath: 'C:\\fixture\\src\\read.ts', line: 37, w: 560},
    {id: 'href', title: 'Fetched', summary: 'https://example.invalid/reference', href: 'https://example.invalid/reference', w: 560},
    {id: 'path-wins-href', title: 'Read', summary: 'src/precedence.ts', filePath: 'C:\\fixture\\src\\precedence.ts', href: 'https://example.invalid/ignored', w: 560},
    {id: 'error', title: 'Edited', summary: 'src/failed.ts', errorSummary: 'Access denied', state: 'error', filePath: 'C:\\fixture\\src\\failed.ts', href: 'https://example.invalid/denied', suffix: '1 KB', diffs, w: 560},
    {id: 'stopped', title: 'Edited', summary: 'src/stopped.ts', state: 'stopped', filePath: 'C:\\fixture\\src\\stopped.ts', href: 'https://example.invalid/stopped', suffix: '1 KB', diffs, w: 560},
    {id: 'terminal-exit', title: 'Ran', summary: 'source summary ignored', terminalDescription: 'Command description retained', errorSummary: 'failure ignored for exit', state: 'error', exitLabel: 'Exit code: 7', filePath: 'C:\\fixture\\would-not-open.ts', href: 'https://example.invalid/exit', suffix: '1 KB', diffs, w: 560},
    {id: 'zero-totals', title: 'Edited', summary: 'src/same.ts', filePath: 'C:\\fixture\\src\\same.ts', diffs: [{path: 'same.ts', oldText: 'a\n', newText: 'a'}], w: 560},
    {id: 'empty-summary', title: 'Edited', summary: '', filePath: 'C:\\fixture\\empty.ts', suffix: '1 KB', diffs, w: 560},
  ].map((v, index) => ({...v, state: v.state ?? 'ok', x: index % 2 ? 680 : 40, y: 110 + Math.floor(index / 2) * 130, totals: v.diffs ? x.totals(v.diffs) : null}));
}
function fixture(x) {
  const m = x.maps, values = models(x);
  const svg = key => '<svg width="14" height="14" viewBox="0 0 16 16" fill="none" aria-hidden="true" stroke-width="1">' + x.artwork[key].map(s => '<' + s.tag + ' ' + Object.entries(s.attributes).map(([k, v]) => k + '="' + esc(v) + '"').join(' ') + '/>').join('') + '</svg>';
  const nested = (text, extra = '', attrs = '') => '<span class="' + m.shimmer.text + (extra ? ' ' + extra : '') + '"' + attrs + '>' + esc(text) + '</span>';
  const row = value => {
    const settled = ['error', 'stopped'].includes(value.state), text = value.terminalDescription ?? (value.state === 'error' && !value.exitLabel ? value.errorSummary ?? value.summary : value.summary);
    let content = '';
    if (text !== '') {
      const textChild = nested(text), link = !settled && !value.exitLabel && value.filePath ? '<button type="button" class="' + m.row.fileLink + '" data-summary data-link="path">' + textChild + '</button>' : !settled && !value.exitLabel && value.href !== undefined ? '<a class="' + m.row.fileLink + '" data-summary data-link="href" href="' + esc(value.href) + '" target="_blank" rel="noopener noreferrer">' + textChild + '</a>' : '<span data-summary class="' + m.row.summary + (!value.exitLabel && value.state === 'error' ? ' ' + m.row.errorSummary : !value.exitLabel && value.state === 'stopped' ? ' ' + m.row.stoppedSummary : '') + '">' + textChild + '</span>';
      content = '<span class="' + m.row.sep + '" data-sep data-shimmer-decoration aria-hidden="true"></span>' + link;
      if (value.exitLabel) content += '<span data-exit-label>' + esc(value.exitLabel) + '</span>';
      if (!settled && value.suffix != null) content += nested(value.suffix, m.row.summarySuffix, ' data-suffix');
      if (!settled && value.totals !== null) content += '<span class="' + m.shimmer.text + ' ' + m.row.summarySuffix + ' ' + m.row.diffStat + '" data-totals><span class="' + m.row.diffAdded + '" data-added>+' + value.totals.added + '</span> <span class="' + m.row.diffRemoved + '" data-removed>-' + value.totals.removed + '</span></span>';
    }
    return '<section class="scenario" style="left:' + value.x + 'px;top:' + value.y + 'px;width:' + value.w + 'px"><p class="caption">' + esc(value.id) + '</p><div class="' + m.row.root + '" data-own-card="' + value.id + '" data-state="' + value.state + '" data-tool="' + (value.title === 'Read' ? 'read' : 'edit') + '"><div class="' + m.disclosure.root + '" data-disclosure-root><div class="' + m.disclosure.header + '"><div class="' + m.disclosure.row + ' ' + m.row.row + '" data-disclosure-row data-expandable role="button" tabindex="0" aria-expanded="false"><span class="' + m.disclosure.leading + ' ' + m.row.leading + '" data-leading>' + svg('down') + '</span><span class="' + m.shimmer.root + '" data-shimmer-root><span class="' + m.shimmer.content + '" data-shimmer-content>' + nested(value.title, m.disclosure.title + ' ' + m.row.title, ' data-title') + content + '</span></span></div></div><div data-local-body hidden>Local disclosure state only</div></div></div></section>';
  };
  const css = ['base', 'motion', 'corner_shape', 'design_platform', 'focus', 'scrollbar', 'gradient_shadow_text'].map(name => x.css['theme_' + name].text).concat([x.css.disclosure.text, x.css.row.text]);
  const data = {values, maps: m};
  const html = '<!doctype html><html><head><meta charset="utf-8"><meta http-equiv="Content-Security-Policy" content="default-src \'none\';style-src \'unsafe-inline\';script-src \'unsafe-inline\';connect-src \'none\';img-src data:"><title>Round117 installed summary reference</title>' + css.map((s, i) => '<style data-installed-css="' + i + '">' + s + '</style>').join('') + '<style>html,body{margin:0;min-height:100%}body{font-family:var(--dsw-font-family);font-size:14px;color:var(--dsw-alias-label-primary);background:var(--dsw-alias-bg-base)}#intro{position:absolute;left:40px;top:20px;font-size:12px;color:var(--dsw-alias-label-secondary)}h1{font-size:18px;font-weight:500;margin:0 0 5px}#intro p{margin:0}.scenario{position:absolute}.caption{font-size:12px;line-height:18px;color:var(--dsw-alias-label-secondary);margin:0 0 8px}[data-local-body]{font-size:11px;color:var(--dsw-alias-label-caption);padding:5px 0}[data-local-body][hidden]{display:none}</style></head><body data-platform="windows"><header id="intro"><h1>Installed summary links + diff totals · isolated Chrome DPR1</h1><p>Exact current CSS, manual base DOM, local callback log. No React / real file or URL navigation / motion / original app / hardware-input claim.</p></header>' + values.map(row).join('') + '<script>\n' + inlineFixture.toString() + '\ninlineFixture(' + JSON.stringify(data).replaceAll('<', '\\u003c') + ');\n</script></body></html>';
  return {html, values};
}
function inlineFixture(data) {
  'use strict';
  const events = [], callbacks = [], rowToggles = [], linkKeys = [], styles = [], styleIDs = new Map();
  const own = id => document.querySelector('[data-own-card="' + id + '"]');
  const rect = element => { if (!element) return null; const b = element.getBoundingClientRect(); return {x: b.x, y: b.y, w: b.width, h: b.height}; };
  const keys = ['display', 'fontSize', 'lineHeight', 'fontFamily', 'fontWeight', 'color', 'backgroundColor', 'flexGrow', 'flexShrink', 'flexBasis', 'minWidth', 'width', 'maxWidth', 'overflowX', 'overflowY', 'textOverflow', 'whiteSpace', 'textDecorationLine', 'textDecorationStyle', 'textDecorationThickness', 'textUnderlineOffset', 'textAlign', 'cursor', 'marginTop', 'marginRight', 'marginBottom', 'marginLeft', 'paddingTop', 'paddingRight', 'paddingBottom', 'paddingLeft', 'borderTopWidth', 'transform', 'alignItems', 'outlineStyle', 'outlineWidth', 'outlineOffset', 'outlineColor'];
  const style = element => element ? Object.fromEntries(keys.map(k => [k, getComputedStyle(element)[k]])) : null;
  const intern = element => { if (!element) return null; const value = style(element), serialized = JSON.stringify(value); if (!styleIDs.has(serialized)) { styleIDs.set(serialized, 'S' + (styles.length + 1)); styles.push({id: styleIDs.get(serialized), value}); } return styleIDs.get(serialized); };
  const box = element => element ? {box: rect(element), style: intern(element), text: element.textContent, scrollWidth: element.scrollWidth, clientWidth: element.clientWidth} : null;
  for (const type of ['mousemove', 'pointerdown', 'pointerup', 'mousedown', 'mouseup', 'click', 'keydown', 'keyup']) document.addEventListener(type, event => {
    const link = event.target.closest('[data-link]'); events.push({type, trusted: event.isTrusted, id: event.target.closest('[data-own-card]')?.dataset.ownCard ?? null, key: event.key ?? null, ctrl: event.ctrlKey, target: link ? link.dataset.link : event.target.closest('[data-disclosure-row]') ? 'row' : null});
  }, true);
  for (const value of data.values) {
    const root = own(value.id), row = root.querySelector('[data-disclosure-row]'), body = root.querySelector('[data-local-body]');
    const toggle = event => { const open = row.getAttribute('aria-expanded') !== 'true'; row.setAttribute('aria-expanded', String(open)); body.hidden = !open; const disclosure = root.querySelector('[data-disclosure-root]'); if (open) disclosure.dataset.open = 'true'; else delete disclosure.dataset.open; rowToggles.push({id: value.id, open, type: event.type, key: event.key ?? null, trusted: event.isTrusted}); };
    row.addEventListener('click', toggle);
    row.addEventListener('keydown', event => { if (event.key !== 'Enter' && event.key !== ' ') return; event.preventDefault(); toggle(event); });
    const link = root.querySelector('[data-link]');
    if (link) {
      link.addEventListener('keydown', event => {
        if (event.key === 'Enter' || event.key === ' ') { event.stopPropagation(); linkKeys.push({id: value.id, key: event.key, trusted: event.isTrusted, defaultPreventedAfterInstalledHandler: event.defaultPrevented}); }
      });
      if (link.dataset.link === 'path') {
        link.addEventListener('click', event => { event.stopPropagation(); callbacks.push({id: value.id, args: value.line === undefined ? [value.filePath] : [value.filePath, {line: value.line}], trusted: event.isTrusted, ctrl: event.ctrlKey, origin: 'click'}); });
        // Local test adapter only. This explicitly activates Space without relying on CDP char/native defaults.
        link.addEventListener('keydown', event => { if (event.key !== ' ') return; event.preventDefault(); callbacks.push({id: value.id, args: value.line === undefined ? [value.filePath] : [value.filePath, {line: value.line}], trusted: event.isTrusted, ctrl: event.ctrlKey, origin: 'explicit-local-Space-handler'}); });
      } else link.addEventListener('click', event => { event.stopPropagation(); });
    }
  }
  function quick(id) {
    const root = own(id), row = root.querySelector('[data-disclosure-row]'), summary = root.querySelector('[data-summary]'), totals = root.querySelector('[data-totals]');
    return {expanded: row.getAttribute('aria-expanded'), hover: row.matches(':hover'), focused: document.activeElement === summary ? 'summary' : document.activeElement === row ? 'row' : document.activeElement?.tagName, focusVisible: summary?.matches(':focus-visible') ?? false, summaryColor: summary ? getComputedStyle(summary).color : null, rowColor: getComputedStyle(row).color, addedColor: root.querySelector('[data-added]') ? getComputedStyle(root.querySelector('[data-added]')).color : null, removedColor: root.querySelector('[data-removed]') ? getComputedStyle(root.querySelector('[data-removed]')).color : null, callbackCount: callbacks.filter(v => v.id === id).length, rowToggleCount: rowToggles.filter(v => v.id === id).length, totalsText: totals?.textContent ?? null};
  }
  function one(id) {
    const root = own(id), row = root.querySelector('[data-disclosure-row]'), link = root.querySelector('[data-link]');
    const parts = Object.fromEntries(['title', 'sep', 'summary', 'suffix', 'totals', 'added', 'removed', 'shimmer-root', 'shimmer-content'].map(name => [name, box(root.querySelector('[data-' + name + ']'))]));
    return {id, own: box(root), row: {...box(row), expanded: row.getAttribute('aria-expanded'), role: row.getAttribute('role'), tabIndex: row.tabIndex}, ...parts, link: link ? {kind: link.dataset.link, tag: link.tagName, type: link.getAttribute('type'), href: link.getAttribute('href'), target: link.getAttribute('target'), rel: link.getAttribute('rel'), tabIndex: link.tabIndex} : null, exitLabel: root.querySelector('[data-exit-label]')?.textContent ?? null, quick: quick(id)};
  }
  function metrics(id) { const root = own(id); return Object.fromEntries(['summary', 'suffix', 'totals', 'added', 'removed', 'sep', 'title'].map(name => [name, style(root.querySelector('[data-' + name + ']'))])); }
  const tokens = () => Object.fromEntries(['--dsw-alias-label-tertiary', '--dsw-alias-label-secondary', '--dsw-alias-label-caption', '--dsw-alias-state-success-primary', '--dsw-alias-state-error-primary', '--dsw-alias-state-warn-label', '--ds-font-family-code'].map(name => [name, getComputedStyle(document.body).getPropertyValue(name).trim()]));
  window.harness = {ready: true, quick, one, metrics, tokens, styles: () => styles, events: () => events, callbacks: () => callbacks, toggles: () => rowToggles, linkKeys: () => linkKeys, box: (id, selector) => rect(own(id).querySelector(selector))};
}

class CDP {
  constructor(socket) {
    this.socket = socket; this.id = 0; this.pending = new Map(); this.listeners = [];
    socket.addEventListener('message', event => {
      const message = JSON.parse(String(event.data));
      if (message.id) { const p = this.pending.get(message.id); if (!p) return; clearTimeout(p.timer); this.pending.delete(message.id); if (message.error) { protocolErrors.push({method: p.method, error: message.error}); p.reject(Error(p.method + ' ' + JSON.stringify(message.error))); } else p.resolve(message.result); }
      else for (const listener of this.listeners) listener(message);
    });
    socket.addEventListener('close', () => { for (const p of this.pending.values()) { clearTimeout(p.timer); p.reject(Error('Owned CDP closed ' + p.method)); } this.pending.clear(); });
  }
  static async connect(url) {
    const u = new URL(url); invariant(u.protocol === 'ws:' && u.hostname === '127.0.0.1' && +u.port === port, 'Wrong CDP URL');
    const socket = new WebSocket(url);
    await new Promise((resolve, reject) => { const timer = setTimeout(() => reject(Error('CDP connect timeout')), 10000); socket.addEventListener('open', () => { clearTimeout(timer); resolve(); }, {once: true}); socket.addEventListener('error', () => { clearTimeout(timer); reject(Error('CDP connect error')); }, {once: true}); });
    return new CDP(socket);
  }
  send(method, params = {}, sessionId) {
    invariant(!method.startsWith('Emulation.'), 'Emulation forbidden'); if (method.startsWith('Input.')) invariant(verified, 'Input before ownership verification');
    methods[method] = (methods[method] ?? 0) + 1; const id = ++this.id;
    return new Promise((resolve, reject) => { const timer = setTimeout(() => { this.pending.delete(id); reject(Error(method + ' timeout')); }, 10000); this.pending.set(id, {resolve, reject, timer, method}); this.socket.send(JSON.stringify({id, method, params, ...(sessionId ? {sessionId} : {})})); });
  }
}
async function waitFor(fn, message, ms = 10000) { const end = Date.now() + ms; do { if (await fn()) return; await delay(40); } while (Date.now() < end); throw Error('Timeout ' + message); }
async function freePort() {
  const server = net.createServer(); await new Promise((resolve, reject) => { server.once('error', reject); server.listen({host: '127.0.0.1', port: 0, exclusive: true}, resolve); });
  const value = server.address().port; invariant(value >= 1024 && ![3080, 20012, 16008, 12516].includes(value), 'Protected port');
  await new Promise((resolve, reject) => server.close(error => error ? reject(error) : resolve())); return value;
}
async function portFree() { return new Promise(resolve => { const server = net.createServer(); server.once('error', () => resolve(false)); server.listen({host: '127.0.0.1', port, exclusive: true}, () => server.close(() => resolve(true))); }); }
async function evaluate(expression) { const r = await page.send('Runtime.evaluate', {expression, returnByValue: true, awaitPromise: true}); invariant(!r.exceptionDetails, 'Fixture evaluation ' + JSON.stringify(r.exceptionDetails)); return r.result.value; }
const frames = () => evaluate('new Promise(resolve=>requestAnimationFrame(()=>requestAnimationFrame(resolve)))');
const quick = id => evaluate('window.harness.quick(' + JSON.stringify(id) + ')');
const one = id => evaluate('window.harness.one(' + JSON.stringify(id) + ')');
const metrics = id => evaluate('window.harness.metrics(' + JSON.stringify(id) + ')');
const targetBox = (id, selector) => evaluate('window.harness.box(' + JSON.stringify(id) + ',' + JSON.stringify(selector) + ')');
async function start() {
  invariant(++spawnCount === 1, 'Second Chrome startup forbidden'); port = await freePort(); await noLinks(PROFILE); await absent(PROFILE); await fs.mkdir(PROFILE); E.writes.push(PROFILE + path.sep + ' (temporary owned profile)');
  const args = ['--headless=new', '--user-data-dir=' + PROFILE, '--remote-debugging-address=127.0.0.1', '--remote-debugging-port=' + port, '--force-device-scale-factor=1', '--enable-automation', '--no-first-run', '--no-default-browser-check', '--disable-background-networking', '--disable-background-timer-throttling', '--disable-backgrounding-occluded-windows', '--disable-renderer-backgrounding', '--disable-component-update', '--disable-sync', '--disable-extensions', '--disable-default-apps', '--disable-domain-reliability', '--disable-client-side-phishing-detection', '--safebrowsing-disable-auto-update', '--disable-features=MediaRouter,OptimizationHints,Translate,AutofillServerCommunication', '--metrics-recording-only', '--password-store=basic', '--no-proxy-server', '--host-resolver-rules=MAP * ~NOTFOUND', '--disable-breakpad', '--disable-gpu', '--window-size=1500,1250', 'about:blank'];
  child = spawn(CHROME, args, {cwd: ROOT, stdio: 'ignore', windowsHide: true, shell: false}); invariant(Number.isSafeInteger(child.pid) && !PROTECTED.has(child.pid), 'Wrong spawned PID');
  E.browser = {startupCount: spawnCount, port, launch: {pid: child.pid, executable: CHROME, args, stdio: 'ignore', shell: false}};
  child.once('error', error => { exitInfo = {error: String(error)}; }); child.once('exit', (code, signal) => { exitInfo = {code, signal}; });
  let version;
  await waitFor(async () => { invariant(!exitInfo, 'Own Chrome exited early'); try { const r = await fetch('http://127.0.0.1:' + port + '/json/version', {signal: AbortSignal.timeout(750)}); if (r.ok) { version = await r.json(); return !!version.webSocketDebuggerUrl; } } catch (error) { if (!['TimeoutError', 'TypeError'].includes(error.name)) throw error; } return false; }, 'own Chrome endpoint', 15000);
  browser = await CDP.connect(version.webSocketDebuggerUrl);
  const processes = (await browser.send('SystemInfo.getProcessInfo')).processInfo, command = (await browser.send('Browser.getBrowserCommandLine')).arguments;
  E.browser.initialProcesses = processes.map(({type, id}) => ({type, id})); E.browser.version = await browser.send('Browser.getVersion');
  invariant(processes.some(p => p.type === 'browser' && p.id === child.pid) && processes.every(p => !PROTECTED.has(p.id)), 'CDP PID mismatch');
  for (const arg of ['--user-data-dir=' + PROFILE, '--remote-debugging-port=' + port, '--force-device-scale-factor=1']) invariant(command.includes(arg), 'Owned argument mismatch ' + arg);
  E.browser.ownership = {pass: true, verifiedBeforeInputAt: new Date().toISOString(), method: 'Spawn PID equals CDP browser PID; exact profile/port/DPR arguments in Browser.getBrowserCommandLine; exclusive loopback port'}; verified = true;
  const target = await browser.send('Target.createTarget', {url: 'about:blank'}), attached = await browser.send('Target.attachToTarget', {targetId: target.targetId, flatten: true});
  E.browser.targetId = target.targetId; page = {send: (method, params = {}) => browser.send(method, params, attached.sessionId)};
  browser.listeners.push(event => { if (event.sessionId !== attached.sessionId) return; if (event.method === 'Runtime.exceptionThrown') exceptions.push(event.params.exceptionDetails); if (event.method === 'Network.requestWillBeSent') requests.push({url: event.params.request.url, method: event.params.request.method}); });
  await page.send('Runtime.enable'); await page.send('Page.enable'); await page.send('Network.enable'); await page.send('Network.setBlockedURLs', {urls: ['http://*', 'https://*', 'ws://*', 'wss://*']});
  const navigation = await page.send('Page.navigate', {url: pathToFileURL(FIXTURE).href}); invariant(!navigation.errorText, 'Fixture navigation ' + navigation.errorText);
  await waitFor(async () => await evaluate('Boolean(window.harness?.ready)'), 'fixture ready'); await frames();
}
async function move(x, y) { await page.send('Input.dispatchMouseEvent', {type: 'mouseMoved', x, y, button: 'none', buttons: 0}); await delay(160); await frames(); }
async function click(box, modifiers = 0) { invariant(box?.w > 0 && box.h > 0, 'Missing clickable box'); const x = box.x + box.w / 2, y = box.y + box.h / 2; await move(x, y); await page.send('Input.dispatchMouseEvent', {type: 'mousePressed', x, y, button: 'left', buttons: 1, clickCount: 1, modifiers}); await page.send('Input.dispatchMouseEvent', {type: 'mouseReleased', x, y, button: 'left', buttons: 0, clickCount: 1, modifiers}); await frames(); }
async function key(value, code, vk) { await page.send('Input.dispatchKeyEvent', {type: 'keyDown', key: value, code, windowsVirtualKeyCode: vk, nativeVirtualKeyCode: vk}); await page.send('Input.dispatchKeyEvent', {type: 'keyUp', key: value, code, windowsVirtualKeyCode: vk, nativeVirtualKeyCode: vk}); await frames(); }
async function focus(id, selector) { await evaluate('document.querySelector(\'[data-own-card="' + id + '"]\').querySelector(' + JSON.stringify(selector) + ').focus()'); await frames(); }
async function capture(name) { const r = await page.send('Page.captureScreenshot', {format: 'png', fromSurface: true, captureBeyondViewport: false}), bytes = Buffer.from(r.data, 'base64'); await writeOwned(OUT[name], bytes); E.screenshots[name] = {path: OUT[name], sha256: hash(bytes)}; }
async function runCase(id, fn) {
  const c = {id, status: 'running', assertions: []}; E.cases.push(c);
  const check = (name, pass, detail) => c.assertions.push({name, pass: !!pass, ...(!pass && detail !== undefined ? {detail} : {})});
  try { await fn(check); c.status = c.assertions.every(v => v.pass) ? 'passed' : 'failed'; } catch (error) { c.status = 'failed'; c.error = error.stack ?? String(error); }
  console.log(c.status.toUpperCase() + ' ' + id + ' ' + c.assertions.filter(v => v.pass).length + '/' + c.assertions.length);
}
async function acceptance(values) {
  await move(5, 5); E.tokens = await evaluate('window.harness.tokens()'); E.browser.viewport = await evaluate('({dpr:devicePixelRatio,innerWidth,innerHeight})');
  const snapshots = Object.fromEntries(await Promise.all(values.map(async v => [v.id, await one(v.id)])));
  E.measurements.scenarios = Object.values(snapshots);
  E.fixtureInputs = values.map(({x, y, ...v}) => v);
  await capture('rest');
  await runCase('source-only-installed-tables', async check => {
    check('17 exact installed diffTotals cases incl256/258 edits', source.sourceOnly.diffTotals.table.length === 17 && source.sourceOnly.diffTotals.table.every(v => v.pass));
    check('8 read offsets and8 path/state model cases', source.sourceOnly.readCallLine.table.length === 8 && source.sourceOnly.readCallLine.table.every(v => v.pass) && source.sourceOnly.toolRowModel.table.length === 8);
    check('terminal exit7/null/signal labels, running/background suppressed', source.sourceOnly.terminalExit.table[0].label === null && source.sourceOnly.terminalExit.table.slice(1, 4).every(v => v.label !== null) && source.sourceOnly.terminalExit.table.slice(4).every(v => v.label === null));
  });
  await runCase('neutral-link-and-totals-exact-CSS', async check => {
    const s = snapshots['short-path'], k = await metrics('short-path');
    check('single startup DPR1, row24, native path button', spawnCount === 1 && E.browser.viewport.dpr === 1 && near(s.row.box.h, 24) && s.link.tag === 'BUTTON' && s.link.type === 'button');
    check('link font13/24 inherit neutral tertiary', k.summary.fontSize === '13px' && k.summary.lineHeight === '24px' && k.summary.color === s.quick.rowColor && s.quick.rowColor === source.lightColors.tertiary.rgb, k.summary.color);
    check('link underline dotted1px offset3', k.summary.textDecorationLine === 'underline' && k.summary.textDecorationStyle === 'dotted' && k.summary.textDecorationThickness === '1px' && k.summary.textUnderlineOffset === '3px');
    check('link flex0 1 auto; min0 ellipsis nowrap hidden', k.summary.flexGrow === '0' && k.summary.flexShrink === '1' && k.summary.flexBasis === 'auto' && k.summary.minWidth === '0px' && k.summary.textOverflow === 'ellipsis' && k.summary.whiteSpace === 'nowrap' && k.summary.overflowX === 'hidden');
    check('button border/margin/padding0 and text-align left', k.summary.borderTopWidth === '0px' && k.summary.marginLeft === '0px' && k.summary.paddingLeft === '0px' && k.summary.paddingTop === '0px' && k.summary.textAlign === 'left');
    check('suffix13/24 flexnone margin4; totals11/24 code margin10 translateY.5', k.suffix.fontSize === '13px' && k.suffix.lineHeight === '24px' && k.suffix.flexGrow === '0' && k.suffix.flexShrink === '0' && k.suffix.marginLeft === '4px' && k.totals.fontSize === '11px' && k.totals.lineHeight === '24px' && k.totals.marginLeft === '10px' && k.totals.transform === 'matrix(1, 0, 0, 1, 0, 0.5)' && k.totals.fontFamily.includes('Consolas'));
    check('totals+2-1 neutral inherited rest; suffix precedes totals gaps4/10', s.totals.text === '+2 -1' && s.quick.addedColor === s.quick.rowColor && s.quick.removedColor === s.quick.rowColor && near(s.suffix.box.x - s.summary.box.x - s.summary.box.w, 4) && near(s.totals.box.x - s.suffix.box.x - s.suffix.box.w, 10));
    check('totals outer top+.5 despite inherited24px line', near(s.totals.box.y - s.row.box.y, 0.5) && near(s.totals.box.h, 24));
  });
  await runCase('plain-flex-and-narrow-ellipsis', async check => {
    const a = snapshots['short-path'], b = snapshots['short-plain'], t = snapshots['tiny-path'], l = snapshots['long-plain'], k = await metrics('short-plain');
    E.measurements.layout = {shortPathTotalsRight: a.totals.box.x + a.totals.box.w, shortPlainTotalsRight: b.totals.box.x + b.totals.box.w, shortPathUnusedRight: a.row.box.x + a.row.box.w - a.totals.box.x - a.totals.box.w, shortPlainUnusedRight: b.row.box.x + b.row.box.w - b.totals.box.x - b.totals.box.w, tinyLinkClientWidth: t.summary.clientWidth, tinyLinkScrollWidth: t.summary.scrollWidth, longPlainClientWidth: l.summary.clientWidth, longPlainScrollWidth: l.summary.scrollWidth};
    check('plain summary flex1 1 auto, no underline', k.summary.flexGrow === '1' && k.summary.flexShrink === '1' && k.summary.flexBasis === 'auto' && k.summary.textDecorationLine === 'none');
    check('tiny190px keeps totals and shrinks link with hidden overflow', near(t.own.box.w, 190) && t.summary.scrollWidth > t.summary.clientWidth && t.summary.clientWidth > 0 && t.totals.box.x + t.totals.box.w <= t.row.box.x + t.row.box.w + 0.1);
    check('long plain fills available width and ellipsizes beside totals', l.summary.scrollWidth > l.summary.clientWidth && near(l.totals.box.x + l.totals.box.w, l.row.box.x + l.row.box.w));
    check('exact containment includes only outer shimmer root/content, nested text span totals', b['shimmer-root'] !== null && b['shimmer-content'] !== null && b.link === null && b.totals !== null);
  });
  await runCase('hover-open-success-error-colors', async check => {
    const box = await targetBox('short-path', '[data-summary]'); await move(box.x + box.w / 2, box.y + box.h / 2);
    const hovered = await quick('short-path'); await capture('hover');
    await click(await targetBox('short-path', '[data-leading]')); await move(5, 5); const opened = await quick('short-path'); await capture('open');
    await focus('short-path', '[data-disclosure-row]'); await key('Enter', 'Enter', 13); await move(5, 5); const closed = await quick('short-path');
    E.measurements.colorTransitions = {hovered, opened, closed};
    check('hover row/link secondary and totals success/error', hovered.hover && hovered.rowColor === source.lightColors.secondary.rgb && hovered.summaryColor === hovered.rowColor && hovered.addedColor === source.lightColors.success.rgb && hovered.removedColor === source.lightColors.error.rgb, hovered);
    check('open nonhover row neutral yet totals remain success/error', opened.expanded === 'true' && !opened.hover && opened.rowColor === source.lightColors.tertiary.rgb && opened.addedColor === hovered.addedColor && opened.removedColor === hovered.removedColor, opened);
    check('explicit row Enter closes; totals return inherited neutral', closed.expanded === 'false' && closed.addedColor === closed.rowColor && closed.removedColor === closed.rowColor);
  });
  await runCase('path-click-callback-line-stopprop', async check => {
    await click(await targetBox('description-path', '[data-summary]'));
    await click(await targetBox('read-line', '[data-summary]'));
    await click(await targetBox('path-wins-href', '[data-summary]'), 2);
    const callbacks = await evaluate('window.harness.callbacks()'); E.measurements.clickCallbacks = callbacks;
    check('description text remains linked to independent path', snapshots['description-path'].summary.text === 'Read project configuration' && callbacks.some(v => v.id === 'description-path' && JSON.stringify(v.args) === JSON.stringify(['C:\\fixture\\config.json'])));
    check('read callback second argument exactly line37', callbacks.some(v => v.id === 'read-line' && JSON.stringify(v.args) === JSON.stringify(['C:\\fixture\\src\\read.ts', {line: 37}])));
    check('path wins href; Ctrl click recorded trusted path callback', snapshots['path-wins-href'].link.kind === 'path' && callbacks.some(v => v.id === 'path-wins-href' && v.ctrl && v.trusted));
    check('all3 path clicks stop propagation; no disclosure toggle', (await Promise.all(['description-path', 'read-line', 'path-wins-href'].map(quick))).every(v => v.rowToggleCount === 0 && v.expanded === 'false'));
  });
  await runCase('keyboard-explicit-local-scope', async check => {
    await focus('read-line', '[data-disclosure-row]'); await key(' ', 'Space', 32); const rowOpen = await quick('read-line'); await key('Enter', 'Enter', 13); const rowClosed = await quick('read-line');
    await key('Tab', 'Tab', 9); const tab = await quick('read-line'); await capture('focus');
    await key(' ', 'Space', 32); const pathSpace = await quick('read-line');
    await focus('href', '[data-summary]'); await key(' ', 'Space', 32); const urlSpace = await quick('href');
    const linkKeys = await evaluate('window.harness.linkKeys()'), callbacks = await evaluate('window.harness.callbacks()');
    E.measurements.keyboard = {rowOpen, rowClosed, tab, pathSpace, urlSpace, linkKeys, localSpaceCallbacks: callbacks.filter(v => v.origin === 'explicit-local-Space-handler')};
    check('explicit local row Space opens and Enter closes', rowOpen.expanded === 'true' && rowClosed.expanded === 'false');
    check('trusted Tab focuses path button and focus-visible', tab.focused === 'summary' && tab.focusVisible);
    check('installed link stopprop leaves default unprevented before local adapter', linkKeys.length === 2 && linkKeys.every(v => v.key === ' ' && v.trusted && !v.defaultPreventedAfterInstalledHandler));
    check('explicit LOCAL Space path callback once; row remains closed', pathSpace.callbackCount === 2 && pathSpace.expanded === 'false' && E.measurements.keyboard.localSpaceCallbacks.length === 1);
    check('URL Space does not activate or toggle row', urlSpace.rowToggleCount === 0 && urlSpace.callbackCount === 0 && urlSpace.expanded === 'false');
  });
  await runCase('settlement-terminal-empty-zero-and-href', async check => {
    const error = snapshots.error, stopped = snapshots.stopped, terminal = snapshots['terminal-exit'], zero = snapshots['zero-totals'], empty = snapshots['empty-summary'], href = snapshots.href;
    check('error/stopped remove both links suffix totals', [error, stopped].every(v => v.link === null && v.suffix === null && v.totals === null));
    check('error first-line color error, stopped original summary warn', error.summary.text === 'Access denied' && error.quick.summaryColor === source.lightColors.error.rgb && stopped.summary.text === 'src/stopped.ts' && stopped.quick.summaryColor === source.lightColors.warn.rgb, {error: error.quick.summaryColor, stopped: stopped.quick.summaryColor});
    check('terminal exit suppresses link, keeps description neutral, exit label present', terminal.link === null && terminal.summary.text === 'Command description retained' && terminal.quick.summaryColor === terminal.quick.rowColor && terminal.exitLabel === 'Exit code: 7' && terminal.suffix === null && terminal.totals === null);
    check('zero0/0 totals visible, empty summary removes whole fragment', zero.totals.text === '+0 -0' && empty.summary === null && empty.sep === null && empty.suffix === null && empty.totals === null);
    check('href anchor uses same CSS with target blank noopener noreferrer', href.link.tag === 'A' && href.link.href === 'https://example.invalid/reference' && href.link.target === '_blank' && href.link.rel === 'noopener noreferrer');
  });
  await runCase('source-input-network-custody', async check => {
    const changed = []; for (const item of source.files) if (hash(await fs.readFile(item.path)) !== item.sha256) changed.push(item.id);
    E.sourceIntegrity = {allUnchanged: changed.length === 0, changedSourceIDs: changed};
    const events = await evaluate('window.harness.events()');
    E.input = {count: events.length, trusted: events.every(v => v.trusted), byType: Object.fromEntries([...new Set(events.map(v => v.type))].map(type => [type, events.filter(v => v.type === type).length])), keyboard: events.filter(v => v.type === 'keydown').map(({id, key, target, trusted}) => ({id, key, target, trusted})), nativeButtonEnterAssertions: 0};
    E.network = {requests, exceptions, protocolErrors};
    check('all installed/reference/prior/runner sources unchanged', changed.length === 0, changed);
    check('file-only page requests and no runtime/protocol errors', requests.every(v => !/^https?:|^wss?:/i.test(v.url)) && exceptions.length === 0 && protocolErrors.length === 0, E.network);
    check('observed clicks/keys/moves trusted after PID/profile/port gate', E.input.trusted && ['mousemove', 'click', 'keydown'].every(k => E.input.byType[k] > 0) && E.browser.ownership.pass);
    check('one startup, no Emulation, Enter assertions only explicit row handler', spawnCount === 1 && Object.keys(methods).every(v => !v.startsWith('Emulation.')) && E.input.keyboard.filter(v => v.key === 'Enter').every(v => v.target === 'row'));
  });
  E.computedStyles = await evaluate('window.harness.styles()');
}
async function cleanup() {
  const c = {target: 'Only spawned exact Chrome PID/profile/port', protectedProcessesSignalled: false, protectedGuiRequested: false}; E.cleanup = c;
  if (!child) { c.pass = true; c.browserNeverStarted = true; return; }
  invariant(!PROTECTED.has(child.pid), 'Protected cleanup PID');
  if (verified && browser?.socket.readyState === WebSocket.OPEN) {
    try { c.lastProcesses = (await browser.send('SystemInfo.getProcessInfo')).processInfo.map(({type, id}) => ({type, id})); invariant(c.lastProcesses.every(v => !PROTECTED.has(v.id)), 'Protected PID in cleanup'); await browser.send('Browser.close'); c.closedBy = 'owned Browser.close'; } catch (error) { c.closeError = String(error); }
  }
  const end = Date.now() + 7000; while (!exitInfo && Date.now() < end) await delay(50);
  if (!exitInfo) { c.killedSpawnedChild = child.kill(); await delay(500); }
  c.exit = exitInfo ?? {pending: true}; if (browser?.socket.readyState === WebSocket.OPEN) browser.socket.close();
  await waitFor(async () => { c.portReleased = await portFree(); return c.portReleased; }, 'owned port release', 5000);
  const ids = [...new Set([...(E.browser?.initialProcesses ?? []), ...(c.lastProcesses ?? []), {id: child.pid}].map(v => v.id))];
  const alive = id => { invariant(!PROTECTED.has(id), 'Protected PID probe'); try { process.kill(id, 0); return true; } catch (error) { if (error.code === 'ESRCH') return false; throw error; } };
  await waitFor(async () => { c.processes = ids.map(pid => ({pid, alive: alive(pid)})); return c.processes.every(v => !v.alive); }, 'owned process exits', 5000);
  c.allObservedProcessesExited = ids.length > 0 && c.processes.every(v => !v.alive);
  if (exitInfo && c.portReleased && c.allObservedProcessesExited) {
    invariant(norm(PROFILE) === norm(BASE + '-profile') && norm(PROFILE).startsWith(norm(path.join(ROOT, '.devdata/r117-tool-summary-'))), 'Wrong profile removal target');
    await noLinks(PROFILE); invariant(norm(await fs.realpath(PROFILE)) === norm(PROFILE), 'Redirected profile'); c.profileRemovalTarget = PROFILE; await fs.rm(PROFILE, {recursive: true, force: false}); c.profileRemoved = true;
  }
  c.pass = exitInfo?.code === 0 && c.portReleased && c.allObservedProcessesExited && c.profileRemoved;
}
const mdLink = (name, file) => '[' + name + '](<' + file.replaceAll('\\', '/') + '>)';
function report() {
  const layout = E.measurements.layout;
  return '# Round117 installed summary links and diff totals\n\n' + E.status.toUpperCase() + ': ' + E.summary.passedCases + '/' + E.cases.length + ' cases; ' + E.summary.passedAssertions + '/' + (E.summary.passedAssertions + E.summary.failedAssertions) + ' assertions. ' + mdLink('Metrics', OUT.json) + ' · ' + mdLink('Unique source/CSS/maps/algorithm tables', OUT.source) + ' · ' + mdLink('Runner', SCRIPT) + '.\n\n' + E.scope.description + '\n\n' +
    '| Surface | Current installed declaration / observation |\n|---|---|\n' +
    '| Link | Native button or href anchor; font13px/24px, color inherit; dotted underline1px with3px offset; text-align:left; border/margin/padding0; flex:0 1 auto, min-width0, ellipsis/nowrap/hidden. |\n' +
    '| Summary | Plain span uses flex:1 1 auto and no underline. Outer TextShimmer root is width:max-content, max-width:100%; the short summary does not automatically fill an otherwise wide row. Constrained long plain text fills available width and pushes totals to the content edge. |\n' +
    '| Suffix / totals | Suffix flex:none,13px/24px, margin-left4px. Totals combine summarySuffix+diffStat: code-font11px/24px, margin-left10px, translateY(.5px); +0 -0 remains visible. |\n' +
    '| Colors | Neutral totals inherit tertiary at rest. Hover row/link uses secondary; hover OR aria-expanded=true selects success/error for additions/deletions. Nonhover open row text remains tertiary. |\n' +
    '| State | error/stopped remove path and href links, suffix and totals. Error first-line replaces summary; stopped retains summary. Terminal exit suppresses links and preserves terminal description; empty summary removes entire fragment. |\n\n' +
    (layout ? 'Measured short path unused right: ' + layout.shortPathUnusedRight + 'px; short plain unused right: ' + layout.shortPlainUnusedRight + 'px. Tiny link client/scroll widths: ' + layout.tinyLinkClientWidth + '/' + layout.tinyLinkScrollWidth + 'px. Long plain client/scroll widths: ' + layout.longPlainClientWidth + '/' + layout.longPlainScrollWidth + 'px.\n\n' : '') +
    'The source-only VM executes the exact current bundled diff engine QL/eT/rT with Bw/QZ/L8. Empty text is zero content lines; remove exactly one terminal LF then split LF, keeping interior blanks and CR. Each line is normalized with LF before structuredPatch(context:3,maxEditLength:256). Count only hunk lines beginning + or -. At256 edits shared lines remain context; above256 a full-fragment fallback counts shared lines as replaced. Seventeen cases include CRLF, final terminator changes, blank lines, repeated diff entries, reorder and limit behavior. Read offset and path/state model tables also execute installed pure functions without importing app modules.\n\n' +
    'Trusted CDP path clicks stop propagation and log only local onOpenFile arguments; read uses {line:37}; description text may link to a separate path; a path wins href; Ctrl click still invokes the path callback. The URL anchor retains target=_blank/rel=noopener noreferrer without any navigation. Explicit row Enter/Space handlers and trusted Tab are measured. Space on the path uses an explicit local keydown activation adapter after installed stopPropagation; this is documented local behavior, not React or native-button default certification. No native-button Enter activation assertion is made.\n\n' +
    'Running/preparing shimmer source/CSS are retained without motion certification. Manual fixture does not run original React, ReadBlock/DiffBlock bodies, disclosure presence/motion, application host dispatch, file opening or URL navigation. Installed source integrity: ' + String(E.sourceIntegrity?.allUnchanged) + '; startup count: ' + spawnCount + '; DPR: ' + E.browser?.viewport?.dpr + '; cleanup: ' + String(E.cleanup?.pass) + '. All observed owned PIDs exited, loopback port released and owned profile removed when cleanup passes. Protected Electron20012/backend16008/LockApp12516 and GUI3080 untouched.\n\n' +
    Object.entries(E.screenshots).map(([name, v]) => mdLink(name, v.path)).join(' · ') + '\n\n' +
    E.cases.filter(v => v.status !== 'passed').map(v => 'Failure retained: ' + v.id + ': ' + (v.error ?? v.assertions.filter(a => !a.pass).map(a => a.name + ' ' + JSON.stringify(a.detail ?? '')).join('; '))).join('\n\n') + (E.errors.length ? '\n\nErrors: ' + E.errors.join('\n\n') : '') + '\n';
}
function textReport() { return ['Round117 installed summary links + totals: ' + E.status.toUpperCase(), 'Run ' + RUN, 'Cases ' + E.summary.passedCases + '/' + E.cases.length + '; assertions ' + E.summary.passedAssertions + '/' + (E.summary.passedAssertions + E.summary.failedAssertions), ...E.cases.map(v => v.status.toUpperCase() + ' ' + v.id + ' ' + v.assertions.filter(a => a.pass).length + '/' + v.assertions.length), ...E.cases.flatMap(v => v.assertions.filter(a => !a.pass).map(a => 'FAIL ' + v.id + ': ' + a.name + ' ' + JSON.stringify(a.detail ?? null))), ...E.cases.filter(v => v.error).map(v => v.id + ': ' + v.error), 'Cleanup ' + JSON.stringify(E.cleanup), 'Errors ' + JSON.stringify(E.errors), ''].join('\n'); }
async function main() {
  await scope(); const x = await extract(), built = fixture(x), inline = built.html.match(/<script>\n([\s\S]*)<\/script>/); invariant(inline, 'Missing inline fixture'); new vm.Script(inline[1], {filename: 'r117-tool-summary-inline.js'});
  if (MODE === '--prepare-only') {
    await writeOwned(OUT.source, JSON.stringify(source) + '\n'); await writeOwned(FIXTURE, built.html);
    const prep = {schema: 'r117-tool-summary-preflight/v1', status: 'prepared-no-browser', runId: RUN, sourceEvidence: OUT.source, sourceEvidenceSHA256: hash(await fs.readFile(OUT.source)), fixture: FIXTURE, fixtureSHA256: hash(built.html), inlineParsed: true, sourceOnlyDiffCases: source.sourceOnly.diffTotals.table.length, chromeNotStarted: true};
    await writeOwned(PREP, JSON.stringify(prep, null, 2) + '\n'); console.log(JSON.stringify({...prep, path: PREP, sourceEvidenceBytes: (await fs.stat(OUT.source)).size})); return;
  }
  const prep = JSON.parse(await fs.readFile(PREP, 'utf8')); invariant(prep.status === 'prepared-no-browser' && prep.runId === RUN && norm(prep.sourceEvidence) === norm(OUT.source), 'Wrong preparation');
  invariant(hash(await fs.readFile(OUT.source)) === prep.sourceEvidenceSHA256 && JSON.stringify(source) === JSON.stringify(JSON.parse(await fs.readFile(OUT.source, 'utf8'))), 'Source/runner changed after preflight');
  invariant(prep.fixtureSHA256 === hash(built.html) && hash(await fs.readFile(FIXTURE)) === prep.fixtureSHA256, 'Prepared fixture changed'); E.preflight = {path: PREP, inlineParsed: true, sourceEvidenceVerified: true, fixtureVerified: true};
  await writeOwned(ATTEMPT, JSON.stringify({round: 117, runId: RUN, startedAt: E.startedAt, prepared: PREP, chromeStartupsAllowed: 1, rerunForbidden: true}, null, 2) + '\n');
  try { await start(); await acceptance(built.values); } catch (error) { E.errors.push(error.stack ?? String(error)); console.error(error.stack ?? String(error)); } finally { try { await cleanup(); } catch (error) { E.errors.push('Cleanup ' + (error.stack ?? String(error))); } }
  E.finishedAt = new Date().toISOString(); E.cdpMethods = methods;
  E.summary = {passedCases: E.cases.filter(v => v.status === 'passed').length, failedCases: E.cases.filter(v => v.status === 'failed').length, passedAssertions: E.cases.flatMap(v => v.assertions).filter(v => v.pass).length, failedAssertions: E.cases.flatMap(v => v.assertions).filter(v => !v.pass).length};
  E.status = E.cases.length === 8 && E.summary.failedCases === 0 && E.errors.length === 0 && E.cleanup?.pass ? 'passed' : 'failed';
  E.compact = {styleRecords: E.computedStyles?.length ?? 0, sourceJSONBytes: (await fs.stat(OUT.source)).size, measurementJSONBytes: 0, targetCombinedJSONBytes: 250000};
  let encoded = JSON.stringify(E) + '\n'; E.compact.measurementJSONBytes = Buffer.byteLength(encoded); encoded = JSON.stringify(E) + '\n'; E.compact.measurementJSONBytes = Buffer.byteLength(encoded); encoded = JSON.stringify(E) + '\n';
  if (Buffer.byteLength(encoded) + E.compact.sourceJSONBytes >= 250000) { E.errors.push('Combined JSON exceeds250KB'); E.status = 'failed'; encoded = JSON.stringify(E) + '\n'; }
  await writeOwned(OUT.text, textReport()); await writeOwned(OUT.report, report()); await writeOwned(OUT.json, encoded);
  console.log(JSON.stringify({status: E.status, ...E.summary, evidence: OUT, combinedJSONBytes: Buffer.byteLength(encoded) + E.compact.sourceJSONBytes, cleanup: E.cleanup?.pass})); if (E.status !== 'passed') process.exitCode = 1;
}
await main();
